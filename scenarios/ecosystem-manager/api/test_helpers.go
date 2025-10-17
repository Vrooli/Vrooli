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
	"time"

	"github.com/gorilla/mux"
)

// setupTestLogger initializes a logger for testing
func setupTestLogger() func() {
	// Store original logger output
	originalOutput := log.Writer()
	originalFlags := log.Flags()

	// Set test logger to discard output unless debugging
	if os.Getenv("TEST_DEBUG") != "1" {
		log.SetOutput(io.Discard)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Return cleanup function
	return func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir     string
	QueueDir    string
	PromptsDir  string
	OriginalWD  string
	OriginalEnv map[string]string
	Cleanup     func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	t.Helper()

	tempDir := t.TempDir()

	// Create queue subdirectories
	queueDir := filepath.Join(tempDir, "queue")
	for _, status := range []string{"pending", "in-progress", "completed", "failed"} {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("Failed to create queue dir %s: %v", status, err)
		}
	}

	// Create prompts directory
	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	// Store original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Store original environment variables
	originalEnv := map[string]string{
		"API_PORT":                 os.Getenv("API_PORT"),
		"VROOLI_LIFECYCLE_MANAGED": os.Getenv("VROOLI_LIFECYCLE_MANAGED"),
	}

	// Set test environment variables
	os.Setenv("API_PORT", "18999")
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	return &TestEnvironment{
		TempDir:     tempDir,
		QueueDir:    queueDir,
		PromptsDir:  promptsDir,
		OriginalWD:  originalWD,
		OriginalEnv: originalEnv,
		Cleanup: func() {
			// Restore environment variables
			for key, value := range originalEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				panic("failed to marshal request body: " + err.Error())
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorField, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	errorMsg, ok := errorField.(string)
	if !ok {
		t.Errorf("Expected error field to be string, got %T", errorField)
		return
	}

	if expectedErrorSubstring != "" && !contains(errorMsg, expectedErrorSubstring) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorMsg)
	}
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

// createTestTask creates a task for testing
func createTestTask(taskType, operation, target string) map[string]interface{} {
	timestamp := time.Now().Format(time.RFC3339)
	return map[string]interface{}{
		"type":       taskType,
		"operation":  operation,
		"target":     target,
		"status":     "pending",
		"created_at": timestamp,
		"updated_at": timestamp,
	}
}

// executeHandler executes a handler with a request and returns the response
func executeHandler(handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	w := makeHTTPRequest(req)

	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewReader([]byte(v))
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			bodyBytes, _ := json.Marshal(v)
			bodyReader = bytes.NewReader(bodyBytes)
		}
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	handler(w, httpReq)
	return w
}

// assertResponseField checks if a response field matches expected value
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expected interface{}) {
	t.Helper()

	actual, exists := response[field]
	if !exists {
		t.Errorf("Expected field '%s' not found in response", field)
		return
	}

	if expected != nil && actual != expected {
		t.Errorf("Expected field '%s' to be %v, got %v", field, expected, actual)
	}
}
