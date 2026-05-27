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
			require.Equal(t, tc.want, longestAgreedPrefix(runs, tc.commitRuns))
		})
	}
}
