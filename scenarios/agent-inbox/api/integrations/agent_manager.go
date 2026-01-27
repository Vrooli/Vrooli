// Package integrations provides clients for external services.
// Each integration is isolated behind a clean interface to enable testing
// and potential swapping of implementations.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/config"
)

// AgentManagerClient provides direct REST API access to agent-manager.
// This client is used for reconciliation during server restart recovery.
// Tool execution flows through the Tool Execution Protocol (ProtocolHandler) instead.
type AgentManagerClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAgentManagerClient creates a new agent-manager client.
// Returns an error if the agent-manager service is not available.
func NewAgentManagerClient() (*AgentManagerClient, error) {
	cfg := config.Default()
	return NewAgentManagerClientWithConfig(cfg.Integration.AgentManagerTimeout)
}

// NewAgentManagerClientWithConfig creates a new agent-manager client with explicit timeout.
// This enables testing and custom configuration injection.
func NewAgentManagerClientWithConfig(timeout time.Duration) (*AgentManagerClient, error) {
	baseURL, err := getAgentManagerURL()
	if err != nil {
		return nil, err
	}

	return &AgentManagerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// getAgentManagerURL discovers the agent-manager API URL.
func getAgentManagerURL() (string, error) {
	// Try environment variable first (set by lifecycle system)
	if url := os.Getenv("AGENT_MANAGER_API_URL"); url != "" {
		return url, nil
	}

	// Fall back to port discovery via CLI
	cmd := exec.Command("vrooli", "scenario", "port", "agent-manager", "API_PORT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("agent-manager not available: %w", err)
	}

	port := strings.TrimSpace(string(output))
	return fmt.Sprintf("http://localhost:%s", port), nil
}

// CheckAgentStatus gets the status of a coding agent run.
// This is used by the reconciliation service to verify agent state after server restarts.
func (c *AgentManagerClient) CheckAgentStatus(ctx context.Context, runID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// =============================================================================
// Agent Mode Types - For agent-inbox integration
// =============================================================================

// RunnerType identifies which agent runner to use.
type RunnerType string

const (
	RunnerTypeClaudeCode RunnerType = "claude-code"
	RunnerTypeCodex      RunnerType = "codex"
	RunnerTypeOpenCode   RunnerType = "opencode"
)

// AgentChatConfig contains configuration for starting an agent chat.
type AgentChatConfig struct {
	// RunnerType specifies which runner to use (claude-code, codex, opencode)
	RunnerType RunnerType `json:"runner_type"`

	// ProjectPath is the directory where the agent will operate
	ProjectPath string `json:"project_path"`

	// Model optionally specifies the model to use (e.g., "claude-opus-4")
	Model string `json:"model,omitempty"`

	// MaxTurns optionally limits the number of conversation turns
	MaxTurns int `json:"max_turns,omitempty"`
}

// AgentChatSession contains the session info returned after starting agent mode.
type AgentChatSession struct {
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id,omitempty"` // Runner-specific session ID for continuation
}

// RunStatus represents the current state of a run.
type RunStatus string

const (
	RunStatusPending     RunStatus = "pending"
	RunStatusStarting    RunStatus = "starting"
	RunStatusRunning     RunStatus = "running"
	RunStatusNeedsReview RunStatus = "needs_review"
	RunStatusComplete    RunStatus = "complete"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
)

// AgentRunStatus contains the status info for a run.
type AgentRunStatus struct {
	RunID           string    `json:"run_id"`
	Status          RunStatus `json:"status"`
	Phase           string    `json:"phase"`
	ProgressPercent int       `json:"progress_percent"`
	SessionID       string    `json:"session_id,omitempty"`
	ErrorMsg        string    `json:"error_msg,omitempty"`
}

// TranslatedEvent represents an agent-manager event translated for inbox rendering.
type TranslatedEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`      // message, tool_call, tool_result, status, error, progress
	Role      string    `json:"role"`      // user, assistant, system, tool
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int64     `json:"sequence"`

	// Tool fields (for tool_call and tool_result types)
	ToolName    string `json:"tool_name,omitempty"`
	ToolInput   string `json:"tool_input,omitempty"`  // JSON string of tool input
	ToolOutput  string `json:"tool_output,omitempty"` // Tool result output
	ToolSuccess bool   `json:"tool_success,omitempty"`

	// Status fields (for status type)
	RunStatus string `json:"run_status,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Progress  int    `json:"progress,omitempty"`
}

// =============================================================================
// Agent Mode Methods
// =============================================================================

// StartAgentChat creates a task and run for agent mode chat.
// This is the entry point for starting an agentic coding session.
func (c *AgentManagerClient) StartAgentChat(ctx context.Context, message string, cfg AgentChatConfig) (*AgentChatSession, error) {
	// First create a task
	taskReq := map[string]interface{}{
		"title":       "Agent Chat Session",
		"description": message,
		"scopePath":   cfg.ProjectPath,
		"projectRoot": cfg.ProjectPath,
	}
	taskBody, _ := json.Marshal(taskReq)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/tasks", bytes.NewReader(taskBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create task request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create task: %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResult map[string]interface{}
	if err := json.Unmarshal(respBody, &taskResult); err != nil {
		return nil, fmt.Errorf("failed to parse task response: %w", err)
	}

	taskID, ok := taskResult["id"].(string)
	if !ok {
		return nil, fmt.Errorf("task response missing id")
	}

	// Now create a run for this task
	runReq := map[string]interface{}{
		"taskId":     taskID,
		"runnerType": string(cfg.RunnerType),
		"runMode":    "in_place", // Agent mode runs in place, not sandboxed
	}
	if cfg.Model != "" {
		runReq["model"] = cfg.Model
	}
	if cfg.MaxTurns > 0 {
		runReq["maxTurns"] = cfg.MaxTurns
	}
	runBody, _ := json.Marshal(runReq)

	req, err = http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs", bytes.NewReader(runBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create run request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create run: %d: %s", resp.StatusCode, string(respBody))
	}

	var runResult map[string]interface{}
	if err := json.Unmarshal(respBody, &runResult); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
	}

	run, _ := runResult["run"].(map[string]interface{})
	if run == nil {
		run = runResult
	}

	runID, _ := run["id"].(string)
	sessionID, _ := run["sessionId"].(string)

	return &AgentChatSession{
		TaskID:    taskID,
		RunID:     runID,
		SessionID: sessionID,
	}, nil
}

