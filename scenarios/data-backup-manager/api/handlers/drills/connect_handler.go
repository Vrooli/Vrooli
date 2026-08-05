package drills

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	d "data-backup-manager/internal/drills"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Service d.Service
	Logger  *log.Logger
}
type connectHandler struct{ deps Deps }

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) PreviewDrill(ctx context.Context, req *connect.Request[drillsv1.PreviewDrillRequest]) (*connect.Response[drillsv1.PreviewDrillResponse], error) {
	p, err := h.deps.Service.Preview(ctx, req.Msg.PlanId, req.Msg.TargetId, req.Msg.DestinationId)
	if err != nil {
		return nil, h.translate("PreviewDrill", err)
	}
	return connect.NewResponse(&drillsv1.PreviewDrillResponse{Eligible: p.Eligible, PlanId: p.PlanID, TargetId: p.TargetID, DestinationId: p.DestinationID, SnapshotId: p.SnapshotID, Warnings: p.Warnings, Reason: p.Reason}), nil
}

func (h *connectHandler) RunDrill(ctx context.Context, req *connect.Request[drillsv1.RunDrillRequest]) (*connect.Response[drillsv1.RunDrillResponse], error) {
	drill, err := h.deps.Service.Run(ctx, req.Msg.PlanId, req.Msg.TargetId, req.Msg.DestinationId, req.Msg.IdempotencyKey, req.Msg.Scheduled)
	if err != nil {
		return nil, h.translate("RunDrill", err)
	}
	return connect.NewResponse(&drillsv1.RunDrillResponse{Drill: toProto(drill)}), nil
}

func (h *connectHandler) GetDrill(ctx context.Context, req *connect.Request[drillsv1.GetDrillRequest]) (*connect.Response[drillsv1.GetDrillResponse], error) {
	drill, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetDrill", err)
	}
	return connect.NewResponse(&drillsv1.GetDrillResponse{Drill: toProto(drill)}), nil
}

func (h *connectHandler) ListDrills(ctx context.Context, req *connect.Request[drillsv1.ListDrillsRequest]) (*connect.Response[drillsv1.ListDrillsResponse], error) {
	list, err := h.deps.Service.List(ctx, req.Msg.PlanId, req.Msg.TargetId, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListDrills", err)
	}
	out := make([]*drillsv1.RecoveryDrill, 0, len(list))
	for _, drill := range list {
		out = append(out, toProto(drill))
	}
	return connect.NewResponse(&drillsv1.ListDrillsResponse{Drills: out}), nil
}

func (h *connectHandler) translate(op string, err error) error {
	var invalid d.ErrInvalid
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var notFound d.ErrNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	var active d.ErrAlreadyActive
	if errors.As(err, &active) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	h.deps.Logger.Printf("drills.%s: %v", op, err)
	return connect.NewError(connect.CodeInternal, err)
}

func toProto(drill d.Drill) *drillsv1.RecoveryDrill {
	p := &drillsv1.RecoveryDrill{Id: drill.ID, PlanId: drill.PlanID, TargetId: drill.TargetID, DestinationId: drill.DestinationID, SnapshotId: drill.SnapshotID, RestoreId: drill.RestoreID, Status: statusToProto(drill.Status), Scheduled: drill.Scheduled, Error: drill.Error, NextAction: drill.NextAction}
	if !drill.RequestedAt.IsZero() {
		p.RequestedAt = timestamppb.New(drill.RequestedAt)
	}
	if !drill.StartedAt.IsZero() {
		p.StartedAt = timestamppb.New(drill.StartedAt)
	}
	if !drill.FinishedAt.IsZero() {
		p.FinishedAt = timestamppb.New(drill.FinishedAt)
	}
	return p
}

func statusToProto(status d.Status) drillsv1.DrillStatus {
	switch status {
	case d.StatusRequested:
		return drillsv1.DrillStatus_DRILL_STATUS_REQUESTED
	case d.StatusRunning:
		return drillsv1.DrillStatus_DRILL_STATUS_RUNNING
	case d.StatusVerified:
		return drillsv1.DrillStatus_DRILL_STATUS_VERIFIED
	case d.StatusFailed:
		return drillsv1.DrillStatus_DRILL_STATUS_FAILED
	default:
		return drillsv1.DrillStatus_DRILL_STATUS_UNSPECIFIED
	}
}
