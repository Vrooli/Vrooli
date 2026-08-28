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
| Contracts | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. | 2m |
| UI Health | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. Control-plane scope is intentionally excluded because this provider requires scenario UI surfaces and cannot meaningfully grade Go source. | 15m |
| API Health | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. Control-plane scope is intentionally excluded because this provider's contract is scenario API-specific and does not expose control-plane validation. | 2m |
| Architecture | Validates structural cohesion through architecture-cartographer. Control-plane scope is intentionally excluded because the provider's current graph and domain contract is scenario-rooted and does not expose control-plane validation. | 3m |
| Documentation | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. Control-plane scope is intentionally excluded because this provider validates documentation/team surfaces and does not expose control-plane validation. | 90s |
| Proto | Validates proto contracts through proto-health. | 2m |

### Comprehensive

Full validation before release or deployment.

```bash
test-genie execute my-scenario --preset comprehensive
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Portability | Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation. | 2m |
| Structure | Validates scenario skeleton and lifecycle wiring through structure-health. | 1m |
| Contracts | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. | 2m |
| UI Health | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. Control-plane scope is intentionally excluded because this provider requires scenario UI surfaces and cannot meaningfully grade Go source. | 15m |
| API Health | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. Control-plane scope is intentionally excluded because this provider's contract is scenario API-specific and does not expose control-plane validation. | 2m |
| Architecture | Validates structural cohesion through architecture-cartographer. Control-plane scope is intentionally excluded because the provider's current graph and domain contract is scenario-rooted and does not expose control-plane validation. | 3m |
| Dependencies | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. Control-plane scope is intentionally excluded because this descriptor's provider contract is scenario/package/resource dependency-specific and does not expose control-plane validation. | 15m |
| Quality | Validates static quality contracts, lint and type policy, and strict config through quality-health. | 2m |
| Documentation | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. Control-plane scope is intentionally excluded because this provider validates documentation/team surfaces and does not expose control-plane validation. | 90s |
| Long-form dictation soak | Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact. Control-plane scope is intentionally excluded because this provider validates a scenario browser/product path and cannot meaningfully grade Go source. | 15m |
| Performance | Validates API/UI build performance and Lighthouse budgets through performance-health. Control-plane scope is intentionally excluded because this provider requires scenario build/UI/runtime surfaces and cannot meaningfully grade the internal Go tree. | 5m |
| Unit | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. | 15m |
| Storage | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager. | 2m |
| Workflow | Validates BAS workflow assets and safe execution through workflow-health. Control-plane scope is intentionally excluded because this provider validates scenario browser workflows and cannot meaningfully grade Go source. | 15m |
| Business | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. Control-plane scope is intentionally excluded because this provider's contract is scenario product-intent specific and does not expose control-plane validation. | 2m |
| Experience | Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. Control-plane scope is intentionally excluded because experience evidence is scenario product-surface specific and does not expose control-plane validation. | 10m |
| Tidiness | Validates file and function quality checks through tidiness-manager. | 2m |
| Security | Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. Control-plane scope is intentionally excluded because this provider's current scanner contract resolves scenario roots and does not expose control-plane target validation. | 3m |
| Measures | Validates measures coverage and per-measure tiering through measures-health. Control-plane scope is intentionally excluded because the provider derives stateful scenario domains and measures, not generic Go-source validation. | 3m |
| Proto | Validates proto contracts through proto-health. | 2m |
| AI Conformance | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation. | 90s |
| Branding | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. Control-plane scope is intentionally excluded because branding evidence is scenario UI/assets-specific and cannot meaningfully grade Go source. | 2m |
| Monetization Conformance | Validates monetization trust boundaries, declarations, and local metering posture. Control-plane scope is intentionally excluded because monetization evidence is scenario product-specific and does not expose control-plane validation. | 90s |
| Search | Validates search-enabled scenarios through Search Hub's search maturity contract. Control-plane scope is intentionally excluded because search readiness is scenario/resource-specific and does not expose control-plane validation. | 90s |
| Provider Conformance | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. Control-plane scope is intentionally excluded because this provider validates phase descriptors and provider contracts, not the internal Go tree. | 90s |
| Component Tests | Runs version-pinned React component and hook contracts through the React Component Library provider. Control-plane scope is intentionally excluded because this provider validates React consumer surfaces and cannot meaningfully grade Go source. | 20m |
| Agent Conformance | Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation. | 45s |
| Templates | Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. Control-plane scope is intentionally excluded because template lineage is scenario-specific and does not expose control-plane validation. | 90s |
| Event Capture Conformance | Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. Control-plane scope is intentionally excluded because receipt-capture declarations are scenario-specific and do not expose control-plane validation. | 45s |

## Preset Comparison

| Phase | Quick | Smoke | Architecture Audit | Comprehensive |
|-------|-------|-------|--------------------|---------------|
| Portability | Adaptive | Adaptive | No | Yes |
| Structure | Adaptive | Adaptive | Yes | Yes |
| Contracts | Adaptive | Adaptive | Yes | Yes |
| UI Health | Adaptive | Adaptive | Yes | Yes |
| API Health | Adaptive | Adaptive | Yes | Yes |
| Architecture | Adaptive | Adaptive | Yes | Yes |
| Dependencies | Adaptive | Adaptive | No | Yes |
| Quality | Adaptive | Adaptive | No | Yes |
| Documentation | Adaptive | Adaptive | Yes | Yes |
| Long-form dictation soak | Adaptive | Adaptive | No | Yes |
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
| Monetization Conformance | Adaptive | Adaptive | No | Yes |
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
