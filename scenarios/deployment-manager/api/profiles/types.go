// Package profiles provides deployment profile management with versioning.
package profiles

import (
	"context"
	"errors"
	"time"
)

// Domain-specific errors for the repository layer.
// These errors abstract away database implementation details from handlers.
var (
	// ErrNotFound indicates the requested profile does not exist.
	ErrNotFound = errors.New("profile not found")
)

// Profile represents a deployment profile stored in the database.
type Profile struct {
	ID        string
	Name      string
	Scenario  string
	Tiers     interface{}
	Swaps     interface{}
	Secrets   interface{}
	Settings  interface{}
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}

// Version represents a versioned snapshot of a profile.
type Version struct {
	ProfileID         string
	Version           int
	Name              string
	Scenario          string
	Tiers             interface{}
	Swaps             interface{}
	Secrets           interface{}
	Settings          interface{}
	CreatedAt         time.Time
	CreatedBy         string
	ChangeDescription string
}

// Repository defines the interface for profile storage operations.
// This seam allows tests to substitute the real database with mocks.
type Repository interface {
	// List returns all profiles ordered by creation date (newest first).
	List(ctx context.Context) ([]Profile, error)

	// Get retrieves a profile by ID or name.
	Get(ctx context.Context, idOrName string) (*Profile, error)

	// Create stores a new profile and returns its generated ID.
	Create(ctx context.Context, profile *Profile) (string, error)

	// Update modifies an existing profile and increments its version.
	Update(ctx context.Context, idOrName string, updates map[string]interface{}) (*Profile, error)

	// Delete removes a profile by ID or name. Returns true if a row was deleted.
	Delete(ctx context.Context, idOrName string) (bool, error)

	// GetVersions returns the version history for a profile.
	GetVersions(ctx context.Context, idOrName string) ([]Version, error)

	// GetScenarioAndTier retrieves just the scenario name and tier count for a profile.
	GetScenarioAndTier(ctx context.Context, idOrName string) (scenario string, tierCount int, err error)
}
