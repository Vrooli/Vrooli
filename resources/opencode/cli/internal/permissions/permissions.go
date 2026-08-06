// Package permissions manages the OpenCode agent's permission config at
// ~/.config/opencode/opencode.json.
//
// Scope is narrow on purpose: the adapter owns only the
// `permission.bash` map (string→"allow"|"ask"|"deny") and only the
// entries it previously wrote. Every other top-level key, every other
// `permission.*` key (e.g. `edit`), and every `permission.bash` entry
// not previously tagged Vrooli-managed round-trips untouched.
//
// CRITICAL — opencode.json carries ONLY opencode-schema-valid keys.
// opencode rejects any unknown top-level key ("Configuration is invalid …
// Unrecognized key"), so the list of which bash patterns this adapter
// manages is NOT stored inline. It lives in the sidecar state file
// (.vrooli-permissions-state.json, see state.go), which is the source of
// truth for "which entries are managed"; hand-written entries are detected
// by their absence from it. A retired pre-1.0 build wrote an inline
// `x-vrooli-managed-permissions` key — that is the bug this design fixes;
// it is now migrated into the sidecar on read and stripped on write.
//
// OpenCode honours the config directly per upstream docs (last-match-wins),
// and the adapter also projects a Vrooli plugin using the installed
// tool.execute.before seam. The plugin's live firing still requires a canary;
// configuration presence alone is not reported as verified enforcement. The adapter
// writes alphabetically-sorted keys so output is deterministic; under
// alphabetical order `*` (0x2A) sorts before letters, so specific
// patterns end up later and win the last-match-wins evaluation. Users
// needing different ordering can hand-edit; drift-check will surface
// it.
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

// legacyManagedKey is the retired pre-1.0 inline top-level key that once
// recorded the managed bash patterns directly in opencode.json. opencode
// rejects it as an unrecognized key, so the adapter no longer writes it: on
// read it is migrated into the sidecar state file, and on write it is
// stripped from opencode.json.
const legacyManagedKey = "x-vrooli-managed-permissions"

// Policy is the canonical in-memory shape of the bash-pattern subset of
// the OpenCode permission file the adapter manages.
type Policy struct {
	BashDeny  []string
	BashAsk   []string
	BashAllow []string
}

// Adapter binds the on-disk opencode.json path.
type Adapter struct {
	// SettingsPath is the absolute path of opencode.json (typically
	// ~/.config/opencode/opencode.json).
	SettingsPath string
	// PluginPath is the OpenCode plugin projection path. An empty value keeps
	// the adapter focused on settings only, which is useful for isolated tests.
	PluginPath string
}

// DefaultAdapter returns an Adapter rooted at $HOME/.config/opencode.
func DefaultAdapter() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	return &Adapter{
		SettingsPath: filepath.Join(home, ".config", "opencode", "opencode.json"),
		PluginPath:   filepath.Join(home, ".config", "opencode", "plugins", "vrooli-policy.js"),
	}, nil
}

// parsedDoc holds the full top-level structure so Save can rebuild it
// preserving unknown keys and unmanaged bash entries.
type parsedDoc struct {
	TopLevel      map[string]json.RawMessage
	BashMap       map[string]string
	OtherPermKeys map[string]json.RawMessage
	// LegacyManaged holds the value of the retired inline managed-list key
	// when an old opencode.json still carries it, so Load/Save can migrate
	// it into the sidecar. The key itself is stripped from TopLevel during
	// parsing so it never round-trips back into the (schema-validated) file.
	LegacyManaged []string
}

