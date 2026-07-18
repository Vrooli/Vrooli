package initiativereview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/review"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

const initiativeWorkflowApplyComplete = "complete"

func initiativeReviewInput(initName string, roundNum int, init any) (*structpb.Value, string, error) {
	raw, err := json.Marshal(init)
	if err != nil {
		return nil, "", err
	}
	var snapshot any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, "", err
	}
	h := sha256.Sum256(append([]byte(fmt.Sprintf("%s/%d\x00", initName, roundNum)), raw...))
	version := "sha256:" + hex.EncodeToString(h[:])
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "initiative", "name": initName, "executionId": fmt.Sprintf("%s/r%d", initName, roundNum), "version": version}, "snapshot": snapshot})
	return input, version, err
}

// ApplyWorkflowRound collects and persists a workflow-owned initiative review
// exactly once, then advances only the local initiative review lifecycle.
func (s *Service) ApplyWorkflowRound(ctx context.Context, initiativeName string, roundNum int) (review.Round, bool, error) {
	itemDir := s.initStore.InitDir(initiativeName)
	round, err := review.LoadRound(itemDir, roundNum)
	if err != nil {
		return review.Round{}, false, err
	}
	if round == nil {
		return review.Round{}, false, fmt.Errorf("initiative review round %d does not exist", roundNum)
	}
	if round.AgentWorkflowApplyState == initiativeWorkflowApplyComplete {
		return *round, true, nil
	}
	if !round.WorkflowOwned() || s.workflow == nil {
		return review.Round{}, false, fmt.Errorf("round is not owned by an initiative review workflow")
	}
	completion, err := s.workflow.CollectWorkflow(ctx, round.AgentWorkflowExecutionID)
	if err != nil {
		return review.Round{}, false, err
	}
	if completion.ExecutionID != round.AgentWorkflowExecutionID || completion.DefinitionDigest != round.AgentWorkflowDefinition || completion.Input == nil {
		return review.Round{}, false, fmt.Errorf("workflow completion does not match initiative review provenance")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	entity, ok := input["entity"].(map[string]any)
	if !ok || entity["kind"] != "initiative" || entity["name"] != initiativeName || entity["version"] != round.AgentWorkflowVersion {
		return review.Round{}, false, fmt.Errorf("workflow completion does not match initiative review snapshot")
	}
	if completion.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED && completion.Output != nil {
		var envelope struct {
			Result struct {
				Assessment     string `json:"assessment"`
				Classification string `json:"classification"`
			} `json:"result"`
		}
		raw, _ := json.Marshal(completion.Output.AsInterface())
		if json.Unmarshal(raw, &envelope) == nil && strings.TrimSpace(envelope.Result.Assessment) != "" && (envelope.Result.Classification == "delivered" || envelope.Result.Classification == "partial" || envelope.Result.Classification == "failed") {
			round.Status, round.AgentAssessment, round.Classification, round.FailureReason = review.RoundStatusComplete, envelope.Result.Assessment, envelope.Result.Classification, ""
		} else {
			round.Status, round.FailureReason = review.RoundStatusFailed, "workflow returned invalid initiative review result"
		}
	} else {
		round.Status, round.FailureReason = review.RoundStatusFailed, firstNonEmpty(completion.TerminalCode, "initiative review workflow did not complete")
	}
	round.AgentWorkflowApplyState = initiativeWorkflowApplyComplete
	round.AgentWorkflowAppliedAt = s.clock().UTC().Format(time.RFC3339)
	if err := review.SaveRound(itemDir, *round); err != nil {
		return review.Round{}, false, err
	}
	s.handleTerminalRound(ctx, initiativeName, *round)
	return *round, false, nil
}

func (s *Service) SetWorkflow(workflow agentmanager.WorkflowInvoker) { s.workflow = workflow }

func (s *Service) SetWorkflowStartGuard(guard agentmanager.WorkflowStartGuard) {
	if workflow, ok := s.workflow.(*agentmanager.WorkflowService); ok {
		workflow.SetStartGuard(guard)
	}
}
