package system

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func writeReport(t *testing.T, dir string, report map[string]any) string {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "last-report.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestEmergencyWatchdogReportIsUndeterminedWithoutAFile(t *testing.T) {
	check := NewEmergencyWatchdogReportCheck(WithReportPath(filepath.Join(t.TempDir(), "missing.json")))
	result := check.Run(context.Background())
	if result.Status != checks.StatusWarning || result.Details["reportState"] != "undetermined" {
		t.Fatalf("missing report = %+v", result)
	}
}

func TestEmergencyWatchdogReportIsUndeterminedWhenStale(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	path := writeReport(t, t.TempDir(), map[string]any{"captured_at": now.Add(-time.Hour), "findings": []string{}})
	check := NewEmergencyWatchdogReportCheck(WithReportPath(path), WithReportClock(func() time.Time { return now }))
	result := check.Run(context.Background())
	if result.Status != checks.StatusWarning || result.Details["reportState"] != "undetermined" {
		t.Fatalf("stale report = %+v", result)
	}
}

func TestEmergencyWatchdogReportCopiesFindingsAndAttribution(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	path := writeReport(t, t.TempDir(), map[string]any{
		"captured_at": now.Add(-time.Minute),
		"findings":    []string{"fork-rate: 2481.0 forks/s exceeds SB16 bar", "cpu-pressure: CPU pressure 96.0% meets or exceeds SB14 bar"},
		"attribution": map[string]any{
			"state":       "read",
			"by_children": []map[string]any{{"pid": 4242, "name": "claude", "children": 300, "delta": 280, "scope": "/user.slice/vrooli-agents.slice/vrooli-agent-abc.scope"}},
		},
	})
	check := NewEmergencyWatchdogReportCheck(WithReportPath(path), WithReportClock(func() time.Time { return now }))
	result := check.Run(context.Background())
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %s: %s", result.Status, result.Message)
	}
	findings, ok := result.Details["findings"].([]map[string]any)
	if !ok || len(findings) != 2 {
		t.Fatalf("findings = %#v", result.Details["findings"])
	}
	fork := findings[0]
	if fork["name"] != "fork-rate" {
		t.Fatalf("first finding = %#v", fork)
	}
	attribution, ok := fork["attribution"].(map[string]any)
	if !ok {
		t.Fatalf("fork-rate finding carries no attribution: %#v", fork)
	}
	top, ok := attribution["top_parent"].(map[string]any)
	if !ok || top["name"] != "claude" || top["pid"] != int64(4242) {
		t.Fatalf("top parent = %#v", attribution["top_parent"])
	}
	if _, has := findings[1]["attribution"]; !has {
		t.Fatalf("cpu-pressure finding must carry the attribution too: %#v", findings[1])
	}
	stranded := writeReport(t, t.TempDir(), map[string]any{"captured_at": now, "findings": []string{"stranded-memory: 20000 MB stranded"}, "attribution": map[string]any{"state": "read", "by_children": []map[string]any{{"pid": 1, "name": "init", "children": 9}}}})
	other := NewEmergencyWatchdogReportCheck(WithReportPath(stranded), WithReportClock(func() time.Time { return now })).Run(context.Background())
	if _, has := other.Details["findings"].([]map[string]any)[0]["attribution"]; has {
		t.Fatalf("a stranded-memory finding is not attributed by parent: %#v", other.Details["findings"])
	}
	clean := writeReport(t, t.TempDir(), map[string]any{"captured_at": now, "findings": []string{}})
	if result := NewEmergencyWatchdogReportCheck(WithReportPath(clean), WithReportClock(func() time.Time { return now })).Run(context.Background()); result.Status != checks.StatusOK {
		t.Fatalf("empty findings = %+v", result)
	}
}

// One incident per finding family: 190 unmanaged-workload lines from one
// watchdog run become one finding with a count, while the storm finding
// keeps its own attribution.
func TestReportAggregatesPerWorkloadFindingsIntoOneFamily(t *testing.T) {
	report := watchdogReport{Findings: []string{"fork-rate: 900 forks/s exceeds SB16 bar", "unmanaged-workload:101: sleep", "unmanaged-workload:102: sleep", "unmanaged-workload:103: sleep"}, Evidence: map[string][]string{"fork-rate": {"/proc/stat"}}}
	findings := aggregateFindings(report)
	if len(findings) != 2 {
		t.Fatalf("findings = %d, want 2 families: %v", len(findings), findings)
	}
	if findings[0]["name"] != "fork-rate" || findings[1]["name"] != "unmanaged-workload" || findings[1]["count"] != 3 {
		t.Fatalf("findings = %v", findings)
	}
}
