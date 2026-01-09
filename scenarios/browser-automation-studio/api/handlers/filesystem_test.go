package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// ListDirectories Tests
// ============================================================================

func TestListDirectories_DefaultPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := ListDirectoriesRequest{
		// Empty path should default to home directory
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ListDirectoriesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Path should be set to home directory or root
	if response.Path == "" {
		t.Error("expected Path to be set")
	}
}

func TestListDirectories_ValidPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Use temp directory which should exist on all systems
	tempDir := os.TempDir()

	body := ListDirectoriesRequest{
		Path: tempDir,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ListDirectoriesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Verify path is absolute
	if !filepath.IsAbs(response.Path) {
		t.Errorf("expected absolute path, got %q", response.Path)
	}
}

func TestListDirectories_InvalidJSON(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestListDirectories_NonExistentPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := ListDirectoriesRequest{
		Path: "/this/path/should/not/exist/anywhere/12345",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for non-existent path, got %d", rr.Code)
	}

	var response map[string]any
	json.Unmarshal(rr.Body.Bytes(), &response)
	details := response["details"].(map[string]any)
	if details["error"] != "path does not exist" {
		t.Errorf("expected 'path does not exist' error, got %v", details["error"])
	}
}

func TestListDirectories_FileNotDirectory(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	body := ListDirectoriesRequest{
		Path: tmpFile.Name(),
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for file (not directory), got %d", rr.Code)
	}

	var response map[string]any
	json.Unmarshal(rr.Body.Bytes(), &response)
	details := response["details"].(map[string]any)
	if details["error"] != "path is not a directory" {
		t.Errorf("expected 'path is not a directory' error, got %v", details["error"])
	}
}

func TestListDirectories_WithSubdirectories(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create a temp directory with subdirectories
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "subdir1")
	subDir2 := filepath.Join(tmpDir, "subdir2")
	os.Mkdir(subDir1, 0755)
	os.Mkdir(subDir2, 0755)

	// Create a file (should not be included in results)
	tmpFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	body := ListDirectoriesRequest{
		Path: tmpDir,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ListDirectoriesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// Should have exactly 2 subdirectories
	if len(response.Entries) != 2 {
		t.Errorf("expected 2 subdirectories, got %d", len(response.Entries))
	}

	// Entries should be sorted alphabetically
	if len(response.Entries) >= 2 {
		if response.Entries[0].Name > response.Entries[1].Name {
			t.Error("expected entries to be sorted alphabetically")
		}
	}
}

func TestListDirectories_ParentPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create a temp directory with a subdirectory
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	body := ListDirectoriesRequest{
		Path: subDir,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response ListDirectoriesResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	// Parent should be the temp directory
	if response.Parent == "" {
		t.Error("expected Parent to be set for non-root directory")
	}
}

func TestListDirectories_RootPath(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	body := ListDirectoriesRequest{
		Path: "/",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response ListDirectoriesResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	// Root path should have empty parent
	if response.Parent != "" {
		t.Errorf("expected empty Parent for root directory, got %q", response.Parent)
	}
}

func TestListDirectories_EmptyDirectory(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create an empty temp directory
	tmpDir, err := os.MkdirTemp("", "emptydir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	body := ListDirectoriesRequest{
		Path: tmpDir,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response ListDirectoriesResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	if len(response.Entries) != 0 {
		t.Errorf("expected 0 entries for empty directory, got %d", len(response.Entries))
	}
}

func TestListDirectories_SpecialDirectoryNames(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Create a temp directory with directories that should be filtered
	tmpDir, err := os.MkdirTemp("", "specialdir")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create normal subdirectory
	os.Mkdir(filepath.Join(tmpDir, "normal"), 0755)
	// Hidden directories starting with . should be included
	os.Mkdir(filepath.Join(tmpDir, ".hidden"), 0755)

	body := ListDirectoriesRequest{
		Path: tmpDir,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fs/list-directories", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ListDirectories(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var response ListDirectoriesResponse
	json.Unmarshal(rr.Body.Bytes(), &response)

	// Should include both directories (hidden and normal)
	if len(response.Entries) < 1 {
		t.Error("expected at least one directory entry")
	}
}

func TestListDirectoriesRequest_Struct(t *testing.T) {
	tests := []struct {
		name       string
		json       string
		expectPath string
	}{
		{
			name:       "with path",
			json:       `{"path":"/some/path"}`,
			expectPath: "/some/path",
		},
		{
			name:       "empty path",
			json:       `{"path":""}`,
			expectPath: "",
		},
		{
			name:       "omitted path",
			json:       `{}`,
			expectPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req ListDirectoriesRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if req.Path != tt.expectPath {
				t.Errorf("expected Path %q, got %q", tt.expectPath, req.Path)
			}
		})
	}
}

func TestDirectoryEntry_Struct(t *testing.T) {
	entry := DirectoryEntry{
		Name: "testdir",
		Path: "/home/user/testdir",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DirectoryEntry
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Name != entry.Name {
		t.Errorf("expected Name %q, got %q", entry.Name, decoded.Name)
	}
	if decoded.Path != entry.Path {
		t.Errorf("expected Path %q, got %q", entry.Path, decoded.Path)
	}
}
