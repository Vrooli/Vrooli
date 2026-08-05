package deployments

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	internalEvidence "deployment-manager/internal/evidence"
	"deployment-manager/shared"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// ApprovalsRepository persists deployment approval records.
type ApprovalsRepository interface {
	Create(ctx context.Context, approval *DeploymentApproval) error
	Get(ctx context.Context, id string) (*DeploymentApproval, error)
	ListByCommit(ctx context.Context, profileID, commitHash string) ([]*DeploymentApproval, error)
	ListByProfile(ctx context.Context, profileID string, limit int) ([]*DeploymentApproval, error)
	UpdateDecision(ctx context.Context, id, decision, reviewer, notes string) error
	MarkStale(ctx context.Context, profileID, platform, exceptCommit string) error
	GetRequiredPlatforms(ctx context.Context, profileID string) ([]string, error)
	SetRequiredPlatforms(ctx context.Context, profileID string, platforms []string) error
	CheckReleaseGate(ctx context.Context, profileID, commitHash string) (*ReleaseGateStatus, error)
	GetRequiredTargets(ctx context.Context, profileID string) ([]RequiredTarget, error)
	SetRequiredTargets(ctx context.Context, profileID string, targets []RequiredTarget) error
}

// SQLApprovalsRepository implements ApprovalsRepository with PostgreSQL.
type SQLApprovalsRepository struct {
	conn         shared.DBTX       // used for regular queries (may be *sql.DB or *sql.Tx)
	db           shared.RoutedDBTX // retained for schema compatibility and transactional configuration
	evidenceRepo internalEvidence.Repository
}

// NewSQLApprovalsRepository creates a new SQL-backed approvals repository.
func NewSQLApprovalsRepository(db shared.RoutedDBTX) *SQLApprovalsRepository {
	return &SQLApprovalsRepository{conn: db, db: db}
}

func (r *SQLApprovalsRepository) WithEvidenceRepository(repo internalEvidence.Repository) *SQLApprovalsRepository {
	r.evidenceRepo = repo
	return r
}

// WithTx returns a new repository instance backed by the given transaction.
// Operations that start their own transactions (SetRequiredTargets)
// are not available on the returned instance.
func (r *SQLApprovalsRepository) WithTx(tx *sql.Tx) *SQLApprovalsRepository {
	return &SQLApprovalsRepository{conn: tx, db: nil}
}

