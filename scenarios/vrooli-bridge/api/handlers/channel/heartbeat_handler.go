package channel

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/relay"
	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/session"

	"connectrpc.com/connect"

	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// LastSeenRecorder is the narrow registry seam the heartbeat handler uses to
// persist a node's last-seen timestamp (so "last seen 2h ago" survives a
// control-plane restart). The registry sqlite repository satisfies it; tests
// substitute a fake. A heartbeat from an unknown node is a no-op there.
type LastSeenRecorder interface {
	TouchLastSeen(ctx context.Context, nodeID string, t time.Time) error
}

type CapabilityInventoryRecorder interface {
	UpdateCapabilityInventory(context.Context, string, []registry.CapabilityObservation, time.Time) error
}

// HeartbeatDeps wires the seams the PresenceService handler needs.
type HeartbeatDeps struct {
	Hub                 *presence.Hub
	LastSeen            LastSeenRecorder
	CapabilityInventory CapabilityInventoryRecorder
	DeliveryAckRecorder interface {
		RecordDeliveryAck(context.Context, runs.DeliveryAck) error
	}
	Audit audit.Sink
	// Verifier, when set, enforces mutual auth: every heartbeat must carry a
	// valid X-Bridge-* signature from the node, verified against its stored
	// credential. Nil disables enforcement (pre-pairing / Phase-1 stub).
	Verifier        *nodeauth.Verifier
	Logger          *log.Logger
	SessionManager  *session.Manager
	SessionAuth     auth.Validator
	SessionRegistry registry.Service
	// SessionPush is the signed, node-facing session transport. It is kept as
	// a callback so the channel handler does not own presence or signing state.
	SessionPush    func(context.Context, string, string, *sessionv1.Frame) error
	RelayResponses interface {
		Deliver(context.Context, string, relay.Response) error
	}
	CredentialReceipts interface {
		RecordCredentialReceipt(context.Context, string, string, int64, bool, string) error
	}
	ScenarioResponses interface {
		Deliver(string, []byte, string, bool, bool, string) error
	}
}

// HeartbeatOption adds a production side effect without forcing focused
// channel tests to construct the complete runs and audit stack.
type HeartbeatOption func(*HeartbeatDeps)

func WithDeliveryAckRecorder(recorder interface {
	RecordDeliveryAck(context.Context, runs.DeliveryAck) error
},
) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.DeliveryAckRecorder = recorder }
}

func WithAuditSink(sink audit.Sink) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.Audit = sink }
}

func WithSessionPush(push func(context.Context, string, string, *sessionv1.Frame) error) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.SessionPush = push }
}

func WithRelayResponseSink(sink interface {
	Deliver(context.Context, string, relay.Response) error
},
) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.RelayResponses = sink }
}

func WithCredentialReceiptRecorder(recorder interface {
	RecordCredentialReceipt(context.Context, string, string, int64, bool, string) error
}) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.CredentialReceipts = recorder }
}

func WithScenarioResponseSink(sink interface {
	Deliver(string, []byte, string, bool, bool, string) error
}) HeartbeatOption {
	return func(d *HeartbeatDeps) { d.ScenarioResponses = sink }
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
	if h.deps.CapabilityInventory != nil {
		inventory := make([]registry.CapabilityObservation, 0, len(snap.Capabilities))
		for _, item := range snap.Capabilities {
			inventory = append(inventory, registry.CapabilityObservation{Capability: item.Capability, ID: item.ID, Label: item.Label, State: item.State, Path: item.Path, Version: item.Version, ProbedAt: item.ProbedAt, Detail: item.Detail})
		}
		if err := h.deps.CapabilityInventory.UpdateCapabilityInventory(ctx, nodeID, inventory, snap.ReportedAt); err != nil {
			h.deps.Logger.Printf("heartbeat capability inventory for %q: %v", nodeID, err)
		}
	}

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
		Compatibility: compatToProto(h.deps.Hub.Compatibility(nodeID)),
	}), nil
}

