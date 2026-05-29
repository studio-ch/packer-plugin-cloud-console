package builder

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"

	"github.com/studio-ch/packer-plugin-xcloud/apiclient"
)

// StepSSHKey generates an ephemeral ed25519 keypair, registers the public key
// as a tenant SSH key, and wires the private key into the communicator. The
// key id is stored in state so StepCreateInstance attaches it; cleanup
// deletes the key unless keep_vm is set.
type StepSSHKey struct{}

func (s *StepSSHKey) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	keyName := fmt.Sprintf("packer-%s-%d", cfg.Name, time.Now().Unix())
	if len(keyName) > 64 {
		keyName = keyName[:64]
	}

	ui.Say("Generating temporary SSH keypair ...")
	authorizedKey, privatePEM, err := generateTempSSHKey(keyName)
	if err != nil {
		state.Put("error", fmt.Errorf("failed to generate ssh key: %w", err))
		ui.Errorf("Could not generate temporary SSH key: %v", err)
		return multistep.ActionHalt
	}

	key, err := client.CreateSSHKey(ctx, apiclient.CreateSSHKeyRequest{
		Name:      keyName,
		PublicKey: authorizedKey,
	})
	if err != nil {
		state.Put("error", fmt.Errorf("failed to register ssh key: %w", err))
		ui.Errorf("Could not register temporary SSH key: %v", err)
		return multistep.ActionHalt
	}

	cfg.Comm.SSHPrivateKey = privatePEM
	state.Put("temp_ssh_key_id", key.ID)
	ui.Sayf("Temporary SSH key %q registered", key.Name)
	return multistep.ActionContinue
}

func (s *StepSSHKey) Cleanup(state multistep.StateBag) {
	id, ok := state.Get("temp_ssh_key_id").(string)
	if !ok || id == "" {
		return
	}
	cfg := state.Get("config").(*Config)
	client := state.Get("client").(*apiclient.Client)
	ui := state.Get("ui").(packer.Ui)

	if cfg.KeepVM {
		ui.Say("Keeping temporary SSH key because keep_vm=true")
		return
	}

	ui.Say("Deleting temporary SSH key ...")
	ctx, cancel := cleanupContext()
	defer cancel()
	if err := client.DeleteSSHKey(ctx, id); err != nil {
		ui.Errorf("Could not delete temporary SSH key: %v", err)
		return
	}
	ui.Say("Temporary SSH key deleted")
}
