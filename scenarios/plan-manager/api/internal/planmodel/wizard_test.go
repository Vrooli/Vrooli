package planmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnlyRecommendedKeepsRecommendedAction(t *testing.T) {
	step := GuidedStep{NextActions: []NextAction{
		{ID: "optional", Kind: NextActionOptional},
		{ID: "recommended", Kind: NextActionRecommended},
		{ID: "recovery", Kind: NextActionRecovery},
	}}

	got := OnlyRecommended(step)

	require.Equal(t, []NextAction{{ID: "recommended", Kind: NextActionRecommended}}, got.NextActions)
	require.Len(t, step.NextActions, 3, "input step must not be structurally truncated")
}

func TestOnlyRecommendedFallsBackToFirstAction(t *testing.T) {
	step := GuidedStep{NextActions: []NextAction{
		{ID: "optional", Kind: NextActionOptional},
		{ID: "recovery", Kind: NextActionRecovery},
	}}

	got := OnlyRecommended(step)

	require.Equal(t, []NextAction{{ID: "optional", Kind: NextActionOptional}}, got.NextActions)
}
