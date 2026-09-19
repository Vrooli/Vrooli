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

func newSessionForUpload(t *testing.T) (*Server, string) {
	t.Helper()
	srv := newFakeTestServer()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.ID) })
	return srv, sess.ID
}

func TestHandleUpload_ValidPNG(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)

	// Minimal PNG header bytes
	pngData := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	req := createUploadRequest(t, sessionID, "test.png", "image/png", pngData)
	rr := httptest.NewRecorder()

	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp uploadResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Path == "" {
		t.Fatal("response missing path")
	}
	if resp.Category != string(uploadImage) {
		t.Fatalf("expected category image, got %q", resp.Category)
	}
	if resp.Name != "test.png" {
		t.Fatalf("expected name test.png, got %q", resp.Name)
	}
	if resp.SizeBytes != int64(len(pngData)) {
		t.Fatalf("expected size %d, got %d", len(pngData), resp.SizeBytes)
	}

	// Verify file exists on disk
	if _, err := os.Stat(resp.Path); os.IsNotExist(err) {
		t.Fatalf("uploaded file does not exist at %s", resp.Path)
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
}

func TestHandleUpload_AcceptsJPEG(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	req := createUploadRequest(t, sessionID, "photo.jpg", "image/jpeg", []byte("fake jpeg"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
}

func TestHandleUpload_AcceptsWebP(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	req := createUploadRequest(t, sessionID, "img.webp", "image/webp", []byte("fake webp"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
}

func TestHandleUpload_AcceptsGIF(t *testing.T) {
	srv, sessionID := newSessionForUpload(t)
	req := createUploadRequest(t, sessionID, "anim.gif", "image/gif", []byte("GIF89a"))
	rr := httptest.NewRecorder()
	srv.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
}

func TestHandleUpload_AcceptsGeneralFileTypes(t *testing.T) {
	cases := []struct {
		filename string
		mime     string
		body     []byte
		category uploadCategory
	}{
		{"report.pdf", "application/pdf", []byte("%PDF-1.7 fake"), uploadPDF},
		{"clip.mp4", "video/mp4", []byte("fake video"), uploadVideo},
		{"voice.mp3", "audio/mpeg", []byte("fake audio"), uploadAudio},
		{"bundle.zip", "application/zip", []byte("PK fake"), uploadArchive},
		{"notes.md", "text/markdown", []byte("# notes"), uploadText},
		{"script.sh", "text/plain", []byte("#!/bin/bash\necho hi"), uploadText},
		{"data.bin", "application/octet-stream", []byte{0x00, 0x01, 0x02}, uploadOther},
	}
	for _, tc := range cases {
		t.Run(tc.filename, func(t *testing.T) {
			srv, sessionID := newSessionForUpload(t)
			req := createUploadRequest(t, sessionID, tc.filename, tc.mime, tc.body)
			rr := httptest.NewRecorder()
			srv.handleUpload(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
			}
			var resp uploadResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if resp.Category != string(tc.category) {
				t.Fatalf("expected category %q, got %q", tc.category, resp.Category)
			}
			os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
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
	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
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

	// The path should be within the session directory, not contain traversal
	if !strings.HasPrefix(resp.Path, filepath.Join(resolveUploadDir(), sessionID)) {
		t.Fatalf("path escaped session dir: %s", resp.Path)
	}
	if strings.Contains(resp.Path, "..") {
		t.Fatalf("path contains traversal: %s", resp.Path)
	}

	os.RemoveAll(filepath.Join(resolveUploadDir(), sessionID))
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
