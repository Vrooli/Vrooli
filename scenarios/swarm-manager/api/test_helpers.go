// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Redirect log output to discard for cleaner test output
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	TasksDir   string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "swarm-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create tasks directory structure
	tasksDir := filepath.Join(tempDir, "tasks")
	dirs := []string{
		filepath.Join(tasksDir, "active"),
		filepath.Join(tasksDir, "backlog", "manual"),
		filepath.Join(tasksDir, "backlog", "generated"),
		filepath.Join(tasksDir, "staged"),
		filepath.Join(tasksDir, "completed"),
		filepath.Join(tasksDir, "failed"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		TasksDir:   tasksDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDB creates a test database connection or uses a mock
func setupTestDB(t *testing.T) *sql.DB {
	// Check if we have test database credentials
	dbHost := os.Getenv("TEST_POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("TEST_POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("TEST_POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("TEST_POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("TEST_POSTGRES_DB")
	if dbName == "" {
		dbName = "swarm_manager_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test requiring database: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test requiring database: %v", err)
		return nil
	}

	// Configure connection pool for tests
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(1 * time.Minute)

	return testDB
}

// TestTask provides a pre-configured task for testing
type TestTask struct {
	Task    *Task
	Path    string
	Cleanup func()
}

// setupTestTask creates a test task with sample data
func setupTestTask(t *testing.T, env *TestEnvironment, taskType, status string) *TestTask {
	task := &Task{
		ID:          fmt.Sprintf("test-task-%s", uuid.New().String()[:8]),
		Title:       "Test Task",
		Description: "This is a test task",
		Type:        taskType,
		Target:      "test-target",
		PriorityEstimates: map[string]interface{}{
			"impact":        8.0,
			"urgency":       "high",
			"success_prob":  0.75,
			"resource_cost": "moderate",
		},
		Dependencies: []string{},
		Blockers:     []string{},
		CreatedAt:    time.Now(),
		CreatedBy:    "test-user",
		Attempts:     []interface{}{},
		Notes:        "Test notes",
		Status:       status,
	}

	// Determine task directory based on status
	var taskDir string
	switch status {
	case "active":
		taskDir = filepath.Join(env.TasksDir, "active")
	case "staged":
		taskDir = filepath.Join(env.TasksDir, "staged")
	case "completed":
		taskDir = filepath.Join(env.TasksDir, "completed")
	case "failed":
		taskDir = filepath.Join(env.TasksDir, "failed")
	default:
		taskDir = filepath.Join(env.TasksDir, "backlog", "manual")
	}

	taskPath := filepath.Join(taskDir, fmt.Sprintf("%s.yaml", task.ID))

	// Save task to file
	if err := saveTaskToFile(*task, taskDir); err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	return &TestTask{
		Task: task,
		Path: taskPath,
		Cleanup: func() {
			os.Remove(taskPath)
		},
	}
}

// TestAgent provides a pre-configured agent for testing
type TestAgent struct {
	Agent   *Agent
	Cleanup func()
}

// setupTestAgent creates a test agent
func setupTestAgent(t *testing.T, name string) *TestAgent {
	agent := &Agent{
		ID:            uuid.New().String(),
		Name:          name,
		Status:        "idle",
		LastHeartbeat: time.Now(),
		ResourceUsage: make(map[string]interface{}),
	}

	return &TestAgent{
		Agent: agent,
		Cleanup: func() {
			// Cleanup agent resources if needed
		},
	}
}

// TestProblem provides a pre-configured problem for testing
type TestProblem struct {
	Problem *Problem
	Cleanup func()
}

// setupTestProblem creates a test problem
func setupTestProblem(t *testing.T, severity, status string) *TestProblem {
	now := time.Now()
	problem := &Problem{
		ID:                 fmt.Sprintf("prob-%s", uuid.New().String()[:8]),
		Title:              "Test Problem",
		Description:        "This is a test problem",
		Severity:           severity,
		Frequency:          "frequent",
		Impact:             "degraded_performance",
		Status:             status,
		DiscoveredAt:       now,
		DiscoveredBy:       "test-scanner",
		LastOccurrence:     &now,
		SourceFile:         "test-file.go",
		AffectedComponents: []string{"component-a"},
		Symptoms:           []string{"symptom-1"},
		Evidence:           make(map[string]interface{}),
		RelatedIssues:      []string{},
		PriorityEstimates: map[string]interface{}{
			"impact":        7.0,
			"urgency":       "high",
			"success_prob":  0.8,
			"resource_cost": "moderate",
		},
		TasksCreated: []string{},
	}

	return &TestProblem{
		Problem: problem,
		Cleanup: func() {
			// Cleanup problem resources if needed
		},
	}
}

// FiberTestRequest represents a Fiber test request
type FiberTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeFiberRequest creates and executes a Fiber test request
func makeFiberRequest(app *fiber.App, req FiberTestRequest) (*httptest.ResponseRecorder, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
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

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Execute request
	resp, err := app.Test(httpReq, -1)
	if err != nil {
		return nil, err
	}

	// Convert to ResponseRecorder for compatibility
	w := httptest.NewRecorder()
	w.Code = resp.StatusCode
	body, _ := ioutil.ReadAll(resp.Body)
	w.Body.Write(body)
	resp.Body.Close()

	return w, nil
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
		// Some error responses might be plain text
		if expectedErrorMessage != "" && !bytes.Contains(w.Body.Bytes(), []byte(expectedErrorMessage)) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, w.Body.String())
		}
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorMessage != "" {
		errStr, ok := errorMsg.(string)
		if !ok {
			t.Errorf("Expected error to be string, got %T", errorMsg)
			return
		}
		if !bytes.Contains([]byte(errStr), []byte(expectedErrorMessage)) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, errStr)
		}
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateTaskRequest creates a test task creation request
func (g *TestDataGenerator) CreateTaskRequest(title, taskType, target string) map[string]interface{} {
	return map[string]interface{}{
		"title":       title,
		"description": "Test task description",
		"type":        taskType,
		"target":      target,
		"priority_estimates": map[string]interface{}{
			"impact":        7.0,
			"urgency":       "medium",
			"success_prob":  0.8,
			"resource_cost": "moderate",
		},
		"created_by": "test-user",
	}
}

// UpdateTaskRequest creates a test task update request
func (g *TestDataGenerator) UpdateTaskRequest(updates map[string]interface{}) map[string]interface{} {
	return updates
}

// ProblemScanRequest creates a test problem scan request
func (g *TestDataGenerator) ProblemScanRequestData(scanPath string, force bool) ProblemScanRequest {
	return ProblemScanRequest{
		ScanPath: scanPath,
		Force:    force,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// createTestApp creates a Fiber app instance for testing
func createTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	setupRoutes(app)
	return app
}

// setTestEnvironmentVars sets required environment variables for tests
func setTestEnvironmentVars() func() {
	original := make(map[string]string)

	envVars := map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": "true",
		"API_PORT":                  "8080",
		"POSTGRES_HOST":             "localhost",
		"POSTGRES_PORT":             "5432",
		"POSTGRES_USER":             "postgres",
		"POSTGRES_PASSWORD":         "postgres",
		"POSTGRES_DB":               "swarm_manager_test",
	}

	for key, value := range envVars {
		original[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	return func() {
		for key, value := range original {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}
}
