//go:build audio_lab

package eval

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScoreTimestampDriftRejectsFiniteInputsThatOverflowDerivedMetrics(t *testing.T) {
	// Each timestamp is finite and individually ordered, but summing these
	// extreme absolute audio positions overflows the least-squares regression.
	// Returning NaN rates as a successful score would break report JSON and
	// make downstream threshold comparisons unreliable.
	reference := []TimedWord{
		{Word: "first", Start: math.MaxFloat64 * .75, End: math.MaxFloat64 * .75},
		{Word: "second", Start: math.MaxFloat64 * .75, End: math.MaxFloat64 * .75},
	}
	hypothesis := append([]TimedWord(nil), reference...)

	got, err := ScoreTimestampDrift(reference, hypothesis)
	require.ErrorContains(t, err, "non-finite derived metrics")
	require.Equal(t, 2, got.MatchedWords)
	require.True(t, math.IsNaN(got.StartDriftMsPerHour) || math.IsInf(got.StartDriftMsPerHour, 0))
}

func TestScoreTimestampDriftRejectsBoundaryErrorMillisecondOverflow(t *testing.T) {
	reference := []TimedWord{{Word: "word", Start: 0, End: 0}}
	hypothesis := []TimedWord{{Word: "word", Start: math.MaxFloat64 * .75, End: math.MaxFloat64 * .75}}

	_, err := ScoreTimestampDrift(reference, hypothesis)
	require.ErrorContains(t, err, "non-finite derived metrics")
}
