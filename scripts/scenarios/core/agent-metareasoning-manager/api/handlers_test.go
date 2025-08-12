package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Mock service for testing handlers
type mockWorkflowService struct {
	workflows      []Workflow
	executionResp  *ExecutionResponse
	generateResp   *WorkflowCreate
	models         []ModelInfo
	platforms      []PlatformInfo
	shouldError    bool
}

func (m *mockWorkflowService) ExecuteWorkflow(workflow *Workflow, req ExecutionRequest) (*ExecutionResponse, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	if m.executionResp != nil {
		return m.executionResp, nil
	}
	return &ExecutionResponse{
		ID:         uuid.New(),
		WorkflowID: workflow.ID,
		Status:     "success",
		Data:       json.RawMessage(`{"test": "result"}`),
		ExecutionMS: 1000,
	}, nil
}

func (m *mockWorkflowService) GenerateWorkflow(prompt, platform, model string, temperature float64) (*WorkflowCreate, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	if m.generateResp != nil {
		return m.generateResp, nil
	}
	return &WorkflowCreate{
		Name:        "Generated Workflow",
		Description: "Generated from: " + prompt,
		Type:        "generated",
		Platform:    platform,
		Config:      json.RawMessage(`{"generated": true}`),
	}, nil
}

func (m *mockWorkflowService) ImportWorkflow(platform string, data json.RawMessage, name string) (*Workflow, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	// Return a Workflow, not WorkflowCreate
	return &Workflow{
		ID:       uuid.New(),
		Name:     name,
		Platform: platform,
		Config:   data,
		Type:     "imported",
	}, nil
}

func (m *mockWorkflowService) ExportWorkflow(workflow *Workflow, format string) (interface{}, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	return map[string]interface{}{
		"format":   format,
		"platform": workflow.Platform,
		"data":     workflow.Config,
	}, nil
}

func (m *mockWorkflowService) GetAvailableModels() ([]ModelInfo, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error")
	}
	if m.models != nil {
		return m.models, nil
	}
	return []ModelInfo{
		{Name: "llama2", SizeMB: 3000},
		{Name: "codellama", SizeMB: 3500},
	}, nil
}

func (m *mockWorkflowService) CheckPlatformStatus() []PlatformInfo {
	if m.platforms != nil {
		return m.platforms
	}
	return []PlatformInfo{
		{Name: "n8n", Description: "n8n automation platform", Status: true, URL: "http://localhost:5678"},
		{Name: "windmill", Description: "Windmill automation platform", Status: true, URL: "http://localhost:8000"},
	}
}

// Setup test handler with mock service
func setupTestHandler() (*Handler, *mockWorkflowService) {
	mockService := &mockWorkflowService{}
	handler := NewHandler(mockService)
	return handler, mockService
}

// Setup test database for handlers that need it
func setupTestHandlerDB(t *testing.T) {
	// Use a simple in-memory map for testing if real DB not available
	if db == nil {
		t.Skip("Database not initialized for handler tests")
	}
}

func TestHealthHandler(t *testing.T) {
	handler, mockService := setupTestHandler()
	
	// Setup mock platforms
	mockService.platforms = []PlatformInfo{
		{Name: "n8n", Description: "n8n platform", Status: true, URL: "http://localhost:5678"},
		{Name: "windmill", Description: "windmill platform", Status: false, URL: "http://localhost:8000"},
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.HealthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status == "" {
		t.Error("Expected health status to be set")
	}
	if resp.Version != "3.0.0" {
		t.Errorf("Expected version 3.0.0, got %s", resp.Version)
	}
	if !resp.Services["n8n"] {
		t.Error("Expected n8n to be healthy")
	}
	if resp.Services["windmill"] {
		t.Error("Expected windmill to be unhealthy")
	}
}

func TestListWorkflowsHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestGetWorkflowHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestCreateWorkflowHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestExecuteWorkflowHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestGenerateWorkflowHandler(t *testing.T) {
	handler, mockService := setupTestHandler()

	tests := []struct {
		name       string
		body       GenerateRequest
		wantStatus int
		shouldFail bool
	}{
		{
			name: "valid generation request",
			body: GenerateRequest{
				Prompt:      "Create a data processing workflow",
				Platform:    "n8n",
				Model:       "llama2",
				Temperature: 0.7,
			},
			wantStatus: http.StatusOK,
			shouldFail: false,
		},
		{
			name: "missing prompt",
			body: GenerateRequest{
				Platform: "n8n",
			},
			wantStatus: http.StatusBadRequest,
			shouldFail: false,
		},
		{
			name: "default platform",
			body: GenerateRequest{
				Prompt: "Test workflow",
			},
			wantStatus: http.StatusOK,
			shouldFail: false,
		},
		{
			name: "service error",
			body: GenerateRequest{
				Prompt:   "Test",
				Platform: "n8n",
			},
			wantStatus: http.StatusInternalServerError,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.shouldError = tt.shouldFail

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/workflows/generate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.GenerateWorkflowHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var resp WorkflowCreate
				err := json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Name == "" {
					t.Error("Expected workflow name to be set")
				}
			}
		})
	}
}

