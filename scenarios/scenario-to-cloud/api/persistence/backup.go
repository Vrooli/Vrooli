package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// cloudRecoveryPointsPostgresDDL stores recovery points and restore receipts.
// Indexed columns carry identity, protection and retention; the full typed
// record lives in the JSON column so the wire shape and the stored shape are
// one document.
const cloudRecoveryPointsPostgresDDL = `
	CREATE TABLE IF NOT EXISTS cloud_recovery_points (
		id TEXT PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		operation_id TEXT,
		release_digest TEXT NOT NULL DEFAULT '',
		schema_version TEXT NOT NULL DEFAULT '',
		retention_policy TEXT NOT NULL DEFAULT '',
		protected BOOLEAN NOT NULL DEFAULT false,
		captured_at TIMESTAMPTZ NOT NULL,
		record JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_recovery_points_deployment_id ON cloud_recovery_points(deployment_id);
	CREATE INDEX IF NOT EXISTS idx_cloud_recovery_points_release_digest ON cloud_recovery_points(release_digest);
	CREATE TABLE IF NOT EXISTS cloud_restore_receipts (
		id TEXT PRIMARY KEY,
		recovery_point_id TEXT NOT NULL REFERENCES cloud_recovery_points(id) ON DELETE CASCADE,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		outcome TEXT NOT NULL,
		started_at TIMESTAMPTZ NOT NULL,
		completed_at TIMESTAMPTZ NOT NULL,
		record JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_restore_receipts_recovery_point_id ON cloud_restore_receipts(recovery_point_id);
`

const cloudRecoveryPointsSQLiteDDL = `
CREATE TABLE IF NOT EXISTS cloud_recovery_points (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 operation_id TEXT,
	 release_digest TEXT NOT NULL DEFAULT '',
	 schema_version TEXT NOT NULL DEFAULT '',
	 retention_policy TEXT NOT NULL DEFAULT '',
	 protected INTEGER NOT NULL DEFAULT 0,
	 captured_at TIMESTAMP NOT NULL,
	 record TEXT NOT NULL,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cloud_recovery_points_deployment_id ON cloud_recovery_points(deployment_id);
CREATE INDEX IF NOT EXISTS idx_cloud_recovery_points_release_digest ON cloud_recovery_points(release_digest);
CREATE TABLE IF NOT EXISTS cloud_restore_receipts (
	 id TEXT PRIMARY KEY,
	 recovery_point_id TEXT NOT NULL REFERENCES cloud_recovery_points(id) ON DELETE CASCADE,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 outcome TEXT NOT NULL,
	 started_at TIMESTAMP NOT NULL,
	 completed_at TIMESTAMP NOT NULL,
	 record TEXT NOT NULL,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_cloud_restore_receipts_recovery_point_id ON cloud_restore_receipts(recovery_point_id);
`

// CreateRecoveryPoint stores a new recovery point. The id is the caller's
// (operation-derived) identity; an existing id with an equal manifest digest
// returns the stored record, a different digest is a typed conflict.
func (r *Repository) CreateRecoveryPoint(ctx context.Context, rp *domain.RecoveryPoint) (*domain.RecoveryPoint, error) {
	if rp == nil || strings.TrimSpace(rp.ID) == "" || strings.TrimSpace(rp.DeploymentID) == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "recovery point id and deployment id are required")
	}
	if strings.TrimSpace(rp.RecoveryKeyRef) == "" || !rp.Encrypted {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "recovery points are always encrypted under a referenced key")
	}
	existing, err := r.GetRecoveryPoint(ctx, rp.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.ManifestDigest == rp.ManifestDigest {
			return existing, nil
		}
		return nil, apierrors.New(apierrors.CodeRecoveryPointConflict, "recovery point id already holds a different capture").
			WithDetail("recovery_point_id", rp.ID).WithDetail("stored_manifest_digest", existing.ManifestDigest).WithDetail("submitted_manifest_digest", rp.ManifestDigest)
	}
	now := time.Now().UTC()
	if rp.CreatedAt.IsZero() {
		rp.CreatedAt = now
	}
	rp.UpdatedAt = rp.CreatedAt
	rp.SchemaVersionRecord = domain.RecoveryPointSchemaVersion
	raw, err := json.Marshal(rp)
	if err != nil {
		return nil, apierrors.Internal("encode recovery point", err)
	}
	const q = `
		INSERT INTO cloud_recovery_points (id, deployment_id, operation_id, release_digest, schema_version, retention_policy, protected, captured_at, record, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)`
	if _, err := r.db.ExecContext(ctx, q, rp.ID, rp.DeploymentID, nullString(rp.OperationID), rp.ReleaseDigest, rp.SchemaVersion, rp.RetentionPolicy, rp.Protected, rp.CapturedAt.UTC(), string(raw), rp.CreatedAt); err != nil {
		return nil, apierrors.Internal("store recovery point", err)
	}
	return rp, nil
}

// GetRecoveryPoint reads one recovery point, nil when absent.
func (r *Repository) GetRecoveryPoint(ctx context.Context, id string) (*domain.RecoveryPoint, error) {
	row := r.db.QueryRowContext(ctx, `SELECT record FROM cloud_recovery_points WHERE id = $1`, id)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, apierrors.Internal("read recovery point", err)
	}
	return decodeRecoveryPoint(raw)
}

