// Package nodereach is the one provider-facing reach for trusted Vrooli
// nodes. It owns Bridge location, authentication headers, bounded timeouts,
// typed relay arguments, and admission errors; consumers choose a node and
// render the result for their product surface.
package nodereach

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/websocket"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/cli-core/cliutil"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
	gateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate/gate_v1connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	"google.golang.org/protobuf/proto"
)

// ErrorKind classifies node-reach failures for product surfaces and logs.
type ErrorKind string

const (
	ErrBridgeUnavailable ErrorKind = "bridge_unavailable"
	ErrNodeNotFound      ErrorKind = "node_not_found"
	ErrNodeUnavailable   ErrorKind = "node_unavailable"
	ErrMissingReauth     ErrorKind = "missing_reauth"
	ErrMissingScope      ErrorKind = "missing_scope"
	ErrTransport         ErrorKind = "transport"
	ErrStreaming         ErrorKind = "streaming_unavailable"
	ErrHandshakeRejected ErrorKind = "handshake_rejected"
	// ErrInvalidRequest covers a caller-side mistake the control plane
	// refused, including an enrollment approval whose confirmation words do
	// not match the derived value.
	ErrInvalidRequest ErrorKind = "invalid_request"
)

// Error retains the address, verb, and missing grant that caused a failed
// operation. It is safe to expose after redacting transport credentials.
type Error struct {
	Kind  ErrorKind
	Node  string
	Verb  string
	Scope string
	Err   error
}

func interactiveTransportScope() string {
	scope, ok := scopecatalog.TransportScope("interactive-session:write")
	if !ok {
		panic("invalid interactive session transport scope")
	}
	return scope
}

