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

// AgentManagerClient provides access to the agent-manager scenario for spawning
// and managing coding agents (Claude Code, Codex, OpenCode).
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

// SpawnCodingAgent creates a new coding agent run.
// This is a two-step process:
// 1. Create a Task with the description and optional context attachments
// 2. Create a Run referencing that task
//
// contextAttachments are skill/context data to inject into the agent's prompt.
// Each attachment should have: type, key, label, content, and optionally tags.
func (c *AgentManagerClient) SpawnCodingAgent(ctx context.Context, taskDescription, runnerType, workspacePath string, timeoutMinutes int, contextAttachments []map[string]interface{}) (map[string]interface{}, error) {
	// Step 1: Create the Task
	taskData := map[string]interface{}{
		"title":        "Chat-initiated task",
		"description":  taskDescription,
		"scope_path":   "agent-inbox/chat-tasks",
		"project_root": workspacePath,
	}

	// Add context attachments if provided (skills converted from agent-inbox)
	if len(contextAttachments) > 0 {
		taskData["context_attachments"] = contextAttachments
	}

	taskReqBody := map[string]interface{}{
		"task": taskData,
	}

	taskBody, err := json.Marshal(taskReqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	taskReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/tasks", bytes.NewReader(taskBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create task request: %w", err)
	}
	taskReq.Header.Set("Content-Type", "application/json")

	taskResp, err := c.httpClient.Do(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager for task creation: %w", err)
	}
	defer taskResp.Body.Close()

	taskRespBody, _ := io.ReadAll(taskResp.Body)
	if taskResp.StatusCode != http.StatusOK && taskResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agent-manager task creation returned %d: %s", taskResp.StatusCode, string(taskRespBody))
	}

	var taskResult map[string]interface{}
	if err := json.Unmarshal(taskRespBody, &taskResult); err != nil {
		return nil, fmt.Errorf("failed to parse task response: %w", err)
	}

	// Extract task ID from the response (nested under "task")
	taskData, ok := taskResult["task"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected task response format: missing 'task' field")
	}
	taskID, ok := taskData["id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("unexpected task response format: missing or invalid task id")
	}

	// Step 2: Create the Run
	// Map runner_type to proto enum string format
	runnerTypeProto := "RUNNER_TYPE_CLAUDE_CODE"
	switch runnerType {
	case "claude-code":
		runnerTypeProto = "RUNNER_TYPE_CLAUDE_CODE"
	case "codex":
		runnerTypeProto = "RUNNER_TYPE_CODEX"
	case "opencode":
		runnerTypeProto = "RUNNER_TYPE_OPENCODE"
	}

	runReqBody := map[string]interface{}{
		"task_id": taskID,
		"inline_config": map[string]interface{}{
			"runner_type":       runnerTypeProto,
			"timeout":           fmt.Sprintf("%ds", timeoutMinutes*60),
			"requires_approval": true,
		},
	}

	runBody, err := json.Marshal(runReqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run request: %w", err)
	}

	runReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs", bytes.NewReader(runBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create run request: %w", err)
	}
	runReq.Header.Set("Content-Type", "application/json")

	runResp, err := c.httpClient.Do(runReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager for run creation: %w", err)
	}
	defer runResp.Body.Close()

	runRespBody, _ := io.ReadAll(runResp.Body)
	if runResp.StatusCode != http.StatusOK && runResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agent-manager run creation returned %d: %s", runResp.StatusCode, string(runRespBody))
	}

	var runResult map[string]interface{}
	if err := json.Unmarshal(runRespBody, &runResult); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
	}

	// Extract run data from response (nested under "run")
	runData, ok := runResult["run"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected run response format: missing 'run' field")
	}
	runID, _ := runData["id"].(string)
	runStatus, _ := runData["status"].(string)

	return map[string]interface{}{
		"success": true,
		"run_id":  runID,
		"task_id": taskID,
		"status":  runStatus,
		"message": fmt.Sprintf("Coding agent spawned successfully. Run ID: %s, Task ID: %s", runID, taskID),
	}, nil
}

// CheckAgentStatus gets the status of a coding agent run.
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

// StopAgent stops a running coding agent.
func (c *AgentManagerClient) StopAgent(ctx context.Context, runID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Agent run %s stopped", runID),
	}, nil
}

// ListActiveAgents returns a list of active agent runs.
func (c *AgentManagerClient) ListActiveAgents(ctx context.Context) (interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/runs?status=running", nil)
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

	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetAgentDiff gets the code changes from an agent run.
func (c *AgentManagerClient) GetAgentDiff(ctx context.Context, runID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/v1/runs/"+runID+"/diff", nil)
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

// ApproveAgentChanges approves and applies changes from an agent run.
func (c *AgentManagerClient) ApproveAgentChanges(ctx context.Context, runID string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs/"+runID+"/approve", nil)
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

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Changes from run %s approved and applied", runID),
	}, nil
}
