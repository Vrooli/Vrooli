// This file executes workflow steps through the run orchestration path.
package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowruntime"

	"github.com/google/uuid"
)

type (
	workflowChildLauncher       struct{ o *Orchestrator }
	workflowSubworkflowLauncher struct{ o *Orchestrator }
)

type workflowLifecycleBroadcaster interface {
	BroadcastWorkflowLifecycle(*domain.WorkflowLifecycleEvent)
}

const (
	workflowExecutionEnv  = "VROOLI_WORKFLOW_EXECUTION_ID"
	workflowNodeEnv       = "VROOLI_WORKFLOW_NODE_ID"
	workflowAttemptEnv    = "VROOLI_WORKFLOW_ATTEMPT_ID"
	workflowExperimentEnv = "VROOLI_WORKFLOW_EXPERIMENT_ID"
	workflowVariantEnv    = "VROOLI_WORKFLOW_VARIANT_ID"
	workflowPromptHashEnv = "VROOLI_WORKFLOW_PROMPT_HASH"
)

func (l workflowChildLauncher) StartFresh(ctx context.Context, req workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	taskID := uuid.NewSHA1(req.AttemptID, []byte("workflow-node-task"))
	task, err := l.o.tasks.Get(ctx, taskID)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	if task == nil {
		scopePath := req.ScopePath
		if scopePath == "" {
			scopePath = "."
		}
		task = &domain.Task{ID: taskID, Title: fmt.Sprintf("Workflow %s node %s", req.ExecutionID, req.NodeID), Description: "Agent Manager workflow node attempt", ScopePath: scopePath, ProjectRoot: l.o.config.DefaultProjectRoot, Status: domain.TaskStatusQueued, CreatedBy: "workflow-runtime"}
		if _, err := l.o.CreateTask(ctx, task); err != nil {
			if existing, getErr := l.o.tasks.Get(ctx, taskID); getErr != nil || existing == nil {
				return workflowruntime.ChildState{}, err
			}
		}
	}
	tag := req.Tag
	if tag == "" {
		tag = "workflow-" + req.ExecutionID.String() + "-" + req.NodeID
	}
	create := CreateRunRequest{
		TaskID: taskID, Prompt: req.Prompt, ResultSpec: req.ResultSpec, IdempotencyKey: req.IdempotencyKey, Tag: tag, Force: req.Force,
		Environment: map[string]string{
			workflowExecutionEnv:  req.ExecutionID.String(),
			workflowNodeEnv:       req.NodeID,
			workflowAttemptEnv:    req.AttemptID.String(),
			workflowExperimentEnv: req.ExperimentID,
			workflowVariantEnv:    req.VariantID,
			workflowPromptHashEnv: req.PromptHash,
		},
	}
	if req.MaxTurns > 0 {
		create.MaxTurns = &req.MaxTurns
	}
	if req.Timeout > 0 {
		create.Timeout = &req.Timeout
	}
	if req.ProfileKey != "" {
		profile, err := l.o.profiles.GetByKey(ctx, req.ProfileKey)
		if err != nil {
			return workflowruntime.ChildState{}, err
		}
		if profile == nil {
			return workflowruntime.ChildState{}, domain.NewNotFoundErrorWithID("AgentProfile", req.ProfileKey)
		}
		create.AgentProfileID = &profile.ID
	} else {
		role := req.RoleRef
		create.RoleRef = &role
	}
	run, err := l.o.CreateRun(ctx, create)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	return childStateFromRun(run), nil
}

func (l workflowChildLauncher) Continue(ctx context.Context, req workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	if req.SourceRunID == nil {
		return workflowruntime.ChildState{}, domain.NewValidationError("sourceRunId", "explicit source Run is required")
	}
	continueRequest := ContinueRunRequest{RunID: *req.SourceRunID, Message: req.Prompt, IdempotencyKey: req.IdempotencyKey, ResultSpec: req.ResultSpec}
	if req.MaxTurns > 0 {
		continueRequest.MaxTurns = &req.MaxTurns
	}
	if req.Timeout > 0 {
		continueRequest.Timeout = &req.Timeout
	}
	run, err := l.o.ContinueRun(ctx, continueRequest)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	return childStateFromRun(run), nil
}

func (l workflowChildLauncher) Inspect(ctx context.Context, id uuid.UUID) (workflowruntime.ChildState, error) {
	run, err := l.o.GetRun(ctx, id)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	return childStateFromRun(run), nil
}

func (l workflowChildLauncher) Stop(ctx context.Context, id uuid.UUID) error {
	return l.o.StopRun(ctx, id)
}

func childStateFromRun(run *domain.Run) workflowruntime.ChildState {
	state := workflowruntime.ChildState{RunID: run.ID, ConversationID: run.ConversationID, Result: run.Result}
	if run.Summary != nil {
		state.Turns = run.Summary.TurnsUsed
		state.Tokens = run.Summary.TokensUsed
		state.CostUSD = run.Summary.CostEstimate
	}
	switch run.Status {
	case domain.RunStatusComplete:
		state.Terminal = true
	case domain.RunStatusNeedsReview:
		// The agent turn finished and its result is durable; the pending review
		// concerns applying the run's file changes, which is orthogonal to the
		// workflow node being done. A manual-review profile (e.g. investigation)
		// would otherwise strand its workflow at the run node forever.
		state.Terminal = true
	case domain.RunStatusFailed, domain.RunStatusCancelled:
		state.Terminal = true
		state.Failed = true
	}
	return state
}

