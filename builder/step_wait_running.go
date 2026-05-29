package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin/apiclient"
)

// StepWaitRunning polls the instance until it reaches status "running" with
// no pending action. It halts the build if the instance enters status
// "error".
type StepWaitRunning struct{}

func (s *StepWaitRunning) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)
	id := state.Get("instance_id").(string)

	ui.Sayf("Waiting for instance %q to become running ...", cfg.Name)

	deadline := time.After(cfg.stateTimeout)
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	var lastStatus string
	for {
		select {
		case <-ctx.Done():
			state.Put("error", fmt.Errorf("context cancelled while waiting for instance to become running"))
			ui.Error("Build was cancelled while waiting for the instance")
			return multistep.ActionHalt
		case <-deadline:
			state.Put("error", fmt.Errorf("timed out waiting for instance to become running"))
			ui.Error("Timed out waiting for instance to reach running state")
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
				err := fmt.Errorf("instance entered error state: %s", msg)
				state.Put("error", err)
				ui.Errorf("Instance %q failed: %s", cfg.Name, msg)
				return multistep.ActionHalt
			case "running":
				if derefStr(inst.PendingAction) == "" {
					ui.Sayf("Instance %q is running", cfg.Name)
					return multistep.ActionContinue
				}
				fallthrough
			default:
				status := inst.Status
				if pa := derefStr(inst.PendingAction); pa != "" {
					status = fmt.Sprintf("%s (pending %s)", inst.Status, pa)
				}
				if status != lastStatus {
					ui.Sayf("Instance status: %s", status)
					lastStatus = status
				}
			}
		}
	}
}

func (s *StepWaitRunning) Cleanup(multistep.StateBag) {}
