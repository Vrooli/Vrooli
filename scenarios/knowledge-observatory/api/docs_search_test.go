package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/dochealth"
	"knowledge-observatory/internal/services/docsearch"
	"knowledge-observatory/internal/services/explorer"
)

func TestHandleDocsSearchFiles(t *testing.T) {
	root, srv := setupDocSearchServer(t)
	_ = root

	body, _ := json.Marshal(FileSearchRequest{
		Pattern:  "README.md",
		Scope:    "scenario",
		Scenario: "alpha",
	})
	req := httptest.NewRequest("POST", "/api/v1/docs/search/files", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleDocsSearchFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded []FileSearchResult
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("expected results")
	}
}

func TestHandleDocsSearchText(t *testing.T) {
	_, srv := setupDocSearchServer(t)
	body, _ := json.Marshal(TextSearchRequest{
		Query:    "hello",
		Scope:    "scenario",
		Scenario: "alpha",
	})
	req := httptest.NewRequest("POST", "/api/v1/docs/search/text", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleDocsSearchText(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded []TextSearchMatch
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("expected matches")
	}
}

func TestHandleListScenarios(t *testing.T) {
	root, srv := setupDocSearchServer(t)
	srv.config = &Config{ScenariosRoot: filepath.Join(root, "scenarios")}

	req := httptest.NewRequest("GET", "/api/v1/scenarios", nil)
	rec := httptest.NewRecorder()

	srv.handleListScenarios(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded []ScenarioSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("expected scenarios")
	}
}

func TestHandleDocsTree(t *testing.T) {
	_, srv := setupDocSearchServer(t)

	req := httptest.NewRequest("GET", "/api/v1/scenarios/alpha/docs", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "alpha"})
	rec := httptest.NewRecorder()

	srv.handleDocsTree(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded explorer.DocTreeNode
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Name != "alpha" {
		t.Fatalf("expected root node to be alpha, got %s", decoded.Name)
	}
	if len(decoded.Children) == 0 {
		t.Fatalf("expected doc tree children")
	}
}

func setupDocSearchServer(t *testing.T) (string, *Server) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosRoot, 0o755); err != nil {
		t.Fatalf("failed to create scenarios root: %v", err)
	}
	makeScenario(t, scenariosRoot, "alpha", []fixtureFile{
		{Path: "README.md", Content: "Alpha Scenario"},
		{Path: "docs/guides/intro.md", Content: "hello world"},
	})

	searchService, err := docsearch.NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("failed to create docsearch service: %v", err)
	}
	healthService, err := dochealth.NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("failed to create dochealth service: %v", err)
	}
	explorerService, err := explorer.NewService(scenariosRoot, healthService)
	if err != nil {
		t.Fatalf("failed to create explorer service: %v", err)
	}
	return root, &Server{docSearchService: searchService, docHealthService: healthService, docExplorerService: explorerService}
}

type fixtureFile struct {
	Path    string
	Content string
}

func makeScenario(t *testing.T, scenariosRoot, name string, files []fixtureFile) {
	path := filepath.Join(scenariosRoot, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	for _, file := range files {
		full := filepath.Join(path, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(full, []byte(file.Content), 0o644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}
}
