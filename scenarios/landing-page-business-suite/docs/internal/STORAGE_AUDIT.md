# Storage Architecture Audit

## Last Updated

2026-07-27

## Current Posture

The scenario uses the lifecycle-routed SQL database. Its schema is split by
domain beneath `api/internal/` and is wired through the supported schema
registry. This scenario remains pre-launch; declarative, idempotent schemas are
the appropriate migration strategy until persisted production customer data
exists.

## Measures Persistence Seam

Measures are read-only aggregates over authoritative domain tables. The HTTP
and measure-registry layer owns declarations, time-window validation, and
transport error handling. `api/internal/measures.SQLRepository` owns the closed
catalog of fixed count queries and their execution.

The repository accepts a measure name only; it does not expose a table-name or
arbitrary-SQL input. This preserves the static-review and injection-safety
properties while keeping persistence out of handlers. Tests assert catalog
coverage for every supported measure, and handler tests verify registry and
Connect paths produce the same aggregate.

## Validation

`storage-manager validate scenario landing-page-business-suite --json` reports
L3 (clean) for schema substrate, isolation safety, and persistence hygiene.

## Follow-up

Revisit the migration strategy before the first production deployment with
customer data. If preserving deployed data requires schema evolution, use the
approved brownfield migration substrate rather than adding ad-hoc migration
logic here.
