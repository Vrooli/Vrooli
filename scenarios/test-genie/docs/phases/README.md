# Test Genie Phases

Test Genie phases are generated from the effective descriptor-backed registry. Provider-backed phase metadata lives in each provider's `.vrooli/test-genie.json`; Test Genie code owns runner bindings, preset composition, and registry validation.

Use `test-genie phases inspect <phase> --json` or `/api/v1/phases/<phase>` to inspect the effective descriptor projection, including provider, descriptor path, docs path, policy, runnability, applicability vocabulary, freshness requirement, profile membership, phase/runtime class, dimensions, and finding source. Provider descriptors must declare `docs.path`; retired `.vrooli/maturity.json` files are rejected.

## Target-kind applicability

Every provider must account for each fleet target kind it is expected to
consider. A supported kind belongs in `targets.kinds`. A kind that cannot be
graded belongs in `targets.notApplicableKinds` with an object containing
`kind` and a reason of at least 40 characters. The reason must name the input
or surface the provider needs and the target lacks. A kind cannot appear in
both arrays. Omitting both declarations is an unreviewed coverage gap and is
rejected by the control-plane debt census.

```json
{
  "targets": {
    "kinds": ["scenario"],
    "selection": "enumerate",
    "notApplicableKinds": [
      {
        "kind": "control-plane",
        "reason": "This provider requires a scenario UI manifest; the control plane has no browser product surface."
      }
    ]
  }
}
```

## The Phase Capability Contract

