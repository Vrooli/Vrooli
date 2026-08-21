package execution

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/transitionrunner"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
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
	preflight := s.processPreflightForItem(ctx, item, false)
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

	// Every backlog kind, including research, executes only against a canonical
	// accepted plan. There is no planless conclusion execution branch.
	if !hasExecutionPlanRef(item) {
		return Record{}, apierr.Conflict("a canonical execution plan is required before starting work")
	}
	return s.startPlanOperationLocked(ctx, records, idx, record, item)
}

// startPlanOperationLocked is the single hard-cut plan-execution consumer. It
// snapshots the authorized Plan Manager frontier and starts the bounded,
// scenario-authored phased-plan workflow. The execution record tracks the
// workflow aggregate and its pinned digests; no operation wrapper or direct
// Run creation/continuation participates in this path.
func (s *Service) startPlanOperationLocked(ctx context.Context, records []Record, idx int, record Record, item backlogItem) (Record, error) {
	// Availability is the runner's, not a per-subject workflow client's. The
	// field this used to check was never set by composition, so plan execution
	// reported "not available" on every start.
	if s.transitionRunner == nil {
		return Record{}, apierr.Unavailable("transition runner is not configured")
	}
	// Plan Manager stays the sole plan authority: resolve the live context from
	// the item's canonical execution_spec plan_ref before hashing the frontier.
	planHandle, err := executionPlanHandle(item)
	if err != nil {
		return Record{}, apierr.BadRequest("%s", err.Error())
	}
	if record.PlanManagerExecutionID == "" {
		client, ok := s.planRenderer.(interface {
			Resume(context.Context, *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error)
		})
		if ok {
			resumed, resumeErr := client.Resume(ctx, &executionv1.ResumeRequest{PlanOrExecution: planHandle})
			if resumeErr != nil {
				return Record{}, apierr.BadGateway("resume plan-manager execution: %s", resumeErr)
			}
			if resumed.GetExecution() == nil || strings.TrimSpace(resumed.GetExecution().GetId()) == "" {
				return Record{}, apierr.BadGateway("plan-manager resume omitted execution")
			}
			record.PlanManagerExecutionID = resumed.GetExecution().GetId()
		}
	}
	workflow, err := s.resolveWorkflow("plan.execute")
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	if !s.strategyMatchesWorkflow(record.ExecutionStrategy, workflow.Key) {
		return Record{}, apierr.BadRequest("execution strategy %q does not match declared plan workflow", record.ExecutionStrategy)
	}
	// Persist the resolved Plan Manager execution before starting: the runner's
	// input builder reprojects the frontier from the durable record, so
	// PlanExecutionID has to be readable there and at apply time.
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, apierr.Internal("persist plan-execution record before start: %s", err.Error())
	}
	started, err := s.transitionRunner.StartWith(ctx, "plan.execute", record.ExecutionID, transitionrunner.PreparedInput{FirstRunNodeID: "slice", Activity: &transitionrunner.Activity{OwnerType: "backlog", OwnerKind: record.BacklogKind, OwnerName: record.BacklogName, Purpose: "process"}})
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	res := agentmanager.WorkflowStart{ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest}
	if len(started.Attempts) > 0 {
		res.RunID = started.Attempts[0].RunID
	}
	if strings.TrimSpace(res.ExecutionID) == "" {
		return Record{}, apierr.BadGateway("phased-plan workflow started but returned no execution id")
	}

	record.RunID = res.RunID
	record.TaskID = res.ExecutionID
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

func (s *Service) strategyMatchesWorkflow(strategyID, workflowKey string) bool {
	strategyID = strings.TrimSpace(strategyID)
	if strategyID == "" {
		strategyID = defaultExecutionStrategy
	}
	for _, strategy := range s.declaredExecutionStrategies() {
		if strategy.ID == strategyID {
			return strategy.WorkflowKey == workflowKey
		}
	}
	return false
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
		if correlation, correlationErr := s.transitionCorrelation(record); correlationErr == nil {
			// One cancel path for every transition. The previous per-transition
			// branches each reached for a workflow client that composition never
			// set, so cancelling any in-flight execution reported "cancel is not
			// supported" regardless of which transition owned it.
			if s.transitionRunner == nil {
				return Record{}, apierr.Unavailable("transition runner is not configured")
			}
			if err := s.transitionRunner.Cancel(ctx, correlation.ExecutionID, "cancel-"+record.ExecutionID, "consumer execution canceled"); err != nil {
				return Record{}, wrapAgentError(err)
			}
			// Close the correlation too. Cancellation ends the engagement with no
			// terminal result to apply, so leaving it claimed would keep the
			// sweeper retrying a cancelled execution indefinitely.
			if err := s.transitionRunner.CloseUnapplied(correlation.ExecutionID, "cancelled"); err != nil {
				return Record{}, err
			}
			record.Status = StatusCanceled
			record.UpdatedAt = nowRFC3339()
			record.FinishedAt = record.UpdatedAt
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
		}
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
