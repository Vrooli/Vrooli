// Package agentmanager adapts the shared Agent Manager transport to test-genie's API.
package agentmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	agentmanagerclient "github.com/vrooli/vrooli/packages/agentmanager-client-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Client preserves test-genie's operation surface while sharing transport and
// generated procedure construction with other scenarios.
type Client struct{ shared *agentmanagerclient.Client }

type resolverFunc func(context.Context) (string, error)

func (f resolverFunc) ResolveBaseURL(ctx context.Context) (string, error) { return f(ctx) }

// NewClient creates a client using standard Agent Manager scenario discovery.
func NewClient(timeout time.Duration) *Client {
	return &Client{shared: agentmanagerclient.New(timeout, agentmanagerclient.WithProto())}
}

// NewClientWithResolver injects target resolution for transport journeys.
func NewClientWithResolver(timeout time.Duration, resolver func(context.Context) (string, error)) *Client {
	return &Client{shared: agentmanagerclient.NewWithResolver(timeout, resolverFunc(resolver), agentmanagerclient.WithProto())}
}

func (c *Client) connectClient(ctx context.Context) (agentmanagerclient.Service, error) {
	if c == nil || c.shared == nil {
		return nil, fmt.Errorf("agent-manager client is not configured")
	}
	return c.shared.Connect(ctx)
}

func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	if c == nil || c.shared == nil {
		return "", fmt.Errorf("agent-manager client is not configured")
	}
	return c.shared.ResolveURL(ctx)
}

func (c *Client) Health(ctx context.Context) (bool, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return false, err
	}
	return s.Health(ctx)
}

func (c *Client) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.ReconcileScenarioProfiles(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("agent-manager reconcile scenario profiles: %w", err)
	}
	return result, nil
}

func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get profile: %w", err)
	}
	return result, nil
}

func (c *Client) GetProfileResponse(ctx context.Context, profileID string) (*apipb.GetProfileResponse, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetProfileResponse(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get profile: %w", err)
	}
	return result, nil
}

func (c *Client) GetRolePolicyCatalog(ctx context.Context) (*apipb.GetRolePolicyCatalogResponse, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetRolePolicyCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get role policy catalog: %w", err)
	}
	return result, nil
}

func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.CreateTask(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("agent-manager create task: %w", err)
	}
	return result, nil
}

func (c *Client) CreateRun(ctx context.Context, request *apipb.CreateRunRequest) (*domainpb.Run, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.CreateRun(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("agent-manager create run: %w", err)
	}
	return result, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run: %w", err)
	}
	return result, nil
}

func (c *Client) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetRunByTag(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run by tag: %w", err)
	}
	return result, nil
}

// ListRunsOptions contains optional filters for listing runs.
type ListRunsOptions = agentmanagerclient.ListRunsOptions

func (c *Client) ListRuns(ctx context.Context, options ListRunsOptions) ([]*domainpb.Run, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.ListRuns(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("agent-manager list runs: %w", err)
	}
	return result, nil
}

func (c *Client) StopRun(ctx context.Context, runID string) error {
	s, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	if err := s.StopRun(ctx, runID); err != nil {
		return fmt.Errorf("agent-manager stop run: %w", err)
	}
	return nil
}

func (c *Client) StopRunByTag(ctx context.Context, tag string) error {
	s, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	if err := s.StopRunByTag(ctx, tag); err != nil {
		return fmt.Errorf("agent-manager stop run by tag: %w", err)
	}
	return nil
}

type StopAllRunsResult struct {
	StoppedCount int
	FailedCount  int
	Errors       []string
}

func (c *Client) StopAllRuns(ctx context.Context, tagPrefix string) (*StopAllRunsResult, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.StopAllRuns(ctx, tagPrefix)
	if err != nil {
		return nil, fmt.Errorf("agent-manager stop all runs: %w", err)
	}
	result := &StopAllRunsResult{}
	if response.GetResult() != nil {
		result.StoppedCount = int(response.Result.GetStoppedCount())
		for _, failure := range response.Result.GetFailures() {
			result.Errors = append(result.Errors, failure.GetError())
		}
		result.FailedCount = len(response.Result.GetFailures())
	}
	return result, nil
}

func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetRunEvents(ctx, runID, afterSequence)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run events: %w", err)
	}
	return result, nil
}

func (c *Client) GetRunnerStatus(ctx context.Context) ([]*domainpb.RunnerStatus, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.GetRunnerStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get runner status: %w", err)
	}
	return result, nil
}

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

func IsTerminalStatus(status domainpb.RunStatus) bool {
	switch status {
	case domainpb.RunStatus_RUN_STATUS_COMPLETE, domainpb.RunStatus_RUN_STATUS_FAILED, domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

func NewUUID() string { return uuid.New().String() }
