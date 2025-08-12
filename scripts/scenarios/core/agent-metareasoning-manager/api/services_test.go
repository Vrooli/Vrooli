package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Mock HTTP client for testing external services
type mockHTTPClient struct {
	responses map[string]*http.Response
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	key := fmt.Sprintf("%s:%s", req.Method, req.URL.Path)
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(bytes.NewBufferString("Not found")),
	}, nil
}

func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestWorkflowServiceCreation(t *testing.T) {
	config := &Config{
		Port:              "8093",
		DatabaseURL:       "postgres://test",
		N8nBase:        "http://localhost:5678",
		WindmillBase:   "http://localhost:8000",
		WindmillWorkspace: "demo",
	}

	service := NewWorkflowService(config)
	if service == nil {
		t.Fatal("Expected service to be created")
	}
	if service.config != config {
		t.Error("Service config not set correctly")
	}
}

func TestCheckPlatformStatus(t *testing.T) {
	// Create test server to mock n8n and windmill
	n8nServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/workflows" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
		}
	}))
	defer n8nServer.Close()

	windmillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"version": "1.2.3"})
		}
	}))
	defer windmillServer.Close()

	config := &Config{
		N8nBase:        n8nServer.URL,
		WindmillBase:   windmillServer.URL,
		WindmillWorkspace: "demo",
	}

	service := NewWorkflowService(config)
	platforms := service.CheckPlatformStatus()

	if len(platforms) != 2 {
		t.Fatalf("Expected 2 platforms, got %d", len(platforms))
	}

	// Check n8n status
	var n8nFound bool
	for _, p := range platforms {
		if p.Name == "n8n" {
			n8nFound = true
			if !p.Status {
				t.Error("Expected n8n to be available")
			}
			if p.URL != n8nServer.URL {
				t.Errorf("Expected n8n URL %s, got %s", n8nServer.URL, p.URL)
			}
		}
	}
	if !n8nFound {
		t.Error("n8n platform not found in status")
	}
}

func TestExecuteWorkflow(t *testing.T) {
	// Mock n8n server
	n8nServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/webhook/test-webhook" {
			// Read request body
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)
			
			// Return mock response
			response := map[string]interface{}{
				"success": true,
				"data":    reqBody,
				"message": "Workflow executed successfully",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer n8nServer.Close()

	config := &Config{
		N8nBase: n8nServer.URL,
	}

	service := NewWorkflowService(config)

	// Test n8n workflow execution
	workflow := &Workflow{
		ID:       uuid.New(),
		Name:     "Test Workflow",
		Platform: "n8n",
		Config: json.RawMessage(fmt.Sprintf(`{
			"nodes": [],
			"connections": {},
			"settings": {
				"executionOrder": "v1"
			},
			"webhook_url": "%s/webhook/test-webhook"
		}`, n8nServer.URL)),
	}

	req := ExecutionRequest{
		Input: json.RawMessage(`{"test": "data"}`),
		Metadata: map[string]interface{}{
			"timeout": 30,
		},
	}

	resp, err := service.ExecuteWorkflow(workflow, req)
	if err != nil {
		t.Fatalf("ExecuteWorkflow failed: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %s", resp.Status)
	}
	if resp.WorkflowID != workflow.ID {
		t.Errorf("WorkflowID mismatch: got %v, want %v", resp.WorkflowID, workflow.ID)
	}
}

