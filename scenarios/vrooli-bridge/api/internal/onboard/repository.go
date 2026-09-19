package onboard

import "context"

// Repository is the persistence seam the onboard service depends on. Production
// wires the sqlite-backed implementation from sqlite.go; service unit tests wire
// the in-memory fake. It persists the durable op record and its append-only
// step-event history.
type Repository interface {
	// Create persists op as a new onboarding op. The implementation populates ID
	// (when empty), CreatedAt, and defaults State to PENDING. Returns the op.
	Create(ctx context.Context, op Op) (Op, error)

	// Get returns the op with the given id or ErrOpNotFound{id}.
	Get(ctx context.Context, id string) (Op, error)

	// List returns ops newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Op, error)

	// Update persists the mutable columns of op (state, node_id, failure_reason,
	// exit_code, started_at, finished_at) matched by op.ID and returns the stored
	// op. Returns ErrOpNotFound when no row matches.
	Update(ctx context.Context, op Op) (Op, error)

	// AppendEvent appends one step event to the op's history. Append-only: there
	// is no update/delete. The event's Sequence is assigned by the caller (the
	// orchestrator's monotonic per-op counter); a duplicate (op_id, sequence) is
	// ignored so a replay is idempotent.
	AppendEvent(ctx context.Context, ev StepEvent) error

	// ListEvents returns an op's full step-event history in Sequence order.
	ListEvents(ctx context.Context, opID string) ([]StepEvent, error)

	// ListNonTerminal returns every op that is not in a terminal state. Used on
	// startup to reconcile ops orphaned by a control-plane restart.
	ListNonTerminal(ctx context.Context) ([]Op, error)

	// DeleteFailed permanently removes a FAILED op and its event history. It must
	// reject every other lifecycle state so a UI cleanup can never affect a live
	// operation or a successfully paired node.
	DeleteFailed(ctx context.Context, id string) error
}
