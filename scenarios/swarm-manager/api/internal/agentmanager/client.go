// Package agentmanager provides Swarm Manager's generated Connect client for
// Agent Manager. Only the plain /health probe remains REST.
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
	"github.com/vrooli/api-core/provenance"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type HTTPClient struct {
	baseURLResolver BaseURLResolver
	httpClient      HTTPDoer
}

type BaseURLResolver func(context.Context) (string, error)
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func NewHTTPClient() *HTTPClient { return NewHTTPClientWithTimeout(20 * time.Second) }
func NewHTTPClientWithTimeout(timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	h := &http.Client{Timeout: timeout, Transport: provenance.ForwardingTransport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return NewHTTPClientWithResolver(resolveAgentManagerBaseURL, h)
}
func NewHTTPClientWithResolver(resolver BaseURLResolver, doer HTTPDoer) *HTTPClient {
	if resolver == nil {
		resolver = resolveAgentManagerBaseURL
	}
	if doer == nil {
		doer = &http.Client{Timeout: 20 * time.Second, Transport: provenance.ForwardingTransport{}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	} else if client, ok := doer.(*http.Client); ok {
		copy := *client
		copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		if copy.Transport == nil {
			copy.Transport = provenance.ForwardingTransport{}
		}
		doer = &copy
	}
	return &HTTPClient{baseURLResolver: resolver, httpClient: doer}
}

var (
	ErrNotAvailable     = errors.New("agent-manager not available")
	ErrRequestFailed    = errors.New("agent-manager request failed")
	ErrWorkflowNotFound = errors.New("agent-manager workflow not found")
	ErrWorkflowNotReady = errors.New("agent-manager workflow result not ready")
)

// ApprovalPendingError keeps the owner's typed approval outcome available to
// callers, while retaining the raw generated detail for future fields.
type ApprovalPendingError struct {
	Detail *apipb.AuthorizationApprovalPending
}

func (e *ApprovalPendingError) Error() string {
	if e == nil || e.Detail == nil {
		return "agent-manager authorization approval pending"
	}
	return "agent-manager authorization approval pending: " + e.Detail.GetDetail()
}

func (c *HTTPClient) ResolveURL(ctx context.Context) (string, error) { return c.baseURLResolver(ctx) }
func (c *HTTPClient) connectClient(ctx context.Context) (apiconnect.AgentManagerServiceClient, error) {
	u, err := c.baseURLResolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotAvailable, err)
	}
	return apiconnect.NewAgentManagerServiceClient(c.httpClient, strings.TrimRight(u, "/")), nil
}
func req[T any](msg *T) *connect.Request[T] { return connect.NewRequest(msg) }
func connectError(err error) error {
	if err == nil {
		return nil
	}
	var ce *connect.Error
	if errors.As(err, &ce) {
		if ce.Code() == connect.CodeUnavailable || ce.Code() == connect.CodeDeadlineExceeded {
			return fmt.Errorf("%w: %w", ErrNotAvailable, err)
		}
		return fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	return err
}

// Health is a REST exception because it is the scenario lifecycle probe, not
// an application RPC. All application calls below use the generated client.
func (c *HTTPClient) Health(ctx context.Context) (bool, error) {
	u, err := c.baseURLResolver(ctx)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrNotAvailable, err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(u, "/")+"/health", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.httpClient.Do(r)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrNotAvailable, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
func (c *HTTPClient) ListProfiles(ctx context.Context, limit, offset int) (*apipb.ListProfilesResponse, error) {
	if limit <= 0 || limit > 5000 || offset < 0 {
		return nil, fmt.Errorf("%w: invalid profile list bounds", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	l, o := int32(limit), int32(offset)
	v, e := a.ListProfiles(ctx, req(&apipb.ListProfilesRequest{Limit: &l, Offset: &o}))
	if e != nil {
		return nil, connectError(e)
	}
	return v.Msg, nil
}
func (c *HTTPClient) EnsureProfile(ctx context.Context, r *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.EnsureProfile(ctx, req(r))
	if e != nil {
		return nil, connectError(e)
	}
	return v.Msg, nil
}
func (c *HTTPClient) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.ReconcileScenarioProfiles(ctx, req(&apipb.ReconcileScenarioProfilesRequest{Scenario: scenario}))
	if e != nil {
		return nil, connectError(e)
	}
	return v.Msg, nil
}
func (c *HTTPClient) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.CreateTask(ctx, req(&apipb.CreateTaskRequest{Task: task}))
	if e != nil {
		return nil, connectError(e)
	}
	if v.Msg.GetTask() == nil || strings.TrimSpace(v.Msg.GetTask().GetId()) == "" {
		return nil, fmt.Errorf("%w: missing task id", ErrRequestFailed)
	}
	return v.Msg.Task, nil
}
func (c *HTTPClient) CreateRun(ctx context.Context, r *apipb.CreateRunRequest) (*domainpb.Run, error) {
	if e := owneridentity.RequireCreateRunCaller(ctx, time.Now(), true); e != nil {
		return nil, e
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	q := req(r)
	proof, e := owneridentity.CreateRunRequestHeaders(ctx, time.Now())
	if e != nil {
		return nil, e
	}
	for k, vs := range proof {
		for _, v := range vs {
			q.Header().Add(k, v)
		}
	}
	v, e := a.CreateRun(ctx, q)
	if e != nil {
		var ce *connect.Error
		if errors.As(e, &ce) {
			for _, d := range ce.Details() {
				value, decodeErr := d.Value()
				if detail, ok := value.(*apipb.AuthorizationApprovalPending); decodeErr == nil && ok {
					return nil, &ApprovalPendingError{Detail: detail}
				}
			}
		}
		return nil, connectError(e)
	}
	if v.Msg.GetRun() == nil || strings.TrimSpace(v.Msg.GetRun().GetId()) == "" {
		return nil, fmt.Errorf("%w: missing run id", ErrRequestFailed)
	}
	return v.Msg.Run, nil
}
func (c *HTTPClient) GetRun(ctx context.Context, id string) (*domainpb.Run, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.GetRun(ctx, req(&apipb.GetRunRequest{RunId: id}))
	if e != nil {
		return nil, connectError(e)
	}
	if v.Msg.GetRun() == nil || strings.TrimSpace(v.Msg.GetRun().GetId()) == "" {
		return nil, fmt.Errorf("%w: missing run id", ErrRequestFailed)
	}
	return v.Msg.Run, nil
}
func (c *HTTPClient) GetRunAccounting(ctx context.Context, id string) (*apipb.RunAccounting, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.GetRunAccounting(ctx, req(&apipb.GetRunAccountingRequest{RunId: id}))
	if e != nil {
		return nil, connectError(e)
	}
	if strings.TrimSpace(v.Msg.GetRunId()) != id {
		return nil, fmt.Errorf("%w: accounting for run %q answered run %q", ErrRequestFailed, id, v.Msg.GetRunId())
	}
	return v.Msg, nil
}
func (c *HTTPClient) GetRunEvents(ctx context.Context, id string, o RunEventsOptions) (*apipb.GetRunEventsResponse, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	r := &apipb.GetRunEventsRequest{RunId: id}
	if o.AfterSequence > 0 {
		x := o.AfterSequence
		r.AfterSequence = &x
	}
	if o.Limit > 0 {
		x := o.Limit
		r.Limit = &x
	}
	v, e := a.GetRunEvents(ctx, req(r))
	if e != nil {
		return nil, connectError(e)
	}
	return v.Msg, nil
}
func (c *HTTPClient) GetRunDiff(ctx context.Context, id string) (*domainpb.RunDiff, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return nil, e
	}
	v, e := a.GetRunDiff(ctx, req(&apipb.GetRunDiffRequest{RunId: id}))
	if e != nil {
		return nil, connectError(e)
	}
	if v.Msg.GetDiff() == nil {
		return nil, fmt.Errorf("%w: missing run diff", ErrRequestFailed)
	}
	return v.Msg.Diff, nil
}
func (c *HTTPClient) ApproveRun(ctx context.Context, id, actor, msg string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	r := &apipb.ApproveRunRequest{RunId: id}
	if x := strings.TrimSpace(actor); x != "" {
		r.Actor = &x
	}
	if x := strings.TrimSpace(msg); x != "" {
		r.CommitMsg = &x
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return e
	}
	_, e = a.ApproveRun(ctx, req(r))
	return connectError(e)
}
func (c *HTTPClient) ContinueRun(ctx context.Context, id, msg string) error {
	return c.continueRun(ctx, id, msg, "")
}
func (c *HTTPClient) ContinueRunWithIdempotencyKey(ctx context.Context, id, msg, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrRequestFailed)
	}
	return c.continueRun(ctx, id, msg, key)
}
func (c *HTTPClient) continueRun(ctx context.Context, id, msg, key string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return e
	}
	_, e = a.ContinueRun(ctx, req(&domainpb.ContinueRunRequest{RunId: id, Message: msg, IdempotencyKey: key}))
	return connectError(e)
}
func (c *HTTPClient) StopRun(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("%w: run id is required", ErrRequestFailed)
	}
	a, e := c.connectClient(ctx)
	if e != nil {
		return e
	}
	_, e = a.StopRun(ctx, req(&apipb.StopRunRequest{RunId: id}))
	return connectError(e)
}

func resolveAgentManagerBaseURL(ctx context.Context) (string, error) {
	u, e := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if e != nil {
		return "", fmt.Errorf("%w: %w", ErrNotAvailable, e)
	}
	return u, nil
}
