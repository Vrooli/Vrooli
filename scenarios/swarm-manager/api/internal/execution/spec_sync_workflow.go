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

type specSyncWorkflowResult struct {
	Result struct {
		Outcome string `json:"outcome"`
		Summary string `json:"summary"`
	} `json:"result"`
}

type specSyncWorkflowSnapshot struct {
	EntityVersion string
	Input         *structpb.Value
	WorkflowKey   string
}

func buildSpecSyncWorkflowSnapshot(record Record) (specSyncWorkflowSnapshot, error) {
	if record.ArchiveContext == nil {
		return specSyncWorkflowSnapshot{}, fmt.Errorf("spec-sync record has no archive context")
	}
	// Keep the workflow input portable and bounded. The local path remains a
	// Swarm-only archive capability, while the scenario identity and archival
	// intent are pinned for the agent's declared work.
	snapshot := map[string]any{
		"scenarioName":   record.ArchiveContext.ScenarioName,
		"presetOrCustom": record.ArchiveContext.PresetOrCustom,
		"preservePaths":  record.ArchiveContext.PreservePaths,
		"preservePreset": record.ArchiveContext.PreservePreset,
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return specSyncWorkflowSnapshot{}, fmt.Errorf("encode spec-sync snapshot: %w", err)
	}
	var jsonSnapshot map[string]any
	if err := json.Unmarshal(encoded, &jsonSnapshot); err != nil {
		return specSyncWorkflowSnapshot{}, fmt.Errorf("decode spec-sync snapshot: %w", err)
	}
	version := digestStrings(record.ExecutionID, string(encoded))
	input, err := structpb.NewValue(map[string]any{
		"entity":   map[string]any{"kind": "scenario", "name": record.ArchiveContext.ScenarioName, "executionId": record.ExecutionID, "version": version},
		"snapshot": jsonSnapshot,
	})
	if err != nil {
		return specSyncWorkflowSnapshot{}, fmt.Errorf("build spec-sync workflow input: %w", err)
	}
	return specSyncWorkflowSnapshot{EntityVersion: version, Input: input}, nil
}

func (s *Service) startSpecSyncWorkflow(ctx context.Context, record Record) (agentmanager.WorkflowStart, specSyncWorkflowSnapshot, error) {
	if s.specSyncWorkflow == nil {
		return agentmanager.WorkflowStart{}, specSyncWorkflowSnapshot{}, agentmanager.ErrNotAvailable
	}
	snapshot, err := buildSpecSyncWorkflowSnapshot(record)
	if err != nil {
		return agentmanager.WorkflowStart{}, specSyncWorkflowSnapshot{}, err
	}
	workflow, err := s.resolveWorkflow("scenario.spec_sync")
	if err != nil {
		return agentmanager.WorkflowStart{}, specSyncWorkflowSnapshot{}, err
	}
	snapshot.WorkflowKey = workflow.Key
	start, err := s.specSyncWorkflow.StartWorkflow(ctx, agentmanager.Invocation{
		Owner: workflow.Owner, WorkflowKey: workflow.Key, Input: snapshot.Input,
		IdempotencyKey: "scenario-spec-sync/" + record.ExecutionID + "/" + snapshot.EntityVersion, FirstRunNodeID: "sync",
	})
	return start, snapshot, err
}

func (s *Service) ApplySpecSyncWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	record := records[idx]
	if record.AgentWorkflowKey != "swarm-manager/scenario-spec-sync" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a scenario spec-sync workflow")
	}
	if record.AgentWorkflowApplyState == workflowApplyComplete {
		return PhasedPlanApplyResult{Record: record, Idempotent: true}, nil
	}
	if record.AgentWorkflowApplyState != workflowApplyClaimed {
		completion, collectErr := s.specSyncWorkflow.CollectWorkflow(ctx, record.AgentWorkflowExecutionID)
		if collectErr != nil {
			return PhasedPlanApplyResult{}, wrapAgentError(collectErr)
		}
		if err := validateSpecSyncCompletion(record, completion); err != nil {
			return PhasedPlanApplyResult{}, err
		}
		if completion.Output == nil {
			return PhasedPlanApplyResult{}, apierr.BadGateway("spec-sync workflow returned no typed result")
		}
		raw, err := json.Marshal(completion.Output.AsInterface())
		if err != nil {
			return PhasedPlanApplyResult{}, apierr.BadGateway("spec-sync workflow returned an unreadable typed result")
		}
		outcome, err := parseSpecSyncWorkflowOutcome(completion.Status, raw)
		if err != nil {
			return PhasedPlanApplyResult{}, err
		}
		record.AgentWorkflowTerminalCode, record.AgentWorkflowBudgetName = completion.TerminalCode, completion.BudgetName
		record.AgentWorkflowResult, record.AgentWorkflowOutcome = raw, outcome
		record.AgentWorkflowApplyState, record.UpdatedAt = workflowApplyClaimed, nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return PhasedPlanApplyResult{}, err
		}
	}
	previous := record.Status
	record.FinishedAt = nowRFC3339()
	if record.AgentWorkflowOutcome == "complete" {
		record.Status = StatusCompleted
		s.handleSpecSyncComplete(ctx, &record)
	} else if record.AgentWorkflowOutcome == "cancelled" {
		record.Status = StatusCanceled
	} else {
		record.Status = StatusFailed
		record.FailureReason = firstNonEmpty(record.AgentWorkflowTerminalCode, record.AgentWorkflowOutcome, "spec-sync workflow failed")
	}
	s.logExecutionEvent(record, previous)
	record.AgentWorkflowApplyState, record.AgentWorkflowAppliedAt = workflowApplyComplete, nowRFC3339()
	record.UpdatedAt = record.AgentWorkflowAppliedAt
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return PhasedPlanApplyResult{}, err
	}
	s.dispatchStatusUpdate(record)
	return PhasedPlanApplyResult{Record: record}, nil
}

func validateSpecSyncCompletion(record Record, completion agentmanager.InvocationCompletion) error {
	if completion.ExecutionID != record.AgentWorkflowExecutionID || completion.DefinitionDigest != record.AgentWorkflowDefinition || completion.Input == nil {
		return apierr.Conflict("workflow result does not match the authorized scenario spec-sync")
	}
	input, ok := completion.Input.AsInterface().(map[string]any)
	if !ok {
		return apierr.Conflict("workflow input is not a valid scenario spec-sync snapshot")
	}
	entity, ok := input["entity"].(map[string]any)
	if !ok || entity["kind"] != "scenario" || entity["name"] != record.BacklogName || entity["executionId"] != record.ExecutionID || entity["version"] != record.AgentWorkflowEntityVersion {
		return apierr.Conflict("workflow result does not match the authorized scenario spec-sync snapshot")
	}
	return nil
}

func parseSpecSyncWorkflowOutcome(status domainpb.WorkflowExecutionStatus, raw []byte) (string, error) {
	if status != domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED {
		if status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED {
			return "cancelled", nil
		}
		return "failed", nil
	}
	var result specSyncWorkflowResult
	if json.Unmarshal(raw, &result) != nil || strings.TrimSpace(result.Result.Summary) == "" {
		return "", apierr.BadGateway("spec-sync workflow succeeded without a typed result")
	}
	if result.Result.Outcome != "complete" && result.Result.Outcome != "needs_attention" {
		return "", apierr.BadGateway("spec-sync workflow returned an invalid outcome")
	}
	return result.Result.Outcome, nil
}
