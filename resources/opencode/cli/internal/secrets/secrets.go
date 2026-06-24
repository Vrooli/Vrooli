// Package secrets resolves the OpenRouter API key the opencode resource needs
// and syncs it into the auth store the raw `opencode` binary reads
// (~/.local/share/opencode/auth.json). It replaces the bash
// `opencode::load_secrets` / `opencode::auth::sync_openrouter` helpers with a
// portable Go implementation: it shells the vault SSOT (`resource-vault`) and
// falls back to the OpenRouter credentials file, never sourcing ad-hoc bash.
package secrets

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
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

// Options injects seams for tests; the zero value uses the real environment.
type Options struct {
	Getenv    func(string) string
	VaultBin  string // defaults to "resource-vault"
	CredFile  string // OpenRouter credentials JSON; defaults to XDG path
	LookVault func(string) (string, error)
	RunVault  func(ctx context.Context, bin string, args ...string) ([]byte, error)
}

func (o Options) getenv(k string) string {
	if o.Getenv != nil {
		return o.Getenv(k)
	}
	return os.Getenv(k)
}

func (o Options) vaultBin() string {
	if strings.TrimSpace(o.VaultBin) != "" {
		return o.VaultBin
	}
	return "resource-vault"
}

// ResolveOpenRouterKey resolves the key from, in order: the environment, the
// vault SSOT (`resource-vault secrets export opencode|openrouter`), and the
// OpenRouter credentials file. Returns "" when no usable key is found.
func ResolveOpenRouterKey(ctx context.Context, o Options) string {
	// 1. Environment (already exported by an outer shell).
	if key := strings.TrimSpace(o.getenv("OPENROUTER_API_KEY")); KeyUsable(key) {
		return key
	}

	// 2. Vault exports.
	lookVault := o.LookVault
	if lookVault == nil {
		lookVault = exec.LookPath
	}
	if _, err := lookVault(o.vaultBin()); err == nil {
		for _, scope := range []string{"opencode", "openrouter"} {
			if key := vaultExportKey(ctx, o, scope); KeyUsable(key) {
				return key
			}
		}
	}

	// 3. Credentials file fallback.
	if key := credentialsFileKey(o); KeyUsable(key) {
		return key
	}
	return ""
}

// vaultExportKey runs `resource-vault secrets export <scope>` and parses the
// exported OPENROUTER_API_KEY from the shell-export lines.
func vaultExportKey(ctx context.Context, o Options, scope string) string {
	run := o.RunVault
	if run == nil {
		run = func(ctx context.Context, bin string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, bin, args...).Output()
		}
	}
	out, err := run(ctx, o.vaultBin(), "secrets", "export", scope)
	if err != nil {
		return ""
	}
	return parseExportedKey(string(out), "OPENROUTER_API_KEY")
}

// parseExportedKey extracts NAME's value from shell-export output lines like
// `export NAME=value` or `NAME="value"`.
func parseExportedKey(out, name string) string {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		if line[:eq] != name {
			continue
		}
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`)
		return val
	}
	return ""
}

func (o Options) credFile() string {
	if strings.TrimSpace(o.CredFile) != "" {
		return o.CredFile
	}
	base := strings.TrimSpace(o.getenv("XDG_CONFIG_HOME"))
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, ".config")
		}
	}
	return filepath.Join(base, "vrooli", "resources", "openrouter", "openrouter-credentials.json")
}

func credentialsFileKey(o Options) string {
	data, err := os.ReadFile(o.credFile())
	if err != nil {
		return ""
	}
	var parsed struct {
		Data struct {
			APIKey string `json:"apiKey"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Data.APIKey)
}

// SyncAuth writes the resolved key into the opencode auth store, merging into
// any existing providers. No-op when the key is not usable. Mirrors
// opencode::auth::sync_openrouter.
func SyncAuth(authPath, key string) error {
	if !KeyUsable(key) {
		return nil
	}
	providers := map[string]json.RawMessage{}
	if data, err := os.ReadFile(authPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		_ = json.Unmarshal(data, &providers)
	}
	entry, _ := json.Marshal(map[string]string{"type": "api", "key": key})
	providers["openrouter"] = entry

	out, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(authPath, out, 0o600); err != nil {
		return err
	}
	return nil
}
