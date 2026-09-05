# Integrations — Vrooli Bridge

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

Bridge's guiding principle is **compose, don't reinvent**: it orchestrates the fleet and delegates every adjacent concern (byte transport, durable runs, off-LAN reach, secrets, owner auth, audit) to the scenario that already owns it. The table below is the canonical list.

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite (`api-core/storage`) | embedded storage | yes | control-plane metadata (registry, pairing, runs, provisioning, audit) | `data` storage class under `~/.vrooli/data` | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| scenario-authenticator | scenario | yes | owner identity on control-plane access | HTTP validate/login (fail-closed, brief cache) | Control-plane access denied if unreachable (fail-closed, like device-sync-hub). |
| Full Vrooli install on each node | provisioned capability | yes (node side) | a node is a real build/test environment, not a thin runner | working-tree onboarding transfers a source-matched `vrooli` + `.fp`, then `vrooli setup` installs Go/git | A node without Vrooli cannot accept jobs, but no preinstalled Go or GitHub clone is required to onboard it. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

These are the compositions that keep bridge from reinventing solved problems. Each is the *owner* of its concern; bridge calls it through a seam.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| scenario-authenticator | required | Owner identity for control-plane access; bridge does not reimplement auth. | HTTP validate/login, fail-closed. |
| test-genie | required (runs) | Durable server-owned run lifecycle is reused for remote execution; when a job *is* a scenario test, test-genie is the actual mechanism on the node. | run-lifecycle semantics (start/wait/follow/abort by id, no polling). |
| device-sync-hub | required (P1) | Byte transport for inbound non-git artifacts (built installers, large fixtures) — "bridge orchestrates, device-sync-hub moves the bytes." Produced run outputs use bridge's bounded authenticated artifact RPC instead. | directed-delivery seam. |
| tunnel-manager | required (off-LAN) | Reach for nodes in another location; the node-agent dials the control plane's tunnel URL. | tunnel route to the control-plane endpoint. |
| workspace-sandbox | required | Accountability substrate for the immutable audit trail of dispatch + provisioning. | audit-record sink. |
| deployment-manager + scenario-to-desktop | consumer (P1) | Consumers of the cross-OS validation gate; deployment-manager owns the verdict, bridge supplies the capability. | gate API contract. |
| secrets-manager | optional (P2) | Path for any node-bound secret; bridge never ships secrets ad hoc. | secret resolution seam. |

### deployment-manager

The cross-OS deployment gate (OT-P1-002) is the headline consumer integration:
**bridge supplies the capability, deployment-manager owns the verdict.**

- **What bridge supplies:** `GateService` (`RunGate` / `GetGate` / `WaitGate` /
  `ListGates`). `RunGate(scenario, target_revision, target_oses[])` selects one
  eligible node per OS, dispatches the native validation run to each (the default
  verb is `scenario test <scenario>`) through the allowlisted dispatch path, and
  records a durable gate. `WaitGate` blocks once and returns the aggregate
  cross-OS verdict — PASSED only when **every** target OS validated green; ANY
  failing OS (a non-zero/aborted run, or a target OS with no eligible node) fails
  the gate, with the offending OS's run id surfaced for log drill-in.
- **What deployment-manager owns:** the promotion decision. Its
  `crossosgate` package (`scenarios/deployment-manager/api/crossosgate/`) speaks
  bridge's `GateService` over the Connect unary JSON protocol — **no proto-module
  dependency on the consumer side** — and maps the aggregate gate verdict to its
  own `ProductionReady` boolean (`POST /api/v1/cross-os-gate/evaluate`). A
  timed-out gate is never assumed green; promotion is withheld until a real green
  is observed (the gate is durable + re-attachable by id).
- **Wiring:** additive and inert by default — the route returns `503` until
  the shared node client (and its configured credentials) points at a bridge
  control plane, so the integration never destabilizes deployment-manager's
  existing flows.
- **Business validation (BRG-P1-002):** a scenario is proven green on real
  Ubuntu / macOS / Windows nodes before being marked production-ready — the
  missing primitive behind "production-ready means validated on every OS we ship
  to." scenario-to-desktop is a second prospective consumer of the same contract.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None | not-applicable | Bridge talks only to Vrooli installs the owner controls; OS service managers (systemd/launchd/Windows Service) are platform facilities, not third-party services, and mDNS (P1) is a LAN protocol, not an external API. | Add only if a hosted cloud-runner provider is integrated (P2). |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| scenario-authenticator | validate/login error or timeout | Control-plane access fails closed; a brief cache absorbs transient blips but never re-admits a revoked owner. | auth middleware tests |
| Node offline (dial-out channel dropped) | presence heartbeat miss | Node marked offline; dispatch to it is rejected/queued, not silently lost. | presence tests |
| test-genie unavailable on a node | run cannot start | Job fails with a clear, audited error; no partial/zombie run. | runs integration tests |
| device-sync-hub unavailable | artifact distribution fails | Job that needs the artifact is not dispatched; reported as a precondition failure. | artifacts integration tests |
| tunnel-manager route down | off-LAN node unreachable | Node shows offline; on-LAN nodes unaffected. | presence/reach tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
