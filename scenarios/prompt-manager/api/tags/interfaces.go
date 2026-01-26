// Package tags provides tag management for prompt categorization.
package tags

// TagRepository defines the interface for tag storage operations.
// This is the testing seam for the tags domain.
// Implementations: Repository (database, production), MockRepository (testing).
type TagRepository interface {
	// GetAll retrieves all tags ordered by name.
	GetAll() ([]Tag, error)

	// Create adds a new tag.
	Create(tag *Tag) error
}
