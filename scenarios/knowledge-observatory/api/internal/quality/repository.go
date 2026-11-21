package quality

import (
	"context"
	"time"
)

// Metric is one quality sample for a collection.
//
// AvgQuality is a generated column: the database computes it from the four
// scores. It is populated on read and ignored on write.
type Metric struct {
	ID             string
	CollectionName string
	Coherence      *float64
	Freshness      *float64
	Redundancy     *float64
	Coverage       *float64
	TotalEntries   int
	AvgQuality     *float64
	MeasuredAt     time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CollectionStat is the current rollup for a collection.
type CollectionStat struct {
	ID                string
	CollectionName    string
	TotalEntries      int
	TotalSearches     int
	AvgSearchScore    *float64
	MostSearchedTerms string // JSON document, empty when unset
	GrowthRate        *float64
	LastUpdated       time.Time
	CreatedAt         time.Time
}

// DashboardRow joins a collection's rollup to its most recent sample. It is the
// shape served by the dashboard_metrics view.
type DashboardRow struct {
	CollectionName string
	TotalEntries   int
	Coherence      *float64
	Freshness      *float64
	Redundancy     *float64
	Coverage       *float64
	AvgQuality     *float64
	TotalSearches  int
	AvgSearchScore *float64
	MeasuredAt     *time.Time
}

// Repository is the quality domain's storage surface. It names no engine, so
// the SQLite implementation and the test fake are interchangeable.
type Repository interface {
	InsertMetric(ctx context.Context, m Metric) (string, error)
	LatestMetric(ctx context.Context, collection string) (Metric, bool, error)
	CountMetrics(ctx context.Context) (int64, error)

	UpsertCollectionStat(ctx context.Context, stat CollectionStat) error
	GetCollectionStat(ctx context.Context, collection string) (CollectionStat, bool, error)

	Dashboard(ctx context.Context) ([]DashboardRow, error)

	// PruneMetricsOlderThan deletes samples measured before cutoff and returns
	// how many rows went away.
	PruneMetricsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)

	// DownsampleMetricsOlderThan collapses samples measured before cutoff to one
	// row per collection per day, keeping the most recent sample of each day and
	// deleting the rest. It returns the number of rows deleted.
	//
	// This is the bounded-growth control for quality_metrics, which otherwise
	// accumulates roughly 10,000 rows a day forever. See
	// docs/internal/STORAGE_AUDIT.md §2.
	DownsampleMetricsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