func (r *SQLApprovalsRepository) Create(ctx context.Context, approval *DeploymentApproval) error {
	// Mark previous approvals for same profile+platform as stale
	if err := r.MarkStale(ctx, approval.ProfileID, approval.Platform, approval.GitCommitHash); err != nil {
		return fmt.Errorf("mark stale: %w", err)
	}

	_, err := r.conn.ExecContext(ctx,
		`INSERT INTO deployment_approvals
			(id, profile_id, git_commit_hash, platform, status, validation_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		approval.ID, approval.ProfileID, approval.GitCommitHash,
		approval.Platform, approval.Status, nullString(approval.ValidationID),
		approval.CreatedAt,
	)
	return err
}

func (r *SQLApprovalsRepository) Get(ctx context.Context, id string) (*DeploymentApproval, error) {
	row := r.conn.QueryRowContext(ctx,
		`SELECT id, profile_id, git_commit_hash, platform, status,
		        approved_by, approved_at, notes, validation_id, created_at, updated_at
		 FROM deployment_approvals WHERE id = $1`, id)
	return scanApproval(row)
}

func (r *SQLApprovalsRepository) ListByCommit(ctx context.Context, profileID, commitHash string) ([]*DeploymentApproval, error) {
	rows, err := r.conn.QueryContext(ctx,
		`SELECT id, profile_id, git_commit_hash, platform, status,
		        approved_by, approved_at, notes, validation_id, created_at, updated_at
		 FROM deployment_approvals
		 WHERE profile_id = $1 AND git_commit_hash = $2
		 ORDER BY platform`, profileID, commitHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApprovals(rows)
}

func (r *SQLApprovalsRepository) ListByProfile(ctx context.Context, profileID string, limit int) ([]*DeploymentApproval, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.conn.QueryContext(ctx,
		`SELECT id, profile_id, git_commit_hash, platform, status,
		        approved_by, approved_at, notes, validation_id, created_at, updated_at
		 FROM deployment_approvals
		 WHERE profile_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`, profileID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApprovals(rows)
}

func (r *SQLApprovalsRepository) UpdateDecision(ctx context.Context, id, decision, reviewer, notes string) error {
	now := time.Now()
	_, err := r.conn.ExecContext(ctx,
		`UPDATE deployment_approvals
		 SET status = $2, approved_by = $3, approved_at = $4, notes = $5, updated_at = $4
		 WHERE id = $1`,
		id, decision, reviewer, now, notes)
	return err
}

func (r *SQLApprovalsRepository) MarkStale(ctx context.Context, profileID, platform, exceptCommit string) error {
	_, err := r.conn.ExecContext(ctx,
		`UPDATE deployment_approvals
		 SET status = 'stale', updated_at = CURRENT_TIMESTAMP
		 WHERE profile_id = $1 AND platform = $2
		   AND git_commit_hash != $3
		   AND status NOT IN ('stale', 'rejected')`,
		profileID, platform, exceptCommit)
	return err
}

func (r *SQLApprovalsRepository) GetRequiredPlatforms(ctx context.Context, profileID string) ([]string, error) {
	rows, err := r.conn.QueryContext(ctx,
		`SELECT platform FROM profile_required_platforms WHERE profile_id = $1 ORDER BY platform`,
		profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		platforms = append(platforms, p)
	}
	return platforms, rows.Err()
}

func (r *SQLApprovalsRepository) SetRequiredPlatforms(ctx context.Context, profileID string, platforms []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort after the transaction has completed

	if _, err := tx.ExecContext(ctx, `DELETE FROM profile_required_platforms WHERE profile_id = $1`, profileID); err != nil {
		return err
	}

	for _, p := range platforms {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO profile_required_platforms (profile_id, platform) VALUES ($1, $2)`,
			profileID, p); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *SQLApprovalsRepository) CheckReleaseGate(ctx context.Context, profileID, commitHash string) (*ReleaseGateStatus, error) {
	// The legacy repository-only seam remains usable by isolated callers and
	// older clients. Production wiring always supplies the evidence repository,
	// which activates the contract-backed gate below.
	if r.evidenceRepo == nil {
		return r.checkLegacyReleaseGate(ctx, profileID, commitHash)
	}
	required, err := r.GetRequiredTargets(ctx, profileID)
	if err != nil {
		return nil, err
	}
	// Preserve the old platform configuration as a compatibility adapter while
	// profiles migrate. It is projected into explicit host targets.
	if len(required) == 0 {
		platforms, legacyErr := r.GetRequiredPlatforms(ctx, profileID)
		if legacyErr != nil {
			return nil, legacyErr
		}
		for _, platform := range platforms {
			required = append(required, RequiredTarget{Ramp: "release", Platform: platform, OS: platform, DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST})
		}
	}

	approvals, err := r.ListByCommit(ctx, profileID, commitHash)
	if err != nil {
		return nil, err
	}

	// Build lookup of approval status by platform
	approvalByPlatform := make(map[string]*DeploymentApproval, len(approvals))
	for _, a := range approvals {
		approvalByPlatform[a.Platform] = a
	}

	gate := &ReleaseGateStatus{
		ProfileID:     profileID,
		GitCommitHash: commitHash,
		Ready:         len(required) > 0,
		Reason:        "",
		Platforms:     make([]PlatformGateStatus, 0, len(required)),
		Targets:       make([]TargetGateStatus, 0, len(required)),
	}
	if len(required) == 0 {
		gate.Reason = "no_required_targets_configured"
	}

	for _, target := range required {
		platform := target.Platform
		ps := PlatformGateStatus{
			Platform: platform,
			Required: true,
		}
		approvalStatus := "missing"
		if a, ok := approvalByPlatform[platform]; ok {
			ps.Status = a.Status
			approvalStatus = a.Status
		} else {
			ps.Status = "missing"
		}
		targetStatus := TargetGateStatus{Target: target, ApprovalStatus: approvalStatus, EvidenceDisposition: "missing"}
		if r.evidenceRepo != nil {
			verdicts, evidenceErr := r.evidenceRepo.List(ctx, profileID, commitHash, 1000)
			if evidenceErr != nil {
				return nil, evidenceErr
			}
			for _, verdict := range verdicts {
				if verdict.Target == nil || !sameTarget(target, verdict.Target) {
					continue
				}
				targetStatus.EvidenceDisposition = verdict.Disposition.String()
				targetStatus.EvidenceRunID = verdict.RunId
				break
			}
		} else {
			// Without an evidence repository, the gate must remain closed.
			targetStatus.EvidenceDisposition = "missing"
		}
		gate.Targets = append(gate.Targets, targetStatus)
		if targetStatus.EvidenceDisposition == commonv1.Disposition_DISPOSITION_FAILED.String() {
			gate.Ready = false
			gate.Reason = "target_evidence_failed"
		} else if targetStatus.EvidenceDisposition != commonv1.Disposition_DISPOSITION_PASSED.String() {
			gate.Ready = false
			gate.Reason = "target_evidence_missing"
		} else if ps.Status != ApprovalStatusApproved {
			gate.Ready = false
			gate.Reason = "approval_missing"
		}
		gate.Platforms = append(gate.Platforms, ps)
	}

	return gate, nil
}

