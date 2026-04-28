# Workspace Sandbox: PostgreSQL → SQLite Greenfield Cutover

## 1. Purpose

Replace the workspace-sandbox PostgreSQL backend with an embedded, pure-Go
SQLite backend (`modernc.org/sqlite`). The result must be a **single-database,
single-driver** scenario with no PostgreSQL code, no compatibility shims, no
migration framework, and no dual-backend abstractions. The scenario has never
been deployed; there are no users to preserve. Do **not** add brownfield
migration code.

## 2. Greenfield Constraint (Hard Rule)

This is a **hard cutover**. The plan is rejected if any of the following remain
in the merged tree:

- Any reference to `lib/pq`, `jackc/pgx`, `POSTGRES_*` env vars, `DATABASE_URL`,
  `POSTGRES_URL`, `pq.Array`, `JSONB`, `TIMESTAMPTZ`, `CREATE TYPE ... AS ENUM`,
  `FOR UPDATE`, `$1/$2/...` placeholders, `ON CONFLICT DO NOTHING` against
  postgres-only constructs, `uuid-ossp`, `pgcrypto`, or PL/pgSQL functions.
- Any "if dialect == postgres" branching, abstract `Dialect` type, or runtime
  driver-selection switch.
- Any `initialization/postgres/` directory or postgres seed/migration SQL.
- Any DatabaseConfig fields for host/port/user/password/sslmode/schema.
- Any "migration_001" / "migration_002" naming or schema-version bookkeeping
  for postgres provenance shims.

The only acceptable backend is SQLite via `modernc.org/sqlite`. Tests, code,
docs, and service.json must all reflect this single reality.

A one-time export of any local development data may be performed manually
before the cutover (out-of-tree shell session); **no migration code** of any
kind enters the repository.

## 3. Required Reading

Run before executing this plan:

```bash
prompt-manager skill read storage-steer cross-platform-readiness implementation-plan-authoring
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

Also read:

- `scenarios/workspace-sandbox/api/main.go` (current DB bootstrap, lines 60–95
  and 413–540).
- `scenarios/workspace-sandbox/api/internal/repository/sandbox_repo.go` (the
  full ~1700-line repository — the largest rewrite surface).
- `scenarios/workspace-sandbox/initialization/postgres/schema.sql` (canonical
  schema being replaced).
- `packages/api-core/database/connect.go` (already supports
  `Driver: "sqlite"` via `modernc.org/sqlite`; we use it as-is).
- `packages/api-core/storage/` (`Resolver`, `ClassData`, `EnsureClassDir`,
  `WriteFileAtomic`) — used for resolving the SQLite file path.

## 4. Problem Statement

`scenarios/workspace-sandbox` was built against PostgreSQL and pulls in heavy
postgres-only features:

- pgSQL stored functions: `vrooli_uuid_v4`, `check_scope_overlap`,
  `get_sandbox_stats`, `update_sandbox_last_used` trigger.
- `TEXT[]` arrays serialized via `pq.Array` (5+ call sites in repository).
- `JSONB` columns for `metadata` and `behavior`.
- `TIMESTAMPTZ` everywhere; `NOW()` defaults.
- `FOR UPDATE` row locks in `CheckScopeOverlap` (used to serialize concurrent
  creates).
- `$1/$2` placeholders with `strconv.Itoa(qb.argNum)` builders.
- Schema-isolated `workspace_sandbox` namespace via `search_path`.
- Three SQL files: `schema.sql`, `migration_001_runtime_schema_alignment.sql`,
  `migration_002_provenance_schema_version.sql`, plus a runtime
  `ensureSchema()` audit in Go that re-validates expected columns.

We want to swap this for a self-contained SQLite database that:

- Lives in a single OS-appropriate path resolved through `api-core/storage`
  (`ClassData`).
- Has one canonical schema file (no migration files, no "v1.0.0" bookkeeping).
- Uses one driver only (`modernc.org/sqlite`, driver name `sqlite`).
- Passes Tier-2 cross-platform readiness (CGO_ENABLED=0 builds clean).

## 5. Scope

**In scope**

- Replacing the SQL schema, repository implementation, repository tests,
  bootstrap path, config struct, env-var contract, and service.json declaration.
- Removing `initialization/postgres/` entirely; replacing with a single
  `initialization/sqlite/schema.sql` (or embedded `//go:embed`).
