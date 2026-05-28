package audit

import (
	"testing"

	"architecture-cartographer/internal/conflicts"
)

func mk(t string, sev conflicts.Severity) conflicts.Conflict {
	return conflicts.Conflict{Type: t, Severity: sev}
}

func TestApplyFilters_IncludeAndExclude(t *testing.T) {
	in := []conflicts.Conflict{
		mk("cycle", conflicts.SeverityError),
		mk("mislocated_file", conflicts.SeverityWarn),
		mk("convergence_drift", conflicts.SeverityWarn),
		mk("authority_fallback", conflicts.SeverityInfo),
	}
	out := applyFilters(in, []string{"mislocated_file", "convergence_drift"}, []string{"convergence_drift"})
	if len(out) != 1 || out[0].Type != "mislocated_file" {
		t.Fatalf("include+exclude composition wrong: %+v", out)
	}
}

func TestApplyFilters_EmptyFiltersPassesThrough(t *testing.T) {
	in := []conflicts.Conflict{mk("cycle", conflicts.SeverityError)}
	out := applyFilters(in, nil, nil)
	if len(out) != 1 {
		t.Fatalf("nil filters must pass through; got %+v", out)
	}
}

func TestDecideOutcome_FailsOnAtOrAboveThreshold(t *testing.T) {
	tests := []struct {
		name   string
		in     []conflicts.Conflict
		failOn conflicts.Severity
		want   Outcome
	}{
		{"only info, warn threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityInfo)}, conflicts.SeverityWarn, OutcomeClean},
		{"warn present, warn threshold → findings", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityWarn, OutcomeFindings},
		{"error present, error threshold → findings", []conflicts.Conflict{mk("x", conflicts.SeverityWarn), mk("y", conflicts.SeverityError)}, conflicts.SeverityError, OutcomeFindings},
		{"warn only, error threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityError, OutcomeClean},
		{"empty → clean", nil, conflicts.SeverityWarn, OutcomeClean},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideOutcome(tc.in, tc.failOn); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestCountBySeverityAndType(t *testing.T) {
	in := []conflicts.Conflict{
		mk("cycle", conflicts.SeverityError),
		mk("cycle", conflicts.SeverityError),
		mk("mislocated_file", conflicts.SeverityWarn),
	}
	bs := countBySeverity(in)
	if bs["error"] != 2 || bs["warn"] != 1 {
		t.Fatalf("severity bucketing wrong: %+v", bs)
	}
	bt := countByType(in)
	if bt["cycle"] != 2 || bt["mislocated_file"] != 1 {
		t.Fatalf("type bucketing wrong: %+v", bt)
	}
}
