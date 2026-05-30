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

// resultsFor wraps findings in a single named phase result.
func resultsFor(phase string, findings ...[]*architecturev1.ArchitectureFinding) []phases.ExecutionResult {
	all := []*architecturev1.ArchitectureFinding{}
	for _, f := range findings {
		all = append(all, f...)
	}
	return []phases.ExecutionResult{
		{Name: phase, Findings: all},
	}
}

// archResults is a convenience for the architecture phase specifically.
func archResults(findings ...[]*architecturev1.ArchitectureFinding) []phases.ExecutionResult {
	return resultsFor(phases.Architecture.String(), findings...)
}

func TestCampaignNudge_BelowThresholdIsNil(t *testing.T) {
	// 4 errors (< 5 severe) and 4 total (< 15) → no nudge.
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 4))
	if n := computeCampaignNudge("demo", res); n != nil {
		t.Fatalf("expected nil below threshold, got %+v", n)
	}
}

func TestCampaignNudge_SevereThresholdFires(t *testing.T) {
	// exactly 5 blocker/error → fires on the severe rule.
	res := archResults(
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, 2),
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 3),
	)
	n := computeCampaignNudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge at severe==5")
	}
	if n.Severe != 5 || n.Total != 5 {
		t.Errorf("counts wrong: %+v", n)
	}
	want := "architecture-cartographer campaign create demo --from-audit <audit-report.json>"
	if n.Command != want {
		t.Errorf("command = %q, want %q", n.Command, want)
	}
}

func TestCampaignNudge_TotalThresholdFires(t *testing.T) {
	// 16 info findings, 0 severe → fires on the total rule (>15).
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, 16))
	n := computeCampaignNudge("demo", res)
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

func TestCampaignNudge_BoundaryTotalIs15IsNil(t *testing.T) {
	// total==15 is NOT > 15, and 0 severe → no nudge (boundary).
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, 15))
	if n := computeCampaignNudge("demo", res); n != nil {
		t.Fatalf("total==15 should not fire, got %+v", n)
	}
}

// TestCampaignNudge_FiresOnNonArchitectureBattery is the decoupled-trigger
// regression: a heavy STANDARDS-only audit (no architecture phase) must
// still nudge, because any battery crossing the threshold is worth tracking.
func TestCampaignNudge_FiresOnNonArchitectureBattery(t *testing.T) {
	res := resultsFor(phases.Standards.String(),
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 20))
	n := computeCampaignNudge("demo", res)
	if n == nil {
		t.Fatalf("nudge must fire on a non-architecture battery crossing threshold")
	}
	if n.Severe != 20 || n.Total != 20 {
		t.Errorf("counts wrong: %+v", n)
	}
	if !strings.Contains(n.Command, "campaign create demo") {
		t.Errorf("command should point at campaign create: %q", n.Command)
	}
}

func TestCampaignNudge_EnvOverride(t *testing.T) {
	t.Setenv(envCampaignSevereThreshold, "2")
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 2))
	n := computeCampaignNudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge with severe threshold lowered to 2")
	}
	if !strings.Contains(n.Reason, "severe≥2") {
		t.Errorf("reason should reflect overridden threshold: %q", n.Reason)
	}
}
