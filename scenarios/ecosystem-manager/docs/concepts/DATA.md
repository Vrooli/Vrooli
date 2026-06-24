# Data — Ecosystem Manager

This document is the canonical data ownership and storage map for
ecosystem-manager. Update it when domains add tables, files, blobs,
retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does ecosystem-manager persist?
- Which domain owns each data shape?
- Where is the source of truth — SQLite or the filesystem?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Ecosystem-manager persists **all runtime state through
`github.com/vrooli/api-core/storage`**, never inside the scenario source
tree. The single production seam that resolves storage locations is
[CODE: `api/pkg/storagepaths`]; it builds a `storage.Resolver` and the
variant-aware `storage.ScenarioNamespace("ecosystem-manager")`, so a
Baseline Modes shadow engagement (which injects `VROOLI_STORAGE_NAMESPACE`)
lands every storage class beside `ecosystem-manager_shadow` and never shares
live's database, queue, or logs.

| Substrate | Storage class | What lives there | Source of truth for |
|---|---|---|---|
| **SQLite** — `<data-root>/vrooli/<namespace>/ecosystem-manager.db` | `ClassData` | Run history, live auto-steer/steering execution state, per-iteration decision traces | Anything *produced by* a run (history, in-flight state, metrics) |
| **Filesystem queue** — `<data-root>/vrooli/<namespace>/queue/<status>/*.yaml` | `ClassData` | The task queue | Task lifecycle (status = directory) |
| **System logs** — `<logs-root>/vrooli/<namespace>/` | `ClassLogs` | Audit log + per-task-run execution logs (`task-runs/`) | Operational history |
| **Settings** — `<config-root>/vrooli/<namespace>/settings.json` | `ClassConfig` | Mutable operator settings | Runtime configuration |
| **Profiles** — `profiles/<id-or-name>/` (scenario tree) | n/a (source) | Auto-steer profile JSON + `metadata.json` | Human-authored, version-controlled config |

The split is deliberate: **profiles and prompts are source assets** that stay
git-tracked in the scenario tree so they can be reviewed, diffed, and
committed; **everything a run produces is runtime state** that lives under the
storage root and is covered by data-backup-manager.

SQLite is opened through [CODE: `api/pkg/storagepaths`]`.SQLiteDSN()` (WAL,
`foreign_keys=ON`, `busy_timeout=10000`, single-writer pool) and the
domain-owned schemas are applied at boot by `database.EnsureSchemas` via
[CODE: `api/pkg/dbschema`]`.AllSchemas()`. There is no Postgres dependency.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| `profile_executions` | auto-steer | SQLite | `api/pkg/autosteer/schema.sql` | Indefinite | Text PK (app-generated UUID), one per task (`UNIQUE(task_id)`); `phase_breakdown` JSON text, `total_iterations`, `total_duration_ms`. |
| `profile_execution_state` | auto-steer | SQLite | `api/pkg/autosteer/schema.sql` | Until execution completes (then folded into history) | `task_id` PK (objective-controller shape); `iteration`, `current_skill`/`current_rationale`, `findings`/`score_history`/`trace`/`metrics` JSON text; `last_updated` maintained in application code. |
| `decision_trace` | auto-steer | SQLite | `api/pkg/autosteer/schema.sql` | Indefinite | One row per controller iteration; persists reasoning after the live `state.Trace` is dropped on finalize. |
| `steering_queue_state` | steering | SQLite | `api/pkg/steering/schema.sql` | Until queue drains | `task_id` PK; `current_index` plus RFC3339 `created_at`/`updated_at`. |
| Auto-steer profiles | auto-steer | Filesystem (scenario tree) | `profiles/<id-or-name>/profile.json` | Until deleted by an operator | Human-authored, version-controlled config; indexed by `profiles/metadata.json`. Intentionally NOT in the DB and NOT under the storage root. |
| Task queue | tasks | Filesystem (`ClassData`) | `<data-root>/…/queue/<status>/` (YAML) | Until task is purged | Directory name *is* the status; transitions are atomic file moves. |
| System logs | operations | Filesystem (`ClassLogs`) | `<logs-root>/…/<date>.log` + `task-runs/` | Indefinite (no rotation today) | Date-stamped audit log and per-task-run execution logs. |
| Settings | settings | Filesystem (`ClassConfig`) | `<config-root>/…/settings.json` | Until overwritten | Mutable runtime settings persisted on change. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `profile_executions` | auto-steer | `api/pkg/autosteer/schema.sql` | auto-steer history service |
| `profile_execution_state` | auto-steer | `api/pkg/autosteer/schema.sql` | auto-steer controller (live state) |
| `decision_trace` | auto-steer | `api/pkg/autosteer/schema.sql` | decision-trace store |
| `steering_queue_state` | steering | `api/pkg/steering/schema.sql` | steering queue runner |
| `profile.json` + `metadata.json` | auto-steer | `profiles/` | profile loader/editor |
| queue YAML | tasks | `<data-root>/…/queue/<status>/` | task queue manager |

## Migrations And Compatibility

Schema ownership is **per-domain and declarative**: each domain ships a
`schema.sql` next to its code and a `Schema()` function; the central
[CODE: `api/pkg/dbschema`]`.AllSchemas()` registry orders them and
`database.EnsureSchemas` applies them on every boot. All tables use
`CREATE TABLE IF NOT EXISTS`, so re-application is a no-op. There are no
triggers, functions, or views — invariants that previously lived in PL/pgSQL
(e.g. `last_updated` maintenance) are now enforced in application code.

`EnsureSchemas` runs a post-apply drift check on SQLite (`PRAGMA table_info`
vs the declared columns) and **fails boot loudly** if a pre-existing table is
missing a declared column. Because `CREATE TABLE IF NOT EXISTS` cannot add a
column to a table that already exists, **any change to an existing table's
columns is a one-shot migration, not a declarative edit** — see the
`storage-steer` skill §5. Document column drops/renames/backfills in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

This was a hard cut-over from PostgreSQL: there is no Postgres compatibility
mode, no dual-read, and no importer for old repo-local queue YAML.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `profiles/<id-or-name>/profile.json` | JSON | auto-steer | Profiles are git-tracked files; "import/export" is copying the directory. |
| `<data-root>/…/queue/<status>/*.yaml` | YAML | tasks | Queue entries are portable files; moving a file changes its status. |
| Run history / metrics | n/a | auto-steer | No structured export endpoint today. Add when a reporting requirement appears. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| `profile_executions` / `decision_trace` | Manual only | Indefinite for analytics | **No auto-purge.** Tables grow unbounded; a retention policy is not yet implemented. |
| `profile_execution_state` / `steering_queue_state` | Completion of the run/queue | Transient by nature | Orphaned rows from interrupted runs are not garbage-collected automatically. |
| Profiles | Operator deletes the directory | Until deleted | n/a |
| System logs | Manual only | Indefinite | **No log rotation.** The logs dir grows without bound. |

## Privacy Notes

Ecosystem-manager stores **no end-user PII**. Its data is about Vrooli's
*own* scenarios and resources plus agent run metadata: operation types,
target scenario/resource names, durations, statuses, PRD completion
percentages, phase metrics, and operator-authored profiles. If a future domain
ever stores personal, regulated, or customer data, update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) before that work lands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — schema-change record
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
