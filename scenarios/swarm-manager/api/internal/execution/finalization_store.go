package execution

import (
	"fmt"
	"strings"
)

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

	// If the execution has been cancelled, don't overwrite the terminal state.
	if record.Status == StatusCanceled {
		return fmt.Errorf("execution %s was canceled", executionID)
	}

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
