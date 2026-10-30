package usageattribution

import (
	"fmt"
	"strings"
	"testing"

	"agent-manager/internal/runsignal"
)

// Only a pattern that passes the precision gate, is still detected and is not
// informational becomes a problem; its time kind comes from the pattern table.
func TestFrictionShowsOnlyPatternsThatPassTheGate(t *testing.T) {
	f := newWeekFixture()
	f.episodes = nil
	for _, pattern := range []string{"wait-misuse", "command-failure", "read-then-reread", "handoff-continuation", "time-to-first-success"} {
		for i := 0; i < 10; i++ {
			f.episodes = append(f.episodes, Episode{RunID: "orchestrator", EpisodeID: fmt.Sprintf("%s-%d", pattern, i), Pattern: pattern, Fingerprint: pattern, Turns: 2, Tokens: 1000, WallClockMS: 8_000})
		}
	}
	result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
	report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Prices: fixturePrices})
	want := map[string]bool{}
	for _, pattern := range []string{"wait-misuse", "command-failure", "read-then-reread", "handoff-continuation", "time-to-first-success"} {
		if runsignal.Shown(pattern) {
			want[pattern] = true
		}
	}
	if len(report.Problems) != len(want) || len(want) == 0 {
		t.Fatalf("problems = %+v, want exactly the shown patterns %v", report.Problems, want)
	}
	for _, p := range report.Problems {
		spec, _ := runsignal.PatternByID(p.Pattern)
		if !want[p.Pattern] || p.TimeKind != spec.TimeKind || p.WallClockMS != 80_000 || p.Tokens != 10_000 {
			t.Fatalf("problem = %+v", p)
		}
	}
}

// Descriptions come from the pattern table.
func TestDescribeReadsThePatternTable(t *testing.T) {
	for _, spec := range runsignal.PatternSpecs() {
		if got := Describe(spec.ID, "search-hub query", "", "tool x", 3); strings.Contains(got, "{") || got == "" {
			t.Errorf("%s description = %q", spec.ID, got)
		}
	}
	if got := Describe("wait-misuse", "", "", "", 4); got != "An agent repeats a short wait 4 times instead of one blocking wait." {
		t.Errorf("wait-misuse = %q", got)
	}
	if got := Describe("command-failure", "unknown flag --x", "", "search-hub query", 0); got != "`search-hub query` failed: unknown flag --x" {
		t.Errorf("command-failure = %q", got)
	}
}

// A problem's sentence names the element most of its episodes share, not the
// first episode seen; with no majority it names none.
func TestProblemDescriptionUsesTheDominantElement(t *testing.T) {
	describe := func(elements ...string) string {
		f := newWeekFixture()
		f.episodes = nil
		for i, element := range elements {
			f.episodes = append(f.episodes, Episode{RunID: "orchestrator", EpisodeID: fmt.Sprintf("e%d", i), Pattern: "command-failure", Fingerprint: "cf", RepeatedElement: element, OwnerCommand: "tool x", Turns: 2, Tokens: 10, WallClockMS: 1000})
		}
		result := Attribute(f.window, f.window.Start, f.turns, fixturePrices)
		report := Recommend(Inputs{Results: []Result{result}, Runs: f.runs, Episodes: f.episodes, Prices: fixturePrices})
		if len(report.Problems) != 1 {
			t.Fatalf("problems = %+v", report.Problems)
		}
		return report.Problems[0].Description
	}
	if got := describe("rare", "common", "common", "common", "rare"); !strings.Contains(got, "common") || strings.Contains(got, "rare") {
		t.Errorf("majority description = %q", got)
	}
	if got := describe("a", "b", "c", "d", "e"); strings.Contains(got, ": ") || strings.ContainsAny(got, "abcde") && strings.Contains(got, "failed: ") {
		t.Errorf("no-majority description = %q, want no element", got)
	}
}
