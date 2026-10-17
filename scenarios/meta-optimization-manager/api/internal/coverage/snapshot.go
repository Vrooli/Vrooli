package coverage

import (
	"context"
	"time"
)

// SnapshotRepository is the short-TTL cache seam for the computed scoreboard.
// The numerator is never persisted as truth — this is a burst-absorbing cache
// only (a UI poll, a CLI loop). A nil SnapshotRepository disables caching; the
// Service then recomputes live on every call. Production wires the SQLite-backed
// implementation; tests use a fake or nil.
type SnapshotRepository interface {
	// Save records the freshly-computed scoreboard. Best-effort: callers ignore
	// the error (a cache write must never fail a read).
	Save(ctx context.Context, status Status) error
	// Latest returns the most recent snapshot when it is younger than ttl
	// relative to now; otherwise (no rows / too stale) returns ok=false so the
	// caller recomputes.
	Latest(ctx context.Context, ttl time.Duration, now time.Time) (Status, bool)
}
