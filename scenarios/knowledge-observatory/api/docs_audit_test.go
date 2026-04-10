package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/docschema"
	"knowledge-observatory/internal/services/dochealth"
)

func TestHandleDocsAudit(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeDocFile(t, filepath.Join(scenario, "README.md"), "# Readme")
	writeDocFile(t, filepath.Join(scenario, "docs", "manifest.json"), `{
		"sections": [{"documents": [{"path": "internal/PROBLEMS.md"}]}]
	}`)
	writeDocFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")

	service, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	srv := &Server{docHealthService: service}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/alpha/docs/audit", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsAudit(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result docschema.AuditResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.ScenarioName != "alpha" {
		t.Fatalf("unexpected scenario name: %s", result.ScenarioName)
	}
	if result.Infrastructure == nil {
		t.Fatal("expected infrastructure result")
	}
	if result.TotalDocs < 2 {
		t.Fatalf("expected at least 2 docs, got %d", result.TotalDocs)
	}
}

func TestHandleDocsAudit_NotFound(t *testing.T) {
	service, err := dochealth.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	srv := &Server{docHealthService: service}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/missing/docs/audit", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "missing"})
	rec := httptest.NewRecorder()

	srv.handleDocsAudit(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDocsAudit_ServiceUnavailable(t *testing.T) {
	srv := &Server{docHealthService: nil}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/alpha/docs/audit", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsAudit(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}
