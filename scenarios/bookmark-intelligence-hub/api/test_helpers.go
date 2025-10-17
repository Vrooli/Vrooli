// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes a test logger with controlled output
func setupTestLogger() func() {
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetPrefix("[test] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return func() {
		log.SetPrefix(originalPrefix)
		log.SetFlags(originalFlags)
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB       *sql.DB
	Config   *Config
	Cleanup  func()
}

// setupTestDatabase creates an isolated test database
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use environment variables or in-memory SQLite for testing
	// For now, we'll use a mock configuration
	config := &Config{
		Port:           0, // Let the OS assign a port
		DatabaseURL:    "postgres://test:test@localhost:5432/test?sslmode=disable",
		HuginnURL:      "http://localhost:3000",
		BrowserlessURL: "http://localhost:3001",
		APIToken:       "test-token",
	}

	// Try to connect to test database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
		return nil
	}

	// Set minimal connection pool for tests
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(1 * time.Minute)

	// Verify connection (with short timeout)
	testDB := &TestDatabase{
		DB:     db,
		Config: config,
		Cleanup: func() {
			if db != nil {
				db.Close()
			}
		},
	}

	return testDB
}

// TestServer manages test server instances
type TestServer struct {
	Server   *Server
	Router   *mux.Router
	Config   *Config
	Cleanup  func()
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T, withDB bool) *TestServer {
	config := &Config{
		Port:           0,
		DatabaseURL:    "postgres://test:test@localhost:5432/test?sslmode=disable",
		HuginnURL:      "http://localhost:3000",
		BrowserlessURL: "http://localhost:3001",
		APIToken:       "test-token",
	}

	var server *Server
	var err error

	if withDB {
		// Try to create server with database
		server, err = NewServer(config)
		if err != nil {
			t.Skipf("Could not create test server with database: %v", err)
			return nil
		}
	} else {
		// Create minimal server without database - use a mock database
		// Open a mock database connection that will fail ping but won't cause nil pointer
		db, _ := sql.Open("postgres", config.DatabaseURL)

		server = &Server{
			config: config,
			db:     db, // Set db even if it can't connect, to avoid nil pointer
		}
		server.setupRoutes()
	}

	return &TestServer{
		Server:  server,
		Router:  server.router,
		Config:  config,
		Cleanup: func() {
			if server != nil {
				server.Close()
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

// makeHTTPRequest creates and returns an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*http.Request, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
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

	return httpReq, nil
}

// executeRequest executes an HTTP request against a handler
func executeRequest(handler http.HandlerFunc, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	httpReq, err := makeHTTPRequest(req)
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)
	return w, nil
}

// executeServerRequest executes a request against the full server router
func executeServerRequest(server *TestServer, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	httpReq, err := makeHTTPRequest(req)
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, httpReq)
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

// assertJSONArray validates that response contains an array
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorField string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	if _, exists := response["error"]; !exists {
		t.Error("Expected 'error' field in error response")
	}

	if expectedErrorField != "" {
		if msg, exists := response["message"]; exists {
			if msgStr, ok := msg.(string); ok && msgStr != expectedErrorField {
				t.Errorf("Expected error message '%s', got '%s'", expectedErrorField, msgStr)
			}
		}
	}
}

// assertHealthResponse validates health check responses
func assertHealthResponse(t *testing.T, w *httptest.ResponseRecorder) {
	response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
		"status": "healthy",
	})

	if response == nil {
		return
	}

	// Validate health response structure
	requiredFields := []string{"status", "timestamp", "version", "database"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Expected field '%s' in health response", field)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateProfileRequest generates a profile creation request
func (g *TestDataGenerator) CreateProfileRequest(name, description string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"settings": map[string]interface{}{
			"auto_approve": true,
		},
	}
}

// ProcessBookmarksRequest generates a bookmark processing request
func (g *TestDataGenerator) ProcessBookmarksRequest(urls []string) map[string]interface{} {
	return map[string]interface{}{
		"urls": urls,
	}
}

// CreateCategoryRequest generates a category creation request
func (g *TestDataGenerator) CreateCategoryRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
	}
}

// ApproveActionsRequest generates an action approval request
func (g *TestDataGenerator) ApproveActionsRequest(actionIDs []string) map[string]interface{} {
	return map[string]interface{}{
		"action_ids": actionIDs,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// Helper to skip tests when database is not available
func requireDatabase(t *testing.T) {
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Skip("Skipping database test (SKIP_DB_TESTS=true)")
	}
}

// Helper to check if running in CI environment
func isCI() bool {
	return os.Getenv("CI") == "true" || os.Getenv("GITHUB_ACTIONS") == "true"
}
