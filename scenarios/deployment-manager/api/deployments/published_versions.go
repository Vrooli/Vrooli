package deployments

import (
	"context"
	"database/sql"
	"time"

	"deployment-manager/shared"
)

// PublishedVersion records a single version publish event (append-only).
type PublishedVersion struct {
	ID            int       `json:"id"`
	ProfileID     string    `json:"profile_id"`
	Platform      string    `json:"platform"`
	Version       string    `json:"version"`
	GitCommitHash string    `json:"git_commit_hash,omitempty"`
	ArtifactID    int64     `json:"artifact_id,omitempty"`
	DeploymentID  string    `json:"deployment_id,omitempty"`
	ReleaseID     string    `json:"release_id,omitempty"`
	PublishedAt   time.Time `json:"published_at"`
}

// PublishedVersionsRepository persists published version records.
type PublishedVersionsRepository interface {
	RecordPublish(ctx context.Context, record *PublishedVersion) error
	GetLatestByProfile(ctx context.Context, profileID string) ([]PublishedVersion, error)
	GetHistory(ctx context.Context, profileID, platform string, limit int) ([]PublishedVersion, error)
}

// SQLPublishedVersionsRepository implements PublishedVersionsRepository with PostgreSQL.
type SQLPublishedVersionsRepository struct {
	db shared.RoutedDBTX
}

// NewSQLPublishedVersionsRepository creates a new SQL-backed published versions repository.
func NewSQLPublishedVersionsRepository(db shared.RoutedDBTX) *SQLPublishedVersionsRepository {
	return &SQLPublishedVersionsRepository{db: db}
}

// RecordPublish inserts a new published version record.
func (r *SQLPublishedVersionsRepository) RecordPublish(ctx context.Context, record *PublishedVersion) error {
	if record.PublishedAt.IsZero() {
		record.PublishedAt = time.Now()
	}
	return r.db.QueryRowContext(ctx,
		`INSERT INTO published_versions
			(profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		record.ProfileID, record.Platform, record.Version,
		nullString(record.GitCommitHash),
		nullInt64(record.ArtifactID),
		nullString(record.DeploymentID),
		nullString(record.ReleaseID),
		record.PublishedAt,
	).Scan(&record.ID)
}

// GetLatestByProfile returns the latest published version per platform for a profile.
func (r *SQLPublishedVersionsRepository) GetLatestByProfile(ctx context.Context, profileID string) ([]PublishedVersion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT ON (platform)
			id, profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at
		 FROM published_versions
		 WHERE profile_id = $1
		 ORDER BY platform, published_at DESC`,
		profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPublishedVersions(rows)
}

// GetHistory returns publish history for a profile, optionally filtered by platform.
func (r *SQLPublishedVersionsRepository) GetHistory(ctx context.Context, profileID, platform string, limit int) ([]PublishedVersion, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error
	if platform != "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at
			 FROM published_versions
			 WHERE profile_id = $1 AND platform = $2
			 ORDER BY published_at DESC
			 LIMIT $3`,
			profileID, platform, limit)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT id, profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at
			 FROM published_versions
			 WHERE profile_id = $1
			 ORDER BY published_at DESC
			 LIMIT $2`,
			profileID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPublishedVersions(rows)
}

func scanPublishedVersions(rows *sql.Rows) ([]PublishedVersion, error) {
	var result []PublishedVersion
	for rows.Next() {
		var pv PublishedVersion
		var gitHash, deploymentID, releaseID sql.NullString
		var artifactID sql.NullInt64

		if err := rows.Scan(
			&pv.ID, &pv.ProfileID, &pv.Platform, &pv.Version,
			&gitHash, &artifactID, &deploymentID, &releaseID, &pv.PublishedAt,
		); err != nil {
			return nil, err
		}

		pv.GitCommitHash = gitHash.String
		pv.DeploymentID = deploymentID.String
		pv.ReleaseID = releaseID.String
		if artifactID.Valid {
			pv.ArtifactID = artifactID.Int64
		}
		result = append(result, pv)
	}
	return result, rows.Err()
}

// nullInt64 converts a zero int64 to sql.NullInt64.
func nullInt64(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}
