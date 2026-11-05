# Health-history load latency fix — 2026-05-07

## Symptom
Operators reported the autoheal dashboard's history endpoints (`/checks/{id}/history`, `/timeline`, `/uptime/history`, `/uptime/stats`, `/checks/trends`) loading slowly as the `health_results` table accumulated rows. Latency scaled roughly linearly with row count, and the retention sweep (`cleanupOldResultsSQLite`) suffered the same pathology, so the table grew faster than it pruned — a feedback loop.

## Root cause
Two compounding issues in `internal/persistence/store_sqlite.go`:

1. **Index-defeating `datetime(...)` wrappers.** Every SQLite read query wrapped `created_at` in `datetime(...)`, which is opaque to the planner. The two indexes defined in `api/internal/<domain>/schema.sql` (`idx_health_results_check_id_created`, `idx_health_results_created_at`) could not be used; SQLite did a full table scan + sort on every dashboard call.
2. **N+1 in `getUptimeHistorySQLite`.** The function selected all distinct `check_id`s in the window, then issued `bucketCount × N` prepared-statement round-trips — one `LIMIT 1` lookup per (bucket, checkID) pair. With `MaxOpenConns: 1`, those ran strictly sequentially. Each query was a full-table scan because of (1).

`created_at` is stored as `time.RFC3339Nano` UTC, which is lexicographically sortable, so the wrappers were unnecessary in the first place.

## Fix
- All read queries and the retention sweep compare `created_at` as a string against Go-computed RFC3339Nano cutoffs. EXPLAIN QUERY PLAN now reports `SEARCH … USING INDEX idx_health_results_*` for every read.
- `getUptimeHistorySQLite` is now a single GROUP BY over `(bucket, status)`, computed via `strftime('%s', created_at)`. Empty buckets are pre-filled in Go, so the response shape is unchanged.
- `openSQLiteTestDB` now loads `api/internal/<domain>/schema.sql` directly so test fixtures match production (with indexes). The previous bare schema masked the planner regression.
- New regression tests in `store_sqlite_planner_test.go` assert index use via `EXPLAIN QUERY PLAN`, and a source-text scan rejects future re-introduction of `datetime(created_at)`.

## Semantics note
The chart consumer (`UptimeTrendChart.tsx`) treats `bucket.{ok,warning,critical,total}` as stacked counts and normalizes per render via `maxTotal`. The rewritten query counts events that fell within each bucket window (Option A from the plan) rather than the previous "status snapshot at bucket boundary" semantic. The visual chart shape is preserved; the absolute counts differ.

## Benchmarks (AMD Ryzen 9 7950X, in-memory SQLite, 100,000 health_results rows)

```
BenchmarkGetUptimeHistory_100k-32    518   4,696,022 ns/op    6448 B/op    382 allocs/op
BenchmarkGetRecentResults_100k-32  42516      56,834 ns/op   18624 B/op    484 allocs/op
```

- `GetUptimeHistory`: ~4.7 ms/op at 100k rows in a single query (vs. previous N+1 pattern that scaled with `bucket × distinct_checks`).
- `GetRecentResults`: ~57 µs/op at 100k rows (index-backed `WHERE check_id = ? ORDER BY created_at DESC LIMIT ?`).

## Validation
- `go test ./internal/persistence/... -count=1` — 13 tests pass, including 5 EXPLAIN-plan regression tests and a source-text guard against future `datetime(created_at)` reintroduction.
- `go test ./internal/persistence/ -bench=. -benchmem -benchtime=2s` — see numbers above.

## Out of scope (follow-up candidates)
- Refactor `internal/persistence/` to the per-domain layout recommended by `storage-steer`.
- Reconsider `MaxOpenConns: 1` for read concurrency.
