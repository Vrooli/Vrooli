package operatingmode

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// modesDir is the on-disk mode data folder, relative to this package.
const modesDir = "../../../modes"

// canonicalSchemaPath is the repo-root schema registry copy the embedded schema
// must stay byte-identical to.
const canonicalSchemaPath = "../../../../../.vrooli/schemas/operating-mode.schema.json"

func loadModeFromDisk(t *testing.T, id string) Definition {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(modesDir, id, ModeFileName))
	if err != nil {
		t.Fatalf("read mode %q: %v", id, err)
	}
	def, err := LoadModeDefinition(raw)
	if err != nil {
		t.Fatalf("load mode %q: %v", id, err)
	}
	return def
}

// TestEmbeddedSchemaMatchesRegistry guards against drift between the package's
// embedded schema copy and the canonical repo-root schema.
func TestEmbeddedSchemaMatchesRegistry(t *testing.T) {
	canonical, err := os.ReadFile(canonicalSchemaPath)
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	if !reflect.DeepEqual(canonical, schemaBytes) {
		t.Fatalf("embedded operating-mode schema drifted from %s; re-copy the canonical schema", canonicalSchemaPath)
	}
}

// TestLoadModesFromDirDiscoversShippedModes proves the data-backed registry
// discovers exactly the three shipped modes from disk and validates them.
func TestLoadModesFromDirDiscoversShippedModes(t *testing.T) {
	defs, err := LoadModesFromDir(modesDir)
	if err != nil {
		t.Fatalf("LoadModesFromDir: %v", err)
	}
	got := SortedModes(defs)
	want := []Mode{ModeHolisticLoop, ModeItemLevel, ModePhasedPlanDrain}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discovered modes = %v, want %v", got, want)
	}
}

func sameSet(a, b []Phase) bool {
	if len(a) != len(b) {
		return false
	}
	as := append([]Phase(nil), a...)
	bs := append([]Phase(nil), b...)
	sort.Slice(as, func(i, j int) bool { return as[i] < as[j] })
	sort.Slice(bs, func(i, j int) bool { return bs[i] < bs[j] })
	return reflect.DeepEqual(as, bs)
}

// TestLoadedBranchingSelectsExpectedPhase proves the generic guard graph routes
// each shipped mode's phases to the correct next phase for representative round
// output — the branching contract, asserted directly against the data-loaded
// modes (the hardcoded Go twins are gone).
func TestLoadedBranchingSelectsExpectedPhase(t *testing.T) {
	cases := []struct {
		mode    Mode
		phase   Phase
		payload map[string]any
		want    []Phase
	}{
		{ModeHolisticLoop, "investigate", map[string]any{}, []Phase{"plan"}},
		{ModeHolisticLoop, "plan", map[string]any{}, []Phase{"execute"}},
		{ModeHolisticLoop, "execute", map[string]any{"replan_needed": true}, []Phase{"investigate"}},
		{ModeHolisticLoop, "execute", map[string]any{"replan_needed": false}, []Phase{"review"}},
		{ModeHolisticLoop, "execute", map[string]any{}, []Phase{"review"}},
		{ModeHolisticLoop, "review", map[string]any{}, []Phase{"reconcile"}},
		{ModePhasedPlanDrain, "prepare_plan", map[string]any{}, []Phase{"execute_next"}},
		{ModePhasedPlanDrain, "execute_next", map[string]any{}, []Phase{"classify_progress"}},
		{ModePhasedPlanDrain, "classify_progress", progressPayload("continue"), []Phase{"execute_next"}},
		{ModePhasedPlanDrain, "classify_progress", progressPayload("replan"), []Phase{"prepare_plan"}},
		{ModePhasedPlanDrain, "classify_progress", progressPayload("complete"), []Phase{"review"}},
		{ModePhasedPlanDrain, "classify_progress", progressPayload("blocked"), nil},
		{ModePhasedPlanDrain, "review", map[string]any{}, []Phase{"reconcile"}},
	}
	for _, tc := range cases {
		t.Run(string(tc.mode)+"/"+string(tc.phase), func(t *testing.T) {
			loaded := loadModeFromDisk(t, string(tc.mode))
			got, _ := selectNextPhases(loaded, tc.phase, NewMapFieldLookup(tc.payload))
			if !sameSet(got, tc.want) {
				t.Fatalf("branching %q/%q payload=%v: got=%v want=%v", tc.mode, tc.phase, tc.payload, got, tc.want)
			}
		})
	}
}

func progressPayload(decision string) map[string]any {
	return map[string]any{"progress": map[string]any{"decision": decision}}
}

// TestNovelBranchingExampleRun proves the vocabulary is generic: the fixture's
// multi-way enum switch, compound `all` numeric guard, and `not` reloop produce
// the declared expected_path through the real generic evaluator.
func TestNovelBranchingExampleRun(t *testing.T) {
	raw, err := os.ReadFile("testdata/novel-branching/mode.json")
	if err != nil {
		t.Fatalf("read novel fixture: %v", err)
	}
	def, err := LoadModeDefinition(raw)
	if err != nil {
		t.Fatalf("load novel fixture: %v", err)
	}
	if err := ValidateLoadedModes(map[Mode]Definition{def.Mode: def}); err != nil {
		t.Fatalf("validate novel fixture: %v", err)
	}

	runRaw, err := os.ReadFile("testdata/novel-branching/example-runs/full-reloop.json")
	if err != nil {
		t.Fatalf("read example-run: %v", err)
	}
	run, err := LoadExampleRun(runRaw)
	if err != nil {
		t.Fatalf("load example-run: %v", err)
	}
	walked, err := WalkExampleRun(def, run)
	if err != nil {
		t.Fatalf("walk example-run: %v", err)
	}
	want := []Phase{"triage", "reproduce", "fix", "verify", "fix", "verify", "close"}
	if !reflect.DeepEqual(walked, want) {
		t.Fatalf("walked path = %v, want %v", walked, want)
	}
}
