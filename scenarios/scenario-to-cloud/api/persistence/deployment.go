package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"scenario-to-cloud/domain"
)

// CreateDeployment inserts a new deployment record.
func (r *Repository) CreateDeployment(ctx context.Context, d *domain.Deployment) error {
	const q = `
		INSERT INTO deployments (
			id, name, scenario_id, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			error_message, error_step,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10,
			$11, $12
		)
	`
	_, err := r.db.ExecContext(ctx, q,
		d.ID,
		d.Name,
		d.ScenarioID,
		d.Status,
		d.Manifest,
		d.BundlePath,
		d.BundleSHA256,
		d.BundleSizeBytes,
		d.ErrorMessage,
		d.ErrorStep,
		d.CreatedAt,
		d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	return nil
}

// GetDeployment retrieves a deployment by ID.
func (r *Repository) GetDeployment(ctx context.Context, id string) (*domain.Deployment, error) {
	const q = `
		SELECT
			id, name, scenario_id, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			setup_result, deploy_result, last_inspect_result,
			error_message, error_step,
			progress_step, progress_percent,
			created_at, updated_at, last_deployed_at, last_inspected_at
		FROM deployments
		WHERE id = $1
	`
	d := &domain.Deployment{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&d.ID,
		&d.Name,
		&d.ScenarioID,
		&d.Status,
		&d.Manifest,
		&d.BundlePath,
		&d.BundleSHA256,
		&d.BundleSizeBytes,
		&d.SetupResult,
		&d.DeployResult,
		&d.LastInspectResult,
		&d.ErrorMessage,
		&d.ErrorStep,
		&d.ProgressStep,
		&d.ProgressPercent,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.LastDeployedAt,
		&d.LastInspectedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	return d, nil
}

// GetDeploymentByHostAndScenario finds an existing deployment for the same host+scenario combination.
func (r *Repository) GetDeploymentByHostAndScenario(ctx context.Context, host, scenarioID string) (*domain.Deployment, error) {
	const q = `
		SELECT
			id, name, scenario_id, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			setup_result, deploy_result, last_inspect_result,
			error_message, error_step,
			progress_step, progress_percent,
			created_at, updated_at, last_deployed_at, last_inspected_at
		FROM deployments
		WHERE scenario_id = $1
		  AND manifest->'target'->'vps'->>'host' = $2
		ORDER BY created_at DESC
		LIMIT 1
	`
	d := &domain.Deployment{}
	err := r.db.QueryRowContext(ctx, q, scenarioID, host).Scan(
		&d.ID,
		&d.Name,
		&d.ScenarioID,
		&d.Status,
		&d.Manifest,
		&d.BundlePath,
		&d.BundleSHA256,
		&d.BundleSizeBytes,
		&d.SetupResult,
		&d.DeployResult,
		&d.LastInspectResult,
		&d.ErrorMessage,
		&d.ErrorStep,
		&d.ProgressStep,
		&d.ProgressPercent,
		&d.CreatedAt,
		&d.UpdatedAt,
		&d.LastDeployedAt,
		&d.LastInspectedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment by host and scenario: %w", err)
	}
	return d, nil
}

// ListDeployments retrieves deployments with optional filtering.
func (r *Repository) ListDeployments(ctx context.Context, filter domain.ListFilter) ([]*domain.Deployment, error) {
	q := `
		SELECT
			id, name, scenario_id, status, manifest,
			bundle_path, bundle_sha256, bundle_size_bytes,
			setup_result, deploy_result, last_inspect_result,
			error_message, error_step,
			progress_step, progress_percent,
			created_at, updated_at, last_deployed_at, last_inspected_at
		FROM deployments
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if filter.Status != nil {
		q += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *filter.Status)
		argNum++
	}
	if filter.ScenarioID != nil {
		q += fmt.Sprintf(" AND scenario_id = $%d", argNum)
		args = append(args, *filter.ScenarioID)
	}

	q += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*domain.Deployment
	for rows.Next() {
		d := &domain.Deployment{}
		if err := rows.Scan(
			&d.ID,
			&d.Name,
			&d.ScenarioID,
			&d.Status,
			&d.Manifest,
			&d.BundlePath,
			&d.BundleSHA256,
			&d.BundleSizeBytes,
			&d.SetupResult,
			&d.DeployResult,
			&d.LastInspectResult,
			&d.ErrorMessage,
			&d.ErrorStep,
			&d.ProgressStep,
			&d.ProgressPercent,
			&d.CreatedAt,
			&d.UpdatedAt,
			&d.LastDeployedAt,
			&d.LastInspectedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		deployments = append(deployments, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating deployments: %w", err)
	}

	return deployments, nil
}

// UpdateDeployment updates an existing deployment record.
func (r *Repository) UpdateDeployment(ctx context.Context, d *domain.Deployment) error {
	const q = `
		UPDATE deployments SET
			name = $2,
			scenario_id = $3,
			status = $4,
			manifest = $5,
			bundle_path = $6,
			bundle_sha256 = $7,
			bundle_size_bytes = $8,
			setup_result = $9,
			deploy_result = $10,
			last_inspect_result = $11,
			error_message = $12,
			error_step = $13,
			updated_at = $14,
			last_deployed_at = $15,
			last_inspected_at = $16
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, q,
		d.ID,
		d.Name,
		d.ScenarioID,
		d.Status,
		d.Manifest,
		d.BundlePath,
		d.BundleSHA256,
		d.BundleSizeBytes,
		d.SetupResult,
		d.DeployResult,
		d.LastInspectResult,
		d.ErrorMessage,
		d.ErrorStep,
		d.UpdatedAt,
		d.LastDeployedAt,
		d.LastInspectedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deployment not found: %s", d.ID)
	}

	return nil
}

// UpdateDeploymentStatus updates just the status and optional error fields.
func (r *Repository) UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error {
	const q = `
		UPDATE deployments SET
			status = $2,
			error_message = $3,
			error_step = $4,
			updated_at = $5
		WHERE id = $1
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, q, id, status, errorMsg, errorStep, now)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deployment not found: %s", id)
	}

	return nil
}

// UpdateDeploymentSetupResult stores the VPS setup result.
func (r *Repository) UpdateDeploymentSetupResult(ctx context.Context, id string, result json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			setup_result = $2,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, result, now)
	if err != nil {
		return fmt.Errorf("failed to update setup result: %w", err)
	}
	return nil
}

// UpdateDeploymentDeployResult stores the VPS deploy result.
func (r *Repository) UpdateDeploymentDeployResult(ctx context.Context, id string, result json.RawMessage, success bool) error {
	const q = `
		UPDATE deployments SET
			deploy_result = $2,
			status = $3,
			last_deployed_at = $4,
			updated_at = $4
		WHERE id = $1
	`
	now := time.Now()
	status := domain.StatusDeployed
	if !success {
		status = domain.StatusFailed
	}
	_, err := r.db.ExecContext(ctx, q, id, result, status, now)
	if err != nil {
		return fmt.Errorf("failed to update deploy result: %w", err)
	}
	return nil
}

// UpdateDeploymentInspectResult stores the latest VPS inspect result.
func (r *Repository) UpdateDeploymentInspectResult(ctx context.Context, id string, result json.RawMessage) error {
	const q = `
		UPDATE deployments SET
			last_inspect_result = $2,
			last_inspected_at = $3,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, result, now)
	if err != nil {
		return fmt.Errorf("failed to update inspect result: %w", err)
	}
	return nil
}

