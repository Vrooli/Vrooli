# Data — Unit Health

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Persistence is embedded SQLite through `modernc.org/sqlite`. The
database path is resolved from the scenario id by `api-core/storage`,
and the API applies domain schemas on startup through
`api-core/database` (`EnsureSchemas`).

Unit Health also keeps a filesystem evidence cache under the scenario
cache directory resolved by `api-core/storage`
(`<cache-dir>/validation-evidence`). It is regenerable: a missing or
unreadable cache disables reuse and the run proceeds without it.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Validation runs (`unit_runs`) | validation (`runhistory`) | SQLite | `api/internal/runhistory/schema.sql` | Newest 50 runs per scenario; older runs pruned on each `Record` | Run id, scenario, start time, status, maturity rung. |
| Command outcomes (`unit_run_commands`, `unit_run_command_identity`) | validation (`runhistory`) | SQLite | `api/internal/runhistory/schema.sql` | Same as parent run | Per-workspace command duration, status, failure class; optional identity JSON for comparability. |
| Coverage samples (`unit_run_coverage`) | validation (`runhistory`) | SQLite | `api/internal/runhistory/schema.sql` | Same as parent run | Per-file coverage percent; feeds runtime-growth and regression diagnostics. |
| Native test observations (`unit_run_native_tests`) | validation (`runhistory`) | SQLite | `api/internal/runhistory/schema.sql` | Same as parent run | Runner's final observation JSON; separate from command-level reliability cohorts. |
| Evidence cache | validation (`evidence`) | Filesystem | `api/internal/evidence/store.go` | Bounded to 512 MiB and 24 h, evicted by the store | Keyed by target and input fingerprints; reuse is reported on the response (`cache_hit`, `cache_miss_reason`). |
| Test-quality rule catalog | validation (`testquality`) | Checked-in JSON | `api/internal/testquality/catalog.json` | Source, not data | The rules; loaded at boot, never written at runtime. |
| Maturity spec | validation | Checked-in JSON | `maturity` block of `.vrooli/test-genie.json` | Source, not data | Capability ladders read through `maturity-go/assessment`. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `unit_runs`, `unit_run_commands`, `unit_run_coverage`, `unit_run_command_identity`, `unit_run_native_tests` | validation / runhistory | `api/internal/runhistory/schema.sql` | `runhistory.Repository`; diagnostics and reliability analyzers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup (ships empty by intent) |

## Migrations And Compatibility

Schema bootstrap is idempotent and forward-only. Domain schema files
use `CREATE TABLE IF NOT EXISTS` and live beside the code that
interprets them. `scenario` and `started_at` are denormalized into the
child tables so history queries are single `SELECT`s, which keeps them
safe under the single-connection pool.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Run history | Each new `Record` for the same scenario | Keep newest 50 runs (`runhistory.DefaultRetention`), cascade to child rows | No operator-facing purge command. |
| Evidence cache | Store eviction | `num[threshold]:512` MiB / `num[threshold]:24` h bounds enforced by `evidence.Store` | No manual invalidation besides removing the directory. |

## Privacy Notes

Persisted data is test metadata about repository targets: command
lines, durations, statuses, file paths, and coverage percentages. No
personal, customer, or financial data is stored. Captured command
output excerpts can include whatever a test prints; treat the cache
directory and SQLite file as local development data. If that changes,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
