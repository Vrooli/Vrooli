package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Workflow repository methods

func (r *repository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*Workflow, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE project_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`)

	var workflows []*Workflow
	err := r.db.SelectContext(ctx, &workflows, query, projectID, limit, offset)
	if err != nil {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to list workflows by project")
		return nil, fmt.Errorf("failed to list workflows by project: %w", err)
	}

	return workflows, nil
}

func (r *repository) CreateWorkflow(ctx context.Context, workflow *Workflow) error {
	query := `
		INSERT INTO workflows (
			id, project_id, name, folder_path, workflow_type, flow_definition, inputs, outputs, expected_outcome, workflow_metadata,
			description, tags, version, is_template, created_by, last_change_source, last_change_description
		)
		VALUES (
			:id, :project_id, :name, :folder_path, :workflow_type, :flow_definition, :inputs, :outputs, :expected_outcome, :workflow_metadata,
			:description, :tags, :version, :is_template, :created_by, :last_change_source, :last_change_description
		)`

	// Generate ID if not set
	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}

	// Set defaults
	if workflow.Version == 0 {
		workflow.Version = 1
	}
	if workflow.WorkflowType == "" {
		workflow.WorkflowType = "flow"
	}

	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		r.log.WithError(err).Error("Failed to create workflow")
		return fmt.Errorf("failed to create workflow: %w", err)
	}

	return nil
}

func (r *repository) GetWorkflow(ctx context.Context, id uuid.UUID) (*Workflow, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE id = ?`)

	var workflow Workflow
	err := r.db.GetContext(ctx, &workflow, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("id", id).Error("Failed to get workflow")
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return &workflow, nil
}

func (r *repository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*Workflow, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE name = ? AND folder_path = ?`)

	var workflow Workflow
	err := r.db.GetContext(ctx, &workflow, query, name, folderPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithFields(logrus.Fields{
			"name":       name,
			"folderPath": folderPath,
		}).Error("Failed to get workflow by name")
		return nil, fmt.Errorf("failed to get workflow by name: %w", err)
	}

	return &workflow, nil
}

func (r *repository) GetWorkflowByProjectAndName(ctx context.Context, projectID uuid.UUID, name string) (*Workflow, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE project_id = ? AND name = ? LIMIT 1`)

	var workflow Workflow
	err := r.db.GetContext(ctx, &workflow, query, projectID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"name":       name,
		}).Error("Failed to get workflow by project and name")
		return nil, fmt.Errorf("failed to get workflow by project and name: %w", err)
	}

	return &workflow, nil
}

func (r *repository) UpdateWorkflow(ctx context.Context, workflow *Workflow) error {
	query := `
		UPDATE workflows
		SET project_id = :project_id,
		    name = :name,
		    folder_path = :folder_path,
		    workflow_type = :workflow_type,
		    flow_definition = :flow_definition,
		    inputs = :inputs,
		    outputs = :outputs,
		    expected_outcome = :expected_outcome,
		    workflow_metadata = :workflow_metadata,
		    description = :description,
		    tags = :tags,
		    version = :version,
		    last_change_source = :last_change_source,
		    last_change_description = :last_change_description,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	if workflow.WorkflowType == "" {
		workflow.WorkflowType = "flow"
	}

	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		r.log.WithError(err).WithField("id", workflow.ID).Error("Failed to update workflow")
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

func (r *repository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM workflows WHERE id = ?`)

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete workflow")
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}

func (r *repository) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if len(workflowIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM workflows WHERE project_id = ? AND id IN (?)`, projectID, workflowIDs)
	if err != nil {
		return fmt.Errorf("build delete project workflows query: %w", err)
	}
	query = r.db.Rebind(query)

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"project_id":   projectID,
			"workflow_ids": workflowIDs,
		}).Error("Failed to bulk delete project workflows")
		return fmt.Errorf("failed to bulk delete project workflows: %w", err)
	}

	return nil
}

func (r *repository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*Workflow, error) {
	var query string
	var args []any

	if folderPath != "" {
		query = r.db.Rebind(`SELECT * FROM workflows WHERE folder_path = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`)
		args = []any{folderPath, limit, offset}
	} else {
		query = r.db.Rebind(`SELECT * FROM workflows ORDER BY created_at DESC LIMIT ? OFFSET ?`)
		args = []any{limit, offset}
	}

	var workflows []*Workflow
	err := r.db.SelectContext(ctx, &workflows, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to list workflows")
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	return workflows, nil
}

// Workflow version repository methods

func (r *repository) CreateWorkflowVersion(ctx context.Context, version *WorkflowVersion) error {
	query := `
		INSERT INTO workflow_versions (id, workflow_id, version, flow_definition, change_description, created_by, created_at)
		VALUES (:id, :workflow_id, :version, :flow_definition, :change_description, :created_by, CURRENT_TIMESTAMP)
	`

	if version.ID == uuid.Nil {
		version.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, version)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"workflow_id": version.WorkflowID,
			"version":     version.Version,
		}).Error("Failed to create workflow version")
		return fmt.Errorf("failed to create workflow version: %w", err)
	}

	return nil
}

func (r *repository) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersion, error) {
	query := r.db.Rebind(`SELECT * FROM workflow_versions WHERE workflow_id = ? AND version = ?`)

	var wfVersion WorkflowVersion
	err := r.db.GetContext(ctx, &wfVersion, query, workflowID, version)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"workflow_id": workflowID,
			"version":     version,
		}).Error("Failed to get workflow version")
		return nil, fmt.Errorf("failed to get workflow version: %w", err)
	}

	return &wfVersion, nil
}

func (r *repository) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*WorkflowVersion, error) {
	query := r.db.Rebind(`SELECT * FROM workflow_versions WHERE workflow_id = ? ORDER BY version DESC LIMIT ? OFFSET ?`)

	var versions []*WorkflowVersion
	err := r.db.SelectContext(ctx, &versions, query, workflowID, limit, offset)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"workflow_id": workflowID,
			"limit":       limit,
			"offset":      offset,
		}).Error("Failed to list workflow versions")
		return nil, fmt.Errorf("failed to list workflow versions: %w", err)
	}

	return versions, nil
}
