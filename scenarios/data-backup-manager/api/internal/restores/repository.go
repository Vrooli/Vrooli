package restores

import "context"

// Repository persists restore/verify records.
//
// seam: Repository persists restore history. Production wires SqliteRepository
// (sqlite.go); tests wire mocks.FakeRepository.
type Repository interface {
	// CreateRestore persists a new restore record and returns it with ID set.
	CreateRestore(ctx context.Context, r Restore) (Restore, error)

	// GetRestore returns the restore record by id, or ErrRestoreNotFound.
	GetRestore(ctx context.Context, id string) (Restore, error)

	// ListRestores returns restore records newest-first, optionally filtered by
	// target id. limit <= 0 returns no rows.
	ListRestores(ctx context.Context, targetID string, limit int) ([]Restore, error)
}
