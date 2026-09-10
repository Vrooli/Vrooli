package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"deployment-manager/shared"
)

// SQLRepository implements Repository with PostgreSQL.
type SQLRepository struct {
	db shared.RoutedDBTX
}

// NewSQLRepository creates a new SQL-backed release repository.
func NewSQLRepository(db shared.RoutedDBTX) *SQLRepository {
	return &SQLRepository{db: db}
}

// Insert creates a release row with per-platform rows in one transaction.
func (r *SQLRepository) Insert(ctx context.Context, release *Release) error {
	if release == nil {
		return fmt.Errorf("release is required")
	}
	if release.Channel == "" {
		release.Channel = "stable"
	}
	if release.Status == "" {
		release.Status = StatusPending
	}
	if release.AuthorizationEpoch == 0 {
		release.AuthorizationEpoch = 1
	}
	if (strings.TrimSpace(release.CandidateID) == "") != (strings.TrimSpace(release.DestinationRevisionID) == "") {
		return fmt.Errorf("release candidate and destination identities must be supplied together")
	}
	if strings.TrimSpace(release.CandidateID) != "" {
		candidate, err := r.GetCandidate(ctx, release.CandidateID)
		if err != nil {
			return fmt.Errorf("validate release candidate: %w", err)
		}
		if candidate == nil || candidate.Candidate.SourceRevision != release.GitCommitHash || candidate.ArtifactManifestDigest != release.ArtifactDigest {
			return fmt.Errorf("release candidate %q does not match the requested source or artifact identity", release.CandidateID)
		}
		destination, err := r.GetDestinationRevision(ctx, release.DestinationRevisionID)
		if err != nil {
			return fmt.Errorf("validate release destination: %w", err)
		}
		if destination == nil || destination.Revision.Channel != release.Channel {
			return fmt.Errorf("release destination %q does not match channel %q", release.DestinationRevisionID, release.Channel)
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort after the transaction has completed

	now := time.Now().UTC()
	release.CreatedAt = now
	release.UpdatedAt = now

	_, err = tx.ExecContext(ctx, `
		INSERT INTO releases
			(id, profile_id, deployment_id, profile_version, git_commit_hash,
			 artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key,
			 readiness_review_key, release_version, channel, status, release_notes, released_by,
			 promoted_from_release_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $18)
	`,
		release.ID, release.ProfileID, nullString(release.DeploymentID),
		nullIntPtr(release.ProfileVersion), release.GitCommitHash,
		nullString(release.ArtifactDigest), nullString(release.CandidateID), nullString(release.DestinationRevisionID), release.AuthorizationEpoch,
		nullString(release.IdempotencyKey), nullString(release.ReadinessReviewKey), release.ReleaseVersion, release.Channel, release.Status,
		nullString(release.ReleaseNotes), nullString(release.ReleasedBy), nullString(release.PromotedFromReleaseID), now,
	)
	if err != nil {
		return fmt.Errorf("insert release: %w", err)
	}

	for _, p := range release.Platforms {
		status := p.Status
		if status == "" {
			status = PlatformStatusPending
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO release_platforms (release_id, platform, status, approval_id)
			VALUES ($1, $2, $3, $4)
		`, release.ID, p.Platform, status, nullString(p.ApprovalID)); err != nil {
			return fmt.Errorf("insert release_platforms(%s): %w", p.Platform, err)
		}
	}

	return tx.Commit()
}

// Get fetches a release plus its platform rows.
func (r *SQLRepository) Get(ctx context.Context, releaseID string) (*Release, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key,
		       release_version, channel, status, release_notes, released_by,
		       promoted_from_release_id, readiness_goal_ref, approved_at_commit,
		       verification_evidence, created_at,
		       published_at, updated_at
		FROM releases
		WHERE id = $1
	`, releaseID)

	rel, err := scanRelease(row)
	if err != nil {
		return nil, err
	}

	platforms, err := r.listPlatforms(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}
	rel.Platforms = platforms
	return rel, nil
}

// GetByIdempotencyKey returns the durable lifecycle record for a repeated
// request, if one exists. A missing key is a normal cache miss.
func (r *SQLRepository) GetByIdempotencyKey(ctx context.Context, profileID, key string) (*Release, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key,
		       release_version, channel, status, release_notes, released_by,
		       promoted_from_release_id, readiness_goal_ref, approved_at_commit,
		       verification_evidence, created_at, published_at, updated_at
		FROM releases
		WHERE profile_id = $1 AND idempotency_key = $2
		LIMIT 1
	`, profileID, key)
	rel, err := scanRelease(row)
	if err != nil && strings.Contains(err.Error(), "release not found") {
		return nil, nil
	}
	return rel, err
}

func (r *SQLRepository) InsertOperation(ctx context.Context, operation *Operation) error {
	now := time.Now().UTC()
	operation.CreatedAt, operation.UpdatedAt = now, now
	if operation.Status == "" {
		operation.Status = OperationQueued
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO release_operations
			(id, release_id, profile_id, idempotency_key, request_snapshot, status, active_stage, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
	`, operation.ID, operation.ReleaseID, operation.ProfileID, nullString(operation.IdempotencyKey), nullString(string(operation.RequestSnapshot)), operation.Status, nullString(operation.ActiveStage), now)
	return err
}

func (r *SQLRepository) GetOperation(ctx context.Context, operationID string) (*Operation, error) {
	operation := &Operation{}
	var idempotencyKey, requestSnapshot, stage, errMsg sql.NullString
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id, release_id, profile_id, idempotency_key, request_snapshot, status, active_stage, error, created_at, updated_at, completed_at
		FROM release_operations WHERE id = $1
	`, operationID).Scan(&operation.ID, &operation.ReleaseID, &operation.ProfileID, &idempotencyKey, &requestSnapshot, &operation.Status, &stage, &errMsg, &operation.CreatedAt, &operation.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	operation.IdempotencyKey, operation.ActiveStage, operation.Error = idempotencyKey.String, stage.String, errMsg.String
	operation.RequestSnapshot = json.RawMessage(requestSnapshot.String)
	if completedAt.Valid {
		value := completedAt.Time
		operation.CompletedAt = &value
	}
	return operation, nil
}

func (r *SQLRepository) ListActiveOperations(ctx context.Context) ([]*Operation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, release_id, profile_id, idempotency_key, request_snapshot, status, active_stage, error, created_at, updated_at, completed_at
		FROM release_operations WHERE status IN ($1, $2, $3) ORDER BY created_at
	`, OperationQueued, OperationRunning, OperationAmbiguous)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var operations []*Operation
	for rows.Next() {
		operation := &Operation{}
		var idempotencyKey, requestSnapshot, stage, errMsg sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&operation.ID, &operation.ReleaseID, &operation.ProfileID, &idempotencyKey, &requestSnapshot, &operation.Status, &stage, &errMsg, &operation.CreatedAt, &operation.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		operation.IdempotencyKey, operation.ActiveStage, operation.Error = idempotencyKey.String, stage.String, errMsg.String
		operation.RequestSnapshot = json.RawMessage(requestSnapshot.String)
		if completedAt.Valid {
			value := completedAt.Time
			operation.CompletedAt = &value
		}
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}

func (r *SQLRepository) UpdateOperation(ctx context.Context, operationID, status, stage, errMsg string) error {
	now := time.Now().UTC()
	var completed interface{}
	if status == OperationComplete || status == OperationFailed || status == OperationCanceled || status == OperationAmbiguous {
		completed = now
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_operations
		SET status = $2, active_stage = $3, error = $4, updated_at = $5, completed_at = COALESCE($6, completed_at)
		WHERE id = $1
	`, operationID, status, nullString(stage), nullString(errMsg), now, completed)
	return err
}

func (r *SQLRepository) AcquireOperationLease(ctx context.Context, operationID, owner string, duration time.Duration) (*OperationLease, bool, error) {
	if duration <= 0 {
		duration = 2 * time.Minute
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback() //nolint:errcheck // best effort after commit
	expiresAt := time.Now().UTC().Add(duration)
	result, err := tx.ExecContext(ctx, `
		INSERT INTO release_operation_leases (operation_id, owner, fence, lease_until)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (operation_id) DO UPDATE SET
			owner = excluded.owner,
			fence = release_operation_leases.fence + 1,
			lease_until = excluded.lease_until
		WHERE release_operation_leases.lease_until <= CURRENT_TIMESTAMP
	`, operationID, owner, expiresAt)
	if err != nil {
		return nil, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, err
	}
	if rows == 0 {
		return nil, false, tx.Rollback()
	}
	var fence uint64
	var storedExpiry time.Time
	if err := tx.QueryRowContext(ctx, `
		SELECT fence, lease_until FROM release_operation_leases WHERE operation_id = $1
	`, operationID).Scan(&fence, &storedExpiry); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return &OperationLease{OperationID: operationID, Owner: owner, Fence: fence, ExpiresAt: storedExpiry}, true, nil
}

func (r *SQLRepository) ReleaseOperationLease(ctx context.Context, lease *OperationLease) error {
	if lease == nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_operation_leases
		SET lease_until = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE operation_id = $1 AND owner = $2 AND fence = $3
	`, lease.OperationID, lease.Owner, lease.Fence)
	return err
}

func (r *SQLRepository) UpdateOperationFenced(ctx context.Context, lease *OperationLease, status, stage, errMsg string) error {
	if lease == nil {
		return fmt.Errorf("operation lease is required")
	}
	now := time.Now().UTC()
	var completed interface{}
	if status == OperationComplete || status == OperationFailed || status == OperationCanceled || status == OperationAmbiguous {
		completed = now
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE release_operations
		SET status = $2, active_stage = $3, error = $4, updated_at = $5, completed_at = COALESCE($6, completed_at)
		WHERE id = $1 AND EXISTS (
			SELECT 1 FROM release_operation_leases
			WHERE operation_id = $1 AND owner = $7 AND fence = $8 AND lease_until > CURRENT_TIMESTAMP
		)
	`, lease.OperationID, status, nullString(stage), nullString(errMsg), now, completed, lease.Owner, lease.Fence)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("operation %q lease is stale", lease.OperationID)
	}
	return nil
}

func (r *SQLRepository) AppendOperationEvent(ctx context.Context, lease *OperationLease, status, stage, message string) error {
	if lease == nil {
		return fmt.Errorf("operation lease is required")
	}
	eventID := fmt.Sprintf("%s-%d", lease.OperationID, time.Now().UnixNano())
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO release_operation_events
			(event_id, operation_id, fence, status, active_stage, message, created_at)
		SELECT $1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP
		WHERE EXISTS (
			SELECT 1 FROM release_operation_leases
			WHERE operation_id = $2 AND owner = $7 AND fence = $3 AND lease_until > CURRENT_TIMESTAMP
		)
	`, eventID, lease.OperationID, lease.Fence, status, nullString(stage), nullString(message), lease.Owner)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("operation %q lease is stale", lease.OperationID)
	}
	return nil
}

// ListByProfile returns recent releases for a profile (newest first).
func (r *SQLRepository) ListByProfile(ctx context.Context, profileID string, limit int) ([]*Release, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key,
		       release_version, channel, status, release_notes, released_by,
		       promoted_from_release_id, readiness_goal_ref, approved_at_commit,
		       verification_evidence, created_at,
		       published_at, updated_at
		FROM releases
		WHERE profile_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, profileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Release
	for rows.Next() {
		rel, err := scanReleaseRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rel)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach platform rows per release (small N; OK to loop).
	for _, rel := range out {
		platforms, err := r.listPlatforms(ctx, rel.ID)
		if err != nil {
			return nil, err
		}
		rel.Platforms = platforms
	}
	return out, nil
}

func (r *SQLRepository) RecordRecoveryReceipt(ctx context.Context, receipt RecoveryReceipt) error {
	if !receipt.Valid() {
		return fmt.Errorf("recovery receipt is incomplete")
	}
	if err := r.validateReleaseReceiptIdentity(ctx, receipt.ReleaseID, receipt.CandidateID, receipt.DestinationRevisionID, ""); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO release_recovery_receipts
			(receipt_id, release_id, candidate_id, destination_revision_id, deployment_id, action, outcome, health, bundle_sha256, external_receipt, observed_at, dry_run)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (release_id, external_receipt) DO NOTHING
	`, receipt.ReceiptID, receipt.ReleaseID, receipt.CandidateID, receipt.DestinationRevisionID, receipt.DeploymentID, receipt.Action, receipt.Outcome, receipt.Health, nullString(receipt.BundleSHA256), receipt.ExternalReceipt, receipt.ObservedAt, receipt.DryRun)
	if err != nil {
		return err
	}
	var stored RecoveryReceipt
	var bundleSHA sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT receipt_id, release_id, candidate_id, destination_revision_id, deployment_id,
		       action, outcome, health, bundle_sha256, external_receipt, observed_at, dry_run
		FROM release_recovery_receipts
		WHERE release_id = $1 AND external_receipt = $2
	`, receipt.ReleaseID, receipt.ExternalReceipt).Scan(
		&stored.ReceiptID, &stored.ReleaseID, &stored.CandidateID, &stored.DestinationRevisionID,
		&stored.DeploymentID, &stored.Action, &stored.Outcome, &stored.Health, &bundleSHA,
		&stored.ExternalReceipt, &stored.ObservedAt, &stored.DryRun)
	if err != nil {
		return fmt.Errorf("load stored recovery receipt: %w", err)
	}
	stored.BundleSHA256 = bundleSHA.String
	if sameRecoveryReceipt(stored, receipt) {
		return nil
	}
	return fmt.Errorf("conflicting recovery receipt for release %q external receipt %q", receipt.ReleaseID, receipt.ExternalReceipt)
}

