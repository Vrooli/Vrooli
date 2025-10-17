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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Suppress logs during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		log.SetOutput(io.Discard)
	}
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Router     *gin.Engine
	Cleanup    func()
	TestServer *httptest.Server
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use the same database as the main application for integration tests
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		dbHost := os.Getenv("POSTGRES_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("POSTGRES_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("POSTGRES_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = "postgres"
		}
		dbName := "contact_book"

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test - database ping failed: %v", err)
		return nil
	}

	return testDB
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Set test mode
	gin.SetMode(gin.TestMode)

	testDB := setupTestDB(t)
	if testDB == nil {
		return nil // Database not available, test will be skipped
	}

	// Store original DB and replace with test DB
	originalDB := db
	db = testDB

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())

	// Setup routes (simplified version without external dependencies)
	router.GET("/health", healthCheck)

	api := router.Group("/api/v1")
	{
		api.GET("/contacts", getPersons)
		api.GET("/contacts/:id", getPerson)
		api.POST("/contacts", createPerson)
		api.GET("/contacts/by-auth/:authID", getPersonByAuthID)
		api.PUT("/contacts/:id", updatePerson)
		api.GET("/relationships", getRelationships)
		api.POST("/relationships", createRelationship)
		api.GET("/analytics", getSocialAnalytics)
		api.POST("/search", searchContacts)
	}

	// Create test server
	testServer := httptest.NewServer(router)

	return &TestEnvironment{
		DB:         testDB,
		Router:     router,
		TestServer: testServer,
		Cleanup: func() {
			testServer.Close()
			testDB.Close()
			db = originalDB
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

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorField string) {
	t.Helper()

	response := assertJSONResponse(t, w, expectedStatus)

	if _, exists := response["error"]; !exists && expectedErrorField != "" {
		t.Errorf("Expected error field in response, got: %v", response)
	}
}

// createTestPerson creates a test person in the database
func createTestPerson(t *testing.T, env *TestEnvironment, fullName string) string {
	t.Helper()

	req := CreatePersonRequest{
		FullName: fullName,
		Emails:   []string{fmt.Sprintf("%s@test.com", fullName)},
		Tags:     []string{"test"},
	}

	w, err := makeHTTPRequest(env, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/contacts",
		Body:   req,
	})

	if err != nil {
		t.Fatalf("Failed to create test person: %v", err)
	}

	response := assertJSONResponse(t, w, http.StatusCreated)
	id, ok := response["id"].(string)
	if !ok {
		t.Fatalf("Failed to get person ID from response: %v", response)
	}

	return id
}

// createTestRelationship creates a test relationship in the database
func createTestRelationship(t *testing.T, env *TestEnvironment, fromID, toID, relType string) string {
	t.Helper()

	strength := 0.8
	req := CreateRelationshipRequest{
		FromPersonID:     fromID,
		ToPersonID:       toID,
		RelationshipType: relType,
		Strength:         &strength,
	}

	w, err := makeHTTPRequest(env, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/relationships",
		Body:   req,
	})

	if err != nil {
		t.Fatalf("Failed to create test relationship: %v", err)
	}

	response := assertJSONResponse(t, w, http.StatusCreated)
	id, ok := response["id"].(string)
	if !ok {
		t.Fatalf("Failed to get relationship ID from response: %v", response)
	}

	return id
}

// cleanupTestPerson removes a test person from the database
func cleanupTestPerson(t *testing.T, env *TestEnvironment, personID string) {
	t.Helper()

	query := "UPDATE persons SET deleted_at = NOW() WHERE id = $1"
	if _, err := env.DB.Exec(query, personID); err != nil {
		t.Logf("Warning: Failed to cleanup test person %s: %v", personID, err)
	}
}

// cleanupTestRelationship removes a test relationship from the database
func cleanupTestRelationship(t *testing.T, env *TestEnvironment, relationshipID string) {
	t.Helper()

	query := "UPDATE relationships SET deleted_at = NOW() WHERE id = $1"
	if _, err := env.DB.Exec(query, relationshipID); err != nil {
		t.Logf("Warning: Failed to cleanup test relationship %s: %v", relationshipID, err)
	}
}

// waitForCondition polls a condition until it's true or times out
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}

// generateUniqueEmail generates a unique email for testing
func generateUniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@test.com", prefix, uuid.New().String()[:8])
}

// compareFloats compares two floats with tolerance
func compareFloats(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
