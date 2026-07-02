package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

func TestSplitCSV(t *testing.T) {
	require.Equal(t, []string{"api", "ui"}, splitCSV([]string{"api,ui", "ui"}))
}

func TestFirstFlag(t *testing.T) {
	require.Equal(t, "scenarios/foo", firstFlag([]string{"", "  scenarios/foo  "}))
	require.Equal(t, "", firstFlag(nil))
}

func TestFindingLinesIncludesPolicyEvidence(t *testing.T) {
	lines := findingLines([]*validationv1.ValidationFinding{{
		Severity:      "error",
		Code:          "UNIT_POLICY_PROJECTION_DRIFT",
		FilePath:      "ui/vite.config.ts",
		Message:       "Native test configuration drifts from the unit policy profile.",
		Evidence:      "coverage.thresholds.lines=70",
		Expected:      "coverage thresholds >= 85",
		Observed:      "coverage thresholds below policy",
		WhyItMatters:  "Template policy cannot be trusted when native config is weaker.",
		Remediation:   "Restore V8 coverage thresholds to the policy baseline.",
		SourceCommand: "unit-health validate scenario demo --json",
	}})

	require.Equal(t, []string{
		"[error] UNIT_POLICY_PROJECTION_DRIFT ui/vite.config.ts: Native test configuration drifts from the unit policy profile.",
		"  evidence: coverage.thresholds.lines=70",
		"  expected: coverage thresholds >= 85",
		"  observed: coverage thresholds below policy",
		"  why: Template policy cannot be trusted when native config is weaker.",
		"  remediation: Restore V8 coverage thresholds to the policy baseline.",
		"  source: unit-health validate scenario demo --json",
	}, lines)
}

func TestFindingDetailLinesOmitsEmptyFields(t *testing.T) {
	lines := findingDetailLines(&validationv1.ValidationFinding{
		Evidence: "role=cli",
		Expected: "Code Facts observes the cli role.",
	})

	require.Equal(t, []string{
		"evidence: role=cli",
		"expected: Code Facts observes the cli role.",
	}, lines)
}
