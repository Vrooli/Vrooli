# Data — Template Manager

Template Manager persists template governance evidence. Template content stays in `templates/`; the scenario stores catalog metadata, validation evidence, debt state, drift snapshots, guidance state, and monitor status.

## Storage Overview

The API uses the api-core storage resolver with SQLite WAL under `~/.vrooli/data/vrooli/template-manager/`. Schema changes are migration-owned and must never recreate the store. Domain repositories own SQL access; handlers and CLI code call services rather than issuing SQL directly.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Template records | registry | SQLite | Template manifests under `templates/` plus registry migrations | Until template content is removed or superseded. | Covers scenario templates, design kits, and resource templates. |
| Validation runs | validation | SQLite | Validation runner/service output | Keep historical runs for trend windows; cleanup policy is a Template Manager operation. | Includes shallow, deep, drift, and scheduler-attributed runs. |
| Validation findings | validation | SQLite | Parsed validation output | Same lifecycle as owning run. | Findings are immutable run evidence; debt entries are the deduplicated operational state. |
| Debt entries | debt | SQLite | Seed data plus mapped findings | Kept as open or resolved history; never silently deleted. | Stable keys deduplicate repeated findings; terminal Test Genie deep-validation summaries are superseded by the next terminal deep run. |
| Drift snapshots | validation | SQLite | Drift engine output | Historical trend window plus latest snapshot per target. | Used by standing, dashboard, and measures. |
| Version-lag records | registry | SQLite | Template CHANGELOG and target generation provenance | Recomputed on validation; latest state retained. | Advisory above L0. |
| Orientation gate definitions | guidance | Template manifests and optional cached normalized rows | `templates/**/template.json` | Follows template manifest lifecycle. | Template Manager owns schema and evaluator; templates own declarative data. |
| Guidance evaluations | guidance | SQLite or computed response cache | Guidance service | Short-lived; can be recomputed from target files. | Avoid storing stale work-order data as truth. |
| Monitor status | monitor | SQLite | Scheduler service | Latest status plus historical scheduler run links. | Tracks last run, next run, in-flight state, and streak. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| template_records | registry | `api/internal/registry/migrations/` | Registry service, validation, dashboard, measures |
| validation_runs | validation | `api/internal/validation/migrations/` | Run history API/CLI/UI |
| validation_findings | validation | `api/internal/validation/migrations/` | Debt mapper, run detail UI |
| debt_entries | debt | `api/internal/debt/migrations/` | Debt API/CLI/UI, measures |
| drift_snapshots | validation | `api/internal/validation/migrations/` | Standing, drift reports, dashboard |
| monitor_state | monitor | `api/internal/monitor/migrations/` | Scheduler status API/CLI/UI |

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| `templates/**/template.json` | JSON | registry/guidance | Read-only source for template metadata and gate definitions. |
| `.vrooli/service.json` in target scenarios | JSON | phase-provider/guidance | Read/write only for the L0 provenance autofix and orientation evaluation. |
| `.vrooli/search.json` | JSON | docs | Planned provider declaration for docs and debt indexing. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Validation runs/findings | Cleanup operation or retention policy. | Keep enough history for trend UI and measures. | Policy lands with validation run service. |
| Debt entries | Source debt closes through verified remediation; a newer terminal Test Genie deep run supersedes only its prior summary entry. | Preserve resolved history for learning loop. | Stable key and supersession policy are enforced by the validation runner. |
| Monitor status | Scheduler writes latest state. | Latest status plus linked historical runs. | Scheduler lands in Phase 7. |

## Privacy Notes

Template Manager records local repository metadata and generated validation findings. It should not persist arbitrary generated scenario source content, secrets, uploaded user data, or test artifacts beyond stable finding summaries and file references.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by bounded context
- [`FLOWS.md`](FLOWS.md) — run and scheduler lifecycles
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external scenario/platform contracts
