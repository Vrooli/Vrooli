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
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	log.SetOutput(ioutil.Discard) // Suppress logs during tests
	return func() { log.SetOutput(os.Stdout) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Cleanup    func()
	Server     *httptest.Server
}

// setupTestDB creates an isolated test database environment with proper cleanup
func setupTestDB(t *testing.T) *TestEnvironment {
	// Use test database
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		dbHost := getEnvOrDefault("POSTGRES_HOST", "localhost")
		dbPort := getEnvOrDefault("POSTGRES_PORT", "5432")
		dbUser := getEnvOrDefault("POSTGRES_USER", "postgres")
		dbPassword := getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
		dbName := getEnvOrDefault("POSTGRES_DB", "competitor_monitor_test")

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping test: failed to open database connection: %v", err)
		return nil
	}

	// Test connection with timeout
	testDB.SetConnMaxLifetime(5 * time.Second)
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Create tables if they don't exist (for testing without full schema)
	createTestTables(testDB)

	// Clean up test data before each test
	cleanupTestData(testDB)

	return &TestEnvironment{
		DB: testDB,
		Cleanup: func() {
			cleanupTestData(testDB)
			testDB.Close()
		},
	}
}

// createTestTables creates minimal tables needed for testing
func createTestTables(testDB *sql.DB) {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS competitors (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			category VARCHAR(100),
			importance VARCHAR(50) DEFAULT 'medium',
			is_active BOOLEAN DEFAULT true,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS monitoring_targets (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			competitor_id TEXT,
			url TEXT NOT NULL,
			target_type VARCHAR(50) NOT NULL,
			selector TEXT,
			check_frequency INTEGER DEFAULT 3600,
			last_checked TIMESTAMP WITH TIME ZONE,
			last_content_hash VARCHAR(64),
			is_active BOOLEAN DEFAULT true,
			config JSONB DEFAULT '{}'
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			competitor_id TEXT,
			title VARCHAR(500),
			priority VARCHAR(50),
			url TEXT,
			category VARCHAR(100),
			summary TEXT,
			insights JSONB,
			actions JSONB,
			relevance_score INTEGER,
			status VARCHAR(50) DEFAULT 'unread',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS change_analyses (
			id TEXT PRIMARY KEY,
			competitor_id TEXT,
			target_url TEXT,
			change_type VARCHAR(50),
			relevance_score INTEGER,
			change_category VARCHAR(100),
			impact_level VARCHAR(50),
			key_insights JSONB,
			recommended_actions JSONB,
			summary TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range tables {
		testDB.Exec(query)
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// cleanupTestData removes all test data from tables
func cleanupTestData(testDB *sql.DB) {
	tables := []string{
		"change_analyses",
		"alerts",
		"monitoring_targets",
		"competitors",
	}

	for _, table := range tables {
		testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// TestCompetitor provides a pre-configured competitor for testing
type TestCompetitor struct {
	Competitor *Competitor
	Cleanup    func()
}

// setupTestCompetitor creates a test competitor with sample data
func setupTestCompetitor(t *testing.T, env *TestEnvironment, name string) *TestCompetitor {
	competitor := &Competitor{
		Name:        name,
		Description: fmt.Sprintf("Test competitor: %s", name),
		Category:    "technology",
		Importance:  "high",
		IsActive:    true,
		Metadata:    json.RawMessage(`{"test": true}`),
	}

	var id string
	err := env.DB.QueryRow(`
		INSERT INTO competitors (name, description, category, importance, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, competitor.Name, competitor.Description, competitor.Category, competitor.Importance, competitor.Metadata).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test competitor: %v", err)
	}

	competitor.ID = id

	return &TestCompetitor{
		Competitor: competitor,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM competitors WHERE id = $1", id)
		},
	}
}

// TestMonitoringTarget provides a pre-configured monitoring target for testing
type TestMonitoringTarget struct {
	Target  *MonitoringTarget
	Cleanup func()
}

// setupTestTarget creates a test monitoring target
func setupTestTarget(t *testing.T, env *TestEnvironment, competitorID string) *TestMonitoringTarget {
	target := &MonitoringTarget{
		CompetitorID:   competitorID,
		URL:            "https://example.com/test",
		TargetType:     "website",
		Selector:       ".content",
		CheckFrequency: 3600,
		IsActive:       true,
		Config:         json.RawMessage(`{"test": true}`),
	}

	var id string
	err := env.DB.QueryRow(`
		INSERT INTO monitoring_targets (competitor_id, url, target_type, selector, check_frequency, config)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, target.CompetitorID, target.URL, target.TargetType, target.Selector, target.CheckFrequency, target.Config).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test target: %v", err)
	}

	target.ID = id

	return &TestMonitoringTarget{
		Target: target,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM monitoring_targets WHERE id = $1", id)
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
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var array []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &array); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return array
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// testHandlerWithRequest is a helper for testing handlers with specific requests
func testHandlerWithRequest(t *testing.T, handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	handler(w, httpReq)
	return w
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateCompetitorRequest creates a test competitor creation request
func (g *TestDataGenerator) CreateCompetitorRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Test competitor: %s", name),
		"category":    "technology",
		"importance":  "high",
		"metadata":    map[string]interface{}{"test": true},
	}
}

// CreateTargetRequest creates a test monitoring target request
func (g *TestDataGenerator) CreateTargetRequest(competitorID string) map[string]interface{} {
	return map[string]interface{}{
		"competitor_id":   competitorID,
		"url":             "https://example.com/test",
		"target_type":     "website",
		"selector":        ".content",
		"check_frequency": 3600,
		"config":          map[string]interface{}{"test": true},
	}
}

// CreateAlertRequest creates a test alert request
func (g *TestDataGenerator) CreateAlertRequest(competitorID string) map[string]interface{} {
	return map[string]interface{}{
		"competitor_id":   competitorID,
		"title":           "Test Alert",
		"priority":        "high",
		"url":             "https://example.com/alert",
		"category":        "product_launch",
		"summary":         "Test alert summary",
		"insights":        map[string]interface{}{"key": "value"},
		"actions":         map[string]interface{}{"action": "review"},
		"relevance_score": 85,
		"status":          "unread",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for: %s", message)
}
