// Package agentmanager provides a Connect client for the agent-manager service.
package agentmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/owneridentity"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// ErrAuthorizationApprovalPending means Agent Manager definitively did not
// start a mutation because operator approval is required.
var ErrAuthorizationApprovalPending = errors.New("agent-manager authorization approval pending")

// Client is a generated Connect client for the agent-manager API.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new agent-manager client.
func NewClient(timeout time.Duration) *Client { return NewClientWithBaseURL(timeout, "") }

// NewClientWithBaseURL creates a new agent-manager client with a fixed base URL.
func NewClientWithBaseURL(timeout time.Duration, baseURL string) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{httpClient: &http.Client{Timeout: timeout}, baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/")}
}

func (c *Client) rpc(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	return c.rpcWithHTTPClient(ctx, c.httpClient)
}

func (c *Client) rpcWithHTTPClient(ctx context.Context, httpClient *http.Client) (apiconnect.AgentManagerServiceClient, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		var err error
		baseURL, err = discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
		if err != nil {
			return nil, fmt.Errorf("resolve agent-manager url: %w", err)
		}
	}
	return apiconnect.NewAgentManagerServiceClient(httpClient, baseURL), nil
}

func missingIsNil[T any](resp *connect.Response[T], err error, op string) (*connect.Response[T], error) {
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agent-manager %s: %w", op, err)
	}
	return resp, nil
}

func approvalPending(err error) error {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return nil
	}
	for _, detail := range connectErr.Details() {
		if detail.Type() != "agent_manager.v1.AuthorizationApprovalPending" {
			continue
		}
		msg, decodeErr := detail.Value()
		if decodeErr != nil {
			return fmt.Errorf("%w: approval detail could not be decoded", ErrAuthorizationApprovalPending)
		}
		if _, ok := msg.(*apipb.AuthorizationApprovalPending); !ok {
			return fmt.Errorf("%w: approval detail had an unexpected type", ErrAuthorizationApprovalPending)
		}
		return fmt.Errorf("%w: operator approval is required; the operation was not started. Retry after approval", ErrAuthorizationApprovalPending)
	}
	return nil
}

// Health checks the agent-manager service health.
func (c *Client) Health(ctx context.Context) (bool, error) {
	rpc, err := c.rpc(ctx)
	if err != nil {
		return false, err
	}
	resp, err := rpc.Health(ctx, connect.NewRequest(&apipb.HealthRequest{}))
	if err != nil {
		return false, fmt.Errorf("agent-manager health: %w", err)
	}
	return resp.Msg.GetStatus() == commonpb.HealthStatus_HEALTH_STATUS_HEALTHY, nil
}

// ReconcileScenarioProfiles activates repository-owned profile sources.
func (c *Client) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.ReconcileScenarioProfiles(ctx, connect.NewRequest(&apipb.ReconcileScenarioProfilesRequest{Scenario: scenario}))
	if err != nil {
		return nil, fmt.Errorf("agent-manager reconcile scenario profiles: %w", err)
	}
	return resp.Msg, nil
}

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.CreateTask(ctx, connect.NewRequest(&apipb.CreateTaskRequest{Task: task}))
	if err != nil {
		if pending := approvalPending(err); pending != nil {
			return nil, pending
		}
		return nil, fmt.Errorf("agent-manager create task: %w", err)
	}
	return resp.Msg.GetTask(), nil
}

// CreateRun starts a new run for a task.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	if req == nil {
		return nil, fmt.Errorf("create run request is required")
	}
	if err := owneridentity.RequireCreateRunCaller(ctx, time.Now(), true); err != nil {
		return nil, err
	}
	auth, err := owneridentity.CreateRunAuthorization(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	rpc, err := c.rpcWithHTTPClient(ctx, &httpClient)
	if err != nil {
		return nil, err
	}
	request := connect.NewRequest(req)
	request.Header().Set("Authorization", auth)
	resp, err := rpc.CreateRun(ctx, request)
	if err != nil {
		if pending := approvalPending(err); pending != nil {
			return nil, pending
		}
		return nil, fmt.Errorf("%s", owneridentity.RedactCreateRunCaller(ctx, fmt.Sprintf("agent-manager create run: %v", err)))
	}
	return resp.Msg.GetRun(), nil
}

// GetRun retrieves a run by ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.GetRun(ctx, connect.NewRequest(&apipb.GetRunRequest{RunId: runID}))
	resp, err = missingIsNil(resp, err, "get run")
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.Msg.GetRun(), nil
}

// GetRunDiff retrieves the diff for a run.
func (c *Client) GetRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.GetRunDiff(ctx, connect.NewRequest(&apipb.GetRunDiffRequest{RunId: runID}))
	resp, err = missingIsNil(resp, err, "get run diff")
	if err != nil || resp == nil {
		return nil, err
	}
	return resp.Msg.GetDiff(), nil
}

// GetRunEvents retrieves events for a run.
func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	request := &apipb.GetRunEventsRequest{RunId: runID}
	if afterSequence > 0 {
		request.AfterSequence = &afterSequence
	}
	resp, err := rpc.GetRunEvents(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run events: %w", err)
	}
	return resp.Msg.GetEvents(), nil
}

// ApproveRun approves and applies changes from a sandboxed run.
func (c *Client) ApproveRun(ctx context.Context, req *apipb.ApproveRunRequest) (*apipb.ApproveRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.ApproveRun(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("agent-manager approve run: %w", err)
	}
	return resp.Msg, nil
}

// RejectRun rejects and discards changes from a sandboxed run.
func (c *Client) RejectRun(ctx context.Context, req *apipb.RejectRunRequest) (*apipb.RejectRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	rpc, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := rpc.RejectRun(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("agent-manager reject run: %w", err)
	}
	return resp.Msg, nil
}
