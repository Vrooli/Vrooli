// Package agentmanager provides an HTTP client for the agent-manager service.
//
// This package wraps the agent-manager REST API using proto-generated types,
// enabling system-monitor to delegate agent execution to agent-manager
// instead of invoking CLI tools directly.
package agentmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
			UseProtoNames: true, // snake_case for JSON
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

// CreateProfile creates a new agent profile.
func (c *Client) CreateProfile(ctx context.Context, profile *domainpb.AgentProfile) (*domainpb.AgentProfile, error) {
	req := &apipb.CreateProfileRequest{Profile: profile}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/profiles", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result apipb.CreateProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Profile, nil
}

// GetProfile retrieves a profile by ID.
func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/profiles/%s", profileID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Profile not found
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

// ListProfiles returns all profiles, optionally filtered by runner type.
func (c *Client) ListProfiles(ctx context.Context, runnerType *domainpb.RunnerType) ([]*domainpb.AgentProfile, error) {
	path := "/api/v1/profiles"
	if runnerType != nil {
		path = fmt.Sprintf("%s?runner_type=%d", path, *runnerType)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.ListProfilesResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Profiles, nil
}

// UpdateProfile updates an existing profile.
func (c *Client) UpdateProfile(ctx context.Context, profileID string, profile *domainpb.AgentProfile) (*domainpb.AgentProfile, error) {
	req := &apipb.UpdateProfileRequest{ProfileId: profileID, Profile: profile}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/api/v1/profiles/%s", profileID), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result apipb.UpdateProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Profile, nil
}

// UpsertProfile creates or updates a profile by name.
// Returns the profile (created or existing) and whether it was created.
func (c *Client) UpsertProfile(ctx context.Context, profile *domainpb.AgentProfile) (*domainpb.AgentProfile, bool, error) {
	// List all profiles and find by name
	profiles, err := c.ListProfiles(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("list profiles: %w", err)
	}

	for _, p := range profiles {
		if p.Name == profile.Name {
			// Update existing profile
			profile.Id = p.Id
			updated, err := c.UpdateProfile(ctx, p.Id, profile)
			if err != nil {
				return nil, false, fmt.Errorf("update profile: %w", err)
			}
			return updated, false, nil
		}
	}

	// Create new profile
	created, err := c.CreateProfile(ctx, profile)
	if err != nil {
		return nil, false, fmt.Errorf("create profile: %w", err)
	}
	return created, true, nil
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

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*domainpb.Task, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/tasks/%s", taskID), nil)
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

	var result apipb.GetTaskResponse
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

// GetRunByTag retrieves a run by its custom tag.
func (c *Client) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/runs/tag/%s", tag), nil)
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

	var result apipb.GetRunByTagResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// ListRuns returns runs with optional filtering.
func (c *Client) ListRuns(ctx context.Context, status *domainpb.RunStatus, tagPrefix string) ([]*domainpb.Run, error) {
	path := "/api/v1/runs?"
	params := []string{}
	if status != nil {
		params = append(params, fmt.Sprintf("status=%d", *status))
	}
	if tagPrefix != "" {
		params = append(params, fmt.Sprintf("tag_prefix=%s", tagPrefix))
	}
	for i, p := range params {
		if i > 0 {
			path += "&"
		}
		path += p
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

// StopRunByTag stops a run identified by its tag.
func (c *Client) StopRunByTag(ctx context.Context, tag string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/runs/tag/%s/stop", tag), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// StopAllRuns stops all runs with an optional tag prefix filter.
func (c *Client) StopAllRuns(ctx context.Context, tagPrefix string) (int, error) {
	req := &apipb.StopAllRunsRequest{}
	if tagPrefix != "" {
		req.TagPrefix = &tagPrefix
	}
	body, err := c.jsonOpts.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/runs/stop-all", body)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, c.parseError(resp)
	}

	var result apipb.StopAllRunsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return 0, err
	}
	if result.Result != nil {
		return int(result.Result.StoppedCount), nil
	}
	return 0, nil
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

// GetRunnerStatus returns status of all runners.
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

// GetAvailableRunners returns only available runners with their supported models.
func (c *Client) GetAvailableRunners(ctx context.Context) ([]*domainpb.RunnerStatus, error) {
	runners, err := c.GetRunnerStatus(ctx)
	if err != nil {
		return nil, err
	}

	available := make([]*domainpb.RunnerStatus, 0)
	for _, r := range runners {
		if r.Available {
			available = append(available, r)
		}
	}
	return available, nil
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

			// Check if terminal state
			switch run.Status {
			case domainpb.RunStatus_RUN_STATUS_COMPLETE,
				domainpb.RunStatus_RUN_STATUS_FAILED,
				domainpb.RunStatus_RUN_STATUS_CANCELLED:
				return run, nil
			}
		}
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