func (e *Error) Error() string {
	parts := []string{fmt.Sprintf("node reach %s", e.Kind)}
	if e.Node != "" {
		parts = append(parts, fmt.Sprintf("node=%q", e.Node))
	}
	if e.Verb != "" {
		parts = append(parts, fmt.Sprintf("verb=%q", e.Verb))
	}
	if e.Scope != "" {
		parts = append(parts, fmt.Sprintf("missing_scope=%q", e.Scope))
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error { return e.Err }

// IsKind reports whether err is a node-reach error of kind.
func IsKind(err error, kind ErrorKind) bool {
	var target *Error
	return errors.As(err, &target) && target.Kind == kind
}

// Config controls Bridge discovery and authentication. BridgeURL is a test or
// operator override; production discovery remains slug-based per request.
type Config struct {
	HTTPClient       *http.Client
	BridgeURL        string
	ResolveBridgeURL func(context.Context) (string, error)
	Token            string
	// TokenProvider supplies a short-lived owner credential when Token is
	// empty. The returned value may include an authorization scheme, such as
	// "LocalSession <credential>"; the node client owns header formatting.
	TokenProvider func(context.Context) (string, error)
	ReauthToken   string
	ScopeResolver func(string) (string, bool)
}

// Client is safe for concurrent use. It does not cache Bridge addresses so a
// restarted local Bridge is found on the next request.
type Client struct {
	httpClient       *http.Client
	bridgeURL        string
	resolveBridgeURL func(context.Context) (string, error)
	token            string
	tokenProvider    func(context.Context) (string, error)
	reauthToken      string
	scopeResolver    func(string) (string, bool)
	dialer           websocket.Dialer
}

// New constructs a node client. An explicit BridgeURL wins over slug discovery;
// endpoint selection is typed configuration and is never overridden by a
// request-time environment variable.
func New(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	resolver := cfg.ResolveBridgeURL
	if resolver == nil {
		resolver = func(ctx context.Context) (string, error) {
			port := cliutil.DetectPortFromVrooli("vrooli-bridge", "API_PORT")()
			if port == "" {
				return "", fmt.Errorf("vrooli-bridge API port is not available")
			}
			return "http://localhost:" + port, nil
		}
	}
	return &Client{
		httpClient: httpClient, bridgeURL: strings.TrimRight(strings.TrimSpace(cfg.BridgeURL), "/"),
		resolveBridgeURL: resolver, token: strings.TrimSpace(cfg.Token), tokenProvider: cfg.TokenProvider, scopeResolver: cfg.ScopeResolver,
		reauthToken: strings.TrimSpace(cfg.ReauthToken),
	}
}

func (c *Client) endpoint(ctx context.Context) (string, error) {
	if c.bridgeURL != "" {
		return c.bridgeURL, nil
	}
	url, err := c.resolveBridgeURL(ctx)
	if err != nil {
		return "", &Error{Kind: ErrBridgeUnavailable, Err: err}
	}
	if strings.TrimSpace(url) == "" {
		return "", &Error{Kind: ErrBridgeUnavailable, Err: errors.New("Bridge URL resolver returned an empty address")}
	}
	return strings.TrimRight(strings.TrimSpace(url), "/"), nil
}

// ResolveURL returns the Bridge endpoint selected for one operation. Consumers
// that need to construct a second protocol on the same control plane (for
// example the session upgrade) may use this value, while discovery and env
// policy remain owned by this package.
func (c *Client) ResolveURL(ctx context.Context) (string, error) {
	return c.endpoint(ctx)
}

// ScenarioURL returns the target-aware base URL for a scenario. The returned
// URL is suitable for a generated Connect client; authentication must still
// be supplied through ConnectTransport.
func (c *Client) ScenarioURL(ctx context.Context, nodeID, scenario string) (string, error) {
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(scenario) == "" {
		return "", &Error{Kind: ErrInvalidRequest, Node: nodeID, Verb: scenario, Err: errors.New("node and scenario are required")}
	}
	baseURL, err := c.endpoint(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(baseURL, "/") + "/api/v1/targets/" + url.PathEscape(nodeID) + "/scenarios/" + url.PathEscape(scenario), nil
}

// ConnectTransport exposes the authenticated transport used by generated
// Connect clients for a target-aware ScenarioURL.
func (c *Client) ConnectTransport(ctx context.Context, baseURL string) connect.HTTPClient {
	return c.transport(ctx, strings.TrimRight(strings.TrimSpace(baseURL), "/"))
}

func (c *Client) requestContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}

func (c *Client) transport(ctx context.Context, baseURL string) connect.HTTPClient {
	return &authTransport{base: c.httpClient, token: c.token, tokenProvider: c.tokenProvider, ctx: ctx}
}

// List returns the current owner-visible fleet snapshot.
func (c *Client) List(ctx context.Context, timeout time.Duration) ([]*registryv1.Node, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := registryconnect.NewNodeRegistryServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.ListNodes(callCtx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no node list")}
	}
	return append([]*registryv1.Node(nil), resp.Msg.GetNodes()...), nil
}

// ListMachines returns Bridge's durable machine/lineage inventory. It is
// separate from List because a machine may have a current node identity that
// changes over time.
func (c *Client) ListMachines(ctx context.Context, timeout time.Duration) ([]*machinesv1.Machine, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := machinesconnect.NewMachineServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.ListMachines(callCtx, connect.NewRequest(&machinesv1.ListMachinesRequest{}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no machine list")}
	}
	return append([]*machinesv1.Machine(nil), resp.Msg.GetMachines()...), nil
}

// GetMachine returns the durable machine detail, including its read-time
// readiness and drift projection.
func (c *Client) GetMachine(ctx context.Context, machineID string, timeout time.Duration) (*machinesv1.GetMachineResponse, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := machinesconnect.NewMachineServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.GetMachine(callCtx, connect.NewRequest(&machinesv1.GetMachineRequest{Id: machineID}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no machine detail")}
	}
	return resp.Msg, nil
}

// CallRequest is a typed short-lived relay request. Args are never flattened
// into CSV; an empty element and commas are preserved as separate protobuf
// repeated-string values.
type CallRequest struct {
	NodeID        string
	Scenario      string
	Command       string
	Args          []string
	Timeout       time.Duration
	MaxResponse   uint64
	RequiredScope string
	Scopes        []string
}

type CallResponse struct {
	CorrelationID string
	Outcome       relayv1.RelayCallOutcome
	Data          []byte
	Reason        string
	ExitCode      int32
	TotalBytes    uint64
}

// ScenarioRequest is the common target-aware scenario read seam. The body
// and response remain protobuf bytes so each scenario keeps ownership of its
// generated request/response types while Bridge owns target selection,
// admission, authentication, and bounded transport.
type ScenarioRequest struct {
	NodeID      string
	Scenario    string
	Service     string
	Method      string
	HTTPMethod  string
	HTTPPath    string
	Body        []byte
	Timeout     time.Duration
	MaxResponse int64
}

// CallScenario invokes one catalog-admitted scenario procedure through the
// target proxy. An empty NodeID means the local scenario URL is not implied;
// callers must use the local service directly, keeping local and remote
// authorities explicit.
func (c *Client) CallScenario(ctx context.Context, req ScenarioRequest) ([]byte, error) {
	if strings.TrimSpace(req.NodeID) == "" || strings.TrimSpace(req.Scenario) == "" || strings.TrimSpace(req.Service) == "" || strings.TrimSpace(req.Method) == "" {
		return nil, &Error{Kind: ErrInvalidRequest, Node: req.NodeID, Verb: req.Scenario, Err: errors.New("node, scenario, service, and method are required")}
	}
	if req.Timeout <= 0 {
		req.Timeout = 8 * time.Second
	}
	if req.MaxResponse <= 0 || req.MaxResponse > 8<<20 {
		req.MaxResponse = 8 << 20
	}
	callCtx, cancel := c.requestContext(ctx, req.Timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	procedure := strings.Trim(strings.TrimSpace(req.Service), "/") + "/" + strings.Trim(strings.TrimSpace(req.Method), "/")
	if strings.TrimSpace(req.HTTPPath) != "" {
		procedure = strings.Trim(strings.TrimSpace(req.HTTPPath), "/")
	}
	path := strings.TrimRight(baseURL, "/") + "/api/v1/targets/" + url.PathEscape(req.NodeID) + "/scenarios/" + url.PathEscape(req.Scenario) + "/" + procedure
	httpReq, err := http.NewRequestWithContext(callCtx, http.MethodPost, path, strings.NewReader(string(req.Body)))
	if err != nil {
		return nil, &Error{Kind: ErrInvalidRequest, Node: req.NodeID, Verb: req.Scenario, Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/proto")
	httpReq.Header.Set("Accept", "application/proto")
	if method := strings.ToUpper(strings.TrimSpace(req.HTTPMethod)); method != "" && method != http.MethodPost {
		httpReq.Header.Set("X-Vrooli-HTTP-Method", method)
	}
	token, err := c.resolveToken(callCtx)
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Scenario, Err: err}
	}
	if token != "" {
		httpReq.Header.Set("Authorization", authHeader(token))
	}
	response, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Scenario, Err: err}
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, req.MaxResponse+1))
	if readErr != nil {
		return nil, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Scenario, Err: readErr}
	}
	if int64(len(body)) > req.MaxResponse {
		return nil, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Scenario, Err: errors.New("scenario response exceeded configured bound")}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Scenario, Err: fmt.Errorf("scenario proxy returned %s: %s", response.Status, strings.TrimSpace(string(body)))}
	}
	return body, nil
}

