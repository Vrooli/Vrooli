// Package agentmanager provides the typed Agent Manager integration.
//
// Implemented proto operations use the generated Connect client. Legacy REST
// calls remain only for operations with no available Connect procedure and
// are tracked in the interoperability migration inventory.
// DOC: docs/internal/SEAMS.md#agent-manager-client
package agentmanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/owneridentity"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Client is an HTTP client for the agent-manager API.
type Client struct {
	baseURLResolver func(context.Context) (string, error)
	httpClient      *http.Client
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return false, fmt.Errorf("create agent-manager health request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
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
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.EnsureProfile(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("agent-manager ensure profile: %w", err)
	}
	return resp.Msg, nil
}

// ReconcileScenarioProfiles reconciles profile files declared by the calling scenario.
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

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*domainpb.Task, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetTask(ctx, connect.NewRequest(&apipb.GetTaskRequest{TaskId: taskID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent-manager get task: %w", err)
	}
	return resp.Msg.GetTask(), nil
}

// =============================================================================
// RUNS
// =============================================================================

// CreateRun starts a new run for a task.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	if err := owneridentity.RequireCreateRunCaller(ctx, time.Now(), true); err != nil {
		return nil, err
	}
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	redirectSafeClient := *c.httpClient
	redirectSafeClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	client := apiconnect.NewAgentManagerServiceClient(&redirectSafeClient, strings.TrimRight(baseURL, "/"))
	request := connect.NewRequest(req)
	authorization, err := owneridentity.CreateRunAuthorization(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	request.Header().Set("Authorization", authorization)
	resp, err := client.CreateRun(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s", owneridentity.RedactCreateRunCaller(ctx, err.Error()))
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

// GetRunByTag retrieves a run by its custom tag.
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

// ListRuns returns runs with optional filtering.
func (c *Client) ListRuns(ctx context.Context, status *domainpb.RunStatus, tagPrefix string) ([]*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	request := &apipb.ListRunsRequest{}
	if status != nil {
		request.Status = status
	}
	if tagPrefix != "" {
		request.TagPrefix = &tagPrefix
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

// StopRunByTag stops a run identified by its tag.
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

// StopAllRuns stops all runs with an optional tag prefix filter.
func (c *Client) StopAllRuns(ctx context.Context, tagPrefix string) (int, error) {
	req := &apipb.StopAllRunsRequest{}
	if tagPrefix != "" {
		req.TagPrefix = &tagPrefix
	}
	client, err := c.connectClient(ctx)
	if err != nil {
		return 0, err
	}
	resp, err := client.StopAllRuns(ctx, connect.NewRequest(req))
	if err != nil {
		return 0, fmt.Errorf("agent-manager stop all runs: %w", err)
	}
	if resp.Msg.GetResult() != nil {
		return int(resp.Msg.GetResult().GetStoppedCount()), nil
	}
	return 0, nil
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

// ResolveURL exposes the computed agent-manager base URL.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	return c.resolveBaseURL(ctx)
}

