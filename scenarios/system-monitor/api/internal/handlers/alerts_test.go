package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts/alertsconnect"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

type fakeAlertSource struct {
	alerts    []*models.Alert
	lastQuery services.AlertQuery
	err       error
}

func (f *fakeAlertSource) ListAlerts(_ context.Context, query services.AlertQuery) ([]*models.Alert, error) {
	f.lastQuery = query
	return f.alerts, f.err
}

func (f *fakeAlertSource) RuleStatuses() []services.RuleStatus {
	return []services.RuleStatus{{RuleID: "orphaned_agents", Status: "unsupported", Reason: "control plane does not classify orphans yet"}}
}

func (f *fakeAlertSource) EvaluatedAt() time.Time {
	return time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
}
func (f *fakeAlertSource) SustainDuration() time.Duration { return 5 * time.Minute }
func (f *fakeAlertSource) MonitoringActive() bool         { return true }

type fakeAcks struct{ acked map[string]string }

func (f *fakeAcks) AcknowledgeAlert(_ context.Context, id, by string) error {
	if id != "cond_1" {
		return errors.New("not found")
	}
	f.acked[id] = by
	return nil
}

func (f *fakeAcks) GetAlert(_ context.Context, id string) (*models.Alert, error) {
	return &models.Alert{ID: id, AckedBy: f.acked[id]}, nil
}

func newAlertsClient(t *testing.T, source AlertSource, acks AlertAcknowledger) alertsconnect.AlertsServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := alertsconnect.NewAlertsServiceHandler(NewAlertsHandler(source, acks, nil))
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return alertsconnect.NewAlertsServiceClient(server.Client(), server.URL)
}

func TestAlertsHandler_ListAlertsProjectsConditionAlerts(t *testing.T) {
	opened := time.Date(2026, 10, 7, 11, 0, 0, 0, time.UTC)
	source := &fakeAlertSource{alerts: []*models.Alert{{
		ID: "cond_1", Type: "condition", Severity: "critical", Message: "Memory pressure is critical", MetricName: "memory_pressure_level", MetricValue: 3, Timestamp: opened,
		Details: map[string]interface{}{"rule_id": "memory_pressure", "key": "host", "threshold": 0.0, "escalated": true, "anomaly_id": "anomaly_x", "contributors": []interface{}{map[string]interface{}{"pid": 42, "owner": "agent-manager"}}},
	}}}
	client := newAlertsClient(t, source, &fakeAcks{acked: map[string]string{}})

	resp, err := client.ListAlerts(context.Background(), connect.NewRequest(&alertspb.ListAlertsRequest{Status: "all", IncludeEvents: true, Limit: 7}))
	if err != nil {
		t.Fatal(err)
	}
	if source.lastQuery != (services.AlertQuery{Status: "all", IncludeEvents: true, Limit: 7}) {
		t.Fatalf("query = %+v", source.lastQuery)
	}
	if len(resp.Msg.GetAlerts()) != 1 {
		t.Fatalf("alerts = %d", len(resp.Msg.GetAlerts()))
	}
	alert := resp.Msg.GetAlerts()[0]
	if alert.GetRuleId() != "memory_pressure" || alert.GetKey() != "host" || !alert.GetEscalated() || alert.GetAnomalyId() != "anomaly_x" {
		t.Fatalf("rule metadata not lifted: %+v", alert)
	}
	if !alert.GetOpenedAt().AsTime().Equal(opened) || alert.GetResolvedAt() != nil {
		t.Fatalf("timestamps = %v / %v", alert.GetOpenedAt(), alert.GetResolvedAt())
	}
	pid := alert.GetDetails().GetFields()["contributors"].GetListValue().GetValues()[0].GetStructValue().GetFields()["pid"].GetNumberValue()
	if pid != 42 {
		t.Fatalf("contributor pid in details = %v, want 42", pid)
	}
	if resp.Msg.GetSustainSeconds() != 300 || !resp.Msg.GetMonitoringActive() {
		t.Fatalf("sustain/monitoring = %d/%v", resp.Msg.GetSustainSeconds(), resp.Msg.GetMonitoringActive())
	}
	if rules := resp.Msg.GetRules(); len(rules) != 1 || rules[0].GetStatus() != "unsupported" || rules[0].GetReason() == "" {
		t.Fatalf("rule states = %+v", rules)
	}
}

func TestAlertsHandler_RejectsUnknownStatus(t *testing.T) {
	client := newAlertsClient(t, &fakeAlertSource{}, &fakeAcks{acked: map[string]string{}})
	_, err := client.ListAlerts(context.Background(), connect.NewRequest(&alertspb.ListAlertsRequest{Status: "open"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
	}
}

func TestAlertsHandler_AcknowledgeAlert(t *testing.T) {
	acks := &fakeAcks{acked: map[string]string{}}
	client := newAlertsClient(t, &fakeAlertSource{}, acks)
	resp, err := client.AcknowledgeAlert(context.Background(), connect.NewRequest(&alertspb.AcknowledgeAlertRequest{Id: "cond_1", AckedBy: "matt"}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetAlert().GetAckedBy() != "matt" {
		t.Fatalf("acked_by = %q", resp.Msg.GetAlert().GetAckedBy())
	}
	_, err = client.AcknowledgeAlert(context.Background(), connect.NewRequest(&alertspb.AcknowledgeAlertRequest{Id: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing alert code = %v, want not_found", connect.CodeOf(err))
	}
}
