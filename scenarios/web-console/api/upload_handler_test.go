package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

// createUploadRequest builds a multipart POST request with the given file content and content type.
func createUploadRequest(t *testing.T, sessionID, filename, contentType string, body []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write part: %v", err)
	}
	w.Close()

	req := httptest.NewRequest("POST", "/api/v1/sessions/"+sessionID+"/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = mux.SetURLVars(req, map[string]string{"id": sessionID})
	return req
}

func newSessionForUpload(t *testing.T) (*Server, string) {
	t.Helper()
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.ID) })
	// Session IDs are fresh, so cleaning this subtree is safe after failure too.
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID)) })
	return srv, sess.ID
}

func TestHandleUpload_AcceptsSupportedFileTypes(t *testing.T) {
	// Category follows the filename extension, not the client-declared MIME.
	// Keep exact outputs and bytes visible for every accepted family.
	cases := []struct {
		name, filename, mime string
		body                 []byte
		category             uploadCategory
	}{
		{name: "PNG", filename: "test.png", mime: "image/png", body: []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, category: uploadImage},
		{name: "JPEG", filename: "photo.jpg", mime: "image/jpeg", body: []byte("fake jpeg"), category: uploadImage},
		{name: "uppercase JPEG extension with unrelated MIME", filename: "PHOTO.JPEG", mime: "application/octet-stream", body: []byte("upper jpeg payload"), category: uploadImage},
		{name: "WebP", filename: "img.webp", mime: "image/webp", body: []byte("fake webp"), category: uploadImage},
		{name: "GIF", filename: "anim.gif", mime: "image/gif", body: []byte("GIF89a"), category: uploadImage},
		{name: "MIME does not override extension", filename: "diagram.png", mime: "text/plain", body: []byte("PNG payload"), category: uploadImage},
		{name: "PDF", filename: "report.pdf", mime: "application/pdf", body: []byte("%PDF-1.7 fake"), category: uploadPDF},
		{name: "video", filename: "clip.mp4", mime: "video/mp4", body: []byte("fake video"), category: uploadVideo},
		{name: "audio", filename: "voice.mp3", mime: "audio/mpeg", body: []byte("fake audio"), category: uploadAudio},
		{name: "archive", filename: "bundle.zip", mime: "application/zip", body: []byte("PK fake"), category: uploadArchive},
		{name: "markdown", filename: "notes.md", mime: "text/markdown", body: []byte("# notes"), category: uploadText},
		{name: "script", filename: "script.sh", mime: "text/plain", body: []byte("#!/bin/bash\necho hi"), category: uploadText},
		{name: "other", filename: "data.bin", mime: "application/octet-stream", body: []byte{0x00, 0x01, 0x02}, category: uploadOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, sessionID := newSessionForUpload(t)
			req := createUploadRequest(t, sessionID, tc.filename, tc.mime, tc.body)
			rr := httptest.NewRecorder()
			srv.handleUpload(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
			}
			var resp uploadResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if resp.Path != filepath.Join(resolveUploadDir(), sessionID, tc.filename) || resp.Name != tc.filename || resp.SizeBytes != int64(len(tc.body)) || resp.Category != string(tc.category) {
				t.Fatalf("upload response = %+v; want name=%q size=%d category=%q in session directory", resp, tc.filename, len(tc.body), tc.category)
			}
			stored, err := os.ReadFile(resp.Path)
			if err != nil || !bytes.Equal(stored, tc.body) {
				t.Fatalf("stored upload = %v, %v; want original %d bytes", stored, err, len(tc.body))
			}
		})
	}
}

func TestHandleUpload_RejectsExecutableExtension(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	req := createUploadRequest(t, sessionID, "malware.exe", "application/octet-stream", []byte("MZ fake"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "invalid_upload_type" {
		t.Fatalf("expected code invalid_upload_type, got %s", resp.Code)
	}
}

// A benign extension must not launder an executable: the magic bytes decide.
func TestHandleUpload_RejectsExecutableDisguisedAsImage(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	elf := []byte{0x7F, 'E', 'L', 'F', 0x02, 0x01, 0x01, 0x00}
	req := createUploadRequest(t, sessionID, "photo.png", "image/png", elf)
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "invalid_upload_type" {
		t.Fatalf("expected code invalid_upload_type, got %s", resp.Code)
	}
}

func TestHandleUpload_RejectsOversizedFile(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)

	// Images are capped at 25 MiB; 26 MiB must be refused even though the
	// request body ceiling is higher.
	bigData := make([]byte, 26<<20)
	req := createUploadRequest(t, sessionID, "huge.png", "image/png", bigData)
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "upload_too_large" {
		t.Fatalf("expected code upload_too_large, got %s", resp.Code)
	}
	// The partial file must not be left behind.
	files, err := os.ReadDir(filepath.Join(resolveUploadDir(), sessionID))
	if err != nil || len(files) != 0 {
		t.Fatalf("upload directory after size rejection = %v, %v; want no partial file", files, err)
	}
}

func TestHandleUpload_MissingSession(t *testing.T) {
	srv := newFakeTestServer()

	req := createUploadRequest(t, "nonexistent-id", "test.png", "image/png", []byte("data"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "session_not_found" {
		t.Fatalf("expected code session_not_found, got %s", resp.Code)
	}
}

func TestHandleUpload_PathTraversalFilename(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	req := createUploadRequest(t, sessionID, "../../../etc/passwd", "image/png", []byte("data"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Resolve relative to the trusted session directory so a sibling directory
	// with the same prefix cannot pass as contained.
	relative, err := filepath.Rel(filepath.Join(resolveUploadDir(), sessionID), resp.Path)
	if err != nil || relative != "passwd" || resp.Name != "passwd" {
		t.Fatalf("path traversal result = relative %q, name %q, err %v", relative, resp.Name, err)
	}

}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal.png", "normal.png"},
		{"../../../etc/passwd", "passwd"},
		{"", "upload"},
		{".", "upload"},
		{"..", "upload"},
		{"file with spaces.png", "file_with_spaces.png"},
		{"<script>alert.png", "script_alert.png"},
	}

	for _, tc := range tests {
		got := sanitizeFilename(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestUploadCategoryFor(t *testing.T) {
	tests := []struct {
		filename string
		category uploadCategory
		ok       bool
	}{
		{"a.png", uploadImage, true},
		{"a.JPG", uploadImage, true},
		{"a.mp4", uploadVideo, true},
		{"a.mp3", uploadAudio, true},
		{"a.pdf", uploadPDF, true},
		{"a.zip", uploadArchive, true},
		{"a.go", uploadText, true},
		{"a.unknown", uploadOther, true},
		{"a.exe", "", false},
		{"a.SO", "", false},
		{"a.jar", "", false},
	}
	for _, tc := range tests {
		cat, ok := uploadCategoryFor(tc.filename)
		if ok != tc.ok || cat != tc.category {
			t.Errorf("uploadCategoryFor(%q) = (%q, %v), want (%q, %v)", tc.filename, cat, ok, tc.category, tc.ok)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	if got := humanBytes(512); got != "512 B" {
		t.Errorf("humanBytes(512) = %q", got)
	}
	if got := humanBytes(25 << 20); got != "25 MiB" {
		t.Errorf("humanBytes(25MiB) = %q", got)
	}
	if got := humanBytes(512 << 20); got != "512 MiB" {
		t.Errorf("humanBytes(512MiB) = %q", got)
	}
}
