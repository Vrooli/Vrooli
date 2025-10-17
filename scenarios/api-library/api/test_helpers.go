
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
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Create a test logger for output
	log.SetPrefix("[test] ")
	return func() {
		log.SetPrefix("")
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Redis      *redis.Client
	Router     *mux.Router
	Cleanup    func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use environment variables or test defaults
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "test_api_library"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "test"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil, func() {}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: cannot ping test database: %v", err)
		return nil, func() {}
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DELETE FROM notes WHERE created_at > NOW() - INTERVAL '1 hour'")
		db.Exec("DELETE FROM pricing_tiers WHERE created_at > NOW() - INTERVAL '1 hour'")
		db.Exec("DELETE FROM api_versions WHERE created_at > NOW() - INTERVAL '1 hour'")
		db.Exec("DELETE FROM apis WHERE created_at > NOW() - INTERVAL '1 hour'")
		db.Close()
	}

	return db, cleanup
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	loggerCleanup := setupTestLogger()

	db, dbCleanup := setupTestDB(t)
	if db == nil {
		return nil
	}

	// Set global db for handlers - using db variable
	var router *mux.Router
	if db != nil {
		router = setupRouter()
	}

	cleanup := func() {
		dbCleanup()
		loggerCleanup()
	}

	return &TestEnvironment{
		DB:      db,
		Router:  router,
		Cleanup: cleanup,
	}
}

// TestAPI provides a pre-configured API for testing
type TestAPI struct {
	API     *API
	Cleanup func()
}

// setupTestAPI creates a test API with sample data
func setupTestAPI(t *testing.T, db *sql.DB, name string) *TestAPI {
	api := &API{
		ID:               uuid.New().String(),
		Name:             name,
		Provider:         "Test Provider",
		Description:      fmt.Sprintf("Test API: %s", name),
		BaseURL:          fmt.Sprintf("https://api.%s.test", strings.ToLower(name)),
		DocumentationURL: fmt.Sprintf("https://docs.%s.test", strings.ToLower(name)),
		Category:         "testing",
		Status:           "active",
		AuthType:         "api_key",
		Tags:             []string{"test", "example"},
		Capabilities:     []string{"testing", "mocking"},
		Version:          "1.0.0",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Insert into database if provided
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO apis (id, name, provider, description, base_url, documentation_url,
				category, status, auth_type, tags, capabilities, version, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			api.ID, api.Name, api.Provider, api.Description, api.BaseURL, api.DocumentationURL,
			api.Category, api.Status, api.AuthType, pq.Array(api.Tags), pq.Array(api.Capabilities),
			api.Version, api.CreatedAt, api.UpdatedAt)
		if err != nil {
			t.Fatalf("Failed to insert test API: %v", err)
		}
	}

	return &TestAPI{
		API: api,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM apis WHERE id = $1", api.ID)
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

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
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
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
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
	return w, httpReq, nil
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

	// Validate expected fields
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

	if expectedErrorMessage != "" && !strings.Contains(fmt.Sprintf("%v", errorMsg), expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%v'", expectedErrorMessage, errorMsg)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchRequest(query string, limit int) SearchRequest {
	return SearchRequest{
		Query: query,
		Limit: limit,
	}
}

// CreateAPIRequest creates a test API creation request
func (g *TestDataGenerator) CreateAPIRequest(name string) API {
	return API{
		Name:             name,
		Provider:         "Test Provider",
		Description:      fmt.Sprintf("Test API: %s", name),
		BaseURL:          fmt.Sprintf("https://api.%s.test", strings.ToLower(name)),
		DocumentationURL: fmt.Sprintf("https://docs.%s.test", strings.ToLower(name)),
		Category:         "testing",
		Status:           "active",
		AuthType:         "api_key",
		Tags:             []string{"test"},
		Capabilities:     []string{"testing"},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
