package validation_run_test

import (
	"testing"

	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
	vrun "development-toolchain-validator/internal/validation_run"
)

func TestEvaluate_ToolSuccessIsPass(t *testing.T) {
	got := vrun.Evaluate(vrun.EvaluatorInput{ToolResult: &vrun.ToolResult{Succeeded: true}})
	if got.Verdict != vr.VerdictPass {
		t.Errorf("verdict = %v, want pass", got.Verdict)
	}
}

func TestEvaluate_ToolFailureMapsToToolFailure(t *testing.T) {
	got := vrun.Evaluate(vrun.EvaluatorInput{ToolResult: &vrun.ToolResult{Succeeded: false, ErrorReason: "nope"}})
	if got.Verdict != vr.VerdictToolFailure {
		t.Errorf("verdict = %v, want tool_failure", got.Verdict)
	}
	if got.ErrorMessage != "nope" {
		t.Errorf("error = %q, want %q", got.ErrorMessage, "nope")
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
