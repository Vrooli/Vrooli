# Architecture — Network Manager

Network Manager is a local-first control plane for diagnosing, filtering, and safely improving home and small-office networks. It uses the generated React/Vite scenario shape but replaces the starter product domain with network-specific domains and adapter contracts.

## Purpose Of This Document

This document explains the scenario shape, product boundaries, contract flow, shared infrastructure, extension rules, maturity, intentional deviations, and documentation architecture.

Detailed domain ownership lives in [`DOMAINS.md`](DOMAINS.md). Workflows live in [`FLOWS.md`](FLOWS.md). Data and retention live in [`DATA.md`](DATA.md). Resource and scenario dependencies live in [`INTEGRATIONS.md`](INTEGRATIONS.md). Test seams live in [`../internal/SEAMS.md`](../internal/SEAMS.md). Test strategy lives in [`../internal/TESTING.md`](../internal/TESTING.md).

## Scenario Shape

```
UI / CLI / Home Automation
        │
        ▼
Network Manager API
        │
        ├── health snapshots
        ├── device inventory
        ├── policy + filtering
        ├── optimization experiments
        ├── approval + rollback
        └── adapter registry
                │
                ├── host adapters: linux, darwin, windows, manual
                ├── resolver adapters: AdGuard Home first, Pi-hole later, Technitium later
                └── router adapters: manual P0, one explicit P1 platform
```

The API owns business rules. The UI and CLI translate operator intent into API calls. Home Automation consumes Network Manager actions and events; it does not own network state. Adapters perform platform-specific reads/writes only after reporting capabilities and required privileges.

## System Boundaries

Network Manager owns:

- network health snapshot definitions and reports,
- resolver policy state and adapter abstraction,
- LAN device inventory and identity-confidence notes,
- optimization experiment plans, scores, approvals, and rollback records,
- Home Automation action/event contracts,
- privacy and retention policy for network metadata.

Network Manager does not own:

- Home Automation device orchestration,
- AdGuard Home, Pi-hole, Technitium, or router implementations,
- ISP account management,
- TLS interception or hidden endpoint surveillance,
- generic network developer tooling unless it is required by this product.

## Contracts And Data Flow

Owned API contracts should use proto/Connect-RPC. REST exceptions are reserved for operational probes or externally dictated shapes. The central product data flow is:

1. Adapter capabilities are discovered.
2. Health snapshot probes collect comparable measurements.
3. Policies and optimization candidates are planned against capabilities.
4. Operators preview and approve persistent changes.
5. Adapter applies the change and returns rollback information.
6. The experiment ledger records before/after evidence.
7. Home Automation receives events for degraded network state or user-facing controls.

## Shared Infrastructure

The generated scenario infrastructure remains valid:

- API module composition and health plumbing stay shared.
- UI design tokens, i18n wiring, accessibility primitives, and feature-folder conventions stay shared.
- CLI command scaffolding stays a thin API wrapper.
- Portable storage should follow the template and `api-core/storage` guidance.

Product vocabulary belongs in product domains, not shared folders.

## Extension Rules

Add new capabilities by extending the owning domain first:

1. Update `PRD.md` and `requirements/` when the capability changes scope.
2. Update [`DOMAINS.md`](DOMAINS.md) before code is added.
3. Add proto contracts for owned operations.
4. Add API service logic and test fakes.
5. Add CLI/UI consumers without duplicating business logic.
6. Add Home Automation events/actions only through the integration domain.
7. Update [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md), and [`INTEGRATIONS.md`](INTEGRATIONS.md) when the change touches state, workflow, or external systems.

## Architecture Maturity

Current maturity: oriented scaffold with a complete product charter and requirements registry. Product implementation has not started.

The first real implementation slice should be read-only health snapshots and adapter capability discovery. Persistent network changes should not be implemented until approval and rollback records exist.

## Intentional Deviations

- The generated notes/example domain is not product scope and should be removed during implementation.
- P0 intentionally excludes router writes. Unsupported router platforms receive diagnostics and manual instructions.
- AdGuard Home is selected as the first resolver adapter; Pi-hole and Technitium are deferred.
- Network Manager is greenfield. The older `network-tools` live scenario has been retired; git history remains the archive if future agents need to inspect prior requirements or code.

## Documentation Architecture

- [`DOMAINS.md`](DOMAINS.md): bounded contexts.
- [`FLOWS.md`](FLOWS.md): temporal workflows and approval/rollback state.
- [`DATA.md`](DATA.md): persisted state, retention, privacy.
- [`INTEGRATIONS.md`](INTEGRATIONS.md): resolver, router, OS, and scenario dependencies.
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md): operator procedures.
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md): durable tradeoffs.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md)
- [`FLOWS.md`](FLOWS.md)
- [`DATA.md`](DATA.md)
- [`INTEGRATIONS.md`](INTEGRATIONS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../internal/TESTING.md`](../internal/TESTING.md)
