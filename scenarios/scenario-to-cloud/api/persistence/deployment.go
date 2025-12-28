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
			created_at, updated_at, last_deployed_at, last_inspected_at
		FROM deployments
		WHERE scenario_id = $1
		  AND manifest->>'target'->>'vps'->>'host' = $2
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
		argNum++
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
