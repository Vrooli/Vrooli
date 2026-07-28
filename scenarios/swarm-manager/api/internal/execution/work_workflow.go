package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	"google.golang.org/protobuf/types/known/structpb"
)

type workWorkflowSnapshot struct {
	EntityVersion  string
	FrontierDigest string
	Input          *structpb.Value
	WorkflowKey    string
}

// buildWorkWorkflowSnapshot creates the consumer-owned immutable boundary for
// one follow-up or correction. It deliberately contains domain facts rather
// than a rendered prompt; the declaration owns the prompt and result contract.
func buildWorkWorkflowSnapshot(item backlogItem, record, parent Record, workType, note string) (workWorkflowSnapshot, error) {
	snapshot := map[string]any{
		"item":                 item,
		"parentExecutionId":    parent.ExecutionID,
		"parentStatus":         parent.Status,
		"followUpType":         workType,
		"operatorNote":         strings.TrimSpace(note),
		"finalizationFeedback": buildFinalizationFeedback(parent.Finalization),
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return workWorkflowSnapshot{}, fmt.Errorf("encode work snapshot: %w", err)
	}
	// Convert through JSON so protobuf's dynamic value sees only the JSON data
	// model, never Go-only aliases such as backlogItem or Status.
	var jsonSnapshot map[string]any
	if err := json.Unmarshal(encoded, &jsonSnapshot); err != nil {
		return workWorkflowSnapshot{}, fmt.Errorf("decode work snapshot: %w", err)
	}
	version := digestStrings(item.Kind, item.Name, record.ExecutionID, string(encoded))
	input, err := structpb.NewValue(map[string]any{
		"entity":   map[string]any{"kind": item.Kind, "name": item.Name, "executionId": record.ExecutionID, "version": version},
		"snapshot": jsonSnapshot,
	})
	if err != nil {
		return workWorkflowSnapshot{}, fmt.Errorf("build work workflow input: %w", err)
	}
	return workWorkflowSnapshot{EntityVersion: version, FrontierDigest: digestStrings(string(encoded)), Input: input}, nil
}

func workTransitionKey(workType string) string {
	if workType == "fixup" {
		return "work.correct"
	}
	return "work.follow_up"
}

// startWorkWorkflow starts one correction or follow-up through the runner's
// registered input builder. The record must already be persisted with its
// OperatorNote: the builder reads the durable record, so an unsaved record
// would produce a different snapshot than the apply-time rebuild.
func (s *Service) startWorkWorkflow(ctx context.Context, record Record, workType string) (agentmanager.WorkflowStart, workWorkflowSnapshot, error) {
	if s.transitionRunner == nil {
		return agentmanager.WorkflowStart{}, workWorkflowSnapshot{}, agentmanager.ErrNotAvailable
	}
	started, err := s.transitionRunner.StartWith(ctx, workTransitionKey(workType), record.ExecutionID, transitionrunner.PreparedInput{FirstRunNodeID: map[bool]string{true: "correct", false: "follow_up"}[workType == "fixup"], Activity: &transitionrunner.Activity{OwnerType: "backlog", OwnerKind: record.BacklogKind, OwnerName: record.BacklogName, Purpose: workType}})
	if err != nil {
		return agentmanager.WorkflowStart{}, workWorkflowSnapshot{}, err
	}
	snapshot := workWorkflowSnapshot{EntityVersion: started.EntityVersion, FrontierDigest: started.FrontierDigest, WorkflowKey: started.WorkflowKey}
	start := agentmanager.WorkflowStart{ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest}
	if len(started.Attempts) > 0 {
		start.RunID = started.Attempts[0].RunID
	}
	return start, snapshot, nil
}

func (s *Service) resolveWorkflow(transitionKey string) (transitions.Locator, error) {
	workflow, err := s.transitionRegistry.ResolveWorkflow(transitionKey)
	if err != nil {
		return transitions.Locator{}, fmt.Errorf("resolve %s workflow: %w", transitionKey, err)
	}
	return workflow, nil
}

