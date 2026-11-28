//go:build audio_lab

package eval

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// This deterministic 61-minute fixture exercises the scoring path at a
// representative long-form word count. The known constant offsets affect the
// absolute error and bias metrics; the known slopes affect only drift rate.
// It validates the metric's arithmetic, not any model's real-world timing.
func TestScoreTimestampDrift_LongFormOffsetAndDriftFixture(t *testing.T) {
	const (
		wordCount     = 10_000
		durationSec   = 61 * 60
		startOffsetMs = 120.0
		startRateMsHr = 75.0
		endOffsetMs   = -80.0
		endRateMsHr   = -40.0
	)
	reference := make([]TimedWord, wordCount)
	hypothesis := make([]TimedWord, wordCount)
	for i := 0; i < wordCount; i++ {
		start := float64(i) * float64(durationSec) / float64(wordCount)
		end := start + 0.4
		elapsedHours := start / 3600
		word := fmt.Sprintf("token%d", i)
		reference[i] = TimedWord{Word: word, Start: start, End: end}
		hypothesis[i] = TimedWord{
			Word:  word,
			Start: start + (startOffsetMs+startRateMsHr*elapsedHours)/1000,
			End:   end + (endOffsetMs+endRateMsHr*elapsedHours)/1000,
		}
	}

	got, err := ScoreTimestampDrift(reference, hypothesis)
	require.NoError(t, err)
	require.Equal(t, wordCount, got.ReferenceWords)
	require.Equal(t, wordCount, got.MatchedWords)
	require.Equal(t, 1.0, got.Coverage)
	require.True(t, got.DriftRateObserved)
	meanStart := float64(wordCount-1) * float64(durationSec) / (2 * float64(wordCount))
	require.InDelta(t, startOffsetMs+startRateMsHr*meanStart/3600, got.StartBiasMs, 1e-6)
	require.InDelta(t, endOffsetMs+endRateMsHr*meanStart/3600, got.EndBiasMs, 1e-6)
	require.InDelta(t, startRateMsHr, got.StartDriftMsPerHour, 1e-5)
	require.InDelta(t, endRateMsHr, got.EndDriftMsPerHour, 1e-5)
	lastStart := float64(wordCount-1) * float64(durationSec) / float64(wordCount)
	require.InDelta(t, 120+75*lastStart/3600, got.MaxStartAbsMs, 1e-6)
	require.InDelta(t, 80+40*lastStart/3600, got.MaxEndAbsMs, 1e-6)
}

// This fixture keeps the long-form regression representative of real ASR
// output: repeated function words surround a localized lexical error. It
// verifies that an edit does not shift every subsequent timing match or hide
// the resulting coverage loss.
func TestScoreTimestampDrift_LongFormLocalizedEditWithRepeatedWords(t *testing.T) {
	const (
		wordCount   = 10_000
		durationSec = 61 * 60
		errorIndex  = wordCount / 2
	)
	reference := make([]TimedWord, wordCount)
	hypothesis := make([]TimedWord, 0, wordCount)
	for i := 0; i < wordCount; i++ {
		start := float64(i) * float64(durationSec) / float64(wordCount)
		word := fmt.Sprintf("token%d", i)
		if i%11 == 0 {
			word = "the"
		}
		end := start + 0.4
		reference[i] = TimedWord{Word: word, Start: start, End: end}
		if i == errorIndex {
			continue // one omitted reference word
		}
		elapsedHours := start / 3600
		hypothesis = append(hypothesis, TimedWord{
			Word:  word,
			Start: start + (120+75*elapsedHours)/1000,
			End:   end + (-80-40*elapsedHours)/1000,
		})
	}

	got, err := ScoreTimestampDrift(reference, hypothesis)
	require.NoError(t, err)
	require.Equal(t, wordCount, got.ReferenceWords)
	require.Equal(t, wordCount-1, got.MatchedWords)
	require.InDelta(t, float64(wordCount-1)/wordCount, got.Coverage, 1e-12)
	require.True(t, got.DriftRateObserved)
	// A single removed word must not make the remaining stream appear shifted.
	meanStart := float64(wordCount-1) * float64(durationSec) / (2 * float64(wordCount))
	require.InDelta(t, 120+75*meanStart/3600, got.StartBiasMs, 0.02)
	require.InDelta(t, -80-40*meanStart/3600, got.EndBiasMs, 0.02)
	require.InDelta(t, 75, got.StartDriftMsPerHour, 0.01)
	require.InDelta(t, -40, got.EndDriftMsPerHour, 0.01)
}
