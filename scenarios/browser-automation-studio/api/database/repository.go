package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Repository defines the interface for database operations
type Repository interface {
	// Workflow operations
	CreateWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflow(ctx context.Context, id uuid.UUID) (*Workflow, error)
	GetWorkflowByName(ctx context.Context, name, folderPath string) (*Workflow, error)
	UpdateWorkflow(ctx context.Context, workflow *Workflow) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error
	ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*Workflow, error)

	// Execution operations
	CreateExecution(ctx context.Context, execution *Execution) error
	GetExecution(ctx context.Context, id uuid.UUID) (*Execution, error)
	UpdateExecution(ctx context.Context, execution *Execution) error
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*Execution, error)

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

// Workflow operations

func (r *repository) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	query := `
		INSERT INTO workflows (id, name, folder_path, flow_definition, description, tags, version, is_template, created_by)
		VALUES (:id, :name, :folder_path, :flow_definition, :description, :tags, :version, :is_template, :created_by)`

	// Generate ID if not set
	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}

	// Set defaults
	if workflow.Version == 0 {
		workflow.Version = 1
	}
	
	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		r.log.WithError(err).Error("Failed to create workflow")
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

func (r *repository) GetWorkflow(ctx context.Context, id uuid.UUID) (*Workflow, error) {
	query := `SELECT * FROM workflows WHERE id = $1`
	
	var workflow Workflow
	err := r.db.GetContext(ctx, &workflow, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to get workflow")
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return &workflow, nil
}

func (r *repository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*Workflow, error) {
	query := `SELECT * FROM workflows WHERE name = $1 AND folder_path = $2`
	
	var workflow Workflow
	err := r.db.GetContext(ctx, &workflow, query, name, folderPath)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"name": name,
			"folderPath": folderPath,
		}).Error("Failed to get workflow by name")
		return nil, fmt.Errorf("failed to get workflow by name: %w", err)
	}

	return &workflow, nil
}

