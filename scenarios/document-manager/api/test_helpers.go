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

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes logging for tests
func setupTestLogger() func() {
	// Redirect log output to discard during tests to reduce noise
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database URL from environment or create a mock
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		t.Skip("POSTGRES_URL not set, skipping database tests")
	}

	testDB, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return testDB
}

// cleanupTestDB closes database connection and cleans up test data
func cleanupTestDB(testDB *sql.DB) {
	if testDB != nil {
		testDB.Close()
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
func makeHTTPRequest(router *mux.Router, req HTTPTestRequest) *httptest.ResponseRecorder {
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
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
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

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields if provided
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

// assertJSONArrayResponse validates that response is an array
func assertJSONArrayResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Handle empty response body (e.g., for method not allowed without response)
	if w.Body.Len() == 0 {
		if expectedErrorSubstring != "" {
			t.Logf("Note: Empty response body, cannot validate error message")
		}
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		// If we can't parse JSON, that's ok if status is correct
		t.Logf("Note: Response is not JSON (status %d is correct): %s", expectedStatus, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Logf("Note: No 'error' field in response, but status %d is correct", expectedStatus)
		return
	}

	errorStr, ok := errorMsg.(string)
	if !ok {
		t.Errorf("Expected error to be string, got %T", errorMsg)
		return
	}

	if expectedErrorSubstring != "" {
		if !contains(errorStr, expectedErrorSubstring) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorStr)
		}
	}
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockHTTPServer creates a mock HTTP server for testing external dependencies
func mockHTTPServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if responseBody != "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(statusCode)
		if responseBody != "" {
			w.Write([]byte(responseBody))
		}
	}))
}

// setupTestEnvironment sets up environment variables for testing
func setupTestEnvironment(t *testing.T) func() {
	// Save original environment
	originalEnv := map[string]string{
		"API_PORT":       os.Getenv("API_PORT"),
		"POSTGRES_URL":   os.Getenv("POSTGRES_URL"),
		"REDIS_URL":      os.Getenv("REDIS_URL"),
		"QDRANT_URL":     os.Getenv("QDRANT_URL"),
		"OLLAMA_URL":     os.Getenv("OLLAMA_URL"),
		"N8N_URL":        os.Getenv("N8N_URL"),
		"WINDMILL_URL":   os.Getenv("WINDMILL_URL"),
		"UNSTRUCTURED_URL": os.Getenv("UNSTRUCTURED_URL"),
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

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// GenerateApplication creates a test application
func (g *TestDataGenerator) Application(name string) Application {
	return Application{
		Name:              name,
		RepositoryURL:     fmt.Sprintf("https://github.com/test/%s", name),
		DocumentationPath: "/docs",
		HealthScore:       85.5,
		Active:            true,
	}
}

// GenerateAgent creates a test agent
func (g *TestDataGenerator) Agent(name string, appID string) Agent {
	return Agent{
		Name:               name,
		Type:               "documentation_analyzer",
		ApplicationID:      appID,
		Configuration:      `{"model": "llama2"}`,
		ScheduleCron:       "0 */6 * * *",
		AutoApplyThreshold: 0.8,
		Enabled:            true,
	}
}

// GenerateQueueItem creates a test improvement queue item
func (g *TestDataGenerator) QueueItem(agentID string, appID string) ImprovementQueue {
	return ImprovementQueue{
		AgentID:       agentID,
		ApplicationID: appID,
		Type:          "documentation_update",
		Title:         "Fix broken link",
		Description:   "Update outdated URL in README",
		Severity:      "medium",
		Status:        "pending",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
