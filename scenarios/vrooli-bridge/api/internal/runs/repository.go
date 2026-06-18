package runs

import "context"

// Repository is the persistence seam the runs service depends on. Production
// wires the sqlite-backed implementation from sqlite.go; service unit tests
// wire mocks.FakeRepository. It persists both the durable run record and its
// append-only event history.
type Repository interface {
	// Create persists r as a new run. The implementation populates ID (when
	// empty) and CreatedAt. Returns the persisted Run.
	Create(ctx context.Context, r Run) (Run, error)

	// Get returns the run with the given id or ErrRunNotFound{id}.
	Get(ctx context.Context, id string) (Run, error)

	// List returns runs newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Run, error)

	// Update persists the mutable lifecycle columns of r (status, exit_code,
	// started_at, finished_at, artifact_refs) matched by r.ID and returns the
	// stored Run. Returns ErrRunNotFound when no row matches.
	Update(ctx context.Context, r Run) (Run, error)

	// AppendEvent appends one event to the run's history. Append-only: there is
	// no update/delete. The event's Sequence is assigned by the caller (the
	// node's monotonic per-run counter).
	AppendEvent(ctx context.Context, ev RunEvent) error

	// ListEvents returns a run's full event history in Sequence order.
	ListEvents(ctx context.Context, runID string) ([]RunEvent, error)
}
