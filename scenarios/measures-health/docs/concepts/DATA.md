# Data — Measures Health

This document maps persisted data and its owner. Update it when a domain adds or changes stored data.

## Storage

The API uses embedded SQLite (`modernc.org/sqlite`). The database path is resolved by `api-core/storage`; modules contribute schemas through `api/internal/modules/registry.go`, and startup applies them through `api-core/database`.

## Data ownership

| Data | Owner | Storage and source of truth | Retention |
|---|---|---|---|
| Validation run history | `measures` / `runhistory` | SQLite `validation_runs`; schema in `api/internal/runhistory/schema.sql`, repository in `api/internal/runhistory/repository.go` | Retained until the local database is removed; no per-record deletion or expiry policy is implemented. |

The table records scenario, pass/fail, error and warning counts, and the run timestamp. `MeasuresService` computes the `failed` and `coverage` measures with time-windowed SQL counts over this table.

The validation and search domains do not persist scenario payloads. Search builds its index from scenario manifests and queries the declared measures through their registered endpoints.

## Schema and migrations

`runhistory.Schema()` is registered by the measures module and applied idempotently at API startup. Keep schema next to the repository that interprets it. Use an explicit migration when a change requires backfill, rename, or data-preserving deletion; record the compatibility decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Import, export, and privacy

There is no import/export path. Stored validation history is local scenario metadata; it contains scenario names and aggregate result counts, not user or customer records. Revisit this assessment if stored fields change.

## Cross-references

- [`DOMAINS.md`](DOMAINS.md) — domain ownership
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — security posture
