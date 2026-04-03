package execution

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) processFinalization(ctx context.Context, executionID string) error {
	if !s.beginFinalization(executionID) {
		return nil
	}
	defer s.endFinalization(executionID)

	record, item, err := s.loadExecutionForFinalization(executionID)
	if err != nil {
		return s.failFinalization(executionID, "", fmt.Sprintf("load execution for finalization: %v", err))
	}

	if !isFinalizationEligible(record) {
		return s.completeFinalizationSkipped(executionID, "execution type does not use post-run checks")
	}

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseScopeDetection); err != nil {
		return err
	}

	scope, err := s.resolveFinalizationScope(ctx, record, item)
	if err != nil {
		return s.failFinalization(executionID, "", fmt.Sprintf("resolve affected scenarios: %v", err))
	}
	if len(scope.affectedScenarios) == 0 {
		skipReason := "no affected scenarios could be resolved from sandbox diff or acceptance_allow"
		if err := s.applyFinalizationScope(executionID, scope, skipReason); err != nil {
			return err
		}
		return s.completeFinalizationSkipped(executionID, skipReason)
	}
	if err := s.applyFinalizationScope(executionID, scope, ""); err != nil {
		return err
	}

	for _, scenarioName := range scope.affectedScenarios {
		if err := s.runScenarioRestartAndHealth(ctx, executionID, scenarioName); err != nil {
			return err
		}
	}

	if err := s.markFinalizationPhase(executionID, FinalizationPhaseReviewing); err != nil {
		return err
	}
	for _, scenarioName := range scope.affectedScenarios {
		if err := s.runScenarioReview(ctx, executionID, scenarioName, scope.sandboxID, item.AcceptanceAllow); err != nil {
			return err
		}
	}

	// Evidence gathering phase (optional, policy-gated). The review agent
	// spawns asynchronously; its failure is non-fatal to finalization.
	if s.isReviewAgentEnabled() {
		if err := s.markFinalizationPhase(executionID, FinalizationPhaseEvidenceGathering); err != nil {
			return err
		}
		if err := s.triggerReviewAgent(ctx, executionID, scope, item); err != nil {
			_ = s.appendFinalizationWarning(executionID, newFinalizationWarning(
				finalizationWarningReviewAgentFailed, "", err.Error(), false,
			))
		}
	}

	return s.finishFinalization(executionID)
}

func (s *Service) runScenarioReview(ctx context.Context, executionID, scenarioName, sandboxID string, acceptanceAllow []string) error {
	scenarioState, err := s.loadScenarioFinalization(executionID, scenarioName)
	if err != nil {
		return err
	}
	if scenarioState.Health.Status != FinalizationStatusCompleted {
		warning := newFinalizationWarning(
			finalizationWarningReviewSkipped,
			scenarioName,
			"review skipped because restart/health checks did not pass",
			false,
		)
		if err := s.appendFinalizationWarning(executionID, warning); err != nil {
			return err
		}
		return s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
			Status:     FinalizationStatusSkipped,
			SkipReason: "restart/health checks did not pass",
		})
	}

	if s.reviewClient == nil {
		return s.failFinalization(executionID, scenarioName, "review client is not configured")
	}

	if err := s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
		Status: FinalizationStatusRunning,
	}); err != nil {
		return err
	}

	req := ReviewRequest{
		ScenarioName: scenarioName,
		SandboxID:    sandboxID,
	}
	if len(scenarioState.ChangedPaths) > 0 {
		req.ExpectedPaths = append([]string(nil), scenarioState.ChangedPaths...)
	} else {
		req.ExpectedPaths = append([]string(nil), acceptanceAllow...)
	}
	if s.reviewThresholdsProvider != nil {
		if th, thErr := s.reviewThresholdsProvider.LoadReviewThresholds(); thErr == nil {
			req.Thresholds = th
		}
	}

	jobID, err := s.reviewClient.TriggerReview(ctx, req)
	if err != nil {
		return s.failFinalization(executionID, scenarioName, fmt.Sprintf("trigger review for %s: %v", scenarioName, err))
	}
	if err := s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
		Status: FinalizationStatusRunning,
		JobID:  jobID,
	}); err != nil {
		return err
	}

	deadline := time.Now().Add(s.finalizationCfg.ReviewPollTimeout)
	for {
		result, done, pollErr := s.reviewClient.PollReview(ctx, jobID)
		if pollErr != nil {
			return s.failFinalization(executionID, scenarioName, fmt.Sprintf("poll review for %s: %v", scenarioName, pollErr))
		}
		if done {
			return s.updateScenarioReviewState(executionID, scenarioName, ScenarioReviewStep{
				Status: FinalizationStatusCompleted,
				JobID:  jobID,
				Result: result,
			})
		}
		if time.Now().After(deadline) {
			return s.failFinalization(executionID, scenarioName, fmt.Sprintf("review for %s timed out after %s", scenarioName, s.finalizationCfg.ReviewPollTimeout))
		}
		select {
		case <-ctx.Done():
			return s.failFinalization(executionID, scenarioName, ctx.Err().Error())
		case <-time.After(s.finalizationCfg.ReviewPollInterval):
		}
	}
}

func (s *Service) finishFinalization(executionID string) error {
	s.mu.Lock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)

	finalization.Status = FinalizationStatusCompleted
	finalization.Phase = FinalizationPhaseCompleted
	finalization.CompletedAt = nowRFC3339()
	classification, summary, hasActionableFailure := summarizeFinalization(*finalization)
	finalization.AggregateClassification = classification
	finalization.AggregateSummary = summary
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	record.FinishedAt = nowRFC3339()

	item, loadErr := s.loadBacklogItemByRecord(record)
	autoSpawnFixup := false
	if loadErr == nil {
		switch {
		case hasActionableFailure:
			policy, _ := s.policyProvider.LoadPolicy()
			if policy.AutoFixup && record.FixupAttempt < policy.MaxFixupAttempts {
				record.Status = StatusNeedsFixup
				record.FailureReason = ""
				autoSpawnFixup = true
			} else {
				record.Status = StatusNeedsFixup
				record.FailureReason = ""
			}
			_ = s.updateBacklogStatus(item, backlogStatusFailed)
		default:
			record.Status = StatusCompleted
			record.FailureReason = ""
			_ = s.updateBacklogStatus(item, backlogStatusCompleted)
		}
	} else {
		record.Status = StatusNeedsFixup
		record.FailureReason = fmt.Sprintf("failed to load backlog item while finishing finalization: %v", loadErr)
	}

	if err := s.store.Save(records); err != nil {
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	s.dispatchStatusUpdate(*record)
	if autoSpawnFixup {
		s.spawnFixupRun(context.Background(), record, item)
	}
	return nil
}

// isReviewAgentEnabled checks the execution policy for the review_agent_enabled flag.
func (s *Service) isReviewAgentEnabled() bool {
	policy, err := s.policyProvider.LoadPolicy()
	if err != nil {
		return false
	}
	return policy.ReviewAgentEnabled
}

// triggerReviewAgent spawns the review agent to gather evidence after finalization.
// The agent runs asynchronously; this method returns after the spawn request is sent.
func (s *Service) triggerReviewAgent(ctx context.Context, executionID string, scope finalizationScope, item backlogItem) error {
	if s.reviewService == nil {
		return fmt.Errorf("review service not configured")
	}
	return s.reviewService.StartReviewForExecution(ctx,
		executionID, item.Kind, item.Name, item.Title,
		s.itemDir(item.Kind, item.Name),
		scope.affectedScenarios, scope.changedPathsByScenario,
	)
}
