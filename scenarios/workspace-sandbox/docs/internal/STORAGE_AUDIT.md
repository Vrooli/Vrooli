# Workspace Sandbox Storage Architecture Audit

## Last Updated
2026-04-28

## Resource Configuration Status
- [x] No shared database resource declared in `.vrooli/service.json`
  (`dependencies.resources` is `{}`). Storage is fully embedded.
- [x] `notes.storage_strategy` documents the embedded SQLite choice.
- [x] No legacy schema name field (SQLite has no schemas).
- [x] No initialization references to a shared resource. The canonical
  schema lives at `initialization/sqlite/schema.sql` and is also embedded
  in the API binary via `//go:embed` (see `api/internal/repository/schema.go`).

## Connection Pattern Status
- [x] Driver name is `database.DriverSQLite` (`modernc.org/sqlite`).
- [x] No hard-coded connection strings; the SQLite file path is resolved
  via `api-core/storage` and overridable through `SQLITE_PATH`.
- [x] DSN appends pragmas every connection needs: `journal_mode=WAL`,
  `foreign_keys=1`, `busy_timeout=5000`, `synchronous=NORMAL`.
- [x] DSN sets `_txlock=immediate` so `db.BeginTx` acquires the SQLite
  reserved lock at transaction open, providing write-serialization for
  Create + CheckScopeOverlap.
- [x] Pool capped to a single open connection
  (`db.SetMaxOpenConns(1)`) since SQLite serializes writes.
- [x] No retry/backoff loop is needed for a local file-backed engine; the
  `busy_timeout` pragma covers transient lock contention.
- [x] Health check is the existing `/health` endpoint, which probes the
  same `*sql.DB` handle.

## Schema Status
- [x] One canonical schema file (`initialization/sqlite/schema.sql`); no
  `migrations/` directory and no migration-numbering scheme.
- [x] All `CREATE TABLE` and `CREATE INDEX` statements use `IF NOT
  EXISTS` (idempotent on every startup).
- [x] Type mapping is uniform:
  - UUIDs: TEXT (canonical 36-char form), generated in Go via
    `github.com/google/uuid`.
  - Timestamps: TEXT (RFC3339Nano UTC).
  - JSON objects (metadata, behavior, audit details, sandbox_state): TEXT
    containing UTF-8 JSON.
  - String/int arrays (tags, reserved_paths, active_pids): TEXT containing
    a JSON array.
  - Booleans: INTEGER 0/1.
  - Status enums: TEXT + CHECK constraint.
- [x] Greenfield posture: no PL/pgSQL functions, no Postgres extensions,
  no schema-version bookkeeping. The previous PG `check_scope_overlap()`
  and `get_sandbox_stats()` functions were inlined as Go logic in
  `api/internal/repository/sandbox_repo.go`.
- [x] No brownfield migrations needed (greenfield cutover; no users yet).

## Abstraction Status
- [x] `Repository` interface defined in
  `api/internal/repository/sandbox_repo.go`; business logic talks to the
  interface, not directly to `*sql.DB`.
- [x] Both `SandboxRepository` and `TxSandboxRepository` share helper
  functions through a small `dbExec` interface (works for both `*sql.DB`
  and `*sql.Tx`).
- [x] Codec helpers (`api/internal/repository/sqlite_codec.go`) centralize
  SQLite-specific encoding/decoding so query call sites read like
  ordinary Go.

## Filesystem Status
- [x] Sandbox metadata DB resolved through `api-core/storage` (`ClassData`
  + `EnsureClassDir`).
- [x] No mutable runtime state under the scenario deploy directory.
- [x] Driver overlay roots remain governed by `cfg.Driver.BaseDir`
  (defaults to XDG data directory).

## Issues Found
- None. The cutover replaces every red-flag pattern documented in the
  storage-steer skill: no shared resource, no hard-coded credentials, no
  schema name collision risk, no direct SQL in handlers, and no
  bypass of `api-core/storage`.

## Priority Fixes
- None outstanding.
