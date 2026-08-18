# Integrations — Token Economy

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## The integration posture, stated once

This scenario is deliberately close to standalone, and that is a product
decision rather than an accident of being new. **A household economy that
requires infrastructure is not a household economy** — it must run on a laptop
with nothing else started. Exactly one hard scenario dependency exists
(`scenario-authenticator`, because a child's view must be genuinely
authenticated rather than a client-side role flag), and everything else is
optional or inbound.

The scenario also has one deliberate **non-integration**: `treasury` is a
*contract sibling*, not a dependency. The two share a grant/mandate shape by
design, neither calls the other, and this scenario must remain fully functional
with `treasury` absent. That congruence is maintained by a parity test
(`TKE-P0-002`) and by recording every intentional divergence in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, all persistence-backed domains | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario is started through lifecycle commands only. |
| `scenario-authenticator` | scenario | **yes** | holders, and every authenticated read/write | Slug resolution; tokens verified locally against published JWKS | Fail closed. Without a verifiable identity the holder boundary cannot be enforced, so authenticated surfaces refuse rather than degrade. Health reports the dependency. |
| `notification-hub` | scenario | no | redemption (approval requests) | Slug resolution; best-effort relay | Degrade silently to in-scenario queue only. The approval queue is first-class and works unchanged; only out-of-band reach is lost. |
| `agent-manager` + `packages/cli-core` | scenario + shared package | no | journal (actor provenance) | `VROOLI_AGENT_IDENTITY_TOKEN`, `X-Agent-Identity-Token`, shared verifier | Degrade to recording the actor as unverified rather than promoting it — the ecosystem's existing runtime-attribution contract. Never blocks a write. |
| `react-component-library` | scenario | no (build-time) | ui | `adoptions apply` provenance | Build-time only; no runtime coupling. |
| `brand-manager` | scenario | no (build-time) | ui | Design tokens referenced, never redefined | Build-time only. |

## Vrooli Resources

The scenario declares **no external Vrooli resources**, and this is a decision
rather than a default. Every candidate was considered and rejected:

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Postgres | not-applicable | Household volume with single-writer mutations; SQLite under a row lock is sufficient and keeps the scenario laptop-runnable. | The P2 real-chain adapter (`TKE-P2-001`), which introduces concurrency this scenario does not otherwise face. |
| Redis | not-applicable | No hot state worth externalizing. Reservations are durable by design, not cached. | Never, under the current product shape. |
| Qdrant / vector store | not-applicable | Nothing here is retrieved by similarity. | Never, under the current product shape. |
| Ollama / any inference | not-applicable | Rule evaluation is deterministic from a closed vocabulary; introducing a model would make refusals unexplainable, which contradicts `TKE-P0-003`. | Never. This is a guardrail, not a gap. |
| Object storage | not-applicable | No binary payloads. Catalog entries carry text and a token cost. | If catalog entries ever carry images. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `scenario-authenticator` | required | A holder's view must be genuinely authenticated. A client-side role flag would make the multi-holder isolation boundary (`TKE-P0-006`) cosmetic. | Resolve by slug; verify tokens locally against the published JWKS; never send a credential or perform a per-request authorization callback. |
| `notification-hub` | optional | Improves reach for approval requests. A minter who is not at the console still learns a child is waiting. | Relay only. The scenario owns the queue; the hub owns delivery. Absence is not degraded mode — it is the baseline. |
| `agent-manager` | optional | Supplies verified actor provenance when the caller is an agent, so an agent-initiated grant is distinguishable after the fact. | Read-only verification through the shared `packages/cli-core` verifier. The scenario does not spawn agents and does not invent a second attribution model. |
| `treasury` | **contract sibling — not a dependency** | Shared grant/mandate shape by design; neither calls the other. | Congruence enforced by parity test; divergences recorded. Unification is a P2 target (`TKE-P2-004`), not a P0 coupling. |
| Any earning-surface scenario | inbound, optional | Any scenario may become an earning surface through the public adapter contract. This scenario does not know or care which. | `TKE-P0-007`. No privileged earner. The first real adapter (`TKE-P1-009`) will be named here when chosen. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Deliberate. No payment processor, no analytics vendor, no push provider, no chain node. Nothing leaves the machine — that is the privacy wedge described in [`DATA.md`](DATA.md). | Introducing any third-party service requires a recorded decision, because it changes the product's central claim. |
| Chain node / RPC provider | deferred | Would be required by the P2 real-value rail (`TKE-P2-001`). | Gated on a recorded custody and regulatory decision, never on a feature request. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` reports unhealthy dependency status; mutations refuse rather than partially apply. | health handler tests |
| `scenario-authenticator` unreachable | JWKS fetch or verification failure | **Fail closed.** Authenticated surfaces refuse; the scenario does not fall back to an unauthenticated view. Health reports the dependency as degraded. | holders isolation tests, health handler tests |
| `notification-hub` unreachable | Relay call error or absent slug | **Degrade silently.** Approval request is queued and visible in the console; no error surfaces to the holder, because the request did succeed. | `TKE-P0-013` integration test asserts approval works with the hub unavailable |
| `agent-manager` unreachable | Identity verification transport error | **Degrade honestly.** Event is written with the actor recorded as unverified, never promoted and never dropped. | `TKE-P0-011` provenance status matrix |
| Earning adapter misbehaving (replay, flood) | Duplicate dedup keys, high submission rate | Replays are idempotent no-ops (`TKE-P0-007`); rate limiting is a P1 concern recorded in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) if it becomes real. | earning service tests |

## Third-Party Packages

All npm/Go/pip packages are governed by **Scenario Dependency Analyzer**. This
scenario adds none beyond the template baseline at documentation time. If one
becomes necessary:

```bash
scenario-dependency-analyzer deps approved search "<purpose>" --surface <ui|api|cli>
scenario-dependency-analyzer deps install <ecosystem>/<package> --scenario token-economy --surface <surface> --apply
```

Never hand-edit `.vrooli/dependencies/approved-dependencies.json` and never run
a raw package manager.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`DOMAINS.md`](DOMAINS.md) — which domain uses which dependency
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why these dependencies and not others
