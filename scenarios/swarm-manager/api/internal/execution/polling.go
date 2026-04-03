package execution

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/pathutil"
)

// ProcessActiveExecutions advances agent-manager-backed executions, drains
// pending items when capacity opens, and drives post-run finalization work.
func (s *Service) ProcessActiveExecutions(ctx context.Context) error {
	s.mu.Lock()
	candidates, err := s.refreshRunningLocked(ctx)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	for _, executionID := range candidates {
		logFinalizationError(executionID, s.processFinalization(ctx, executionID))
	}

	// Drain pending items when capacity is available.
	s.drainPendingLocked(ctx)

	return nil
}

// drainPendingLocked starts pending executions when concurrency slots are available.
func (s *Service) drainPendingLocked(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	gov, govErr := s.governanceProvider.LoadGovernance()
	if govErr != nil {
		return
	}

	records, err := s.store.Load()
	if err != nil {
		return
	}

	// Collect pending records sorted by creation time (oldest first).
	var pending []Record
	for _, r := range records {
		if r.Status == StatusPending {
			pending = append(pending, r)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].CreatedAt < pending[j].CreatedAt
	})

	for _, p := range pending {
		active := countActiveExecutions(records)
		if active >= gov.MaxConcurrentExecutions {
			break
		}
		started, startErr := s.startLocked(ctx, p.ExecutionID)
		if startErr != nil {
			if errors.Is(startErr, errAtCapacity) {
				break
			}
			slog.Warn("drain: failed to start execution", "execution_id", p.ExecutionID, "err", startErr)
			continue
		}
		// Refresh records to get updated state after start.
		if updated, loadErr := s.store.Load(); loadErr == nil {
			records = updated
		}
		_ = started
	}
}

