package graph

// Per-run in-memory graph cache (planned, Phase 5).
//
// service.go's cross-run cache is the Repository's FindByHash lookup. A
// per-run in-memory cache keyed by (scenario, contentHash) is a planned
// optimisation to avoid re-extracting a graph already produced this run.
// The key type lands here alongside its production wiring in Phase 5;
// it is intentionally omitted until then so it is not dead code.
