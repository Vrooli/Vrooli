package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := debugManager.logger
	debugManager.logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() { debugManager.logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "app-debugger-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// HTTPTestRequest defines a structured HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// makeHTTPRequest creates an HTTP request for testing
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default content type for JSON
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates that response is valid JSON and returns the parsed data
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
		t.Errorf("Failed to parse JSON response: %v, body: %s", err, w.Body.String())
		return nil
	}

	return data
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	// Error responses should have some content
	if w.Body.Len() == 0 {
		t.Error("Expected error message in response body, got empty response")
	}
}

// createTestLogFile creates a temporary log file for testing
func createTestLogFile(t *testing.T, dir string, filename string, content string) string {
	t.Helper()

	logPath := filepath.Join(dir, filename)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}

	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	return logPath
}

// assertStringContains checks if a string contains a substring
func assertStringContains(t *testing.T, str, substr string) {
	t.Helper()
	if !contains(str, substr) {
		t.Errorf("Expected string to contain %q, got: %s", substr, str)
	}
}

// assertStringNotContains checks if a string does not contain a substring
func assertStringNotContains(t *testing.T, str, substr string) {
	t.Helper()
	if contains(str, substr) {
		t.Errorf("Expected string to not contain %q, got: %s", substr, str)
	}
}

// contains is a simple helper to check if a string contains a substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 || indexOf(str, substr) >= 0)
}

// indexOf returns the index of substr in str, or -1 if not found
func indexOf(str, substr string) int {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// assertMapKeyExists checks if a map contains a specific key
func assertMapKeyExists(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, exists := m[key]; !exists {
		t.Errorf("Expected map to contain key %q, keys: %v", key, getMapKeys(m))
	}
}

// getMapKeys returns all keys from a map
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
