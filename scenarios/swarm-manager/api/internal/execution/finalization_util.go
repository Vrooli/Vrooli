package execution

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"swarm-manager/internal/pathutil"
)

func summarizeFinalization(finalization Finalization) (classification string, summary string, hasActionableFailure bool) {
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

func effectiveFinalization(record Record) *Finalization {
	if record.Finalization != nil {
		return record.Finalization
	}
	if record.LegacyReviewResult == nil && record.LegacyReviewJobID == "" && record.LegacyReviewSkipReason == "" && record.LegacyReviewStartedAt == "" {
		return nil
	}
	finalization := &Finalization{
		Eligible:          true,
		Status:            FinalizationStatusCompleted,
		Phase:             FinalizationPhaseCompleted,
		ScopeSource:       FinalizationScopeAcceptanceAllow,
		StartedAt:         record.LegacyReviewStartedAt,
		CompletedAt:       record.FinishedAt,
		AffectedScenarios: []string{},
		Warnings:          []FinalizationWarning{},
		Scenarios:         []ScenarioFinalization{},
	}
	if record.Status == StatusValidating {
		finalization.Status = FinalizationStatusRunning
		finalization.Phase = FinalizationPhaseReviewing
	}
	if record.LegacyReviewSkipReason != "" {
		finalization.Status = FinalizationStatusSkipped
		finalization.Phase = FinalizationPhaseSkipped
		finalization.SkipReason = record.LegacyReviewSkipReason
		finalization.AggregateClassification = FinalizationAggregateSkipped
		finalization.AggregateSummary = record.LegacyReviewSkipReason
		return finalization
	}
	if record.LegacyReviewResult != nil {
		finalization.AggregateClassification = record.LegacyReviewResult.Classification
		finalization.AggregateSummary = record.LegacyReviewResult.Summary
		finalization.Scenarios = []ScenarioFinalization{{
			ScenarioName: record.BacklogName,
			Restart:      RestartResult{Status: FinalizationStatusCompleted},
			Health:       HealthCheckResult{Status: FinalizationStatusCompleted, SchemaValid: true},
			Review: ScenarioReviewStep{
				Status: FinalizationStatusCompleted,
				JobID:  record.LegacyReviewResult.JobID,
				Result: record.LegacyReviewResult,
			},
		}}
		return finalization
	}
	finalization.AggregateClassification = FinalizationAggregateNotAssessable
	return finalization
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

func migrateLegacyFinalizationState(record *Record, item backlogItem) bool {
	if record == nil || record.Finalization != nil {
		return false
	}
	finalization := effectiveFinalization(*record)
	if finalization == nil {
		return false
	}
	if len(finalization.Scenarios) == 1 && finalization.Scenarios[0].ScenarioName == record.BacklogName {
		scenarios := pathutil.ScenariosFromGlobs(item.AcceptanceAllow)
		if len(scenarios) > 0 {
			finalization.Scenarios[0].ScenarioName = scenarios[0]
			finalization.AffectedScenarios = []string{scenarios[0]}
		}
	}
	record.Finalization = finalization
	record.LegacyReviewResult = nil
	record.LegacyReviewJobID = ""
	record.LegacyReviewSkipReason = ""
	record.LegacyReviewStartedAt = ""
	return true
}

func logFinalizationError(executionID string, err error) {
	if err == nil {
		return
	}
	slog.Error("finalization error", "execution_id", executionID, "err", err)
}
