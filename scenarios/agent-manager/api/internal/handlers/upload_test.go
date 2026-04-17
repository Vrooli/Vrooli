package handlers

import (
	"agent-manager/internal/storage"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gorilla/mux"
)

// createMultipartRequest builds a multipart POST request with a single file field.
func createMultipartRequest(t *testing.T, fieldName, fileName string, content []byte, contentType string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/v1/attachments/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// newUploadHandler creates a Handler configured only with the given storage service.
// The orchestration service is nil because upload handlers do not use it.
func newUploadHandler(t *testing.T, storageSvc storage.Service) *Handler {
	t.Helper()
	return New(nil, WithStorage(storageSvc))
}

// minimalPNG is a minimal valid PNG header (8-byte signature + enough data for detection).
var minimalPNG = append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'},
	bytes.Repeat([]byte{0x00}, 64)...)

// minimalJPEG is a minimal valid JPEG header.
var minimalJPEG = append([]byte{0xff, 0xd8, 0xff, 0xe0},
	bytes.Repeat([]byte{0x00}, 64)...)

func TestUploadAttachment_ValidPNG(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	req := createMultipartRequest(t, "file", "screenshot.png", minimalPNG, "image/png")
	rr := httptest.NewRecorder()

	h.UploadAttachment(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
	if resp["file_name"] != "screenshot.png" {
		t.Errorf("expected file_name=screenshot.png, got %v", resp["file_name"])
	}
	if resp["url"] == nil || resp["url"] == "" {
		t.Error("expected non-empty url in response")
	}
	if mockStorage.UploadCalls != 1 {
		t.Errorf("expected 1 upload call, got %d", mockStorage.UploadCalls)
	}
}

func TestUploadAttachment_ValidJPEG(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	req := createMultipartRequest(t, "file", "photo.jpg", minimalJPEG, "image/jpeg")
	rr := httptest.NewRecorder()

	h.UploadAttachment(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if resp["file_name"] != "photo.jpg" {
		t.Errorf("expected file_name=photo.jpg, got %v", resp["file_name"])
	}
}

func TestUploadAttachment_InvalidMIME(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	htmlContent := []byte("<html><body>hello</body></html>")
	req := createMultipartRequest(t, "file", "page.html", htmlContent, "text/html")
	rr := httptest.NewRecorder()

	h.UploadAttachment(rr, req)

	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected status 415, got %d: %s", rr.Code, rr.Body.String())
	}
	if mockStorage.UploadCalls != 0 {
		t.Errorf("expected 0 upload calls for rejected type, got %d", mockStorage.UploadCalls)
	}
}

func TestUploadAttachment_MissingFile(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	// POST with an empty multipart form (no file field)
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/attachments/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	h.UploadAttachment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestUploadAttachment_NoStorage(t *testing.T) {
	// Handler created without storage option
	h := New(nil)

	req := createMultipartRequest(t, "file", "test.png", minimalPNG, "image/png")
	rr := httptest.NewRecorder()

	h.UploadAttachment(rr, req)

	// writeSimpleError returns 400 (validation error) when storage is nil
	if rr.Code < 400 {
		t.Fatalf("expected error status, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServeUpload_Success(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	// Create a request with a valid path variable
	req := httptest.NewRequest("GET", "/api/v1/uploads/abc123/test.png", nil)
	req = mux.SetURLVars(req, map[string]string{"path": "abc123/test.png"})
	rr := httptest.NewRecorder()

	h.ServeUpload(rr, req)

	// The mock returns a path under /mock-storage, which won't exist on disk,
	// so ServeFile will return 404. The important thing is that it called
	// GetFilePath and didn't return a 400 error.
	if rr.Code == http.StatusBadRequest {
		t.Fatal("expected path to be accepted, got 400")
	}
	if mockStorage.GetFilePathCalls != 1 {
		t.Errorf("expected 1 GetFilePath call, got %d", mockStorage.GetFilePathCalls)
	}
}

func TestServeUpload_PathTraversal(t *testing.T) {
	mockStorage := storage.NewMockService()
	h := newUploadHandler(t, mockStorage)

	req := httptest.NewRequest("GET", "/api/v1/uploads/../../../etc/passwd", nil)
	req = mux.SetURLVars(req, map[string]string{"path": "../../../etc/passwd"})
	rr := httptest.NewRecorder()

	h.ServeUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for path traversal, got %d", rr.Code)
	}
	if mockStorage.GetFilePathCalls != 0 {
		t.Errorf("expected 0 GetFilePath calls for rejected path, got %d", mockStorage.GetFilePathCalls)
	}
}
