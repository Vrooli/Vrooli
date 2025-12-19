// Package workflow provides workflow and execution management services.
// This file defines the service interfaces for clean dependency injection
// and clear responsibility boundaries.
package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// CatalogService manages the workflow and project catalog.
// It handles CRUD operations, versioning, and file synchronization.
// This service is stateless and can be safely shared across handlers.
type CatalogService interface {
	// Health checks
	CheckHealth() string
	CheckAutomationHealth(ctx context.Context) (bool, error)

	// Project management
	CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error
	GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error)
	GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error)
	UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error
	DeleteProject(ctx context.Context, id uuid.UUID) error
	ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error)
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error)
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
	EnsureSeedProject(ctx context.Context) (*database.ProjectIndex, error)
	HydrateProject(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error)

	// Workflow CRUD
	CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
	GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error)
	UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error)
	DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error)
	ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error)

	// Workflow resolution (used by ExecutionService)
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error)
	GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*basapi.WorkflowSummary, error)

	// Versioning
	ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error)
	GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error)
	RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error)

	// File synchronization
	SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error

	// AI workflow modification
	ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error)
}

// ExecutionService manages workflow execution lifecycle.
// It handles starting, stopping, and monitoring workflow executions.
type ExecutionService interface {
	// Execution control
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)
	ExecuteWorkflowAPI(ctx context.Context, req *basapi.ExecuteWorkflowRequest) (*basapi.ExecuteWorkflowResponse, error)
	ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error)
	ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error)
	StopExecution(ctx context.Context, executionID uuid.UUID) error
	ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.ExecutionIndex, error)

	// Execution queries
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.ExecutionIndex, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	UpdateExecution(ctx context.Context, execution *database.ExecutionIndex) error
	GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*basexecution.ExecutionScreenshot, error)
	HydrateExecutionProto(ctx context.Context, execIndex *database.ExecutionIndex) (*basexecution.Execution, error)

	// Timeline and export
	GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error)
	GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error)
	DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error)
	ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error
}

// WorkflowResolver is a minimal interface for resolving workflows during execution.
// This allows ExecutionService to look up workflows without depending on full CatalogService.
// It's implemented by CatalogService and can be injected into ExecutionService.
type WorkflowResolver interface {
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error)
	GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*basapi.WorkflowSummary, error)
}

// Compile-time checks to ensure WorkflowService implements all interfaces
var (
	_ CatalogService   = (*WorkflowService)(nil)
	_ ExecutionService = (*WorkflowService)(nil)
	_ WorkflowResolver = (*WorkflowService)(nil)
)