func (l workflowSubworkflowLauncher) Start(ctx context.Context, req workflowruntime.SubworkflowRequest) (workflowruntime.SubworkflowState, error) {
	var revision *domain.WorkflowRevision
	var err error
	if strings.TrimSpace(req.Version) == "" {
		revision, err = l.o.workflows.GetActive(ctx, req.Owner, req.WorkflowKey)
	} else {
		var revisions []*domain.WorkflowRevision
		revisions, err = l.o.workflows.List(ctx, req.Owner, req.WorkflowKey, repository.ListFilter{Limit: 1000})
		if err == nil {
			for _, candidate := range revisions {
				if candidate.SemanticVersion == req.Version {
					revision = candidate
					break
				}
			}
		}
	}
	if err != nil {
		return workflowruntime.SubworkflowState{}, err
	}
	if revision == nil {
		return workflowruntime.SubworkflowState{}, domain.NewNotFoundErrorWithID("WorkflowRevision", req.WorkflowKey+"@"+req.Version)
	}
	execution, err := l.o.workflowEngine.StartChild(ctx, revision, req.Input, req.IdempotencyKey, req.ParentExecutionID, req.ParentAttemptID, req.Depth)
	if err != nil {
		return workflowruntime.SubworkflowState{}, err
	}
	execution, err = l.o.driveWorkflowExecution(ctx, execution.ID)
	if err != nil {
		return workflowruntime.SubworkflowState{}, err
	}
	return subworkflowState(execution), nil
}

func (l workflowSubworkflowLauncher) Inspect(ctx context.Context, id uuid.UUID) (workflowruntime.SubworkflowState, error) {
	execution, err := l.o.driveWorkflowExecution(ctx, id)
	if err != nil {
		return workflowruntime.SubworkflowState{}, err
	}
	return subworkflowState(execution), nil
}

func (l workflowSubworkflowLauncher) Cancel(ctx context.Context, id uuid.UUID, reason string) error {
	_, err := l.o.CancelWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: id, IdempotencyKey: "parent-cancel/" + id.String(), Reason: reason})
	return err
}

func subworkflowState(execution *domain.WorkflowExecution) workflowruntime.SubworkflowState {
	if execution == nil {
		return workflowruntime.SubworkflowState{}
	}
	return workflowruntime.SubworkflowState{ExecutionID: execution.ID, Terminal: execution.Status.Terminal(), Failed: execution.Status == domain.WorkflowExecutionFailed, Status: execution.Status, Output: execution.Output, TerminalReason: execution.TerminalReason, BudgetUsage: execution.BudgetUsage}
}

func (o *Orchestrator) StartWorkflowExecution(ctx context.Context, req StartWorkflowExecutionRequest) (*domain.WorkflowExecution, error) {
	if o.workflowEngine == nil {
		return nil, domain.NewConfigInvalidError("workflowEngine", "workflow interpreter is not configured", nil)
	}
	owner, key := strings.TrimSpace(req.Owner), strings.TrimSpace(req.WorkflowKey)
	var revision *domain.WorkflowRevision
	var err error
	if strings.TrimSpace(req.DefinitionDigest) != "" {
		revision, err = o.workflows.GetByDigest(ctx, strings.TrimSpace(req.DefinitionDigest))
	} else {
		revision, err = o.workflows.GetActive(ctx, owner, key)
	}
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, domain.NewNotFoundErrorWithID("WorkflowRevision", key+req.DefinitionDigest)
	}
	if owner != "" && revision.Owner != owner {
		return nil, domain.NewValidationError("owner", "does not own selected workflow revision")
	}
	if err := o.enforceWorkflowTrigger(ctx, revision, req); err != nil {
		return nil, err
	}
	execution, err := o.workflowEngine.Start(ctx, revision, req.Input, strings.TrimSpace(req.IdempotencyKey))
	if err != nil {
		return nil, err
	}
	return o.driveWorkflowExecution(ctx, execution.ID)
}

// TriggerPolicyError is a typed denial that explains the classified initiator,
// configured policy, and resolved workflow caller chain. It is intentionally
// returned before Engine.Start so a denied request creates no execution.
type TriggerPolicyError struct {
	WorkflowKey string
	Initiator   domain.WorkflowInitiator
	Decision    string
	Evidence    []string
}

func (e *TriggerPolicyError) Error() string {
	return fmt.Sprintf("workflow trigger policy denied %s for %s (%s; chain=%s)", e.Initiator, e.WorkflowKey, e.Decision, strings.Join(e.Evidence, " -> "))
}

