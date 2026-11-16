package aisearch

import (
	"fmt"
	"sort"
)

// SplitFixedDenominator partitions the digest-validated acceptance set. It
// cannot accidentally shrink the tuning or heldout denominator by using the
// current mutable suite's case list.
func SplitFixedDenominator(denominator FixedDenominator, explicitlyHeldout, excluded map[string]bool) (tuning, heldout, notEvaluated []string, err error) {
	if err := denominator.Validate(); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid fixed denominator: %w", err)
	}
	ids := make([]string, 0, len(denominator.Cases))
	for _, c := range denominator.Cases {
		ids = append(ids, c.ID)
	}
	tuning, heldout, notEvaluated = FixedEvaluationSplit(ids, explicitlyHeldout, excluded)
	return tuning, heldout, notEvaluated, nil
}

// FixedEvaluationSplit partitions a frozen denominator using owner-authored
// membership only. Explicitly held-out IDs are validation cases; generated or
// otherwise excluded IDs are in neither fold; all remaining IDs are tuning
// cases. It intentionally does not invent held-out membership from case IDs.
// Callers must validate labels and freeze the denominator before comparison.
func FixedEvaluationSplit(caseIDs []string, explicitlyHeldout, excluded map[string]bool) (tuning, heldout, notEvaluated []string) {
	seen := make(map[string]struct{}, len(caseIDs))
	for _, id := range caseIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if explicitlyHeldout[id] {
			heldout = append(heldout, id)
		} else if excluded[id] {
			notEvaluated = append(notEvaluated, id)
		} else {
			tuning = append(tuning, id)
		}
	}
	sort.Strings(tuning)
	sort.Strings(heldout)
	sort.Strings(notEvaluated)
	return tuning, heldout, notEvaluated
}
