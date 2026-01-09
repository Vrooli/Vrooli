package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDocsHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	manifest := []byte(`{
		"version": "1.0.0",
		"title": "Docs",
		"defaultDocument": "guide.md",
		"sections": [
			{
				"id": "getting-started",
				"title": "Start",
				"documents": [{ "path": "guide.md", "title": "Guide" }]
			}
		]
	}`)

	docContent := "# Hello\nContent"

	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "guide.md"), []byte(docContent), 0o644); err != nil {
		t.Fatalf("failed to write doc: %v", err)
	}

	t.Setenv("SCENARIO_TO_DESKTOP_DOCS_DIR", tmpDir)

	server := NewServer(0)

	t.Run("manifest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs/manifest", nil)
		w := httptest.NewRecorder()

		server.docsManifestHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp DocsManifest
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse manifest response: %v", err)
		}
		if resp.DefaultDocument != "guide.md" {
			t.Fatalf("unexpected default document: %s", resp.DefaultDocument)
		}
	})

	t.Run("content", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs/content?path=guide.md", nil)
		w := httptest.NewRecorder()

		server.docsContentHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp DocsContentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse content response: %v", err)
		}
		if resp.Content == "" {
			t.Fatal("expected content to be populated")
		}
	})

	t.Run("reject traversal", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/docs/content?path=../../etc/passwd", nil)
		w := httptest.NewRecorder()

		server.docsContentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for traversal, got %d", w.Code)
		}
	})
}