// UpdateDeploymentBundle stores bundle information after building.
func (r *Repository) UpdateDeploymentBundle(ctx context.Context, id, bundlePath, bundleSHA256 string, sizeBytes int64) error {
	const q = `
		UPDATE deployments SET
			bundle_path = $2,
			bundle_sha256 = $3,
			bundle_size_bytes = $4,
			updated_at = $5
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, bundlePath, bundleSHA256, sizeBytes, now)
	if err != nil {
		return fmt.Errorf("failed to update bundle info: %w", err)
	}
	return nil
}

// DeleteDeployment removes a deployment record.
func (r *Repository) DeleteDeployment(ctx context.Context, id string) error {
	const q = `DELETE FROM deployments WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deployment not found: %s", id)
	}

	return nil
}

// CountDeploymentsByBundleSHA256 returns the number of deployments using a specific bundle.
// Used to check if a bundle can be safely deleted (no other deployments reference it).
func (r *Repository) CountDeploymentsByBundleSHA256(ctx context.Context, sha256 string) (int, error) {
	const q = `SELECT COUNT(*) FROM deployments WHERE bundle_sha256 = $1`
	var count int
	if err := r.db.QueryRowContext(ctx, q, sha256).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count deployments: %w", err)
	}
	return count, nil
}

// UpdateDeploymentProgress updates the current progress step and percentage.
func (r *Repository) UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error {
	const q = `
		UPDATE deployments SET
			progress_step = $2,
			progress_percent = $3,
			updated_at = $4
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, step, percent, now)
	if err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}
	return nil
}

// ResetDeploymentProgress clears the progress fields when starting a new deployment.
func (r *Repository) ResetDeploymentProgress(ctx context.Context, id string) error {
	const q = `
		UPDATE deployments SET
			progress_step = NULL,
			progress_percent = 0,
			updated_at = $2
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, now)
	if err != nil {
		return fmt.Errorf("failed to reset progress: %w", err)
	}
	return nil
}

