package initiatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"swarm-manager/internal/backlog"
	"testing"

	"github.com/gorilla/mux"
)

func setupTestHandlerWithInit(t *testing.T, name string) (*Handler, string) {
	t.Helper()
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: map[string]backlog.BacklogItem{}}
	svc := NewService(store, loader)
	handler := NewHandler(svc)

	// Create a test initiative.
	_, err := svc.Create(CreateRequest{
		Name:  name,
		Title: "Test Initiative",
	})
	if err != nil {
		t.Fatalf("failed to create test initiative: %v", err)
	}

	return handler, svc.InitDir(name)
}

func TestListInitiativeFiles_EmptyInitiative(t *testing.T) {
	handler, _ := setupTestHandlerWithInit(t, "test-init")

	req := httptest.NewRequest("GET", "/api/v1/initiatives/test-init/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.ListInitiativeFiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Should contain initiative.json at minimum.
	body := rr.Body.String()
	if !containsString(body, "initiative.json") {
		t.Errorf("expected response to contain initiative.json, got: %s", body)
	}
}

func TestListInitiativeFiles_WithAdditionalFiles(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	// Add files.
	if err := os.WriteFile(filepath.Join(initDir, "notes.md"), []byte("# Notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(initDir, "decisions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(initDir, "decisions", "d1.md"), []byte("decision"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/initiatives/test-init/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.ListInitiativeFiles(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !containsString(body, "notes.md") {
		t.Errorf("expected response to contain notes.md")
	}
	if !containsString(body, "decisions") {
		t.Errorf("expected response to contain decisions directory")
	}
}

func TestListInitiativeFiles_NotFound(t *testing.T) {
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: map[string]backlog.BacklogItem{}}
	svc := NewService(store, loader)
	handler := NewHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/initiatives/nonexistent/files", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	rr := httptest.NewRecorder()

	handler.ListInitiativeFiles(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetInitiativeFileContent_Success(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	if err := os.WriteFile(filepath.Join(initDir, "readme.md"), []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/initiatives/test-init/files/readme.md", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init", "filepath": "readme.md"})
	rr := httptest.NewRecorder()

	handler.GetInitiativeFileContent(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "# Hello" {
		t.Errorf("expected '# Hello', got %q", rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected content-type text/plain, got %q", ct)
	}
}

func TestGetInitiativeFileContent_NotFound(t *testing.T) {
	handler, _ := setupTestHandlerWithInit(t, "test-init")

	req := httptest.NewRequest("GET", "/api/v1/initiatives/test-init/files/missing.txt", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init", "filepath": "missing.txt"})
	rr := httptest.NewRecorder()

	handler.GetInitiativeFileContent(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestGetInitiativeFileContent_DirectoryPath(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	if err := os.MkdirAll(filepath.Join(initDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/v1/initiatives/test-init/files/subdir", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init", "filepath": "subdir"})
	rr := httptest.NewRecorder()

	handler.GetInitiativeFileContent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for directory path, got %d", rr.Code)
	}
}

func TestUploadInitiativeFile_Success(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	body, contentType := createMultipartUpload(t, "strategy.md", "# Strategy")
	req := httptest.NewRequest("POST", "/api/v1/initiatives/test-init/files", body)
	req.Header.Set("Content-Type", contentType)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.UploadInitiativeFile(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify file on disk.
	data, err := os.ReadFile(filepath.Join(initDir, "strategy.md"))
	if err != nil {
		t.Fatalf("file should exist on disk: %v", err)
	}
	if string(data) != "# Strategy" {
		t.Errorf("expected '# Strategy', got %q", string(data))
	}
}

func TestUploadInitiativeFile_ProtectedPath(t *testing.T) {
	handler, _ := setupTestHandlerWithInit(t, "test-init")

	body, contentType := createMultipartUpload(t, "initiative.json", "{}")
	req := httptest.NewRequest("POST", "/api/v1/initiatives/test-init/files", body)
	req.Header.Set("Content-Type", contentType)
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.UploadInitiativeFile(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for protected path, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestOperateInitiativeFile_Delete(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	// Create a file to delete.
	filePath := filepath.Join(initDir, "to-delete.md")
	if err := os.WriteFile(filePath, []byte("delete me"), 0o644); err != nil {
		t.Fatal(err)
	}

	reqBody := map[string]string{
		"operation":   "delete",
		"source_path": "to-delete.md",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/initiatives/test-init/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.OperateInitiativeFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestOperateInitiativeFile_Rename(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	if err := os.WriteFile(filepath.Join(initDir, "old.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	reqBody := map[string]string{
		"operation":        "rename",
		"source_path":      "old.md",
		"destination_path": "new.md",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/initiatives/test-init/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.OperateInitiativeFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if _, err := os.Stat(filepath.Join(initDir, "new.md")); os.IsNotExist(err) {
		t.Error("renamed file should exist")
	}
	if _, err := os.Stat(filepath.Join(initDir, "old.md")); !os.IsNotExist(err) {
		t.Error("old file should not exist")
	}
}

func TestOperateInitiativeFile_ProtectedSource(t *testing.T) {
	handler, _ := setupTestHandlerWithInit(t, "test-init")

	reqBody := map[string]string{
		"operation":   "delete",
		"source_path": "initiative.json",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/initiatives/test-init/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.OperateInitiativeFile(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for protected source, got %d", rr.Code)
	}
}

func TestOperateInitiativeFile_InitiativeNotFound(t *testing.T) {
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: map[string]backlog.BacklogItem{}}
	svc := NewService(store, loader)
	handler := NewHandler(svc)

	reqBody := map[string]string{
		"operation":   "delete",
		"source_path": "file.txt",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/initiatives/nonexistent/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	rr := httptest.NewRecorder()

	handler.OperateInitiativeFile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestOperateInitiativeFile_Copy(t *testing.T) {
	handler, initDir := setupTestHandlerWithInit(t, "test-init")

	if err := os.WriteFile(filepath.Join(initDir, "source.md"), []byte("copy me"), 0o644); err != nil {
		t.Fatal(err)
	}

	reqBody := map[string]string{
		"operation":        "copy",
		"source_path":      "source.md",
		"destination_path": "copied.md",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("PATCH", "/api/v1/initiatives/test-init/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"name": "test-init"})
	rr := httptest.NewRecorder()

	handler.OperateInitiativeFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	// Both files should exist.
	if _, err := os.Stat(filepath.Join(initDir, "source.md")); os.IsNotExist(err) {
		t.Error("source should still exist after copy")
	}
	if _, err := os.Stat(filepath.Join(initDir, "copied.md")); os.IsNotExist(err) {
		t.Error("copied file should exist")
	}
}

// --- helpers ---

func containsString(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}

func createMultipartUpload(t *testing.T, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fmt.Fprint(part, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf, writer.FormDataContentType()
}