func parseDoc(data []byte) (*parsedDoc, error) {
	doc := &parsedDoc{
		TopLevel:      map[string]json.RawMessage{},
		BashMap:       map[string]string{},
		OtherPermKeys: map[string]json.RawMessage{},
	}
	if err := json.Unmarshal(data, &doc.TopLevel); err != nil {
		return nil, fmt.Errorf("parse opencode.json: %w", err)
	}
	if raw, ok := doc.TopLevel["permission"]; ok {
		var permAll map[string]json.RawMessage
		if err := json.Unmarshal(raw, &permAll); err != nil {
			return nil, fmt.Errorf("parse permission: %w", err)
		}
		for k, v := range permAll {
			if k == "bash" {
				// Bash may be either a default-all string ("ask") or a
				// map. Only the map form has manageable entries.
				var asMap map[string]string
				if err := json.Unmarshal(v, &asMap); err == nil {
					doc.BashMap = asMap
					continue
				}
				// Default-all form: round-trip under OtherPermKeys so
				// it survives Save.
				doc.OtherPermKeys["bash"] = v
				continue
			}
			doc.OtherPermKeys[k] = v
		}
	}
	// Migrate + strip the retired inline managed-list key. Capturing its
	// value lets managedSet fall back to it once (until the sidecar is
	// written); deleting it from TopLevel guarantees Save never re-emits the
	// schema-invalid key.
	if raw, ok := doc.TopLevel[legacyManagedKey]; ok {
		var lst []string
		if err := json.Unmarshal(raw, &lst); err == nil {
			doc.LegacyManaged = lst
		}
		delete(doc.TopLevel, legacyManagedKey)
	}
	return doc, nil
}

// managedSet returns the set of bash patterns this adapter owns. The sidecar
// state file is the source of truth; the retired inline key is consulted only
// as a one-time migration fallback for configs written before the sidecar
// carried the managed list.
func (a *Adapter) managedSet(doc *parsedDoc) (map[string]struct{}, error) {
	st, err := a.LoadState()
	if err != nil {
		return nil, err
	}
	var managed []string
	switch {
	case st != nil && len(st.ManagedBash) > 0:
		managed = st.ManagedBash
	case doc != nil:
		managed = doc.LegacyManaged
	}
	out := make(map[string]struct{}, len(managed))
	for _, k := range managed {
		out[k] = struct{}{}
	}
	return out, nil
}

// Load reads and parses the opencode.json file. A missing file resolves
// to an empty Policy so callers can use Save to create a new file.
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
	doc, err := parseDoc(data)
	if err != nil {
		return Policy{}, err
	}
	managed, err := a.managedSet(doc)
	if err != nil {
		return Policy{}, err
	}
	p := Policy{}
	for pat, action := range doc.BashMap {
		// Only surface entries we previously wrote — hand-written
		// entries stay in the file but aren't reported as policy.
		if _, ok := managed[pat]; !ok {
			continue
		}
		switch action {
		case "deny":
			p.BashDeny = append(p.BashDeny, pat)
		case "ask":
			p.BashAsk = append(p.BashAsk, pat)
		case "allow":
			p.BashAllow = append(p.BashAllow, pat)
		}
	}
	sort.Strings(p.BashDeny)
	sort.Strings(p.BashAsk)
	sort.Strings(p.BashAllow)
	return p, nil
}

