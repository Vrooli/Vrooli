// Package skills provides the core domain types and operations for skill management.
package skills

import (
	"context"
	"time"

	"prompt-manager/metrics"
)

// MetricsAdapter adapts metrics.Repository to the MetricsService interface.
// This allows the skills package to depend on an interface while using the
// metrics package implementation in production.
type MetricsAdapter struct {
	repo *metrics.Repository
}

// NewMetricsAdapter creates a new adapter wrapping a metrics.Repository.
func NewMetricsAdapter(repo *metrics.Repository) *MetricsAdapter {
	return &MetricsAdapter{repo: repo}
}

// WithContext returns a request-scoped metrics adapter when handlers run
// against a routed database.
func (a *MetricsAdapter) WithContext(ctx context.Context) MetricsService {
	return &MetricsAdapter{repo: a.repo.WithContext(ctx)}
}

// Get retrieves metrics for a specific skill.
func (a *MetricsAdapter) Get(skillID string) (*SkillMetrics, error) {
	m, err := a.repo.Get(skillID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	// Convert metrics.SkillMetrics to skills.SkillMetrics
	return &SkillMetrics{
		SkillID:             m.SkillID,
		UsageCount:          m.UsageCount,
		LastUsed:            m.LastUsed,
		EffectivenessRating: m.EffectivenessRating,
		Notes:               m.Notes,
	}, nil
}

// RecordUsage increments the usage count and updates last_used timestamp.
func (a *MetricsAdapter) RecordUsage(skillID string) (int, time.Time, error) {
	return a.repo.RecordUsage(skillID)
}

// SetRating sets the effectiveness rating for a skill.
func (a *MetricsAdapter) SetRating(skillID string, rating int, notes *string) error {
	return a.repo.SetRating(skillID, rating, notes)
}

// Delete removes metrics for a skill.
func (a *MetricsAdapter) Delete(skillID string) error {
	return a.repo.Delete(skillID)
}
