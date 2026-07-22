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

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	workflowApplyClaimed  = "claimed"
	workflowApplyComplete = "complete"
)

type phasedPlanResult struct {
	Outcome string                  `json:"outcome"`
	Summary string                  `json:"summary,omitempty"`
	Reason  string                  `json:"reason,omitempty"`
	Blocker phasedPlanResultBlocker `json:"blocker,omitempty"`
	Handoff string                  `json:"handoff,omitempty"`
}

type phasedPlanSnapshot struct {
	PlanReference  string
	FrontierDigest string
	ExecutionID    string
	EntityKind     string
	EntityName     string
	EntityVersion  string
	MaxSlices      int
	WriteScope     []string
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

// SetPhasedPlanWorkflow installs the generic declared-workflow seam. The
// execution domain owns its immutable snapshot and terminal application; the
// Agent Manager transport owns no plan-shaped command/result API.
func (s *Service) SetPhasedPlanWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phasedPlanWorkflow = workflow
}

// SetWorkWorkflow installs the generic declared-workflow seam for bounded
// follow-up and correction work. The domain adapter owns snapshots and apply;
// the workflow owns the agent invocation and typed extraction.
func (s *Service) SetWorkWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workWorkflow = workflow
}

// SetSpecSyncWorkflow installs the declaration-backed scenario spec-sync seam.
func (s *Service) SetSpecSyncWorkflow(workflow agentmanager.WorkflowInvoker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.specSyncWorkflow = workflow
}

// SetWorkflowStartGuard applies server-owned transition policy to the default
// generic workflow adapter without coupling execution to registry internals.
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
		MaxSlices: 6, WriteScope: append([]string(nil), item.AcceptanceAllow...),
	}, nil
}

func (snapshot phasedPlanSnapshot) input() (*structpb.Value, error) {
	writeScope := make([]any, len(snapshot.WriteScope))
	for i, path := range snapshot.WriteScope {
		writeScope[i] = path
	}
	return structpb.NewValue(map[string]any{
		"plan": map[string]any{"reference": snapshot.PlanReference, "frontierDigest": snapshot.FrontierDigest},
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
	if record.AgentWorkflowApplyState == workflowApplyComplete || isWorkflowConsumerTerminal(record.Status) {
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
	signaler, ok := s.phasedPlanWorkflow.(interface {
		SignalWorkflow(context.Context, string, string, *structpb.Value, string) error
	})
	if !ok {
		return Record{}, apierr.BadRequest("approval is not supported by current workflow service")
	}
	payload, err := structpb.NewValue(map[string]any{"executionId": record.ExecutionID, "actor": record.AgentWorkflowApprovalBy})
	if err != nil {
		return Record{}, apierr.Internal("encode workflow approval: %s", err.Error())
	}
	if err := signaler.SignalWorkflow(ctx, record.AgentWorkflowExecutionID, "slice_approved", payload, "approve-"+record.ExecutionID); err != nil {
		return Record{}, wrapAgentError(err)
	}
	return record, nil
}

// ApplyPhasedPlanWorkflow validates and applies one authorized terminal result.
// A persisted claim is sufficient to resume after a process crash without
// collecting mutable external state again.
func (s *Service) ApplyPhasedPlanWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record := records[idx]
	if strings.TrimSpace(record.AgentWorkflowExecutionID) == "" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a phased-plan workflow")
	}
	if record.AgentWorkflowApplyState == workflowApplyComplete {
		return PhasedPlanApplyResult{Record: record, Idempotent: true}, nil
	}
	if record.AgentWorkflowApplyState != workflowApplyClaimed {
		completion, collectErr := s.phasedPlanWorkflow.CollectWorkflow(ctx, record.AgentWorkflowExecutionID)
		if collectErr != nil {
			return PhasedPlanApplyResult{}, wrapAgentError(collectErr)
		}
		if err := s.validatePhasedPlanCompletion(ctx, record, completion); err != nil {
			return PhasedPlanApplyResult{}, err
		}
		record.AgentWorkflowDefinition = completion.DefinitionDigest
		record.AgentWorkflowTerminalCode = completion.TerminalCode
		record.AgentWorkflowBudgetName = completion.BudgetName
		resultRaw, parseErr := phasedPlanResultFromCompletion(completion)
		if parseErr != nil {
			return PhasedPlanApplyResult{}, parseErr
		}
		record.AgentWorkflowResult = append(json.RawMessage(nil), resultRaw...)
		record.AgentWorkflowAttempts = make([]WorkflowAttemptProvenance, 0, len(completion.Attempts))
		for _, attempt := range completion.Attempts {
			if attempt == nil {
				continue
			}
			record.AgentWorkflowAttempts = append(record.AgentWorkflowAttempts, WorkflowAttemptProvenance{
				NodeID: attempt.NodeId, Ordinal: attempt.Ordinal, Strategy: attempt.Strategy,
				RunID: attempt.RunId, ConversationID: attempt.ConversationId,
				SourceAttemptID: attempt.SourceAttemptId, ProfileIdentity: attempt.ProfileIdentity,
			})
		}
		result, parseErr := parsePhasedPlanOutcome(completion.Status, resultRaw)
		if parseErr != nil {
			return PhasedPlanApplyResult{}, parseErr
		}
		record.AgentWorkflowOutcome = result.Outcome
		record.AgentWorkflowApplyState = workflowApplyClaimed
		record.UpdatedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return PhasedPlanApplyResult{}, err
		}
	}

	if err := s.finishPhasedPlanClaim(&record); err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record.AgentWorkflowApplyState = workflowApplyComplete
	record.AgentWorkflowAppliedAt = nowRFC3339()
	record.UpdatedAt = record.AgentWorkflowAppliedAt
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return PhasedPlanApplyResult{}, err
	}
	s.dispatchStatusUpdate(record)
	return PhasedPlanApplyResult{Record: record}, nil
}