// Save writes the policy to disk. It rebuilds `permission.bash` preserving
// hand-written entries (those not in the managed list) and replacing only
// the entries it previously owned, writes opencode.json with ONLY
// schema-valid keys (the retired managed-list key is stripped), then records
// the new managed list + fingerprint in the sidecar state file. The sidecar
// write is what makes the managed set authoritative across calls, so Save is
// self-consistent: two consecutive Saves correctly drop the first's entries.
func (a *Adapter) Save(p Policy, writtenByVersion string) error {
	if err := os.MkdirAll(filepath.Dir(a.SettingsPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(a.SettingsPath), err)
	}

	doc := &parsedDoc{
		TopLevel:      map[string]json.RawMessage{},
		BashMap:       map[string]string{},
		OtherPermKeys: map[string]json.RawMessage{},
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

	// Drop any previously-managed bash entries; hand-written entries
	// (not in the managed list) stay.
	managed, err := a.managedSet(doc)
	if err != nil {
		return err
	}
	newBash := map[string]string{}
	for pat, action := range doc.BashMap {
		if _, ok := managed[pat]; ok {
			continue
		}
		newBash[pat] = action
	}

	// Add the policy's entries.
	for _, pat := range p.BashDeny {
		newBash[pat] = "deny"
	}
	for _, pat := range p.BashAsk {
		newBash[pat] = "ask"
	}
	for _, pat := range p.BashAllow {
		newBash[pat] = "allow"
	}

	// Rebuild permission.* preserving other keys.
	permAll := map[string]json.RawMessage{}
	for k, v := range doc.OtherPermKeys {
		permAll[k] = v
	}
	if len(newBash) > 0 {
		encoded, err := marshalOrderedStringMap(newBash)
		if err != nil {
			return fmt.Errorf("encode permission.bash: %w", err)
		}
		permAll["bash"] = encoded
	} else {
		delete(permAll, "bash")
	}

	if len(permAll) == 0 {
		delete(doc.TopLevel, "permission")
	} else {
		encoded, err := marshalOrderedRawMap(permAll)
		if err != nil {
			return fmt.Errorf("encode permission: %w", err)
		}
		doc.TopLevel["permission"] = encoded
	}

	out, err := marshalOrderedRawMap(doc.TopLevel)
	if err != nil {
		return fmt.Errorf("encode opencode.json: %w", err)
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

	// The sidecar is the source of truth for the managed list; persist it
	// (plus fingerprint/version) so the next Load/Save sees these entries as
	// managed without an inline marker in opencode.json.
	if err := a.WriteState(p, writtenByVersion); err != nil {
		return err
	}
	return a.syncPlugin(p)
}

// RenderPlugin returns the native OpenCode plugin projection. The hook sends
// a normalized event to the local policy runner and refuses the tool call if
// the runner returns a non-zero decision. No shell interpolation is used.
func (a *Adapter) RenderPlugin() string {
	runner := os.Getenv("VROOLI_AGENT_POLICY_RUNNER")
	if strings.TrimSpace(runner) == "" {
		runner = "vrooli-policy-runner"
	}
	return fmt.Sprintf(`// Managed by Vrooli. Do not edit; reconcile permissions to regenerate.
export const VrooliPolicy = async () => ({
  "tool.execute.before": async (input) => {
    if (!input || input.tool !== "bash") return;
    const event = {
      contract_version: "agent-policy/v1",
      runner: "opencode",
      tool: input.tool,
      arguments: input.args ? [JSON.stringify(input.args)] : [],
      occurred_at: new Date().toISOString()
    };
    const result = Bun.spawnSync([%q, "hook", "--runner", "opencode"], {
      stdin: "pipe",
      stdout: "pipe",
      stderr: "pipe",
      input: JSON.stringify(event)
    });
    if (result.exitCode !== 0) {
      throw new Error("Vrooli policy runner denied or could not evaluate this tool call");
    }
  }
});
`, runner)
}

func (a *Adapter) syncPlugin(p Policy) error {
	if strings.TrimSpace(a.PluginPath) == "" {
		return nil
	}
	if len(p.BashDeny) == 0 {
		if err := os.Remove(a.PluginPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove OpenCode policy plugin: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.PluginPath), 0o755); err != nil {
		return fmt.Errorf("create OpenCode plugin directory: %w", err)
	}
	tmp := a.PluginPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(a.RenderPlugin()), 0o644); err != nil {
		return fmt.Errorf("write OpenCode policy plugin: %w", err)
	}
	if err := os.Rename(tmp, a.PluginPath); err != nil {
		return fmt.Errorf("publish OpenCode policy plugin: %w", err)
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

func marshalOrderedRawMap(m map[string]json.RawMessage) (json.RawMessage, error) {
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

func marshalOrderedStringMap(m map[string]string) (json.RawMessage, error) {
	raw := map[string]json.RawMessage{}
	for k, v := range m {
		enc, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		raw[k] = enc
	}
	return marshalOrderedRawMap(raw)
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
