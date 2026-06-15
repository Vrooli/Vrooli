package contracts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryIncludesMigratedRuleIDs(t *testing.T) {
	var got []string
	for _, contract := range Registry() {
		got = append(got, contract.RuleIDs...)
	}
	for _, ruleID := range []string{
		RuleTSConfigStrict,
		RuleESLintSafetyRules,
		RuleTSDangerousPatterns,
		RuleESLintTypedConfig,
		RuleNodeBuildTypecheck,
		RuleTestingConfigStrict,
		RuleGoModPresent,
		RuleGoLintConfigPresent,
		RuleGoLintRequiredLinters,
		RuleMakefileQualityGates,
	} {
		require.Contains(t, got, ruleID)
	}
}

func TestByRuleFindsContract(t *testing.T) {
	contract, ok := ByRule(RuleTSConfigStrict)
	require.True(t, ok)
	require.Equal(t, "typescript-react-vite-ui", contract.ID)
	require.True(t, contract.AutofixAvailable)
}
