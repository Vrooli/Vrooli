# Workspace Sandbox Storage Architecture Audit

## Last Updated
2026-04-30 (added diff-archive hybrid storage section)

## Resource Configuration Status
- [x] No shared database resource declared in `.vrooli/service.json`
  (`dependencies.resources` is `{}`). Storage is fully embedded.
- [x] `notes.storage_strategy` documents the embedded SQLite choice.
- [x] No legacy schema name field (SQLite has no schemas).
- [x] No references to a shared resource. The canonical
  runtime schema is embedded from `api/internal/repository/schema.sql`
  via `//go:embed` (see `api/internal/repository/schema.go`). The stale
  `api/internal/<domain>/schema.sql` duplicate was removed so there is
  no second schema source to drift.

## Connection Pattern Status
- [x] Driver name is `database.DriverSQLite` (`modernc.org/sqlite`).
- [x] No hard-coded connection strings; the SQLite file path is resolved
  through the service-owned `api-core/storage` contract. Application-level
  database-path overrides are rejected.
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
- [x] One canonical runtime schema file (`api/internal/repository/schema.sql`); no
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
- [x] Driver overlay roots remain governed by `cfg.Driver.BaseDir`, which is
  selected by the platform path policy in `internal/config` and validated
  before startup. No desktop-specific directory variable or alternate-path
  fallback participates in selection.

## Diff-archive storage (hybrid DB + filesystem)
The diff-archive subsystem (Phases 1–4 of the diff-archive plan) is the
canonical example of the storage-steer §9.4 hybrid pattern in this
scenario: small, queryable metadata in SQLite next to content-addressed
blobs on disk. See `ARCHIVE_DESIGN.md` for the atomicity policy.

### Metadata in SQLite
- [x] One row per archived sandbox in `sandbox_diff_archives`
  (`api/internal/repository/schema.sql`). `sandbox_id` is the primary
  key with `ON DELETE CASCADE` from the `sandboxes` row, so retention
  of the parent sandbox cascades cleanly.
- [x] Inserted transactionally with the sandbox status flip via
  `ArchiveRepository.Insert(ctx, tx, …)` — the same `*sql.Tx` that
  carries the `UPDATE sandboxes SET status = …` write. Snapshot failure
  aborts the terminal transition; we never persist a half-archived row.
- [x] Denormalized `project_root`, `owner`, `agent_manager_run_id`, and
  `total_blob_bytes` so the retention reconciler and the History tab
  can filter / sort / sum without joining other tables.
- [x] CHECK constraints on `archive_state` (`complete` | `not_captured`)
  and `sandbox_status` (`approved` | `rejected` | `deleted`) make
  out-of-taxonomy values impossible at the storage layer.
- [x] Indexes on `snapshot_at`, `project_root`,
  `agent_manager_run_id`, `sandbox_status`, and `owner` cover the
  History listing, retention sweep, and run-id lookup queries.
- [x] Schema migrated forward via the registered-step walker added in
  Phase 1 (`schema.go ExpectedSchemaVersion = 2`); no manual
  brownfield migrations needed.

### Content-addressed blobs on disk
- [x] Per-file blobs and the unified-diff blob stored under
  `archives/<sandbox_id>/<sha256>.gz` via `api-core/storage`
  `ClassData` resolver. The class is **`ClassData`** (durable;
  retention measured in months) — never `ClassCache` or `ClassState`.
- [x] Filenames are content-hashes (SHA-256 of the uncompressed input);
  identical content within a sandbox dedupes naturally.
- [x] All writes go through `storage.WriteFileAtomic` so a torn write
  never leaves a half-blob visible to readers. Hash mismatch on Get
  raises a typed error so silent corruption surfaces loudly.
- [x] Blob writes happen *before* the SQL transaction commits; on
  rollback the partial-write set is best-effort cleaned via
  `BlobStore.DeleteSandbox` so the next snapshot starts from a clean
  directory.
- [x] Retention deletes blobs first, then the row; a half-deleted
  archive is impossible. Blob-delete failures leave the row in place
  and are reported via `ArchiveRetentionReport.BlobFailures` so the
  next reconciler pass can retry.

### Boundary of responsibility
- [x] `applied_changes` (the agent-manager-owned audit table) is
  **decoupled** from the archive table — no foreign keys between the
  two. They have different audiences and different retention horizons.
  Retention sweeps on `sandbox_diff_archives` never touch
  `applied_changes`.
- [x] Live-vs-archive endpoint resolution is a single seam at
  `handlers/diff.go GetDiff`: status drives whether `Service.GetDiff`
  (live overlay) or `Service.GetArchive` (durable snapshot) is called.
  Both return `*types.DiffResult`; the only wire-visible difference is
  the `archiveState` field. agent-manager and the UI are oblivious to
  which path served their response.

### Retention configuration
- [x] `RetentionConfig` (age / total-size / per-project caps) lives in
  a `FileRetentionStore` backed by `api-core/storage` `ClassConfig`
  (atomic tmp+rename writes). Loaded at boot from
  `WORKSPACE_SANDBOX_RETENTION_*` env vars; mutable at runtime via
  `PUT /api/v1/config/retention`. The reconciler reads through a
  `RetentionPolicyProvider` closure on every tick, so PUTs take effect
  on the next pass without service restart.

## Issues Found
- None. The diff-archive landing replaces zero pre-existing patterns —
  it is greenfield work that follows the storage-steer hybrid recipe
  end-to-end. The original cutover findings stand: no shared resource,
  no hard-coded credentials, no schema name collision risk, no direct
  SQL in handlers, and no bypass of `api-core/storage`.

## Priority Fixes
- None outstanding.
