// enrollment.go holds the operations an owner performs on the fleet itself:
// listing the machines asking to join, deciding them, and changing or removing
// what an already-linked machine may do. They live beside the reach operations
// so a product surface that manages machines never has to build a second
// Bridge client with its own credential handling.
package nodereach

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
)

// Enrollment is the owner-visible state of joining: who is asking, and the
// postures the control plane offers for answering. Both halves come from one
// call because a console that renders a pending request must render the
// presets alongside it.
type Enrollment struct {
	Requests []*pairingv1.PairingRequest
	Presets  []*pairingv1.PermissionPreset
}

// ListEnrollment returns pending join requests and the control plane's
// permission presets. Decided requests are excluded unless includeDecided is
// set, because the owner-facing question is always "who is waiting".
func (c *Client) ListEnrollment(ctx context.Context, includeDecided bool, timeout time.Duration) (Enrollment, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return Enrollment{}, err
	}
	client := pairingconnect.NewPairingServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.ListPairingRequests(callCtx, connect.NewRequest(&pairingv1.ListPairingRequestsRequest{IncludeDecided: includeDecided}))
	if err != nil {
		return Enrollment{}, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return Enrollment{}, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no enrollment state")}
	}
	return Enrollment{
		Requests: append([]*pairingv1.PairingRequest(nil), resp.Msg.GetRequests()...),
		Presets:  append([]*pairingv1.PermissionPreset(nil), resp.Msg.GetPresets()...),
	}, nil
}

// DecideRequest is one owner decision on a pending join request. Approval
// carries the confirmation words the owner read off the joining machine; the
// control plane refuses an approval whose words do not match the value it
// derived from both public keys.
type DecideRequest struct {
	RequestID         string
	Approve           bool
	Scopes            []string
	ConfirmationWords []string
}

// Decide answers one pending join request and returns the minted node id when
// the request was approved.
func (c *Client) Decide(ctx context.Context, req DecideRequest, timeout time.Duration) (string, error) {
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		return "", &Error{Kind: ErrInvalidRequest, Err: errors.New("a join request id is required")}
	}
	if req.Approve && len(req.ConfirmationWords) == 0 {
		return "", &Error{Kind: ErrInvalidRequest, Err: errors.New("approval requires the confirmation words shown by the joining machine")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return "", err
	}
	client := pairingconnect.NewPairingServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.ApprovePairing(callCtx, connect.NewRequest(&pairingv1.ApprovePairingRequest{
		RequestId:         requestID,
		Approve:           req.Approve,
		Scopes:            append([]string(nil), req.Scopes...),
		ConfirmationWords: append([]string(nil), req.ConfirmationWords...),
	}))
	if err != nil {
		return "", c.decisionError(err)
	}
	if resp == nil || resp.Msg == nil {
		return "", &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no decision result")}
	}
	return resp.Msg.GetNodeId(), nil
}

// CodeRequest mints a single-use join code for a machine that discovery cannot
// reach. Name, when set, wins over whatever the joining machine proposes, and
// Scopes is the grant the machine holds the moment it redeems the code, so a
// code path is no less governed than an approved request.
type CodeRequest struct {
	Name       string
	Scopes     []string
	TTLSeconds int64
}

// IssueCode mints a single-use join code.
func (c *Client) IssueCode(ctx context.Context, req CodeRequest, timeout time.Duration) (*pairingv1.IssuePairingCodeResponse, error) {
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := pairingconnect.NewPairingServiceClient(c.transport(callCtx, baseURL), baseURL)
	resp, err := client.IssuePairingCode(callCtx, connect.NewRequest(&pairingv1.IssuePairingCodeRequest{
		Name:       strings.TrimSpace(req.Name),
		Scopes:     append([]string(nil), req.Scopes...),
		TtlSeconds: req.TTLSeconds,
	}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Err: errors.New("Bridge returned no join code")}
	}
	return resp.Msg, nil
}

// SetScopes replaces what one linked node is allowed to do. The registry
// treats the update as the desired post-state of the owner-editable surface,
// so the node's other fields are carried through unchanged.
func (c *Client) SetScopes(ctx context.Context, nodeID string, scopes []string, timeout time.Duration) (*registryv1.Node, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return nil, &Error{Kind: ErrInvalidRequest, Err: errors.New("a node id is required")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return nil, err
	}
	client := registryconnect.NewNodeRegistryServiceClient(c.transport(callCtx, baseURL), baseURL)
	current, err := client.GetNode(callCtx, connect.NewRequest(&registryv1.GetNodeRequest{Id: nodeID}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Node: nodeID, Err: err}
	}
	node := current.Msg.GetNode()
	if node == nil {
		return nil, &Error{Kind: ErrNodeNotFound, Node: nodeID, Err: errors.New("Bridge holds no such machine")}
	}
	resp, err := client.UpdateNode(callCtx, connect.NewRequest(&registryv1.UpdateNodeRequest{
		Id:           nodeID,
		Name:         node.GetName(),
		Endpoint:     node.GetEndpoint(),
		Capabilities: append([]string(nil), node.GetCapabilities()...),
		Scopes:       append([]string(nil), scopes...),
		Revision:     node.GetRevision(),
		Kind:         node.GetKind(),
	}))
	if err != nil {
		return nil, &Error{Kind: ErrTransport, Node: nodeID, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return nil, &Error{Kind: ErrTransport, Node: nodeID, Err: errors.New("Bridge returned no updated machine")}
	}
	return resp.Msg.GetNode(), nil
}

// Forget removes one node from the fleet. It is the owner's undo for a machine
// that is gone or was linked in error.
//
// The registry refuses to remove a node that is still authorized, because a
// record deleted while its credentials remain valid would leave a machine that
// can still act with nothing left to describe it. Revocation is therefore not a
// separate decision an operator makes first — it is the safety half of the same
// intent, so this performs both. Revoking an already-revoked node is accepted,
// which keeps a retry after a partial failure safe.
func (c *Client) Forget(ctx context.Context, nodeID string, timeout time.Duration) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return &Error{Kind: ErrInvalidRequest, Err: errors.New("a node id is required")}
	}
	callCtx, cancel := c.requestContext(ctx, timeout)
	defer cancel()
	baseURL, err := c.endpoint(callCtx)
	if err != nil {
		return err
	}
	client := registryconnect.NewNodeRegistryServiceClient(c.transport(callCtx, baseURL), baseURL)
	if _, err := client.RevokeNode(callCtx, connect.NewRequest(&registryv1.RevokeNodeRequest{Id: nodeID})); err != nil {
		var connectErr *connect.Error
		// A node already revoked is the state this step wants; anything else
		// stops the removal, because removing an authorized node is the outcome
		// the registry exists to prevent.
		if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeFailedPrecondition {
			return &Error{Kind: ErrTransport, Node: nodeID, Err: err}
		}
	}
	if _, err := client.RemoveNode(callCtx, connect.NewRequest(&registryv1.RemoveNodeRequest{Id: nodeID})); err != nil {
		return &Error{Kind: ErrTransport, Node: nodeID, Err: err}
	}
	return nil
}

// decisionError keeps the control plane's refusal legible. A mismatch between
// the words the owner confirmed and the words the control plane derived is a
// safety outcome an operator must see as such, not a generic transport fault.
func (c *Client) decisionError(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeInvalidArgument {
		return &Error{Kind: ErrInvalidRequest, Err: err}
	}
	return &Error{Kind: ErrTransport, Err: err}
}
