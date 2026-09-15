package execution

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
	if record.Status == StatusCancelling {
		return Record{}, apierr.Conflict("execution cancellation is pending terminal accounting")
	}
	if record.Status == StatusStarting || record.Status == StatusRunning || record.Status == StatusNeedsReview || record.Status == StatusCompleted {
		return record, nil
	}
	if record.Status == StatusCanceled {
		return Record{}, apierr.BadRequest("cannot start canceled execution")
	}
	if err := s.checkPlanWork(ctx, record.BacklogKind, record.BacklogName); err != nil {
		return Record{}, err
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
	if record.ExecutionMode != firstNonEmpty(item.ExecutionMode, defaultExecutionMode) {
		return Record{}, apierr.Conflict("execution strategy differs from the currently accepted item")
	}
	if record.ApprovalDigest != "" && (item.PlanAcceptance == nil || record.ApprovalDigest != digestStrings(item.PlanAcceptance.SubjectVersion, item.PlanAcceptance.PlanContentHash)) {
		return Record{}, apierr.Conflict("execution was queued under a different accepted work contract")
	}
	if err := s.prepareExecutionGrantLocked(ctx, records, &record, item); err != nil {
		return Record{}, err
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
	if extensions, scopeErr := s.readScopeExtensions(ctx, record, item); scopeErr != nil {
		return Record{}, scopeErr
	} else {
		record.ScopeExtensions = extensions
	}
	// Goal mode is one Agent Manager run: no workflow, no slice cap, no
	// per-session reviewer.
	if record.ExecutionMode == transitions.ExecutionModeGoal {
		return s.launchGoalRun(ctx, records, idx, record, item, planHandle)
	}
	_, err = s.resolveWorkflow("plan.execute")
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	_, ok := s.workflowForStrategy(record.ExecutionMode)
	if !ok {
		return Record{}, apierr.BadRequest("execution strategy %q is not declared", record.ExecutionMode)
	}
	// Persist the resolved Plan Manager execution before starting: the runner's
	// input builder reprojects the frontier from the durable record, so
	// PlanExecutionID has to be readable there and at apply time.
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, apierr.Internal("persist plan-execution record before start: %s", err.Error())
	}
	workflowKey, _ := s.workflowForStrategy(record.ExecutionMode)
	firstRunNodeID := "slice"
	if record.ExecutionMode == transitions.ExecutionModeGoal {
		firstRunNodeID = "goal"
	}
	var executionPreferences *domainpb.ExecutionPreferences
	if preferences := record.ExecutionPreferences; preferences != nil {
		executionPreferences = &domainpb.ExecutionPreferences{PreferredRunner: preferences.PreferredRunner, Model: preferences.Model, Effort: preferences.Effort}
	}
	started, err := s.startTransition(ctx, "plan.execute", record, "process", transitionrunner.PreparedInput{FirstRunNodeID: firstRunNodeID, WorkflowKeyOverride: workflowKey, ExecutionPreferences: executionPreferences})
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

func (s *Service) workflowForStrategy(modeID string) (string, bool) {
	if strings.TrimSpace(modeID) == "" {
		modeID = defaultExecutionMode
	}
	modes, err := s.declaredExecutionModes()
	if err != nil {
		return "", false
	}
	for _, mode := range modes {
		if mode.ID == strings.TrimSpace(modeID) {
			return mode.WorkflowKey, strings.TrimSpace(mode.WorkflowKey) != ""
		}
	}
	return "", false
}

// isDeclaredExecutionMode reports whether a mode id is declared by the
// plan.execute transition. It separates "undeclared" from "declared but has no
// workflow" (goal mode), which the launch path handles differently.
func (s *Service) isDeclaredExecutionMode(modeID string) bool {
	modes, err := s.declaredExecutionModes()
	if err != nil {
		return false
	}
	modeID = strings.TrimSpace(modeID)
	if modeID == "" {
		modeID = defaultExecutionMode
	}
	for _, mode := range modes {
		if mode.ID == modeID {
			return true
		}
	}
	return false
}

// launchGoalRun composes the goal message and creates exactly one Agent Manager
// run for a goal-mode execution. It persists the run id, execution mode, and
// message digest, then returns the record as starting. Reconcile projects the
// terminal_class/stop_reason after the run ends.
func (s *Service) launchGoalRun(ctx context.Context, records []Record, idx int, record Record, item backlogItem, planHandle string) (Record, error) {
	if s.goalRunCreator == nil {
		return Record{}, apierr.Unavailable("goal run creator is not configured")
	}
	input := s.goalMessageInput(ctx, item, record.PlanManagerExecutionID)
	if note := strings.TrimSpace(record.OperatorNote); note != "" {
		input.OperatorNote = note
	}
	message, err := ComposeGoalMessage(input)
	if err != nil {
		var tooLong *GoalMessageTooLongError
		if errors.As(err, &tooLong) {
			return Record{}, apierr.BadRequest("goal_message_too_long: %s", tooLong.Error())
		}
		return Record{}, apierr.BadRequest("compose goal message: %s", err)
	}
	// A fresh-run resume carries the parent's last handoff so the agent continues
	// from durable state rather than re-deriving it.
	if strings.TrimSpace(record.ContinuationOf) != "" && strings.TrimSpace(record.LastHandoff) != "" {
		message += "\n\nEarlier handoff:\n" + record.LastHandoff
		if len([]rune(message)) > GoalPromptMaxChars {
			return Record{}, apierr.BadRequest("goal_prompt_too_long: resume message with handoff is %d characters, over the %d limit", len([]rune(message)), GoalPromptMaxChars)
		}
	}
	runReq := agentmanager.GoalRunRequest{
		ProfileKey:     "swarm-manager/goal-work",
		Prompt:         message,
		Until:          RenderFinishLine(input),
		IdempotencyKey: "goal/" + record.ExecutionID + "/1/1",
		Tag:            "swarm-execution-" + record.ExecutionID,
		// The Agent Manager task scope is the workspace root, the same
		// projectRoot the sliced route hands the workflow snapshot. Agent
		// Manager requires a non-empty scope_path, and the goal message's
		// boundary slot carries the acceptance_allow globs the run may edit.
		ScopePath:   filepath.Dir(s.repoRoot),
		ProjectRoot: filepath.Dir(s.repoRoot),
	}
	if prefs := record.ExecutionPreferences; prefs != nil {
		runReq.PreferredRunner = prefs.PreferredRunner
		runReq.Model = prefs.Model
		runReq.Effort = prefs.Effort
	}
	created, err := s.goalRunCreator.CreateGoalRun(ctx, runReq)
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	record.RunID = created.RunID
	record.TaskID = created.TaskID
	record.GoalMessageDigest = digestStrings(message)
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

// Cancel withdraws execution authority. Workflow-owned plan execution remains
// cancelling until the original owner supplies complete terminal accounting.
func (s *Service) Cancel(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	if record.Cancellation != nil {
		return s.cancelPlanExecutionLocked(ctx, records, idx)
	}
	if record.Status == StatusPending && s.transitionRunner != nil {
		// Submission may have reached the owner while its acknowledgement was
		// lost. Only a proven absent local intent permits immediate cancellation.
		if _, intentErr := s.transitionRunner.GetDispatchIntent("plan.execute", record.ExecutionID); !errors.Is(intentErr, os.ErrNotExist) {
			return s.cancelPlanExecutionLocked(ctx, records, idx)
		}
	}
	// A terminal goal verdict may still hold a reservation when the owner did
	// not emit a usage receipt. Permit the operator to move that record through
	// the normal cancellation/write-off path so a new accepted attempt cannot be
	// stranded behind an unresolvable reservation.
	if record.Status == StatusStarting || record.Status == StatusRunning || record.Status == StatusNeedsReview || record.Status == StatusNeedsAttention {
		correlation, correlationErr := s.transitionCorrelation(record)
		if record.WorkflowGrant != nil || (correlationErr == nil && correlation.TransitionKey == "plan.execute") {
			return s.cancelPlanExecutionLocked(ctx, records, idx)
		}
	}

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
