package main

import (
	"encoding/json"
	"testing"
)

func TestStatsResponseMatchesCurrentStatsContract(t *testing.T) {
	body := []byte(`{
		"generated_at":"2026-07-26T00:00:00Z",
		"event_count":12,
		"history":{"history_days":30,"has_history":true,"min_sample_meaningful":7},
		"timing":{"avg_lead_time_hours":4.5,"median_lead_time_hours":3.5,"avg_execution_minutes":12,"median_execution_minutes":10,"execution_duration_samples":2},
		"scope":{"goals":[{"name":"reliability","total":3,"completed":1}]},
		"dashboard":{"velocity_trend":[{"week_start":"2026-07-20","completed":1,"completed_items":[{"kind":"feature","name":"restore-stats"}]}],"velocity_weeks_covered":1},
		"session":{"session_created_backlog_items":2,"session_created_goals":1}
	}`)

	var stats StatsResponse
	if err := json.Unmarshal(body, &stats); err != nil {
		t.Fatalf("unmarshal stats response: %v", err)
	}
	if got := stats.Scope.Goals[0].Name; got != "reliability" {
		t.Fatalf("goal name = %q, want reliability", got)
	}
	if got := stats.Session.SessionCreatedGoals; got != 1 {
		t.Fatalf("session-created goals = %d, want 1", got)
	}
	if got := stats.Dashboard.VelocityTrend[0].CompletedItems[0].Name; got != "restore-stats" {
		t.Fatalf("completed item = %q, want restore-stats", got)
	}
}

func TestPrometheusValue(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"agent_manager_runs_with_provenance_total 42", 42},
		{`metric{a="b"} 3.5`, 3.5},
		{"  17  ", 17},
		{"", 0},
		{"notanumber", 0},
	}
	for _, c := range cases {
		if got := prometheusValue(c.in); got != c.want {
			t.Errorf("prometheusValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseAdoptionRow(t *testing.T) {
	line := `agent_manager_sandbox_adoption_total{run_mode="auto",sandbox_mode="on",manual_review="false"} 7`
	row := parseSandboxAdoptionRow(line)
	if row == nil {
		t.Fatal("expected non-nil row")
	}
	if row.RunMode != "auto" || row.SandboxMode != "on" || row.ManualReview != "false" || row.Count != 7 {
		t.Errorf("row = %+v", row)
	}

	// Malformed (no braces) returns nil.
	if got := parseSandboxAdoptionRow("agent_manager_sandbox_adoption_total 1"); got != nil {
		t.Errorf("malformed line should return nil, got %+v", got)
	}
}

func TestParseSandboxAdoptionMetrics(t *testing.T) {
	body := []byte(`# HELP something
# TYPE counter
agent_manager_sandbox_adoption_total{run_mode="manual",sandbox_mode="off",manual_review="true"} 2
agent_manager_sandbox_adoption_total{run_mode="auto",sandbox_mode="on",manual_review="false"} 5
agent_manager_runs_with_provenance_total 8
agent_manager_runs_without_provenance_total 2
unrelated_metric 99
`)
	got := parseSandboxAdoptionMetrics(body)

	if got.RunsWithProvenance != 8 || got.RunsWithoutProvenance != 2 {
		t.Errorf("provenance = %v/%v, want 8/2", got.RunsWithProvenance, got.RunsWithoutProvenance)
	}
	if got.AttributionRate != 0.8 {
		t.Errorf("attribution rate = %v, want 0.8", got.AttributionRate)
	}
	if len(got.Breakdown) != 2 {
		t.Fatalf("breakdown len = %d, want 2", len(got.Breakdown))
	}
	// Sorted by run_mode: auto before manual.
	if got.Breakdown[0].RunMode != "auto" || got.Breakdown[1].RunMode != "manual" {
		t.Errorf("breakdown order = %q, %q", got.Breakdown[0].RunMode, got.Breakdown[1].RunMode)
	}
}

func TestParseSandboxAdoptionMetrics_NoProvenance(t *testing.T) {
	got := parseSandboxAdoptionMetrics([]byte(""))
	if got.AttributionRate != 0 || len(got.Breakdown) != 0 {
		t.Errorf("empty metrics = %+v", got)
	}
}
