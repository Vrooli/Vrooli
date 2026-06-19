package tunnel

import (
	"context"
	"time"
)

// MetricsRepository is the persistence seam the tunnel service depends on.
// Production wires the sqlite-backed implementation from sqlite.go; service
// unit tests wire a fake. Keep the surface narrow — new methods land here when
// the service proves it needs them.
type MetricsRepository interface {
	// Store persists s. The implementation populates ID and ScrapedAt when
	// zero-valued and returns the stored sample.
	Store(ctx context.Context, s MetricsSample) (MetricsSample, error)

	// Query returns samples scraped within [from, to] (inclusive), ordered by
	// scraped_at ascending. A zero from/to is treated as unbounded on that end.
	Query(ctx context.Context, from, to time.Time) ([]MetricsSample, error)

	// Latest returns the most recently scraped sample, or ErrNoMetrics when the
	// table is empty.
	Latest(ctx context.Context) (MetricsSample, error)
}
