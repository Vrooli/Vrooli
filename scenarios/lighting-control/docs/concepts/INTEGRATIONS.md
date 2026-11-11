# Integrations — Lighting Control

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |
| OpenRGB | opt-in host resource | no | control | Locally bound SDK client using the resource-owned backend lifecycle; it must never install, start, or probe hardware implicitly. E2 currently pins Linux amd64 OpenRGB 1.1 / SDK v6. Each operation negotiates v6 on its connection because negotiated protocol state is connection-local. A complete local write, correlated backend ACK/status, cached controller state, and independently observed physical output are distinct facts. | Preserve desired state; reconcile once at API boot and on the bounded 30-second control cadence so durable Off and zone state retry after a backend/device return. This is not evidence of physical output. Report unavailable/unsupported distinctly and retain physical observation as a separate gate. |
| system-monitor | local scenario API | no | policy | Resolve through `api-core/discovery`; read generated Connect `MetricsService/GetCurrentMetrics` and `GetDetailedMetrics`. Load and memory pressure retain independent observed timestamps and availability. | Stale or unavailable readings select static fallback and expose age/reason; zero is not assumed to mean healthy. |
| Agent Manager | local scenario API | no | policy | Resolve through `api-core/discovery`; use bounded `AgentManagerService/ListRuns` and count unique, non-imported pending, starting, and running records. Timestamp successful collection locally. | Missing/stale list disables activity pulse and exposes the static fallback. The owner API has no snapshot timestamp. |
| infrastructure-manager | local scenario API | no | policy | Discover generated `ConditionService/GetCondition(PROJECTION_SUBSTRATE)`; consume in-scope substrate readings with original observation time, trust and owner band verdicts. | Missing, stale or ungradeable host evidence selects configured fallback. Fresh independent memory warning/failure remains actionable. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| OpenRGB | opt-in, externally controlled lifecycle | Declared as optional only after the E2 service/SDK handoff is settled. Default remains disabled. | Setup/status UI and lifecycle call need the E2 owner contract; no duplicate installer/process owner. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| system-monitor | optional | Existing owner provides metrics; no telemetry duplication. | Use `api-core/discovery.ResolveScenarioURLDefault`, metrics current/pressure contracts, and source quality metadata. |
| agent-manager | optional | Existing owner provides active run records. | Use discovered Connect `ListRuns`; count pending/starting/running and apply client receipt freshness because the source response has no snapshot timestamp. |
| infrastructure-manager | optional runtime-only reader | Existing host/substrate owner grades machine condition. | `.vrooli/service.json` declares `runtime_only: true`, `required: false`, `startup_policy: ignore` through SDA `deps declare-runtime`. Light Desk never starts the owner or enables its health team. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| OpenRGB | SDK absent/refused, permission error, protocol mismatch | Leave stored desired state intact; distinguish backend readiness from device count and command acceptance. | control adapter fixtures and operated reconnect journey |
| system-monitor | timeout, typed degraded response, missing timestamp | Use bounded stale fallback; no automatic unhealthy color based on utilization alone. | policy source fixture tests |
| Agent Manager | timeout or unavailable scenario | Omit activity pulse and use the configured static fallback. | policy source fixture tests |

## Health and activity evidence

Health means the available host/substrate condition, not general scenario uptime. Light Desk preserves owner trust and band decisions for each in-scope substrate reading. Ghost, shelved, saturated and out-of-scope readings do not force a red lamp; unavailable or stale gradeable coverage remains visible. A healthy memory-pressure reading alone cannot establish healthy host coverage. Independent fresh memory pressure can still report attention/failure when other host evidence is incomplete. High CPU/GPU usage alone never makes Health unhealthy.

Combined takes its color from that health assessment and requests a gentle activity pulse only with fresh active-run evidence and supported zone capability. Unknown/stale activity keeps a usable health color steady. Unknown health uses the configured static fallback and brightness cap. Active-run evidence excludes imported historical sessions and deduplicates IDs across bounded pages; source failure is unknown, not zero active work.

Each source retains its own freshness and reason in effective policy evidence. API service `/health` is not host-health input. The consumer performs no host repair, team activation, dependency installation, service startup or hardware scan to obtain these readings.

## Protocol evidence

- OpenRGB 1.1 declares SDK protocol version 6 in the [official release record](https://openrgb.org/releases.html).
- The pinned local OpenRGB `release_1.1` source defines protocol version 6 and packet IDs: controller count 0, controller data 1, ACK 10, protocol negotiation 40, client name 50, server name 51, and UpdateZoneLEDs 1051. `NetworkServer.cpp` stores negotiated version per client connection; v6 controller-data requests therefore carry no version payload. The exact pinned release negotiation returns packets 40 and 51 followed by correlated ACK 10 for packet 40. Controller count returns packet 0 followed by correlated ACK 10 for packet 0; controller-data returns packet 1 followed by ACK for packet 1. Mode operations 1101/1103 and UpdateZoneLEDs1051 require their correlated ACK. Known asynchronous updates may interleave; malformed, unknown and out-of-sequence frames fail closed. The client correlates that ACK before reporting backend processing.
- A full local packet write records `local_write_completed`. A valid ACK records `backend_acknowledged` and its numeric status; status OK means the backend processed the request. Neither ACK nor any cached backend state proves the device applied the change or that light was visible. Physical output requires separate observation evidence. Exact source hashes and line references are preserved under `~/.vrooli/evidence/host-hardware-lighting/openrgb-protocol-f13/`.
- E2 owns the resource manifest, acquisition, process, and lifecycle. The lighting scenario only connects to its loopback SDK endpoint after explicit backend enablement. The current E2 implementation is Linux amd64 only and has not yet qualified live hardware access.

## Headless runtime handoff

The scenario API starts with an unbound in-memory `BackendBinding`; startup does not infer a port or probe OpenRGB. Native lifecycle launches expose an operator-private binding socket. The control-plane owner pins the exact current scenario PID/incarnation, runtime instance and component generation, verifies its identity before and after the operation, and supplies the literal loopback backend endpoint and incarnation. Revision and operation identity make uncertain retries idempotent and reject competing writers. Replaced processes cannot inherit a stale binding.

Quiescence revokes the endpoint and waits for the shared scenario writer and active SDK request to drain before the resource owner stops OpenRGB. Failed backend validation also attempts bounded quiescence. The API never persists the endpoint or backend incarnation. The standalone authenticated human maintenance route is registered only outside native lifecycle launches.

Settings and the scenario CLI call the typed SetupService for status, preparation, enable and disable. The API verifies the exact operation authority before invoking a fixed root command; callers supply an installation selection, never shell text or an SDK endpoint. Installation preparation, machine-local permission enrollment and runtime enablement are separate facts. Read-only status on a disabled host does not start services or probe controllers. Root owners retain artifact, permission and process authority. Backend ACK remains processing evidence, with physical output separately unverified.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
