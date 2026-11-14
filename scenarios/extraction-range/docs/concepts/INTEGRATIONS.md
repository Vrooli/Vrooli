# Integrations — Extraction Range

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
| SQLite | embedded storage | yes | API health and range scoreboard | resolved by `api-core/storage` from the scenario id | Best synthetic scores persist in the range-owned table; database health is reported by `/health`. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario is set up and started through lifecycle commands. |

## Vrooli Resources

The range proving ground is self-contained: it serves its own target pages and
does not crawl external sites or require a shared resource. SQLite backs the
durable, synthetic best-scoreboard rows and health checks.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None. | not-applicable | All current practice pages and score evaluation run in this scenario; no external resource is needed. | Revisit only if a new approved requirement needs a shared resource. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Extraction tools call the stable HTTP surface; the range itself stays standalone. | Revisit if an approved product slice composes a shared Vrooli capability. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | No real website or third-party service is accessed by the proving ground. | Real-target research remains outside this scenario and must obey each site's rules. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Extraction client | malformed score JSON, unknown tool/level or payload over 2 MiB | `/score` returns 400 without updating the scoreboard. | range handler tests |
| Synthetic resilience level | first request is 429/503 | Client honors retry status; a later request serves the challenge records. | range handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
