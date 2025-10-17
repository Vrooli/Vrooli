package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Create a buffer to capture log output during tests
	log.SetOutput(ioutil.Discard) // Silence logs during tests
	return func() {
		log.SetOutput(os.Stderr) // Restore default
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Server     *Server
	Router     *mux.Router
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "network-tools-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a mock server without database
	router := mux.NewRouter()

	// Create mock server (database connection optional for unit tests)
	server := &Server{
		config: &Config{
			Port:        "15000",
			DatabaseURL: "mock://localhost/test",
		},
		db:          nil, // Will be mocked as needed
		router:      router,
		client:      &http.Client{},
		rateLimiter: NewRateLimiter(100, 60),
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Server:     server,
		Router:     router,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
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
func makeHTTPRequest(handler http.HandlerFunc, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

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
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	if bodyReader == nil {
		bodyReader = bytes.NewReader([]byte{})
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Add headers
	if req.Headers != nil {
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}
	}

	// Add query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add URL vars (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateFn func(data map[string]interface{}) error) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v\nBody: %s", err, w.Body.String())
	}

	if validateFn != nil {
		if err := validateFn(response); err != nil {
			t.Errorf("Response validation failed: %v", err)
		}
	}
}

// assertSuccessResponse validates a successful API response
func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	var response map[string]interface{}
	assertJSONResponse(t, w, expectedStatus, func(data map[string]interface{}) error {
		response = data

		// Check for success field
		if success, ok := data["success"].(bool); ok && !success {
			return fmt.Errorf("expected success=true, got success=%v", success)
		}

		return nil
	})

	return response
}

// assertErrorResponse validates an error API response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, errorContains string) {
	t.Helper()

	assertJSONResponse(t, w, expectedStatus, func(data map[string]interface{}) error {
		// Check for success=false
		if success, ok := data["success"].(bool); ok && success {
			return fmt.Errorf("expected success=false for error response")
		}

		// Check for error message
		if errorMsg, ok := data["error"].(string); ok {
			if errorContains != "" && !contains(errorMsg, errorContains) {
				return fmt.Errorf("expected error to contain '%s', got '%s'", errorContains, errorMsg)
			}
		} else {
			return fmt.Errorf("expected error field in response")
		}

		return nil
	})
}

// contains checks if a string contains a substring (case-sensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(substr) == 0 ||
		(len(str) > 0 && len(substr) > 0 && containsHelper(str, substr)))
}

func containsHelper(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// assertResponseField validates a specific field in the response
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expectedValue interface{}) {
	t.Helper()

	value, ok := response[field]
	if !ok {
		t.Errorf("Expected field '%s' not found in response", field)
		return
	}

	if expectedValue != nil && value != expectedValue {
		t.Errorf("Expected field '%s' to be %v, got %v", field, expectedValue, value)
	}
}

// mockHTTPServer creates a mock HTTP server for testing external HTTP requests
func mockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// createTestHTTPRequest creates a test HTTPRequest object
func createTestHTTPRequest(url, method string, headers map[string]string, body interface{}) HTTPRequest {
	return HTTPRequest{
		URL:     url,
		Method:  method,
		Headers: headers,
		Body:    body,
		Options: HTTPOptions{
			TimeoutMs:       5000,
			FollowRedirects: true,
			VerifySSL:       true,
		},
	}
}

// createTestDNSRequest creates a test DNSRequest object
func createTestDNSRequest(query, recordType string) DNSRequest {
	return DNSRequest{
		Query:      query,
		RecordType: recordType,
		Options: DNSOptions{
			TimeoutMs: 5000,
			Recursive: true,
		},
	}
}

// createTestConnectivityRequest creates a test ConnectivityRequest object
func createTestConnectivityRequest(target, testType string) ConnectivityRequest {
	return ConnectivityRequest{
		Target:   target,
		TestType: testType,
		Options: ConnectivityOptions{
			Count:     3,
			TimeoutMs: 5000,
		},
	}
}
