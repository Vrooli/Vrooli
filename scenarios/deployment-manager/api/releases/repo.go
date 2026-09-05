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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback is best-effort after the transaction has completed

	if release.Channel == "" {
		release.Channel = "stable"
	}
	if release.Status == "" {
		release.Status = StatusPending
	}
	now := time.Now().UTC()
	release.CreatedAt = now
	release.UpdatedAt = now

	_, err = tx.ExecContext(ctx, `
		INSERT INTO releases
			(id, profile_id, deployment_id, profile_version, git_commit_hash,
			 artifact_digest, readiness_review_key, release_version, channel, status, release_notes, released_by,
			 promoted_from_release_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
	`,
		release.ID, release.ProfileID, nullString(release.DeploymentID),
		nullIntPtr(release.ProfileVersion), release.GitCommitHash,
		nullString(release.ArtifactDigest), nullString(release.ReadinessReviewKey), release.ReleaseVersion, release.Channel, release.Status,
		nullString(release.ReleaseNotes), nullString(release.ReleasedBy),
		nullString(release.PromotedFromReleaseID), now,
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
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, readiness_review_key,
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

// ListByProfile returns recent releases for a profile (newest first).
func (r *SQLRepository) ListByProfile(ctx context.Context, profileID string, limit int) ([]*Release, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, readiness_review_key,
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

// UpdateStatus transitions the release status, stamping published_at on terminal success.
func (r *SQLRepository) UpdateStatus(ctx context.Context, releaseID, status string) error {
	now := time.Now().UTC()
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

// SetVerificationEvidence persists the per-platform verify outcomes as JSON.
func (r *SQLRepository) SetVerificationEvidence(ctx context.Context, releaseID string, items []VerificationItem) error {
	data, err := json.Marshal(items)
	if err != nil {
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
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_platforms
		SET status = $3, lpbs_artifact_id = $4, published_at = CURRENT_TIMESTAMP, error = NULL
		WHERE release_id = $1 AND platform = $2
	`, releaseID, platform, PlatformStatusPublished, artifactID)
	return err
}

// MarkPlatformStatus updates status and optional error for a platform row.
func (r *SQLRepository) MarkPlatformStatus(ctx context.Context, releaseID, platform, status, errMsg string) error {
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
	_, err := r.db.ExecContext(ctx, `
		UPDATE releases
		SET status = $4, updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = $1 AND channel = $2 AND id <> $3 AND status = $5
	`, profileID, channel, exceptReleaseID, StatusSuperseded, StatusPublished)
	return err
}

// AcquireProfileLock takes a transaction-scoped advisory lock for the profile.
// Uses a connection that is dedicated to the lock; the returned release function
// closes that connection to release the lock.
func (r *SQLRepository) AcquireProfileLock(ctx context.Context, profileID string) (bool, func(), error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return false, nil, err
	}
	var acquired bool
	// pg_try_advisory_lock with a 64-bit key derived from the profile id.
	if err := conn.QueryRowContext(ctx,
		`SELECT pg_try_advisory_lock(hashtextextended('release:' || $1, 0))`,
		profileID,
	).Scan(&acquired); err != nil {
		_ = conn.Close()
		return false, nil, err
	}
	if !acquired {
		_ = conn.Close()
		return false, func() {}, nil
	}

	release := func() {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx),
			`SELECT pg_advisory_unlock(hashtextextended('release:' || $1, 0))`,
			profileID,
		)
		_ = conn.Close()
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
	var deploymentID, artifactDigest, readinessReviewKey, releaseNotes, releasedBy, promotedFrom, readinessGoalRef, approvedAtCommit sql.NullString
	var profileVersion sql.NullInt32
	var publishedAt sql.NullTime
	var evidence []byte

	err := row.Scan(
		&rel.ID, &rel.ProfileID, &deploymentID, &profileVersion,
		&rel.GitCommitHash, &artifactDigest, &readinessReviewKey, &rel.ReleaseVersion, &rel.Channel, &rel.Status,
		&releaseNotes, &releasedBy, &promotedFrom, &readinessGoalRef, &approvedAtCommit, &evidence,
		&rel.CreatedAt, &publishedAt, &rel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rel.DeploymentID = deploymentID.String
	rel.ArtifactDigest = artifactDigest.String
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
		_ = json.Unmarshal(evidence, &rel.VerificationEvidence)
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
