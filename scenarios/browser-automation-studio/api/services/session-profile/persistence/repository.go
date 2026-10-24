// Package persistence provides data access for session profile management.
package persistence

import "errors"

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrInvalidProfileID = errors.New("invalid profile id")
)

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

	// Update changes the current profile under exclusive write ownership.
	// A rejected callback or commit leaves the previous snapshot unchanged.
	// The callback must not call another repository writer or change the identity.
	Update(id ProfileID, modify func(*SessionProfile) error) (*SessionProfile, error)

	// Delete removes a profile by ID.
	Delete(id ProfileID) error
}
