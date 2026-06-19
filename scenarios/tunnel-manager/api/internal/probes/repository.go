package probes

import "context"

// LatestPair holds the most recent internal and external probe for one
// subdomain. Either pointer is nil when no probe of that kind has been
// recorded yet. Returned by LatestPerRoute to feed Service.Classify
// without the service re-implementing the "latest per (subdomain, kind)"
// query.
type LatestPair struct {
	Subdomain string
	Internal  *ProbeResult
	External  *ProbeResult
}

// Repository is the persistence seam the probes service depends on.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire a fake. Keep the surface narrow — new methods
// land here when the service proves it needs them.
type Repository interface {
	// Persist stores r. The implementation populates ID and CreatedAt
	// when zero-valued and returns the stored row.
	Persist(ctx context.Context, r ProbeResult) (ProbeResult, error)

	// List returns probe history newest-first. A non-empty subdomain
	// filters to that route only; limit caps the rows returned (<= 0
	// applies the implementation default).
	List(ctx context.Context, subdomain string, limit int) ([]ProbeResult, error)

	// LatestPerRoute returns, per subdomain, the most recent internal and
	// external probe. Ordered by subdomain so Classify output is stable.
	LatestPerRoute(ctx context.Context) ([]LatestPair, error)
}
