package httpserver

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"

	"io"
	"net/http"
	"net/http/httptest"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestServerResolveScenarioDirUsesScenarioRoot(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}

	server := &Server{
		scenarios: &stubScenarioDirectory{scenarioRoot: root},
		logger:    log.New(io.Discard, "", 0),
	}

	if got := server.resolveScenarioDir("demo"); got != scenarioDir {
		t.Fatalf("resolveScenarioDir() = %q, want %q", got, scenarioDir)
	}
}

func TestServerResolveScenarioDirUsesRepoContractRoot(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	scenarioDir := filepath.Join(repoRoot, "scenarios", "test-genie")
	chdirHTTPTest(t, filepath.Join(repoRoot, "scenarios", "test-genie", "api"))

	server := &Server{logger: log.New(io.Discard, "", 0)}

	if got := server.resolveScenarioDir("test-genie"); got != scenarioDir {
		t.Fatalf("resolveScenarioDir() = %q, want %q", got, scenarioDir)
	}
}

func TestHandleGetConfigUsesRepoContractPaths(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD() error: %v", err)
	}
	chdirHTTPTest(t, filepath.Join(repoRoot, "scenarios", "test-genie", "api"))

	server := &Server{logger: log.New(io.Discard, "", 0)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	rec := httptest.NewRecorder()

	server.handleGetConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got := payload["repoRoot"]; got != repoRoot {
		t.Fatalf("repoRoot = %#v, want %q", got, repoRoot)
	}
	if got := payload["scenariosPath"]; got != filepath.Join(repoRoot, "scenarios") {
		t.Fatalf("scenariosPath = %#v", got)
	}
	if got := payload["testGeniePath"]; got != filepath.Join(repoRoot, "scenarios", "test-genie") {
		t.Fatalf("testGeniePath = %#v", got)
	}
}

func TestServerLoadRequirementsFromFilesBuildsSnapshot(t *testing.T) {
	scenarioDir := t.TempDir()
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	moduleDir := filepath.Join(requirementsDir, "01-internal-orchestrator")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements module dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(requirementsDir, "index.json"), []byte(`{
  "imports": ["01-internal-orchestrator/module.json"],
  "requirements": [
    {
      "id": "INDEX-REQ",
      "title": "Index requirement",
      "status": "complete",
      "validation": [{"type":"test","ref":"requirements/index.json","status":"implemented"}]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("failed to write index.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(moduleDir, "module.json"), []byte(`{
  "requirements": [
    {
      "id": "ORCH-REQ",
      "title": "Orchestrator requirement",
      "status": "in_progress",
      "validation": [{"type":"test","ref":"api/internal/orchestrator/suite_execution_test.go","phase":"integration","status":"implemented"}]
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("failed to write module.json: %v", err)
	}

	server := &Server{logger: log.New(io.Discard, "", 0)}
	snapshot := server.loadRequirementsFromFiles(scenarioDir, "demo")

	if snapshot.ScenarioName != "demo" {
		t.Fatalf("ScenarioName = %q, want demo", snapshot.ScenarioName)
	}
	if snapshot.Summary.TotalRequirements != 2 {
		t.Fatalf("TotalRequirements = %d, want 2", snapshot.Summary.TotalRequirements)
	}
	if snapshot.Summary.TotalValidations != 2 {
		t.Fatalf("TotalValidations = %d, want 2", snapshot.Summary.TotalValidations)
	}
	if len(snapshot.Modules) != 2 {
		t.Fatalf("len(Modules) = %d, want 2", len(snapshot.Modules))
	}
	if snapshot.Summary.ByDeclaredStatus["complete"] != 1 {
		t.Fatalf("complete count = %d, want 1", snapshot.Summary.ByDeclaredStatus["complete"])
	}
	if snapshot.Summary.ByDeclaredStatus["in_progress"] != 1 {
		t.Fatalf("in_progress count = %d, want 1", snapshot.Summary.ByDeclaredStatus["in_progress"])
	}
	if snapshot.Summary.ByLiveStatus["passed"] != 1 {
		t.Fatalf("passed count = %d, want 1", snapshot.Summary.ByLiveStatus["passed"])
	}
	if snapshot.Summary.ByLiveStatus["not_run"] != 1 {
		t.Fatalf("not_run count = %d, want 1", snapshot.Summary.ByLiveStatus["not_run"])
	}
}

func chdirHTTPTest(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}
