# Test Presets Reference

Test Genie presets bundle effective registry phases for common validation loops. This document is generated from descriptor-backed phase specs plus Test Genie-owned preset declarations; edit provider `.vrooli/test-genie.json` descriptors or preset code instead of hand-editing these tables.

Timeout values are runtime budgets, not estimates. Runtime estimates are calculated from recent per-phase history when available.

## Available Presets

### Quick

Fast sanity check during development.

```bash
test-genie execute my-scenario --preset quick
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario skeleton and lifecycle wiring through structure-health. | 1m |
| DOCS | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Business | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. | 2m |
| Unit | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. | 15m |
| Proto | Validates proto contracts through proto-health. | 2m |

### Smoke

Core validation before pushing or handing off changes.

```bash
test-genie execute my-scenario --preset smoke
```

| Phase | Description | Timeout |
|-------|-------------|---------|
| Structure | Validates scenario skeleton and lifecycle wiring through structure-health. | 1m |
| API | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. | 2m |
| Quality | Validates static quality contracts, lint and type policy, and strict config through quality-health. | 2m |
| DOCS | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Business | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. | 2m |
| Proto | Validates proto contracts through proto-health. | 2m |

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
| API | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. | 2m |
| DOCS | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Architecture | Validates structural cohesion through architecture-cartographer. | 2m |
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
| API | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. | 2m |
| Architecture | Validates structural cohesion through architecture-cartographer. | 2m |
| Dependencies | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. | 15m |
| Quality | Validates static quality contracts, lint and type policy, and strict config through quality-health. | 2m |
| DOCS | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. | 1m |
| Performance | Validates API/UI build performance and Lighthouse budgets through performance-health. | 5m |
| Unit | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. | 15m |
| Storage | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-health. | 2m |
| Workflow | Validates BAS workflow assets and safe execution through workflow-health. | 15m |
| Business | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. | 2m |
| Tidiness | Validates file and function quality checks through tidiness-manager. | 2m |
| Security | Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. | 3m |
| Measures | Validates measures coverage and per-measure tiering through measures-health. | 3m |
| Proto | Validates proto contracts through proto-health. | 2m |
| Branding | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. | 2m |
| Search | Validates search-enabled scenarios through Search Hub's search maturity contract. | 90s |

## Preset Comparison

| Phase | Quick | Smoke | Architecture Audit | Comprehensive |
|-------|-------|-------|--------------------|---------------|
| Structure | Yes | Yes | Yes | Yes |
| Contracts | No | No | Yes | Yes |
| UI Health | No | No | Yes | Yes |
| API | No | Yes | Yes | Yes |
| Architecture | No | No | Yes | Yes |
| Dependencies | No | No | No | Yes |
| Quality | No | Yes | No | Yes |
| DOCS | Yes | Yes | Yes | Yes |
| Performance | No | No | No | Yes |
| Unit | Yes | No | No | Yes |
| Storage | No | No | No | Yes |
| Workflow | No | No | No | Yes |
| Business | Yes | Yes | No | Yes |
| Tidiness | No | No | No | Yes |
| Security | No | No | No | Yes |
| Measures | No | No | No | Yes |
| Proto | Yes | Yes | Yes | Yes |
| Branding | No | No | No | Yes |
| Search | No | No | No | Yes |

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
