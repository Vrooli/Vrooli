package deployments

import (
	"context"
	"database/sql"
	"fmt"
	"time"
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
}

// SQLApprovalsRepository implements ApprovalsRepository with PostgreSQL.
type SQLApprovalsRepository struct {
	db *sql.DB
}

// NewSQLApprovalsRepository creates a new SQL-backed approvals repository.
func NewSQLApprovalsRepository(db *sql.DB) *SQLApprovalsRepository {
	return &SQLApprovalsRepository{db: db}
}

// EnsureSchema creates the approval tables if they don't exist.
func (r *SQLApprovalsRepository) EnsureSchema(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS deployment_approvals (
			id              TEXT PRIMARY KEY,
			profile_id      TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
			git_commit_hash TEXT NOT NULL,
			platform        TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			approved_by     TEXT,
			approved_at     TIMESTAMPTZ,
			notes           TEXT,
			validation_id   TEXT,
			created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (profile_id, git_commit_hash, platform)
		);
		CREATE TABLE IF NOT EXISTS profile_required_platforms (
			profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
			platform   TEXT NOT NULL,
			PRIMARY KEY (profile_id, platform)
		);
		CREATE INDEX IF NOT EXISTS idx_approvals_profile_commit
			ON deployment_approvals (profile_id, git_commit_hash);
		CREATE INDEX IF NOT EXISTS idx_approvals_pending
			ON deployment_approvals (status) WHERE status = 'pending';
	`)
	return err
}

func (r *SQLApprovalsRepository) Create(ctx context.Context, approval *DeploymentApproval) error {
	// Mark previous approvals for same profile+platform as stale
	if err := r.MarkStale(ctx, approval.ProfileID, approval.Platform, approval.GitCommitHash); err != nil {
		return fmt.Errorf("mark stale: %w", err)
	}

	_, err := r.db.ExecContext(ctx,
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
	row := r.db.QueryRowContext(ctx,
		`SELECT id, profile_id, git_commit_hash, platform, status,
		        approved_by, approved_at, notes, validation_id, created_at, updated_at
		 FROM deployment_approvals WHERE id = $1`, id)
	return scanApproval(row)
}

func (r *SQLApprovalsRepository) ListByCommit(ctx context.Context, profileID, commitHash string) ([]*DeploymentApproval, error) {
	rows, err := r.db.QueryContext(ctx,
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
	rows, err := r.db.QueryContext(ctx,
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
	_, err := r.db.ExecContext(ctx,
		`UPDATE deployment_approvals
		 SET status = $2, approved_by = $3, approved_at = $4, notes = $5, updated_at = $4
		 WHERE id = $1`,
		id, decision, reviewer, now, notes)
	return err
}

func (r *SQLApprovalsRepository) MarkStale(ctx context.Context, profileID, platform, exceptCommit string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE deployment_approvals
		 SET status = 'stale', updated_at = NOW()
		 WHERE profile_id = $1 AND platform = $2
		   AND git_commit_hash != $3
		   AND status NOT IN ('stale', 'rejected')`,
		profileID, platform, exceptCommit)
	return err
}

func (r *SQLApprovalsRepository) GetRequiredPlatforms(ctx context.Context, profileID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
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
	defer tx.Rollback() //nolint:errcheck

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
	required, err := r.GetRequiredPlatforms(ctx, profileID)
	if err != nil {
		return nil, err
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
		Ready:         true,
		Platforms:     make([]PlatformGateStatus, 0, len(required)),
	}

	for _, platform := range required {
		ps := PlatformGateStatus{
			Platform: platform,
			Required: true,
		}
		if a, ok := approvalByPlatform[platform]; ok {
			ps.Status = a.Status
		} else {
			ps.Status = "missing"
		}
		if ps.Status != ApprovalStatusApproved {
			gate.Ready = false
		}
		gate.Platforms = append(gate.Platforms, ps)
	}

	return gate, nil
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
