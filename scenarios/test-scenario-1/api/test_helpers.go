package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := logger
	// Silence logs during tests unless verbose mode is enabled
	if os.Getenv("TEST_VERBOSE") != "true" {
		logger = log.New(ioutil.Discard, "[test] ", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	}
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Store      *TaskStore
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "test-scenario-1-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create a fresh task store for this test
	originalStore := store
	store = &TaskStore{
		tasks: make(map[uuid.UUID]*Task),
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Store:      store,
		Cleanup: func() {
			store = originalStore
			os.RemoveAll(tempDir)
		},
	}
}

// TestTask provides a pre-configured task for testing
type TestTask struct {
	Task    *Task
	Cleanup func()
}

// setupTestTask creates a test task with sample data
func setupTestTask(t *testing.T, title string) *TestTask {
	task := &Task{
		ID:          uuid.New(),
		Title:       title,
		Description: fmt.Sprintf("Test task: %s", title),
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"test": true},
	}

	if err := store.Create(task); err != nil {
		t.Fatalf("Failed to create test task: %v", err)
	}

	return &TestTask{
		Task: task,
		Cleanup: func() {
			store.Delete(task.ID)
		},
	}
}

// setupMultipleTestTasks creates multiple test tasks
func setupMultipleTestTasks(t *testing.T, count int) []*TestTask {
	tasks := make([]*TestTask, count)
	for i := 0; i < count; i++ {
		tasks[i] = setupTestTask(t, fmt.Sprintf("Task %d", i+1))
	}
	return tasks
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
		// If not JSON, check if the response body contains the error message
		if expectedErrorMessage != "" && !strings.Contains(w.Body.String(), expectedErrorMessage) {
			t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, w.Body.String())
		}
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

// assertTextResponse validates plain text responses
func assertTextResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedContent string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if expectedContent != "" && !strings.Contains(w.Body.String(), expectedContent) {
		t.Errorf("Expected response to contain '%s', got '%s'", expectedContent, w.Body.String())
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

// CreateTaskRequest creates a test task creation request
func (g *TestDataGenerator) CreateTaskRequest(title, description string) CreateTaskRequest {
	return CreateTaskRequest{
		Title:       title,
		Description: description,
		Metadata:    map[string]interface{}{"test": true},
	}
}

// UpdateTaskRequest creates a test task update request
func (g *TestDataGenerator) UpdateTaskRequest(title, description, status *string) UpdateTaskRequest {
	return UpdateTaskRequest{
		Title:       title,
		Description: description,
		Status:      status,
		Metadata:    map[string]interface{}{"updated": true},
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// Common test scenarios
type TestScenarios struct{}

// TaskNotFound tests handlers with non-existent task IDs
func (s *TestScenarios) TaskNotFound(t *testing.T, handler http.HandlerFunc, method, path string) {
	nonExistentID := uuid.New()
	req := HTTPTestRequest{
		Method:  method,
		Path:    strings.Replace(path, "{id}", nonExistentID.String(), 1),
		URLVars: map[string]string{"id": nonExistentID.String()},
	}

	// For PUT/POST methods, include valid JSON body
	if method == "PUT" || method == "POST" {
		req.Body = map[string]interface{}{"title": "Test"}
	}

	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	handler(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent task, got %d", w.Code)
	}
}

// InvalidUUID tests handlers with invalid UUID formats
func (s *TestScenarios) InvalidUUID(t *testing.T, handler http.HandlerFunc, method, path string) {
	req := HTTPTestRequest{
		Method:  method,
		Path:    strings.Replace(path, "{id}", "invalid-uuid", 1),
		URLVars: map[string]string{"id": "invalid-uuid"},
	}

	w, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	handler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}
}

// Global test scenarios instance
var Scenarios = &TestScenarios{}
