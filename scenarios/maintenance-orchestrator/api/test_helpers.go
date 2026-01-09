package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
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
	originalLogger := logger
	logger = log.New(io.Discard, "[test] ", log.LstdFlags)
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Orchestrator *Orchestrator
	Router       *mux.Router
	Cleanup      func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Save original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create new orchestrator
	orchestrator := NewOrchestrator()

	// Setup router with all handlers
	router := mux.NewRouter()
	router.Use(corsMiddleware)

	// Health endpoint
	router.HandleFunc("/health", healthHandler(startTime)).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/scenarios", handleGetScenarios(orchestrator)).Methods("GET")
	v1.HandleFunc("/scenarios/{id}/activate", handleActivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenarios/{id}/deactivate", handleDeactivateScenario(orchestrator)).Methods("POST")
	v1.HandleFunc("/presets", handleGetPresets(orchestrator)).Methods("GET")
	v1.HandleFunc("/presets", handleCreatePreset(orchestrator)).Methods("POST")
	v1.HandleFunc("/presets/active", handleGetActivePresets(orchestrator)).Methods("GET")
	v1.HandleFunc("/presets/{id}/apply", handleApplyPreset(orchestrator)).Methods("POST")
	v1.HandleFunc("/status", handleGetStatus(orchestrator, startTime)).Methods("GET")
	v1.HandleFunc("/stop-all", handleStopAll(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenario-statuses", handleGetScenarioStatuses()).Methods("GET")
	v1.HandleFunc("/all-scenarios", handleListAllScenarios()).Methods("GET")
	v1.HandleFunc("/scenarios/{name}/add-tag", handleAddMaintenanceTag()).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/remove-tag", handleRemoveMaintenanceTag()).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/preset-assignments", handleGetScenarioPresetAssignments(orchestrator)).Methods("GET")
	v1.HandleFunc("/scenarios/{name}/preset-assignments", handleUpdateScenarioPresetAssignments(orchestrator)).Methods("POST")
	v1.HandleFunc("/scenarios/{name}/port", handleGetScenarioPort()).Methods("GET")
	v1.HandleFunc("/scenarios/{id}/start", handleStartScenario()).Methods("POST")
	v1.HandleFunc("/scenarios/{id}/stop", handleStopScenario()).Methods("POST")

	return &TestEnvironment{
		Orchestrator: orchestrator,
		Router:       router,
		Cleanup: func() {
			if err := os.Chdir(originalWD); err != nil {
				t.Logf("Warning: failed to restore working directory: %v", err)
			}
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) *httptest.ResponseRecorder {
	var body io.Reader
	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			panic(err)
		}
		body = bytes.NewBuffer(jsonBody)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, body)

	// Add headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Add query params
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set default content type
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && w.Body.Len() > 0 {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var result map[string]interface{}
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
		}
	}

	return result
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	body := w.Body.String()
	if expectedMessage != "" && body != expectedMessage+"\n" {
		t.Errorf("Expected error message '%s', got '%s'", expectedMessage, body)
	}
}

// createTestScenario creates a test scenario for use in tests
func createTestScenario(id, name, displayName string, tags []string, isActive bool) *MaintenanceScenario {
	return &MaintenanceScenario{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
		Description: "Test scenario: " + displayName,
		IsActive:    isActive,
		Tags:        tags,
		Endpoint:    "http://localhost:8080",
		Port:        8080,
	}
}

// createTestPreset creates a test preset for use in tests
func createTestPreset(id, name, description string, states map[string]bool) *Preset {
	return &Preset{
		ID:          id,
		Name:        name,
		Description: description,
		States:      states,
		IsDefault:   false,
		IsActive:    false,
	}
}

// populateTestScenarios adds common test scenarios to an orchestrator
func populateTestScenarios(orch *Orchestrator) {
	scenarios := []*MaintenanceScenario{
		createTestScenario("test-scenario-1", "test-scenario-1", "Test Scenario 1", []string{"test", "maintenance"}, false),
		createTestScenario("test-scenario-2", "test-scenario-2", "Test Scenario 2", []string{"test", "security"}, false),
		createTestScenario("test-scenario-3", "test-scenario-3", "Test Scenario 3", []string{"maintenance"}, false),
	}

	for _, scenario := range scenarios {
		orch.AddScenario(scenario)
	}
}

// populateTestPresets adds common test presets to an orchestrator
func populateTestPresets(orch *Orchestrator) {
	presets := []*Preset{
		createTestPreset("preset-1", "Test Preset 1", "First test preset", map[string]bool{
			"test-scenario-1": true,
			"test-scenario-2": true,
		}),
		createTestPreset("preset-2", "Test Preset 2", "Second test preset", map[string]bool{
			"test-scenario-3": true,
		}),
	}

	for _, preset := range presets {
		orch.presets[preset.ID] = preset
	}
}