// isTransitionWorkflow keeps Agent Manager workflow identifiers private to the
// transition registry. Execution behavior branches on domain transitions, not
// copied workflow-key literals.
func (s *Service) isTransitionWorkflow(workflowKey, transitionKey string) bool {
	workflow, err := s.resolveWorkflow(transitionKey)
	return err == nil && workflow.Key == workflowKey
}

// ApplyWorkWorkflow applies one terminal typed result exactly once. No polling
// worker reads this workflow: callers explicitly cross this domain boundary.
func (s *Service) ApplyWorkWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	if s.transitionRunner == nil {
		return PhasedPlanApplyResult{}, agentmanager.ErrNotAvailable
	}
	workflowExecutionID, err := s.transitionExecutionID(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	before, err := s.transitionRunner.GetCorrelation(workflowExecutionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	correlation, err := s.transitionRunner.ApplyExecution(ctx, workflowExecutionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	if correlation.TransitionKey != "work.follow_up" && correlation.TransitionKey != "work.correct" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a work workflow")
	}
	record, err := s.Get(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	return PhasedPlanApplyResult{Record: record, Idempotent: before.ApplyState == transitionrun.ApplyStateComplete && correlation.AppliedTime != ""}, nil
}

func (s *Service) transitionExecutionID(ctx context.Context, executionID string) (string, error) {
	if _, err := s.transitionRunner.GetCorrelation(executionID); err == nil {
		return executionID, nil
	}
	record, err := s.Get(ctx, executionID)
	if err != nil {
		return "", err
	}
	if record.AgentWorkflowExecutionID == "" {
		return "", fmt.Errorf("execution %q has no transition workflow", executionID)
	}
	return record.AgentWorkflowExecutionID, nil
}

// SetTransitionRunner installs the shared workflow lifecycle owner.
func (s *Service) SetTransitionRunner(runner *transitionrunner.Runner) { s.transitionRunner = runner }

// RegisterTransitionAdapter exposes correction and follow-up through the
// runner. Execution retains only its domain-state projection.
//
// The registered input builders must NOT take s.mu. The runner calls them from
// inside StartWith, and the start paths that reach StartWith already hold the
// lock — taking it again self-deadlocks. They are pure reads of durable store
// state, so the snapshot they produce needs no additional synchronization.
func (s *Service) RegisterTransitionAdapter(registrar transitionrunner.Registrar) {
	registrar.RegisterInput("work.correct", s.buildWorkTransitionInput)
	registrar.RegisterInput("work.follow_up", s.buildWorkTransitionInput)
	registrar.RegisterInput("scenario.spec_sync", s.buildSpecSyncTransitionInput)
	registrar.RegisterInput("plan.execute", s.buildPlanExecuteTransitionInput)
	registrar.RegisterApply("apply_correction_outcome", s.applyWorkTransition)
	registrar.RegisterApply("apply_follow_up", s.applyWorkTransition)
	registrar.RegisterApply("apply_scenario_spec_sync", s.applySpecSyncTransition)
	registrar.RegisterApply("apply_plan_execution", s.applyPlanExecuteTransition)
}

// buildPlanExecuteTransitionInput reprojects the authorized plan frontier from
// current state. Re-rendering through Plan Manager is deliberate: the frontier
// digest hashes the rendered plan, so a rebuild is the only way an apply can
// notice that the plan changed while the workflow was running.
func (s *Service) buildPlanExecuteTransitionInput(ctx context.Context, executionID string) (transitionrunner.Snapshot, error) {
	record, item, err := s.loadRecordAndItem(executionID)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	planHandle, err := executionPlanHandle(item)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	snapshot, err := buildPhasedPlanSnapshot(item, record, planHandle, rendered)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	input, err := snapshot.input()
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.Snapshot{Input: input, EntityVersion: snapshot.EntityVersion, FrontierDigest: snapshot.FrontierDigest}, nil
}

func (s *Service) buildSpecSyncTransitionInput(_ context.Context, executionID string) (transitionrunner.Snapshot, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	snapshot, err := buildSpecSyncWorkflowSnapshot(records[idx])
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.Snapshot{Input: snapshot.Input, EntityVersion: snapshot.EntityVersion}, nil
}

// buildWorkTransitionInput reprojects one correction or follow-up. Everything
// it needs is durable: the parent is reachable through ParentExecutionID and
// the operator's steering was persisted as OperatorNote when the record was
// created, so the rebuild reproduces the snapshot byte for byte.
func (s *Service) buildWorkTransitionInput(_ context.Context, executionID string) (transitionrunner.Snapshot, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	record := records[idx]
	item, err := s.loadBacklogItemByRecord(&record)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	parent := record
	if strings.TrimSpace(record.ParentExecutionID) != "" {
		parentRecords, parentIdx, parentErr := s.loadRecordLocked(record.ParentExecutionID)
		if parentErr != nil {
			return transitionrunner.Snapshot{}, fmt.Errorf("load parent execution %q: %w", record.ParentExecutionID, parentErr)
		}
		parent = parentRecords[parentIdx]
	}
	workType := record.Operation
	if strings.TrimSpace(workType) == "" {
		workType = "followup"
	}
	snapshot, err := buildWorkWorkflowSnapshot(item, record, parent, workType, record.OperatorNote)
	if err != nil {
		return transitionrunner.Snapshot{}, err
	}
	return transitionrunner.Snapshot{Input: snapshot.Input, EntityVersion: snapshot.EntityVersion, FrontierDigest: snapshot.FrontierDigest}, nil
}

// loadRecordAndItem resolves a record and its backlog item together.
func (s *Service) loadRecordAndItem(executionID string) (Record, backlogItem, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, backlogItem{}, err
	}
	record := records[idx]
	item, err := s.loadBacklogItemByRecord(&record)
	if err != nil {
		return Record{}, backlogItem{}, err
	}
	return record, item, nil
}

func (s *Service) applyWorkTransition(_ context.Context, executionID string, outcome transitionrunner.Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := records[idx]
	if outcome.TransitionKey != "work.correct" && outcome.TransitionKey != "work.follow_up" {
		return fmt.Errorf("execution %q is not a work transition", executionID)
	}
	if outcome.TransitionKey == "work.correct" && outcome.Name == "corrected" {
		outcome.Name = "complete"
	}
	record.AgentWorkflowTerminalCode, record.AgentWorkflowBudgetName = outcome.TerminalCode, outcome.BudgetName
	record.AgentWorkflowResult, record.AgentWorkflowOutcome = append([]byte(nil), outcome.Result...), outcome.Name
	if err := s.finishWorkWorkflowClaim(&record); err != nil {
		return err
	}
	record.AgentWorkflowAppliedAt = nowRFC3339()
	record.UpdatedAt = record.AgentWorkflowAppliedAt
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(record)
	return nil
}

func (s *Service) finishWorkWorkflowClaim(record *Record) error {
	item, err := s.loadBacklogItemByRecord(record)
	if err != nil {
		return err
	}
	previous := record.Status
	record.FinishedAt = nowRFC3339()
	switch record.AgentWorkflowOutcome {
	case "complete":
		record.Status = StatusCompleted
		candidates := []string{}
		s.applyCompletedTransition(record, item, &candidates)
	case "cancelled":
		record.Status = StatusCanceled
		if err := s.updateBacklogStatus(item, restoreBacklogStatus(*record)); err != nil {
			return err
		}
	default:
		record.Status = StatusNeedsReview
		record.FailureReason = firstNonEmpty(record.AgentWorkflowTerminalCode, record.AgentWorkflowOutcome, "work workflow needs attention")
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	}
	s.logExecutionEvent(*record, previous)
	return nil
}

// transitionApplyComplete reports whether the shared correlation has already
// consumed this workflow's terminal result. It is the single source of truth
// for apply state; the execution record deliberately keeps no copy.
func (s *Service) transitionApplyComplete(workflowExecutionID string) bool {
	if s.transitionRunner == nil || strings.TrimSpace(workflowExecutionID) == "" {
		return false
	}
	correlation, err := s.transitionRunner.GetCorrelation(workflowExecutionID)
	return err == nil && correlation.ApplyState == transitionrun.ApplyStateComplete
}
