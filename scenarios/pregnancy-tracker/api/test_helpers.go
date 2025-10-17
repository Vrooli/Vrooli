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

	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Redirect log output to reduce noise during tests
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	OriginalDB *sql.DB
	Cleanup    func()
}

// setupTestDB creates an isolated test database environment with proper cleanup
func setupTestDB(t *testing.T) *TestEnvironment {
	// Store original database connection
	originalDB := db

	// Create test database connection
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Build from components for testing
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			getEnvOrDefault("POSTGRES_USER", "postgres"),
			getEnvOrDefault("POSTGRES_PASSWORD", "postgres"),
			getEnvOrDefault("POSTGRES_HOST", "localhost"),
			getEnvOrDefault("POSTGRES_PORT", "5432"),
			getEnvOrDefault("POSTGRES_DB", "pregnancy_tracker_test"))
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Set connection pool settings
	testDB.SetMaxOpenConns(10)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := testDB.Ping(); err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Set search path
	_, err = testDB.Exec("SET search_path TO pregnancy_tracker, public")
	if err != nil {
		t.Logf("Warning: Failed to set search path: %v", err)
	}

	// Replace global db with test db
	db = testDB

	return &TestEnvironment{
		DB:         testDB,
		OriginalDB: originalDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("TRUNCATE TABLE daily_logs, kick_counts, appointments, pregnancies CASCADE")
			testDB.Close()
			db = originalDB
		},
	}
}

// getEnvOrDefault retrieves environment variable with fallback for testing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// TestPregnancy provides a pre-configured pregnancy for testing
type TestPregnancy struct {
	Pregnancy *Pregnancy
	UserID    string
	Cleanup   func()
}

// setupTestPregnancy creates a test pregnancy with sample data
func setupTestPregnancy(t *testing.T, userID string) *TestPregnancy {
	if userID == "" {
		userID = "test-user-123"
	}

	lmpDate := time.Now().AddDate(0, 0, -70) // 10 weeks pregnant
	dueDate := lmpDate.AddDate(0, 0, 280)    // Standard 40-week pregnancy

	var id string
	err := db.QueryRow(`
		INSERT INTO pregnancies (user_id, lmp_date, due_date, current_week, current_day, pregnancy_type, outcome)
		VALUES ($1, $2, $3, 10, 0, 'singleton', 'ongoing')
		RETURNING id
	`, userID, lmpDate, dueDate).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test pregnancy: %v", err)
	}

	pregnancy := &Pregnancy{
		ID:            id,
		UserID:        userID,
		LMPDate:       lmpDate,
		DueDate:       dueDate,
		CurrentWeek:   10,
		CurrentDay:    0,
		PregnancyType: "singleton",
		Outcome:       "ongoing",
		CreatedAt:     time.Now(),
	}

	return &TestPregnancy{
		Pregnancy: pregnancy,
		UserID:    userID,
		Cleanup: func() {
			db.Exec("DELETE FROM pregnancies WHERE id = $1", id)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	UserID  string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request) {
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
				panic(fmt.Errorf("failed to marshal request body: %v", err))
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

	// Set user ID header
	if req.UserID != "" {
		httpReq.Header.Set("X-User-ID", req.UserID)
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	return w, httpReq
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

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// PregnancyStartRequest creates a test pregnancy start request
func (g *TestDataGenerator) PregnancyStartRequest(lmpDate time.Time) map[string]interface{} {
	return map[string]interface{}{
		"lmp_date": lmpDate.Format("2006-01-02"),
		"due_date": lmpDate.AddDate(0, 0, 280).Format("2006-01-02"),
	}
}

// DailyLogRequest creates a test daily log request
func (g *TestDataGenerator) DailyLogRequest() DailyLog {
	weight := 150.5
	return DailyLog{
		Weight:        &weight,
		BloodPressure: "120/80",
		Symptoms:      []string{"fatigue", "nausea"},
		Mood:          8,
		Energy:        7,
		Notes:         "Feeling good today",
		Photos:        []string{},
	}
}

// KickCountRequest creates a test kick count request
func (g *TestDataGenerator) KickCountRequest() KickCount {
	sessionStart := time.Now().Add(-1 * time.Hour)
	sessionEnd := time.Now()
	return KickCount{
		SessionStart: sessionStart,
		SessionEnd:   &sessionEnd,
		Count:        10,
		Duration:     60,
		Notes:        "Normal activity",
	}
}

// AppointmentRequest creates a test appointment request
func (g *TestDataGenerator) AppointmentRequest() Appointment {
	return Appointment{
		Date:     time.Now().AddDate(0, 0, 7),
		Type:     "ultrasound",
		Provider: "Dr. Smith",
		Location: "Medical Center",
		Notes:    "Routine checkup",
		Results:  map[string]interface{}{"heartbeat": "normal"},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// setupTestEnvironment sets up complete test environment
func setupTestEnvironment(t *testing.T) (*TestEnvironment, func()) {
	// Setup encryption key for tests
	if os.Getenv("ENCRYPTION_KEY") == "" {
		os.Setenv("ENCRYPTION_KEY", "test-encryption-key-32-bytes!!")
	}
	if os.Getenv("PRIVACY_MODE") == "" {
		os.Setenv("PRIVACY_MODE", "high")
	}
	if os.Getenv("MULTI_TENANT") == "" {
		os.Setenv("MULTI_TENANT", "true")
	}

	loggerCleanup := setupTestLogger()
	env := setupTestDB(t)

	return env, func() {
		env.Cleanup()
		loggerCleanup()
	}
}
