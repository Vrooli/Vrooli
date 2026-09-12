package registry

import (
	"context"
	"log"
	"time"

	"vrooli-bridge/internal/auth"
	internalpresence "vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"

	"connectrpc.com/connect"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

// Presence is the read seam the registry handler uses to overlay live
// online/offline state onto stored node records. Production wires the presence
// hub; tests substitute a fake. A nil Presence (the Phase-1-without-presence
// case) is treated as "everything offline" by the handler's nilSafe wrapper.
type Presence interface {
	// IsOnline reports whether the node currently holds a dial-out channel.
	IsOnline(nodeID string) bool
	// Dispatchable reports whether the node is online AND protocol-compatible
	// (not flagged). An online-but-flagged node reads NEEDS_UPDATE in the
	// overlay (OT-P1-001).
	Dispatchable(nodeID string) bool
}

// ReadinessPresence is an optional extension implemented by the live hub. It
// keeps the registry handler's existing presence seam backwards-compatible
// with focused fakes while exposing the independent operator facts.
type ReadinessPresence interface {
	Readiness(nodeID string) internalpresence.ReadinessFacts
}

type CapabilityPresence interface {
	Health(nodeID string) (internalpresence.HealthSnapshot, bool)
}

// CredentialRevoker severs a node's mutual-auth credential. The pairing service
// satisfies it; revocation calls it so a single RevokeNode kills durable
// identity AND the node's ability to authenticate (SECURITY.md atomic
// revocation). Optional — a nil revoker (no pairing wired) skips credential
// destruction.
type CredentialRevoker interface {
	RevokeCredential(ctx context.Context, nodeID string) error
}

// Disconnector force-closes a node's live dial-out channel. The presence hub
// satisfies it; revocation calls it so a revoked node's held SSE stream drops
// at once rather than lingering until the next reconnect.
type Disconnector interface {
	Disconnect(nodeID string) int
}

// Deps wires the seams the Connect registry handler needs.
type Deps struct {
	Service            registry.Service
	Presence           Presence
	Credentials        CredentialRevoker
	Disconnect         Disconnector
	Logger             *log.Logger
	PresenceStaleAfter time.Duration
}

type connectHandler struct {
	deps Deps
}

func (h *connectHandler) nodeProto(n registry.Node) *registryv1.Node {
	online := h.deps.Presence.IsOnline(n.ID)
	dispatchable := h.deps.Presence.Dispatchable(n.ID)
	if r, ok := h.deps.Presence.(ReadinessPresence); ok {
		out := domainToProto(n, online, dispatchable, h.deps.PresenceStaleAfter, r.Readiness(n.ID))
		if p, ok := h.deps.Presence.(CapabilityPresence); ok {
			applyCapabilityInventory(out, p, n.ID)
		}
		return out
	}
	out := domainToProto(n, online, dispatchable, h.deps.PresenceStaleAfter)
	if p, ok := h.deps.Presence.(CapabilityPresence); ok {
		applyCapabilityInventory(out, p, n.ID)
	}
	return out
}

// NewConnectHandler constructs the handler, defaulting the logger and the
// presence reader (nil → all offline).
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Presence == nil {
		d.Presence = offlinePresence{}
	}
	return &connectHandler{deps: d}
}

// offlinePresence is the nil-safe default: no live channels are known, so every
// node reads offline. Replaced by the real hub once presence is wired.
type offlinePresence struct{}

func (offlinePresence) IsOnline(string) bool { return false }

func (offlinePresence) Dispatchable(string) bool { return false }

