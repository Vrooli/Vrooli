package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// AgentManagerClient is a lightweight HTTP client for agent-manager APIs.
type AgentManagerClient struct {
	httpClient     *http.Client
	resolver       *discovery.Resolver
	maxRetries     int
	retryBaseDelay time.Duration

	mu            sync.Mutex
	cachedBaseURL string
}

// NewAgentManagerClient creates a new agent-manager client with the given timeout.
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		httpClient:     &http.Client{Timeout: timeout},
		maxRetries:     2,
		retryBaseDelay: 200 * time.Millisecond,
	}
}

func (c *AgentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	c.mu.Lock()
	cached := c.cachedBaseURL
	c.mu.Unlock()
	if cached != "" {
		return cached, nil
	}

	var url string
	var err error
	if c.resolver != nil {
		url, err = c.resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
	} else {
		url, err = discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	}
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.cachedBaseURL = url
	c.mu.Unlock()
	return url, nil
}

func (c *AgentManagerClient) clearCachedURL() {
	c.mu.Lock()
	c.cachedBaseURL = ""
	c.mu.Unlock()
}

// isRetryable returns true for transport-level errors and gateway status codes.
func isRetryable(err error, statusCode int) bool {
	if err != nil {
		// Connection refused, timeout, DNS errors.
		if _, ok := err.(net.Error); ok {
			return true
		}
		return true // Any transport error is retryable.
	}
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryDelay computes exponential backoff capped at 2s.
func (c *AgentManagerClient) retryDelay(attempt int) time.Duration {
	d := c.retryBaseDelay
	for i := 0; i < attempt; i++ {
		d *= 2
	}
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	return d
}

// ListProfiles calls GET /api/v1/profiles on agent-manager.
func (c *AgentManagerClient) ListProfiles(ctx context.Context) (*wireListProfilesResponse, error) {
	var result wireListProfilesResponse
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
func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*wireGetRunResponse, error) {
	var result wireGetRunResponse
	if err := c.doGet(ctx, "/api/v1/runs/"+runID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRunEvents calls GET /api/v1/runs/{id}/events on agent-manager.
func (c *AgentManagerClient) GetRunEvents(ctx context.Context, runID string, afterSequence, limit int) (*wireGetRunEventsResponse, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events?afterSequence=%d&limit=%d", runID, afterSequence, limit)
	var result wireGetRunEventsResponse
	if err := c.doGet(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRunDiff calls GET /api/v1/runs/{id}/diff on agent-manager.
func (c *AgentManagerClient) GetRunDiff(ctx context.Context, runID string) (*wireGetRunDiffResponse, error) {
	var result wireGetRunDiffResponse
	if err := c.doGet(ctx, "/api/v1/runs/"+runID+"/diff", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ContinueRun calls POST /api/v1/runs/{id}/continue on agent-manager.
func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, req AgentContinueRequest) (*wireContinueRunResponse, error) {
	var result wireContinueRunResponse
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/continue", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApproveRun calls POST /api/v1/runs/{id}/approve on agent-manager.
func (c *AgentManagerClient) ApproveRun(ctx context.Context, runID string, req AgentApproveRequest) (*wireApproveRunResponse, error) {
	var result wireApproveRunResponse
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/approve", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// RejectRun calls POST /api/v1/runs/{id}/reject on agent-manager.
func (c *AgentManagerClient) RejectRun(ctx context.Context, runID string, req AgentRejectRequest) (*wireRejectRunResponse, error) {
	var result wireRejectRunResponse
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/reject", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StopRun calls POST /api/v1/runs/{id}/stop on agent-manager.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) (*wireStopRunResponse, error) {
	var result wireStopRunResponse
	if err := c.doJSON(ctx, "/api/v1/runs/"+runID+"/stop", struct{}{}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type ensureProfileDefaults struct {
	Name                 string `json:"name"`
	ProfileKey           string `json:"profile_key"`
	Description          string `json:"description,omitempty"`
	RunnerType           int    `json:"runner_type,omitempty"`
	MaxTurns             int    `json:"max_turns,omitempty"`
	SkipPermissionPrompt bool   `json:"skip_permission_prompt,omitempty"`
}

type ensureProfileRequest struct {
	ProfileKey     string                 `json:"profile_key"`
	Defaults       *ensureProfileDefaults `json:"defaults,omitempty"`
	UpdateExisting bool                   `json:"update_existing"`
}

// EnsureDefaultProfile creates (or confirms existence of) the default
// git-control-tower-reviewer profile in agent-manager.
func (c *AgentManagerClient) EnsureDefaultProfile(ctx context.Context) (*wireEnsureProfileResponse, error) {
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
	var result wireEnsureProfileResponse
	if err := c.doJSON(ctx, "/api/v1/profiles/ensure", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListRuns calls GET /api/v1/runs on agent-manager.
func (c *AgentManagerClient) ListRuns(ctx context.Context, tagPrefix string, limit int) (*wireListRunsResponse, error) {
	path := fmt.Sprintf("/api/v1/runs?tag_prefix=%s&limit=%d", tagPrefix, limit)
	var result wireListRunsResponse
	if err := c.doGet(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *AgentManagerClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay(attempt - 1)):
			}
		}

		baseURL, err := c.resolveBaseURL(ctx)
		if err != nil {
			lastErr = fmt.Errorf("resolve agent-manager url: %w", err)
			c.clearCachedURL()
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("agent-manager request failed: %w", err)
			c.clearCachedURL()
			if isRetryable(err, 0) {
				continue
			}
			return lastErr
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			amErr := parseAgentManagerError(resp)
			resp.Body.Close()
			if isRetryable(nil, resp.StatusCode) {
				lastErr = amErr
				c.clearCachedURL()
				continue
			}
			return amErr
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(result)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		return nil
	}
	return lastErr
}

func (c *AgentManagerClient) doGet(ctx context.Context, path string, result interface{}) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay(attempt - 1)):
			}
		}

		baseURL, err := c.resolveBaseURL(ctx)
		if err != nil {
			lastErr = fmt.Errorf("resolve agent-manager url: %w", err)
			c.clearCachedURL()
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("agent-manager request failed: %w", err)
			c.clearCachedURL()
			if isRetryable(err, 0) {
				continue
			}
			return lastErr
		}

		if resp.StatusCode != http.StatusOK {
			amErr := parseAgentManagerError(resp)
			resp.Body.Close()
			if isRetryable(nil, resp.StatusCode) {
				lastErr = amErr
				c.clearCachedURL()
				continue
			}
			return amErr
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(result)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		return nil
	}
	return lastErr
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
