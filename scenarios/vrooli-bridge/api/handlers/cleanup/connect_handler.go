package cleanup

import (
	"context"
	"errors"
	"log"
	"strings"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/cleanup"
	"vrooli-bridge/internal/nodeauth"

	"connectrpc.com/connect"

	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
)

type connectHandler struct{ deps Deps }

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func ownerID(ctx context.Context) (string, error) {
	id, err := auth.RequireOwner(ctx)
	if err != nil {
		return "", auth.ToConnectError(err)
	}
	if id.OwnerID != "" {
		return id.OwnerID, nil
	}
	if id.Email != "" {
		return id.Email, nil
	}
	return "owner", nil
}

func (h *connectHandler) PrepareCleanup(ctx context.Context, req *connect.Request[cleanupv1.PrepareCleanupRequest]) (*connect.Response[cleanupv1.PrepareCleanupResponse], error) {
	actor, err := ownerID(ctx)
	if err != nil {
		return nil, err
	}
	target, err := h.deps.Service.Prepare(ctx, cleanup.StartInput{MachineID: req.Msg.GetMachineId(), NodeID: req.Msg.GetNodeId(), Target: req.Msg.GetTarget(), Scope: req.Msg.GetScope()}, actor)
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.PrepareCleanupResponse{Target: &cleanupv1.CleanupTarget{MachineId: target.MachineID, NodeId: target.NodeID, Target: target.Target, Scope: target.Scope, Transport: target.Transport, TransportReason: target.TransportReason, OperatorId: target.OperatorID, SealingPublicKey: append([]byte(nil), target.SealingPublicKey...), OperationId: target.OperationID, Capabilities: append([]string(nil), target.Capabilities...), ApprovedScopes: append([]string(nil), target.ApprovedScopes...)}}), nil
}

func (h *connectHandler) ProvisionBreakGlass(ctx context.Context, req *connect.Request[cleanupv1.ProvisionBreakGlassRequest]) (*connect.Response[cleanupv1.ProvisionBreakGlassResponse], error) {
	actor, err := ownerID(ctx)
	if err != nil {
		return nil, err
	}
	operator := strings.TrimSpace(req.Msg.GetOperatorId())
	if operator != "" && operator != actor {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("operator_id must match the authenticated owner"))
	}
	operator = actor
	op, err := h.deps.Service.ProvisionBreakGlass(ctx, cleanup.ProvisionInput{MachineID: req.Msg.GetMachineId(), NodeID: req.Msg.GetNodeId(), Target: req.Msg.GetTarget(), Scope: req.Msg.GetScope(), OperationID: req.Msg.GetOperationId(), SealedPassphrase: req.Msg.GetSealedPassphrase(), OperatorID: operator})
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.ProvisionBreakGlassResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) ResetBreakGlass(ctx context.Context, req *connect.Request[cleanupv1.ResetBreakGlassRequest]) (*connect.Response[cleanupv1.ResetBreakGlassResponse], error) {
	actor, err := ownerID(ctx)
	if err != nil {
		return nil, err
	}
	op, err := h.deps.Service.ResetBreakGlass(ctx, cleanup.ResetInput{MachineID: req.Msg.GetMachineId(), NodeID: req.Msg.GetNodeId(), Target: req.Msg.GetTarget(), Scope: req.Msg.GetScope()}, actor)
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.ResetBreakGlassResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) StartCleanup(ctx context.Context, req *connect.Request[cleanupv1.StartCleanupRequest]) (*connect.Response[cleanupv1.StartCleanupResponse], error) {
	actor, err := ownerID(ctx)
	if err != nil {
		return nil, err
	}
	op, err := h.deps.Service.Start(ctx, cleanup.StartInput{MachineID: req.Msg.GetMachineId(), NodeID: req.Msg.GetNodeId(), Target: req.Msg.GetTarget(), Scope: req.Msg.GetScope()}, actor)
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.StartCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) GetCleanup(ctx context.Context, req *connect.Request[cleanupv1.GetCleanupRequest]) (*connect.Response[cleanupv1.GetCleanupResponse], error) {
	if _, err := ownerID(ctx); err != nil {
		return nil, err
	}
	op, events, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapCleanupError(err)
	}
	out := make([]*cleanupv1.CleanupEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, eventToProto(ev))
	}
	return connect.NewResponse(&cleanupv1.GetCleanupResponse{Operation: opToProto(op), Events: out}), nil
}

