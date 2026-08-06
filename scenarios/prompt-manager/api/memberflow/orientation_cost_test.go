package memberflow

import (
	"os"
	"path/filepath"
	"testing"
)

// writeOrientationFixture builds a store with one team, N member directories,
// a topic catalog, decision contexts, and a canon document of the given length.
func writeOrientationFixture(t *testing.T, members int, topics int, decisions int, canonLines int) (string, string) {
	t.Helper()
	root := t.TempDir()
	storeDir := filepath.Join(root, "scenarios", "prompt-manager", "store")
	teamDir := filepath.Join(storeDir, "teams", "fixture-team")
	if err := os.MkdirAll(filepath.Join(teamDir, "members"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for i := 0; i < members; i++ {
		if err := os.MkdirAll(filepath.Join(teamDir, "members", "member-"+string(rune('a'+i))), 0o755); err != nil {
			t.Fatalf("mkdir member: %v", err)
		}
	}

	canonPath := "docs/fixture/OPERATING_MODEL.md"
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(canonPath)), 0o755); err != nil {
		t.Fatalf("mkdir canon: %v", err)
	}
	body := make([]byte, 0, canonLines)
	for i := 0; i < canonLines-1; i++ {
		body = append(body, '\n')
	}
	if err := os.WriteFile(filepath.Join(root, canonPath), body, 0o644); err != nil {
		t.Fatalf("write canon: %v", err)
	}

	team := `{"kind":"team","schemaVersion":1,"id":"fixture-team","topicCatalog":[`
	for i := 0; i < topics; i++ {
		if i > 0 {
			team += ","
		}
		team += `{"prefix":"fixture-` + string(rune('a'+i)) + `/*","status":"live","purpose":"p"}`
	}
	team += `],"operatingContract":{"schemaVersion":1,"decisionContexts":{`
	for i := 0; i < decisions; i++ {
		if i > 0 {
			team += ","
		}
		team += `"ctx-` + string(rune('a'+i)) + `":{}`
	}
	team += `},"documents":{"planOfRecord":[{"id":"fixture-canon","paths":[{"base":"repo-root","path":"` + canonPath + `"}],"writePolicy":"operator-curated-via-decisions"}]}}}`
	if err := os.WriteFile(filepath.Join(teamDir, "team.json"), []byte(team), 0o644); err != nil {
		t.Fatalf("write team.json: %v", err)
	}
	return storeDir, root
}

func TestOrientationCostSumsEveryDeclaredSurface(t *testing.T) {
	storeDir, repoRoot := writeOrientationFixture(t, 3, 2, 4, 100)
	report, err := ComputeOrientationCost(storeDir, repoRoot)
	if err != nil {
		t.Fatalf("ComputeOrientationCost: %v", err)
	}
	if len(report.Teams) != 1 {
		t.Fatalf("teams = %d, want 1", len(report.Teams))
	}
	got := report.Teams[0]
	want := OrientationComponents{Members: 3, CanonLines: 100, Topics: 2, DecisionContexts: 4}
	if got.Components != want {
		t.Fatalf("components = %+v, want %+v", got.Components, want)
	}
	// 3*10 + 2*2 + 4*3 + 100/50 = 30 + 4 + 12 + 2 = 48
	if got.Composite != 48 {
		t.Fatalf("composite = %d, want 48", got.Composite)
	}
	if len(got.MissingCanon) != 0 {
		t.Fatalf("missingCanon = %v", got.MissingCanon)
	}
}

// Each component must move the composite on its own, otherwise a team could
// grow along an unweighted axis without the ratchet noticing.
func TestOrientationCostRisesWithEveryComponent(t *testing.T) {
	baseStore, baseRoot := writeOrientationFixture(t, 2, 1, 1, 50)
	base, err := ComputeOrientationCost(baseStore, baseRoot)
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	baseline := base.Teams[0].Composite

	cases := []struct {
		name                              string
		members, topics, decisions, canon int
	}{
		{"one more member", 3, 1, 1, 50},
		{"one more topic", 2, 2, 1, 50},
		{"one more decision context", 2, 1, 2, 50},
		{"a page more canon", 2, 1, 1, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, root := writeOrientationFixture(t, tc.members, tc.topics, tc.decisions, tc.canon)
			report, err := ComputeOrientationCost(store, root)
			if err != nil {
				t.Fatalf("ComputeOrientationCost: %v", err)
			}
			if report.Teams[0].Composite <= baseline {
				t.Fatalf("composite = %d, want > baseline %d", report.Teams[0].Composite, baseline)
			}
		})
	}
}

// A declared document that is not on disk must be named, not silently skipped:
// a broken reference would otherwise make a team read as cheaper to orient in.
func TestOrientationCostReportsCanonThatIsNotOnDisk(t *testing.T) {
	storeDir, repoRoot := writeOrientationFixture(t, 1, 0, 0, 10)
	if err := os.Remove(filepath.Join(repoRoot, "docs/fixture/OPERATING_MODEL.md")); err != nil {
		t.Fatalf("remove canon: %v", err)
	}
	report, err := ComputeOrientationCost(storeDir, repoRoot)
	if err != nil {
		t.Fatalf("ComputeOrientationCost: %v", err)
	}
	got := report.Teams[0]
	if len(got.MissingCanon) != 1 || got.MissingCanon[0] != "docs/fixture/OPERATING_MODEL.md" {
		t.Fatalf("missingCanon = %v", got.MissingCanon)
	}
	if got.Components.CanonLines != 0 {
		t.Fatalf("canonLines = %d, want 0", got.Components.CanonLines)
	}
}

// The report must state that one reading cannot band anything. A composite
// presented without that caveat invites exactly the misread the trend band
// exists to prevent.
func TestOrientationCostReportCarriesTheTrendCaveat(t *testing.T) {
	storeDir, repoRoot := writeOrientationFixture(t, 1, 1, 1, 10)
	report, err := ComputeOrientationCost(storeDir, repoRoot)
	if err != nil {
		t.Fatalf("ComputeOrientationCost: %v", err)
	}
	if report.Note == "" {
		t.Fatal("report omits the trend caveat")
	}
}

func TestOrientationCostToleratesAnEmptyStore(t *testing.T) {
	report, err := ComputeOrientationCost(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatalf("ComputeOrientationCost: %v", err)
	}
	if len(report.Teams) != 0 {
		t.Fatalf("teams = %+v, want none", report.Teams)
	}
}
