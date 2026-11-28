package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func sqlitePreviewRequest(srv *Server, session, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/blob?representation=sqlite-snapshot", nil)
	req = mux.SetURLVars(req, map[string]string{"id": session, "previewId": id})
	rec := httptest.NewRecorder()
	srv.handleFilePreviewBlob(rec, req)
	return rec
}
func TestFilePreviewSQLiteSnapshotCapabilityAndHeaders(t *testing.T) {
	root := t.TempDir()
	fixture, err := os.ReadFile("../fixtures/file-preview/project-atlas.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(root, "database.sqlite")
	if err = os.WriteFile(filename, fixture, 0600); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	session := newPreviewSession(t, srv)
	resolved := resolvePreview(t, srv, session, filename)
	if resolved.Kind != "database" {
		t.Fatal(resolved.Kind)
	}
	rec := sqlitePreviewRequest(srv, session, resolved.PreviewID)
	if rec.Code != 200 || !bytes.HasPrefix(rec.Body.Bytes(), []byte("SQLite format 3\x00")) {
		t.Fatalf("%d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Type") != "application/vnd.sqlite3" || rec.Header().Get("X-Preview-Snapshot-Time") == "" {
		t.Fatal("missing safe headers", rec.Header())
	}
	if sqlitePreviewRequest(srv, "different-session", resolved.PreviewID).Code != 404 {
		t.Fatal("session mismatch admitted")
	}
	textPath := filepath.Join(root, "text.txt")
	if err = os.WriteFile(textPath, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	text := resolvePreview(t, srv, session, textPath)
	if sqlitePreviewRequest(srv, session, text.PreviewID).Code != 400 {
		t.Fatal("non-database capability admitted")
	}
	if err = os.WriteFile(filename, append(fixture, 0), 0600); err != nil {
		t.Fatal(err)
	}
	if sqlitePreviewRequest(srv, session, resolved.PreviewID).Code != 409 {
		t.Fatal("stale database admitted")
	}
}
func TestFilePreviewSQLiteInvalidKeepsOriginalDownload(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bad.db")
	if err := os.WriteFile(path, bytes.Repeat([]byte("not sqlite\x00"), 20), 0600); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	session := newPreviewSession(t, srv)
	resolved := resolvePreview(t, srv, session, path)
	if sqlitePreviewRequest(srv, session, resolved.PreviewID).Code != 422 {
		t.Fatal("invalid database admitted")
	}
	if blobRequest(t, srv, http.MethodGet, session, resolved.PreviewID, "").Code != 200 {
		t.Fatal("original download lost")
	}
}
