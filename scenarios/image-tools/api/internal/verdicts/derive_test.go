package verdicts

import (
	"strings"
	"testing"
)

func claim(grade Grade, date string) Claim {
	return Claim{ModelID: "m", Operation: "op", Grade: grade, Reason: "claim " + string(grade), Kind: KindQualification, Ref: "r", Date: date}
}

// TestDeriveRules states the verdict rule table: how curated claims, recent
// job history and operator ratings combine.
func TestDeriveRules(t *testing.T) {
	jobs := func(ok, bad int) Signals { return Signals{Jobs: JobStats{Succeeded: ok, Failed: bad}} }
	rated := func(n int, mean float64) Signals { return Signals{Ratings: RatingStats{Count: n, Mean: mean}} }
	both := func(a, b Signals) Signals { return Signals{Jobs: a.Jobs, Ratings: b.Ratings} }

	cases := []struct {
		name   string
		claims []Claim
		sig    Signals
		want   Grade
		reason string // substring
	}{
		{"nothing is untested", nil, Signals{}, Untested, "no qualification"},
		{"claim sets the grade", []Claim{claim(Recommended, "2026-11-21")}, Signals{}, Recommended, "claim recommended"},
		{"newest claim wins", []Claim{claim(Recommended, "2026-11-01"), claim(NotRecommended, "2026-11-21")}, Signals{}, NotRecommended, "claim not_recommended"},
		{"later entry wins a date tie", []Claim{claim(Works, "2026-11-21"), claim(Broken, "2026-11-21")}, Signals{}, Broken, "claim broken"},

		{"one success is experimental", nil, jobs(1, 0), Experimental, "not enough history"},
		{"steady success is works", nil, jobs(8, 1), Works, "8 of 9"},
		{"mixed history stays experimental", nil, jobs(4, 3), Experimental, "4 of 7"},
		{"repeated failure with no success is broken", nil, jobs(0, 3), Broken, "none succeeded"},
		{"two failures are not yet broken", nil, jobs(0, 2), Untested, "no qualification"},

		{"stale broken claim lifted by live success", []Claim{claim(Broken, "2026-11-01")}, jobs(9, 1), Experimental, "re-qualify"},
		{"broken claim ignores a lone success", []Claim{claim(Broken, "2026-11-01")}, jobs(1, 0), Broken, "claim broken"},
		{"recommended demoted by sustained failure", []Claim{claim(Recommended, "2026-11-21")}, jobs(2, 5), Experimental, "only 2 of 7"},
		{"recommended survives a short bad run", []Claim{claim(Recommended, "2026-11-21")}, jobs(1, 3), Recommended, "claim recommended"},
		{"experimental claim is not touched by failures", []Claim{claim(Experimental, "2026-11-21")}, jobs(0, 9), Experimental, "claim experimental"},

		{"high ratings recommend", nil, rated(3, 4.3), Recommended, "4.3 of 5 over 3"},
		{"middling ratings make an untested model work", nil, rated(4, 3.0), Works, "3.0 of 5"},
		{"low ratings withdraw a recommendation", []Claim{claim(Recommended, "2026-11-21")}, rated(3, 1.7), NotRecommended, "1.7 of 5"},
		{"two ratings are only evidence", []Claim{claim(Works, "2026-11-21")}, rated(2, 1.0), Works, "claim works"},
		{"middling ratings leave a claim alone", []Claim{claim(Works, "2026-11-21")}, rated(5, 3.2), Works, "claim works"},
		{"ratings never lift broken", []Claim{claim(Broken, "2026-11-21")}, rated(9, 5.0), Broken, "claim broken"},
		{"failure demotion is not overruled by ratings", []Claim{claim(Works, "2026-11-21")}, both(jobs(1, 6), rated(3, 4.5)), Experimental, "only 1 of 7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Derive(tc.claims, tc.sig, DefaultPolicy)
			if got.Grade != tc.want {
				t.Fatalf("grade = %s (%s), want %s", got.Grade, got.Reason, tc.want)
			}
			if !strings.Contains(got.Reason, tc.reason) {
				t.Fatalf("reason %q does not contain %q", got.Reason, tc.reason)
			}
		})
	}
}

