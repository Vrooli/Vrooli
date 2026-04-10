// DOC: docs/internal/SEAMS.md#postgresql-database
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/reference/data-model.md
package reference

import (
	"context"
)

// Repository defines the interface for reference scenario persistence.
// This abstraction allows swapping storage implementations (PostgreSQL, SQLite, in-memory)
// without affecting business logic.
// [REQ:P0-001] Reference Scenario Database Schema
type Repository interface {
	// Create stores a new reference scenario and returns the created entity.
	Create(ctx context.Context, input CreateInput) (*Reference, error)

	// GetByID retrieves a reference by its UUID.
	GetByID(ctx context.Context, id string) (*Reference, error)

	// GetBySlug retrieves a reference by its unique slug.
	GetBySlug(ctx context.Context, slug string) (*Reference, error)

	// List retrieves references with optional filtering and pagination.
	List(ctx context.Context, opts ListOptions) ([]*Reference, error)

	// Update modifies an existing reference.
	Update(ctx context.Context, id string, input UpdateInput) (*Reference, error)

	// Delete removes a reference by ID.
	Delete(ctx context.Context, id string) error
}
