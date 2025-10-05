// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Redirect log output to discard during tests (unless verbose)
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stdout)
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use in-memory SQLite for testing or test PostgreSQL
	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		// Skip DB tests if no test database configured
		t.Skip("TEST_DB_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "metareasoning-test")
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
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
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
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
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
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
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

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
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
		errorStr, ok := errorMsg.(string)
		if !ok {
			t.Errorf("Expected error to be string, got %T", errorMsg)
			return
		}
		if errorStr != expectedErrorMessage {
			t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, errorStr)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ReasoningRequest creates a test reasoning request
func (g *TestDataGenerator) ReasoningRequest(inputText, reasoningType string) ReasoningRequest {
	return ReasoningRequest{
		Input: inputText,
		Type:  reasoningType,
		Model: "llama3.2",
	}
}

// AnalyzeRequest creates a test analyze request
func (g *TestDataGenerator) AnalyzeRequest(inputText, analysisType string) AnalyzeRequest {
	return AnalyzeRequest{
		Input: inputText,
		Type:  analysisType,
		Model: "llama3.2",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// MockOllamaClient mocks the Ollama client for testing
type MockOllamaClient struct {
	GenerateFunc func(prompt, model string) (string, error)
}

func (m *MockOllamaClient) Generate(prompt, model string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(prompt, model)
	}
	// Default mock response
	return `{"pros": [], "cons": [], "recommendation": "test", "confidence": 0.8}`, nil
}

// MockDiscoveryService creates a mock discovery service for testing
func MockDiscoveryService(t *testing.T) *DiscoveryService {
	db := setupTestDB(t)
	return NewDiscoveryService(db, "http://localhost:5678", "http://localhost:5681", "http://localhost:6333")
}

// createTestWorkflow creates a test workflow in the database
func createTestWorkflow(t *testing.T, db *sql.DB, platform, platformID, name string) uuid.UUID {
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO workflow_registry (id, platform, platform_id, name, tags, embedding_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, platform, platformID, name, []string{"test"}, fmt.Sprintf("%s_%s", platform, platformID))
	if err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}
	return id
}

// cleanupTestWorkflow removes a test workflow from the database
func cleanupTestWorkflow(t *testing.T, db *sql.DB, id uuid.UUID) {
	_, err := db.Exec(`DELETE FROM workflow_registry WHERE id = $1`, id)
	if err != nil {
		t.Logf("Warning: Failed to cleanup test workflow: %v", err)
	}
}
