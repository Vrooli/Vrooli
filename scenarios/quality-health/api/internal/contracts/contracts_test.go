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
	require.Equal(t, "typescript-static-quality", contract.ID)
	require.True(t, contract.AutofixAvailable)
}

func TestListMatchesTypeScriptByLanguageAlone(t *testing.T) {
	// Language-keyed: the TS pack must apply to a typescript surface with no
	// framework and no surface kind (any surface name, not just "ui").
	got := List("typescript", "", "", nil)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	require.Contains(t, ids, "typescript-static-quality")
	require.NotContains(t, ids, "go-static-quality")
}

func TestListMatchesGoByLanguageAlone(t *testing.T) {
	// Language-keyed: the Go pack must apply to a go surface regardless of
	// surface kind (worker/edge/etc., not just api/cli).
	got := List("go", "", "worker", nil)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	require.Contains(t, ids, "go-static-quality")
	require.NotContains(t, ids, "typescript-static-quality")
}
