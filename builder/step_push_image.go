package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin-cloud-console/apiclient"
)

// StepPushImage enqueues an image-push job for the stopped instance and polls
// it until it reaches a terminal state. On success it records the pushed
// image / digest / registered name in state. No-op when push_image is unset.
type StepPushImage struct{}

func (s *StepPushImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	if cfg.PushImage == "" {
		return multistep.ActionContinue
	}

	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)
	id := state.Get("instance_id").(string)

	ui.Sayf("Pushing instance %q to %s ...", cfg.Name, cfg.PushImage)
	job, err := client.CreatePushJob(ctx, id, apiclient.CreatePushJobRequest{
		OCIReference: cfg.PushImage,
		Auth:         registryAuth(cfg.PushCredential, cfg.PushUsername, cfg.PushPassword),
		Precache:     cfg.PushPrecache,
	})
	if err != nil {
		state.Put("error", fmt.Errorf("failed to enqueue image push: %w", err))
		ui.Errorf("Could not enqueue image push for %q: %v", cfg.Name, err)
		return multistep.ActionHalt
	}

	deadline := time.After(cfg.stateTimeout)
	ticker := time.NewTicker(cfg.pollInterval)
	defer ticker.Stop()

	var lastStatus string
	for {
		select {
		case <-ctx.Done():
			state.Put("error", fmt.Errorf("context cancelled while waiting for image push"))
			ui.Error("Build was cancelled while waiting for the image push")
			return multistep.ActionHalt
		case <-deadline:
			state.Put("error", fmt.Errorf("timed out waiting for image push to finish"))
			ui.Error("Timed out waiting for the image push to finish")
			return multistep.ActionHalt
		case <-ticker.C:
			j, err := client.GetPushJob(ctx, job.ID)
			if err != nil {
				ui.Errorf("Could not check push job status yet: %v", err)
				continue
			}
			switch j.Status {
			case "succeeded":
				digest := derefStr(j.Digest)
				ui.Sayf("Pushed image to %s@%s", cfg.PushImage, digest)
				state.Put("pushed_image", cfg.PushImage)
				state.Put("pushed_digest", digest)
				if name := derefStr(j.RegisteredImageName); name != "" {
					state.Put("registered_image_name", name)
				}
				return multistep.ActionContinue
			case "failed", "failed_registration":
				msg := derefStr(j.Error)
				err := fmt.Errorf("image push %s: %s", j.Status, msg)
				state.Put("error", err)
				ui.Errorf("Image push %s: %s", j.Status, msg)
				return multistep.ActionHalt
			case "cancelled":
				err := fmt.Errorf("image push was cancelled")
				state.Put("error", err)
				ui.Error("Image push was cancelled")
				return multistep.ActionHalt
			default:
				if j.Status != lastStatus {
					ui.Sayf("Image push status: %s", j.Status)
					lastStatus = j.Status
				}
			}
		}
	}
}

func (s *StepPushImage) Cleanup(multistep.StateBag) {}
