// Package agentmanager provides the scenario-to-cloud Agent Manager adapter.
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

// Client calls Agent Manager through the canonical shared Connect client.
type Client struct{ remote *agentmanagerclient.Client }

// URLResolver allows lifecycle and transport tests to choose their own target.
type URLResolver = agentmanagerclient.URLResolver

func NewClient(timeout time.Duration) *Client {
	return &Client{remote: agentmanagerclient.New(timeout)}
}

// NewClientWithResolver injects target resolution for focused transport tests.
func NewClientWithResolver(timeout time.Duration, resolver URLResolver) *Client {
	return &Client{remote: agentmanagerclient.NewWithResolver(timeout, resolver)}
}

func (c *Client) connectClient(ctx context.Context) (agentmanagerclient.Service, error) {
	if c == nil || c.remote == nil {
		return nil, fmt.Errorf("agent manager client is not configured")
	}
	return c.remote.Connect(ctx)
}

// ResolveURL exposes the currently resolved Agent Manager base URL.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	if c == nil || c.remote == nil {
		return "", fmt.Errorf("agent manager client is not configured")
	}
	return c.remote.ResolveURL(ctx)
}

func (c *Client) Health(ctx context.Context) (bool, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return false, err
	}
	return client.Health(ctx)
}

func (c *Client) EnsureProfile(ctx context.Context, req *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.EnsureProfile(ctx, req)
}

func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetProfile(ctx, profileID)
}

func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	created, err := client.CreateTask(ctx, task)
	if err != nil {
		return nil, approvalPending(err)
	}
	return created, nil
}

func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	run, err := client.CreateRun(ctx, req)
	if err != nil {
		return nil, approvalPending(err)
	}
	return run, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetRun(ctx, runID)
}

func (c *Client) StopRun(ctx context.Context, runID string) error {
	client, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	return client.StopRun(ctx, runID)
}

func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	client, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.GetRunEvents(ctx, runID, afterSequence)
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
