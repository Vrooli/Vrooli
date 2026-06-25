package execution

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/pathutil"
)

// isInspectableStatus reports whether a record is in a state the inspector pass
// should poll agent-manager about.
func isInspectableStatus(status Status) bool {
	return status == StatusStarting || status == StatusRunning || status == StatusNeedsReview
}

// isTerminalStatus reports whether a status is a final run outcome.
func isTerminalStatus(status Status) bool {
	return status == StatusCompleted || status == StatusFailed || status == StatusCanceled
}

// ProcessActiveExecutions advances agent-manager-backed executions, drains
// pending items when capacity opens, and drives post-run finalization work.
func (s *Service) ProcessActiveExecutions(ctx context.Context) error {
	s.mu.Lock()
	holdCandidates, candidates, err := s.refreshRunningLocked(ctx)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	// Pre-merge engagement holds (plan P-b.3): open shadow restore points from
	// the actual diff and approve the merge for runs parked at needs_review.
	// Runs outside the service lock (baseline start is slow); a failure leaves the
	// run held for a later cycle/operator.
	for _, executionID := range holdCandidates {
		if holdErr := s.processEngagementHold(ctx, executionID); holdErr != nil {
			slog.Warn("baseline engagement: pre-merge hold failed (run left held)",
				"execution_id", executionID, "err", holdErr)
		}
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
		if active >= laneCapacity(gov, agentactivity.LaneExecute) {
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

func (s *Service) refreshRunningLocked(ctx context.Context) (holdCandidates []string, finalizationCandidates []string, err error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, nil, err
	}

	changed := false
	changedRecords := make(map[string]Record)
	finalizationCandidates = make([]string, 0)
	holdCandidates = make([]string, 0)

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
			if !isInspectableStatus(record.Status) || strings.TrimSpace(record.RunID) == "" {
				continue
			}

			tracker := s.ensureRunTracker(record.RunID)

			// Max-age staleness check.
			if time.Since(tracker.FirstSeen) > s.maxRunAge {
				s.markRunFailed(record, "run exceeded maximum age timeout",
					"run exceeded max age, marking failed", "failed to set backlog status to in_review after max-age timeout",
					slog.Duration("age", time.Since(tracker.FirstSeen)))
				changed = true
				changedRecords[record.ExecutionID] = *record
				continue
			}

			runState, err := s.inspector.GetRunState(ctx, record.RunID)
			if err != nil {
				tracker.ConsecutiveErrors++
				if tracker.ConsecutiveErrors >= s.maxConsecutiveErrors {
					s.markRunFailed(record, "lost contact with agent-manager run",
						"run hit max consecutive errors, marking failed", "failed to set backlog status to in_review after consecutive-errors timeout",
						slog.Int("errors", tracker.ConsecutiveErrors))
					changed = true
					changedRecords[record.ExecutionID] = *record
				} else {
					slog.Warn("GetRunState error",
						"execution_id", record.ExecutionID,
						"run_id", record.RunID,
						"err", err,
						"consecutive", tracker.ConsecutiveErrors)
				}
				continue
			}
			tracker.ConsecutiveErrors = 0

			nextStatus, reason := mapRunStatus(runState.Status, runState.ErrorMsg, tracker, s.maxConsecutiveUnknown)
			if nextStatus == record.Status {
				continue
			}
			prevStatus := record.Status
			record.Status = nextStatus
			record.FailureReason = reason
			record.UpdatedAt = nowRFC3339()
			// Only set FinishedAt for terminal statuses
			if isTerminalStatus(nextStatus) {
				s.applyTerminalTransition(ctx, record, runState, nextStatus, &finalizationCandidates)
			}
			changed = true
			changedRecords[record.ExecutionID] = *record
			s.logExecutionEvent(*record, prevStatus)
		}
	}

	// Collect pre-merge engagement-hold candidates: runs parked at needs_review
	// whose hold hasn't been processed yet and that aren't already in flight.
	// Done after the inspector pass so a same-cycle transition into needs_review
	// is caught immediately. Dormant unless the engagement machinery is active.
	if s.engagementHoldActive() {
		for i := range records {
			record := &records[i]
			if record.Status != StatusNeedsReview || strings.TrimSpace(record.EngagementHoldAt) != "" {
				continue
			}
			if _, exists := s.processingHolds[record.ExecutionID]; exists {
				continue
			}
			holdCandidates = append(holdCandidates, record.ExecutionID)
		}
		holdCandidates = pathutil.UniqueSortedStrings(holdCandidates)
	}

	if changed {
		if err := s.store.Save(records); err != nil {
			return nil, nil, err
		}
		for _, record := range changedRecords {
			s.dispatchStatusUpdate(record)
		}
	}
	return holdCandidates, pathutil.UniqueSortedStrings(finalizationCandidates), nil
}

