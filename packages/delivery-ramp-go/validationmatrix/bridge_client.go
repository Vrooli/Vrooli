// Package bridgevalidation adapts the provider-neutral desktop validation
// contract to vrooli-bridge's typed dispatch and durable run APIs.
//
// Bridge owns reachability, authorization, and durable remote job identity. It
// does not own a desktop stream or BAS semantics; a dispatched job therefore
// cannot become a desktop PASS without target-owned evidence.
package validationmatrix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

type Registry interface {
	ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error)
}

type Dispatcher interface {
	DispatchJob(context.Context, *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error)
}

type Runs interface {
	WaitRun(context.Context, *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error)
	// GetRun is required because WaitRun reports only terminal status. A
	// dispatched probe's payload arrives as run log events, so recovering it
	// needs the run detail.
	GetRun(context.Context, *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error)
}

type Client struct {
	registry   Registry
	dispatcher Dispatcher
	runs       Runs
	// platform scopes discovery to one ramp's targets. A single probed node may
	// serve several platforms, and each ramp must see only its own.
	platform   string
	hostProber HostProber
}

// ClientOption configures optional discovery behaviour without widening the
// constructor for every ramp that does not need it.
type ClientOption func(*Client)

// WithPlatform scopes discovered targets to one platform ("ios", "android",
// "desktop"). Without it a client reports every platform a node can serve.
func WithPlatform(platform string) ClientOption {
	return func(c *Client) { c.platform = strings.ToLower(strings.TrimSpace(platform)) }
}

// WithHostProber overrides remote host-fact resolution, so tests can classify
// nodes without dispatching a job.
func WithHostProber(prober HostProber) ClientOption {
	return func(c *Client) { c.hostProber = prober }
}

func NewClient(baseURL, token string, httpClient *http.Client, options ...ClientOption) *Client {
	if strings.TrimSpace(baseURL) == "" {
		return nil
	}
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	transport := httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	if strings.TrimSpace(token) != "" {
		httpClient = &http.Client{Transport: bearerTransport{base: transport, token: token}, Timeout: httpClient.Timeout}
	}
	baseURL = strings.TrimRight(baseURL, "/")
	client := &Client{
		registry:   registryconnect.NewNodeRegistryServiceClient(httpClient, baseURL),
		dispatcher: dispatchconnect.NewDispatchServiceClient(httpClient, baseURL),
		runs:       runsconnect.NewRunsServiceClient(httpClient, baseURL),
	}
	for _, option := range options {
		option(client)
	}
	if client.hostProber == nil {
		client.hostProber = newDispatchHostProber(client.dispatcher, client.runs)
	}
	return client
}

// NewClientFromEnv builds a bridge client from an explicitly configured URL.
//
// It returns nil when no URL is set. A ramp whose primary execution path is a
// bridge node must not accept that silently: resolve the bridge scenario URL at
// the composition root and use NewClient, so an unset variable cannot disable
// remote execution while a healthy fleet runs beside it.
func NewClientFromEnv(options ...ClientOption) *Client {
	return NewClient(os.Getenv("VROOLI_BRIDGE_URL"), os.Getenv("VROOLI_BRIDGE_API_TOKEN"), nil, options...)
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

// NewClientForTesting binds typed fakes without exposing generated transport
// details to the matrix service tests.
func NewClientForTesting(registry Registry, dispatcher Dispatcher, runs Runs, options ...ClientOption) *Client {
	client := &Client{registry: registry, dispatcher: dispatcher, runs: runs}
	for _, option := range options {
		option(client)
	}
	if client.hostProber == nil {
		client.hostProber = newDispatchHostProber(dispatcher, runs)
	}
	return client
}

func (c *Client) Discover(ctx context.Context) ([]deliveryramp.Target, error) {
	if c == nil || c.registry == nil {
		return []deliveryramp.Target{unavailableTarget("bridge client is not configured")}, nil
	}
	response, err := c.registry.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list bridge nodes: %w", err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("bridge returned no node inventory")
	}
	targets := make([]deliveryramp.Target, 0, len(response.Msg.Nodes))
	for _, node := range response.Msg.Nodes {
		if node == nil {
			continue
		}
		// Only a reachable, authorized node is worth probing; probing an
		// offline node would spend a dispatch to learn what its status already
		// says.
		var facts HostFacts
		var factsErr error
		if nodeReachable(node) {
			facts, factsErr = c.probeHost(ctx, node.Id)
		}
		targets = append(targets, nodeTarget(node, facts, factsErr, c.platform))
	}
	if len(targets) == 0 {
		targets = append(targets, unavailableTarget("bridge fleet has no registered nodes"))
	}
	return targets, nil
}