// TestDeriveEvidenceNamesEverySource proves a verdict carries a pointer for the
// claim, the job history and the ratings that shaped it.
func TestDeriveEvidenceNamesEverySource(t *testing.T) {
	got := Derive([]Claim{claim(Works, "2026-11-21")}, Signals{Jobs: JobStats{Succeeded: 5}, Ratings: RatingStats{Count: 3, Mean: 3.0}}, DefaultPolicy)
	kinds := map[string]bool{}
	for _, e := range got.Evidence {
		if e.Ref == "" || e.Summary == "" {
			t.Fatalf("evidence %+v has no ref or summary", e)
		}
		kinds[e.Kind] = true
	}
	for _, kind := range []string{KindQualification, KindJobHistory, KindOperatorRating} {
		if !kinds[kind] {
			t.Fatalf("evidence kinds %v are missing %s", kinds, kind)
		}
	}
}

func TestRollupKeepsTheBestGradeAndNamesTheRest(t *testing.T) {
	got := Rollup([]Verdict{
		{Operation: "edit_instruct", Grade: Recommended, Reason: "kept identity"},
		{Operation: "upscale", Grade: NotRecommended, Reason: "soft"},
		{Operation: "caption", Grade: Untested, Reason: "none"},
	})
	if got.Grade != Recommended {
		t.Fatalf("roll-up grade = %s", got.Grade)
	}
	if !strings.Contains(got.Reason, "edit_instruct: kept identity") || !strings.Contains(got.Reason, "upscale not recommended") {
		t.Fatalf("roll-up reason = %q", got.Reason)
	}
	if strings.Contains(got.Reason, "caption") {
		t.Fatalf("an untested operation must not appear in the roll-up reason: %q", got.Reason)
	}
	if got := Rollup([]Verdict{{Operation: "x", Grade: Untested}}); got.Grade != Untested {
		t.Fatalf("all-untested roll-up = %s", got.Grade)
	}
}

func TestApplyHold(t *testing.T) {
	hold := Hold{Reason: "needs a manual download", Ref: "seed"}
	t.Run("no evidence reads as its hold", func(t *testing.T) {
		got := ApplyHold(Derive(nil, Signals{}, DefaultPolicy), hold)
		if got.Grade != Untested || got.Reason != "disabled: needs a manual download" || len(got.Evidence) != 1 || got.Evidence[0].Kind != KindCatalogHold {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("untested job history still reads as its hold", func(t *testing.T) {
		got := ApplyHold(Derive(nil, Signals{Jobs: JobStats{Failed: 1}}, DefaultPolicy), hold)
		if got.Grade != Untested || got.Reason != "disabled: needs a manual download" {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("a runtime hold is broken", func(t *testing.T) {
		got := ApplyHold(Derive(nil, Signals{}, DefaultPolicy), Hold{Reason: "cannot load", Breaks: true})
		if got.Grade != Broken {
			t.Fatalf("grade = %s", got.Grade)
		}
	})
	t.Run("evidence keeps its grade and reason", func(t *testing.T) {
		claim := Claim{ModelID: "m", Operation: "op", Grade: Works, Reason: "judged fine", Kind: KindQualification, Ref: "r", Date: "2026-11-22"}
		got := ApplyHold(Derive([]Claim{claim}, Signals{}, DefaultPolicy), hold)
		if got.Grade != Works || got.Reason != "judged fine" || len(got.Evidence) != 2 {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("no reason changes nothing", func(t *testing.T) {
		base := Derive(nil, Signals{}, DefaultPolicy)
		if got := ApplyHold(base, Hold{}); got.Reason != base.Reason || len(got.Evidence) != 0 {
			t.Fatalf("got %+v", got)
		}
	})
}
