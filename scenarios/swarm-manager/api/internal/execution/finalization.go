package execution

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/pathutil"
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

func (s *Service) completeFinalizationSkipped(executionID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusSkipped
	finalization.Phase = FinalizationPhaseSkipped
	finalization.SkipReason = strings.TrimSpace(reason)
	finalization.CompletedAt = nowRFC3339()
	finalization.AggregateClassification = FinalizationAggregateSkipped
	finalization.AggregateSummary = strings.TrimSpace(reason)
	record.Finalization = finalization
	record.Status = StatusCompleted
	record.FinishedAt = nowRFC3339()
	record.FailureReason = ""
	record.UpdatedAt = nowRFC3339()
	if item, loadErr := s.loadBacklogItemByRecord(record); loadErr == nil {
		_ = s.updateBacklogStatus(item, backlogStatusCompleted)
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) failFinalization(executionID, scenarioName, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusFailed
	finalization.Phase = FinalizationPhaseFailed
	finalization.CompletedAt = nowRFC3339()
	finalization.AggregateClassification = FinalizationAggregateNotAssessable
	finalization.AggregateSummary = strings.TrimSpace(message)
	if warning := strings.TrimSpace(message); warning != "" {
		finalization.Warnings = append(finalization.Warnings, newFinalizationWarning(
			finalizationWarningFinalizationInfra,
			scenarioName,
			warning,
			false,
		))
	}
	record.Finalization = finalization
	record.Status = StatusFailed
	record.FailureReason = strings.TrimSpace(message)
	record.FinishedAt = nowRFC3339()
	record.UpdatedAt = nowRFC3339()
	if item, loadErr := s.loadBacklogItemByRecord(record); loadErr == nil {
		_ = s.updateBacklogStatus(item, backlogStatusFailed)
	}
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func ensureFinalization(record *Record) *Finalization {
	if record.Finalization != nil {
		record.Finalization.Warnings = append([]FinalizationWarning(nil), record.Finalization.Warnings...)
		record.Finalization.AffectedScenarios = append([]string(nil), record.Finalization.AffectedScenarios...)
		record.Finalization.Scenarios = append([]ScenarioFinalization(nil), record.Finalization.Scenarios...)
		return record.Finalization
	}
	record.Finalization = &Finalization{
		Eligible:                isFinalizationEligible(*record),
		Status:                  FinalizationStatusPending,
		Phase:                   FinalizationPhaseScopeDetection,
		ScopeSource:             FinalizationScopeNone,
		Warnings:                []FinalizationWarning{},
		AffectedScenarios:       []string{},
		Scenarios:               []ScenarioFinalization{},
		StartedAt:               nowRFC3339(),
		AggregateSummary:        "",
		AggregateClassification: "",
	}
	return record.Finalization
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

func (s *Service) applyFinalizationScope(executionID string, scope finalizationScope, skipReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusRunning
	finalization.Phase = FinalizationPhaseScopeDetection
	finalization.ScopeSource = scope.source
	finalization.SkipReason = strings.TrimSpace(skipReason)
	finalization.AffectedScenarios = append([]string(nil), scope.affectedScenarios...)
	finalization.Warnings = append(finalization.Warnings, scope.warnings...)
	finalization.Scenarios = make([]ScenarioFinalization, 0, len(scope.affectedScenarios))
	for _, scenarioName := range scope.affectedScenarios {
		finalization.Scenarios = append(finalization.Scenarios, ScenarioFinalization{
			ScenarioName: scenarioName,
			ChangedPaths: append([]string(nil), scope.changedPathsByScenario[scenarioName]...),
			Restart:      RestartResult{Status: FinalizationStatusPending},
			Health:       HealthCheckResult{Status: FinalizationStatusPending},
			Review:       ScenarioReviewStep{Status: FinalizationStatusPending},
		})
	}
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) markFinalizationPhase(executionID, phase string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Status = FinalizationStatusRunning
	finalization.Phase = strings.TrimSpace(phase)
	record.Finalization = finalization
	record.Status = StatusValidating
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) appendFinalizationWarning(executionID string, warning FinalizationWarning) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	finalization.Warnings = append(finalization.Warnings, warning)
	record.Finalization = finalization
	record.UpdatedAt = nowRFC3339()
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(*record)
	return nil
}

func (s *Service) updateScenarioRestartState(executionID, scenarioName string, restart RestartResult) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Restart = restart
	})
}

func (s *Service) updateScenarioHealthState(executionID, scenarioName string, health HealthCheckResult) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Health = health
	})
}

func (s *Service) updateScenarioReviewState(executionID, scenarioName string, review ScenarioReviewStep) error {
	return s.updateScenarioFinalization(executionID, scenarioName, func(scenario *ScenarioFinalization) {
		scenario.Review = review
	})
}

func (s *Service) updateScenarioFinalization(executionID, scenarioName string, mutate func(*ScenarioFinalization)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := &records[idx]
	finalization := ensureFinalization(record)
	for i := range finalization.Scenarios {
		if finalization.Scenarios[i].ScenarioName != scenarioName {
			continue
		}
		mutate(&finalization.Scenarios[i])
		record.Finalization = finalization
		record.UpdatedAt = nowRFC3339()
		if err := s.store.Save(records); err != nil {
			return err
		}
		s.dispatchStatusUpdate(*record)
		return nil
	}
	return fmt.Errorf("scenario finalization not found for %s/%s", executionID, scenarioName)
}

func (s *Service) loadScenarioFinalization(executionID, scenarioName string) (ScenarioFinalization, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return ScenarioFinalization{}, err
	}
	record := records[idx]
	finalization := effectiveFinalization(record)
	if finalization == nil {
		return ScenarioFinalization{}, fmt.Errorf("execution %s has no finalization state", executionID)
	}
	for _, scenario := range finalization.Scenarios {
		if scenario.ScenarioName == scenarioName {
			return scenario, nil
		}
	}
	return ScenarioFinalization{}, fmt.Errorf("scenario finalization not found for %s/%s", executionID, scenarioName)
}

func (s *Service) loadExecutionForFinalization(executionID string) (Record, backlogItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, backlogItem{}, err
	}
	record := records[idx]
	item, loadErr := s.loadBacklogItemByRecord(&record)
	if loadErr != nil {
		return Record{}, backlogItem{}, loadErr
	}
	return record, item, nil
}

func (s *Service) beginFinalization(executionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processingFinalizations == nil {
		s.processingFinalizations = map[string]struct{}{}
	}
	if _, exists := s.processingFinalizations[executionID]; exists {
		return false
	}
	s.processingFinalizations[executionID] = struct{}{}
	return true
}

func (s *Service) endFinalization(executionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.processingFinalizations == nil {
		return
	}
	delete(s.processingFinalizations, executionID)
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
	log.Printf("[execution] finalization error for %s: %v", executionID, err)
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
