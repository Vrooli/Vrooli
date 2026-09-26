package calibrate

import (
	"testing"

	"github.com/stretchr/testify/require"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

func TestCalibrationCaseLinesKeepMismatchEvidence(t *testing.T) {
	lines := calibrationCaseLines([]*validationv1.CaseOutcome{{Id: "C001", RuleId: "assertion-observation", Matched: false, Status: "mismatch", Differences: []string{"missing observation"}}})
	require.Equal(t, "[mismatch] C001 rule=assertion-observation: missing observation", lines[0])
}

func TestHoldoutLinesExposeNonPromotion(t *testing.T) {
	lines := holdoutLines([]*validationv1.HoldoutComparison{{HoldoutId: "h1", RuleId: "assertion-observation", Labelled: 30, Observed: 30, FpRate: 0.01, PromotionAllowed: false}})
	require.Contains(t, lines[1], "labelled=30")
	require.Contains(t, lines[1], "promotion_allowed=false")
}
