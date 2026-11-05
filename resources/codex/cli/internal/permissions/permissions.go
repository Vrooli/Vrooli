// Package permissions manages the Codex agent's permission config at
// $CODEX_HOME/config.toml (user scope) or the system requirements.toml
// (admin scope). Native execution projection supports user scope only.
//
// Scope is narrow on purpose: the adapter owns only a Vrooli-namespaced
// section, `[vrooli.permissions]`, containing three string arrays —
// `bash_deny`, `bash_ask`, `bash_allow`. Every other top-level key and
// section round-trips untouched for v1 rule-only operations. V2 execution
// documents additionally own the reserved native profile; activation is explicit.
//
// Codex versions at and beyond the pinned resource version expose a
// hooks.json PreToolUse command surface. The native sandbox/approval controls
// remain separate; the adapter records both and reports hook firing as
// unverified until a runtime canary proves the installed binary invokes it.
package permissions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/vrooli/agentharness"
)

// Scope selects which Codex config file the adapter targets.
type Scope string

const (
	// ScopeUser writes ~/.codex/config.toml.
	ScopeUser = "user"
	// ScopeAdmin targets the system requirements.toml.
	ScopeAdmin = "admin"
)

// vrooliSectionKey is the top-level TOML table the adapter owns.
const vrooliSectionKey = "vrooli"

// Policy is the canonical in-memory shape of the bash-pattern subset of
// the Codex permission config the adapter manages.
type Policy struct {
	BashDeny            []string
	BashAsk             []string
	BashAllow           []string
	Execution           *agentharness.ExecutionPermissions
	ExecutionActive     bool
	NativeFingerprint   string
	HookFingerprint     string
	SnapshotDigest      string
	StateSnapshotDigest string
}

// Adapter binds the on-disk config path. SettingsPath is selected by
// DefaultAdapter based on Scope.
type Adapter struct {
	SettingsPath string
	HookPath     string
	Scope        Scope
	LastBackup   string
}

// DefaultAdapter honors CODEX_HOME (default $HOME/.codex) with the
// given scope (defaults to ScopeUser).
func DefaultAdapter(scope Scope) (*Adapter, error) {
	if scope == "" {
		scope = ScopeUser
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home: %w", err)
	}
	configHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if configHome == "" {
		configHome = filepath.Join(home, ".codex")
	}
	configHome, err = filepath.Abs(configHome)
	if err != nil {
		return nil, err
	}
	if scope == ScopeAdmin {
		// User files cannot masquerade as administrator-enforced requirements.
		path := "/etc/codex/requirements.toml"
		if runtime.GOOS == "windows" {
			base := os.Getenv("ProgramData")
			if base == "" {
				return nil, errors.New("ProgramData is required for admin requirements")
			}
			path = filepath.Join(base, "OpenAI", "Codex", "requirements.toml")
		}
		return &Adapter{SettingsPath: path, Scope: scope}, nil
	}
	if scope != ScopeUser {
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}
	return &Adapter{SettingsPath: filepath.Join(configHome, "config.toml"), HookPath: filepath.Join(configHome, "hooks.json"), Scope: scope}, nil
}

// Load reads and parses the config file. A missing file is not an
// error — it resolves to an empty Policy so callers can use Save to
// create a new file.
func (a *Adapter) Load() (Policy, error) {
	snapshot, err := agentharness.ReadPermissionFile(a.SettingsPath)
	if err != nil {
		return Policy{}, err
	}
	doc := map[string]any{}
	if len(snapshot.Data) > 0 {
		if err := toml.Unmarshal(snapshot.Data, &doc); err != nil {
			return Policy{}, fmt.Errorf("parse %s: %w", a.SettingsPath, err)
		}
	}
	section, _ := doc[vrooliSectionKey].(map[string]any)
	perms, _ := section["permissions"].(map[string]any)
	p := Policy{BashDeny: decodeStringArray(perms, "bash_deny"), BashAsk: decodeStringArray(perms, "bash_ask"), BashAllow: decodeStringArray(perms, "bash_allow")}
	if raw, ok := section["execution"]; ok {
		encoded, err := json.Marshal(raw)
		if err != nil {
			return Policy{}, err
		}
		if err = json.Unmarshal(encoded, &p.Execution); err != nil {
			return Policy{}, err
		}
		if err = agentharness.ValidateExecutionPermissions(p.Execution); err != nil {
			return Policy{}, fmt.Errorf("invalid recorded execution intent: %w", err)
		}
		p.ExecutionActive, _ = section["execution_active"].(bool)
		p.NativeFingerprint = executionFingerprint(doc, p.ExecutionActive)
	}
	hooks, err := a.hookSnapshot()
	if err != nil {
		return Policy{}, err
	}
	p.HookFingerprint, err = a.hookFingerprint(hooks.Data)
	if err != nil {
		return Policy{}, err
	}
	p.SnapshotDigest = agentharness.PermissionFilesDigest(snapshot, hooks)
	return p, nil
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
	return agentharness.NewHookBroker().WithLock(a.SettingsPath, func() error { return a.saveLocked(p) })
}

