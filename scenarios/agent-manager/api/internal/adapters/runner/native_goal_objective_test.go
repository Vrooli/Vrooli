package runner

import (
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

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
		{"delegation of a short until", NativeGoalDelegation, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NativeGoalObjectiveMatches(domain.RunnerTypeCodex, tc.observed, configured); got != tc.want {
				t.Fatalf("match(%q) = %v, want %v", tc.observed, got, tc.want)
			}
		})
	}
}

// The native goal is the full `until` only while the harness keeps it
// verbatim: through compaction for Codex, within the field limit for Claude.
func TestNativeGoalObjectiveForKeepsOnlyWhatTheHarnessRetains(t *testing.T) {
	for _, tc := range []struct {
		name   string
		runner domain.RunnerType
		until  string
		want   string
	}{
		{"codex kernel", domain.RunnerTypeCodex, strings.Repeat("k", 600), strings.Repeat("k", 600)},
		{"codex objective compaction would omit", domain.RunnerTypeCodex, strings.Repeat("k", 601), NativeGoalDelegation},
		{"codex counts characters, not bytes", domain.RunnerTypeCodex, strings.Repeat("é", 600), strings.Repeat("é", 600)},
		{"claude field limit", domain.RunnerTypeClaudeCode, strings.Repeat("c", 4000), strings.Repeat("c", 4000)},
		{"claude over the field limit", domain.RunnerTypeClaudeCode, strings.Repeat("c", 4001), NativeGoalDelegation},
		{"harness without a measured limit", domain.RunnerTypeOpenCode, "finish", NativeGoalDelegation},
		{"no until", domain.RunnerTypeCodex, "  ", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NativeGoalObjectiveFor(tc.runner, tc.until); got != tc.want {
				t.Fatalf("NativeGoalObjectiveFor(%s, %d chars) = %.40q, want %.40q", tc.runner, len([]rune(tc.until)), got, tc.want)
			}
		})
	}
	if limit := len([]rune(NativeGoalDelegation)); limit > NativeGoalVerbatimLimit(domain.RunnerTypeCodex) {
		t.Fatalf("the delegation itself (%d chars) would not survive Codex compaction", limit)
	}
}

// A delegated goal's markers report the delegation, and they must still
// correlate with the run; sessions launched under the earlier 4,000-character
// limit report the full `until`, which must keep matching too.
func TestNativeGoalObjectiveMatchesTheDelegationOnlyWhenItWasInstalled(t *testing.T) {
	long := strings.Repeat("deliver every outcome ", 40)
	if !NativeGoalObjectiveMatches(domain.RunnerTypeCodex, NativeGoalDelegation, long) {
		t.Fatal("the delegation installed for a long until must match it")
	}
	if !NativeGoalObjectiveMatches(domain.RunnerTypeCodex, NativeGoalObjective(long), long) {
		t.Fatal("a full until installed before the compaction limit must keep matching")
	}
	if NativeGoalObjectiveMatches(domain.RunnerTypeClaudeCode, NativeGoalDelegation, long) {
		t.Fatal("Claude installs this until in full, so the delegation cannot be its goal")
	}
	if NativeGoalObjectiveMatches(domain.RunnerTypeCodex, "some other objective", long) {
		t.Fatal("an unrelated objective must not match")
	}
}
