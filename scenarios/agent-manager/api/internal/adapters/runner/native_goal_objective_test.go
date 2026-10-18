package runner

import "testing"

func TestNativeGoalObjectiveMatchesLaunchNormalizationWithoutWhitespaceCollapse(t *testing.T) {
	configured := "  first  line\r\nsecond\rthird\tline  "
	if got, want := NativeGoalObjective(configured), "first  line second third\tline"; got != want {
		t.Fatalf("NativeGoalObjective = %q, want %q", got, want)
	}
	cases := []struct {
		name, observed string
		want           bool
	}{
		{"launch transformed", "first  line second third\tline", true},
		{"raw source preserved", "first  line\r\nsecond\rthird\tline", true},
		{"transformed exact suffix", "first  line second third\tline\nattached prompt", true},
		{"raw exact suffix", "first  line\r\nsecond\rthird\tline\nattached prompt", true},
		{"internal spaces changed", "first line second third\tline", false},
		{"tab changed to space", "first  line second third line", false},
		{"term replaced", "first  line second other\tline", false},
		{"suffix delimiter required", "first  line second third\tlinesuffix", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NativeGoalObjectiveMatches(tc.observed, configured); got != tc.want {
				t.Fatalf("match(%q) = %v, want %v", tc.observed, got, tc.want)
			}
		})
	}
}
