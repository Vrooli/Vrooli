package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"scenario-to-cloud/domain"
)

// GetCloudRecoveryOperation returns the durable owner operation by ID.
func (r *Repository) GetCloudRecoveryOperation(ctx context.Context, id string) (*domain.RecoveryOperation, error) {
	const q = `
		SELECT id, deployment_id, idempotency_key, action,
		       expected_bundle_sha256, repair_bundle_sha256, repair_bundle_path,
		       data_compatibility, status, receipt, error_message,
		       created_at, updated_at, started_at, completed_at
		FROM cloud_recovery_operations
		WHERE id = $1`
	return r.scanCloudRecoveryOperation(r.db.QueryRowContext(ctx, q, id))
}

// GetCloudRecoveryOperationByKey resolves an idempotent recovery request.
func (r *Repository) GetCloudRecoveryOperationByKey(ctx context.Context, deploymentID, key string) (*domain.RecoveryOperation, error) {
	const q = `
		SELECT id, deployment_id, idempotency_key, action,
		       expected_bundle_sha256, repair_bundle_sha256, repair_bundle_path,
		       data_compatibility, status, receipt, error_message,
		       created_at, updated_at, started_at, completed_at
		FROM cloud_recovery_operations
		WHERE deployment_id = $1 AND idempotency_key = $2`
	return r.scanCloudRecoveryOperation(r.db.QueryRowContext(ctx, q, deploymentID, key))
}

// CreateCloudRecoveryOperation creates an idempotent repair operation. The
// unique deployment/key constraint makes concurrent retries converge on one
// owner operation without replaying the remote effect.
func (r *Repository) CreateCloudRecoveryOperation(ctx context.Context, operation *domain.RecoveryOperation) (*domain.RecoveryOperation, error) {
	if operation == nil {
		return nil, fmt.Errorf("recovery operation is required")
	}
	if operation.ID == "" || operation.DeploymentID == "" || operation.IdempotencyKey == "" {
		return nil, fmt.Errorf("recovery operation identity is required")
	}
	if operation.CreatedAt.IsZero() {
		operation.CreatedAt = time.Now().UTC()
	}
	operation.UpdatedAt = operation.CreatedAt
	if operation.Status == "" {
		operation.Status = "pending"
	}
	const q = `
		INSERT INTO cloud_recovery_operations (
			id, deployment_id, idempotency_key, action,
			expected_bundle_sha256, repair_bundle_sha256, repair_bundle_path,
			data_compatibility, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
		ON CONFLICT (deployment_id, idempotency_key) DO NOTHING`
	if _, err := r.db.ExecContext(ctx, q,
		operation.ID, operation.DeploymentID, operation.IdempotencyKey,
		operation.Action, operation.ExpectedBundleSHA, operation.RepairBundleSHA,
		operation.RepairBundlePath, operation.DataCompatibility, operation.Status,
		operation.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("create recovery operation: %w", err)
	}
	return r.GetCloudRecoveryOperationByKey(ctx, operation.DeploymentID, operation.IdempotencyKey)
}

// ClaimCloudRecoveryOperation moves a pending operation to running exactly
// once. This is the owner-side fence that prevents duplicate repair pipelines.
func (r *Repository) ClaimCloudRecoveryOperation(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC()
	const q = `
		UPDATE cloud_recovery_operations
		SET status = 'running', started_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, q, id, now)
	if err != nil {
		return false, fmt.Errorf("claim recovery operation: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read recovery claim result: %w", err)
	}
	return count == 1, nil
}

// CompleteCloudRecoveryOperation persists the owner effect receipt only after
// the repaired deployment has been observed healthy.
func (r *Repository) CompleteCloudRecoveryOperation(ctx context.Context, id string, receipt domain.CloudRecoveryReceipt) error {
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode recovery receipt: %w", err)
	}
	now := time.Now().UTC()
	const q = `
		UPDATE cloud_recovery_operations
		SET status = 'succeeded', receipt = $2, error_message = NULL,
		    completed_at = $3, updated_at = $3
		WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, receiptJSON, now)
	if err != nil {
		return fmt.Errorf("complete recovery operation: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("read recovery completion result: %w", err)
	} else if count == 0 {
		return fmt.Errorf("recovery operation not found: %s", id)
	}
	return nil
}

// FailCloudRecoveryOperation records a terminal owner failure without
// fabricating an effect receipt.
func (r *Repository) FailCloudRecoveryOperation(ctx context.Context, id, message string) error {
	now := time.Now().UTC()
	const q = `
		UPDATE cloud_recovery_operations
		SET status = 'failed', error_message = $2, completed_at = $3, updated_at = $3
		WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, message, now)
	if err != nil {
		return fmt.Errorf("fail recovery operation: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("read recovery failure result: %w", err)
	} else if count == 0 {
		return fmt.Errorf("recovery operation not found: %s", id)
	}
	return nil
}

type recoveryOperationScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) scanCloudRecoveryOperation(row recoveryOperationScanner) (*domain.RecoveryOperation, error) {
	operation := &domain.RecoveryOperation{}
	var expectedSHA, receiptJSON, errorMessage sql.NullString
	var startedAt, completedAt sql.NullTime
	if err := row.Scan(
		&operation.ID, &operation.DeploymentID, &operation.IdempotencyKey,
		&operation.Action, &expectedSHA, &operation.RepairBundleSHA,
		&operation.RepairBundlePath, &operation.DataCompatibility, &operation.Status,
		&receiptJSON, &errorMessage, &operation.CreatedAt, &operation.UpdatedAt,
		&startedAt, &completedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan recovery operation: %w", err)
	}
	if expectedSHA.Valid {
		operation.ExpectedBundleSHA = expectedSHA.String
	}
	if errorMessage.Valid {
		operation.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		value := startedAt.Time
		operation.StartedAt = &value
	}
	if completedAt.Valid {
		value := completedAt.Time
		operation.CompletedAt = &value
	}
	if receiptJSON.Valid && receiptJSON.String != "" {
		var receipt domain.CloudRecoveryReceipt
		if err := json.Unmarshal([]byte(receiptJSON.String), &receipt); err != nil {
			return nil, fmt.Errorf("decode recovery receipt: %w", err)
		}
		operation.Receipt = &receipt
	}
	return operation, nil
}