func (a *Adapter) saveLocked(p Policy) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	before, err := agentharness.ReadPermissionFile(a.SettingsPath)
	if err != nil {
		return err
	}
	hooks, err := a.hookSnapshot()
	if err != nil {
		return err
	}
	if p.SnapshotDigest != "" && p.SnapshotDigest != agentharness.PermissionFilesDigest(before, hooks) {
		return errors.New("native files changed since preview; plan again")
	}
	doc := map[string]any{}
	if len(before.Data) > 0 {
		if err := toml.Unmarshal(before.Data, &doc); err != nil {
			return err
		}
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

	if p.Execution != nil {
		if a.Scope == ScopeAdmin {
			return errors.New("native execution projection supports user scope only")
		}
		if err := agentharness.ValidateExecutionPermissions(p.Execution); err != nil {
			return err
		}
		if err := projectExecution(doc, p.Execution, p.ExecutionActive); err != nil {
			return err
		}
	}
	out, err := toml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode %s: %w", a.SettingsPath, err)
	}
	backup := ""
	if before.Exists && string(before.Data) != string(out) {
		backup, err = retainBackup(before)
		if err != nil {
			return err
		}
	}
	a.LastBackup = backup
	if err := agentharness.PublishPermissionFile(before, out); err != nil {
		return err
	}
	if a.HookPath != "" {
		if err := a.savePreToolHook(p); err != nil {
			current, readErr := agentharness.ReadPermissionFile(a.SettingsPath)
			if readErr == nil && string(current.Data) == string(out) {
				if before.Exists {
					readErr = agentharness.PublishPermissionFile(current, before.Data)
				} else {
					readErr = os.Remove(a.SettingsPath)
				}
			} else if readErr == nil {
				readErr = errors.New("config changed after publication; rollback refused")
			}
			return fmt.Errorf("hook reconciliation failed: %w; config rollback: %v; backup: %s; hook migration may have applied, inspect and re-plan", err, readErr, backup)
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
	broker := agentharness.NewHookBroker()
	target := agentharness.HookTarget{Agent: "codex", Path: a.HookPath}
	if _, err := broker.Migrate(target); err != nil {
		return err
	}
	if hook == nil {
		_, err := broker.Remove(target, "PreToolUse", "vrooli-policy-runner")
		return err
	}
	groups := hook["hooks"].(map[string]any)
	entries := groups["PreToolUse"].([]any)
	group := entries[0].(map[string]any)
	inner := group["hooks"].([]any)[0].(map[string]any)
	matcher, _ := group["matcher"].(string)
	_, err := broker.Reconcile(target, agentharness.HookRegistration{
		Event: "PreToolUse", ID: "vrooli-policy-runner", Matcher: matcher, Hook: inner,
	})
	return err
}

// Fingerprint returns the sha256 hex of the canonical Policy projection.
// drift-check compares this against the state-file value.
func Fingerprint(p Policy) string {
	canon := struct {
		Allow  []string `json:"allow"`
		Ask    []string `json:"ask"`
		Deny   []string `json:"deny"`
		Native string   `json:"native,omitempty"`
		Hook   string   `json:"hook,omitempty"`
	}{
		Native: p.NativeFingerprint, Hook: p.HookFingerprint,
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
