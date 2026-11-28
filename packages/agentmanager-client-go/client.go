// Package agentmanagerclient shares service discovery and generated Connect
// procedures used by Agent Manager scenario clients.
package agentmanagerclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const scenarioID = "agent-manager"

// URLResolver supplies the current Agent Manager address.
type URLResolver interface {
	ResolveBaseURL(context.Context) (string, error)
}

type discoveryResolver struct{}

func (discoveryResolver) ResolveBaseURL(ctx context.Context) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, scenarioID)
}

// Client resolves Agent Manager and creates a fresh generated client for each
// operation group.
type Client struct {
	resolver  URLResolver
	http      connect.HTTPClient
	protoJSON bool
	headers   func(context.Context) http.Header
}

// Service is the common generated-procedure subset used by migrated scenarios.
type Service interface {
	Health(context.Context) (bool, error)
	HealthResponse(context.Context) (*apipb.HealthResponse, error)
	ListProfiles(context.Context) (*apipb.ListProfilesResponse, error)
	ReconcileScenarioProfiles(context.Context, string) (*apipb.ReconcileScenarioProfilesResponse, error)
	EnsureProfile(context.Context, *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error)
	GetProfile(context.Context, string) (*domainpb.AgentProfile, error)
	GetProfileResponse(context.Context, string) (*apipb.GetProfileResponse, error)
	GetRolePolicyCatalog(context.Context) (*apipb.GetRolePolicyCatalogResponse, error)
	CreateTask(context.Context, *domainpb.Task) (*domainpb.Task, error)
	CreateTaskResponse(context.Context, *domainpb.Task) (*apipb.CreateTaskResponse, error)
	GetTask(context.Context, string) (*apipb.GetTaskResponse, error)
	CreateRun(context.Context, *apipb.CreateRunRequest) (*domainpb.Run, error)
	CreateRunResponse(context.Context, *apipb.CreateRunRequest) (*apipb.CreateRunResponse, error)
	CreateRunWithHeaders(context.Context, *apipb.CreateRunRequest, http.Header) (*domainpb.Run, error)
	GetRun(context.Context, string) (*domainpb.Run, error)
	GetRunResponse(context.Context, string) (*apipb.GetRunResponse, error)
	GetRunDiff(context.Context, string) (*domainpb.RunDiff, error)
	GetRunDiffResponse(context.Context, string) (*apipb.GetRunDiffResponse, error)
	GetRunReport(context.Context, string) (*apipb.RunReport, error)
	GetRunByTag(context.Context, string) (*domainpb.Run, error)
	ListRuns(context.Context, ListRunsOptions) ([]*domainpb.Run, error)
	ListRunsResponse(context.Context, *apipb.ListRunsRequest) (*apipb.ListRunsResponse, error)
	StopRun(context.Context, string) error
	StopRunResponse(context.Context, string) (*apipb.StopRunResponse, error)
	StopRunByTag(context.Context, string) error
	StopAllRuns(context.Context, string) (*apipb.StopAllRunsResponse, error)
	GetRunEvents(context.Context, string, int64) ([]*domainpb.RunEvent, error)
	GetRunnerStatus(context.Context) ([]*domainpb.RunnerStatus, error)
	ContinueRun(context.Context, string, string, []string) (*domainpb.ContinueRunResponse, error)
	ApproveRun(context.Context, *apipb.ApproveRunRequest) (*apipb.ApproveRunResponse, error)
	RejectRun(context.Context, *apipb.RejectRunRequest) (*apipb.RejectRunResponse, error)
	GetRunEventsResponse(context.Context, string, RunEventsOptions) (*apipb.GetRunEventsResponse, error)
}

// RunEventsOptions preserves optional pagination fields at the wire boundary.
type RunEventsOptions struct {
	AfterSequence *int64
	Limit         *int32
}

// ListRunsOptions contains optional filters for listing runs.
type ListRunsOptions struct {
	TagPrefix string
	ProfileID string
	Status    *domainpb.RunStatus
	Limit     int
}

// Option configures the generated Connect client's wire behavior.
type Option func(*Client)

// WithProtoJSON selects the ProtoJSON Connect protocol for services that use it.
func WithProtoJSON() Option { return func(c *Client) { c.protoJSON = true } }

// WithProto selects the binary protobuf Connect protocol.
func WithProto() Option { return func(c *Client) { c.protoJSON = false } }

// WithHTTPClient injects a shared HTTP transport, including scenario-specific
// timeout, tracing, or test behavior.
func WithHTTPClient(client connect.HTTPClient) Option {
	return func(c *Client) { c.http = client }
}

// WithRequestHeaders adds context-derived headers to every generated request.
func WithRequestHeaders(headers func(context.Context) http.Header) Option {
	return func(c *Client) { c.headers = headers }
}

// New creates a client using scenario discovery and timeout for each request.
func New(timeout time.Duration, options ...Option) *Client {
	return NewWithResolver(timeout, discoveryResolver{}, options...)
}

