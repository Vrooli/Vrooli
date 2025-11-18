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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes a test logger that suppresses verbose output
func setupTestLogger() func() {
	// Redirect log output to null during tests unless verbose flag is set
	if !testing.Verbose() {
		log.SetOutput(ioutil.Discard)
		return func() { log.SetOutput(os.Stderr) }
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Server     *Server
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "scenario-to-desktop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create template directory structure
	templateDir := filepath.Join(tempDir, "templates")
	if err := os.MkdirAll(filepath.Join(templateDir, "vanilla"), 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create template dirs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(templateDir, "advanced"), 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create advanced template dir: %v", err)
	}

	// Create sample template files
	if err := createSampleTemplates(templateDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create sample templates: %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create a test server
	server := NewServer(0) // Use port 0 for testing
	server.templateDir = templateDir

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Server:     server,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// createSampleTemplates creates sample template files for testing
func createSampleTemplates(templateDir string) error {
	// Create basic-app.json
	basicTemplate := map[string]interface{}{
		"name":        "basic-app",
		"description": "Basic desktop application template",
		"framework":   "electron",
		"type":        "basic",
		"features":    []string{"native-menus", "auto-updater"},
	}
	basicJSON, _ := json.MarshalIndent(basicTemplate, "", "  ")
	if err := ioutil.WriteFile(filepath.Join(templateDir, "advanced", "basic.json"), basicJSON, 0644); err != nil {
		return err
	}

	// Create advanced-app.json
	advancedTemplate := map[string]interface{}{
		"name":        "advanced-app",
		"description": "Advanced desktop application template",
		"framework":   "electron",
		"type":        "advanced",
		"features":    []string{"system-tray", "notifications", "global-shortcuts"},
	}
	advancedJSON, _ := json.MarshalIndent(advancedTemplate, "", "  ")
	if err := ioutil.WriteFile(filepath.Join(templateDir, "advanced", "advanced.json"), advancedJSON, 0644); err != nil {
		return err
	}

	// Create multi-window template
	multiWindowTemplate := map[string]interface{}{
		"name":        "multi-window-app",
		"description": "Multi-window desktop application template",
		"framework":   "electron",
		"type":        "multi_window",
		"features":    []string{"window-management", "inter-window-communication"},
	}
	multiWindowJSON, _ := json.MarshalIndent(multiWindowTemplate, "", "  ")
	if err := ioutil.WriteFile(filepath.Join(templateDir, "advanced", "multi_window.json"), multiWindowJSON, 0644); err != nil {
		return err
	}

	return nil
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
	var bodyReader *bytes.Buffer
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = &bytes.Buffer{}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default content type
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add URL variables to request context
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status code %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, shouldContainMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status code %d, got %d", expectedStatus, w.Code)
	}

	body := w.Body.String()
	if shouldContainMessage != "" && !strings.Contains(body, shouldContainMessage) {
		t.Errorf("Expected error message to contain '%s', got: %s", shouldContainMessage, body)
	}
}

// assertFieldExists checks if a field exists in the response
func assertFieldExists(t *testing.T, response map[string]interface{}, fieldName string) interface{} {
	t.Helper()

	value, exists := response[fieldName]
	if !exists {
		t.Errorf("Expected field '%s' to exist in response", fieldName)
		return nil
	}
	return value
}

// assertFieldValue checks if a field has the expected value
func assertFieldValue(t *testing.T, response map[string]interface{}, fieldName string, expected interface{}) {
	t.Helper()

	actual := assertFieldExists(t, response, fieldName)
	if actual != expected {
		t.Errorf("Expected field '%s' to have value '%v', got '%v'", fieldName, expected, actual)
	}
}

// TestDesktopConfig provides a valid desktop configuration for testing
type TestDesktopConfig struct {
	Config  *DesktopConfig
	Cleanup func()
}

// setupTestDesktopConfig creates a valid desktop configuration for testing
func setupTestDesktopConfig(t *testing.T, appName string) *TestDesktopConfig {
	tempDir, err := ioutil.TempDir("", "desktop-output-*")
	if err != nil {
		t.Fatalf("Failed to create temp output dir: %v", err)
	}

	config := &DesktopConfig{
		AppName:          appName,
		AppDisplayName:   appName + " Display",
		AppDescription:   "Test desktop application",
		Version:          "1.0.0",
		Author:           "Test Author",
		License:          "MIT",
		AppID:            "com.test." + strings.ToLower(strings.ReplaceAll(appName, " ", "")),
		AppURL:           "https://example.com",
		ServerType:       "node",
		ServerPort:       3000,
		ServerPath:       "./server",
		APIEndpoint:      "http://localhost:3000",
		ScenarioPath:     "./dist",
		ScenarioName:     appName,
		AutoManageVrooli: false,
		VrooliBinaryPath: "vrooli",
		DeploymentMode:   "external-server",
		Framework:        "electron",
		TemplateType:     "basic",
		Features:         map[string]interface{}{},
		Window:           map[string]interface{}{},
		Platforms:        []string{"win", "mac", "linux"},
		OutputPath:       tempDir,
		Styling:          map[string]interface{}{},
	}

	return &TestDesktopConfig{
		Config: config,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// createTestBuildStatus creates a test build status
func createTestBuildStatus(buildID, status string) *BuildStatus {
	return &BuildStatus{
		BuildID:      buildID,
		ScenarioName: "test-scenario",
		Status:       status,
		Framework:    "electron",
		TemplateType: "basic",
		Platforms:    []string{"win", "mac", "linux"},
		OutputPath:   "/tmp/test-output",
		CreatedAt:    time.Now(),
		BuildLog:     []string{},
		ErrorLog:     []string{},
		Artifacts:    make(map[string]string),
		Metadata:     make(map[string]interface{}),
	}
}

// waitForBuildStatus waits for a build to reach a specific status
func waitForBuildStatus(server *Server, buildID string, expectedStatus string, timeout time.Duration) (*BuildStatus, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if status, exists := server.buildStatuses[buildID]; exists {
			if status.Status == expectedStatus {
				return status, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for build status '%s'", expectedStatus)
}

// assertBuildStatusExists checks that a build status exists
func assertBuildStatusExists(t *testing.T, server *Server, buildID string) *BuildStatus {
	t.Helper()

	status, exists := server.buildStatuses[buildID]
	if !exists {
		t.Fatalf("Expected build status for ID %s to exist", buildID)
	}
	return status
}

// assertBuildStatusValue checks a field in build status
func assertBuildStatusValue(t *testing.T, status *BuildStatus, field string, expected interface{}) {
	t.Helper()

	var actual interface{}
	switch field {
	case "status":
		actual = status.Status
	case "framework":
		actual = status.Framework
	case "template_type":
		actual = status.TemplateType
	default:
		t.Fatalf("Unknown build status field: %s", field)
		return
	}

	if actual != expected {
		t.Errorf("Expected build status field '%s' to be '%v', got '%v'", field, expected, actual)
	}
}

// createValidGenerateRequest creates a valid request for desktop generation
func createValidGenerateRequest() map[string]interface{} {
	return map[string]interface{}{
		"app_name":         "TestApp",
		"app_display_name": "Test Application",
		"app_description":  "A test desktop application",
		"version":          "1.0.0",
		"author":           "Test Author",
		"license":          "MIT",
		"app_id":           "com.test.app",
		"app_url":          "https://example.com",
		"server_type":      "node",
		"server_port":      3000,
		"server_path":      "./server",
		"api_endpoint":     "http://localhost:3000",
		"scenario_path":    "./dist",
		"framework":        "electron",
		"template_type":    "basic",
		"platforms":        []string{"win", "mac", "linux"},
		"output_path":      "/tmp/test-output",
		"features":         map[string]interface{}{},
		"window":           map[string]interface{}{},
		"styling":          map[string]interface{}{},
	}
}

// createJSONRequest creates an HTTP request with JSON body
func createJSONRequest(method, path string, body interface{}) *http.Request {
	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	return req
}