func (r *SQLApprovalsRepository) checkLegacyReleaseGate(ctx context.Context, profileID, commitHash string) (*ReleaseGateStatus, error) {
	required, err := r.GetRequiredPlatforms(ctx, profileID)
	if err != nil {
		return nil, err
	}
	approvals, err := r.ListByCommit(ctx, profileID, commitHash)
	if err != nil {
		return nil, err
	}
	byPlatform := make(map[string]string, len(approvals))
	for _, approval := range approvals {
		byPlatform[approval.Platform] = approval.Status
	}
	gate := &ReleaseGateStatus{ProfileID: profileID, GitCommitHash: commitHash, Ready: len(required) > 0, Reason: "", Platforms: make([]PlatformGateStatus, 0, len(required)), Targets: make([]TargetGateStatus, 0, len(required))}
	if len(required) == 0 {
		gate.Reason = "no_required_platforms_configured"
	}
	for _, platform := range required {
		status := byPlatform[platform]
		if status == "" {
			status = "missing"
		}
		gate.Platforms = append(gate.Platforms, PlatformGateStatus{Platform: platform, Required: true, Status: status})
		if status != ApprovalStatusApproved {
			gate.Ready = false
			gate.Reason = "platforms_not_approved"
		}
	}
	return gate, nil
}

func sameTarget(required RequiredTarget, actual *commonv1.EvidenceTarget) bool {
	return required.Ramp == actual.Ramp && required.Platform == actual.Platform && required.OS == actual.Os && required.DeviceKind == actual.DeviceKind
}

func (r *SQLApprovalsRepository) GetRequiredTargets(ctx context.Context, profileID string) ([]RequiredTarget, error) {
	rows, err := r.conn.QueryContext(ctx, `SELECT ramp, platform, os, device_kind FROM profile_required_targets WHERE profile_id = $1 ORDER BY ramp, platform, os, device_kind`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var targets []RequiredTarget
	for rows.Next() {
		var target RequiredTarget
		var deviceKind int32
		if err := rows.Scan(&target.Ramp, &target.Platform, &target.OS, &deviceKind); err != nil {
			return nil, err
		}
		target.DeviceKind = commonv1.DeviceKind(deviceKind)
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

func (r *SQLApprovalsRepository) SetRequiredTargets(ctx context.Context, profileID string, targets []RequiredTarget) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort after the transaction has completed
	if _, err := tx.ExecContext(ctx, `DELETE FROM profile_required_targets WHERE profile_id = $1`, profileID); err != nil {
		return err
	}
	for _, target := range targets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO profile_required_targets (profile_id, ramp, platform, os, device_kind) VALUES ($1, $2, $3, $4, $5)`, profileID, target.Ramp, target.Platform, target.OS, int32(target.DeviceKind)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// nullString converts an empty string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// scanApproval scans a single row into a DeploymentApproval.
func scanApproval(row *sql.Row) (*DeploymentApproval, error) {
	a := &DeploymentApproval{}
	var approvedBy, notes, validationID sql.NullString
	var approvedAt sql.NullTime

	err := row.Scan(
		&a.ID, &a.ProfileID, &a.GitCommitHash, &a.Platform, &a.Status,
		&approvedBy, &approvedAt, &notes, &validationID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("approval not found")
	}
	if err != nil {
		return nil, err
	}

	a.ApprovedBy = approvedBy.String
	a.Notes = notes.String
	a.ValidationID = validationID.String
	if approvedAt.Valid {
		a.ApprovedAt = &approvedAt.Time
	}
	return a, nil
}

// scanApprovals scans multiple rows into DeploymentApproval slices.
func scanApprovals(rows *sql.Rows) ([]*DeploymentApproval, error) {
	var result []*DeploymentApproval
	for rows.Next() {
		a := &DeploymentApproval{}
		var approvedBy, notes, validationID sql.NullString
		var approvedAt sql.NullTime

		if err := rows.Scan(
			&a.ID, &a.ProfileID, &a.GitCommitHash, &a.Platform, &a.Status,
			&approvedBy, &approvedAt, &notes, &validationID,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}

		a.ApprovedBy = approvedBy.String
		a.Notes = notes.String
		a.ValidationID = validationID.String
		if approvedAt.Valid {
			a.ApprovedAt = &approvedAt.Time
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
