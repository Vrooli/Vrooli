package monitor

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	monitorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor"
	monitorconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor/monitor_v1connect"
)

type handlers struct {
	client monitorconnect.MonitorServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: monitorconnect.NewMonitorServiceClient(httpClient, baseURL)}
}

func (h *handlers) statusCall(_ cliapp.OperationContext) (*monitorv1.GetMonitorStatusResponse, error) {
	resp, err := h.client.GetMonitorStatus(context.Background(), connect.NewRequest(&monitorv1.GetMonitorStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get monitor status", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(_ cliapp.OperationContext, msg *monitorv1.GetMonitorStatusResponse) cliapp.ListReport {
	status := msg.Status
	if status == nil {
		return cliapp.ListReport{Summary: []string{"Monitor status is unavailable."}}
	}
	return cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Monitor is %s; next run %s.", enabledLabel(status.Enabled), formatProtoTimestamp(status.NextRunAt))},
		ResultsHeading: "Monitor status",
		Results: []string{
			fmt.Sprintf("interval=%ds in_flight=%t last_status=%s last_run=%s green_streak=%d", status.IntervalSeconds, status.InFlight, status.LastStatus, emptyDash(status.LastRunId), status.GreenStreak),
		},
	}
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "unscheduled"
	}
	return value.UTC().Format(time.RFC3339)
}

func formatProtoTimestamp(value interface{ AsTime() time.Time }) string {
	if value == nil {
		return "unscheduled"
	}
	return formatTimestamp(value.AsTime())
}