Every descriptor declares a **[Phase Capability Contract](../concepts/phase-capability-contract.md)**: a first-class North Star, a gated L0–L4 ladder, a provider-returned standing, and a structured remediation doc. The `docs.path` target must follow the fixed [remediation-doc skeleton](../concepts/phase-capability-contract.md#the-remediation-doc-skeleton) — the five H2 headings `North Star` / `The rungs and their gates` / `What each finding means` / `The canonical fix` / `How to verify` — so a doc-search topic emitted in run output resolves to the exact remediation section with no per-descriptor glue. The contract is validated by the provider-conformance contract checks. Every catalog entry's contract posture is tracked in the drift-guarded [capability-contract inventory](capability-contract-inventory.md). Seed a new descriptor's skeleton with `test-genie phases scaffold <name>`.

## Phase Summary

| Order | Phase | Timeout | Selection | Provider Readiness | Gating | Runtime | Source | Purpose |
|-------|-------|---------|-----------|--------------------|--------|---------|--------|---------|
| 1 | [Portability](portability/README.md) | 2m | `default_when_applicable` | `none` | `gating` | No | validation-provider | Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation. |
| 2 | [Structure](structure/README.md) | 1m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario skeleton and lifecycle wiring through structure-health. |
| 3 | [Contracts](contracts/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. |
| 4 | [UI Health](ui-health/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. Control-plane scope is intentionally excluded because this provider requires scenario UI surfaces and cannot meaningfully grade Go source. |
| 5 | [API Health](api/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. Control-plane scope is intentionally excluded because this provider's contract is scenario API-specific and does not expose control-plane validation. |
| 6 | [Architecture](architecture/README.md) | 3m | `default_when_applicable` | `best_effort` | `high_confidence_gating` | No | validation-provider | Validates structural cohesion through architecture-cartographer. Control-plane scope is intentionally excluded because the provider's current graph and domain contract is scenario-rooted and does not expose control-plane validation. |
| 7 | [Dependencies](dependencies/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. Control-plane scope is intentionally excluded because this descriptor's provider contract is scenario/package/resource dependency-specific and does not expose control-plane validation. |
| 8 | [Quality](quality/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates static quality contracts, lint and type policy, and strict config through quality-health. |
| 9 | [Documentation](docs/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. Control-plane scope is intentionally excluded because this provider validates documentation/team surfaces and does not expose control-plane validation. |
| 10 | [Long-form dictation soak](../audio-tools/docs/test-genie/soak/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact. Control-plane scope is intentionally excluded because this provider validates a scenario browser/product path and cannot meaningfully grade Go source. |
| 11 | [Performance](performance/README.md) | 5m | `default_when_applicable` | `best_effort` | `gating` | Yes | validation-provider | Validates API/UI build performance and Lighthouse budgets through performance-health. Control-plane scope is intentionally excluded because this provider requires scenario build/UI/runtime surfaces and cannot meaningfully grade the internal Go tree. |
| 12 | [Unit](unit/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. |
| 13 | [Storage](storage/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager. |
| 14 | [Workflow](workflow/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates BAS workflow assets and safe execution through workflow-health. Control-plane scope is intentionally excluded because this provider validates scenario browser workflows and cannot meaningfully grade Go source. |
| 15 | [Business](business/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. Control-plane scope is intentionally excluded because this provider's contract is scenario product-intent specific and does not expose control-plane validation. |
| 16 | [Experience](experience/README.md) | 10m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. Control-plane scope is intentionally excluded because experience evidence is scenario product-surface specific and does not expose control-plane validation. |
| 17 | [Tidiness](tidiness/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates file and function quality checks through tidiness-manager. |
| 18 | [Security](security/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. Control-plane scope is intentionally excluded because this provider's current scanner contract resolves scenario roots and does not expose control-plane target validation. |
| 19 | [Measures](measures/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates measures coverage and per-measure tiering through measures-health. Control-plane scope is intentionally excluded because the provider derives stateful scenario domains and measures, not generic Go-source validation. |
| 20 | [Proto](proto/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates proto contracts through proto-health. |
| 21 | [AI Conformance](ai-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation. |
| 22 | [Branding](branding/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. Control-plane scope is intentionally excluded because branding evidence is scenario UI/assets-specific and cannot meaningfully grade Go source. |
| 23 | [Monetization Conformance](monetization-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates monetization trust boundaries, declarations, and local metering posture. Control-plane scope is intentionally excluded because monetization evidence is scenario product-specific and does not expose control-plane validation. |
| 24 | [Search](search/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates search-enabled scenarios through Search Hub's search maturity contract. Control-plane scope is intentionally excluded because search readiness is scenario/resource-specific and does not expose control-plane validation. |
| 25 | [Provider Conformance](provider-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. Control-plane scope is intentionally excluded because this provider validates phase descriptors and provider contracts, not the internal Go tree. |
| 26 | [Component Tests](component-tests/README.md) | 20m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs version-pinned React component and hook contracts through the React Component Library provider. Control-plane scope is intentionally excluded because this provider validates React consumer surfaces and cannot meaningfully grade Go source. |
| 27 | [Agent Conformance](agent-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation. |
| 28 | [Templates](templates/README.md) | 90s | `comprehensive_when_applicable` | `required_when_applicable` | `advisory` | Yes | validation-provider | Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. Control-plane scope is intentionally excluded because template lineage is scenario-specific and does not expose control-plane validation. |
| 29 | [Event Capture Conformance](event-capture-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. Control-plane scope is intentionally excluded because receipt-capture declarations are scenario-specific and do not expose control-plane validation. |

## Static Phases

- [Portability](portability/README.md) - Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation.
- [Structure](structure/README.md) - Validates scenario skeleton and lifecycle wiring through structure-health.
- [Contracts](contracts/README.md) - Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health.
- [API Health](api/README.md) - Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. Control-plane scope is intentionally excluded because this provider's contract is scenario API-specific and does not expose control-plane validation.
- [Architecture](architecture/README.md) - Validates structural cohesion through architecture-cartographer. Control-plane scope is intentionally excluded because the provider's current graph and domain contract is scenario-rooted and does not expose control-plane validation.
- [Dependencies](dependencies/README.md) - Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. Control-plane scope is intentionally excluded because this descriptor's provider contract is scenario/package/resource dependency-specific and does not expose control-plane validation.
- [Quality](quality/README.md) - Validates static quality contracts, lint and type policy, and strict config through quality-health.
- [Documentation](docs/README.md) - Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. Control-plane scope is intentionally excluded because this provider validates documentation/team surfaces and does not expose control-plane validation.
- [Unit](unit/README.md) - Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health.
- [Storage](storage/README.md) - Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager.
- [Business](business/README.md) - Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. Control-plane scope is intentionally excluded because this provider's contract is scenario product-intent specific and does not expose control-plane validation.
- [Experience](experience/README.md) - Validates scenario-experience-spec/v1 contracts and experience maturity through experience-manager. Control-plane scope is intentionally excluded because experience evidence is scenario product-surface specific and does not expose control-plane validation.
- [Tidiness](tidiness/README.md) - Validates file and function quality checks through tidiness-manager.
- [Security](security/README.md) - Validates secrets, Go SAST, Go vulnerability data, and JavaScript dependencies through security-health. Control-plane scope is intentionally excluded because this provider's current scanner contract resolves scenario roots and does not expose control-plane target validation.
- [Measures](measures/README.md) - Validates measures coverage and per-measure tiering through measures-health. Control-plane scope is intentionally excluded because the provider derives stateful scenario domains and measures, not generic Go-source validation.
- [Proto](proto/README.md) - Validates proto contracts through proto-health.
- [AI Conformance](ai-conformance/README.md) - Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation.
- [Branding](branding/README.md) - Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. Control-plane scope is intentionally excluded because branding evidence is scenario UI/assets-specific and cannot meaningfully grade Go source.
- [Search](search/README.md) - Validates search-enabled scenarios through Search Hub's search maturity contract. Control-plane scope is intentionally excluded because search readiness is scenario/resource-specific and does not expose control-plane validation.
- [Provider Conformance](provider-conformance/README.md) - Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. Control-plane scope is intentionally excluded because this provider validates phase descriptors and provider contracts, not the internal Go tree.
- [Agent Conformance](agent-conformance/README.md) - Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. Control-plane scope is intentionally excluded because this provider's contract is scenario-consumer specific and does not expose control-plane validation.

## Runtime Phases

- [UI Health](ui-health/README.md) - Validates UI manifests, interop, standards, and BAS runtime evidence through ui-health. Control-plane scope is intentionally excluded because this provider requires scenario UI surfaces and cannot meaningfully grade Go source.
- [Long-form dictation soak](../audio-tools/docs/test-genie/soak/README.md) - Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact. Control-plane scope is intentionally excluded because this provider validates a scenario browser/product path and cannot meaningfully grade Go source.
- [Performance](performance/README.md) - Validates API/UI build performance and Lighthouse budgets through performance-health. Control-plane scope is intentionally excluded because this provider requires scenario build/UI/runtime surfaces and cannot meaningfully grade the internal Go tree.
- [Workflow](workflow/README.md) - Validates BAS workflow assets and safe execution through workflow-health. Control-plane scope is intentionally excluded because this provider validates scenario browser workflows and cannot meaningfully grade Go source.
- [Monetization Conformance](monetization-conformance/README.md) - Validates monetization trust boundaries, declarations, and local metering posture. Control-plane scope is intentionally excluded because monetization evidence is scenario product-specific and does not expose control-plane validation.
- [Component Tests](component-tests/README.md) - Runs version-pinned React component and hook contracts through the React Component Library provider. Control-plane scope is intentionally excluded because this provider validates React consumer surfaces and cannot meaningfully grade Go source.
- [Templates](templates/README.md) - Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. Control-plane scope is intentionally excluded because template lineage is scenario-specific and does not expose control-plane validation.
- [Event Capture Conformance](event-capture-conformance/README.md) - Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. Control-plane scope is intentionally excluded because receipt-capture declarations are scenario-specific and do not expose control-plane validation.

## Running Phases

```bash
test-genie execute my-scenario --phases <descriptor-a>,<descriptor-b>
test-genie execute my-scenario --preset comprehensive
```

## Configuration

Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).

## Presets

Preset and profile definitions are documented in [Presets Reference](../reference/presets.md). Quick and smoke are adaptive profiles; concrete preset membership is generated from the effective registry.
