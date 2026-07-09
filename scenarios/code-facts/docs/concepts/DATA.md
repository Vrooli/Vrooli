# Data

## Purpose Of This Document

Describe Code Facts data ownership, storage, and retention.

## Storage Overview

Code Facts owns derived evidence and cache metadata. Provider graph data and normalized reports can be recomputed from source and should not be treated as durable user data.

## Data Ownership

| Data | Owner Domain | Durable? | Notes |
|---|---|---:|---|
| Target context | target | no | Recomputed per request. |
| Surface inventory | surface | no | Recomputed from metadata. |
| Normalized facts | facts | cacheable | Derived from provider graphs. |
| Proof evidence | proof | cacheable | Derived from normalized facts and metadata. |
| Cache metadata | cache | yes | Stores keys, hashes, timestamps, hit/miss/stale reasons. |

## Schema Map

Proto packages live under `packages/proto/schemas/code-facts/v1/`. The API stores cache entries in SQLite table `code_facts_cache_entries`, keyed by cache key with scope, target root, logical identity, analyzer/provider versions, source/config hashes, graph hash, compressed serialized payload, payload byte count, timestamps, and hit count.

## Migrations And Compatibility

Greenfield contracts avoid compatibility aliases. Cache schemas use guarded additive SQLite migrations for retained columns, then startup sweep removes rows with stale cache schema versions because cache payloads are disposable derived data.

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
- [../reference/cache.md](../reference/cache.md)
