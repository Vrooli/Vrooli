package monitor

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	monitoring "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/monitor"

	"connectrpc.com/connect"
	monitorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deps struct {
	Service *monitoring.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetMonitorStatus(ctx context.Context, _ *connect.Request[monitorv1.GetMonitorStatusRequest]) (*connect.Response[monitorv1.GetMonitorStatusResponse], error) {
	if h.deps.Service == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("monitor service is not configured"))
	}
	status, err := h.deps.Service.Status(ctx)
	if err != nil {
		h.deps.Logger.Printf("monitor.GetMonitorStatus: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("get monitor status"))
	}
	return connect.NewResponse(&monitorv1.GetMonitorStatusResponse{Status: statusToProto(status)}), nil
}

func statusToProto(status catalog.MonitorStatus) *monitorv1.MonitorStatus {
	return &monitorv1.MonitorStatus{
		Enabled:         status.Enabled,
		IntervalSeconds: status.IntervalSeconds,
		InFlight:        status.InFlight,
		LastRunId:       status.LastRunID,
		LastStatus:      status.LastStatus,
		LastStartedAt:   optionalTimestamp(status.LastStartedAt),
		LastFinishedAt:  optionalTimestamp(status.LastFinishedAt),
		NextRunAt:       optionalTimestamp(status.NextRunAt),
		GreenStreak:     status.GreenStreak,
		UpdatedAt:       optionalTimestamp(status.UpdatedAt),
	}
}

func optionalTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}
