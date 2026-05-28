// Package permissions manages the Claude Code agent's permission config
// at ~/.claude/settings.json.
//
// Scope is narrow on purpose: the adapter owns only the
// `permissions.{allow,ask,deny}` arrays and any `hooks.PreToolUse`
// entry tagged `"managedBy": "vrooli"`. Every other top-level key, and
// every hook entry not tagged as Vrooli-managed, round-trips untouched.
//
// The PreToolUse hook is paired with every Bash deny rule as a defensive
// backstop against the upstream `permissions.deny` enforcement bug
// (anthropics/claude-code#18846, #29026). The hook script itself is
// embedded into the resource binary and materialized into
// ~/.claude/.vrooli-hooks/ on first Save so settings.json only contains a
// path reference.
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
)

// ManagedByMarker is the value of the `managedBy` field that flags a
// hook entry as owned by this adapter. Hand-written hook entries that
// omit this marker are preserved across Save calls.
const ManagedByMarker = "vrooli"

// HookScriptName is the file name materialized under HookScriptDir.
const HookScriptName = "pretooluse-bash-deny.sh"

//go:embed pretooluse-bash-deny.sh
var embeddedHookScript []byte

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

// Adapter binds the on-disk settings.json plus the hook-script
// materialization directory.
type Adapter struct {
	// SettingsPath is the absolute path of settings.json (typically
	// ~/.claude/settings.json).
	SettingsPath string
	// HookScriptDir is the directory the hook script is materialized
	// into (typically ~/.claude/.vrooli-hooks). Save creates it.
	HookScriptDir string
}

// DefaultAdapter returns an Adapter rooted at $HOME/.claude.
func DefaultAdapter() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	return &Adapter{
		SettingsPath:  filepath.Join(home, ".claude", "settings.json"),
		HookScriptDir: filepath.Join(home, ".claude", ".vrooli-hooks"),
	}, nil
}

// HookScriptPath is the absolute path of the materialized hook script.
func (a *Adapter) HookScriptPath() string {
	return filepath.Join(a.HookScriptDir, HookScriptName)
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
// BashDeny entry is present, the hook script is materialized into
// HookScriptDir and a Vrooli-managed PreToolUse entry is added (or
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

	// Rebuild PreToolUse: drop the existing managedBy=vrooli entry,
	// then re-add a fresh one if Hooks && BashDeny non-empty.
	preToolUse := doc.Hooks["PreToolUse"]
	filtered := make([]json.RawMessage, 0, len(preToolUse))
	for _, raw := range preToolUse {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			filtered = append(filtered, raw)
			continue
		}
		if mb, ok := probe["managedBy"]; ok {
			var s string
			if json.Unmarshal(mb, &s) == nil && s == ManagedByMarker {
				continue
			}
		}
		filtered = append(filtered, raw)
	}
	if p.Hooks && len(p.BashDeny) > 0 {
		if err := a.materializeHookScript(); err != nil {
			return err
		}
		entry := buildHookEntry(a.HookScriptPath(), p.BashDeny)
		raw, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("encode hook entry: %w", err)
		}
		filtered = append(filtered, raw)
	}
	if len(filtered) > 0 {
		doc.Hooks["PreToolUse"] = filtered
	} else {
		delete(doc.Hooks, "PreToolUse")
	}

	if len(doc.Hooks) == 0 {
		delete(doc.TopLevel, "hooks")
	} else {
		hooksTopRaw := map[string]json.RawMessage{}
		for evt, arr := range doc.Hooks {
			encoded, err := json.Marshal(arr)
			if err != nil {
				return fmt.Errorf("encode hooks.%s: %w", evt, err)
			}
			hooksTopRaw[evt] = encoded
		}
		encoded, err := marshalOrderedMap(hooksTopRaw)
		if err != nil {
			return fmt.Errorf("encode hooks: %w", err)
		}
		doc.TopLevel["hooks"] = encoded
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
	if err := os.WriteFile(tmp, pretty, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, a.SettingsPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmp, a.SettingsPath, err)
	}
	return nil
}

func (a *Adapter) materializeHookScript() error {
	if err := os.MkdirAll(a.HookScriptDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", a.HookScriptDir, err)
	}
	path := a.HookScriptPath()
	// Always overwrite so upgrades propagate.
	if err := os.WriteFile(path, embeddedHookScript, 0o755); err != nil {
		return fmt.Errorf("write hook script %s: %w", path, err)
	}
	return nil
}

// buildHookEntry constructs the canonical PreToolUse entry. The command
// invokes the materialized script with the deny patterns as arguments;
// the script does the actual glob match and exit-2 refusal.
func buildHookEntry(scriptPath string, patterns []string) map[string]any {
	args := make([]string, 0, len(patterns))
	for _, p := range patterns {
		args = append(args, shellQuote(p))
	}
	command := scriptPath
	if len(args) > 0 {
		command = scriptPath + " " + strings.Join(args, " ")
	}
	return map[string]any{
		"matcher":   "Bash",
		"managedBy": ManagedByMarker,
		"patterns":  append([]string(nil), patterns...),
		"hooks": []map[string]string{
			{
				"type":    "command",
				"command": command,
			},
		},
	}
}

// RenderHook returns the hook entry as a JSON object, primarily for
// docs/debugging surfaces. The same shape is what Save writes.
func (a *Adapter) RenderHook(p Policy) map[string]any {
	if !p.Hooks || len(p.BashDeny) == 0 {
		return nil
	}
	return buildHookEntry(a.HookScriptPath(), p.BashDeny)
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

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`*?[]&|;<>()") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
