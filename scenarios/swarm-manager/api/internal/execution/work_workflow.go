package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type workWorkflowResult struct {
	Result struct {
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
	} `json:"result"`
}

type workWorkflowSnapshot struct {
	EntityVersion  string
	FrontierDigest string
	Input          *structpb.Value
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

func workWorkflowKey(workType string) string {
	if workType == "fixup" {
		return "swarm-manager/work-correct"
	}
	return "swarm-manager/work-follow-up"
}

func (s *Service) startWorkWorkflow(ctx context.Context, item backlogItem, record, parent Record, workType, note string) (agentmanager.WorkflowStart, workWorkflowSnapshot, error) {
	if s.workWorkflow == nil {
		return agentmanager.WorkflowStart{}, workWorkflowSnapshot{}, agentmanager.ErrNotAvailable
	}
	snapshot, err := buildWorkWorkflowSnapshot(item, record, parent, workType, note)
	if err != nil {
		return agentmanager.WorkflowStart{}, workWorkflowSnapshot{}, err
	}
	start, err := s.workWorkflow.StartWorkflow(ctx, agentmanager.Invocation{
		Owner: "swarm-manager", WorkflowKey: workWorkflowKey(workType), Input: snapshot.Input,
		IdempotencyKey: "work/" + record.ExecutionID + "/" + snapshot.EntityVersion,
		FirstRunNodeID: map[bool]string{true: "correct", false: "follow_up"}[workType == "fixup"],
	})
	return start, snapshot, err
}

// ApplyWorkWorkflow applies one terminal typed result exactly once. No polling
// worker reads this workflow: callers explicitly cross this domain boundary.
func (s *Service) ApplyWorkWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record := records[idx]
	if record.AgentWorkflowKey != "swarm-manager/work-follow-up" && record.AgentWorkflowKey != "swarm-manager/work-correct" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a work workflow")
	}
	if record.AgentWorkflowApplyState == workflowApplyComplete {
		return PhasedPlanApplyResult{Record: record, Idempotent: true}, nil
	}
	if record.AgentWorkflowApplyState != workflowApplyClaimed {
		completion, collectErr := s.workWorkflow.CollectWorkflow(ctx, record.AgentWorkflowExecutionID)
		if collectErr != nil {
			return PhasedPlanApplyResult{}, wrapAgentError(collectErr)
		}
		if err := validateWorkWorkflowCompletion(record, completion); err != nil {
			return PhasedPlanApplyResult{}, err
		}
		if completion.Output == nil {
			return PhasedPlanApplyResult{}, apierr.BadGateway("work workflow returned an unreadable typed result")
		}
		raw, err := json.Marshal(completion.Output.AsInterface())
		if err != nil {
			return PhasedPlanApplyResult{}, apierr.BadGateway("work workflow returned an unreadable typed result")
		}
		outcome, err := parseWorkWorkflowOutcome(record.AgentWorkflowKey, completion.Status, raw)
		if err != nil {
			return PhasedPlanApplyResult{}, err
		}
		record.AgentWorkflowTerminalCode = completion.TerminalCode
		record.AgentWorkflowBudgetName = completion.BudgetName
		record.AgentWorkflowResult = raw
		record.AgentWorkflowOutcome = outcome
		record.AgentWorkflowApplyState = workflowApplyClaimed
		record.UpdatedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return PhasedPlanApplyResult{}, err
		}
	}
	if err := s.finishWorkWorkflowClaim(&record); err != nil {
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

func validateWorkWorkflowCompletion(record Record, completion agentmanager.InvocationCompletion) error {
	if completion.ExecutionID != record.AgentWorkflowExecutionID || completion.DefinitionDigest != record.AgentWorkflowDefinition || completion.Input == nil {
		return apierr.Conflict("workflow result does not match the authorized work execution")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	if !ok {
		return apierr.Conflict("workflow input is not a valid work snapshot")
	}
	entity, ok := input["entity"].(map[string]any)
	if !ok || entity["kind"] != record.BacklogKind || entity["name"] != record.BacklogName || entity["executionId"] != record.ExecutionID || entity["version"] != record.AgentWorkflowEntityVersion {
		return apierr.Conflict("workflow result does not match the authorized work snapshot")
	}
	return nil
}

func parseWorkWorkflowOutcome(key string, status domainpb.WorkflowExecutionStatus, raw []byte) (string, error) {
	if status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		switch status {
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
			return "cancelled", nil
		case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED:
			return "needs_attention", nil
		default:
			return "failed", nil
		}
	}
	var result workWorkflowResult
	if json.Unmarshal(raw, &result) != nil || strings.TrimSpace(result.Result.Summary) == "" {
		return "", apierr.BadGateway("work workflow succeeded without a typed result")
	}
	valid := map[string]bool{"complete": true, "proposed": true, "needs_attention": true, "abstained": true}
	if key == "swarm-manager/work-correct" {
		valid = map[string]bool{"corrected": true, "needs_attention": true, "abstained": true}
		if result.Result.Outcome == "corrected" {
			return "complete", nil
		}
	}
	if !valid[result.Result.Outcome] {
		return "", apierr.BadGateway("work workflow returned an invalid outcome")
	}
	return result.Result.Outcome, nil
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
