// DOC: docs/internal/SEAMS.md#postgresql-database
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/reference/data-model.md
package skill

import (
	"context"
)

// Repository defines the interface for skill connection persistence.
// This abstraction allows swapping storage implementations (PostgreSQL, SQLite, in-memory)
// without affecting business logic.
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store
type Repository interface {
	// Connect creates a new skill-reference connection and returns the created entity.
	Connect(ctx context.Context, input ConnectInput) (*Connection, error)

	// GetByID retrieves a connection by its UUID.
	GetByID(ctx context.Context, id string) (*Connection, error)

	// GetByReferenceAndSkill retrieves a connection by reference ID and skill ID.
	GetByReferenceAndSkill(ctx context.Context, referenceID, skillID string) (*Connection, error)

	// List retrieves connections with optional filtering and pagination.
	List(ctx context.Context, opts ListOptions) ([]*Connection, error)

	// Update modifies an existing connection (e.g., to refresh version/hash).
	Update(ctx context.Context, id string, input UpdateInput) (*Connection, error)

	// Disconnect removes a skill-reference connection by ID.
	Disconnect(ctx context.Context, id string) error

	// DisconnectByReferenceAndSkill removes a connection by reference ID and skill ID.
	DisconnectByReferenceAndSkill(ctx context.Context, referenceID, skillID string) error
}