func TestExecuteWindmillWorkflow(t *testing.T) {
	// Mock Windmill server
	windmillServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/w/demo/jobs/run/p/test-script" {
			// Return mock job response
			response := map[string]interface{}{
				"id":     "job-123",
				"status": "completed",
				"result": map[string]interface{}{
					"output": "Script executed successfully",
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer windmillServer.Close()

	config := &Config{
		WindmillBase:   windmillServer.URL,
		WindmillWorkspace: "demo",
	}

	service := NewWorkflowService(config)

	// Test Windmill workflow execution
	workflow := &Workflow{
		ID:       uuid.New(),
		Name:     "Test Windmill Script",
		Platform: "windmill",
		Config: json.RawMessage(`{
			"script_path": "test-script",
			"language": "python3",
			"content": "def main(): return {'result': 'success'}"
		}`),
	}

	req := ExecutionRequest{
		Metadata: map[string]interface{}{
			"arg1": "value1",
		},
	}

	resp, err := service.ExecuteWorkflow(workflow, req)
	if err != nil {
		t.Fatalf("ExecuteWorkflow (Windmill) failed: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %s", resp.Status)
	}
}

func TestGenerateWorkflow(t *testing.T) {
	// Mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/generate" {
			// Read request
			var req OllamaGenerateRequest
			json.NewDecoder(r.Body).Decode(&req)
			
			// Return mock generated workflow
			mockWorkflow := map[string]interface{}{
				"name":        "Generated Workflow",
				"description": "AI-generated workflow",
				"nodes": []interface{}{
					map[string]interface{}{
						"id":   "1",
						"type": "webhook",
						"name": "Webhook Trigger",
					},
				},
			}
			
			response := OllamaGenerateResponse{
				Response: jsonString(mockWorkflow),
				Done:     true,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ollamaServer.Close()

	// Note: This test would need integration with actual service configuration
	// For now, we'll test with the mock server URL

	config := &Config{}
	service := NewWorkflowService(config)

	// Test workflow generation
	workflow, err := service.GenerateWorkflow(
		"Create a workflow that processes webhooks",
		"n8n",
		"llama2",
		0.7,
	)
	if err != nil {
		t.Fatalf("GenerateWorkflow failed: %v", err)
	}

	if workflow.Name != "Generated Workflow" {
		t.Errorf("Expected name 'Generated Workflow', got %s", workflow.Name)
	}
	if workflow.Platform != "n8n" {
		t.Errorf("Expected platform 'n8n', got %s", workflow.Platform)
	}
}

func TestImportWorkflow(t *testing.T) {
	config := &Config{}
	service := NewWorkflowService(config)

	tests := []struct {
		name     string
		platform string
		data     json.RawMessage
		wfName   string
		wantErr  bool
	}{
		{
			name:     "import n8n workflow",
			platform: "n8n",
			data: json.RawMessage(`{
				"name": "Imported n8n Workflow",
				"nodes": [
					{"id": "1", "type": "webhook", "name": "Start"}
				],
				"connections": {}
			}`),
			wfName:  "Custom Import Name",
			wantErr: false,
		},
		{
			name:     "import windmill script",
			platform: "windmill",
			data: json.RawMessage(`{
				"path": "f/scripts/test",
				"summary": "Test script",
				"content": "def main():\n    return {'result': 'success'}",
				"language": "python3"
			}`),
			wfName:  "Windmill Import",
			wantErr: false,
		},
		{
			name:     "invalid platform",
			platform: "invalid",
			data:     json.RawMessage(`{}`),
			wfName:   "Test",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow, err := service.ImportWorkflow(tt.platform, tt.data, tt.wfName)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("ImportWorkflow failed: %v", err)
			}
			
			if workflow.Name != tt.wfName {
				t.Errorf("Expected name %s, got %s", tt.wfName, workflow.Name)
			}
			if workflow.Platform != tt.platform {
				t.Errorf("Expected platform %s, got %s", tt.platform, workflow.Platform)
			}
		})
	}
}

func TestExportWorkflow(t *testing.T) {
	config := &Config{}
	service := NewWorkflowService(config)

	tests := []struct {
		name     string
		workflow *Workflow
		format   string
		wantErr  bool
	}{
		{
			name: "export n8n as json",
			workflow: &Workflow{
				ID:       uuid.New(),
				Name:     "Test n8n Export",
				Platform: "n8n",
				Config: json.RawMessage(`{
					"nodes": [{"id": "1", "type": "webhook"}],
					"connections": {}
				}`),
			},
			format:  "json",
			wantErr: false,
		},
		{
			name: "export windmill as yaml",
			workflow: &Workflow{
				ID:       uuid.New(),
				Name:     "Test Windmill Export",
				Platform: "windmill",
				Config: json.RawMessage(`{
					"script_path": "test",
					"content": "def main(): pass"
				}`),
			},
			format:  "yaml",
			wantErr: false,
		},
		{
			name: "invalid format",
			workflow: &Workflow{
				ID:       uuid.New(),
				Platform: "n8n",
				Config:   json.RawMessage(`{}`),
			},
			format:  "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exported, err := service.ExportWorkflow(tt.workflow, tt.format)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("ExportWorkflow failed: %v", err)
			}
			
			// Check that export contains expected fields
			exportMap, ok := exported.(map[string]interface{})
			if !ok {
				t.Fatal("Expected export to be a map")
			}
			
			if exportMap["format"] != tt.format {
				t.Errorf("Expected format %s in export", tt.format)
			}
			if exportMap["platform"] != tt.workflow.Platform {
				t.Errorf("Expected platform %s in export", tt.workflow.Platform)
			}
		})
	}
}

func TestGetAvailableModels(t *testing.T) {
	// Mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/tags" {
			response := OllamaListResponse{
				Models: []OllamaModel{
					{
						Name:       "llama2:latest",
						ModifiedAt: time.Now(),
						Size:       1234567890,
					},
					{
						Name:       "codellama:latest",
						ModifiedAt: time.Now(),
						Size:       2345678901,
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ollamaServer.Close()

	// Note: This test would need integration with actual service configuration
	// For now, we'll test with the mock server URL

	config := &Config{}
	service := NewWorkflowService(config)

	models, err := service.GetAvailableModels()
	if err != nil {
		t.Fatalf("GetAvailableModels failed: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}

	if models[0] != "llama2:latest" {
		t.Errorf("Expected first model 'llama2:latest', got %s", models[0])
	}
	if models[1] != "codellama:latest" {
		t.Errorf("Expected second model 'codellama:latest', got %s", models[1])
	}
}

func TestExtractWorkflowFromResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantName string
		wantErr  bool
	}{
		{
			name: "valid json workflow",
			response: `Here's the workflow:
			{
				"name": "Test Workflow",
				"description": "A test",
				"nodes": []
			}`,
			wantName: "Test Workflow",
			wantErr:  false,
		},
		{
			name: "json in code block",
			response: "```json\n{\"name\": \"Code Block Workflow\", \"nodes\": []}\n```",
			wantName: "Code Block Workflow",
			wantErr:  false,
		},
		{
			name:     "invalid json",
			response: "This is not valid JSON",
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWorkflowFromResponse(tt.response)
			
			if tt.wantErr {
				// Check if result is empty or invalid
				var parsed map[string]interface{}
				err := json.Unmarshal([]byte(result), &parsed)
				if err == nil && parsed["name"] != nil {
					t.Error("Expected invalid result but got valid JSON")
				}
				return
			}
			
			var workflow map[string]interface{}
			err := json.Unmarshal([]byte(result), &workflow)
			if err != nil {
				t.Fatalf("Failed to parse extracted workflow: %v", err)
			}
			
			if workflow["name"] != tt.wantName {
				t.Errorf("Expected name %s, got %v", tt.wantName, workflow["name"])
			}
		})
	}
}

// Helper function to convert interface to JSON string
func jsonString(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}