// ReportDeliveryAck records a node receipt after authenticating the node and
// checking that the frame was actually enqueued for that same node. The ack
// happens before the agent starts the represented operation, so it is a
// transport fact rather than a proxy for job completion.
func (h *heartbeatHandler) ReportDeliveryAck(ctx context.Context, req *connect.Request[presencev1.ReportDeliveryAckRequest]) (*connect.Response[presencev1.ReportDeliveryAckResponse], error) {
	ack := req.Msg.GetAck()
	if ack == nil || ack.GetFrameId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("delivery acknowledgement requires frame_id"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nodeauth.ErrMissingProof)
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(
			nodeID,
			req.Header().Get(nodeauth.HeaderTS),
			req.Header().Get(nodeauth.HeaderSig),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}
	receivedAt := time.Time{}
	if ts := ack.GetReceivedAt(); ts != nil {
		receivedAt = ts.AsTime()
	}
	deliveryAck := presence.DeliveryAck{
		NodeID: nodeID, FrameID: ack.GetFrameId(), RunID: ack.GetRunId(),
		OpID: ack.GetOpId(), ReceivedAt: receivedAt,
	}
	if err := h.deps.Hub.RecordDeliveryAck(deliveryAck); err != nil {
		h.auditRejectedDeliveryAck(ctx, nodeID, ack, err)
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if h.deps.DeliveryAckRecorder != nil {
		if err := h.deps.DeliveryAckRecorder.RecordDeliveryAck(ctx, runs.DeliveryAck{
			NodeID: deliveryAck.NodeID, FrameID: deliveryAck.FrameID, RunID: deliveryAck.RunID,
			OpID: deliveryAck.OpID, ReceivedAt: deliveryAck.ReceivedAt,
		}); err != nil {
			h.deps.Logger.Printf("channel.ReportDeliveryAck: persist %q: %v", deliveryAck.FrameID, err)
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
	}
	return connect.NewResponse(&presencev1.ReportDeliveryAckResponse{Accepted: true}), nil
}

// ReportRelayResponse accepts only a response authenticated by the node that
// owns the correlation. The broker then performs the second binding check
// (correlation -> node) before waking the waiting relay call.
func (h *heartbeatHandler) ReportRelayResponse(ctx context.Context, req *connect.Request[presencev1.ReportRelayResponseRequest]) (*connect.Response[presencev1.ReportRelayResponseResponse], error) {
	response := req.Msg.GetResponse()
	if response == nil || response.GetCorrelationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("relay response requires correlation_id"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nodeauth.ErrMissingProof)
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(nodeID, req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}
	kind := relayResponseKind(response.GetKind())
	if kind == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("relay response kind is required"))
	}
	if h.deps.RelayResponses == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("relay response transport unavailable"))
	}
	if err := h.deps.RelayResponses.Deliver(ctx, nodeID, relay.Response{
		CorrelationID: response.GetCorrelationId(),
		Kind:          kind,
		Sequence:      response.GetSequence(),
		Data:          append([]byte(nil), response.GetData()...),
		Reason:        response.GetReason(),
		ExitCode:      response.GetExitCode(),
		TotalBytes:    response.GetTotalBytes(),
	}); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewResponse(&presencev1.ReportRelayResponseResponse{Accepted: true}), nil
}

// ReportCredentialReceipt records only metadata from a node-side credential
// ingest. Authentication is the same node proof used by every other
// node-facing RPC; the receipt deliberately contains no credential value.
func (h *heartbeatHandler) ReportCredentialReceipt(ctx context.Context, req *connect.Request[presencev1.ReportCredentialReceiptRequest]) (*connect.Response[presencev1.ReportCredentialReceiptResponse], error) {
	receipt := req.Msg.GetReceipt()
	if receipt == nil || receipt.GetGrantId() == "" || receipt.GetNodeId() == "" || receipt.GetLogicalId() == "" || receipt.GetField() == "" || receipt.GetGeneration() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("credential receipt requires grant, node, address, and positive generation"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" || nodeID != receipt.GetNodeId() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("credential receipt node proof mismatch"))
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(nodeID, req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}
	if h.deps.CredentialReceipts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("credential receipt recorder unavailable"))
	}
	if err := h.deps.CredentialReceipts.RecordCredentialReceipt(ctx, receipt.GetGrantId(), nodeID, receipt.GetGeneration(), receipt.GetAccepted(), receipt.GetReason()); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewResponse(&presencev1.ReportCredentialReceiptResponse{Accepted: true}), nil
}

