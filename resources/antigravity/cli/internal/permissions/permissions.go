// Package permissions manages the Antigravity agent's native command/file/URL
// permission grants at ~/.gemini/antigravity-cli/settings.json.
//
// Antigravity enforces permission grants from its own settings file: the
// `permissions` object holds the global allow/deny/ask rules for files,
// commands, and URLs (see the bundled `antigravity_guide` SKILL.md, section
// "Configuration Settings (settings.json)" and "Global Settings → Permission
// Grants / Command Allowlist / Denylist"). This adapter owns ONLY the three
// managed arrays (`deny`, `ask`, `allow`) inside that `permissions` object; every
// other top-level settings key (model, toolPermission, trustedWorkspaces, …) and
// every other key inside `permissions` round-trips untouched.
//
// Scope: this is the NATIVE enforcement seam (branch (a) of the plan) — unlike a
// hook backstop, Antigravity reads these grants directly. Antigravity exposes no
// user-writable PreToolUse hook-dir contract (its hooks are compiled-in), so the
// settings.json `permissions` object is the single enforcement surface.
//
// SCHEMA (confirmed 2026-06-29 against antigravity.google/docs/cli-permissions +
// the on-disk antigravity-cli settings written by agy 1.0.13): the `permissions`
// object holds three string arrays — `allow`, `deny`, `ask` — evaluated in the
// order Deny > Ask > Allow. Each entry is an `action(target)` rule string, e.g.
// `command(git)`, `command(rm -rf)`, `read_file(/var/log/app)`, `read_file(*)`,
// `mcp(*)`; the global wildcard `*` matches all targets in an action namespace.
// This adapter is vocabulary-agnostic (it stores whatever rule string the caller
// passes), so the three managed arrays map 1:1 onto the native object.
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
	"strings"
)

// Scope selects which Antigravity settings file the adapter targets. Antigravity
// supports a global file and project-scoped grants; only the global (user) scope
// is managed here. Project-scoped grants live in per-workspace project files and
// are a documented follow-up.
type Scope string

// ScopeUser writes the global ~/.gemini/antigravity-cli/settings.json.
const ScopeUser Scope = "user"

const (
	// permissionSectionKey is the native settings.json object Antigravity reads
	// permission grants from.
	permissionSectionKey = "permissions"
	// denyKey/askKey/allowKey are the managed arrays inside the permissions
	// object. See the SCHEMA CONFIRMATION note in the package doc.
	denyKey  = "deny"
	askKey   = "ask"
	allowKey = "allow"
)

// Policy is the canonical in-memory shape of the rule subset the adapter
// manages. Patterns use Antigravity's native `action(target)` rule-string
// vocabulary (`command(git)`, `command(rm -rf)`, `read_file(*)`, `mcp(*)`, ...).
// The field names retain the Bash* prefix for parity with the other
// coding-agent resource adapters, but the stored strings are Antigravity rules.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
}

// Adapter binds the on-disk settings file.
type Adapter struct {
	// SettingsPath is the absolute path of the target settings file
	// (~/.gemini/antigravity-cli/settings.json).
	SettingsPath string
	// Scope is the config scope this adapter targets.
	Scope Scope
}

// DefaultAdapter returns an Adapter rooted at $HOME/.gemini/antigravity-cli with
// the given scope (defaults to ScopeUser).
func DefaultAdapter(scope Scope) (*Adapter, error) {
	if scope == "" {
		scope = ScopeUser
	}
	if scope != ScopeUser {
		return nil, fmt.Errorf("unsupported scope %q (only user/global is managed; project-scoped grants are a follow-up)", scope)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	return &Adapter{
		SettingsPath: filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"),
		Scope:        scope,
	}, nil
}

// Load reads and parses the settings file. A missing file is not an error — it
// resolves to an empty Policy so callers can use Save to create one.
func (a *Adapter) Load() (Policy, error) {
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Policy{}, nil
		}
		return Policy{}, fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return Policy{}, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return Policy{}, fmt.Errorf("parse %s: %w", a.SettingsPath, err)
	}
	perms, _ := doc[permissionSectionKey].(map[string]any)
	return Policy{
		BashDeny:  decodeStringArray(perms, denyKey),
		BashAsk:   decodeStringArray(perms, askKey),
		BashAllow: decodeStringArray(perms, allowKey),
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

// Save writes the policy to disk, preserving every other top-level key and every
// non-managed key inside the `permissions` object.
func (a *Adapter) Save(p Policy) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	doc := map[string]any{}
	if data, err := os.ReadFile(a.SettingsPath); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if uerr := json.Unmarshal(data, &doc); uerr != nil {
			return fmt.Errorf("parse %s: %w", a.SettingsPath, uerr)
		}
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}

	// Rebuild the permissions object preserving any unknown keys.
	section, _ := doc[permissionSectionKey].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}
	for _, key := range []string{denyKey, askKey, allowKey} {
		delete(section, key)
	}
	if len(p.BashDeny) > 0 {
		section[denyKey] = toAnySlice(sortedCopy(p.BashDeny))
	}
	if len(p.BashAsk) > 0 {
		section[askKey] = toAnySlice(sortedCopy(p.BashAsk))
	}
	if len(p.BashAllow) > 0 {
		section[allowKey] = toAnySlice(sortedCopy(p.BashAllow))
	}
	if len(section) == 0 {
		delete(doc, permissionSectionKey)
	} else {
		doc[permissionSectionKey] = section
	}

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", a.SettingsPath, err)
	}
	out = append(out, '\n')
	return atomicWrite(a.SettingsPath, out, 0o644)
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

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
