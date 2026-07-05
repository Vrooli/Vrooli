package autofix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCaseScaffoldPreviewApplyAndIdempotency(t *testing.T) { // [REQ:EXPERIEN-P1-002] [REQ:EXPERIEN-P1-003]
	root := fixtureScenario(t, true)
	reg := NewRegistry()

	preview, err := reg.Preview(root, []string{RuleCaseScaffold})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if len(preview) != 2 {
		t.Fatalf("preview candidates = %d, want case + registry: %+v", len(preview), preview)
	}
	casePath := filepath.Join(root, "bas", "cases", "experience-spec", "home.json")
	if _, err := os.Stat(casePath); !os.IsNotExist(err) {
		t.Fatalf("preview wrote case file")
	}

	applied, err := ApplySequential(reg, root, []string{RuleCaseScaffold})
	if err != nil {
		t.Fatalf("ApplySequential: %v", err)
	}
	if len(applied) != len(preview) {
		t.Fatalf("applied candidates = %d, want %d", len(applied), len(preview))
	}
	data, err := os.ReadFile(casePath)
	if err != nil {
		t.Fatalf("read scaffolded case: %v", err)
	}
	if !strings.Contains(string(data), `"spec_entry_id": "home"`) || !strings.Contains(string(data), `[data-testid=\"primary-action\"]`) {
		t.Fatalf("scaffolded case missing spec linkage or selector:\n%s", data)
	}
	again, err := ApplySequential(reg, root, []string{RuleCaseScaffold})
	if err != nil {
		t.Fatalf("second ApplySequential: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second apply should be no-op, got %+v", again)
	}
}

func TestBindingDriftRepairAddsDeterministicPlaceholders(t *testing.T) { // [REQ:EXPERIEN-P1-003]
	root := fixtureScenario(t, false)
	reg := NewRegistry()
	applied, err := ApplySequential(reg, root, []string{RuleBindingDriftRepair})
	if err != nil {
		t.Fatalf("ApplySequential: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("applied candidates = %d, want 1: %+v", len(applied), applied)
	}
	page, err := os.ReadFile(filepath.Join(root, "experience", "pages", "home.json"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !strings.Contains(string(page), `"testid": "primary"`) {
		t.Fatalf("binding placeholder missing:\n%s", page)
	}
}

func fixtureScenario(t *testing.T, withBinding bool) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"PRD.md": `# Fixture

## Operational Targets
- [ ] OT-P0-005 | UI page | Build the page.
`,
		"experience/index.json": `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "description": "Fixture experience contract.",
  "pages": [{"id": "home", "path": "pages/home.json", "title": "Home", "status": "active"}],
  "journeys": []
}`,
		"experience/pages/home.json": pageFixture(withBinding),
		"bas/registry.json":          `{"scenario":"demo","metadata":{"execution_mode":"observer"},"playbooks":[]}`,
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func pageFixture(withBinding bool) string {
	binding := `{}`
	if withBinding {
		binding = `{"primary": {"testid": "primary-action"}}`
	}
	return `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {
    "id": "home",
    "title": "Home",
    "routes": ["/"],
    "purpose": "Home page proves scaffold generation.",
    "prd_refs": ["OT-P0-005"]
  },
  "priorities": [{"statement": "Primary action first.", "notes": ""}],
  "states": [{"id": "default", "description": "Default state."}],
  "elements": [{"id": "primary", "role": "button", "name": "Primary", "description": "Primary action."}],
  "claims": [{
    "id": "primary-visible",
    "type": "element-present",
    "statement": "Primary action is visible.",
    "tier": "machine",
    "elements": ["primary"],
    "states": ["default"]
  }],
  "bindings": {"elements": ` + binding + `},
  "sketch": {"regions": [{"id": "main", "elements": ["primary"]}]}
}`
}
