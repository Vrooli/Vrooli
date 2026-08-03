# Test Presets Reference

Test Genie presets define common validation loops. Quick and smoke are adaptive budget profiles selected from applicable phases and recent measured history; architecture-audit and comprehensive have concrete registry-derived membership. This document is generated from descriptor-backed phase specs plus Test Genie-owned preset declarations; edit provider `.vrooli/test-genie.json` descriptors or preset/profile code instead of hand-editing these tables.

Timeout values are runtime budgets, not estimates. Runtime estimates are calculated from recent per-phase history when available. Use `test-genie phases plan <scenario> --preset <name>` to inspect selected and omitted phases before execution.

## Available Presets

### Quick

Fast sanity check during development.

```bash
test-genie execute my-scenario --preset quick
```

- Strategy: `budget_fast_feedback`
- Budget: 3m
- Candidates: applicable descriptor-backed phases after `.vrooli/testing.json` enablement and skip filters.
- Selection: required/gating phases first, then budget-fitting phases using conservative measured duration estimates.
- Omitted phases: reported by plan output with stable reason codes such as `omitted_budget_exceeded` or `omitted_unknown_estimate`.

### Smoke

Core validation before pushing or handing off changes.

```bash
test-genie execute my-scenario --preset smoke
```

- Strategy: `budget_smoke`
- Budget: 7m
- Candidates: applicable descriptor-backed phases after `.vrooli/testing.json` enablement and skip filters.
- Selection: required/gating phases first, then budget-fitting phases using conservative measured duration estimates.
- Omitted phases: reported by plan output with stable reason codes such as `omitted_budget_exceeded` or `omitted_unknown_estimate`.

### Architecture Audit

Surface conformance and architectural shape without runtime-heavy phases.

```bash
test-genie execute my-scenario --preset architecture-audit
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario skeleton and lifecycle wiring through structure-health. | 1m |
| Contracts | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. | 90s |
| UI Health | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. | 5m |
| API Health | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. | 2m |
| Architecture | Validates structural cohesion through architecture-cartographer. | 2m |
| Documentation | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Proto | Validates proto contracts through proto-health. | 2m |

### Comprehensive

Full validation before release or deployment.

```bash
test-genie execute my-scenario --preset comprehensive
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario skeleton and lifecycle wiring through structure-health. | 1m |
| Contracts | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. | 90s |
| UI Health | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. | 5m |
| API Health | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. | 2m |
| Architecture | Validates structural cohesion through architecture-cartographer. | 2m |
| Dependencies | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. | 15m |
| Quality | Validates static quality contracts, lint and type policy, and strict config through quality-health. | 2m |
| Documentation | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Performance | Validates API/UI build performance and Lighthouse budgets through performance-health. | 5m |
| Unit | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. | 15m |
| Storage | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager. | 2m |
| Workflow | Validates BAS workflow assets and safe execution through workflow-health. | 15m |
| Business | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. | 2m |
| Experience | Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. | 2m |
| Tidiness | Validates file and function quality checks through tidiness-manager. | 2m |
| Security | Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. | 3m |
| Measures | Validates measures coverage and per-measure tiering through measures-health. | 3m |
| Proto | Validates proto contracts through proto-health. | 2m |
| AI Conformance | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. | 90s |
| Branding | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. | 2m |
| Search | Validates search-enabled scenarios through Search Hub's search maturity contract. | 90s |
| Provider Conformance | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. | 90s |
| Component Tests | Runs version-pinned React component and hook contracts through the React Component Library provider. | 5m |
| Agent Conformance | Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. | 45s |
| Templates | Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. | 90s |
| Event Capture Conformance | Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. | 45s |

## Preset Comparison

| Phase | Quick | Smoke | Architecture Audit | Comprehensive |
|-------|-------|-------|--------------------|---------------|
| Structure | Adaptive | Adaptive | Yes | Yes |
| Contracts | Adaptive | Adaptive | Yes | Yes |
| UI Health | Adaptive | Adaptive | Yes | Yes |
| API Health | Adaptive | Adaptive | Yes | Yes |
| Architecture | Adaptive | Adaptive | Yes | Yes |
| Dependencies | Adaptive | Adaptive | No | Yes |
| Quality | Adaptive | Adaptive | No | Yes |
| Documentation | Adaptive | Adaptive | Yes | Yes |
| Performance | Adaptive | Adaptive | No | Yes |
| Unit | Adaptive | Adaptive | No | Yes |
| Storage | Adaptive | Adaptive | No | Yes |
| Workflow | Adaptive | Adaptive | No | Yes |
| Business | Adaptive | Adaptive | No | Yes |
| Experience | Adaptive | Adaptive | No | Yes |
| Tidiness | Adaptive | Adaptive | No | Yes |
| Security | Adaptive | Adaptive | No | Yes |
| Measures | Adaptive | Adaptive | No | Yes |
| Proto | Adaptive | Adaptive | Yes | Yes |
| AI Conformance | Adaptive | Adaptive | No | Yes |
| Branding | Adaptive | Adaptive | No | Yes |
| Search | Adaptive | Adaptive | No | Yes |
| Provider Conformance | Adaptive | Adaptive | No | Yes |
| Component Tests | Adaptive | Adaptive | No | Yes |
| Agent Conformance | Adaptive | Adaptive | No | Yes |
| Templates | Adaptive | Adaptive | No | Yes |
| Event Capture Conformance | Adaptive | Adaptive | No | Yes |

## Custom Presets

Define custom presets in `.vrooli/testing.json`:

```json
{
  "presets": {
    "ci-fast": ["structure", "unit"],
    "nightly": ["structure", "dependencies", "unit", "business", "performance"]
  }
}
```
