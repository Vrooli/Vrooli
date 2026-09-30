package validation_run

import (
	manifest "development-toolchain-validator/internal/manifest"
	vr "development-toolchain-validator/internal/validation_record"
)

// EvaluatorInput is the pure-policy input the run evaluator consumes.
type EvaluatorInput struct {
	Manifest manifest.Manifest
	Summary  RunSummary

	// RunFailureReason is non-empty when the agent-manager run itself
	// failed; the evaluator short-circuits to VerdictRunFailure.
	RunFailureReason string

	// ToolResult is set for TUPLE_KIND_TOOL runs. Skill runs leave it
	// zero.
	ToolResult *ToolResult
}

// EvaluatorVerdict pairs the final terminal verdict with the
// manifest evaluator's per-path violations (when applicable) and a
// human-safe error message (when applicable).
type EvaluatorVerdict struct {
	Verdict      vr.Verdict
	ErrorMessage string
	Violations   []manifest.Violation
}

// Evaluate composes manifest-based diff classification with
// run-failure / tool-failure detection. Pure policy — no I/O.
func Evaluate(in EvaluatorInput) EvaluatorVerdict {
	if in.ToolResult != nil {
		tr := in.ToolResult
		switch {
		case !tr.Ran:
			// The tool could not be executed at all (missing binary,
			// unknown tool, failed preparatory step, timeout before any
			// run) — this is a run failure, not a tool/template defect.
			return EvaluatorVerdict{
				Verdict:      vr.VerdictRunFailure,
				ErrorMessage: tr.ErrorReason,
			}
		case !tr.ExpectationMet:
			// The tool ran but its success expectation did not hold:
			// either the tool regressed or the template/golden drifted.
			return EvaluatorVerdict{
				Verdict:      vr.VerdictToolFailure,
				ErrorMessage: tr.ErrorReason,
			}
		default:
			return EvaluatorVerdict{Verdict: vr.VerdictPass}
		}
	}
	if in.RunFailureReason != "" {
		return EvaluatorVerdict{
			Verdict:      vr.VerdictRunFailure,
			ErrorMessage: in.RunFailureReason,
		}
	}
	pol := manifest.Evaluate(in.Manifest, in.Summary.DiffPaths)
	if pol.Kind == manifest.VerdictUnexpectedMutation {
		msg := "diff violates manifest"
		if len(pol.Violations) > 0 {
			msg = pol.Violations[0].Reason
		}
		return EvaluatorVerdict{
			Verdict:      vr.VerdictUnexpectedMutation,
			ErrorMessage: msg,
			Violations:   pol.Violations,
		}
	}
	return EvaluatorVerdict{Verdict: vr.VerdictPass}
}
