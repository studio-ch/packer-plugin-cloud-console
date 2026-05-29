package builder

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin/apiclient"
)

// StepCreateInstance creates the builder VM. The instance starts in status
// "pending"; the worker drives it to running asynchronously (StepWaitRunning
// polls for it).
type StepCreateInstance struct{}

func (s *StepCreateInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	imageRef := cfg.Image
	if v, ok := state.GetOk("vm_image"); ok {
		imageRef = v.(string)
	}

	sshKeyIDs := append([]string(nil), cfg.SSHKeyIDs...)
	if v, ok := state.GetOk("temp_ssh_key_id"); ok {
		sshKeyIDs = append(sshKeyIDs, v.(string))
	}

	req := apiclient.CreateInstanceRequest{
		RegionID:   cfg.RegionID,
		Name:       cfg.Name,
		CPUCores:   cfg.CPUCores,
		MemoryGib:  cfg.Memory,
		DiskGib:    cfg.Disk,
		ImageRef:   imageRef,
		NetworkRef: cfg.Network,
		SSHKeyIDs:  sshKeyIDs,
	}
	if cfg.AdminUsername != "" {
		req.AdminUsername = cfg.AdminUsername
	}
	if cfg.useElasticIP {
		req.PendingElasticIP = &apiclient.PendingElasticIP{Mode: "allocate"}
	}

	ui.Sayf("Creating instance %q from image %q ...", cfg.Name, imageRef)
	inst, err := client.CreateInstance(ctx, req)
	if err != nil {
		state.Put("error", fmt.Errorf("failed to create instance: %w", err))
		ui.Errorf("Could not create instance %q: %v", cfg.Name, err)
		return multistep.ActionHalt
	}

	state.Put("instance_id", inst.ID)

	// Resolve the SSH username from the server-resolved adminUsername,
	// falling back to the configured value, then "admin".
	username := derefStr(inst.AdminUsername)
	if username == "" {
		username = cfg.AdminUsername
	}
	if username == "" {
		username = "admin"
	}
	cfg.Comm.SSHUsername = username

	ui.Sayf("Instance %q created (id %s)", cfg.Name, inst.ID)
	return multistep.ActionContinue
}

func (s *StepCreateInstance) Cleanup(state multistep.StateBag) {
	id, ok := state.Get("instance_id").(string)
	if !ok || id == "" {
		return
	}
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if cfg.KeepVM {
		ui.Sayf("Keeping instance %q because keep_vm=true", cfg.Name)
		return
	}

	ui.Sayf("Deleting instance %q ...", cfg.Name)
	ctx, cancel := cleanupContext()
	defer cancel()
	if err := client.DeleteInstance(ctx, id); err != nil {
		ui.Errorf("Could not delete instance %q: %v", cfg.Name, err)
		return
	}
	ui.Sayf("Instance %q deletion requested", cfg.Name)
}
