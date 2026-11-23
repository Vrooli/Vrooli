// Package agentmanager provides the Knowledge Observatory Agent Manager adapter.
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
	shared "github.com/vrooli/vrooli/packages/agentmanager-client-go"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var ErrAuthorizationApprovalPending = errors.New("agent-manager authorization approval pending")

type baseURLResolver struct{ baseURL string }

func (r baseURLResolver) ResolveBaseURL(ctx context.Context) (string, error) {
	if r.baseURL != "" {
		return r.baseURL, nil
	}
	return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
}

// Client preserves Knowledge Observatory's validation and authorization policy
// while sharing discovery and generated Connect transport.
type Client struct {
	httpClient *http.Client
	remote     *shared.Client
}

func NewClient(timeout time.Duration) *Client { return NewClientWithBaseURL(timeout, "") }

func NewClientWithBaseURL(timeout time.Duration, baseURL string) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	httpClient := &http.Client{Timeout: timeout}
	return &Client{
		httpClient: httpClient,
		remote:     shared.NewWithResolver(timeout, baseURLResolver{baseURL: baseURL}, shared.WithHTTPClient(httpClient), shared.WithProto()),
	}
}

func (c *Client) rpc(ctx context.Context) (shared.Service, error) { return c.remote.Connect(ctx) }

func (c *Client) rpcWithHTTPClient(ctx context.Context, httpClient *http.Client) (shared.Service, error) {
	return c.remote.ConnectWithHTTPClient(ctx, httpClient)
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
		message, decodeErr := detail.Value()
		if decodeErr != nil {
			return fmt.Errorf("%w: approval detail could not be decoded", ErrAuthorizationApprovalPending)
		}
		if _, ok := message.(*apipb.AuthorizationApprovalPending); !ok {
			return fmt.Errorf("%w: approval detail had an unexpected type", ErrAuthorizationApprovalPending)
		}
		return fmt.Errorf("%w: operator approval is required; the operation was not started. Retry after approval", ErrAuthorizationApprovalPending)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) (bool, error) {
	service, err := c.rpc(ctx)
	if err != nil {
		return false, err
	}
	response, err := service.HealthResponse(ctx)
	if err != nil {
		return false, fmt.Errorf("agent-manager health: %w", err)
	}
	return response.GetStatus() == commonpb.HealthStatus_HEALTH_STATUS_HEALTHY, nil
}

func (c *Client) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.ReconcileScenarioProfiles(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("agent-manager reconcile scenario profiles: %w", err)
	}
	return response, nil
}

func (c *Client) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.CreateTaskResponse(ctx, task)
	if err != nil {
		if pending := approvalPending(err); pending != nil {
			return nil, pending
		}
		return nil, fmt.Errorf("agent-manager create task: %w", err)
	}
	return response.GetTask(), nil
}

// CreateRun retains Knowledge Observatory's caller authorization, no-redirect
// transport and redacted error policy; the common Connect request is shared.
func (c *Client) CreateRun(ctx context.Context, req *apipb.CreateRunRequest) (*domainpb.Run, error) {
	if req == nil {
		return nil, fmt.Errorf("create run request is required")
	}
	if err := owneridentity.RequireCreateRunCaller(ctx, time.Now(), true); err != nil {
		return nil, err
	}
	authorization, err := owneridentity.CreateRunAuthorization(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	service, err := c.rpcWithHTTPClient(ctx, &httpClient)
	if err != nil {
		return nil, err
	}
	result, err := service.CreateRunWithHeaders(ctx, req, http.Header{"Authorization": []string{authorization}})
	if err != nil {
		if pending := approvalPending(err); pending != nil {
			return nil, pending
		}
		return nil, fmt.Errorf("%s", owneridentity.RedactCreateRunCaller(ctx, fmt.Sprintf("agent-manager create run: %v", err)))
	}
	return result, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	result, err := service.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run: %w", err)
	}
	return result, nil
}

func (c *Client) GetRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.GetRunDiffResponse(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run diff: %w", err)
	}
	if response == nil {
		return nil, nil
	}
	return response.GetDiff(), nil
}

func (c *Client) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	events, err := service.GetRunEvents(ctx, runID, afterSequence)
	if err != nil {
		return nil, fmt.Errorf("agent-manager get run events: %w", err)
	}
	return events, nil
}

func (c *Client) ApproveRun(ctx context.Context, req *apipb.ApproveRunRequest) (*apipb.ApproveRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.ApproveRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agent-manager approve run: %w", err)
	}
	return response, nil
}

func (c *Client) RejectRun(ctx context.Context, req *apipb.RejectRunRequest) (*apipb.RejectRunResponse, error) {
	if req == nil || strings.TrimSpace(req.RunId) == "" {
		return nil, fmt.Errorf("run_id is required")
	}
	service, err := c.rpc(ctx)
	if err != nil {
		return nil, err
	}
	response, err := service.RejectRun(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("agent-manager reject run: %w", err)
	}
	return response, nil
}
