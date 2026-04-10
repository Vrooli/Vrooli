// Package metrics provides usage tracking and effectiveness ratings for skills.
package metrics

import (
	"database/sql"
	"time"
)

// Repository handles database operations for skill metrics.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new metrics repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Get retrieves metrics for a specific skill.
func (r *Repository) Get(skillID string) (*SkillMetrics, error) {
	var metrics SkillMetrics
	err := r.db.QueryRow(`
		SELECT skill_id, usage_count, last_used, effectiveness_rating, notes
		FROM skill_metrics WHERE skill_id = $1
	`, skillID).Scan(&metrics.SkillID, &metrics.UsageCount, &metrics.LastUsed, &metrics.EffectivenessRating, &metrics.Notes)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// RecordUsage increments the usage count and updates last_used timestamp.
// Uses upsert to create the record if it doesn't exist.
func (r *Repository) RecordUsage(skillID string) (int, time.Time, error) {
	query := `
		INSERT INTO skill_metrics (skill_id, usage_count, last_used)
		VALUES ($1, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (skill_id) DO UPDATE
		SET usage_count = skill_metrics.usage_count + 1,
		    last_used = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING usage_count, last_used
	`

	var usageCount int
	var lastUsed time.Time
	err := r.db.QueryRow(query, skillID).Scan(&usageCount, &lastUsed)
	if err != nil {
		return 0, time.Time{}, err
	}
	return usageCount, lastUsed, nil
}

// SetRating sets the effectiveness rating for a skill.
// Uses upsert to create the record if it doesn't exist.
func (r *Repository) SetRating(skillID string, rating int, notes *string) error {
	query := `
		INSERT INTO skill_metrics (skill_id, effectiveness_rating, notes)
		VALUES ($1, $2, $3)
		ON CONFLICT (skill_id) DO UPDATE
		SET effectiveness_rating = $2,
		    notes = COALESCE($3, skill_metrics.notes),
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, skillID, rating, notes)
	return err
}

// Delete removes metrics for a skill (used when deleting a skill).
func (r *Repository) Delete(skillID string) error {
	_, err := r.db.Exec("DELETE FROM skill_metrics WHERE skill_id = $1", skillID)
	return err
}
