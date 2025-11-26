package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Redirect log output to suppress noise during tests
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Add headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set Content-Type for JSON bodies
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add URL vars if using mux
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates that the response is valid JSON with expected status
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates that the response contains an error
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// setupTestEnv configures environment variables for testing
func setupTestEnv() func() {
	originalEnv := make(map[string]string)

	// Save original environment
	envVars := []string{"API_PORT", "PORT", "WINDMILL_BASE_URL",
		"VAULT_URL", "MINIO_URL", "UNSTRUCTURED_URL", "QDRANT_URL", "VROOLI_LIFECYCLE_MANAGED"}

	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Set test environment
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "9999")
	os.Setenv("WINDMILL_BASE_URL", "http://localhost:8000")
	os.Setenv("VAULT_URL", "http://localhost:8200")
	os.Setenv("MINIO_URL", "http://localhost:9000")
	os.Setenv("UNSTRUCTURED_URL", "http://localhost:8000")

	// Return cleanup function
	return func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
}

// createTestServer creates a test HTTP server with all routes configured
func createTestServer() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/health", healthHandler).Methods("GET")

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/documents", documentsHandler).Methods("GET")
	api.HandleFunc("/jobs", jobsHandler).Methods("GET")
	api.HandleFunc("/workflows", workflowsHandler).Methods("GET")

	return r
}