- Updating `api-core/database` consumers: `Driver: "sqlite"`,
  `SQLITE_PATH` from env or resolver-derived path.
- Updating `docs/RESEARCH.md`, `docs/PROGRESS.md` notes, `README.md`, and the
  scenario PRD where they reference postgres.
- Updating tests to use real on-disk SQLite (`t.TempDir()`/`:memory:`) rather
  than `go-sqlmock` mocks so query-shape regressions are caught.

**Out of scope**

- Any data-migration code, dual-backend abstraction, or "phase rollout".
- Performance tuning of SQLite beyond the standard pragmas (WAL, busy_timeout,
  foreign_keys=ON).
- Changing the public HTTP/CLI contract — handler/CLI shape stays identical.
- Re-architecting the repository interface; only its implementation moves.

## 6. Current Technical Context (Files of Interest)

| Area | File(s) | Notes |
|------|---------|-------|
| DB bootstrap | `api/main.go` (~L60–95, L413–540) | `buildDSNWithSearchPath`, `ensureSchema`, postgres column audit. |
| Schema | `initialization/postgres/schema.sql` (274 lines) + 2 migration files | Two enums via DO blocks, two PL/pgSQL functions, one trigger. |
| Repo (impl) | `api/internal/repository/sandbox_repo.go` (1725 lines) | Both `SandboxRepository` and `TxSandboxRepository`; uses `pq.Array`, `$N` placeholders, `RETURNING`, `FOR UPDATE`, calls `check_scope_overlap()` and `get_sandbox_stats()`. |
| Repo (tests) | `api/internal/repository/sandbox_repo_test.go` (1673 lines) and `sandbox_repo_provenance_v1_test.go` (145 lines) | Built on `DATA-DOG/go-sqlmock` with regex-matched postgres queries. |
| Config | `api/internal/config/config.go` (`DatabaseConfig`, lines 331–359) and `config_test.go` (`POSTGRES_*` env list at L110–112, L228–237). |
| Service manifest | `.vrooli/service.json` (lines 178–185: `dependencies.resources.postgres`). |
| Env contract | `api/main.go` (`POSTGRES_HOST/PORT/USER/PASSWORD/DB/SSLMODE/URL`, `DATABASE_URL`). |
| Driver imports | `api/main.go:16` (`_ "github.com/lib/pq"`); `api/internal/repository/sandbox_repo.go:15` (`"github.com/lib/pq"`). |
| Docs that mention postgres | `docs/RESEARCH.md:120`, `docs/PROGRESS.md` (multiple lines). |

## 7. Target End State

**Storage backend**

- SQLite database file at the path resolved by:
  ```
  resolver.Path(Options{ScenarioID: "workspace-sandbox"}, ClassData, "workspace-sandbox.db")
  ```
  with `EnsureClassDir(..., ClassData, 0o755)` called at startup. `SQLITE_PATH`
  env var overrides this (consumed by `api-core/database`).
- `modernc.org/sqlite` is the **only** SQL driver imported.
- Pragmas applied on connect: `journal_mode=WAL`, `foreign_keys=ON`,
  `busy_timeout=5000`, `synchronous=NORMAL`. Set as DSN params, not via runtime
  `db.Exec`, so they apply to every pooled connection. Pool is sized to a
  single open connection (`SetMaxOpenConns(1)`) since SQLite serializes writes
  and the rest of the codebase assumes the connection works under contention.

**Schema**

- One file: `initialization/sqlite/schema.sql` (idempotent, all-in-one), embedded
  in the binary via `//go:embed`. The file is also kept on disk so the lifecycle
  bootstrapper can read it for parity with other scenarios.
