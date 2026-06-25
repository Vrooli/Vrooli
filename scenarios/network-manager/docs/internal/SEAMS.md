# Seams — Network Manager

## Wire contracts live in proto, not seams

Proto definitions will own the API wire shape. This document records substitutable implementation boundaries, not request/response schema.

## Workflow transitions are not seams

Optimization and policy transitions are modeled in [`../concepts/FLOWS.md`](../concepts/FLOWS.md). A seam is an interface where production uses one implementation and tests use a fake.

## Current seams

| Seam | Production role | Test fake | Requirements |
|---|---|---|---|
| `SnapshotProbeRunner` | Runs DNS, gateway, WAN, loss, jitter, throughput, and host probes. | Deterministic fake probes. | `NM-P0-001`, `NM-P0-005` |
| `ResolverAdapter` | Talks to AdGuard Home first, later Pi-hole/Technitium. Production verifies AdGuard health and previews upstreams through the resource-backed control API; direct upstream writes still fail closed. | Fake resolver with capability and failure knobs. | `NM-P0-002`, `NM-P0-003` |
| `ResolverPolicyAdapter` | Applies approved global AdGuard user rules and global protection pause/resume only after capturing rollback handles. Client/group targets remain unsupported until AdGuard client identity mapping exists. | Fake policy adapter plus fake AdGuard control API. | `NM-P0-003` |
| `SecretResolver` | Resolves AdGuard credentials from `resource-vault content get` using the stored `token_ref`; never reads Vault HTTP APIs or plaintext request payloads. | Fake secret resolver returning credentials or missing-secret failures. | `NM-P0-002`, `NM-P0-008` |
| `DeviceDiscoverySource` | Imports AdGuard Home configured/auto client evidence through the governed resolver backend and secret reference; future router/host sources can plug into the same seam. | Fake inventory sources with ambiguous identity cases plus fake AdGuard clients API. | `NM-P0-004` |
| `AdapterRegistry` | Selects host, resolver, router, and manual adapters. | Fake registry for Linux/macOS/Windows/manual capabilities. | `NM-P0-006` |
| `OptimizationScorer` | Scores candidate configs by reliability-first metrics. | Fixed-score scorer for workflow tests. | `NM-P0-005` |
| `ApprovalStore` | Persists approvals and rollback handles. | In-memory store. | `NM-P0-003`, `NM-P0-005`, `NM-P0-008` |
| `HomeAutomationPublisher` | Publishes actions/events to Home Automation. | Capturing publisher. | `NM-P0-007` |
| `RetentionPolicy` | Applies privacy and audit retention rules. | Fake clock + in-memory records. | `NM-P0-008` |
| `MonitoringSnapshotService` | Runs and reads snapshot evidence for monitoring comparisons. | Fake snapshot service with baseline/current metrics. | `NM-P1-007` |

## Adding a new seam

Add a seam only when it crosses a real boundary: external system, clock/time, storage, adapter capability, scoring policy, event publisher, or long-running workflow. Keep interfaces small and domain-owned.

## Architecture Alignment Notes

Network Manager's highest-risk seams are resolver/router/host adapters. They must report capabilities before use and must not let unsupported platforms look supported.

## UI-side seams

UI tests should fake generated API clients and operation-state polling. UI components should not know how resolver/router adapters work.

## What is NOT a seam

- A helper function inside one domain.
- A React component prop.
- A data transformation with no external dependency.
- A workflow state transition.

## API contract manifest

Endpoint metadata remains generated from API registration. Do not hand-edit `.vrooli/endpoints.json`.

## Cross-references

- [`../concepts/FLOWS.md`](../concepts/FLOWS.md)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
- [`TESTING.md`](TESTING.md)
