package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	cleanup func()
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "file-tools-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create subdirectories for testing
	testDirs := []string{
		filepath.Join(tempDir, "input"),
		filepath.Join(tempDir, "output"),
		filepath.Join(tempDir, "archives"),
		filepath.Join(tempDir, "temp"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create test subdirectory %s: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// createTestFile creates a test file with specified content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// createTestFiles creates multiple test files
func createTestFiles(t *testing.T, dir string, files map[string]string) []string {
	var paths []string
	for filename, content := range files {
		path := createTestFile(t, dir, filename, content)
		paths = append(paths, path)
	}
	return paths
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, recorder.Body.String())
	}

	return response
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || bytes.Contains([]byte(s), []byte(substr)))
}

// TestFileSet represents a set of test files with metadata
type TestFileSet struct {
	Files   map[string]string // filename -> content
	TempDir string
	Paths   []string
	Cleanup func()
}
