# Integrations — Decentralized Deploy

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.


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
| Kubo 0.43.1 | managed host tool | required for local IPFS deployment | deployment storage adapter | pinned tool manifest `internal/tools/kubo/tool.json`; isolated offline repository per operation | Fail closed if managed executable is absent or its repository ownership marker does not match. |

## Vrooli Resources

No shared Vrooli resource is currently required. Kubo is a managed host tool,
not a shared resource. Add external resources only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| Compute Manager | caller-supplied reference | Compute Manager owns providers; the deployment caller passes the compute reference, so CI1 does not create a compute-provider boundary. | Storage/name operations consume the reference without selecting or provisioning compute. |

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
