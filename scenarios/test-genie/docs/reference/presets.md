# Test Presets Reference

Test Genie presets bundle catalog phases for common validation loops. This document is generated from `api/internal/orchestrator/phases`; edit the catalog or preset declarations instead of hand-editing these tables.

Timeout values are runtime budgets, not estimates. Runtime estimates are calculated from recent per-phase history when available.

## Available Presets

### Quick

Fast sanity check during development.

```bash
test-genie execute my-scenario --preset quick
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario layout, manifests, and JSON health before any tests run. | 15m |
| Standards | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config). | 1m |
| DOCS | Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService. | 1m |
| Business | Audits requirements modules to guarantee operational targets stay mapped. | 15m |
| Unit | Delegates test execution, coverage, test architecture, test quality, and flake/runtime diagnostics to the unit-health scenario, mapping coverage findings into the FINDING_SOURCE_COVERAGE channel that feeds the ecosystem-manager `coverage` dimension. | 15m |
| Proto | Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension. | 2m |

### Smoke

Core validation before pushing or handing off changes.

```bash
test-genie execute my-scenario --preset smoke
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario layout, manifests, and JSON health before any tests run. | 15m |
| Standards | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config). | 1m |
| Quality | Delegates static quality contracts, lint/type policy, and strict config validation to quality-health. | 2m |
| DOCS | Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService. | 1m |
| Business | Audits requirements modules to guarantee operational targets stay mapped. | 15m |
| Integration | Exercises the CLI/Bats suite plus scenario-local orchestrator listings. | 15m |
| Proto | Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension. | 2m |

### Architecture Audit

Surface conformance and architectural shape without runtime-heavy phases.

```bash
test-genie execute my-scenario --preset architecture-audit
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario layout, manifests, and JSON health before any tests run. | 15m |
| Contracts | Validates cli/manifest.json bindings against proto descriptors via cli-health. | 1m |
| UI Health | Validates ui/manifest.json bindings, slot directories, and overlay rules via ui-health. | 1m |
| DOCS | Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService. | 1m |
| Standards | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config). | 1m |
| Architecture | Delegates structural-cohesion validation to architecture-cartographer through ScenarioValidationService; blocker findings gate only when the architecture authority is high-confidence unless TEST_GENIE_ARCHITECTURE_GATE overrides rollout mode. | 2m |
| Proto | Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension. | 2m |

### Comprehensive

Full validation before release or deployment.

```bash
test-genie execute my-scenario --preset comprehensive
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario layout, manifests, and JSON health before any tests run. | 15m |
| Contracts | Validates cli/manifest.json bindings against proto descriptors via cli-health. | 1m |
| UI Health | Validates ui/manifest.json bindings, slot directories, and overlay rules via ui-health. | 1m |
| Standards | Runs scenario-auditor standards rules (PRD/service.json/proxy/lifecycle config). | 1m |
| Architecture | Delegates structural-cohesion validation to architecture-cartographer through ScenarioValidationService; blocker findings gate only when the architecture authority is high-confidence unless TEST_GENIE_ARCHITECTURE_GATE overrides rollout mode. | 2m |
| Dependencies | Delegates dependency readiness, runtime dependency status, governance, release-age policy, security index availability, and graph drift to scenario-dependency-analyzer through ScenarioValidationService. | 15m |
| Quality | Delegates static quality contracts, lint/type policy, and strict config validation to quality-health. | 2m |
| DOCS | Delegates docs Markdown, mermaid, link, reference, and manifest validation to knowledge-observatory through ScenarioValidationService. | 1m |
| Performance | Benchmarks Go API and UI builds, runs Lighthouse audits via Google Lighthouse CLI to validate performance, accessibility, and SEO. | 15m |
| Smoke | Validates UI loads correctly, establishes iframe-bridge communication, and has no critical errors. | 15m |
| Unit | Delegates test execution, coverage, test architecture, test quality, and flake/runtime diagnostics to the unit-health scenario, mapping coverage findings into the FINDING_SOURCE_COVERAGE channel that feeds the ecosystem-manager `coverage` dimension. | 15m |
| Integration | Exercises the CLI/Bats suite plus scenario-local orchestrator listings. | 15m |
| Playbooks | Executes Vrooli Ascension workflows declared under bas/ to validate end-to-end UI flows. | 15m |
| Business | Audits requirements modules to guarantee operational targets stay mapped. | 15m |
| Tidiness | Delegates file/function quality checks to tidiness-manager through ScenarioValidationService and maps assessment findings into the FINDING_SOURCE_TIDINESS channel. | 2m |
| Security | Delegates security posture validation to security-health (secrets, Go SAST, Go vuln-DB, JS deps) and maps findings into the FINDING_SOURCE_SECURITY channel that gates the ecosystem-manager R1 ladder rung. | 3m |
| Measures | Delegates measures-coverage validation to measures-health (stateful-domain coverage + per-measure tier) and maps findings into the FINDING_SOURCE_MEASURES channel that feeds the ecosystem-manager soft `measures` ladder dimension. | 3m |
| Proto | Delegates proto contract validation to proto-health and maps findings into the FINDING_SOURCE_PROTO channel that feeds the ecosystem-manager soft `proto-health` R2 ladder dimension. | 2m |

## Preset Comparison

| Phase | Quick | Smoke | Architecture Audit | Comprehensive |
|-------|-------|-------|--------------------|---------------|
| Structure | Yes | Yes | Yes | Yes |
| Contracts | No | No | Yes | Yes |
| UI Health | No | No | Yes | Yes |
| Standards | Yes | Yes | Yes | Yes |
| Architecture | No | No | Yes | Yes |
| Dependencies | No | No | No | Yes |
| Quality | No | Yes | No | Yes |
| DOCS | Yes | Yes | Yes | Yes |
| Performance | No | No | No | Yes |
| Smoke | No | No | No | Yes |
| Unit | Yes | No | No | Yes |
| Integration | No | Yes | No | Yes |
| Playbooks | No | No | No | Yes |
| Business | Yes | Yes | No | Yes |
| Tidiness | No | No | No | Yes |
| Security | No | No | No | Yes |
| Measures | No | No | No | Yes |
| Proto | Yes | Yes | Yes | Yes |

## Custom Presets

Define custom presets in `.vrooli/testing.json`:

```json
{
  "presets": {
    "ci-fast": ["structure", "unit"],
    "nightly": ["structure", "dependencies", "unit", "integration", "business", "performance"]
  }
}
```
