package execution

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// runBaselineDiffs compares each affected scenario against the baseline captured
// before execution (Record.PreExecBaselines) and records a structured
// new-vs-pre-existing delta on the scenario's finalization state.
//
// It is additive to the absolute GCT review that follows: the absolute review
// still runs and still produces gct-review-results; the baseline diff augments
// it with regression attribution. The phase is best-effort — a diff failure or
// a missing baseline degrades that scenario to not_comparable with a warning,
// and the feature being disabled (or no client) skips the phase entirely. It
// never fails finalization except on genuine state-write infrastructure errors.
func (s *Service) runBaselineDiffs(ctx context.Context, executionID string, scope finalizationScope, preExecBaselines map[string]string) error {
	if !s.finalizationCfg.BaselineDiffEnabled || s.baselineClient == nil {
		return nil
	}
	if len(scope.affectedScenarios) == 0 {
		return nil
	}

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseBaselineDiff); err != nil {
		return err
	}

	for _, scenarioName := range scope.affectedScenarios {
		baselineName := preExecBaselines[scenarioName]
		if baselineName == "" {
			// Affected but no pre-exec baseline: the sandbox diff expanded scope
			// beyond the declared acceptance_allow, so there is nothing to
			// compare against. Surface as not_comparable; the absolute review
			// still covers the scenario.
			if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningBaselineScopeExpanded,
				scenarioName,
				fmt.Sprintf("No pre-execution baseline for %q (touched outside acceptance_allow); regressions cannot be separated from pre-existing failures.", scenarioName),
				false,
			)); err != nil {
				return err
			}
			if err := s.updateScenarioBaselineDiff(executionID, scenarioName, notComparableDiff(scenarioName)); err != nil {
				return err
			}
			continue
		}

		diffCtx, cancel := context.WithTimeout(ctx, s.finalizationCfg.BaselineDiffTimeout)
		result, err := s.baselineClient.Diff(diffCtx, scenarioName, baselineName)
		cancel()
		if err != nil {
			slog.Warn("baseline diff failed; marking scenario not_comparable",
				"execution_id", executionID, "scenario", scenarioName, "baseline", baselineName, "err", err)
			if warnErr := s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningBaselineDiffFailed,
				scenarioName,
				fmt.Sprintf("baseline diff for %q failed: %v", scenarioName, err),
				false,
			)); warnErr != nil {
				return warnErr
			}
			if updErr := s.updateScenarioBaselineDiff(executionID, scenarioName, notComparableDiff(scenarioName)); updErr != nil {
				return updErr
			}
			continue
		}

		diff := result
		if err := s.updateScenarioBaselineDiff(executionID, scenarioName, &diff); err != nil {
			return err
		}
		// A genuine regression (a surface that passed in the pre-exec baseline
		// and fails now) is attributable to this change, so surface it as a
		// first-class, non-retryable warning. summarizeFinalization reads the
		// same recorded verdict to gate the finalization outcome when
		// BaselineRegressionGateEnabled is set (plan P6 §200-201) — the warning
		// is the audit/UI signal, the gate is the decision.
		if diff.HasNewRegressions() {
			if err := s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningBaselineRegression,
				scenarioName,
				fmt.Sprintf("%q introduced %d regression(s) on %s; this change turned previously-passing surface(s) red.",
					scenarioName, len(diff.Regressions), strings.Join(diff.RegressedSurfaces, ", ")),
				false,
			)); err != nil {
				return err
			}
		}
		slog.Info("baseline diff complete",
			"execution_id", executionID,
			"scenario", scenarioName,
			"verdict", diff.Verdict,
			"regressions", len(diff.Regressions),
			"preexisting", len(diff.PreExisting),
		)
	}

	return nil
}

// cleanupPreExecBaselines deletes the pre-execution baselines captured for this
// execution once finalization has consumed their diffs. The review agent reads
// the diff *results* (persisted on the record), not the live baselines, so they
// are safe to remove here. Best-effort: failures are logged, never fatal.
// Skipped when BaselineRetainAfterFinalization is set (debugging/audit).
//
// Baselines are content-addressed by working-tree state, so a concurrent
// execution may share a name; deleting it degrades that peer's diff to
// not_comparable (handled gracefully) rather than corrupting anything.
func (s *Service) cleanupPreExecBaselines(ctx context.Context, preExecBaselines map[string]string) {
	if s.baselineClient == nil || len(preExecBaselines) == 0 {
		return
	}
	if s.finalizationCfg.BaselineRetainAfterFinalization {
		return
	}
	for scenario, name := range preExecBaselines {
		if name == "" {
			continue
		}
		if err := s.baselineClient.Delete(ctx, scenario, name); err != nil {
			slog.Warn("baseline cleanup: delete failed",
				"scenario", scenario, "name", name, "err", err)
		}
	}
}

// notComparableDiff builds the marker result used when a scenario cannot be
// compared (no baseline, or the diff call failed).
func notComparableDiff(scenario string) *BaselineDiffResult {
	return &BaselineDiffResult{
		ScenarioName: scenario,
		Verdict:      baselineVerdictNotComparable,
		ExitCode:     2,
		Comparable:   false,
	}
}