// ContinueChat sends a follow-up message to an existing agent run.
// Uses the continuation API to maintain conversation state.
func (c *AgentManagerClient) ContinueChat(ctx context.Context, runID, message string) error {
	reqBody := map[string]interface{}{
		"message": message,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs/"+runID+"/continue", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create continue request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("continue failed: %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetEvents retrieves events for a run, optionally filtered by sequence.
func (c *AgentManagerClient) GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error) {
	url := fmt.Sprintf("%s/api/v1/runs/%s/events?after_sequence=%d", c.baseURL, runID, afterSequence)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create events request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get events failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Events []map[string]interface{} `json:"events"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse events response: %w", err)
	}

	// Translate events to inbox format
	events := make([]*TranslatedEvent, 0, len(result.Events))
	for _, raw := range result.Events {
		event := translateEvent(raw)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// translateEvent converts an agent-manager event to TranslatedEvent format.
func translateEvent(raw map[string]interface{}) *TranslatedEvent {
	id, _ := raw["id"].(string)
	eventType, _ := raw["eventType"].(string)
	sequence, _ := raw["sequence"].(float64)
	timestampStr, _ := raw["timestamp"].(string)
	data, _ := raw["data"].(map[string]interface{})

	timestamp, _ := time.Parse(time.RFC3339, timestampStr)

	event := &TranslatedEvent{
		ID:        id,
		Sequence:  int64(sequence),
		Timestamp: timestamp,
	}

	switch eventType {
	case "message":
		event.Type = "message"
		event.Role, _ = data["role"].(string)
		event.Content, _ = data["content"].(string)

	case "tool_call":
		event.Type = "tool_call"
		event.Role = "assistant"
		event.ToolName, _ = data["toolName"].(string)
		if input, ok := data["input"]; ok {
			inputBytes, _ := json.Marshal(input)
			event.ToolInput = string(inputBytes)
		}

	case "tool_result":
		event.Type = "tool_result"
		event.Role = "tool"
		event.ToolName, _ = data["toolName"].(string)
		event.ToolOutput, _ = data["output"].(string)
		event.ToolSuccess, _ = data["success"].(bool)
		if errMsg, ok := data["error"].(string); ok && errMsg != "" {
			event.ToolOutput = errMsg
			event.ToolSuccess = false
		}

	case "status":
		event.Type = "status"
		event.Role = "system"
		event.RunStatus, _ = data["newStatus"].(string)
		if reason, ok := data["reason"].(string); ok {
			event.Content = reason
		}

	case "error":
		event.Type = "error"
		event.Role = "system"
		event.Content, _ = data["message"].(string)

	case "log":
		// Skip log events for now (they're mostly internal)
		return nil

	default:
		// Unknown event type, skip
		return nil
	}

	return event
}

// StopRun stops a running agent run.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to create stop request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetRunStatus gets the current status of a run.
func (c *AgentManagerClient) GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Run map[string]interface{} `json:"run"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	run := result.Run
	if run == nil {
		// Maybe the response is the run directly
		if err := json.Unmarshal(respBody, &run); err != nil {
			return nil, fmt.Errorf("failed to parse run response: %w", err)
		}
	}

	status := &AgentRunStatus{
		RunID: runID,
	}

	if s, ok := run["status"].(string); ok {
		status.Status = RunStatus(s)
	}
	if p, ok := run["phase"].(string); ok {
		status.Phase = p
	}
	if pp, ok := run["progressPercent"].(float64); ok {
		status.ProgressPercent = int(pp)
	}
	if sid, ok := run["sessionId"].(string); ok {
		status.SessionID = sid
	}
	if em, ok := run["errorMsg"].(string); ok {
		status.ErrorMsg = em
	}

	return status, nil
}

// GetWebSocketURL returns the WebSocket URL for connecting to agent-manager events.
func (c *AgentManagerClient) GetWebSocketURL() string {
	// Convert http:// to ws://
	wsURL := strings.Replace(c.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	return wsURL + "/api/v1/ws"
}
