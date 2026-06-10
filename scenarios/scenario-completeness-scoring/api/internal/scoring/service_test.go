package scoring

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/freshness-go/treedigest"
)

// writeFixtureScenario lays down a minimal real scenario tree: requirements
// registry, service manifest, a passed unit phase with empty findings, and a
// tiny UI. Returns the scenario root.
func writeFixtureScenario(t *testing.T, scenariosRoot, name string) string {
	t.Helper()
	root := filepath.Join(scenariosRoot, name)

	files := map[string]string{
		".vrooli/service.json": `{"name":"` + name + `","category":"utility"}`,
		"requirements/index.json": `{
			"imports": ["01-core/module.json"]
		}`,
		"requirements/01-core/module.json": `{
			"module_id": "MOD-P0-001",
			"requirements": [
				{"id": "REQ-1", "status": "complete", "validation": [{"type": "test", "phase": "unit"}]},
				{"id": "REQ-2", "status": "pending", "validation": [{"type": "test", "phase": "unit"}]}
			]
		}`,
		"coverage/runs/20260610-000000-fixture/phase-results/unit.json": `{
			"phase": "unit", "scenario": "` + name + `", "status": "passed",
			"updated_at": "2026-06-10T00:00:00Z", "findings": []
		}`,
		"ui/src/App.tsx": "export function App() {\n  return fetch(\"/api/v1/scores\");\n}\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// stampRunIndex writes coverage/runs.index.json with one passed run for the
// required phases at the scenario's CURRENT digest.
func stampRunIndex(t *testing.T, root string) {
	t.Helper()
	digest, err := treedigest.Compute(root)
	if err != nil {
		t.Fatalf("compute digest: %v", err)
	}
	type phase struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	index := []map[string]any{{
		"run_id":       "20260610-000000-fixture",
		"scenario":     filepath.Base(root),
		"started_at":   "2026-06-10T00:00:00Z",
		"completed_at": "2026-06-10T00:00:05Z",
		"status":       "passed",
		"tree_digest":  digest,
		"phases": []phase{
			{Name: "structure", Status: "passed"},
			{Name: "standards", Status: "passed"},
			{Name: "docs", Status: "passed"},
			{Name: "business", Status: "passed"},
			{Name: "unit", Status: "passed"},
		},
	}}
	raw, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "coverage", "runs.index.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestGetScoreStalenessLoop is the end-to-end honesty check: a scenario
// whose run index is stamped at the current digest reads fresh; editing any
// source file flips every phase to stale with the copy-pastable refresh
// command; restoring the byte-state returns to fresh.
//
// The run index itself lives under coverage/ which treedigest excludes, so
// stamping it does not perturb the digest it records.
func TestGetScoreStalenessLoop(t *testing.T) {
	scenariosRoot := t.TempDir()
	root := writeFixtureScenario(t, scenariosRoot, "fixture")
	stampRunIndex(t, root)

	svc, err := New(WithScenariosRoot(scenariosRoot), WithClock(func() time.Time {
		return time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	}))
	if err != nil {
		t.Fatal(err)
	}

	verdicts := func() map[string]string {
		res, err := svc.GetScore("fixture")
		if err != nil {
			t.Fatalf("GetScore: %v", err)
		}
		out := map[string]string{}
		for _, p := range res.Freshness.Phases {
			out[p.Phase] = p.Verdict
		}
		return out
	}

	for phase, v := range verdicts() {
		if v != "fresh" {
			t.Fatalf("pre-edit phase %s = %q, want fresh", phase, v)
		}
	}

	// Edit a source file -> every verdict flips stale, refresh command names
	// the quick preset.
	appPath := filepath.Join(root, "ui", "src", "App.tsx")
	original, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appPath, append(original, []byte("// edited\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := svc.GetScore("fixture")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range res.Freshness.Phases {
		if p.Verdict != "stale" {
			t.Fatalf("post-edit phase %s = %q, want stale", p.Phase, p.Verdict)
		}
	}
	if res.Freshness.SuggestedCommand != "test-genie execute fixture --preset quick" {
		t.Fatalf("suggested command = %q", res.Freshness.SuggestedCommand)
	}

	// Revert -> fresh again (digest is content-addressed, not mtime-based).
	if err := os.WriteFile(appPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	for phase, v := range verdicts() {
		if v != "fresh" {
			t.Fatalf("post-revert phase %s = %q, want fresh", phase, v)
		}
	}
}

// TestGetScoreAssemblesAllSections sanity-checks the assembled payload on
// the fixture scenario.
func TestGetScoreAssemblesAllSections(t *testing.T) {
	scenariosRoot := t.TempDir()
	root := writeFixtureScenario(t, scenariosRoot, "fixture")
	stampRunIndex(t, root)

	svc, err := New(WithScenariosRoot(scenariosRoot))
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.GetScore("fixture")
	if err != nil {
		t.Fatal(err)
	}

	if res.Scenario != "fixture" || res.Category != "utility" {
		t.Fatalf("identity wrong: %+v", res)
	}
	if res.Composite.Score <= 0 || res.Composite.Classification == "" {
		t.Fatalf("composite empty: %+v", res.Composite)
	}
	if !res.Maturity.BuildPassing {
		t.Fatal("unit passed but build not passing")
	}
	if res.Freshness.Digest == "" {
		t.Fatal("digest missing")
	}
	if len(res.Degradations) != 0 {
		t.Fatalf("unexpected degradations: %+v", res.Degradations)
	}
	// 1 of 2 requirements passing -> a high recommendation must exist.
	if len(res.Recommends) == 0 || len(res.ActionPlan) == 0 {
		t.Fatalf("expected recommendations and plan: %+v", res.Recommends)
	}
}

func TestGetScoreUnknownScenario(t *testing.T) {
	svc, err := New(WithScenariosRoot(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetScore("nope"); !errors.Is(err, ErrUnknownScenario) {
		t.Fatalf("err = %v, want ErrUnknownScenario", err)
	}
	if _, err := svc.GetScore("../escape"); !errors.Is(err, ErrUnknownScenario) {
		t.Fatalf("path traversal not rejected: %v", err)
	}
}
