package execution

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
)

// Start starts a pending/failed execution now.
func (s *Service) Start(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked(ctx, executionID)
}

func (s *Service) startLocked(ctx context.Context, executionID string) (Record, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	if record.Status == StatusStarting || record.Status == StatusRunning || record.Status == StatusNeedsReview || record.Status == StatusCompleted {
		return record, nil
	}
	if record.Status == StatusCanceled {
		return Record{}, apierr.BadRequest("cannot start canceled execution")
	}

	// Concurrency gate. Backlog item processing always lives in the
	// Execute lane — no derivation needed here. service_queue.QueueBacklog
	// catches errAtCapacity and leaves the record pending so the poller
	// drains it later (preserving pre-P2 enqueue-on-saturation semantics).
	if gov, govErr := s.governanceProvider.LoadGovernance(); govErr == nil {
		active := countActiveExecutions(records)
		if active >= laneCapacity(gov, agentactivity.LaneExecute) {
			return Record{}, apierr.Wrap(errAtCapacity, http.StatusConflict, "execute lane saturated")
		}
	}

	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return Record{}, err
	}
	preflight := s.processPreflightForItem(item, false)
	if !preflight.Ready && (!record.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return Record{}, apierr.BadRequest("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}

	// Baseline Modes exclusivity (plan P-b.4): with shadow engagement on, refuse
	// to start an owner whose projected scope (acceptance_allow) intersects a
	// scenario already engaged under a different owner. Block-at-start, never
	// queue. No-op when the engagement machinery is off. Force does not bypass —
	// the conflict is a data-safety invariant, not a readiness heuristic.
	if err := s.checkExclusivityAtStart(item, ownerKeyFor(record.BacklogKind, record.BacklogName)); err != nil {
		return Record{}, err
	}

	// Pre-execution baseline capture: pin a GCT baseline of each declared
	// scenario's current state so finalization can separate regressions this
	// item causes from pre-existing failures. An execution-domain prep step
	// independent of how the agent is launched. Cheap synchronous planning here;
	// the slow snapshot runs detached. Best-effort — never blocks the start.
	record.PreExecBaselines = s.capturePreExecBaselinesLocked(ctx, item)

	// Plan-backed items start as an execution-run operation against their
	// plan-execution target. Research-conclusion items have no execution plan_ref
	// and cannot target a plan-execution operation, so they start the
	// research-conclude operation against their backlog-item target instead.
	if hasExecutionPlanRef(item) {
		return s.startPlanOperationLocked(ctx, records, idx, record, item)
	}
	return s.startConclusionOperationLocked(ctx, records, idx, record)
}

// startPlanOperationLocked starts a plan-backed item's primary execution as an
// execution-run operation against its plan-execution target. The operation
// runner spawns the agent through the operating-mode engine's chokepoint
// (execution-drain -> phased-plan-drain, inheriting the item's write-scope
// containment via the plan-execution target adapter) and returns the live run
// association; the execution record keeps tracking the returned agent run id for
// status and finalization (transitional — the completion authority consolidates
// on the workflow in slice C, note d789cb50).
func (s *Service) startPlanOperationLocked(ctx context.Context, records []Record, idx int, record Record, item backlogItem) (Record, error) {
	if s.operationStarter == nil {
		return Record{}, apierr.Unavailable("execution operation runner is not available")
	}
	// Plan Manager stays the sole plan authority — the operation resolves the live
	// plan context there from the item's canonical execution_spec plan_ref.
	planHandle, err := executionPlanHandle(item)
	if err != nil {
		return Record{}, apierr.BadRequest("%s", err.Error())
	}
	res, err := s.operationStarter.StartOperation(ctx, OperationStartRequest{
		Operation:        operationExecutionRun,
		OperationVersion: operationVersionPinned,
		TargetKind:       targetKindPlanExecution,
		TargetID:         planHandle,
		IdempotencyKey:   "exec-" + record.ExecutionID,
		RequestedBy:      record.StartedBy,
	})
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	// A start that yields no trackable agent run id could never be polled to a
	// terminal status — the record would strand in "starting" (the poller skips
	// run-id-less records and Cancel refuses them). Reap the dangling operation and
	// fail the start so the record stays pending/retryable instead.
	if strings.TrimSpace(res.RunID) == "" {
		if cerr := s.operationStarter.CancelOperation(ctx, OperationCancelRequest{
			TargetKind: targetKindPlanExecution, TargetID: planHandle, ExecutionID: res.ExecutionID,
		}); cerr != nil {
			slog.Warn("execution: reap of run-id-less start failed", "execution_id", record.ExecutionID, "err", cerr)
		}
		return Record{}, apierr.BadGateway("execution operation started but returned no run id; agent-manager may be unavailable")
	}

	record.RunID = res.RunID
	record.OpWorkflowID = res.WorkflowID
	record.OpExecutionID = res.ExecutionID
	record.StartedAt = nowRFC3339()
	record.FinishedAt = ""
	record.FailureReason = ""
	record.Status = StatusStarting
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusUpdate(record)
	return record, nil
}

// startConclusionOperationLocked starts a research-conclusion item's primary
// execution as a research-conclude operation against its backlog-item target.
// Research/idea items have no execution plan_ref (their deliverable is the
// conclusion itself), so they cannot target a plan-execution operation; the
// research-conclude mode reads the item spec directly. The operation runner spawns
// the agent through the operating-mode engine's chokepoint and returns the live
// run association; the completion bridge's commit-execution-round handler drives
// the record to terminal, so the poller defers it (OpExecutionID set). Mirrors
// startPlanOperationLocked (no Go-side prompt: the mode owns it).
func (s *Service) startConclusionOperationLocked(ctx context.Context, records []Record, idx int, record Record) (Record, error) {
	if s.operationStarter == nil {
		return Record{}, apierr.Unavailable("execution operation runner is not available")
	}
	res, err := s.operationStarter.StartOperation(ctx, OperationStartRequest{
		Operation:        operationResearchConclude,
		OperationVersion: operationVersionPinned,
		TargetKind:       targetKindBacklogItem,
		TargetID:         record.BacklogKind + "/" + record.BacklogName,
		IdempotencyKey:   "exec-" + record.ExecutionID,
		RequestedBy:      record.StartedBy,
	})
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	// A start that yields no trackable agent run id could never be driven to a
	// terminal status — reap the dangling operation and fail the start so the
	// record stays pending/retryable (mirrors startPlanOperationLocked).
	if strings.TrimSpace(res.RunID) == "" {
		if cerr := s.operationStarter.CancelOperation(ctx, OperationCancelRequest{
			TargetKind: targetKindBacklogItem, TargetID: record.BacklogKind + "/" + record.BacklogName, ExecutionID: res.ExecutionID,
		}); cerr != nil {
			slog.Warn("execution: reap of run-id-less conclusion start failed", "execution_id", record.ExecutionID, "err", cerr)
		}
		return Record{}, apierr.BadGateway("research-conclude operation started but returned no run id; agent-manager may be unavailable")
	}
	record.RunID = res.RunID
	record.OpWorkflowID = res.WorkflowID
	record.OpExecutionID = res.ExecutionID
	record.StartedAt = nowRFC3339()
	record.FinishedAt = ""
	record.FailureReason = ""
	record.Status = StatusStarting
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusUpdate(record)
	return record, nil
}

// reapOperationForRecord marks a canceled run's operation execution canceled in
// the durable workflow, so the record does not linger "running". Best-effort: it
// only applies to operation-started records (OpExecutionID set) and never fails
// the cancel — a missed reap is recovered by slice-C run reconciliation.
func (s *Service) reapOperationForRecord(ctx context.Context, record Record) {
	if s.operationStarter == nil || strings.TrimSpace(record.OpExecutionID) == "" {
		return
	}
	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return
	}
	handle, err := executionPlanHandle(item)
	if err != nil {
		return
	}
	if err := s.operationStarter.CancelOperation(ctx, OperationCancelRequest{
		TargetKind:  targetKindPlanExecution,
		TargetID:    handle,
		ExecutionID: record.OpExecutionID,
	}); err != nil {
		slog.Warn("execution: reap operation on cancel failed",
			"execution_id", record.ExecutionID, "op_execution_id", record.OpExecutionID, "err", err)
	}
}

