// Package bridge is the reach adapter over api-core/nodereach: typed dispatch
// for effectful verbs (durable Bridge runs, reattachable by run id), the
// signed relay for reads, and node registry facts for negotiation. Cloud keeps
// no machine inventory of its own; identity, grants and delivery belong to
// vrooli-bridge.
package bridge

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/nodereach"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// Scenario is the CLI namespace every cloud verb lives under on the node.
const Scenario = "vrooli"

// Client is the subset of nodereach the adapter needs; nodereach.Client
// satisfies it, tests use fakes.
type Client interface {
	Dispatch(ctx context.Context, req nodereach.DispatchRequest) (nodereach.DispatchResponse, error)
	Wait(ctx context.Context, runID string, timeout time.Duration) (*runsv1.Run, bool, error)
	Call(ctx context.Context, req nodereach.CallRequest) (nodereach.CallResponse, error)
	List(ctx context.Context, timeout time.Duration) ([]*registryv1.Node, error)
}

// ArtifactRequest is the Bridge-owned input for one target placement. The
// source path is read by Bridge's configured device-sync-hub adapter; bytes do
// not enter scenario-to-cloud's durable state.
type ArtifactRequest struct {
	NodeID          string
	Name            string
	SourceRef       string
	DestinationPath string
}

// ArtifactResult is the bounded status returned by Bridge's artifacts owner.
type ArtifactResult struct {
	DistributionID string
	DeliveryRef    string
	Delivered      bool
	Failed         bool
	Detail         string
}

// ArtifactClient is optional on the test seam so command and negotiation tests
// do not need an artifact service. Production wiring supplies the generated
// Bridge artifacts client through NewArtifactClient.
type ArtifactClient interface {
	Distribute(context.Context, ArtifactRequest) (ArtifactResult, error)
	Get(context.Context, string) (ArtifactResult, error)
}

type bridgeAPI interface {
	ResolveURL(context.Context) (string, error)
	ConnectTransport(context.Context, string) connect.HTTPClient
}

type connectArtifactClient struct{ api bridgeAPI }

// NewArtifactClient binds Bridge's generated artifacts service to the narrow
// client seam used by Adapter. The owner token remains inside nodereach's
// authenticated Connect transport.
func NewArtifactClient(api bridgeAPI) ArtifactClient { return &connectArtifactClient{api: api} }

func (c *connectArtifactClient) client(ctx context.Context) (artifactsconnect.ArtifactsServiceClient, error) {
	if c == nil || c.api == nil {
		return nil, errors.New("bridge artifacts client is not configured")
	}
	baseURL, err := c.api.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	return artifactsconnect.NewArtifactsServiceClient(c.api.ConnectTransport(ctx, baseURL), baseURL), nil
}

func (c *connectArtifactClient) Distribute(ctx context.Context, req ArtifactRequest) (ArtifactResult, error) {
	client, err := c.client(ctx)
	if err != nil {
		return ArtifactResult{}, err
	}
	resp, err := client.DistributeArtifact(ctx, connect.NewRequest(&artifactsv1.DistributeArtifactRequest{
		NodeId: req.NodeID, Name: req.Name, SourceRef: req.SourceRef, DestinationPath: req.DestinationPath,
	}))
	if err != nil {
		return ArtifactResult{}, err
	}
	if resp == nil || resp.Msg == nil {
		return ArtifactResult{}, errors.New("Bridge returned no artifact distribution")
	}
	return artifactResult(resp.Msg.GetDistributionId(), resp.Msg.GetDeliveryRef(), resp.Msg.GetStatus(), ""), nil
}

