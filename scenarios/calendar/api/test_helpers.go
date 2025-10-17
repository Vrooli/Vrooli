package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Initialize logger if not already initialized
	if appLogger == nil {
		InitLogger("calendar-test", "1.0.0-test")
	}
	return func() {
		// Reset logger if needed
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
	tempDir, err := ioutil.TempDir("", "calendar-test")
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

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	URLVars     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP request for testing
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyBytes []byte
	var err error

	if req.Body != nil {
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set URL vars if using gorilla/mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	return httptest.NewRecorder(), nil
}

// createHTTPRequestFromData creates an actual HTTP request from test data
func createHTTPRequestFromData(reqData HTTPTestRequest) (*http.Request, error) {
	var bodyBytes []byte
	var err error

	if reqData.Body != nil {
		bodyBytes, err = json.Marshal(reqData.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	httpReq, err := http.NewRequest(reqData.Method, reqData.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers
	for key, value := range reqData.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(reqData.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range reqData.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set URL vars if using gorilla/mux
	if len(reqData.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, reqData.URLVars)
	}

	return httpReq, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v. Body: %s", err, rr.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response is not valid JSON: %v", err)
	}

	if errorMsg, exists := response["error"]; exists {
		if errorStr, ok := errorMsg.(string); ok && expectedError != "" {
			if errorStr != expectedError {
				t.Errorf("Expected error '%s', got '%s'", expectedError, errorStr)
			}
		}
	} else if expectedError != "" {
		t.Error("Response should contain 'error' field")
	}
}

// TestEvent provides a pre-configured event for testing
type TestEvent struct {
	Event   *Event
	Cleanup func()
}

// setupTestEvent creates a test event with sample data
func setupTestEvent(t *testing.T, userID string, title string, startTime time.Time) *TestEvent {
	if title == "" {
		title = "Test Event"
	}
	if userID == "" {
		userID = "test-user"
	}

	event := &Event{
		UserID:           userID,
		Title:            title,
		Description:      "Test event description",
		StartTime:        startTime,
		EndTime:          startTime.Add(1 * time.Hour),
		Timezone:         "UTC",
		EventType:        "meeting",
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         make(map[string]interface{}),
		AutomationConfig: make(map[string]interface{}),
	}

	return &TestEvent{
		Event:   event,
		Cleanup: func() {},
	}
}

// Helper functions for common test data

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// boolPtr returns a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// timePtr returns a pointer to a time
func timePtr(t time.Time) *time.Time {
	return &t
}

// assertFieldExists checks if a field exists in a map
func assertFieldExists(t *testing.T, data map[string]interface{}, fieldName string) {
	if _, exists := data[fieldName]; !exists {
		t.Errorf("Response should contain '%s' field", fieldName)
	}
}

// assertFieldValue checks if a field has a specific value
func assertFieldValue(t *testing.T, data map[string]interface{}, fieldName string, expectedValue interface{}) {
	if value, exists := data[fieldName]; exists {
		if value != expectedValue {
			t.Errorf("Expected %s to be %v, got %v", fieldName, expectedValue, value)
		}
	} else {
		t.Errorf("Field '%s' does not exist in response", fieldName)
	}
}

// TestTimeRange provides common time ranges for testing
type TestTimeRange struct {
	Start time.Time
	End   time.Time
}

// commonTimeRanges returns frequently used time ranges
func commonTimeRanges() map[string]TestTimeRange {
	now := time.Now()
	return map[string]TestTimeRange{
		"past": {
			Start: now.Add(-2 * time.Hour),
			End:   now.Add(-1 * time.Hour),
		},
		"current": {
			Start: now.Add(-30 * time.Minute),
			End:   now.Add(30 * time.Minute),
		},
		"future": {
			Start: now.Add(1 * time.Hour),
			End:   now.Add(2 * time.Hour),
		},
	}
}