// ReportScenarioResponse accepts one bounded response from the node-side
// scenario proxy and binds delivery to the authenticated node through the
// scenario broker. The request body is opaque to Bridge.
func (h *heartbeatHandler) ReportScenarioResponse(ctx context.Context, req *connect.Request[presencev1.ReportScenarioResponseRequest]) (*connect.Response[presencev1.ReportScenarioResponseResponse], error) {
	response := req.Msg.GetResponse()
	if response == nil || response.GetCorrelationId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario response requires correlation_id"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nodeauth.ErrMissingProof)
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(nodeID, req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}
	if h.deps.ScenarioResponses == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("scenario response transport unavailable"))
	}
	if err := h.deps.ScenarioResponses.Deliver(response.GetCorrelationId(), response.GetResponse(), response.GetError(), response.GetTimedOut(), response.GetTruncated(), nodeID); err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewResponse(&presencev1.ReportScenarioResponseResponse{Accepted: true}), nil
}

func relayResponseKind(kind sharedv1.RelayResponseKind) string {
	switch kind {
	case sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_ACCEPTED:
		return relay.KindAccepted
	case sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_DATA:
		return relay.KindData
	case sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_COMPLETED:
		return relay.KindCompleted
	case sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_FAILED:
		return relay.KindFailed
	case sharedv1.RelayResponseKind_RELAY_RESPONSE_KIND_TERMINATED:
		return relay.KindTerminated
	default:
		return ""
	}
}

// ReportSessionFrame is the node-to-control-plane half of the interactive
// session transport. It is authenticated with the same per-node proof as
// heartbeats and is bound to the Bridge session's registered node before a
// byte is delivered to the owner WebSocket.
func (h *heartbeatHandler) ReportSessionFrame(ctx context.Context, req *connect.Request[presencev1.ReportSessionFrameRequest]) (*connect.Response[presencev1.ReportSessionFrameResponse], error) {
	envelope := req.Msg.GetFrame()
	if envelope == nil || envelope.GetSessionId() == "" || envelope.GetFrame() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session frame requires session_id and frame"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nodeauth.ErrMissingProof)
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(nodeID, req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
	}
	if h.deps.SessionManager == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("session transport unavailable"))
	}
	state, err := h.deps.SessionManager.Get(envelope.GetSessionId())
	if err != nil || state.NodeID != nodeID {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("session is not owned by this node"))
	}
	frame := envelope.GetFrame()
	switch payload := frame.Payload.(type) {
	case *sessionv1.Frame_Data:
		if err := h.deps.SessionManager.DeliverOutput(ctx, state.ID, payload.Data.GetSequence(), payload.Data.GetData()); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		h.recordSessionOutput(ctx, state, payload.Data.GetData())
	case *sessionv1.Frame_Close:
		if err := h.deps.SessionManager.Close(ctx, state.ID, payload.Close.GetReason()); err != nil && !errors.Is(err, session.ErrUnknown) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("node session frame must be data or close"))
	}
	return connect.NewResponse(&presencev1.ReportSessionFrameResponse{Accepted: true}), nil
}

func (h *heartbeatHandler) recordSessionOutput(ctx context.Context, state session.State, data []byte) {
	if h.deps.Audit == nil || len(data) == 0 {
		return
	}
	const maxAuditBytes = 64 * 1024
	truncated := len(data) > maxAuditBytes
	if truncated {
		data = data[:maxAuditBytes]
	}
	detail := base64.StdEncoding.EncodeToString(data)
	if truncated {
		detail += ":truncated"
	}
	if _, err := h.deps.Audit.Append(ctx, audit.Record{
		Action: audit.ActionSessionDataOut, Outcome: audit.OutcomeCompleted,
		Actor: "node:" + state.NodeID, NodeID: state.NodeID, RunID: state.ID,
		Detail: fmt.Sprintf("out:%s", detail),
	}); err != nil && h.deps.Logger != nil {
		h.deps.Logger.Printf("channel.ReportSessionFrame: audit output: %v", err)
	}
}

func (h *heartbeatHandler) auditRejectedDeliveryAck(ctx context.Context, nodeID string, ack *sharedv1.DeliveryAck, reason error) {
	if h.deps.Audit == nil {
		return
	}
	_, err := h.deps.Audit.Append(ctx, audit.Record{
		Action: audit.ActionDispatch, Actor: "node:" + nodeID, NodeID: nodeID,
		Outcome: audit.OutcomeRejected, Detail: "delivery acknowledgement rejected: " + reason.Error(),
		RunID: ack.GetRunId(),
	})
	if err != nil {
		h.deps.Logger.Printf("channel.ReportDeliveryAck: audit rejected receipt: %v", err)
	}
}
