package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
	Readiness bool      `json:"readiness"`
	Version   string    `json:"version"`
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Redirect logs to discard during tests unless verbose mode is needed
	originalOutput := log.Writer()
	log.SetOutput(ioutil.Discard)

	// Initialize slog logger for tests (redirected to discard)
	originalLogger := logger
	logger = slog.New(slog.NewJSONHandler(ioutil.Discard, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return func() {
		log.SetOutput(originalOutput)
		logger = originalLogger
	}
}

// HTTPTestRequest encapsulates HTTP test request data
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates an HTTP request and recorder for testing
func makeHTTPRequest(req HTTPTestRequest) (*http.Request, *httptest.ResponseRecorder, error) {
	var bodyBytes []byte
	var err error

	if req.Body != nil {
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, nil, err
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, err
	}

	// Set default content type for JSON requests
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Apply custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Apply URL variables (for mux path variables)
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	recorder := httptest.NewRecorder()
	return httpReq, recorder, nil
}

// serveTestRequest sends a typed test request through the real API router.
// Case inputs stay at each caller; only encoding, recorder and router wiring
// are shared.
func serveTestRequest(t *testing.T, req HTTPTestRequest) *httptest.ResponseRecorder {
	t.Helper()
	httpReq, response, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("prepare API test request: %v", err)
	}
	setupTestRouter().ServeHTTP(response, httpReq)
	return response
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if target != nil {
		if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
			t.Errorf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedContains string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	if expectedContains != "" && !strings.Contains(w.Body.String(), expectedContains) {
		t.Errorf("Expected response body to contain %q, got %q", expectedContains, w.Body.String())
	}
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter() *mux.Router {
	return newRouter(db)
}

// initTestWheels initializes wheels for testing
func initTestWheels() {
	memoryMu.Lock()
	defer memoryMu.Unlock()
	wheels = getDefaultWheels()
	history = []SpinResult{}
}

// resetTestState resets global state between tests
func resetTestState() {
	db = nil
	memoryMu.Lock()
	defer memoryMu.Unlock()
	wheels = []Wheel{}
	history = []SpinResult{}
}
