package conflicts

import "context"

// Repository is the persistence seam for conflicts. Production wires
// the sqlite-backed implementation; service tests wire mocks.FakeRepository.
type Repository interface {
	UpsertConflict(ctx context.Context, c Conflict) (Conflict, error)
	GetConflict(ctx context.Context, id string) (Conflict, error)
	ListConflicts(ctx context.Context, f ListConflictsFilter) (ConflictPage, error)
}

// ListConflictsFilter scopes ListConflicts.
type ListConflictsFilter struct {
	Scenario  string
	Types     []string
	PageSize  int
	PageToken string
}

// ConflictPage is the paginated result.
type ConflictPage struct {
	Conflicts     []Conflict
	NextPageToken string
}
