package facts

import (
	"context"
	"database/sql"

	internalfacts "code-facts/internal/facts"
)

func Schema() string {
	return internalfacts.CacheSchema()
}

func DefaultCacheMaxBytes() int64 {
	return internalfacts.DefaultCacheMaxBytes
}

func MigrateSchema(ctx context.Context, db *sql.DB) error {
	return internalfacts.MigrateCacheSchema(ctx, db)
}

func SweepCache(ctx context.Context, db *sql.DB, maxBytes int64) (internalfacts.CacheSweepResult, error) {
	return internalfacts.NewSQLiteCacheRepository(db, maxBytes).Sweep(ctx)
}

func CacheMetrics(ctx context.Context, db *sql.DB, maxBytes int64) (map[string]any, error) {
	stats, err := internalfacts.NewSQLiteCacheRepository(db, maxBytes).Stats(ctx)
	if err != nil {
		return nil, err
	}
	utilization := float64(0)
	if stats.BudgetBytes > 0 {
		utilization = float64(stats.TotalPayloadBytes) / float64(stats.BudgetBytes)
	}
	return map[string]any{
		"cache_total_rows":          stats.TotalRows,
		"cache_total_payload_bytes": stats.TotalPayloadBytes,
		"cache_budget_bytes":        stats.BudgetBytes,
		"cache_utilization":         utilization,
		"cache_last_sweep_at_unix":  stats.LastSweepAtUnix,
	}, nil
}
