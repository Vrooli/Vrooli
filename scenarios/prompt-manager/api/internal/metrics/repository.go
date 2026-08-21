// Package metrics provides usage tracking and effectiveness ratings for skills.
package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Repository handles database operations for skill metrics.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db  databaseExecutor
	ctx context.Context
}

type databaseExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// NewRepository creates a new metrics repository.
func NewRepository(db databaseExecutor) *Repository {
	return &Repository{db: db, ctx: context.Background()}
}

// WithContext returns a request-scoped repository. RoutedDB uses this context
// to select a leased test pool without mutable shared request state.
func (r *Repository) WithContext(ctx context.Context) *Repository {
	copy := *r
	copy.ctx = ctx
	return &copy
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseOptionalTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || raw.String == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, raw.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Get retrieves metrics for a specific skill.
func (r *Repository) Get(skillID string) (*SkillMetrics, error) {
	var (
		metrics  SkillMetrics
		lastUsed sql.NullString
	)
	err := r.db.QueryRowContext(r.ctx, `
		SELECT skill_id, usage_count, last_used, effectiveness_rating, notes
		FROM skill_metrics WHERE skill_id = ?
	`, skillID).Scan(&metrics.SkillID, &metrics.UsageCount, &lastUsed, &metrics.EffectivenessRating, &metrics.Notes)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsed, err := parseOptionalTime(lastUsed)
	if err != nil {
		return nil, err
	}
	metrics.LastUsed = parsed
	return &metrics, nil
}

// RecordUsage increments the usage count and updates last_used timestamp.
// Uses upsert to create the record if it doesn't exist.
func (r *Repository) RecordUsage(skillID string) (int, time.Time, error) {
	now := time.Now().UTC()
	nowText := formatTime(now)
	query := `
		INSERT INTO skill_metrics (id, skill_id, usage_count, last_used, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, ?)
		ON CONFLICT(skill_id) DO UPDATE
		SET usage_count = skill_metrics.usage_count + 1,
		    last_used = excluded.last_used,
		    updated_at = excluded.updated_at
		RETURNING usage_count
	`

	var usageCount int
	err := r.db.QueryRowContext(r.ctx, query, uuid.NewString(), skillID, nowText, nowText, nowText).Scan(&usageCount)
	if err != nil {
		return 0, time.Time{}, err
	}
	return usageCount, now, nil
}

// SetRating sets the effectiveness rating for a skill.
// Uses upsert to create the record if it doesn't exist.
func (r *Repository) SetRating(skillID string, rating int, notes *string) error {
	now := formatTime(time.Now().UTC())
	query := `
		INSERT INTO skill_metrics (id, skill_id, usage_count, effectiveness_rating, notes, created_at, updated_at)
		VALUES (?, ?, 0, ?, ?, ?, ?)
		ON CONFLICT(skill_id) DO UPDATE
		SET effectiveness_rating = excluded.effectiveness_rating,
		    notes = COALESCE(excluded.notes, skill_metrics.notes),
		    updated_at = excluded.updated_at
	`
	_, err := r.db.ExecContext(r.ctx, query, uuid.NewString(), skillID, rating, notes, now, now)
	return err
}

// Delete removes metrics for a skill (used when deleting a skill).
func (r *Repository) Delete(skillID string) error {
	_, err := r.db.ExecContext(r.ctx, "DELETE FROM skill_metrics WHERE skill_id = ?", skillID)
	return err
}
