// Package permissions manages the Claude Code agent's permission config
// at ~/.claude/settings.json.
//
// Scope is narrow on purpose: the adapter owns only the
// `permissions.{allow,ask,deny}` arrays and any `hooks.PreToolUse`
// entry tagged `"managedBy": "vrooli"`. Every other top-level key, and
// every hook entry not tagged as Vrooli-managed, round-trips untouched.
//
// The PreToolUse hook is a native Go matcher paired with every Bash deny rule.
// Native permission rules remain authoritative; the hook receives JSON as data
// and never evaluates the command text.
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

	"github.com/vrooli/agentharness"
)

// ManagedByMarker is the value of the `managedBy` field that flags a
// hook entry as owned by this adapter. Hand-written hook entries that
// omit this marker are preserved across Save calls.
const ManagedByMarker = "vrooli"

// HookStateDirName is the directory beside Claude's settings file that holds
// the hook's audit log.
const HookStateDirName = ".vrooli-hooks"

// GuardCommandEnv overrides the executable Claude invokes for the PreToolUse
// decision. It exists for tests and for operators running an out-of-tree build.
const GuardCommandEnv = "VROOLI_CLAUDE_HOOK_GUARD"

// GuardSubcommand is the verb that performs one PreToolUse decision.
var GuardSubcommand = []string{"permissions", "hook-guard"}

// GuardCommand resolves the executable that evaluates the PreToolUse hook. It
// defaults to the running binary so the installed hook always executes the
// same implementation that wrote it, with no PATH lookup to go stale.
func GuardCommand() string {
	if override := strings.TrimSpace(os.Getenv(GuardCommandEnv)); override != "" {
		return override
	}
	executable, err := os.Executable()
	if err != nil {
		return "resource-claude-code"
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		return resolved
	}
	return executable
}

// Policy is the canonical in-memory shape of the bash-pattern subset of
// the Claude Code permissions file the adapter manages.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
	// Hooks enables PreToolUse hook materialization paired with the
	// BashDeny patterns. Defaults to true on Save; only set false in
	// tests that exercise the no-hook path explicitly.
	Hooks bool
}

// Adapter binds the on-disk settings.json plus the managed hook path beside it.
type Adapter struct {
	// SettingsPath is the absolute path of settings.json (typically
	// ~/.claude/settings.json).
	SettingsPath string
	// HookStateDir is the directory holding the hook's audit log.
	HookStateDir string
}

// HookResult is the stable result returned by the generic hook seam. Hook
// identifiers are scoped to an event and settings file, preserving unrelated
// user-owned hooks.
type HookResult struct {
	Status     string `json:"status"`
	Code       string `json:"code"`
	Reason     string `json:"reason"`
	Event      string `json:"event"`
	Identifier string `json:"identifier"`
	Scope      string `json:"scope"`
	Settings   string `json:"settingsPath"`
}

func (a *Adapter) ReconcileHook(event, identifier string, hook map[string]any) (HookResult, error) {
	result := HookResult{Event: event, Identifier: identifier, Scope: "", Settings: a.SettingsPath}
	brokerResult, err := agentharness.NewHookBroker().Reconcile(
		agentharness.HookTarget{Agent: "claude-code", Path: a.SettingsPath},
		agentharness.HookRegistration{Event: event, ID: identifier, Hook: hook},
	)
	result.Status, result.Code, result.Reason = brokerResult.Status, brokerResult.Code, brokerResult.Reason
	return result, err
}

// RemoveHook removes one hook idempotently while preserving other hooks in
// the same matcher group and event.
func (a *Adapter) RemoveHook(event, identifier string) (HookResult, error) {
	result := HookResult{Event: event, Identifier: identifier, Settings: a.SettingsPath}
	brokerResult, err := agentharness.NewHookBroker().Remove(
		agentharness.HookTarget{Agent: "claude-code", Path: a.SettingsPath},
		event, identifier,
	)
	result.Status, result.Code, result.Reason = brokerResult.Status, brokerResult.Code, brokerResult.Reason
	return result, err
}

