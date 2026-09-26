package health

import (
	"context"
	"fmt"
	"time"
)

// DefaultRetention is the rolling window after which audit rows are
// eligible for eviction (90 days).
const DefaultRetention = 90 * 24 * time.Hour

// EvictBefore deletes audit rows older than the supplied cutoff from both
// audit tables. Returns the total number of rows removed.
//
// Eviction is a separate concern from observation; callers (typically a
// background ticker) drive it. The store itself never silently drops
// rows during reads.
func (s *Store) EvictBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	stamp := cutoff.UTC().Format(time.RFC3339Nano)
	var total int64

	res, err := s.db.ExecContext(ctx, `DELETE FROM model_health_audit WHERE timestamp < ?`, stamp)
	if err != nil {
		return total, fmt.Errorf("health: evict model audit: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		total += n
	}

	res, err = s.db.ExecContext(ctx, `DELETE FROM runner_health_audit WHERE timestamp < ?`, stamp)
	if err != nil {
		return total, fmt.Errorf("health: evict runner audit: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil {
		total += n
	}
	return total, nil
}

// EvictByRetention is a convenience wrapper for `EvictBefore(now-retention)`.
// retention<=0 disables eviction (returns 0, nil).
func (s *Store) EvictByRetention(ctx context.Context, retention time.Duration) (int64, error) {
	if retention <= 0 {
		return 0, nil
	}
	return s.EvictBefore(ctx, time.Now().Add(-retention))
}
