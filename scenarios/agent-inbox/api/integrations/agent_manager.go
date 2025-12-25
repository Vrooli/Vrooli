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
func (c *AgentManagerClient) SpawnCodingAgent(ctx context.Context, task, runnerType, workspacePath string, timeoutMinutes int) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"task": map[string]interface{}{
			"name":        "Chat-initiated task",
			"description": task,
		},
		"runner_type":     runnerType,
		"workspace_path":  workspacePath,
		"timeout_seconds": timeoutMinutes * 60,
		"auto_approve":    false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v1/runs", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"run_id":  result["id"],
		"status":  result["status"],
		"message": fmt.Sprintf("Coding agent spawned successfully. Run ID: %v", result["id"]),
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