func (h *connectHandler) RegisterNode(ctx context.Context, req *connect.Request[registryv1.RegisterNodeRequest]) (*connect.Response[registryv1.RegisterNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	node, err := h.deps.Service.Register(ctx, registry.RegisterInput{
		Name:         req.Msg.Name,
		Kind:         kindString(req.Msg.Kind),
		OS:           req.Msg.Os,
		Arch:         req.Msg.Arch,
		MachineArch:  req.Msg.MachineArch,
		BinaryArch:   req.Msg.BinaryArch,
		Endpoint:     req.Msg.Endpoint,
		Capabilities: req.Msg.Capabilities,
		Scopes:       req.Msg.Scopes,
	})
	if err != nil {
		connectErr := registry.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.RegisterNode: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&registryv1.RegisterNodeResponse{
		Node: h.nodeProto(node),
	}), nil
}

func kindString(kind registryv1.NodeKind) string {
	switch kind {
	case registryv1.NodeKind_NODE_KIND_SSH:
		return registry.KindSSH
	case registryv1.NodeKind_NODE_KIND_ATTACHED:
		return registry.KindAttached
	case registryv1.NodeKind_NODE_KIND_CONTROL_PLANE:
		return registry.KindControlPlane
	case registryv1.NodeKind_NODE_KIND_UNSPECIFIED:
		return ""
	default:
		return registry.KindAgent
	}
}

func (h *connectHandler) ListNodes(ctx context.Context, _ *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	nodes, err := h.deps.Service.List(ctx)
	if err != nil {
		h.deps.Logger.Printf("registry.ListNodes: %v", err)
		return nil, registry.ToConnectError(err)
	}
	resp := &registryv1.ListNodesResponse{Nodes: make([]*registryv1.Node, 0, len(nodes))}
	for _, n := range nodes {
		resp.Nodes = append(resp.Nodes, h.nodeProto(n))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetNode(ctx context.Context, req *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	node, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := registry.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.GetNode(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&registryv1.GetNodeResponse{
		Node: h.nodeProto(node),
	}), nil
}

func (h *connectHandler) UpdateNode(ctx context.Context, req *connect.Request[registryv1.UpdateNodeRequest]) (*connect.Response[registryv1.UpdateNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	node, err := h.deps.Service.Update(ctx, registry.UpdateInput{
		ID:           req.Msg.Id,
		Name:         req.Msg.Name,
		Endpoint:     req.Msg.Endpoint,
		Capabilities: req.Msg.Capabilities,
		Scopes:       req.Msg.Scopes,
		Revision:     req.Msg.Revision,
		Kind:         kindString(req.Msg.Kind),
	})
	if err != nil {
		connectErr := registry.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.UpdateNode(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&registryv1.UpdateNodeResponse{
		Node: h.nodeProto(node),
	}), nil
}

func (h *connectHandler) RevokeNode(ctx context.Context, req *connect.Request[registryv1.RevokeNodeRequest]) (*connect.Response[registryv1.RevokeNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	node, err := h.deps.Service.Revoke(ctx, req.Msg.Id)
	if err != nil {
		connectErr := registry.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("registry.RevokeNode(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}

	// Atomic revocation (SECURITY.md): durable identity is revoked above; now
	// sever the node's mutual-auth credential AND force-drop its live channel,
	// so its job/provisioning/heartbeat rights all die in this one operation.
	// Credential destruction is best-effort-logged: a failure must not leave the
	// node marked active, but the durable revoke already stands.
	if h.deps.Credentials != nil {
		if err := h.deps.Credentials.RevokeCredential(ctx, node.ID); err != nil {
			h.deps.Logger.Printf("registry.RevokeNode(%q): revoke credential: %v", node.ID, err)
		}
	}
	if h.deps.Disconnect != nil {
		h.deps.Disconnect.Disconnect(node.ID)
	}

	// A revoked node is offline by definition; do not consult presence.
	return connect.NewResponse(&registryv1.RevokeNodeResponse{
		Node: domainToProto(node, false, false, h.deps.PresenceStaleAfter),
	}), nil
}

func (h *connectHandler) RemoveNode(ctx context.Context, req *connect.Request[registryv1.RemoveNodeRequest]) (*connect.Response[registryv1.RemoveNodeResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if err := h.deps.Service.Remove(ctx, req.Msg.Id); err != nil {
		return nil, registry.ToConnectError(err)
	}
	return connect.NewResponse(&registryv1.RemoveNodeResponse{RemovedNodeId: req.Msg.Id}), nil
}

func (h *connectHandler) GetNodeReadiness(ctx context.Context, req *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeReadinessResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	node, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, registry.ToConnectError(err)
	}
	return connect.NewResponse(&registryv1.GetNodeReadinessResponse{Node: h.nodeProto(node)}), nil
}
