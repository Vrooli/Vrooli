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
	"strings"
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

func TestHandleUpload_ValidPNG(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	// Minimal PNG header bytes
	pngData := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	req := createUploadRequest(t, sess.ID, "test.png", "image/png", pngData)
	rr := httptest.NewRecorder()

	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	path, ok := resp["path"]
	if !ok || path == "" {
		t.Fatal("response missing path")
	}

	// Verify file exists on disk
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("uploaded file does not exist at %s", path)
	}
	// Clean up
	os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID))
}

func TestHandleUpload_AcceptsJPEG(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	req := createUploadRequest(t, sess.ID, "photo.jpg", "image/jpeg", []byte("fake jpeg"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID))
}

func TestHandleUpload_AcceptsWebP(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	req := createUploadRequest(t, sess.ID, "img.webp", "image/webp", []byte("fake webp"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID))
}

func TestHandleUpload_AcceptsGIF(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	req := createUploadRequest(t, sess.ID, "anim.gif", "image/gif", []byte("GIF89a"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID))
}

func TestHandleUpload_RejectsNonImageMIME(t *testing.T) {
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	req := createUploadRequest(t, sess.ID, "script.sh", "text/plain", []byte("#!/bin/bash"))
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
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	// Create a body that exceeds 20MB
	bigData := make([]byte, 21<<20)
	req := createUploadRequest(t, sess.ID, "huge.png", "image/png", bigData)
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge && rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 413 or 400, got %d: %s", rr.Code, rr.Body.String())
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
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer func() { _ = srv.sessions.Delete(context.Background(), sess.ID) }()

	req := createUploadRequest(t, sess.ID, "../../../etc/passwd", "image/png", []byte("data"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The path should be within the session directory, not contain traversal
	if !strings.HasPrefix(resp["path"], filepath.Join(resolveUploadDir(), sess.ID)) {
		t.Fatalf("path escaped session dir: %s", resp["path"])
	}
	if strings.Contains(resp["path"], "..") {
		t.Fatalf("path contains traversal: %s", resp["path"])
	}

	os.RemoveAll(filepath.Join(resolveUploadDir(), sess.ID))
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
