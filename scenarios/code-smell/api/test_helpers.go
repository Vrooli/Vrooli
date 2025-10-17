package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Silence logs during tests unless DEBUG is set
	if os.Getenv("DEBUG") == "" {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    string
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewBufferString(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Create response recorder
	recorder := httptest.NewRecorder()

	return recorder, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedKeys []string) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	for _, key := range expectedKeys {
		if _, ok := response[key]; !ok {
			t.Errorf("Expected key '%s' not found in response", key)
		}
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Error("Response does not contain 'error' field")
		return
	}

	if expectedErrorSubstring != "" && !contains(errorMsg, expectedErrorSubstring) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedErrorSubstring, errorMsg)
	}
}

// assertResponseField checks if a response field has the expected value
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expectedValue interface{}) {
	t.Helper()

	value, ok := response[field]
	if !ok {
		t.Errorf("Field '%s' not found in response", field)
		return
	}

	if value != expectedValue {
		t.Errorf("Field '%s': expected %v, got %v", field, expectedValue, value)
	}
}

// assertResponseArrayLength checks if a response array field has the expected length
func assertResponseArrayLength(t *testing.T, response map[string]interface{}, field string, expectedLength int) {
	t.Helper()

	value, ok := response[field]
	if !ok {
		t.Errorf("Field '%s' not found in response", field)
		return
	}

	array, ok := value.([]interface{})
	if !ok {
		t.Errorf("Field '%s' is not an array", field)
		return
	}

	if len(array) != expectedLength {
		t.Errorf("Field '%s': expected length %d, got %d", field, expectedLength, len(array))
	}
}

// createTestServer creates a new test server instance
func createTestServer() *Server {
	// Use a test port
	os.Setenv("CODE_SMELL_API_PORT", "8091")
	return NewServer()
}

// makeJSONBody creates a JSON body string from a map
func makeJSONBody(data map[string]interface{}) string {
	jsonBytes, _ := json.Marshal(data)
	return string(jsonBytes)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// executeHandler executes a handler with the given request and returns the response
func executeHandler(handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != "" {
		bodyReader = bytes.NewBufferString(req.Body)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	handler(recorder, httpReq)

	return recorder
}
