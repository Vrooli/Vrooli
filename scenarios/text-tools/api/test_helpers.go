package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Suppress logging during tests unless VERBOSE is set
	if os.Getenv("VERBOSE") != "true" {
		// Redirect logs to /dev/null during tests
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		devNull, _ := os.Open(os.DevNull)
		os.Stdout = devNull
		os.Stderr = devNull

		return func() {
			devNull.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server *Server
	Config *Config
	Cleanup func()
}

// setupTestServer creates a test server with mock configuration
func setupTestServer(t *testing.T) *TestEnvironment {
	config := &Config{
		Port:        "8080",
		DatabaseURL: "",  // Use mock database or in-memory
		OllamaURL:   "http://localhost:11434",
		RedisURL:    "redis://localhost:6379",
	}

	server := NewServer(config)

	// Initialize without actual external dependencies for unit tests
	// Skip database/resource initialization for pure unit tests

	return &TestEnvironment{
		Server: server,
		Config: config,
		Cleanup: func() {
			// Cleanup resources if needed
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
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
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
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
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

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorMessage != "" {
		errorStr := fmt.Sprintf("%v", errorMsg)
		if errorStr != expectedErrorMessage {
			t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, errorStr)
		}
	}
}

// assertResponseContains checks if response contains expected substring
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	body := w.Body.String()
	if expectedSubstring != "" && !testStringContains(body, expectedSubstring) {
		t.Errorf("Expected response to contain '%s', got: %s", expectedSubstring, body)
	}
}

// testStringContains checks if a string contains a substring
func testStringContains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) && testStringContainsHelper(s, substr))
}

func testStringContainsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestDataFactory provides utilities for generating test data
type TestDataFactory struct{}

// DiffRequest creates a test diff request
func (f *TestDataFactory) DiffRequest(text1, text2 string) DiffRequest {
	return DiffRequest{
		Text1: text1,
		Text2: text2,
		Options: DiffOptions{
			Type: "line",
		},
	}
}

// SearchRequest creates a test search request
func (f *TestDataFactory) SearchRequest(text, pattern string) SearchRequest {
	return SearchRequest{
		Text:    text,
		Pattern: pattern,
		Options: SearchOptions{
			Regex: false,
		},
	}
}

// TransformRequest creates a test transform request
func (f *TestDataFactory) TransformRequest(text string, transformType string) TransformRequest {
	return TransformRequest{
		Text: text,
		Transformations: []Transformation{
			{
				Type: transformType,
			},
		},
	}
}

// AnalyzeRequest creates a test analyze request
func (f *TestDataFactory) AnalyzeRequest(text string, analyses []string) AnalyzeRequest {
	return AnalyzeRequest{
		Text:     text,
		Analyses: analyses,
	}
}

// Global test data factory instance
var TestData = &TestDataFactory{}
