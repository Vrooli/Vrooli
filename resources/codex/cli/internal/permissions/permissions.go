// Package permissions manages the Codex agent's permission config at
// ~/.codex/config.toml (user scope) or ~/.codex/requirements.toml
// (admin scope).
//
// Scope is narrow on purpose: the adapter owns only a Vrooli-namespaced
// section, `[vrooli.permissions]`, containing three string arrays —
// `bash_deny`, `bash_ask`, `bash_allow`. Every other top-level key and
// section (including all Codex-native `[profiles.*]`, `sandbox_mode`,
// `approval_policy`, etc.) round-trips untouched.
//
// Codex versions at and beyond the pinned resource version expose a
// hooks.json PreToolUse command surface. The native sandbox/approval controls
// remain separate; the adapter records both and reports hook firing as
// unverified until a runtime canary proves the installed binary invokes it.
//
// Duplicated structurally from resources/claude-code and
// resources/opencode permissions packages per the duplicate-before-extract
// memory. Phase 4 will extract the shared canonical Policy + state shape
// into packages/cli-core/agentpolicy.
package permissions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/pelletier/go-toml/v2"
)

// Scope selects which Codex config file the adapter targets.
type Scope string

const (
	// ScopeUser writes ~/.codex/config.toml.
	ScopeUser Scope = "user"
	// ScopeAdmin writes ~/.codex/requirements.toml (admin-enforced).
	ScopeAdmin Scope = "admin"
)

// vrooliSectionKey is the top-level TOML table the adapter owns.
const vrooliSectionKey = "vrooli"

// Policy is the canonical in-memory shape of the bash-pattern subset of
// the Codex permission config the adapter manages.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
}

// Adapter binds the on-disk config path. SettingsPath is selected by
// DefaultAdapter based on Scope.
type Adapter struct {
	SettingsPath string
	HookPath     string
	Scope        Scope
}

// DefaultAdapter returns an Adapter rooted at $HOME/.codex with the
// given scope (defaults to ScopeUser).
func DefaultAdapter(scope Scope) (*Adapter, error) {
	if scope == "" {
		scope = ScopeUser
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	file := "config.toml"
	if scope == ScopeAdmin {
		file = "requirements.toml"
	}
	return &Adapter{
		SettingsPath: filepath.Join(home, ".codex", file),
		HookPath:     filepath.Join(home, ".codex", "hooks.json"),
		Scope:        scope,
	}, nil
}

// Load reads and parses the config file. A missing file is not an
// error — it resolves to an empty Policy so callers can use Save to
// create a new file.
func (a *Adapter) Load() (Policy, error) {
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Policy{}, nil
		}
		return Policy{}, fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}
	if len(data) == 0 {
		return Policy{}, nil
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Policy{}, fmt.Errorf("parse %s: %w", a.SettingsPath, err)
	}
	section, _ := doc[vrooliSectionKey].(map[string]any)
	perms, _ := section["permissions"].(map[string]any)
	return Policy{
		BashDeny:  decodeStringArray(perms, "bash_deny"),
		BashAsk:   decodeStringArray(perms, "bash_ask"),
		BashAllow: decodeStringArray(perms, "bash_allow"),
	}, nil
}

func decodeStringArray(m map[string]any, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Save writes the policy to disk. It preserves every top-level key
// other than the `vrooli` section.
func (a *Adapter) Save(p Policy) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	doc := map[string]any{}
	if data, err := os.ReadFile(a.SettingsPath); err == nil && len(data) > 0 {
		if uerr := toml.Unmarshal(data, &doc); uerr != nil {
			return fmt.Errorf("parse %s: %w", a.SettingsPath, uerr)
		}
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}

	// Rebuild [vrooli] preserving any unknown sub-keys (e.g. future
	// Vrooli-managed fields besides permissions).
	section, _ := doc[vrooliSectionKey].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}

	perms := map[string]any{}
	if len(p.BashDeny) > 0 {
		perms["bash_deny"] = sortedCopy(p.BashDeny)
	}
	if len(p.BashAsk) > 0 {
		perms["bash_ask"] = sortedCopy(p.BashAsk)
	}
	if len(p.BashAllow) > 0 {
		perms["bash_allow"] = sortedCopy(p.BashAllow)
	}
	if len(perms) == 0 {
		delete(section, "permissions")
	} else {
		section["permissions"] = perms
	}

	if len(section) == 0 {
		delete(doc, vrooliSectionKey)
	} else {
		doc[vrooliSectionKey] = section
	}

	out, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode %s: %w", a.SettingsPath, err)
	}
	tmp := a.SettingsPath + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, a.SettingsPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, a.SettingsPath, err)
	}
	if a.HookPath != "" {
		if err := a.savePreToolHook(p); err != nil {
			return err
		}
	}
	return nil
}

// RenderHook returns the native Codex PreToolUse projection. The command is
// configurable because the policy runner is host-owned; no shell-specific
// wrapper is assumed.
func (a *Adapter) RenderHook(p Policy) map[string]any {
	if len(p.BashDeny) == 0 {
		return nil
	}
	command := os.Getenv("VROOLI_AGENT_POLICY_RUNNER")
	if command == "" {
		command = "vrooli-policy-runner"
	}
	return map[string]any{
		"managedBy": "vrooli",
		"hooks": map[string]any{
			"PreToolUse": []any{map[string]any{
				"matcher": "shell_command|exec_command|apply_patch|mcp",
				"hooks":   []any{map[string]any{"type": "command", "command": command + " hook --runner codex"}},
			}},
		},
	}
}

func (a *Adapter) savePreToolHook(p Policy) error {
	hook := a.RenderHook(p)
	if hook == nil {
		if err := os.Remove(a.HookPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove codex hook: %w", err)
		}
		return nil
	}
	data, err := json.MarshalIndent(hook, "", "  ")
	if err != nil {
		return fmt.Errorf("encode codex hook: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(a.HookPath), 0o755); err != nil {
		return fmt.Errorf("mkdir codex hook directory: %w", err)
	}
	tmp := a.HookPath + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write codex hook: %w", err)
	}
	if err := os.Rename(tmp, a.HookPath); err != nil {
		return fmt.Errorf("publish codex hook: %w", err)
	}
	return nil
}

// Fingerprint returns the sha256 hex of the canonical Policy projection.
// drift-check compares this against the state-file value.
func Fingerprint(p Policy) string {
	canon := struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
	}{
		Allow: sortedCopy(p.BashAllow),
		Ask:   sortedCopy(p.BashAsk),
		Deny:  sortedCopy(p.BashDeny),
	}
	data, _ := json.Marshal(canon)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
