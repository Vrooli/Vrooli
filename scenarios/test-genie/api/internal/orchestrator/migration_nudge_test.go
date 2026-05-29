package orchestrator

import (
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"test-genie/internal/orchestrator/phases"
)

func findingOf(sev architecturev1.FindingSeverity) *architecturev1.ArchitectureFinding {
	return &architecturev1.ArchitectureFinding{Severity: sev}
}

func repeatFindings(sev architecturev1.FindingSeverity, n int) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, findingOf(sev))
	}
	return out
}

// archResults wraps findings in an architecture-phase-present result set so
// the gate is satisfied.
func archResults(findings ...[]*architecturev1.ArchitectureFinding) []phases.ExecutionResult {
	all := []*architecturev1.ArchitectureFinding{}
	for _, f := range findings {
		all = append(all, f...)
	}
	return []phases.ExecutionResult{
		{Name: phases.Architecture.String(), Findings: all},
	}
}

func TestMigrationNudge_BelowThresholdIsNil(t *testing.T) {
	// 4 errors (< 5 severe) and 4 total (< 15) → no nudge.
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 4))
	if n := computeMigrationNudge("demo", res); n != nil {
		t.Fatalf("expected nil below threshold, got %+v", n)
	}
}

func TestMigrationNudge_SevereThresholdFires(t *testing.T) {
	// exactly 5 blocker/error → fires on the severe rule.
	res := archResults(
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, 2),
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 3),
	)
	n := computeMigrationNudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge at severe==5")
	}
	if n.Severe != 5 || n.Total != 5 {
		t.Errorf("counts wrong: %+v", n)
	}
	want := "architecture-cartographer migration create demo --from-audit <audit-report.json>"
	if n.Command != want {
		t.Errorf("command = %q, want %q", n.Command, want)
	}
}

func TestMigrationNudge_TotalThresholdFires(t *testing.T) {
	// 16 info findings, 0 severe → fires on the total rule (>15).
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, 16))
	n := computeMigrationNudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge at total==16")
	}
	if n.Severe != 0 || n.Total != 16 {
		t.Errorf("counts wrong: %+v", n)
	}
	if n.BySeverity["info"] != 16 {
		t.Errorf("bySeverity wrong: %+v", n.BySeverity)
	}
}

func TestMigrationNudge_BoundaryTotalIs15IsNil(t *testing.T) {
	// total==15 is NOT > 15, and 0 severe → no nudge (boundary).
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, 15))
	if n := computeMigrationNudge("demo", res); n != nil {
		t.Fatalf("total==15 should not fire, got %+v", n)
	}
}

func TestMigrationNudge_GatedOnArchitecturePhase(t *testing.T) {
	// Findings exist but the architecture phase did NOT run → no nudge.
	res := []phases.ExecutionResult{
		{Name: phases.Standards.String(), Findings: repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 20)},
	}
	if n := computeMigrationNudge("demo", res); n != nil {
		t.Fatalf("nudge must be gated on the architecture phase running, got %+v", n)
	}
}

func TestMigrationNudge_EnvOverride(t *testing.T) {
	t.Setenv(envMigrationSevereThreshold, "2")
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 2))
	n := computeMigrationNudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge with severe threshold lowered to 2")
	}
	if !strings.Contains(n.Reason, "severe≥2") {
		t.Errorf("reason should reflect overridden threshold: %q", n.Reason)
	}
}
