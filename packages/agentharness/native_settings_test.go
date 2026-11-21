package agentharness

import (
	"path/filepath"
	"testing"
)

const nativeDoc = `{
  "schema_version": "vrooli.agent-native-settings/v1",
  "runner": "codex",
  "settings": [
    {"id": "slots", "path": ["features", "multi_agent_v2", "max_concurrent_threads_per_session"], "value": 8, "rationale": "r"},
    {"id": "hint", "path": ["features", "multi_agent_v2", "multi_agent_mode_hint_text"], "value": "use sub-agents", "rationale": "r"}
  ]
}`

func mustDoc(t *testing.T, raw string) NativeSettingsDocument {
	t.Helper()
	doc, err := ParseNativeSettingsDocument([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func actions(plan NativeSettingsPlan) map[string]string {
	out := map[string]string{}
	for _, c := range plan.Changes {
		out[c.ID] = c.Action
	}
	return out
}

func TestNativeSettingsIntegersStayIntegers(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	if _, ok := doc.Settings[0].Value.(int64); !ok {
		t.Fatalf("declared 8 decoded as %T, want int64 so TOML writes an integer", doc.Settings[0].Value)
	}
}

func TestNativeSettingsValidateRejectsOverlapAndUnknownFields(t *testing.T) {
	overlap := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[
	  {"id":"a","path":["features"],"value":1,"rationale":"r"},
	  {"id":"b","path":["features","x"],"value":1,"rationale":"r"}]}`
	if _, err := ParseNativeSettingsDocument([]byte(overlap)); err == nil {
		t.Fatal("overlapping paths accepted")
	}
	unknown := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[],"extra":1}`
	if _, err := ParseNativeSettingsDocument([]byte(unknown)); err == nil {
		t.Fatal("unknown field accepted")
	}
	noRationale := `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[{"id":"a","path":["a"],"value":1}]}`
	if _, err := ParseNativeSettingsDocument([]byte(noRationale)); err == nil {
		t.Fatal("setting without rationale accepted")
	}
}

func TestNativeSettingsFreshInstallAddsAndPreservesOtherKeys(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	live := map[string]any{"model": "gpt", "features": map[string]any{"other": true}}
	plan := PlanNativeSettings(live, doc, nil, false)
	if got := actions(plan); got["slots"] != NativeSettingAdd || got["hint"] != NativeSettingAdd {
		t.Fatalf("actions = %v", got)
	}
	if err := ApplyNativeSettings(live, plan); err != nil {
		t.Fatal(err)
	}
	features := live["features"].(map[string]any)
	if features["other"] != true || live["model"] != "gpt" {
		t.Fatalf("unmanaged keys lost: %v", live)
	}
	if v, _ := lookupPath(live, doc.Settings[0].Path); v != int64(8) {
		t.Fatalf("slots = %#v", v)
	}
	state := NextNativeSettingsState(nil, plan, "/x/config.toml", "test")
	second := PlanNativeSettings(live, doc, &state, false)
	if second.Mutates() || second.ChangesState() {
		t.Fatalf("second plan not idempotent: %v", actions(second))
	}
}

func TestNativeSettingsAdoptsMatchingValueWithoutWriting(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	// TOML decodes integers as int64; the hand-installed value must be adopted.
	live := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{
		"max_concurrent_threads_per_session": int64(8), "multi_agent_mode_hint_text": "use sub-agents",
	}}}
	plan := PlanNativeSettings(live, doc, nil, false)
	if got := actions(plan); got["slots"] != NativeSettingAdopt || got["hint"] != NativeSettingAdopt {
		t.Fatalf("actions = %v", got)
	}
	if plan.Mutates() || !plan.ChangesState() {
		t.Fatal("adopt must record state without writing the native file")
	}
}

func TestNativeSettingsUserValueIsAConflictUnlessForced(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	live := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{"max_concurrent_threads_per_session": int64(4)}}}
	plan := PlanNativeSettings(live, doc, nil, false)
	if actions(plan)["slots"] != NativeSettingConflict || len(plan.Conflicts()) != 1 {
		t.Fatalf("actions = %v", actions(plan))
	}
	if err := ApplyNativeSettings(live, plan); err != nil {
		t.Fatal(err)
	}
	if v, _ := lookupPath(live, doc.Settings[0].Path); v != int64(4) {
		t.Fatalf("user value overwritten: %#v", v)
	}
	forced := PlanNativeSettings(live, doc, nil, true)
	if actions(forced)["slots"] != NativeSettingUpdate {
		t.Fatalf("forced actions = %v", actions(forced))
	}
}

