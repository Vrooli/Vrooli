// Package skills provides the core domain types and operations for skill management.
//
// DOC: docs/concepts/ARCHITECTURE.md#interface-based-design
// DOC: docs/internal/SEAMS.md#1-skillsskillstore-interface
package skills

import (
	"context"
	"time"
)

// SkillStore defines the interface for skill storage operations.
// This is the primary testing seam for the skills domain.
// Implementations: Store (file-based, production), MockStore (testing).
type SkillStore interface {
	// GetAll returns all skills from all folders.
	GetAll() ([]Metadata, error)

	// FindByID searches all folders for a skill with the given ID.
	// Returns the skill metadata and the folder it was found in.
	FindByID(id string) (*Metadata, string, error)

	// LoadMetadata loads skill metadata from a folder's metadata.json.
	LoadMetadata(folder string) ([]Metadata, error)

	// SaveMetadata saves skill metadata to a folder's metadata.json.
	SaveMetadata(folder string, skills []Metadata) error

	// GetContent reads a skill's markdown content from disk.
	GetContent(folder, filename string) (string, error)

	// SaveContent writes a skill's markdown content to disk.
	SaveContent(folder, filename, content string) error

	// DeleteContent removes a skill's markdown file from disk.
	DeleteContent(folder, filename string) error

	// GetVersions returns version history for a skill.
	GetVersions(skillID string) ([]SkillVersion, error)

	// SaveVersion saves the current state as a new version.
	SaveVersion(skillID, folder string, skill *Metadata, content string) error

	// GetVersionContent returns the content of a specific version.
	GetVersionContent(skillID string, version int) (*SkillVersion, error)

	// LoadVersions loads all version files for a folder.
	LoadVersions(folder string) (map[string]*VersionFile, error)

	// SaveVersions saves version files for a folder.
	SaveVersions(folder string, versions map[string]*VersionFile) error
}

// MetricsService defines the interface for skill metrics operations.
// This is the testing seam for metrics functionality used by skills handlers.
// Implementations: metrics.Repository (database, production), MockMetricsService (testing).
type MetricsService interface {
	// Get retrieves metrics for a specific skill.
	Get(skillID string) (*SkillMetrics, error)

	// RecordUsage increments the usage count and updates last_used timestamp.
	RecordUsage(skillID string) (int, time.Time, error)

	// SetRating sets the effectiveness rating for a skill.
	SetRating(skillID string, rating int, notes *string) error

	// Delete removes metrics for a skill.
	Delete(skillID string) error
}

// SkillMetrics represents usage metrics for a skill.
// This mirrors metrics.SkillMetrics to avoid circular imports.
type SkillMetrics struct {
	SkillID             string     `json:"skillId"`
	UsageCount          int        `json:"usageCount"`
	LastUsed            *time.Time `json:"lastUsed,omitempty"`
	EffectivenessRating *int       `json:"effectivenessRating,omitempty"`
	Notes               *string    `json:"notes,omitempty"`
}

// AISearchIndexer defines the interface for AI search index operations.
// This allows skills handlers to trigger index updates without circular imports.
// Implementation is provided by the aisearch.Service.
type AISearchIndexer interface {
	// IndexSkill indexes a single skill into the vector store.
	IndexSkill(ctx context.Context, skillID string) error

	// DeleteFromIndex removes a skill from the vector index.
	DeleteFromIndex(ctx context.Context, skillID string) error
}