func (o *Orchestrator) enforceWorkflowTrigger(ctx context.Context, revision *domain.WorkflowRevision, req StartWorkflowExecutionRequest) error {
	initiator := req.Initiator
	if initiator == "" {
		initiator = domain.WorkflowInitiatorProgrammatic
	}
	policy := revision.Definition.Trigger
	evidence := []string{"initiator=" + string(initiator)}
	var verified *IdentityVerifyResult
	if initiator == domain.WorkflowInitiatorAgent {
		var err error
		verified, err = o.VerifyIdentityToken(ctx, strings.TrimSpace(req.IdentityToken))
		if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
			// A missing or unverifiable token cannot prove an agent origin. Keep the
			// documented fail-open behavior, but make it visible to operators.
			evidence = append(evidence, "identity=unverified")
			initiator = domain.WorkflowInitiatorProgrammatic
		}
	}
	if !policy.Allows(initiator) {
		return o.denyWorkflowTrigger(revision.Key, initiator, "initiator_not_allowed", evidence)
	}
	if initiator != domain.WorkflowInitiatorAgent {
		decision := "non_agent"
		if verified == nil && len(evidence) > 1 {
			decision = "identity_unverified"
		}
		o.auditWorkflowTrigger(revision.Key, initiator, "allow", decision, evidence)
		return nil
	}
	evidence = append(evidence, "run="+verified.Claims.RunID.String())
	callerID, err := o.workflowExecutions.ExecutionIDForRun(ctx, verified.Claims.RunID)
	if err != nil {
		return fmt.Errorf("resolve workflow caller identity: %w", err)
	}
	if callerID == uuid.Nil {
		o.auditWorkflowTrigger(revision.Key, initiator, "allow", "run_not_in_workflow", evidence)
		return nil
	}
	chain, err := o.workflowCallerChain(ctx, callerID)
	if err != nil {
		return err
	}
	count := 0
	for _, item := range chain {
		evidence = append(evidence, item.WorkflowKey+"/"+item.ID.String())
		if item.WorkflowKey == revision.Key {
			count++
		}
	}
	if count == 0 {
		o.auditWorkflowTrigger(revision.Key, initiator, "allow", "different_workflow", evidence)
		return nil
	}
	if policy.SelfTriggerMode() == domain.WorkflowSelfTriggerDeny {
		return o.denyWorkflowTrigger(revision.Key, initiator, "self_trigger_denied", evidence)
	}
	if count >= policy.SelfTrigger.MaxDepth {
		return o.denyWorkflowTrigger(revision.Key, initiator, "self_trigger_depth_reached", evidence)
	}
	o.auditWorkflowTrigger(revision.Key, initiator, "allow", "self_trigger_within_depth", evidence)
	return nil
}

func (o *Orchestrator) workflowCallerChain(ctx context.Context, id uuid.UUID) ([]*domain.WorkflowExecution, error) {
	chain := make([]*domain.WorkflowExecution, 0, 4)
	seen := map[uuid.UUID]bool{}
	for id != uuid.Nil {
		if seen[id] {
			return nil, fmt.Errorf("workflow caller chain contains cycle at %s", id)
		}
		seen[id] = true
		execution, err := o.workflowExecutions.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if execution == nil {
			return nil, fmt.Errorf("workflow caller execution %s is unavailable", id)
		}
		chain = append(chain, execution)
		if execution.ParentExecutionID == nil {
			break
		}
		id = *execution.ParentExecutionID
	}
	return chain, nil
}

func (o *Orchestrator) denyWorkflowTrigger(workflowKey string, initiator domain.WorkflowInitiator, decision string, evidence []string) error {
	o.auditWorkflowTrigger(workflowKey, initiator, "deny", decision, evidence)
	return &TriggerPolicyError{WorkflowKey: workflowKey, Initiator: initiator, Decision: decision, Evidence: evidence}
}

func (o *Orchestrator) auditWorkflowTrigger(workflowKey string, initiator domain.WorkflowInitiator, outcome, decision string, evidence []string) {
	slog.Info("workflow trigger policy decision", "workflow_key", workflowKey, "initiator", initiator, "outcome", outcome, "decision", decision, "evidence", evidence)
}

func (o *Orchestrator) GetWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	if o.workflowExecutions == nil {
		return nil, domain.NewConfigInvalidError("workflowExecutionRepository", "not configured", nil)
	}
	execution, err := o.workflowExecutions.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if execution == nil {
		return nil, domain.NewNotFoundError("WorkflowExecution", id)
	}
	return execution, nil
}

func (o *Orchestrator) ListWorkflowExecutions(ctx context.Context, req ListWorkflowExecutionsRequest) ([]*domain.WorkflowExecution, error) {
	if o.workflowExecutions == nil {
		return nil, domain.NewStateError("WorkflowExecution", "unavailable", "list", "workflow execution repository is not configured")
	}
	return o.workflowExecutions.List(ctx, repository.WorkflowExecutionListFilter{
		ListFilter:  repository.ListFilter{Limit: req.Limit, Offset: req.Offset},
		Owner:       strings.TrimSpace(req.Owner),
		WorkflowKey: strings.TrimSpace(req.WorkflowKey),
		Status:      req.Status,
	})
}

func (o *Orchestrator) AdvanceWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	if o.workflowEngine == nil {
		return nil, domain.NewConfigInvalidError("workflowEngine", "not configured", nil)
	}
	return o.driveWorkflowExecution(ctx, id)
}

