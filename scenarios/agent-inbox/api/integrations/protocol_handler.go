// Package integrations provides clients for external services.
//
// This file implements the ProtocolHandler for executing tools via the
// Tool Execution Protocol. Any scenario that implements the standard
// POST /api/v1/tools/execute endpoint can be handled by this handler.
//
// Protocol Contract:
//
//	Request:  POST /api/v1/tools/execute
//	          { "tool_name": "...", "arguments": {...} }
//
//	Response: { "success": true, "result": {...}, "is_async": false }
//	     or:  { "success": true, "is_async": true, "run_id": "...", "status": "pending" }
//	     or:  { "success": false, "error": "...", "code": "..." }
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ProtocolHandler implements ScenarioHandler for any protocol-compliant scenario.
// It forwards tool execution requests to the scenario's /api/v1/tools/execute endpoint.
type ProtocolHandler struct {
	scenarioName string
	baseURL      string
	httpClient   HTTPClient
	toolNames    map[string]bool // Set of tool names this handler supports
}

// ProtocolHandlerConfig contains configuration for creating a ProtocolHandler.
type ProtocolHandlerConfig struct {
	ScenarioName string
	BaseURL      string
	ToolNames    []string
	HTTPClient   HTTPClient
	Timeout      time.Duration
}

// NewProtocolHandler creates a new ProtocolHandler.
func NewProtocolHandler(cfg ProtocolHandlerConfig) *ProtocolHandler {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 5 * time.Minute // Long timeout for tool execution
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	// Build tool names set
	toolNames := make(map[string]bool)
	for _, name := range cfg.ToolNames {
		toolNames[name] = true
	}

	return &ProtocolHandler{
		scenarioName: cfg.ScenarioName,
		baseURL:      cfg.BaseURL,
		httpClient:   httpClient,
		toolNames:    toolNames,
	}
}

// Scenario returns the name of the scenario this handler serves.
func (h *ProtocolHandler) Scenario() string {
	return h.scenarioName
}

// CanHandle checks if this handler can execute the given tool.
func (h *ProtocolHandler) CanHandle(toolName string) bool {
	return h.toolNames[toolName]
}

// Execute runs the tool by forwarding to the scenario's execute endpoint.
func (h *ProtocolHandler) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Build request payload
	payload := map[string]interface{}{
		"tool_name": toolName,
		"arguments": args,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := h.baseURL + "/api/v1/tools/execute"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result ExecutionProtocolResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, string(body))
	}

	// Handle error responses
	if !result.Success {
		return nil, fmt.Errorf("tool execution failed: %s (code: %s)", result.Error, result.Code)
	}

	// Return the result, including async metadata if present
	response := make(map[string]interface{})
	if result.Result != nil {
		response = result.Result
	}

	// Include async metadata in the response
	if result.IsAsync {
		response["is_async"] = true
		response["run_id"] = result.RunID
		response["status"] = result.Status
	}

	return response, nil
}

// AddTool registers a tool name that this handler can handle.
// This is useful when tools are discovered dynamically.
func (h *ProtocolHandler) AddTool(toolName string) {
	h.toolNames[toolName] = true
}

// RemoveTool unregisters a tool name.
func (h *ProtocolHandler) RemoveTool(toolName string) {
	delete(h.toolNames, toolName)
}

// ToolCount returns the number of tools this handler can handle.
func (h *ProtocolHandler) ToolCount() int {
	return len(h.toolNames)
}

// ExecutionProtocolResponse represents the response from a Tool Execution Protocol endpoint.
type ExecutionProtocolResponse struct {
	// Success indicates whether the tool executed successfully
	Success bool `json:"success"`

	// Result contains the tool output (for successful executions)
	Result map[string]interface{} `json:"result,omitempty"`

	// Error contains the error message (for failed executions)
	Error string `json:"error,omitempty"`

	// Code is a machine-readable error code (for failed executions)
	Code string `json:"code,omitempty"`

	// IsAsync indicates whether this is a long-running operation
	IsAsync bool `json:"is_async"`

	// RunID is the unique identifier for async operations
	RunID string `json:"run_id,omitempty"`

	// Status is the current status for async operations
	Status string `json:"status,omitempty"`
}
