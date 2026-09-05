# Integrations — API Health

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
| SQLite | embedded storage | yes | provider runtime, future probe/fix sessions | resolved by `api-core/storage` from the scenario id | API Health reports its own health as unhealthy if DB bootstrap fails. |
| Vrooli lifecycle | local platform | yes | probe, validation, CLI | `.vrooli/service.json`, lifecycle API/ports | Live probes degrade or fail with an explicit lifecycle discovery finding; static validation still runs. |
| test-genie | scenario | no at runtime | validation/cutover | shared `ScenarioValidationService` provider contract | Provider can validate locally; provider-contract checks and delegated phase cutover wait for Test Genie. |
| scenario-auditor | source migration reference | no | migration | source files under `scenarios/scenario-auditor/api/rules/api/` | No runtime effect; migration ledger work pauses if source is removed before accounting completes. |

## Vrooli Resources

API Health declares no external resources. It should remain runnable with only
embedded SQLite and local source trees.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None. | not-applicable | Validation is source/lifecycle based and should not require vector stores, browsers, or model resources. | Add only if a future declared domain has a non-optional resource need. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| test-genie | optional consumer | Delegated provider cutover and provider-contract checks. | `scenario-validation/v1.ScenarioValidationService`; provider conformance in `docs/reference/health-maturity-assessments.md`. |
| scenario-auditor | disabled migration source | API Health reads old API rule intent during migration but must not call it. | Migration ledger only; no runtime API/CLI contract. |
| quality-health | neighboring authority | Owns static lint/type/config quality that API Health must not duplicate. | Boundary reference. |
| security-health | neighboring authority | Owns security headers/CORS/scanners. | Boundary reference. |
| cli-health | neighboring authority | Owns CLI manifest/runtime conformance. | Boundary reference. |
| proto-health | neighboring authority | Owns proto and generated-code health. | Boundary reference. |
| ui-health | neighboring authority | Owns UI validation and runtime rendering. | Boundary reference. |
| storage-manager | neighboring authority | Owns storage isolation and migration checks. | Boundary reference. |
| performance-health | neighboring authority | Owns benchmarking and budgets. | Boundary reference. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| None. | not-applicable | API Health validates local Vrooli scenario API surfaces. | Add only for future explicit integrations. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` or migration error | API Health's own `/health` reports unhealthy. | health handler tests |
| Target lifecycle API | port/URL missing, connection refused, timeout | Static validation still returns; live probe emits `api_health.health_probe_failed` when execution was requested. | planned probe tests |
| test-genie unavailable | provider-contract command cannot reach consumer | Local `api-health validate scenario` still works; cutover validation is deferred. | planned integration tests |
| scenario-auditor source missing | migration ledger cannot compare old rule inventory | Implementation blocks migration accounting but not provider runtime. | planned migration ledger test |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
