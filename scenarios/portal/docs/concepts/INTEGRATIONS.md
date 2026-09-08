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
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default; no shared resource is required for briefs. | Add only when a domain needs shared resource behavior. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| `search-hub` | optional | Brief recall producer. | Connect `RoutingService.Query`; timeout, error, missing service, and weak results yield a persisted withheld verdict. |
| `agent-manager` | optional | Runs Portal agent chats after a brief is built. | Agent admission and event stream; unavailable manager leaves ordinary LLM chat usable. |
| `prompt-manager` | optional | Resolves selected operator skills for LLM prompts and owns published skill guidance. | Skill resolution failure omits optional skill context; it does not bypass brief gating. |
| `audio-tools` | optional | Voice input/output for the Portal shell. | Voice failure leaves typed chat and brief flows usable. |
| `compute-manager` | optional | Optional compute/model capability discovery for future operator surfaces. | Missing capability is reported as unavailable; briefs do not execute or provision compute. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |
| External agent harnesses | optional | A hook can request `EXTERNAL_HARNESS` context only after a canary-verified capability declaration. | Capability reader refuses unverified installation; hook exits 0 with no stderr on all delivery failures. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| search-hub | query timeout, error, or low confidence | Persist `WITHHELD_BUDGET`, `WITHHELD_DEGRADED`, or `WITHHELD_LOW_CONFIDENCE`; never inject partial context. | `packages/agentbrief-go` gate tests and Portal brief service tests |
| agent-manager | admission or stream unavailable | Preserve the user message and report the agent run failure; LLM and inspector remain available. | agentchat service tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
