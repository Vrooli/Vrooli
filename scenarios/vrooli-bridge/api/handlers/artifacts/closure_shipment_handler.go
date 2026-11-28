package artifacts

import (
	"context"
	"errors"

	internalartifacts "vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/auth"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/coreset"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// StartClosureShipment is the owner-authenticated control-plane entry point.
// It validates the canonical BR1 closure, verifies the source archive exactly
// matches the declared inventory, persists intent, then queues signed Bridge
// channel commands to the bound target.
func (h *connectHandler) StartClosureShipment(ctx context.Context, req *connect.Request[artifactsv1.StartClosureShipmentRequest]) (*connect.Response[artifactsv1.StartClosureShipmentResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.ClosureShipments == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("closure shipment coordinator unavailable"))
	}
	m := req.Msg
	if m.GetClosure() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("canonical BR1 closure is required"))
	}
	closure := internalartifacts.ClosureShipmentRequest{IdempotencyKey: m.GetIdempotencyKey(), TargetID: m.GetTargetId(), SourceRef: m.GetSourceRef(), Destination: m.GetDestination(), InventoryDigest: m.GetInventoryDigest(), PriorRevision: m.GetPriorRevision(), NewRevision: m.GetNewRevision(), SelectionReceiptRef: m.GetSelectionReceiptRef(), SelectionDigest: m.GetSelectionDigest(), DeploymentID: m.GetDeploymentId(), MachineID: m.GetMachineId(), NodeID: m.GetNodeId(), EnrollmentGeneration: m.GetEnrollmentGeneration(), DesiredRevision: m.GetDesiredRevision(), SelectionAuthorization: req.Header().Get("Authorization")}
	closure.Closure.SchemaVersion = int(m.GetClosure().GetSchemaVersion())
	closure.Closure.Digest = m.GetClosure().GetDigest()
	for _, member := range m.GetClosure().GetMembers() {
		if member == nil || member.GetComponent() == nil || member.GetRoot() == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("closure member is incomplete"))
		}
		out := coreset.ClosureMember{Component: coreset.ComponentID{Kind: member.GetComponent().GetKind(), Name: member.GetComponent().GetName()}, Selected: member.GetSelected(), Root: coreset.ComponentID{Kind: member.GetRoot().GetKind(), Name: member.GetRoot().GetName()}}
		for _, reason := range member.GetPath() {
			out.Path = append(out.Path, reasonFromProto(reason))
		}
		for _, reason := range member.GetReasons() {
			out.Reasons = append(out.Reasons, reasonFromProto(reason))
		}
		closure.Closure.Members = append(closure.Closure.Members, out)
	}
	for _, file := range m.GetFiles() {
		if file == nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("closure file entry is nil"))
		}
		closure.Files = append(closure.Files, internalartifacts.ClosureFile{Path: file.GetPath(), SHA256: file.GetSha256(), Mode: file.GetMode(), Size: int64(file.GetSizeBytes())})
	}
	shipment, err := h.deps.ClosureShipments.Start(ctx, closure)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.StartClosureShipmentResponse{ShipmentId: shipment.ID, State: string(shipment.State), ClosureDigest: shipment.ClosureDigest, InventoryDigest: shipment.InventoryDigest}), nil
}

func reasonFromProto(r *channelv1.ClosureReason) coreset.ClosureReason {
	if r == nil {
		return coreset.ClosureReason{}
	}
	id := func(v *channelv1.ClosureComponentID) coreset.ComponentID {
		if v == nil {
			return coreset.ComponentID{}
		}
		return coreset.ComponentID{Kind: v.GetKind(), Name: v.GetName()}
	}
	return coreset.ClosureReason{From: id(r.GetFrom()), To: id(r.GetTo()), Relation: r.GetRelation(), DeclaredBy: r.GetDeclaredBy(), Detail: r.GetDetail(), Required: r.GetRequired(), Policy: r.GetPolicy()}
}

