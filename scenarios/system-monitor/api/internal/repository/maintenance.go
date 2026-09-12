package repository

// DOC: docs/internal/SEAMS.md#maintenance-repository

import (
	"context"
	"time"
)

// MaintenanceRepository exposes storage-lifecycle operations for the metrics
// table: estimating and pruning old rows, inspecting the database footprint,
// and compacting reclaimed space. Backends that cannot perform an operation
// return ErrNotSupported.
type MaintenanceRepository interface {
	// EstimateMetricRetention reports the rows that a prune at cutoff would
	// remove, without modifying any data.
	EstimateMetricRetention(ctx context.Context, cutoff time.Time) (RetentionEstimate, error)

	// PruneMetricsBefore deletes metric rows with a timestamp strictly older
	// than cutoff and reports how many were removed.
	PruneMetricsBefore(ctx context.Context, cutoff time.Time) (RetentionResult, error)

	// SQLiteStats reports the current database storage footprint.
	SQLiteStats(ctx context.Context) (DatabaseStats, error)

	// Compact reclaims free space in the database and reports the footprint
	// before and after.
	Compact(ctx context.Context) (CompactionResult, error)
}

// RetentionEstimate describes the rows a retention prune would remove.
type RetentionEstimate struct {
	RowCount       int64
	PayloadBytes   int64
	OldestAffected time.Time
	NewestAffected time.Time
	Cutoff         time.Time
}

// RetentionResult describes the outcome of a retention prune.
type RetentionResult struct {
	DeletedRows int64
	Cutoff      time.Time
}

// DatabaseStats describes the storage footprint of the metrics database.
type DatabaseStats struct {
	PageSize      int64
	PageCount     int64
	FreelistCount int64
	SizeBytes     int64 // PageSize * PageCount (logical size)
	MetricRows    int64
}

// CompactionResult describes the footprint before and after compaction.
type CompactionResult struct {
	StatsBefore    DatabaseStats
	StatsAfter     DatabaseStats
	ReclaimedBytes int64
}
