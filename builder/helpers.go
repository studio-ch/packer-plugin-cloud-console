package builder

import (
	"context"
	"strings"
	"time"

	"github.com/studio-ch/packer-plugin/apiclient"
)

// cleanupContext returns a short-lived context for teardown API calls so a
// cancelled build still gets a chance to delete the resources it created.
func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// registryAuth derives the discriminated registry auth union from a saved
// credential id, or an ad-hoc username/password, or anonymous.
func registryAuth(credentialID, username, password string) apiclient.RegistryAuth {
	switch {
	case credentialID != "":
		return apiclient.SavedAuth(credentialID)
	case username != "":
		return apiclient.AdhocAuth(username, password)
	default:
		return apiclient.AnonymousAuth()
	}
}

// sanitizeImageName coerces an arbitrary name into the catalog image-name
// shape: lowercase, leading alnum, only [a-z0-9-], capped at 119 chars.
func sanitizeImageName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == ' ' || r == '_' || r == '.':
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "packer"
	}
	if len(out) > 119 {
		out = strings.Trim(out[:119], "-")
	}
	return out
}
