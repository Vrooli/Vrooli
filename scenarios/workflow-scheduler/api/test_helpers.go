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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Suppress logs during tests
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestDatabase manages test database connection
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use environment variables for test database
	dbURL := os.Getenv("TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Ping to verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Initialize schema
	if err := InitializeDatabase(db); err != nil {
		t.Logf("Warning: Database initialization: %v", err)
	}

	return &TestDatabase{
		DB: db,
		Cleanup: func() {
			// Clean up test data
			db.Exec("DELETE FROM executions")
			db.Exec("DELETE FROM schedules")
			db.Exec("DELETE FROM audit_log")
			db.Exec("DELETE FROM schedule_metrics")
			db.Close()
		},
	}
}

// TestApp wraps the App for testing
type TestApp struct {
	App    *App
	DB     *sql.DB
	Router *mux.Router
}

// setupTestApp creates a test application instance
func setupTestApp(t *testing.T) *TestApp {
	testDB := setupTestDatabase(t)

	app := &App{
		DB:     testDB.DB,
		Router: mux.NewRouter(),
	}
	app.setRoutes()

	// Don't start scheduler in tests
	app.Scheduler = NewScheduler(testDB.DB)

	return &TestApp{
		App:    app,
		DB:     testDB.DB,
		Router: app.Router,
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

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(router *mux.Router, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	// Build URL with query params
	url := req.Path
	if len(req.QueryParams) > 0 {
		url += "?"
		first := true
		for key, value := range req.QueryParams {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	httpReq, err := http.NewRequest(req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httpReq)

	return rr, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	if target != nil {
		if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode error response: %v", err)
	}

	if expectedError != "" {
		if errMsg, ok := response["error"]; !ok || errMsg != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, errMsg)
		}
	}
}

// createTestSchedule creates a test schedule in the database
func createTestSchedule(t *testing.T, db *sql.DB, schedule *Schedule) *Schedule {
	t.Helper()

	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}
	if schedule.Name == "" {
		schedule.Name = "Test Schedule"
	}
	if schedule.CronExpression == "" {
		schedule.CronExpression = "0 * * * *"
	}
	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}
	if schedule.TargetType == "" {
		schedule.TargetType = "webhook"
	}
	if schedule.TargetMethod == "" {
		schedule.TargetMethod = "POST"
	}
	if schedule.Status == "" {
		schedule.Status = "active"
	}

	_, err := db.Exec(`
		INSERT INTO schedules (
			id, name, description, cron_expression, timezone,
			target_type, target_url, target_method, status, enabled,
			overlap_policy, max_retries, retry_strategy, timeout_seconds,
			catch_up_missed, priority, owner, team
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`,
		schedule.ID, schedule.Name, schedule.Description, schedule.CronExpression, schedule.Timezone,
		schedule.TargetType, schedule.TargetURL, schedule.TargetMethod, schedule.Status, schedule.Enabled,
		schedule.OverlapPolicy, schedule.MaxRetries, schedule.RetryStrategy, schedule.TimeoutSeconds,
		schedule.CatchUpMissed, schedule.Priority, schedule.Owner, schedule.Team,
	)

	if err != nil {
		t.Fatalf("Failed to create test schedule: %v", err)
	}

	return schedule
}

// createTestExecution creates a test execution in the database
func createTestExecution(t *testing.T, db *sql.DB, scheduleID string, status string) string {
	t.Helper()

	executionID := uuid.New().String()
	now := time.Now()

	_, err := db.Exec(`
		INSERT INTO executions (
			id, schedule_id, scheduled_time, start_time, end_time,
			status, attempt_count, is_manual_trigger
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		executionID, scheduleID, now, now, now.Add(time.Second),
		status, 1, false,
	)

	if err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}

	return executionID
}

// cleanupTestData removes all test data from the database
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	db.Exec("DELETE FROM executions")
	db.Exec("DELETE FROM schedules")
	db.Exec("DELETE FROM audit_log")
	db.Exec("DELETE FROM schedule_metrics")
}
