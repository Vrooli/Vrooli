# Domains — Network Manager

This document maps Network Manager's bounded contexts and product ownership.

## Purpose Of This Document

Use this document to decide which domain owns each concept, table, proto operation, endpoint, CLI command, UI surface, and test seam. System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflows belong in [`FLOWS.md`](FLOWS.md). Storage details belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Refs |
|---|---|---|---|---|---|---|
| snapshot | Produce comparable network health measurements. | Reporting / measurement | Snapshot runs and probe results. | API, CLI, UI | `NM-P0-001` | path:packages/proto/schemas/network-manager/v1/snapshot, path:api/handlers/snapshot, path:cli/domains/snapshot |
| resolver | Manage DNS/filtering backends, AdGuard Home first. | Integration / policy | Resolver config, backend status, adapter capability reports. | API, CLI, UI | `NM-P0-002`, `NM-P1-005`, `NM-P2-001` | path:packages/proto/schemas/network-manager/v1/resolver, path:api/handlers/resolver, path:cli/domains/resolver |
| policy | Preview, apply, pause, resume, and roll back DNS filtering policy. | Policy / command | Policy profiles, allow/deny lists, rollback records. | API, CLI, UI | `NM-P0-003`, `NM-P1-001`, `NM-P1-002` | path:packages/proto/schemas/network-manager/v1/policy, path:api/handlers/policy, path:cli/domains/policy |
| inventory | Track network clients and identity confidence. | Entity / reconciliation | Devices, groups, identifiers, last-seen records. | API, CLI, UI | `NM-P0-004` | path:packages/proto/schemas/network-manager/v1/inventory, path:api/handlers/inventory, path:cli/domains/devices |
| optimization | Run baseline/candidate/after experiments and score candidates. | Workflow / decision support | Experiment runs, measurements, scores, approvals. | API, CLI, UI | `NM-P0-005` | path:packages/proto/schemas/network-manager/v1/optimization, path:api/handlers/optimization, path:cli/domains/optimize |
| adapters | Normalize OS, resolver, router, and manual capabilities. | Platform abstraction | Capability snapshots and unsupported reasons. | API, CLI | `NM-P0-006`, `NM-P1-003`, `NM-P1-004`, `NM-P1-008` | path:packages/proto/schemas/network-manager/v1/adapters, path:api/handlers/adapters, path:cli/domains/adapters |
| homeintegration | Provide actions/events consumed by Home Automation. | Integration contract | Event log and action audit references. | API, CLI, events | `NM-P0-007` | path:packages/proto/schemas/network-manager/v1/home_integration, path:api/handlers/homeintegration, path:cli/domains/home |
| privacy | Apply retention, visibility, and audit-mode rules. | Governance / policy | Retention settings, visibility settings, audit profile. | API, CLI, UI | `NM-P0-008`, `NM-P1-006` | path:packages/proto/schemas/network-manager/v1/privacy, path:api/handlers/privacy, path:cli/domains/privacy |
| monitoring | Schedule recurring checks and detect regressions. | Monitoring / alerting | Monitor schedules, baseline comparisons, alerts. | API, CLI, UI | `NM-P1-007` | path:packages/proto/schemas/network-manager/v1/monitoring, path:api/internal/monitoring, path:api/handlers/monitoring, path:cli/domains/monitoring |

## Domain Details

### snapshot

Collects gateway reachability, WAN reachability, DNS timing, IPv4/IPv6 state, loss, jitter, throughput, and host facts. Snapshot outputs must be repeatable enough to compare baseline/candidate/after runs.

### resolver

Owns backend-neutral resolver concepts and the first AdGuard Home adapter. It should expose capabilities rather than leaking backend-specific quirks into UI or optimization logic.

### policy

Owns DNS filtering policy and operator-visible change plans. Persistent changes must be previewable and reversible.

### inventory

Reconciles LAN-visible clients from resolver, DHCP/router, and host observations. Identity confidence is first-class because randomized MAC addresses and stale hostnames are normal.

### optimization

Coordinates measurement-backed experiments. It chooses candidate configurations, invokes snapshot collection, scores outcomes, and records approval/rollback state.

### adapters

Owns the OS-agnostic control-plane boundary. Host/router/resolver capabilities are reported before an action is offered.

### homeintegration

Publishes actions and events for Home Automation. It is an integration boundary, not a second source of network truth.

### privacy

Centralizes query-log visibility, retention, household defaults, and small-office audit-mode differences.

### monitoring

Persists recurring snapshot schedules, compares fresh snapshots against a baseline snapshot, and records open regression alerts. Current execution is operator-triggered/advisory; autonomous background scheduling can be added later without changing the stored schedule and alert contract.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Capability report | Adapter-declared supported actions, privileges, rollback support, and unsupported reasons. | adapters |
| Health snapshot | Comparable report of measured network quality and local network facts. | snapshot |
| Policy profile | Named DNS/filtering behavior for devices or groups. | policy |
| Device identity confidence | Explanation of how reliable a client identity is. | inventory |
| Experiment ledger | Durable before/candidate/after evidence for optimization. | optimization |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| multi-site | P2 scope; not needed for first home deployment. | Small-office or managed-service use case needs multiple networks. |
| roaming | P2 scope; requires VPN or endpoint-local design. | Need policy off the local network. |
| advanced-analysis | P2 scope; high privacy and platform risk. | Operator explicitly wants topology, packet capture, or security scanning. |

## Non-Domains

- Template example domains are not product scope.
- Home Automation remains a consumer, not a Network Manager subdomain.
- AdGuard Home, Pi-hole, Technitium, OpenWrt, OPNsense, pfSense, and UniFi are external systems behind adapters.
- Generic developer network testing is not a product domain unless it directly supports this scenario's diagnostics.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`FLOWS.md`](FLOWS.md)
- [`DATA.md`](DATA.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
