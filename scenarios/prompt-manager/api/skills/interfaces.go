// Package prompts provides the core domain types and operations for prompt management.
package prompts

import "time"

// PromptStore defines the interface for prompt storage operations.
// This is the primary testing seam for the prompts domain.
// Implementations: Store (file-based, production), MockStore (testing).
type PromptStore interface {
	// GetAll returns all prompts from all folders.
	GetAll() ([]Metadata, error)

	// FindByID searches all folders for a prompt with the given ID.
	// Returns the prompt metadata and the folder it was found in.
	FindByID(id string) (*Metadata, string, error)

	// LoadMetadata loads prompt metadata from a folder's metadata.json.
	LoadMetadata(folder string) ([]Metadata, error)

	// SaveMetadata saves prompt metadata to a folder's metadata.json.
	SaveMetadata(folder string, prompts []Metadata) error

	// GetContent reads a prompt's markdown content from disk.
	GetContent(folder, filename string) (string, error)

	// SaveContent writes a prompt's markdown content to disk.
	SaveContent(folder, filename, content string) error

	// DeleteContent removes a prompt's markdown file from disk.
	DeleteContent(folder, filename string) error
}

// MetricsService defines the interface for prompt metrics operations.
// This is the testing seam for metrics functionality used by prompts handlers.
// Implementations: metrics.Repository (database, production), MockMetricsService (testing).
type MetricsService interface {
	// Get retrieves metrics for a specific prompt.
	Get(promptID string) (*PromptMetrics, error)

	// RecordUsage increments the usage count and updates last_used timestamp.
	RecordUsage(promptID string) (int, time.Time, error)

	// SetRating sets the effectiveness rating for a prompt.
	SetRating(promptID string, rating int, notes *string) error

	// Delete removes metrics for a prompt.
	Delete(promptID string) error
}

// PromptMetrics represents usage metrics for a prompt.
// This mirrors metrics.PromptMetrics to avoid circular imports.
type PromptMetrics struct {
	PromptID            string     `json:"promptId"`
	UsageCount          int        `json:"usageCount"`
	LastUsed            *time.Time `json:"lastUsed,omitempty"`
	EffectivenessRating *int       `json:"effectivenessRating,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
}
