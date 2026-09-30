package rules

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
)

func TestMaturitySpecCoversQualityHealthFindings(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	require.Len(t, spec.Capabilities, 5)

	for _, rule := range Registry() {
		mapping, ok := spec.Findings[rule.ID]
		if !ok {
			t.Fatalf("maturity spec missing rule %s", rule.ID)
		}
		require.NotEmpty(t, mapping.CapabilityID, "rule %s must declare capability_id", rule.ID)
	}
	mapping, ok := spec.Findings[RuleCoverageGap]
	if !ok {
		t.Fatalf("maturity spec missing rule %s", RuleCoverageGap)
	}
	require.NotEmpty(t, mapping.CapabilityID, "rule %s must declare capability_id", RuleCoverageGap)
	require.NotEmpty(t, spec.Fallback.CapabilityID, "fallback must declare capability_id")
}
