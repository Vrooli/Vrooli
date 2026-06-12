# Data

## Purpose Of This Document

Describe Code Facts data ownership, storage, and retention before implementation.

## Storage Overview

Phase 5 defines contracts only. Later phases may use SQLite or filesystem-backed storage for cache indexes and report metadata. Provider graph data can be recomputed and should not be treated as durable user data.

## Data Ownership

| Data | Owner Domain | Durable? | Notes |
|---|---|---:|---|
| Target context | target | no | Recomputed per request. |
| Surface inventory | surface | no | Recomputed from metadata. |
| Normalized facts | facts | cacheable | Derived from provider graphs. |
| Proof evidence | proof | cacheable | Derived from normalized facts and metadata. |
| Cache metadata | cache | yes | Stores keys, hashes, timestamps, hit/miss/stale reasons. |

## Schema Map

Planned proto packages live under `packages/proto/schemas/code-facts/v1/` and should be split by domain when Phase 6 lands.

## Migrations And Compatibility

Greenfield contracts should avoid compatibility aliases. Cache schemas may use additive migrations after storage lands.

## Import / Export

Planned exports are JSON/proto-shaped describe reports and compact baseline summaries. Imports are target paths and options, not uploaded source archives.

## Retention And Deletion

Cache entries are derived data and may be cleared per target or globally. Durable retention policy is deferred until Phase 9 chooses storage.

## Privacy Notes

Reports may include file paths, symbols, and snippets/ranges. Do not include source file contents beyond required evidence ranges unless a caller explicitly requests it in a future contract.

## Cross-References

- [DOMAINS.md](DOMAINS.md)
- [../reference/cache.md](../reference/cache.md)
