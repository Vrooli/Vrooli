// Package metrics provides usage tracking and effectiveness ratings for skills.
// Metrics are stored in PostgreSQL while skills themselves are stored in files.
package metrics

import "time"

// SkillMetrics represents usage metrics for a skill.
type SkillMetrics struct {
	SkillID             string     `json:"skillId"`
	UsageCount          int        `json:"usageCount"`
	LastUsed            *time.Time `json:"lastUsed,omitempty"`
	EffectivenessRating *int       `json:"effectivenessRating,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
}

// RatingRequest is the request body for setting effectiveness rating.
type RatingRequest struct {
	Rating int     `json:"rating"`
	Notes  *string `json:"notes,omitempty"`
}

// UsageResponse is returned after recording skill usage.
type UsageResponse struct {
	Status     string    `json:"status"`
	UsageCount int       `json:"usageCount"`
	LastUsed   time.Time `json:"lastUsed"`
}

// RatingResponse is returned after setting effectiveness rating.
type RatingResponse struct {
	Status string `json:"status"`
	Rating int    `json:"rating"`
}
