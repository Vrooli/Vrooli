package strategy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLongestAgreedPrefix exercises the token-level agreement core of
// LocalAgreement directly: it gates commits, so its edge cases (too few runs,
// partial-word divergence, full divergence) must be exact.
func TestLongestAgreedPrefix(t *testing.T) {
	cases := []struct {
		name       string
		runs       []string
		commitRuns int
		want       string
	}{
		{
			name:       "fewer runs than commitRuns yields nothing",
			runs:       []string{"hello world"},
			commitRuns: 2,
			want:       "",
		},
		{
			name:       "identical runs agree on the full text",
			runs:       []string{"hello world", "hello world"},
			commitRuns: 2,
			want:       "hello world",
		},
		{
			name:       "agreement stops at the first differing token",
			runs:       []string{"the quick red", "the quick brown"},
			commitRuns: 2,
			want:       "the quick",
		},
		{
			name:       "partial-word does not commit until confirmed (token-level, not char-level)",
			runs:       []string{"hello wor", "hello world"},
			commitRuns: 2,
			want:       "hello",
		},
		{
			name:       "full divergence on the first token yields nothing",
			runs:       []string{"alpha beta", "gamma delta"},
			commitRuns: 2,
			want:       "",
		},
		{
			name:       "three runs must all agree (commitRuns=3)",
			runs:       []string{"a b c", "a b c", "a b x"},
			commitRuns: 3,
			want:       "a b",
		},
		{
			name:       "only the last commitRuns runs are considered when more are passed",
			runs:       []string{"totally different", "a b", "a b"},
			commitRuns: 2,
			want:       "a b",
		},
		{
			name:       "empty transcripts agree on nothing",
			runs:       []string{"", ""},
			commitRuns: 2,
			want:       "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The caller trims `runs` to the last commitRuns before calling, so
			// mirror that to exercise the function as it is actually used.
			runs := tc.runs
			if len(runs) > tc.commitRuns {
				runs = runs[len(runs)-tc.commitRuns:]
			}
			require.Equal(t, tc.want, longestAgreedPrefix(runs, tc.commitRuns, 0))
		})
	}
}

// TestLongestAgreedPrefix_MaxTokensCap proves the maxTokens cap bounds
// the agreement walk regardless of how long the agreeing prefix could
// be. This is the Phase-B knob that keeps variance accumulation
// bounded on long uncommitted buffers.
func TestLongestAgreedPrefix_MaxTokensCap(t *testing.T) {
	runs := []string{
		"the quick brown fox jumps high",
		"the quick brown fox flies low",
	}
	// Unbounded: agreement naturally stops at token 5 ("jumps" vs "flies").
	require.Equal(t, "the quick brown fox", longestAgreedPrefix(runs, 2, 0))
	// Cap at 4: walk stops at the cap before reaching the natural divergence.
	require.Equal(t, "the quick brown fox", longestAgreedPrefix(runs, 2, 4))
	// Cap at 2: only the first two tokens considered.
	require.Equal(t, "the quick", longestAgreedPrefix(runs, 2, 2))
}

// TestLongestAgreedPrefix_CaseAndPunctuationNormalization is the
// regression test for "Whisper jitters capitalization/punctuation".
// Without normalization, "Hello world" vs "hello world." would yield
// zero agreement at position 0 and the algorithm would never commit
// on real audio.
func TestLongestAgreedPrefix_CaseAndPunctuationNormalization(t *testing.T) {
	runs := []string{
		"Hello, World how are you",
		"hello world.  How are you?",
	}
	got := longestAgreedPrefix(runs, 2, 0)
	// Returned tokens use the FIRST run's verbatim form (so committed
	// text preserves Whisper's chosen casing/punct), and the walk
	// proceeds case- and trailing-punct-insensitively.
	require.Equal(t, "Hello, World how are you", got)
}

func TestLongestAgreedPrefix_UnicodePunctuationAndSymbols(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "leading smart quotes and parentheses",
			a:    "“Hello (world)” next",
			b:    "hello world next",
			want: "“Hello (world)” next",
		},
		{
			name: "intra word punctuation",
			a:    "D.C. office",
			b:    "DC office",
			want: "D.C. office",
		},
		{
			name: "hyphen folds only when both sides are same letters",
			a:    "well-known term",
			b:    "wellknown term",
			want: "well-known term",
		},
		{
			name: "real word difference still blocks",
			a:    "well-known term",
			b:    "well worn term",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, longestAgreedPrefix([]string{tc.a, tc.b}, 2, 0))
		})
	}
}
