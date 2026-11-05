// Package metrics provides usage tracking and effectiveness ratings for skills.
package metrics

import "time"

// MetricsRepository defines the interface for skill metrics storage operations.
// This is the testing seam for the metrics domain.
// Implementations: Repository (database, production), MockRepository (testing).
type MetricsRepository interface {
	// Get retrieves metrics for a specific skill.
	Get(skillID string) (*SkillMetrics, error)

	// RecordUsage increments the usage count and updates last_used timestamp.
	// Returns (usageCount, lastUsed, error).
	RecordUsage(skillID string) (int, time.Time, error)

	// SetRating sets the effectiveness rating for a skill.
	SetRating(skillID string, rating int, notes *string) error

	// Delete removes metrics for a skill (used when deleting a skill).
	Delete(skillID string) error
}
