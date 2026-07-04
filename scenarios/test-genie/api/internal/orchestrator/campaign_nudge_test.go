package orchestrator

import (
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"test-genie/internal/orchestrator/phases"
)

const testArtifactPath = "coverage/runs/run-xyz/findings.json"

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

// nudge is a test shim with the default FAIL verdict + a fixed artifact path.
func nudge(scenario string, res []phases.ExecutionResult) *CampaignNudge {
	return computeCampaignNudge(scenario, SuiteVerdictFail, testArtifactPath, res)
}

func TestCampaignNudge_BelowThresholdIsNil(t *testing.T) {
	// 4 errors (< 5 severe) and 4 actionable (< 15), even on FAIL → no nudge.
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 4))
	if n := nudge("demo", res); n != nil {
		t.Fatalf("expected nil below threshold, got %+v", n)
	}
}

func TestCampaignNudge_SevereThresholdFires(t *testing.T) {
	// exactly 5 blocker/error → fires on the severe rule.
	res := archResults(
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER, 2),
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 3),
	)
	n := nudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge at severe==5")
	}
	if n.Severe != 5 || n.Total != 5 {
		t.Errorf("counts wrong: %+v", n)
	}
	want := "architecture-cartographer campaign create demo --from-audit " + testArtifactPath
	if n.Command != want {
		t.Errorf("command = %q, want %q", n.Command, want)
	}
	if n.ArtifactPath != testArtifactPath {
		t.Errorf("artifact path = %q, want %q", n.ArtifactPath, testArtifactPath)
	}
}

// TestCampaignNudge_SevereFiresOnGreen: the severe trigger ignores verdict —
// 5 errors on an otherwise-PASS suite still nudge (real rot worth tracking).
func TestCampaignNudge_SevereFiresOnGreen(t *testing.T) {
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 5))
	n := computeCampaignNudge("demo", SuiteVerdictPass, testArtifactPath, res)
	if n == nil {
		t.Fatalf("severe trigger must fire regardless of verdict")
	}
}

// TestCampaignNudge_VolumeFiresOnFailWithWarns: 16 warnings on a FAIL suite
// trips the volume rule (blocker+error+warn > 15 AND verdict != PASS).
func TestCampaignNudge_VolumeFiresOnFailWithWarns(t *testing.T) {
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, 16))
	n := nudge("demo", res)
	if n == nil {
		t.Fatalf("expected volume nudge at 16 warns on FAIL")
	}
	if n.Severe != 0 || n.Total != 16 {
		t.Errorf("counts wrong: %+v", n)
	}
	if n.BySeverity["warn"] != 16 {
		t.Errorf("bySeverity wrong: %+v", n.BySeverity)
	}
}

// TestCampaignNudge_VolumeSilentOnGreen: the SAME 16-warning load on a PASS
// suite does NOT nudge — a green suite blocks no one. This is the assessment's
// "near-permanent banner" fix.
func TestCampaignNudge_VolumeSilentOnGreen(t *testing.T) {
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, 16))
	if n := computeCampaignNudge("demo", SuiteVerdictPass, testArtifactPath, res); n != nil {
		t.Fatalf("green suite must not nudge on warning volume, got %+v", n)
	}
	// PARTIAL is non-passing, so the same load DOES nudge there.
	if n := computeCampaignNudge("demo", SuiteVerdictPartial, testArtifactPath, res); n == nil {
		t.Fatalf("partial suite should nudge on warning volume")
	}
}

// TestCampaignNudge_InfoFloodNeverCounts: 100 info findings never count toward
// volume, even on FAIL — info is advisory by definition.
func TestCampaignNudge_InfoFloodNeverCounts(t *testing.T) {
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_INFO, 100))
	if n := nudge("demo", res); n != nil {
		t.Fatalf("info flood must never nudge, got %+v", n)
	}
}

func TestCampaignNudge_BoundaryActionableIs15IsNil(t *testing.T) {
	// 15 warns on FAIL is NOT > 15, and 0 severe → no nudge (boundary).
	res := archResults(repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, 15))
	if n := nudge("demo", res); n != nil {
		t.Fatalf("actionable==15 should not fire, got %+v", n)
	}
}

// TestCampaignNudge_FiresOnNonArchitectureBattery is the decoupled-trigger
// regression: a heavy quality-only audit (no architecture phase) must still
// nudge, because any battery crossing the threshold is worth tracking.
func TestCampaignNudge_FiresOnNonArchitectureBattery(t *testing.T) {
	res := resultsFor(phases.Quality.String(),
		repeatFindings(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, 20))
	n := nudge("demo", res)
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
	n := nudge("demo", res)
	if n == nil {
		t.Fatalf("expected nudge with severe threshold lowered to 2")
	}
	if !strings.Contains(n.Reason, "≥2") {
		t.Errorf("reason should reflect overridden threshold: %q", n.Reason)
	}
}