func sameRecoveryReceipt(left, right RecoveryReceipt) bool {
	return left.ReceiptID == right.ReceiptID && left.ReleaseID == right.ReleaseID &&
		left.CandidateID == right.CandidateID && left.DestinationRevisionID == right.DestinationRevisionID &&
		left.DeploymentID == right.DeploymentID && left.Action == right.Action &&
		left.Outcome == right.Outcome && left.Health == right.Health &&
		left.BundleSHA256 == right.BundleSHA256 && left.ExternalReceipt == right.ExternalReceipt &&
		left.ObservedAt.Equal(right.ObservedAt) && left.DryRun == right.DryRun
}

func (r *SQLRepository) ListRecoveryReceipts(ctx context.Context, releaseID string) ([]RecoveryReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT receipt_id, release_id, candidate_id, destination_revision_id, deployment_id, action, outcome, health, bundle_sha256, external_receipt, observed_at, dry_run
		FROM release_recovery_receipts WHERE release_id = $1 ORDER BY observed_at
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RecoveryReceipt
	for rows.Next() {
		var receipt RecoveryReceipt
		var bundleSHA sql.NullString
		if err := rows.Scan(&receipt.ReceiptID, &receipt.ReleaseID, &receipt.CandidateID, &receipt.DestinationRevisionID, &receipt.DeploymentID, &receipt.Action, &receipt.Outcome, &receipt.Health, &bundleSHA, &receipt.ExternalReceipt, &receipt.ObservedAt, &receipt.DryRun); err != nil {
			return nil, err
		}
		receipt.BundleSHA256 = bundleSHA.String
		out = append(out, receipt)
	}
	return out, rows.Err()
}

