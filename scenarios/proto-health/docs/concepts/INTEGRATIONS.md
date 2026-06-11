# Integrations — Proto Health

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
| SQLite | embedded storage | yes | API, notes reference | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| None yet. | not-applicable | Generated scenario is standalone. | Add when this scenario calls or composes another scenario. |

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