func (c *Client) onboardClient(ctx context.Context) (onboardconnect.OnboardServiceClient, context.Context, context.CancelFunc, error) {
	callCtx, cancel := c.requestContext(ctx, 30*time.Second)
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return onboardconnect.NewOnboardServiceClient(c.transport(callCtx, baseURL), baseURL), callCtx, cancel, nil
}

// PreflightOnboarding resolves an SSH target without touching it.
func (c *Client) PreflightOnboarding(ctx context.Context, req *onboardv1.PreflightOnboardingRequest) (*onboardv1.PreflightOnboardingResponse, error) {
	client, callCtx, cancel, err := c.onboardClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := client.PreflightOnboarding(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding preflight", Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding preflight", Err: errors.New("Bridge returned no preflight result")}
	}
	return resp.Msg, nil
}

// StartOnboarding starts a durable SSH onboarding operation. The request is
// passed through once; the caller must not retain the password after return.
func (c *Client) StartOnboarding(ctx context.Context, req *onboardv1.StartOnboardingRequest) (*onboardv1.StartOnboardingResponse, error) {
	client, callCtx, cancel, err := c.onboardClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := client.StartOnboarding(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding start", Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding start", Err: errors.New("Bridge returned no onboarding result")}
	}
	return resp.Msg, nil
}

// GetOnboarding reattaches to a durable operation after a client disconnect.
func (c *Client) GetOnboarding(ctx context.Context, req *onboardv1.GetOnboardingRequest) (*onboardv1.GetOnboardingResponse, error) {
	client, callCtx, cancel, err := c.onboardClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := client.GetOnboarding(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding get", Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding get", Err: errors.New("Bridge returned no onboarding state")}
	}
	return resp.Msg, nil
}

