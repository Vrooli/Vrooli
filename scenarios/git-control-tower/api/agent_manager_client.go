package main

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

// AgentManagerClient is a lightweight HTTP client for agent-manager APIs.
type AgentManagerClient struct {
	httpClient *http.Client
	resolver   *discovery.Resolver
}

// NewAgentManagerClient creates a new agent-manager client with the given timeout.
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *AgentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.resolver != nil {
		return c.resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
	}
	return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
}

// ListProfiles calls GET /api/v1/profiles on agent-manager.
func (c *AgentManagerClient) ListProfiles(ctx context.Context) (*AgentProfileListResponse, error) {
	var result AgentProfileListResponse
	if err := c.doGet(ctx, "/api/v1/profiles", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask calls POST /api/v1/tasks on agent-manager.
func (c *AgentManagerClient) CreateTask(ctx context.Context, req agentTaskCreateRequest) (*agentTaskCreateResponse, error) {
	var result agentTaskCreateResponse
	if err := c.doJSON(ctx, "/api/v1/tasks", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateRun calls POST /api/v1/runs on agent-manager.
func (c *AgentManagerClient) CreateRun(ctx context.Context, req agentRunCreateInternalRequest) (*agentRunCreateInternalResponse, error) {
	var result agentRunCreateInternalResponse
	if err := c.doJSON(ctx, "/api/v1/runs", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRun calls GET /api/v1/runs/{id} on agent-manager.
func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*AgentRun, error) {
	var result AgentRun
	if err := c.doGet(ctx, "/api/v1/runs/"+runID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRunEvents calls GET /api/v1/runs/{id}/events on agent-manager.
func (c *AgentManagerClient) GetRunEvents(ctx context.Context, runID string, afterSequence, limit int) (*AgentRunEventsResponse, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events?afterSequence=%d&limit=%d", runID, afterSequence, limit)
	var result AgentRunEventsResponse
	if err := c.doGet(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRunDiff calls GET /api/v1/runs/{id}/diff on agent-manager.
func (c *AgentManagerClient) GetRunDiff(ctx context.Context, runID string) (*AgentRunDiffResponse, error) {
	var result AgentRunDiffResponse
	if err := c.doGet(ctx, "/api/v1/runs/"+runID+"/diff", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ContinueRun calls POST /api/v1/runs/{id}/continue on agent-manager.
func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, req AgentContinueRequest) (*AgentRun, error) {
	var result AgentRun
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/continue", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveRun calls POST /api/v1/runs/{id}/approve on agent-manager.
func (c *AgentManagerClient) ApproveRun(ctx context.Context, runID string, req AgentApproveRequest) (*AgentRun, error) {
	var result AgentRun
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/approve", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RejectRun calls POST /api/v1/runs/{id}/reject on agent-manager.
func (c *AgentManagerClient) RejectRun(ctx context.Context, runID string, req AgentRejectRequest) (*AgentRun, error) {
	var result AgentRun
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/reject", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StopRun calls POST /api/v1/runs/{id}/stop on agent-manager.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) (*AgentRun, error) {
	var result AgentRun
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/stop", struct{}{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type ensureProfileDefaults struct {
	Name                 string `json:"name"`
	ProfileKey           string `json:"profileKey"`
	Description          string `json:"description,omitempty"`
	RunnerType           int    `json:"runnerType,omitempty"`
	MaxTurns             int    `json:"maxTurns,omitempty"`
	SkipPermissionPrompt bool   `json:"skipPermissionPrompt,omitempty"`
}

type ensureProfileRequest struct {
	ProfileKey     string                 `json:"profileKey"`
	Defaults       *ensureProfileDefaults `json:"defaults,omitempty"`
	UpdateExisting bool                   `json:"updateExisting"`
}

type ensureProfileResponse struct {
	Profile *AgentProfile `json:"profile"`
	Created bool          `json:"created"`
	Updated bool          `json:"updated"`
}

// EnsureDefaultProfile creates (or confirms existence of) the default
// git-control-tower-reviewer profile in agent-manager.
func (c *AgentManagerClient) EnsureDefaultProfile(ctx context.Context) (*ensureProfileResponse, error) {
	req := ensureProfileRequest{
		ProfileKey: "git-control-tower-reviewer",
		Defaults: &ensureProfileDefaults{
			Name:                 "Git Control Tower Reviewer",
			ProfileKey:           "git-control-tower-reviewer",
			Description:          "Default profile for git-control-tower scenario reviews",
			RunnerType:           1, // RUNNER_TYPE_CLAUDE_CODE
			MaxTurns:             50,
			SkipPermissionPrompt: true,
		},
		UpdateExisting: false,
	}
	var result ensureProfileResponse
	if err := c.doJSON(ctx, "/api/v1/profiles/ensure", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRuns calls GET /api/v1/runs on agent-manager.
func (c *AgentManagerClient) ListRuns(ctx context.Context, scopePrefix string, limit int) (*AgentRunListResponse, error) {
	path := fmt.Sprintf("/api/v1/runs?scopePrefix=%s&limit=%d", scopePrefix, limit)
	var result AgentRunListResponse
	if err := c.doGet(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *AgentManagerClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve agent-manager url: %w", err)
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return parseAgentManagerError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *AgentManagerClient) doGet(ctx context.Context, path string, result interface{}) error {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve agent-manager url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseAgentManagerError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseAgentManagerError(resp *http.Response) error {
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
