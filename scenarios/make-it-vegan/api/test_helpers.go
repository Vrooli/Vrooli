//go:build !test
// +build !test

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Suppress verbose logging during tests
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "make-it-vegan-test")
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

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP test request and returns the response recorder
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyBytes []byte
	var err error

	if req.Body != nil {
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add URL vars if using mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields ...string) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v. Body: %s", err, w.Body.String())
		return nil
	}

	// Check for expected fields
	for _, field := range expectedFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field '%s' not found in response: %v", field, response)
		}
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// assertStringContains checks if a string contains a substring
func assertStringContains(t *testing.T, haystack, needle, context string) {
	t.Helper()

	if haystack == "" {
		t.Errorf("%s: expected non-empty string", context)
		return
	}

	if needle != "" && !contains(haystack, needle) {
		t.Errorf("%s: expected string to contain '%s', got '%s'", context, needle, haystack)
	}
}

// contains is a helper function to check if a string contains a substring
func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || hasSubstring(haystack, needle))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []HTTPTestRequest
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]HTTPTestRequest, 0),
	}
}

// AddScenario adds a test scenario
func (b *TestScenarioBuilder) AddScenario(req HTTPTestRequest) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, req)
	return b
}

// AddInvalidJSON adds a scenario with invalid JSON
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	// We'll use a raw request for this
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   nil, // Will be replaced with invalid JSON in test
	})
	return b
}

// AddMissingField adds a scenario with a required field missing
func (b *TestScenarioBuilder) AddMissingField(method, path string, body map[string]interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, HTTPTestRequest{
		Method: method,
		Path:   path,
		Body:   body,
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []HTTPTestRequest {
	return b.scenarios
}

// setupTestCache creates a test cache client for isolated testing
func setupTestCache(t *testing.T) *CacheClient {
	// Create a cache client that's disabled for predictable testing
	return &CacheClient{
		redis:  nil,
		Enable: false,
	}
}

// setupTestDB creates a test vegan database
func setupTestDB() *VeganDatabase {
	return InitVeganDatabase()
}
