package sweep

import (
	"sort"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// score.go turns an immutable EvalRun into the numbers the sweep optimizes and
// constrains. The PRIMARY OBJECTIVE is recall over the positive (non-gibberish)
// cases: a per-case 0/1 hit vector whose mean is the arm's recall@k. Negative
// (gibberish) cases are NOT part of the objective — they feed the multi-objective
// CONSTRAINTS (the gibberish ceiling) instead, so an arm can never "win" by
// trading junk-rejection for recall. Cases with no expectation at all are
// ignored (they are informational n/a labels, not a signal).

// isPositiveCase reports whether a suite case counts toward the recall
// objective: it carries a positive expectation (an expected id or a top-K
// requirement) and is not a negative/gibberish case.
func isPositiveCase(c *evalv1.EvalCase) bool {
	if c.GetExpectNoStrongHit() || hasStringTag(c.GetTags(), "gibberish") {
		return false
	}
	return len(c.GetExpectIds()) > 0 || c.GetExpectWithinTopK() > 0
}

// positiveCaseIDs returns the suite's positive case ids in a stable (sorted)
// order so every fold, vector, and bootstrap resample is reproducible.
func positiveCaseIDs(suite *evalv1.EvalSuite) []string {
	ids := make([]string, 0, len(suite.GetCases()))
	for _, c := range suite.GetCases() {
		if isPositiveCase(c) {
			ids = append(ids, c.GetCaseId())
		}
	}
	sort.Strings(ids)
	return ids
}

// generatedCaseIDs returns the set of case ids the corpus marks as machine
// `generated` (Phase 7). The sweep always holds these out of the tuning fold so
// auto-generated cases can never inflate a winner's selection score (overfit
// guard #2). Until Phase 7 marks any case, this is empty and harmless.
func generatedCaseIDs(suite *evalv1.EvalSuite) map[string]bool {
	out := map[string]bool{}
	for _, c := range suite.GetCases() {
		if hasStringTag(c.GetTags(), "generated") {
			out[c.GetCaseId()] = true
		}
	}
	return out
}

// recallByCase builds the per-positive-case 0/1 hit vector for one run: 1 when
// the case's outcome is "met" (the expected id landed within top-K and the
// score band held), 0 otherwise. Keyed by case id so two runs of the same suite
// align case-for-case for the paired bootstrap. Non-positive and absent cases
// are omitted.
func recallByCase(suite *evalv1.EvalSuite, run *evalv1.EvalRun) map[string]float64 {
	positive := map[string]bool{}
	for _, c := range suite.GetCases() {
		if isPositiveCase(c) {
			positive[c.GetCaseId()] = true
		}
	}
	out := make(map[string]float64, len(positive))
	for _, cr := range run.GetResults() {
		if !positive[cr.GetCaseId()] {
			continue
		}
		if cr.GetOutcome() == "met" {
			out[cr.GetCaseId()] = 1
		} else {
			out[cr.GetCaseId()] = 0
		}
	}
	return out
}

// meanOver returns the mean of recall[id] over the given ids (0 when ids is
// empty). It is how a fold's recall (tuning fold, held-out fold, or the whole
// positive set) is read off the per-case map.
func meanOver(recall map[string]float64, ids []string) float64 {
	if len(ids) == 0 {
		return 0
	}
	var sum float64
	for _, id := range ids {
		sum += recall[id]
	}
	return sum / float64(len(ids))
}

// vectorOver returns the recall values for ids in the given order — the aligned
// per-case vector the paired bootstrap resamples.
func vectorOver(recall map[string]float64, ids []string) []float64 {
	out := make([]float64, len(ids))
	for i, id := range ids {
		out[i] = recall[id]
	}
	return out
}

func hasStringTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
