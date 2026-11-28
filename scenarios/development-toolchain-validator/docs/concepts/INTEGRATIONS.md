# Integrations — Development Toolchain Validator

## Purpose Of This Document

This guide records the validator's declared scenario dependencies and their
startup requirements.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, notes reference | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

The generated template does not declare external Vrooli resources. Add
resources to `.vrooli/service.json` only when a real scenario domain
requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None yet. | not-applicable | SQLite is embedded by default. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

| Scenario | Enabled | Required | Startup policy | Declared purpose and configuration | Degraded behavior |
|---|---:|---:|---|---|---|
| agent-manager | yes | yes | `must_start` | Spawns sandboxed skill runs; provides run summary + diff reference for skill validation. Its manifest config reconciles `.vrooli/agent-manager/default.json` and `.vrooli/agent-manager/divergence-probe.json`. | No degraded behavior declared. |
| prompt-manager | yes | yes | `must_start` | Source of truth for skill catalog (id/version/content hash) mirrored by `skill_catalog`. | No degraded behavior declared. |
| workspace-sandbox | yes | no | `try_start` | Reads sandbox diff content when content-rule manifest evaluation needs raw file content. | Degrades to a path-only verdict when unavailable. |

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
