package fleet

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSweepComputesWorstFirstDebt(t *testing.T) { // [REQ:EXPERIEN-P1-005]
	root := t.TempDir()
	scenariosDir := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(filepath.Join(scenariosDir, "empty", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenariosDir, "empty", ".vrooli", "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeScenario(t, scenariosDir, "clean")
	summary, err := Sweep(context.Background(), root)
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if summary.ScenarioCount != 2 || summary.WithExperienceCount != 1 || summary.TotalPages != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Scenarios[0].Scenario != "empty" {
		t.Fatalf("first scenario = %s, want empty", summary.Scenarios[0].Scenario)
	}
	if summary.Scenarios[1].MaxDepth != 2 || summary.Scenarios[1].Status != "green" {
		t.Fatalf("clean row = %+v", summary.Scenarios[1])
	}
}

func writeScenario(t *testing.T, root, name string) {
	t.Helper()
	scenarioDir := filepath.Join(root, name)
	exp := filepath.Join(root, name, "experience")
	if err := os.MkdirAll(filepath.Join(exp, "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	index := `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "` + name + `",
  "description": "demo",
  "pages": [{"id": "home", "path": "pages/home.json", "title": "Home", "status": "active"}],
  "journeys": []
}`
	page := `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id": "home", "title": "Home", "routes": ["/"], "purpose": "Show a useful home page", "prd_refs": []},
  "priorities": [{"statement": "Primary work is visible", "notes": ""}],
  "states": [],
  "elements": [{"id": "title", "role": "heading", "name": "Home", "description": "Main heading"}],
  "claims": [{"id": "title-visible", "type": "element-present", "statement": "The title is visible", "tier": "machine", "elements": ["title"], "states": [], "viewports": [], "locales": [], "params": {}, "rationale": ""}],
  "bindings": {"elements": {"title": {"testid": "home-title", "selector": "", "note": ""}}},
  "sketch": {"regions": [{"id": "main", "elements": ["title"]}]}
}`
	if err := os.WriteFile(filepath.Join(exp, "index.json"), []byte(index), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(exp, "pages", "home.json"), []byte(page), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "DESIGN.md"), []byte("# Design\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