// WaitOnboarding blocks once on the server-owned onboarding operation.
func (c *Client) WaitOnboarding(ctx context.Context, req *onboardv1.WaitOnboardingRequest) (*onboardv1.WaitOnboardingResponse, error) {
	client, callCtx, cancel, err := c.onboardClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := client.WaitOnboarding(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding wait", Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding wait", Err: errors.New("Bridge returned no onboarding wait result")}
	}
	return resp.Msg, nil
}

func (c *Client) CancelOnboarding(ctx context.Context, req *onboardv1.CancelOnboardingRequest) (*onboardv1.CancelOnboardingResponse, error) {
	client, callCtx, cancel, err := c.onboardClient(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	resp, err := client.CancelOnboarding(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding cancel", Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Verb: "onboarding cancel", Err: errors.New("Bridge returned no onboarding cancellation result")}
	}
	return resp.Msg, nil
}

// Call relays one admitted command to a node.
func (c *Client) Call(ctx context.Context, req CallRequest) (CallResponse, error) {
	if err := c.validateRequest(req.NodeID, req.Command, req.RequiredScope, req.Scopes); err != nil {
		return CallResponse{}, err
	}
	callCtx, cancel := c.requestContext(ctx, req.Timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return CallResponse{}, err
	}
	client := relayconnect.NewRelayServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.Call(callCtx, connect.NewRequest(&relayv1.RelayCallRequest{
		NodeId: req.NodeID, Scenario: req.Scenario, Command: req.Command,
		Args: append([]string(nil), req.Args...), TimeoutSeconds: seconds(req.Timeout), MaxResponseBytes: req.MaxResponse,
	}))
	if err != nil {
		return CallResponse{}, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Command, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return CallResponse{}, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Command, Err: errors.New("Bridge returned no relay response")}
	}
	return CallResponse{CorrelationID: resp.Msg.GetCorrelationId(), Outcome: resp.Msg.GetOutcome(), Data: append([]byte(nil), resp.Msg.GetData()...), Reason: resp.Msg.GetReason(), ExitCode: resp.Msg.GetExitCode(), TotalBytes: resp.Msg.GetTotalBytes()}, nil
}

type DispatchRequest struct {
	NodeID   string
	Scenario string
	Verb     string
	Args     []string
	Timeout  time.Duration
}

type DispatchResponse struct {
	RunID  string
	Detail string
}

// Dispatch queues a typed job on a node.
func (c *Client) Dispatch(ctx context.Context, req DispatchRequest) (DispatchResponse, error) {
	if strings.TrimSpace(req.NodeID) == "" || strings.TrimSpace(req.Verb) == "" {
		return DispatchResponse{}, &Error{Kind: ErrNodeNotFound, Node: req.NodeID, Verb: req.Verb, Err: errors.New("node id and verb are required")}
	}
	callCtx, cancel := c.requestContext(ctx, req.Timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return DispatchResponse{}, err
	}
	client := dispatchconnect.NewDispatchServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.DispatchJob(callCtx, connect.NewRequest(&dispatchv1.DispatchJobRequest{NodeId: req.NodeID, Scenario: req.Scenario, Verb: req.Verb, Args: append([]string(nil), req.Args...), TimeoutSeconds: seconds(req.Timeout)}))
	if err != nil {
		return DispatchResponse{}, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Verb, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return DispatchResponse{}, &Error{Kind: ErrTransport, Node: req.NodeID, Verb: req.Verb, Err: errors.New("Bridge returned no dispatch response")}
	}
	return DispatchResponse{RunID: resp.Msg.GetRunId()}, nil
}

