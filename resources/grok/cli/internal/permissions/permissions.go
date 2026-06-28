// Package permissions manages the Grok Build agent's permission config at
// ~/.grok/config.toml (user scope) or ~/.grok/requirements.toml (admin
// scope), plus a paired PreToolUse hard-deny hook under ~/.grok/hooks/.
//
// Scope is narrow on purpose: the adapter owns only the three string
// arrays `deny`, `ask`, `allow` inside the native `[permission]` table,
// and the Vrooli-named hook files it materializes. Every other top-level
// key/table (and every other key inside `[permission]`, e.g. `rules`)
// round-trips untouched.
//
// Unlike Codex (whose `[vrooli.permissions]` block is intent-only),
// Grok ENFORCES these rules natively: the decision flow evaluates
// `deny > ask > allow` from the native `[permission]` section before
// falling through to prompt policy (see ~/.grok/docs/user-guide/
// 22-permissions-and-safety.md). The paired PreToolUse hook is a
// belt-and-suspenders backstop that hard-denies matching Bash commands
// before any other check and applies even under `--always-approve`.
package permissions

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Scope selects which Grok config file the adapter targets.
type Scope string

const (
	// ScopeUser writes ~/.grok/config.toml.
	ScopeUser Scope = "user"
	// ScopeAdmin writes ~/.grok/requirements.toml (higher precedence in
	// Grok's config merge).
	ScopeAdmin Scope = "admin"
)

// permissionSectionKey is the native TOML table Grok reads rules from.
const permissionSectionKey = "permission"

// HookScriptName is the shared backstop script materialized under HooksDir.
const HookScriptName = "vrooli-bash-deny.sh"

//go:embed pretooluse-bash-deny.sh
var embeddedHookScript []byte

// Policy is the canonical in-memory shape of the rule subset the adapter
// manages. Patterns use the Claude/Grok rule-string vocabulary
// (`Bash(git *)`, `Read`, `Edit(**/*.rs)`, `Grep`, ...), which the native
// `[permission]` section accepts verbatim and which is uniform across all
// coding-agent resources.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
	// Hooks enables PreToolUse hook materialization paired with the
	// BashDeny patterns. Defaults to true on Load; only set false in
	// tests that exercise the no-hook path explicitly.
	Hooks bool
}

// Adapter binds the on-disk config file plus the hook-materialization
// directory.
type Adapter struct {
	// SettingsPath is the absolute path of the target config file
	// (~/.grok/config.toml or ~/.grok/requirements.toml).
	SettingsPath string
	// HooksDir is the directory hook files are materialized into
	// (~/.grok/hooks). Save creates it when needed.
	HooksDir string
	// Scope is the config scope this adapter targets.
	Scope Scope
}

// DefaultAdapter returns an Adapter rooted at $HOME/.grok with the given
// scope (defaults to ScopeUser).
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
		SettingsPath: filepath.Join(home, ".grok", file),
		HooksDir:     filepath.Join(home, ".grok", "hooks"),
		Scope:        scope,
	}, nil
}

// HookScriptPath is the absolute path of the materialized shared hook
// script.
func (a *Adapter) HookScriptPath() string {
	return filepath.Join(a.HooksDir, HookScriptName)
}

// HookConfigPath is the absolute path of this scope's PreToolUse hook
// JSON. Admin scope gets a distinct filename so the two scopes never
// clobber each other's backstop.
func (a *Adapter) HookConfigPath() string {
	name := "vrooli-bash-deny.json"
	if a.Scope == ScopeAdmin {
		name = "vrooli-bash-deny.admin.json"
	}
	return filepath.Join(a.HooksDir, name)
}

