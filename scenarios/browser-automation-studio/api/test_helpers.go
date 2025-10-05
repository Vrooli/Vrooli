// +build testing

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// setupTestLogger initializes the logger for testing
func setupTestLogger() func() {
	// Create a test logger that doesn't output
	testLog := logrus.New()
	testLog.SetOutput(ioutil.Discard)
	testLog.SetLevel(logrus.PanicLevel)

	return func() {
		// Cleanup function - restore default logger if needed
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "browser-automation-studio-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) (*database.DB, func()) {
	// Use environment variable for test database URL
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}

	if dbURL == "" {
		t.Skip("No database URL configured - skipping database tests")
	}

	log := logrus.New()
	log.SetOutput(ioutil.Discard)

	db, err := database.NewConnection(log)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
	}

	return db, cleanup
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

// makeHTTPRequest creates an HTTP request for testing
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader *bytes.Buffer
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	recorder := httptest.NewRecorder()
	return recorder, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessageContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var errorResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
		t.Errorf("Failed to decode error response: %v. Body: %s", err, w.Body.String())
		return
	}

	if expectedMessageContains != "" {
		errorMsg, ok := errorResp["error"].(string)
		if !ok {
			t.Errorf("Expected error message in response, got: %v", errorResp)
			return
		}

		if !containsString(errorMsg, expectedMessageContains) {
			t.Errorf("Expected error message to contain %q, got %q", expectedMessageContains, errorMsg)
		}
	}
}

// containsString checks if a string contains a substring (case-insensitive)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		bytes.Contains([]byte(s), []byte(substr)))
}

// createTestProject creates a project for testing
func createTestProject(t *testing.T, repo database.Repository, name string) *database.Project {
	t.Helper()

	project := &database.Project{
		ID:          uuid.New(),
		Name:        name,
		Description: fmt.Sprintf("Test project: %s", name),
		FolderPath:  fmt.Sprintf("/test/%s", name),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	ctx := context.Background()
	if err := repo.CreateProject(ctx, project); err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	return project
}

// createTestWorkflow creates a workflow for testing
func createTestWorkflow(t *testing.T, repo database.Repository, projectID *uuid.UUID, name string) *database.Workflow {
	t.Helper()

	workflow := &database.Workflow{
		ID:         uuid.New(),
		ProjectID:  projectID,
		Name:       name,
		FolderPath: "/test",
		FlowDefinition: database.JSONMap{
			"nodes": []interface{}{},
			"edges": []interface{}{},
		},
		Tags:      []string{},
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ctx := context.Background()
	if err := repo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatalf("Failed to create test workflow: %v", err)
	}

	return workflow
}

// createTestExecution creates an execution for testing
func createTestExecution(t *testing.T, repo database.Repository, workflowID uuid.UUID) *database.Execution {
	t.Helper()

	execution := &database.Execution{
		ID:         uuid.New(),
		WorkflowID: workflowID,
		Status:     "pending",
		StartedAt:  time.Now(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	ctx := context.Background()
	if err := repo.CreateExecution(ctx, execution); err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}

	return execution
}

// cleanupTestData cleans up test data from the database
func cleanupTestData(t *testing.T, db *database.DB) {
	t.Helper()

	// Delete test data in reverse dependency order
	queries := []string{
		"DELETE FROM execution_logs WHERE execution_id IN (SELECT id FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%'))",
		"DELETE FROM screenshots WHERE execution_id IN (SELECT id FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%'))",
		"DELETE FROM extracted_data WHERE execution_id IN (SELECT id FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%'))",
		"DELETE FROM executions WHERE workflow_id IN (SELECT id FROM workflows WHERE folder_path LIKE '/test%')",
		"DELETE FROM workflows WHERE folder_path LIKE '/test%'",
		"DELETE FROM projects WHERE folder_path LIKE '/test%'",
	}

	ctx := context.Background()
	for _, query := range queries {
		if _, err := db.ExecContext(ctx, query); err != nil {
			t.Logf("Warning: Failed to cleanup test data: %v", err)
		}
	}
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}
