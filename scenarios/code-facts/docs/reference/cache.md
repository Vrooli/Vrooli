# Cache

Code Facts caches derived graph and report payloads. Source code is never owned by Code Facts.

Cache keys include:

- Code Facts analyzer/schema version.
- Provider analyzer version.
- Canonical target root.
- Parse unit id, root, and config path for graph entries.
- Request options and requested fact families for report entries.
- Config hashes such as `go.mod`, `go.sum`, `tsconfig.json`, `package.json`, and lockfiles.
- Source content fingerprints for bounded Go/TypeScript/JavaScript parse-unit files. Fingerprints are memoized by path, size, and nanosecond mtime, so mtime-only churn keeps the same cache key while content edits invalidate it.
- Provider graph hash when a graph result is available.

Cache scopes:

- `graph`: parse unit plus provider options to provider graph payload.
- `report`: selected fact families plus target/options to `CodeFactsReport`.

Callers can pass `use_cache=false` on describe/proof/surface requests to bypass lookup and force fresh extraction. Fresh results refresh cache entries with the latest source/config hash evidence.

## Retention

The cache is bounded by logical identity and bytes:

- `logical_key`: scope, target root, Code Facts analyzer, fact-family key, and unit identity. New writes delete older rows with the same logical key before storing the new row.
- `payload_bytes`: compressed payload size for SQLite rows. This is the value used for byte-budget enforcement.
- `codec`: SQLite payloads are stored with `codec=gzip`; memory cache entries remain plain strings because compression is a disk concern.
- `CODE_FACTS_CACHE_MAX_BYTES`: total cache byte budget. Default is 2 GiB. `0` means unlimited and is intended for tests.
- Eviction: writes and startup sweeps remove least-recently-used rows until total payload bytes fit the budget. The row being written is protected from eviction.
- Startup sweep: removes rows whose cache schema version is stale, then enforces the byte budget.

## Observability

`code-facts cache status <target>` and `code-facts cache inspect <target>` report target entries plus whole-cache totals:

- `entries`, `entries_metadata`: matching rows for the requested target/key.
- `total_rows`, `total_payload_bytes`: all retained rows and bytes.
- `budget_bytes`, `utilization`: configured budget and current use.
- `scopes`: per-scope row and byte counts.
- `last_sweep_at_unix`: last cache sweep observed by the process.

The `/health` and `/api/v1/health` payloads include cache metrics in the `metrics` block: `cache_total_rows`, `cache_total_payload_bytes`, `cache_budget_bytes`, `cache_utilization`, and `cache_last_sweep_at_unix`.

Cache metadata exposes key, logical key, scope, state (`hit`, `miss`, `bypassed`, or `stored`), reason, analyzer version, provider version, schema version, source hash, config hash, graph hash, payload bytes, codec, age, and hit count.
Status and inspection queries read only those metadata columns. They never load
the compressed graph/report bodies, so diagnostics memory is independent of
the retained payload volume.

## Clearing

`code-facts cache clear <target> --dry-run` reports matches without deleting derived entries. `code-facts cache clear <target>` clears entries for one target. `code-facts cache clear --all --dry-run` previews all rows, and `code-facts cache clear --all` deletes all cache entries.
