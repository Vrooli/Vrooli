package registry

import (
	"time"

	"github.com/vrooli/api-core/targetmodel"
	internalpresence "vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"

	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

// domainToProto translates a domain Node into its wire shape, stamping the
// presence overlay (online + status) the handler computed from the live
// presence reader. The domain layer never imports proto; this is the single
// translation point (api-steer §7).
func domainToProto(n registry.Node, online, dispatchable bool, staleAfter time.Duration, facts ...internalpresence.ReadinessFacts) *registryv1.Node {
	readiness := internalpresence.ReadinessFacts{ChannelHeld: online, ProtocolCompatible: dispatchable, Dispatchable: dispatchable}
	if len(facts) > 0 {
		readiness = facts[0]
	}
	// LastSeenAt is durable evidence and is the source of truth for freshness.
	// The in-memory presence overlay can outlive a stale heartbeat, so never let
	// a held channel or a copied readiness default turn an old node fresh again.
	if staleAfter <= 0 {
		staleAfter = internalpresence.DefaultHeartbeatStaleAfter
	}
	heartbeatFresh, heartbeatAge := targetmodel.HeartbeatFresh(n.LastSeenAt, time.Now().UTC(), staleAfter)
	readiness.HeartbeatFresh = heartbeatFresh
	readiness.HeartbeatAge = heartbeatAge
	effectiveOnline := online && heartbeatFresh
	effectiveDispatchable := dispatchable && heartbeatFresh
	out := &registryv1.Node{
		Id:                    n.ID,
		Name:                  n.Name,
		Kind:                  kindProto(n.Kind),
		Os:                    n.OS,
		Arch:                  n.Arch,
		MachineArch:           n.MachineArch,
		BinaryArch:            n.BinaryArch,
		Revision:              n.Revision,
		Endpoint:              n.Endpoint,
		Capabilities:          append([]string(nil), n.Capabilities...),
		Scopes:                append([]string(nil), n.Scopes...),
		Online:                effectiveOnline,
		Status:                statusFor(n, effectiveOnline, effectiveDispatchable),
		CreatedAt:             timestamppb.New(n.CreatedAt),
		UpdatedAt:             timestamppb.New(n.UpdatedAt),
		RegistryRecordPresent: true,
		HeartbeatFresh:        readiness.HeartbeatFresh,
		HeartbeatAgeSeconds:   int64(readiness.HeartbeatAge / time.Second),
		ChannelHeld:           readiness.ChannelHeld,
		ProtocolCompatible:    readiness.ProtocolCompatible,
		Dispatchable:          effectiveDispatchable,
	}
	if !n.LastSeenAt.IsZero() {
		out.LastSeenAt = timestamppb.New(n.LastSeenAt)
	}
	if !n.RevokedAt.IsZero() {
		out.RevokedAt = timestamppb.New(n.RevokedAt)
	}
	return out
}

func kindProto(kind string) registryv1.NodeKind {
	switch kind {
	case registry.KindSSH:
		return registryv1.NodeKind_NODE_KIND_SSH
	case registry.KindAttached:
		return registryv1.NodeKind_NODE_KIND_ATTACHED
	case registry.KindControlPlane:
		return registryv1.NodeKind_NODE_KIND_CONTROL_PLANE
	default:
		return registryv1.NodeKind_NODE_KIND_AGENT
	}
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