// DefaultAdapter returns an Adapter rooted at $HOME/.claude.
func DefaultAdapter() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	return &Adapter{
		SettingsPath: filepath.Join(home, ".claude", "settings.json"),
		HookStateDir: filepath.Join(home, ".claude", HookStateDirName),
	}, nil
}

// HookLogPath is the append-only audit log the PreToolUse guard writes.
func (a *Adapter) HookLogPath() string {
	return filepath.Join(a.HookStateDir, "log")
}

// Load reads and parses the settings file. A missing file is not an
// error — it resolves to an empty Policy so callers can use Save to
// create a new file from scratch.
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
	doc, err := parseDoc(data)
	if err != nil {
		return Policy{}, err
	}
	p := Policy{Hooks: true}
	if doc.Permissions != nil {
		p.BashAllow = decodeStringArray(doc.Permissions.All, "allow")
		p.BashAsk = decodeStringArray(doc.Permissions.All, "ask")
		p.BashDeny = decodeStringArray(doc.Permissions.All, "deny")
	}
	return p, nil
}

func decodeStringArray(m map[string]json.RawMessage, key string) []string {
	raw, ok := m[key]
	if !ok {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	return arr
}

// parsedDoc holds the full top-level structure so Save can rebuild it
// preserving unknown keys.
type parsedDoc struct {
	TopLevel    map[string]json.RawMessage
	Permissions *parsedPermissions
	Hooks       map[string][]json.RawMessage
}

type parsedPermissions struct {
	All map[string]json.RawMessage
}

func parseDoc(data []byte) (*parsedDoc, error) {
	doc := &parsedDoc{
		TopLevel: map[string]json.RawMessage{},
		Hooks:    map[string][]json.RawMessage{},
	}
	if err := json.Unmarshal(data, &doc.TopLevel); err != nil {
		return nil, fmt.Errorf("parse settings.json: %w", err)
	}
	if raw, ok := doc.TopLevel["permissions"]; ok {
		pp := &parsedPermissions{All: map[string]json.RawMessage{}}
		if err := json.Unmarshal(raw, &pp.All); err != nil {
			return nil, fmt.Errorf("parse permissions: %w", err)
		}
		// Type-check the three known sub-fields are arrays.
		for _, key := range []string{"allow", "ask", "deny"} {
			if r, ok := pp.All[key]; ok {
				var arr []string
				if err := json.Unmarshal(r, &arr); err != nil {
					return nil, fmt.Errorf("parse permissions.%s: %w", key, err)
				}
			}
		}
		doc.Permissions = pp
	}
	if raw, ok := doc.TopLevel["hooks"]; ok {
		var hooksMap map[string]json.RawMessage
		if err := json.Unmarshal(raw, &hooksMap); err != nil {
			return nil, fmt.Errorf("parse hooks: %w", err)
		}
		for evt, entries := range hooksMap {
			var arr []json.RawMessage
			if err := json.Unmarshal(entries, &arr); err != nil {
				return nil, fmt.Errorf("parse hooks.%s: %w", evt, err)
			}
			doc.Hooks[evt] = arr
		}
	}
	return doc, nil
}

// Save writes the policy to disk. If Hooks is true and at least one
// BashDeny entry is present, a Vrooli-managed PreToolUse entry is added (or
// updated) in settings.json. Hand-written hook entries and unrelated
// top-level keys are preserved verbatim.
func (a *Adapter) Save(p Policy) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	// Load existing doc to preserve unknown top-level keys and
	// user-owned hook entries. Missing file is fine.
	doc := &parsedDoc{
		TopLevel: map[string]json.RawMessage{},
		Hooks:    map[string][]json.RawMessage{},
	}
	if data, err := os.ReadFile(a.SettingsPath); err == nil && len(data) > 0 {
		parsed, perr := parseDoc(data)
		if perr != nil {
			return perr
		}
		doc = parsed
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %s: %w", a.SettingsPath, err)
	}

	// Rebuild permissions, preserving any unknown sub-keys.
	permsAll := map[string]json.RawMessage{}
	if doc.Permissions != nil {
		for k, v := range doc.Permissions.All {
			if k == "allow" || k == "ask" || k == "deny" {
				continue
			}
			permsAll[k] = v
		}
	}
	if len(p.BashAllow) > 0 {
		if r, err := marshalStrings(p.BashAllow); err == nil {
			permsAll["allow"] = r
		}
	}
	if len(p.BashAsk) > 0 {
		if r, err := marshalStrings(p.BashAsk); err == nil {
			permsAll["ask"] = r
		}
	}
	if len(p.BashDeny) > 0 {
		if r, err := marshalStrings(p.BashDeny); err == nil {
			permsAll["deny"] = r
		}
	}
	if len(permsAll) == 0 {
		delete(doc.TopLevel, "permissions")
	} else {
		encoded, err := marshalOrderedMap(permsAll)
		if err != nil {
			return fmt.Errorf("encode permissions: %w", err)
		}
		doc.TopLevel["permissions"] = encoded
	}

	out, err := marshalOrderedMap(doc.TopLevel)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	pretty, err := prettifyJSON(out)
	if err != nil {
		return err
	}
	tmp := a.SettingsPath + ".tmp"
	broker := agentharness.NewHookBroker()
	if err := broker.WithLock(a.SettingsPath, func() error {
		if err := os.WriteFile(tmp, pretty, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", tmp, err)
		}
		if err := os.Rename(tmp, a.SettingsPath); err != nil {
			return fmt.Errorf("rename %s -> %s: %w", tmp, a.SettingsPath, err)
		}
		return nil
	}); err != nil {
		return err
	}

	hook := buildHookEntry(GuardCommand(), p.BashDeny)
	if p.Hooks && len(p.BashDeny) > 0 {
		inner := hook["hooks"].([]map[string]string)[0]
		_, err = broker.Reconcile(
			agentharness.HookTarget{Agent: "claude-code", Path: a.SettingsPath},
			agentharness.HookRegistration{
				Event: "PreToolUse", ID: "vrooli-policy-runner",
				Group: map[string]any{"matcher": hook["matcher"], "patterns": hook["patterns"]},
				Hook:  map[string]any{"type": inner["type"], "command": inner["command"]},
			},
		)
		return err
	}
	_, err = broker.Remove(
		agentharness.HookTarget{Agent: "claude-code", Path: a.SettingsPath},
		"PreToolUse", "vrooli-policy-runner",
	)
	return err
}

