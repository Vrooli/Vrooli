package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// mockHTTPClient is a test double for HTTPClient.
type mockHTTPClient struct {
	response *http.Response
	err      error
	lastReq  *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	return m.response, m.err
}

func makeResponse(statusCode int, body interface{}) *http.Response {
	bodyBytes, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Header:     make(http.Header),
	}
}

func TestNewProtocolHandler(t *testing.T) {
	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"tool_a", "tool_b"},
	})

	if handler.Scenario() != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got %s", handler.Scenario())
	}

	if handler.ToolCount() != 2 {
		t.Errorf("expected 2 tools, got %d", handler.ToolCount())
	}
}

func TestProtocolHandler_CanHandle(t *testing.T) {
	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"supported_tool"},
	})

	if !handler.CanHandle("supported_tool") {
		t.Error("expected CanHandle to return true for supported tool")
	}

	if handler.CanHandle("unsupported_tool") {
		t.Error("expected CanHandle to return false for unsupported tool")
	}
}

func TestProtocolHandler_Execute_Success(t *testing.T) {
	mockClient := &mockHTTPClient{
		response: makeResponse(http.StatusOK, ExecutionProtocolResponse{
			Success: true,
			Result: map[string]interface{}{
				"deployments": []interface{}{},
				"count":       float64(0),
			},
		}),
	}

	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"list_deployments"},
		HTTPClient:   mockClient,
	})

	result, err := handler.Execute(context.Background(), "list_deployments", map[string]interface{}{
		"status": "deployed",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request
	if mockClient.lastReq == nil {
		t.Fatal("no request was made")
	}
	if mockClient.lastReq.URL.String() != "http://localhost:8080/api/v1/tools/execute" {
		t.Errorf("unexpected URL: %s", mockClient.lastReq.URL.String())
	}
	if mockClient.lastReq.Method != "POST" {
		t.Errorf("expected POST, got %s", mockClient.lastReq.Method)
	}

	// Verify result
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if _, ok := resultMap["deployments"]; !ok {
		t.Error("expected 'deployments' in result")
	}
}

func TestProtocolHandler_Execute_AsyncSuccess(t *testing.T) {
	mockClient := &mockHTTPClient{
		response: makeResponse(http.StatusAccepted, ExecutionProtocolResponse{
			Success: true,
			Result: map[string]interface{}{
				"deployment_id": "dep-123",
				"message":       "Deployment started",
			},
			IsAsync: true,
			RunID:   "run-456",
			Status:  "pending",
		}),
	}

	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"execute_deployment"},
		HTTPClient:   mockClient,
	})

	result, err := handler.Execute(context.Background(), "execute_deployment", map[string]interface{}{
		"deployment": "my-deployment",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	// Verify async metadata is included
	if !resultMap["is_async"].(bool) {
		t.Error("expected is_async=true in result")
	}
	if resultMap["run_id"] != "run-456" {
		t.Errorf("expected run_id='run-456', got %v", resultMap["run_id"])
	}
	if resultMap["status"] != "pending" {
		t.Errorf("expected status='pending', got %v", resultMap["status"])
	}
}

func TestProtocolHandler_Execute_Error(t *testing.T) {
	mockClient := &mockHTTPClient{
		response: makeResponse(http.StatusNotFound, ExecutionProtocolResponse{
			Success: false,
			Error:   "deployment not found",
			Code:    "not_found",
		}),
	}

	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"check_deployment_status"},
		HTTPClient:   mockClient,
	})

	_, err := handler.Execute(context.Background(), "check_deployment_status", map[string]interface{}{
		"deployment": "nonexistent",
	})

	if err == nil {
		t.Fatal("expected error for failed tool execution")
	}

	// Verify error message contains details
	errMsg := err.Error()
	if !contains(errMsg, "deployment not found") {
		t.Errorf("expected error to contain 'deployment not found', got: %s", errMsg)
	}
	if !contains(errMsg, "not_found") {
		t.Errorf("expected error to contain 'not_found', got: %s", errMsg)
	}
}

func TestProtocolHandler_AddRemoveTool(t *testing.T) {
	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "test-scenario",
		BaseURL:      "http://localhost:8080",
		ToolNames:    []string{"initial_tool"},
	})

	if handler.ToolCount() != 1 {
		t.Errorf("expected 1 tool, got %d", handler.ToolCount())
	}

	// Add a tool
	handler.AddTool("new_tool")
	if handler.ToolCount() != 2 {
		t.Errorf("expected 2 tools after add, got %d", handler.ToolCount())
	}
	if !handler.CanHandle("new_tool") {
		t.Error("expected CanHandle to return true for newly added tool")
	}

	// Remove a tool
	handler.RemoveTool("initial_tool")
	if handler.ToolCount() != 1 {
		t.Errorf("expected 1 tool after remove, got %d", handler.ToolCount())
	}
	if handler.CanHandle("initial_tool") {
		t.Error("expected CanHandle to return false for removed tool")
	}
}

func TestProtocolHandler_Scenario(t *testing.T) {
	handler := NewProtocolHandler(ProtocolHandlerConfig{
		ScenarioName: "my-scenario",
		BaseURL:      "http://localhost:8080",
	})

	if handler.Scenario() != "my-scenario" {
		t.Errorf("expected scenario 'my-scenario', got %s", handler.Scenario())
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
