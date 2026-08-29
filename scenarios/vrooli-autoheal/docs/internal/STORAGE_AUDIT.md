# vrooli-autoheal Storage Audit

## Last Updated
2026-08-29

## Storage Posture
- Primary and only runtime persistence backend: SQLite (file-backed, scenario-scoped)
- No Postgres compatibility, fallback, or migration path in runtime startup

## Runtime Backend
- Fixed backend: SQLite
- Connection uses `modernc.org/sqlite` via `api-core/database`

## SQLite Path Resolution
- No database-path environment variable is read. The path is resolved by `api-core/storage` from the scenario id.
- Otherwise path is resolved with `api-core/storage`:
  - profile: `auto`
  - app: `vrooli`
  - scenario: `vrooli-autoheal`
  - file: `autoheal.sqlite` under scenario `data` class

## Declared Storage Surface
- `autoheal.sqlite` is the only active, non-regenerable database and resolves to `~/.vrooli/data/vrooli/vrooli-autoheal/autoheal.sqlite` on this host.
- `autoheal.sqlite-wal` and `autoheal.sqlite-shm` are explicitly declared as regenerable SQLite sidecars in the same data class.
- `deployment/deployment-report.json` is declared as a small regenerable lifecycle report in the data class.
- `retention/enforcement-receipt.json` is declared as bounded regenerable state proving the latest framework retention run.
- The database working-set budget is 1 GiB. Scheduled per-table ceilings total
  704 MiB: health results 256 MiB, action logs 64 MiB, and 128 MiB each for
  system events, host inventory snapshots, and incident observations.
- `health_results` and `action_logs` share the named 24-hour operational-history
  window. Enforcement runs on startup and every 15 minutes; the offline
  `vrooli-autoheal retention enforce --compact` command is the explicit
  physical-reclamation path.

## Legacy Data Requiring Governed Reclamation
- `$USER_DATA_DIR/vrooli/vrooli-autoheal/autoheal.sqlite` is a 41,913,749,504-byte inactive database last modified 2026-06-12. It is not regenerable and must not be deleted until the plan's summary export and reader-proof steps complete.
- The scenario-local 9,354,616,832-byte database and its WAL/SHM sidecars were
  removed by three explicit named commands on 2026-08-29 after integrity,
  open-reader, literal-path, and summary-export proof. The retained evidence is
  `path:scenarios/vrooli-autoheal/docs/internal/ORPHANED_DATABASE_SUMMARY_2026-08-29.md`.
- The governed sweep reduced the active database from 3,742,261,248 bytes to
  253,382,656 bytes. It reduced `health_results` from 660,352 to 47,865 rows
  and `action_logs` from 32,716 to 2,969 rows; `PRAGMA integrity_check` returned
  `ok`. After lifecycle restart the database was 253,730,816 bytes plus normal
  WAL/SHM sidecars, and `/health` reported SQLite connected and ready.

## Schema Layout
- Active schema file: `api/internal/persistence/schema.sql`
- API startup initializes SQLite schema idempotently on boot.

## Notes
- This is a greenfield storage posture: sqlite-only, no legacy cutover scaffolding.
- The offline retention CLI filters framework filesystem budgets and refuses a
  foreign ambient storage namespace. This prevents an installed CLI launched
  from another scenario from opening or pruning that scenario's database.

## Query-shape invariants (read path)
- **Never wrap `created_at` in `datetime(...)` in WHERE/ORDER BY.** `created_at` is stored as `time.RFC3339Nano` UTC (with trailing `Z`) and is therefore lexicographically sortable and comparable. Wrapping it in `datetime(...)` makes the column expression opaque to the SQLite planner, which forces full-table scans and defeats the indexes `idx_health_results_check_id_created` and `idx_health_results_created_at`.
- Compute time-window cutoffs in Go (`rfc3339NanoCutoff`) and pass them as bound parameters; do not call `datetime('now', ?)` inside SQL.
- `internal/persistence/store_sqlite_planner_test.go` enforces this with `EXPLAIN QUERY PLAN` assertions and a source-text scan (`TestNoDatetimeWrappersInQueries`). New SQL must keep these green.

## Aggregation invariants
- `GetUptimeHistory` is a single GROUP BY over `health_results` (bucketed via `strftime('%s', created_at)`). The previous implementation issued `bucketCount × distinctCheckIDs` prepared-statement round-trips; do not reintroduce that shape — the planner regression test plus the `BenchmarkGetUptimeHistory_100k` / `BenchmarkGetRecentResults_100k` benchmarks document the expected order of magnitude.

## Layout posture (non-conforming, deferred)
- The persistence package is a single centralized `internal/persistence/` package rather than the per-domain layout the `storage-steer` skill recommends. Refactor is deferred; tracked as a follow-up — out of scope for the 2026-05-07 history-load fix.