// AppendHistoryEvent adds a new event to the deployment history.
func (r *Repository) AppendHistoryEvent(ctx context.Context, id string, event domain.HistoryEvent) error {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal history event: %w", err)
	}

	const q = `
		UPDATE deployments SET
			deployment_history = COALESCE(deployment_history, '[]'::jsonb) || $2::jsonb,
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err = r.db.ExecContext(ctx, q, id, eventJSON, now)
	if err != nil {
		return fmt.Errorf("failed to append history event: %w", err)
	}
	return nil
}

// GetDeploymentHistory retrieves the deployment history for a deployment.
func (r *Repository) GetDeploymentHistory(ctx context.Context, id string) ([]domain.HistoryEvent, error) {
	const q = `
		SELECT COALESCE(deployment_history, '[]'::jsonb)
		FROM deployments
		WHERE id = $1
	`
	var historyJSON []byte
	err := r.db.QueryRowContext(ctx, q, id).Scan(&historyJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment history: %w", err)
	}

	var history []domain.HistoryEvent
	if err := json.Unmarshal(historyJSON, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment history: %w", err)
	}

	return history, nil
}

// StartDeploymentRun begins a new deployment execution run with a fresh run_id.
// This clears completed_steps and sets status atomically to prevent race conditions.
// Returns the new run_id, or error if deployment is already running.
func (r *Repository) StartDeploymentRun(ctx context.Context, id, runID string) error {
	const q = `
		UPDATE deployments SET
			run_id = $2,
			completed_steps = '[]'::jsonb,
			status = 'setup_running',
			error_message = NULL,
			error_step = NULL,
			progress_step = NULL,
			progress_percent = 0,
			updated_at = $3
		WHERE id = $1
		  AND status NOT IN ('setup_running', 'deploying')
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, q, id, runID, now)
	if err != nil {
		return fmt.Errorf("failed to start deployment run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("deployment is already running or not found")
	}

	return nil
}

// MarkStepCompleted records a step as completed for idempotent replay.
// Uses JSONB append to add the step ID to completed_steps array.
func (r *Repository) MarkStepCompleted(ctx context.Context, id, stepID string) error {
	const q = `
		UPDATE deployments SET
			completed_steps = COALESCE(completed_steps, '[]'::jsonb) || to_jsonb($2::text),
			updated_at = $3
		WHERE id = $1
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, q, id, stepID, now)
	if err != nil {
		return fmt.Errorf("failed to mark step completed: %w", err)
	}
	return nil
}

// GetCompletedSteps retrieves the list of completed step IDs for a deployment.
func (r *Repository) GetCompletedSteps(ctx context.Context, id string) ([]string, error) {
	const q = `
		SELECT COALESCE(completed_steps, '[]'::jsonb)
		FROM deployments
		WHERE id = $1
	`
	var stepsJSON []byte
	err := r.db.QueryRowContext(ctx, q, id).Scan(&stepsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get completed steps: %w", err)
	}

	var steps []string
	if err := json.Unmarshal(stepsJSON, &steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal completed steps: %w", err)
	}

	return steps, nil
}

// GetRunID retrieves the current run_id for a deployment.
func (r *Repository) GetRunID(ctx context.Context, id string) (*string, error) {
	const q = `SELECT run_id FROM deployments WHERE id = $1`
	var runID *string
	err := r.db.QueryRowContext(ctx, q, id).Scan(&runID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get run_id: %w", err)
	}
	return runID, nil
}
