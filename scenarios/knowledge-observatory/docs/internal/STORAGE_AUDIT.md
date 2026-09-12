# knowledge-observatory storage audit

Audit date: **2026-08-01**. Performed before any storage code changed.

Authority: `storage-steer` (architecture) and `cross-platform-readiness` §3
(engine selection). Findings come from `storage-manager`, not hand-rolled greps.

## 1. Current state

| Property | Value |
|----------|-------|
| Relational engine | PostgreSQL (`required: true`) |
| Vector engine | Qdrant (`required: true`) |
| Schema location | `api/internal/<domain>/schema.sql` (274 lines, all 13 tables) |
| Seed location | `api/internal/<domain>/seed.sql` (111 lines) |
| Migrations folder | none |
| Database | `vrooli_knowledge_observatory`, schema `knowledge_observatory` |
| Database size | 252 MB |

### storage-manager verdict (baseline)

`storage-manager validate scenario knowledge-observatory` → **failed**, local
maturity **L1**, 2 errors and 8 warnings.

| Severity | Code | Location |
|----------|------|----------|
| ERROR | `ROUTED_SEAMS_UNWIRED` | `api/main.go` |
| ERROR | `FILE_ROUTED_SEAMS_UNWIRED` | `api/main.go` |
| WARNING | `DIRECT_SQL_IN_HANDLERS` | `api/collections.go:269` |
| WARNING | `STORAGE_NAMESPACE_HARDCODED` | `api/internal/aisearch/keys.go:24` |
| WARNING | `SQL_DB_HANDLE_CAPTURE` | `api/server.go:74`, `api/server.go:125` |
| WARNING | `SQL_DB_HANDLE_CAPTURE` | `api/internal/adapters/deepsearchstore/postgres.go:16` |
| WARNING | `SQL_DB_HANDLE_CAPTURE` | `api/internal/adapters/docaccessstore/postgres.go:13` |
| WARNING | `SQL_DB_HANDLE_CAPTURE` | `api/internal/adapters/dochealingstore/postgres.go:16` |
| WARNING | `SQL_DB_HANDLE_CAPTURE` | `api/internal/adapters/metadatastore/postgres.go:15` |

The two errors are a **fail-closed gate**: while they stand, test-genie refuses
destructive E2E playbooks for this scenario rather than risk mutating real data.

`storage-manager advisor engines` ranks this scenario a Postgres→SQLite candidate
at **fitness 0.80**.

## 2. Row inventory

Counted with exact `count(*)` queries on 2026-08-01.

> **Do not use `pg_stat_user_tables.n_live_tup` for this inventory.** It reported
> zero for every table on this database while the tables held over a million
> rows. Only exact counts are trustworthy here.

| Table | Rows | Owning domain |
|-------|------|---------------|
| `quality_metrics` | **1,230,625** | `quality` |
| `knowledge_relationships` | 3,869 | `graph` |
| `doc_access_log` | 435 | `docaccess` |
| `collection_stats` | 95 | `quality` |
| `search_history` | 5 | `search` |
| `alerts` | 0 | `alerts` |
| `deep_search_jobs` | 0 | `deepsearch` |
| `doc_heal_jobs` | 0 | `dochealing` |
| `external_id_map` | 0 | `metadata` |
| `ingest_history` | 0 | `ingest` |
| `ingest_jobs` | 0 | `ingest` |
| `knowledge_metadata` | 0 | `metadata` |
| `user_preferences` | 0 | `preferences` |
| **Total** | **1,235,029** | |

`dashboard_metrics` is a **VIEW**, not a base table. It derives from
`collection_stats` and carries no rows of its own. It is owned by `quality`.

All 13 base tables have exactly one owner; no table is unassigned or assigned
twice.

### Unbounded growth in `quality_metrics`

`quality_metrics` accumulates roughly **10,000 rows per day** across 39
collections (one sample per collection every ~5.6 minutes) and has done so since
2026-01-25. There is no retention policy. This is the source of 99.6 % of the
scenario's rows and effectively all of its 252 MB.

This is a defect, not a data set. It is recorded here because it changes the
migration plan.

### Dead tables

