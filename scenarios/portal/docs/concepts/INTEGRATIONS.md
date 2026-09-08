# Integrations — Portal

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
| SQLite | embedded storage | yes | API, brief persistence, chat persistence | resolved by `api-core/storage` from the scenario id | API health reports the storage failure; Portal cannot persist messages or briefs until storage recovers. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI, dependency startup | `.vrooli/service.json` and lifecycle phases | Start, stop, and restart through `vrooli scenario` or the scenario Makefile. A stopped optional dependency must not prevent Portal from starting. |

## Vrooli Resources

Portal declares no shared Vrooli resources in `.vrooli/service.json`. SQLite is
embedded in the Portal API and is not a separately managed resource.

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `search-hub` | optional | Brief recall producer. | Connect `RoutingService.Query`; timeout, error, missing service, and weak results yield a persisted withheld verdict. |
| `agent-manager` | optional | Runs Portal agent chats after a brief is built. | Agent admission and event stream; unavailable manager leaves ordinary LLM chat usable. |
| `prompt-manager` | optional | Resolves selected operator skills for LLM prompts and owns published skill guidance. | Skill resolution failure omits optional skill context; it does not bypass brief gating. |
| `audio-tools` | optional | Voice input/output for the Portal shell. | Voice failure leaves typed chat and brief flows usable. |
| `compute-manager` | optional | Optional compute/model capability discovery for future operator surfaces. | Missing capability is reported as unavailable; briefs do not execute or provision compute. |
| `device-control` | optional | Owner-authorized desktop control and surface/session operations. | Portal keeps chat and read-only surfaces available; side-effecting controls report a missing grant or unavailable provider before execution. |
| `scenario-authenticator` | optional | Operator account and session surfaces for protected Portal routes. | Local owner flows remain available; account-dependent surfaces report authentication unavailable. |
| `vrooli-bridge` | optional | Remote-device and attached-node surfaces. | Local Portal surfaces remain available; remote-device views report the bridge as unavailable. |
| `web-console` | optional | Remote terminal targets and operator surfaces. | Local chat remains available; remote terminal targets report the console as unavailable. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| OpenRouter | optional external API | LLM completion provider for Portal chat. | Missing credentials or an upstream failure produces a completion error while brief inspection and local Portal surfaces remain available. |
| External agent harnesses | optional | A hook can request `EXTERNAL_HARNESS` context only after a canary-verified capability declaration. | Capability reader refuses unverified installation; hook exits 0 with no stderr on all delivery failures. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| search-hub | query timeout, error, or low confidence | Persist `WITHHELD_BUDGET`, `WITHHELD_DEGRADED`, or `WITHHELD_LOW_CONFIDENCE`; never inject partial context. | `packages/agentbrief-go` gate tests and Portal brief service tests |
| agent-manager | admission or stream unavailable | Preserve the user message and report the agent run failure; LLM and inspector remain available. | agentchat service tests |
| prompt-manager | skill lookup unavailable | Omit optional skill context and keep the Portal prompt and brief contract available. | integration registry tests |
| audio-tools | provider unavailable or timeout | Keep typed chat and workspace controls available; report voice input/output unavailable. | integration registry tests |
| compute-manager | readiness query unavailable | Omit compute-node surfaces and report readiness unavailable. | integration registry tests |
| device-control | provider unavailable or grant denied | Refuse the requested side effect and preserve read-only Portal surfaces. | integration registry tests |
| scenario-authenticator | session/account lookup unavailable | Keep local owner flows available and report protected account surfaces unavailable. | authentication and health tests |
| vrooli-bridge | bridge unavailable | Keep local surfaces available and report remote attached-device surfaces unavailable. | integration registry tests |
| web-console | remote target unavailable | Keep local chat available and report remote terminal targets unavailable. | integration registry tests |
| OpenRouter | missing key, timeout, or provider error | Report the completion failure; do not fail Portal startup or brief inspection. | completion service tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
