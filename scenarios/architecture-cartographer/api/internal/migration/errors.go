package migration

import "fmt"

// ErrMigrationNotFound is returned when a migration id does not resolve.
type ErrMigrationNotFound struct{ ID string }

func (e ErrMigrationNotFound) Error() string {
	return fmt.Sprintf("migration %q not found", e.ID)
}

// ErrFindingNotFound is returned when a (migration, stableID) pair does not
// resolve.
type ErrFindingNotFound struct {
	MigrationID string
	StableID    string
}

func (e ErrFindingNotFound) Error() string {
	return fmt.Sprintf("finding %q not found in migration %q", e.StableID, e.MigrationID)
}

// ErrInvalidInput is returned for empty/invalid required arguments.
type ErrInvalidInput struct{ Reason string }

func (e ErrInvalidInput) Error() string {
	return fmt.Sprintf("invalid input: %s", e.Reason)
}
