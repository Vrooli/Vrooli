// Package agentmanager projects web-search typed inputs and results through the
// Agent Manager declared workflow API. Execution and waiting remain owner-held.
package agentmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/provenance"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

var (
	ErrNotAvailable  = errors.New("agent-manager unavailable")
	ErrRequestFailed = errors.New("agent-manager request failed")
)

type Client interface {
	Start(context.Context, *apipb.StartWorkflowExecutionRequest) (*domainpb.WorkflowExecution, error)
	Get(context.Context, string) (*domainpb.WorkflowExecution, error)
	Result(context.Context, string) (*domainpb.WorkflowExecution, error)
	Wait(context.Context, string, int) (*domainpb.WorkflowExecution, bool, error)
}
type CancelClient interface {
	Cancel(context.Context, *apipb.WorkflowExecutionOperationRequest) (*domainpb.WorkflowExecution, error)
}
type HTTPClient struct {
	Resolver func(context.Context) (string, error)
	HTTP     *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{Resolver: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	}, HTTP: &http.Client{Timeout: 100 * time.Second, Transport: provenance.ForwardingTransport{}}}
}

func (c *HTTPClient) client(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	u, err := c.Resolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotAvailable, err)
	}
	return apiconnect.NewAgentManagerServiceClient(c.HTTP, u), nil
}

func request[T any](ctx context.Context, m *T) *connect.Request[T] {
	r := connect.NewRequest(m)
	return r
}

func (c *HTTPClient) Start(ctx context.Context, r *apipb.StartWorkflowExecutionRequest) (*domainpb.WorkflowExecution, error) {
	a, e := c.client(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.StartWorkflowExecution(ctx, request(ctx, r))
	if e != nil {
		return nil, e
	}
	return v.Msg.Execution, nil
}

func (c *HTTPClient) Get(ctx context.Context, id string) (*domainpb.WorkflowExecution, error) {
	a, e := c.client(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.GetWorkflowExecution(ctx, request(ctx, &apipb.GetWorkflowExecutionRequest{ExecutionId: id}))
	if e != nil {
		return nil, e
	}
	return v.Msg.Execution, nil
}

func (c *HTTPClient) Result(ctx context.Context, id string) (*domainpb.WorkflowExecution, error) {
	a, e := c.client(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.GetWorkflowExecutionResult(ctx, request(ctx, &apipb.GetWorkflowExecutionResultRequest{ExecutionId: id, ExplicitlyAuthorized: true}))
	if e != nil {
		return nil, e
	}
	return v.Msg.Execution, nil
}

func (c *HTTPClient) Wait(ctx context.Context, id string, n int) (*domainpb.WorkflowExecution, bool, error) {
	if n < 1 || n > 90 {
		return nil, false, fmt.Errorf("wait timeout must be between 1 and 90 seconds")
	}
	a, e := c.client(ctx)
	if e != nil {
		return nil, false, e
	}
	v, e := a.WaitWorkflowExecution(ctx, request(ctx, &apipb.WaitWorkflowExecutionRequest{ExecutionId: id, TimeoutSeconds: int32(n)}))
	if e != nil {
		return nil, false, e
	}
	return v.Msg.Execution, v.Msg.TimedOut, nil
}

func (c *HTTPClient) Cancel(ctx context.Context, r *apipb.WorkflowExecutionOperationRequest) (*domainpb.WorkflowExecution, error) {
	a, e := c.client(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.CancelWorkflowExecution(ctx, request(ctx, r))
	if e != nil {
		return nil, e
	}
	return v.Msg.Execution, nil
}