// NewWithResolver injects target resolution for focused transport tests.
func NewWithResolver(timeout time.Duration, resolver URLResolver, options ...Option) *Client {
	// ProtoJSON is the default because the first shared consumer's established
	// contract uses it. Consumers with the generated client's protobuf default
	// select WithProto explicitly.
	client := &Client{resolver: resolver, http: &http.Client{Timeout: timeout}, protoJSON: true}
	for _, option := range options {
		option(client)
	}
	return client
}

// ResolveURL returns the current Agent Manager base URL.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	if c == nil || c.resolver == nil {
		return "", fmt.Errorf("agent manager client is not configured")
	}
	return c.resolver.ResolveBaseURL(ctx)
}

// Connect resolves the current service address and returns its shared RPC surface.
func (c *Client) Connect(ctx context.Context) (Service, error) {
	return c.ConnectWithHTTPClient(ctx, c.http)
}

// ConnectWithHTTPClient uses a per-operation transport while retaining the
// configured resolver, wire protocol, and request-header behavior.
func (c *Client) ConnectWithHTTPClient(ctx context.Context, httpClient connect.HTTPClient) (Service, error) {
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager url: %w", err)
	}
	if httpClient == nil {
		return nil, fmt.Errorf("agent manager client is not configured")
	}
	options := []connect.ClientOption{}
	if c.protoJSON {
		options = append(options, connect.WithProtoJSON())
	}
	client := apiconnect.NewAgentManagerServiceClient(httpClient, strings.TrimRight(baseURL, "/"), options...)
	return &service{client: client, headers: c.headers}, nil
}

type service struct {
	client  apiconnect.AgentManagerServiceClient
	headers func(context.Context) http.Header
}

func request[T any](s *service, ctx context.Context, message *T) *connect.Request[T] {
	return requestWithHeaders(s, ctx, message, nil)
}

func requestWithHeaders[T any](s *service, ctx context.Context, message *T, extra http.Header) *connect.Request[T] {
	req := connect.NewRequest(message)
	if s.headers != nil {
		for name, values := range s.headers(ctx) {
			req.Header()[name] = append([]string(nil), values...)
		}
	}
	for name, values := range extra {
		req.Header()[name] = append([]string(nil), values...)
	}
	return req
}

func (s *service) Health(ctx context.Context) (bool, error) {
	response, err := s.HealthResponse(ctx)
	if err != nil {
		return false, err
	}
	return response.GetReadiness(), nil
}

