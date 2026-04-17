# browser-automation-studio Storage Architecture Audit

## Last Updated
2026-04-16 — written immediately after the SQLite-only migration.

## Summary
- **Single backend**: SQLite via `modernc.org/sqlite` (pure Go, CGO-free). Postgres support was removed in this audit cycle.
- **Path policy**: Database file and runtime artifact roots both flow through `github.com/vrooli/api-core/storage` (`ProfileAuto`).
- **Schema**: One canonical file at `initialization/storage/sqlite/schema.sql`, applied idempotently at startup.
- **Greenfield posture**: No legacy migration shims. Anyone with an existing Postgres install starts fresh.

## Resource Configuration Status
- [x] Storage backend declared explicitly. **Note**: SQLite is embedded — no resource entry in `service.json` (per `cross-platform-readiness/SKILL.md` §3.5, embedded SQLite is documented in `notes`/`environment` rather than `dependencies.resources`).
- [x] Schema initialization referenced via setup `condition.checks[].path` (`initialization/storage/sqlite`).
- [n/a] No Redis (no caching/session backend). Service uses an in-memory `sync.RWMutex` + `map` cache in `services/credits/service.go:60` for credit aggregation.
- [n/a] No Qdrant.
- [x] MinIO declared in `service.json` `dependencies.resources` for screenshot/artifact object storage.

## Connection Pattern Status
- [x] Environment-driven configuration. `BAS_SQLITE_PATH` overrides; `DATABASE_URL=file:/abs/path.db` honored as a secondary override; otherwise canonical resolver path is used.
- [x] No hard-coded credentials. SQLite has no auth surface; the only path source is env or resolver.
- [x] Connection retry with exponential backoff + jitter (`api/database/connection.go:43-86`, configured via `BAS_DB_BASE_RETRY_DELAY_MS`/`BAS_DB_MAX_RETRY_DELAY_MS`/`BAS_DB_RETRY_JITTER_FACTOR`).
- [x] Pool fixed at 1 open connection (`SetMaxOpenConns(1)` at `api/database/connection.go:84`) — required for SQLite single-writer semantics. `ConnMaxLifetime` honored from config.
- [x] Health check exposed as `(*DB).HealthCheck()` (`api/database/connection.go:130`) using `PingContext` with `DatabasePingTimeout`.
- [x] WAL mode + tuned pragmas applied in DSN (`busy_timeout=10s`, `journal_mode=WAL`, `cache_size=-2000`, `mmap_size=256MB`, `synchronous=NORMAL`, `temp_store=MEMORY`, `foreign_keys=ON`).

## Schema Status
- [x] `initialization/storage/sqlite/schema.sql` exists and is fully idempotent (`IF NOT EXISTS` on every table, index, and trigger).
- [x] Tables use proper FK constraints with `ON DELETE CASCADE` where appropriate. UNIQUE constraints exist where upserts target them.
- [x] Greenfield default applied — no `applyIndexSchemaMigrations` or `migrateWorkflowUniqueConstraint*` shim code; clean schema only.
- [x] UX metrics tables (`ux_interaction_traces`, `ux_cursor_paths`, `ux_execution_metrics`) added in this audit cycle. Previously the repository code referenced these tables but no schema defined them — silent failure pre-fix.
- [x] No sql migrations directory. Schema is consolidated.

## Abstraction Status
- [x] `database.Repository` interface (`api/database/repository.go`) abstracts storage from business logic.
- [x] `services/uxmetrics/repository.Repository` interface (`api/services/uxmetrics/interfaces.go`) — has `MockRepository` (mock.go) and `Repository` (sqlite.go) implementations.
- [x] Credits service is the one place where business logic talks to `*sql.DB` directly (`api/services/credits/service.go`). This is intentional — JSONB merge / read-modify-write semantics for `credit_usage` are not generic enough to belong behind a generic repo interface. See `docs/SEAMS.md` for the boundary rationale.
- [x] No raw SQL in handler layer (verified by grep over `api/handlers/`).

## Filesystem Status
- [x] DB file path resolved via `storage.NewResolver` (`api/database/connection.go:137`).
- [x] Recording / replay artifact root resolved via `storage.NewResolver` (`api/internal/paths/recordings.go:126`, `resolveScenarioStoragePath`).
- [x] `paths.ResolveRecordingsRoot` and `paths.ResolveSessionProfilesRoot` are the canonical resolvers consumed by `wire.BuildDependencies` and `handlers.InitDefaultDeps`. No scenario-local `./data` writes in production code paths.
- [x] `FileWriter` (`api/automation/execution-writer/file_writer.go`) writes execution results under the resolver-provided `dataDir`, not a scenario-relative path.
- [x] Atomic-style writes used for the DB (SQLite's WAL handles atomicity); execution result JSON files use a single `os.WriteFile` per file (acceptable for crash semantics — partial files are detected on next read).
- [n/a] `storage.WriteFileAtomic` is not currently used. Most writes are JSON result files where a torn write is just a corrupt file the next read can discard. If we add settings/state files, switch to `WriteFileAtomic`.

## Issues Found
1. **None blocking.** The SQLite-only migration consolidated all known divergence.
2. **`MaxIdleConns` and `BAS_DB_MAX_*_CONNS` are documented in `docs/CONTROL_SURFACE.md` but unused** — only `MaxOpenConns(1)` and `ConnMaxLifetime` are applied. Already updated `CONTROL_SURFACE.md` to reflect SQLite single-connection; the env vars themselves are no longer wired in `connection.go`. If anyone sets them they're silently ignored. Consider removing the parsing in `config/config.go` or warn-on-set.
3. **Per-month `credit_usage` upsert is read-modify-write under a transaction** (`upsertUsage` in `services/credits/service.go`). Correct under SQLite's default isolation, but in a multi-process deployment two writers could both observe an old row before either inserts. SQLite's single-writer model + `MaxOpenConns(1)` makes this safe within one process — call out the assumption if BAS ever splits the API into multiple replicas (which would break SQLite anyway, see Portability Audit).

## Priority Fixes
1. **Low — clean up unused DB pool envs.** Drop `BAS_DB_MAX_OPEN_CONNS`/`BAS_DB_MAX_IDLE_CONNS` parsing from `config/config.go` so it can't surprise an operator who tries to tune them. Audit took this lower priority because the documentation already reflects reality.
2. **Low — consider `WriteFileAtomic`** for execution result + timeline JSON writes in `automation/execution-writer/file_writer.go`. Today a kill at the wrong moment yields a half-written JSON file; the reader handles this defensively, but atomic temp-rename is the safer convention.

## Areas Not Yet Audited
- Backup / restore strategy for the SQLite file (operational concern outside this audit scope).
- Performance of `JSON_*` calls inside SQLite vs. doing JSON parsing in Go for the credits aggregation. Both work; performance has not been measured.
