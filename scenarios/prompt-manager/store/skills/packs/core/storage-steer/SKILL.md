---
name: "storage-steer"
description: "Steers scenarios toward engine-independent, per-domain storage architecture: schema embedded next to the code that interprets it (SchemaProvider/EnsureSchemas substrate in api-core/database), repository interfaces hide the engine, the same per-domain rule applies to Qdrant collections and Redis namespaces, and migrations follow a three-tier story (declarative greenfield, out-of-tree scripts for greenfield-with-data, versioned migrations only when production schema evolution earns them)."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer"]
  tags: ["storage","database","schema-architecture","per-domain-schema","schema-provider","ensure-schemas","migrations","repository-pattern","sqlite","postgres","qdrant","redis","isolation","greenfield"]
  status: "active"
  defaultScope: "work-scope"
  targetDimensions: ["storage"]
  programmaticHome: "storage-manager:storage"
  revision: 52
  createdAt: "2026-01-30T20:00:00-05:00"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["git-control-tower", "prompt-manager", "swarm-manager", "test-genie", "vrooli"]
    commands: ["git-control-tower baseline", "prompt-manager skill", "prompt-manager skill read", "swarm-manager captures", "test-genie", "test-genie phase", "vrooli", "vrooli scenario"]
  origin:
    kind: "authored"
---
## Steer focus: Storage Architecture

**Scope mode: `architecture`.** Hold to that row of `work-scope` for this session.

Prioritize **engine-independent, per-domain storage architecture** in `scenarios/{{TARGET}}/`: schema lives next to the code that interprets it, repository interfaces hide the engine, and every store (SQL tables, Qdrant collections, Redis namespaces) follows the same per-domain rule. Greenfield by default; brownfield substrate only when production data evolution demands it. The destination is a production-ready, engine-portable storage layer where a domain is added by creating files in one folder, deleted by removing one folder, and any engine swapped behind its repository interface without business-logic churn. Do not break functionality, regress tests, or add features.

Required reading: `prompt-manager skill read cross-platform-readiness visited-tracker-tools`. `cross-platform-readiness` is the engine-selection authority (its §3 fitness table decides *which* store; this skill decides *how* to architect it): SQLite via `modernc.org/sqlite` (pure Go, CGO-free) for most relational data, PostgreSQL for concurrent multi-writer or managed-DB needs, Qdrant beside a relational store for vectors, Redis beside it for TTL cache, MinIO or filesystem beside it for blobs (metadata in the DB). Per-domain ownership applies to all of them.

### The programmatic counterpart: `storage-manager`

This skill says what good storage architecture is; **`storage-manager` measures whether a scenario conforms and gates test-isolation safety.** This skill's detection has graduated (`programmaticHome: storage-manager:storage`, dimension `storage`), so treat storage-manager as the source of findings and this steer as the judgment layer; it superseded the five retired `scenario-auditor` DB and storage rules (`routed_database_drivers`, `routed_database_handle_capture`, `database_backoff`, `db_rows_close`, `storage_namespace_helpers`), so run the validator instead of hand-rolled greps.

- `storage-manager validate scenario {{TARGET}}` is the audit: schema layout, per-domain ownership, migration hygiene, persistence-seam adoption and the SQL and file isolation proof, as a maturity assessment with `file:line` findings and remediation. `validate prove-isolation {{TARGET}}` runs only the proof.
- `storage-manager fix preview|apply {{TARGET}}` applies the deterministic, idempotent autofixes (`ENSURE_SCHEMAS_NOT_WIRED`, `DB_ROWS_NOT_CLOSED`); `fleet scan`, `advisor engines`, `advisor migrations` give fleet inventory, the Postgres-to-SQLite fitness advisor and migration hygiene; `declare inspect|suggest {{TARGET}}` and `declare check` compare `storage.entries` with observed bytes.
- `vrooli scenario test {{TARGET}}` runs the same engine as the delegated **storage phase**, before playbooks. Its L2 verdict is the fail-closed gate: when SQL or file isolation cannot be statically proven (`ROUTED_SEAMS_UNWIRED`, `FILE_ROUTED_SEAMS_UNWIRED`, `STORAGE_ISOLATION_UNVERIFIED`) test-genie refuses destructive E2E playbooks. Your changes must keep it green. A capability the analyzers lack is filed with `swarm-manager captures create` so storage-manager learns it.

### 1. Scope

In scope: the per-domain schema pattern for any store, repository and service abstraction, the greenfield-versus-brownfield migration strategy, cross-store ownership and namespacing, the storage audit and its documentation, cross-domain coupling. Out of scope: engine choice and the filesystem runtime contract (`cross-platform-readiness` §3, §4 and `package:api-core/storage`), query optimization, domain modeling, backup and DBA concerns, ORM or driver details.

