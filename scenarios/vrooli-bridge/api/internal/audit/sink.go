package audit

import "context"

// Sink is the WRITE seam: the append-only accountability substrate the
// dispatch/provision domains route records to. It is the only path that creates
// an audit record. The default production wiring is the local SQLite store
// (sqlite.go); a workspace-sandbox-backed implementation is the documented
// alternative behind this same seam.
//
// Append must be best-effort-resilient from the caller's perspective: a sink
// failure is logged by the caller but must not, on its own, undo the operation
// being audited (the operation's own success path decides that). The sink
// itself, however, must never silently drop a record without returning an error.
type Sink interface {
	// Append writes one immutable record. The implementation populates ID (when
	// empty) and RecordedAt. Returns ErrInvalidRecord when actor/node are empty.
	Append(ctx context.Context, r Record) (Record, error)
}

// Reader is the READ seam: the owner-gated query side the AuditService handler
// depends on.
type Reader interface {
	// List returns records newest-first by RecordedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]Record, error)
}

// Store is the union both production wirings satisfy (the SQLite store
// implements both). Callers depend on the narrow Sink or Reader, never Store
// directly, so a write-only or read-only substitute is always possible.
type Store interface {
	Sink
	Reader
}
