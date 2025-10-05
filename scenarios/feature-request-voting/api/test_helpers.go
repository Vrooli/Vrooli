//go:build testing
// +build testing

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes logger for testing
func setupTestLogger() func() {
	// Suppress logs during tests unless VERBOSE is set
	if os.Getenv("VERBOSE") != "true" {
		log.SetOutput(ioutil.Discard)
	}
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestServer wraps Server for testing
type TestServer struct {
	Server *Server
	DB     *sql.DB
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *sql.DB {
	// Use test database environment variables
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

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "feature_voting_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Database not available: %v", err)
		return nil
	}

	// Quick connectivity check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("Database not available: %v", err)
		return nil
	}

	// Check if schema is initialized
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'scenarios'").Scan(&count)
	if err != nil || count == 0 {
		t.Skipf("Database schema not initialized (run migrations first)")
		return nil
	}

	return db
}

// cleanupTestDB cleans up test database
func cleanupTestDB(db *sql.DB) {
	if db != nil {
		// Clean up test data
		db.Exec("TRUNCATE TABLE votes CASCADE")
		db.Exec("TRUNCATE TABLE comments CASCADE")
		db.Exec("TRUNCATE TABLE status_changes CASCADE")
		db.Exec("TRUNCATE TABLE feature_requests CASCADE")
		db.Exec("TRUNCATE TABLE scenarios CASCADE")
		db.Exec("TRUNCATE TABLE users CASCADE")
		db.Close()
	}
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T) *TestServer {
	db := setupTestDB(t)
	if db == nil {
		return nil
	}

	server := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	server.setupRoutes()

	return &TestServer{
		Server: server,
		DB:     db,
	}
}

// TestScenario represents a test scenario
type TestScenario struct {
	ID          string
	Name        string
	DisplayName string
	Description string
}

// createTestScenario creates a scenario for testing
func createTestScenario(t *testing.T, db *sql.DB, name string) *TestScenario {
	id := uuid.New().String()
	displayName := fmt.Sprintf("Test Scenario: %s", name)
	description := fmt.Sprintf("Description for %s", name)

	_, err := db.Exec(`
		INSERT INTO scenarios (id, name, display_name, description, auth_config)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, id, name, displayName, description, `{"mode": "public"}`)

	if err != nil {
		t.Fatalf("Failed to create test scenario: %v", err)
	}

	return &TestScenario{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
		Description: description,
	}
}

// TestFeatureRequest represents a test feature request
type TestFeatureRequest struct {
	ID          string
	ScenarioID  string
	Title       string
	Description string
	Status      string
	Priority    string
	VoteCount   int
}

// createTestFeatureRequest creates a feature request for testing
func createTestFeatureRequest(t *testing.T, db *sql.DB, scenarioID string, title string) *TestFeatureRequest {
	id := uuid.New().String()
	description := fmt.Sprintf("Description for %s", title)

	_, err := db.Exec(`
		INSERT INTO feature_requests (id, scenario_id, title, description, status, priority, tags)
		VALUES ($1, $2, $3, $4, $5::feature_status, $6::priority_level, $7)
	`, id, scenarioID, title, description, "proposed", "medium", []string{})

	if err != nil {
		t.Fatalf("Failed to create test feature request: %v", err)
	}

	return &TestFeatureRequest{
		ID:          id,
		ScenarioID:  scenarioID,
		Title:       title,
		Description: description,
		Status:      "proposed",
		Priority:    "medium",
		VoteCount:   0,
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
func makeHTTPRequest(server *Server, req HTTPTestRequest) *httptest.ResponseRecorder {
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
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
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
	server.router.ServeHTTP(w, httpReq)
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

	if expectedErrorMessage != "" && !strings.Contains(errorMsg.(string), expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, errorMsg)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateFeatureRequestPayload creates a test feature request payload
func (g *TestDataGenerator) CreateFeatureRequestPayload(scenarioID, title, description string) CreateFeatureRequestRequest {
	return CreateFeatureRequestRequest{
		ScenarioID:  scenarioID,
		Title:       title,
		Description: description,
		Priority:    "medium",
		Tags:        []string{"test"},
	}
}

// UpdateFeatureRequestPayload creates a test update payload
func (g *TestDataGenerator) UpdateFeatureRequestPayload(title, description, status string) UpdateFeatureRequestRequest {
	return UpdateFeatureRequestRequest{
		Title:       &title,
		Description: &description,
		Status:      &status,
	}
}

// VotePayload creates a test vote payload
func (g *TestDataGenerator) VotePayload(value int) VoteRequest {
	return VoteRequest{
		Value: value,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