func (c *connectArtifactClient) Get(ctx context.Context, id string) (ArtifactResult, error) {
	client, err := c.client(ctx)
	if err != nil {
		return ArtifactResult{}, err
	}
	resp, err := client.GetDistribution(ctx, connect.NewRequest(&artifactsv1.GetDistributionRequest{Id: id}))
	if err != nil {
		return ArtifactResult{}, err
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetDistribution() == nil {
		return ArtifactResult{}, errors.New("Bridge returned no artifact distribution state")
	}
	d := resp.Msg.GetDistribution()
	return artifactResult(d.GetId(), d.GetDeliveryRef(), d.GetStatus(), d.GetDetail()), nil
}

func artifactResult(id, ref string, status artifactsv1.DeliveryStatus, detail string) ArtifactResult {
	return ArtifactResult{
		DistributionID: id,
		DeliveryRef:    ref,
		Delivered:      status == artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED,
		Failed:         status == artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED,
		Detail:         detail,
	}
}

// Adapter runs reach commands through the Bridge.
type Adapter struct {
	Client         Client
	Artifacts      ArtifactClient
	DefaultTimeout time.Duration
}

var _ reach.Reach = (*Adapter)(nil)

func (a *Adapter) timeout(cmd reach.Command) time.Duration {
	switch {
	case cmd.Timeout > 0:
		return cmd.Timeout
	case a.DefaultTimeout > 0:
		return a.DefaultTimeout
	default:
		return 10 * time.Minute
	}
}

// classify maps a nodereach error to a reach kind without reading message text.
func classify(err error, target identity.TargetRef) error {
	var ne *nodereach.Error
	if errors.As(err, &ne) {
		switch ne.Kind {
		case nodereach.ErrNodeNotFound:
			return &reach.Error{Kind: reach.KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "node is not enrolled", Err: err}
		case nodereach.ErrNodeUnavailable:
			return &reach.Error{Kind: reach.KindTargetOffline, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
		case nodereach.ErrMissingScope, nodereach.ErrMissingReauth:
			return &reach.Error{Kind: reach.KindScopeMissing, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
		case nodereach.ErrHandshakeRejected, nodereach.ErrStreaming:
			return &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
		case nodereach.ErrInvalidRequest:
			return &reach.Error{Kind: reach.KindInvalidArgument, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
		case nodereach.ErrBridgeUnavailable:
			return &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
		}
	}
	return &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Err: err}
}

// Exec implements reach.Reach. Effectful commands become durable dispatch
// runs; reads go through the relay so their output is returned inline.
func (a *Adapter) Exec(ctx context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if a == nil || a.Client == nil {
		return reach.Result{}, &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge adapter not configured"}
	}
	if err := reach.ValidateCommand(cmd); err != nil {
		return reach.Result{}, err
	}
	node := strings.TrimSpace(target.NodeID)
	if node == "" {
		return reach.Result{}, &reach.Error{Kind: reach.KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "target binding has no enrolled node"}
	}
	if cmd.IsObservation() && cmd.Observation == nil {
		// The relay exposes scenario verbs, never arbitrary host programs; the
		// facts an observation program would read are the node agent's to
		// report through a typed verb.
		return reach.Result{}, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge relay does not run host observation program " + cmd.Program}
	}
	argv := cmd.Argv()[1:]
	if cmd.Effectful {
		return a.dispatch(ctx, target, node, argv, cmd)
	}
	return a.relay(ctx, target, node, argv, cmd)
}

func (a *Adapter) relay(ctx context.Context, target identity.TargetRef, node string, argv []string, cmd reach.Command) (reach.Result, error) {
	resp, err := a.Client.Call(ctx, nodereach.CallRequest{NodeID: node, Scenario: Scenario, Command: argv[0], Args: argv[1:], Timeout: a.timeout(cmd), RequiredScope: cmd.RequiredScope})
	if err != nil {
		return reach.Result{}, classify(err, target)
	}
	out := reach.Result{ExitCode: int(resp.ExitCode), Stdout: string(resp.Data), CorrelationID: resp.CorrelationID, Transport: identity.TransportBridge}
	switch resp.Outcome {
	case relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_FAILED:
		return out, nil
	case relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_TERMINATED:
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "relay call terminated: " + resp.Reason}
	default:
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "relay outcome unspecified"}
	}
}

func (a *Adapter) dispatch(ctx context.Context, target identity.TargetRef, node string, argv []string, cmd reach.Command) (reach.Result, error) {
	timeout := a.timeout(cmd)
	dr, err := a.Client.Dispatch(ctx, nodereach.DispatchRequest{NodeID: node, Scenario: Scenario, Verb: strings.Join(strings.Fields(cmd.Verb), " "), Args: cmd.Args, Timeout: timeout})
	if err != nil {
		return reach.Result{}, classify(err, target)
	}
	return a.Attach(ctx, target, dr.RunID, timeout)
}

// Attach waits on an existing Bridge run. An operation that lost its reply
// reattaches here by run id instead of dispatching again.
func (a *Adapter) Attach(ctx context.Context, target identity.TargetRef, runID string, timeout time.Duration) (reach.Result, error) {
	run, done, err := a.Client.Wait(ctx, runID, timeout)
	if err != nil {
		return reach.Result{RunID: runID, Transport: identity.TransportBridge}, classify(err, target)
	}
	out := reach.Result{RunID: runID, Transport: identity.TransportBridge}
	if run != nil {
		out.ExitCode = int(run.GetExitCode())
	}
	if !done {
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "run " + runID + " still pending; reattach by run id"}
	}
	if run == nil {
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "run " + runID + " returned no record"}
	}
	switch run.GetStatus() {
	case runsv1.RunStatus_RUN_STATUS_PASSED, runsv1.RunStatus_RUN_STATUS_FAILED:
		return out, nil
	case runsv1.RunStatus_RUN_STATUS_ABORTED:
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "run " + runID + " aborted"}
	default:
		return out, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "run " + runID + " ended in state " + run.GetStatus().String()}
	}
}

