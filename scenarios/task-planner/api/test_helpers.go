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
)

// testLoggerType provides controlled logging during tests
type testLoggerType struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Redirect log output to avoid test pollution
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Service    *TaskPlannerService
	Router     *mux.Router
	Cleanup    func()
	TestAppID  uuid.UUID
	TestTaskID uuid.UUID
}

// setupTestEnvironment creates an isolated test environment with in-memory database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Create in-memory database for testing
	db, err := sql.Open("postgres", "postgres://localhost:5433/vrooli_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Configure connection pool for tests
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(1 * time.Minute)

	// Create service
	service := NewTaskPlannerService(db, "http://localhost:6333")

	// Setup routes
	r := mux.NewRouter()
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/apps", service.GetApps).Methods("GET")
	r.HandleFunc("/api/tasks", service.GetTasks).Methods("GET")
	r.HandleFunc("/api/parse-text", service.ParseText).Methods("POST")
	r.HandleFunc("/api/tasks/{taskId}/research", service.ResearchTask).Methods("POST")
	r.HandleFunc("/api/tasks/status", service.UpdateTaskStatus).Methods("PUT")
	r.HandleFunc("/api/tasks/status-history", service.GetTaskStatusHistory).Methods("GET")

	// Create test app and task IDs
	testAppID := uuid.New()
	testTaskID := uuid.New()

	return &TestEnvironment{
		DB:         db,
		Service:    service,
		Router:     r,
		TestAppID:  testAppID,
		TestTaskID: testTaskID,
		Cleanup: func() {
			db.Close()
		},
	}
}

// HTTPTestRequest defines a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
	URLVars     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Build URL with query params
	url := req.Path
	if len(req.QueryParams) > 0 {
		url += "?"
		first := true
		for k, v := range req.QueryParams {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
	}

	httpReq, err := http.NewRequest(req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set URL vars if using mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
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

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	response := assertJSONResponse(t, w, expectedStatus)

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Fatalf("Expected error field in response, got: %v", response)
	}

	if expectedErrorContains != "" && !contains(errorMsg, expectedErrorContains) {
		t.Errorf("Expected error to contain '%s', got: %s", expectedErrorContains, errorMsg)
	}
}

// assertSuccessResponse validates a successful response with expected fields
func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedFields []string) map[string]interface{} {
	t.Helper()

	response := assertJSONResponse(t, w, http.StatusOK)

	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in response, got: %v", field, response)
		}
	}

	return response
}

// createTestApp inserts a test app into the database
func createTestApp(t *testing.T, env *TestEnvironment, name string) uuid.UUID {
	t.Helper()

	appID := uuid.New()
	apiToken := uuid.New().String()

	query := `
		INSERT INTO apps (id, name, display_name, type, api_token, total_tasks, completed_tasks, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := env.DB.Exec(query, appID, name, name+" Display", "web-application", apiToken, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create test app: %v", err)
	}

	return appID
}

// createTestTask inserts a test task into the database
func createTestTask(t *testing.T, env *TestEnvironment, appID uuid.UUID, title string, status string) uuid.UUID {
	t.Helper()

	taskID := uuid.New()
	tags := []string{"test", "automated"}
	tagsJSON, _ := json.Marshal(tags)

	query := `
		INSERT INTO tasks (id, app_id, title, description, status, priority, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := env.DB.Exec(query, taskID, appID, title, "Test description for "+title, status, "medium", tagsJSON)
	if err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	return taskID
}

// cleanupTestData removes test data from the database
func cleanupTestData(t *testing.T, env *TestEnvironment) {
	t.Helper()

	// Clean up in reverse order of dependencies
	queries := []string{
		"DELETE FROM task_status_history WHERE task_id IN (SELECT id FROM tasks WHERE app_id = $1)",
		"DELETE FROM tasks WHERE app_id = $1",
		"DELETE FROM unstructured_sessions WHERE app_id = $1",
		"DELETE FROM apps WHERE id = $1",
	}

	for _, query := range queries {
		_, err := env.DB.Exec(query, env.TestAppID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup test data: %v", err)
		}
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))))
}

// skipIfNoDatabase skips the test if database is not available
func skipIfNoDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	if err := db.Ping(); err != nil {
		t.Skipf("Database not available: %v", err)
	}
}

// waitForCondition polls a condition function until it returns true or timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, checkInterval time.Duration) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(checkInterval)
	}
	return false
}
