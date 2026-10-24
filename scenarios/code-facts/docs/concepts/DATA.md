# Data

## Purpose Of This Document

Describe Code Facts data ownership, storage, and retention.

## Storage Overview

Code Facts owns a transactional source catalog, immutable index generations, derived evidence, and cache metadata. Source files and protobuf schemas remain authoritative. Catalog, vector, graph, and cache records are derived and can be rebuilt.

## Data Ownership

| Data | Owner Domain | Durable? | Notes |
|---|---|---:|---|
| Target context | targets | no | Recomputed per request. |
| Source identity and role | catalog | yes | One normalized row per file and generation. |
| Generation state | catalog | yes | Shadow, active, retired, or failed. Only one generation can be active. |
| Descriptor contracts | proof | derived | Resolved from `image.binpb` and joined to `.proto` provenance. |
| Search documents and cards | catalog | derived | Refer to catalog source identities and generations. |
| Graph projections | analysis | derived | Refer to catalog source identities and generations. |
| Surface inventory | surface | no | Recomputed from metadata. |
| Normalized facts | facts | cacheable | Derived from provider graphs. |
| Proof evidence | proof | cacheable | Derived from normalized facts and metadata. |
| Cache metadata | cache | yes | Stores keys, hashes, timestamps, hit/miss/stale reasons. |

## Schema Map

Proto packages live under `packages/proto/schemas/code-facts/v1/`.

The catalog owns these SQLite tables:

| Table | Purpose |
|---|---|
| `code_facts_generations` | Generation policy, source digest, descriptor digest, state, timestamps, and failure evidence. |
| `code_facts_source_files` | Stable file identity, path, language, role, scope, authority, owner, hash, size, and search eligibility. |
| `code_facts_documents` | Normalized searchable or evidentiary projections with mutable source ranges. |
| `code_facts_search_documents` | External FTS content, filters, stable identity, freshness metadata, and mutable ranges. |
| `code_facts_search_fts` | Rebuildable FTS5 term index over titles, exact and split identifiers, paths, bodies, contracts, and aliases. |
| `code_facts_cards` | Selective semantic-card eligibility and embedding recipe metadata. |
| `code_facts_graph_facts` | Generation-fenced relationships and proof status. |
| `code_facts_index_jobs` | Bounded reconcile and reindex progress. |

The existing cache uses `code_facts_cache_entries`. It is not the source catalog and its row count is not an indexed-document count.

File IDs derive from canonical repository paths. Symbol IDs derive from language-qualified semantic names. Contract IDs derive from resolved protobuf names. Content-anchor IDs derive from owner plus a normalized declaration signature. Mutable line ranges are metadata and never identity inputs.

The committed descriptor image is the resolved protobuf authority. `.proto` files are the comment, snippet, source-hash, and source-range authority. Missing `SourceCodeInfo` produces `range_missing`; the system does not invent a line number. Compact generated aliases point back to authoritative contract IDs and do not index generated client bodies.

## Migrations And Compatibility

Catalog migrations apply the schema and migration ledger entry in one SQLite transaction. Restart repeats are idempotent. A failed statement rolls back its tables and version marker. A shadow generation stays isolated until validation and activation complete. Activation retires the former generation and promotes the shadow generation in one transaction.

Search writes update normalized external-content rows. SQLite triggers keep the FTS5 table synchronized in the same transaction. Serving queries join every candidate to the active catalog generation and current source hash. This freshness fence rejects stale or deleted rows before ranking. Exact identifier and path requests use indexed relational fast paths; natural-language requests use weighted BM25. Neither path opens repository source files.

Cache schemas use guarded additive SQLite migrations for retained columns. The startup sweep removes stale cache-schema rows because cache payloads are disposable derived data.

## Import / Export

Planned exports are JSON/proto-shaped describe reports and compact baseline summaries. Imports are target paths and options, not uploaded source archives.

## Retention And Deletion

Cache entries are derived data and may be cleared per target, previewed with dry-run support, or cleared fleet-wide with `code-facts cache clear --all`.

Retention is automatic:

- Each write computes a logical identity from scope, target root, analyzer, family key, and unit identity. Older rows for the same logical identity are deleted in the same write transaction.
- SQLite payloads are gzip-compressed and counted by compressed `payload_bytes`.
- `CODE_FACTS_CACHE_MAX_BYTES` sets the total cache budget. The default is 2 GiB; `0` disables eviction for tests only.
- Writes and startup sweeps evict least-recently-used rows until the table is under budget. Startup sweep also removes rows from stale cache schema versions.
- Cache status and health metrics expose total rows, total payload bytes, budget bytes, utilization, per-scope row/byte counts, and the last startup/write sweep timestamp.

## Privacy Notes

Reports may include file paths, symbols, and snippets/ranges. Do not include source file contents beyond required evidence ranges unless a caller explicitly requests it in a future contract.

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [../reference/corpus-policy.md](../reference/corpus-policy.md)
- [../reference/cache.md](../reference/cache.md)
