package execution

import (
	"context"
	"encoding/json"
	"fmt"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/stringsx"
	"swarm-manager/internal/transitionrunner"

	"google.golang.org/protobuf/types/known/structpb"
)

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
	if s.transitionRunner == nil {
		return agentmanager.WorkflowStart{}, specSyncWorkflowSnapshot{}, agentmanager.ErrNotAvailable
	}
	started, err := s.transitionRunner.StartWith(ctx, "scenario.spec_sync", record.ExecutionID, transitionrunner.PreparedInput{FirstRunNodeID: "sync", Activity: &transitionrunner.Activity{OwnerType: "backlog", OwnerKind: record.BacklogKind, OwnerName: record.BacklogName, Purpose: "spec_sync"}})
	if err != nil {
		return agentmanager.WorkflowStart{}, specSyncWorkflowSnapshot{}, err
	}
	snapshot := specSyncWorkflowSnapshot{EntityVersion: started.EntityVersion, WorkflowKey: started.WorkflowKey}
	start := agentmanager.WorkflowStart{ExecutionID: started.ExecutionID, DefinitionDigest: started.DefinitionDigest}
	if len(started.Attempts) > 0 {
		start.RunID = started.Attempts[0].RunID
	}
	return start, snapshot, nil
}

func (s *Service) ApplySpecSyncWorkflow(ctx context.Context, executionID string) (PhasedPlanApplyResult, error) {
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
	if correlation.TransitionKey != "scenario.spec_sync" {
		return PhasedPlanApplyResult{}, apierr.BadRequest("execution is not owned by a scenario spec-sync workflow")
	}
	record, err := s.Get(ctx, executionID)
	if err != nil {
		return PhasedPlanApplyResult{}, err
	}
	return PhasedPlanApplyResult{Record: record, Idempotent: before.ApplyState == "complete"}, nil
}

func (s *Service) applySpecSyncTransition(ctx context.Context, executionID string, outcome transitionrunner.Outcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return err
	}
	record := records[idx]
	if outcome.TransitionKey != "scenario.spec_sync" {
		return fmt.Errorf("execution %q is not a scenario spec-sync transition", executionID)
	}
	workflowOutcome := outcome.Name
	if workflowOutcome == "synced" {
		workflowOutcome = "complete"
	}
	previous := record.Status
	record.FinishedAt = nowRFC3339()
	if workflowOutcome == "complete" {
		record.Status = StatusCompleted
		s.handleSpecSyncComplete(ctx, &record)
	} else if outcome.Name == "cancelled" {
		record.Status = StatusCanceled
	} else {
		record.Status = StatusFailed
		record.FailureReason = stringsx.FirstNonEmpty(outcome.TerminalCode, outcome.Name, "spec-sync workflow failed")
	}
	s.logExecutionEvent(record, previous)
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return err
	}
	s.dispatchStatusUpdate(record)
	return nil
}
