package provision

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/provision"

	"connectrpc.com/connect"

	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`
// (cli-core/cliutil.DryRunHeader). SyncToRevision honours it by validating then
// short-circuiting before the first side effect.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect provision handler needs. Verifier enforces
// per-node mutual auth on the node-facing ReportProvisionEvent (nil disables it,
// the pre-pairing stub). Operator verbs are owner-gated via auth.RequireOwner.
type Deps struct {
	Service  provision.Service
	Verifier *nodeauth.Verifier
	Logger   *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler, defaulting the logger.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// SyncToRevision validates the node + revision, then (unless a dry-run) creates
// a durable provisioning op, audits it, and pushes the privileged
// ProvisionCommand to the node. Owner-gated.
func (h *connectHandler) SyncToRevision(ctx context.Context, req *connect.Request[provisionv1.SyncToRevisionRequest]) (*connect.Response[provisionv1.SyncToRevisionResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if actor == "" {
		actor = "owner"
	}

	dryRun := req.Header().Get(dryRunHeader) == "true"

	dec, err := h.deps.Service.Sync(ctx, provision.SyncInput{
		Actor:            actor,
		NodeID:           req.Msg.NodeId,
		TargetRevision:   req.Msg.TargetRevision,
		RollbackRevision: req.Msg.RollbackRevision,
		TimeoutSeconds:   req.Msg.TimeoutSeconds,
		DryRun:           dryRun,
	})
	if err != nil {
		connectErr := provision.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("provision.SyncToRevision(node=%q rev=%q): %v", req.Msg.NodeId, req.Msg.TargetRevision, err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&provisionv1.SyncToRevisionResponse{
		OpId:             dec.OpID,
		DryRun:           dec.DryRun,
		NodeId:           dec.NodeID,
		TargetRevision:   dec.TargetRevision,
		RollbackRevision: dec.RollbackRevision,
	}), nil
}

func (h *connectHandler) GetProvisioningOp(ctx context.Context, req *connect.Request[provisionv1.GetProvisioningOpRequest]) (*connect.Response[provisionv1.GetProvisioningOpResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	op, events, err := h.deps.Service.GetOp(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapErr("GetProvisioningOp", req.Msg.Id, err)
	}
	resp := &provisionv1.GetProvisioningOpResponse{
		Op:     domainOpToProto(op),
		Events: make([]*provisionv1.ProvisionEvent, 0, len(events)),
	}
	for _, ev := range events {
		resp.Events = append(resp.Events, domainEventToProto(ev))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListProvisioningOps(ctx context.Context, req *connect.Request[provisionv1.ListProvisioningOpsRequest]) (*connect.Response[provisionv1.ListProvisioningOpsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.ListOps(ctx, provision.ListFilter{
		NodeID: req.Msg.NodeId,
		Limit:  int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("provision.ListProvisioningOps: %v", err)
		return nil, provision.ToConnectError(err)
	}
	resp := &provisionv1.ListProvisioningOpsResponse{Ops: make([]*provisionv1.ProvisioningOp, 0, len(list))}
	for _, op := range list {
		resp.Ops = append(resp.Ops, domainOpToProto(op))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) WaitProvisioningOp(ctx context.Context, req *connect.Request[provisionv1.WaitProvisioningOpRequest]) (*connect.Response[provisionv1.WaitProvisioningOpResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	timeout := time.Duration(req.Msg.TimeoutSeconds) * time.Second
	op, timedOut, err := h.deps.Service.Wait(ctx, req.Msg.Id, timeout)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, h.mapErr("WaitProvisioningOp", req.Msg.Id, err)
	}
	return connect.NewResponse(&provisionv1.WaitProvisioningOpResponse{
		Op:       domainOpToProto(op),
		TimedOut: timedOut,
	}), nil
}

func (h *connectHandler) GetNodeVersion(ctx context.Context, req *connect.Request[provisionv1.GetNodeVersionRequest]) (*connect.Response[provisionv1.GetNodeVersionResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	v, err := h.deps.Service.GetNodeVersion(ctx, req.Msg.NodeId)
	if err != nil {
		var none provision.ErrNoNodeVersion
		if errors.As(err, &none) {
			// A never-provisioned node is not an error — report has_version=false.
			return connect.NewResponse(&provisionv1.GetNodeVersionResponse{HasVersion: false}), nil
		}
		return nil, h.mapErr("GetNodeVersion", req.Msg.NodeId, err)
	}
	return connect.NewResponse(&provisionv1.GetNodeVersionResponse{
		Version:    domainVersionToProto(v),
		HasVersion: true,
	}), nil
}

// ReportProvisionEvent ingests one ProvisionEvent streamed back from the
// node-agent's privileged helper. It is NODE-facing: the agent signs the call
// with its per-node Ed25519 credential, and a node may only report against its
// OWN ops (a cross-node forge is rejected). A terminal EXIT flips the op
// terminal and wakes block-once waiters.
func (h *connectHandler) ReportProvisionEvent(ctx context.Context, req *connect.Request[provisionv1.ReportProvisionEventRequest]) (*connect.Response[provisionv1.ReportProvisionEventResponse], error) {
	ev := req.Msg.GetEvent()
	if ev == nil || strings.TrimSpace(ev.GetOpId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a provision event with an op_id is required"))
	}

	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(
			req.Header().Get(nodeauth.HeaderNode),
			req.Header().Get(nodeauth.HeaderTS),
			req.Header().Get(nodeauth.HeaderSig),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		// A node may only report against its own ops. Look the op up and reject a
		// cross-node report. An unknown op is acknowledged (accepted=false) so a
		// confused node stops re-sending.
		op, _, err := h.deps.Service.GetOp(ctx, ev.GetOpId())
		if err != nil {
			var notFound provision.ErrOpNotFound
			if errors.As(err, &notFound) {
				return connect.NewResponse(&provisionv1.ReportProvisionEventResponse{Accepted: false}), nil
			}
			h.deps.Logger.Printf("provision.ReportProvisionEvent: lookup op %q: %v", ev.GetOpId(), err)
			return nil, provision.ToConnectError(err)
		}
		if op.NodeID != proof.NodeID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("a node may only report events for its own provisioning ops"))
		}
	}

	accepted, err := h.deps.Service.AppendEvent(ctx, protoEventToDomain(ev))
	if err != nil {
		var invalid provision.ErrInvalidOp
		if errors.As(err, &invalid) {
			return nil, connect.NewError(connect.CodeInvalidArgument, invalid)
		}
		h.deps.Logger.Printf("provision.ReportProvisionEvent(op %q): %v", ev.GetOpId(), err)
		return nil, provision.ToConnectError(err)
	}
	return connect.NewResponse(&provisionv1.ReportProvisionEventResponse{Accepted: accepted}), nil
}

// mapErr logs internal errors and returns the Connect translation.
func (h *connectHandler) mapErr(op, id string, err error) error {
	connectErr := provision.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("provision.%s(%q): %v", op, id, err)
	}
	return connectErr
}
