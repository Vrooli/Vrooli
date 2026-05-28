// Package permissions manages the OpenCode agent's permission config at
// ~/.config/opencode/opencode.json.
//
// Scope is narrow on purpose: the adapter owns only the
// `permission.bash` map (string→"allow"|"ask"|"deny") and only the
// entries it previously wrote. Every other top-level key, every other
// `permission.*` key (e.g. `edit`), and every `permission.bash` entry
// not previously tagged Vrooli-managed round-trips untouched.
//
// Unlike Claude Code there is no hook backstop — OpenCode honours the
// config directly per upstream docs (last-match-wins). The adapter
// writes alphabetically-sorted keys so output is deterministic; under
// alphabetical order `*` (0x2A) sorts before letters, so specific
// patterns end up later and win the last-match-wins evaluation. Users
// needing different ordering can hand-edit; drift-check will surface
// it.
//
// Duplicated structurally from resources/claude-code/cli/internal/permissions
// per the duplicate-before-extract memory. A Phase 4 follow-up will
// extract the shared canonical Policy + state shape into
// packages/cli-core/agentpolicy.
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

// ManagedByMarker labels bash-map entries owned by this adapter. Stored
// in a sidecar field on the parsed doc, not on each map value (OpenCode
// values are plain "allow"/"ask"/"deny" strings).
const ManagedByMarker = "vrooli"

// managedSidecarKey is a Vrooli-only top-level key that records which
// bash patterns this adapter currently owns. Hand-written entries are
// detected by absence from this list and never touched.
const managedSidecarKey = "x-vrooli-managed-permissions"

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
}

// DefaultAdapter returns an Adapter rooted at $HOME/.config/opencode.
func DefaultAdapter() (*Adapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve $HOME: %w", err)
	}
	return &Adapter{
		SettingsPath: filepath.Join(home, ".config", "opencode", "opencode.json"),
	}, nil
}

// parsedDoc holds the full top-level structure so Save can rebuild it
// preserving unknown keys and unmanaged bash entries.
type parsedDoc struct {
	TopLevel       map[string]json.RawMessage
	BashMap        map[string]string
	OtherPermKeys  map[string]json.RawMessage
	ManagedSidecar []string
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
	if raw, ok := doc.TopLevel[managedSidecarKey]; ok {
		var lst []string
		if err := json.Unmarshal(raw, &lst); err == nil {
			doc.ManagedSidecar = lst
		}
	}
	return doc, nil
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
	p := Policy{}
	managed := map[string]struct{}{}
	for _, k := range doc.ManagedSidecar {
		managed[k] = struct{}{}
	}
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

// Save writes the policy to disk. The adapter rebuilds `permission.bash`
// preserving hand-written entries (those not listed in the managed
// sidecar) and replacing only the entries it previously owned.
func (a *Adapter) Save(p Policy) error {
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
	// (not in the sidecar list) stay.
	managed := map[string]struct{}{}
	for _, k := range doc.ManagedSidecar {
		managed[k] = struct{}{}
	}
	newBash := map[string]string{}
	for pat, action := range doc.BashMap {
		if _, ok := managed[pat]; ok {
			continue
		}
		newBash[pat] = action
	}

	// Add the policy's entries.
	newManaged := make([]string, 0, len(p.BashDeny)+len(p.BashAsk)+len(p.BashAllow))
	for _, pat := range p.BashDeny {
		newBash[pat] = "deny"
		newManaged = append(newManaged, pat)
	}
	for _, pat := range p.BashAsk {
		newBash[pat] = "ask"
		newManaged = append(newManaged, pat)
	}
	for _, pat := range p.BashAllow {
		newBash[pat] = "allow"
		newManaged = append(newManaged, pat)
	}
	sort.Strings(newManaged)

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

	if len(newManaged) == 0 {
		delete(doc.TopLevel, managedSidecarKey)
	} else {
		encoded, err := json.Marshal(newManaged)
		if err != nil {
			return fmt.Errorf("encode managed sidecar: %w", err)
		}
		doc.TopLevel[managedSidecarKey] = encoded
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
	return nil
}

// RenderHook is intentionally a no-op for OpenCode: the upstream agent
// honours `permission.bash` directly. Returned for API symmetry with
// the Claude adapter; callers should not depend on a non-nil value.
func (a *Adapter) RenderHook(_ Policy) map[string]any {
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