// NudgeDrive is the WorkflowNudger's work function: an idempotent drive of the
// execution that discards the returned snapshot and surfaces only the error.
func (o *Orchestrator) NudgeDrive(ctx context.Context, id uuid.UUID) error {
	_, err := o.AdvanceWorkflowExecution(ctx, id)
	return err
}

// WaitWorkflowExecutionResult is the typed result of a blocking wait.
type WaitWorkflowExecutionResult struct {
	Execution *domain.WorkflowExecution
	TimedOut  bool
}

// WaitWorkflowExecution blocks until the execution is terminal or the deadline
// passes, mirroring the test-genie runs-wait contract. It is event-driven (no
// ticker): it subscribes to the terminal notifier, re-reads durable state to
// close the subscribe/settle race, then blocks on its wake channel. timeout <= 0
// blocks until terminal (or the caller's ctx is cancelled). Cancelling the
// waiter NEVER cancels the execution — the wait only reads execution state; the
// execution is driven independently by the engine, the completion nudge, and
// the reconciler backstop.
func (o *Orchestrator) WaitWorkflowExecution(ctx context.Context, id uuid.UUID, timeout time.Duration) (*WaitWorkflowExecutionResult, error) {
	waitCtx := ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	for {
		execution, err := o.GetWorkflowExecution(ctx, id)
		if err != nil {
			return nil, err
		}
		if execution.Status.Terminal() {
			return &WaitWorkflowExecutionResult{Execution: execution}, nil
		}

		// Subscribe BEFORE the settle-gap re-read so a terminal transition
		// committing between the read above and here still wakes us.
		wake := o.workflowWaiters.subscribe(id)
		execution, err = o.GetWorkflowExecution(ctx, id)
		if err != nil {
			o.workflowWaiters.unsubscribe(id, wake)
			return nil, err
		}
		if execution.Status.Terminal() {
			o.workflowWaiters.unsubscribe(id, wake)
			return &WaitWorkflowExecutionResult{Execution: execution}, nil
		}

		select {
		case <-wake:
			// Settled (or spuriously notified) — loop to re-read the terminal state.
		case <-waitCtx.Done():
			o.workflowWaiters.unsubscribe(id, wake)
			// A deadline the caller did not set (our timeout) is a timed-out
			// wait, not an error: the execution keeps running.
			if timeout > 0 && errors.Is(waitCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
				current, getErr := o.GetWorkflowExecution(ctx, id)
				if getErr != nil {
					return nil, getErr
				}
				return &WaitWorkflowExecutionResult{Execution: current, TimedOut: true}, nil
			}
			return nil, waitCtx.Err()
		}
	}
}

func (o *Orchestrator) GetWorkflowExecutionTrace(ctx context.Context, id uuid.UUID, afterSequence int64, limit int) (*WorkflowExecutionTrace, error) {
	execution, err := o.GetWorkflowExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 200
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, id)
	if err != nil {
		return nil, err
	}
	if revision, revisionErr := o.workflows.GetByDigest(ctx, execution.DefinitionDigest); revisionErr == nil && revision != nil {
		nodes := make(map[string]domain.WorkflowNode, len(revision.Definition.Nodes))
		for _, node := range revision.Definition.Nodes {
			nodes[node.ID] = node
		}
		identities := make(map[uuid.UUID]string, len(attempts))
		for _, attempt := range attempts {
			node := nodes[attempt.NodeID]
			switch {
			case node.Run != nil && node.Run.ProfileKey != "":
				attempt.ProfileIdentity = "profile:" + node.Run.ProfileKey
			case node.Run != nil && node.Run.RoleRef != "":
				attempt.ProfileIdentity = "role:" + node.Run.RoleRef
			case node.Child != nil:
				attempt.ProfileIdentity = "workflow:" + node.Child.WorkflowKey
			}
			identities[attempt.ID] = attempt.ProfileIdentity
		}
		for _, attempt := range attempts {
			if attempt.Strategy == domain.WorkflowAttemptContinue && attempt.SourceAttemptID != nil {
				attempt.ProfileIdentity = identities[*attempt.SourceAttemptID]
			}
		}
	}
	journal, err := o.workflowExecutions.ListJournal(ctx, id, afterSequence, limit)
	if err != nil {
		return nil, err
	}
	return &WorkflowExecutionTrace{Execution: execution, Attempts: attempts, Journal: journal}, nil
}

// ListWorkflowExecutionRuns is the bounded node-to-Run projection for callers
// that need dispatched Run identities without reimplementing an attempt scan.
func (o *Orchestrator) ListWorkflowExecutionRuns(ctx context.Context, id uuid.UUID) ([]*domain.WorkflowNodeAttempt, error) {
	trace, err := o.GetWorkflowExecutionTrace(ctx, id, 0, 1)
	if err != nil {
		return nil, err
	}
	return trace.Attempts, nil
}

