// +build testing

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
	"path/filepath"
	"strings"
	"testing"

	"app-monitor-api/config"

	"github.com/gin-gonic/gin"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	gin.SetMode(gin.TestMode)
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
		gin.SetMode(gin.DebugMode)
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
	tempDir, err := os.MkdirTemp("", "app-monitor-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create necessary subdirectories
	dirs := []string{"data", "logs"}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create %s directory: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Add query params
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set Content-Type for JSON bodies
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedBody map[string]interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectedBody == nil {
		return
	}

	var actualBody map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &actualBody); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v. Body: %s", err, w.Body.String())
	}

	for key, expectedValue := range expectedBody {
		actualValue, ok := actualBody[key]
		if !ok {
			t.Errorf("Expected key '%s' not found in response", key)
			continue
		}

		// For nested maps, do a simple equality check
		if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
			actualMap, ok := actualValue.(map[string]interface{})
			if !ok {
				t.Errorf("Key '%s': expected map, got %T", key, actualValue)
				continue
			}
			// Recursive check for nested maps
			for nestedKey, nestedExpected := range expectedMap {
				if nestedActual, ok := actualMap[nestedKey]; !ok || nestedActual != nestedExpected {
					t.Errorf("Key '%s.%s': expected %v, got %v", key, nestedKey, nestedExpected, nestedActual)
				}
			}
		} else if actualValue != expectedValue {
			t.Errorf("Key '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectedError == "" {
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v. Body: %s", err, w.Body.String())
	}

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Errorf("Response does not contain 'error' field. Body: %s", w.Body.String())
		return
	}

	if !strings.Contains(errorMsg, expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, errorMsg)
	}
}

// setupTestConfig creates a minimal test configuration
func setupTestConfig() *config.Config {
	// Set minimal environment variables for testing
	os.Setenv("API_PORT", "0") // Use random port for testing
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	cfg, _ := config.LoadConfig()
	return cfg
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T) (*Server, func()) {
	cleanup := setupTestLogger()

	cfg := setupTestConfig()

	server, err := NewServer(cfg)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create test server: %v", err)
	}

	return server, cleanup
}

// TestApp provides a pre-configured app for testing
type TestApp struct {
	ID          string
	Name        string
	DisplayName string
	Status      string
	Type        string
}

// createTestApp creates a test app instance
func createTestApp(name string) *TestApp {
	return &TestApp{
		ID:          name,
		Name:        name,
		DisplayName: strings.ReplaceAll(strings.Title(name), "-", " "),
		Status:      "active",
		Type:        "scenario",
	}
}

// assertResponseContains checks if response contains expected substring
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	body := w.Body.String()
	if !strings.Contains(body, expected) {
		t.Errorf("Expected response to contain '%s', got: %s", expected, body)
	}
}

// assertResponseNotContains checks if response does not contain substring
func assertResponseNotContains(t *testing.T, w *httptest.ResponseRecorder, unexpected string) {
	t.Helper()

	body := w.Body.String()
	if strings.Contains(body, unexpected) {
		t.Errorf("Expected response to NOT contain '%s', got: %s", unexpected, body)
	}
}

// assertContentType validates the response content type
func assertContentType(t *testing.T, w *httptest.ResponseRecorder, expectedContentType string) {
	t.Helper()

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, expectedContentType) {
		t.Errorf("Expected content type to contain '%s', got '%s'", expectedContentType, contentType)
	}
}

// mockHTTPServer creates a mock HTTP server for testing external calls
func mockHTTPServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}
