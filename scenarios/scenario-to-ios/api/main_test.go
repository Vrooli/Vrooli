package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	s := &Server{
		router: http.NewServeMux(),
		logger: nil,
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}

	if response["service"] != "scenario-to-ios" {
		t.Errorf("Expected service 'scenario-to-ios', got '%s'", response["service"])
	}
}

func TestHandleBuildValidation(t *testing.T) {
	// Use a discarding logger for tests to avoid cluttering output
	s := &Server{
		router: http.NewServeMux(),
		logger: log.New(io.Discard, "", log.LstdFlags),
	}

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Wrong method GET",
			method:         "GET",
			body:           `{}`,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "Only POST method allowed",
		},
		{
			name:           "Empty scenario name",
			method:         "POST",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "scenario_name is required",
		},
		{
			name:           "Path traversal attack",
			method:         "POST",
			body:           `{"scenario_name": "../../../etc/passwd"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid scenario_name format",
		},
		{
			name:           "Path with dots",
			method:         "POST",
			body:           `{"scenario_name": "test..scenario"}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid scenario_name format",
		},
		{
			name:           "Invalid JSON",
			method:         "POST",
			body:           `{invalid}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid JSON",
		},
		{
			name:           "Nonexistent scenario",
			method:         "POST",
			body:           `{"scenario_name": "nonexistent-scenario-12345"}`,
			expectedStatus: http.StatusNotFound,
			expectedError:  "Scenario not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/ios/build", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			s.handleBuild(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response BuildResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Success {
				t.Error("Expected success=false for error case")
			}

			if !strings.Contains(response.Error, tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, response.Error)
			}
		})
	}
}

func TestHandleBuildSuccess(t *testing.T) {
	// Use a discarding logger for tests to avoid cluttering output
	s := &Server{
		router: http.NewServeMux(),
		logger: log.New(io.Discard, "", log.LstdFlags),
	}

	// Use simple-test which we know exists
	body := `{"scenario_name": "simple-test"}`
	req := httptest.NewRequest("POST", "/api/v1/ios/build", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleBuild(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response BuildResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false with error: %s", response.Error)
	}

	if response.BuildID == "" {
		t.Error("Expected build_id to be set")
	}

	if !strings.HasPrefix(response.BuildID, "ios-simple-test-") {
		t.Errorf("Expected build_id to start with 'ios-simple-test-', got '%s'", response.BuildID)
	}

	if response.Metadata["scenario_name"] != "simple-test" {
		t.Errorf("Expected scenario_name 'simple-test', got '%s'", response.Metadata["scenario_name"])
	}

	// Verify template expansion happened
	if response.Metadata["status"] != "template_expanded" {
		t.Errorf("Expected status 'template_expanded', got '%s'", response.Metadata["status"])
	}

	// Verify project directory exists
	projectPath := response.Metadata["project_path"]
	if projectPath == "" {
		t.Error("Expected project_path in metadata")
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Errorf("Expected project directory to exist at %s", projectPath)
	}

	// Clean up
	buildDir := response.Metadata["build_directory"]
	if buildDir != "" {
		os.RemoveAll(buildDir)
	}
}

func TestTemplateExpander(t *testing.T) {
	expander := NewTemplateExpander("test-scenario")

	if expander.scenarioName != "test-scenario" {
		t.Errorf("Expected scenarioName 'test-scenario', got '%s'", expander.scenarioName)
	}

	if expander.appName != "Test Scenario" {
		t.Errorf("Expected appName 'Test Scenario', got '%s'", expander.appName)
	}

	if expander.bundleID != "com.vrooli.test-scenario" {
		t.Errorf("Expected bundleID 'com.vrooli.test-scenario', got '%s'", expander.bundleID)
	}

	if expander.version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", expander.version)
	}
}

func TestFormatAppName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test-scenario", "Test Scenario"},
		{"simple-test", "Simple Test"},
		{"my-awesome-app", "My Awesome App"},
		{"singleword", "Singleword"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatAppName(tt.input)
			if result != tt.expected {
				t.Errorf("formatAppName(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatBundleID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test-scenario", "com.vrooli.test-scenario"},
		{"my-app", "com.vrooli.my-app"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatBundleID(tt.input)
			if result != tt.expected {
				t.Errorf("formatBundleID(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReplacePlaceholders(t *testing.T) {
	expander := NewTemplateExpander("test-app")

	content := `
App Name: {{APP_NAME}}
Bundle ID: {{BUNDLE_ID}}
Scenario: {{SCENARIO_NAME}}
Version: {{VERSION}}
Build: {{BUILD_NUMBER}}
`

	result := expander.replacePlaceholders(content)

	if !strings.Contains(result, "App Name: Test App") {
		t.Error("Expected APP_NAME to be replaced")
	}

	if !strings.Contains(result, "Bundle ID: com.vrooli.test-app") {
		t.Error("Expected BUNDLE_ID to be replaced")
	}

	if !strings.Contains(result, "Scenario: test-app") {
		t.Error("Expected SCENARIO_NAME to be replaced")
	}

	if !strings.Contains(result, "Version: 1.0.0") {
		t.Error("Expected VERSION to be replaced")
	}

	if !strings.Contains(result, "Build: 1") {
		t.Error("Expected BUILD_NUMBER to be replaced")
	}
}
