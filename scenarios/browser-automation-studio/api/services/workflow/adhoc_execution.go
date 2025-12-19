package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

const (
	adhocWorkflowIndexName       = "adhoc"
	adhocWorkflowIndexFolderPath = "/__adhoc"
)

func (s *WorkflowService) ensureAdhocWorkflowIndex(ctx context.Context) (uuid.UUID, error) {
	existing, err := s.repo.GetWorkflowByName(ctx, adhocWorkflowIndexName, adhocWorkflowIndexFolderPath)
	if err == nil && existing != nil && existing.ID != uuid.Nil {
		return existing.ID, nil
	}
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return uuid.Nil, err
	}

	index := &database.WorkflowIndex{
		ID:         uuid.New(),
		ProjectID:  nil,
		Name:       adhocWorkflowIndexName,
		FolderPath: adhocWorkflowIndexFolderPath,
		FilePath:   "",
		Version:    1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if createErr := s.repo.CreateWorkflow(ctx, index); createErr != nil {
		// Possible concurrent creation or unique constraint; attempt to read it back.
		latest, getErr := s.repo.GetWorkflowByName(ctx, adhocWorkflowIndexName, adhocWorkflowIndexFolderPath)
		if getErr == nil && latest != nil && latest.ID != uuid.Nil {
			return latest.ID, nil
		}
		if getErr != nil {
			return uuid.Nil, getErr
		}
		return uuid.Nil, createErr
	}
	return index.ID, nil
}

// ExecuteAdhocWorkflowAPI executes a provided workflow definition without persisting it as a workflow index.
// Artifacts are still recorded to disk using the standard recorder.
func (s *WorkflowService) ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.FlowDefinition == nil {
		return nil, fmt.Errorf("flow_definition is required")
	}

	executionID := uuid.New()
	workflowID, err := s.ensureAdhocWorkflowIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensure adhoc workflow index: %w", err)
	}

	now := time.Now().UTC()
	wf := &basapi.WorkflowSummary{
		Id:            workflowID.String(),
		ProjectId:     "",
		Name:          strings.TrimSpace(req.GetMetadata().GetName()),
		FolderPath:    "/",
		Description:   strings.TrimSpace(req.GetMetadata().GetDescription()),
		Version:       1,
		CreatedAt:     autocontracts.TimeToTimestamp(now),
		UpdatedAt:     autocontracts.TimeToTimestamp(now),
		FlowDefinition: req.FlowDefinition,
		LastChangeSource: basbase.ChangeSource_CHANGE_SOURCE_MANUAL,
		LastChangeDescription: "Adhoc execution",
	}
	if strings.TrimSpace(wf.Name) == "" {
		wf.Name = "adhoc"
	}

	store, params, env, artifactCfg := executionParametersToMaps(req.Parameters)

	execIndex := &database.ExecutionIndex{
		ID:        executionID,
		WorkflowID: workflowID,
		Status:    database.ExecutionStatusPending,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateExecution(ctx, execIndex); err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	// Use the standard async runner so status polling, stop requests, and result indexing work.
	s.startExecutionRunnerWithNamespaces(wf, executionID, store, params, env, artifactCfg)

	if !req.WaitForCompletion {
		return &basexecution.ExecuteAdhocResponse{
			ExecutionId: executionID.String(),
			Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
			Message:     "Execution started",
		}, nil
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			latest, err := s.repo.GetExecution(ctx, executionID)
			if err != nil {
				return nil, err
			}
			if latest.CompletedAt == nil {
				continue
			}

			resp := &basexecution.ExecuteAdhocResponse{
				ExecutionId: latest.ID.String(),
				Status:      enums.StringToExecutionStatus(latest.Status),
				Message:     "Execution completed",
				CompletedAt: autocontracts.TimePtrToTimestamp(latest.CompletedAt),
			}
			if strings.TrimSpace(latest.ErrorMessage) != "" {
				msg := latest.ErrorMessage
				resp.Status = basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
				resp.Message = "Execution failed"
				resp.Error = &msg
			}
			return resp, nil
		}
	}
}
