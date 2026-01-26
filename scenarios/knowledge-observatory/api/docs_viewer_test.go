package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"knowledge-observatory/internal/services/viewer"
)

func TestHandleDocsContent(t *testing.T) {
	root, relPath := setupDocViewerServer(t)
	_ = root

	req := httptest.NewRequest("GET", "/api/v1/docs/content?path="+relPath+"&format=raw", nil)
	rec := httptest.NewRecorder()

	srv := &Server{docViewerService: mustViewerService(t, filepath.Join(root, "scenarios"))}
	srv.handleDocsContent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var decoded DocsContentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Path != relPath {
		t.Fatalf("expected path %q, got %q", relPath, decoded.Path)
	}
	if decoded.Content == "" {
		t.Fatalf("expected content")
	}
}

func TestHandleDocsViewerReset(t *testing.T) {
	root, relPath := setupDocViewerServer(t)

	body, _ := json.Marshal(DocsResetRequest{
		Path:           relPath,
		MaxAgeDays:     30,
		KeepMinEntries: 1,
		PreviewOnly:    true,
	})
	req := httptest.NewRequest("POST", "/api/v1/docs/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	srv := &Server{docViewerService: mustViewerService(t, filepath.Join(root, "scenarios"))}
	srv.handleDocsViewerReset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var decoded DocsResetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Path != relPath {
		t.Fatalf("expected path %q, got %q", relPath, decoded.Path)
	}
	if decoded.RemovedCount == 0 {
		t.Fatalf("expected removed entries")
	}
}

func setupDocViewerServer(t *testing.T) (string, string) {
	root := t.TempDir()
	scenariosRoot := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(filepath.Join(scenariosRoot, "alpha", "docs", "internal"), 0o755); err != nil {
		t.Fatalf("failed to create scenario: %v", err)
	}
	content := `# Problems

## 2000-01-01: Ancient issue
- Legacy note

## 2100-01-01: Recent issue
- Current note
`
	file := filepath.Join(scenariosRoot, "alpha", "docs", "internal", "PROBLEMS.md")
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write problems doc: %v", err)
	}
	relPath := filepath.ToSlash(filepath.Join("scenarios", "alpha", "docs", "internal", "PROBLEMS.md"))
	return root, relPath
}

func mustViewerService(t *testing.T, root string) *viewer.Service {
	t.Helper()
	svc, err := viewer.NewService(root)
	if err != nil {
		t.Fatalf("failed to create viewer service: %v", err)
	}
	return svc
}
