// Responsibility: report when a provider's quota last had work that could
// spend it, so a stale forecast window can be idle rather than unknown.
package database

import (
	"context"
	"time"

	"agent-manager/internal/usageattribution"
	"agent-manager/internal/usageforecast"
)

// UsageActivityRepository implements usageforecast.ActivitySource.
type UsageActivityRepository struct{ db *DB }

// NewUsageActivityRepository builds the activity source.
func NewUsageActivityRepository(db *DB) *UsageActivityRepository {
	return &UsageActivityRepository{db: db}
}

var _ usageforecast.ActivitySource = (*UsageActivityRepository)(nil)

// LastProviderActivity is now while an Agent Manager run or an indexed
// session of the provider's runner is live, otherwise the newest indexed
// usage report of that runner (zero when there is none). An unknown provider
// has no attributable runner and reports now, which keeps its windows unknown.
func (r *UsageActivityRepository) LastProviderActivity(ctx context.Context, provider string) (time.Time, error) {
	runner := usageattribution.RunnerFor(provider)
	if runner == "" {
		return time.Now().UTC(), nil
	}
	var live int
	if err := r.db.GetContext(ctx, &live, `SELECT
		(SELECT COUNT(*) FROM runs WHERE status = 'running' AND harness_kind = ?) +
		(SELECT COUNT(*) FROM transcript_files WHERE active = 1 AND harness = ?)`, runner, runner); err != nil {
		return time.Time{}, err
	}
	if live > 0 {
		return time.Now().UTC(), nil
	}
	var newest *string
	if err := r.db.GetContext(ctx, &newest, `SELECT MAX(u.observed_at) FROM transcript_usage u JOIN transcript_files f ON f.file_key = u.file_key WHERE f.harness = ?`, runner); err != nil {
		return time.Time{}, err
	}
	if newest == nil {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, *newest)
}
