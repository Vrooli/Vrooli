package channel

import (
	"context"
	"errors"

	internalartifacts "vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/nodeauth"

	"connectrpc.com/connect"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
)

// ReportClosureShipmentReceipt persists one authenticated target state. The
// receipt's target binding must match the credential identity before it reaches
// the durable artifacts owner service.
func (h *heartbeatHandler) ReportClosureShipmentReceipt(ctx context.Context, req *connect.Request[presencev1.ReportClosureShipmentReceiptRequest]) (*connect.Response[presencev1.ReportClosureShipmentReceiptResponse], error) {
	r := req.Msg.GetReceipt()
	if r == nil || r.GetShipmentId() == "" || r.GetIdempotencyKey() == "" || r.GetTargetId() == "" || r.GetSourceRef() == "" || r.GetDestination() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("closure shipment receipt is missing immutable identity fields"))
	}
	nodeID := req.Header().Get(nodeauth.HeaderNode)
	if nodeID == "" || nodeID != r.GetTargetId() {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("closure shipment receipt target does not match authenticated node"))
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
	if h.deps.ClosureShipmentReceipts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("closure shipment receipt recorder unavailable"))
	}
	if h.deps.ClosureShipmentReceiptTargetValidator == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("current closure shipment target validator unavailable"))
	}
	state, err := closureShipmentState(r.GetState())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	files := make([]internalartifacts.StagedFile, 0, len(r.GetActualFiles()))
	for _, f := range r.GetActualFiles() {
		if f == nil {
			continue
		}
		files = append(files, internalartifacts.StagedFile{Path: f.GetPath(), SHA256: f.GetSha256(), Mode: f.GetMode(), Size: int64(f.GetSizeBytes())})
	}
	receipt := internalartifacts.ClosureNodeReceipt{
		ShipmentID: r.GetShipmentId(), IdempotencyKey: r.GetIdempotencyKey(), TargetID: nodeID, SourceRef: r.GetSourceRef(), Destination: r.GetDestination(),
		ClosureDigest: r.GetClosureDigest(), ExpectedInventoryDigest: r.GetInventoryDigest(), ActualInventoryDigest: r.GetActualInventoryDigest(),
		PriorRevision: r.GetPriorRevision(), NewRevision: r.GetNewRevision(), ActiveRevision: r.GetActiveRevision(), StageRef: r.GetStageRef(), State: state, Result: r.GetResult(), Error: r.GetError(), ActualFiles: files,
		SelectionReceiptRef: r.GetSelectionReceiptRef(), SelectionDigest: r.GetSelectionDigest(), DeploymentID: r.GetDeploymentId(), MachineID: r.GetMachineId(), NodeID: r.GetNodeId(), EnrollmentGeneration: r.GetEnrollmentGeneration(), DesiredRevision: r.GetDesiredRevision(),
	}
	if err := h.deps.ClosureShipmentReceiptTargetValidator(ctx, receipt); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	err = h.deps.ClosureShipmentReceipts.RecordNodeReceipt(ctx, receipt)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(&presencev1.ReportClosureShipmentReceiptResponse{Accepted: true}), nil
}

func closureShipmentState(state channelv1.ClosureShipmentReceiptState) (internalartifacts.ClosureShipmentState, error) {
	switch state {
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_STAGED:
		return internalartifacts.ShipmentStaged, nil
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_VERIFIED:
		return internalartifacts.ShipmentVerified, nil
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ACTIVATED:
		return internalartifacts.ShipmentActivated, nil
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_ROLLED_BACK:
		return internalartifacts.ShipmentRolledBack, nil
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_REJECTED:
		return internalartifacts.ShipmentRejected, nil
	case channelv1.ClosureShipmentReceiptState_CLOSURE_SHIPMENT_RECEIPT_STATE_UNCERTAIN:
		return internalartifacts.ShipmentUncertain, nil
	default:
		return "", errors.New("closure shipment receipt state is unspecified")
	}
}

var _ interface {
	ReportClosureShipmentReceipt(context.Context, *connect.Request[presencev1.ReportClosureShipmentReceiptRequest]) (*connect.Response[presencev1.ReportClosureShipmentReceiptResponse], error)
} = (*heartbeatHandler)(nil)
