// Package integrations provides clients for external services.
// Each integration is isolated behind a clean interface to enable testing
// and potential swapping of implementations.
package integrations

import (
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
