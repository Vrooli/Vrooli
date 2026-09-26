# Test Genie Phases

Test Genie phases are generated from the effective descriptor-backed registry. Provider-backed phase metadata lives in each provider's `.vrooli/test-genie.json`; Test Genie code owns runner bindings, preset composition, and registry validation.

Use `test-genie phases inspect <phase> --json` or `/api/v1/phases/<phase>` to inspect the effective descriptor projection, including provider, descriptor path, docs path, policy, runnability, applicability vocabulary, freshness requirement, profile membership, phase/runtime class, dimensions, and finding source. Provider descriptors must declare `docs.path`; retired `.vrooli/maturity.json` files are rejected.

## The Phase Capability Contract

Every descriptor declares a **[Phase Capability Contract](../concepts/phase-capability-contract.md)**: a first-class North Star, a gated L0–L4 ladder, a provider-returned standing, and a structured remediation doc. The `docs.path` target must follow the fixed [remediation-doc skeleton](../concepts/phase-capability-contract.md#the-remediation-doc-skeleton) — the five H2 headings `North Star` / `The rungs and their gates` / `What each finding means` / `The canonical fix` / `How to verify` — so a doc-search topic emitted in run output resolves to the exact remediation section with no per-descriptor glue. The contract is validated by the provider-conformance contract checks. Every catalog entry's contract posture is tracked in the drift-guarded [capability-contract inventory](capability-contract-inventory.md). Seed a new descriptor's skeleton with `test-genie phases scaffold <name>`.

## Phase Summary

| Order | Phase | Timeout | Selection | Provider Readiness | Gating | Runtime | Source | Purpose |
|-------|-------|---------|-----------|--------------------|--------|---------|--------|---------|
| 1 | [Portability](portability/README.md) | 2m | `default_when_applicable` | `none` | `gating` | No | validation-provider | Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation. |
| 2 | [Structure](structure/README.md) | 1m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario skeleton and lifecycle wiring through structure-health. |
| 3 | [Contracts](contracts/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. |
| 4 | [Code Facts](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates target-aware code evidence and fact provenance. |
| 5 | [Go Code Graph](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates live Go graph extraction for declared targets. |
| 6 | [TypeScript Code Graph](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates live TypeScript graph extraction for declared targets. |
| 7 | [UI Health](ui-health/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. |
| 8 | [API Health](api/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. |
| 9 | [Architecture](architecture/README.md) | 3m | `default_when_applicable` | `best_effort` | `high_confidence_gating` | No | validation-provider | Validates structural cohesion through architecture-cartographer. |
| 10 | [Dependencies](dependencies/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. |
| 11 | [Quality](quality/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates static quality contracts, lint and type policy, and strict config through quality-health. |
| 12 | [Documentation](docs/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. |
| 13 | [Long-form dictation soak](../audio-tools/docs/test-genie/soak/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact. |
| 14 | [Performance](performance/README.md) | 5m | `default_when_applicable` | `best_effort` | `gating` | Yes | validation-provider | Validates API/UI build performance and Lighthouse budgets through performance-health. |
| 15 | [Unit](unit/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. |
| 16 | [Storage](storage/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager. |
| 17 | [Workflow](workflow/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates BAS workflow assets and safe execution through workflow-health. |
| 18 | [Business](business/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. |
| 19 | [Experience](experience/README.md) | 10m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. |
| 20 | [Tidiness](tidiness/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates file and function quality checks through tidiness-manager. |
| 21 | [Security](security/README.md) | 10m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates scenarios and the control plane for secrets, Go SAST, Go vulnerability data, and JavaScript dependency risk through one path-first Security Health scanner pipeline. |
| 22 | [Measures](measures/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates measures coverage and per-measure tiering through measures-health. |
| 23 | [Proto](proto/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates proto contracts through proto-health. |
| 24 | [AI Conformance](ai-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. |
| 25 | [Branding](branding/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. |
| 26 | [Monetization Conformance](monetization-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates monetization trust boundaries, declarations, and local metering posture. |
| 27 | [Search](search/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates search-enabled scenarios through Search Hub's search maturity contract. |
| 28 | [Provider Conformance](provider-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. |
| 29 | [Component Tests](component-tests/README.md) | 20m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs version-pinned React component and hook contracts through the React Component Library provider. |
| 30 | [Agent Conformance](agent-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. |
| 31 | [Templates](templates/README.md) | 90s | `comprehensive_when_applicable` | `required_when_applicable` | `advisory` | Yes | validation-provider | Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. |
| 32 | [Event Capture Conformance](event-capture-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. |

## Static Phases

- [Portability](portability/README.md) - Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation.
- [Structure](structure/README.md) - Validates scenario skeleton and lifecycle wiring through structure-health.
- [Contracts](contracts/README.md) - Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health.
- [API Health](api/README.md) - Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health.
- [Architecture](architecture/README.md) - Validates structural cohesion through architecture-cartographer.
- [Dependencies](dependencies/README.md) - Validates dependency readiness, governance, runtime status, release-age policy, and graph drift.
- [Quality](quality/README.md) - Validates static quality contracts, lint and type policy, and strict config through quality-health.
- [Documentation](docs/README.md) - Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory.
- [Unit](unit/README.md) - Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health.
- [Storage](storage/README.md) - Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager.
- [Business](business/README.md) - Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health.
- [Experience](experience/README.md) - Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager.
- [Tidiness](tidiness/README.md) - Validates file and function quality checks through tidiness-manager.
- [Security](security/README.md) - Validates scenarios and the control plane for secrets, Go SAST, Go vulnerability data, and JavaScript dependency risk through one path-first Security Health scanner pipeline.
- [Measures](measures/README.md) - Validates measures coverage and per-measure tiering through measures-health.
- [Proto](proto/README.md) - Validates proto contracts through proto-health.
- [AI Conformance](ai-conformance/README.md) - Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness.
- [Branding](branding/README.md) - Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager.
- [Search](search/README.md) - Validates search-enabled scenarios through Search Hub's search maturity contract.
- [Provider Conformance](provider-conformance/README.md) - Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance.
- [Agent Conformance](agent-conformance/README.md) - Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles.

## Runtime Phases

- [Code Facts](architecture/README.md) - Validates target-aware code evidence and fact provenance.
- [Go Code Graph](architecture/README.md) - Validates live Go graph extraction for declared targets.
- [TypeScript Code Graph](architecture/README.md) - Validates live TypeScript graph extraction for declared targets.
- [UI Health](ui-health/README.md) - Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health.
- [Long-form dictation soak](../audio-tools/docs/test-genie/soak/README.md) - Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact.
- [Performance](performance/README.md) - Validates API/UI build performance and Lighthouse budgets through performance-health.
- [Workflow](workflow/README.md) - Validates BAS workflow assets and safe execution through workflow-health.
- [Monetization Conformance](monetization-conformance/README.md) - Validates monetization trust boundaries, declarations, and local metering posture.
- [Component Tests](component-tests/README.md) - Runs version-pinned React component and hook contracts through the React Component Library provider.
- [Templates](templates/README.md) - Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager.
- [Event Capture Conformance](event-capture-conformance/README.md) - Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot.

## Running Phases

```bash
test-genie execute my-scenario --phases <descriptor-a>,<descriptor-b>
test-genie execute my-scenario --preset comprehensive
```

## Configuration

Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).

## Presets

Preset and profile definitions are documented in [Presets Reference](../reference/presets.md). Quick and smoke are adaptive profiles; concrete preset membership is generated from the effective registry.
