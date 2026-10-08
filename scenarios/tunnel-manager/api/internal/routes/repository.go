package routes

import "context"

// Repository is the persistence seam the routes service depends on.
// Production wires the sqlite-backed implementation from sqlite.go;
// service unit tests wire a fake. Keep the surface narrow — new methods
// land here when the service proves it needs them.
type Repository interface {
	// Create persists r. The implementation populates ID, CreatedAt, and
	// UpdatedAt when zero-valued. Returns ErrRouteConflict when the
	// subdomain is already taken.
	Create(ctx context.Context, r Route) (Route, error)

	// Get returns the route with the given ID or ErrRouteNotFound{ID}.
	Get(ctx context.Context, id string) (Route, error)

	// GetBySubdomain returns the route for a subdomain or
	// ErrRouteNotFound when none matches. Used to enforce the
	// one-route-per-subdomain invariant and by other domains reconciling
	// against the manifest.
	GetBySubdomain(ctx context.Context, subdomain string) (Route, error)

	// List returns all routes ordered by subdomain. A non-empty tier
	// filters to that tier only.
	List(ctx context.Context, tier Tier) ([]Route, error)

	// Update persists the supplied route by ID, refreshing UpdatedAt.
	// Returns ErrRouteNotFound when the ID does not exist.
	Update(ctx context.Context, r Route) (Route, error)

	// Delete removes the route by ID. Returns true when a row was
	// removed, false when the ID did not exist.
	Delete(ctx context.Context, id string) (bool, error)
}
