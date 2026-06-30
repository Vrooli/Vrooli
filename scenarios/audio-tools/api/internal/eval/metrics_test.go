package eval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWER_BreakdownAndRate(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		hyp     string
		wantS   int
		wantI   int
		wantD   int
		wantN   int
		wantErr float64
	}{
		{"identical", "the quick brown fox", "the quick brown fox", 0, 0, 0, 4, 0},
		{"one_substitution", "the quick brown fox", "the quick green fox", 1, 0, 0, 4, 0.25},
		{"one_deletion", "the quick brown fox", "the quick fox", 0, 0, 1, 4, 0.25},
		{"one_insertion", "the quick fox", "the quick brown fox", 0, 1, 0, 3, 1.0 / 3.0},
		{"all_wrong", "alpha beta", "gamma delta", 2, 0, 0, 2, 1.0},
		{"empty_hyp_all_deletions", "alpha beta gamma", "", 0, 0, 3, 3, 1.0},
		{"empty_ref_empty_hyp", "", "", 0, 0, 0, 0, 0},
		{"empty_ref_with_hyp", "", "spurious words", 0, 2, 0, 0, 1.0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			opts := DefaultNormalizeOptions()
			got := WER(Tokenize(tc.ref, opts), Tokenize(tc.hyp, opts))
			require.Equal(t, tc.wantS, got.Substitutions, "substitutions")
			require.Equal(t, tc.wantI, got.Insertions, "insertions")
			require.Equal(t, tc.wantD, got.Deletions, "deletions")
			require.Equal(t, tc.wantN, got.RefWords, "ref words")
			require.InDelta(t, tc.wantErr, got.Rate(), 1e-9, "rate")
		})
	}
}

// TestWER_NormalizationCollapsesCosmetics proves capitalization and
// punctuation differences do not count as errors — the WER reflects real
// recognition errors only.
func TestWER_NormalizationCollapsesCosmetics(t *testing.T) {
	opts := DefaultNormalizeOptions()
	got := WER(Tokenize("Hello, World! How are you?", opts), Tokenize("hello world how are you", opts))
	require.Equal(t, 0, got.Total(), "case + punctuation differences must normalize to zero error")
	require.InDelta(t, 0.0, got.Rate(), 1e-9)
}

func TestCER_CharacterLevel(t *testing.T) {
	// "kitten" -> "sitten" (1 sub) is the classic Levenshtein example.
	got := CER("kitten", "sitting")
	// kitten->sitting = 3 edits (k->s, e->i, +g). Standard distance.
	require.Equal(t, 3, got.Total())
}

func TestRTF(t *testing.T) {
	require.InDelta(t, 0.5, RTF(500*time.Millisecond, time.Second), 1e-9)
	require.InDelta(t, 2.0, RTF(2*time.Second, time.Second), 1e-9)
	require.Equal(t, 0.0, RTF(time.Second, 0), "undefined audio duration -> 0")
}

func TestPercentile(t *testing.T) {
	samples := []float64{10, 20, 30, 40, 50}
	require.InDelta(t, 30, P50(samples), 1e-9)
	require.InDelta(t, 10, Percentile(samples, 0), 1e-9)
	require.InDelta(t, 50, Percentile(samples, 100), 1e-9)
	// p95 of 5 evenly-spaced samples interpolates between 40 and 50.
	require.InDelta(t, 48, P95(samples), 1e-9)
	require.Equal(t, 0.0, P50(nil), "empty -> 0")
	require.InDelta(t, 7, P50([]float64{7}), 1e-9, "single sample")
	// Percentile must not mutate the caller's slice ordering.
	unsorted := []float64{50, 10, 30}
	_ = P50(unsorted)
	require.Equal(t, []float64{50, 10, 30}, unsorted, "input slice must be untouched")
}

func TestMean(t *testing.T) {
	require.InDelta(t, 20, Mean([]float64{10, 20, 30}), 1e-9)
	require.Equal(t, 0.0, Mean(nil))
}

func TestPartialRevisions(t *testing.T) {
	require.Equal(t, 0, PartialRevisions(nil))
	require.Equal(t, 1, PartialRevisions([]string{"hello"}))
	// "hello" shown, then unchanged, then changed twice -> 3 revisions.
	require.Equal(t, 3, PartialRevisions([]string{"hello", "hello", "hello world", "hello world how"}))
}

