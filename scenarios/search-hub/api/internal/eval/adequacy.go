package eval

import (
	"fmt"
	"sort"
	"strings"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// adequacy.go is the GRADING side of the corpus: warn-level findings on whether
// a golden corpus is large enough, has negatives, lacks duplicates, spreads
// across difficulty, and covers the live index. It is deliberately separate from
// validate.go: Validate is the HARD GATE (a malformed suite is rejected and
// never persisted); CheckAdequacy NEVER fails — every finding is informational.
// Optimizing against a thin corpus is the central overfit risk this whole plan
// guards against, so the harness surfaces "this corpus is thin" loudly without
// ever blocking the operator who chooses to run it anyway.

// MinPositiveCases is the floor below which a corpus is flagged "too few cases"
// to tune against safely. At the sweep's default held-out fraction (1/3) a
// 12-case corpus still leaves a non-trivial validation fold; below that the
// held-out split is too small to detect overfitting. Advisory, not enforced.
const MinPositiveCases = 12

// difficultyTags are the labels that mark a case's difficulty band. A corpus
// that is all one band (e.g. every case "strong") over-reports recall because it
// never exercises the hard retrievals; thin_difficulty flags that.
var difficultyTags = []string{"strong", "weak", "weak-real", "hard"}

// CheckAdequacy returns warn-level findings about a suite's corpus. It never
// returns an error and never fails — each finding tells an operator the corpus
// the eval numbers rest on is thin, so they grow it (e.g. via `evals generate`).
//
// strata is the set of live index strata observed for the provider (each leaf's
// type/group/origin buckets). When empty the coverage-gap check is skipped — a
// purely structural caller (GetSuite, which has no live sample) passes nil and
// gets the count/negatives/duplicate/difficulty findings; a caller that sampled
// the index (corpusgen) passes the sampled strata to also get coverage gaps.
//
// Findings are returned in a stable order (whole-corpus findings first, then
// per-subject findings sorted by subject) so output and tests are deterministic.
func CheckAdequacy(s *evalv1.EvalSuite, strata []string) []*evalv1.AdequacyWarning {
	if s == nil {
		return nil
	}
	var out []*evalv1.AdequacyWarning
	warn := func(code, msg, subject string) {
		out = append(out, &evalv1.AdequacyWarning{Code: code, Message: msg, Subject: subject})
	}

	positives := 0
	negatives := 0
	difficultyClasses := map[string]bool{}
	for _, c := range s.GetCases() {
		switch {
		case isNegativeCase(c):
			negatives++
		case isPositiveCase(c):
			positives++
			for _, d := range difficultyTags {
				if hasTag(c.GetTags(), d) {
					difficultyClasses[d] = true
				}
			}
		}
	}

	// 1. Too few positive cases — the corpus is trivially overfittable.
	if positives < MinPositiveCases {
		warn("too_few_cases",
			fmt.Sprintf("only %d positive case(s) — below the floor of %d; optimizing against this few overfits", positives, MinPositiveCases),
			"")
	}

	// 2. No negatives — nothing constrains junk leakage, so a tuning can win by
	//    accepting garbage. The sweep's gibberish ceiling has no teeth here.
	if negatives == 0 {
		warn("no_negatives",
			"no negative (expect_no_strong_hit) cases — junk-rejection is unmeasured, so the sweep's gibberish ceiling cannot constrain a winner",
			"")
	}

	// 3. Thin difficulty spread — every positive case is the same (or no) band.
	if positives >= MinPositiveCases && len(difficultyClasses) <= 1 {
		band := "untagged"
		if len(difficultyClasses) == 1 {
			band = sortedKeys(difficultyClasses)[0]
		}
		warn("thin_difficulty",
			fmt.Sprintf("all positive cases share one difficulty band (%s) — recall is over-reported without hard cases", band),
			"")
	}

	// 4. Duplicate queries — two cases asking the same thing inflate any count.
	for _, q := range duplicateQueries(s.GetCases()) {
		warn("duplicate_query", fmt.Sprintf("query %q appears on more than one case", q), q)
	}

	// 5. Coverage gaps vs the live index (only when a sample was supplied).
	for _, st := range uncoveredStrata(s.GetCases(), strata) {
		warn("coverage_gap", fmt.Sprintf("no case covers index stratum %q", st), st)
	}

	return out
}

// isPositiveCase reports whether a case counts toward recall: it carries a
// positive expectation and is not a negative case. Mirrors the sweep's objective
// predicate so adequacy and scoring agree on what a "positive case" is.
func isPositiveCase(c *evalv1.EvalCase) bool {
	if isNegativeCase(c) {
		return false
	}
	return len(c.GetExpectIds()) > 0 || c.GetExpectWithinTopK() > 0
}

// isNegativeCase reports whether a case is a junk/no-answer case (the
// constraint-side cases, never the objective).
func isNegativeCase(c *evalv1.EvalCase) bool {
	return c.GetExpectNoStrongHit() || hasTag(c.GetTags(), "gibberish")
}

// duplicateQueries returns, sorted, each normalized query that appears on more
// than one case (case-insensitive, whitespace-collapsed).
func duplicateQueries(cases []*evalv1.EvalCase) []string {
	seen := map[string]int{}
	for _, c := range cases {
		seen[normalizeQuery(c.GetQuery())]++
	}
	var dups []string
	for q, n := range seen {
		if n > 1 && q != "" {
			dups = append(dups, q)
		}
	}
	sort.Strings(dups)
	return dups
}

// uncoveredStrata returns, sorted, the strata in `strata` that no case carries
// as a tag. Empty when strata is empty (the structural-only path).
func uncoveredStrata(cases []*evalv1.EvalCase, strata []string) []string {
	if len(strata) == 0 {
		return nil
	}
	covered := map[string]bool{}
	for _, c := range cases {
		for _, t := range c.GetTags() {
			covered[t] = true
		}
	}
	var gaps []string
	for _, st := range strata {
		st = strings.TrimSpace(st)
		if st != "" && !covered[st] {
			gaps = append(gaps, st)
		}
	}
	sort.Strings(gaps)
	return uniqueSorted(gaps)
}

func normalizeQuery(q string) string {
	return strings.ToLower(strings.Join(strings.Fields(q), " "))
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:0:0]
	var prev string
	for i, s := range in {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}
