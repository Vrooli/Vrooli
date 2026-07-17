package execution

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"swarm-manager/internal/pathutil"
)

// summarizeFinalization folds the per-scenario finalization steps into an
// aggregate classification, a human summary, and the hasActionableFailure flag
// that routes the backlog item to needs-fixup/hand-back (vs accepted).
//
// gateRegressions enables the Baseline Modes promote gate (plan P6 §200-201):
// when set, a scenario whose recorded before/after diff verdict is a genuine
// regression counts as an actionable failure even if every other step passed,
// so a change that turns a previously-passing surface red is handed back
// instead of silently accepted. When false the verdict is still recorded and
// warned (see runBaselineDiffs) but does not gate the outcome.
func summarizeFinalization(finalization Finalization, gateRegressions bool) (classification string, summary string, hasActionableFailure bool) {
	if finalization.Status == FinalizationStatusSkipped {
		return FinalizationAggregateSkipped, strings.TrimSpace(finalization.SkipReason), false
	}

	hasReadyWithNotes := false
	summaries := make([]string, 0, len(finalization.Scenarios))
	for _, scenario := range finalization.Scenarios {
		if scenario.Restart.Status != "" && scenario.Restart.Status != FinalizationStatusCompleted {
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s restart failed", scenario.ScenarioName))
			continue
		}
		if scenario.Health.Status != "" && scenario.Health.Status != FinalizationStatusCompleted {
			hasActionableFailure = true
			if strings.TrimSpace(scenario.Health.Details) != "" {
				summaries = append(summaries, fmt.Sprintf("%s health failed: %s", scenario.ScenarioName, scenario.Health.Details))
			} else {
				summaries = append(summaries, fmt.Sprintf("%s health failed", scenario.ScenarioName))
			}
			continue
		}
		switch {
		case scenario.Review.Status == FinalizationStatusSkipped:
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s review skipped: %s", scenario.ScenarioName, scenario.Review.SkipReason))
		case scenario.Review.Result == nil:
			hasActionableFailure = true
			summaries = append(summaries, fmt.Sprintf("%s review unavailable", scenario.ScenarioName))
		default:
			switch scenario.Review.Result.Classification {
			case FinalizationAggregateReady:
				summaries = append(summaries, fmt.Sprintf("%s ready", scenario.ScenarioName))
			case FinalizationAggregateReadyWithNotes:
				hasReadyWithNotes = true
				summaries = append(summaries, fmt.Sprintf("%s ready with notes", scenario.ScenarioName))
			default:
				hasActionableFailure = true
				summaries = append(summaries, fmt.Sprintf("%s needs follow-up: %s", scenario.ScenarioName, scenario.Review.Result.Summary))
			}
		}

		// Baseline regression gate (plan P6 §200-201): a change that turned a
		// previously-passing surface red is a regression attributable to this
		// item, so hand it back even when the absolute review came back ready.
		// Only the genuine "regression" verdict gates — new-failure /
		// pre-existing / not-comparable are not this change's fault.
		if gateRegressions && scenario.BaselineDiff != nil && scenario.BaselineDiff.HasNewRegressions() {
			hasActionableFailure = true
			surfaces := strings.Join(scenario.BaselineDiff.RegressedSurfaces, ", ")
			if surfaces == "" {
				surfaces = "tests"
			}
			summaries = append(summaries, fmt.Sprintf("%s introduced %d regression(s) [%s]",
				scenario.ScenarioName, len(scenario.BaselineDiff.Regressions), surfaces))
		}
	}

	classification = FinalizationAggregateReady
	switch {
	case hasActionableFailure:
		classification = FinalizationAggregateNeedsWork
	case hasReadyWithNotes:
		classification = FinalizationAggregateReadyWithNotes
	}

	if len(finalization.Warnings) > 0 {
		summaries = append(summaries, fmt.Sprintf("%d warning(s)", len(finalization.Warnings)))
	}
	summary = strings.Join(summaries, "; ")
	return classification, summary, hasActionableFailure
}

func newFinalizationWarning(code, scenarioName, message string, retryable bool) FinalizationWarning {
	return FinalizationWarning{
		Code:         strings.TrimSpace(code),
		ScenarioName: strings.TrimSpace(scenarioName),
		Message:      strings.TrimSpace(message),
		Retryable:    retryable,
		CreatedAt:    nowRFC3339(),
	}
}

func unionSortedStrings(base []string, extras []string) []string {
	return pathutil.UniqueSortedStrings(append(append([]string{}, base...), extras...))
}

func mapKeysSorted(values map[string][]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func logFinalizationError(executionID string, err error) {
	if err == nil {
		return
	}
	slog.Error("finalization error", "execution_id", executionID, "err", err)
}