func (s *Service) validatePhasedPlanCompletion(ctx context.Context, record Record, completion agentmanager.InvocationCompletion) error {
	if completion.ExecutionID != record.AgentWorkflowExecutionID || completion.DefinitionDigest != record.AgentWorkflowDefinition || completion.Input == nil {
		return apierr.Conflict("workflow result does not match the authorized execution frontier")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	if !ok {
		return apierr.Conflict("workflow input is not a valid execution snapshot")
	}
	consumer, consumerOK := input["consumer"].(map[string]any)
	plan, planOK := input["plan"].(map[string]any)
	if !consumerOK || !planOK || stringValue(consumer["executionId"]) != record.ExecutionID ||
		stringValue(consumer["entityKind"]) != record.BacklogKind || stringValue(consumer["entityName"]) != record.BacklogName ||
		stringValue(consumer["entityVersion"]) != record.AgentWorkflowEntityVersion || stringValue(plan["frontierDigest"]) != record.AgentWorkflowFrontier {
		return apierr.Conflict("workflow result does not match the authorized execution frontier")
	}
	item, err := s.loadBacklogItemByRecord(&record)
	if err != nil {
		return err
	}
	handle, err := executionPlanHandle(item)
	if err != nil {
		return apierr.Conflict("execution plan frontier is no longer available")
	}
	rendered, err := resolveRenderedPlanContent(ctx, item, s.planRenderer)
	if err != nil {
		return apierr.Conflict("execution plan frontier can no longer be rendered")
	}
	snapshot, err := buildPhasedPlanSnapshot(item, record, handle, rendered)
	if err != nil || snapshot.FrontierDigest != record.AgentWorkflowFrontier || snapshot.EntityVersion != record.AgentWorkflowEntityVersion {
		return apierr.Conflict("execution plan frontier changed while the workflow was running")
	}
	return nil
}

func phasedPlanResultFromCompletion(completion agentmanager.InvocationCompletion) (json.RawMessage, error) {
	if completion.Output == nil {
		return nil, nil
	}
	output, ok := completion.Output.AsInterface().(map[string]any)
	if !ok {
		return nil, apierr.BadGateway("workflow output is not an object")
	}
	result, ok := output["result"]
	if !ok {
		return nil, apierr.BadGateway("workflow output omitted result")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, apierr.BadGateway("workflow result is not serializable")
	}
	return raw, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
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
	case "complete", "blocked", "abstained":
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
	case "blocked", "abstained":
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