func TestImportWorkflowHandler(t *testing.T) {
	handler, mockService := setupTestHandler()

	tests := []struct {
		name       string
		body       ImportRequest
		wantStatus int
		shouldFail bool
	}{
		{
			name: "valid import",
			body: ImportRequest{
				Platform: "n8n",
				Name:     "Imported Workflow",
				Data:     json.RawMessage(`{"nodes": []}`),
			},
			wantStatus: http.StatusCreated,
			shouldFail: false,
		},
		{
			name: "missing platform",
			body: ImportRequest{
				Name: "Test",
				Data: json.RawMessage(`{}`),
			},
			wantStatus: http.StatusBadRequest,
			shouldFail: false,
		},
		{
			name: "service error",
			body: ImportRequest{
				Platform: "windmill",
				Name:     "Test",
				Data:     json.RawMessage(`{}`),
			},
			wantStatus: http.StatusInternalServerError,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.shouldError = tt.shouldFail

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/workflows/import", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ImportWorkflowHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusCreated {
				var resp WorkflowCreate
				err := json.NewDecoder(w.Body).Decode(&resp)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Name != tt.body.Name {
					t.Errorf("Expected name %s, got %s", tt.body.Name, resp.Name)
				}
			}
		})
	}
}

func TestGetModelsHandler(t *testing.T) {
	handler, mockService := setupTestHandler()

	mockService.models = []string{"llama2:latest", "codellama:latest", "mistral:latest"}

	req := httptest.NewRequest("GET", "/models", nil)
	w := httptest.NewRecorder()

	handler.GetModelsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp ModelsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Count != 3 {
		t.Errorf("Expected 3 models, got %d", resp.Count)
	}
	if len(resp.Models) != 3 {
		t.Errorf("Expected 3 models in array, got %d", len(resp.Models))
	}
}

func TestGetPlatformsHandler(t *testing.T) {
	handler, mockService := setupTestHandler()

	mockService.platforms = []PlatformInfo{
		{Name: "n8n", Description: "n8n platform", Status: true, URL: "http://localhost:5678"},
		{Name: "windmill", Description: "windmill platform", Status: false, URL: "http://localhost:8000"},
	}

	req := httptest.NewRequest("GET", "/platforms", nil)
	w := httptest.NewRecorder()

	handler.GetPlatformsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp PlatformsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Platforms) != 2 {
		t.Errorf("Expected 2 platforms, got %d", len(resp.Platforms))
	}

	// Check n8n platform
	n8n := resp.Platforms[0]
	if n8n.Name != "n8n" {
		t.Errorf("Expected first platform to be n8n, got %s", n8n.Name)
	}
	if !n8n.Status {
		t.Error("Expected n8n to be available")
	}

	// Check windmill platform
	windmill := resp.Platforms[1]
	if windmill.Name != "windmill" {
		t.Errorf("Expected second platform to be windmill, got %s", windmill.Name)
	}
	if windmill.Status {
		t.Error("Expected windmill to be unavailable")
	}
}

func TestAnalyzeHandler(t *testing.T) {
	// This test requires database mock for workflow lookup
	t.Skip("Requires database mock implementation")
}

func TestSearchWorkflowsHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestExportWorkflowHandler(t *testing.T) {
	// This test requires database mock for workflow lookup
	t.Skip("Requires database mock implementation")
}

func TestCloneWorkflowHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestGetHistoryHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestGetMetricsHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

func TestGetStatsHandler(t *testing.T) {
	// This test requires database mock
	t.Skip("Requires database mock implementation")
}

// Helper functions for testing - moved to utils_test.go to avoid duplication

// Add context helper for testing authenticated endpoints
func addUserContext(req *http.Request, user string) *http.Request {
	ctx := context.WithValue(req.Context(), "user", user)
	return req.WithContext(ctx)
}

// Test router setup
func TestSetupRoutes(t *testing.T) {
	handler, _ := setupTestHandler()
	router := setupRoutes(handler)

	if router == nil {
		t.Fatal("Expected router to be created")
	}

	// Test that routes are registered
	testRoutes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"GET", "/platforms"},
		{"GET", "/models"},
		{"GET", "/stats"},
		{"GET", "/workflows"},
		{"POST", "/workflows"},
		{"GET", "/workflows/{id}"},
		{"PUT", "/workflows/{id}"},
		{"DELETE", "/workflows/{id}"},
		{"POST", "/workflows/{id}/execute"},
		{"POST", "/analyze/{type}"},
		{"GET", "/workflows/search"},
		{"POST", "/workflows/generate"},
		{"POST", "/workflows/import"},
		{"GET", "/workflows/{id}/export"},
		{"POST", "/workflows/{id}/clone"},
		{"GET", "/workflows/{id}/history"},
		{"GET", "/workflows/{id}/metrics"},
	}

	for _, tr := range testRoutes {
		// Create a test request
		req := httptest.NewRequest(tr.method, tr.path, nil)
		match := &mux.RouteMatch{}
		
		// Check if route exists
		if !router.Match(req, match) {
			// Some routes have path variables, try with example values
			testPath := tr.path
			if tr.path == "/workflows/{id}" ||
			   tr.path == "/workflows/{id}/execute" ||
			   tr.path == "/workflows/{id}/export" ||
			   tr.path == "/workflows/{id}/clone" ||
			   tr.path == "/workflows/{id}/history" ||
			   tr.path == "/workflows/{id}/metrics" {
				testPath = strings.Replace(tr.path, "{id}", "123e4567-e89b-12d3-a456-426614174000", 1)
			} else if tr.path == "/analyze/{type}" {
				testPath = strings.Replace(tr.path, "{type}", "data-analysis", 1)
			}
			
			req = httptest.NewRequest(tr.method, testPath, nil)
			if !router.Match(req, match) {
				t.Errorf("Route %s %s not found", tr.method, tr.path)
			}
		}
	}
}

// Standard packages are imported at the top of the file

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}