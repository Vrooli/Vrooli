# Test Genie Phases

Test Genie phases are generated from the effective descriptor-backed registry. Provider-backed phase metadata lives in each provider's `.vrooli/test-genie.json`; Test Genie code owns runner bindings, preset composition, and registry validation.

Use `test-genie phases inspect <phase> --json` or `/api/v1/phases/<phase>` to inspect the effective descriptor projection, including provider, descriptor path, docs path, policy, runnability, applicability vocabulary, freshness requirement, profile membership, phase/runtime class, dimensions, and finding source. Provider descriptors must declare `docs.path`; retired `.vrooli/maturity.json` files are rejected.

## Phase Summary

| Order | Phase | Timeout | Selection | Provider Readiness | Gating | Runtime | Source | Purpose |
|-------|-------|---------|-----------|--------------------|--------|---------|--------|---------|
| 1 | [Structure](structure/README.md) | 1m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario skeleton and lifecycle wiring through structure-health. |
| 2 | [Contracts](contracts/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. |
| 3 | [UI Health](ui-health/README.md) | 5m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. |
| 4 | [API Health](api/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. |
| 5 | [Architecture](architecture/README.md) | 2m | `default_when_applicable` | `best_effort` | `high_confidence_gating` | No | validation-provider | Validates structural cohesion through architecture-cartographer. |
| 6 | [Dependencies](dependencies/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. |
| 7 | [Quality](quality/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates static quality contracts, lint and type policy, and strict config through quality-health. |
| 8 | [Documentation](docs/README.md) | 1m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. |
| 9 | [Performance](performance/README.md) | 5m | `default_when_applicable` | `best_effort` | `gating` | Yes | validation-provider | Validates API/UI build performance and Lighthouse budgets through performance-health. |
| 10 | [Unit](unit/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. |
| 11 | [Storage](storage/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-health. |
| 12 | [Workflow](workflow/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates BAS workflow assets and safe execution through workflow-health. |
| 13 | [Business](business/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. |
| 14 | [Experience](experience/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. |
| 15 | [Tidiness](tidiness/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates file and function quality checks through tidiness-manager. |
| 16 | [Security](security/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. |
| 17 | [Measures](measures/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates measures coverage and per-measure tiering through measures-health. |
| 18 | [Proto](proto/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates proto contracts through proto-health. |
| 19 | [AI Conformance](ai-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. |
| 20 | [Branding](branding/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. |
| 21 | [Search](search/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates search-enabled scenarios through Search Hub's search maturity contract. |
| 22 | [Provider Conformance](provider-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. |

## Static Phases

- [Structure](structure/README.md) - Validates scenario skeleton and lifecycle wiring through structure-health.
- [Contracts](contracts/README.md) - Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health.
- [API Health](api/README.md) - Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health.
- [Architecture](architecture/README.md) - Validates structural cohesion through architecture-cartographer.
- [Dependencies](dependencies/README.md) - Validates dependency readiness, governance, runtime status, release-age policy, and graph drift.
- [Quality](quality/README.md) - Validates static quality contracts, lint and type policy, and strict config through quality-health.
- [Documentation](docs/README.md) - Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory.
- [Unit](unit/README.md) - Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health.
- [Storage](storage/README.md) - Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-health.
- [Business](business/README.md) - Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health.
- [Experience](experience/README.md) - Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager.
- [Tidiness](tidiness/README.md) - Validates file and function quality checks through tidiness-manager.
- [Security](security/README.md) - Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health.
- [Measures](measures/README.md) - Validates measures coverage and per-measure tiering through measures-health.
- [Proto](proto/README.md) - Validates proto contracts through proto-health.
- [AI Conformance](ai-conformance/README.md) - Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness.
- [Branding](branding/README.md) - Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager.
- [Search](search/README.md) - Validates search-enabled scenarios through Search Hub's search maturity contract.
- [Provider Conformance](provider-conformance/README.md) - Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance.

## Runtime Phases

- [UI Health](ui-health/README.md) - Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health.
- [Performance](performance/README.md) - Validates API/UI build performance and Lighthouse budgets through performance-health.
- [Workflow](workflow/README.md) - Validates BAS workflow assets and safe execution through workflow-health.

## Running Phases

```bash
test-genie execute my-scenario --phases structure,unit
test-genie execute my-scenario --preset comprehensive
```

## Configuration

Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).

## Presets

Preset and profile definitions are documented in [Presets Reference](../reference/presets.md). Quick and smoke are adaptive profiles; concrete preset membership is generated from the effective registry.
