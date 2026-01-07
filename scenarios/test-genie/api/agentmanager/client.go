// Package agentmanager provides an HTTP client for the agent-manager service.
//
// This package wraps the agent-manager REST API using proto-generated types,
// enabling test-genie to delegate agent execution to agent-manager for test
// generation tasks.
package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Client is an HTTP client for the agent-manager API.
type Client struct {
	httpClient *http.Client
	jsonOpts   protojson.MarshalOptions
}

// NewClient creates a new agent-manager client.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		jsonOpts: protojson.MarshalOptions{
			UseProtoNames: false, // lowerCamelCase to match agent-manager HTTP handlers
		},
	}
}

// Health checks the agent-manager service health.
func (c *Client) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// =============================================================================
// PROFILES
// =============================================================================

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (c *Client) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	body, err := c.jsonOpts.Marshal(req)
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

	var result apipb.EnsureProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetProfile retrieves a profile by ID.
func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/profiles/%s", profileID), nil)
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

	var result apipb.GetProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Profile, nil
}

// =============================================================================
// TASKS
// =============================================================================

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	req := &apipb.CreateTaskRequest{Task: task}
	body, err := c.jsonOpts.Marshal(req)
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

	var result apipb.CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Task, nil
}

// =============================================================================
// RUNS
// =============================================================================

// CreateRun starts a new run for a task.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	body, err := c.jsonOpts.Marshal(req)
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

	var result apipb.CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
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

	var result apipb.GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRunByTag retrieves a run by its tag.
func (c *Client) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/runs/tag/%s", url.PathEscape(tag)), nil)
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

	var result apipb.GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// ListRunsOptions contains optional filters for listing runs.
type ListRunsOptions struct {
	TagPrefix string
	ProfileID string
	Status    *domainpb.RunStatus
	Limit     int
}

// ListRuns lists runs with optional filtering by tag prefix, profile ID, and status.
// Filters are passed as query parameters (agent-manager ignores JSON body on GET).
func (c *Client) ListRuns(ctx context.Context, opts ListRunsOptions) ([]*domainpb.Run, error) {
	// Build query parameters
	query := url.Values{}
	if opts.TagPrefix != "" {
		query.Set("tagPrefix", opts.TagPrefix)
	}
	if opts.ProfileID != "" {
		query.Set("profileId", opts.ProfileID)
	}
	if opts.Status != nil {
		query.Set("status", opts.Status.String())
	}
	if opts.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", opts.Limit))
	}

	path := "/api/v1/runs"
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.ListRunsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Runs, nil
}

// StopRun stops a running run.
func (c *Client) StopRun(ctx context.Context, runID string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/runs/%s/stop", runID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// StopRunByTag stops a run by its tag.
func (c *Client) StopRunByTag(ctx context.Context, tag string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/runs/tag/%s/stop", url.PathEscape(tag)), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// StopAllRunsResult contains the result of stopping multiple runs.
type StopAllRunsResult struct {
	StoppedCount int
	FailedCount  int
	Errors       []string
}

// StopAllRuns stops all runs matching an optional tag prefix.
func (c *Client) StopAllRuns(ctx context.Context, tagPrefix string) (*StopAllRunsResult, error) {
	req := &apipb.StopAllRunsRequest{}
	if tagPrefix != "" {
		req.TagPrefix = &tagPrefix
	}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/runs/stop-all", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.StopAllRunsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	stopResult := &StopAllRunsResult{}
	if result.Result != nil {
		stopResult.StoppedCount = int(result.Result.StoppedCount)
		for _, f := range result.Result.Failures {
			stopResult.Errors = append(stopResult.Errors, f.Error)
		}
		stopResult.FailedCount = len(result.Result.Failures)
	}
	return stopResult, nil
}

// =============================================================================
// RUN EVENTS
// =============================================================================

// GetRunEvents retrieves events for a run.
func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events", runID)
	if afterSequence > 0 {
		path = fmt.Sprintf("%s?after_sequence=%d", path, afterSequence)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunEventsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Events, nil
}

// =============================================================================
// RUNNERS
// =============================================================================

// GetRunnerStatus retrieves status of all runners.
func (c *Client) GetRunnerStatus(ctx context.Context) ([]*domainpb.RunnerStatus, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/runners", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.GetRunnerStatusResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Runners, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// WaitForRun polls until a run reaches a terminal state.
func (c *Client) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*domainpb.Run, error) {
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

			if IsTerminalStatus(run.Status) {
				return run, nil
			}
		}
	}
}

// IsTerminalStatus returns true if the run status is terminal (no longer running).
func IsTerminalStatus(status domainpb.RunStatus) bool {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE,
		domainpb.RunStatus_RUN_STATUS_FAILED,
		domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

// NewUUID generates a new UUID string.
func NewUUID() string {
	return uuid.New().String()
}

// =============================================================================
// INTERNAL
// =============================================================================

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
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

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return url, nil
}

// ResolveURL exposes the computed agent-manager base URL.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	return c.resolveBaseURL(ctx)
}

func (c *Client) parseResponse(resp *http.Response, msg proto.Message) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(body, msg); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *Client) parseError(resp *http.Response) error {
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
