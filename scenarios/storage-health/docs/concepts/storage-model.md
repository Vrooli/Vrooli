# Storage Model

The mental model `storage-health` enforces and dogfoods. Read this with
`test-isolation-contract.md` (the safety throughline) and the finding
catalog in `.vrooli/maturity.json`.

## Engines

A scenario's storage surface is composed from a small, fixed set of
engines, classified from `.vrooli/service.json` resources plus code-facts:

| Engine | Role | Isolation concern |
|---|---|---|
| **SQLite** | Default per-scenario relational store (`${SCENARIO_DATA_DIR}/<scenario>.db`). | Routed via `*database.RoutedDB`; a per-run test pool installs on the live process. |
| **Postgres** | Shared relational engine for scenarios that genuinely need it. | Same routed seams; a Postgres-fit scenario is a Postgres→SQLite advisor candidate. |
| **Qdrant** | Vector collections. | Must use the variant-aware `storage.Collection(domain)` namespace, never a hard-coded collection name. |
| **Redis** | Keys / prefixes. | Must use `storage.RedisKey` / `storage.RedisPrefix`, never a hard-coded prefix. |
| **file** | Filesystem data dirs. | Routed through `filerouting.RoutedRoots`; config is seeded into a leased temporary root while data, cache, logs, and state start empty. |

`storage-health` itself uses **SQLite only**, dogfooding every convention
below.

## The api-core namespace seams

Variant-aware namespacing is what keeps shadow and live instances of the
same scenario from colliding. The seams live in
`packages/api-core/storage`:

- `storage.Collection(domain)` — Qdrant collection name, namespaced.
- `storage.RedisKey(...)` / `storage.RedisPrefix(...)` — Redis key/prefix, namespaced.
- `storage.ScenarioNamespace(...)` — the namespace token shared by all engines.
- `VROOLI_STORAGE_NAMESPACE` — the environment variable that carries the
  active namespace (e.g. a `@shadow` variant) into the running process.

A hard-coded collection/prefix bypasses this and is shadow-unsafe →
`STORAGE_NAMESPACE_HARDCODED` (L2). The SQL and file-isolation seams
(`database.Open → *RoutedDB`, `EnsureSchemas`, `TestModeMiddleware`,
`devrouting.RegisterWithFileRoots`, and context-aware `RoutedRoots.Pick`) are documented in full in
`test-isolation-contract.md`.

## Per-domain embedded idempotent schema

The substrate convention storage-health enforces:

- **Per-domain, embedded.** Each domain owns its schema next to its code
  (`api/internal/<domain>/schema.sql` + `schema.go`); there is no central
  schema file. Violations: `SCHEMA_CENTRALIZED` (L1), `SCHEMA_NOT_PER_DOMAIN`
  (L3, gated on ≥2 code-facts domains).
- **Idempotent, CREATE-only.** Schema is re-run-safe (`CREATE ... IF NOT
  EXISTS`); no non-idempotent DDL (`SCHEMA_NOT_IDEMPOTENT`, L1) and no
  `ALTER` in embedded schema (`SCHEMA_HAS_ALTER`, L1 — `ALTER` belongs in a
  migration).
- **Wired.** `database.EnsureSchemas` applies the embedded schema at
  startup; missing wiring is `ENSURE_SCHEMAS_NOT_WIRED` (L1).
- **System schema empty.** No tables in the system schema
  (`SYSTEM_SQL_NOT_EMPTY`, L3); no cross-domain hard foreign keys
  (`CROSS_DOMAIN_HARD_FK`, L3).

Persistence access then follows the seam conventions — no raw `sql.Open`,
no routed-driver imports outside api-core, no captured `*sql.DB` handles,
rows always closed, no direct SQL in handlers, no SQLite single-connection
deadlock (the L3 persistence-hygiene findings).

## Maturity ladder (L0–L4)

Summarized from `.vrooli/maturity.json` (`dimension: storage`,
`provider: storage-health`). Every finding maps to a `local_level_impact`
rung:

| Rung | Name | Meaning | Representative findings |
|---|---|---|---|
| **L0** | Storage unresolvable | The target can't be resolved or its storage surface classified (service.json missing/invalid, API surface unreadable). | `STORAGE_TARGET_UNRESOLVABLE` (advisory) |
| **L1** | Substrate present & idempotent | Engines recognized; embedded schema is per-domain, CREATE-only, idempotent, ALTER-free. | `SCHEMA_CENTRALIZED`, `SCHEMA_NOT_IDEMPOTENT`, `SCHEMA_HAS_ALTER`, `ENSURE_SCHEMAS_NOT_WIRED` |
| **L2** | Isolation-safe | **The safety rung.** SQL and file test-isolation seams are statically proven and namespaces are variant-aware (shadow-safe). A fail-closed precondition for destructive test-genie playbooks. | `ROUTED_SEAMS_UNWIRED`, `FILE_ROUTED_SEAMS_UNWIRED` (`safety_blocker`), `STORAGE_NAMESPACE_HARDCODED`, `STORAGE_ISOLATION_UNVERIFIED`, `FILE_ISOLATION_UNVERIFIED` |
| **L3** | Persistence hygiene | Persistence follows the seam conventions; no raw `sql.Open`, routed-driver imports, captured handles, unclosed rows, direct SQL in handlers, or SQLite pool deadlock. | `RAW_SQL_OPEN`, `ROUTED_DRIVER_IMPORT`, `SQL_DB_HANDLE_CAPTURE`, `DB_ROWS_NOT_CLOSED`, `DIRECT_SQL_IN_HANDLERS`, `SQLITE_POOL_DEADLOCK`, plus the L3 schema findings |
| **L4** | Operational readiness | *Advisory.* Data-persisting scenarios have a registered backup/restore target and carry no outstanding migration debt. Gaps inform, they do not block. | `BACKUP_TARGET_MISSING`, `MIGRATION_DEBT` |

Severities and `global_impact` (`safety_blocker` / `evolvability_gap` /
`advisory`) and `fix_class` (`auto` vs `manual`) are authoritative in
`.vrooli/maturity.json`; this table is a summary, not the source of truth.

## Storage stage (greenfield signal)

A heuristic `storage_stage` is derived from the `maturity` field in
`.vrooli/service.json` (`greenfield` / `pilot` / `production` / `sunset`,
default `greenfield`) plus presence of a committed `migrations/`
directory. It is **advisory and operator-overridable**: it informs
migration-debt findings (informational) but never hard-fails an
unambiguous violation. "Not yet deployed ⇒ greenfield ⇒ no migrations
expected."

## Cross-References

- `test-isolation-contract.md` — the routed-DB isolation contract (safety core).
- `overview.md` — concept index.
- `.vrooli/maturity.json` — authoritative ladder + finding catalog.
- `../../PRD.md` — operational targets.
