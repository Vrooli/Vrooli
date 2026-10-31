package support

import (
	"strings"
	"testing"
	"time"

	alertspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/alerts"
	settingspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/settings"
	"google.golang.org/protobuf/types/known/timestamppb"

	"system-monitor/cli/internal/testutil"
)

func evaluated(alerts ...*alertspb.Alert) *alertspb.ListAlertsResponse {
	return &alertspb.ListAlertsResponse{EvaluatedAt: timestamppb.New(time.Unix(1000, 0)), MonitoringActive: true, Alerts: alerts}
}

// Regression: the server reports maintenance_state "active" while monitoring
// runs normally. The CLI used to read that as "in maintenance" and showed
// MAINTENANCE for every healthy host.
func TestOverallStatus_ActiveMonitoringIsNotMaintenance(t *testing.T) {
	monitoring := MonitoringActive(&settingspb.SystemSettings{Active: true}, "active")
	testutil.Equal(t, monitoring, true)
	testutil.Equal(t, OverallStatus(monitoring, evaluated()), "HEALTHY")
}

func TestOverallStatus_PausedMonitoringIsInactive(t *testing.T) {
	testutil.Equal(t, MonitoringActive(&settingspb.SystemSettings{Active: false}, "active"), false)
	testutil.Equal(t, MonitoringActive(&settingspb.SystemSettings{Active: true}, "inactive"), false)
	// Even with a critical alert on record, a paused monitor is not making
	// a statement about the host.
	critical := &alertspb.Alert{Severity: "critical"}
	testutil.Equal(t, OverallStatus(false, evaluated(critical)), "INACTIVE")
}

func TestOverallStatus_FollowsMostSevereActiveServerAlert(t *testing.T) {
	warning := &alertspb.Alert{Severity: "warning"}
	critical := &alertspb.Alert{Severity: "critical"}
	resolvedCritical := &alertspb.Alert{Severity: "critical", ResolvedAt: timestamppb.New(time.Unix(2000, 0))}
	testutil.Equal(t, OverallStatus(true, evaluated(warning)), "WARNING")
	testutil.Equal(t, OverallStatus(true, evaluated(warning, critical)), "CRITICAL")
	testutil.Equal(t, OverallStatus(true, evaluated(resolvedCritical)), "HEALTHY")
}

func TestOverallStatus_NoEvaluationYetIsUnknown(t *testing.T) {
	testutil.Equal(t, OverallStatus(true, &alertspb.ListAlertsResponse{MonitoringActive: true}), "UNKNOWN")
	testutil.Equal(t, OverallStatus(true, nil), "UNKNOWN")
}

func TestAlertLinesAndRuleLines(t *testing.T) {
	lines := AlertLines([]*alertspb.Alert{{Severity: "critical", RuleId: "memory_pressure", Message: "Memory pressure is critical", Escalated: true, AnomalyId: "anomaly_1"}})
	if len(lines) != 1 || !strings.Contains(lines[0], "[CRITICAL] memory_pressure: Memory pressure is critical") || !strings.Contains(lines[0], "anomaly_1") {
		t.Fatalf("lines = %v", lines)
	}
	testutil.Equal(t, AlertLines(nil)[0], "No active alerts.")
	rules := RuleLines([]*alertspb.RuleState{{RuleId: "cpu_usage", Status: "supported"}, {RuleId: "orphaned_agents", Status: "unsupported", Reason: "control plane does not classify orphans yet"}})
	if len(rules) != 1 || !strings.Contains(rules[0], "orphaned_agents: unsupported") {
		t.Fatalf("rule lines = %v", rules)
	}
}
