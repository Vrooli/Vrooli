package uiinterop

import "testing"

func TestApplyRuleDefSeverityMakesMetadataAuthoritative(t *testing.T) {
	result := RuleResult{Violations: []Violation{{Severity: "low"}, {Severity: "critical"}}}
	applyRuleDefSeverity(&result, RuleDef{Severity: "high"})
	for i, violation := range result.Violations {
		if violation.Severity != "high" {
			t.Fatalf("violation %d severity = %q, want metadata severity", i, violation.Severity)
		}
	}
}
