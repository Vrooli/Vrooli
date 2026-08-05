# vrooli-autoheal Storage Audit

## Last Updated
2026-05-07

## Storage Posture
- Primary and only runtime persistence backend: SQLite (file-backed, scenario-scoped)
- No Postgres compatibility, fallback, or migration path in runtime startup

## Runtime Backend
- Fixed backend: SQLite
- Connection uses `modernc.org/sqlite` via `api-core/database`

## SQLite Path Resolution
- If `SQLITE_PATH` or `SQLITE_DB` is set, that path is used directly.
- Otherwise path is resolved with `api-core/storage`:
  - profile: `auto`
  - app: `vrooli`
  - scenario: `vrooli-autoheal`
  - file: `autoheal.sqlite` under scenario `data` class

## Schema Layout
- Active schema file: `api/internal/persistence/schema.sql`
- API startup initializes SQLite schema idempotently on boot.

## Notes
- This is a greenfield storage posture: sqlite-only, no legacy cutover scaffolding.

## Query-shape invariants (read path)
- **Never wrap `created_at` in `datetime(...)` in WHERE/ORDER BY.** `created_at` is stored as `time.RFC3339Nano` UTC (with trailing `Z`) and is therefore lexicographically sortable and comparable. Wrapping it in `datetime(...)` makes the column expression opaque to the SQLite planner, which forces full-table scans and defeats the indexes `idx_health_results_check_id_created` and `idx_health_results_created_at`.
- Compute time-window cutoffs in Go (`rfc3339NanoCutoff`) and pass them as bound parameters; do not call `datetime('now', ?)` inside SQL.
- `internal/persistence/store_sqlite_planner_test.go` enforces this with `EXPLAIN QUERY PLAN` assertions and a source-text scan (`TestNoDatetimeWrappersInQueries`). New SQL must keep these green.

## Aggregation invariants
- `GetUptimeHistory` is a single GROUP BY over `health_results` (bucketed via `strftime('%s', created_at)`). The previous implementation issued `bucketCount × distinctCheckIDs` prepared-statement round-trips; do not reintroduce that shape — the planner regression test plus the `BenchmarkGetUptimeHistory_100k` / `BenchmarkGetRecentResults_100k` benchmarks document the expected order of magnitude.

## Layout posture (non-conforming, deferred)
- The persistence package is a single centralized `internal/persistence/` package rather than the per-domain layout the `storage-steer` skill recommends. Refactor is deferred; tracked as a follow-up — out of scope for the 2026-05-07 history-load fix.
