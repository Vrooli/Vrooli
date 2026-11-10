// Package permissions manages Vrooli's Codex permission intent and its native
// projections for $CODEX_HOME (user scope) or the system requirements.toml
// directory (admin scope). Native execution projection supports user scope
// only.
//
// Codex has no native seam for command-pattern rules or for the portable
// execution document, so that intent lives in a Vrooli-owned sidecar beside
// the config (.vrooli-permissions-intent.json, see Intent). config.toml only
// ever receives keys Codex reads: v2 execution documents own the reserved
// `[permissions.vrooli]` profile, and activation selects it and sets the
// approval keys; every other key and table round-trips untouched. Rule-only
// operations do not touch config.toml at all.
//
// Codex versions at and beyond the pinned resource version expose a
// hooks.json PreToolUse command surface. The native sandbox/approval controls
// remain separate; the adapter records both and reports hook firing as
// unverified until a runtime canary proves the installed binary invokes it.
package permissions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

// Policy is the in-memory view of the intent plus fingerprints of its native
// projections.
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
	// LegacyTable reports a retired `[vrooli]` table still in config.toml;
	// the next write or `resource-codex ensure` moves it out.
	LegacyTable bool
}

// Adapter binds the on-disk config path. SettingsPath is selected by
// DefaultAdapter based on Scope.
type Adapter struct {
	SettingsPath string
	HookPath     string
	Scope        Scope
	LastBackup   string
}

// DefaultAdapter honors CODEX_HOME (default $HOME/.codex) with the given
// scope (defaults to ScopeUser). A Web Console session home links its
// config.toml to the shared config; the adapter follows the link so the
// intent, state and hooks live beside the one shared file.
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
	settings, err := agentharness.ResolveNativeFilePath(filepath.Join(configHome, "config.toml"))
	if err != nil {
		return nil, err
	}
	return &Adapter{SettingsPath: settings, HookPath: filepath.Join(filepath.Dir(settings), "hooks.json"), Scope: scope}, nil
}

// snapshots are the exact native and sidecar bytes one operation works from.
type snapshots struct {
	config, hooks, intent agentharness.PermissionFileSnapshot
	doc                   map[string]any
	recorded              Intent
	legacy                bool
}

func (a *Adapter) readSnapshots() (snapshots, error) {
	var s snapshots
	var err error
	if s.config, err = agentharness.ReadPermissionFile(a.SettingsPath); err != nil {
		return s, err
	}
	if s.hooks, err = a.hookSnapshot(); err != nil {
		return s, err
	}
	if s.intent, err = agentharness.ReadPermissionFile(a.IntentPath()); err != nil {
		return s, err
	}
	s.doc = map[string]any{}
	if len(s.config.Data) > 0 {
		if err = toml.Unmarshal(s.config.Data, &s.doc); err != nil {
			return s, fmt.Errorf("parse %s: %w", a.SettingsPath, err)
		}
	}
	legacy, hasLegacy, err := legacyIntent(s.doc)
	if err != nil {
		return s, err
	}
	s.legacy = hasLegacy
	if s.recorded, err = decodeIntent(s.intent); err != nil {
		return s, err
	}
	if !s.intent.Exists && hasLegacy {
		s.recorded = legacy
	}
	delete(s.doc, legacyTableKey)
	return s, nil
}

func (s snapshots) digest() string {
	return agentharness.PermissionFilesDigest(s.config, s.hooks, s.intent)
}

// Load reads the intent and fingerprints its native projections. Missing
// files are not errors: they resolve to an empty Policy.
func (a *Adapter) Load() (Policy, error) {
	s, err := a.readSnapshots()
	if err != nil {
		return Policy{}, err
	}
	p := Policy{BashDeny: s.recorded.BashDeny, BashAsk: s.recorded.BashAsk, BashAllow: s.recorded.BashAllow, Execution: s.recorded.Execution, ExecutionActive: s.recorded.ExecutionActive, LegacyTable: s.legacy}
	if p.Execution != nil {
		p.NativeFingerprint = executionFingerprint(s.doc, p.ExecutionActive)
	}
	if p.HookFingerprint, err = a.hookFingerprint(s.hooks.Data); err != nil {
		return Policy{}, err
	}
	p.SnapshotDigest = s.digest()
	return p, nil
}

// Save records the policy and reconciles its native projections. A nil
// Execution keeps the recorded execution intent.
func (a *Adapter) Save(p Policy) error {
	return agentharness.NewHookBroker().WithLock(a.SettingsPath, func() error { return a.saveLocked(p) })
}