func TestAlign_TieBreakDeterministic(t *testing.T) {
	// Run a non-trivial alignment many times; the breakdown must be stable.
	ref := []string{"a", "b", "c", "d", "e"}
	hyp := []string{"a", "x", "c", "y", "e"}
	first := Align(ref, hyp)
	for i := 0; i < 20; i++ {
		require.Equal(t, first, Align(ref, hyp), "alignment breakdown must be deterministic")
	}
	require.Equal(t, 2, first.Substitutions)
	require.Equal(t, 0, first.Insertions)
	require.Equal(t, 0, first.Deletions)
}

func TestAlignOperations_AllEditKinds(t *testing.T) {
	ref := []string{"alpha", "bravo", "charlie", "delta"}
	hyp := []string{"alpha", "brown", "charlie", "echo", "delta"}

	counts, ops := AlignOperations(ref, hyp)
	require.Equal(t, EditCounts{Substitutions: 1, Insertions: 1}, counts)
	require.Equal(t, []EditOperation{
		{Kind: "match", ReferenceToken: "alpha", HypothesisToken: "alpha", ReferenceIndex: 0, HypothesisIndex: 0},
		{Kind: "substitution", ReferenceToken: "bravo", HypothesisToken: "brown", ReferenceIndex: 1, HypothesisIndex: 1},
		{Kind: "match", ReferenceToken: "charlie", HypothesisToken: "charlie", ReferenceIndex: 2, HypothesisIndex: 2},
		{Kind: "insertion", HypothesisToken: "echo", ReferenceIndex: -1, HypothesisIndex: 3},
		{Kind: "match", ReferenceToken: "delta", HypothesisToken: "delta", ReferenceIndex: 3, HypothesisIndex: 4},
	}, ops)

	counts, ops = AlignOperations([]string{"alpha", "bravo"}, []string{"alpha"})
	require.Equal(t, EditCounts{Deletions: 1}, counts)
	require.Equal(t, "deletion", ops[1].Kind)
	require.Equal(t, "bravo", ops[1].ReferenceToken)
}

func TestWERResult_RateGuards(t *testing.T) {
	require.InDelta(t, 0.0, WERResult{RefWords: 0, HypWords: 0}.Rate(), 1e-9)
	require.InDelta(t, 1.0, WERResult{RefWords: 0, HypWords: 3, EditCounts: EditCounts{Insertions: 3}}.Rate(), 1e-9)
}

func TestSafetyGates_CleanPasses(t *testing.T) {
	opts := DefaultNormalizeOptions()
	ref := Tokenize("alpha bravo charlie", opts)
	hyp := Tokenize("alpha bravo charlie", opts)
	_, ops := AlignOperations(ref, hyp)

	got := EvaluateSafety(ref, hyp, ops, []CommitState{
		{Text: "alpha", AtMs: 100},
		{Text: "alpha bravo", AtMs: 200},
		{Text: "alpha bravo charlie", AtMs: 300},
	}, SafetyOptions{})

	require.True(t, got.Passed)
	require.True(t, got.RetractionFree)
	require.True(t, got.DroppedSpanFree)
	require.Equal(t, 0, got.MaxDroppedSpanWords)
	require.Equal(t, DefaultDroppedSpanThresholdWords, got.DroppedSpanThresholdWords)
}

func TestSafetyGates_DetectsCommittedTextRetraction(t *testing.T) {
	opts := DefaultNormalizeOptions()
	ref := Tokenize("alpha bravo charlie", opts)
	hyp := Tokenize("alpha delta charlie", opts)
	_, ops := AlignOperations(ref, hyp)

	got := EvaluateSafety(ref, hyp, ops, []CommitState{
		{Text: "alpha bravo", AtMs: 100},
		{Text: "alpha delta", AtMs: 200},
	}, SafetyOptions{})

	require.False(t, got.Passed)
	require.False(t, got.RetractionFree)
	require.Len(t, got.RetractionEvents, 1)
	require.Equal(t, "alpha bravo", got.RetractionEvents[0].PreviousText)
	require.Equal(t, "alpha delta", got.RetractionEvents[0].CurrentText)
}

func TestSafetyGates_DetectsThresholdSizedDroppedSpan(t *testing.T) {
	opts := DefaultNormalizeOptions()
	ref := Tokenize("one two three four five six", opts)
	hyp := Tokenize("one six", opts)
	_, ops := AlignOperations(ref, hyp)

	got := EvaluateSafety(ref, hyp, ops, []CommitState{{Text: "one six", AtMs: 100}}, SafetyOptions{
		DroppedSpanThresholdWords: 4,
	})

	require.False(t, got.Passed)
	require.True(t, got.RetractionFree)
	require.False(t, got.DroppedSpanFree)
	require.Equal(t, 4, got.MaxDroppedSpanWords)
}
