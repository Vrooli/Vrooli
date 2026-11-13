package sweep

import (
	"sort"

	aisearch "github.com/vrooli/ai-go/search"
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
	if hasStringTag(c.GetTags(), aisearch.TagGenerated) && !hasStringTag(c.GetTags(), aisearch.TagHeldout) {
		return false
	}
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
		if c.GetStatus() != "candidate" && isPositiveCase(c) {
			ids = append(ids, c.GetCaseId())
		}
	}
	sort.Strings(ids)
	return ids
}

// generatedCaseIDs returns machine-generated case IDs. They are excluded from
// both folds until the dataset owner reviews the labels and explicitly tags
// them as heldout.
func generatedCaseIDs(suite *evalv1.EvalSuite) map[string]bool {
	out := map[string]bool{}
	for _, c := range suite.GetCases() {
		if hasStringTag(c.GetTags(), "generated") {
			out[c.GetCaseId()] = true
		}
	}
	return out
}

func ownerHeldoutCaseIDs(suite *evalv1.EvalSuite) map[string]bool {
	out := map[string]bool{}
	for _, c := range suite.GetCases() {
		if hasStringTag(c.GetTags(), aisearch.TagHeldout) {
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
		if c.GetStatus() != "candidate" && isPositiveCase(c) {
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

func completePositiveRun(suite *evalv1.EvalSuite, run *evalv1.EvalRun) string {
	if run == nil {
		return "run is missing"
	}
	if run.GetDegraded() {
		return "run is degraded: " + run.GetDegradedReason()
	}
	want := positiveCaseIDs(suite)
	seen := make(map[string]bool, len(want))
	for _, result := range run.GetResults() {
		if result == nil || !containsCaseID(want, result.GetCaseId()) {
			continue
		}
		if seen[result.GetCaseId()] {
			return "duplicate result for fixed-denominator case " + result.GetCaseId()
		}
		seen[result.GetCaseId()] = true
		switch result.GetOutcome() {
		case "met", "below_expectation", "above_expectation", "unexpected_hit", "answered_by_sibling", "misrouted", "no_result", "thin_margin":
		default:
			return "fixed-denominator case " + result.GetCaseId() + " has non-gradeable outcome " + result.GetOutcome()
		}
	}
	for _, id := range want {
		if !seen[id] {
			return "missing fixed-denominator case " + id
		}
	}
	return ""
}

func containsCaseID(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
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
