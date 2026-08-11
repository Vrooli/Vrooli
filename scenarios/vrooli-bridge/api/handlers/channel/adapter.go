// Package channel is the node-facing edge of the dial-out wire protocol: the
// SSE stream a node holds open (control-plane → node push) and the
// PresenceService heartbeat intake (node → control-plane). It translates the
// channel/presence proto wire types to the presence domain's shapes and feeds
// the in-memory presence hub. Phase 1 authenticates with a stub node credential
// (the ?node= query param); Phase 2 swaps in per-node Ed25519 mutual auth.
package channel

import (
	"vrooli-bridge/internal/compat"
	"vrooli-bridge/internal/presence"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// compatToProto translates the domain compatibility verdict into the wire
// CompatibilityStatus the heartbeat/handshake reports back to the node. An
// Unspecified domain verdict (a node that never reported a version) maps to OK
// on the wire — it is dispatchable, so the node should not see "needs update".
func compatToProto(s compat.Status) sharedv1.CompatibilityStatus {
	switch s {
	case compat.StatusNeedsUpdate:
		return sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_NEEDS_UPDATE
	case compat.StatusIncompatible:
		return sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_INCOMPATIBLE
	default:
		return sharedv1.CompatibilityStatus_COMPATIBILITY_STATUS_OK
	}
}

// protoHealthToDomain translates a wire HealthSnapshot into the presence
// domain's shape. A nil snapshot yields the zero value (not-ready).
func protoHealthToDomain(h *sharedv1.HealthSnapshot) presence.HealthSnapshot {
	if h == nil {
		return presence.HealthSnapshot{}
	}
	snap := presence.HealthSnapshot{
		ToolchainPresent:   h.GetToolchainPresent(),
		DiskHeadroomBytes:  h.GetDiskHeadroomBytes(),
		ContainerRuntimeUp: h.GetContainerRuntimeUp(),
		Details:            h.GetDetails(),
	}
	if ts := h.GetReportedAt(); ts != nil {
		snap.ReportedAt = ts.AsTime()
	}
	return snap
}