func (o *Orchestrator) SignalWorkflowExecution(ctx context.Context, req WorkflowExecutionSignalRequest) (*WorkflowExecutionOperationResult, error) {
	execution, idempotent, err := o.workflowEngine.Signal(ctx, req.ExecutionID, strings.TrimSpace(req.Signal), req.Payload, strings.TrimSpace(req.IdempotencyKey), req.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	if !idempotent {
		execution, err = o.driveWorkflowExecution(ctx, req.ExecutionID)
		if err != nil {
			return nil, err
		}
	}
	return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, nil
}

func (o *Orchestrator) CancelWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error) {
	execution, idempotent, err := o.workflowEngine.Cancel(ctx, req.ExecutionID, strings.TrimSpace(req.IdempotencyKey), strings.TrimSpace(req.Reason), req.ExpectedVersion)
	if err != nil {
		return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, err
	}
	if execution.Status == domain.WorkflowExecutionCancelled {
		o.onWorkflowExecutionSettled(execution)
		return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, nil
	}
	execution, err = o.cleanupWorkflowChildren(ctx, req.ExecutionID, "parent workflow cancelled")
	if err != nil {
		return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, err
	}
	o.broadcastWorkflowLifecycle(ctx, execution)
	o.onWorkflowExecutionSettled(execution)
	return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, nil
}

func (o *Orchestrator) cleanupWorkflowChildren(ctx context.Context, executionID uuid.UUID, reason string) (*domain.WorkflowExecution, error) {
	execution, getErr := o.workflowExecutions.Get(ctx, executionID)
	if getErr != nil || execution == nil {
		return execution, getErr
	}
	journal, journalErr := o.workflowExecutions.ListJournal(ctx, executionID, 0, 0)
	if journalErr != nil {
		return nil, journalErr
	}
	for _, entry := range journal {
		if entry.Kind != domain.WorkflowJournalCleanup {
			continue
		}
		var disposition struct {
			Retry int `json:"retry"`
		}
		if json.Unmarshal(entry.Payload, &disposition) == nil && disposition.Retry == execution.BudgetUsage.Retries {
			return execution, nil
		}
	}
	attempts, listErr := o.workflowExecutions.ListAttempts(ctx, executionID)
	if listErr != nil {
		return nil, listErr
	}
	stoppedRuns, stoppedWorkflows := 0, 0
	var failures []string
	for _, attempt := range attempts {
		if attempt.ChildExecutionID != nil && attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			if cancelErr := (workflowSubworkflowLauncher{o: o}).Cancel(ctx, *attempt.ChildExecutionID, reason); cancelErr != nil {
				failures = append(failures, "child workflow "+attempt.ChildExecutionID.String()+": "+cancelErr.Error())
			} else {
				stoppedWorkflows++
			}
		}
		if attempt.RunID == nil || attempt.Status == domain.WorkflowAttemptCompleted || attempt.Status == domain.WorkflowAttemptFailed {
			continue
		}
		if stopErr := o.StopRun(ctx, *attempt.RunID); stopErr != nil {
			failures = append(failures, "Run "+attempt.RunID.String()+": "+stopErr.Error())
		} else {
			stoppedRuns++
		}
	}
	return o.workflowEngine.RecordCleanupDisposition(ctx, executionID, stoppedRuns, stoppedWorkflows, failures)
}

func (o *Orchestrator) RetryWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error) {
	execution, idempotent, err := o.workflowEngine.Retry(ctx, req.ExecutionID, strings.TrimSpace(req.IdempotencyKey), req.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	if !idempotent {
		execution, err = o.driveWorkflowExecution(ctx, req.ExecutionID)
	}
	return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, err
}

func (o *Orchestrator) ResumeWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error) {
	execution, idempotent, err := o.workflowEngine.Resume(ctx, req.ExecutionID, strings.TrimSpace(req.IdempotencyKey), req.ExpectedVersion)
	if err != nil {
		return nil, err
	}
	if !idempotent {
		execution, err = o.driveWorkflowExecution(ctx, req.ExecutionID)
	}
	return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, err
}

// driveWorkflowExecution runs the pull loop and then fires completion-driven
// side effects when the execution has settled terminal: it wakes any blocking
// WaitWorkflowExecution callers and nudges the parent execution of a finished
// child workflow so parent progress needs no polling. The loop itself is
// unchanged crash-safe; the settle hook is purely a notification.
func (o *Orchestrator) driveWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	execution, err := o.driveWorkflowExecutionLoop(ctx, id)
	o.onWorkflowExecutionSettled(execution)
	return execution, err
}

// FailWorkflowExecution records an infrastructure failure outside the engine
// pull loop (for example, a recovered workflow-nudger panic). The engine owns
// the CAS transition so this cannot clobber a concurrent terminal advance.
func (o *Orchestrator) FailWorkflowExecution(ctx context.Context, id uuid.UUID, code, message string) (*domain.WorkflowExecution, error) {
	if o.workflowEngine == nil {
		return nil, domain.NewStateError("workflow engine", "unavailable", "fail", "workflow execution is not configured")
	}
	execution, err := o.workflowEngine.Fail(ctx, id, code, message)
	o.onWorkflowExecutionSettled(execution)
	return execution, err
}

