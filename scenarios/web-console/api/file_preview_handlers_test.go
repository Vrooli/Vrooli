package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"

	filePreviewH "web-console/handlers/file_preview"
	"web-console/internal/filepreview"
)

// newFilePreviewTestServer builds a fake-PTY server wired with a filepreview
// resolver rooted at root and an in-memory preview store.
func newFilePreviewTestServer(t *testing.T, root string) *Server {
	t.Helper()
	srv := newFakeTestServer()
	srv.filePreviewResolver = &filepreview.Resolver{ProjectRoot: root}
	srv.filePreviews = filepreview.NewStore(time.Minute)
	return srv
}

func newPreviewSession(t *testing.T, srv *Server) string {
	t.Helper()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.ID) })
	return sess.ID
}

func resolvePreview(t *testing.T, srv *Server, sessionID, path string) filePreviewH.ResolveResult {
	t.Helper()
	res, err := newFilePreviewAdapter(srv).Resolve(context.Background(), filePreviewH.ResolveInput{
		SessionID: sessionID,
		Path:      path,
	})
	if err != nil {
		t.Fatalf("resolve %q: %v", path, err)
	}
	return res
}

func blobRequest(t *testing.T, srv *Server, method, sessionID, previewID string, rangeHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/v1/sessions/"+sessionID+"/file-previews/"+previewID+"/blob", nil)
	req = mux.SetURLVars(req, map[string]string{"id": sessionID, "previewId": previewID})
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	rec := httptest.NewRecorder()
	srv.handleFilePreviewBlob(rec, req)
	return rec
}

func TestFilePreview_ResolveTextAndGetContent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "plan.md"), []byte("# Plan\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)

	res := resolvePreview(t, srv, sessionID, "docs/plan.md:2")
	if res.Kind != "markdown" || !res.TextContentAvailable {
		t.Fatalf("kind=%q textAvail=%v", res.Kind, res.TextContentAvailable)
	}
	if res.PreviewID == "" || res.BlobURL == "" {
		t.Fatalf("missing preview id/url: %+v", res)
	}
	if res.Line != 2 || !res.HasLine {
		t.Fatalf("line=%d hasLine=%v", res.Line, res.HasLine)
	}

	text, err := newFilePreviewAdapter(srv).GetTextContent(context.Background(), sessionID, res.PreviewID)
	if err != nil {
		t.Fatalf("get text: %v", err)
	}
	if text.Content != "# Plan\nbody\n" {
		t.Fatalf("content=%q", text.Content)
	}
}

func TestFilePreview_GetTextContent_RejectsMedia(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.png"), pngBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "a.png")
	if res.Kind != "image" || !res.SupportsRange {
		t.Fatalf("kind=%q supportsRange=%v", res.Kind, res.SupportsRange)
	}
	if res.TextContentAvailable {
		t.Fatal("image must not offer text content")
	}
	if _, err := newFilePreviewAdapter(srv).GetTextContent(context.Background(), sessionID, res.PreviewID); err == nil {
		t.Fatal("expected GetTextContent to reject an image preview")
	}
}

func TestFilePreview_Blob_FullContentAndHeaders(t *testing.T) {
	root := t.TempDir()
	body := pngBytes()
	if err := os.WriteFile(filepath.Join(root, "logo.png"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "logo.png")

	rec := blobRequest(t, srv, http.MethodGet, sessionID, res.PreviewID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if got := rec.Body.Bytes(); len(got) != len(body) {
		t.Fatalf("body len=%d want %d", len(got), len(body))
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type=%q", ct)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Fatal("missing Accept-Ranges")
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Fatal("missing Content-Disposition")
	}
}

func TestFilePreview_Blob_Head(t *testing.T) {
	root := t.TempDir()
	body := pngBytes()
	if err := os.WriteFile(filepath.Join(root, "logo.png"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "logo.png")

	rec := blobRequest(t, srv, http.MethodHead, sessionID, res.PreviewID, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD should have empty body, got %d bytes", rec.Body.Len())
	}
}

func TestFilePreview_Blob_Range(t *testing.T) {
	root := t.TempDir()
	body := make([]byte, 1000)
	for i := range body {
		body[i] = byte(i % 251)
	}
	if err := os.WriteFile(filepath.Join(root, "clip.mp4"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "clip.mp4")

	rec := blobRequest(t, srv, http.MethodGet, sessionID, res.PreviewID, "bytes=100-199")
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status=%d want 206", rec.Code)
	}
	if cr := rec.Header().Get("Content-Range"); cr != "bytes 100-199/1000" {
		t.Fatalf("content-range=%q", cr)
	}
	if rec.Body.Len() != 100 {
		t.Fatalf("partial body len=%d want 100", rec.Body.Len())
	}
}

func TestFilePreview_Blob_InvalidRange(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.bin.mp4"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "a.bin.mp4")

	rec := blobRequest(t, srv, http.MethodGet, sessionID, res.PreviewID, "bytes=5000-6000")
	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("status=%d want 416", rec.Code)
	}
}

func TestFilePreview_Blob_UnknownID(t *testing.T) {
	srv := newFilePreviewTestServer(t, t.TempDir())
	sessionID := newPreviewSession(t, srv)
	rec := blobRequest(t, srv, http.MethodGet, sessionID, "deadbeefdeadbeef", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}

func TestFilePreview_Blob_SessionMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.png"), pngBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "a.png")

	rec := blobRequest(t, srv, http.MethodGet, "some-other-session", res.PreviewID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 for session mismatch", rec.Code)
	}
}

func TestFilePreview_Blob_FileChanged(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.png")
	if err := os.WriteFile(fp, pngBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "a.png")

	// Rewrite with different size + a forced mtime change.
	if err := os.WriteFile(fp, append(pngBytes(), 0xff, 0xfe), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(fp, future, future)

	rec := blobRequest(t, srv, http.MethodGet, sessionID, res.PreviewID, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d want 409 for changed file", rec.Code)
	}
}

func TestFilePreview_Blob_DeletedFile(t *testing.T) {
	root := t.TempDir()
	fp := filepath.Join(root, "a.png")
	if err := os.WriteFile(fp, pngBytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	res := resolvePreview(t, srv, sessionID, "a.png")
	if err := os.Remove(fp); err != nil {
		t.Fatal(err)
	}
	rec := blobRequest(t, srv, http.MethodGet, sessionID, res.PreviewID, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404 for deleted file", rec.Code)
	}
}

func TestFilePreview_Resolve_SessionNotFound(t *testing.T) {
	srv := newFilePreviewTestServer(t, t.TempDir())
	_, err := newFilePreviewAdapter(srv).Resolve(context.Background(), filePreviewH.ResolveInput{
		SessionID: "nope",
		Path:      "a.txt",
	})
	if err == nil {
		t.Fatal("expected session-not-found error")
	}
}

// pngBytes returns a minimal valid PNG signature + IHDR-ish header so
// classification detects image/png.
func pngBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	}
}
