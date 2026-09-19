package validation_run_test

import (
	"testing"

	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
	vrun "development-toolchain-validator/internal/validation_run"
)

func TestEvaluate_ToolRanAndExpectationMetIsPass(t *testing.T) {
	got := vrun.Evaluate(vrun.EvaluatorInput{ToolResult: &vrun.ToolResult{Ran: true, ExpectationMet: true}})
	if got.Verdict != vr.VerdictPass {
		t.Errorf("verdict = %v, want pass", got.Verdict)
	}
}

func TestEvaluate_ToolRanButExpectationMissedIsToolFailure(t *testing.T) {
	// The tool ran but its success expectation did not hold → the tool or
	// the template/golden regressed.
	got := vrun.Evaluate(vrun.EvaluatorInput{ToolResult: &vrun.ToolResult{
		Ran: true, ExpectationMet: false, ErrorReason: "2 phase(s) failed",
	}})
	if got.Verdict != vr.VerdictToolFailure {
		t.Errorf("verdict = %v, want tool_failure", got.Verdict)
	}
	if got.ErrorMessage != "2 phase(s) failed" {
		t.Errorf("error = %q, want %q", got.ErrorMessage, "2 phase(s) failed")
	}
}

func TestEvaluate_ToolCouldNotRunIsRunFailure(t *testing.T) {
	// The tool never executed (missing binary, failed prep step) → this is
	// a run failure, NOT a tool/template regression.
	got := vrun.Evaluate(vrun.EvaluatorInput{ToolResult: &vrun.ToolResult{
		Ran: false, ErrorReason: "binary missing",
	}})
	if got.Verdict != vr.VerdictRunFailure {
		t.Errorf("verdict = %v, want run_failure", got.Verdict)
	}
	if got.ErrorMessage != "binary missing" {
		t.Errorf("error = %q, want %q", got.ErrorMessage, "binary missing")
	}
}

func TestEvaluate_RunFailureShortCircuits(t *testing.T) {
	got := vrun.Evaluate(vrun.EvaluatorInput{RunFailureReason: "agent crashed"})
	if got.Verdict != vr.VerdictRunFailure {
		t.Errorf("verdict = %v, want run_failure", got.Verdict)
	}
}

func TestEvaluate_DelegatesToManifest(t *testing.T) {
	m := manifest.Manifest{AllowedPaths: []string{"src/**"}}
	summary := vrun.RunSummary{DiffPaths: []manifest.DiffFile{{Path: "secrets/x"}}}
	got := vrun.Evaluate(vrun.EvaluatorInput{Manifest: m, Summary: summary})
	if got.Verdict != vr.VerdictUnexpectedMutation {
		t.Errorf("verdict = %v, want unexpected_mutation", got.Verdict)
	}
	if len(got.Violations) != 1 {
		t.Errorf("violations = %d, want 1", len(got.Violations))
	}
}

func TestEvaluate_WildcardManifestPasses(t *testing.T) {
	m := manifest.Manifest{WildcardAllowed: true}
	summary := vrun.RunSummary{DiffPaths: []manifest.DiffFile{{Path: "anywhere"}}}
	got := vrun.Evaluate(vrun.EvaluatorInput{Manifest: m, Summary: summary})
	if got.Verdict != vr.VerdictPass {
		t.Errorf("verdict = %v, want pass", got.Verdict)
	}
}