// Deliver delegates each file to Bridge's owner-authenticated artifacts
// service and waits for the durable distribution to reach a terminal state.
// There is deliberately no SSH fallback: a Bridge-bound target either proves
// placement through Bridge or returns a typed transport failure.
func (a *Adapter) Deliver(ctx context.Context, target identity.TargetRef, delivery reach.Delivery) (reach.DeliveryReceipt, error) {
	if err := reach.ValidateDelivery(delivery); err != nil {
		return reach.DeliveryReceipt{}, err
	}
	if a == nil || a.Artifacts == nil {
		return reach.DeliveryReceipt{Transport: identity.TransportBridge}, &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge artifact delivery is not configured"}
	}
	if _, ok := ctx.Deadline(); !ok {
		timeout := a.DefaultTimeout
		if timeout <= 0 {
			timeout = 10 * time.Minute
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	receipt := reach.DeliveryReceipt{Transport: identity.TransportBridge, Files: make([]reach.DeliveredFile, 0, len(delivery.Files))}
	for _, file := range delivery.Files {
		name := filepath.Base(file.LocalPath)
		result, err := a.Artifacts.Distribute(ctx, ArtifactRequest{
			NodeID: target.NodeID, Name: name, SourceRef: file.LocalPath, DestinationPath: file.RemotePath,
		})
		if err != nil {
			return receipt, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge artifact distribution failed", Err: err}
		}
		if strings.TrimSpace(result.DistributionID) == "" {
			return receipt, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge artifact distribution returned no id"}
		}
		result, err = a.waitArtifact(ctx, result.DistributionID, result)
		if err != nil {
			return receipt, &reach.Error{Kind: reach.KindTransport, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge artifact placement was not confirmed", Err: err}
		}
		receipt.Files = append(receipt.Files, reach.DeliveredFile{Role: file.Role, RemotePath: file.RemotePath, SHA256: file.SHA256})
	}
	return receipt, nil
}

func (a *Adapter) waitArtifact(ctx context.Context, id string, initial ArtifactResult) (ArtifactResult, error) {
	if initial.Delivered {
		return initial, nil
	}
	if initial.Failed {
		return ArtifactResult{}, fmt.Errorf("distribution %s failed: %s", id, initial.Detail)
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ArtifactResult{}, ctx.Err()
		case <-ticker.C:
			result, err := a.Artifacts.Get(ctx, id)
			if err != nil {
				return ArtifactResult{}, err
			}
			if result.Delivered {
				return result, nil
			}
			if result.Failed {
				return ArtifactResult{}, fmt.Errorf("distribution %s failed: %s", id, result.Detail)
			}
		}
	}
}

// Negotiate reads the node record from the registry: online, dispatchable,
// protocol-compatible, the granted scopes and the node platform.
func (a *Adapter) Negotiate(ctx context.Context, target identity.TargetRef) (reach.Capabilities, error) {
	caps := reach.Capabilities{Transport: identity.TransportBridge}
	if a == nil || a.Client == nil {
		return caps, &reach.Error{Kind: reach.KindUnavailable, Transport: identity.TransportBridge, Target: target.Key(), Detail: "bridge adapter not configured"}
	}
	nodes, err := a.Client.List(ctx, a.timeout(reach.Command{}))
	if err != nil {
		return caps, classify(err, target)
	}
	for _, n := range nodes {
		if n.GetId() != target.NodeID {
			continue
		}
		if n.GetRevokedAt() != nil {
			return caps, &reach.Error{Kind: reach.KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "node enrollment revoked"}
		}
		caps.Online = n.GetOnline()
		caps.Scopes = append([]string(nil), n.GetScopes()...)
		caps.ProtocolVersion = n.GetRevision()
		caps.NativeCLI = n.GetDispatchable()
		if n.GetOs() != "" && n.GetArch() != "" {
			caps.Platform = strings.ToLower(n.GetOs()) + "/" + strings.ToLower(n.GetArch())
		}
		if !n.GetOnline() {
			return caps, &reach.Error{Kind: reach.KindTargetOffline, Transport: identity.TransportBridge, Target: target.Key(), Detail: "node offline"}
		}
		if !n.GetProtocolCompatible() || !n.GetDispatchable() {
			return caps, &reach.Error{Kind: reach.KindProtocolUnsupported, Transport: identity.TransportBridge, Target: target.Key(), Detail: "node protocol incompatible or not dispatchable"}
		}
		return caps, nil
	}
	return caps, &reach.Error{Kind: reach.KindEnrollmentRevoked, Transport: identity.TransportBridge, Target: target.Key(), Detail: "node not found in the registry"}
}
