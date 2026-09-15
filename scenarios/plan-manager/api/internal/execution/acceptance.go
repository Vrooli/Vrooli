package execution

import "strings"

// assessmentFailure checks the completeness of caller judgment, not the truth of
// a test result. Producer observations remain unchanged and independently readable.
func assessmentFailure(a OutcomeAssessment) string {
	if strings.TrimSpace(a.Summary) == "" {
		return "record an outcome assessment with a summary, evidence references, and limitations; broad validation is advisory"
	}
	if len(a.UnmetOutcomes) > 0 {
		return "intended outcomes remain unmet: " + strings.Join(a.UnmetOutcomes, "; ")
	}
	for _, ref := range a.Evidence {
		if strings.TrimSpace(ref) != "" {
			return ""
		}
	}
	return "outcome assessment requires a concrete observation, review, or focused-check reference; a green scenario suite is not required"
}

func assessmentForScopeFailure(a OutcomeAssessment, generation int) string {
	if reason := assessmentFailure(a); reason != "" {
		return reason
	}
	if a.ScopeGeneration != generation {
		return "review outcome assessment after scope expansion; retain applicable observations"
	}
	return ""
}
