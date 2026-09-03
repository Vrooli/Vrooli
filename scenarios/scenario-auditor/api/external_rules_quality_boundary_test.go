package main

import "testing"

func TestQualityHealthRulesAreNotScenarioAuditorExternalRules(t *testing.T) {
	registerDefaultExternalProviders()

	qualityRuleIDs := []string{
		"TS_CONFIG_STRICT",
		"ESLINT_SAFETY_RULES",
		"TS_DANGEROUS_PATTERNS",
		"ESLINT_TYPED_CONFIG",
		"TYPECHECK_PLANNER_COVERAGE",
		"TESTING_CONFIG_LINT_STRICT",
		"GO_MOD_PRESENT_FOR_API_OR_CLI",
		"GO_LINT_CONFIG_PRESENT",
		"GO_LINT_REQUIRED_LINTERS",
		"MAKEFILE_QUALITY_GATES",
	}

	for _, ruleID := range qualityRuleIDs {
		if isExternalRule(ruleID) {
			t.Fatalf("%s must be owned by quality-health, not scenario-auditor external providers", ruleID)
		}
	}
}
