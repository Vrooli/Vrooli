// Package metrics provides usage tracking and effectiveness ratings for prompts.
// Metrics are stored in PostgreSQL while prompts themselves are stored in files.
package metrics

import "time"

// PromptMetrics represents usage metrics for a prompt.
type PromptMetrics struct {
	PromptID            string     `json:"promptId"`
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

// UsageResponse is returned after recording prompt usage.
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