// ReconcileUnarmedWorkflowWaits finds waiting executions whose latest durable
// wait intent has no deadline. Such a state cannot be woken by a timeout and
// therefore needs bounded operator visibility and eventual terminal cleanup.
func (o *Orchestrator) ReconcileUnarmedWorkflowWaits(ctx context.Context, warningAfter, failAfter time.Duration) error {
	if o.workflowExecutions == nil || o.workflowEngine == nil || warningAfter <= 0 || failAfter <= warningAfter {
		return nil
	}
	waiting, err := o.workflowExecutions.List(ctx, repository.WorkflowExecutionListFilter{Status: domain.WorkflowExecutionWaiting, ListFilter: repository.ListFilter{Limit: 200}})
	if err != nil {
		return err
	}
	now := o.now()
	for _, execution := range waiting {
		journal, journalErr := o.workflowExecutions.ListJournal(ctx, execution.ID, 0, 0)
		if journalErr != nil || workflowWaitHasDeadline(journal) {
			if journalErr != nil {
				obs.Component("workflow-liveness").Warn("waiting workflow journal read failed", "executionId", execution.ID.String(), obs.KeyError, journalErr.Error())
			}
			continue
		}
		age := now.Sub(execution.UpdatedAt)
		if age >= failAfter {
			if _, failErr := o.FailWorkflowExecution(ctx, execution.ID, "unarmed_wait_reaped", fmt.Sprintf("workflow remained waiting without an armed timeout for %v", age.Round(time.Second))); failErr != nil && !errors.Is(failErr, workflowruntime.ErrConcurrentAdvance) {
				obs.Component("workflow-liveness").Warn("unarmed wait reap failed", "executionId", execution.ID.String(), obs.KeyError, failErr.Error())
			}
			continue
		}
		if age >= warningAfter {
			if _, diagnosticErr := o.workflowEngine.RecordDiagnostic(ctx, execution.ID, "unarmed_wait", fmt.Sprintf("workflow waiting without an armed timeout for %v", age.Round(time.Second))); diagnosticErr != nil && !errors.Is(diagnosticErr, workflowruntime.ErrConcurrentAdvance) {
				obs.Component("workflow-liveness").Warn("unarmed wait diagnostic failed", "executionId", execution.ID.String(), obs.KeyError, diagnosticErr.Error())
			}
		}
	}
	return nil
}

func workflowWaitHasDeadline(journal []*domain.WorkflowJournalEntry) bool {
	for i := len(journal) - 1; i >= 0; i-- {
		entry := journal[i]
		if entry.Kind != domain.WorkflowJournalWait {
			continue
		}
		var intent struct {
			Deadline time.Time `json:"deadline"`
		}
		return json.Unmarshal(entry.Payload, &intent) == nil && !intent.Deadline.IsZero()
	}
	return false
}

// onWorkflowExecutionSettled fires the completion-nudge notifications for an
// execution that has reached a terminal status. It is idempotent (a second
// call finds no waiters and re-enqueues a deduped parent nudge) and nil-safe.
func (o *Orchestrator) onWorkflowExecutionSettled(execution *domain.WorkflowExecution) {
	if execution == nil || !execution.Status.Terminal() {
		return
	}
	if o.workflowWaiters != nil {
		o.workflowWaiters.notify(execution.ID)
	}
	if execution.ParentExecutionID != nil && o.workflowNudger != nil {
		o.workflowNudger.Enqueue(*execution.ParentExecutionID)
	}
	o.attributeExperimentForExecution(context.Background(), execution)
}

// attributeExperimentForExecution emits primary evidence only after the
// workflow settles. It joins each treatment assignment to the declared
// evaluator node's structured verdict; treatment completion is guardrail data,
// never a quality metric.
func (o *Orchestrator) attributeExperimentForExecution(ctx context.Context, execution *domain.WorkflowExecution) {
	outcomeClient, ok := o.promptClient.(promptmanager.OutcomeClient)
	if !ok || outcomeClient == nil || execution == nil || o.workflowExecutions == nil || o.workflows == nil {
		return
	}
	revision, err := o.workflows.GetByDigest(ctx, execution.DefinitionDigest)
	if err != nil || revision == nil || revision.Definition.ExperimentEvaluator == nil {
		return
	}
	contract := revision.Definition.ExperimentEvaluator
	attempts, err := o.workflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		return
	}
	var evaluatorAttempt *domain.WorkflowNodeAttempt
	for _, attempt := range attempts {
		if attempt.NodeID == contract.EvaluatorNodeID && attempt.Status == domain.WorkflowAttemptCompleted && attempt.RunID != nil {
			evaluatorAttempt = attempt
		}
	}
	if evaluatorAttempt == nil {
		return
	}
	evaluatorRun, err := o.GetRun(ctx, *evaluatorAttempt.RunID)
	if err != nil || evaluatorRun == nil || evaluatorRun.Result == nil || evaluatorRun.Result.Structured == nil {
		return
	}
	verdict, ok := workflowVerdict(evaluatorRun.Result.Structured.Value, contract.VerdictPointer)
	if !ok {
		verdict = ""
	}
	success := false
	for _, allowed := range contract.SuccessVerdicts {
		if verdict == allowed {
			success = true
			break
		}
	}
	for _, treatment := range attempts {
		if treatment.ExperimentID == "" || treatment.VariantID == "" || !containsWorkflowNode(contract.TreatmentNodeIDs, treatment.NodeID) {
			continue
		}
		guardrail := map[string]any{"status": "unknown", "tokensUsed": 0}
		if treatment.RunID != nil {
			if run, getErr := o.GetRun(ctx, *treatment.RunID); getErr == nil && run != nil {
				guardrail["status"] = string(run.Status)
				if run.Summary != nil {
					guardrail["tokensUsed"] = run.Summary.TokensUsed
				}
			}
		}
		status := outcomeStatus(evaluatorRun.Result.Structured, verdict)
		var successPointer *bool
		if status == "complete" {
			successPointer = &success
		}
		data, marshalErr := json.Marshal(map[string]any{"guardrail": guardrail})
		if marshalErr != nil {
			continue
		}
		assignmentID := "workflow-assignment/" + execution.ID.String() + "/node/" + treatment.NodeID
		controlled := &promptmanager.ControlledExperimentOutcome{AssignmentID: assignmentID, ExecutionID: execution.ID.String(), EvaluatorAttemptID: evaluatorAttempt.ID.String(), EvaluatorRunID: evaluatorAttempt.RunID.String(), Verdict: verdict, Success: successPointer, OutcomeStatus: status, RubricHash: contract.RubricHash, EvaluatorPromptHash: contract.EvaluatorPromptHash, StructuredSchemaHash: evaluatorRun.Result.Structured.SchemaDigest}
		if err := outcomeClient.RecordExperimentOutcome(ctx, treatment.ExperimentID, promptmanager.ExperimentOutcome{IdempotencyKey: "workflow-evaluator-outcome/" + execution.ID.String() + "/" + treatment.ID.String(), VariantID: treatment.VariantID, Source: "agent-manager-evaluator", Data: data, Controlled: controlled}); err != nil {
			obs.Component("experiment-attribution").Warn("record evaluator outcome failed", obs.KeyError, err.Error())
		}
	}
}

