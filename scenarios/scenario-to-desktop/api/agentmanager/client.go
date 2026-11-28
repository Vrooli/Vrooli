// Package agentmanager provides the scenario-to-desktop Agent Manager adapter.
package agentmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	shared "github.com/vrooli/vrooli/packages/agentmanager-client-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type URLResolver = shared.URLResolver

// Client shares Agent Manager discovery and generated Connect transport while
// keeping scenario-to-desktop's approval and lifecycle policy at this boundary.
type Client struct{ shared *shared.Client }

func NewClient(timeout time.Duration) *Client {
	return &Client{shared: shared.New(timeout, shared.WithProtoJSON())}
}

func NewClientWithResolver(timeout time.Duration, resolver URLResolver) *Client {
	return &Client{shared: shared.NewWithResolver(timeout, resolver, shared.WithProtoJSON())}
}

func (c *Client) connectClient(ctx context.Context) (shared.Service, error) {
	if c == nil || c.shared == nil {
		return nil, fmt.Errorf("agent-manager client is not configured")
	}
	return c.shared.Connect(ctx)
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
	return s.ReconcileScenarioProfiles(ctx, scenario)
}

func (c *Client) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetProfile(ctx, profileID)
}

func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.CreateTask(ctx, task)
	if err != nil {
		return nil, approvalPending(err)
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
		return nil, approvalPending(err)
	}
	return result, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetRun(ctx, runID)
}

func (c *Client) StopRun(ctx context.Context, runID string) error {
	s, err := c.connectClient(ctx)
	if err != nil {
		return err
	}
	return s.StopRun(ctx, runID)
}

func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	s, err := c.connectClient(ctx)
	if err != nil {
		return nil, err
	}
	return s.GetRunEvents(ctx, runID, afterSequence)
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
	if c == nil || c.shared == nil {
		return "", fmt.Errorf("agent-manager client is not configured")
	}
	return c.shared.ResolveURL(ctx)
}