// GetClosureShipment returns the durable shipment receipt by shipment ID or
// idempotency key. Owner authentication is required for control-plane reads.
func (h *connectHandler) GetClosureShipment(ctx context.Context, req *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.GetClosureShipmentResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.ClosureShipments == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("closure shipment coordinator unavailable"))
	}
	shipment, err := h.deps.ClosureShipments.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	r := &channelv1.ClosureShipmentReceipt{ShipmentId: shipment.ID, IdempotencyKey: shipment.IdempotencyKey, TargetId: shipment.TargetID, SourceRef: shipment.SourceRef, Destination: shipment.Destination, ClosureDigest: shipment.ClosureDigest, InventoryDigest: shipment.InventoryDigest, ActualInventoryDigest: shipment.ActualInventoryDigest, PriorRevision: shipment.PriorRevision, NewRevision: shipment.NewRevision, ActiveRevision: shipment.ActiveRevision, StageRef: shipment.StageRef, State: receiptState(shipment.State), Result: shipment.Result, SelectionReceiptRef: shipment.SelectionReceiptRef, SelectionDigest: shipment.SelectionDigest, DeploymentId: shipment.DeploymentID, MachineId: shipment.MachineID, NodeId: shipment.NodeID, EnrollmentGeneration: shipment.EnrollmentGeneration, DesiredRevision: shipment.DesiredRevision}
	for _, f := range shipment.ActualFiles {
		r.ActualFiles = append(r.ActualFiles, &channelv1.ClosureFile{Path: f.Path, Sha256: f.SHA256, Mode: f.Mode, SizeBytes: uint64(f.Size)})
	}
	return connect.NewResponse(&artifactsv1.GetClosureShipmentResponse{Receipt: r}), nil
}

// ReconcileClosureShipment requests the target's durable state after a lost
// response. The authenticated target receipt updates the owner record later;
// callers must re-read GetClosureShipment before taking the next transition.
func (h *connectHandler) ReconcileClosureShipment(ctx context.Context, req *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.ClosureShipmentControlResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.ClosureShipments == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("closure shipment coordinator unavailable"))
	}
	shipment, err := h.deps.ClosureShipments.Reconcile(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.ClosureShipmentControlResponse{ShipmentId: shipment.ID, Action: "reconcile", Queued: true, ObservedState: string(shipment.State)}), nil
}

// RollbackClosureShipment requests target-side compare-and-swap restoration of
// the prior revision. The durable receipt remains the status authority.
func (h *connectHandler) RollbackClosureShipment(ctx context.Context, req *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.ClosureShipmentControlResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if h.deps.ClosureShipments == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("closure shipment coordinator unavailable"))
	}
	shipment, err := h.deps.ClosureShipments.Rollback(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&artifactsv1.ClosureShipmentControlResponse{ShipmentId: shipment.ID, Action: "rollback", Queued: true, ObservedState: string(shipment.State)}), nil
}

func receiptState(state internalartifacts.ClosureShipmentState) channelv1.ClosureShipmentReceiptState {
	switch state {
	case internalartifacts.ShipmentStaged:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED
	case internalartifacts.ShipmentVerified:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED
	case internalartifacts.ShipmentActivated:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED
	case internalartifacts.ShipmentRolledBack:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK
	case internalartifacts.ShipmentRejected:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_REJECTED
	case internalartifacts.ShipmentUncertain:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNCERTAIN
	default:
		return channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNSPECIFIED
	}
}

var _ interface {
	StartClosureShipment(context.Context, *connect.Request[artifactsv1.StartClosureShipmentRequest]) (*connect.Response[artifactsv1.StartClosureShipmentResponse], error)
} = (*connectHandler)(nil)
var _ interface {
	GetClosureShipment(context.Context, *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.GetClosureShipmentResponse], error)
} = (*connectHandler)(nil)
var _ interface {
	ReconcileClosureShipment(context.Context, *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.ClosureShipmentControlResponse], error)
} = (*connectHandler)(nil)
var _ interface {
	RollbackClosureShipment(context.Context, *connect.Request[artifactsv1.GetClosureShipmentRequest]) (*connect.Response[artifactsv1.ClosureShipmentControlResponse], error)
} = (*connectHandler)(nil)