- `initialization/postgres/` deleted entirely. No migration directory.
- Type mapping (single mapping, applied uniformly):

  | Postgres                | SQLite                                 |
  |-------------------------|----------------------------------------|
  | `UUID`                  | `TEXT` (canonical 36-char form)        |
  | `JSONB`                 | `TEXT` (JSON stored as text; SQLite has json1) |
  | `TEXT[]`                | `TEXT` (JSON array stored as text)     |
  | `TIMESTAMPTZ`           | `TEXT` (RFC3339Nano, UTC) — pick TEXT over INTEGER for human-readable diffs |
  | `INTEGER[]`             | `TEXT` (JSON array)                    |
  | `BIGINT`                | `INTEGER`                              |
  | `BOOLEAN`               | `INTEGER` (0/1)                        |
  | `sandbox_status` ENUM   | `TEXT` + `CHECK (status IN (...))`     |
  | `change_type` ENUM-like | `TEXT` + `CHECK (...)`                 |

  Choose TEXT-JSON (not BLOB-JSON) so existing `json.Marshal(...)` call sites
  in `sandbox_repo.go` continue to work; readers `json.Unmarshal` from
  `string`. Times are formatted with `time.RFC3339Nano`; the helper functions
  for parse/format live in a single new file
  `api/internal/repository/sqlite_codec.go`.

**Functions and triggers**

- `check_scope_overlap` and `get_sandbox_stats` are removed from SQL and
  reimplemented in Go inside `sandbox_repo.go`:
  - Overlap check becomes a parameterized `SELECT id, scope_path, status FROM
    sandboxes WHERE project_root = ? AND no_lock = 0 AND status IN
    ('creating','active','stopped') AND ? != id_or_null` followed by an
    in-Go prefix-match against the JSON-decoded `reserved_paths` array. To
    serialize concurrent `Create` calls (the previous `FOR UPDATE` use case),
    wrap the overlap-check + insert in a single `BEGIN IMMEDIATE` transaction;
    SQLite's `BEGIN IMMEDIATE` acquires the reserved write lock and gives
    equivalent mutual exclusion.
  - Stats becomes a single `SELECT COUNT(*) FILTER (...)` query; SQLite
    supports the `FILTER` clause as of 3.30.
- The `sandbox_last_used_trigger` becomes a SQLite trigger
  (`AFTER UPDATE OF status, active_pids ON sandboxes`).

**Repository changes**

- Remove all `pq.Array(...)` wrappers — replace with `mustJSON(slice)` (helper
  that returns a `string`).
- Replace `$N` placeholders with `?` (SQLite supports both, but the codebase
  must standardize on `?` for consistency; the dynamic builder
  `qb.argNum` is replaced by simple `?` accumulation).
- Replace direct calls to `check_scope_overlap()` / `get_sandbox_stats()` with
  the new Go implementations described above.
- Delete `TxSandboxRepository`'s `FOR UPDATE` query; replace with the
  `BEGIN IMMEDIATE` pattern at the call sites that need write serialization.
- Time scanning: introduce `scanTime(*string) time.Time` helper. All
  `TIMESTAMPTZ` columns are read as `*string` and parsed.

**Config changes**

- `DatabaseConfig` is reduced to:
  ```go
  type DatabaseConfig struct {
      Path string // SQLite file path (defaults via api-core/storage resolver)
  }
  ```
- All `POSTGRES_*` env reads in `config.go` and `main.go` are deleted.
  Only `SQLITE_PATH` is honored as an override.

**Service manifest**

- `.vrooli/service.json` `dependencies.resources` becomes `{}` (no shared
  resources required). Add `notes.storage_strategy: "Embedded SQLite via
  modernc.org/sqlite; no shared database resource"` for clarity.

**Docs**

- `docs/RESEARCH.md`, `docs/PROGRESS.md`, `README.md`, and `PRD.md` updated to
  state SQLite is the storage backend. PROGRESS.md historical entries that
  reference `initialization/postgres/seed.sql` are corrected or pruned (they
  are historical narrative — replace the "Database Schema Changes" sections
  with a one-line note pointing at the new SQLite schema file).
- `docs/internal/STORAGE_AUDIT.md` is created and filled per `storage-steer`
  audit template, recording the new posture.
- `docs/internal/PORTABILITY_AUDIT.md` is created/updated per
  `cross-platform-readiness` template noting Tier-2 readiness improvement.

## 8. Implementation Strategy (Phased)

The phases are linear; each ends with a gate that must pass before the next
begins. The whole sequence ships in a single PR — there is no staged rollout.

### Phase A — Schema authoring

