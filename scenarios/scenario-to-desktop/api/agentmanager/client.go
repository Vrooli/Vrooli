// Package agentmanager provides the scenario-to-desktop Agent Manager adapter.
package agentmanager

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// URLResolver resolves the Agent Manager base URL.
type URLResolver interface {
	ResolveBaseURL(context.Context) (string, error)
}

type discoveryResolver struct{}

func (*discoveryResolver) ResolveBaseURL(ctx context.Context) (string, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return baseURL, nil
}

// Client calls Agent Manager procedures through its generated Connect client.
type Client struct {
	httpClient  *http.Client
	urlResolver URLResolver
}

func NewClient(timeout time.Duration) *Client {
	return &Client{httpClient: &http.Client{Timeout: timeout}, urlResolver: &discoveryResolver{}}
}

func NewClientWithResolver(timeout time.Duration, resolver URLResolver) *Client {
	c := NewClient(timeout)
	c.urlResolver = resolver
	return c
}

func (c *Client) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	baseURL, err := c.urlResolver.ResolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}
	return apiconnect.NewAgentManagerServiceClient(c.httpClient, baseURL, connect.WithProtoJSON()), nil
}

func (c *Client) Health(ctx context.Context) (bool, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return false, err
	}
	response, err := client.Health(ctx, connect.NewRequest(&apipb.HealthRequest{}))
	if err != nil {
		return false, err
	}
	return response.Msg.GetReadiness(), nil
}

func (c *Client) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ReconcileScenarioProfiles(ctx, connect.NewRequest(&apipb.ReconcileScenarioProfilesRequest{Scenario: scenario}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetProfile(ctx, connect.NewRequest(&apipb.GetProfileRequest{ProfileId: profileID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetProfile(), nil
}

func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.CreateTask(ctx, connect.NewRequest(&apipb.CreateTaskRequest{Task: task}))
	if err != nil {
		return nil, approvalPending(err)
	}
	return response.Msg.GetTask(), nil
}

func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.CreateRun(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, approvalPending(err)
	}
	return response.Msg.GetRun(), nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetRun(ctx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRun(), nil
}

func (c *Client) StopRun(ctx context.Context, runID string) error {
	client, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.StopRun(ctx, connect.NewRequest(&apipb.StopRunRequest{RunId: runID}))
	return err
}

func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	req := &apipb.GetRunEventsRequest{RunId: runID}
	if afterSequence > 0 {
		req.AfterSequence = &afterSequence
	}
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetRunEvents(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetEvents(), nil
}

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
			switch run.Status {
			case domainpb.RunStatus_RUN_STATUS_COMPLETE, domainpb.RunStatus_RUN_STATUS_FAILED, domainpb.RunStatus_RUN_STATUS_CANCELLED:
				return run, nil
			}
		}
	}
}

func NewUUID() string { return uuid.New().String() }

func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	return c.urlResolver.ResolveBaseURL(ctx)
}
