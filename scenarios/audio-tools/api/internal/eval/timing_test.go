//go:build audio_lab

package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScoreTimestampDrift_ReportsMatchedWordBoundaryErrors(t *testing.T) {
	ref := []TimedWord{{"Hello", 0, .2}, {"world", .3, .5}, {"again", .6, .8}, {"today", .9, 1.1}}
	hyp := []TimedWord{{"hello!", .01, .22}, {"world", .32, .54}, {"wrong", .61, .81}, {"today", .95, 1.2}, {"extra", 1.3, 1.4}}
	got, err := ScoreTimestampDrift(ref, hyp)
	require.NoError(t, err)
	require.Equal(t, 4, got.ReferenceWords)
	require.Equal(t, 3, got.MatchedWords)
	require.InDelta(t, .75, got.Coverage, 1e-9)
	require.InDelta(t, 80.0/3, got.StartBiasMs, 1e-9) // 10, 20, 50 ms
	require.InDelta(t, 160.0/3, got.EndBiasMs, 1e-9)  // 20, 40, 100 ms
	require.InDelta(t, 20, got.StartAbsP50Ms, 1e-9)
	require.InDelta(t, 47, got.StartAbsP95Ms, 1e-9)
	require.InDelta(t, 40, got.EndAbsP50Ms, 1e-9)
	require.InDelta(t, 94, got.EndAbsP95Ms, 1e-9)
	require.InDelta(t, 50, got.MaxStartAbsMs, 1e-9)
	require.InDelta(t, 100, got.MaxEndAbsMs, 1e-9)

	encoded, err := json.Marshal(ClipResult{TimestampDrift: &got})
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"timestamp_drift"`)
	require.Contains(t, string(encoded), `"matched_words":3`)
}

func TestScoreTimestampDrift_UnmatchedTokensReduceCoverage(t *testing.T) {
	got, err := ScoreTimestampDrift(
		[]TimedWord{{"one", 0, .1}, {"two", .2, .3}, {"three", .4, .5}},
		[]TimedWord{{"one", 0, .1}, {"wrong", .2, .3}},
	)
	require.NoError(t, err)
	require.Equal(t, 1, got.MatchedWords)
	require.InDelta(t, 1.0/3, got.Coverage, 1e-9)
	require.Zero(t, got.StartAbsP50Ms)
}

func TestScoreTimestampDrift_DoesNotReportZeroBoundaryErrorWithoutMatches(t *testing.T) {
	got, err := ScoreTimestampDrift(
		[]TimedWord{{"reference", 0, .2}},
		[]TimedWord{{"hypothesis", .1, .3}},
	)
	require.ErrorContains(t, err, "no exact lexical matches")
	// Lexical coverage is known, but boundary metrics are not. Callers must
	// preserve that distinction instead of serializing a misleading zero.
	require.Equal(t, 0, got.MatchedWords)
	require.Zero(t, got.Coverage)
}

func TestScoreTimestampDrift_SeparatesConstantOffsetFromProgressiveDrift(t *testing.T) {
	ref := []TimedWord{{"one", 0, .1}, {"two", 900, 900.1}, {"three", 1800, 1800.1}}
	constant := []TimedWord{{"one", .25, .35}, {"two", 900.25, 900.35}, {"three", 1800.25, 1800.35}}
	got, err := ScoreTimestampDrift(ref, constant)
	require.NoError(t, err)
	require.True(t, got.DriftRateObserved)
	require.InDelta(t, 0, got.StartDriftMsPerHour, 1e-6)
	require.InDelta(t, 0, got.EndDriftMsPerHour, 1e-6)
	require.InDelta(t, 250, got.StartAbsP50Ms, 1e-6)

	creeping := []TimedWord{{"one", 0, .1}, {"two", 900.5, 900.6}, {"three", 1801, 1801.1}}
	got, err = ScoreTimestampDrift(ref, creeping)
	require.NoError(t, err)
	require.True(t, got.DriftRateObserved)
	require.InDelta(t, 2000, got.StartDriftMsPerHour, 1e-6)
	require.InDelta(t, 2000, got.EndDriftMsPerHour, 1e-6)

	got, err = ScoreTimestampDrift(ref[:1], constant[:1])
	require.NoError(t, err)
	require.False(t, got.DriftRateObserved, "one matching position cannot establish a rate")
}

func TestScoreTimestampDrift_RejectsInvalidEvidence(t *testing.T) {
	valid := []TimedWord{{"word", 0, .1}}
	for name, pair := range map[string][2][]TimedWord{
		"empty reference":   {nil, valid},
		"empty hypothesis":  {valid, nil},
		"negative time":     {[]TimedWord{{"word", -1, .1}}, valid},
		"reversed interval": {valid, []TimedWord{{"word", .2, .1}}},
		"non-finite":        {valid, []TimedWord{{"word", 0, math.NaN()}}},
		"out of order":      {[]TimedWord{{"a", .3, .4}, {"b", .2, .5}}, valid},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ScoreTimestampDrift(pair[0], pair[1])
			require.Error(t, err)
		})
	}
}

func TestTimestampMatchesPreservesCanonicalMinimumEditPath(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 500; trial++ {
		n, m := rng.Intn(9), rng.Intn(9)
		ref, hyp := make([]string, n), make([]string, m)
		for i := range ref {
			ref[i] = string(rune('a' + rng.Intn(4)))
		}
		for i := range hyp {
			hyp[i] = string(rune('a' + rng.Intn(4)))
		}
		_, operations := AlignOperations(ref, hyp)
		want := make([][2]int, 0)
		for _, operation := range operations {
			if operation.Kind == "match" {
				want = append(want, [2]int{operation.ReferenceIndex, operation.HypothesisIndex})
			}
		}
		got, err := timestampMatches(ref, hyp)
		require.NoError(t, err)
		require.Equal(t, want, got, "trial=%d ref=%v hyp=%v", trial, ref, hyp)
	}
}

func TestTimestampMatchesRepeatedPrefixDoesNotForceEarlyMatch(t *testing.T) {
	// A shared leading token may be matched later on the canonical optimal
	// path when repeated tokens compete with deletion/insertion placement.
	ref, hyp := []string{"d", "a", "d", "d", "b", "d"}, []string{"d", "d"}
	got, err := timestampMatches(ref, hyp)
	require.NoError(t, err)
	require.Equal(t, [][2]int{{3, 0}, {5, 1}}, got)
}

func TestTimestampMatchesRejectsOversizedInputBeforeAllocation(t *testing.T) {
	words := make([]string, 24_000)
	for i := range words {
		words[i] = "word"
	}
	hypothesis := append([]string(nil), words...)
	hypothesis[len(hypothesis)-1] = "different"
	_, err := timestampMatches(words, hypothesis)
	require.ErrorContains(t, err, "exceeds 20000 words per side")
}

func TestTimestampMatchesAdmittedTwentyThousandWordIdentity(t *testing.T) {
	words := make([]string, 20_000)
	for i := range words {
		words[i] = "word"
	}
	got, err := timestampMatches(words, words)
	require.NoError(t, err)
	require.Len(t, got, len(words))
	require.Equal(t, [2]int{0, 0}, got[0])
	require.Equal(t, [2]int{len(words) - 1, len(words) - 1}, got[len(got)-1])
}

func TestTimestampMatchesLongExactSuffixAvoidsFullAlignment(t *testing.T) {
	words := make([]string, 20_000)
	for i := range words {
		words[i] = fmt.Sprintf("word%d", i)
	}
	hypothesis := append([]string(nil), words...)
	hypothesis[0] = "corrected"

	got, err := timestampMatches(words, hypothesis)
	require.NoError(t, err)
	require.Len(t, got, len(words)-1)
	require.Equal(t, [2]int{1, 1}, got[0])
	require.Equal(t, [2]int{len(words) - 1, len(words) - 1}, got[len(got)-1])
}

func TestTimestampMatchesUniquePrefixAndSuffixPreserveCanonicalPath(t *testing.T) {
	const wordCount = 20_000
	ref := make([]string, wordCount)
	for i := range ref {
		ref[i] = fmt.Sprintf("word%d", i)
	}
	hyp := append([]string(nil), ref...)
	const changed = wordCount / 2
	hyp[changed] = "corrected"

	got, err := timestampMatches(ref, hyp)
	require.NoError(t, err)
	require.Len(t, got, wordCount-1)
	require.Equal(t, [2]int{changed - 1, changed - 1}, got[changed-1])
	require.Equal(t, [2]int{changed + 1, changed + 1}, got[changed])
	require.Equal(t, [2]int{wordCount - 1, wordCount - 1}, got[len(got)-1])
}

func TestScoreTimestampDriftRejectsOversizedIdentityBeforeFastPath(t *testing.T) {
	words := make([]TimedWord, maxTimestampWords+1)
	for i := range words {
		start := float64(i) * .36
		words[i] = TimedWord{Word: "word", Start: start, End: start + .2}
	}
	_, err := ScoreTimestampDrift(words, words)
	require.ErrorContains(t, err, "exceeds 20000 words per side")
}

func TestTimestampMatchesRejectsOversizedIdentityBeforeFastPath(t *testing.T) {
	words := make([]string, maxTimestampWords+1)
	for i := range words {
		words[i] = "word"
	}
	_, err := timestampMatches(words, words)
	require.ErrorContains(t, err, "exceeds 20000 words per side")
}

func BenchmarkScoreTimestampDrift_61MinuteWordCount(b *testing.B) {
	const wordCount = 10_000
	ref, hyp := make([]TimedWord, wordCount), make([]TimedWord, wordCount)
	for i := 0; i < wordCount; i++ {
		word := "word"
		if i%17 == 0 {
			word = "changed"
		}
		start := float64(i) * .36
		ref[i] = TimedWord{Word: word, Start: start, End: start + .2}
		hyp[i] = TimedWord{Word: word, Start: start + .025, End: start + .23}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ScoreTimestampDrift(ref, hyp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScoreTimestampDrift_20000WordIdentity(b *testing.B) {
	const wordCount = 20_000
	ref, hyp := make([]TimedWord, wordCount), make([]TimedWord, wordCount)
	for i := 0; i < wordCount; i++ {
		start := float64(i) * .36
		ref[i] = TimedWord{Word: "word", Start: start, End: start + .2}
		hyp[i] = TimedWord{Word: "word", Start: start + .025, End: start + .23}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ScoreTimestampDrift(ref, hyp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScoreTimestampDrift_20000WordFirstTokenCorrection(b *testing.B) {
	const wordCount = 20_000
	ref, hyp := make([]TimedWord, wordCount), make([]TimedWord, wordCount)
	for i := 0; i < wordCount; i++ {
		word := fmt.Sprintf("word%d", i)
		start := float64(i) * .36
		ref[i] = TimedWord{Word: word, Start: start, End: start + .2}
		hyp[i] = TimedWord{Word: word, Start: start + .025, End: start + .23}
	}
	hyp[0].Word = "corrected"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ScoreTimestampDrift(ref, hyp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScoreTimestampDrift_20000WordMiddleCorrection(b *testing.B) {
	const wordCount = 20_000
	ref, hyp := make([]TimedWord, wordCount), make([]TimedWord, wordCount)
	for i := 0; i < wordCount; i++ {
		word := fmt.Sprintf("word%d", i)
		start := float64(i) * .36
		ref[i] = TimedWord{Word: word, Start: start, End: start + .2}
		hyp[i] = TimedWord{Word: word, Start: start + .025, End: start + .23}
	}
	hyp[wordCount/2].Word = "corrected"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ScoreTimestampDrift(ref, hyp); err != nil {
			b.Fatal(err)
		}
	}
}
