// Package secrets resolves the OpenRouter API key the opencode resource needs
// and syncs it into the auth store the raw `opencode` binary reads
// (~/.local/share/opencode/auth.json). It replaces the bash
// `opencode::load_secrets` / `opencode::auth::sync_openrouter` helpers with a
// portable Go implementation. The credential authority resolves values before
// process launch and injects the value only into this process's environment.
package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// SyncOpenRouterAuth mirrors the credential-authority value into the auth
// file consumed by the upstream OpenCode binary. The authority remains the
// source of truth; this file is only the runtime adapter OpenCode requires.
// Unrelated provider entries and fields are preserved byte-for-semantic-value.
func SyncOpenRouterAuth(path, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if !KeyUsable(key) {
		return false, nil
	}

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read OpenCode auth: %w", err)
	}
	auth := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &auth); err != nil {
			return false, fmt.Errorf("parse OpenCode auth: %w", err)
		}
	}

	provider := map[string]json.RawMessage{}
	if raw := auth["openrouter"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &provider); err != nil {
			return false, fmt.Errorf("parse OpenRouter auth: %w", err)
		}
	}
	var currentKey, currentType string
	_ = json.Unmarshal(provider["key"], &currentKey)
	_ = json.Unmarshal(provider["type"], &currentType)
	if currentKey == key && currentType == "api" {
		return false, nil
	}
	provider["type"] = json.RawMessage(`"api"`)
	encodedKey, err := json.Marshal(key)
	if err != nil {
		return false, fmt.Errorf("encode OpenRouter auth: %w", err)
	}
	provider["key"] = encodedKey
	encodedProvider, err := json.Marshal(provider)
	if err != nil {
		return false, fmt.Errorf("encode OpenRouter provider auth: %w", err)
	}
	auth["openrouter"] = encodedProvider
	encoded, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode OpenCode auth: %w", err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("mkdir OpenCode auth dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".auth.json.tmp-*")
	if err != nil {
		return false, fmt.Errorf("create OpenCode auth temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("protect OpenCode auth temp file: %w", err)
	}
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return false, fmt.Errorf("write OpenCode auth: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return false, fmt.Errorf("close OpenCode auth temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return false, fmt.Errorf("install OpenCode auth: %w", err)
	}
	return true, nil
}