func (a *Adapter) saveLocked(p Policy) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}
	s, err := a.readSnapshots()
	if err != nil {
		return err
	}
	if p.SnapshotDigest != "" && p.SnapshotDigest != s.digest() {
		return errors.New("native files changed since preview; plan again")
	}
	next := Intent{SettingsPath: a.SettingsPath, BashDeny: p.BashDeny, BashAsk: p.BashAsk, BashAllow: p.BashAllow, Execution: s.recorded.Execution, ExecutionActive: s.recorded.ExecutionActive}
	config := s.config.Data
	if s.legacy {
		if config, err = stripLegacyTable(config); err != nil {
			return fmt.Errorf("%s: %w", a.SettingsPath, err)
		}
	}
	if p.Execution != nil {
		if a.Scope == ScopeAdmin {
			return errors.New("native execution projection supports user scope only")
		}
		if err := agentharness.ValidateExecutionPermissions(p.Execution); err != nil {
			return err
		}
		unchanged := map[string]any{}
		if err := toml.Unmarshal(config, &unchanged); err != nil {
			return err
		}
		if err := projectExecution(s.doc, p.Execution, p.ExecutionActive, s.recorded.Execution != nil); err != nil {
			return err
		}
		// Re-encoding drops comments and key order, so rewrite only on change.
		if !reflect.DeepEqual(s.doc, unchanged) {
			if config, err = toml.Marshal(s.doc); err != nil {
				return fmt.Errorf("encode %s: %w", a.SettingsPath, err)
			}
		}
		next.Execution, next.ExecutionActive = p.Execution, p.ExecutionActive
	}
	intent, err := encodeIntent(next)
	if err != nil {
		return err
	}
	a.LastBackup = ""
	configChanged := !bytes.Equal(config, s.config.Data)
	if configChanged {
		if a.LastBackup, err = agentharness.BackupNativeFile(s.config, "permissions"); err != nil {
			return err
		}
		if err = agentharness.PublishPermissionFile(s.config, config); err != nil {
			return err
		}
	}
	rollbackConfig := func() error {
		if !configChanged {
			return nil
		}
		return restoreFile(s.config, config)
	}
	if err = agentharness.PublishPermissionFile(s.intent, intent); err != nil {
		return fmt.Errorf("intent publication failed: %w; config rollback: %v; backup: %s", err, rollbackConfig(), a.LastBackup)
	}
	if a.HookPath != "" {
		if err := a.savePreToolHook(p); err != nil {
			intentErr := restoreFile(s.intent, intent)
			return fmt.Errorf("hook reconciliation failed: %w; config rollback: %v; intent rollback: %v; backup: %s; hook migration may have applied, inspect and re-plan", err, rollbackConfig(), intentErr, a.LastBackup)
		}
	}
	return nil
}

// restoreFile puts back the pre-image of a file this operation wrote, unless
// someone changed it since.
func restoreFile(before agentharness.PermissionFileSnapshot, written []byte) error {
	current, err := agentharness.ReadPermissionFile(before.Path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current.Data, written) {
		return errors.New("file changed after publication; rollback refused")
	}
	if before.Exists {
		return agentharness.PublishPermissionFile(current, before.Data)
	}
	return os.Remove(before.Path)
}

// MigrateLegacyTable moves a retired `[vrooli]` table out of config.toml into
// the intent sidecar. An existing sidecar is authoritative, so the table is
// then only removed. It changes nothing else: hooks and state are left for
// the next reconcile. It reports what it did; empty means nothing to move.
func (a *Adapter) MigrateLegacyTable() (string, error) {
	var report string
	err := agentharness.NewHookBroker().WithLock(a.SettingsPath, func() error {
		s, err := a.readSnapshots()
		if err != nil || !s.legacy {
			return err
		}
		stripped, err := stripLegacyTable(s.config.Data)
		if err != nil {
			return fmt.Errorf("%s: %w", a.SettingsPath, err)
		}
		action := "removed the legacy [vrooli] table; " + a.IntentPath() + " already holds the intent"
		if !s.intent.Exists {
			recorded := s.recorded
			recorded.SettingsPath = a.SettingsPath
			intent, err := encodeIntent(recorded)
			if err != nil {
				return err
			}
			// Intent first: if the config write then fails, the sidecar already
			// wins and a later run only removes the table.
			if err = agentharness.PublishPermissionFile(s.intent, intent); err != nil {
				return err
			}
			action = "moved the legacy [vrooli] table to " + a.IntentPath()
		}
		backup, err := agentharness.BackupNativeFile(s.config, "permissions")
		if err != nil {
			return err
		}
		if err = agentharness.PublishPermissionFile(s.config, stripped); err != nil {
			return err
		}
		report = fmt.Sprintf("%s: %s (backup: %s)", a.SettingsPath, action, backup)
		return nil
	})
	return report, err
}

// permissionHookMatcher selects the Codex tools the deny-rule hook guards.
// Codex 0.167 names the shell tool "Bash" in a hook payload (the earlier
// "shell_command" and "exec_command" names are kept for older builds), and an
// MCP tool "mcp__<server>__<tool>". A matcher naming only the older shell
// names never fires for a shell command (verified with a throwaway CODEX_HOME).
const permissionHookMatcher = "Bash|shell_command|exec_command|apply_patch|mcp__.*"

// RenderHook returns the native Codex PreToolUse projection. The command is
// the installed policy runner by absolute path (see PolicyRunnerCommand). Ownership
// markers go on the hook entry only: Codex rejects the whole hooks.json for an
// unknown top-level key.
func (a *Adapter) RenderHook(p Policy) map[string]any {
	if len(p.BashDeny) == 0 {
		return nil
	}
	return map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{map[string]any{
				"matcher": permissionHookMatcher,
				"hooks":   []any{map[string]any{"type": "command", "command": PolicyRunnerCommand() + " hook --runner codex", "timeout": defaultHookTimeoutSeconds}},
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
