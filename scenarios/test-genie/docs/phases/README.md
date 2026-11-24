# Test Genie Phases

Test Genie phases are generated from the effective descriptor-backed registry. Provider-backed phase metadata lives in each provider's `.vrooli/test-genie.json`; Test Genie code owns runner bindings, preset composition, and registry validation.

Use `test-genie phases inspect "unit" --json` or `/api/v1/phases/unit` to inspect the effective descriptor projection, including provider, descriptor path, docs path, policy, runnability, applicability vocabulary, freshness requirement, profile membership, phase/runtime class, dimensions, and finding source. Provider descriptors must declare `docs.path`; retired `.vrooli/maturity.json` files are rejected.

## The Phase Capability Contract

Every descriptor declares a **[Phase Capability Contract](../concepts/phase-capability-contract.md)**: a first-class North Star, a gated L0–L4 ladder, a provider-returned standing, and a structured remediation doc. The `docs.path` target must follow the fixed [remediation-doc skeleton](../concepts/phase-capability-contract.md#the-remediation-doc-skeleton) — the five H2 headings `North Star` / `The rungs and their gates` / `What each finding means` / `The canonical fix` / `How to verify` — so a doc-search topic emitted in run output resolves to the exact remediation section with no per-descriptor glue. The contract is validated by the provider-conformance contract checks. Every catalog entry's contract posture is tracked in the drift-guarded [capability-contract inventory](capability-contract-inventory.md). Seed a new descriptor's skeleton with `test-genie phases scaffold <name>`.

## Phase Summary

| Order | Phase | Timeout | Selection | Provider Readiness | Gating | Runtime | Source | Purpose |
|-------|-------|---------|-----------|--------------------|--------|---------|--------|---------|
| 1 | [Portability](portability/README.md) | 2m | `default_when_applicable` | `none` | `gating` | No | validation-provider | Runs the deployability resolver against declared resource inputs and the observed host OS. Control-plane scope is intentionally excluded because this provider's contract is scenario/resource deployability-specific and does not expose control-plane validation. |
| 2 | [Structure](structure/README.md) | 1m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates scenario skeleton and lifecycle wiring through structure-health. |
| 3 | [Contracts](contracts/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates CLI manifest, proto bindings, and runtime CLI surface through cli-health. |
| 4 | [Code Facts](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates target-aware code evidence and fact provenance. |
| 5 | [Experience](experience/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Owns the single served-UI experience verdict: static UI contracts plus runtime evaluation and evidence. |
| 6 | [Go Code Graph](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates live Go graph extraction for declared targets. |
| 7 | [TypeScript Code Graph](architecture/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates live TypeScript graph extraction for declared targets. |
| 8 | [API Health](api/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates API readiness, health contracts, route semantics, and runtime hygiene through api-health. |
| 9 | [Architecture](architecture/README.md) | 3m | `default_when_applicable` | `best_effort` | `high_confidence_gating` | No | validation-provider | Validates structural cohesion through architecture-cartographer. |
| 10 | [Dependencies](dependencies/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates dependency readiness, governance, runtime status, release-age policy, and graph drift. |
| 11 | [Quality](quality/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates static quality contracts, lint and type policy, and strict config through quality-health. |
| 12 | [Documentation](docs/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates documentation Markdown, Mermaid, links, references, and manifests through knowledge-observatory. |
| 13 | [Long-form dictation soak](/scenarios/audio-tools/docs/internal/soak-qualification.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact. |
| 14 | [Performance](performance/README.md) | 5m | `default_when_applicable` | `best_effort` | `gating` | Yes | validation-provider | Validates API/UI build performance and Lighthouse budgets through performance-health. |
| 15 | [Unit](unit/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates test execution, coverage, architecture, quality, and runtime diagnostics through unit-health. |
| 16 | [Storage](storage/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates storage conventions, migration hygiene, persistence seams, and test isolation through storage-manager. |
| 17 | [Workflow](workflow/README.md) | 15m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates BAS workflow assets and safe execution through workflow-health. |
| 18 | [Business](business/README.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates PRD, requirements registry, OT linkage, and evidence traceability through business-health. |
| 19 | [Channel Conformance](/scenarios/switchboard/docs/internal/PROBLEMS.md) | 2m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the shared seven-case adapter contract for every registered Switchboard channel. |
| 20 | [Experience Contract](experience-contract/README.md) | 10m | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates authored scenario-experience-spec/v1 contracts and their maturity; served-UI verdicts belong to the experience phase. |
| 21 | [Tidiness](tidiness/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates file and function quality checks through tidiness-manager. |
| 22 | [Security](security/README.md) | 10m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates scenarios and the control plane for secrets, Go SAST, Go vulnerability data, and JavaScript dependency risk through one path-first Security Health scanner pipeline. |
| 23 | [Measures](measures/README.md) | 3m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates measures coverage and per-measure tiering through measures-health. |
| 24 | [Programs](programs/README.md) | 6m | `default_when_applicable` | `best_effort` | `advisory` | No | validation-provider | Validates declared program contracts, sources, and fixture evidence. |
| 25 | [Proto](proto/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates proto contracts through proto-health. |
| 26 | [AI Conformance](ai-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness. |
| 27 | [Branding](branding/README.md) | 2m | `default_when_applicable` | `best_effort` | `gating` | No | validation-provider | Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager. |
| 28 | [Skill Set](skill-set/README.md) | 90s | `default_when_applicable` | `best_effort` | `advisory` | No | validation-provider | Validates scenario skill declarations and their reachable sources. |
| 29 | [Monetization Conformance](monetization-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates monetization trust boundaries, declarations, and meter admission/reporting posture. |
| 30 | [Search](search/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates search-enabled scenarios through Search Hub's search maturity contract. |
| 31 | [Provider Conformance](provider-conformance/README.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance. |
| 32 | [Stripe Test Outcomes](stripe-test-outcomes/README.md) | 30m | `never_by_default` | `required_when_applicable` | `gating` | Yes | validation-provider | Reconciles the LPBS test-mode catalog against Stripe and executes payment outcome journeys with durable provider evidence. |
| 33 | [Cloud Production Health](cloud-production-health/README.md) | 30m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the CL3 owner contract suites and fails closed until staging and production showcase receipts exist. |
| 34 | [Component Tests](component-tests/README.md) | 20m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs version-pinned React component and hook contracts through the React Component Library provider. |
| 35 | [React Consumer Contracts](consumer-contract/README.md) | 10m | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Runs the usage-contract fleet lint and downward-only per-contract ratchet across every React UI in the repository. |
| 36 | [Paid Credential Leak Proof](/scenarios/landing-page-business-suite/docs/testing/credential-leak-proof.md) | 30m | `never_by_default` | `required_when_applicable` | `gating` | Yes | validation-provider | Uses synthetic canaries derived only from declared credential identity/field/scope metadata, runs LPBS and migrated service-owner identity contracts for audience, scope, source, versioned overlap, outage, restart and recovery, and scans every bounded subprocess output before returning metadata-only receipts. |
| 37 | [Agent Conformance](agent-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | No | validation-provider | Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles. |
| 38 | [Templates](templates/README.md) | 90s | `comprehensive_when_applicable` | `required_when_applicable` | `advisory` | Yes | validation-provider | Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager. |
| 39 | [Event Capture Conformance](event-capture-conformance/README.md) | 45s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot. |
| 40 | [Library Fault Injection](/scenarios/program-runtime/docs/phases/fault-injection.md) | 4m | `explicit_only` | `required_when_applicable` | `gating` | No | validation-provider | Runs Data Backup Manager recovery simulators and a generated library-wide Program Runtime child-envelope matrix with per-program capability requirements and observable fallbacks. |
| 41 | [Local Model Evaluation](/scenarios/ai-gateway/docs/operations/model-eval.md) | 90s | `default_when_applicable` | `required_when_applicable` | `gating` | Yes | validation-provider | Censuses observed runtime models and checks independently reviewed retrieval, policy-role, gateway-task-role and real consumer evidence, then remains blocked until separate LM2 inference admission allows execution. |

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
- [Experience Contract](experience-contract/README.md) - Validates authored scenario-experience-spec/v1 contracts and their maturity; served-UI verdicts belong to the experience phase.
- [Tidiness](tidiness/README.md) - Validates file and function quality checks through tidiness-manager.
- [Security](security/README.md) - Validates scenarios and the control plane for secrets, Go SAST, Go vulnerability data, and JavaScript dependency risk through one path-first Security Health scanner pipeline.
- [Measures](measures/README.md) - Validates measures coverage and per-measure tiering through measures-health.
- [Programs](programs/README.md) - Validates declared program contracts, sources, and fixture evidence.
- [Proto](proto/README.md) - Validates proto contracts through proto-health.
- [AI Conformance](ai-conformance/README.md) - Validates AI-using scenarios for provider-neutral routing, resource boundary hygiene, embedding metadata safety, and AI Gateway adoption readiness.
- [Branding](branding/README.md) - Validates brand identity, design tokens, typography, logos, favicons, contrast, and applied brand markers through brand-manager.
- [Skill Set](skill-set/README.md) - Validates scenario skill declarations and their reachable sources.
- [Search](search/README.md) - Validates search-enabled scenarios through Search Hub's search maturity contract.
- [Provider Conformance](provider-conformance/README.md) - Validates Test Genie phase-provider descriptors: descriptor structure, embedded maturity, policy safety, stale-file absence, and live provider-contract conformance.
- [Agent Conformance](agent-conformance/README.md) - Validates that coding-agent consumers use Agent Manager through declared, portable role-based profiles.
- [Library Fault Injection](/scenarios/program-runtime/docs/phases/fault-injection.md) - Runs Data Backup Manager recovery simulators and a generated library-wide Program Runtime child-envelope matrix with per-program capability requirements and observable fallbacks.

## Runtime Phases

- [Code Facts](architecture/README.md) - Validates target-aware code evidence and fact provenance.
- [Experience](experience/README.md) - Owns the single served-UI experience verdict: static UI contracts plus runtime evaluation and evidence.
- [Go Code Graph](architecture/README.md) - Validates live Go graph extraction for declared targets.
- [TypeScript Code Graph](architecture/README.md) - Validates live TypeScript graph extraction for declared targets.
- [Long-form dictation soak](/scenarios/audio-tools/docs/internal/soak-qualification.md) - Runs the provider-owned accelerated browser qualification for the virtual-replay dictation cell and gates on its complete conformance artifact.
- [Performance](performance/README.md) - Validates API/UI build performance and Lighthouse budgets through performance-health.
- [Workflow](workflow/README.md) - Validates BAS workflow assets and safe execution through workflow-health.
- [Channel Conformance](/scenarios/switchboard/docs/internal/PROBLEMS.md) - Runs the shared seven-case adapter contract for every registered Switchboard channel.
- [Monetization Conformance](monetization-conformance/README.md) - Validates monetization trust boundaries, declarations, and meter admission/reporting posture.
- [Stripe Test Outcomes](stripe-test-outcomes/README.md) - Reconciles the LPBS test-mode catalog against Stripe and executes payment outcome journeys with durable provider evidence.
- [Cloud Production Health](cloud-production-health/README.md) - Runs the CL3 owner contract suites and fails closed until staging and production showcase receipts exist.
- [Component Tests](component-tests/README.md) - Runs version-pinned React component and hook contracts through the React Component Library provider.
- [React Consumer Contracts](consumer-contract/README.md) - Runs the usage-contract fleet lint and downward-only per-contract ratchet across every React UI in the repository.
- [Paid Credential Leak Proof](/scenarios/landing-page-business-suite/docs/testing/credential-leak-proof.md) - Uses synthetic canaries derived only from declared credential identity/field/scope metadata, runs LPBS and migrated service-owner identity contracts for audience, scope, source, versioned overlap, outage, restart and recovery, and scans every bounded subprocess output before returning metadata-only receipts.
- [Templates](templates/README.md) - Validates scenario template provenance, orientation standing, drift, migration lag, and inherited template debt through template-manager.
- [Event Capture Conformance](event-capture-conformance/README.md) - Validates opt-in receipt-capture declarations against published protobuf contracts and the reconciled global policy snapshot.
- [Local Model Evaluation](/scenarios/ai-gateway/docs/operations/model-eval.md) - Censuses observed runtime models and checks independently reviewed retrieval, policy-role, gateway-task-role and real consumer evidence, then remains blocked until separate LM2 inference admission allows execution.

## Running Phases

```bash
test-genie execute my-scenario --phases <descriptor-a>,<descriptor-b>
test-genie execute my-scenario --preset comprehensive
```

## Configuration

Per-phase overrides live in `.vrooli/testing.json` under `phases.<phase>` and are validated by [`schemas/testing.schema.json`](../../schemas/testing.schema.json).

## Presets

Preset and profile definitions are documented in [Presets Reference](../reference/presets.md). Quick and smoke are adaptive profiles; concrete preset membership is generated from the effective registry.
