package httpserver

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"

	appruns "test-genie/internal/app/runs"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
)

func TestOpaqueRunArtifactHandlerStreamsCatalogBytesWithSafeHeaders(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	runID := "run-safe"
	if err := sharedruns.NewIndex(scenarioDir).Append(sharedruns.RunRecord{
		RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
	}); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(sharedartifacts.RunDir(scenarioDir, runID), "dom.html")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("<script>unsafe()</script>"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Unix(100, 0), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedartifacts.WriteArtifactCatalog(scenarioDir, catalog); err != nil {
		t.Fatal(err)
	}
	server := &Server{
		runsService: appruns.NewService(root, nil, nil, nil),
		logger:      log.New(io.Discard, "", 0),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/demo/runs/run-safe/artifacts/"+catalog.Artifacts[0].ID, nil)
	req = mux.SetURLVars(req, map[string]string{"name": "demo", "runId": runID, "artifactId": catalog.Artifacts[0].ID})
	rec := httptest.NewRecorder()
	server.handleGetRunArtifactByID(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Fatal("HTML evidence is missing an inert sandbox policy")
	}
	if rec.Body.String() != "<script>unsafe()</script>" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestOpaqueRunArtifactHandlerRejectsInvalidAndForeignIDs(t *testing.T) {
	root := t.TempDir()
	server := &Server{runsService: appruns.NewService(root, nil, nil, nil), logger: log.New(io.Discard, "", 0)}

	req := httptest.NewRequest(http.MethodGet, "/artifacts/..%2F..%2Fetc%2Fpasswd", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "demo", "runId": "run-a", "artifactId": "../../etc/passwd"})
	rec := httptest.NewRecorder()
	server.handleGetRunArtifactByID(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("encoded traversal status = %d", rec.Code)
	}

	for _, runID := range []string{"run-a", "run-b"} {
		scenarioDir := filepath.Join(root, "demo")
		if err := sharedruns.NewIndex(scenarioDir).Append(sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(sharedartifacts.RunDir(scenarioDir, runID), "evidence.txt")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(runID), 0o644); err != nil {
			t.Fatal(err)
		}
		catalog, err := sharedartifacts.DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Unix(100, 0), false)
		if err != nil {
			t.Fatal(err)
		}
		if err := sharedartifacts.WriteArtifactCatalog(scenarioDir, catalog); err != nil {
			t.Fatal(err)
		}
	}
	foreign, err := sharedartifacts.ReadArtifactCatalog(filepath.Join(root, "demo"), "run-b")
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest(http.MethodGet, "/artifacts/"+foreign.Artifacts[0].ID, nil)
	req = mux.SetURLVars(req, map[string]string{"name": "demo", "runId": "run-a", "artifactId": foreign.Artifacts[0].ID})
	rec = httptest.NewRecorder()
	server.handleGetRunArtifactByID(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("foreign ID status = %d body=%s", rec.Code, rec.Body.String())
	}
}
