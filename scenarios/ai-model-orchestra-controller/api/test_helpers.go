package main

import (
	"bytes"
	"context"
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
	"github.com/redis/go-redis/v9"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Create a test logger that writes to stdout
	_ = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() {
		// Cleanup function (currently no-op, but can be extended)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// For this scenario, we don't need file system isolation
	// Just return a cleanup function
	return &TestEnvironment{
		Cleanup: func() {
			// Cleanup function
		},
	}
}

// TestAppState provides a pre-configured application state for testing
type TestAppState struct {
	App     *AppState
	Cleanup func()
}

// setupTestAppState creates a test application state
func setupTestAppState(t *testing.T) *TestAppState {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	app := &AppState{
		Logger: logger,
	}

	return &TestAppState{
		App: app,
		Cleanup: func() {
			// Close connections if they exist
			if app.DB != nil {
				app.DB.Close()
			}
			if app.Redis != nil {
				app.Redis.Close()
			}
			if app.DockerClient != nil {
				app.DockerClient.Close()
			}
		},
	}
}

// setupTestDatabase creates a test database connection (optional - won't fail if unavailable)
func setupTestDatabase(t *testing.T, app *AppState) {
	// Skip database setup if running in CI or environment variables not set
	if os.Getenv("SKIP_DB_TESTS") == "true" {
		t.Log("Skipping database setup (SKIP_DB_TESTS=true)")
		return
	}

	// Try to initialize database, but don't fail if it's not available
	db, err := initDatabase(app.Logger)
	if err != nil {
		t.Logf("Database not available for testing (will use mocks): %v", err)
		return
	}

	app.DB = db

	// Initialize schema
	if err := initSchema(db); err != nil {
		t.Logf("Warning: Failed to initialize schema: %v", err)
	}
}

// setupTestRedis creates a test Redis connection (optional - won't fail if unavailable)
func setupTestRedis(t *testing.T, app *AppState) {
	// Skip Redis setup if running in CI or environment variables not set
	if os.Getenv("SKIP_REDIS_TESTS") == "true" {
		t.Log("Skipping Redis setup (SKIP_REDIS_TESTS=true)")
		return
	}

	// Try to initialize Redis, but don't fail if it's not available
	rdb, err := initRedis(app.Logger)
	if err != nil {
		t.Logf("Redis not available for testing (will use mocks): %v", err)
		return
	}

	app.Redis = rdb
}

// setupTestOllama creates a test Ollama client (optional - won't fail if unavailable)
func setupTestOllama(t *testing.T, app *AppState) {
	// Skip Ollama setup if running in CI or environment variables not set
	if os.Getenv("SKIP_OLLAMA_TESTS") == "true" {
		t.Log("Skipping Ollama setup (SKIP_OLLAMA_TESTS=true)")
		return
	}

	// Try to initialize Ollama, but don't fail if it's not available
	client, err := initOllama(app.Logger)
	if err != nil {
		t.Logf("Ollama not available for testing (will use mocks): %v", err)
		return
	}

	app.OllamaClient = client
}

// createMockOllamaClient creates a mock Ollama client for testing
func createMockOllamaClient() *OllamaClient {
	return &OllamaClient{
		BaseURL: "http://mock-ollama:11434",
		Client:  &http.Client{Timeout: 5 * time.Second},
		CircuitBreaker: &CircuitBreaker{
			maxFailures:  5,
			resetTimeout: 60 * time.Second,
			state:        Closed,
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

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		// Some error responses might not be JSON
		if expectedErrorContains != "" && !bytes.Contains(w.Body.Bytes(), []byte(expectedErrorContains)) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorContains, w.Body.String())
		}
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorContains != "" {
		errorStr := fmt.Sprintf("%v", errorMsg)
		if !bytes.Contains([]byte(errorStr), []byte(expectedErrorContains)) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorContains, errorStr)
		}
	}
}

// createMockDatabase creates an in-memory test database
func createMockDatabase(t *testing.T) (*sql.DB, func()) {
	// Use PostgreSQL test database if available, otherwise skip
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("Skipping database test (TEST_DATABASE_URL not set)")
		return nil, func() {}
	}

	db, err := sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Skipf("Failed to open test database: %v", err)
		return nil, func() {}
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		t.Skipf("Failed to initialize test schema: %v", err)
		return nil, func() {}
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE model_metrics, orchestrator_requests, system_resources CASCADE")
		db.Close()
	}

	return db, cleanup
}

// createMockRedisClient creates a mock Redis client for testing
func createMockRedisClient(t *testing.T) (*redis.Client, func()) {
	// Use real Redis if available, otherwise return nil
	if os.Getenv("TEST_REDIS_URL") == "" {
		return nil, func() {}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("TEST_REDIS_URL"),
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Logf("Redis not available for testing: %v", err)
		return nil, func() {}
	}

	cleanup := func() {
		// Clean up test data
		rdb.FlushDB(ctx)
		rdb.Close()
	}

	return rdb, cleanup
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// setTestEnv sets environment variables for testing and returns a cleanup function
func setTestEnv(t *testing.T, envVars map[string]string) func() {
	originalEnv := make(map[string]string)

	for key, value := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

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
