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

// isInspectableStatus reports whether a record is in a state the fail-closed
// correlation guard (inspectRunningRecordsLocked) should examine. There is no
// agent-manager status polling anymore: completion arrives exclusively through
// the operation runner's commit-execution-round path.
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
	// Workflow-owned records are applied only through the explicit workflow
	// apply endpoint. This legacy housekeeping pass must not poll Agent Manager
	// or decide when a workflow's terminal result becomes domain state.
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

	// Continuous goal-directed auto-enqueue (D4, default OFF): when enabled,
	// enqueue ready goal items via the governed QueueBacklog path before the
	// drain, so they compete for lane slots under the same caps as manual work.
	s.autoEnqueueGoalItems(ctx)

	// Drain pending items when capacity is available.
	s.drainPendingLocked(ctx)

	return nil
}

// drainRef is the goal-priority map key for a pending record.
func drainRef(r Record) string { return r.BacklogKind + "/" + r.BacklogName }

// sortPendingForDrain orders pending records for the drain: items in a goal
// sort before ungoaled items, higher goal priority first; ties and ungoaled
// items fall back to FIFO by CreatedAt. With a nil/empty priority map this is
// exactly FIFO — behavior-preserving for ungoaled work.
func sortPendingForDrain(pending []Record, priorities map[string]int) {
	sort.SliceStable(pending, func(i, j int) bool {
		pi, goaledI := priorities[drainRef(pending[i])]
		pj, goaledJ := priorities[drainRef(pending[j])]
		if goaledI != goaledJ {
			return goaledI // goaled items drain before ungoaled
		}
		if goaledI && goaledJ && pi != pj {
			return pi > pj // higher goal priority first
		}
		return pending[i].CreatedAt < pending[j].CreatedAt
	})
}

// goalItemPriorities returns the per-item goal-priority map from the optional
// provider, tolerating a nil provider or a read error by returning nil (the
// drain then falls back to FIFO).
func (s *Service) goalItemPriorities() map[string]int {
	if s.goalPriorityProvider == nil {
		return nil
	}
	m, err := s.goalPriorityProvider.ItemGoalPriorities()
	if err != nil {
		slog.Warn("drain: goal priorities unavailable; falling back to FIFO", "err", err)
		return nil
	}
	return m
}

// autoEnqueueGoalItems enqueues ready goal items through QueueBacklog when
// continuous auto-enqueue is enabled (D4). Every enqueue flows through the same
// governance (lane caps, preflight, circuit breaker, cost caps, queue depth) as
// manual queueing; items that cannot be queued (already queued, at depth, etc.)
// are skipped. Runs outside the service lock — QueueBacklog takes it per call.
func (s *Service) autoEnqueueGoalItems(ctx context.Context) {
	if s.autoDrainProvider == nil || s.goalReadyProvider == nil {
		return
	}
	if !s.autoDrainProvider.AutoDrainEnabled() {
		return
	}
	items, err := s.goalReadyProvider.ReadyGoalItems()
	if err != nil {
		slog.Warn("auto-drain: ready goal items unavailable", "err", err)
		return
	}
	for _, it := range items {
		_, qErr := s.QueueBacklog(ctx, CreateRequest{BacklogKind: it.Kind, BacklogName: it.Name})
		if qErr == nil {
			continue
		}
		// Queue-depth is the natural stop signal; other per-item failures
		// (already queued, preflight, circuit breaker) are expected and skipped.
		if strings.Contains(qErr.Error(), "queue depth limit exceeded") {
			return
		}
	}
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

	// Collect pending records, then order them goal-priority-first with a FIFO
	// fallback (identical to the old FIFO-by-CreatedAt when nothing is goaled).
	var pending []Record
	for _, r := range records {
		if r.Status == StatusPending {
			pending = append(pending, r)
		}
	}
	sortPendingForDrain(pending, s.goalItemPriorities())

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

	finalizationCandidates = s.collectValidatingCandidatesLocked(records)
	s.inspectRunningRecordsLocked(ctx, records, &changed, changedRecords, &finalizationCandidates)
	holdCandidates = s.collectHoldCandidatesLocked(records)

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

// collectValidatingCandidatesLocked returns IDs of records in StatusValidating
// that have finalization state and are not already being processed.
func (s *Service) collectValidatingCandidatesLocked(records []Record) []string {
	candidates := make([]string, 0)
	for i := range records {
		record := &records[i]
		if record.Status == StatusValidating && record.Finalization != nil {
			if _, exists := s.processingFinalizations[record.ExecutionID]; !exists {
				candidates = append(candidates, record.ExecutionID)
			}
		}
	}
	return candidates
}

// inspectRunningRecordsLocked enforces the single-completion-authority
// invariant over active records. Runner-owned records (every record created
// since the Phase-6 cutover carries OpExecutionID) are driven to terminal by
// the completion bridge's commit-execution-round handler; the startup
// reconciliation sweep is the backstop for a runner-owned record whose bridge
// delivery is lost. The legacy agent-manager poll loop that drove
// pre-migration records is gone (Phase 9): the Phase-8 migration imported
// every pre-cutover record, so an active record WITHOUT an operation
// correlation can no longer legitimately exist — fail it closed instead of
// silently stranding it.
func (s *Service) inspectRunningRecordsLocked(_ context.Context, records []Record, changed *bool, changedRecords map[string]Record, _ *[]string) {
	for i := range records {
		record := &records[i]
		if !isInspectableStatus(record.Status) || strings.TrimSpace(record.RunID) == "" {
			continue
		}
		if strings.TrimSpace(record.OpExecutionID) != "" {
			continue
		}
		if strings.TrimSpace(record.AgentWorkflowExecutionID) != "" {
			continue
		}
		s.markRunFailed(record, "record has no operation correlation; the legacy poll driver was removed after the state migration",
			"active record without op_execution_id (impossible post-migration), marking failed", "failed to set backlog status to in_review for uncorrelated record",
			slog.String("backlog_ref", record.BacklogKind+"/"+record.BacklogName))
		*changed = true
		changedRecords[record.ExecutionID] = *record
	}
}

// collectHoldCandidatesLocked returns IDs of needs_review records that are
// eligible for pre-merge engagement-hold processing this cycle. Returns an
// empty slice (not nil) when the engagement machinery is inactive.
func (s *Service) collectHoldCandidatesLocked(records []Record) []string {
	if !s.engagementHoldActive() {
		return []string{}
	}
	var candidates []string
	for i := range records {
		record := &records[i]
		if record.Status != StatusNeedsReview || strings.TrimSpace(record.EngagementHoldAt) != "" {
			continue
		}
		if _, exists := s.processingHolds[record.ExecutionID]; exists {
			continue
		}
		candidates = append(candidates, record.ExecutionID)
	}
	return pathutil.UniqueSortedStrings(candidates)
}

// markRunFailed transitions a record to failed (today only the fail-closed
// uncorrelated-record guard), emits the execution event, and best-effort lands
// the backlog item in in_review for operator triage.
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
}

// applyTerminalTransition finalizes a record that has reached a terminal status:
// it stamps FinishedAt and routes the backlog item per outcome (finalization
// handoff, spec-sync archive, in_review, or restore).
func (s *Service) applyTerminalTransition(ctx context.Context, record *Record, runState agentmanager.RunState, nextStatus Status, finalizationCandidates *[]string) {
	if strings.TrimSpace(runState.FinishedAt) != "" {
		record.FinishedAt = runState.FinishedAt
	} else {
		record.FinishedAt = nowRFC3339()
	}

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
