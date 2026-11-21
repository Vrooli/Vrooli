# Integrations


Track provider, resource, and consumer relationships for Code Facts.

## Dependency Inventory

| Dependency | Kind | Phase | Purpose |
|---|---|---|---|
| api-core/gograph | shared package | active | Go module loading, extraction profiles, deterministic graph facts, hashing, and common graph projection used in-process. |
| typescript-code-graph | scenario provider | Phase 8 | TypeScript project graph and usage facts. |
| proto-health | consumer | Phase 12 | Consumes proto adoption and endpoint proof facts. |
| descriptorimage | shared package | planned | CF-P0-019 remains in progress; Code Facts does not yet compose descriptor-backed contract indexing. |
| ai-go/search | shared package | active | Provides weighted admission, optional semantic storage profiles, reciprocal-rank fusion, and shared Search Hub contracts. |
| search-hub | scenario consumer | active | Registers and evaluates scoped `code` and `contracts` local-index leaves and reads generation/freshness status. |

## Vrooli Resources

SQLite stores the source catalog, generations, derived projections, jobs, and cache metadata. Exact catalog operations do not require Ollama, Qdrant, or a reranker.

Qdrant stores selective code-card vectors under generation-specific collections and a serving alias. Promotion moves the alias before the SQLite active pointer; query-time generation and source-hash fences prevent a transient alias mismatch from returning stale evidence. Ollama supplies the governed embedding model. Both are optional for exact lexical service.

## Scenario Dependencies

The Go provider calls API Core `gograph` in-process. TypeScript and JavaScript providers use Connect-RPC clients behind the provider interfaces in `api/internal/facts`. Unit tests use local modules or fakes and must not require live graph services.

## Third-Party Services

None.

## Failure Modes

- Provider unavailable: return `unsupported` or `unknown` evidence with provider diagnostic.
- Ambiguous target: return target-resolution error and no provider call.
- Stale cache: return stale reason and recompute unless caller disables recomputation.
- Catalog build failure: retain the active generation and mark the shadow generation failed.
- Vector alias promotion failure: leave SQLite on the former active generation and mark the durable promotion failed.
- SQLite activation failure after alias promotion: restore the former alias and mark the durable promotion rolled back.
- Watcher event loss: the five-minute manifest audit supplies the missing change.
- Process restart: mark running jobs interrupted and preserve their durable cursor and error evidence.

Descriptor reload and proto provenance behavior belong to planned CF-P0-019; no Code Facts runtime path currently claims those guarantees.

## Cross-References

- [../reference/configuration.md](../reference/configuration.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
