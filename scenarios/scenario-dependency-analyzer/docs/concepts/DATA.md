# Data

## Purpose Of This Document

Describe SDA-owned data, persistence boundaries, and import/export behavior.

## Storage Overview

SDA stores analysis and deployment metadata in SQLite. Actual interface graph responses are assembled from upstream facts and persisted as a derived cache keyed by request/fleet signature; the cache stores no source contents and is rebuildable.

## Data Ownership

SDA owns graph interpretation, the derived interface graph cache, drift records, deployment reports, and bundle metadata. `proto-health` owns proto surface facts; `code-facts` owns language import facts.

Deployment reports for canonical scenario subjects are persisted in that
scenario's data namespace. Dot-prefixed synthetic subjects used by isolated
test runs are request-scoped and are not retained as durable scenario data.
Reports over the 32 MiB read limit are not reusable; when the same synthetic
subject is recomputed, its oversized legacy report is removed by this owner.

## Schema Map

Schema providers live under API domain packages, including `api/internal/store/schema.sql`.

## Migrations And Compatibility

Schema changes should be additive where possible and covered by API tests using in-memory SQLite.

## Import / Export

The CLI and API export graph and DAG data as JSON for downstream scenarios and operators.

## Retention And Deletion

Local SQLite data may be regenerated from scenario manifests and upstream fact services.

## Privacy Notes

SDA reads repository metadata and imports. It should not persist secrets or source contents.

## Cross-References

- `ARCHITECTURE.md`
- `INTEGRATIONS.md`
- `../internal/SECURITY.md`
