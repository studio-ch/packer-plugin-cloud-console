package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin-xcloud/apiclient"
)

// StepShutdown gracefully shuts the instance down and waits until it reports
// status "stopped". Only relevant when push_image is configured; the build
// only adds this step in that case.
type StepShutdown struct{}

func (s *StepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	if cfg.PushImage == "" {
		return multistep.ActionContinue
	}

	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)
	id := state.Get("instance_id").(string)

	ui.Sayf("Shutting down instance %q before pushing the image ...", cfg.Name)
	if err := client.ShutdownInstance(ctx, id); err != nil {
		state.Put("error", fmt.Errorf("failed to request shutdown: %w", err))
		ui.Errorf("Could not request shutdown for instance %q: %v", cfg.Name, err)
		return multistep.ActionHalt
	}

	deadline := time.After(cfg.stateTimeout)
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	var lastStatus string
	for {
		select {
		case <-ctx.Done():
			state.Put("error", fmt.Errorf("context cancelled while waiting for instance to stop"))
			ui.Error("Build was cancelled while waiting for the instance to stop")
			return multistep.ActionHalt
		case <-deadline:
			state.Put("error", fmt.Errorf("timed out waiting for instance to stop"))
			ui.Error("Timed out waiting for instance to reach stopped state")
			return multistep.ActionHalt
		case <-ticker.C:
			inst, err := client.GetInstance(ctx, id)
			if err != nil {
				ui.Errorf("Could not check instance status yet: %v", err)
				continue
			}
			switch inst.Status {
			case "error":
				msg := derefStr(inst.LastError)
				err := fmt.Errorf("instance entered error state while stopping: %s", msg)
				state.Put("error", err)
				ui.Errorf("Instance %q failed while stopping: %s", cfg.Name, msg)
				return multistep.ActionHalt
			case "stopped":
				ui.Sayf("Instance %q is stopped", cfg.Name)
				return multistep.ActionContinue
			default:
				if inst.Status != lastStatus {
					ui.Sayf("Instance status: %s", inst.Status)
					lastStatus = inst.Status
				}
			}
		}
	}
}

func (s *StepShutdown) Cleanup(multistep.StateBag) {}