// Wait blocks once on a durable Bridge run and returns its latest run state.
// Consumers must use the returned state instead of polling the control plane.
func (c *Client) Wait(ctx context.Context, runID string, timeout time.Duration) (*runsv1.Run, bool, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, false, &Error{Kind: ErrTransport, Err: errors.New("run id is required")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, false, err
	}
	client := runsconnect.NewRunsServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.WaitRun(callCtx, connect.NewRequest(&runsv1.WaitRunRequest{Id: runID, TimeoutSeconds: seconds(timeout)}))
	if err != nil {
		return nil, false, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetRun() == nil {
		return nil, false, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no run state")}
	}
	return resp.Msg.GetRun(), resp.Msg.GetTimedOut(), nil
}

// Get retrieves a durable run and its state after a client disconnect.
func (c *Client) Get(ctx context.Context, runID string, timeout time.Duration) (*runsv1.Run, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("run id is required")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := runsconnect.NewRunsServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.GetRun(callCtx, connect.NewRequest(&runsv1.GetRunRequest{Id: runID}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetRun() == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no run state")}
	}
	return resp.Msg.GetRun(), nil
}

// RunGate starts a durable cross-platform validation gate.
func (c *Client) RunGate(ctx context.Context, req *gatev1.RunGateRequest, timeout time.Duration) (*gatev1.RunGateResponse, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := gateconnect.NewGateServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.RunGate(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no gate response")}
	}
	return resp.Msg, nil
}

// WaitGate blocks once for a durable cross-platform validation gate.
func (c *Client) WaitGate(ctx context.Context, req *gatev1.WaitGateRequest, timeout time.Duration) (*gatev1.WaitGateResponse, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := gateconnect.NewGateServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.WaitGate(callCtx, connect.NewRequest(req))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no gate result")}
	}
	return resp.Msg, nil
}

// OpenRequest describes a typed interactive session. The frame stream is
// introduced in the streaming phase; keeping the operation typed now prevents
// consumers from inventing private WebSocket contracts.
type OpenRequest struct {
	NodeID     string
	Command    string
	Args       []string
	SessionID  string
	Shell      string
	WorkingDir string
	Width      uint32
	Height     uint32
}

// Session is the typed bidirectional Bridge channel. Frames are translated
// here, keeping protobuf and WebSocket details out of product consumers.
type Session struct {
	conn         *websocket.Conn
	mu           sync.Mutex
	readCh       chan []byte
	doneCh       chan struct{}
	readyCh      chan error
	closed       bool
	seq          uint64
	pending      []byte
	terminalErr  error
	terminal     TerminalStatus
	reconnectCtx context.Context
	cancel       context.CancelFunc
	endpoint     string
	header       http.Header
	dialer       websocket.Dialer
	timeout      time.Duration
}

// TerminalStatus explains why the Bridge ended a session. It remains
// available after Read returns so callers can render a close reason without
// parsing a transport error string.
type TerminalStatus struct {
	Code   string
	Reason string
}

// Open connects to a node's interactive session endpoint and waits for the
// server's open frame before returning. Credentials are sent only as a server
// side authorization header.
func (c *Client) Open(ctx context.Context, req OpenRequest, timeout time.Duration) (*Session, error) {
	if strings.TrimSpace(req.NodeID) == "" {
		return nil, &Error{Kind: ErrNodeNotFound, Node: req.NodeID, Verb: req.Command, Err: errors.New("node id is required")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, &Error{Kind: ErrStreaming, Node: req.NodeID, Verb: req.Command, Err: fmt.Errorf("invalid Bridge URL %q", baseURL)}
	}
	u.Scheme = websocketScheme(u.Scheme)
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/channel/session"
	query := u.Query()
	query.Set("node", req.NodeID)
	// Session transport uses Bridge's ordinary write-effect grant. The query is
	// retained for compatibility with test-only handlers; production handlers
	// resolve grants from the registry.
	query.Set("scopes", interactiveTransportScope())
	if req.SessionID != "" {
		query.Set("session_id", req.SessionID)
	}
	if req.Shell != "" {
		query.Set("shell", req.Shell)
	}
	if req.WorkingDir != "" {
		query.Set("working_dir", req.WorkingDir)
	}
	if req.Command != "" {
		query.Set("command", req.Command)
	}
	u.RawQuery = query.Encode()
	header := http.Header{}
	token, tokenErr := c.resolveToken(callCtx)
	if tokenErr != nil {
		return nil, &Error{Kind: ErrStreaming, Node: req.NodeID, Verb: req.Command, Err: fmt.Errorf("resolve Bridge owner credential: %w", tokenErr)}
	}
	if token != "" {
		header.Set("Authorization", authHeader(token))
	}
	if c.reauthToken != "" {
		header.Set("X-Bridge-Owner-Reauth", c.reauthToken)
	}
	dialer := c.dialer
	dialer.HandshakeTimeout = timeout
	conn, response, err := dialer.DialContext(callCtx, u.String(), header)
	if err != nil {
		if response != nil && response.Body != nil {
			defer response.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
			reason := strings.TrimSpace(string(body))
			if reason == "" {
				reason = response.Status
			}
			kind := ErrTransport
			switch response.StatusCode {
			case http.StatusUnauthorized:
				kind = ErrMissingReauth
			case http.StatusForbidden:
				kind = ErrMissingScope
			case http.StatusNotFound:
				kind = ErrNodeNotFound
			case http.StatusServiceUnavailable:
				kind = ErrNodeUnavailable
			case http.StatusSwitchingProtocols:
				kind = ErrHandshakeRejected
			}
			return nil, &Error{Kind: kind, Node: req.NodeID, Verb: req.Command, Scope: interactiveTransportScope(), Err: fmt.Errorf("Bridge rejected session handshake: %s", reason)}
		}
		kind := ErrTransport
		if errors.Is(callCtx.Err(), context.DeadlineExceeded) {
			kind = ErrStreaming
		}
		return nil, &Error{Kind: kind, Node: req.NodeID, Verb: req.Command, Err: err}
	}
	reconnectCtx, reconnectCancel := context.WithCancel(context.Background())
	session := &Session{
		conn: conn, readCh: make(chan []byte, 64), doneCh: make(chan struct{}), readyCh: make(chan error, 1),
		reconnectCtx: reconnectCtx, cancel: reconnectCancel, endpoint: u.String(), header: header.Clone(), dialer: dialer, timeout: timeout,
	}
	go session.readLoop()
	select {
	case readyErr := <-session.readyCh:
		if readyErr != nil {
			_ = session.Close()
			return nil, &Error{Kind: ErrStreaming, Node: req.NodeID, Verb: req.Command, Err: readyErr}
		}
		return session, nil
	case <-callCtx.Done():
		_ = session.Close()
		return nil, &Error{Kind: ErrStreaming, Node: req.NodeID, Verb: req.Command, Err: callCtx.Err()}
	}
}

func (s *Session) readLoop() {
	defer close(s.doneCh)
	defer close(s.readCh)
	for {
		s.mu.Lock()
		conn := s.conn
		closed := s.closed
		s.mu.Unlock()
		if closed {
			return
		}
		kind, payload, err := conn.ReadMessage()
		if err != nil {
			_ = conn.Close()
			if s.reconnect() {
				continue
			}
			reconnectErr := fmt.Errorf("Bridge session reconnect exhausted: %w", err)
			s.setTerminalStatus("reconnect_exhausted", reconnectErr.Error(), reconnectErr)
			signalReady(s.readyCh, err)
			return
		}
		if kind != websocket.BinaryMessage {
			continue
		}
		var frame sessionv1.Frame
		if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &frame); err != nil {
			decodeErr := fmt.Errorf("decode Bridge session frame: %w", err)
			s.setTerminalStatus("protocol_error", decodeErr.Error(), decodeErr)
			signalReady(s.readyCh, decodeErr)
			return
		}
		switch payload := frame.Payload.(type) {
		case *sessionv1.Frame_Open:
			signalReady(s.readyCh, nil)
		case *sessionv1.Frame_Data:
			if data := append([]byte(nil), payload.Data.GetData()...); len(data) > 0 {
				select {
				case s.readCh <- data:
				case <-s.doneCh:
					return
				}
			}
		case *sessionv1.Frame_Close:
			reason := strings.TrimSpace(payload.Close.GetReason())
			if reason == "" {
				reason = "session closed by Bridge"
			}
			code := strings.TrimSpace(payload.Close.GetCode())
			if code == "" {
				code = "closed"
			}
			closeErr := errors.New(reason)
			s.setTerminalStatus(code, reason, closeErr)
			signalReady(s.readyCh, closeErr)
			return
		}
	}
}

const (
	maxReconnectAttempts  = 5
	initialReconnectDelay = 250 * time.Millisecond
	maxReconnectDelay     = 5 * time.Second
)

func (s *Session) reconnect() bool {
	for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
		delay := initialReconnectDelay << attempt
		if delay > maxReconnectDelay {
			delay = maxReconnectDelay
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-s.reconnectCtx.Done():
			timer.Stop()
			return false
		}

		conn, _, err := s.dialer.DialContext(s.reconnectCtx, s.endpoint, s.header)
		if err == nil {
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				_ = conn.Close()
				return false
			}
			s.conn = conn
			s.mu.Unlock()
			return true
		}
	}
	return false
}

