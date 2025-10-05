package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing and returns cleanup function
func setupTestLogger() func() {
	// Capture original log output
	originalOutput := log.Writer()

	// Set log output to stdout for test visibility (optional: use io.Discard to silence)
	log.SetOutput(os.Stdout)
	log.SetPrefix("[test] ")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return func() {
		log.SetOutput(originalOutput)
		log.SetPrefix("")
		log.SetFlags(log.LstdFlags)
	}
}

// TestDatabase provides database mock utilities
type TestDatabase struct {
	MockDB *sql.DB
	Mock   sqlmock.Sqlmock
	cleanup func()
}

// setupTestDatabase creates a mock database for testing
func setupTestDatabase(t *testing.T) *TestDatabase {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}

	// Store original db
	originalDB := db
	db = mockDB

	return &TestDatabase{
		MockDB: mockDB,
		Mock:   mock,
		cleanup: func() {
			mockDB.Close()
			db = originalDB
		},
	}
}

// Cleanup cleans up the test database
func (td *TestDatabase) Cleanup() {
	td.cleanup()
}

// HTTPTestRequest represents an HTTP test request configuration
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLParams   map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and configures an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader io.Reader

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

	recorder := httptest.NewRecorder()
	return recorder, httpReq, nil
}

// executeGinRequest executes a Gin request with proper context setup
func executeGinRequest(req HTTPTestRequest, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	// Build request body
	var bodyReader io.Reader
	if req.Body != nil {
		var bodyBytes []byte
		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, _ = json.Marshal(v)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}
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

	ctx.Request = httpReq

	// Set URL parameters
	if req.URLParams != nil {
		var params gin.Params
		for key, value := range req.URLParams {
			params = append(params, gin.Param{Key: key, Value: value})
		}
		ctx.Params = params
	}

	// Execute handler
	handler(ctx)

	return recorder
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

	// Validate expected fields if provided
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

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
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
		t.Error("Expected 'error' field in response")
		return
	}

	errorStr, ok := errorMsg.(string)
	if !ok {
		t.Errorf("Expected error to be string, got %T", errorMsg)
		return
	}

	if expectedErrorSubstring != "" && !strings.Contains(errorStr, expectedErrorSubstring) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorStr)
	}
}

// TestDataFactory provides utilities for generating test data
type TestDataFactory struct{}

// NewTestSuite creates a test suite with default values
func (f *TestDataFactory) TestSuite(scenarioName string, suiteType string) *TestSuite {
	now := time.Now().UTC().Truncate(time.Second)
	return &TestSuite{
		ID:           uuid.New(),
		ScenarioName: scenarioName,
		SuiteType:    suiteType,
		TestCases:    []TestCase{},
		CoverageMetrics: CoverageMetrics{
			CodeCoverage:     0.0,
			BranchCoverage:   0.0,
			FunctionCoverage: 0.0,
		},
		GeneratedAt: now,
		Status:      "active",
	}
}

// NewTestCase creates a test case with default values
func (f *TestDataFactory) TestCase(suiteID uuid.UUID, name string) *TestCase {
	now := time.Now().UTC().Truncate(time.Second)
	return &TestCase{
		ID:             uuid.New(),
		SuiteID:        suiteID,
		Name:           name,
		Description:    fmt.Sprintf("Test case: %s", name),
		TestType:       "unit",
		TestCode:       "assert true",
		ExpectedResult: "pass",
		Timeout:        120,
		Dependencies:   []string{},
		Tags:           []string{},
		Priority:       "medium",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewTestExecution creates a test execution with default values
func (f *TestDataFactory) TestExecution(suiteID uuid.UUID, executionType string) *TestExecution {
	now := time.Now().UTC().Truncate(time.Second)
	return &TestExecution{
		ID:            uuid.New(),
		SuiteID:       suiteID,
		ExecutionType: executionType,
		StartTime:     now,
		Status:        "running",
		Results:       []TestResult{},
		PerformanceMetrics: PerformanceMetrics{
			ExecutionTime: 0.0,
			ResourceUsage: make(map[string]interface{}),
			ErrorCount:    0,
		},
		Environment: "test",
	}
}

// NewTestResult creates a test result with default values
func (f *TestDataFactory) TestResult(executionID uuid.UUID, testCaseID uuid.UUID, status string) *TestResult {
	now := time.Now().UTC().Truncate(time.Second)
	return &TestResult{
		ID:          uuid.New(),
		ExecutionID: executionID,
		TestCaseID:  testCaseID,
		Status:      status,
		Duration:    1.5,
		Assertions:  []AssertionResult{},
		Artifacts:   make(map[string]interface{}),
		StartedAt:   now,
		CompletedAt: now.Add(1 * time.Second),
	}
}

// Global test data factory instance
var TestData = &TestDataFactory{}

// mockRunTestSuite is a mock implementation for testing
func mockRunTestSuite(suiteID uuid.UUID, executionID uuid.UUID, executionType string, environment string, timeout int) {
	// Mock implementation - does nothing
}

// waitForCondition waits for a condition to be true with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// contextWithTimeout creates a context with timeout
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
