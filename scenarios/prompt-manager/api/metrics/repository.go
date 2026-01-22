// Package metrics provides usage tracking and effectiveness ratings for prompts.
package metrics

import (
	"database/sql"
	"time"
)

// Repository handles database operations for prompt metrics.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new metrics repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Get retrieves metrics for a specific prompt.
func (r *Repository) Get(promptID string) (*PromptMetrics, error) {
	var metrics PromptMetrics
	err := r.db.QueryRow(`
		SELECT prompt_id, usage_count, last_used, effectiveness_rating, notes
		FROM prompt_metrics WHERE prompt_id = $1
	`, promptID).Scan(&metrics.PromptID, &metrics.UsageCount, &metrics.LastUsed, &metrics.EffectivenessRating, &metrics.Notes)

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
func (r *Repository) RecordUsage(promptID string) (int, time.Time, error) {
	query := `
		INSERT INTO prompt_metrics (prompt_id, usage_count, last_used)
		VALUES ($1, 1, CURRENT_TIMESTAMP)
		ON CONFLICT (prompt_id) DO UPDATE
		SET usage_count = prompt_metrics.usage_count + 1,
		    last_used = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		RETURNING usage_count, last_used
	`

	var usageCount int
	var lastUsed time.Time
	err := r.db.QueryRow(query, promptID).Scan(&usageCount, &lastUsed)
	if err != nil {
		return 0, time.Time{}, err
	}
	return usageCount, lastUsed, nil
}

// SetRating sets the effectiveness rating for a prompt.
// Uses upsert to create the record if it doesn't exist.
func (r *Repository) SetRating(promptID string, rating int, notes *string) error {
	query := `
		INSERT INTO prompt_metrics (prompt_id, effectiveness_rating, notes)
		VALUES ($1, $2, $3)
		ON CONFLICT (prompt_id) DO UPDATE
		SET effectiveness_rating = $2,
		    notes = COALESCE($3, prompt_metrics.notes),
		    updated_at = CURRENT_TIMESTAMP
	`
	_, err := r.db.Exec(query, promptID, rating, notes)
	return err
}

// Delete removes metrics for a prompt (used when deleting a prompt).
func (r *Repository) Delete(promptID string) error {
	_, err := r.db.Exec("DELETE FROM prompt_metrics WHERE prompt_id = $1", promptID)
	return err
}
