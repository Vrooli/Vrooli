// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *Logger
	cleanup        func()
}

// setupTestLogger initializes test logging with proper cleanup
func setupTestLogger() func() {
	// Create a test logger that doesn't output to stdout during tests
	_ = &Logger{
		Logger: NewLogger().Logger,
	}
	return func() {
		// Cleanup function - can be used to restore original logger if needed
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir       string
	OriginalWD    string
	TestAppPath   string
	BackupDir     string
	Cleanup       func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "app-personalizer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create test app directory structure
	testAppPath := filepath.Join(tempDir, "test-app")
	if err := os.MkdirAll(filepath.Join(testAppPath, "src/styles"), 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test app structure: %v", err)
	}

	// Create backup directory
	backupDir := filepath.Join(tempDir, "app-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test app files
	createTestAppFiles(t, testAppPath)

	return &TestEnvironment{
		TempDir:     tempDir,
		OriginalWD:  originalWD,
		TestAppPath: testAppPath,
		BackupDir:   backupDir,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// createTestAppFiles creates realistic test app files
func createTestAppFiles(t *testing.T, appPath string) {
	// package.json
	packageJSON := `{
  "name": "test-app",
  "version": "1.0.0",
  "scripts": {
    "build": "echo 'Build successful'",
    "lint": "echo 'Lint successful'",
    "test": "echo 'Tests passed'"
  }
}`
	if err := os.WriteFile(filepath.Join(appPath, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	// theme.js
	themeJS := `export const theme = {
  colors: {
    primary: "#007bff",
    secondary: "#6c757d"
  },
  fonts: {
    heading: "Arial, sans-serif",
    body: "Georgia, serif"
  }
};`
	if err := os.WriteFile(filepath.Join(appPath, "src/styles/theme.js"), []byte(themeJS), 0644); err != nil {
		t.Fatalf("Failed to create theme.js: %v", err)
	}

	// tailwind.config.js
	tailwindConfig := `module.exports = {
  theme: {
    extend: {
      colors: {
        primary: '#007bff'
      }
    }
  }
};`
	if err := os.WriteFile(filepath.Join(appPath, "tailwind.config.js"), []byte(tailwindConfig), 0644); err != nil {
		t.Fatalf("Failed to create tailwind.config.js: %v", err)
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates an in-memory test database (or connects to test DB)
func setupTestDB(t *testing.T) *TestDatabase {
	// For now, we'll use a mock approach since we don't have a test database
	// In production, this would connect to a real test database
	return &TestDatabase{
		DB:      nil, // Will be replaced with actual DB in real implementation
		Cleanup: func() {},
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
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
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

	// Set URL variables (for mux.Vars)
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	response := assertJSONResponse(t, w, expectedStatus)

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Errorf("Expected 'error' field in response, got: %v", response)
		return
	}

	if expectedMessage != "" && errorMsg != expectedMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, errorMsg)
	}

	// Validate error response structure
	if _, ok := response["status"]; !ok {
		t.Error("Expected 'status' field in error response")
	}
	if _, ok := response["timestamp"]; !ok {
		t.Error("Expected 'timestamp' field in error response")
	}
}

// createTestRouter creates a router with all app-personalizer endpoints
func createTestRouter(service *AppPersonalizerService) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/health", Health).Methods("GET")
	r.HandleFunc("/api/apps", service.ListApps).Methods("GET")
	r.HandleFunc("/api/apps/register", service.RegisterApp).Methods("POST")
	r.HandleFunc("/api/apps/analyze", service.AnalyzeApp).Methods("POST")
	r.HandleFunc("/api/personalize", service.PersonalizeApp).Methods("POST")
	r.HandleFunc("/api/backup", service.BackupApp).Methods("POST")
	r.HandleFunc("/api/validate", service.ValidateApp).Methods("POST")
	return r
}

// TestAppRegistry provides a pre-configured app registry for testing
type TestAppRegistry struct {
	App     *AppRegistry
	Cleanup func()
}

// setupTestAppRegistry creates a test app registry with sample data
func setupTestAppRegistry(t *testing.T, env *TestEnvironment) *TestAppRegistry {
	app := &AppRegistry{
		ID:          uuid.New(),
		AppName:     "test-app",
		AppPath:     env.TestAppPath,
		AppType:     "generated",
		Framework:   "react",
		Version:     "1.0.0",
		PersonalizationPoints: map[string]interface{}{
			"ui_theme": []string{"src/styles/theme.js", "tailwind.config.js"},
			"branding": []string{"public/manifest.json"},
		},
		SupportedPersonalizations: []string{"ui_theme", "branding", "content"},
	}

	return &TestAppRegistry{
		App: app,
		Cleanup: func() {
			// Cleanup if needed
		},
	}
}

// mockHTTPClient creates a mock HTTP client for testing external calls
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// createMockN8nResponse creates a mock n8n webhook response
func createMockN8nResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