func (c *Client) Execute(ctx context.Context, request CellRequest) CellResult {
	if c == nil || c.dispatcher == nil || c.runs == nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "bridge durable dispatch is not configured"}
	}
	if request.Cell == nil || strings.TrimSpace(request.Cell.GetTargetId()) == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge cell has no target identity"}
	}
	nodeID := strings.TrimPrefix(request.Cell.GetTargetId(), "bridge:")
	if nodeID == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge target identity is malformed"}
	}
	command := strings.TrimSpace(request.Command)
	if command == "" {
		command = DefaultCommand
	}
	dispatched, err := c.dispatcher.DispatchJob(ctx, connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:   nodeID,
		Verb:     command,
		Scenario: request.Cell.GetScenarioName(),
		Args:     append([]string(nil), request.Args...),
	}))
	if err != nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("bridge dispatch unavailable: %v", err), Retryable: true}
	}
	if dispatched == nil || dispatched.Msg == nil || strings.TrimSpace(dispatched.Msg.RunId) == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge accepted no durable run identity; desktop evidence was not claimed"}
	}
	runID := dispatched.Msg.RunId
	waited, err := c.runs.WaitRun(ctx, connect.NewRequest(&runsv1.WaitRunRequest{Id: runID, TimeoutSeconds: 900}))
	if err != nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s could not be reattached: %v", runID, err), Retryable: true}
	}
	if waited == nil || waited.Msg == nil || waited.Msg.Run == nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s returned no durable result", runID)}
	}
	run := waited.Msg.Run
	evidence := bridgeEvidence(nodeID, runID, request.ArtifactDigest)
	identity := ExecutionIdentity{NodeID: nodeID, JobID: runID, RunID: runID, ArtifactDigest: request.ArtifactDigest}
	switch run.Status {
	case runsv1.RunStatus_RUN_STATUS_PASSED:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge job passed, but bridge does not provide desktop evidence; target-owned evidence is required", Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	case runsv1.RunStatus_RUN_STATUS_FAILED:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("bridge validation job failed (exit %d)", run.ExitCode), Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	case runsv1.RunStatus_RUN_STATUS_ABORTED:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "bridge validation job was aborted", Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	default:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge wait returned a non-terminal run", Evidence: []*domainv1.LayeredEvidence{evidence}, Retryable: true, Identity: identity}
	}
}

// nodeReachable reports whether a node can accept a dispatched job right now.
func nodeReachable(node *registryv1.Node) bool {
	return node != nil && node.Online &&
		node.Status == registryv1.NodeStatus_NODE_STATUS_ONLINE &&
		hasDispatchScope(node.Scopes)
}

func (c *Client) probeHost(ctx context.Context, nodeID string) (HostFacts, error) {
	if c == nil || c.hostProber == nil {
		return HostFacts{}, fmt.Errorf("host prober is not configured")
	}
	return c.hostProber.ProbeHost(ctx, nodeID)
}

