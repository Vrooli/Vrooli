package supplement

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/supplement"
	"nutrition-planner/internal/decimalx"
	internal "nutrition-planner/internal/supplement"
	workspace "nutrition-planner/internal/workspace"
)

type connectHandler struct {
	repo   internal.Repository
	ws     workspace.Service
	logger *log.Logger
}

func NewConnectHandler(repo internal.Repository, ws workspace.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{repo: repo, ws: ws, logger: logger}
}

func (h *connectHandler) scope(ctx context.Context, id string) error {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified actor required"))
	}
	if id == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("workspace_id is required"))
	}
	if _, err := h.ws.Get(ctx, id, p.Subject); err != nil {
		var nf workspace.ErrNotFound
		if errors.As(err, &nf) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var forbidden workspace.ErrForbidden
		if errors.As(err, &forbidden) {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func (h *connectHandler) ListSchedules(ctx context.Context, req *connect.Request[v1.ListSchedulesRequest]) (*connect.Response[v1.ListSchedulesResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	items, err := h.repo.List(ctx, req.Msg.WorkspaceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListSchedulesResponse{Schedules: make([]*v1.Schedule, 0, len(items))}
	for _, item := range items {
		out.Schedules = append(out.Schedules, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateSchedule(ctx context.Context, req *connect.Request[v1.CreateScheduleRequest]) (*connect.Response[v1.CreateScheduleResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	dose, err := decimalx.Parse(req.Msg.Dose)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dose: invalid decimal"))
	}
	created, err := h.repo.Create(ctx, internal.Schedule{WorkspaceID: req.Msg.WorkspaceId, ProductRevisionID: req.Msg.ProductRevisionId, Dose: dose, DoseUnit: req.Msg.DoseUnit, Weekdays: ints(req.Msg.Weekdays), StartDate: req.Msg.StartDate, EndDate: req.Msg.EndDate, Confirmed: req.Msg.Confirmed})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.CreateScheduleResponse{Schedule: toProto(created)}), nil
}

func (h *connectHandler) UpdateSchedule(ctx context.Context, req *connect.Request[v1.UpdateScheduleRequest]) (*connect.Response[v1.UpdateScheduleResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	dose, err := decimalx.Parse(req.Msg.Dose)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("dose: invalid decimal"))
	}
	updated, err := h.repo.Update(ctx, internal.UpdateInput{WorkspaceID: req.Msg.WorkspaceId, ID: req.Msg.Id, ExpectedRevision: req.Msg.ExpectedRevision, Dose: dose, DoseUnit: req.Msg.DoseUnit, Weekdays: ints(req.Msg.Weekdays), StartDate: req.Msg.StartDate, EndDate: req.Msg.EndDate, Paused: req.Msg.Paused, Confirmed: req.Msg.Confirmed})
	if err != nil {
		var conflict internal.ErrConflict
		if errors.As(err, &conflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.UpdateScheduleResponse{Schedule: toProto(updated)}), nil
}

func (h *connectHandler) GetSchedule(ctx context.Context, req *connect.Request[v1.GetScheduleRequest]) (*connect.Response[v1.GetScheduleResponse], error) {
	if err := h.scope(ctx, req.Msg.WorkspaceId); err != nil {
		return nil, err
	}
	item, err := h.repo.Get(ctx, req.Msg.Id, req.Msg.WorkspaceId, req.Msg.Revision)
	if err != nil {
		var nf internal.ErrNotFound
		if errors.As(err, &nf) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetScheduleResponse{Schedule: toProto(item)}), nil
}

func ints(v []int32) []int {
	out := make([]int, 0, len(v))
	for _, item := range v {
		out = append(out, int(item))
	}
	return out
}

func toProto(s internal.Schedule) *v1.Schedule {
	return &v1.Schedule{Id: s.ID, Revision: s.Revision, ProductRevisionId: s.ProductRevisionID, Dose: s.Dose.String(), DoseUnit: s.DoseUnit, Weekdays: ints32(s.Weekdays), StartDate: s.StartDate, EndDate: s.EndDate, Paused: s.Paused, Confirmed: s.Confirmed, CreatedAt: s.CreatedAt.Format(time.RFC3339Nano)}
}

func ints32(v []int) []int32 {
	out := make([]int32, 0, len(v))
	for _, item := range v {
		out = append(out, int32(item))
	}
	return out
}
