package builder

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin/apiclient"
)

// StepRegisterImage resolves the base image. When pull_image is set it
// registers a temporary OCI image in the tenant catalog and deletes it on
// cleanup; otherwise it uses the existing catalog image verbatim.
type StepRegisterImage struct{}

func (s *StepRegisterImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	if cfg.PullImage == "" {
		state.Put("vm_image", cfg.Image)
		ui.Sayf("Using existing image %q", cfg.Image)
		return multistep.ActionContinue
	}

	client := state.Get("client").(*apiclient.Client)
	name := sanitizeImageName(cfg.Name + "-image")

	ui.Sayf("Registering temporary image %q from %s ...", name, cfg.PullImage)
	img, err := client.RegisterImage(ctx, apiclient.RegisterImageRequest{
		RegionID:     cfg.RegionID,
		Name:         name,
		OCIReference: cfg.PullImage,
		Auth:         registryAuth(cfg.PullCredential, cfg.PullUsername, cfg.PullPassword),
		Precache:     cfg.PullPrecache,
	})
	if err != nil {
		state.Put("error", fmt.Errorf("failed to register image: %w", err))
		ui.Errorf("Could not register temporary image %q: %v", name, err)
		return multistep.ActionHalt
	}

	state.Put("created_image", img.Name)
	state.Put("vm_image", img.Name)
	ui.Sayf("Temporary image %q is registered", img.Name)
	return multistep.ActionContinue
}

func (s *StepRegisterImage) Cleanup(state multistep.StateBag) {
	name, ok := state.Get("created_image").(string)
	if !ok || name == "" {
		return
	}
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if cfg.KeepVM {
		ui.Sayf("Keeping temporary image %q because keep_vm=true", name)
		return
	}

	ui.Sayf("Deleting temporary image %q ...", name)
	ctx, cancel := cleanupContext()
	defer cancel()
	if err := client.DeleteImage(ctx, cfg.RegionID, name); err != nil {
		ui.Errorf("Could not delete temporary image %q: %v", name, err)
		return
	}
	ui.Sayf("Temporary image %q deleted", name)
}
