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

type conclusionWorkflowResult struct {
	Result struct {
		Handoff struct {
			Summary  string `json:"summary"`
			Progress string `json:"progress"`
			NextStep string `json:"next_step"`
		} `json:"handoff"`
	} `json:"result"`
}

// conclusionWorkflowSnapshot is the immutable execution-side boundary for a
// planless research conclusion. Agent Manager owns bounded turns; execution
// retains authority over the item and eventual terminal application.
type conclusionWorkflowSnapshot struct {
	EntityVersion string
	Input         *structpb.Value
}

func buildConclusionWorkflowSnapshot(item backlogItem, record Record, operatorNote string) (conclusionWorkflowSnapshot, error) {
	encoded, err := json.Marshal(struct {
		Item      backlogItem `json:"item"`
		Execution string      `json:"execution_id"`
	}{Item: item, Execution: record.ExecutionID})
	if err != nil {
		return conclusionWorkflowSnapshot{}, fmt.Errorf("encode conclusion snapshot: %w", err)
	}
	version := digestStrings(string(encoded))
	var snapshot map[string]any
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return conclusionWorkflowSnapshot{}, fmt.Errorf("decode conclusion snapshot: %w", err)
	}
	input, err := structpb.NewValue(map[string]any{
		"entity":   map[string]any{"kind": item.Kind, "name": item.Name, "version": version},
		"snapshot": snapshot, "operatorNote": strings.TrimSpace(operatorNote),
	})
	if err != nil {
		return conclusionWorkflowSnapshot{}, fmt.Errorf("build conclusion workflow input: %w", err)
	}
	return conclusionWorkflowSnapshot{EntityVersion: version, Input: input}, nil
}

func (s *Service) startConclusionWorkflow(ctx context.Context, item backlogItem, record Record, operatorNote string) (agentmanager.WorkflowStart, conclusionWorkflowSnapshot, error) {
	if s.conclusionWorkflow == nil {
		return agentmanager.WorkflowStart{}, conclusionWorkflowSnapshot{}, agentmanager.ErrNotAvailable
	}
	snapshot, err := buildConclusionWorkflowSnapshot(item, record, operatorNote)
	if err != nil {
		return agentmanager.WorkflowStart{}, conclusionWorkflowSnapshot{}, err
	}
	start, err := s.conclusionWorkflow.StartWorkflow(ctx, agentmanager.Invocation{
		Owner: "swarm-manager", WorkflowKey: "swarm-manager/research-conclude", Input: snapshot.Input,
		IdempotencyKey: "research-conclude/" + record.ExecutionID + "/" + snapshot.EntityVersion,
		FirstRunNodeID: "conclude",
	})
	return start, snapshot, err
}

// ApplyConclusionWorkflow is the explicit, exactly-once domain apply boundary
// for research.conclude. Agent Manager owns collection and terminal state;
// Swarm verifies the immutable item snapshot before changing its ledger.
func (s *Service) ApplyConclusionWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record := records[idx]
	if record.AgentWorkflowKey != "swarm-manager/research-conclude" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a research conclusion workflow")
	}
	if record.AgentWorkflowApplyState == workflowApplyComplete {
		return PhasedPlanApplyResult{Record: record, Idempotent: true}, nil
	}
	if record.AgentWorkflowApplyState != workflowApplyClaimed {
		completion, collectErr := s.conclusionWorkflow.CollectWorkflow(ctx, record.AgentWorkflowExecutionID)
		if collectErr != nil {
			return PhasedPlanApplyResult{}, wrapAgentError(collectErr)
		}
		if err := s.validateConclusionCompletion(record, completion); err != nil {
			return PhasedPlanApplyResult{}, err
		}
		raw := []byte("null")
		if completion.Output != nil {
			raw, err = json.Marshal(completion.Output.AsInterface())
			if err != nil {
				return PhasedPlanApplyResult{}, apierr.BadGateway("workflow returned an unreadable typed result")
			}
		}
		outcome, err := parseConclusionOutcome(completion.Status, raw)
		if err != nil {
			return PhasedPlanApplyResult{}, err
		}
		record.AgentWorkflowTerminalCode = completion.TerminalCode
		record.AgentWorkflowBudgetName = completion.BudgetName
		record.AgentWorkflowResult = raw
		record.AgentWorkflowOutcome = outcome
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
		record.AgentWorkflowApplyState = workflowApplyClaimed
		record.UpdatedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return PhasedPlanApplyResult{}, err
		}
	}
	if err := s.finishConclusionClaim(&record); err != nil {
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

func (s *Service) validateConclusionCompletion(record Record, completion agentmanager.InvocationCompletion) error {
	if completion.ExecutionID != record.AgentWorkflowExecutionID || completion.DefinitionDigest != record.AgentWorkflowDefinition || completion.Input == nil {
		return apierr.Conflict("workflow result does not match the authorized research conclusion")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	if !ok {
		return apierr.Conflict("workflow input is not a valid research conclusion snapshot")
	}
	entity, ok := input["entity"].(map[string]any)
	if !ok || entity["kind"] != record.BacklogKind || entity["name"] != record.BacklogName || entity["version"] != record.AgentWorkflowEntityVersion {
		return apierr.Conflict("workflow result does not match the authorized research conclusion snapshot")
	}
	item, err := s.loadBacklogItemByRecord(&record)
	if err != nil {
		return err
	}
	snapshot, err := buildConclusionWorkflowSnapshot(item, record, fmt.Sprint(input["operatorNote"]))
	if err != nil || snapshot.EntityVersion != record.AgentWorkflowEntityVersion {
		return apierr.Conflict("research item changed while the workflow was running")
	}
	return nil
}

func parseConclusionOutcome(status domainpb.WorkflowExecutionStatus, raw []byte) (string, error) {
	switch status {
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED:
		var result conclusionWorkflowResult
		if len(raw) == 0 || json.Unmarshal(raw, &result) != nil || strings.TrimSpace(result.Result.Handoff.Summary) == "" || result.Result.Handoff.Progress != "complete" {
			return "", apierr.BadGateway("research conclusion workflow succeeded without a complete typed handoff")
		}
		return "complete", nil
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED:
		return "blocked", nil
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED:
		return "budget_exhausted", nil
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
		return "cancelled", nil
	default:
		return "failed", nil
	}
}

func (s *Service) finishConclusionClaim(record *Record) error {
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
		record.FailureReason = firstNonEmpty(record.AgentWorkflowTerminalCode, record.AgentWorkflowOutcome, "research conclusion needs attention")
		if err := s.updateBacklogStatus(item, backlogStatusInReview); err != nil {
			return err
		}
	}
	s.logExecutionEvent(*record, previous)
	return nil
}
