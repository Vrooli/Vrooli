# Integrations — Source Ledger

## Scenario Dependencies

| Dependency | Use | Failure behavior |
|---|---|---|
| `ai-gateway` | Classification, embeddings, and compaction summaries. | Journal append remains available; enrichment and compaction report bounded failure. |
| `search-hub` | One scope-faceted federated provider. | Local CLI/API remain available; provider registration is retried and visible. |
| `data-backup-manager` | Planned backup/restore integration; not declared or wired in the current runtime. | No backup target is registered by Source Ledger today; production backup readiness requires a separately verified procedure. |

SQLite is in-process and is not a resource dependency. The source ledger owns
its database path through the storage resolver. Backup-target registration is
future work; this guide does not claim a `data-backup-manager` integration is
active.

## Consumer Boundary

`vrooli-memory` is the first consumer. It retains harness-specific import,
projection, prompt, and capture behavior. It does not retain a local read
replica of a ledger scope. A ledger outage returns a typed unavailable result
for live recall while existing projected wake blocks remain intact.

## Discovery and Registration

Source Ledger owns the federated descriptor at `.vrooli/search.json`. It
registers a single `source-ledger.scopes` provider with Search Hub. The router
passes the requested scope as a query facet; an empty scope selects the
agent-memory ledger. New scopes use the same provider and do not require
registry mutations.

## Security and Data Rules

Unified read across scopes is intentional for this product. This phase does
not add access-control partitioning. All mutation paths remain append-safe,
scope-aware, and subject to the journal high-water-mark guard.

## Architecture Maturity

Governed inference and Search Hub are live integrations; Search Hub is optional.
Backup registration remains a separate production-readiness surface for the
non-regenerable ledger database.

## Contracts And Data Flow

External integrations receive explicit scope-aware requests and return bounded
failure states; they never become the journal authority.


This document records declared runtime dependencies and distinguishes planned
integrations from wired dependencies.

## Dependency Inventory

Governed inference and Search Hub are active runtime surfaces. The
`data-backup-manager` remains a planned backup-drill owner, but there is no manifest dependency
or runtime integration yet. It is not required for normal append or recall.

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
