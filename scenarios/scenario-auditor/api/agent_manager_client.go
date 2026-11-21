package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type agentManagerClient struct {
	httpClient      *http.Client
	baseURLResolver func(context.Context) (string, error)
}

func newAgentManagerClient(timeout time.Duration) *agentManagerClient {
	return &agentManagerClient{httpClient: &http.Client{Timeout: timeout}}
}

func (c *agentManagerClient) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
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

func (c *agentManagerClient) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
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

func (c *agentManagerClient) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
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

func (c *agentManagerClient) GetTask(ctx context.Context, taskID string) (*domainpb.Task, error) {
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

func (c *agentManagerClient) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
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
	return resp.Msg.GetProfile(), nil
}

func (c *agentManagerClient) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
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

func (c *agentManagerClient) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
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

func (c *agentManagerClient) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
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

func (c *agentManagerClient) StopRunByTag(ctx context.Context, tag string) error {
	client, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.StopRunByTag(ctx, connect.NewRequest(&apipb.StopRunByTagRequest{Tag: tag}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("agent-manager stop run by tag: %w", err)
	}
	return nil
}

func (c *agentManagerClient) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*domainpb.Run, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		run, err := c.GetRun(ctx, runID)
		if err != nil {
			return nil, err
		}
		if run == nil {
			return nil, fmt.Errorf("agent-manager run %s not found", runID)
		}
		if isTerminalRunStatus(run.Status) {
			return run, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (c *agentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.baseURLResolver != nil {
		return c.baseURLResolver(ctx)
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return baseURL, nil
}

func (c *agentManagerClient) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	return apiconnect.NewAgentManagerServiceClient(c.httpClient, strings.TrimRight(baseURL, "/")), nil
}
