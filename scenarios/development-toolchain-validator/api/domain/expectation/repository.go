package expectation

import (
	"context"
)

// StructuralRepository defines persistence operations for structural expectations.
// [REQ:REQ-P0-004] Structural Expectation Config
type StructuralRepository interface {
	// Create adds a new structural expectation.
	Create(ctx context.Context, input CreateStructuralInput) (*StructuralExpectation, error)

	// GetByID retrieves a structural expectation by ID.
	GetByID(ctx context.Context, id string) (*StructuralExpectation, error)

	// List retrieves structural expectations with optional filtering.
	List(ctx context.Context, opts ListOptions) ([]*StructuralExpectation, error)

	// Delete removes a structural expectation by ID.
	Delete(ctx context.Context, id string) error

	// DeleteByConnection removes all structural expectations for a connection.
	DeleteByConnection(ctx context.Context, connectionID string) error
}

// CLIRepository defines persistence operations for CLI assertions.
// [REQ:REQ-P0-005] CLI Tool Expectation Config
type CLIRepository interface {
	// Create adds a new CLI assertion.
	Create(ctx context.Context, input CreateCLIInput) (*CLIAssertion, error)

	// GetByID retrieves a CLI assertion by ID.
	GetByID(ctx context.Context, id string) (*CLIAssertion, error)

	// List retrieves CLI assertions with optional filtering.
	List(ctx context.Context, opts ListOptions) ([]*CLIAssertion, error)

	// Delete removes a CLI assertion by ID.
	Delete(ctx context.Context, id string) error

	// DeleteByConnection removes all CLI assertions for a connection.
	DeleteByConnection(ctx context.Context, connectionID string) error
}
