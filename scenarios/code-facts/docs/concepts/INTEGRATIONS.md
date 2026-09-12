# Integrations

## Purpose Of This Document

Track provider, resource, and consumer relationships for Code Facts.

## Dependency Inventory

| Dependency | Kind | Phase | Purpose |
|---|---|---|---|
| go-code-graph | scenario provider | Phase 8 | Go module graph and usage facts. |
| typescript-code-graph | scenario provider | Phase 8 | TypeScript project graph and usage facts. |
| proto-health | consumer | Phase 12 | Consumes proto adoption and endpoint proof facts. |
| descriptorimage | shared package | active | Loads the committed protobuf descriptor image and preserves last known-good snapshots. |
| ai-go/search | shared package | active | Provides weighted admission, optional semantic storage profiles, reciprocal-rank fusion, and shared Search Hub contracts. |
| search-hub | scenario consumer | active | Registers and evaluates scoped `code` and `contracts` local-index leaves and reads generation/freshness status. |

## Vrooli Resources

SQLite stores the source catalog, generations, derived projections, jobs, and cache metadata. Exact catalog and descriptor operations do not require Ollama, Qdrant, or a reranker.

Qdrant stores selective code-card vectors under generation-specific collections and a serving alias. Promotion moves the alias before the SQLite active pointer; query-time generation and source-hash fences prevent a transient alias mismatch from returning stale evidence. Ollama supplies the governed embedding model. Both are optional for exact lexical service.

## Scenario Dependencies

Provider dependencies are called through Connect-RPC clients hidden behind analyzer seams. Unit tests use fakes and must not require live providers.

## Third-Party Services

None.

## Failure Modes

- Provider unavailable: return `unsupported` or `unknown` evidence with provider diagnostic.
- Ambiguous target: return target-resolution error and no provider call.
- Stale cache: return stale reason and recompute unless caller disables recomputation.
- Descriptor reload failure: serve the previous valid descriptor generation and expose the failure timestamp and message.
- Proto provenance missing: return the resolved declaration with `range_missing`; do not invent a source range.
- Catalog build failure: retain the active generation and mark the shadow generation failed.
- Vector alias promotion failure: leave SQLite on the former active generation and mark the durable promotion failed.
- SQLite activation failure after alias promotion: restore the former alias and mark the durable promotion rolled back.
- Watcher event loss: the five-minute manifest audit supplies the missing change.
- Process restart: mark running jobs interrupted and preserve their durable cursor and error evidence.

## Cross-References

- [../reference/configuration.md](../reference/configuration.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
