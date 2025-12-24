package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

func (s *WorkflowService) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error) {
	if s == nil {
		return nil, fmt.Errorf("workflow service not configured")
	}
	return s.repo.ListExecutions(ctx, workflowID, limit, offset)
}

// ResumeExecution resumes a failed or stopped execution from its last checkpoint.
// It creates a new execution record linked to the original and starts execution
// from the step after the last successfully completed step.
func (s *WorkflowService) ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error) {
	if s == nil {
		return nil, errors.New("workflow service not configured")
	}

	// 1. Validate execution is resumable (includes workflow version check)
	if err := s.ValidateResumable(ctx, executionID); err != nil {
		return nil, fmt.Errorf("execution %s cannot be resumed: %w", executionID, err)
	}

	// 2. Load original execution
	originalExec, err := s.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load execution: %w", err)
	}

	// 3. Extract checkpoint state from disk (timeline.proto.json)
	checkpoint, err := s.ExtractCheckpointState(ctx, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract checkpoint: %w", err)
	}

	// 4. Load workflow for execution
	workflowSummary, err := s.resolveWorkflowForExecution(ctx, originalExec.WorkflowID, checkpoint.WorkflowVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// 5. Merge parameters (new params override checkpoint params)
	mergedParams := make(map[string]any)
	for k, v := range checkpoint.Params {
		mergedParams[k] = v
	}
	if parameters != nil {
		for k, v := range parameters {
			mergedParams[k] = v
		}
	}

	// 6. Extract resume_url if provided in parameters
	resumeURL := ""
	if urlVal, ok := parameters["resume_url"]; ok {
		if urlStr, isStr := urlVal.(string); isStr {
			resumeURL = strings.TrimSpace(urlStr)
			delete(mergedParams, "resume_url") // Don't pass as a workflow parameter
		}
	}

	// 7. Create new execution record with resume reference
	now := time.Now().UTC()
	newExec := &database.ExecutionIndex{
		ID:            uuid.New(),
		WorkflowID:    originalExec.WorkflowID,
		Status:        database.ExecutionStatusPending,
		StartedAt:     now,
		ResumedFromID: &executionID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.repo.CreateExecution(ctx, newExec); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// 8. Write initial execution snapshot
	_ = s.writeExecutionSnapshot(ctx, newExec, &basexecution.Execution{
		ExecutionId:     newExec.ID.String(),
		WorkflowId:      originalExec.WorkflowID.String(),
		WorkflowVersion: int32(checkpoint.WorkflowVersion),
		Status:          enums.StringToExecutionStatus(newExec.Status),
		TriggerType:     basbase.TriggerType_TRIGGER_TYPE_RESUME,
		StartedAt:       autocontracts.TimeToTimestamp(now),
		CreatedAt:       autocontracts.TimeToTimestamp(now),
		UpdatedAt:       autocontracts.TimeToTimestamp(now),
		Parameters: &basexecution.ExecutionParameters{
			InitialStore:  convertParamsToProto(checkpoint.Variables),
			InitialParams: convertParamsToProto(mergedParams),
			Env:           convertParamsToProto(checkpoint.Env),
		},
	})

	// 9. Start execution with resume configuration
	s.startResumedExecution(workflowSummary, newExec.ID, checkpoint, mergedParams, &executionID, resumeURL)

	return newExec, nil
}

// startResumedExecution starts execution from a checkpoint.
func (s *WorkflowService) startResumedExecution(
	workflow *basapi.WorkflowSummary,
	executionID uuid.UUID,
	checkpoint *CheckpointState,
	params map[string]any,
	resumedFromID *uuid.UUID,
	resumeURL string,
) {
	ctx, cancel := context.WithCancel(context.Background())
	s.storeExecutionCancel(executionID, cancel)
	go s.executeResumedWorkflowAsync(ctx, workflow, executionID, checkpoint, params, resumedFromID, resumeURL)
}

// executeResumedWorkflowAsync runs a resumed workflow asynchronously.
func (s *WorkflowService) executeResumedWorkflowAsync(
	ctx context.Context,
	workflow *basapi.WorkflowSummary,
	executionID uuid.UUID,
	checkpoint *CheckpointState,
	params map[string]any,
	resumedFromID *uuid.UUID,
	resumeURL string,
) {
	defer s.cancelExecutionByID(executionID)

	persistenceCtx := context.Background()
	execIndex, err := s.repo.GetExecution(persistenceCtx, executionID)
	if err != nil {
		return
	}

	// Update status to running
	execIndex.Status = database.ExecutionStatusRunning
	execIndex.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, nil, nil, execIndex.UpdatedAt)

	// Build execution plan
	plan, _, err := autoexecutor.BuildContractsPlan(ctx, executionID, workflow)
	if err != nil {
		execIndex.Status = database.ExecutionStatusFailed
		execIndex.ErrorMessage = err.Error()
		now := time.Now().UTC()
		execIndex.CompletedAt = &now
		execIndex.UpdatedAt = now
		errMsg := execIndex.ErrorMessage
		_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, &errMsg, execIndex.CompletedAt, execIndex.UpdatedAt)
		return
	}

	// Resolve engine name from environment (same as regular execution)
	engineName := autoengine.FromEnv().Resolve("")

	// Configure the execution request with resume settings
	req := autoexecutor.Request{
		Plan:               plan,
		EngineName:         engineName,
		EngineFactory:      s.engineFactory,
		Recorder:           s.artifactRecorder,
		EventSink:          s.newEventSink(),
		HeartbeatInterval:  2 * time.Second,
		WorkflowResolver:   s,
		PlanCompiler:       s.planCompiler,
		MaxSubflowDepth:    5,
		// Resume-specific fields
		StartFromStepIndex: checkpoint.LastStepIndex,
		InitialVariables:   checkpoint.Variables,
		InitialStore:       checkpoint.Variables,
		InitialParams:      params,
		Env:                checkpoint.Env,
		ResumedFromID:      resumedFromID,
	}

	// Apply resume URL if provided (overrides start_url)
	if resumeURL != "" {
		req.StartURL = resumeURL
	}

	executor := s.executor
	if executor == nil {
		executor = autoexecutor.NewSimpleExecutor(nil)
	}
	runErr := executor.Execute(ctx, req)

	// Update final status
	status := database.ExecutionStatusCompleted
	errMsg := ""
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) || strings.Contains(strings.ToLower(runErr.Error()), "cancel") {
			status = database.ExecutionStatusFailed
			errMsg = "execution cancelled"
		} else {
			status = database.ExecutionStatusFailed
			errMsg = runErr.Error()
		}
	}

	now := time.Now().UTC()
	execIndex.Status = status
	execIndex.ErrorMessage = errMsg
	execIndex.CompletedAt = &now
	execIndex.UpdatedAt = now
	var errPtr *string
	if strings.TrimSpace(execIndex.ErrorMessage) != "" {
		errPtr = &execIndex.ErrorMessage
	}
	_ = s.repo.UpdateExecutionStatus(persistenceCtx, execIndex.ID, execIndex.Status, errPtr, execIndex.CompletedAt, execIndex.UpdatedAt)
}