func (s *Session) setTerminalErr(err error) {
	s.setTerminalStatus("transport_error", err.Error(), err)
}

func (s *Session) setTerminalStatus(code, reason string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminalErr == nil {
		s.terminalErr = err
		s.terminal = TerminalStatus{Code: code, Reason: reason}
	}
}

// TerminalStatus returns the final Bridge close or transport status, if one
// has been observed.
func (s *Session) TerminalStatus() (TerminalStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminalErr == nil {
		return TerminalStatus{}, false
	}
	return s.terminal, true
}

func signalReady(ch chan error, err error) {
	select {
	case ch <- err:
	default:
	}
}

func (s *Session) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	if len(s.pending) > 0 {
		n := copy(dst, s.pending)
		s.pending = s.pending[n:]
		return n, nil
	}
	select {
	case data, ok := <-s.readCh:
		if !ok {
			s.mu.Lock()
			err := s.terminalErr
			s.mu.Unlock()
			if err != nil {
				return 0, err
			}
			return 0, io.EOF
		}
		n := copy(dst, data)
		s.pending = data[n:]
		return n, nil
	case <-s.doneCh:
		s.mu.Lock()
		err := s.terminalErr
		s.mu.Unlock()
		if err != nil {
			return 0, err
		}
		return 0, io.EOF
	}
}

