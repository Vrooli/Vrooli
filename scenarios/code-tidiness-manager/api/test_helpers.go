// +build testing

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Silence logs during tests
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Processor  *TidinessProcessor
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "tidiness-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create test file structure
	testDirs := []string{
		filepath.Join(tempDir, "src"),
		filepath.Join(tempDir, "test"),
		filepath.Join(tempDir, "docs"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create test dir %s: %v", dir, err)
		}
	}

	// Initialize processor without database
	processor := NewTidinessProcessor(nil)

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Processor:  processor,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// createTestFile creates a test file with given content
func createTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// createTestFiles creates multiple test files for scanning
func createTestFiles(t *testing.T, env *TestEnvironment) []string {
	t.Helper()

	files := []string{
		createTestFile(t, env.TempDir, "test.bak", "backup file"),
		createTestFile(t, env.TempDir, "file~", "backup file"),
		createTestFile(t, env.TempDir, ".test.swp", "vim swap"),
		createTestFile(t, env.TempDir, ".DS_Store", "mac file"),
		createTestFile(t, filepath.Join(env.TempDir, "src"), "main.go", "package main"),
		createTestFile(t, filepath.Join(env.TempDir, "src"), "TODO.txt", "TODO: fix this"),
	}

	return files
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(method, endpoint string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// executeRequest executes an HTTP request against a handler and returns the response
func executeRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// assertJSONResponse validates a JSON response structure
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if status := rr.Code; status != expectedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v\nBody: %s",
			status, expectedStatus, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v\nBody: %s", err, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	return response
}

// assertErrorResponse validates an error response structure
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedErrorType string) {
	t.Helper()

	if status := rr.Code; status != expectedStatus {
		t.Errorf("handler returned wrong status code: got %v want %v", status, expectedStatus)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		// For non-JSON error responses
		if expectedStatus >= 400 {
			return
		}
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if success, ok := response["success"].(bool); ok && success {
		t.Error("Expected success=false in error response, got success=true")
	}

	if expectedErrorType != "" && response["error"] != nil {
		if errorObj, ok := response["error"].(map[string]interface{}); ok {
			if errorType, ok := errorObj["type"].(string); ok {
				if errorType != expectedErrorType {
					t.Errorf("Expected error type %s, got %s", expectedErrorType, errorType)
				}
			}
		}
	}
}

// assertScanResponse validates a CodeScanResponse structure
func assertScanResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"success", "scan_id", "started_at", "completed_at", "statistics", "results_by_type", "issues", "total_issues"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Missing required field in scan response: %s", field)
		}
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true in scan response")
	}

	if scanID, ok := response["scan_id"].(string); !ok || scanID == "" {
		t.Error("Expected non-empty scan_id in scan response")
	}
}

// assertPatternResponse validates a PatternAnalysisResponse structure
func assertPatternResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"success", "analysis_id", "analysis_type", "started_at", "completed_at", "statistics", "issues_by_type", "recommendations"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Missing required field in pattern response: %s", field)
		}
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true in pattern response")
	}

	if analysisID, ok := response["analysis_id"].(string); !ok || analysisID == "" {
		t.Error("Expected non-empty analysis_id in pattern response")
	}
}

// assertCleanupResponse validates a CleanupExecutionResponse structure
func assertCleanupResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	requiredFields := []string{"success", "execution_id", "dry_run", "started_at", "completed_at", "statistics", "results"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Missing required field in cleanup response: %s", field)
		}
	}

	if executionID, ok := response["execution_id"].(string); !ok || executionID == "" {
		t.Error("Expected non-empty execution_id in cleanup response")
	}
}

// withTimeout wraps a context with timeout for testing
func withTimeout(t *testing.T, duration time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), duration)
}

// mockProcessor creates a mock TidinessProcessor for testing
func mockProcessor() *TidinessProcessor {
	return NewTidinessProcessor(nil)
}
