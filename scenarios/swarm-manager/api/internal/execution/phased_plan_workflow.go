package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/transitionrunner"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	"google.golang.org/protobuf/types/known/structpb"
)

type phasedPlanResult struct {
	Outcome string                  `json:"outcome"`
	Summary string                  `json:"summary,omitempty"`
	Reason  string                  `json:"reason,omitempty"`
	Blocker phasedPlanResultBlocker `json:"blocker,omitempty"`
	Handoff string                  `json:"handoff,omitempty"`
}

type phasedPlanSnapshot struct {
	PlanReference   string
	FrontierDigest  string
	ExecutionID     string
	PlanExecutionID string
	EntityKind      string
	EntityName      string
	EntityVersion   string
	MaxSlices       int
	WriteScope      []string
}

type phasedPlanResultBlocker struct {
	Code      string `json:"code,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

func (b *phasedPlanResultBlocker) UnmarshalJSON(data []byte) error {
	var text string
	if json.Unmarshal(data, &text) == nil {
		b.Summary = text
		return nil
	}
	type blockerAlias phasedPlanResultBlocker
	var value blockerAlias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*b = phasedPlanResultBlocker(value)
	return nil
}

// PhasedPlanApplyResult reports whether the transition was newly applied or
// was an idempotent replay of a previously completed callback.
type PhasedPlanApplyResult struct {
	Record     Record `json:"record"`
	Idempotent bool   `json:"idempotent"`
}

// SetPhasedPlanWorkflow installs a workflow transport and builds a local
// runner from it.
//
// TEST-ONLY SEAM. Production composition never calls this: it constructs one
// shared runner and injects it through SetTransitionRunner. Do not add a
// production caller, and do not gate behavior on the field it sets — code that
// did exactly that reported "phased-plan workflow service is not available" for
// every real plan execution, because nothing ever set it.
func (s *Service) SetPhasedPlanWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phasedPlanWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetWorkWorkflow installs a workflow transport for bounded follow-up and
// correction work.
//
// TEST-ONLY SEAM. See SetPhasedPlanWorkflow.
func (s *Service) SetWorkWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetSpecSyncWorkflow installs a workflow transport for scenario spec-sync.
//
// TEST-ONLY SEAM. See SetPhasedPlanWorkflow.
func (s *Service) SetSpecSyncWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.specSyncWorkflow = workflow
	s.configureLocalTransitionRunner()
}

// SetWorkflowStartGuard applies transition policy to a locally-installed
// transport. In production the guard is attached to the shared runner's
// transport at composition, so this is a no-op there.
func (s *Service) SetWorkflowStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := s.phasedPlanWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
	if workflow, ok := s.workWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
	if workflow, ok := s.specSyncWorkflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}

func buildPhasedPlanSnapshot(item backlogItem, record Record, planHandle string, rendered renderedPlanContent) (phasedPlanSnapshot, error) {
	itemBytes, err := json.Marshal(item)
	if err != nil {
		return phasedPlanSnapshot{}, fmt.Errorf("encode backlog snapshot: %w", err)
	}
	frontier := digestStrings(planHandle, rendered.Markdown)
	return phasedPlanSnapshot{
		PlanReference: planHandle, FrontierDigest: frontier, ExecutionID: record.ExecutionID,
		EntityKind: item.Kind, EntityName: item.Name, EntityVersion: digestStrings(string(itemBytes)),
		MaxSlices: firstPositive(record.MaxSlices, 6), WriteScope: append([]string(nil), item.AcceptanceAllow...), PlanExecutionID: record.PlanManagerExecutionID,
	}, nil
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func (snapshot phasedPlanSnapshot) input() (*structpb.Value, error) {
	writeScope := make([]any, len(snapshot.WriteScope))
	for i, path := range snapshot.WriteScope {
		writeScope[i] = path
	}
	return structpb.NewValue(map[string]any{
		"plan":            map[string]any{"reference": snapshot.PlanReference, "frontierDigest": snapshot.FrontierDigest},
		"planExecutionId": snapshot.PlanExecutionID,
		"consumer": map[string]any{
			"executionId": snapshot.ExecutionID, "entityKind": snapshot.EntityKind,
			"entityName": snapshot.EntityName, "entityVersion": snapshot.EntityVersion,
		},
		"constraints": map[string]any{"maxSlices": snapshot.MaxSlices, "writeScope": writeScope},
	})
}

func digestStrings(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ApprovePhasedPlanWorkflow durably records the operator decision before
// delivering its idempotent workflow signal.
func (s *Service) ApprovePhasedPlanWorkflow(ctx context.Context, executionID, actor string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	if strings.TrimSpace(record.AgentWorkflowExecutionID) == "" {
		return Record{}, apierr.BadRequest("execution is not owned by a phased-plan workflow")
	}
	// Apply state lives on the shared correlation, not on this record. Keeping a
	// second copy here meant two sources of truth for whether a terminal result
	// had already been consumed.
	if s.transitionApplyComplete(record.AgentWorkflowExecutionID) || isWorkflowConsumerTerminal(record.Status) {
		return Record{}, apierr.Conflict("execution workflow is already terminal")
	}
	if strings.TrimSpace(record.AgentWorkflowApprovalAt) == "" {
		record.AgentWorkflowApprovalAt = nowRFC3339()
		record.AgentWorkflowApprovalBy = strings.TrimSpace(actor)
		record.UpdatedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
	}
	payload, err := structpb.NewValue(map[string]any{"executionId": record.ExecutionID, "actor": record.AgentWorkflowApprovalBy})
	if err != nil {
		return Record{}, apierr.Internal("encode workflow approval: %s", err.Error())
	}
	var signalErr error
	if s.transitionRunner != nil {
		signalErr = s.transitionRunner.Signal(ctx, record.AgentWorkflowExecutionID, "slice_approved", payload, "approve-"+record.ExecutionID)
	} else if signaler, ok := s.phasedPlanWorkflow.(interface {
		SignalWorkflow(context.Context, string, string, *structpb.Value, string) error
	}); ok {
		signalErr = signaler.SignalWorkflow(ctx, record.AgentWorkflowExecutionID, "slice_approved", payload, "approve-"+record.ExecutionID)
	} else {
		return Record{}, apierr.BadRequest("approval is not supported by current workflow service")
	}
	if signalErr != nil {
		return Record{}, wrapAgentError(signalErr)
	}
	return record, nil
}

// ApplyPhasedPlanWorkflow validates and applies one authorized terminal result.
// A persisted claim is sufficient to resume after a process crash without
// collecting mutable external state again.
func (s *Service) ApplyPhasedPlanWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
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
	if _, err := s.transitionRunner.ApplyExecution(ctx, workflowExecutionID); err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record, err := s.Get(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	return PhasedPlanApplyResult{Record: record, Idempotent: before.ApplyState == "complete"}, nil
}

func (s *Service) applyPlanExecuteTransition(ctx context.Context, executionID string, outcome transitionrunner.Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := records[idx]
	if outcome.TransitionKey != "plan.execute" {
		return fmt.Errorf("execution %q is not a plan execution transition", executionID)
	}
	result, err := parsePhasedPlanOutcome(domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, outcome.Result)
	if err != nil {
		return err
	}
	record.AgentWorkflowTerminalCode, record.AgentWorkflowBudgetName = outcome.TerminalCode, outcome.BudgetName
	record.AgentWorkflowResult, record.AgentWorkflowOutcome = append(json.RawMessage(nil), outcome.Result...), result.Outcome
	record.AgentWorkflowAttempts = make([]WorkflowAttemptProvenance, 0, len(outcome.Attempts))
	for _, attempt := range outcome.Attempts {
		record.AgentWorkflowAttempts = append(record.AgentWorkflowAttempts, WorkflowAttemptProvenance{NodeID: attempt.NodeID, Ordinal: attempt.Ordinal, Strategy: attempt.Strategy, RunID: attempt.RunID, ConversationID: attempt.ConversationID, SourceAttemptID: attempt.SourceAttemptID, ProfileIdentity: attempt.ProfileIdentity})
	}
	if err := s.reconcilePlanManagerCompletion(ctx, &record); err != nil {
		return err
	}
	if err := s.finishPhasedPlanClaim(&record); err != nil {
		return err
	}
	record.AgentWorkflowAppliedAt = nowRFC3339()
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(record)
	return nil
}

// reconcilePlanManagerCompletion makes Plan Manager the final authority for a
// drain that claims completion. The agent can complete phases through the
// bound execution, but Swarm never infers the terminal plan state from a
// handoff alone.
func (s *Service) reconcilePlanManagerCompletion(ctx context.Context, record *Record) error {
	if record.AgentWorkflowOutcome != "complete" || strings.TrimSpace(record.PlanManagerExecutionID) == "" || strings.TrimSpace(record.PlanManagerReconciledAt) != "" {
		return nil
	}
	client, ok := s.planRenderer.(interface {
		GetStatus(context.Context, *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error)
		Complete(context.Context, *executionv1.CompleteRequest) (*executionv1.CompleteResponse, error)
	})
	if !ok {
		return nil
	}
	status, err := client.GetStatus(ctx, &executionv1.GetStatusRequest{ExecutionId: record.PlanManagerExecutionID})
	if err != nil {
		return apierr.BadGateway("get plan-manager execution status: %s", err)
	}
	if status.GetExecution() == nil {
		return apierr.BadGateway("plan-manager status omitted execution")
	}
	if !status.GetExecution().GetComplete() {
		if _, err := client.Complete(ctx, &executionv1.CompleteRequest{ExecutionId: record.PlanManagerExecutionID}); err != nil {
			return apierr.BadGateway("reconcile plan-manager execution completion: %s", err)
		}
		status, err = client.GetStatus(ctx, &executionv1.GetStatusRequest{ExecutionId: record.PlanManagerExecutionID})
		if err != nil {
			return apierr.BadGateway("verify reconciled plan-manager execution: %s", err)
		}
		if status.GetExecution() == nil || !status.GetExecution().GetComplete() {
			return apierr.Conflict("drain claimed complete but bound plan-manager execution is unfinished")
		}
	}
	record.PlanManagerReconciledAt = nowRFC3339()
	return nil
}

func parsePhasedPlanOutcome(status domainpb.WorkflowExecutionStatus, raw json.RawMessage) (phasedPlanResult, error) {
	if status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		var result phasedPlanResult
		_ = json.Unmarshal(raw, &result)
		switch status {
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED:
			result.Outcome = "blocked"
			return result, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED:
			result.Outcome = "abstained"
			return result, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED:
			return phasedPlanResult{Outcome: "budget_exhausted"}, nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
			return phasedPlanResult{Outcome: "cancelled"}, nil
		default:
			return phasedPlanResult{Outcome: "failed"}, nil
		}
	}
	var result phasedPlanResult
	if len(raw) == 0 || json.Unmarshal(raw, &result) != nil {
		return phasedPlanResult{}, apierr.BadGateway("workflow succeeded without a valid typed result")
	}
	switch result.Outcome {
	case "complete", "completed", "blocked", "needs_review", "needs_attention", "abstained", "failed", "budget_exhausted", "cancelled":
		if result.Outcome == "completed" {
			result.Outcome = "complete"
		}
		return result, nil
	default:
		return phasedPlanResult{}, apierr.BadGateway("workflow returned unsupported terminal outcome %q", result.Outcome)
	}
}

func (s *Service) finishPhasedPlanClaim(record *Record) error {
	item, err := s.loadBacklogItemByRecord(record)
	if err != nil {
		return err
	}
	result, _ := parsePhasedPlanOutcome(domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, record.AgentWorkflowResult)
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
	case "blocked", "needs_review", "needs_attention", "abstained":
		record.Status = StatusNeedsReview
		record.FailureReason = firstNonEmpty(result.Blocker.Summary, result.Reason, result.Summary, "workflow "+record.AgentWorkflowOutcome)
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	default:
		record.Status = StatusFailed
		record.FailureReason = firstNonEmpty(record.AgentWorkflowTerminalCode, record.AgentWorkflowOutcome)
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	}
	s.logExecutionEvent(*record, previous)
	return nil
}

func isWorkflowConsumerTerminal(status Status) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}
