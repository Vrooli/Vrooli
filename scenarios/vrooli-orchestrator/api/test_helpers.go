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
	originalLogger *Logger
}

// setupTestLogger initializes the logger for testing
func setupTestLogger() func() {
	_ = log.New(io.Discard, "", 0)
	return func() {
		// Restore original if needed
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Service    *OrchestratorService
	Router     *mux.Router
	TempDBName string
	Cleanup    func()
}

// setupTestDB creates an isolated test database
func setupTestDB(t *testing.T) *TestEnvironment {
	// Get database connection details from environment
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		t.Skip("POSTGRES_PASSWORD not set, skipping database tests")
	}

	// Create a unique test database name
	testDBName := fmt.Sprintf("vrooli_orchestrator_test_%d", time.Now().UnixNano())

	// Connect to postgres database to create test database
	adminDBURL := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort)

	adminDB, err := sql.Open("postgres", adminDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}

	// Create test database
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", testDBName))
	if err != nil {
		adminDB.Close()
		t.Fatalf("Failed to create test database: %v", err)
	}
	adminDB.Close()

	// Connect to test database
	testDBURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, testDBName)

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	if err := initSchema(testDB); err != nil {
		testDB.Close()
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create service
	service := NewOrchestratorService(testDB)

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/health", Health).Methods("GET")
	router.HandleFunc("/api/v1/profiles", service.ListProfiles).Methods("GET")
	router.HandleFunc("/api/v1/profiles", service.CreateProfile).Methods("POST")
	router.HandleFunc("/api/v1/profiles/{profileName}", service.GetProfile).Methods("GET")
	router.HandleFunc("/api/v1/profiles/{profileName}", service.UpdateProfile).Methods("PUT")
	router.HandleFunc("/api/v1/profiles/{profileName}", service.DeleteProfile).Methods("DELETE")
	router.HandleFunc("/api/v1/profiles/{profileName}/activate", service.ActivateProfile).Methods("POST")
	router.HandleFunc("/api/v1/profiles/current/deactivate", service.DeactivateProfile).Methods("POST")
	router.HandleFunc("/api/v1/status", service.GetStatus).Methods("GET")

	env := &TestEnvironment{
		DB:         testDB,
		Service:    service,
		Router:     router,
		TempDBName: testDBName,
		Cleanup: func() {
			testDB.Close()

			// Drop test database
			adminDB, err := sql.Open("postgres", adminDBURL)
			if err == nil {
				adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName))
				adminDB.Close()
			}
		},
	}

	return env
}

// initSchema initializes the database schema for testing
func initSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS profiles (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL,
			display_name VARCHAR(255) NOT NULL,
			description TEXT,
			metadata JSONB DEFAULT '{}',
			resources JSONB DEFAULT '[]',
			scenarios JSONB DEFAULT '[]',
			auto_browser JSONB DEFAULT '[]',
			environment_vars JSONB DEFAULT '{}',
			idle_shutdown_minutes INTEGER,
			dependencies JSONB DEFAULT '[]',
			status VARCHAR(50) DEFAULT 'inactive',
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS active_profile (
			id INTEGER PRIMARY KEY DEFAULT 1,
			profile_id VARCHAR(255) REFERENCES profiles(id) ON DELETE SET NULL,
			activated_at TIMESTAMP,
			CONSTRAINT single_active_profile CHECK (id = 1)
		);

		INSERT INTO active_profile (id, profile_id, activated_at)
		VALUES (1, NULL, NULL)
		ON CONFLICT (id) DO NOTHING;
	`

	_, err := db.Exec(schema)
	return err
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(router *mux.Router, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
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

	// Set default content type for POST/PUT
	if req.Method == "POST" || req.Method == "PUT" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(rr, httpReq)

	return rr, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, rr.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	t.Helper()

	response := assertJSONResponse(t, rr, expectedStatus)

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Errorf("Expected error field in response, got: %v", response)
		return
	}

	if expectedErrorSubstring != "" {
		if !containsIgnoreCase(errorMsg, expectedErrorSubstring) {
			t.Errorf("Expected error to contain '%s', got: %s", expectedErrorSubstring, errorMsg)
		}
	}
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 &&
		bytes.Contains(bytes.ToLower([]byte(s)), bytes.ToLower([]byte(substr)))))
}

// createTestProfile creates a test profile in the database
func createTestProfile(t *testing.T, env *TestEnvironment, name string, status string) *Profile {
	t.Helper()

	profileData := map[string]interface{}{
		"name":         name,
		"display_name": fmt.Sprintf("Test Profile %s", name),
		"description":  fmt.Sprintf("Test profile for testing: %s", name),
		"resources":    []string{"postgres", "redis"},
		"scenarios":    []string{"test-scenario"},
		"status":       status,
	}

	profile, err := env.Service.profileManager.CreateProfile(profileData)
	if err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	return profile
}

// cleanupProfiles removes all profiles from the database
func cleanupProfiles(env *TestEnvironment) {
	env.DB.Exec("DELETE FROM profiles")
	env.DB.Exec("UPDATE active_profile SET profile_id = NULL, activated_at = NULL WHERE id = 1")
}

// assertProfileField validates a specific field in a profile response
func assertProfileField(t *testing.T, profile map[string]interface{}, fieldName string, expectedValue interface{}) {
	t.Helper()

	actualValue, exists := profile[fieldName]
	if !exists {
		t.Errorf("Expected field '%s' in profile response", fieldName)
		return
	}

	if expectedValue != nil && actualValue != expectedValue {
		t.Errorf("Expected %s='%v', got '%v'", fieldName, expectedValue, actualValue)
	}
}

// assertProfileListResponse validates a profile list response
func assertProfileListResponse(t *testing.T, response map[string]interface{}, expectedCount int) []interface{} {
	t.Helper()

	profiles, ok := response["profiles"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'profiles' array in response")
	}

	count, ok := response["count"].(float64)
	if !ok {
		t.Errorf("Expected 'count' field in response")
	} else if int(count) != expectedCount {
		t.Errorf("Expected count=%d, got %d", expectedCount, int(count))
	}

	if len(profiles) != expectedCount {
		t.Errorf("Expected %d profiles, got %d", expectedCount, len(profiles))
	}

	return profiles
}

// waitForDatabaseCondition waits for a database condition to be met
func waitForDatabaseCondition(t *testing.T, db *sql.DB, query string, timeout time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var result bool
		err := db.QueryRow(query).Scan(&result)
		if err == nil && result {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