func (s *Session) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, io.ErrClosedPipe
	}
	err := s.writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: s.seq, Data: append([]byte(nil), data...)}}})
	s.seq++
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

func (s *Session) Resize(width, height uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return io.ErrClosedPipe
	}
	return s.writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Resize{Resize: &sessionv1.Resize{Columns: width, Rows: height}}})
}

func (s *Session) writeFrame(frame *sessionv1.Frame) error {
	payload, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}

func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	if s.cancel != nil {
		s.cancel()
	}
	_ = s.writeFrame(&sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Code: "close", Reason: "node client session closed"}}})
	err := s.conn.Close()
	s.mu.Unlock()
	return err
}

func authHeader(token string) string {
	if strings.HasPrefix(token, "Bearer ") || strings.HasPrefix(token, "LocalSession ") {
		return token
	}
	return "Bearer " + token
}

func (c *Client) resolveToken(ctx context.Context) (string, error) {
	if token := strings.TrimSpace(c.token); token != "" || c.tokenProvider == nil {
		return strings.TrimSpace(c.token), nil
	}
	token, err := c.tokenProvider(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

func websocketScheme(scheme string) string {
	if scheme == "https" {
		return "wss"
	}
	return "ws"
}

func (c *Client) validateRequest(node, verb, requiredScope string, heldScopes []string) error {
	if strings.TrimSpace(node) == "" {
		return &Error{Kind: ErrNodeNotFound, Node: node, Verb: verb, Err: errors.New("node id is required")}
	}
	if strings.TrimSpace(verb) == "" {
		return &Error{Kind: ErrTransport, Node: node, Err: errors.New("command is required")}
	}
	if c.scopeResolver != nil && strings.TrimSpace(requiredScope) == "" {
		if scope, ok := c.scopeResolver(verb); ok {
			requiredScope = scope
		}
	}
	if requiredScope != "" && len(heldScopes) > 0 && !scopecatalog.Resolve(heldScopes, requiredScope) {
		return &Error{Kind: ErrMissingScope, Node: node, Verb: verb, Scope: requiredScope, Err: errors.New("node does not hold the required scope")}
	}
	return nil
}

func seconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64((d + time.Second - 1) / time.Second)
}

type authTransport struct {
	base          *http.Client
	token         string
	tokenProvider func(context.Context) (string, error)
	ctx           context.Context
}

func (t *authTransport) Do(req *http.Request) (*http.Response, error) {
	request := req.Clone(t.ctx)
	token := strings.TrimSpace(t.token)
	if token == "" && t.tokenProvider != nil {
		var err error
		token, err = t.tokenProvider(t.ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve Bridge owner credential: %w", err)
		}
		token = strings.TrimSpace(token)
	}
	if token != "" {
		request.Header.Set("Authorization", authHeader(token))
	}
	return t.base.Do(request)
}
