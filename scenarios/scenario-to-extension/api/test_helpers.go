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
	// Suppress logs during tests unless DEBUG is set
	if os.Getenv("DEBUG") != "true" {
		log.SetOutput(ioutil.Discard)
	}
	return func() {
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Config     *Config
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "scenario-to-extension-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create test subdirectories
	templatesDir := filepath.Join(tempDir, "templates", "vanilla")
	dataDir := filepath.Join(tempDir, "data", "extensions")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create templates dir: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create data dir: %v", err)
	}

	// Create test template files
	manifestContent := `{
  "manifest_version": 3,
  "name": "{{APP_NAME}}",
  "description": "{{APP_DESCRIPTION}}",
  "version": "{{VERSION}}",
  "permissions": {{PERMISSIONS}},
  "host_permissions": {{HOST_PERMISSIONS}}
}`
	if err := ioutil.WriteFile(filepath.Join(templatesDir, "manifest.json"), []byte(manifestContent), 0644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create manifest template: %v", err)
	}

	// Create test config
	testConfig := &Config{
		Port:           3201,
		APIEndpoint:    "http://localhost:3201",
		TemplatesPath:  filepath.Join(tempDir, "templates"),
		OutputPath:     dataDir,
		BrowserlessURL: "http://localhost:3000",
		Debug:          false,
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Config:     testConfig,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// setupTestConfig initializes global config for testing
func setupTestConfig(env *TestEnvironment) func() {
	originalConfig := config
	config = env.Config
	builds = make(map[string]*ExtensionBuild)
	return func() {
		config = originalConfig
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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
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
	return w, nil
}

// executeRequest executes an HTTP test request against a handler
func executeRequest(handler http.HandlerFunc, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	w, err := makeHTTPRequest(req)
	if err != nil {
		return nil, err
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, nil)
	if req.Body != nil {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		httpReq = httptest.NewRequest(req.Method, req.Path, bytes.NewReader(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	handler(w, httpReq)
	return w, nil
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// assertTextResponse validates plain text error responses
func assertTextResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) string {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}
	return w.Body.String()
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// GenerateExtensionRequest creates a test extension generation request
func (g *TestDataGenerator) GenerateExtensionRequest(scenarioName, appName, apiEndpoint string) ExtensionGenerateRequest {
	return ExtensionGenerateRequest{
		ScenarioName: scenarioName,
		TemplateType: "full",
		Config: ExtensionConfig{
			AppName:         appName,
			Description:     "Test extension description",
			APIEndpoint:     apiEndpoint,
			Version:         "1.0.0",
			AuthorName:      "Test Author",
			License:         "MIT",
			Permissions:     []string{"storage", "activeTab"},
			HostPermissions: []string{"<all_urls>"},
		},
	}
}

// GenerateTestRequest creates a test extension test request
func (g *TestDataGenerator) GenerateTestRequest(extensionPath string, sites []string) ExtensionTestRequest {
	if sites == nil {
		sites = []string{"https://example.com"}
	}
	return ExtensionTestRequest{
		ExtensionPath: extensionPath,
		TestSites:     sites,
		Screenshot:    true,
		Headless:      true,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