func (r *repository) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	query := `
		UPDATE workflows 
		SET name = :name, folder_path = :folder_path, flow_definition = :flow_definition, 
		    description = :description, tags = :tags, version = :version, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		r.log.WithError(err).WithField("id", workflow.ID).Error("Failed to update workflow")
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

func (r *repository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workflows WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete workflow")
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}

func (r *repository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*Workflow, error) {
	var query string
	var args []interface{}

	if folderPath != "" {
		query = `SELECT * FROM workflows WHERE folder_path = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{folderPath, limit, offset}
	} else {
		query = `SELECT * FROM workflows ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	var workflows []*Workflow
	err := r.db.SelectContext(ctx, &workflows, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to list workflows")
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	return workflows, nil
}

// Execution operations

func (r *repository) CreateExecution(ctx context.Context, execution *Execution) error {
	query := `
		INSERT INTO executions (id, workflow_id, workflow_version, status, trigger_type, 
		                       trigger_metadata, parameters, progress, current_step)
		VALUES (:id, :workflow_id, :workflow_version, :status, :trigger_type, 
		        :trigger_metadata, :parameters, :progress, :current_step)`

	// Generate ID if not set
	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		r.log.WithError(err).Error("Failed to create execution")
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (r *repository) GetExecution(ctx context.Context, id uuid.UUID) (*Execution, error) {
	query := `SELECT * FROM executions WHERE id = $1`
	
	var execution Execution
	err := r.db.GetContext(ctx, &execution, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

func (r *repository) UpdateExecution(ctx context.Context, execution *Execution) error {
	query := `
		UPDATE executions 
		SET status = :status, completed_at = :completed_at, error = :error, 
		    result = :result, progress = :progress, current_step = :current_step
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		r.log.WithError(err).WithField("id", execution.ID).Error("Failed to update execution")
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

func (r *repository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*Execution, error) {
	var query string
	var args []interface{}

	if workflowID != nil {
		query = `SELECT * FROM executions WHERE workflow_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{*workflowID, limit, offset}
	} else {
		query = `SELECT * FROM executions ORDER BY started_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	}

	var executions []*Execution
	err := r.db.SelectContext(ctx, &executions, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to list executions")
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// Screenshot operations

func (r *repository) CreateScreenshot(ctx context.Context, screenshot *Screenshot) error {
	query := `
		INSERT INTO screenshots (id, execution_id, step_name, storage_url, thumbnail_url, 
		                        width, height, size_bytes, metadata)
		VALUES (:id, :execution_id, :step_name, :storage_url, :thumbnail_url, 
		        :width, :height, :size_bytes, :metadata)`

	// Generate ID if not set
	if screenshot.ID == uuid.Nil {
		screenshot.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, screenshot)
	if err != nil {
		r.log.WithError(err).Error("Failed to create screenshot")
		return fmt.Errorf("failed to create screenshot: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionScreenshots(ctx context.Context, executionID uuid.UUID) ([]*Screenshot, error) {
	query := `SELECT * FROM screenshots WHERE execution_id = $1 ORDER BY timestamp ASC`
	
	var screenshots []*Screenshot
	err := r.db.SelectContext(ctx, &screenshots, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		return nil, fmt.Errorf("failed to get screenshots: %w", err)
	}

	return screenshots, nil
}

// Log operations

func (r *repository) CreateExecutionLog(ctx context.Context, log *ExecutionLog) error {
	query := `
		INSERT INTO execution_logs (id, execution_id, level, step_name, message, metadata)
		VALUES (:id, :execution_id, :level, :step_name, :message, :metadata)`

	// Generate ID if not set
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, log)
	if err != nil {
		r.log.WithError(err).Error("Failed to create execution log")
		return fmt.Errorf("failed to create execution log: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]*ExecutionLog, error) {
	query := `SELECT * FROM execution_logs WHERE execution_id = $1 ORDER BY timestamp ASC`
	
	var logs []*ExecutionLog
	err := r.db.SelectContext(ctx, &logs, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution logs")
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}

// Extracted data operations

func (r *repository) CreateExtractedData(ctx context.Context, data *ExtractedData) error {
	query := `
		INSERT INTO extracted_data (id, execution_id, step_name, data_key, data_value, data_type, metadata)
		VALUES (:id, :execution_id, :step_name, :data_key, :data_value, :data_type, :metadata)`

	// Generate ID if not set
	if data.ID == uuid.Nil {
		data.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, data)
	if err != nil {
		r.log.WithError(err).Error("Failed to create extracted data")
		return fmt.Errorf("failed to create extracted data: %w", err)
	}

	return nil
}

func (r *repository) GetExecutionExtractedData(ctx context.Context, executionID uuid.UUID) ([]*ExtractedData, error) {
	query := `SELECT * FROM extracted_data WHERE execution_id = $1 ORDER BY timestamp ASC`
	
	var data []*ExtractedData
	err := r.db.SelectContext(ctx, &data, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get extracted data")
		return nil, fmt.Errorf("failed to get extracted data: %w", err)
	}

	return data, nil
}

// Folder operations

func (r *repository) CreateFolder(ctx context.Context, folder *WorkflowFolder) error {
	query := `
		INSERT INTO workflow_folders (id, path, parent_id, name, description)
		VALUES (:id, :path, :parent_id, :name, :description)`

	// Generate ID if not set
	if folder.ID == uuid.Nil {
		folder.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, folder)
	if err != nil {
		r.log.WithError(err).Error("Failed to create folder")
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}

func (r *repository) GetFolder(ctx context.Context, path string) (*WorkflowFolder, error) {
	query := `SELECT * FROM workflow_folders WHERE path = $1`
	
	var folder WorkflowFolder
	err := r.db.GetContext(ctx, &folder, query, path)
	if err != nil {
		r.log.WithError(err).WithField("path", path).Error("Failed to get folder")
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	return &folder, nil
}

func (r *repository) ListFolders(ctx context.Context) ([]*WorkflowFolder, error) {
	query := `SELECT * FROM workflow_folders ORDER BY path ASC`
	
	var folders []*WorkflowFolder
	err := r.db.SelectContext(ctx, &folders, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to list folders")
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}