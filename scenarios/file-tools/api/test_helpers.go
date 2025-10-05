package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	cleanup func()
}

// setupTestLogger initializes a test logger with cleanup
func setupTestLogger() func() {
	// Disable logging during tests to reduce noise
	return func() {}
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

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			bodyBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if provided (for mux routing)
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	recorder := httptest.NewRecorder()
	return recorder, nil
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

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	response := assertJSONResponse(t, recorder, expectedStatus)

	if success, ok := response["success"].(bool); ok && success {
		t.Error("Expected success to be false for error response")
	}

	if expectedErrorContains != "" {
		errorMsg, ok := response["error"].(string)
		if !ok {
			t.Error("Expected error field in response")
		} else if errorMsg != expectedErrorContains && !contains(errorMsg, expectedErrorContains) {
			t.Errorf("Expected error containing '%s', got '%s'", expectedErrorContains, errorMsg)
		}
	}
}

// assertSuccessResponse validates a success response
func assertSuccessResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	response := assertJSONResponse(t, recorder, expectedStatus)

	// Some endpoints may not include a success field
	if success, ok := response["success"].(bool); ok && !success {
		t.Errorf("Expected success to be true, got false. Error: %v", response["error"])
	}

	return response
}

// createTestServer creates a test server with mocked database
func createTestServer(t *testing.T) *Server {
	// Note: For real testing, you'd need to set up a test database
	// This is a minimal mock for handler testing
	server := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test",
			APIToken:    "test-token",
		},
		router: mux.NewRouter(),
	}

	return server
}

// testAuthToken generates a test authorization token
func testAuthToken(apiToken string) string {
	return "Bearer " + apiToken
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// readFile reads file content as string
func readFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}

// getFileSize returns the size of a file in bytes
func getFileSize(t *testing.T, path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to get file info for %s: %v", path, err)
	}
	return info.Size()
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || bytes.Contains([]byte(s), []byte(substr)))
}

// generateUUID generates a new UUID string
func generateUUID() string {
	return uuid.New().String()
}

// mockOperationID generates a mock operation ID for testing
func mockOperationID() string {
	return uuid.New().String()
}

// TestFileSet represents a set of test files with metadata
type TestFileSet struct {
	Files   map[string]string // filename -> content
	TempDir string
	Paths   []string
	Cleanup func()
}

// createTestFileSet creates a complete test file set
func createTestFileSet(t *testing.T, files map[string]string) *TestFileSet {
	env := setupTestDirectory(t)

	var paths []string
	for filename, content := range files {
		path := createTestFile(t, env.TempDir, filename, content)
		paths = append(paths, path)
	}

	return &TestFileSet{
		Files:   files,
		TempDir: env.TempDir,
		Paths:   paths,
		Cleanup: env.Cleanup,
	}
}