func (s *Service) refreshRunningLocked(ctx context.Context) ([]string, error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	changed := false
	changedRecords := make(map[string]Record)
	finalizationCandidates := make([]string, 0)

	for i := range records {
		record := &records[i]
		if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
			if migrateLegacyFinalizationState(record, item) {
				changed = true
				changedRecords[record.ExecutionID] = *record
			}
		}
	}

	for i := range records {
		record := &records[i]
		if record.Status == StatusValidating && effectiveFinalization(*record) != nil {
			if _, exists := s.processingFinalizations[record.ExecutionID]; !exists {
				finalizationCandidates = append(finalizationCandidates, record.ExecutionID)
			}
		}
	}

	// Handle running/starting/needs_review records.
	if s.inspector != nil {
		for i := range records {
			record := &records[i]
			if (record.Status != StatusStarting && record.Status != StatusRunning && record.Status != StatusNeedsReview) || strings.TrimSpace(record.RunID) == "" {
				continue
			}
			runState, err := s.inspector.GetRunState(ctx, record.RunID)
			if err != nil {
				continue
			}
			nextStatus, reason := mapRunStatus(runState.Status, runState.ErrorMsg)
			if nextStatus == record.Status {
				continue
			}
			prevStatus := record.Status
			record.Status = nextStatus
			record.FailureReason = reason
			record.UpdatedAt = nowRFC3339()
			// Only set FinishedAt for terminal statuses
			if nextStatus == StatusCompleted || nextStatus == StatusFailed || nextStatus == StatusCanceled {
				if strings.TrimSpace(runState.FinishedAt) != "" {
					record.FinishedAt = runState.FinishedAt
				} else {
					record.FinishedAt = nowRFC3339()
				}
				// Post-completion hook: archive scenario after successful spec-sync
				if record.ArchiveContext != nil {
					if nextStatus == StatusCompleted {
						s.handleSpecSyncComplete(ctx, record)
					}
					// For spec-sync failures, leave status as failed for UI recovery
				} else if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
					if nextStatus == StatusCompleted {
						if isFinalizationEligible(*record) {
							record.Status = StatusValidating
							record.Finalization = &Finalization{
								Eligible:          true,
								Status:            FinalizationStatusPending,
								Phase:             FinalizationPhaseScopeDetection,
								ScopeSource:       FinalizationScopeNone,
								Warnings:          []FinalizationWarning{},
								AffectedScenarios: []string{},
								Scenarios:         []ScenarioFinalization{},
								StartedAt:         nowRFC3339(),
							}
							record.FinishedAt = ""
							finalizationCandidates = append(finalizationCandidates, record.ExecutionID)
						} else {
							record.Finalization = &Finalization{
								Eligible:                false,
								Status:                  FinalizationStatusSkipped,
								Phase:                   FinalizationPhaseSkipped,
								ScopeSource:             FinalizationScopeNone,
								SkipReason:              "execution type does not use post-run checks",
								StartedAt:               nowRFC3339(),
								CompletedAt:             nowRFC3339(),
								AggregateClassification: FinalizationAggregateSkipped,
								AggregateSummary:        "execution type does not use post-run checks",
								Warnings:                []FinalizationWarning{},
								AffectedScenarios:       []string{},
								Scenarios:               []ScenarioFinalization{},
							}
							_ = s.updateBacklogStatus(item, backlogStatusCompleted)
						}
						// Clear circuit breaker on success.
						_ = s.circuitBreaker.RecordSuccess(record.BacklogKind + "/" + record.BacklogName)
					} else if nextStatus == StatusFailed {
						_ = s.updateBacklogStatus(item, backlogStatusFailed)
						// Record failure in circuit breaker.
						cbKey := record.BacklogKind + "/" + record.BacklogName
						if cbGov, cbGovErr := s.governanceProvider.LoadGovernance(); cbGovErr == nil {
							_ = s.circuitBreaker.RecordFailure(cbKey, cbGov.CircuitBreakerThreshold)
						}
					} else if nextStatus == StatusCanceled {
						_ = s.updateBacklogStatus(item, restoreBacklogStatus(*record))
					}
				}
			}
			changed = true
			changedRecords[record.ExecutionID] = *record
			s.logExecutionEvent(*record, prevStatus)
		}
	}

	if changed {
		if err := s.store.Save(records); err != nil {
			return nil, err
		}
		for _, record := range changedRecords {
			s.dispatchStatusUpdate(record)
		}
		return pathutil.UniqueSortedStrings(finalizationCandidates), nil
	}
	return pathutil.UniqueSortedStrings(finalizationCandidates), nil
}

func mapRunStatus(status, errorMsg string) (Status, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting":
		return StatusStarting, ""
	case "running":
		return StatusRunning, ""
	case "needs_review":
		return StatusNeedsReview, ""
	case "complete":
		return StatusCompleted, ""
	case "failed":
		reason := strings.TrimSpace(errorMsg)
		if reason == "" {
			reason = "agent-manager run failed"
		}
		return StatusFailed, reason
	case "cancelled":
		return StatusCanceled, ""
	default:
		return StatusRunning, ""
	}
}

func matchesFilters(record Record, filters ListFilters) bool {
	if strings.TrimSpace(filters.Status) != "" && string(record.Status) != strings.TrimSpace(filters.Status) {
		return false
	}
	if strings.TrimSpace(filters.Mode) != "" && string(record.Mode) != strings.TrimSpace(filters.Mode) {
		return false
	}
	if strings.TrimSpace(filters.BacklogKind) != "" && record.BacklogKind != strings.TrimSpace(filters.BacklogKind) {
		return false
	}
	if strings.TrimSpace(filters.BacklogName) != "" && record.BacklogName != strings.TrimSpace(filters.BacklogName) {
		return false
	}
	if strings.TrimSpace(filters.StartedBy) != "" && record.StartedBy != strings.TrimSpace(filters.StartedBy) {
		return false
	}
	if strings.TrimSpace(filters.CreatedFrom) != "" {
		from, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedFrom))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.Before(from) {
				return false
			}
		}
	}
	if strings.TrimSpace(filters.CreatedTo) != "" {
		to, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedTo))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.After(to) {
				return false
			}
		}
	}
	return true
}
