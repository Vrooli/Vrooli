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
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
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
	return s.ExecuteAdhocWorkflowAPIWithOptions(ctx, req, nil)
}

// ExecuteAdhocWorkflowAPIWithOptions executes a provided workflow definition without persisting it as a workflow index.
// Artifacts are still recorded to disk using the standard recorder.
func (s *WorkflowService) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
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

	store, params, env, artifactCfg, execBrowserProfile, projectRoot, startURL := executionParametersToMaps(req.Parameters)
	if err := validateSeedRequirements(req.FlowDefinition, store, params, env); err != nil {
		return nil, err
	}
	if err := s.validateAdhocSubflows(ctx, req.FlowDefinition, workflowID, projectRoot); err != nil {
		return nil, err
	}

	// Extract adhoc workflow's default browser profile and merge with execution override
	var workflowBrowserProfile *archiveingestion.BrowserProfile
	if req.FlowDefinition != nil && req.FlowDefinition.Settings != nil && req.FlowDefinition.Settings.BrowserProfile != nil {
		workflowBrowserProfile = archiveingestion.BrowserProfileFromProto(req.FlowDefinition.Settings.BrowserProfile)
	}
	finalBrowserProfile := archiveingestion.MergeBrowserProfiles(workflowBrowserProfile, execBrowserProfile)

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
	s.startExecutionRunnerWithOptions(wf, executionID, store, params, env, artifactCfg, finalBrowserProfile, opts, projectRoot, startURL)

	// Adhoc runs return immediately; callers should poll the execution ID.
	return &basexecution.ExecuteAdhocResponse{
		ExecutionId: executionID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		Message:     "Execution started (adhoc). Poll executions API for status.",
	}, nil
}
