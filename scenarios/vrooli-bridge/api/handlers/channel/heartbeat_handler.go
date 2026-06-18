package channel

import (
	"context"
	"errors"
	"log"
	"time"

	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"

	"connectrpc.com/connect"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
)

// LastSeenRecorder is the narrow registry seam the heartbeat handler uses to
// persist a node's last-seen timestamp (so "last seen 2h ago" survives a
// control-plane restart). The registry sqlite repository satisfies it; tests
// substitute a fake. A heartbeat from an unknown node is a no-op there.
type LastSeenRecorder interface {
	TouchLastSeen(ctx context.Context, nodeID string, t time.Time) error
}

// HeartbeatDeps wires the seams the PresenceService handler needs.
type HeartbeatDeps struct {
	Hub      *presence.Hub
	LastSeen LastSeenRecorder
	// Verifier, when set, enforces mutual auth: every heartbeat must carry a
	// valid X-Bridge-* signature from the node, verified against its stored
	// credential. Nil disables enforcement (pre-pairing / Phase-1 stub).
	Verifier *nodeauth.Verifier
	Logger   *log.Logger
}

type heartbeatHandler struct {
	deps HeartbeatDeps
}

// NewHeartbeatHandler constructs the PresenceService Connect handler.
func NewHeartbeatHandler(d HeartbeatDeps) *heartbeatHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &heartbeatHandler{deps: d}
}

// ReportHeartbeat records the node's liveness + self-reported readiness in the
// presence hub and persists its last-seen timestamp. It returns the
// compatibility verdict (OK in Phase 1; the handshake negotiation in Phase 2
// flips it to NEEDS_UPDATE for a stale agent).
func (h *heartbeatHandler) ReportHeartbeat(ctx context.Context, req *connect.Request[presencev1.ReportHeartbeatRequest]) (*connect.Response[presencev1.ReportHeartbeatResponse], error) {
	hb := req.Msg.GetHeartbeat()
	nodeID := hb.GetNodeId()
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingNodeID)
	}

	// Mutual auth: verify the node's signature over (node_id, timestamp) against
	// its stored credential. A failure (missing/forged/stale proof, unknown or
	// revoked node) is Unauthenticated, so a revoked node's heartbeats stop
	// being accepted immediately.
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(
			req.Header().Get(nodeauth.HeaderNode),
			req.Header().Get(nodeauth.HeaderTS),
			req.Header().Get(nodeauth.HeaderSig),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if proof.NodeID != nodeID {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("auth node id does not match heartbeat node id"))
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}

	snap := protoHealthToDomain(hb.GetHealth())
	h.deps.Hub.Heartbeat(nodeID, snap)

	if h.deps.LastSeen != nil {
		seenAt := snap.ReportedAt
		if seenAt.IsZero() {
			seenAt = time.Now()
		}
		if err := h.deps.LastSeen.TouchLastSeen(ctx, nodeID, seenAt); err != nil {
			// Persisting last-seen is best-effort; a failure must not drop the
			// heartbeat (presence is already updated in memory).
			h.deps.Logger.Printf("channel.ReportHeartbeat: touch last_seen for %q: %v", nodeID, err)
		}
	}

	return connect.NewResponse(&presencev1.ReportHeartbeatResponse{
		Compatibility: channelv1.CompatibilityStatus_COMPATIBILITY_STATUS_OK,
	}), nil
}
