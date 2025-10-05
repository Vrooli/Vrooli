package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Disable logging during tests unless explicitly needed
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "kids-dashboard-test")
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
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader *bytes.Buffer

	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// makeHTTPRequestComplete creates a test HTTP request and returns both recorder and request
func makeHTTPRequestComplete(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Buffer

	if req.Body != nil {
		// Handle both string and map/struct bodies
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBuffer([]byte(v))
		default:
			jsonBody, err := json.Marshal(req.Body)
			if err != nil {
				return nil, nil, err
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	} else {
		bodyReader = bytes.NewBuffer([]byte{})
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, nil, err
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	return w, httpReq, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// createTestScenarioFiles creates sample service.json files for testing
func createTestScenarioFiles(t *testing.T, baseDir string) {
	scenarios := []struct {
		name   string
		config ServiceConfig
	}{
		{
			name: "retro-game-launcher",
			config: ServiceConfig{
				Name:        "retro-game-launcher",
				Description: "Play classic arcade games!",
				Category:    []string{"games", "kid-friendly"},
				Deployment: struct {
					Port int `json:"port"`
				}{
					Port: 3301,
				},
			},
		},
		{
			name: "picker-wheel",
			config: ServiceConfig{
				Name:        "picker-wheel",
				Description: "Spin the wheel for fun choices!",
				Category:    []string{"games", "kids"},
				Deployment: struct {
					Port int `json:"port"`
				}{
					Port: 3302,
				},
			},
		},
		{
			name: "system-monitor",
			config: ServiceConfig{
				Name:        "system-monitor",
				Description: "System administration tool",
				Category:    []string{"system", "admin"},
				Deployment: struct {
					Port int `json:"port"`
				}{
					Port: 4000,
				},
			},
		},
	}

	for _, scenario := range scenarios {
		scenarioDir := filepath.Join(baseDir, "scenarios", scenario.name, ".vrooli")
		if err := os.MkdirAll(scenarioDir, 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		configPath := filepath.Join(scenarioDir, "service.json")
		configData, err := json.MarshalIndent(scenario.config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}

		if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}
	}
}
