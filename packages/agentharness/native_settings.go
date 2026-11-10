package agentharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"time"
)

// Native settings are harness configuration keys that Vrooli's operating
// model depends on but that are not permissions: for example Codex's
// multi-agent thread slots and sub-agent instructions. A resource declares
// them in a repository document (resources/<runner>/native-settings.json) so
// every installation gets the same defaults through `resource-<runner>
// ensure`, rather than through a hand edit on one machine.
//
// Ownership is per key. Vrooli writes a key only when it is absent, when it
// still holds the value Vrooli last wrote, or when the operator forces it. A
// value the user chose is reported as a conflict and left alone; `decline`
// records a permanent opt-out. The reconciler works on a decoded document
// (map[string]any) so each resource keeps its own file format.

// NativeSettingsSchemaVersion identifies the declared document format.
const NativeSettingsSchemaVersion = "vrooli.agent-native-settings/v1"

// NativeSettingsStateSchemaVersion identifies the sidecar format.
const NativeSettingsStateSchemaVersion = 1

// Native setting actions reported by a plan.
const (
	NativeSettingAdd       = "add"
	NativeSettingUpdate    = "update"
	NativeSettingAdopt     = "adopt"
	NativeSettingRemove    = "remove"
	NativeSettingRelease   = "release"
	NativeSettingUnchanged = "unchanged"
	NativeSettingConflict  = "conflict"
	NativeSettingDeclined  = "declined"
	// NativeSettingHeld is a value_from setting whose policy could not be read:
	// the live value stays exactly as it is and nothing is retired.
	NativeSettingHeld = "held"
)

// NativeSettingsDocument is the declared, repository-owned desired state.
type NativeSettingsDocument struct {
	SchemaVersion string          `json:"schema_version"`
	Runner        string          `json:"runner"`
	Description   string          `json:"description,omitempty"`
	Reference     string          `json:"reference,omitempty"`
	Settings      []NativeSetting `json:"settings"`
}

// NativeSetting is one managed key. Path addresses nested tables or objects.
// The desired value is either Value, or ValueFrom: a dotted path into the
// resource's model-policy.json (for example "interactive_default.model"), so a
// value the policy already owns is never written down a second time. A
// ValueFrom the policy does not define means the setting is not declared. A
// policy that cannot be read at all holds the setting instead (see
// NativeSettingHeld): an unreadable policy is not a decision to retire a value.
type NativeSetting struct {
	ID              string   `json:"id"`
	Path            []string `json:"path"`
	Value           any      `json:"value,omitempty"`
	ValueFrom       string   `json:"value_from,omitempty"`
	VerifiedVersion string   `json:"verified_version,omitempty"`
	Rationale       string   `json:"rationale"`
}

// DottedPath renders the key path for reports.
func (s NativeSetting) DottedPath() string { return strings.Join(s.Path, ".") }

// NativeSettingsState records what Vrooli last wrote, per setting ID.
type NativeSettingsState struct {
	SchemaVersion int                             `json:"schemaVersion"`
	SettingsPath  string                          `json:"settingsPath"`
	Managed       map[string]ManagedNativeSetting `json:"managed"`
	Declined      []string                        `json:"declined,omitempty"`
	WrittenByVer  string                          `json:"writtenByVersion,omitempty"`
	WrittenAt     time.Time                       `json:"writtenAt"`
}

// ManagedNativeSetting is the value Vrooli wrote at a path.
type ManagedNativeSetting struct {
	Path  []string `json:"path"`
	Value any      `json:"value"`
}

// NativeSettingChange is one planned action.
type NativeSettingChange struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Action   string `json:"action"`
	Live     any    `json:"live,omitempty"`
	Desired  any    `json:"desired,omitempty"`
	Reason   string `json:"reason,omitempty"`
	path     []string
	value    any
	verified string
}

// NativeSettingsPlan is the full set of actions for one document.
type NativeSettingsPlan struct {
	Runner       string                `json:"runner"`
	SettingsPath string                `json:"settings_path"`
	Changes      []NativeSettingChange `json:"changes"`
}

// Mutates reports whether applying the plan changes the native file.
func (p NativeSettingsPlan) Mutates() bool {
	for _, change := range p.Changes {
		switch change.Action {
		case NativeSettingAdd, NativeSettingUpdate, NativeSettingRemove:
			return true
		}
	}
	return false
}