### 2. The canonical pattern: domain-owned storage

Every domain owns the setup of every store it touches: `internal/<dom>/schema.sql`, `qdrant.go` and `redis.go` each expose a `Schema()` or `Setup()` function, `internal/modules/registry.go` collects them (`AllSchemas`, `AllSetups`), and `api/main.go` applies them at boot (`database.EnsureSchemas(...)` and the Qdrant and Redis equivalents). The reason is cohesion (one diff per logical change), deletability (remove the folder plus registry edits) and explicit coupling. Whatever store a domain wires to tomorrow (Neo4j, ClickHouse, MinIO), the new file goes in `internal/<dom>/` with a `Setup()` function, is registered, and is applied at boot: same shape every time. Worked examples: `templates/scenarios/react-vite/api/internal/notes/` (canonical SQL), `scenarios/agent-manager/api/internal/database/`.

Per-store shape. **SQL:** `schema.sql` (tables, indexes, triggers the domain owns), `schema.go` (`//go:embed`, `func Schema() string`), `types.go`, `repository.go` (interface), `sqlite.go` or `postgres.go`, `service.go`. **Qdrant:** `qdrant.go` builds the collection name through `storage.Collection("<domain>")`. **Redis:** `redis.go` builds keys through `storage.RedisKey("<domain>", ...)`. Never hardcode a Redis prefix or Qdrant collection embedding the scenario slug: compose through `storage.Collection`, `storage.RedisKey` and `storage.RedisPrefix` so the variant resolves at runtime from the lifecycle environment, which is what lets a scenario be developed as a shadow while live keeps serving (Baseline Modes); for SQLite and data paths pass `storage.ScenarioNamespace(slug)` as `Options.ScenarioID`. The scenario-auditor `storage-namespace-v1` standard (`storage_namespace_helpers`, a standards-battery finding) flags hardcoded namespaces.

Cross-cutting infrastructure (postgres extensions, custom types, cross-domain views) lives in `internal/database/system.sql`, empty by default in SQLite scenarios and tested for emptiness as a tripwire: a `CREATE TABLE` there means the table belongs to a domain that does not exist yet, so create the domain first.

### 3. Migration strategy: greenfield versus brownfield

The line is whether the scenario has real users in production, not whether the dev database holds data. Take the signal from the operator's words, deployment evidence, or `PROGRESS.md`, `PROBLEMS.md` or `ARCHITECTURE.md`, and ask if unclear: misclassifying greenfield as brownfield burdens the code with migrations forever, the reverse risks user data on the next schema change.

