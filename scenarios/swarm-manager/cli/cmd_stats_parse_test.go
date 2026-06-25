package main

import "testing"

func TestFormatDurationSeconds(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0m"},
		{-5, "0m"},
		{30, "30s"},
		{59, "59s"},
		{90, "2m"},
		{3599, "60m"},
		{7200, "2.0h"},
	}
	for _, c := range cases {
		if got := formatDurationSeconds(c.in); got != c.want {
			t.Errorf("formatDurationSeconds(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestParseScalarValue(t *testing.T) {
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
		if got := parseScalarValue(c.in); got != c.want {
			t.Errorf("parseScalarValue(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseAdoptionRow(t *testing.T) {
	line := `agent_manager_sandbox_adoption_total{run_mode="auto",sandbox_mode="on",manual_review="false"} 7`
	row := parseAdoptionRow(line)
	if row == nil {
		t.Fatal("expected non-nil row")
	}
	if row.RunMode != "auto" || row.SandboxMode != "on" || row.ManualReview != "false" || row.Count != 7 {
		t.Errorf("row = %+v", row)
	}

	// Malformed (no braces) returns nil.
	if got := parseAdoptionRow("agent_manager_sandbox_adoption_total 1"); got != nil {
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
