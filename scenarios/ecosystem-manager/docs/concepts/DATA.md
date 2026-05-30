# Data — Ecosystem Manager

This document is the canonical data ownership and storage map for
ecosystem-manager. Update it when domains add tables, files, blobs,
retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does ecosystem-manager persist?
- Which domain owns each data shape?
- Where is the source of truth — PostgreSQL or the filesystem?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Ecosystem-manager uses **two** storage substrates, not one.

| Substrate | What lives there | Source of truth for |
|---|---|---|
| PostgreSQL — database `vrooli_ecosystem_manager` | Run history, daily metric aggregates, live auto-steer/steering execution state, execution feedback | Anything *produced by* a run (history, in-flight state, metrics) |
| Filesystem stores | Auto-steer profiles, the task queue, system logs | Anything *human-authored or operationally durable* (config + queue + logs) |

The Postgres schema is defined in
[`initialization/postgres/schema.sql`](../../initialization/postgres/schema.sql)
and applied idempotently on startup. The split is deliberate: profiles and
the queue are version-controllable, hand-editable config that intentionally
stays **out of the database** so they can be reviewed, diffed, and committed.

[CODE: `initialization/postgres/schema.sql`] defines all tables, triggers,
and views. [CODE: `profiles/metadata.json`] indexes filesystem profiles.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| `task_executions` | analytics | Postgres | `schema.sql` | Indefinite | One row per task run; timing, status, PRD before/after. |
| `operation_metrics` | analytics | Postgres | `schema.sql` | Indefinite | Daily-bucketed aggregation; maintained by trigger `update_operation_metrics()`, never written by app code directly. |
| `profile_executions` | auto-steer | Postgres | `schema.sql` | Indefinite | UUID PK, one per task (`UNIQUE(task_id)`); `start_metrics`/`end_metrics`/`phase_breakdown` JSONB, `total_iterations`, `user_rating`/`user_comments`. |
| `profile_execution_state` | auto-steer | Postgres | `schema.sql` | Until execution completes (then folded into history) | `task_id` PK; `current_phase_index`, `current_phase_iteration`, `auto_steer_iteration`, `phase_history`/`metrics`/`phase_start_metrics` JSONB; `last_updated` maintained by trigger. |
| `steering_queue_state` | steering | Postgres | `schema.sql` | Until queue drains | `task_id` PK; `queue` JSONB (ordered steer-mode strings), `current_index`. |
| `execution_feedback_entries` | insights | Postgres | `schema.sql` | Indefinite | UUID PK; `category`/`severity`/`suggested_action`/`comments`/`metadata`; indexed by `execution_task_id`. |
| Auto-steer profiles | auto-steer | Filesystem | `profiles/<id-or-name>/profile.json` | Until deleted by an operator | Human-authored, version-controlled config; indexed by `profiles/metadata.json`. Intentionally NOT in the DB. |
| Task queue | tasks | Filesystem | `queue/<status>/` (YAML) | Until task is purged | Directory name *is* the status; status transitions are atomic file moves between directories. |
| System logs | operations | Filesystem | `logs/<date>.log` | Indefinite (no rotation today) | Date-stamped operational logs. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `task_executions` | analytics | `schema.sql` | analytics repository/handlers; insert trigger feeds metrics |
| `operation_metrics` | analytics | `schema.sql` | metrics views/queries; written only by trigger |
| `profile_executions` | auto-steer | `schema.sql` | auto-steer history service |
| `profile_execution_state` | auto-steer | `schema.sql` | auto-steer controller (live state) |
| `steering_queue_state` | steering | `schema.sql` | steering queue runner |
| `execution_feedback_entries` | insights | `schema.sql` | execution-feedback handlers |
| `task_execution_summary` (view) | analytics | `schema.sql` | reporting/UI stats |
| `recent_task_activity` (view) | analytics | `schema.sql` | dashboard recent-activity feed |
| `profile.json` + `metadata.json` | auto-steer | `profiles/` | profile loader/editor |
| queue YAML | tasks | `queue/<status>/` | task queue manager |

## Migrations And Compatibility

The Postgres schema is **idempotent bootstrap**, applied on every startup.
All tables use `CREATE TABLE IF NOT EXISTS`; triggers/functions/views use
`CREATE OR REPLACE` (functions/views) and `DROP TRIGGER IF EXISTS` +
`CREATE TRIGGER` (triggers). Two triggers are load-bearing:

- `trigger_update_operation_metrics` (`BEFORE INSERT OR UPDATE` on
  `task_executions`) computes `duration_minutes` and upserts the daily
  `operation_metrics` row.
- `trigger_profile_execution_state_updated` keeps
  `profile_execution_state.last_updated` current.

There is **no formal migration tool**. Column drops, renames, or
backfills are manual and must be documented in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.
Because bootstrap is `IF NOT EXISTS`, adding a column to an existing
deployment is a manual `ALTER TABLE` — the schema file alone will not
alter a table that already exists.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `profiles/<id-or-name>/profile.json` | JSON | auto-steer | Profiles are git-tracked files; "import/export" is copying the directory. |
| `queue/<status>/*.yaml` | YAML | tasks | Queue entries are portable files; moving a file changes its status. |
| Run history / metrics | n/a | analytics | No structured export endpoint today. Add when a reporting requirement appears. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| `task_executions` / `operation_metrics` | Manual only | Retained indefinitely for analytics/learning | **No auto-purge.** Tables grow unbounded; a retention policy is not yet implemented. |
| `profile_executions` | Manual only | Indefinite | No auto-purge. |
| `profile_execution_state` / `steering_queue_state` | Completion of the run/queue | Transient by nature | Orphaned rows from interrupted runs are not garbage-collected automatically. |
| `execution_feedback_entries` | Manual only | Indefinite | No auto-purge. |
| Profiles | Operator deletes the directory | Until deleted | n/a |
| System logs | Manual only | Indefinite | **No log rotation.** `logs/` grows without bound. |

## Privacy Notes

Ecosystem-manager stores **no end-user PII**. Its data is about Vrooli's
*own* scenarios and resources plus agent run metadata: operation types,
target scenario/resource names, durations, statuses, PRD completion
percentages, phase metrics, and operator-authored profiles/feedback. The
`user_rating`/`user_comments` fields on `profile_executions` are operator
feedback about a run, not third-party personal data. If a future domain
ever stores personal, regulated, or customer data, update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) before that work lands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — schema-change record
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
