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

SQLite stores the source catalog, generations, derived projections, jobs,
lexical index, and model-scoped semantic vectors. `.vrooli/service.json`
declares no Vrooli resource dependencies: Code Facts does not use Qdrant or an
Ollama client. Optional embeddings go through AI Gateway's typed
`InferenceService.Embed` contract; if the gateway or role is unavailable,
semantic search degrades or is omitted while lexical search remains available.

## Scenario Dependencies

The following runtime scenario dependencies mirror
`.vrooli/service.json`:

| Scenario | Required? | Startup policy | Purpose | Unavailable behavior |
|---|---|---|---|---|
| `architecture-cartographer` | no | `try_start` | File-domain facts for targets requiring domain partitioning. | Falls back to local source and import evidence. |
| `search-hub` | no | `try_start` | Registers the scoped `code` and `contracts` search leaves. | Code Facts remains directly queryable; registration retries on next lifecycle start. |
| `go-code-graph` | no | `try_start` | Go analyzer backend for Go source graph extraction. | Go graph facts are unavailable for targets needing analyzer execution. |
| `typescript-code-graph` | no | `try_start` | TypeScript analyzer backend for TypeScript source graph extraction. | TypeScript graph facts are unavailable for targets needing analyzer execution. |
| `ai-gateway` | no | `ignore` | Typed embedding provider for semantic code search. | Semantic mode degrades or is omitted; lexical search continues. |

The Go provider calls API Core `gograph` in-process. Language providers use
Connect-RPC clients behind interfaces in `api/internal/facts`. Unit tests use
local modules or fakes and do not require live graph services.

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
