package control

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"connectrpc.com/connect"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/control"
)

type connectHandler struct {
	service *control.Service
	logger  *log.Logger
}

func NewConnectHandler(service *control.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) GetBackendStatus(ctx context.Context, _ *connect.Request[controlv1.GetBackendStatusRequest]) (*connect.Response[controlv1.GetBackendStatusResponse], error) {
	return connect.NewResponse(&controlv1.GetBackendStatusResponse{Backend: h.service.BackendStatus(ctx)}), nil
}

func (h *connectHandler) ListControllers(ctx context.Context, req *connect.Request[controlv1.ListControllersRequest]) (*connect.Response[controlv1.ListControllersResponse], error) {
	backend, controllers, err := h.service.ListControllers(ctx, req.Msg.IncludeUnavailable)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.ListControllersResponse{Backend: backend, Controllers: controllers}), nil
}

func (h *connectHandler) EnrollController(ctx context.Context, req *connect.Request[controlv1.EnrollControllerRequest]) (*connect.Response[controlv1.EnrollControllerResponse], error) {
	c, err := h.service.Enroll(ctx, req.Msg.ControllerId, req.Msg.Enrolled)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.EnrollControllerResponse{Controller: c}), nil
}

func (h *connectHandler) SetZoneState(ctx context.Context, req *connect.Request[controlv1.SetZoneStateRequest]) (*connect.Response[controlv1.SetZoneStateResponse], error) {
	r, err := h.service.SetZoneState(ctx, req.Msg.Desired)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.SetZoneStateResponse{Result: r}), nil
}

func (h *connectHandler) SetManualOff(ctx context.Context, req *connect.Request[controlv1.SetManualOffRequest]) (*connect.Response[controlv1.SetManualOffResponse], error) {
	o, results, err := h.service.SetManualOff(ctx, req.Msg.AllEnrolled, req.Msg.ControllerIds, req.Msg.ExpiresAt)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.SetManualOffResponse{Override: o, Results: results}), nil
}

func (h *connectHandler) GetDesiredState(ctx context.Context, _ *connect.Request[controlv1.GetDesiredStateRequest]) (*connect.Response[controlv1.GetDesiredStateResponse], error) {
	zones, override, err := h.service.DesiredState(ctx)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.GetDesiredStateResponse{Zones: zones, ManualOverride: override}), nil
}

func (h *connectHandler) ClearManualOverride(ctx context.Context, _ *connect.Request[controlv1.ClearManualOverrideRequest]) (*connect.Response[controlv1.ClearManualOverrideResponse], error) {
	if err := h.service.ClearManualOverride(ctx); err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.ClearManualOverrideResponse{Cleared: true}), nil
}

func (h *connectHandler) Reconcile(ctx context.Context, req *connect.Request[controlv1.ReconcileRequest]) (*connect.Response[controlv1.ReconcileResponse], error) {
	zones, err := h.service.Reconcile(ctx)
	if err != nil {
		return nil, domainError(err)
	}
	return connect.NewResponse(&controlv1.ReconcileResponse{Zones: zones, CompletedAt: timestamppb.Now()}), nil
}

func domainError(err error) error {
	switch {
	case errors.Is(err, control.ErrInvalidState):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, control.ErrControllerNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, control.ErrAmbiguousZone):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, control.ErrNotEnrolled):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, control.ErrUnsupported):
		return connect.NewError(connect.CodeUnimplemented, err)
	case errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, control.ErrControllerNotFound)
	case errors.Is(err, control.ErrBackendUnavailable), errors.Is(err, control.ErrProtocolMismatch), errors.Is(err, control.ErrPermission), errors.Is(err, control.ErrTransport):
		return connect.NewError(connect.CodeUnavailable, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
