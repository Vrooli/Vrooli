package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"scenario-to-cloud/domain"
)

// CreateInvestigation inserts a new investigation record.
func (r *Repository) CreateInvestigation(ctx context.Context, inv *domain.Investigation) error {
	const q = `
		INSERT INTO deployment_investigations (
			id, deployment_id, deployment_run_id, status, findings, progress,
			details, agent_run_id, error_message,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11
		)
	`
	_, err := r.db.ExecContext(ctx, q,
		inv.ID,
		inv.DeploymentID,
		inv.DeploymentRunID,
		inv.Status,
		inv.Findings,
		inv.Progress,
		inv.Details,
		inv.AgentRunID,
		inv.ErrorMessage,
		inv.CreatedAt,
		inv.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create investigation: %w", err)
	}
	return nil
}

// GetInvestigation retrieves an investigation by ID.
func (r *Repository) GetInvestigation(ctx context.Context, id string) (*domain.Investigation, error) {
	const q = `
		SELECT
			id, deployment_id, deployment_run_id, status, findings, progress,
			details, agent_run_id, error_message,
			created_at, updated_at, completed_at
		FROM deployment_investigations
		WHERE id = $1
	`
	inv := &domain.Investigation{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&inv.ID,
		&inv.DeploymentID,
		&inv.DeploymentRunID,
		&inv.Status,
		&inv.Findings,
		&inv.Progress,
		&inv.Details,
		&inv.AgentRunID,
		&inv.ErrorMessage,
		&inv.CreatedAt,
		&inv.UpdatedAt,
		&inv.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get investigation: %w", err)
	}
	return inv, nil
}

// GetInvestigationForDeployment retrieves an investigation by ID scoped to a deployment.
func (r *Repository) GetInvestigationForDeployment(ctx context.Context, deploymentID, id string) (*domain.Investigation, error) {
	const q = `
		SELECT
			id, deployment_id, deployment_run_id, status, findings, progress,
			details, agent_run_id, error_message,
			created_at, updated_at, completed_at
		FROM deployment_investigations
		WHERE id = $1 AND deployment_id = $2
	`
	inv := &domain.Investigation{}
	err := r.db.QueryRowContext(ctx, q, id, deploymentID).Scan(
		&inv.ID,
		&inv.DeploymentID,
		&inv.DeploymentRunID,
		&inv.Status,
		&inv.Findings,
		&inv.Progress,
		&inv.Details,
		&inv.AgentRunID,
		&inv.ErrorMessage,
		&inv.CreatedAt,
		&inv.UpdatedAt,
		&inv.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get investigation: %w", err)
	}
	return inv, nil
}

// ListInvestigations retrieves investigations for a deployment, ordered by creation time (newest first).
func (r *Repository) ListInvestigations(ctx context.Context, deploymentID string, limit int) ([]*domain.Investigation, error) {
	q := `
		SELECT
			id, deployment_id, deployment_run_id, status, findings, progress,
			details, agent_run_id, error_message,
			created_at, updated_at, completed_at
		FROM deployment_investigations
		WHERE deployment_id = $1
		ORDER BY created_at DESC
	`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := r.db.QueryContext(ctx, q, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list investigations: %w", err)
	}
	defer rows.Close()

	var investigations []*domain.Investigation
	for rows.Next() {
		inv := &domain.Investigation{}
		if err := rows.Scan(
			&inv.ID,
			&inv.DeploymentID,
			&inv.DeploymentRunID,
			&inv.Status,
			&inv.Findings,
			&inv.Progress,
			&inv.Details,
			&inv.AgentRunID,
			&inv.ErrorMessage,
			&inv.CreatedAt,
			&inv.UpdatedAt,
			&inv.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan investigation: %w", err)
		}
		investigations = append(investigations, inv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating investigations: %w", err)
	}

	return investigations, nil
}

// GetActiveInvestigation returns the currently running investigation for a deployment, if any.
func (r *Repository) GetActiveInvestigation(ctx context.Context, deploymentID string) (*domain.Investigation, error) {
	const q = `
		SELECT
			id, deployment_id, deployment_run_id, status, findings, progress,
			details, agent_run_id, error_message,
			created_at, updated_at, completed_at
		FROM deployment_investigations
		WHERE deployment_id = $1
		  AND status IN ('pending', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`
	inv := &domain.Investigation{}
	err := r.db.QueryRowContext(ctx, q, deploymentID).Scan(
		&inv.ID,
		&inv.DeploymentID,
		&inv.DeploymentRunID,
		&inv.Status,
		&inv.Findings,
		&inv.Progress,
		&inv.Details,
		&inv.AgentRunID,
		&inv.ErrorMessage,
		&inv.CreatedAt,
		&inv.UpdatedAt,
		&inv.CompletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active investigation: %w", err)
	}
	return inv, nil
}

// UpdateInvestigationStatus updates the status of an investigation.
func (r *Repository) UpdateInvestigationStatus(ctx context.Context, id string, status domain.InvestigationStatus) error {
	previous, err := r.validateInvestigationTransition(ctx, id, status)
	if err != nil {
		return err
	}
	now := time.Now()
	var completedAt *time.Time
	if status == domain.InvestigationStatusCompleted || status == domain.InvestigationStatusFailed || status == domain.InvestigationStatusCancelled {
		completedAt = &now
	}

	const q = `
		UPDATE deployment_investigations SET
			status = $2,
			updated_at = $3,
			completed_at = $4
		WHERE id = $1 AND status = $5
	`
	result, err := r.db.ExecContext(ctx, q, id, status, now, completedAt, previous)
	if err != nil {
		return fmt.Errorf("failed to update investigation status: %w", err)
	}
	return requireInvestigationTransitionApplied(result, id, previous, status)
}

// validateInvestigationTransition loads the persisted source state and asks the
// domain owner to validate the requested edge. The later UPDATE compares that
// source state again, so concurrent completion/cancellation cannot overwrite a
// newer decision.
func (r *Repository) validateInvestigationTransition(ctx context.Context, id string, next domain.InvestigationStatus) (domain.InvestigationStatus, error) {
	var previous domain.InvestigationStatus
	if err := r.db.QueryRowContext(ctx, `SELECT status FROM deployment_investigations WHERE id = $1`, id).Scan(&previous); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("investigation not found: %s", id)
		}
		return "", fmt.Errorf("failed to read investigation status: %w", err)
	}
	if err := (&domain.Investigation{Status: previous}).TransitionStatus(next); err != nil {
		return "", err
	}
	return previous, nil
}

