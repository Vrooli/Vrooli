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

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes a test logger that suppresses output
func setupTestLogger() func() {
	// Suppress log output during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "1" {
		log.SetOutput(io.Discard)
	}

	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Config  *Config
	Router  *http.Handler
	Cleanup func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Set required environment variables for testing
	os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test_db")
	os.Setenv("API_PORT", "9999")
	os.Setenv("N8N_BASE_URL", "http://localhost:5678")
	os.Setenv("WINDMILL_BASE_URL", "http://localhost:8000")
	os.Setenv("QDRANT_URL", "http://localhost:6333")

	config := loadConfig()

	return &TestEnvironment{
		Config: config,
		Cleanup: func() {
			// Cleanup environment variables
			os.Unsetenv("POSTGRES_URL")
			os.Unsetenv("API_PORT")
			os.Unsetenv("N8N_BASE_URL")
			os.Unsetenv("WINDMILL_BASE_URL")
			os.Unsetenv("QDRANT_URL")
		},
	}
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(t *testing.T, router *http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rr := httptest.NewRecorder()
	if router != nil {
		(*router).ServeHTTP(rr, req)
	} else {
		t.Fatal("Router is nil")
	}

	return rr
}

// assertJSONResponse validates that a response contains valid JSON
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status code %d, got %d", expectedStatus, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v\nBody: %s", err, rr.Body.String())
	}

	return response
}

// assertErrorResponse validates that a response is an error with expected format
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	response := assertJSONResponse(t, rr, expectedStatus)

	if success, ok := response["success"].(bool); ok {
		if success {
			t.Error("Expected success to be false for error response")
		}
	}

	if expectedMessage != "" {
		if message, ok := response["message"].(string); ok {
			if message != expectedMessage {
				t.Errorf("Expected message '%s', got '%s'", expectedMessage, message)
			}
		}
	}
}

// assertSuccessResponse validates that a response indicates success
func assertSuccessResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()

	response := assertJSONResponse(t, rr, http.StatusOK)

	if success, ok := response["success"].(bool); ok {
		if !success {
			t.Error("Expected success to be true")
		}
	}

	return response
}

// assertArrayResponse validates that a response contains an array field
func assertArrayResponse(t *testing.T, response map[string]interface{}, fieldName string, minLength int) []interface{} {
	t.Helper()

	field, ok := response[fieldName]
	if !ok {
		t.Fatalf("Response missing '%s' field", fieldName)
	}

	array, ok := field.([]interface{})
	if !ok {
		t.Fatalf("Field '%s' is not an array", fieldName)
	}

	if len(array) < minLength {
		t.Errorf("Expected at least %d items in '%s', got %d", minLength, fieldName, len(array))
	}

	return array
}

// assertFieldExists validates that a field exists in a map
func assertFieldExists(t *testing.T, data map[string]interface{}, fieldName string) interface{} {
	t.Helper()

	value, ok := data[fieldName]
	if !ok {
		t.Errorf("Expected field '%s' to exist", fieldName)
		return nil
	}

	return value
}

// assertStringField validates that a field is a string with expected value
func assertStringField(t *testing.T, data map[string]interface{}, fieldName, expectedValue string) {
	t.Helper()

	value := assertFieldExists(t, data, fieldName)
	if value == nil {
		return
	}

	strValue, ok := value.(string)
	if !ok {
		t.Errorf("Field '%s' is not a string", fieldName)
		return
	}

	if expectedValue != "" && strValue != expectedValue {
		t.Errorf("Expected field '%s' to be '%s', got '%s'", fieldName, expectedValue, strValue)
	}
}

// assertIntField validates that a field is an integer
func assertIntField(t *testing.T, data map[string]interface{}, fieldName string) int {
	t.Helper()

	value := assertFieldExists(t, data, fieldName)
	if value == nil {
		return 0
	}

	// JSON numbers are float64
	floatValue, ok := value.(float64)
	if !ok {
		t.Errorf("Field '%s' is not a number", fieldName)
		return 0
	}

	return int(floatValue)
}

// createTestRouter creates a router instance for testing
func createTestRouter(config *Config) http.Handler {
	return setupRoutes(config)
}

// mockJobsData returns mock job data for testing
func mockJobsData() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"id":                  1,
			"job_title":           "Senior Software Engineer",
			"company_name":        "Tech Innovations Inc",
			"location":            "San Francisco, CA",
			"experience_required": 5,
			"candidate_count":     12,
			"status":              "active",
		},
		map[string]interface{}{
			"id":                  2,
			"job_title":           "Data Scientist",
			"company_name":        "AI Solutions Corp",
			"location":            "New York, NY",
			"experience_required": 3,
			"candidate_count":     8,
			"status":              "active",
		},
	}
}

// mockCandidatesData returns mock candidate data for testing
func mockCandidatesData() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"id":              1,
			"candidate_name":  "Alice Johnson",
			"email":           "alice.johnson@email.com",
			"score":           0.92,
			"experience_years": 6,
			"parsed_skills":   []string{"Python", "Machine Learning", "PostgreSQL", "Docker"},
			"status":          "reviewed",
			"job_id":          1,
		},
		map[string]interface{}{
			"id":              2,
			"candidate_name":  "Bob Smith",
			"email":           "bob.smith@email.com",
			"score":           0.85,
			"experience_years": 4,
			"parsed_skills":   []string{"JavaScript", "React", "Node.js", "AWS"},
			"status":          "pending",
			"job_id":          1,
		},
	}
}
