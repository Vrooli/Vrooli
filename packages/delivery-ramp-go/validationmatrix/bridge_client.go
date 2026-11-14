// Package bridgevalidation adapts the provider-neutral desktop validation
// contract to vrooli-bridge's typed dispatch and durable run APIs.
//
// Bridge owns reachability, authorization, and durable remote job identity. It
// does not own a desktop stream or BAS semantics; a dispatched job therefore
// cannot become a desktop PASS without target-owned evidence.
package validationmatrix

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
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

// ArtifactPlacer places a local artifact on a target node and returns the
// destination path that the dispatched validator must consume.
type ArtifactPlacer interface {
	Place(context.Context, string, CellRequest) (string, error)
}

type artifactPlacerFunc func(context.Context, string, CellRequest) (string, error)

func (f artifactPlacerFunc) Place(ctx context.Context, nodeID string, request CellRequest) (string, error) {
	return f(ctx, nodeID, request)
}

type Client struct {
	// node is the production transport. The legacy seams below remain only for
	// focused tests that exercise the matrix without a live Bridge.
	node       *nodereach.Client
	registry   Registry
	dispatcher Dispatcher
	runs       Runs
	// platform scopes discovery to one ramp's targets. A single probed node may
	// serve several platforms, and each ramp must see only its own.
	platform       string
	hostProber     HostProber
	tokenProvider  func(context.Context) (string, error)
	artifactPlacer ArtifactPlacer
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

// WithTokenProvider supplies a short-lived owner credential when the static
// bridge token is absent or expired.
func WithTokenProvider(provider func(context.Context) (string, error)) ClientOption {
	return func(c *Client) { c.tokenProvider = provider }
}

// WithArtifactPlacer replaces Bridge's production artifact service in tests.
func WithArtifactPlacer(placer ArtifactPlacer) ClientOption {
	return func(c *Client) { c.artifactPlacer = placer }
}

func NewClient(baseURL, token string, httpClient *http.Client, options ...ClientOption) *Client {
	client := &Client{}
	for _, option := range options {
		option(client)
	}
	client.node = nodereach.New(nodereach.Config{HTTPClient: httpClient, BridgeURL: baseURL, Token: token, TokenProvider: client.tokenProvider})
	if client.hostProber == nil {
		client.hostProber = newNodeHostProber(client.node)
	}
	if client.artifactPlacer == nil {
		client.artifactPlacer = artifactPlacerFunc(client.distributeArtifact)
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
	return NewClient("", os.Getenv("VROOLI_BRIDGE_API_TOKEN"), nil, options...)
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
	if c == nil {
		return []deliveryramp.Target{unavailableTarget("bridge client is not configured")}, nil
	}
	var nodes []*registryv1.Node
	var err error
	if c.node != nil {
		nodes, err = c.node.List(ctx, 10*time.Second)
	} else if c.registry != nil {
		var response *connect.Response[registryv1.ListNodesResponse]
		response, err = c.registry.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
		if response != nil && response.Msg != nil {
			nodes = response.Msg.GetNodes()
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list bridge nodes: %w", err)
	}
	if nodes == nil && c.node == nil && c.registry == nil {
		return nil, fmt.Errorf("bridge client is not configured")
	}
	targets := make([]deliveryramp.Target, 0, len(nodes))
	for _, node := range nodes {
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
	if c != nil && c.node != nil {
		return c.executeNode(ctx, request)
	}
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
	dispatchScenario := request.Cell.GetScenarioName()
	if command == "scenario-to-desktop validate-artifact" {
		dispatchScenario = "scenario-to-desktop"
	}
	dispatched, err := c.dispatcher.DispatchJob(ctx, connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:   nodeID,
		Verb:     command,
		Scenario: dispatchScenario,
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

func (c *Client) retrieveEvidenceBundle(ctx context.Context, runID string) (map[string][]byte, error) {
	baseURL, err := c.node.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	client := artifactsconnect.NewArtifactsServiceClient(c.node.ConnectTransport(ctx, baseURL), baseURL)
	response, err := client.GetRunArtifact(ctx, connect.NewRequest(&artifactsv1.GetRunArtifactRequest{RunId: runID, Name: "evidence-bundle.tar.gz"}))
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil || len(response.Msg.Data) == 0 {
		return nil, fmt.Errorf("run %s has no evidence-bundle.tar.gz output", runID)
	}
	return readEvidenceBundle(response.Msg.Data)
}

func readEvidenceBundle(data []byte) (map[string][]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open gzip archive: %w", err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	members := make(map[string][]byte)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, fmt.Errorf("read evidence archive: %w", nextErr)
		}
		if header == nil || header.Name == "" || header.Size < 0 || header.Size > 64<<20 {
			return nil, fmt.Errorf("invalid evidence archive member")
		}
		payload, readErr := io.ReadAll(io.LimitReader(reader, header.Size+1))
		if readErr != nil || int64(len(payload)) != header.Size {
			return nil, fmt.Errorf("incomplete evidence archive member %q", header.Name)
		}
		members[header.Name] = payload
	}
	return members, nil
}

func expandEvidenceBundle(bundle map[string][]byte, nodeID, runID string, request CellRequest) ([]*domainv1.LayeredEvidence, error) {
	items := []struct {
		name string
		kind domainv1.LayeredEvidence_Kind
	}{
		{"launch-trace.json", domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME},
		{"machine-assertion.json", domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION},
	}
	// The target command always records the declared journey, even for the
	// platform launch mode. Platform mode changes the required capability
	// predicate; it does not erase the target's workflow receipt.
	items = append(items, struct {
		name string
		kind domainv1.LayeredEvidence_Kind
	}{"journey-sidecar.json", domainv1.LayeredEvidence_KIND_BAS_WORKFLOW})
	result := make([]*domainv1.LayeredEvidence, 0, len(items))
	for _, item := range items {
		data, ok := bundle[item.name]
		if !ok || len(data) == 0 {
			return result, fmt.Errorf("required member %q is absent", item.name)
		}
		var assertion struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &assertion); err != nil {
			return result, fmt.Errorf("member %q is not JSON: %w", item.name, err)
		}
		if !strings.EqualFold(strings.TrimSpace(assertion.Status), "passed") {
			return result, fmt.Errorf("member %q has status %q", item.name, assertion.Status)
		}
		sum := sha256.Sum256(data)
		mediaType := "application/json"
		result = append(result, &domainv1.LayeredEvidence{Kind: item.kind, EvidenceId: fmt.Sprintf("%s-%s", strings.TrimSuffix(item.name, ".json"), runID), Uri: fmt.Sprintf("bridge://%s/runs/%s/artifacts/%s", nodeID, runID, url.PathEscape(item.name)), Sha256: "sha256:" + hex.EncodeToString(sum[:]), MediaType: &mediaType, Redacted: true})
	}
	return result, nil
}

func (c *Client) executeNode(ctx context.Context, request CellRequest) CellResult {
	if request.Cell == nil || strings.TrimSpace(request.Cell.GetTargetId()) == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge cell has no target identity"}
	}
	nodeID := strings.TrimPrefix(request.Cell.GetTargetId(), "bridge:")
	if nodeID == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge target identity is malformed"}
	}
	if strings.TrimSpace(request.ArtifactPath) != "" {
		if c.artifactPlacer == nil {
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "bridge artifact placement is not configured", Retryable: true}
		}
		destination, err := c.artifactPlacer.Place(ctx, nodeID, request)
		if err != nil {
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "bridge artifact placement unavailable: " + err.Error(), Retryable: true}
		}
		request.Args = append([]string{"--artifact", destination, "--artifact-digest", request.ArtifactDigest}, request.Args...)
	}
	command := strings.TrimSpace(request.Command)
	if command == "" {
		command = DefaultCommand
	}
	dispatchTimeout := 120 * time.Second
	if command == "scenario-to-desktop validate-artifact" {
		dispatchTimeout = 900 * time.Second
	}
	dispatchScenario := request.Cell.GetScenarioName()
	if command == "scenario-to-desktop validate-artifact" {
		// The cell scenario is the product being validated. Bridge's typed
		// dispatch scenario is the installed CLI owner, which is separate from
		// the product name and is also carried explicitly in request.Args.
		dispatchScenario = "scenario-to-desktop"
	}
	dispatched, err := c.node.Dispatch(ctx, nodereach.DispatchRequest{NodeID: nodeID, Scenario: dispatchScenario, Verb: command, Args: request.Args, Timeout: dispatchTimeout})
	if err != nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("bridge dispatch unavailable: %v", err), Retryable: true}
	}
	if strings.TrimSpace(dispatched.RunID) == "" {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge accepted no durable run identity; desktop evidence was not claimed"}
	}
	run, timedOut, err := c.node.Wait(ctx, dispatched.RunID, 900*time.Second)
	if err != nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s could not be reattached: %v", dispatched.RunID, err), Retryable: true}
	}
	if run == nil {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s returned no durable result", dispatched.RunID)}
	}
	evidence := bridgeEvidence(nodeID, dispatched.RunID, request.ArtifactDigest)
	identity := ExecutionIdentity{NodeID: nodeID, JobID: dispatched.RunID, RunID: dispatched.RunID, ArtifactDigest: request.ArtifactDigest}
	if timedOut {
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s timed out", dispatched.RunID), Retryable: true, Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	}
	switch run.Status {
	case runsv1.RunStatus_RUN_STATUS_PASSED:
		bundle, bundleErr := c.retrieveEvidenceBundle(ctx, dispatched.RunID)
		if bundleErr != nil {
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge evidence retrieval failed: " + bundleErr.Error(), Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
		}
		remoteEvidence, expandErr := expandEvidenceBundle(bundle, nodeID, dispatched.RunID, request)
		if expandErr != nil {
			return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge evidence bundle incomplete: " + expandErr.Error(), Evidence: append([]*domainv1.LayeredEvidence{evidence}, remoteEvidence...), Identity: identity}
		}
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "bridge validation and target-owned evidence completed", Evidence: append([]*domainv1.LayeredEvidence{evidence}, remoteEvidence...), Identity: identity}
	case runsv1.RunStatus_RUN_STATUS_FAILED:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("bridge validation job failed (exit %d)", run.ExitCode), Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	case runsv1.RunStatus_RUN_STATUS_ABORTED:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "bridge validation job was aborted", Evidence: []*domainv1.LayeredEvidence{evidence}, Identity: identity}
	default:
		return CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge wait returned a non-terminal run", Evidence: []*domainv1.LayeredEvidence{evidence}, Retryable: true, Identity: identity}
	}
}

