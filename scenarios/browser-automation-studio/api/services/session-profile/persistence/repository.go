// Package persistence provides data access for session profile management.
package persistence

// Repository defines the persistence interface for session profiles.
// It isolates the session-profile service from its storage implementation.
type Repository interface {
	// Get retrieves a profile by ID.
	// Returns nil, nil if the profile does not exist.
	Get(id ProfileID) (*SessionProfile, error)

	// List returns all profiles sorted by last_used_at (desc) then created_at.
	List() ([]SessionProfile, error)

	// Create persists a new profile.
	// The profile must have a valid ID set before calling.
	Create(profile *SessionProfile) error

	// Save atomically persists the entire profile.
	// This replaces all the separate SaveX() methods with a single atomic operation.
	// The implementation writes to a temp file then renames for crash safety.
	Save(profile *SessionProfile) error

	// Delete removes a profile by ID.
	Delete(id ProfileID) error
}
