package registry

import (
	"vrooli-bridge/internal/registry"

	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

// domainToProto translates a domain Node into its wire shape, stamping the
// presence overlay (online + status) the handler computed from the live
// presence reader. The domain layer never imports proto; this is the single
// translation point (api-steer §7).
func domainToProto(n registry.Node, online, dispatchable bool) *registryv1.Node {
	out := &registryv1.Node{
		Id:           n.ID,
		Name:         n.Name,
		Os:           n.OS,
		Arch:         n.Arch,
		Revision:     n.Revision,
		Endpoint:     n.Endpoint,
		Capabilities: append([]string(nil), n.Capabilities...),
		Scopes:       append([]string(nil), n.Scopes...),
		Online:       online,
		Status:       statusFor(n, online, dispatchable),
		CreatedAt:    timestamppb.New(n.CreatedAt),
		UpdatedAt:    timestamppb.New(n.UpdatedAt),
	}
	if !n.LastSeenAt.IsZero() {
		out.LastSeenAt = timestamppb.New(n.LastSeenAt)
	}
	if !n.RevokedAt.IsZero() {
		out.RevokedAt = timestamppb.New(n.RevokedAt)
	}
	return out
}

// statusFor computes the overlaid status. Revocation is terminal and wins over
// any lingering connection; otherwise the live presence decides: an online node
// whose agent protocol is flagged (online but NOT dispatchable) reads
// NEEDS_UPDATE (OT-P1-001 protocol-compat gating); a fully-compatible online
// node reads ONLINE; everything else OFFLINE.
func statusFor(n registry.Node, online, dispatchable bool) registryv1.NodeStatus {
	switch {
	case n.Revoked():
		return registryv1.NodeStatus_NODE_STATUS_REVOKED
	case online && !dispatchable:
		return registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE
	case online:
		return registryv1.NodeStatus_NODE_STATUS_ONLINE
	default:
		return registryv1.NodeStatus_NODE_STATUS_OFFLINE
	}
}
