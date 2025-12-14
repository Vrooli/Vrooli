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

// AdhocExecutionParams contains typed execution parameters for namespace-aware adhoc execution.
// These parameters support the ${@namespace/path} variable interpolation system.
type AdhocExecutionParams struct {
	// FlowDefinition is the workflow to execute.
	FlowDefinition map[string]any
	// Name is the human-readable name for the execution.
	Name string
	// ProjectRoot is the absolute path to the project root for workflowPath resolution.
	// Example: "/home/user/Vrooli/scenarios/my-scenario/bas"
	ProjectRoot string
	// InitialParams are read-only input parameters (@params/ namespace).
	// Subflows inherit parent's params unless explicitly overridden.
	InitialParams map[string]any
	// InitialStore is pre-seeded mutable runtime state (@store/ namespace).
	// Modified via setVariable steps and storeResult params.
	InitialStore map[string]any
	// Env contains project/user configuration (@env/ namespace).
	// Read-only, inherited by all subflows unchanged.
	Env map[string]any
	// LegacyParameters is the deprecated flat parameters map for backward compatibility.
	// If InitialStore is empty, this maps to @store/ namespace.
	LegacyParameters map[string]any
}

// ExecutionService handles execution lifecycle, telemetry, and health.
type ExecutionService interface {
	ExecuteWorkflow(ctx context.Context, workflowID uuid.UUID, parameters map[string]any) (*database.Execution, error)
	ExecuteAdhocWorkflow(ctx context.Context, flowDefinition map[string]any, parameters map[string]any, name string) (*database.Execution, error)
	// ExecuteAdhocWorkflowWithParams executes an adhoc workflow with namespace-aware parameters.
	// This is the preferred method for callers that support the new variable interpolation system.
	ExecuteAdhocWorkflowWithParams(ctx context.Context, params AdhocExecutionParams) (*database.Execution, error)
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*database.Execution, error)
	GetExecution(ctx context.Context, executionID uuid.UUID) (*database.Execution, error)
	StopExecution(ctx context.Context, executionID uuid.UUID) error
	ResumeExecution(ctx context.Context, executionID uuid.UUID, parameters map[string]any) (*database.Execution, error)
	GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*database.Screenshot, error)
	GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*export.ExecutionTimeline, error)
	CheckAutomationHealth(ctx context.Context) (bool, error)
}

// ExportService describes replay/export readiness and materialisation.
type ExportService interface {
	DescribeExecutionExport(ctx context.Context, executionID uuid.UUID) (*ExecutionExportPreview, error)
	ExportToFolder(ctx context.Context, executionID uuid.UUID, outputDir string, storageClient storage.StorageInterface) error
}
