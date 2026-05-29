package builder

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin-xcloud/apiclient"
)

// StepCreateNetwork creates a temporary tenant network when the build opted
// in (createNetwork). By default the builder references an existing network
// (typically "default") and this step is a no-op.
type StepCreateNetwork struct{}

func (s *StepCreateNetwork) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	if !cfg.createNetwork {
		return multistep.ActionContinue
	}

	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	ui.Sayf("Creating temporary network %q ...", cfg.Network)
	if err := client.CreateNetwork(ctx, apiclient.CreateNetworkRequest{
		RegionID: cfg.RegionID,
		Name:     cfg.Network,
	}); err != nil {
		state.Put("error", fmt.Errorf("failed to create network: %w", err))
		ui.Errorf("Could not create temporary network %q: %v", cfg.Network, err)
		return multistep.ActionHalt
	}

	state.Put("created_network", cfg.Network)
	ui.Sayf("Temporary network %q is ready", cfg.Network)
	return multistep.ActionContinue
}

func (s *StepCreateNetwork) Cleanup(state multistep.StateBag) {
	name, ok := state.Get("created_network").(string)
	if !ok || name == "" {
		return
	}
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if cfg.KeepVM {
		ui.Sayf("Keeping temporary network %q because keep_vm=true", name)
		return
	}

	ui.Sayf("Deleting temporary network %q ...", name)
	ctx, cancel := cleanupContext()
	defer cancel()
	if err := client.DeleteNetwork(ctx, cfg.RegionID, name); err != nil {
		ui.Errorf("Could not delete temporary network %q: %v", name, err)
		return
	}
	ui.Sayf("Temporary network %q deleted", name)
}
