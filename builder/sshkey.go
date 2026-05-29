package builder

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// generateTempSSHKey creates an ephemeral ed25519 keypair and returns the
// public key in authorized_keys format and the private key as an OpenSSH PEM
// block. Used when the template configures the ssh communicator without any
// ssh_key_ids — the plugin registers the public key, injects it into the VM
// via /v1/ssh-keys, and uses the private key for the communicator.
func generateTempSSHKey(comment string) (authorizedKey string, privatePEM []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", nil, fmt.Errorf("wrap ssh public key: %w", err)
	}
	authorizedKey = string(ssh.MarshalAuthorizedKey(sshPub))

	block, err := ssh.MarshalPrivateKey(priv, comment)
	if err != nil {
		return "", nil, fmt.Errorf("marshal ssh private key: %w", err)
	}
	privatePEM = pem.EncodeToMemory(block)

	return authorizedKey, privatePEM, nil
}