// UpdateStatus transitions the release status, stamping published_at on terminal success.
func (r *SQLRepository) UpdateStatus(ctx context.Context, releaseID, status string) error {
	now := time.Now().UTC()
	if lease, ok := operationLeaseFromContext(ctx); ok {
		if status == StatusPublished {
			_, err := r.db.ExecContext(ctx, `
				UPDATE releases
				SET status = $2, updated_at = $3, published_at = COALESCE(published_at, $3)
				WHERE id = $1 AND EXISTS (
					SELECT 1 FROM release_operations o
					JOIN release_operation_leases l ON l.operation_id = o.id
					WHERE o.release_id = releases.id AND l.operation_id = $4 AND l.owner = $5 AND l.fence = $6 AND l.lease_until > CURRENT_TIMESTAMP
				)
			`, releaseID, status, now, lease.OperationID, lease.Owner, lease.Fence)
			return err
		}
		_, err := r.db.ExecContext(ctx, `
			UPDATE releases SET status = $2, updated_at = $3
			WHERE id = $1 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = releases.id AND l.operation_id = $4 AND l.owner = $5 AND l.fence = $6 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, releaseID, status, now, lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	if status == StatusPublished {
		_, err := r.db.ExecContext(ctx, `
			UPDATE releases
			SET status = $2, updated_at = $3, published_at = COALESCE(published_at, $3)
			WHERE id = $1
		`, releaseID, status, now)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE releases SET status = $2, updated_at = $3 WHERE id = $1
	`, releaseID, status, now)
	return err
}

// SetDeploymentID records the exact owner deployment identity only after the
// owner has returned a validated receipt. Operation workers use the same
// fencing predicate as other release writes.
func (r *SQLRepository) SetDeploymentID(ctx context.Context, releaseID, deploymentID string) error {
	if strings.TrimSpace(releaseID) == "" || strings.TrimSpace(deploymentID) == "" {
		return fmt.Errorf("release and deployment identities are required")
	}
	if lease, ok := operationLeaseFromContext(ctx); ok {
		_, err := r.db.ExecContext(ctx, `
			UPDATE releases
			SET deployment_id = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = releases.id AND l.operation_id = $3 AND l.owner = $4 AND l.fence = $5 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, releaseID, deploymentID, lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE releases SET deployment_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, releaseID, deploymentID)
	return err
}

// SetVerificationEvidence persists the per-platform verify outcomes as JSON.
func (r *SQLRepository) SetVerificationEvidence(ctx context.Context, releaseID string, items []VerificationItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	if lease, ok := operationLeaseFromContext(ctx); ok {
		_, err = r.db.ExecContext(ctx, `
			UPDATE releases SET verification_evidence = $2, updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = releases.id AND l.operation_id = $3 AND l.owner = $4 AND l.fence = $5 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, releaseID, data, lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE releases SET verification_evidence = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1
	`, releaseID, data)
	return err
}

// SetReadinessApproval records the goal and exact commit that cleared
// readiness. It is intentionally separate from platform approval state.
func (r *SQLRepository) SetReadinessApproval(ctx context.Context, releaseID, goalRef, approvedCommit string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE releases
		SET readiness_goal_ref = $2, approved_at_commit = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, releaseID, nullString(goalRef), nullString(approvedCommit))
	return err
}

// GetReadinessByProfileCommit returns the latest release-side readiness
// projection for the exact commit being evaluated.
func (r *SQLRepository) GetReadinessByProfileCommit(ctx context.Context, profileID, commit string) (*ReadinessRecord, error) {
	var goalRef, approvedCommit sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT readiness_goal_ref, approved_at_commit
		FROM releases
		WHERE profile_id = $1 AND git_commit_hash = $2
		ORDER BY created_at DESC LIMIT 1
	`, profileID, commit).Scan(&goalRef, &approvedCommit)
	if err == sql.ErrNoRows {
		return &ReadinessRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	record := &ReadinessRecord{
		VerdictPresent:   approvedCommit.Valid,
		ReadinessGoalRef: goalRef.String,
		ApprovedAtCommit: approvedCommit.String,
		// The approval commit is written by RecordApproval only after
		// swarm-manager reports the readiness goal closed. Release publication
		// is a separate lifecycle transition and must not stand in for goal
		// closure.
		GoalClosed: approvedCommit.Valid,
	}
	var waiver ReadinessWaiver
	err = r.db.QueryRowContext(ctx, `
		SELECT reason, actor, git_commit_hash, created_at FROM readiness_waivers
		WHERE profile_id = $1 AND git_commit_hash = $2
	`, profileID, commit).Scan(&waiver.Reason, &waiver.Actor, &waiver.Commit, &waiver.At)
	if err == nil {
		record.Waiver = &waiver
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	return record, nil
}

// GetLatestReadiness returns the newest release-side readiness projection for
// a profile, for read-only surfaces such as the Offer Desk ladder.
func (r *SQLRepository) GetLatestReadiness(ctx context.Context, profileID string) (*ReadinessRecord, error) {
	var goalRef, approvedCommit sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT readiness_goal_ref, approved_at_commit FROM releases
		WHERE profile_id = $1 ORDER BY created_at DESC LIMIT 1
	`, profileID).Scan(&goalRef, &approvedCommit)
	if err == sql.ErrNoRows {
		return &ReadinessRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &ReadinessRecord{VerdictPresent: approvedCommit.Valid, ReadinessGoalRef: goalRef.String, ApprovedAtCommit: approvedCommit.String, GoalClosed: approvedCommit.Valid}, nil
}

// RecordReadinessWaiver records a reasoned, actor-bound exception for exactly
// one commit. The primary key makes repeated recording idempotent.
func (r *SQLRepository) RecordReadinessWaiver(ctx context.Context, profileID, commit, reason, actor string) error {
	if strings.TrimSpace(profileID) == "" || strings.TrimSpace(commit) == "" || strings.TrimSpace(reason) == "" || strings.TrimSpace(actor) == "" {
		return fmt.Errorf("profile, commit, reason, and actor are required")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO readiness_waivers (profile_id, git_commit_hash, reason, actor)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (profile_id, git_commit_hash) DO UPDATE SET reason = EXCLUDED.reason, actor = EXCLUDED.actor
	`, profileID, commit, reason, actor)
	return err
}

// MarkPlatformPublished stamps the artifact id and flips the platform to published.
func (r *SQLRepository) MarkPlatformPublished(ctx context.Context, releaseID, platform string, artifactID int64) error {
	if lease, ok := operationLeaseFromContext(ctx); ok {
		_, err := r.db.ExecContext(ctx, `
			UPDATE release_platforms
			SET status = $3, lpbs_artifact_id = $4, published_at = CURRENT_TIMESTAMP, error = NULL
			WHERE release_id = $1 AND platform = $2 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = release_platforms.release_id AND l.operation_id = $5 AND l.owner = $6 AND l.fence = $7 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, releaseID, platform, PlatformStatusPublished, artifactID, lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_platforms
		SET status = $3, lpbs_artifact_id = $4, published_at = CURRENT_TIMESTAMP, error = NULL
		WHERE release_id = $1 AND platform = $2
	`, releaseID, platform, PlatformStatusPublished, artifactID)
	return err
}

// MarkPlatformStatus updates status and optional error for a platform row.
func (r *SQLRepository) MarkPlatformStatus(ctx context.Context, releaseID, platform, status, errMsg string) error {
	if lease, ok := operationLeaseFromContext(ctx); ok {
		verifiedAt := ""
		if status == PlatformStatusPublished {
			verifiedAt = ", verified_at = CURRENT_TIMESTAMP"
		}
		_, err := r.db.ExecContext(ctx, `
			UPDATE release_platforms
			SET status = $3, error = $4`+verifiedAt+`
			WHERE release_id = $1 AND platform = $2 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = release_platforms.release_id AND l.operation_id = $5 AND l.owner = $6 AND l.fence = $7 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, releaseID, platform, status, nullString(errMsg), lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	if status == PlatformStatusPublished {
		_, err := r.db.ExecContext(ctx, `
			UPDATE release_platforms
			SET status = $3, verified_at = CURRENT_TIMESTAMP, error = $4
			WHERE release_id = $1 AND platform = $2
		`, releaseID, platform, status, nullString(errMsg))
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_platforms
		SET status = $3, error = $4
		WHERE release_id = $1 AND platform = $2
	`, releaseID, platform, status, nullString(errMsg))
	return err
}

// MarkSuperseded marks prior published releases for the same profile+channel
// as superseded so that only one release is current per (profile, channel).
func (r *SQLRepository) MarkSuperseded(ctx context.Context, profileID, channel, exceptReleaseID string) error {
	if lease, ok := operationLeaseFromContext(ctx); ok {
		_, err := r.db.ExecContext(ctx, `
			UPDATE releases
			SET status = $4, updated_at = CURRENT_TIMESTAMP
			WHERE profile_id = $1 AND channel = $2 AND id <> $3 AND status = $5 AND EXISTS (
				SELECT 1 FROM release_operations o
				JOIN release_operation_leases l ON l.operation_id = o.id
				WHERE o.release_id = $3 AND l.operation_id = $6 AND l.owner = $7 AND l.fence = $8 AND l.lease_until > CURRENT_TIMESTAMP
			)
		`, profileID, channel, exceptReleaseID, StatusSuperseded, StatusPublished, lease.OperationID, lease.Owner, lease.Fence)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE releases
		SET status = $4, updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = $1 AND channel = $2 AND id <> $3 AND status = $5
	`, profileID, channel, exceptReleaseID, StatusSuperseded, StatusPublished)
	return err
}

// AcquireProfileLock takes a durable profile-scoped lock. The lock row is
// portable across SQLite and PostgreSQL and remains held until release is
// called, so request cancellation cannot silently abandon ownership.
func (r *SQLRepository) AcquireProfileLock(ctx context.Context, profileID string) (bool, func(), error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO release_profile_locks (profile_id, acquired_at)
		VALUES ($1, CURRENT_TIMESTAMP)
		ON CONFLICT (profile_id) DO NOTHING
	`, profileID)
	if err != nil {
		_ = tx.Rollback()
		return false, nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, nil, err
	}
	if rows == 0 {
		_ = tx.Rollback()
		return false, func() {}, nil
	}
	if err := tx.Commit(); err != nil {
		return false, nil, err
	}
	release := func() {
		_, _ = r.db.ExecContext(context.WithoutCancel(ctx),
			`DELETE FROM release_profile_locks WHERE profile_id = $1`, profileID)
	}
	return true, release, nil
}

func (r *SQLRepository) listPlatforms(ctx context.Context, releaseID string) ([]ReleasePlatform, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT release_id, platform, status, approval_id, lpbs_artifact_id,
		       published_at, verified_at, error
		FROM release_platforms
		WHERE release_id = $1
		ORDER BY platform
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReleasePlatform
	for rows.Next() {
		var p ReleasePlatform
		var approvalID, errMsg sql.NullString
		var artifactID sql.NullInt64
		var publishedAt, verifiedAt sql.NullTime
		if err := rows.Scan(&p.ReleaseID, &p.Platform, &p.Status,
			&approvalID, &artifactID, &publishedAt, &verifiedAt, &errMsg,
		); err != nil {
			return nil, err
		}
		p.ApprovalID = approvalID.String
		p.Error = errMsg.String
		if artifactID.Valid {
			p.LPBSArtifactID = artifactID.Int64
		}
		if publishedAt.Valid {
			t := publishedAt.Time
			p.PublishedAt = &t
		}
		if verifiedAt.Valid {
			t := verifiedAt.Time
			p.VerifiedAt = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanRelease(row *sql.Row) (*Release, error) {
	rel, err := scanReleaseFields(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("release not found")
	}
	return rel, err
}

func scanReleaseRow(rows *sql.Rows) (*Release, error) {
	return scanReleaseFields(rows)
}

func scanReleaseFields(row rowScanner) (*Release, error) {
	rel := &Release{}
	var deploymentID, artifactDigest, candidateID, destinationRevisionID, idempotencyKey, readinessReviewKey, releaseNotes, releasedBy, promotedFrom, readinessGoalRef, approvedAtCommit sql.NullString
	var profileVersion sql.NullInt32
	var authorizationEpoch sql.NullInt64
	var publishedAt sql.NullTime
	var evidence []byte

	err := row.Scan(
		&rel.ID, &rel.ProfileID, &deploymentID, &profileVersion,
		&rel.GitCommitHash, &artifactDigest, &candidateID, &destinationRevisionID, &authorizationEpoch, &idempotencyKey, &readinessReviewKey, &rel.ReleaseVersion, &rel.Channel, &rel.Status,
		&releaseNotes, &releasedBy, &promotedFrom, &readinessGoalRef, &approvedAtCommit, &evidence,
		&rel.CreatedAt, &publishedAt, &rel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rel.DeploymentID = deploymentID.String
	rel.ArtifactDigest = artifactDigest.String
	rel.CandidateID = candidateID.String
	rel.DestinationRevisionID = destinationRevisionID.String
	if authorizationEpoch.Valid && authorizationEpoch.Int64 > 0 {
		rel.AuthorizationEpoch = uint64(authorizationEpoch.Int64)
	}
	rel.IdempotencyKey = idempotencyKey.String
	rel.ReadinessReviewKey = readinessReviewKey.String
	rel.ReleaseNotes = releaseNotes.String
	rel.ReleasedBy = releasedBy.String
	rel.PromotedFromReleaseID = promotedFrom.String
	rel.ReadinessGoalRef = readinessGoalRef.String
	rel.ApprovedAtCommit = approvedAtCommit.String
	if profileVersion.Valid {
		rel.ProfileVersion = int(profileVersion.Int32)
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		rel.PublishedAt = &t
	}
	if len(evidence) > 0 {
		if err := json.Unmarshal(evidence, &rel.VerificationEvidence); err != nil {
			return nil, fmt.Errorf("decode release verification evidence: %w", err)
		}
	}
	return rel, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullIntPtr(n int) sql.NullInt32 {
	if n == 0 {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: int32(n), Valid: true}
}
