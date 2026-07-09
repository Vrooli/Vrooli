package operatingmode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// modesDir is the on-disk mode data folder, relative to this package.
const modesDir = "../../../modes"

// canonicalSchemaPath is the repo-root schema registry copy the embedded schema
// must stay byte-identical to.
const canonicalSchemaPath = "../../../../../.vrooli/schemas/operating-mode.schema.json"

// promptSkillsRoot is Prompt Manager's committed core-skill store, relative to
// this package. Shipped operating-mode prompt catalog entries must resolve from
// this tree in a clean checkout; runtime sync cannot repair a missing source
// asset.
const promptSkillsRoot = "../../../../prompt-manager/store/skills/packs/core"

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

// TestShippedPromptCatalogResolvesCommittedSkills closes the source-contract
// loop between data-backed mode definitions and Prompt Manager. Registry
// validation proves catalog metadata is internally consistent; this test also
// proves every non-delegated phase's referenced skill is committed, active, and
// backed by non-empty prompt source in a clean checkout.
func TestShippedPromptCatalogResolvesCommittedSkills(t *testing.T) {
	type skillMetadata struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Entry  string `json:"entry"`
	}

	entries := PromptCatalogEntries()
	if len(entries) == 0 {
		t.Fatal("shipped prompt catalog is empty")
	}

	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		t.Run(entry.SkillID, func(t *testing.T) {
			if _, duplicate := seen[entry.SkillID]; duplicate {
				t.Fatalf("duplicate shipped prompt skill %q", entry.SkillID)
			}
			seen[entry.SkillID] = struct{}{}

			skillDir := filepath.Join(promptSkillsRoot, entry.SkillID)
			metadataBytes, err := os.ReadFile(filepath.Join(skillDir, "skill.json"))
			if err != nil {
				t.Fatalf("read prompt skill metadata for %s/%s: %v", entry.Mode, entry.Phase, err)
			}

			var metadata skillMetadata
			if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
				t.Fatalf("decode prompt skill metadata: %v", err)
			}
			if metadata.ID != entry.SkillID {
				t.Fatalf("prompt skill metadata id = %q, want %q", metadata.ID, entry.SkillID)
			}
			if metadata.Status != "active" {
				t.Fatalf("prompt skill %q status = %q, want active", entry.SkillID, metadata.Status)
			}
			if metadata.Entry == "" || filepath.Base(metadata.Entry) != metadata.Entry {
				t.Fatalf("prompt skill %q entry = %q, want a local filename", entry.SkillID, metadata.Entry)
			}

			content, err := os.ReadFile(filepath.Join(skillDir, metadata.Entry))
			if err != nil {
				t.Fatalf("read prompt skill source: %v", err)
			}
			if strings.TrimSpace(string(content)) == "" {
				t.Fatalf("prompt skill %q source is empty", entry.SkillID)
			}
		})
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
		{ModeHolisticLoop, "execute", map[string]any{"progress": "blocked"}, []Phase{"investigate"}},
		{ModeHolisticLoop, "execute", map[string]any{"progress": "complete"}, []Phase{"review"}},
		// A delegated execute round with no derived progress matches no
		// parent guard: routing waits for the edge-classified value.
		{ModeHolisticLoop, "execute", map[string]any{}, nil},
		{ModeHolisticLoop, "review", map[string]any{}, []Phase{"reconcile"}},
		// The generic drain routes on the edge-derived `progress` field, hoisted
		// into the round payload by classification-on-transition.
		{ModePhasedPlanDrain, "execute", map[string]any{"progress": "continue"}, []Phase{"execute"}},
		{ModePhasedPlanDrain, "execute", map[string]any{"progress": "complete"}, nil},
		{ModePhasedPlanDrain, "execute", map[string]any{"progress": "blocked"}, nil},
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
	walked, err := WalkExampleRun(map[Mode]Definition{def.Mode: def}, def, run)
	if err != nil {
		t.Fatalf("walk example-run: %v", err)
	}
	want := []Phase{"triage", "reproduce", "fix", "verify", "fix", "verify", "close"}
	if !reflect.DeepEqual(walked, want) {
		t.Fatalf("walked path = %v, want %v", walked, want)
	}
}
