package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// AgentManagerClient provides HTTP client for agent-manager API
type AgentManagerClient struct {
	httpClient *http.Client
}

// NewAgentManagerClient creates a new agent-manager client
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ProfileRef identifies a profile by key and optional defaults
type ProfileRef struct {
	ProfileKey string        `json:"profileKey"`
	Defaults   *AgentProfile `json:"defaults,omitempty"`
}

// AgentProfile defines the configuration for running an agent
type AgentProfile struct {
	Name                 string        `json:"name"`
	ProfileKey           string        `json:"profileKey"`
	Description          string        `json:"description,omitempty"`
	RunnerType           string        `json:"runnerType"`
	Model                string        `json:"model,omitempty"`
	ModelPreset          string        `json:"modelPreset,omitempty"`
	MaxTurns             int32         `json:"maxTurns,omitempty"`
	Timeout              time.Duration `json:"timeout,omitempty"`
	AllowedTools         []string      `json:"allowedTools,omitempty"`
	SkipPermissionPrompt bool          `json:"skipPermissionPrompt,omitempty"`
	RequiresSandbox      bool          `json:"requiresSandbox,omitempty"`
	RequiresApproval     bool          `json:"requiresApproval,omitempty"`
	CreatedBy            string        `json:"createdBy,omitempty"`
}

// Task represents a task for agent execution
type Task struct {
	ID          string `json:"id,omitempty"`
	Prompt      string `json:"prompt"`
	WorkingDir  string `json:"workingDir,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreateTaskRequest is the request for creating a task
type CreateTaskRequest struct {
	Task *Task `json:"task"`
}

// CreateTaskResponse is the response from creating a task
type CreateTaskResponse struct {
	Task *Task `json:"task"`
}

// CreateRunRequest is the request for creating a run
type CreateRunRequest struct {
	TaskID     string      `json:"taskId"`
	ProfileRef *ProfileRef `json:"profileRef,omitempty"`
	Tag        *string     `json:"tag,omitempty"`
	RunMode    string      `json:"runMode,omitempty"`
}

// Run represents an agent run
type Run struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	ProfileID string `json:"agentProfileId,omitempty"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt,omitempty"`
	EndedAt   string `json:"endedAt,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CreateRunResponse is the response from creating a run
type CreateRunResponse struct {
	Run *Run `json:"run"`
}

// GetRunResponse is the response from getting a run
type GetRunResponse struct {
	Run *Run `json:"run"`
}

// EnsureProfileRequest requests a profile by key
type EnsureProfileRequest struct {
	ProfileKey     string        `json:"profileKey"`
	Defaults       *AgentProfile `json:"defaults,omitempty"`
	UpdateExisting bool          `json:"updateExisting,omitempty"`
}

// EnsureProfileResponse is the response from ensure profile
type EnsureProfileResponse struct {
	Profile *AgentProfile `json:"profile"`
	Created bool          `json:"created"`
	Updated bool          `json:"updated"`
}

// Health checks the agent-manager service health
func (c *AgentManagerClient) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed
func (c *AgentManagerClient) EnsureProfile(ctx context.Context, req *EnsureProfileRequest) (*EnsureProfileResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/profiles/ensure", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result EnsureProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask creates a new task
func (c *AgentManagerClient) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	req := CreateTaskRequest{Task: task}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/tasks", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Task, nil
}

// CreateRun starts a new run for a task
func (c *AgentManagerClient) CreateRun(ctx context.Context, req *CreateRunRequest) (*Run, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRun retrieves a run by ID
func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*Run, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// WaitForRun polls until a run reaches a terminal state
func (c *AgentManagerClient) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*Run, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			run, err := c.GetRun(ctx, runID)
			if err != nil {
				return nil, err
			}
			if run == nil {
				return nil, fmt.Errorf("run %s not found", runID)
			}

			// Check if terminal state
			switch run.Status {
			case "RUN_STATUS_COMPLETE", "RUN_STATUS_FAILED", "RUN_STATUS_CANCELLED",
				"complete", "failed", "cancelled":
				return run, nil
			}
		}
	}
}

// doRequest performs an HTTP request to agent-manager
func (c *AgentManagerClient) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *AgentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return url, nil
}

func (c *AgentManagerClient) parseResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *AgentManagerClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("agent-manager error: status %d, body: %s", resp.StatusCode, string(body))
}
