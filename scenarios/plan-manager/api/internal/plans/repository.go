package plans

import "context"

// Repository is the persistence seam for the plans SSOT. Production wires the
// SQLite implementation (sqlite.go) over the ~/.vrooli home store; service unit
// tests substitute a fake. The surface is intentionally narrow — plans/phases/
// references round-trip as a whole plan; only the supersession graph queries
// across plans (the *Edge methods).
type Repository interface {
	// Save upserts a whole plan keyed by id (including its phases/references in
	// the document). Callers pass a fully-formed Plan; the service owns id/hash/
	// status/timestamp assignment.
	Save(ctx context.Context, p Plan) error
	// Get returns the plan matching an id or slug; ok=false when absent.
	Get(ctx context.Context, idOrSlug string) (Plan, bool, error)
	// List returns plans matching the filter, newest-first by created_at.
	List(ctx context.Context, filter ListFilter) ([]Plan, error)
	// ListEdges returns supersession/dependency edges touching planID (as from or
	// to). An empty planID returns the whole graph.
	ListEdges(ctx context.Context, planID string) ([]PlanEdge, error)
	// SaveEdge inserts a graph edge (idempotent on the (from,to,kind) key).
	SaveEdge(ctx context.Context, e PlanEdge) error
}
