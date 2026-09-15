# Integrations — Source repository

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
| SQLite | embedded storage | yes | API and distribution repositories | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable; artifacts remain under the scenario data namespace. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

No shared Vrooli resource is required. SQLite is embedded because export
metadata is local scenario state; source output documents runtime resources
instead of bundling host state or credentials.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None | not-applicable | SQLite is embedded and source runtime requirements are documented, not provisioned here. | Add only when a supported export mode requires a shared resource. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Scenario Dependency Analyzer | planned | Authoritative source closure and local-module rewrite proposals. | Missing or unavailable closure remains unresolved; no private analyzer is implemented here. |
| Deployment Manager | planned | Exact-candidate readiness and release approval. | Approval is unavailable until the exact artifact/evidence tuple is supplied. |
| Git Control Tower | planned | Source sharing entrypoint and provenance context. | Link to one ramp-owned distribution; no duplicate export state. |
| Test Genie | planned | Server-owned clean verification receipts. | Unknown/unavailable evidence never becomes a pass. |
| Integration Hub | unavailable | Host connector and destination read-back, separately owned. | Publication handoff remains human-only and unverified without connector evidence. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario has no third-party dependency. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