`ingest_jobs`, `alerts`, and `user_preferences` are declared in the central
schema and referenced by **no Go code** (only `seed.sql` writes to `alerts` and
`user_preferences`). All three hold zero rows. This is the deletability bug that
`storage-steer` §10.2 red-flags: a table in a central schema survives the removal
of its feature and is recreated on every boot.

They are given owners below rather than deleted, so the restructure stays
behaviour-preserving. Deleting them is a follow-up.

## 3. Migration classification

**Greenfield**, per `storage-steer` §5.

Evidence: this is a single-workstation development instance. There are no
external users and no shipped data. The repository therefore gains **no**
`migrations/` folder and no `MigrationProvider` substrate.

**Greenfield does not mean recreate.** The existing rows move through a one-shot
script under `/tmp/knowledge-observatory/`, which is personal scratch and is
never committed.

### Operator decision on `quality_metrics` (2026-08-01)

The plan was authored against a stale inventory of 208 rows. The real inventory
is 1.23 M rows, which is a material difference, so the disposition was referred
to the operator rather than assumed.

**Decision: downsample, then migrate.**

| Data | Treatment |
|------|-----------|
| `quality_metrics` newer than 30 days | Migrate every sample at full resolution |
| `quality_metrics` older than 30 days | Collapse to one row per collection per day |
| Every other table | Migrate 1:1, no filtering |

Every trend line is preserved; only redundant intra-day resolution in historical
data is collapsed. A retention policy is added as part of the migration so the
growth stays bounded.

## 4. Target architecture

Per-domain ownership. Every domain owns its tables, its schema, and its
repository implementations in one folder, so a domain can be added or deleted
without touching a central file.

```
api/internal/<domain>/
  schema.sql      -- CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS only
  schema.go       -- //go:embed schema.sql, exports Schema() string
  repository.go   -- engine-neutral repository interface for the domain
  sqlite.go       -- SQLite implementation
  mocks/          -- co-located test fake
```

| Domain | Tables |
|--------|--------|
| `quality` | `quality_metrics`, `collection_stats`, `dashboard_metrics` (view) |
| `search` | `search_history` |
| `ingest` | `ingest_history`, `ingest_jobs` |
| `metadata` | `knowledge_metadata`, `external_id_map` |
| `graph` | `knowledge_relationships` |
| `deepsearch` | `deep_search_jobs` |
| `dochealing` | `doc_heal_jobs` |
| `docaccess` | `doc_access_log` |
| `alerts` | `alerts` |
| `preferences` | `user_preferences` |

`internal/database/system.sql` holds cross-cutting objects only. It is expected
to stay **empty**, and a test enforces that (the tripwire required by
`storage-steer` §4.1).

### Favourable starting point

The scenario already has a ports-and-adapters shape: `internal/ports/ports.go`
declares `VectorStore`, `Embedder`, `MetadataStore`, and `DocAccessLogger`, and
`internal/adapters/*/postgres.go` implements them. `storage-steer` §8 wants
exactly this seam, so adding SQLite is a new file per domain rather than a
rewrite of business logic.

### No foreign keys to unwind

The current schema declares **no** `REFERENCES` clauses. There are no
cross-domain hard foreign keys to replace with soft IDs, so the per-domain split
is clean.

### Engine decision

**SQLite** via `modernc.org/sqlite` (pure Go, CGO-free), per
`cross-platform-readiness` §3 and the 0.80 fitness score. Nothing here needs
multi-writer concurrency or managed-database operations.

**Qdrant stays required.** Vectors are its job, and the hybrid
relational-plus-vector shape is endorsed by `storage-steer` §3. This migration
removes one required resource, not two.

## 5. Backups

A pre-migration snapshot is taken through `data-backup-manager` before any data
moves, and a second snapshot immediately before the move itself. The second is
the one that matters if the migration script misbehaves.

## 6. Progress

| Phase | State | Evidence |
|-------|-------|----------|
| 7 — Audit and classification | **done** | This document. Backups at `~/.vrooli/backups/knowledge-observatory/pre-sqlite-migration-20260801.sql.gz` and `pre-move-20260801b.sql.gz` (42 MB each), both verified to contain all 13 `COPY` blocks. |
| 8 — Per-domain schema ownership | **done** | Central `api/internal/<domain>/schema.sql` deleted. All 13 tables owned by exactly one `internal/<domain>/schema.sql`. |
| 9 — SQLite behind the ports | **done** | 10 domains have a SQLite implementation and a round-trip test covering every column. Scenario boots and serves on SQLite. |
| 10 — Clear isolation and hygiene findings | **done except one** | `storage-manager` reports `status=passed errors=0 warnings=1`. The remaining warning is the Qdrant namespace, escalated below. |
| 11 — Data move and Postgres removal | **done** | 256,935 rows migrated, `postgres` resource removed, scenario starts healthy with no credential gap. |

