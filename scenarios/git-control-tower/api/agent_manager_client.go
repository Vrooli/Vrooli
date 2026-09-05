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
)

// AgentManagerClient is a lightweight HTTP client for agent-manager APIs.
type AgentManagerClient struct {
	BaseClient
	maxRetries     int
	retryBaseDelay time.Duration

	mu            sync.Mutex
	cachedBaseURL string
}

// NewAgentManagerClient creates a new agent-manager client with the given timeout.
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		BaseClient:     NewBaseClient("agent-manager", timeout),
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

	url, err := c.BaseClient.resolveBaseURL(ctx)
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
		if _, ok := err.(net.Error); ok {
			return true
		}
		return true
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

// ReconcileProfiles applies Git Control Tower's manifest-declared profile source.
func (c *AgentManagerClient) ReconcileProfiles(ctx context.Context) error {
	var result struct {
		Results []struct {
			ProfileKey string `json:"profile_key"`
			ProfileID  string `json:"profile_id"`
		} `json:"results"`
	}
	if err := c.doJSON(ctx, "/api/v1/profiles/reconcile-scenario", map[string]string{"scenario": "git-control-tower"}, &result); err != nil {
		return err
	}
	for _, item := range result.Results {
		if item.ProfileKey == "git-control-tower/reviewer" && item.ProfileID != "" {
			return nil
		}
	}
	return fmt.Errorf("profile reconciliation returned no git-control-tower/reviewer profile")
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

// UploadAttachment streams a multipart upload to agent-manager's /api/v1/attachments/upload.
func (c *AgentManagerClient) UploadAttachment(ctx context.Context, body io.Reader, contentType string) (*wireUploadAttachmentResponse, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/attachments/upload", body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.clearCachedURL()
		return nil, fmt.Errorf("agent-manager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result wireUploadAttachmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *AgentManagerClient) doJSON(ctx context.Context, path string, body, result interface{}) error {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	return c.doWithRetry(ctx, func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return c.httpClient.Do(req)
	}, result, func(code int) bool {
		return code == http.StatusOK || code == http.StatusCreated
	})
}

func (c *AgentManagerClient) doGet(ctx context.Context, path string, result interface{}) error {
	return c.doWithRetry(ctx, func(baseURL string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		return c.httpClient.Do(req)
	}, result, func(code int) bool {
		return code == http.StatusOK
	})
}

// doWithRetry executes an HTTP request with retry logic.
// makeReq builds the request given a resolved base URL.
// isSuccess checks whether a status code indicates success.
func (c *AgentManagerClient) doWithRetry(
	ctx context.Context,
	makeReq func(baseURL string) (*http.Response, error),
	result interface{},
	isSuccess func(int) bool,
) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := c.waitForRetry(ctx, attempt); err != nil {
			return err
		}

		baseURL, err := c.resolveBaseURL(ctx)
		if err != nil {
			lastErr = fmt.Errorf("resolve agent-manager url: %w", err)
			c.clearCachedURL()
			continue
		}

		resp, err := makeReq(baseURL)
		if err != nil {
			lastErr = fmt.Errorf("agent-manager request failed: %w", err)
			c.clearCachedURL()
			if isRetryable(err, 0) {
				continue
			}
			return lastErr
		}

		if !isSuccess(resp.StatusCode) {
			amErr := c.parseError(resp)
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

// waitForRetry sleeps with backoff for retry attempts > 0, respecting context cancellation.
func (c *AgentManagerClient) waitForRetry(ctx context.Context, attempt int) error {
	if attempt == 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.retryDelay(attempt - 1)):
		return nil
	}
}