func TestNativeSettingsUpdatesOnlyWhatVrooliWrote(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	state := &NativeSettingsState{Managed: map[string]ManagedNativeSetting{
		"slots": {Path: doc.Settings[0].Path, Value: int64(6)},
		"hint":  {Path: doc.Settings[1].Path, Value: "old hint"},
	}}
	live := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{
		"max_concurrent_threads_per_session": int64(6), // still Vrooli's old value
		"multi_agent_mode_hint_text":         "user edited",
	}}}
	plan := PlanNativeSettings(live, doc, state, false)
	got := actions(plan)
	if got["slots"] != NativeSettingUpdate || got["hint"] != NativeSettingConflict {
		t.Fatalf("actions = %v", got)
	}
}

func TestNativeSettingsRemovedDeclarationCleansUpOwnValueOnly(t *testing.T) {
	doc := mustDoc(t, `{"schema_version":"vrooli.agent-native-settings/v1","runner":"codex","settings":[]}`)
	path := []string{"features", "multi_agent_v2", "max_concurrent_threads_per_session"}
	other := []string{"features", "multi_agent_v2", "multi_agent_mode_hint_text"}
	state := &NativeSettingsState{Managed: map[string]ManagedNativeSetting{
		"slots": {Path: path, Value: int64(8)},
		"hint":  {Path: other, Value: "vrooli"},
	}}
	live := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{
		"max_concurrent_threads_per_session": int64(8),
		"multi_agent_mode_hint_text":         "user",
	}}}
	plan := PlanNativeSettings(live, doc, state, false)
	if got := actions(plan); got["slots"] != NativeSettingRemove || got["hint"] != NativeSettingRelease {
		t.Fatalf("actions = %v", got)
	}
	if err := ApplyNativeSettings(live, plan); err != nil {
		t.Fatal(err)
	}
	if _, ok := lookupPath(live, path); ok {
		t.Fatal("Vrooli-written value not removed")
	}
	if v, _ := lookupPath(live, other); v != "user" {
		t.Fatal("user value removed")
	}
	next := NextNativeSettingsState(state, plan, "", "")
	if len(next.Managed) != 0 {
		t.Fatalf("state still manages %v", next.Managed)
	}
	// An emptied table is pruned.
	empty := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{"k": int64(1)}}}
	deletePath(empty, []string{"features", "multi_agent_v2", "k"})
	if len(empty) != 0 {
		t.Fatalf("empty tables kept: %v", empty)
	}
}

func TestNativeSettingsDeclineIsPermanentUntilAccepted(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	state := DeclineNativeSetting(nil, "slots", true)
	plan := PlanNativeSettings(map[string]any{}, doc, &state, true)
	if actions(plan)["slots"] != NativeSettingDeclined {
		t.Fatalf("declined setting planned as %v", actions(plan)["slots"])
	}
	accepted := DeclineNativeSetting(&state, "slots", false)
	if actions(PlanNativeSettings(map[string]any{}, doc, &accepted, false))["slots"] != NativeSettingAdd {
		t.Fatal("accept did not restore management")
	}
}

func TestNativeSettingsStateRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := NativeSettingsStatePath(filepath.Join(dir, "config.toml"))
	in := NativeSettingsState{SettingsPath: "p", Managed: map[string]ManagedNativeSetting{"slots": {Path: []string{"a", "b"}, Value: int64(8)}}, Declined: []string{"z", "a"}}
	if err := WriteNativeSettingsState(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadNativeSettingsState(path)
	if err != nil || out == nil {
		t.Fatalf("load: %v %v", out, err)
	}
	if out.Managed["slots"].Value != int64(8) || out.Declined[0] != "a" {
		t.Fatalf("round trip = %+v", out)
	}
	missing, err := LoadNativeSettingsState(filepath.Join(dir, "none.json"))
	if err != nil || missing != nil {
		t.Fatalf("missing state = %v %v", missing, err)
	}
}

func TestNativeSettingsNonTableParentIsAnError(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	live := map[string]any{"features": "not a table"}
	plan := PlanNativeSettings(live, doc, nil, false)
	if err := ApplyNativeSettings(live, plan); err == nil {
		t.Fatal("writing under a scalar parent succeeded")
	}
}

// A failed state write after a successful config write leaves the sidecar
// recording the old value. The next plan must repair it, or a later change of
// the declaration would mistake Vrooli's own value for a user's.
func TestNativeSettingsRepairsAStaleSidecar(t *testing.T) {
	doc := mustDoc(t, nativeDoc)
	stale := &NativeSettingsState{Managed: map[string]ManagedNativeSetting{
		"slots": {Path: doc.Settings[0].Path, Value: int64(6)},
	}}
	live := map[string]any{"features": map[string]any{"multi_agent_v2": map[string]any{"max_concurrent_threads_per_session": int64(8)}}}
	plan := PlanNativeSettings(live, doc, stale, false)
	if actions(plan)["slots"] != NativeSettingAdopt || !plan.ChangesState() {
		t.Fatalf("stale sidecar not repaired: %v", actions(plan))
	}
	next := NextNativeSettingsState(stale, plan, "", "")
	if next.Managed["slots"].Value != int64(8) {
		t.Fatalf("repaired value = %#v", next.Managed["slots"].Value)
	}
}
