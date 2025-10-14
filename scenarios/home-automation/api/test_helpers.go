package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing with cleanup
func setupTestLogger() func() {
	// Redirect log output to discard during tests
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
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
	tempDir, err := ioutil.TempDir("", "home-automation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
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
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
		}

		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	if _, exists := req.Headers["Content-Type"]; !exists && req.Body != nil {
		req.Headers["Content-Type"] = "application/json"
	}

	// Apply headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Apply URL vars (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Apply query params
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	rr := httptest.NewRecorder()
	return rr, nil
}

// TestHelper provides common testing utilities
type TestHelper struct {
	t   *testing.T
	app *App
}

// setupTestApp creates a test app instance
func setupTestApp(t *testing.T) *App {
	app := &App{
		DB: MockDatabaseConnection(t),
	}
	// Initialize with test dependencies
	if app.DB != nil {
		app.DeviceController = NewDeviceController(app.DB)
		app.SafetyValidator = NewSafetyValidator(app.DB)
		app.CalendarScheduler = NewCalendarScheduler(app.DB)
		if app.CalendarScheduler != nil && app.DeviceController != nil {
			app.CalendarScheduler.SetDeviceController(app.DeviceController)
		}
	}
	return app
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{
		t:   t,
		app: setupTestApp(t),
	}
}

// AssertJSONResponse checks if response body is valid JSON and contains expected fields
func (h *TestHelper) AssertJSONResponse(rr *httptest.ResponseRecorder, expectedFields []string) map[string]interface{} {
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		h.t.Errorf("Response is not valid JSON: %v", err)
		return nil
	}

	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			h.t.Errorf("Response missing required field: %s", field)
		}
	}

	return response
}

// AssertStatusCode checks if response has expected status code
func (h *TestHelper) AssertStatusCode(rr *httptest.ResponseRecorder, expected int) {
	if rr.Code != expected {
		h.t.Errorf("Expected status %d, got %d. Body: %s", expected, rr.Code, rr.Body.String())
	}
}

// AssertStatusCodeInRange checks if response status is in acceptable range
func (h *TestHelper) AssertStatusCodeInRange(rr *httptest.ResponseRecorder, acceptable ...int) {
	for _, code := range acceptable {
		if rr.Code == code {
			return
		}
	}
	h.t.Errorf("Expected status in %v, got %d. Body: %s", acceptable, rr.Code, rr.Body.String())
}

// assertJSONResponse is a standalone helper for JSON response validation
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedFields []string) map[string]interface{} {
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v. Body: %s", err, rr.Body.String())
		return nil
	}

	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response missing required field: %s", field)
		}
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedCode int) {
	if rr.Code != expectedCode {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedCode, rr.Code, rr.Body.String())
	}

	// Most errors should return some response body
	if rr.Body.Len() == 0 {
		t.Error("Expected error response body, got empty")
	}
}

// MockDatabaseConnection creates a mock database connection for testing
func MockDatabaseConnection(t *testing.T) *sql.DB {
	// In production tests, use sqlmock or testcontainers
	// For now, return nil to allow structure testing
	return nil
}

// TestPattern represents a reusable test pattern
type TestPattern struct {
	Name          string
	Setup         func(t *testing.T) *App
	Execute       func(t *testing.T, app *App) *httptest.ResponseRecorder
	AssertSuccess func(t *testing.T, rr *httptest.ResponseRecorder)
	AssertFailure func(t *testing.T, rr *httptest.ResponseRecorder)
	Cleanup       func(t *testing.T, app *App)
}

// RunTestPattern executes a test pattern
func RunTestPattern(t *testing.T, pattern TestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		app := pattern.Setup(t)
		defer func() {
			if pattern.Cleanup != nil {
				pattern.Cleanup(t, app)
			}
		}()

		rr := pattern.Execute(t, app)

		if rr.Code >= 200 && rr.Code < 300 {
			if pattern.AssertSuccess != nil {
				pattern.AssertSuccess(t, rr)
			}
		} else {
			if pattern.AssertFailure != nil {
				pattern.AssertFailure(t, rr)
			}
		}
	})
}

// PerformanceTimer helps measure execution time
type PerformanceTimer struct {
	startTime time.Time
}

// StartTimer creates and starts a performance timer
func StartTimer() *PerformanceTimer {
	return &PerformanceTimer{
		startTime: time.Now(),
	}
}

// Elapsed returns the elapsed time since the timer started
func (pt *PerformanceTimer) Elapsed() time.Duration {
	return time.Since(pt.startTime)
}

// AssertMaxDuration fails the test if execution exceeded max duration
func (pt *PerformanceTimer) AssertMaxDuration(t *testing.T, max time.Duration, operation string) {
	elapsed := pt.Elapsed()
	if elapsed > max {
		t.Errorf("%s took %v, expected < %v", operation, elapsed, max)
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
