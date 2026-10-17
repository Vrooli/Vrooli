package hygiene

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func withFreshnessOutput(t *testing.T, out []byte, err error) {
	t.Helper()
	prev := testFreshnessOutput
	testFreshnessOutput = func(context.Context, string) ([]byte, error) { return out, err }
	t.Cleanup(func() { testFreshnessOutput = prev })
}

func freshnessCheck(t *testing.T, report Report) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == "test_freshness" {
			return check
		}
	}
	t.Fatalf("no test_freshness check in %+v", report.Checks)
	return Check{}
}

func TestCheckTestFreshnessWarnsOnStaleScenarios(t *testing.T) {
	withFreshnessOutput(t, []byte(`{
		"checked": true,
		"scenarios": ["demo"],
		"warnings": [{"scenario": "demo", "stale_phases": ["unit", "business"], "command": "test-genie execute demo --preset quick"}]
	}`), nil)

	var report Report
	Service{Root: "/repo"}.checkTestFreshness(&report)

	check := freshnessCheck(t, report)
	if check.Passed || check.Severity != SeverityWarning {
		t.Fatalf("stale scenarios must fail the check at warning severity: %+v", check)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected one finding per stale scenario: %+v", report.Findings)
	}
	finding := report.Findings[0]
	if finding.Severity != SeverityWarning {
		t.Errorf("finding severity = %s, want warning", finding.Severity)
	}
	if !strings.Contains(finding.Message, "[unit, business]") || !strings.Contains(finding.Message, "demo") {
		t.Errorf("unexpected finding message: %q", finding.Message)
	}
	if len(finding.NextActions) != 1 || finding.NextActions[0].Command != "test-genie execute demo --preset quick" {
		t.Errorf("finding must carry the suggested command: %+v", finding.NextActions)
	}

	// Warning findings must not block under the default fail-on policy.
	report.finish(SeverityError)
	if !report.Success || report.Warnings != 1 {
		t.Fatalf("freshness warnings must stay non-blocking: success=%v warnings=%d", report.Success, report.Warnings)
	}
}

func TestCheckTestFreshnessPassesWhenFresh(t *testing.T) {
	withFreshnessOutput(t, []byte(`{"checked": true, "scenarios": ["demo"]}`), nil)
	var report Report
	Service{Root: "/repo"}.checkTestFreshness(&report)
	check := freshnessCheck(t, report)
	if !check.Passed || len(report.Findings) != 0 {
		t.Fatalf("fresh change-set must pass cleanly: %+v %+v", check, report.Findings)
	}
}

func TestCheckTestFreshnessSkipsOnDegradation(t *testing.T) {
	cases := []struct {
		name string
		out  []byte
		err  error
	}{
		{"cli missing", nil, errNoTestGenieCLI},
		{"timeout", nil, errors.New("context deadline exceeded")},
		{"junk output", []byte("not json"), nil},
		{"api unreachable", []byte(`{"checked": false}`), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withFreshnessOutput(t, tc.out, tc.err)
			var report Report
			Service{Root: "/repo"}.checkTestFreshness(&report)
			check := freshnessCheck(t, report)
			if !check.Passed || check.Severity != SeverityInfo || !strings.Contains(check.Message, "skipped") {
				t.Fatalf("degradation must be a passing skipped info check: %+v", check)
			}
			if len(report.Findings) != 0 {
				t.Fatalf("degradation must not produce findings: %+v", report.Findings)
			}
		})
	}
}

func TestCheckTestFreshnessTruncationNote(t *testing.T) {
	withFreshnessOutput(t, []byte(`{
		"checked": true,
		"scenarios": ["a", "b", "c", "d", "e"],
		"truncated": true,
		"warnings": [{"scenario": "a", "stale_phases": ["unit"], "command": "test-genie execute a --preset quick"}]
	}`), nil)
	var report Report
	Service{Root: "/repo"}.checkTestFreshness(&report)
	check := freshnessCheck(t, report)
	if !strings.Contains(check.Message, "only the first few were checked") {
		t.Fatalf("truncation must be surfaced in the check message: %q", check.Message)
	}
}