// distributeArtifact uses Bridge's existing device-sync-hub-backed artifact
// service. The matrix never sends artifact bytes through the dispatch queue.
func (c *Client) distributeArtifact(ctx context.Context, nodeID string, request CellRequest) (string, error) {
	path, err := filepath.Abs(strings.TrimSpace(request.ArtifactPath))
	if err != nil {
		return "", fmt.Errorf("resolve artifact path: %w", err)
	}
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("read artifact %q: %w", path, err)
	}
	baseURL, err := c.node.ResolveURL(ctx)
	if err != nil {
		return "", err
	}
	client := artifactsconnect.NewArtifactsServiceClient(c.node.ConnectTransport(ctx, baseURL), baseURL)
	destination := filepath.Join("/tmp", "vrooli-desktop-validation", request.RunID, filepath.Base(path))
	fileRef := (&url.URL{Scheme: "file", Path: path}).String()
	response, err := client.DistributeArtifact(ctx, connect.NewRequest(&artifactsv1.DistributeArtifactRequest{
		NodeId: nodeID, Name: filepath.Base(path), SourceRef: fileRef, DestinationPath: destination,
	}))
	if err != nil {
		return "", err
	}
	if response == nil || response.Msg == nil || strings.TrimSpace(response.Msg.DistributionId) == "" {
		return "", fmt.Errorf("artifact service returned no distribution identity")
	}
	if response.Msg.Status == artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED {
		return "", fmt.Errorf("distribution %s failed", response.Msg.DistributionId)
	}
	if response.Msg.Status == artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED {
		return destination, nil
	}
	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("wait distribution %s: %w", response.Msg.DistributionId, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
		status, getErr := client.GetDistribution(ctx, connect.NewRequest(&artifactsv1.GetDistributionRequest{Id: response.Msg.DistributionId}))
		if getErr != nil {
			return "", getErr
		}
		if status == nil || status.Msg == nil || status.Msg.Distribution == nil {
			return "", fmt.Errorf("distribution %s returned no status", response.Msg.DistributionId)
		}
		switch status.Msg.Distribution.Status {
		case artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED:
			return destination, nil
		case artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED:
			return "", fmt.Errorf("distribution %s failed: %s", response.Msg.DistributionId, status.Msg.Distribution.Detail)
		}
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
		reason, missing = deliveryramp.ReasonBridgeRevoked, "trusted bridge node"
		nextAction = "re-register the node before dispatching to it"
	case !node.Online || node.Status != registryv1.NodeStatus_NODE_STATUS_ONLINE:
		reason, missing = deliveryramp.ReasonBridgeOffline, "online bridge node"
		nextAction = "bring the node online, then probe again"
	case !hasDispatchScope(node.Scopes):
		reason, missing = deliveryramp.ReasonBridgeNoDispatchScope, "bridge dispatch scope"
		nextAction = "grant the node a bridge write scope, then probe again"
	case factsErr != nil:
		reason, missing = deliveryramp.ReasonBridgeNoHostProbe, "host toolchain probe"
		nextAction = "ensure the node answers `host inventory --json` over dispatch, then probe again"
	default:
		class, matched := selectPlatformClass(facts, platform)
		if !matched {
			reason = deliveryramp.ReasonBridgeNoCapability
			missing = platformCapabilityName(platform)
			nextAction = "install the toolchain for this platform on the node, then probe again"
			break
		}
		capabilities, deviceKind = class.Capabilities, class.DeviceKind
		if class.Missing != "" {
			reason, missing, nextAction = deliveryramp.ReasonBridgeNoCapability, class.Missing, class.NextAction
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
