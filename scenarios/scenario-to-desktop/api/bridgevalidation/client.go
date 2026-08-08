// Package bridgevalidation adapts the provider-neutral desktop validation
// contract to vrooli-bridge's typed dispatch and durable run APIs.
//
// Bridge owns reachability, authorization, and durable remote job identity. It
// does not own a desktop stream or BAS semantics; a dispatched job therefore
// cannot become a desktop PASS without target-owned evidence.
package bridgevalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
	"scenario-to-desktop-api/targetinventory"
	"scenario-to-desktop-api/validationmatrix"
)

type Registry interface {
	ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error)
}

type Dispatcher interface {
	DispatchJob(context.Context, *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error)
}

type Runs interface {
	WaitRun(context.Context, *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error)
}

type Client struct {
	registry   Registry
	dispatcher Dispatcher
	runs       Runs
}

func NewClient(baseURL, token string, httpClient *http.Client) *Client {
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
	return &Client{
		registry:   registryconnect.NewNodeRegistryServiceClient(httpClient, baseURL),
		dispatcher: dispatchconnect.NewDispatchServiceClient(httpClient, baseURL),
		runs:       runsconnect.NewRunsServiceClient(httpClient, baseURL),
	}
}

func NewClientFromEnv() *Client {
	return NewClient(os.Getenv("VROOLI_BRIDGE_URL"), os.Getenv("VROOLI_BRIDGE_API_TOKEN"), nil)
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
func NewClientForTesting(registry Registry, dispatcher Dispatcher, runs Runs) *Client {
	return &Client{registry: registry, dispatcher: dispatcher, runs: runs}
}

func (c *Client) Discover(ctx context.Context) ([]targetinventory.Target, error) {
	if c == nil || c.registry == nil {
		return []targetinventory.Target{unavailableTarget("bridge client is not configured")}, nil
	}
	response, err := c.registry.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list bridge nodes: %w", err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("bridge returned no node inventory")
	}
	targets := make([]targetinventory.Target, 0, len(response.Msg.Nodes))
	for _, node := range response.Msg.Nodes {
		if node == nil {
			continue
		}
		targets = append(targets, nodeTarget(node))
	}
	if len(targets) == 0 {
		targets = append(targets, unavailableTarget("bridge fleet has no registered nodes"))
	}
	return targets, nil
}

func (c *Client) ExecuteBridge(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if c == nil || c.dispatcher == nil || c.runs == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "bridge durable dispatch is not configured"}
	}
	if request.Cell == nil || strings.TrimSpace(request.Cell.GetTargetId()) == "" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge cell has no target identity"}
	}
	nodeID := strings.TrimPrefix(request.Cell.GetTargetId(), "bridge:")
	if nodeID == "" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_REFUSED, Reason: "bridge target identity is malformed"}
	}
	dispatched, err := c.dispatcher.DispatchJob(ctx, connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:   nodeID,
		Verb:     "scenario test",
		Scenario: request.Cell.GetScenarioName(),
		Args:     nil,
	}))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: fmt.Sprintf("bridge dispatch unavailable: %v", err), Retryable: true}
	}
	if dispatched == nil || dispatched.Msg == nil || strings.TrimSpace(dispatched.Msg.RunId) == "" {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge accepted no durable run identity; desktop evidence was not claimed"}
	}
	runID := dispatched.Msg.RunId
	waited, err := c.runs.WaitRun(ctx, connect.NewRequest(&runsv1.WaitRunRequest{Id: runID, TimeoutSeconds: 900}))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s could not be reattached: %v", runID, err), Retryable: true}
	}
	if waited == nil || waited.Msg == nil || waited.Msg.Run == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: fmt.Sprintf("bridge run %s returned no durable result", runID)}
	}
	run := waited.Msg.Run
	evidence := bridgeEvidence(nodeID, runID)
	switch run.Status {
	case runsv1.RunStatus_RUN_STATUS_PASSED:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge job passed, but bridge does not provide desktop evidence; target-owned evidence is required", Evidence: []*domainv1.LayeredEvidence{evidence}}
	case runsv1.RunStatus_RUN_STATUS_FAILED:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: fmt.Sprintf("bridge validation job failed (exit %d)", run.ExitCode), Evidence: []*domainv1.LayeredEvidence{evidence}}
	case runsv1.RunStatus_RUN_STATUS_ABORTED:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "bridge validation job was aborted", Evidence: []*domainv1.LayeredEvidence{evidence}}
	default:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: "bridge wait returned a non-terminal run", Evidence: []*domainv1.LayeredEvidence{evidence}, Retryable: true}
	}
}

func nodeTarget(node *registryv1.Node) targetinventory.Target {
	capabilities := make([]domainv1.ValidationTargetCapability, 0, len(node.Capabilities))
	for _, raw := range node.Capabilities {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "electron-cdp", "desktop", "desktop.electron-cdp":
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP)
		case "native-window", "desktop.native-window":
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW)
		case "process-metrics", "desktop.process-metrics":
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS)
		case "offline-network", "desktop.offline-network":
			capabilities = append(capabilities, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_OFFLINE_NETWORK)
		}
	}
	available := node.Online && node.Status == registryv1.NodeStatus_NODE_STATUS_ONLINE
	reason := "bridge node is offline or not dispatchable"
	if available && len(capabilities) == 0 {
		reason = "bridge node is online but declares no desktop capability"
		available = false
	} else if available {
		reason = "bridge node is online; desktop evidence remains target-owned"
	}
	return targetinventory.Target{
		Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "bridge:" + node.Id, DisplayName: node.Name, Capabilities: capabilities, Available: available, Reason: stringPtr(reason)},
		Kind:       "bridge", NodeID: node.Id, OS: node.Os, Architecture: node.Arch, Mode: "remote", Reason: reason,
		Health: targetinventory.TargetHealth{Status: bridgeHealth(node, available), Reason: reason},
		BridgeTrust: &targetinventory.BridgeTrust{
			Registered:         node.Status != registryv1.NodeStatus_NODE_STATUS_REVOKED,
			Online:             node.Online,
			DispatchAuthorized: hasDispatchScope(node.Scopes),
			Reason:             bridgeTrustReason(node),
		},
	}
}

func bridgeEvidence(nodeID, runID string) *domainv1.LayeredEvidence {
	uri := fmt.Sprintf("bridge://%s/runs/%s", nodeID, runID)
	digest := sha256.Sum256([]byte(uri))
	return &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "bridge-run-" + runID, Uri: uri, Sha256: "sha256:" + hex.EncodeToString(digest[:]), Redacted: true}
}

func unavailableTarget(reason string) targetinventory.Target {
	return targetinventory.Target{
		Descriptor: &domainv1.ValidationTargetDescriptor{TargetId: "bridge:unavailable", DisplayName: "Bridge fleet", Available: false, Reason: stringPtr(reason)},
		Kind:       "bridge", Mode: "remote", Reason: reason,
		Health:      targetinventory.TargetHealth{Status: "unavailable", Reason: reason},
		BridgeTrust: &targetinventory.BridgeTrust{Reason: "bridge node identity was not verified"},
	}
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

func hasDispatchScope(scopes []string) bool {
	for _, scope := range scopes {
		normalized := strings.ToLower(strings.TrimSpace(scope))
		if normalized == "scenario test" || normalized == "scenario test*" {
			return true
		}
	}
	return false
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

func stringPtr(value string) *string { return &value }
