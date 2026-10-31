package overview

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	alertsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts/alertsconnect"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	"google.golang.org/protobuf/types/known/timestamppb"

	"system-monitor/cli/internal/testutil"
)

// fakeAlertsServer is the API boundary: it serves server-owned alert state.
type fakeAlertsServer struct {
	alertsconnect.UnimplementedAlertsServiceHandler
	resp *alertspb.ListAlertsResponse
	got  *alertspb.ListAlertsRequest
}

func (f *fakeAlertsServer) ListAlerts(_ context.Context, req *connect.Request[alertspb.ListAlertsRequest]) (*connect.Response[alertspb.ListAlertsResponse], error) {
	f.got = req.Msg
	return connect.NewResponse(f.resp), nil
}

func newTestHandlers(t *testing.T, server *fakeAlertsServer) *handlers {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := alertsconnect.NewAlertsServiceHandler(server)
	mux.Handle(path, handler)
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	return &handlers{alerts: alertsconnect.NewAlertsServiceClient(httpServer.Client(), httpServer.URL)}
}

// A host at 53% memory while thrashing swap: no client-side rule would fire,
// but the server's memory-pressure alert must drive the status.
func TestStatusReadsServerAlerts(t *testing.T) {
	server := &fakeAlertsServer{resp: &alertspb.ListAlertsResponse{
		EvaluatedAt:      timestamppb.New(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)),
		MonitoringActive: true,
		SustainSeconds:   300,
		Alerts: []*alertspb.Alert{{
			Id: "cond_memory_pressure_host_1", Type: "condition", Severity: "critical", RuleId: "memory_pressure",
			Message: "Memory pressure is critical: swap-in 47.0 MiB/s.", OpenedAt: timestamppb.New(time.Date(2026, 10, 7, 11, 50, 0, 0, time.UTC)),
		}},
		Rules: []*alertspb.RuleState{{RuleId: "orphaned_agents", Status: "unsupported", Reason: "control plane does not classify orphans yet"}},
	}}
	h := newTestHandlers(t, server)

	alerts, err := h.fetchAlerts(&alertspb.ListAlertsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	metrics := &metricspb.MetricsResponse{MemoryUsage: 53, CpuUsage: 12}
	report := buildStatusReport(metrics, &settingspb.SystemSettings{Active: true}, "active", alerts)

	testutil.Equal(t, report.OverallStatus, "CRITICAL")
	testutil.Equal(t, report.MonitoringState, "running")
	testutil.Equal(t, report.ActiveAlerts, 1)
	var alertItems []string
	for _, group := range report.Triage {
		if group.Heading == "Alerts" {
			alertItems = group.Items
		}
	}
	joined := strings.Join(alertItems, "\n")
	if !strings.Contains(joined, "memory_pressure: Memory pressure is critical") || !strings.Contains(joined, "orphaned_agents: unsupported") {
		t.Fatalf("alert section = %q, want the server alert and the unsupported rule", joined)
	}
	if strings.Contains(strings.Join(report.Status, "\n"), "MAINTENANCE") {
		t.Fatal("active monitoring rendered as MAINTENANCE")
	}
}

func TestAlertsCommandRequestsServerFilters(t *testing.T) {
	server := &fakeAlertsServer{resp: &alertspb.ListAlertsResponse{MonitoringActive: true}}
	h := newTestHandlers(t, server)
	if _, err := h.fetchAlerts(&alertspb.ListAlertsRequest{Status: "all", IncludeEvents: true, Limit: 5}); err != nil {
		t.Fatal(err)
	}
	testutil.Equal(t, server.got.GetStatus(), "all")
	testutil.Equal(t, server.got.GetIncludeEvents(), true)
	testutil.Equal(t, server.got.GetLimit(), int32(5))

	report := buildAlertsReport(server.resp)
	if !strings.Contains(strings.Join(report.Summary, "\n"), "never (evaluator has not run)") {
		t.Fatalf("summary = %v, want an explicit not-yet-evaluated state", report.Summary)
	}
	testutil.Equal(t, report.Results[0], "No active alerts.")
}
