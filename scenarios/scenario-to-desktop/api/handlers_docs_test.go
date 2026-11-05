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

func TestDocsFileAndIconPreviewHandlersEnforcePathBoundaries(t *testing.T) {
	docsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte("# Desktop Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "notes.txt"), []byte("plain notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_TO_DESKTOP_DOCS_DIR", docsDir)
	server := NewServer(0)
	t.Cleanup(func() { shutdownServer(t, server) })

	for _, check := range []struct{ path, contentType, body string }{
		{"/docs/guide.md", "text/markdown; charset=utf-8", "# Desktop Guide\n"},
		{"/docs/notes.txt", "text/plain; charset=utf-8", "plain notes"},
	} {
		req := httptest.NewRequest(http.MethodGet, check.path, nil)
		response := httptest.NewRecorder()
		server.docsFileHandler(response, req)
		if response.Code != http.StatusOK || response.Header().Get("Content-Type") != check.contentType || response.Body.String() != check.body {
			t.Fatalf("docs file %q = status %d type %q body %q", check.path, response.Code, response.Header().Get("Content-Type"), response.Body.String())
		}
	}
	for _, path := range []string{"/docs/", "/docs/../secrets", "/docs/missing.md"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		server.docsFileHandler(response, req)
		if response.Code != http.StatusNotFound && response.Code != http.StatusBadRequest {
			t.Fatalf("invalid docs path %q status = %d", path, response.Code)
		}
	}
	preview := httptest.NewRecorder()
	server.iconPreviewHandler(preview, httptest.NewRequest(http.MethodGet, "/api/v1/icons/preview?path=scenarios/scenario-to-desktop/ui/public/public/logo.png", nil))
	if preview.Code != http.StatusOK || preview.Body.Len() == 0 {
		t.Fatalf("icon preview status=%d size=%d", preview.Code, preview.Body.Len())
	}
	for _, query := range []string{"", "?path=README.md", "?path=../../outside.png"} {
		response := httptest.NewRecorder()
		server.iconPreviewHandler(response, httptest.NewRequest(http.MethodGet, "/api/v1/icons/preview"+query, nil))
		if response.Code != http.StatusBadRequest && response.Code != http.StatusForbidden {
			t.Fatalf("icon query %q status=%d", query, response.Code)
		}
	}
}
