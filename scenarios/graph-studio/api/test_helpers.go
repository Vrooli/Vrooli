//go:build testing
// +build testing

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

	"github.com/google/uuid"
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
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Router     *mux.Router
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Initialize validator
	validator = NewGraphValidator()

	// Setup router
	router := mux.NewRouter()

	return &TestEnvironment{
		Router: router,
		Cleanup: func() {
			// Cleanup resources
		},
	}
}

// TestGraph provides a pre-configured graph for testing
type TestGraph struct {
	Graph   *Graph
	Cleanup func()
}

// setupTestGraph creates a test graph with sample data
func setupTestGraph(t *testing.T, graphType string, userID string) *TestGraph {
	t.Helper()

	if userID == "" {
		userID = uuid.New().String()
	}

	description := fmt.Sprintf("Test graph: %s", graphType)

	var graphData map[string]interface{}
	switch graphType {
	case "mind-maps":
		graphData = map[string]interface{}{
			"root": map[string]interface{}{
				"id":    "root",
				"label": "Root Node",
				"children": []interface{}{
					map[string]interface{}{
						"id":    "child1",
						"label": "Child 1",
					},
				},
			},
		}
	case "bpmn":
		graphData = map[string]interface{}{
			"processes": []interface{}{
				map[string]interface{}{
					"id":   "process1",
					"name": "Test Process",
					"tasks": []interface{}{
						map[string]interface{}{
							"id":   "task1",
							"name": "Task 1",
							"type": "userTask",
						},
					},
				},
			},
		}
	case "network-graphs":
		graphData = map[string]interface{}{
			"nodes": []interface{}{
				map[string]interface{}{
					"id":    "node1",
					"label": "Node 1",
				},
			},
			"edges": []interface{}{},
		}
	case "mermaid":
		graphData = map[string]interface{}{
			"diagram": "graph TD\n    A --> B",
		}
	default:
		graphData = map[string]interface{}{}
	}

	graph := &Graph{
		ID:          uuid.New(),
		Name:        fmt.Sprintf("Test %s Graph", graphType),
		Type:        graphType,
		Description: &description,
		Data:        graphData,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		Tags:        []string{"test"},
		Permissions: GraphPermissions{
			Public:       false,
			AllowedUsers: []string{userID},
			Editors:      []string{userID},
		},
	}

	return &TestGraph{
		Graph: graph,
		Cleanup: func() {
			// Cleanup if needed
		},
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		if str, ok := req.Body.(string); ok {
			bodyReader = bytes.NewBufferString(str)
		} else {
			jsonData, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %v", err)
			}
			bodyReader = bytes.NewBuffer(jsonData)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default Content-Type for requests with body
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add URL variables to context if needed
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if target != nil && w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), target); err != nil {
			t.Errorf("Failed to unmarshal response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
		return
	}

	var errorResp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
		t.Errorf("Failed to unmarshal error response: %v", err)
		return
	}

	if msg, ok := errorResp["error"].(string); ok {
		if expectedMessage != "" && msg != expectedMessage {
			t.Errorf("Expected error message '%s', got '%s'", expectedMessage, msg)
		}
	} else {
		t.Error("Response does not contain 'error' field")
	}
}

// assertResponseContains checks if response body contains expected string
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Response body is empty")
		return
	}

	// For JSON responses, we can check the content
	var jsonData interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &jsonData); err == nil {
		// It's valid JSON, convert back to string for comparison
		jsonStr, _ := json.Marshal(jsonData)
		body = string(jsonStr)
	}

	if expected != "" && !contains(body, expected) {
		t.Errorf("Expected response to contain '%s', got: %s", expected, body)
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

// findSubstring is a helper for contains
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// createTestRouter creates a router with test handlers
func createTestRouter() *mux.Router {
	router := mux.NewRouter()

	// Health endpoints
	router.HandleFunc("/health", HealthHandler).Methods("GET")
	router.HandleFunc("/api/v1/health/detailed", DetailedHealthHandler).Methods("GET")

	// Stats endpoints
	router.HandleFunc("/api/v1/stats", DashboardStatsHandler).Methods("GET")
	router.HandleFunc("/api/v1/metrics", MetricsHandler).Methods("GET")

	// Plugin endpoints
	router.HandleFunc("/api/v1/plugins", GetPluginsHandler).Methods("GET")
	router.HandleFunc("/api/v1/conversions", GetConversionsHandler).Methods("GET")

	// Graph CRUD endpoints
	router.HandleFunc("/api/v1/graphs", GetGraphsHandler).Methods("GET")
	router.HandleFunc("/api/v1/graphs", CreateGraphHandler).Methods("POST")
	router.HandleFunc("/api/v1/graphs/{id}", GetGraphHandler).Methods("GET")
	router.HandleFunc("/api/v1/graphs/{id}", UpdateGraphHandler).Methods("PUT")
	router.HandleFunc("/api/v1/graphs/{id}", DeleteGraphHandler).Methods("DELETE")

	// Graph operations
	router.HandleFunc("/api/v1/graphs/{id}/validate", ValidateGraphHandler).Methods("POST")
	router.HandleFunc("/api/v1/graphs/{id}/convert", ConvertGraphHandler).Methods("POST")
	router.HandleFunc("/api/v1/graphs/{id}/render", RenderGraphHandler).Methods("POST")

	return router
}

// TestPlugin provides a mock plugin for testing
type TestPlugin struct {
	ID          string
	Name        string
	Description string
	Category    string
	Enabled     bool
}

// createTestPlugin creates a test plugin
func createTestPlugin(id string, enabled bool) *TestPlugin {
	return &TestPlugin{
		ID:          id,
		Name:        fmt.Sprintf("Test Plugin %s", id),
		Description: fmt.Sprintf("Test plugin for %s", id),
		Category:    "test",
		Enabled:     enabled,
	}
}

// waitForCondition waits for a condition to be true or times out
func waitForCondition(timeout time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
