package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/cli-core/cliutil"
)

// TargetResolver resolves a human node name to one point-in-time shared
// target observation. Implementations may read Bridge's registry, but the
// discovery package deliberately depends on the provider-neutral interface.
type TargetResolver interface {
	ResolveTarget(context.Context, string) (targetmodel.Target, error)
}

// RelayRequest is the provider-neutral request sent over a node transport.
// Scenario is the target scenario name after address parsing; Command and
// Args retain the typed command shape used by bridge admission.
type RelayRequest struct {
	NodeID           string
	Scenario         string
	Command          string
	Args             []string
	TimeoutSeconds   int64
	MaxResponseBytes uint64
}

type RelayResponse struct {
	CorrelationID string
	Outcome       string
	Data          []byte
	Reason        string
	ExitCode      int32
	TotalBytes    uint64
}

// RelayTransport is the only transport-specific dependency of Resolver. A
// bridge adapter owns Connect/authentication and returns the same bounded
// response that the channel relay produced.
type RelayTransport interface {
	Call(context.Context, RelayRequest) (RelayResponse, error)
}

// ScenarioResolution describes either a local URL or a completed remote
// relay call. A remote resolution intentionally has no URL: nodes dial out,
// so inventing an inbound endpoint would violate the bridge contract.
type ScenarioResolution struct {
	Address   string
	Node      string
	Scenario  string
	Variant   string
	Target    targetmodel.Target
	Transport targetmodel.TransportKind
	URL       string
	Response  RelayResponse
}

// ResolveScenario applies the node axis to discovery. A local address follows
// the existing port lookup path unchanged. An addressed node is checked from
// the shared target observation, admitted against its scopes, and then sent
// through the node's signed relay transport.
func (r *Resolver) ResolveScenario(ctx context.Context, address, portKey, command string, args []string) (ScenarioResolution, error) {
	node, scenario, variant, err := cliutil.SplitAddress(address)
	if err != nil {
		return ScenarioResolution{}, &Error{Kind: ErrInvalidInput, Scenario: address, PortKey: portKey, Err: err}
	}

	localName := scenario
	if variant != "" {
		localName = scenario + "@" + variant
	}
	if node == "" {
		url, err := r.ResolveScenarioURL(ctx, localName, portKey)
		if err != nil {
			return ScenarioResolution{}, err
		}
		return ScenarioResolution{
			Address: address, Scenario: scenario, Variant: variant,
			Transport: targetmodel.TransportLocal, URL: url,
		}, nil
	}

	command = strings.TrimSpace(command)
	if command == "" {
		command = "scenario status"
	}
	if r.targetResolver == nil {
		return ScenarioResolution{}, r.remoteError(ErrRemoteTransportUnavailable, node, localName, command, errors.New("target resolver is not configured"))
	}
	target, err := r.targetResolver.ResolveTarget(ctx, node)
	if err != nil {
		return ScenarioResolution{}, r.remoteError(ErrNodeUnpaired, node, localName, command, err)
	}
	if target.Revoked || target.BridgeTrust != nil && !target.BridgeTrust.Registered {
		return ScenarioResolution{}, r.remoteError(ErrNodeUnpaired, node, localName, command, errors.New("node is not paired or has been revoked"))
	}
	if !target.Available || !target.Transport.Available || target.BridgeTrust != nil && !target.BridgeTrust.Online {
		reason := strings.TrimSpace(target.Reason)
		if reason == "" {
			reason = "node is offline or not dispatchable"
		}
		return ScenarioResolution{}, r.remoteError(ErrNodeOffline, node, localName, command, errors.New(reason))
	}
	if r.commandScope == nil {
		return ScenarioResolution{}, r.remoteError(ErrRemoteTransportUnavailable, node, localName, command, errors.New("catalog command-scope resolver is not configured"))
	}
	required, ok := r.commandScope(command)
	if !ok || strings.TrimSpace(required) == "" {
		return ScenarioResolution{}, r.remoteError(ErrNodeOutOfScope, node, localName, command, errors.New("command is not present in the derived scope catalog"))
	}
	if !scopecatalog.Resolve(target.Scopes, required) {
		return ScenarioResolution{}, r.remoteError(ErrNodeOutOfScope, node, localName, command, fmt.Errorf("node lacks required scope %q", required))
	}
	_, effect, concrete := strings.Cut(required, ":")
	if !concrete || strings.TrimSpace(effect) == "" {
		return ScenarioResolution{}, r.remoteError(ErrNodeOutOfScope, node, localName, command, fmt.Errorf("catalog returned malformed required scope %q", required))
	}
	transportScope := "vrooli-bridge:" + effect
	if !scopecatalog.Resolve(target.Scopes, transportScope) {
		return ScenarioResolution{}, r.remoteError(ErrNodeOutOfScope, node, localName, command, fmt.Errorf("node lacks required transport scope %q", transportScope))
	}
	if r.relay == nil {
		return ScenarioResolution{}, r.remoteError(ErrRemoteTransportUnavailable, node, localName, command, errors.New("relay transport is not configured"))
	}
	nodeID := strings.TrimSpace(target.NodeID)
	if nodeID == "" {
		nodeID = target.ID
	}
	response, err := r.relay.Call(ctx, RelayRequest{
		NodeID: nodeID, Scenario: localName, Command: command,
		Args: append([]string(nil), args...),
	})
	if err != nil {
		return ScenarioResolution{}, r.remoteError(ErrRemoteCallFailed, node, localName, command, err)
	}
	return ScenarioResolution{
		Address: address, Node: node, Scenario: scenario, Variant: variant,
		Target: target, Transport: targetmodel.TransportBridge, Response: response,
	}, nil
}

func (r *Resolver) remoteError(kind ErrorKind, node, scenario, command string, err error) *Error {
	return &Error{Kind: kind, Node: node, Scenario: scenario, PortKey: command, Err: err}
}

func (s ScenarioResolution) Validate() error {
	if strings.TrimSpace(s.Scenario) == "" {
		return fmt.Errorf("scenario resolution: scenario is required")
	}
	if s.Transport == targetmodel.TransportLocal && strings.TrimSpace(s.URL) == "" {
		return fmt.Errorf("scenario resolution: local URL is required")
	}
	if s.Transport == targetmodel.TransportBridge && strings.TrimSpace(s.Node) == "" {
		return fmt.Errorf("scenario resolution: bridge node is required")
	}
	return nil
}
