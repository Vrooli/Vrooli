package orchestration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
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

func (l workflowChildLauncher) StartFresh(ctx context.Context, req workflowruntime.ChildRequest) (workflowruntime.ChildState, error) {
	taskID := uuid.NewSHA1(req.AttemptID, []byte("workflow-node-task"))
	task, err := l.o.tasks.Get(ctx, taskID)
	if err != nil {
		return workflowruntime.ChildState{}, err
	}
	if task == nil {
		task = &domain.Task{ID: taskID, Title: fmt.Sprintf("Workflow %s node %s", req.ExecutionID, req.NodeID), Description: "Agent Manager workflow node attempt", ScopePath: ".", ProjectRoot: l.o.config.DefaultProjectRoot, Status: domain.TaskStatusQueued, CreatedBy: "workflow-runtime"}
		if _, err := l.o.CreateTask(ctx, task); err != nil {
			if existing, getErr := l.o.tasks.Get(ctx, taskID); getErr != nil || existing == nil {
				return workflowruntime.ChildState{}, err
			}
		}
	}
	create := CreateRunRequest{TaskID: taskID, Prompt: req.Prompt, ResultSpec: req.ResultSpec, IdempotencyKey: req.IdempotencyKey, Tag: "workflow-" + req.ExecutionID.String() + "-" + req.NodeID}
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
	run, err := l.o.ContinueRun(ctx, ContinueRunRequest{RunID: *req.SourceRunID, Message: req.Prompt, IdempotencyKey: req.IdempotencyKey})
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
	_, _, err := l.o.workflowEngine.Cancel(ctx, id, "parent-cancel/"+id.String(), reason, 0)
	return err
}

func subworkflowState(execution *domain.WorkflowExecution) workflowruntime.SubworkflowState {
	if execution == nil {
		return workflowruntime.SubworkflowState{}
	}
	return workflowruntime.SubworkflowState{ExecutionID: execution.ID, Terminal: execution.Status.Terminal(), Failed: execution.Status != domain.WorkflowExecutionSucceeded, Output: execution.Output, BudgetUsage: execution.BudgetUsage}
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
	execution, err := o.workflowEngine.Start(ctx, revision, req.Input, strings.TrimSpace(req.IdempotencyKey))
	if err != nil {
		return nil, err
	}
	return o.driveWorkflowExecution(ctx, execution.ID)
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
	if err != nil || idempotent {
		return &WorkflowExecutionOperationResult{Execution: execution, Idempotent: idempotent}, err
	}
	attempts, listErr := o.workflowExecutions.ListAttempts(ctx, req.ExecutionID)
	if listErr != nil {
		return nil, listErr
	}
	stoppedRuns, stoppedWorkflows := 0, 0
	var failures []string
	for _, attempt := range attempts {
		if attempt.ChildExecutionID != nil && attempt.Status != domain.WorkflowAttemptCompleted && attempt.Status != domain.WorkflowAttemptFailed {
			if cancelErr := (workflowSubworkflowLauncher{o: o}).Cancel(ctx, *attempt.ChildExecutionID, "parent workflow cancelled"); cancelErr != nil {
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
	execution, err = o.workflowEngine.RecordCancellationDisposition(ctx, req.ExecutionID, stoppedRuns, stoppedWorkflows, failures)
	if err != nil {
		return nil, err
	}
	o.broadcastWorkflowLifecycle(ctx, execution)
	return &WorkflowExecutionOperationResult{Execution: execution}, nil
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

func (o *Orchestrator) driveWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error) {
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
			return before, nil
		}
		latest, err = o.workflowEngine.Advance(ctx, id)
		if err != nil {
			return latest, err
		}
		o.broadcastWorkflowLifecycle(ctx, latest)
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
		if _, advanceErr := o.driveWorkflowExecution(ctx, execution.ID); advanceErr != nil {
			return advanceErr
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
