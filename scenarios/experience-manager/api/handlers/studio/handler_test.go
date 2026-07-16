package studio

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"experience-manager/internal/authoring"
	"experience-manager/internal/reconcile"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
)

type fakeEvidenceRepository struct {
	filter reconcile.EvidenceFilter
	rows   []reconcile.Evidence
}

func (r *fakeEvidenceRepository) SaveEvidence(context.Context, reconcile.Evidence) error {
	return nil
}

func (r *fakeEvidenceRepository) ListEvidence(_ context.Context, filter reconcile.EvidenceFilter) ([]reconcile.Evidence, error) {
	r.filter = filter
	return r.rows, nil
}

func TestStartAuthoringSessionRequiresRepository(t *testing.T) {
	h := &handler{service: authoring.Service{RepoRoot: t.TempDir()}}

	_, err := h.StartAuthoringSession(context.Background(), connect.NewRequest(&contractv1.StartAuthoringSessionRequest{
		Scenario: "demo",
	}))
	if err == nil {
		t.Fatal("StartAuthoringSession succeeded without repository")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("code = %s, want internal", got)
	}
}

func TestListSpecReturnsParsedPages(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	writeStudioFixture(t, scenarioDir)
	h := &handler{service: authoring.Service{RepoRoot: root}}

	resp, err := h.ListSpec(context.Background(), connect.NewRequest(&contractv1.ListSpecRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ListSpec: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" || len(resp.Msg.GetPages()) != 1 || len(resp.Msg.GetComponents()) != 1 {
		t.Fatalf("response = %+v", resp.Msg)
	}
	page := resp.Msg.GetPages()[0]
	if page.GetId() != "home" || page.GetPath() != "pages/home.json" || page.GetStatus() != "active" {
		t.Fatalf("page = %+v", page)
	}
	component := resp.Msg.GetComponents()[0]
	if component.GetId() != "button" || component.GetPath() != "components/button.json" || component.GetStatus() != "active" {
		t.Fatalf("component = %+v", component)
	}
}

func TestListEvidenceReturnsRepositoryRows(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	writeStudioFixture(t, scenarioDir)
	repo := &fakeEvidenceRepository{rows: []reconcile.Evidence{{
		ID:         "ev-1",
		Scenario:   "demo",
		PageID:     "home",
		Route:      "/",
		StateID:    "default",
		ClaimID:    "primary-present",
		ClaimType:  "element-present",
		Verdict:    "passed",
		CaptureRef: "scenario=demo,path=/",
		AXNodeJSON: `{"role":"button"}`,
		Message:    "claim proven",
		CheckedAt:  "2026-07-05T12:00:00Z",
	}}}
	h := &handler{service: authoring.Service{RepoRoot: root, Evidence: repo}}

	resp, err := h.ListEvidence(context.Background(), connect.NewRequest(&contractv1.ListEvidenceRequest{
		Scenario: "demo",
		Page:     "home",
		Claim:    "primary-present",
		Limit:    5,
	}))
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if repo.filter.Scenario != "demo" || repo.filter.PageID != "home" || repo.filter.ClaimID != "primary-present" || repo.filter.Limit != 5 {
		t.Fatalf("filter = %+v", repo.filter)
	}
	if len(resp.Msg.GetEvidence()) != 1 {
		t.Fatalf("evidence rows = %d", len(resp.Msg.GetEvidence()))
	}
	got := resp.Msg.GetEvidence()[0]
	if got.GetId() != "ev-1" || got.GetVerdict() != "passed" || got.GetAxNodeJson() == "" {
		t.Fatalf("evidence = %+v", got)
	}
}

func TestListEvidenceReturnsComponentRows(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	writeStudioFixture(t, scenarioDir)
	repo := &fakeEvidenceRepository{rows: []reconcile.Evidence{{ID: "ev-component", Scenario: "demo", DocumentKind: "component", PageID: "button", ComponentID: "button", ExampleName: "primary", ClaimID: "action-present", Verdict: "passed"}}}
	h := &handler{service: authoring.Service{RepoRoot: root, Evidence: repo}}

	resp, err := h.ListEvidence(context.Background(), connect.NewRequest(&contractv1.ListEvidenceRequest{Scenario: "demo", Component: "button", Limit: 5}))
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if repo.filter.ComponentID != "button" || repo.filter.PageID != "" {
		t.Fatalf("filter = %+v", repo.filter)
	}
	if len(resp.Msg.GetEvidence()) != 1 || resp.Msg.GetEvidence()[0].GetExampleName() != "primary" || resp.Msg.GetEvidence()[0].GetComponentId() != "button" {
		t.Fatalf("response = %+v", resp.Msg)
	}
}

func TestListEvidenceRequiresEvidenceRepository(t *testing.T) {
	h := &handler{service: authoring.Service{RepoRoot: t.TempDir()}}

	_, err := h.ListEvidence(context.Background(), connect.NewRequest(&contractv1.ListEvidenceRequest{
		Scenario: "demo",
		Page:     "home",
	}))
	if err == nil {
		t.Fatal("ListEvidence succeeded without repository")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Fatalf("code = %s, want internal", got)
	}
}

func TestModuleExposesStudioEndpoints(t *testing.T) {
	mod := Module(nil, t.TempDir())
	if mod.Name != "studio" {
		t.Fatalf("module name = %q", mod.Name)
	}
	if len(mod.Endpoints) != 12 {
		t.Fatalf("endpoints = %d", len(mod.Endpoints))
	}
	for _, endpoint := range mod.Endpoints {
		if endpoint.Path == "" || endpoint.Method != "POST" {
			t.Fatalf("invalid endpoint descriptor: %+v", endpoint)
		}
	}
}

func writeStudioFixture(t *testing.T, scenarioDir string) {
	t.Helper()
	files := map[string]string{
		"PRD.md": "## Operational Targets\n- [ ] OT-P0-001 | Demo | Build demo.\n",
		"experience/index.json": `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [{"id":"home","path":"pages/home.json","title":"Home","status":"active"}],
  "journeys": [],
  "components": [{"id":"button","path":"components/button.json","title":"Button","status":"active"}]
}`,
		"experience/pages/home.json": `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"Home page proves studio handler mapping.","prd_refs":["OT-P0-001"]},
  "states": [{"id":"default","description":"Default state."}],
  "elements": [{"id":"primary","role":"button","name":"Primary","description":"Primary action."}],
  "claims": [{"id":"primary-present","type":"element-present","statement":"Primary action is visible.","tier":"machine","elements":["primary"],"states":["default"]}],
  "bindings": {"elements": {"primary": {"testid":"primary-action"}}}
}`,
		"experience/components/button.json": `{
  "kind": "experience-component",
  "schemaVersion": "1.1.0",
  "component": {"id":"button","title":"Button","purpose":"Button component proves studio handler mapping.","examplesRef":"../../library/components/Button/versions/1.2.0/examples.json","prd_refs":["OT-P0-001"]},
  "states": [{"id":"primary","example":"primary","description":"Primary state."}],
  "elements": [{"id":"action","role":"button","name":"Primary","description":"Button action."}],
  "claims": [{"id":"action-present","type":"element-present","statement":"Button action is visible.","tier":"machine","elements":["action"],"states":["primary"]}],
  "bindings": {"elements": {"action": {"selector":"button"}}}
}`,
	}
	for rel, content := range files {
		path := filepath.Join(scenarioDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
