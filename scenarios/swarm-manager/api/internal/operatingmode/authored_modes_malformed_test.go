package operatingmode

import "testing"

// declaredOutputForPhase loads a shipped mode from disk and returns the declared
// output of its (first non-delegated) phase — the contract the resolution ladder
// validates a round's result against.
func declaredOutputForPhase(t *testing.T, mode, phase string) *DeclaredOutput {
	t.Helper()
	def := loadModeFromDisk(t, mode)
	pd, err := def.PhaseDefinition(Phase(phase))
	if err != nil {
		t.Fatalf("phase %q: %v", phase, err)
	}
	if pd.DeclaredOutput == nil {
		t.Fatalf("mode %q phase %q declares no output", mode, phase)
	}
	return pd.DeclaredOutput
}

// TestAuthoredHandoffModesFailClosedOnMalformedOutput mutates each required
// field of a handoff mode's declared output and asserts the ladder gate
// (validateDeclaredOutput) reports it — a missing required field is missing, an
// out-of-type field is a violation. This is the malformed-output fail-closed
// contract for the authored operation modes.
func TestAuthoredHandoffModesFailClosedOnMalformedOutput(t *testing.T) {
	declared := declaredOutputForPhase(t, "backlog-research", "research")

	// The enriched (decision-B) workshop/research round result: handoff + the
	// lettered-option decisions + the 5-dimension readiness self-assessment.
	selfAssessment := func() map[string]any {
		return map[string]any{
			"problem_clarity": 2, "scope_defined": 2, "approach_solid": 2, "testable": 2, "risk_awareness": 2,
		}
	}
	good := map[string]any{
		"handoff": map[string]any{
			"summary": "did work", "blockers": []any{"none"}, "next_step": "next",
			"changed_files": []any{"spec.md"}, "tests": []any{"none"},
		},
		"decisions":       []any{},
		"self_assessment": selfAssessment(),
	}
	if missing, violations := validateDeclaredOutput(declared, good); len(missing) > 0 || len(violations) > 0 {
		t.Fatalf("valid enriched round rejected: missing=%v violations=%v", missing, violations)
	}

	// Decision-B required fields fail closed when absent.
	for _, drop := range []string{"decisions", "self_assessment"} {
		mutated := map[string]any{
			"handoff":         good["handoff"],
			"decisions":       []any{},
			"self_assessment": selfAssessment(),
		}
		delete(mutated, drop)
		if missing, _ := validateDeclaredOutput(declared, mutated); !containsPath(missing, drop) {
			t.Errorf("dropping required %q did not fail closed; missing=%v", drop, missing)
		}
	}
	// A missing self-assessment dimension is reported at its dotted path.
	{
		sa := selfAssessment()
		delete(sa, "risk_awareness")
		mutated := map[string]any{"handoff": good["handoff"], "decisions": []any{}, "self_assessment": sa}
		if missing, _ := validateDeclaredOutput(declared, mutated); !containsPath(missing, "self_assessment.risk_awareness") {
			t.Errorf("dropping self_assessment.risk_awareness did not fail closed; missing=%v", missing)
		}
	}

	// Drop each required handoff subfield in turn: every one must be reported missing.
	for _, sub := range []string{"summary", "blockers", "next_step", "changed_files", "tests"} {
		h := map[string]any{"summary": "s", "blockers": []any{"none"}, "next_step": "n", "changed_files": []any{"none"}, "tests": []any{"none"}}
		delete(h, sub)
		missing, _ := validateDeclaredOutput(declared, map[string]any{"handoff": h})
		if !containsPath(missing, "handoff."+sub) {
			t.Errorf("dropping required handoff.%s did not fail closed; missing=%v", sub, missing)
		}
	}

	// Missing the whole handoff object → missing.
	if missing, _ := validateDeclaredOutput(declared, map[string]any{}); !containsPath(missing, "handoff") {
		t.Errorf("missing handoff object not reported; missing=%v", missing)
	}

	// Wrong type for a required field → violation.
	bad := map[string]any{"handoff": map[string]any{
		"summary": 42, "blockers": []any{"none"}, "next_step": "n", "changed_files": []any{"none"}, "tests": []any{"none"},
	}}
	if _, violations := validateDeclaredOutput(declared, bad); len(violations) == 0 {
		t.Errorf("non-string handoff.summary not reported as a violation")
	}
}

// TestAuthoredReviewModesFailClosedOnBadVerdict proves the review modes' verdict
// enum is enforced: an out-of-enum verdict is a violation, a missing verdict is
// missing, and a declared verdict value passes.
func TestAuthoredReviewModesFailClosedOnBadVerdict(t *testing.T) {
	for _, tc := range []struct{ mode, phase string }{
		{"backlog-review", "review"},
		{"initiative-review-loop", "review"},
	} {
		declared := declaredOutputForPhase(t, tc.mode, tc.phase)
		if _, violations := validateDeclaredOutput(declared, map[string]any{"verdict": "shipped"}); len(violations) == 0 {
			t.Errorf("%s: out-of-enum verdict not reported as a violation", tc.mode)
		}
		if missing, _ := validateDeclaredOutput(declared, map[string]any{}); !containsPath(missing, "verdict") {
			t.Errorf("%s: missing verdict not reported", tc.mode)
		}
		if missing, violations := validateDeclaredOutput(declared, map[string]any{"verdict": "ready"}); len(missing) > 0 || len(violations) > 0 {
			t.Errorf("%s: valid verdict rejected: missing=%v violations=%v", tc.mode, missing, violations)
		}
	}
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
