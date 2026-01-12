// Package records provides desktop app record management.
package records

import "context"

// Store persists desktop app generation records.
type Store interface {
	// Upsert saves or updates a record.
	Upsert(record *DesktopAppRecord) error

	// Get retrieves a record by ID.
	Get(id string) (*DesktopAppRecord, bool)

	// List returns all records.
	List() []*DesktopAppRecord

	// DeleteByScenario removes all records for a scenario.
	DeleteByScenario(scenario string) int
}

// BuildStore provides access to build status information.
type BuildStore interface {
	// Get retrieves a build status by ID.
	Get(id string) (*BuildStatus, bool)

	// Update modifies a build status.
	Update(id string, fn func(*BuildStatus)) error
}

// BuildStatus represents the status of a desktop build.
// This is a simplified view used by the records package.
type BuildStatus struct {
	Status   string
	Metadata map[string]interface{}
}

// Service provides record management operations.
type Service interface {
	// ListWithBuilds returns all records with their build statuses.
	ListWithBuilds(ctx context.Context) ([]RecordWithBuild, error)

	// Move moves a record's output to a new location.
	Move(ctx context.Context, recordID string, target, destinationPath string) (*MoveResult, error)
}
