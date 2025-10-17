package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server      *Server
	TestDB      *sql.DB
	DBConnStr   string
	Cleanup     func()
	OldPort     string
	OldQdrant   string
	OldPostgres string
}

// setupTestEnvironment creates an isolated test environment with mock database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Save original environment variables
	oldPort := os.Getenv("API_PORT")
	oldQdrant := os.Getenv("QDRANT_URL")
	oldPostgres := os.Getenv("POSTGRES_URL")

	// Set test environment variables
	os.Setenv("API_PORT", "0") // Use random available port for testing
	os.Setenv("QDRANT_URL", "http://localhost:6333")
	os.Setenv("TESTING_MODE", "true") // Skip retries in test mode

	// Use an intentionally invalid Postgres URL to skip DB connection in tests
	// The server handles nil DB gracefully, so tests will run without database
	os.Setenv("POSTGRES_URL", "postgres://invalid:invalid@localhost:1/invalid?connect_timeout=1")

	// Create test server (will handle nil DB gracefully due to connection failure)
	server, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Force DB to nil to ensure tests don't wait for connection attempts
	if server.db != nil {
		server.db.Close()
		server.db = nil
	}

	return &TestEnvironment{
		Server:      server,
		OldPort:     oldPort,
		OldQdrant:   oldQdrant,
		OldPostgres: oldPostgres,
		Cleanup: func() {
			// Restore original environment
			if oldPort != "" {
				os.Setenv("API_PORT", oldPort)
			} else {
				os.Unsetenv("API_PORT")
			}
			if oldQdrant != "" {
				os.Setenv("QDRANT_URL", oldQdrant)
			} else {
				os.Unsetenv("QDRANT_URL")
			}
			if oldPostgres != "" {
				os.Setenv("POSTGRES_URL", oldPostgres)
			} else {
				os.Unsetenv("POSTGRES_URL")
			}
		},
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
func makeHTTPRequest(server *Server, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = strings.NewReader(v)
		case []byte:
			bodyReader = bytes.NewReader(v)
		default:
			jsonBody, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonBody)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default headers
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if _, hasContentType := req.Headers["Content-Type"]; !hasContentType && req.Body != nil {
		req.Headers["Content-Type"] = "application/json"
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()

	// Route the request to the appropriate handler
	switch {
	case req.Path == "/health" || strings.HasPrefix(req.Path, "/api/v1/knowledge/health"):
		server.healthHandler(w, httpReq)
	case strings.HasPrefix(req.Path, "/api/v1/knowledge/search"):
		server.searchHandler(w, httpReq)
	case strings.HasPrefix(req.Path, "/api/v1/knowledge/graph"):
		server.graphHandler(w, httpReq)
	case strings.HasPrefix(req.Path, "/api/v1/knowledge/metrics"):
		server.metricsHandler(w, httpReq)
	case strings.HasPrefix(req.Path, "/api/v1/knowledge/timeline"):
		server.timelineHandler(w, httpReq)
	default:
		http.NotFound(w, httpReq)
	}

	return w, nil
}

// assertStatusCode validates the HTTP status code
func assertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Errorf("Expected status code %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// assertJSONResponse validates a JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
	}

	if err := json.NewDecoder(w.Body).Decode(target); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContain string) {
	t.Helper()

	assertStatusCode(t, w, expectedStatus)

	body := w.Body.String()
	if shouldContain != "" && !strings.Contains(body, shouldContain) {
		t.Errorf("Expected error message to contain %q, got: %s", shouldContain, body)
	}
}

// assertFieldExists validates that a field exists in a map
func assertFieldExists(t *testing.T, data map[string]interface{}, field string) {
	t.Helper()
	if _, exists := data[field]; !exists {
		t.Errorf("Expected field %q to exist in response", field)
	}
}

// assertFieldValue validates a field value in a map
func assertFieldValue(t *testing.T, data map[string]interface{}, field string, expected interface{}) {
	t.Helper()
	if val, exists := data[field]; !exists {
		t.Errorf("Expected field %q to exist", field)
	} else if val != expected {
		t.Errorf("Expected field %q to be %v, got %v", field, expected, val)
	}
}

// createTestContext creates a test context with timeout
func createTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// MockQdrantCLI mocks the resource-qdrant CLI for testing
type MockQdrantCLI struct {
	CollectionsOutput string
	SearchOutput      string
	InfoOutput        string
	ShouldError       bool
	ErrorMessage      string
}

// setupMockQdrantCLI creates a mock CLI for testing
func setupMockQdrantCLI(t *testing.T, mock *MockQdrantCLI) func() {
	t.Helper()

	// In real implementation, we would swap out the execResourceQdrant function
	// For now, this is a placeholder showing the pattern

	return func() {
		// Restore original CLI behavior
	}
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("Timeout waiting for condition: %s", description)
			}
		}
	}
}

// assertResponseTime validates response time is within acceptable limits
func assertResponseTime(t *testing.T, start time.Time, maxDuration time.Duration, operation string) {
	t.Helper()

	duration := time.Since(start)
	if duration > maxDuration {
		t.Errorf("%s took %v, expected less than %v", operation, duration, maxDuration)
	}
}

// createSearchRequest creates a test search request
func createSearchRequest(query string, opts ...func(*SearchRequest)) SearchRequest {
	req := SearchRequest{
		Query:     query,
		Limit:     10,
		Threshold: 0.5,
	}

	for _, opt := range opts {
		opt(&req)
	}

	return req
}

// withCollection sets the collection for a search request
func withCollection(collection string) func(*SearchRequest) {
	return func(req *SearchRequest) {
		req.Collection = collection
	}
}

// withLimit sets the limit for a search request
func withLimit(limit int) func(*SearchRequest) {
	return func(req *SearchRequest) {
		req.Limit = limit
	}
}

// withThreshold sets the threshold for a search request
func withThreshold(threshold float64) func(*SearchRequest) {
	return func(req *SearchRequest) {
		req.Threshold = threshold
	}
}