// nodeTarget projects one registry node into a target for the requested
// platform.
//
// Platform capability comes from probed host facts, never from
// node.Capabilities: that field carries the node's allowlisted dispatch verbs
// ("host inventory*", "setup*"), which share no vocabulary with platform names.
// Deriving capability from it reported every node as capability-less.
func nodeTarget(node *registryv1.Node, facts HostFacts, factsErr error, platform string) deliveryramp.Target {
	var (
		capabilities []string
		available    bool
		reason       string
		missing      string
		nextAction   string
		deviceKind   = "host"
	)

	switch {
	case node.Status == registryv1.NodeStatus_NODE_STATUS_REVOKED:
		reason, missing = targetmodel.ReasonBridgeRevoked, "trusted bridge node"
		nextAction = "re-register the node before dispatching to it"
	case !node.Online || node.Status != registryv1.NodeStatus_NODE_STATUS_ONLINE:
		reason, missing = targetmodel.ReasonBridgeOffline, "online bridge node"
		nextAction = "bring the node online, then probe again"
	case !hasDispatchScope(node.Scopes):
		reason, missing = targetmodel.ReasonBridgeNoDispatchScope, "bridge dispatch scope"
		nextAction = "grant the node a bridge write scope, then probe again"
	case factsErr != nil:
		reason, missing = targetmodel.ReasonBridgeNoHostProbe, "host toolchain probe"
		nextAction = "ensure the node answers `host inventory --json` over dispatch, then probe again"
	default:
		class, matched := selectPlatformClass(facts, platform)
		if !matched {
			reason = targetmodel.ReasonBridgeNoCapability
			missing = platformCapabilityName(platform)
			nextAction = "install the toolchain for this platform on the node, then probe again"
			break
		}
		capabilities, deviceKind = class.Capabilities, class.DeviceKind
		if class.Missing != "" {
			reason, missing, nextAction = targetmodel.ReasonBridgeNoCapability, class.Missing, class.NextAction
			break
		}
		available, reason = true, class.Reason
		missing, nextAction = "", ""
	}

	resolvedPlatform := platform
	if resolvedPlatform == "" {
		resolvedPlatform = "desktop"
	}
	return deliveryramp.Target{
		ID: "bridge:" + node.Id, Label: node.Name, Platform: resolvedPlatform, DeviceKind: deviceKind, Capabilities: capabilities,
		NodeID: node.Id, OS: node.Os, Architecture: node.Arch, Mode: "remote", Reason: reason,
		Available: available, MissingCapability: missing, NextAction: nextAction,
		Transport: deliveryramp.Transport{Kind: deliveryramp.TransportBridge, ID: node.Id, Trust: "bridge", Available: available, Reason: reason},
		Health:    deliveryramp.TargetHealth{Status: bridgeHealth(node, available), Reason: reason},
		BridgeTrust: &deliveryramp.BridgeTrust{
			Registered:         node.Status != registryv1.NodeStatus_NODE_STATUS_REVOKED,
			Online:             node.Online,
			DispatchAuthorized: hasDispatchScope(node.Scopes),
			Reason:             bridgeTrustReason(node),
		},
	}
}

func bridgeEvidence(nodeID, runID, artifactDigest string) *domainv1.LayeredEvidence {
	uri := fmt.Sprintf("bridge://%s/runs/%s", nodeID, runID)
	if strings.TrimSpace(artifactDigest) != "" {
		uri += "?artifact=" + url.QueryEscape(artifactDigest)
	}
	digest := sha256.Sum256([]byte(uri))
	return &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "bridge-run-" + runID, Uri: uri, Sha256: "sha256:" + hex.EncodeToString(digest[:]), Redacted: true}
}

func unavailableTarget(reason string) deliveryramp.Target {
	return deliveryramp.UnavailableTarget(reason, "bridge inventory")
}

func bridgeHealth(node *registryv1.Node, available bool) string {
	if node == nil || node.Status == registryv1.NodeStatus_NODE_STATUS_REVOKED {
		return "revoked"
	}
	if available {
		return "healthy"
	}
	if node.Online {
		return "degraded"
	}
	return "offline"
}

// hasDispatchScope reports whether a node's granted scopes permit dispatch.
//
// Bridge grants coarse scopes ("vrooli-bridge:read", "vrooli-bridge:write") and
// separately allowlists verbs per node. Matching the legacy "scenario test"
// string rejected every real node, because no node has ever carried a scope by
// that name.
func hasDispatchScope(scopes []string) bool {
	for _, scope := range scopes {
		switch strings.ToLower(strings.TrimSpace(scope)) {
		case "vrooli-bridge:write", "vrooli-bridge:admin":
			return true
		}
	}
	return false
}

// selectPlatformClass resolves the class a node can serve for the requested
// platform. An empty request means "any class this node can prove", which keeps
// a platform-agnostic caller working.
func selectPlatformClass(facts HostFacts, platform string) (platformClass, bool) {
	classes := classifyHost(facts)
	if len(classes) == 0 {
		return platformClass{}, false
	}
	if strings.TrimSpace(platform) == "" {
		return classes[0], true
	}
	return capabilityClassFor(classes, platform)
}

func platformCapabilityName(platform string) string {
	switch platform {
	case "ios":
		return "Apple build toolchain"
	case "android":
		return "Android SDK platform-tools"
	default:
		return "bridge host capability"
	}
}

func bridgeTrustReason(node *registryv1.Node) string {
	if node == nil || node.Status == registryv1.NodeStatus_NODE_STATUS_REVOKED {
		return "bridge node is revoked or missing"
	}
	if !hasDispatchScope(node.Scopes) {
		return "registered identity has no scenario-test dispatch scope"
	}
	return "registered identity and scenario-test dispatch scope verified"
}
