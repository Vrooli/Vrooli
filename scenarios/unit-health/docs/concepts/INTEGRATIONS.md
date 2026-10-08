# Integrations — Unit Health

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
| SQLite | embedded storage | yes | API (health probe, run history) | resolved by `api-core/storage` from the scenario id | `/health` reports the dependency unhealthy; validation still runs, without persisted history. |
| Code Facts | scenario (Connect-RPC) | yes for discovery | validation / `discovery.CodeFactsClient` | `code-facts/v1/facts` proto; URL resolved at call time via the scenario resolver | Discovery falls back to a local heuristic inventory and reports `degraded_reason`. |
| Vrooli CLI | host tool | optional | validation handler (host inventory for metrics) | `vrooli-cli-go` `HostCaptureEnvironment` | Non-fatal; metrics environment falls back to the stdlib baseline. |
| maturity-go | shared Go package | yes | validation handler (assessment, provider description) | `maturity` block of `.vrooli/test-genie.json` | Spec load failure logs and returns responses without a shared assessment. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

`.vrooli/service.json` declares no external Vrooli resources. Add
resources only when a real domain requires them.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None. | not-applicable | SQLite is embedded; the evidence cache is on the local filesystem. | Add when PRD/requirements demand shared resource behavior. |

## Scenario Dependencies

`.vrooli/service.json` declares no scenario dependencies. Code Facts is
resolved at call time rather than declared, so an unavailable Code Facts
is a degradation, not a startup failure.

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| code-facts | runtime, undeclared | Surface and parse-unit discovery for the validation target. | `CodeFactsService` Connect client in `api/internal/discovery/discovery.go`. |
| test-genie (consumer) | inbound | Test Genie calls this scenario through the shared `ScenarioValidationService` (`ValidateScenario`, `DescribeProvider`, `PreviewFix`, `ApplyFix`). | `packages/proto/schemas/scenario-validation/v1`. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | Validation runs entirely against the local repository and host toolchains. | Add when PRD/requirements require external APIs, webhooks, auth, payments, or data feeds. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| Code Facts | Connect error or unresolved scenario URL | Fallback inventory, `degraded_reason` set, degraded status. | `api/internal/discovery` tests, `api/internal/validation/service_test.go` |
| Test toolchain (go, pnpm, bats, python) | Missing binary or command timeout | Command result carries a failure class; findings and maturity reflect it instead of aborting the run. | `api/internal/executor` tests, `api/internal/validation/execution_test.go` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
