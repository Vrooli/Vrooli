package validation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository persists visual validation records.
type Repository interface {
	Create(ctx context.Context, record *Record) error
	Get(ctx context.Context, id string) (*Record, error)
	ListByProfile(ctx context.Context, profileID string) ([]*Record, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateReview(ctx context.Context, id, decision, reviewer, notes string) error
	UpdateVideo(ctx context.Context, id, videoPath string, sizeBytes, durationMs int64) error
}

// SQLRepository implements Repository with PostgreSQL.
type SQLRepository struct {
	db *sql.DB
}

// NewSQLRepository creates a new SQL-backed repository.
func NewSQLRepository(db *sql.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) Create(ctx context.Context, record *Record) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO visual_validations (id, profile_id, deployment_id, smoke_test_id, status, platform, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		record.ID, record.ProfileID, record.DeploymentID, record.SmokeTestID,
		record.Status, record.Platform, record.CreatedAt,
	)
	return err
}

func (r *SQLRepository) Get(ctx context.Context, id string) (*Record, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, profile_id, deployment_id, smoke_test_id, status,
		        video_path, video_size_bytes, video_duration_ms, platform,
		        review_decision, reviewed_by, review_notes,
		        created_at, completed_at, reviewed_at
		 FROM visual_validations WHERE id = $1`, id)

	rec := &Record{}
	var videoPath sql.NullString
	var videoSize, videoDuration sql.NullInt64
	var deploymentID, reviewDecision, reviewedBy, reviewNotes sql.NullString
	var completedAt, reviewedAt sql.NullTime

	err := row.Scan(
		&rec.ID, &rec.ProfileID, &deploymentID, &rec.SmokeTestID, &rec.Status,
		&videoPath, &videoSize, &videoDuration, &rec.Platform,
		&reviewDecision, &reviewedBy, &reviewNotes,
		&rec.CreatedAt, &completedAt, &reviewedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan validation: %w", err)
	}

	rec.DeploymentID = deploymentID.String
	rec.VideoURL = videoPath.String
	rec.VideoSizeBytes = videoSize.Int64
	rec.VideoDurationMs = videoDuration.Int64
	rec.ReviewDecision = reviewDecision.String
	rec.ReviewedBy = reviewedBy.String
	rec.ReviewNotes = reviewNotes.String
	if completedAt.Valid {
		rec.CompletedAt = &completedAt.Time
	}
	if reviewedAt.Valid {
		rec.ReviewedAt = &reviewedAt.Time
	}

	return rec, nil
}

func (r *SQLRepository) ListByProfile(ctx context.Context, profileID string) ([]*Record, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, profile_id, deployment_id, smoke_test_id, status,
		        video_path, video_size_bytes, video_duration_ms, platform,
		        review_decision, reviewed_by, review_notes,
		        created_at, completed_at, reviewed_at
		 FROM visual_validations WHERE profile_id = $1
		 ORDER BY created_at DESC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		rec := &Record{}
		var videoPath sql.NullString
		var videoSize, videoDuration sql.NullInt64
		var deploymentID, reviewDecision, reviewedBy, reviewNotes sql.NullString
		var completedAt, reviewedAt sql.NullTime

		if err := rows.Scan(
			&rec.ID, &rec.ProfileID, &deploymentID, &rec.SmokeTestID, &rec.Status,
			&videoPath, &videoSize, &videoDuration, &rec.Platform,
			&reviewDecision, &reviewedBy, &reviewNotes,
			&rec.CreatedAt, &completedAt, &reviewedAt,
		); err != nil {
			return nil, err
		}

		rec.DeploymentID = deploymentID.String
		rec.VideoURL = videoPath.String
		rec.VideoSizeBytes = videoSize.Int64
		rec.VideoDurationMs = videoDuration.Int64
		rec.ReviewDecision = reviewDecision.String
		rec.ReviewedBy = reviewedBy.String
		rec.ReviewNotes = reviewNotes.String
		if completedAt.Valid {
			rec.CompletedAt = &completedAt.Time
		}
		if reviewedAt.Valid {
			rec.ReviewedAt = &reviewedAt.Time
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *SQLRepository) UpdateStatus(ctx context.Context, id, status string) error {
	var completedAt *time.Time
	if status == "passed" || status == "failed" || status == "review_required" {
		now := time.Now()
		completedAt = &now
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE visual_validations SET status = $1, completed_at = $2 WHERE id = $3`,
		status, completedAt, id)
	return err
}

func (r *SQLRepository) UpdateReview(ctx context.Context, id, decision, reviewer, notes string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE visual_validations SET review_decision = $1, reviewed_by = $2, review_notes = $3, reviewed_at = $4 WHERE id = $5`,
		decision, reviewer, notes, now, id)
	return err
}

func (r *SQLRepository) UpdateVideo(ctx context.Context, id, videoPath string, sizeBytes, durationMs int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE visual_validations SET video_path = $1, video_size_bytes = $2, video_duration_ms = $3 WHERE id = $4`,
		videoPath, sizeBytes, durationMs, id)
	return err
}
