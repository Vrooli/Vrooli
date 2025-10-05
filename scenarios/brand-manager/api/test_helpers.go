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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Redirect log output to discard during tests to reduce noise
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// setupTestDB creates an in-memory SQLite database for testing
// Note: This requires converting postgres-specific queries to be more generic
func setupTestDB(t *testing.T) *sql.DB {
	// For now, we'll use a mock approach since brand-manager uses postgres-specific syntax
	// In production, we'd need a test postgres instance or use sqlmock
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}

	// Test connection - skip test if DB not available
	if err := db.Ping(); err != nil {
		t.Skipf("Database not available for testing: %v", err)
	}

	return db
}

// cleanupTestDB closes database connection and cleans up test data
func cleanupTestDB(db *sql.DB, t *testing.T) {
	if db != nil {
		// Clean up test data
		db.Exec("DELETE FROM integration_requests WHERE 1=1")
		db.Exec("DELETE FROM brands WHERE 1=1")
		db.Close()
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates an HTTP request and returns the response recorder
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if using gorilla/mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates that the response is valid JSON with expected status
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates that the response is an error with expected status
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	response := assertJSONResponse(t, w, expectedStatus)

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Errorf("Expected 'error' field in response")
	}

	if expectedMessage != "" && errorMsg != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, errorMsg)
	}

	status, ok := response["status"].(float64)
	if !ok {
		t.Errorf("Expected 'status' field in response")
	}

	if int(status) != expectedStatus {
		t.Errorf("Expected status %d in response body, got %d", expectedStatus, int(status))
	}
}

// createTestBrand creates a test brand in the database
func createTestBrand(t *testing.T, db *sql.DB, name string) uuid.UUID {
	brandID := uuid.New()

	brandColors := `{"primary": "#FF5733", "secondary": "#33FF57"}`
	assets := `[]`
	metadata := `{}`

	_, err := db.Exec(`
		INSERT INTO brands (id, name, short_name, slogan, ad_copy, description,
			brand_colors, logo_url, favicon_url, assets, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`,
		brandID, name, name[:3], "Test Slogan", "Test Ad Copy", "Test Description",
		brandColors, "http://example.com/logo.png", "http://example.com/favicon.ico",
		assets, metadata)

	if err != nil {
		t.Fatalf("Failed to create test brand: %v", err)
	}

	return brandID
}

// createTestIntegration creates a test integration request in the database
func createTestIntegration(t *testing.T, db *sql.DB, brandID uuid.UUID, targetPath string) uuid.UUID {
	integrationID := uuid.New()

	requestPayload := `{"test": true}`
	responsePayload := `{}`

	_, err := db.Exec(`
		INSERT INTO integration_requests (id, brand_id, target_app_path, integration_type,
			claude_session_id, status, request_payload, response_payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`,
		integrationID, brandID, targetPath, "full", "", "pending", requestPayload, responsePayload)

	if err != nil {
		t.Fatalf("Failed to create test integration: %v", err)
	}

	return integrationID
}

// MockHTTPClient creates a mock HTTP client for testing external service calls
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"success": true}`)),
	}, nil
}

// mockN8nResponse creates a mock response for n8n webhook calls
func mockN8nResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// skipIfNoDatabase skips the test if database is not available
func skipIfNoDatabase(t *testing.T, db *sql.DB) {
	if db == nil {
		t.Skip("Database not available for testing")
	}

	if err := db.Ping(); err != nil {
		t.Skipf("Database not available: %v", err)
	}
}

// setTestEnv sets environment variables for testing
func setTestEnv(t *testing.T) func() {
	originalEnv := make(map[string]string)
	testEnv := map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
		"API_PORT":                 "8090",
		"N8N_BASE_URL":             "http://localhost:5678",
		"WINDMILL_BASE_URL":        "http://localhost:8000",
		"COMFYUI_BASE_URL":         "http://localhost:8188",
		"MINIO_ENDPOINT":           "localhost:9000",
		"VAULT_ADDR":               "http://localhost:8200",
	}

	// Save and set test environment
	for key, value := range testEnv {
		originalEnv[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

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