1. Write `initialization/sqlite/schema.sql` from scratch (do not translate the
   postgres file mechanically; rewrite for SQLite's idiom). Includes:
   - `CREATE TABLE IF NOT EXISTS sandboxes (...)` with TEXT/INTEGER columns
     per the type mapping in §7 and CHECK constraints for `status`,
     `scope_path`, `project_root`.
   - `sandbox_changes`, `sandbox_audit_log`, `applied_changes` tables (same
     columns, retyped). `applied_changes` includes the auditability columns
     inline (`run_outcome`, `provenance_state`, `conversation_id`, `cost_usd`,
     `agent_manager_run_id`) — they are part of the canonical schema, not a
     migration. The `schema_version` column is **not** included; it was
     dropped on the current branch by migration_003 and the proto package
     path now carries version semantics
     (`packages/proto/schemas/workspace-sandbox/v1/...`).
   - All indexes from the postgres file, retyped (partial indexes via
     `WHERE` are supported by SQLite).
   - The `sandbox_last_used_trigger` rewritten as a SQLite `AFTER UPDATE OF`
     trigger.
   - No functions, no enums, no extensions.
2. Embed it: add `//go:embed initialization/sqlite/schema.sql` in a new
   `api/internal/repository/schema.go` exporting `SchemaSQL string`.

### Phase B — Bootstrap & connection

1. In `api/main.go`:
   - Replace `_ "github.com/lib/pq"` with `_ "modernc.org/sqlite"`.
   - Delete `buildDSNWithSearchPath`, `appendSearchPath`, the
     `POSTGRES_*` validation, and the `ensureSchema` column-audit loop.
   - Add `resolveSQLitePath()` using `api-core/storage` resolver (override via
     `SQLITE_PATH`).
   - Call `database.Connect(ctx, database.Config{ Driver: "sqlite", DSN:
     pathWithPragmas })` where `pathWithPragmas` appends
     `?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)`.
   - After connect, `db.SetMaxOpenConns(1)` (per `storage-steer` §4.5
     SQLite guidance).
   - Run `db.ExecContext(ctx, repository.SchemaSQL)` once; the file is
     idempotent so this is safe on every startup.
2. Remove the `migration_001` / `migration_002` files and their references.

### Phase C — Repository rewrite

1. In `api/internal/repository/sandbox_repo.go`:
   - Delete the `pq` import.
   - Add `sqlite_codec.go` with helpers: `marshalStrings([]string) string`,
     `unmarshalStrings(string) []string`, `marshalInts`, `unmarshalInts`,
     `marshalJSON(any) string`, `unmarshalJSON(string, any)`, `formatTime`,
     `parseTime`.
   - Replace every `pq.Array(...)` argument with the marshal helper.
   - Replace every `$N` with `?` (delete the dynamic `qb.argNum` builder; use
     a `[]any` arg slice and `strings.Repeat("?,", n)` patterns where needed).
   - Rewrite `CheckScopeOverlap` to query `sandboxes` directly and apply the
     prefix-match in Go after unmarshalling `reserved_paths`. The Tx variant
     opens with `BEGIN IMMEDIATE` (set on the `sql.TxOptions` via custom
     wrapper, since `database/sql` doesn't expose it directly — use
     `db.ExecContext(ctx, "BEGIN IMMEDIATE")` then `Tx`-equivalent flow, OR
     run the overlap-check + insert through a single `*sql.Conn` with
     `BEGIN IMMEDIATE; ...; COMMIT;` orchestrated by a small helper).
   - Rewrite `GetStats` to a single inline query.
   - Update all timestamp reads to scan into `*string` and parse.
2. Re-run `gofumpt -w`, `go vet ./...`.

### Phase D — Tests

1. Replace `go-sqlmock` with a real on-disk SQLite test harness:
   - New helper in `api/internal/repository/testdb_test.go`:
     ```go
     func newTestDB(t *testing.T) *sql.DB {
         path := filepath.Join(t.TempDir(), "test.db")
         db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
         ...
         _, err = db.Exec(SchemaSQL)
         ...
         t.Cleanup(func() { db.Close() })
         return db
     }
     ```
   - Rewrite `sandbox_repo_test.go` and `sandbox_repo_provenance_v1_test.go`
     against this harness. Tests check **behavior** (insert -> read
     round-trip; overlap detection; stats counts; optimistic-version bumps),
     not raw query shape.
2. Update `api/internal/config/config_test.go`:
   - Drop `POSTGRES_*` and `DATABASE_URL` env-var assertions.
   - Add a single test: `SQLITE_PATH` overrides the default resolver path.
3. Drop `github.com/DATA-DOG/go-sqlmock` from `go.mod` once unused.
4. Add `modernc.org/sqlite` to `go.mod`.

### Phase E — Manifest, docs, hygiene

1. Edit `.vrooli/service.json`:
   - Remove `dependencies.resources.postgres` (set to `{}` or omit `resources`).
   - Add `notes.storage_strategy` per §7.
2. Delete `initialization/postgres/` (entire directory).
3. Update `docs/RESEARCH.md`, `docs/PROGRESS.md`, `README.md`, `PRD.md` to
   reflect SQLite. Treat PROGRESS.md historical narrative as living doc:
   prune obsolete postgres sections rather than annotating.
4. Create `docs/internal/STORAGE_AUDIT.md` and
   `docs/internal/PORTABILITY_AUDIT.md` per the templates in
   `storage-steer` §11.3 and `cross-platform-readiness` §8.3.
5. Run a grep gate (Phase F).

### Phase F — Verification gate

Run all of these and confirm clean output before marking done:

```bash
cd scenarios/workspace-sandbox/api

# 1. Build and static-build
go build ./...
CGO_ENABLED=0 go build ./...

# 2. Tests
go test ./... -timeout 300s

# 3. Vet, format
go vet ./...
gofumpt -l . | tee /tmp/gofumpt.out  # must be empty

# 4. Greenfield grep gates (must all return zero results)
rg -i 'lib/pq|jackc/pgx|POSTGRES_[A-Z_]+|DATABASE_URL|POSTGRES_URL' .
rg 'pq\.Array|::jsonb|TIMESTAMPTZ|FOR UPDATE|uuid-ossp|pgcrypto' . ../initialization
rg 'check_scope_overlap|get_sandbox_stats|vrooli_uuid_v4' .
test ! -d ../initialization/postgres
test -f ../initialization/sqlite/schema.sql

# 5. Manifest gate
jq '.dependencies.resources' ../.vrooli/service.json
# expected: {} or absent

# 6. Restart scenario end-to-end (per feedback_use_vrooli_scenario_restart)
vrooli scenario restart workspace-sandbox
vrooli scenario logs workspace-sandbox --tail 100
# health endpoint must report ready; sandbox create + list + approve flow
# must succeed against a clean .db file
```

## 9. Contract Decisions

- **Public HTTP / CLI surface:** unchanged. Same routes, same JSON shapes.
  Internal storage type changes are invisible to callers.
- **Env var contract:** drops every `POSTGRES_*` and `DATABASE_URL`. Adds
  exactly one optional override: `SQLITE_PATH`. If unset, the path is derived
  from `api-core/storage` (`ClassData/workspace-sandbox/workspace-sandbox.db`).
- **Schema name:** removed. SQLite has no schemas; the database file itself
  is the unit of isolation.
- **UUIDs:** generated in Go via `github.com/google/uuid` (already imported)
  and stored as canonical 36-char TEXT. SQLite never generates UUIDs.
- **Time semantics:** all timestamps are stored as RFC3339Nano UTC strings.
  Reads pass through a `parseTime` helper that returns `time.Time` in UTC.
- **Concurrency:** `BEGIN IMMEDIATE` replaces `FOR UPDATE` for the
  Create/CheckScopeOverlap pair. Pool is single-connection; reads serialize
  through the connection. This is acceptable because the scenario is
  single-host and write-light by design.

## 10. Testing Plan

Automated tests only — no manual checklists. All tests run under `go test
./...` from `scenarios/workspace-sandbox/api/`.

- **Repository round-trip (`sandbox_repo_test.go`)** — for each of Create,
  Get, Update, Delete, List, CheckScopeOverlap, GetStats, GetAuditLog,
  GetPendingChangeFiles, MarkChangesCommitted: write -> read -> assert
  equality of every field including `metadata` JSON, `behavior` JSON,
  `tags`/`reserved_paths`/`active_pids` arrays, and timestamps.
- **Optimistic locking** — bump version twice with a stale `expectedVersion`
  and assert the second update returns the documented "version conflict"
  error.
- **Overlap detection** — exhaustive matrix mirroring the existing
  `assumptions_test.go` cases (ancestor, descendant, sibling, exact,
  cross-project).
- **Concurrency smoke test** — fire 8 goroutines doing `Create` with
  overlapping `reserved_paths`; assert exactly one succeeds. This exercises
  `BEGIN IMMEDIATE`.
- **Schema embedding** — assert `repository.SchemaSQL` is non-empty and
  applies cleanly to a fresh `:memory:` database.
- **Config** — `SQLITE_PATH` override is read; absence falls back to the
  resolver-derived path.
- **Static build** — CI check `CGO_ENABLED=0 go build ./...` succeeds (covers
  cross-platform readiness).
- **Greenfield grep gate** — a `Makefile` target `make verify-greenfield`
  that runs the rg commands from §8 Phase F and exits non-zero on any hit.
  Wired into `make test`.

## 11. Rollout / Validation Checklist

- [ ] All Phase F commands pass clean.
- [ ] `vrooli scenario restart workspace-sandbox` brings the scenario up with
      no manual DB provisioning.
- [ ] `~/.local/share/vrooli/workspace-sandbox/data/workspace-sandbox.db` (or
      platform equivalent) is created on first start; deleting it and
      restarting yields a fresh, working install.
- [ ] `docs/internal/STORAGE_AUDIT.md` records the new posture and shows zero
      red-flag hits.
- [ ] `docs/internal/PORTABILITY_AUDIT.md` records Tier-2 readiness with
      embedded SQLite, CGO_ENABLED=0 verified.

## 12. Risks and Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Subtle behavior drift from rewriting `check_scope_overlap` in Go | Med | Property-style overlap tests (§10) drawn from existing `assumptions_test.go` matrix. |
| `BEGIN IMMEDIATE` semantics differ from `FOR UPDATE` under high concurrency | Low–Med | Concurrency smoke test (§10) plus `SetMaxOpenConns(1)` to bound contention; the scenario is single-host and write-light. |
| `modernc.org/sqlite` perf vs `mattn/go-sqlite3` | Low | Pure-Go is the explicit Tier-2 requirement; perf is acceptable for sandbox metadata workloads. |
| JSON-encoded arrays slow down indexed lookups | Low | Indexes that previously used `unnest(reserved_paths)` are replaced with index on `project_root` plus in-Go prefix match; dataset is bounded (sandboxes per host). |
| Existing local dev databases blocking the cutover | Low | One-time manual export performed out-of-tree before merge; no migration code lands. |

## 13. Non-goals / Prohibited Patterns

- **No** `Dialect` enum, `Driver` interface, or runtime backend switch.
- **No** dual-driver imports — `lib/pq` must be **gone** from `go.mod`.
- **No** `migrations/` directory and **no** migration-numbering scheme.
- **No** "schema_version" bookkeeping table.
- **No** keep-around-for-history postgres SQL files anywhere in the tree.
- **No** generic `// removed for SQLite cutover` comments — delete cleanly.
- **No** keeping `POSTGRES_*` env vars as deprecated/ignored fallbacks.
- **No** mock-only tests; the new tests must drive a real SQLite database.

## 14. Definition of Done

A reviewer pulling the branch fresh observes:

1. `rg -i 'postgres|lib/pq|pq\.Array|JSONB|TIMESTAMPTZ' scenarios/workspace-sandbox`
   returns **only** mentions inside this plan file (`docs/plans/...`) and
   commit-message-style historical entries that have been pruned. The
   `api/`, `cli/`, `ui/`, `initialization/`, `.vrooli/`, and runtime docs
   have **zero** hits.
2. `cd scenarios/workspace-sandbox/api && CGO_ENABLED=0 go build ./...` and
   `go test ./...` both succeed within standard timeouts.
3. `vrooli scenario restart workspace-sandbox` starts cleanly on a host with
   no PostgreSQL service running and no `POSTGRES_*` env vars set.
4. `cat .vrooli/service.json | jq '.dependencies.resources'` returns `{}`.
5. `ls initialization/` shows `sqlite/` and **no** `postgres/`.
6. `docs/internal/STORAGE_AUDIT.md` and
   `docs/internal/PORTABILITY_AUDIT.md` exist and are filled out.
7. The `make verify-greenfield` target exists and exits 0.

If any of those fail, the plan is not yet done.
