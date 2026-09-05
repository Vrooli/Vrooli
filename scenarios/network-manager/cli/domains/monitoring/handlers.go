package monitoring

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	monitoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring"
	monitoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring/monitoring_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client monitoringconnect.MonitoringServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return handlers{
		core:   core,
		client: monitoringconnect.NewMonitoringServiceClient(httpClient, baseURL),
	}
}

func (h handlers) schedules(ctx cliapp.RunContext) error {
	resp, err := h.client.ListMonitoringSchedules(context.Background(), connect.NewRequest(&monitoringv1.ListMonitoringSchedulesRequest{
		IncludeDisabled: ctx.BoolFlag("include-disabled"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list monitoring schedules", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetSchedules()))
	for _, schedule := range resp.Msg.GetSchedules() {
		results = append(results, formatSchedule(schedule))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d monitoring schedule(s).", len(results))},
		ResultsHeading: "Schedules",
		Results:        results,
	})
}

func (h handlers) scheduleSet(ctx cliapp.RunContext) error {
	schedule := &monitoringv1.MonitoringSchedule{
		Id:                   ctx.Flag("id"),
		Name:                 ctx.Flag("name"),
		Profile:              ctx.Flag("profile"),
		BaselineSnapshotId:   ctx.Flag("baseline-snapshot-id"),
		Enabled:              !ctx.BoolFlag("disabled"),
		IntervalMinutes:      parseInt32(ctx.Flag("interval-minutes")),
		LatencyThresholdMs:   parseInt32(ctx.Flag("latency-threshold-ms")),
		UnavailableThreshold: parseInt32(ctx.Flag("unavailable-threshold")),
	}
	resp, err := h.client.UpsertMonitoringSchedule(context.Background(), connect.NewRequest(&monitoringv1.UpsertMonitoringScheduleRequest{Schedule: schedule}))
	if err != nil {
		return cliapp.WrapAPIError("upsert monitoring schedule", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{"Saved monitoring schedule."},
		Changes: []string{formatSchedule(resp.Msg.GetSchedule())},
		NextCommand: []string{
			fmt.Sprintf("`network-manager monitoring run %s` - run this check now", resp.Msg.GetSchedule().GetId()),
			"`network-manager monitoring alerts` - list regression alerts",
		},
	})
}

func (h handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.RunMonitoringCheck(context.Background(), connect.NewRequest(&monitoringv1.RunMonitoringCheckRequest{
		ScheduleId: ctx.Positional("schedule_id"),
		DryRun:     ctx.BoolFlag("dry-run"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run monitoring check", err, nil)
	}
	run := resp.Msg.GetRun()
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Monitoring run %s: %s.", run.GetId(), run.GetStatus())},
		Changes: append([]string{
			fmt.Sprintf("snapshot=%s regression=%t", run.GetSnapshotId(), run.GetRegressionDetected()),
			run.GetSummary(),
		}, run.GetEffects()...),
	})
}

func (h handlers) alerts(ctx cliapp.RunContext) error {
	resp, err := h.client.ListMonitoringAlerts(context.Background(), connect.NewRequest(&monitoringv1.ListMonitoringAlertsRequest{
		ScheduleId: ctx.Flag("schedule-id"),
		OpenOnly:   ctx.BoolFlag("open-only"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list monitoring alerts", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.GetAlerts()))
	for _, alert := range resp.Msg.GetAlerts() {
		results = append(results, fmt.Sprintf("%s %s %s: %s", alert.GetSeverity(), alert.GetStatus(), alert.GetSnapshotId(), alert.GetSummary()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d monitoring alert(s).", len(results))},
		ResultsHeading: "Alerts",
		Results:        results,
	})
}

func formatSchedule(schedule *monitoringv1.MonitoringSchedule) string {
	if schedule == nil {
		return "Monitoring schedule unavailable."
	}
	return fmt.Sprintf("%s %s profile=%s baseline=%s interval=%dm enabled=%t", schedule.GetId(), schedule.GetName(), schedule.GetProfile(), schedule.GetBaselineSnapshotId(), schedule.GetIntervalMinutes(), schedule.GetEnabled())
}

func parseInt32(value string) int32 {
	if value == "" {
		return 0
	}
	n, _ := strconv.ParseInt(value, 10, 32)
	return int32(n)
}
