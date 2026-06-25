package authoring

import "context"

// SessionStore is the persistence seam for authoring sessions. Production wires
// the SQLite implementation (sqlite.go) over the ~/.vrooli home store so a
// session survives across CLI invocations; service unit tests substitute a fake
// or in-memory map. The surface is intentionally narrow — a session round-trips
// as a whole record (its sections live in the document column).
type SessionStore interface {
	// Save upserts a whole session keyed by id (including its sections in the
	// document). The service owns id/timestamp assignment.
	Save(ctx context.Context, s Session) error
	// Get returns the session matching an id; ok=false when absent.
	Get(ctx context.Context, id string) (Session, bool, error)
}
