package review

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const reviewWorkflowApplyComplete = "complete"

// ApplyWorkflowRound collects and applies one terminal independent-review
// result. It is the explicit local mutation boundary for declared reviews.
func (s *Service) ApplyWorkflowRound(ctx context.Context, kind, name string, roundNum int) (Round, bool, error) {
	itemDir := s.resolveItemDir(kind, name)
	round, err := LoadRound(itemDir, roundNum)
	if err != nil {
		return Round{}, false, fmt.Errorf("load review round: %w", err)
	}
	if round == nil {
		return Round{}, false, fmt.Errorf("review round %d does not exist", roundNum)
	}
	if round.AgentWorkflowApplyState == reviewWorkflowApplyComplete {
		return *round, true, nil
	}
	if strings.TrimSpace(round.AgentWorkflowExecutionID) == "" || s.workflow == nil {
		return Round{}, false, fmt.Errorf("round is not owned by an independent review workflow")
	}
	completion, err := s.workflow.CollectWorkflow(ctx, round.AgentWorkflowExecutionID)
	if err != nil {
		return Round{}, false, err
	}
	if completion.ExecutionID != round.AgentWorkflowExecutionID || completion.DefinitionDigest != round.AgentWorkflowDefinition || completion.Input == nil {
		return Round{}, false, fmt.Errorf("workflow completion does not match review round provenance")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	entity, ok := input["entity"].(map[string]any)
	if !ok || entity["kind"] != kind || entity["name"] != name || entity["executionId"] != round.ExecutionID || entity["version"] != round.AgentWorkflowVersion {
		return Round{}, false, fmt.Errorf("workflow completion does not match review snapshot")
	}
	result := json.RawMessage(nil)
	if completion.Output != nil {
		value, marshalErr := json.Marshal(completion.Output.AsInterface())
		if marshalErr != nil {
			return Round{}, false, marshalErr
		}
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if json.Unmarshal(value, &envelope) == nil {
			result = envelope.Result
		}
	}
	outcome := "failed"
	if completion.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		handoff := parseReviewHandoff(result)
		switch handoff.Verdict {
		case "ready", "ready_with_notes":
			outcome = "accepted"
		case "needs_work":
			outcome = "changes-requested"
		}
	}
	FinalizeRoundFromResult(round, result, outcome)
	round.AgentWorkflowApplyState = reviewWorkflowApplyComplete
	round.AgentWorkflowAppliedAt = s.now().Format(time.RFC3339)
	if err := SaveRound(itemDir, *round); err != nil {
		return Round{}, false, err
	}
	if s.onRoundTerminal != nil {
		s.onRoundTerminal(ctx, kind, name, *round)
	}
	return *round, false, nil
}
