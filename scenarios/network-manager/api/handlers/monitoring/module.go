package monitoring

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainmonitoring "network-manager/internal/monitoring"
	domainsnapshot "network-manager/internal/snapshot"

	monitoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring"
	monitoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring/monitoring_v1connect"
)

type handler struct {
	service *domainmonitoring.Service
}

func Module(db domainmonitoring.SQLExecutor) module.Module {
	snapshotService := domainsnapshot.NewService(domainsnapshot.Config{
		Repo:   domainsnapshot.NewSQLiteRepository(db),
		Runner: domainsnapshot.RealProbeRunner{},
	})
	service := domainmonitoring.NewService(domainmonitoring.Config{
		Repo:      domainmonitoring.NewSQLiteRepository(db),
		Snapshots: snapshotService,
	})
	path, h := monitoringconnect.NewMonitoringServiceHandler(&handler{service: service})
	return module.Module{Name: "monitoring", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainmonitoring.Schema() }

func (h *handler) ListMonitoringSchedules(ctx context.Context, req *connect.Request[monitoringv1.ListMonitoringSchedulesRequest]) (*connect.Response[monitoringv1.ListMonitoringSchedulesResponse], error) {
	schedules, err := h.service.ListSchedules(ctx, req.Msg.GetIncludeDisabled())
	if err != nil {
		return nil, monitoringError(err)
	}
	return connect.NewResponse(&monitoringv1.ListMonitoringSchedulesResponse{Schedules: toProtoSchedules(schedules)}), nil
}

func (h *handler) UpsertMonitoringSchedule(ctx context.Context, req *connect.Request[monitoringv1.UpsertMonitoringScheduleRequest]) (*connect.Response[monitoringv1.UpsertMonitoringScheduleResponse], error) {
	schedule, err := h.service.UpsertSchedule(ctx, fromProtoSchedule(req.Msg.GetSchedule()))
	if err != nil {
		return nil, monitoringError(err)
	}
	return connect.NewResponse(&monitoringv1.UpsertMonitoringScheduleResponse{Schedule: toProtoSchedule(schedule)}), nil
}

func (h *handler) RunMonitoringCheck(ctx context.Context, req *connect.Request[monitoringv1.RunMonitoringCheckRequest]) (*connect.Response[monitoringv1.RunMonitoringCheckResponse], error) {
	run, err := h.service.RunCheck(ctx, req.Msg.GetScheduleId(), req.Msg.GetDryRun())
	if err != nil {
		return nil, monitoringError(err)
	}
	return connect.NewResponse(&monitoringv1.RunMonitoringCheckResponse{Run: toProtoRun(run)}), nil
}

func (h *handler) ListMonitoringAlerts(ctx context.Context, req *connect.Request[monitoringv1.ListMonitoringAlertsRequest]) (*connect.Response[monitoringv1.ListMonitoringAlertsResponse], error) {
	alerts, err := h.service.ListAlerts(ctx, req.Msg.GetScheduleId(), req.Msg.GetOpenOnly())
	if err != nil {
		return nil, monitoringError(err)
	}
	return connect.NewResponse(&monitoringv1.ListMonitoringAlertsResponse{Alerts: toProtoAlerts(alerts)}), nil
}

func monitoringError(err error) error {
	switch {
	case errors.Is(err, domainmonitoring.ErrNotFound), errors.Is(err, domainsnapshot.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func fromProtoSchedule(schedule *monitoringv1.MonitoringSchedule) domainmonitoring.Schedule {
	if schedule == nil {
		return domainmonitoring.Schedule{}
	}
	return domainmonitoring.Schedule{
		ID:                   schedule.GetId(),
		Name:                 schedule.GetName(),
		Profile:              schedule.GetProfile(),
		BaselineSnapshotID:   schedule.GetBaselineSnapshotId(),
		IntervalMinutes:      int(schedule.GetIntervalMinutes()),
		Enabled:              schedule.GetEnabled(),
		LatencyThresholdMS:   int(schedule.GetLatencyThresholdMs()),
		UnavailableThreshold: int(schedule.GetUnavailableThreshold()),
	}
}

func toProtoSchedules(schedules []domainmonitoring.Schedule) []*monitoringv1.MonitoringSchedule {
	out := make([]*monitoringv1.MonitoringSchedule, 0, len(schedules))
	for _, schedule := range schedules {
		out = append(out, toProtoSchedule(schedule))
	}
	return out
}

func toProtoSchedule(schedule domainmonitoring.Schedule) *monitoringv1.MonitoringSchedule {
	return &monitoringv1.MonitoringSchedule{
		Id:                   schedule.ID,
		Name:                 schedule.Name,
		Profile:              schedule.Profile,
		BaselineSnapshotId:   schedule.BaselineSnapshotID,
		IntervalMinutes:      int32(schedule.IntervalMinutes),
		Enabled:              schedule.Enabled,
		LatencyThresholdMs:   int32(schedule.LatencyThresholdMS),
		UnavailableThreshold: int32(schedule.UnavailableThreshold),
		Effects:              schedule.Effects,
		CreatedAt:            formatTime(schedule.CreatedAt),
		UpdatedAt:            formatTime(schedule.UpdatedAt),
	}
}

func toProtoRun(run domainmonitoring.Run) *monitoringv1.MonitoringRun {
	return &monitoringv1.MonitoringRun{
		Id:                 run.ID,
		ScheduleId:         run.ScheduleID,
		SnapshotId:         run.SnapshotID,
		Status:             run.Status,
		Summary:            run.Summary,
		RegressionDetected: run.RegressionDetected,
		Alerts:             toProtoAlerts(run.Alerts),
		Effects:            run.Effects,
		CreatedAt:          formatTime(run.CreatedAt),
	}
}

func toProtoAlerts(alerts []domainmonitoring.Alert) []*monitoringv1.MonitoringAlert {
	out := make([]*monitoringv1.MonitoringAlert, 0, len(alerts))
	for _, alert := range alerts {
		out = append(out, &monitoringv1.MonitoringAlert{
			Id:         alert.ID,
			ScheduleId: alert.ScheduleID,
			SnapshotId: alert.SnapshotID,
			Severity:   alert.Severity,
			Status:     alert.Status,
			Summary:    alert.Summary,
			Evidence:   alert.Evidence,
			CreatedAt:  formatTime(alert.CreatedAt),
		})
	}
	return out
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(domainmonitoring.TimeFormat)
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("monitoring_schedules_list", monitoringconnect.MonitoringServiceListMonitoringSchedulesProcedure, "List monitoring schedules"),
	connectEndpoint("monitoring_schedules_upsert", monitoringconnect.MonitoringServiceUpsertMonitoringScheduleProcedure, "Create or update a monitoring schedule"),
	connectEndpoint("monitoring_run_check", monitoringconnect.MonitoringServiceRunMonitoringCheckProcedure, "Run monitoring check"),
	connectEndpoint("monitoring_alerts_list", monitoringconnect.MonitoringServiceListMonitoringAlertsProcedure, "List monitoring alerts"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "monitoring", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"monitoring": "proto response"}}}
}
