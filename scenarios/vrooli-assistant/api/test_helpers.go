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
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Silence logs during tests unless verbose mode is enabled
	originalOutput := log.Writer()
	if os.Getenv("TEST_VERBOSE") != "1" {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestDB manages test database lifecycle
type TestDB struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an isolated test database
func setupTestDB(t *testing.T) *TestDB {
	// Use a dedicated test database
	testDBURL := os.Getenv("TEST_POSTGRES_URL")
	if testDBURL == "" {
		// Fall back to constructing from parts
		dbHost := getEnvOrDefault("TEST_POSTGRES_HOST", "localhost")
		dbPort := getEnvOrDefault("TEST_POSTGRES_PORT", "5432")
		dbUser := getEnvOrDefault("TEST_POSTGRES_USER", "postgres")
		dbPassword := getEnvOrDefault("TEST_POSTGRES_PASSWORD", "postgres")
		dbName := getEnvOrDefault("TEST_POSTGRES_DB", "vrooli_assistant_test")

		testDBURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v. Set TEST_POSTGRES_URL or run with database available", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		t.Skipf("Cannot ping test database: %v", err)
		return nil
	}

	// Clear existing test data
	cleanupTestData(t, testDB)

	// Set the global db variable
	originalDB := db
	db = testDB

	return &TestDB{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(t, testDB)
			db = originalDB
			testDB.Close()
		},
	}
}

// cleanupTestData removes all test data from tables
func cleanupTestData(t *testing.T, testDB *sql.DB) {
	tables := []string{"agent_sessions", "issues"}
	for _, table := range tables {
		_, err := testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			// Table might not exist yet, that's ok
			t.Logf("Failed to clean %s: %v", table, err)
		}
	}
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest, handler http.Handler) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type for POST/PUT requests
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars (for mux variables like {id})
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Execute request
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httpReq)

	return recorder, nil
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	body := w.Body.String()
	if expectedMessage != "" && body != expectedMessage+"\n" {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, body)
	}
}

// assertFieldExists checks if a field exists in response
func assertFieldExists(t *testing.T, response map[string]interface{}, field string) {
	if _, exists := response[field]; !exists {
		t.Errorf("Expected field '%s' in response, but it was missing. Response: %v", field, response)
	}
}

// assertFieldEquals checks if a field has expected value
func assertFieldEquals(t *testing.T, response map[string]interface{}, field string, expected interface{}) {
	actual, exists := response[field]
	if !exists {
		t.Errorf("Field '%s' not found in response", field)
		return
	}
	if actual != expected {
		t.Errorf("Field '%s': expected '%v', got '%v'", field, expected, actual)
	}
}

// createTestIssue creates a test issue in the database
func createTestIssue(t *testing.T, testDB *sql.DB) string {
	issueID := "test-issue-" + generateUUID()
	_, err := testDB.Exec(`
		INSERT INTO issues (id, timestamp, description, status, screenshot_path, scenario_name, url, context_data)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7)`,
		issueID,
		"Test issue description",
		"captured",
		"/tmp/test-screenshot.png",
		"test-scenario",
		"http://localhost:3000/test",
		`{"test": true}`,
	)
	if err != nil {
		t.Fatalf("Failed to create test issue: %v", err)
	}
	return issueID
}

// createTestAgentSession creates a test agent session
func createTestAgentSession(t *testing.T, testDB *sql.DB, issueID string) string {
	sessionID := "test-session-" + generateUUID()
	_, err := testDB.Exec(`
		INSERT INTO agent_sessions (id, issue_id, agent_type, start_time, status)
		VALUES ($1, $2, $3, NOW(), $4)`,
		sessionID,
		issueID,
		"test-agent",
		"running",
	)
	if err != nil {
		t.Fatalf("Failed to create test agent session: %v", err)
	}
	return sessionID
}

// generateUUID creates a simple UUID-like string for testing
func generateUUID() string {
	return uuid.New().String()
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/assistant/status", statusHandler).Methods("GET")
	router.HandleFunc("/api/v1/assistant/capture", captureHandler).Methods("POST")
	router.HandleFunc("/api/v1/assistant/spawn-agent", spawnAgentHandler).Methods("POST")
	router.HandleFunc("/api/v1/assistant/history", historyHandler).Methods("GET")
	router.HandleFunc("/api/v1/assistant/issues/{id}", issueHandler).Methods("GET")
	router.HandleFunc("/api/v1/assistant/issues/{id}/status", updateStatusHandler).Methods("PUT")
	return router
}
