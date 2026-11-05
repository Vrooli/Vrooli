package campaign

import (
	"strings"
	"testing"

	campaignv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/campaign"
)

func mk(code, source, severity, effort string, regressed bool, locs int) Finding {
	loc := make([]string, locs)
	for i := range loc {
		loc[i] = "p"
	}
	return Finding{
		StableID:  "afid:" + code,
		Code:      code,
		Source:    source,
		Severity:  severity,
		Effort:    effort,
		Regressed: regressed,
		Locations: loc,
	}
}

// fixedSet is the shared mixed worklist used to pin every profile's order.
//
//	A: advisory architecture cycle, BLOCKER, large effort
//	B: gating standards error, small effort
//	C: gating docs warning, trivial effort
//	D: gating standards error, REGRESSED
//	E: advisory architecture mislocation, warning, large effort
func fixedSet() []Finding {
	return []Finding{
		mk("cycle/cross", "architecture", "blocker", "large", false, 2),
		mk("lint_error", "standards", "error", "small", false, 1),
		mk("doc_link", "docs", "warn", "trivial", false, 1),
		mk("build_fail", "standards", "error", "medium", true, 1),
		mk("mislocated_pkg", "architecture", "warn", "large", false, 3),
	}
}

func codes(findings []Finding) []string {
	out := make([]string, len(findings))
	for i, f := range findings {
		out[i] = f.Code
	}
	return out
}

func TestOrderProfilesProduceDistinctOrderings(t *testing.T) {
	cases := []struct {
		name    string
		profile campaignv1.RankProfile
		want    []string
	}{
		{
			// FAST: gating sources first (advisory architecture sinks), then
			// severity desc, then cheapest effort, then fewest locations.
			name:    "fast",
			profile: campaignv1.RankProfile_RANK_PROFILE_FAST,
			want:    []string{"lint_error", "build_fail", "doc_link", "cycle/cross", "mislocated_pkg"},
		},
		{
			// BALANCED (legacy): regressions, then cycles, then severity desc.
			name:    "balanced",
			profile: campaignv1.RankProfile_RANK_PROFILE_BALANCED,
			want:    []string{"build_fail", "cycle/cross", "lint_error", "doc_link", "mislocated_pkg"},
		},
		{
			// LONG_TERM: regressions, then structural root causes (arch/struct/
			// cycle/mislocation) first, then severity desc.
			name:    "long_term",
			profile: campaignv1.RankProfile_RANK_PROFILE_LONG_TERM,
			want:    []string{"build_fail", "cycle/cross", "mislocated_pkg", "lint_error", "doc_link"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := codes(Order(fixedSet(), tc.profile))
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("profile %s order = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestOrderUnspecifiedFallsBackToBalanced(t *testing.T) {
	got := codes(Order(fixedSet(), campaignv1.RankProfile_RANK_PROFILE_UNSPECIFIED))
	want := codes(Order(fixedSet(), campaignv1.RankProfile_RANK_PROFILE_BALANCED))
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("unspecified order = %v, want balanced %v", got, want)
	}
}

func TestGatesSuite(t *testing.T) {
	gating := []string{"standards", "structure", "cli", "ui", "docs"}
	for _, s := range gating {
		if !gatesSuite(s) {
			t.Errorf("source %q should gate the suite", s)
		}
	}
	for _, s := range []string{"architecture", "tidiness"} {
		if gatesSuite(s) {
			t.Errorf("source %q is advisory and must not gate", s)
		}
	}
}

func TestStructuralRootCause(t *testing.T) {
	yes := []Finding{
		mk("anything", "architecture", "warn", "", false, 0),
		mk("anything", "structure", "warn", "", false, 0),
		mk("cycle/x", "standards", "warn", "", false, 0),
		mk("mislocated_pkg", "docs", "warn", "", false, 0),
	}
	for _, f := range yes {
		if !structuralRootCause(f) {
			t.Errorf("%+v should be a structural root cause", f)
		}
	}
	no := mk("lint_error", "standards", "error", "", false, 0)
	if structuralRootCause(no) {
		t.Errorf("%+v should NOT be a structural root cause", no)
	}
}
