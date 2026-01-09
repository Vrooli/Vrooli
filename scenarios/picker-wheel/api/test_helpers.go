package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
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

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "picker-wheel-test")
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

	if expectedContains != "" {
		body := w.Body.String()
		if body == "" || (expectedContains != "" && len(body) == 0) {
			t.Errorf("Expected response body to contain '%s', got empty body", expectedContains)
		}
	}
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter() *mux.Router {
	router := mux.NewRouter()

	// Register all handlers
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/wheels", getWheelsHandler).Methods("GET")
	router.HandleFunc("/api/wheels", createWheelHandler).Methods("POST")
	router.HandleFunc("/api/wheels/{id}", getWheelHandler).Methods("GET")
	router.HandleFunc("/api/wheels/{id}", deleteWheelHandler).Methods("DELETE")
	router.HandleFunc("/api/history", getHistoryHandler).Methods("GET")
	router.HandleFunc("/api/spin", spinHandler).Methods("POST")

	return router
}

// initTestWheels initializes wheels for testing
func initTestWheels() {
	wheels = getDefaultWheels()
	history = []SpinResult{}
}

// resetTestState resets global state between tests
func resetTestState() {
	wheels = []Wheel{}
	history = []SpinResult{}
	db = nil
}
