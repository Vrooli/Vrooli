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
)

// TestLogger manages logging configuration during tests
type TestLogger struct {
	originalFlags int
	originalOutput io.Writer
}

// setupTestLogger initializes logging for tests with cleanup
func setupTestLogger() func() {
	// Suppress logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
	}

	return func() {
		log.SetOutput(os.Stderr)
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var body io.Reader
	if req.Body != nil {
		jsonData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonData)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, body)
	if err != nil {
		return nil, err
	}

	// Set default content type
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
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

	// Execute request
	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response status and structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Verify Content-Type is text/plain for standard http.Error responses
	contentType := w.Header().Get("Content-Type")
	if contentType != "" && contentType != "text/plain; charset=utf-8" {
		t.Logf("Note: Error response has Content-Type: %s", contentType)
	}

	// Verify body is not empty
	if w.Body.Len() == 0 {
		t.Error("Error response body should not be empty")
	}
}

// assertHealthResponse validates health check response
func assertHealthResponse(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	var health HealthResponse
	assertJSONResponse(t, w, http.StatusOK, &health)

	if health.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", health.Status)
	}

	if health.Service != "seo-optimizer-api" {
		t.Errorf("Expected service 'seo-optimizer-api', got '%s'", health.Service)
	}

	if health.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

// setupTestSEOProcessor initializes SEO processor for testing
func setupTestSEOProcessor() {
	if seoProcessor == nil {
		seoProcessor = NewSEOProcessor()
	}
}

// createTestServer creates a test HTTP server
func createTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	setupTestSEOProcessor()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", corsMiddleware(healthHandler))
	mux.HandleFunc("/api/seo-audit", corsMiddleware(seoAuditHandler))
	mux.HandleFunc("/api/keyword-research", corsMiddleware(keywordResearchHandler))
	mux.HandleFunc("/api/content-optimize", corsMiddleware(contentOptimizeHandler))
	mux.HandleFunc("/api/competitor-analysis", corsMiddleware(competitorAnalysisHandler))

	return httptest.NewServer(mux)
}