func containsWorkflowNode(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func outcomeStatus(result *domain.StructuredResult, verdict string) string {
	if result == nil || result.Status != domain.StructuredResultSuccess || verdict == "" {
		return "incomplete"
	}
	return "complete"
}

// workflowVerdict implements RFC 6901 traversal for the evaluator's declared
// pointer and only accepts a scalar string verdict.
func workflowVerdict(raw json.RawMessage, pointer string) (string, bool) {
	if pointer == "" || pointer[0] != '/' || len(raw) == 0 {
		return "", false
	}
	var current any
	if json.Unmarshal(raw, &current) != nil {
		return "", false
	}
	for _, segment := range strings.Split(pointer[1:], "/") {
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		switch typed := current.(type) {
		case map[string]any:
			current = typed[segment]
		case []any:
			index := -1
			if _, err := fmt.Sscanf(segment, "%d", &index); err != nil || index < 0 || index >= len(typed) {
				return "", false
			}
			current = typed[index]
		default:
			return "", false
		}
		if current == nil {
			return "", false
		}
	}
	verdict, ok := current.(string)
	return verdict, ok
}

// nudgeWorkflowForRun enqueues an idempotent drive of the workflow execution
// that owns a just-terminal run. Non-workflow runs resolve to uuid.Nil and are
// ignored. Called from the run-terminal transition points in the executors.
func (o *Orchestrator) nudgeWorkflowForRun(runID uuid.UUID) {
	if o.workflowNudger == nil || o.workflowExecutions == nil {
		return
	}
	executionID, err := o.workflowExecutions.ExecutionIDForRun(context.Background(), runID)
	if err != nil {
		obs.Component("workflow-nudge").Warn("resolve owning execution for terminal run failed",
			obs.KeyRunID, runID.String(),
			obs.KeyError, err.Error())
		return
	}
	if executionID == uuid.Nil {
		return
	}
	o.workflowNudger.Enqueue(executionID)
}

func (o *Orchestrator) driveWorkflowExecutionLoop(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
	var latest *domain.WorkflowExecution
	for range 32 {
		before, err := o.workflowExecutions.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if before == nil {
			return nil, domain.NewNotFoundError("WorkflowExecution", id)
		}
		if before.Status.Terminal() {
			if before.Status == domain.WorkflowExecutionFailed || before.Status == domain.WorkflowExecutionBudgetExhausted || before.Status == domain.WorkflowExecutionCancelled {
				return o.cleanupWorkflowChildren(ctx, id, "parent workflow terminated")
			}
			return before, nil
		}
		latest, err = o.workflowEngine.Advance(ctx, id)
		if err != nil {
			return latest, err
		}
		o.broadcastWorkflowLifecycle(ctx, latest)
		if latest.Status == domain.WorkflowExecutionFailed || latest.Status == domain.WorkflowExecutionBudgetExhausted || latest.Status == domain.WorkflowExecutionCancelled {
			return o.cleanupWorkflowChildren(ctx, id, "parent workflow terminated")
		}
		if latest.Status.Terminal() || latest.Version == before.Version {
			return latest, nil
		}
	}
	return latest, domain.NewStateError("WorkflowExecution", "running", "advance", "advance step limit reached")
}

func (o *Orchestrator) broadcastWorkflowLifecycle(ctx context.Context, execution *domain.WorkflowExecution) {
	broadcaster, ok := o.broadcaster.(workflowLifecycleBroadcaster)
	if !ok || execution == nil {
		return
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		return
	}
	journal, err := o.workflowExecutions.ListJournal(ctx, execution.ID, 0, 1000)
	if err != nil || len(journal) == 0 {
		return
	}
	entry := journal[len(journal)-1]
	event := &domain.WorkflowLifecycleEvent{ExecutionID: execution.ID, DefinitionDigest: execution.DefinitionDigest, Status: execution.Status, NodeID: execution.CurrentNodeID, JournalSequence: entry.Sequence, JournalKind: entry.Kind, BudgetUsage: execution.BudgetUsage, TerminalReason: execution.TerminalReason}
	digest := sha256.Sum256(entry.Payload)
	event.JournalPayloadDigest = fmt.Sprintf("sha256:%x", digest[:])
	if len(attempts) > 0 {
		attempt := attempts[len(attempts)-1]
		event.NodeID, event.Strategy, event.RunID, event.ConversationID, event.SourceAttemptID = attempt.NodeID, attempt.Strategy, attempt.RunID, attempt.ConversationID, attempt.SourceAttemptID
		if revision, getErr := o.workflows.GetByDigest(ctx, execution.DefinitionDigest); getErr == nil && revision != nil {
			for _, node := range revision.Definition.Nodes {
				if node.ID == attempt.NodeID && node.Run != nil {
					event.ProfileIdentity = node.Run.ProfileKey
					if event.ProfileIdentity == "" {
						event.ProfileIdentity = node.Run.RoleRef
					}
					break
				}
			}
		}
	}
	broadcaster.BroadcastWorkflowLifecycle(event)
}

func (o *Orchestrator) RecoverWorkflowExecutions(ctx context.Context) error {
	if o.workflowExecutions == nil {
		return nil
	}
	executions, err := o.workflowExecutions.ListRecoverable(ctx, 1000)
	if err != nil {
		return err
	}
	for _, execution := range executions {
		var advanceErr error
		if execution.Status == domain.WorkflowExecutionCancelling || execution.Status == domain.WorkflowExecutionFailed || execution.Status == domain.WorkflowExecutionBudgetExhausted || execution.Status == domain.WorkflowExecutionCancelled {
			_, advanceErr = o.cleanupWorkflowChildren(ctx, execution.ID, "workflow recovery cleanup")
		} else {
			_, advanceErr = o.driveWorkflowExecution(ctx, execution.ID)
		}
		if advanceErr != nil {
			if errors.Is(advanceErr, workflowruntime.ErrConcurrentAdvance) {
				obs.Component("workflow-recovery").Debug("workflow recovery raced a concurrent advance", "executionId", execution.ID.String())
				continue
			}
			obs.Component("workflow-recovery").Warn("workflow recovery failed", "executionId", execution.ID.String(), obs.KeyError, advanceErr.Error())
		}
	}
	return nil
}

func (o *Orchestrator) SimulateWorkflow(ctx context.Context, req SimulateWorkflowRequest) (*WorkflowSimulation, error) {
	var revision *domain.WorkflowRevision
	var err error
	if strings.TrimSpace(req.DefinitionDigest) != "" {
		revision, err = o.workflows.GetByDigest(ctx, strings.TrimSpace(req.DefinitionDigest))
	} else {
		revision, err = o.workflows.GetActive(ctx, strings.TrimSpace(req.Owner), strings.TrimSpace(req.WorkflowKey))
	}
	if err != nil {
		return nil, err
	}
	if revision == nil {
		return nil, domain.NewNotFoundErrorWithID("WorkflowRevision", req.WorkflowKey+req.DefinitionDigest)
	}
	result := &WorkflowSimulation{Valid: true, DefinitionDigest: revision.Digest}
	if err := structuredresult.ValidateValue(revision.Definition.InputSchema, req.Input); err != nil {
		result.Valid = false
		result.Diagnostics = append(result.Diagnostics, domain.WorkflowDiagnostic{Code: "input_invalid", Path: "input", Message: err.Error()})
	}
	for _, node := range revision.Definition.Nodes {
		plan := WorkflowNodePlan{NodeID: node.ID, Kind: node.Kind}
		if node.Run != nil {
			plan.ExecutionStrategy = "fresh_run"
			plan.ProfileKey = node.Run.ProfileKey
			plan.RoleRef = node.Run.RoleRef
		}
		if node.Continue != nil {
			plan.ExecutionStrategy = "continue"
			plan.ContinuationSource = node.Continue.ConversationFromNode
		}
		if node.Child != nil {
			plan.ExecutionStrategy, plan.ChildWorkflowKey, plan.ChildWorkflowVersion = "child_workflow", node.Child.WorkflowKey, node.Child.Version
		}
		if node.Wait != nil {
			plan.WaitSignal, plan.WaitTimeoutSeconds = node.Wait.Signal, node.Wait.TimeoutSeconds
		}
		if node.Join != nil {
			plan.JoinStrategy, plan.JoinQuorum = node.Join.Strategy, node.Join.Quorum
		}
		if node.Branch != nil {
			plan.Parallel = node.Branch.Parallel
		}
		if node.End != nil {
			result.PossibleTerminalNodes = append(result.PossibleTerminalNodes, node.ID)
		}
		result.Nodes = append(result.Nodes, plan)
	}
	return result, nil
}
