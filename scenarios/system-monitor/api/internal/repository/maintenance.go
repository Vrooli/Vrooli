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

	// PruneCollectorMetricsBefore deletes one collector's metric rows older
	// than cutoff. It serves per-collector retention windows that are shorter
	// than the global metrics window (e.g. large device-graph snapshots).
	PruneCollectorMetricsBefore(ctx context.Context, collector string, cutoff time.Time) (RetentionResult, error)

	// ReleaseFreePages returns free pages to the filesystem without a full
	// rewrite (SQLite incremental vacuum) in batches of at most batchPages,
	// releasing the write lock between batches, and reports the bytes
	// released. It is a no-op on a file still in auto_vacuum=NONE mode.
	// Backends without page-level storage return ErrNotSupported.
	ReleaseFreePages(ctx context.Context, batchPages int64) (int64, error)

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
	// AutoVacuum is SQLite's mode: 0 NONE, 1 FULL, 2 INCREMENTAL.
	AutoVacuum int64
	// Path is the database file, for free-space checks; empty in memory.
	Path string
	// WALBytes is the size of the -wal sidecar.
	WALBytes int64
}

// AutoVacuumIncremental is the PRAGMA auto_vacuum value of incremental mode.
const AutoVacuumIncremental = 2

// CompactionResult describes the footprint before and after compaction.
type CompactionResult struct {
	StatsBefore    DatabaseStats
	StatsAfter     DatabaseStats
	ReclaimedBytes int64
}