// buildHookEntry constructs the canonical PreToolUse entry. The command invokes
// the native Go matcher with each pattern as a quoted argument, keeping exact
// Claude Bash(...) semantics at the resource edge.
func buildHookEntry(guardCommand string, patterns []string) map[string]any {
	commandParts := append([]string{shellQuote(guardCommand)}, GuardSubcommand...)
	for _, pattern := range patterns {
		commandParts = append(commandParts, shellQuote(pattern))
	}
	return map[string]any{
		"matcher":   "Bash",
		"managedBy": ManagedByMarker,
		"patterns":  append([]string(nil), patterns...),
		"hooks": []map[string]string{
			{
				"type":    "command",
				"command": strings.Join(commandParts, " "),
			},
		},
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// RenderHook returns the hook entry as a JSON object, primarily for
// docs/debugging surfaces. The same shape is what Save writes.
func (a *Adapter) RenderHook(p Policy) map[string]any {
	if !p.Hooks || len(p.BashDeny) == 0 {
		return nil
	}
	return buildHookEntry(GuardCommand(), p.BashDeny)
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

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func marshalStrings(s []string) (json.RawMessage, error) {
	// Preserve insertion order — these are user-meaningful patterns;
	// shuffling them is surprising.
	return json.Marshal(s)
}

// marshalOrderedMap encodes a map[string]json.RawMessage with
// alphabetically sorted keys so output is deterministic.
func marshalOrderedMap(m map[string]json.RawMessage) (json.RawMessage, error) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	buf.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			buf.WriteString(",")
		}
		kJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kJSON)
		buf.WriteString(":")
		buf.Write(m[k])
	}
	buf.WriteString("}")
	return json.RawMessage(buf.String()), nil
}

func prettifyJSON(in json.RawMessage) ([]byte, error) {
	var v any
	if err := json.Unmarshal(in, &v); err != nil {
		return nil, err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	out = append(out, '\n')
	return out, nil
}
