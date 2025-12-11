package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Repository defines the interface for database operations
type Repository interface {
	// Project operations
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, id uuid.UUID) (*Project, error)
	GetProjectByName(ctx context.Context, name string) (*Project, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, id uuid.UUID) error
	ListProjects(ctx context.Context, limit, offset int) ([]*Project, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*ProjectStats, error)

	// Workflow operations
	CreateWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, id uuid.UUID) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name, folderPath string) (*Workflow, error)
	GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error
	DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error
	ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*Workflow, error)
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Workflow, error)
	CreateWorkflowVersion(ctx context.Context, version *WorkflowVersion) error
	GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersion, error)
	ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*WorkflowVersion, error)

	// Execution operations
	CreateExecution(ctx context.Context, execution *Execution) error
	GetExecution(ctx context.Context, id uuid.UUID) (*Execution, error)
	UpdateExecution(ctx context.Context, execution *Execution) error
	DeleteExecution(ctx context.Context, id uuid.UUID) error
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*Execution, error)
	CreateExecutionStep(ctx context.Context, step *ExecutionStep) error
	UpdateExecutionStep(ctx context.Context, step *ExecutionStep) error
	ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*ExecutionStep, error)
	CreateExecutionArtifact(ctx context.Context, artifact *ExecutionArtifact) error
	ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*ExecutionArtifact, error)

	// Screenshot operations
	CreateScreenshot(ctx context.Context, screenshot *Screenshot) error
	GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*Screenshot, error)

	// Log operations
	CreateExecutionLog(ctx context.Context, log *ExecutionLog) error
	GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*ExecutionLog, error)

	// Extracted data operations
	CreateExtractedData(ctx context.Context, data *ExtractedData) error
	GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*ExtractedData, error)

	// Folder operations
	CreateFolder(ctx context.Context, folder *WorkflowFolder) error
	GetFolder(ctx context.Context, path string) (*WorkflowFolder, error)
	ListFolders(ctx context.Context) ([]*WorkflowFolder, error)

	// Export operations
	CreateExport(ctx context.Context, export *Export) error
	GetExport(ctx context.Context, id uuid.UUID) (*Export, error)
	UpdateExport(ctx context.Context, export *Export) error
	DeleteExport(ctx context.Context, id uuid.UUID) error
	ListExports(ctx context.Context, limit, offset int) ([]*ExportWithDetails, error)
	ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*Export, error)
	ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*Export, error)

	// Recovery operations for progress continuity
	FindStaleExecutions(ctx context.Context, staleThreshold time.Duration) ([]*Execution, error)
	MarkExecutionInterrupted(ctx context.Context, id uuid.UUID, reason string) error
	GetLastSuccessfulStepIndex(ctx context.Context, executionID uuid.UUID) (int, error)
	UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error
	GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*ExecutionStep, error)
	GetResumableExecution(ctx context.Context, id uuid.UUID) (*Execution, int, error)
}

// repository implements the Repository interface
type repository struct {
	db  *DB
	log *logrus.Logger
}

// NewRepository creates a new repository instance
func NewRepository(db *DB, log *logrus.Logger) Repository {
	return &repository{
		db:  db,
		log: log,
	}
}

// Implementation methods are organized across focused files:
// - repository_projects.go: Project CRUD and stats operations
// - repository_workflows.go: Workflow and workflow version operations
// - repository_executions.go: Execution, step, and artifact operations
// - repository_artifacts.go: Screenshot, log, and extracted data operations
// - repository_folders.go: Folder operations

// Compile-time interface enforcement
var _ Repository = (*repository)(nil)