func (h *connectHandler) PlanCleanup(ctx context.Context, req *connect.Request[cleanupv1.PlanCleanupRequest]) (*connect.Response[cleanupv1.PlanCleanupResponse], error) {
	if _, err := ownerID(ctx); err != nil {
		return nil, err
	}
	op, err := h.deps.Service.Plan(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.PlanCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) ConfirmCleanup(ctx context.Context, req *connect.Request[cleanupv1.ConfirmCleanupRequest]) (*connect.Response[cleanupv1.ConfirmCleanupResponse], error) {
	actor, err := ownerID(ctx)
	if err != nil {
		return nil, err
	}
	operator := strings.TrimSpace(req.Msg.GetOperatorId())
	if operator != "" && operator != actor {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("operator_id must match the authenticated owner"))
	}
	operator = actor
	op, err := h.deps.Service.Confirm(ctx, cleanup.ConfirmInput{ID: req.Msg.GetId(), Target: req.Msg.GetTarget(), PlanHash: req.Msg.GetPlanHash(), SealedPassphrase: req.Msg.GetSealedPassphrase(), Capability: req.Msg.GetCapability(), OperatorID: operator})
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.ConfirmCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) ApplyCleanup(ctx context.Context, req *connect.Request[cleanupv1.ApplyCleanupRequest]) (*connect.Response[cleanupv1.ApplyCleanupResponse], error) {
	if _, err := ownerID(ctx); err != nil {
		return nil, err
	}
	op, err := h.deps.Service.Apply(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.ApplyCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) VerifyCleanup(ctx context.Context, req *connect.Request[cleanupv1.VerifyCleanupRequest]) (*connect.Response[cleanupv1.VerifyCleanupResponse], error) {
	if _, err := ownerID(ctx); err != nil {
		return nil, err
	}
	op, err := h.deps.Service.Verify(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.VerifyCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) CancelCleanup(ctx context.Context, req *connect.Request[cleanupv1.CancelCleanupRequest]) (*connect.Response[cleanupv1.CancelCleanupResponse], error) {
	if _, err := ownerID(ctx); err != nil {
		return nil, err
	}
	op, err := h.deps.Service.Cancel(ctx, req.Msg.GetId(), req.Msg.GetReason())
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.CancelCleanupResponse{Operation: opToProto(op)}), nil
}

func (h *connectHandler) ReportCleanupEvent(ctx context.Context, req *connect.Request[cleanupv1.ReportCleanupEventRequest]) (*connect.Response[cleanupv1.ReportCleanupEventResponse], error) {
	ev := req.Msg.GetEvent()
	if ev == nil || strings.TrimSpace(ev.GetOperationId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cleanup event with operation_id is required"))
	}
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(req.Header().Get(nodeauth.HeaderNode), req.Header().Get(nodeauth.HeaderTS), req.Header().Get(nodeauth.HeaderSig))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		op, _, err := h.deps.Service.Get(ctx, ev.GetOperationId())
		if err != nil {
			return nil, mapCleanupError(err)
		}
		if op.NodeID != proof.NodeID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("a node may only report cleanup events for its own operation"))
		}
	}
	accepted, err := h.deps.Service.AppendEvent(ctx, eventFromProto(ev))
	if err != nil {
		return nil, mapCleanupError(err)
	}
	return connect.NewResponse(&cleanupv1.ReportCleanupEventResponse{Accepted: accepted}), nil
}
