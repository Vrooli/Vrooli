package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func thumbnailRequest(srv *Server, sessionID, previewID string, query url.Values, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/file-previews/"+previewID+"/thumbnail?"+query.Encode(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": sessionID, "previewId": previewID})
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	srv.handleFilePreviewThumbnail(rec, req)
	return rec
}

func realPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewNRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func thumbnailFixture(t *testing.T) (*Server, string, string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "shot.png"), realPNG(t, 600, 300), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := newFilePreviewTestServer(t, root)
	sessionID := newPreviewSession(t, srv)
	dir := resolvePreview(t, srv, sessionID, root)
	file := resolvePreview(t, srv, sessionID, filepath.Join(root, "shot.png"))
	return srv, sessionID, dir.PreviewID, file.PreviewID
}

func TestFilePreview_Thumbnail_ServesADirectoryChildByName(t *testing.T) {
	srv, sessionID, dirID, _ := thumbnailFixture(t)
	rec := thumbnailRequest(srv, sessionID, dirID, url.Values{"entry": {"shot.png"}, "size": {"120"}}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content type %q", ct)
	}
	img, err := png.Decode(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 120 || b.Dy() != 60 {
		t.Fatalf("size %dx%d, want 120x60", b.Dx(), b.Dy())
	}
	if rec.Header().Get("ETag") == "" || rec.Header().Get("Cache-Control") != "private, max-age=600" {
		t.Fatalf("missing cache headers: %v", rec.Header())
	}
}

func TestFilePreview_Thumbnail_ServesAFilePreviewDirectly(t *testing.T) {
	srv, sessionID, _, fileID := thumbnailFixture(t)
	if rec := thumbnailRequest(srv, sessionID, fileID, url.Values{}, nil); rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if rec := thumbnailRequest(srv, sessionID, fileID, url.Values{"entry": {"other.png"}}, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("entry on a file preview: status=%d want 400", rec.Code)
	}
}

func TestFilePreview_Thumbnail_AnswersRevalidationWithoutDecoding(t *testing.T) {
	srv, sessionID, dirID, _ := thumbnailFixture(t)
	q := url.Values{"entry": {"shot.png"}}
	first := thumbnailRequest(srv, sessionID, dirID, q, nil)
	again := thumbnailRequest(srv, sessionID, dirID, q, http.Header{"If-None-Match": {first.Header().Get("ETag")}})
	if again.Code != http.StatusNotModified || again.Body.Len() != 0 {
		t.Fatalf("status=%d len=%d, want an empty 304", again.Code, again.Body.Len())
	}
}

func TestFilePreview_Thumbnail_RejectsEscapesAndOtherSessions(t *testing.T) {
	srv, sessionID, dirID, _ := thumbnailFixture(t)
	for _, entry := range []string{"", ".", "..", "../etc/passwd", "sub/shot.png", `..\shot.png`} {
		if rec := thumbnailRequest(srv, sessionID, dirID, url.Values{"entry": {entry}}, nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("entry %q: status=%d want 400", entry, rec.Code)
		}
	}
	if rec := thumbnailRequest(srv, "another-session", dirID, url.Values{"entry": {"shot.png"}}, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("session mismatch: status=%d want 404", rec.Code)
	}
	if rec := thumbnailRequest(srv, sessionID, dirID, url.Values{"entry": {"missing.png"}}, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("missing child: status=%d want 404", rec.Code)
	}
}

func TestFilePreview_Thumbnail_UnsupportedFormatIsNotAServerFault(t *testing.T) {
	srv, sessionID, dirID, _ := thumbnailFixture(t)
	rec := thumbnailRequest(srv, sessionID, dirID, url.Values{"entry": {"notes.txt"}}, nil)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want 415", rec.Code)
	}
	if rec.Header().Get("ETag") != "" || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("a refusal must not be cached: %v", rec.Header())
	}
}
