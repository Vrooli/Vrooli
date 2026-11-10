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
| OpenRGB | opt-in host resource | no | control | Locally bound SDK client using the resource-owned backend lifecycle; it must never install, start, or probe hardware implicitly. E2 currently pins Linux amd64 OpenRGB 1.1 / SDK v6. Each operation negotiates v6 on its connection because negotiated protocol state is connection-local. A complete local write, correlated backend ACK/status, cached controller state, and independently observed physical output are distinct facts. | Preserve desired state; report unavailable/unsupported distinctly and reconcile after an explicit backend start or reconnect. Physical output needs independent observation evidence. |
| system-monitor | local scenario API | no | policy | Resolve through `api-core/discovery`; read `GET /api/v1/metrics/current` and `/api/v1/metrics/pressure`. Inspect metric state, observed timestamp, freshness and pressure availability. | Stale or unavailable readings select static fallback and expose age/reason; zero is not assumed to mean healthy. |
| Agent Manager | local scenario API | no | policy | Resolve through `api-core/discovery`; use bounded `AgentManagerService/ListRuns` and count only pending, starting, and running states. Timestamp the successful response locally. | Missing/stale list disables activity pulse and exposes the static fallback. The owner API has no snapshot timestamp. |

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

## Protocol evidence

- OpenRGB 1.1 declares SDK protocol version 6 in the [official release record](https://openrgb.org/releases.html).
- The pinned local OpenRGB `release_1.1` source defines protocol version 6 and packet IDs: controller count 0, controller data 1, ACK 10, protocol negotiation 40, client name 50, server name 51, and UpdateZoneLEDs 1051. `NetworkServer.cpp` stores negotiated version per client connection; v6 controller-data requests therefore carry no version payload. Negotiation returns packets 40 and 51 without an ACK. Controller count is replied to by the listener without an ACK; controller-worker operations such as controller data and UpdateZoneLEDs return ACK 10 with the original packet ID and status. The client correlates that ACK before reporting backend processing.
- A full local packet write records `local_write_completed`. A valid ACK records `backend_acknowledged` and its numeric status; status OK means the backend processed the request. Neither ACK nor any cached backend state proves the device applied the change or that light was visible. Physical output requires separate observation evidence. Exact source hashes and line references are preserved under `~/.vrooli/evidence/host-hardware-lighting/openrgb-protocol-f13/`.
- E2 owns the resource manifest, acquisition, process, and lifecycle. The lighting scenario only connects to its loopback SDK endpoint after explicit backend enablement. The current E2 implementation is Linux amd64 only and has not yet qualified live hardware access.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