func requireInvestigationTransitionApplied(result sql.Result, id string, previous, next domain.InvestigationStatus) error {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("investigation transition %q -> %q was superseded for %s", previous, next, id)
	}
	return nil
}

// UpdateInvestigationRunID sets the agent run ID for tracking.
func (r *Repository) UpdateInvestigationRunID(ctx context.Context, id, runID string) error {
	const q = `
		UPDATE deployment_investigations SET
			agent_run_id = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, runID, now)
	if err != nil {
		return fmt.Errorf("failed to update investigation run ID: %w", err)
	}
	return nil
}

// UpdateInvestigationFindings stores the investigation findings and details.
// Also sets progress to 100 and status to completed.
func (r *Repository) UpdateInvestigationFindings(ctx context.Context, id, findings string, details json.RawMessage) error {
	previous, err := r.validateInvestigationTransition(ctx, id, domain.InvestigationStatusCompleted)
	if err != nil {
		return err
	}
	now := time.Now()
	const q = `
		UPDATE deployment_investigations SET
			findings = $2,
			details = $3,
			status = $4,
			progress = 100,
			updated_at = $5,
			completed_at = $5
		WHERE id = $1 AND status = $6
	`
	result, err := r.db.ExecContext(ctx, q, id, findings, details, domain.InvestigationStatusCompleted, now, previous)
	if err != nil {
		return fmt.Errorf("failed to update investigation findings: %w", err)
	}
	return requireInvestigationTransitionApplied(result, id, previous, domain.InvestigationStatusCompleted)
}

// UpdateInvestigationProgress updates the progress percentage.
func (r *Repository) UpdateInvestigationProgress(ctx context.Context, id string, progress int) error {
	const q = `
		UPDATE deployment_investigations SET
			progress = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, progress, now)
	if err != nil {
		return fmt.Errorf("failed to update investigation progress: %w", err)
	}
	return nil
}

// UpdateInvestigationError sets an error message and marks the investigation as failed.
func (r *Repository) UpdateInvestigationError(ctx context.Context, id, errorMsg string) error {
	previous, err := r.validateInvestigationTransition(ctx, id, domain.InvestigationStatusFailed)
	if err != nil {
		return err
	}
	now := time.Now()
	const q = `
		UPDATE deployment_investigations SET
			status = $2,
			error_message = $3,
			updated_at = $4,
			completed_at = $4
		WHERE id = $1 AND status = $5
	`
	result, err := r.db.ExecContext(ctx, q, id, domain.InvestigationStatusFailed, errorMsg, now, previous)
	if err != nil {
		return fmt.Errorf("failed to update investigation error: %w", err)
	}
	return requireInvestigationTransitionApplied(result, id, previous, domain.InvestigationStatusFailed)
}

// UpdateInvestigationErrorWithDetails sets an error message and details, then marks the investigation as failed.
// This is useful when the agent ran but produced no output - we still want to capture execution metadata.
func (r *Repository) UpdateInvestigationErrorWithDetails(ctx context.Context, id, errorMsg string, details json.RawMessage) error {
	previous, err := r.validateInvestigationTransition(ctx, id, domain.InvestigationStatusFailed)
	if err != nil {
		return err
	}
	now := time.Now()
	const q = `
		UPDATE deployment_investigations SET
			status = $2,
			error_message = $3,
			details = $4,
			updated_at = $5,
			completed_at = $5
		WHERE id = $1 AND status = $6
	`
	result, err := r.db.ExecContext(ctx, q, id, domain.InvestigationStatusFailed, errorMsg, details, now, previous)
	if err != nil {
		return fmt.Errorf("failed to update investigation error with details: %w", err)
	}
	return requireInvestigationTransitionApplied(result, id, previous, domain.InvestigationStatusFailed)
}

// DeleteInvestigation removes an investigation record.
func (r *Repository) DeleteInvestigation(ctx context.Context, id string) error {
	const q = `DELETE FROM deployment_investigations WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("failed to delete investigation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("investigation not found: %s", id)
	}

	return nil
}
