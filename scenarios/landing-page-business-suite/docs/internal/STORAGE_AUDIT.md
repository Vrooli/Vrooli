# Storage Architecture Audit

## Last Updated

2026-09-07

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

The new `desktoplink` domain follows the same substrate: its declarative schema
is embedded beside the repository that interprets it, and `SQLRepository` is the
only production persistence seam. The domain stores link metadata and audit
events, never LPBS website credentials or raw authorization codes.

The current `storage-manager validate scenario landing-page-business-suite`
run is not clean: it reports 26 findings, including pre-existing open-row,
direct-writer, direct-SQL-in-handler, private-key-path, and accountability
findings elsewhere in the scenario. Those findings are outside the desktop-link
change boundary and remain tracked for the scenario's broader storage cleanup.

The `businessaccount` domain now follows the same per-domain substrate. Its
declarative account and membership tables live beside the repository that
interprets them; transport handlers use only the repository contract. Account
membership is the authorization boundary for desktop-link account selection,
and the default personal account is created idempotently for existing users.

## Follow-up

The desktop-link tables are greenfield declarative tables in this change. Before
the first production deployment with customer data, use the approved brownfield
migration substrate if schema evolution is required; do not add ad-hoc migration
logic to the scenario.
