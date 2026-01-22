// Package prompts provides the core domain types and operations for prompt management.
package prompts

import (
	"time"

	"prompt-manager/metrics"
)

// MetricsAdapter adapts metrics.Repository to the MetricsService interface.
// This allows the prompts package to depend on an interface while using the
// metrics package implementation in production.
type MetricsAdapter struct {
	repo *metrics.Repository
}

// NewMetricsAdapter creates a new adapter wrapping a metrics.Repository.
func NewMetricsAdapter(repo *metrics.Repository) *MetricsAdapter {
	return &MetricsAdapter{repo: repo}
}

// Get retrieves metrics for a specific prompt.
func (a *MetricsAdapter) Get(promptID string) (*PromptMetrics, error) {
	m, err := a.repo.Get(promptID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	// Convert metrics.PromptMetrics to prompts.PromptMetrics
	return &PromptMetrics{
		PromptID:            m.PromptID,
		UsageCount:          m.UsageCount,
		LastUsed:            m.LastUsed,
		EffectivenessRating: m.EffectivenessRating,
		Notes:               m.Notes,
	}, nil
}

// RecordUsage increments the usage count and updates last_used timestamp.
func (a *MetricsAdapter) RecordUsage(promptID string) (int, time.Time, error) {
	return a.repo.RecordUsage(promptID)
}

// SetRating sets the effectiveness rating for a prompt.
func (a *MetricsAdapter) SetRating(promptID string, rating int, notes *string) error {
	return a.repo.SetRating(promptID, rating, notes)
}

// Delete removes metrics for a prompt.
func (a *MetricsAdapter) Delete(promptID string) error {
	return a.repo.Delete(promptID)
}