// ChangesState reports whether applying the plan changes the sidecar.
func (p NativeSettingsPlan) ChangesState() bool {
	for _, change := range p.Changes {
		if change.Action != NativeSettingUnchanged && change.Action != NativeSettingConflict && change.Action != NativeSettingDeclined && change.Action != NativeSettingHeld {
			return true
		}
	}
	return false
}

// Conflicts lists settings left alone because the user owns the live value.
func (p NativeSettingsPlan) Conflicts() []NativeSettingChange {
	var out []NativeSettingChange
	for _, change := range p.Changes {
		if change.Action == NativeSettingConflict {
			out = append(out, change)
		}
	}
	return out
}

// ReadNativeSettingsDocument strictly decodes and validates a document.
func ReadNativeSettingsDocument(path string) (NativeSettingsDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return NativeSettingsDocument{}, err
	}
	return ParseNativeSettingsDocument(data)
}

// ParseNativeSettingsDocument decodes numbers exactly so an integer stays an
// integer when projected into TOML.
func ParseNativeSettingsDocument(data []byte) (NativeSettingsDocument, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var doc NativeSettingsDocument
	if err := decoder.Decode(&doc); err != nil {
		return doc, fmt.Errorf("parse native settings document: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return doc, errors.New("parse native settings document: trailing data")
	}
	for i := range doc.Settings {
		doc.Settings[i].Value = nativeValue(doc.Settings[i].Value)
	}
	return doc, doc.Validate()
}

// Validate rejects ambiguous or overlapping declarations.
func (d NativeSettingsDocument) Validate() error {
	if d.SchemaVersion != NativeSettingsSchemaVersion {
		return fmt.Errorf("native settings schema_version %q, want %q", d.SchemaVersion, NativeSettingsSchemaVersion)
	}
	if strings.TrimSpace(d.Runner) == "" {
		return errors.New("native settings document needs a runner")
	}
	ids := map[string]bool{}
	paths := make([]string, 0, len(d.Settings))
	for _, s := range d.Settings {
		if strings.TrimSpace(s.ID) == "" {
			return errors.New("native setting needs an id")
		}
		if ids[s.ID] {
			return fmt.Errorf("duplicate native setting id %q", s.ID)
		}
		ids[s.ID] = true
		if len(s.Path) == 0 || slices.Contains(s.Path, "") {
			return fmt.Errorf("native setting %q needs a non-empty path", s.ID)
		}
		if (s.Value == nil) == (s.ValueFrom == "") {
			return fmt.Errorf("native setting %q needs exactly one of value and value_from", s.ID)
		}
		if strings.TrimSpace(s.Rationale) == "" {
			return fmt.Errorf("native setting %q needs a rationale", s.ID)
		}
		paths = append(paths, s.DottedPath())
	}
	sort.Strings(paths)
	for i := 1; i < len(paths); i++ {
		if paths[i] == paths[i-1] || strings.HasPrefix(paths[i], paths[i-1]+".") {
			return fmt.Errorf("native setting paths overlap: %s and %s", paths[i-1], paths[i])
		}
	}
	return nil
}

// ResolveNativeSettings fills every value_from setting from the policy document
// and drops the ones the policy does not define. The input is not modified.
func ResolveNativeSettings(d NativeSettingsDocument, policy map[string]any) NativeSettingsDocument {
	resolved := d
	resolved.Settings = make([]NativeSetting, 0, len(d.Settings))
	for _, s := range d.Settings {
		if s.ValueFrom != "" {
			value, ok := lookupPath(policy, strings.Split(s.ValueFrom, "."))
			if !ok || value == nil || value == "" {
				continue
			}
			s.Value = nativeValue(value)
		}
		resolved.Settings = append(resolved.Settings, s)
	}
	return resolved
}

// NativeSettingsStatePath is the sidecar next to the native file.
func NativeSettingsStatePath(settingsPath string) string {
	return filepath.Join(filepath.Dir(settingsPath), ".vrooli-native-settings-state.json")
}

// LoadNativeSettingsState reads the sidecar; a missing one is not an error.
func LoadNativeSettingsState(path string) (*NativeSettingsState, error) {
	snapshot, err := ReadPermissionFile(path)
	if err != nil {
		return nil, fmt.Errorf("read native settings state: %w", err)
	}
	if !snapshot.Exists {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(snapshot.Data))
	decoder.UseNumber()
	var state NativeSettingsState
	if err := decoder.Decode(&state); err != nil {
		return nil, fmt.Errorf("parse native settings state %s (it only records what Vrooli wrote; remove it to start fresh): %w", path, err)
	}
	for id, managed := range state.Managed {
		managed.Value = nativeValue(managed.Value)
		state.Managed[id] = managed
	}
	return &state, nil
}

// WriteNativeSettingsState publishes the sidecar under the shared lock.
func WriteNativeSettingsState(path string, state NativeSettingsState) error {
	state.SchemaVersion = NativeSettingsStateSchemaVersion
	state.WrittenAt = time.Now().UTC()
	sort.Strings(state.Declined)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return NewHookBroker().WithLock(path, func() error {
		before, err := ReadPermissionFile(path)
		if err != nil {
			return err
		}
		return PublishPermissionFile(before, data)
	})
}

// PlanNativeSettings compares the live document with the declaration. With
// force, conflicts become updates; declined settings stay declined.
func PlanNativeSettings(live map[string]any, desired NativeSettingsDocument, state *NativeSettingsState, force bool) NativeSettingsPlan {
	plan := NativeSettingsPlan{Runner: desired.Runner}
	managed := map[string]ManagedNativeSetting{}
	declined := map[string]bool{}
	if state != nil {
		plan.SettingsPath = state.SettingsPath
		for id, m := range state.Managed {
			managed[id] = m
		}
		for _, id := range state.Declined {
			declined[id] = true
		}
	}
	declared := map[string]bool{}
	for _, s := range desired.Settings {
		declared[s.ID] = true
		if s.ValueFrom != "" && s.Value == nil {
			plan.Changes = append(plan.Changes, NativeSettingChange{ID: s.ID, Path: s.DottedPath(), Action: NativeSettingHeld, Reason: "model policy unavailable; the live value is left as it is"})
			continue
		}
		change := NativeSettingChange{ID: s.ID, Path: s.DottedPath(), Desired: s.Value, path: s.Path, value: s.Value, verified: s.VerifiedVersion}
		liveValue, present := lookupPath(live, s.Path)
		if present {
			change.Live = liveValue
		}
		record, wasManaged := managed[s.ID]
		ownsLive := wasManaged && slices.Equal(record.Path, s.Path) && present && nativeEqual(liveValue, record.Value)
		switch {
		case declined[s.ID]:
			change.Action, change.Reason = NativeSettingDeclined, "declined by the operator; run `settings accept` to manage it again"
		case present && nativeEqual(liveValue, s.Value) && ownsLive:
			change.Action = NativeSettingUnchanged
		case present && nativeEqual(liveValue, s.Value):
			// Also repairs a sidecar left stale by a failed state write.
			change.Action, change.Reason = NativeSettingAdopt, "live value already matches; record it as Vrooli-managed"
		case !present && !wasManaged:
			change.Action = NativeSettingAdd
		case ownsLive:
			change.Action, change.Reason = NativeSettingUpdate, "live value is the one Vrooli last wrote"
		case force:
			change.Action, change.Reason = NativeSettingUpdate, "forced over a user-owned value"
		case !present:
			change.Action, change.Reason = NativeSettingConflict, "removed since Vrooli wrote it; run `settings decline` to keep it removed or apply with --force to restore it"
		default:
			change.Action, change.Reason = NativeSettingConflict, "user-owned value differs; left alone (apply with --force to replace, or `settings decline` to stop reporting)"
		}
		plan.Changes = append(plan.Changes, change)
	}
	ids := make([]string, 0, len(managed))
	for id := range managed {
		if !declared[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		record := managed[id]
		change := NativeSettingChange{ID: id, Path: strings.Join(record.Path, "."), path: record.Path}
		liveValue, present := lookupPath(live, record.Path)
		if present {
			change.Live = liveValue
		}
		if present && nativeEqual(liveValue, record.Value) {
			change.Action, change.Reason = NativeSettingRemove, "no longer declared; removing the value Vrooli wrote"
		} else {
			change.Action, change.Reason = NativeSettingRelease, "no longer declared; the live value changed since Vrooli wrote it, so it stays"
		}
		plan.Changes = append(plan.Changes, change)
	}
	return plan
}

// ApplyNativeSettings mutates live according to the plan.
func ApplyNativeSettings(live map[string]any, plan NativeSettingsPlan) error {
	for _, change := range plan.Changes {
		switch change.Action {
		case NativeSettingAdd, NativeSettingUpdate:
			if err := setPath(live, change.path, change.value); err != nil {
				return fmt.Errorf("%s: %w", change.ID, err)
			}
		case NativeSettingRemove:
			deletePath(live, change.path)
		}
	}
	return nil
}

// NextNativeSettingsState returns the sidecar after a successful apply.
func NextNativeSettingsState(prev *NativeSettingsState, plan NativeSettingsPlan, settingsPath, version string) NativeSettingsState {
	next := NativeSettingsState{SettingsPath: settingsPath, Managed: map[string]ManagedNativeSetting{}, WrittenByVer: version}
	if prev != nil {
		for id, m := range prev.Managed {
			next.Managed[id] = m
		}
		next.Declined = append(next.Declined, prev.Declined...)
	}
	for _, change := range plan.Changes {
		switch change.Action {
		case NativeSettingAdd, NativeSettingUpdate, NativeSettingAdopt:
			next.Managed[change.ID] = ManagedNativeSetting{Path: change.path, Value: change.value}
		case NativeSettingRemove, NativeSettingRelease, NativeSettingDeclined:
			delete(next.Managed, change.ID)
		}
	}
	return next
}

// DeclineNativeSetting records a permanent opt-out; accept reverses it.
func DeclineNativeSetting(state *NativeSettingsState, id string, decline bool) NativeSettingsState {
	next := NativeSettingsState{Managed: map[string]ManagedNativeSetting{}}
	if state != nil {
		next = *state
		next.Managed = map[string]ManagedNativeSetting{}
		for k, v := range state.Managed {
			next.Managed[k] = v
		}
		next.Declined = append([]string(nil), state.Declined...)
	}
	next.Declined = slices.DeleteFunc(next.Declined, func(existing string) bool { return existing == id })
	if decline {
		next.Declined = append(next.Declined, id)
		delete(next.Managed, id)
	}
	return next
}

// VerifiedVersion returns the version a setting was verified on, if any.
func (c NativeSettingChange) VerifiedVersion() string { return c.verified }

func lookupPath(doc map[string]any, path []string) (any, bool) {
	var current any = doc
	for _, key := range path {
		table, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = table[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func setPath(doc map[string]any, path []string, value any) error {
	table := doc
	for i, key := range path[:len(path)-1] {
		next, ok := table[key]
		if !ok {
			created := map[string]any{}
			table[key] = created
			table = created
			continue
		}
		nested, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("%s is not a table", strings.Join(path[:i+1], "."))
		}
		table = nested
	}
	table[path[len(path)-1]] = value
	return nil
}

// deletePath removes the key and any table it leaves empty.
func deletePath(doc map[string]any, path []string) {
	if len(path) == 1 {
		delete(doc, path[0])
		return
	}
	nested, ok := doc[path[0]].(map[string]any)
	if !ok {
		return
	}
	deletePath(nested, path[1:])
	if len(nested) == 0 {
		delete(doc, path[0])
	}
}

// nativeValue converts json.Number into int64 or float64 so a declared 8
// projects as a TOML integer, not 8.0.
func nativeValue(v any) any {
	switch typed := v.(type) {
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return i
		}
		f, _ := typed.Float64()
		return f
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = nativeValue(item)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, item := range typed {
			out[k] = nativeValue(item)
		}
		return out
	default:
		return v
	}
}

// nativeEqual compares values across decoders (TOML int64, JSON numbers).
func nativeEqual(a, b any) bool {
	return reflect.DeepEqual(canonicalValue(a), canonicalValue(b))
}

func canonicalValue(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var out any
	if decoder.Decode(&out) != nil {
		return v
	}
	return canonicalNumbers(out)
}

func canonicalNumbers(v any) any {
	switch typed := v.(type) {
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return fmt.Sprintf("%d", i)
		}
		f, _ := typed.Float64()
		return fmt.Sprintf("%g", f)
	case []any:
		for i := range typed {
			typed[i] = canonicalNumbers(typed[i])
		}
		return typed
	case map[string]any:
		for k := range typed {
			typed[k] = canonicalNumbers(typed[k])
		}
		return typed
	default:
		return v
	}
}
