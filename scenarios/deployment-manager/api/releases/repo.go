package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SQLRepository implements Repository with PostgreSQL.
type SQLRepository struct {
	db *sql.DB
}

// NewSQLRepository creates a new SQL-backed release repository.
func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

// EnsureSchema creates the releases tables if they don't exist.
func (r *SQLRepository) EnsureSchema(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS releases (
			id                        TEXT PRIMARY KEY,
			profile_id                TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
			deployment_id             TEXT,
			profile_version           INTEGER,
			git_commit_hash           TEXT NOT NULL,
			release_version           TEXT NOT NULL,
			channel                   TEXT NOT NULL DEFAULT 'stable',
			status                    TEXT NOT NULL DEFAULT 'pending',
			release_notes             TEXT,
			released_by               TEXT,
			promoted_from_release_id  TEXT REFERENCES releases(id) ON DELETE SET NULL,
			verification_evidence     JSONB,
			created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			published_at              TIMESTAMPTZ,
			updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (profile_id, git_commit_hash, channel)
		);
		CREATE TABLE IF NOT EXISTS release_platforms (
			release_id        TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
			platform          TEXT NOT NULL,
			status            TEXT NOT NULL DEFAULT 'pending',
			approval_id       TEXT,
			lpbs_artifact_id  BIGINT,
			published_at      TIMESTAMPTZ,
			verified_at       TIMESTAMPTZ,
			error             TEXT,
			PRIMARY KEY (release_id, platform)
		);
		CREATE INDEX IF NOT EXISTS idx_releases_profile_channel
			ON releases (profile_id, channel, created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_releases_status
			ON releases (status);
		CREATE INDEX IF NOT EXISTS idx_releases_commit
			ON releases (profile_id, git_commit_hash);
		CREATE INDEX IF NOT EXISTS idx_releases_deployment
			ON releases (deployment_id);
		CREATE INDEX IF NOT EXISTS idx_release_platforms_status
			ON release_platforms (status);
	`)
	return err
}

// Insert creates a release row with per-platform rows in one transaction.
func (r *SQLRepository) Insert(ctx context.Context, release *Release) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

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
			 release_version, channel, status, release_notes, released_by,
			 promoted_from_release_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12)
	`,
		release.ID, release.ProfileID, nullString(release.DeploymentID),
		nullIntPtr(release.ProfileVersion), release.GitCommitHash,
		release.ReleaseVersion, release.Channel, release.Status,
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
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash,
		       release_version, channel, status, release_notes, released_by,
		       promoted_from_release_id, verification_evidence, created_at,
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
		SELECT id, profile_id, deployment_id, profile_version, git_commit_hash,
		       release_version, channel, status, release_notes, released_by,
		       promoted_from_release_id, verification_evidence, created_at,
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
		UPDATE releases SET verification_evidence = $2, updated_at = NOW() WHERE id = $1
	`, releaseID, data)
	return err
}

// MarkPlatformPublished stamps the artifact id and flips the platform to published.
func (r *SQLRepository) MarkPlatformPublished(ctx context.Context, releaseID, platform string, artifactID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE release_platforms
		SET status = $3, lpbs_artifact_id = $4, published_at = NOW(), error = NULL
		WHERE release_id = $1 AND platform = $2
	`, releaseID, platform, PlatformStatusPublished, artifactID)
	return err
}

// MarkPlatformStatus updates status and optional error for a platform row.
func (r *SQLRepository) MarkPlatformStatus(ctx context.Context, releaseID, platform, status, errMsg string) error {
	if status == PlatformStatusPublished {
		_, err := r.db.ExecContext(ctx, `
			UPDATE release_platforms
			SET status = $3, verified_at = NOW(), error = $4
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
		SET status = $4, updated_at = NOW()
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
		_, _ = conn.ExecContext(context.Background(),
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
	var deploymentID, releaseNotes, releasedBy, promotedFrom sql.NullString
	var profileVersion sql.NullInt32
	var publishedAt sql.NullTime
	var evidence []byte

	err := row.Scan(
		&rel.ID, &rel.ProfileID, &deploymentID, &profileVersion,
		&rel.GitCommitHash, &rel.ReleaseVersion, &rel.Channel, &rel.Status,
		&releaseNotes, &releasedBy, &promotedFrom, &evidence,
		&rel.CreatedAt, &publishedAt, &rel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rel.DeploymentID = deploymentID.String
	rel.ReleaseNotes = releaseNotes.String
	rel.ReleasedBy = releasedBy.String
	rel.PromotedFromReleaseID = promotedFrom.String
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
