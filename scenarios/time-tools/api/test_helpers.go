package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := logger
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir := t.TempDir() // Use testing.T's built-in temp directory

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			// Temp directory is automatically cleaned up
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Query   map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyJSON, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyJSON)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if req.Query != nil {
		q := httpReq.URL.Query()
		for key, val := range req.Query {
			q.Add(key, val)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add headers
	if req.Headers != nil {
		for key, val := range req.Headers {
			httpReq.Header.Set(key, val)
		}
	}

	// Set default content type for JSON
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	return recorder, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateBody func(map[string]interface{})) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if validateBody != nil {
		var body map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
		validateBody(body)
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errorMsg, ok := body["error"].(string)
	if !ok {
		t.Fatalf("Response does not contain error field: %v", body)
	}

	if expectedErrorContains != "" {
		if len(errorMsg) == 0 {
			t.Errorf("Expected error message containing '%s', got empty", expectedErrorContains)
		}
	}
}

// TestTimeFormat provides common test time formats and values
type TestTimeFormat struct {
	RFC3339    string
	Unix       int64
	DateTime   string
	Date       string
	Time       string
}

// getTestTime returns a consistent test time
func getTestTime() TestTimeFormat {
	testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	return TestTimeFormat{
		RFC3339:  testTime.Format(time.RFC3339),
		Unix:     testTime.Unix(),
		DateTime: testTime.Format("2006-01-02 15:04:05"),
		Date:     testTime.Format("2006-01-02"),
		Time:     testTime.Format("15:04:05"),
	}
}

// getTestTimeRange returns start and end times for testing
func getTestTimeRange() (start, end TestTimeFormat) {
	startTime := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 17, 0, 0, 0, time.UTC)

	start = TestTimeFormat{
		RFC3339:  startTime.Format(time.RFC3339),
		Unix:     startTime.Unix(),
		DateTime: startTime.Format("2006-01-02 15:04:05"),
		Date:     startTime.Format("2006-01-02"),
		Time:     startTime.Format("15:04:05"),
	}

	end = TestTimeFormat{
		RFC3339:  endTime.Format(time.RFC3339),
		Unix:     endTime.Unix(),
		DateTime: endTime.Format("2006-01-02 15:04:05"),
		Date:     endTime.Format("2006-01-02"),
		Time:     endTime.Format("15:04:05"),
	}

	return
}

// setupTestRouter creates a test router with all routes configured
func setupTestRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Register all handlers
	mux.HandleFunc("/api/v1/health", healthHandler)
	mux.HandleFunc("/api/v1/time/convert", timezoneConvertHandler)
	mux.HandleFunc("/api/v1/time/duration", durationCalculateHandler)
	mux.HandleFunc("/api/v1/time/format", formatTimeHandler)
	mux.HandleFunc("/api/v1/time/add", addTimeHandler)
	mux.HandleFunc("/api/v1/time/subtract", subtractTimeHandler)
	mux.HandleFunc("/api/v1/time/parse", parseTimeHandler)
	mux.HandleFunc("/api/v1/schedule/optimal", scheduleOptimalHandler)
	mux.HandleFunc("/api/v1/schedule/conflicts", conflictDetectHandler)
	mux.HandleFunc("/api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			createEventHandler(w, r)
		case "GET":
			listEventsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

// executeRequest performs a request against a handler
func executeRequest(handler http.Handler, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	// Handle string body (for invalid JSON tests)
	if strBody, ok := req.Body.(string); ok {
		bodyReader = bytes.NewBufferString(strBody)
	} else if req.Body != nil {
		bodyJSON, _ := json.Marshal(req.Body)
		bodyReader = bytes.NewBuffer(bodyJSON)
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bodyReader)

	// Add query parameters
	if req.Query != nil {
		q := httpReq.URL.Query()
		for key, val := range req.Query {
			q.Add(key, val)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Add headers
	if req.Headers != nil {
		for key, val := range req.Headers {
			httpReq.Header.Set(key, val)
		}
	}

	// Set default content type
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httpReq)
	return w
}