// markRunFailed transitions a record to failed (max-age / lost-contact paths),
// emits the execution event, best-effort lands the backlog item in in_review,
// and clears the run tracker.
func (s *Service) markRunFailed(record *Record, failureReason, warnMsg, backlogWarnMsg string, extra slog.Attr) {
	slog.Warn(warnMsg,
		"execution_id", record.ExecutionID,
		"run_id", record.RunID,
		extra.Key, extra.Value)
	prevStatus := record.Status
	record.Status = StatusFailed
	record.FailureReason = failureReason
	record.UpdatedAt = nowRFC3339()
	record.FinishedAt = record.UpdatedAt
	s.logExecutionEvent(*record, prevStatus)
	if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
		// Land in in_review (not terminal) so the review agent can document the
		// timeout and the user decides whether to mark failed, retry, or followup.
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			slog.Warn(backlogWarnMsg,
				"execution_id", record.ExecutionID, "backlog_ref", record.BacklogKind+"/"+record.BacklogName, "err", err)
		}
	}
	s.deleteRunTracker(record.RunID)
}

// applyTerminalTransition finalizes a record that has reached a terminal status:
// it stamps FinishedAt, tears down the tracker, and routes the backlog item per
// outcome (finalization handoff, spec-sync archive, in_review, or restore).
func (s *Service) applyTerminalTransition(ctx context.Context, record *Record, runState agentmanager.RunState, nextStatus Status, finalizationCandidates *[]string) {
	if strings.TrimSpace(runState.FinishedAt) != "" {
		record.FinishedAt = runState.FinishedAt
	} else {
		record.FinishedAt = nowRFC3339()
	}
	s.deleteRunTracker(record.RunID)

	// Post-completion hook: archive scenario after successful spec-sync.
	if record.ArchiveContext != nil {
		if nextStatus == StatusCompleted {
			s.handleSpecSyncComplete(ctx, record)
		}
		// For spec-sync failures, leave status as failed for UI recovery.
		return
	}

	item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if loadErr != nil {
		return
	}
	switch nextStatus {
	case StatusCompleted:
		s.applyCompletedTransition(record, item, finalizationCandidates)
	case StatusFailed:
		// Land in in_review so review agent documents the failure and user decides
		// terminal via review-decide. Circuit-breaker accounting still fires.
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			slog.Warn("failed to set backlog status to in_review after run failure",
				"execution_id", record.ExecutionID, "backlog_ref", record.BacklogKind+"/"+record.BacklogName, "err", err)
		}
		cbKey := record.BacklogKind + "/" + record.BacklogName
		if cbGov, cbGovErr := s.governanceProvider.LoadGovernance(); cbGovErr == nil {
			_ = s.circuitBreaker.RecordFailure(cbKey, cbGov.CircuitBreakerThreshold)
		}
	case StatusCanceled:
		if err := s.updateBacklogStatus(item, restoreBacklogStatus(*record)); err != nil {
			slog.Warn("failed to restore backlog status after run cancellation",
				"execution_id", record.ExecutionID, "backlog_ref", record.BacklogKind+"/"+record.BacklogName, "err", err)
		}
	}
}

// applyCompletedTransition routes a completed run into finalization (eligible)
// or directly to review_pending (ineligible), then clears the circuit breaker.
func (s *Service) applyCompletedTransition(record *Record, item backlogItem, finalizationCandidates *[]string) {
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
		*finalizationCandidates = append(*finalizationCandidates, record.ExecutionID)
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
		// No review agent runs for non-eligible items; go straight to
		// review_pending so the user decides the terminal state instead of
		// auto-completing.
		if err := s.updateBacklogStatus(item, backlogStatusReviewPending); err != nil {
			slog.Warn("failed to set backlog status to review_pending for non-finalization item",
				"execution_id", record.ExecutionID, "backlog_ref", record.BacklogKind+"/"+record.BacklogName, "err", err)
		}
	}
	// Clear circuit breaker on success.
	_ = s.circuitBreaker.RecordSuccess(record.BacklogKind + "/" + record.BacklogName)
}

func mapRunStatus(status, errorMsg string, tracker *runTracker, maxConsecutiveUnknown int) (Status, string) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "pending", "starting":
		tracker.ConsecutiveUnknown = 0
		return StatusStarting, ""
	case "running":
		tracker.ConsecutiveUnknown = 0
		return StatusRunning, ""
	case "needs_review":
		tracker.ConsecutiveUnknown = 0
		return StatusNeedsReview, ""
	case "complete":
		tracker.ConsecutiveUnknown = 0
		return StatusCompleted, ""
	case "failed":
		tracker.ConsecutiveUnknown = 0
		reason := strings.TrimSpace(errorMsg)
		if reason == "" {
			reason = "agent-manager run failed"
		}
		return StatusFailed, reason
	case "cancelled":
		tracker.ConsecutiveUnknown = 0
		return StatusCanceled, ""
	default:
		tracker.ConsecutiveUnknown++
		if tracker.ConsecutiveUnknown == 1 {
			slog.Warn("unknown run status from agent-manager", "status", status)
		}
		if tracker.ConsecutiveUnknown >= maxConsecutiveUnknown {
			return StatusFailed, "unknown run status: " + status
		}
		return StatusRunning, ""
	}
}

func (s *Service) ensureRunTracker(runID string) *runTracker {
	if t, ok := s.runTrackers[runID]; ok {
		return t
	}
	t := &runTracker{FirstSeen: time.Now()}
	s.runTrackers[runID] = t
	return t
}

func (s *Service) deleteRunTracker(runID string) {
	delete(s.runTrackers, runID)
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
