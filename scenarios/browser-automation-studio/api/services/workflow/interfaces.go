package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/storage"
)

// CatalogService exposes project and workflow catalog operations (CRUD + versioning + AI assist).
type CatalogService interface {
	CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error)
	ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error)
	GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error)
	UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input WorkflowUpdateInput) (*database.Workflow, error)
	ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*WorkflowVersionSummary, error)
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersionSummary, error)
	RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, changeDescription string) (*database.Workflow, error)
	ModifyWorkflow(ctx context.Context, workflowID uuid.UUID, prompt string, currentFlow map[string]any) (*database.Workflow, error)

	CreateProject(ctx context.Context, project *database.Project) error
	GetProjectByName(ctx context.Context, name string) (*database.Project, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error)
	GetProject(ctx context.Context, projectID uuid.UUID) (*database.Project, error)
	ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error)
	UpdateProject(ctx context.Context, project *database.Project) error
	DeleteProject(ctx context.Context, projectID uuid.UUID) error
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error)
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error)
}

// ExecutionService handles execution lifecycle, telemetry, and health.
type ExecutionService interface {
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
	ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error)
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error)
	GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error)
	StopExecution(ctx context.Context, executionID uuid.UUID) error
	GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error)
	GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error)
	CheckAutomationHealth(ctx context.Context) (bool, error)
}

// ExportService describes replay/export readiness and materialisation.
type ExportService interface {
	DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error)
	ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error
}