### Deviations from the plan, and why

1. **`data-backup-manager` would not start** (repeated timeouts), so both snapshots
   were taken with a direct verified `pg_dump`. The snapshot requirement is met;
   the tool path is not.
2. **The migration script was Python, not Go.** The plan named
   `/tmp/knowledge-observatory/migrate-postgres-to-sqlite.go`. A Go script in
   `/tmp` needs its own module scaffolding with `replace` directives for the
   in-repo packages; `psql COPY` to CSV plus the standard-library `sqlite3`
   module needed none. The script lived at
   `/tmp/knowledge-observatory/migrate-postgres-to-sqlite.py`, was run twice to
   prove idempotency, and was **deleted**. It was never committed.
3. **The Postgres driver was removed with `go mod tidy`**, not through
   `scenario-dependency-analyzer`. The analyzer exposes `deps install` and
   `deps reconcile` only — there is no uninstall verb — so removing the import
   and tidying is the available mechanism.
4. **A bug in `packages/api-core/database` had to be fixed first.** Its schema
   drift check read columns via `PRAGMA table_info`, which omits generated
   columns, so any schema declaring one was reported as permanently drifted at
   boot — and the error text sent the operator to write an `ADD COLUMN`
   migration that SQLite rejects for generated columns. Switched to
   `PRAGMA table_xinfo` (a strict superset, so it can only remove false
   positives) with a regression test. This is outside the scenario, but it
   blocked `quality_metrics.avg_quality`.

### Open item for the operator: Qdrant collection namespace

`api/internal/aisearch/keys.go:24` still hardcodes `vrooli-docs`. Resolving it
through `storage.Collection("docs")` — which is what clears the last warning and
takes isolation safety to its maximum — would rename the live collection to
`knowledge-observatory_docs`, orphaning **12,788 indexed points**.

Rebuilding them means re-embedding the whole documentation corpus, and the
`ollama` resource this scenario embeds with is currently `enabled: false`. The
reindex is therefore not obviously safe to run here, which is exactly the case
plan Phase 10 step 10 says to escalate rather than guess. No other scenario
references `vrooli-docs`, so the rename is self-contained when it does happen.

## 7. Final state

| Property | Value |
|----------|-------|
| Relational engine | **SQLite** (`modernc.org/sqlite`, pure Go, CGO-free) |
| Database file | `<data>/knowledge-observatory/knowledge-observatory.db`, resolved through `storage.ScenarioNamespace` so a shadow never aliases live |
| Vector engine | Qdrant (`required: true`, unchanged) |
| Required credentials | **none** — `POSTGRES_PASSWORD` is gone |
| Pattern | Per-domain schema ownership via `internal/modules.AllSchemas()`; `internal/database/system.sql` is empty and a test enforces it |
| storage-manager verdict | **passed** — 0 errors, 1 warning (the Qdrant namespace above) |
| Retention | `quality_metrics` keeps 30 days at full resolution, then one row per collection per day (`api/retention.go`) |

### Migration result

| Table | Postgres | SQLite | Note |
|-------|----------|--------|------|
| `quality_metrics` | 1,230,625 | 256,935 migrated | Downsampled per the operator decision; all 39 collections and all 4,609 collection-days preserved, span 2026-01-25 .. 2026-08-01 intact |
| `collection_stats` | 95 | 95 | 1:1 |
| `knowledge_relationships` | 3,869 | 3,869 | 1:1 |
| `doc_access_log` | 435 | 435 | 1:1 |
| `search_history` | 5 | 5 | 1:1 |
| 8 empty tables | 0 | 0 | 1:1 |

The migration was run twice; the second run changed nothing, proving it is safe
to re-run. Without the retention policy the downsample would only have reset the
clock — the table would have returned to a million rows within four months.

