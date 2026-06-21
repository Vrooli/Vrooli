# Test Genie Phases

Test Genie phases are declared in the catalog at [`api/internal/orchestrator/phases/catalog.go`](../../api/internal/orchestrator/phases/catalog.go). This document is generated from that catalog; edit the catalog when phase metadata changes.

## Phase Summary

| Order | Phase | Timeout | Optional | Runtime | Source | Purpose |
|-------|-------|---------|----------|---------|--------|---------|
| 1 | [Structure](structure/README.md) | 1m | No | No | validation-provider | Delegates scenario skeleton + lifecycle-wiring validation to structure-health, which reconciles code-facts ground truth against declared service.json intent (profile-aware) and maps findings into the FINDING_SOURCE_STRUCTURE channel before any tests run. |
| 2 | [Contracts](contracts/README.md) | 1m | No | No | validation-provider | Validates cli/manifest.json bindings against proto descriptors via cli-health. |
| 3 | [UI Health](ui-health/README.md) | 1m | No | No | validation-provider | Validates ui/manifest.json bindings, slot directories, and overlay rules via ui-health. |
| 4 | [Standards](standards/README.md) | 1m | No | No | validation-provider | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config). |
| 5 | [Architecture](architecture/README.md) | 2m | Yes | No | validation-provider | Delegates structural-cohesion validation to architecture-cartographer through ScenarioValidationService; blocker findings gate only when the architecture authority is high-confidence unless TEST_GENIE_ARCHITECTURE_GATE overrides rollout mode. |
| 6 | [Dependencies](dependencies/README.md) | 15m | No | No | validation-provider | Delegates dependency readiness, runtime dependency status, governance, release-age policy, security index availability, and graph drift to scenario-dependency-analyzer through ScenarioValidationService. |
| 7 | [Quality](quality/README.md) | 2m | No | No | validation-provider | Delegates static quality contracts, lint/type policy, and strict config validation to quality-health. |
| 8 | [DOCS](docs/README.md) | 1m | No | No | validation-provider | Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService. |
| 9 | [Performance](performance/README.md) | 5m | Yes | Yes | validation-provider | Delegates Go API and UI build benchmarking plus Lighthouse audits (performance, accessibility, SEO) to the performance-health scenario through ScenarioValidationService. |
| 10 | [Smoke](smoke/README.md) | 15m | Yes | Yes | native | Validates UI loads correctly, establishes iframe-bridge communication, and has no critical errors. |
| 11 | [Unit](unit/README.md) | 15m | No | No | validation-provider | Delegates test execution, coverage, test architecture, test quality, and flake/runtime diagnostics to the unit-health scenario, mapping coverage findings into the FINDING_SOURCE_COVERAGE channel that feeds the ecosystem-manager `coverage` dimension. |
| 12 | [Integration](integration/README.md) | 15m | No | Yes | native | Exercises the CLI/Bats suite plus scenario-local orchestrator listings. |
| 13 | [Playbooks](playbooks/README.md) | 15m | No | Yes | native | Executes Vrooli Ascension workflows declared under bas/ to validate end-to-end UI flows. |
| 14 | [Business](business/README.md) | 15m | No | No | native | Audits requirements modules to guarantee operational targets stay mapped. |
| 15 | [Tidiness](tidiness/README.md) | 2m | Yes | No | validation-provider | Delegates file/function quality checks to tidiness-manager through ScenarioValidationService and maps assessment findings into the FINDING_SOURCE_TIDINESS channel. |
| 16 | [Security](security/README.md) | 3m | Yes | No | validation-provider | Delegates security posture validation to security-health (secrets, Go SAST, Go vuln-DB, JS deps) and maps findings into the FINDING_SOURCE_SECURITY channel that gates the ecosystem-manager R1 ladder rung. |
| 17 | [Measures](measures/README.md) | 3m | Yes | No | validation-provider | Delegates measures-coverage validation to measures-health (stateful-domain coverage + per-measure tier) and maps findings into the FINDING_SOURCE_MEASURES channel that feeds the ecosystem-manager soft `measures` ladder dimension. |
| 18 | [Proto](proto/README.md) | 2m | Yes | No | validation-provider | Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension. |

## Static Phases

- [Structure](structure/README.md) - Delegates scenario skeleton + lifecycle-wiring validation to structure-health, which reconciles code-facts ground truth against declared service.json intent (profile-aware) and maps findings into the FINDING_SOURCE_STRUCTURE channel before any tests run.
- [Contracts](contracts/README.md) - Validates cli/manifest.json bindings against proto descriptors via cli-health.
- [UI Health](ui-health/README.md) - Validates ui/manifest.json bindings, slot directories, and overlay rules via ui-health.
- [Standards](standards/README.md) - Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config).
- [Architecture](architecture/README.md) - Delegates structural-cohesion validation to architecture-cartographer through ScenarioValidationService; blocker findings gate only when the architecture authority is high-confidence unless TEST_GENIE_ARCHITECTURE_GATE overrides rollout mode.
- [Dependencies](dependencies/README.md) - Delegates dependency readiness, runtime dependency status, governance, release-age policy, security index availability, and graph drift to scenario-dependency-analyzer through ScenarioValidationService.
- [Quality](quality/README.md) - Delegates static quality contracts, lint/type policy, and strict config validation to quality-health.
- [DOCS](docs/README.md) - Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService.
- [Unit](unit/README.md) - Delegates test execution, coverage, test architecture, test quality, and flake/runtime diagnostics to the unit-health scenario, mapping coverage findings into the FINDING_SOURCE_COVERAGE channel that feeds the ecosystem-manager `coverage` dimension.
- [Business](business/README.md) - Audits requirements modules to guarantee operational targets stay mapped.
- [Tidiness](tidiness/README.md) - Delegates file/function quality checks to tidiness-manager through ScenarioValidationService and maps assessment findings into the FINDING_SOURCE_TIDINESS channel.
- [Security](security/README.md) - Delegates security posture validation to security-health (secrets, Go SAST, Go vuln-DB, JS deps) and maps findings into the FINDING_SOURCE_SECURITY channel that gates the ecosystem-manager R1 ladder rung.
- [Measures](measures/README.md) - Delegates measures-coverage validation to measures-health (stateful-domain coverage + per-measure tier) and maps findings into the FINDING_SOURCE_MEASURES channel that feeds the ecosystem-manager soft `measures` ladder dimension.
- [Proto](proto/README.md) - Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension.

## Runtime Phases

- [Performance](performance/README.md) - Delegates Go API and UI build benchmarking plus Lighthouse audits (performance, accessibility, SEO) to the performance-health scenario through ScenarioValidationService.
- [Smoke](smoke/README.md) - Validates UI loads correctly, establishes iframe-bridge communication, and has no critical errors.
- [Integration](integration/README.md) - Exercises the CLI/Bats suite plus scenario-local orchestrator listings.
- [Playbooks](playbooks/README.md) - Executes Vrooli Ascension workflows declared under bas/ to validate end-to-end UI flows.

## Running Phases

```bash
test-genie execute my-scenario --phases structure,unit
test-genie execute my-scenario --preset comprehensive
```

## Configuration

Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).

## Presets

Preset membership is generated from the same catalog and documented in [Presets Reference](../reference/presets.md).
