# API Endpoints — Network Manager

The machine-readable endpoint source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json). This document describes the planned product API surface; implementation will add proto/Connect contracts and refresh the endpoint manifest.

## System

### `GET /health`

Lifecycle health endpoint for API readiness. This remains an operational REST exception.

### Planned product services

| Service | Planned operations | Requirements |
|---|---|---|
| `SnapshotService` | `RunSnapshot`, `GetSnapshot`, `ListSnapshots`, `ExportSnapshotReport` | `NM-P0-001` |
| `ResolverService` | `GetResolverStatus`, `ConfigureAdGuardHome`, `UpdateUpstreams`, `CheckResolverHealth` | `NM-P0-002` |
| `PolicyService` | `PreviewPolicyChange`, `ApplyPolicyChange`, `RollbackPolicyChange`, `PauseFiltering`, `ResumeFiltering`, `ListPolicyProfiles`, `UpsertPolicyProfile`, `EvaluatePolicySchedule`, `DiagnoseEncryptedDnsBypass`, `GetEndpointDohGuidance` | `NM-P0-003`, `NM-P1-001`, `NM-P1-002`, `NM-P1-004`, `NM-P1-008` |
| `InventoryService` | `RefreshInventory`, `ListDevices`, `UpdateDeviceGroup`, `ExplainDeviceIdentity` | `NM-P0-004` |
| `OptimizationService` | `CreateOptimizationRun`, `RunCandidate`, `ScoreCandidates`, `ApproveCandidate`, `RollbackOptimization` | `NM-P0-005` |
| `AdapterService` | `ListCapabilities`, `ExplainUnsupportedAction`, `GetPlatformSummary` | `NM-P0-006` |
| `HomeIntegrationService` | `ListActions`, `InvokeAction`, `ListRecentEvents` | `NM-P0-007` |
| `PrivacyService` | `GetRetentionSettings`, `UpdateRetentionSettings`, `GetVisibilitySettings` | `NM-P0-008` |

## Snapshot API

Snapshot endpoints must return partial results with confidence flags when a probe is unsupported or times out. Unsupported probes are not failures.

## Resolver API

Resolver endpoints must hide backend-specific details behind capability reports. AdGuard Home is the first backend; Pi-hole and Technitium are deferred adapters.

## Policy API

Policy mutation endpoints must support preview before apply and a rollback handle after apply when the backend supports rollback. Household profile endpoints persist policy intent and evaluate schedules without claiming live resolver/router enforcement. Guidance endpoints diagnose IPv6/encrypted-DNS bypass and endpoint/browser DoH controls as read-only reports; they must not mutate router, firewall, browser, or endpoint policy.

## Optimization API

Optimization endpoints must expose operation state and never apply persistent changes without approval. Candidate scoring is reliability-first: latency, jitter, packet loss, DNS performance, stability, then throughput.

## Adding a new endpoint

1. Confirm the endpoint maps to a PRD operational target and requirement.
2. Add or extend proto messages/services under `packages/proto/schemas/network-manager/v1/<domain>/`.
3. Implement a thin handler that delegates to domain service logic.
4. Bind CLI/UI consumers to generated clients.
5. Regenerate endpoint metadata; do not edit `.vrooli/endpoints.json` by hand.
6. Add `[REQ:ID]` tagged tests for the new behavior.

## Cross-references

- [`cli-commands.md`](cli-commands.md)
- [`configuration.md`](configuration.md)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
- [`../internal/TESTING.md`](../internal/TESTING.md)