func (s *service) HealthResponse(ctx context.Context) (*apipb.HealthResponse, error) {
	response, err := s.client.Health(ctx, request(s, ctx, &apipb.HealthRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) ListProfiles(ctx context.Context) (*apipb.ListProfilesResponse, error) {
	response, err := s.client.ListProfiles(ctx, request(s, ctx, &apipb.ListProfilesRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) ReconcileScenarioProfiles(ctx context.Context, scenario string) (*apipb.ReconcileScenarioProfilesResponse, error) {
	response, err := s.client.ReconcileScenarioProfiles(ctx, request(s, ctx, &apipb.ReconcileScenarioProfilesRequest{Scenario: scenario}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) EnsureProfile(ctx context.Context, requestMessage *apipb.EnsureProfileRequest) (*apipb.EnsureProfileResponse, error) {
	response, err := s.client.EnsureProfile(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetProfile(ctx context.Context, profileID string) (*domainpb.AgentProfile, error) {
	response, err := s.client.GetProfile(ctx, request(s, ctx, &apipb.GetProfileRequest{ProfileId: profileID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetProfile(), nil
}

func (s *service) GetProfileResponse(ctx context.Context, profileID string) (*apipb.GetProfileResponse, error) {
	response, err := s.client.GetProfile(ctx, request(s, ctx, &apipb.GetProfileRequest{ProfileId: profileID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRolePolicyCatalog(ctx context.Context) (*apipb.GetRolePolicyCatalogResponse, error) {
	response, err := s.client.GetRolePolicyCatalog(ctx, request(s, ctx, &apipb.GetRolePolicyCatalogRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) CreateTask(ctx context.Context, task *domainpb.Task) (*domainpb.Task, error) {
	response, err := s.client.CreateTask(ctx, request(s, ctx, &apipb.CreateTaskRequest{Task: task}))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetTask(), nil
}

func (s *service) CreateTaskResponse(ctx context.Context, task *domainpb.Task) (*apipb.CreateTaskResponse, error) {
	response, err := s.client.CreateTask(ctx, request(s, ctx, &apipb.CreateTaskRequest{Task: task}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetTask(ctx context.Context, taskID string) (*apipb.GetTaskResponse, error) {
	response, err := s.client.GetTask(ctx, request(s, ctx, &apipb.GetTaskRequest{TaskId: taskID}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) CreateRun(ctx context.Context, requestMessage *apipb.CreateRunRequest) (*domainpb.Run, error) {
	response, err := s.client.CreateRun(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRun(), nil
}

func (s *service) CreateRunResponse(ctx context.Context, req *apipb.CreateRunRequest) (*apipb.CreateRunResponse, error) {
	response, err := s.client.CreateRun(ctx, request(s, ctx, req))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) CreateRunWithHeaders(ctx context.Context, req *apipb.CreateRunRequest, headers http.Header) (*domainpb.Run, error) {
	response, err := s.client.CreateRun(ctx, requestWithHeaders(s, ctx, req, headers))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRun(), nil
}

func (s *service) GetRun(ctx context.Context, runID string) (*domainpb.Run, error) {
	response, err := s.client.GetRun(ctx, request(s, ctx, &apipb.GetRunRequest{RunId: runID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRun(), nil
}

// GetRunResponse returns the generated response and preserves NotFound as an
// error for consumers whose local policy treats a missing run as exceptional.
func (s *service) GetRunResponse(ctx context.Context, runID string) (*apipb.GetRunResponse, error) {
	response, err := s.client.GetRun(ctx, request(s, ctx, &apipb.GetRunRequest{RunId: runID}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRunDiff(ctx context.Context, runID string) (*domainpb.RunDiff, error) {
	response, err := s.client.GetRunDiff(ctx, request(s, ctx, &apipb.GetRunDiffRequest{RunId: runID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetDiff(), nil
}

func (s *service) GetRunDiffResponse(ctx context.Context, runID string) (*apipb.GetRunDiffResponse, error) {
	response, err := s.client.GetRunDiff(ctx, request(s, ctx, &apipb.GetRunDiffRequest{RunId: runID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRunReport(ctx context.Context, runID string) (*apipb.RunReport, error) {
	response, err := s.client.GetRunReport(ctx, request(s, ctx, &apipb.GetRunReportRequest{RunId: runID}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRunByTag(ctx context.Context, tag string) (*domainpb.Run, error) {
	response, err := s.client.GetRunByTag(ctx, request(s, ctx, &apipb.GetRunByTagRequest{Tag: tag}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRun(), nil
}

func (s *service) ListRuns(ctx context.Context, options ListRunsOptions) ([]*domainpb.Run, error) {
	requestMessage := &apipb.ListRunsRequest{}
	if options.TagPrefix != "" {
		requestMessage.TagPrefix = &options.TagPrefix
	}
	if options.ProfileID != "" {
		requestMessage.AgentProfileId = &options.ProfileID
	}
	if options.Status != nil {
		requestMessage.Status = options.Status
	}
	if options.Limit > 0 {
		limit := int32(options.Limit)
		requestMessage.Limit = &limit
	}
	response, err := s.client.ListRuns(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRuns(), nil
}

func (s *service) ListRunsResponse(ctx context.Context, requestMessage *apipb.ListRunsRequest) (*apipb.ListRunsResponse, error) {
	response, err := s.client.ListRuns(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) StopRun(ctx context.Context, runID string) error {
	_, err := s.client.StopRun(ctx, request(s, ctx, &apipb.StopRunRequest{RunId: runID}))
	return err
}

func (s *service) StopRunResponse(ctx context.Context, runID string) (*apipb.StopRunResponse, error) {
	response, err := s.client.StopRun(ctx, request(s, ctx, &apipb.StopRunRequest{RunId: runID}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) StopRunByTag(ctx context.Context, tag string) error {
	_, err := s.client.StopRunByTag(ctx, request(s, ctx, &apipb.StopRunByTagRequest{Tag: tag}))
	return err
}

func (s *service) StopAllRuns(ctx context.Context, tagPrefix string) (*apipb.StopAllRunsResponse, error) {
	requestMessage := &apipb.StopAllRunsRequest{}
	if tagPrefix != "" {
		requestMessage.TagPrefix = &tagPrefix
	}
	response, err := s.client.StopAllRuns(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRunEvents(ctx context.Context, runID string, afterSequence int64) ([]*domainpb.RunEvent, error) {
	requestMessage := &apipb.GetRunEventsRequest{RunId: runID}
	if afterSequence > 0 {
		requestMessage.AfterSequence = &afterSequence
	}
	response, err := s.client.GetRunEvents(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetEvents(), nil
}

func (s *service) GetRunEventsResponse(ctx context.Context, runID string, options RunEventsOptions) (*apipb.GetRunEventsResponse, error) {
	requestMessage := &apipb.GetRunEventsRequest{RunId: runID, AfterSequence: options.AfterSequence, Limit: options.Limit}
	response, err := s.client.GetRunEvents(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) ContinueRun(ctx context.Context, runID, message string, attachmentIDs []string) (*domainpb.ContinueRunResponse, error) {
	response, err := s.client.ContinueRun(ctx, request(s, ctx, &domainpb.ContinueRunRequest{RunId: runID, Message: message, AttachmentIds: attachmentIDs}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) ApproveRun(ctx context.Context, requestMessage *apipb.ApproveRunRequest) (*apipb.ApproveRunResponse, error) {
	response, err := s.client.ApproveRun(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) RejectRun(ctx context.Context, requestMessage *apipb.RejectRunRequest) (*apipb.RejectRunResponse, error) {
	response, err := s.client.RejectRun(ctx, request(s, ctx, requestMessage))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (s *service) GetRunnerStatus(ctx context.Context) ([]*domainpb.RunnerStatus, error) {
	response, err := s.client.GetRunnerStatus(ctx, request(s, ctx, &apipb.GetRunnerStatusRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg.GetRunners(), nil
}