// Cancel cancels a scheduled record before it starts.
func (s *Service) Cancel(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]

	prevStatus := record.Status
	switch record.Status {
	case StatusPending:
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		return record, nil
	case StatusStarting, StatusRunning, StatusNeedsReview:
		if s.stopper == nil {
			return Record{}, apierr.BadRequest("cancel is not supported by current agent service")
		}
		if strings.TrimSpace(record.RunID) == "" {
			return Record{}, apierr.BadRequest("execution has no run id")
		}
		if err := s.stopper.StopRun(ctx, record.RunID); err != nil {
			return Record{}, err
		}
		// Reap the operation execution so its durable workflow record does not
		// linger "running" and the refresh driver stops polling the stopped run.
		// The StopRun above is the cooperative cancel; this only updates bookkeeping.
		s.reapOperationForRecord(ctx, record)
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		return record, nil
	case StatusValidating, StatusNeedsFixup:
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		// Also mark finalization as failed so the UI stops showing the progress indicator.
		if record.Finalization != nil {
			record.Finalization.Status = FinalizationStatusFailed
			record.Finalization.Phase = FinalizationPhaseFailed
			record.Finalization.CompletedAt = nowRFC3339()
		}
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		return record, nil
	default:
		return Record{}, apierr.BadRequest("only pending/starting/running/needs_review/validating/needs_fixup executions can be canceled")
	}
}

// TriggerReview reruns the unified post-run finalization flow for a terminal
// execution.
// DOC: docs/internal/SEAMS.md#trigger-review-api
func (s *Service) TriggerReview(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := &records[idx]

	switch record.Status {
	case StatusCompleted, StatusNeedsFixup, StatusFailed:
		// Valid terminal statuses for triggering review
	default:
		return Record{}, apierr.BadRequest("cannot trigger post-run checks for execution in %q status", record.Status)
	}

	if _, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr != nil {
		return Record{}, apierr.NotFound("backlog item not found for post-run checks")
	}
	if !isFinalizationEligible(*record) {
		return Record{}, apierr.BadRequest("execution type %q does not support post-run checks", record.effectiveRunType())
	}

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
	record.LegacyReviewResult = nil
	record.LegacyReviewJobID = ""
	record.LegacyReviewSkipReason = ""
	record.LegacyReviewStartedAt = ""
	record.FinishedAt = ""
	record.UpdatedAt = nowRFC3339()
	record.FailureReason = ""

	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusUpdate(*record)
	return *record, nil
}

// Retry semantics now live in retry.go as RetryAsNewAttempt — the in-place
// retry was removed as part of the retry-as-new-attempt rewrite. Audit
// preservation requires that the failed execution row remain untouched and
// that retries spawn a new Record parented to the prior one.
