// Package secrets resolves the OpenRouter API key the opencode resource needs
// and syncs it into the auth store the raw `opencode` binary reads
// (~/.local/share/opencode/auth.json). It replaces the bash
// `opencode::load_secrets` / `opencode::auth::sync_openrouter` helpers with a
// portable Go implementation. The credential authority resolves values before
// process launch and injects the value only into this process's environment.
package secrets

import (
	"os"
	"strings"
)

// KeyUsable reports whether a resolved OpenRouter key looks like a real
// credential (sk-or- prefix, length floor) rather than a placeholder/stub the
// secrets backend can emit. Mirrors opencode::openrouter::key_usable.
func KeyUsable(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if valueLooksInvalid(key) {
		return false
	}
	if !strings.HasPrefix(key, "sk-or-") {
		return false
	}
	return len(key) >= 40
}

// valueLooksInvalid flags placeholder/error sentinels the backend may return.
func valueLooksInvalid(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return true
	}
	if strings.HasPrefix(v, "auto-null-") {
		return true
	}
	return strings.Contains(v, "[ERROR]") || strings.Contains(v, "Failed to retrieve secret") || strings.Contains(v, "❌")
}

// Options injects the environment seam for tests; the zero value reads the
// process environment populated by the credential authority.
type Options struct {
	Getenv func(string) string
}

func (o Options) getenv(k string) string {
	if o.Getenv != nil {
		return o.Getenv(k)
	}
	return os.Getenv(k)
}

// ResolveOpenRouterKey accepts only the ephemeral value injected into this
// process by the credential authority. It intentionally has no Vault or
// resource-private-file compatibility path.
func ResolveOpenRouterKey(o Options) string {
	if key := strings.TrimSpace(o.getenv("OPENROUTER_API_KEY")); KeyUsable(key) {
		return key
	}
	return ""
}
