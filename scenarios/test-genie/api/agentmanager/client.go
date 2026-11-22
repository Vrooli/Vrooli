// Package agentmanager provides an HTTP client for the agent-manager service.
//
// This package wraps the agent-manager REST API using proto-generated types,
// enabling test-genie to delegate agent execution to agent-manager for test
// generation tasks.
package agentmanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Client is an HTTP client for the agent-manager API.
type Client struct {
	httpClient      *http.Client
	baseURLResolver func(context.Context) (string, error)
}

// NewClient creates a new agent-manager client.
func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Health checks the agent-manager service health.
func (c *Client) Health(ctx context.Context) (bool, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return false, fmt.Errorf("create agent-manager health request: %w", err)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (c *Client) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ReconcileScenarioProfiles(ctx, connect.NewRequest(&apipb.ReconcileScenarioProfilesRequest{Scenario: scenario}))
	if err != nil {
		return nil, fmt.Errorf("agent-manager reconcile scenario profiles: %w", err)
	}
	return resp.Msg, nil
}

// =============================================================================
// PROFILES
// =============================================================================

// GetProfile retrieves a profile by ID.
func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	resp, err := c.GetProfileResponse(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.Profile, nil
}

// GetProfileResponse retrieves a profile and any attached metadata.
func (c *Client) GetProfileResponse(ctx context.Context, profileID string) (*apipb.GetProfileResponse, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetProfile(ctx, connect.NewRequest(&apipb.GetProfileRequest{ProfileId: profileID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent-manager get profile: %w", err)
	}
	return resp.Msg, nil
}

// GetRolePolicyCatalog returns portable role choices. Concrete models remain
// resource-owned execution evidence and are never returned as user input.
func (c *Client) GetRolePolicyCatalog(ctx context.Context) (*apipb.GetRolePolicyCatalogResponse, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRolePolicyCatalog(ctx, connect.NewRequest(&apipb.GetRolePolicyCatalogRequest{}))
	if err != nil {
		return nil, fmt.Errorf("agent-manager get role policy catalog: %w", err)
	}
	return resp.Msg, nil
}

// =============================================================================
// TASKS
// =============================================================================

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.CreateTask(ctx, connect.NewRequest(&apipb.CreateTaskRequest{Task: task}))
	if err != nil {
		return nil, fmt.Errorf("agent-manager create task: %w", err)
	}
	return resp.Msg.GetTask(), nil
}

// =============================================================================
// RUNS
// =============================================================================

// CreateRun starts a new run for a task.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.CreateRun(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("agent-manager create run: %w", err)
	}
	return resp.Msg.GetRun(), nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRun(ctx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run: %w", err)
	}
	return resp.Msg.GetRun(), nil
}

// GetRunByTag retrieves a run by its tag.
func (c *Client) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRunByTag(ctx, connect.NewRequest(&apipb.GetRunByTagRequest{Tag: tag}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run by tag: %w", err)
	}
	return resp.Msg.GetRun(), nil
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
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	request := &apipb.ListRunsRequest{}
	if opts.TagPrefix != "" {
		request.TagPrefix = &opts.TagPrefix
	}
	if opts.ProfileID != "" {
		request.AgentProfileId = &opts.ProfileID
	}
	if opts.Status != nil {
		request.Status = opts.Status
	}
	if opts.Limit > 0 {
		limit := int32(opts.Limit)
		request.Limit = &limit
	}
	resp, err := client.ListRuns(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("agent-manager list runs: %w", err)
	}
	return resp.Msg.GetRuns(), nil
}

// StopRun stops a running run.
func (c *Client) StopRun(ctx context.Context, runID string) error {
	client, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	if _, err := client.StopRun(ctx, connect.NewRequest(&apipb.StopRunRequest{RunId: runID})); err != nil {
		return fmt.Errorf("agent-manager stop run: %w", err)
	}
	return nil
}

// StopRunByTag stops a run by its tag.
func (c *Client) StopRunByTag(ctx context.Context, tag string) error {
	client, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	if _, err := client.StopRunByTag(ctx, connect.NewRequest(&apipb.StopRunByTagRequest{Tag: tag})); err != nil {
		return fmt.Errorf("agent-manager stop run by tag: %w", err)
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
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resultResponse, err := client.StopAllRuns(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("agent-manager stop all runs: %w", err)
	}

	stopResult := &StopAllRunsResult{}
	if resultResponse.Msg.GetResult() != nil {
		stopResult.StoppedCount = int(resultResponse.Msg.Result.StoppedCount)
		for _, f := range resultResponse.Msg.Result.Failures {
			stopResult.Errors = append(stopResult.Errors, f.Error)
		}
		stopResult.FailedCount = len(resultResponse.Msg.Result.Failures)
	}
	return stopResult, nil
}

// =============================================================================
// RUN EVENTS
// =============================================================================

// GetRunEvents retrieves events for a run.
func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	request := &apipb.GetRunEventsRequest{RunId: runID}
	if afterSequence > 0 {
		request.AfterSequence = &afterSequence
	}
	resp, err := client.GetRunEvents(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run events: %w", err)
	}
	return resp.Msg.GetEvents(), nil
}

// =============================================================================
// RUNNERS
// =============================================================================

// GetRunnerStatus retrieves status of all runners.
func (c *Client) GetRunnerStatus(ctx context.Context) ([]*domainpb.RunnerStatus, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetRunnerStatus(ctx, connect.NewRequest(&apipb.GetRunnerStatusRequest{}))
	if err != nil {
		return nil, fmt.Errorf("agent-manager get runner status: %w", err)
	}
	return resp.Msg.GetRunners(), nil
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

func (c *Client) resolveBaseURL(ctx context.Context) (string, error) {
	if c.baseURLResolver != nil {
		return c.baseURLResolver(ctx)
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return url, nil
}

func (c *Client) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	return apiconnect.NewAgentManagerServiceClient(c.httpClient, strings.TrimRight(baseURL, "/")), nil
}

// ResolveURL exposes the computed agent-manager base URL.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	return c.resolveBaseURL(ctx)
}
