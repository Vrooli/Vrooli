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

Source Ledger owns the federated descriptor at `.vrooli/search.json`. At boot
it materializes one provider for every scope in the policy registry, then
self-registers all descriptors with Search Hub. A newly-created scope updates
the descriptor and triggers the same bounded registration path. The stable
agent scope is `source-ledger.agent-memory`; other scopes use
`source-ledger.scope.<scope-id>`. Every request template carries its scope
explicitly, so federation cannot cross ledgers accidentally.

## Security and Data Rules

Unified read across scopes is intentional for this product. This phase does
not add access-control partitioning. All mutation paths remain append-safe,
scope-aware, and subject to the journal high-water-mark guard.

## Architecture Maturity

Search Hub and governed inference are live optional integrations. Backup
registration remains a separate production-readiness surface for the
non-regenerable ledger database.

## Contracts And Data Flow

External integrations receive explicit scope-aware requests and return bounded
failure states; they never become the journal authority.

## Purpose Of This Document

This document records planned dependencies and their degraded behavior.

## Dependency Inventory

Governed inference and Search Hub are active runtime surfaces. Data-backup-
manager remains the owner of the backup drill and is not required for normal
append or recall.

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
