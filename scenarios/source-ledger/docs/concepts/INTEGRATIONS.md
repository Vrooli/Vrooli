# Integrations — Source Ledger

## Scenario Dependencies

| Dependency | Use | Failure behavior |
|---|---|---|
| `ai-gateway` | Classification, embeddings, and compaction summaries. | Journal append remains available; enrichment and compaction report bounded failure. |
| `search-hub` | One federated provider per scope. | Local CLI/API remain available; provider registration is retried and visible. |
| `data-backup-manager` | Non-regenerable journal backup and restore drills. | The scenario must not claim production readiness without a verified backup. |

SQLite is in-process and is not a resource dependency. The source ledger owns
its database path through the storage resolver and registers that same path as
the backup target.

## Consumer Boundary

`vrooli-memory` is the first consumer. It retains harness-specific import,
projection, prompt, and capture behavior. It does not retain a local read
replica of a ledger scope. A ledger outage returns a typed unavailable result
for live recall while existing projected wake blocks remain intact.

## Discovery and Registration

Resolve the source-ledger endpoint once at consumer startup through Vrooli
discovery. Cache the result for requests. Register one search-hub provider per
scope with an explicit scope in its request template.

## Security and Data Rules

Unified read across scopes is intentional for this product. This phase does
not add access-control partitioning. All mutation paths remain append-safe,
scope-aware, and subject to the journal high-water-mark guard.

## Architecture Maturity

Integration intent is recorded before implementation: search-hub and governed
inference are planned scenario dependencies, while backup registration is
introduced at corpus migration when a non-regenerable ledger database exists.

## Contracts And Data Flow

External integrations receive explicit scope-aware requests and return bounded
failure states; they never become the journal authority.

## Purpose Of This Document

This document records planned dependencies and their degraded behavior.

## Dependency Inventory

Governed inference, search-hub, and data-backup-manager are planned integration
surfaces; none is started by the contract-only scaffold.

## Vrooli Resources

No local resource is required before the service contract exists; SQLite is
in-process and owned through the storage resolver.

## Third-Party Services

No unmanaged third-party service is introduced in this phase.

## Failure Modes

Dependency failure must pause derived enrichment or federation visibly while
leaving append and local authority semantics explicit.

## Cross-References

- [`../reference/configuration.md`](../reference/configuration.md)
- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DATA.md`](DATA.md)
