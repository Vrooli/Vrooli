package conformance_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/conformance"
)

func TestMeasureQualityRecordsRecognitionAndPresentationMetrics(t *testing.T) {
	assertions := conformance.MeasureQuality(conformance.QualityObservation{
		Reference:             "Hello world.",
		Hypothesis:            "Hello world.",
		MaxWER:                0,
		MinPunctuationRate:    0.5,
		MinCapitalisationRate: 0.5,
	})
	require.Len(t, assertions, 3)
	for _, assertion := range assertions {
		require.Equal(t, conformance.OutcomePassed, assertion.Outcome)
		require.True(t, conformance.QualityDetailIsMachineReadable(assertion))
	}
}

func TestMeasureQualityFailsObservedQualityRegression(t *testing.T) {
	assertions := conformance.MeasureQuality(conformance.QualityObservation{
		Reference:             "Hello world.",
		Hypothesis:            "goodbye world",
		MaxWER:                0,
		MinPunctuationRate:    1,
		MinCapitalisationRate: 1,
	})
	require.Equal(t, conformance.OutcomeFailed, assertions[0].Outcome)
	require.Equal(t, conformance.OutcomeFailed, assertions[1].Outcome)
}