**Greenfield (default).** No `migrations/` folder; `schema.sql` always describes the clean desired state. A new table or index is free (`CREATE ... IF NOT EXISTS`; boot's `EnsureSchemas` creates it). Any change to the columns of an existing table (add, drop, rename, retype) is two steps: edit `schema.sql` to the final shape and write a one-shot script that brings the existing database there (`EnsureSchemas` cannot alter an existing table, and on SQLite boot fails loudly naming the missing table and column). **Always migrate, never recreate**; never decide the data is disposable. The script lives in `/tmp/<scenario>/migrate-<what>.{sql,go,sh}` (or `~/.cache/vrooli/<scenario>/` across reboots), is never committed, is safe to re-run (guard with `PRAGMA table_info` before `ALTER`), runs with the scenario stopped, and is deleted once boot's drift check passes. Under a Baseline Modes engagement (`git-control-tower baseline start`) migrations live in the engagement's managed, ordered per-baseline folder, each transactional and idempotent, and `baseline promote` dry-runs them against a copy of live before applying them in the quiesce window. Renaming existing Redis keys or Qdrant collections to adopt the variant-aware helpers is a throwaway script run once with the scenario stopped after `data-backup-manager safety backup-now --scenario <s>`; committed code carries no adoption shim.

**Brownfield (real users).** Schema evolution is ordered, transactional and applied once per environment: `internal/<dom>/migrations/` with `migrations.go` (`Migrations() embed.FS`, `DomainID()`), `001_initial.sql` replacing `schema.sql`, one logical change per timestamped file, up-only (recovery is from backup), a transaction each, and a per-domain version table (`schema_migrations_<domain>`). The substrate (`MigrationProvider` wrapping `pressly/goose` in `packages/api-core/database`) is not built: if you need it first, stop, write no ad-hoc migration tooling, describe the change to the operator and request the substrate as its own plan.

### 4. Coupling, repositories, isolation

Prefer soft foreign keys across domain boundaries (store the id, declare no `REFERENCES`); within-domain keys are encouraged. A hard cross-domain key blocks deleting a domain and ordering `EnsureSchemas` providers; genuine need for cross-domain integrity means the domains should merge or a third should own both (`system.sql` is not the answer). Business logic depends on a `Repository` interface in `internal/<dom>/repository.go`; implementations (`sqlite.go`, `postgres.go`) and a co-located fake (`mocks/repository.go`) sit in the same folder, and handlers, CLI commands and other domains call `Service`, never `Repository`; adding an engine is one implementing file and a constructor switch in `module.go`. Isolation: SQLite scenarios get their own file under `package:api-core/storage` paths, Postgres scenarios a scenario-named schema (`SET search_path TO {{TARGET}}`) declared in `service.json`, Redis keys `{scenario}:{domain}:`, Qdrant collections `{scenario}_{domain}_{purpose}`, applied even when the scenario is the only consumer.

### 5. Audit and `storage.entries`

Run `storage-manager validate scenario {{TARGET}}` first, read its findings, apply its autofixes, and inspect manually only what its analyzers do not cover. Its findings map onto these red flags: no per-domain `schema.sql` (schema centralized); a deprecated `internal/store/` or `internal/db/` package holding schema; non-idempotent schema (no `IF NOT EXISTS`); SQL in handlers; hard cross-domain keys; versioned migrations in a greenfield scenario, or destructive schema change on production data without them; Redis keys or Qdrant collections without scenario and domain prefix; hardcoded credentials; filesystem runtime writes bypassing `package:api-core/storage`.

Every durable filesystem claim belongs in the owner's `storage.entries` map; an entry may narrow the owner's platform set with `platforms`, never widen it. Pick one rung: `owned` (Vrooli owns the writer and location: a relative path under a class root or a portable token), `relocatable` (the upstream tool exposes a lever: portable default plus a `relocation` block with `key`, optional `config`, and `scope` of `process-tree`, `session`, `daemon` or `container`), `pinned` (no lever: the observed default with a budget and reclaim command, which is complete; never penalize a lever the upstream lacks). Portable tokens are `$USER_HOME`, `$USER_CONFIG_DIR`, `$USER_CACHE_DIR`, `$USER_STATE_DIR`, `$USER_DATA_DIR`, `$TEMP_DIR` (never XDG names); a string `path` suffices when one token fits, otherwise an object with `linux`, `macos`, `windows` keys (a missing key is a defect, explicit `null` declares absence). Declare `class` (`config`, `data`, `cache`, `logs`, `state`), `regenerable`, `sensitive`, `rationale`, a measured `budget` (`max_bytes` or `max_age` with number and date), and `reclaim.command` when an operator can reclaim it; a `format: sqlite` entry declares both `-wal` and `-shm`. Lint codes (`STORAGE_PATH_NOT_PORTABLE`, `STORAGE_PATH_PLATFORM_MISMATCH`, `STORAGE_PATH_BRANCH_MISSING`, `STORAGE_TOKEN_SUPERSEDABLE`, `contradictory_storage_platforms`) each name their fix; validate emitted blocks against `common.schema.json`.

For a non-conforming scenario recommend the documented pattern and record durable findings, but do not refactor pre-emptively: it is a per-scenario decision with the user, and migration help for a real legacy scenario is escalated.

### 6. Findings, memory and output

Record only durable findings concisely (date, verified pattern, migration classification, `file:line` evidence, highest-impact recommendations) in the scenario's existing `docs/internal/PROBLEMS.md` or `docs/ARCHITECTURE.md`; create no standalone audit file (a legacy `STORAGE_AUDIT.md` is optional input). At session start read `scenarios/{{TARGET}}/.vrooli/service.json`, those documents, `packages/api-core/database/schemas.go` and `packages/api-core/docs/storage.md`; track visited files with `visited-tracker-tools` (location `scenarios/{{TARGET}}`, tag `storage-architecture`). At session end verify existing claims against code, correct inaccuracies, update the greenfield-or-brownfield classification if the data state changed, and note what was not audited.

You may add per-domain `schema.sql`, `schema.go`, `qdrant.go` and `redis.go`, refactor a centralized schema into the per-domain layout when scope permits, add repository interfaces and the modules registry, and namespace Redis keys and Qdrant collections by scenario and domain. You must keep `{{TARGET}}` functional and non-regressed, keep schema idempotent and next to the code that interprets it, pick the migration strategy from whether real users exist, use environment variables for connection details, route mutable filesystem state through `package:api-core/storage`, and escalate when brownfield migrations are needed. You must not invent versioned-migration tooling inline, centralize schema across domains, write SQL in handlers, hardcode credentials, add hard cross-domain foreign keys, skip scenario and domain prefixes, or make superficial renames that do not improve the storage architecture.