// ListRecoveryPoints returns a deployment's recovery points, newest first.
func (r *Repository) ListRecoveryPoints(ctx context.Context, deploymentID string) ([]domain.RecoveryPoint, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT record FROM cloud_recovery_points WHERE deployment_id = $1 ORDER BY captured_at DESC, id DESC`, deploymentID)
	if err != nil {
		return nil, apierrors.Internal("list recovery points", err)
	}
	defer rows.Close()
	out := []domain.RecoveryPoint{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, apierrors.Internal("scan recovery point", err)
		}
		rp, err := decodeRecoveryPoint(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, *rp)
	}
	if err := rows.Err(); err != nil {
		return nil, apierrors.Internal("iterate recovery points", err)
	}
	return out, nil
}

// SetRecoveryPointProtection records what protects a recovery point. It is
// the only mutation a stored point admits: captures are immutable.
func (r *Repository) SetRecoveryPointProtection(ctx context.Context, id string, protected bool, protectedBy []string) (*domain.RecoveryPoint, error) {
	rp, err := r.GetRecoveryPoint(ctx, id)
	if err != nil {
		return nil, err
	}
	if rp == nil {
		return nil, apierrors.New(apierrors.CodeRecoveryPointNotFound, "recovery point not found").WithDetail("recovery_point_id", id)
	}
	rp.Protected = protected
	rp.ProtectedBy = protectedBy
	if rp.ProtectedBy == nil {
		rp.ProtectedBy = []string{}
	}
	rp.UpdatedAt = time.Now().UTC()
	raw, err := json.Marshal(rp)
	if err != nil {
		return nil, apierrors.Internal("encode recovery point", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE cloud_recovery_points SET protected = $2, record = $3, updated_at = $4 WHERE id = $1`, id, protected, string(raw), rp.UpdatedAt); err != nil {
		return nil, apierrors.Internal("update recovery point protection", err)
	}
	return rp, nil
}

// DeleteRecoveryPoint removes a stored point. A protected point is refused
// with recovery_point_protected naming its holders; retention must clear the
// protection first, and only the references' owners can.
func (r *Repository) DeleteRecoveryPoint(ctx context.Context, id string) error {
	rp, err := r.GetRecoveryPoint(ctx, id)
	if err != nil {
		return err
	}
	if rp == nil {
		return apierrors.New(apierrors.CodeRecoveryPointNotFound, "recovery point not found").WithDetail("recovery_point_id", id)
	}
	if rp.Protected {
		return apierrors.New(apierrors.CodeRecoveryPointProtected, "recovery point is referenced and cannot be deleted").
			WithDetail("recovery_point_id", id).WithDetail("protected_by", rp.ProtectedBy)
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM cloud_recovery_points WHERE id = $1 AND protected = $2`, id, false); err != nil {
		return apierrors.Internal("delete recovery point", err)
	}
	return nil
}

// CreateRestoreReceipt stores one restore attempt.
func (r *Repository) CreateRestoreReceipt(ctx context.Context, receipt *domain.RestoreReceipt) (*domain.RestoreReceipt, error) {
	if receipt == nil || receipt.ID == "" || receipt.RecoveryPointID == "" || receipt.DeploymentID == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "restore receipt identity is required")
	}
	if receipt.InvariantResults == nil {
		receipt.InvariantResults = []domain.InvariantResult{}
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return nil, apierrors.Internal("encode restore receipt", err)
	}
	const q = `
		INSERT INTO cloud_restore_receipts (id, recovery_point_id, deployment_id, outcome, started_at, completed_at, record)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	if _, err := r.db.ExecContext(ctx, q, receipt.ID, receipt.RecoveryPointID, receipt.DeploymentID, receipt.Outcome, receipt.StartedAt.UTC(), receipt.CompletedAt.UTC(), string(raw)); err != nil {
		return nil, apierrors.Internal("store restore receipt", err)
	}
	return receipt, nil
}

// ListRestoreReceipts returns the restore attempts of one recovery point,
// newest first.
func (r *Repository) ListRestoreReceipts(ctx context.Context, recoveryPointID string) ([]domain.RestoreReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT record FROM cloud_restore_receipts WHERE recovery_point_id = $1 ORDER BY started_at DESC, id DESC`, recoveryPointID)
	if err != nil {
		return nil, apierrors.Internal("list restore receipts", err)
	}
	defer rows.Close()
	out := []domain.RestoreReceipt{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, apierrors.Internal("scan restore receipt", err)
		}
		var receipt domain.RestoreReceipt
		if err := json.Unmarshal([]byte(raw), &receipt); err != nil {
			return nil, apierrors.Internal("decode restore receipt", err)
		}
		out = append(out, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, apierrors.Internal("iterate restore receipts", err)
	}
	return out, nil
}

func decodeRecoveryPoint(raw string) (*domain.RecoveryPoint, error) {
	var rp domain.RecoveryPoint
	if err := json.Unmarshal([]byte(raw), &rp); err != nil {
		return nil, apierrors.Internal("decode recovery point", fmt.Errorf("%w", err))
	}
	if rp.BindingIDs == nil {
		rp.BindingIDs = []string{}
	}
	if rp.CredentialVersionRefs == nil {
		rp.CredentialVersionRefs = []string{}
	}
	if rp.Checksums == nil {
		rp.Checksums = map[string]domain.BindingChecksum{}
	}
	return &rp, nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