// Load reads and parses the config file. A missing file is not an error —
// it resolves to an empty Policy so callers can use Save to create one.
func (a *Adapter) Load() (Policy, error) {
	data, err := os.ReadFile(a.SettingsPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Policy{Hooks: true}, nil
		}
		return Policy{}, fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}
	if len(data) == 0 {
		return Policy{Hooks: true}, nil
	}
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return Policy{}, fmt.Errorf("parse %s: %w", a.SettingsPath, err)
	}
	perms, _ := doc[permissionSectionKey].(map[string]any)
	return Policy{
		BashDeny:  decodeStringArray(perms, "deny"),
		BashAsk:   decodeStringArray(perms, "ask"),
		BashAllow: decodeStringArray(perms, "allow"),
		Hooks:     true,
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

// Save writes the policy to disk, preserving every other top-level key
// and every non-managed key inside `[permission]`. When Hooks is true and
// at least one Bash(...) deny pattern is present, the PreToolUse backstop
// hook is materialized; otherwise this scope's hook JSON is removed.
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

	// Rebuild [permission] preserving any unknown keys (e.g. `rules`).
	section, _ := doc[permissionSectionKey].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}
	for _, key := range []string{"deny", "ask", "allow"} {
		delete(section, key)
	}
	if len(p.BashDeny) > 0 {
		section["deny"] = toAnySlice(sortedCopy(p.BashDeny))
	}
	if len(p.BashAsk) > 0 {
		section["ask"] = toAnySlice(sortedCopy(p.BashAsk))
	}
	if len(p.BashAllow) > 0 {
		section["allow"] = toAnySlice(sortedCopy(p.BashAllow))
	}
	if len(section) == 0 {
		delete(doc, permissionSectionKey)
	} else {
		doc[permissionSectionKey] = section
	}

	out, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode %s: %w", a.SettingsPath, err)
	}
	if err := atomicWrite(a.SettingsPath, out, 0o644); err != nil {
		return err
	}

	return a.syncHook(p)
}

// syncHook materializes (or removes) this scope's PreToolUse backstop.
func (a *Adapter) syncHook(p Policy) error {
	globs := bashDenyGlobs(p.BashDeny)
	if !p.Hooks || len(globs) == 0 {
		// Remove this scope's hook JSON; leave the shared script (it is
		// harmless and may be referenced by the other scope).
		if err := os.Remove(a.HookConfigPath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove hook %s: %w", a.HookConfigPath(), err)
		}
		return nil
	}
	if err := os.MkdirAll(a.HooksDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", a.HooksDir, err)
	}
	// Always (re)write the shared script so upgrades propagate.
	if err := os.WriteFile(a.HookScriptPath(), embeddedHookScript, 0o755); err != nil {
		return fmt.Errorf("write hook script %s: %w", a.HookScriptPath(), err)
	}
	entry, err := json.MarshalIndent(buildHookConfig(a.HookScriptPath(), globs), "", "  ")
	if err != nil {
		return fmt.Errorf("encode hook config: %w", err)
	}
	entry = append(entry, '\n')
	if err := atomicWrite(a.HookConfigPath(), entry, 0o644); err != nil {
		return err
	}
	return nil
}

// buildHookConfig constructs the Grok-native PreToolUse hook JSON. The
// command invokes the shared script with the deny globs as quoted args;
// the script does the glob match and emits an explicit deny decision.
func buildHookConfig(scriptPath string, globs []string) map[string]any {
	quoted := make([]string, 0, len(globs))
	for _, g := range globs {
		quoted = append(quoted, shellQuote(g))
	}
	command := scriptPath
	if len(quoted) > 0 {
		command = scriptPath + " " + strings.Join(quoted, " ")
	}
	return map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{
							"type":    "command",
							"command": command,
							"timeout": 5,
						},
					},
				},
			},
		},
	}
}

// RenderHook returns the hook JSON, primarily for docs/debugging. The
// same shape is what Save writes.
func (a *Adapter) RenderHook(p Policy) map[string]any {
	globs := bashDenyGlobs(p.BashDeny)
	if !p.Hooks || len(globs) == 0 {
		return nil
	}
	return buildHookConfig(a.HookScriptPath(), globs)
}

// bashDenyGlobs extracts the inner glob of each `Bash(<glob>)` deny
// pattern. Non-Bash patterns (`Read(...)`, `Grep`, `Edit(...)`) are left
// to the native `[permission]` enforcement and skipped here, since the
// hook matcher only fires for the Bash tool.
func bashDenyGlobs(deny []string) []string {
	out := make([]string, 0, len(deny))
	for _, d := range deny {
		d = strings.TrimSpace(d)
		if inner, ok := strings.CutPrefix(d, "Bash("); ok {
			if glob, ok := strings.CutSuffix(inner, ")"); ok && glob != "" {
				out = append(out, glob)
			}
		}
	}
	return out
}

// Fingerprint returns the sha256 hex of the canonical Policy projection.
// drift-check compares this against the state-file value.
func Fingerprint(p Policy) string {
	canon := struct {
		Allow []string `json:"allow"`
		Ask   []string `json:"ask"`
		Deny  []string `json:"deny"`
		Hooks bool     `json:"hooks"`
	}{
		Allow: sortedCopy(p.BashAllow),
		Ask:   sortedCopy(p.BashAsk),
		Deny:  sortedCopy(p.BashDeny),
		Hooks: p.Hooks,
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

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`*?[]&|;<>()") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
