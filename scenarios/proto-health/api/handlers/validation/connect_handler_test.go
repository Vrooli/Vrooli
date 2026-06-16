package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	internal "proto-health/internal/validation"
)

func TestBuildMaturityAssessmentMapsProtoFindings(t *testing.T) {
	spec := &assessment.Spec{
		Provider: "proto-health",
		Phase:    "proto",
		Version:  "1.0.0",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No proto surface"},
			{ID: "L1", Name: "Organized proto packages"},
			{ID: "L2", Name: "Generated artifacts synchronized"},
		},
		Findings: map[string]assessment.FindingMapping{
			internal.CodeGenOutOfSync: {
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactFoundationBlocker,
				Dimension:           "proto-health",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"proto-contract-audit"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "proto-health",
			SeverityDefault:  "SEVERITY_WARNING",
		},
	}
	require.NoError(t, assessment.ValidateSpec(*spec))

	got, err := buildMaturityAssessment(internal.Report{
		Scenario: "demo",
		Findings: []internal.Finding{{
			Code:       internal.CodeGenOutOfSync,
			Severity:   internal.SeverityError,
			Location:   "packages/proto/gen",
			Message:    "generated proto artifacts are out of sync",
			Suggestion: "run cd packages/proto && make generate",
		}},
	}, spec)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "proto-health", got.GetProvider())
	require.Equal(t, "proto", got.GetPhase())
	require.Equal(t, "L1", got.GetLocal().GetCurrentLevel())
	require.Equal(t, "L2", got.GetLocal().GetNextLevel())
	require.Equal(t, []string{internal.CodeGenOutOfSync}, got.GetLocal().GetBlockingFindingCodes())
	require.EqualValues(t, 1, got.GetFindingsByGlobalImpact()["foundation_blocker"])
	require.Equal(t, []string{"proto-contract-audit"}, got.GetRecommendedSkillIds())
	require.Equal(t, "proto-health", got.GetFindings()[0].GetMaturity().GetDimension())
}
