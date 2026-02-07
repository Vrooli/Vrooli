package main

import (
	"bytes"
	"context"
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

func TestHandleDocsHealth(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeDocFile(t, filepath.Join(scenario, "README.md"), "# Readme")
	writeDocFile(t, filepath.Join(scenario, "docs", "manifest.json"), "{}")
	writeDocFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")

	service, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	srv := &Server{docHealthService: service}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/alpha/docs/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsHealth(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var decoded ScenarioDocHealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.ScenarioName != "alpha" {
		t.Fatalf("unexpected scenario name: %s", decoded.ScenarioName)
	}
	if decoded.TotalDocs == 0 {
		t.Fatalf("expected total docs to be positive")
	}
}

func TestHandleDocsResetPreview(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := "# Problems\n\n## 2025-01-01: Old\n"
	writeDocFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), content)

	service, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	srv := &Server{docHealthService: service}

	payload, _ := json.Marshal(DocResetRequest{DocType: "problems", MaxAgeDays: 1, Preview: true})
	req := httptest.NewRequest("POST", "/api/v1/scenarios/alpha/docs/reset", bytes.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsReset(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := os.ReadFile(filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(updated) != content {
		t.Fatalf("expected preview to leave file unchanged")
	}
}

func TestHandleDocsHealthErrors(t *testing.T) {
	service, err := dochealth.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	server := &Server{docHealthService: service}

	req := httptest.NewRequest("GET", "/api/v1/scenarios/missing/docs/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "missing"})
	rec := httptest.NewRecorder()

	server.handleDocsHealth(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func writeDocFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestMissingDocSeverity(t *testing.T) {
	if got := missingDocSeverity(docschema.DocTypeReadme); got != "error" {
		t.Fatalf("expected error for readme, got %s", got)
	}
	if got := missingDocSeverity(docschema.DocTypeProgress); got != "warning" {
		t.Fatalf("expected warning for progress, got %s", got)
	}
}

func TestDocHealthServiceUnavailable(t *testing.T) {
	server := &Server{docHealthService: nil}
	req := httptest.NewRequest("GET", "/api/v1/scenarios/alpha/docs/health", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	server.handleDocsHealth(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestDocHealthServiceResetUnsupported(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "alpha")
	if err := os.MkdirAll(filepath.Join(scenario, "docs", "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeDocFile(t, filepath.Join(scenario, "docs", "internal", "PROBLEMS.md"), "# Problems")

	service, err := dochealth.NewService(root)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	server := &Server{docHealthService: service}
	payload, _ := json.Marshal(DocResetRequest{DocType: "readme", MaxAgeDays: 1, Preview: true})
	req := httptest.NewRequest("POST", "/api/v1/scenarios/alpha/docs/reset", bytes.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	server.handleDocsReset(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDocHealthServiceScenarioPathValidation(t *testing.T) {
	service, err := dochealth.NewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if _, err := service.ValidateScenario(context.Background(), "../bad"); err == nil {
		t.Fatalf("expected error for invalid scenario name")
	}
}

func TestComputeFixCategory_IncludesTemporaryDocs(t *testing.T) {
	if got := computeFixCategory(nil, nil, nil, []string{"IMPLEMENTATION_PLAN.md"}); got != "all_agent" {
		t.Fatalf("expected all_agent for temporary docs only, got %s", got)
	}
	if got := computeFixCategory([]docschema.MisplacedDoc{{ActualPath: "a", ExpectedPath: "b"}}, nil, nil, []string{"IMPLEMENTATION_PLAN.md"}); got != "mixed" {
		t.Fatalf("expected mixed when misplaced and temporary docs exist, got %s", got)
	}
}
