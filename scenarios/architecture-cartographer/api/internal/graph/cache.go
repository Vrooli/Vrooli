package graph

// Cache is the in-memory cache key shape used by service.go to avoid
// re-extracting a graph that was already produced this run. The
// Repository's FindByHash lookup is the cross-run cache; this in-memory
// map is the per-run optimisation. Phase 5 adds the production wiring.
type cacheKey struct {
	scenario    string
	contentHash string
}
