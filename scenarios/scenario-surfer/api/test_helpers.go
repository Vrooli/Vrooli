
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
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for testing
func setupTestLogger() func() {
	// Discard logs during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		log.SetOutput(io.Discard)
		return func() { log.SetOutput(os.Stderr) }
	}

	// Keep logging for verbose mode
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir        string
	OriginalEnv    map[string]string
	Router         *mux.Router
	Cleanup        func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Store original environment variables
	originalEnv := map[string]string{
		"VROOLI_LIFECYCLE_MANAGED": os.Getenv("VROOLI_LIFECYCLE_MANAGED"),
		"API_PORT":                 os.Getenv("API_PORT"),
	}

	// Set test environment variables
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "27999") // Use test port

	// Create test router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/scenarios/healthy", getHealthyScenariosHandler).Methods("GET")
	api.HandleFunc("/scenarios/status", getAllScenariosHandler).Methods("GET")
	api.HandleFunc("/scenarios/debug", getDebugStatusHandler).Methods("GET")
	api.HandleFunc("/issues/report", reportIssueHandler).Methods("POST")

	// CORS middleware
	router.Use(corsMiddleware)

	return &TestEnvironment{
		TempDir:     os.TempDir(),
		OriginalEnv: originalEnv,
		Router:      router,
		Cleanup: func() {
			// Restore original environment
			for key, value := range originalEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
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

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		jsonBody, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}

	// Build URL with query parameters
	url := req.Path
	if len(req.QueryParams) > 0 {
		params := []string{}
		for key, value := range req.QueryParams {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		url = fmt.Sprintf("%s?%s", url, strings.Join(params, "&"))
	}

	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %v", err)
	}

	// Set headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute request
	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	if expectedMessage != "" && !strings.Contains(body, expectedMessage) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMessage, body)
	}
}

// assertResponseField validates that a response field exists and matches expected value
func assertResponseField(t *testing.T, response map[string]interface{}, field string, expectedValue interface{}) {
	t.Helper()

	value, exists := response[field]
	if !exists {
		t.Errorf("Expected field '%s' to exist in response", field)
		return
	}

	if expectedValue != nil && value != expectedValue {
		t.Errorf("Expected field '%s' to be '%v', got '%v'", field, expectedValue, value)
	}
}

// assertResponseHasField validates that a response field exists
func assertResponseHasField(t *testing.T, response map[string]interface{}, field string) {
	t.Helper()

	if _, exists := response[field]; !exists {
		t.Errorf("Expected field '%s' to exist in response. Response: %+v", field, response)
	}
}

// createTestIssueReport creates a test issue report
func createTestIssueReport(scenario, title, description string) IssueReport {
	return IssueReport{
		Scenario:    scenario,
		Title:       title,
		Description: description,
	}
}

// mockVrooliCommand is a helper to mock vrooli command execution (for unit tests)
type MockCommandExecutor struct {
	Commands []string
	Outputs  []string
	Errors   []error
	Index    int
}

// GetNextOutput returns the next mocked output
func (m *MockCommandExecutor) GetNextOutput() (string, error) {
	if m.Index >= len(m.Outputs) {
		return "", fmt.Errorf("no more mocked outputs available")
	}

	output := m.Outputs[m.Index]
	var err error
	if m.Index < len(m.Errors) {
		err = m.Errors[m.Index]
	}

	m.Index++
	return output, err
}

// createMockScenarioResponse creates a mock vrooli scenario status response
func createMockScenarioResponse(scenarios []ScenarioInfo) string {
	response := struct {
		Success   bool           `json:"success"`
		Scenarios []ScenarioInfo `json:"scenarios"`
		Summary   struct {
			Total   int `json:"total_scenarios"`
			Running int `json:"running"`
			Stopped int `json:"stopped"`
		} `json:"summary"`
	}{
		Success:   true,
		Scenarios: scenarios,
	}

	// Calculate summary
	response.Summary.Total = len(scenarios)
	for _, s := range scenarios {
		if s.Status == "running" {
			response.Summary.Running++
		} else {
			response.Summary.Stopped++
		}
	}

	jsonBytes, _ := json.Marshal(response)
	return string(jsonBytes)
}

// createTestScenario creates a test scenario for mocking
func createTestScenario(name, status, health string, uiPort int) ScenarioInfo {
	ports := map[string]int{}
	if uiPort > 0 {
		ports["ui"] = uiPort
	}

	return ScenarioInfo{
		Name:        name,
		Status:      status,
		Health:      health,
		Ports:       ports,
		Tags:        []string{"test", "automation"},
		DisplayName: fmt.Sprintf("Test %s", name),
		Description: fmt.Sprintf("Test scenario: %s", name),
	}
}

// measureRequestDuration measures how long a request takes
func measureRequestDuration(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// assertRequestPerformance ensures request completes within expected time
func assertRequestPerformance(t *testing.T, duration time.Duration, maxDuration time.Duration, endpoint string) {
	t.Helper()

	if duration > maxDuration {
		t.Errorf("Request to %s took %v, expected less than %v", endpoint, duration, maxDuration)
	}
}
