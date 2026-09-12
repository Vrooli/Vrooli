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

// KeyUsable reports whether a resolved API key looks like a real
// credential rather than a placeholder/stub. Generic across providers.
func KeyUsable(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if valueLooksInvalid(key) {
		return false
	}
	return len(key) >= 20
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
	Getenv   func(string) string
	AuthPath string // path to auth.json, checked as fallback when env is empty
}

func (o Options) getenv(k string) string {
	if o.Getenv != nil {
		return o.Getenv(k)
	}
	return os.Getenv(k)
}

// resolveKeyFromAuth reads a provider's key from the OpenCode auth file.
func resolveKeyFromAuth(authPath, providerID string) string {
	if authPath == "" {
		return ""
	}
	data, err := os.ReadFile(authPath)
	if err != nil {
		return ""
	}
	var auth map[string]struct {
		Key string `json:"key"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return ""
	}
	prov, ok := auth[providerID]
	if !ok || prov.Type != "api" {
		return ""
	}
	return strings.TrimSpace(prov.Key)
}

// ResolveOpenRouterKey resolves the OpenRouter API key from the process
// environment, falling back to the auth file when the env var is empty.
func ResolveOpenRouterKey(o Options) string {
	if key := strings.TrimSpace(o.getenv("OPENROUTER_API_KEY")); KeyUsable(key) {
		return key
	}
	return resolveKeyFromAuth(o.AuthPath, "openrouter")
}

// ResolveOpenCodeGoKey resolves the OpenCode Go subscription key from the
// process environment, falling back to the auth file when the env var is empty.
func ResolveOpenCodeGoKey(o Options) string {
	if key := strings.TrimSpace(o.getenv("OPENCODE_GO_KEY")); KeyUsable(key) {
		return key
	}
	return resolveKeyFromAuth(o.AuthPath, "opencode-go")
}

// SyncOpenCodeGoAuth syncs the Go API key into the OpenCode auth file
// under the "opencode-go" provider key.
func SyncOpenCodeGoAuth(path, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if !KeyUsable(key) {
		return false, nil
	}
	return syncProviderAuth(path, "opencode-go", key)
}

// SyncOpenRouterAuth mirrors the credential-authority value into the auth
// file consumed by the upstream OpenCode binary.
func SyncOpenRouterAuth(path, key string) (bool, error) {
	key = strings.TrimSpace(key)
	if !KeyUsable(key) {
		return false, nil
	}
	return syncProviderAuth(path, "openrouter", key)
}

// syncProviderAuth writes or updates a provider entry in the OpenCode auth file.
func syncProviderAuth(path, providerID, key string) (bool, error) {
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
	if raw := auth[providerID]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &provider); err != nil {
			return false, fmt.Errorf("parse %s auth: %w", providerID, err)
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
		return false, fmt.Errorf("encode %s auth: %w", providerID, err)
	}
	provider["key"] = encodedKey
	encodedProvider, err := json.Marshal(provider)
	if err != nil {
		return false, fmt.Errorf("encode %s provider auth: %w", providerID, err)
	}
	auth[providerID] = encodedProvider
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
