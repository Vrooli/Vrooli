# AI Conformance Phase

The `ai-conformance` phase validates scenarios that declare or expose AI usage. It is descriptor-backed by `ai-gateway` and applies when a target declares an AI capability or provides an AI descriptor.

Test Genie evaluates applicability before provider readiness. Non-AI scenarios omit this phase from normal runs; applicable scenarios are validated through AI Gateway's shared `scenario-validation/v1` provider contract.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton, so a doc-search topic emitted in a run's scorecard resolves to the exact remediation section.

## North Star

An AI-using scenario touches no provider directly: it owns no provider credentials, secrets, URLs, or HTTP calls; it never hard-codes a concrete model slug, context window, or embedding dimension; it requests capability by role/profile and lets resource-owned or AI-Gateway policy pick the model; its embedding stores record role, dimension, content-version, and a retarget strategy so a model change is an intentional migration rather than silent corruption; and where a contract covers the operation, generic text AI usage is routed through AI Gateway profiles (with any low-level exception explicitly reviewed). At maximum maturity every capability ladder is at its top rung — provider access **governed**, model selection **clean**, embedding metadata **clean**, gateway adoption **clean** — leaving the scenario ready for production AI assurance gates.

## The rungs and their gates

AI Gateway reports a short ladder per capability (`resource_boundary`, `model_policy`, `embedding_safety`, `gateway_adoption`). Each capability shares the same three-rung shape; the phase-level ladder (L0 inventory-unavailable → L1 boundary hygiene → L2 role/policy → L3 embedding correctness → L4 gateway adoption → L5 operational assurance) stacks these capability rungs in dependency order.

| Rung | What it means (per capability) | Next unlock |
|---|---|---|
| L0 Unknown | The AI source/config/persistence surface cannot be inspected, so coupling may be hidden. | Make AI-relevant source, config, and schema/migration surfaces readable. |
| L1 Present-but-unsafe (`Unsafe` / `Coupled` / `Recommended`) | The surface is inspectable and the capability's debt is visible: direct provider calls, hard-coded slugs/dimensions, missing embedding metadata, or un-adopted gateway. | Clear that capability's findings — move calls/credentials behind resources or the gateway; use role/profile requests; record embedding metadata + retarget strategy; adopt the gateway or document a reviewed exception. |
| L2 Clean (`Governed` / `Clean`) | The capability's findings are absent: provider access governed, model policy clean, embedding metadata clean, gateway adoption clean. | Top rung for this capability; keep the phase ladder climbing toward L5 operational assurance. |

## What each finding means

Each finding caps the capability it names at L1 until cleared; only ERROR/BLOCKER severities fail the phase, so `WARNING`/`INFO` findings are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `ai.direct_ollama_http` / `ai.direct_openrouter_http` | resource_boundary | L1 | ERROR | Yes |
| `ai.invalid_provider_secret_env` | resource_boundary | L1 | ERROR | Yes |
| `ai.invalid_provider_url_env` | resource_boundary | L1 | WARNING | No |
| `ai.concrete_model_slug` | model_policy | L1 | WARNING | No |
| `ai.hardcoded_context_window` | model_policy | L1 | WARNING | No |
| `ai.resource_gateway_missing_role` | model_policy | L1 | WARNING | No |
| `ai.hardcoded_embedding_dimensions` | embedding_safety | L1 | ERROR | Yes |
| `ai.embedding_metadata_missing` | embedding_safety | L1 | ERROR | Yes |
| `ai.gateway_not_adopted` | gateway_adoption | L1 | INFO | No |
| `ai.unreviewed_exception` | gateway_adoption | L1 | WARNING | No |

## The canonical fix

- **Resource-boundary findings** (`ai.direct_ollama_http`, `ai.direct_openrouter_http`, `ai.invalid_provider_secret_env`, `ai.invalid_provider_url_env`) → remove direct provider HTTP calls and provider-owned secret/url settings; route the call through a resource or an AI Gateway profile so credentials and topology are owned by the boundary, not the scenario. Provider migration needs scenario-specific role/profile intent — load `prompt-manager skill read ecosystem-fit`.
- **Model-policy findings** (`ai.concrete_model_slug`, `ai.hardcoded_context_window`, `ai.resource_gateway_missing_role`) → replace concrete model slugs and hard-coded context budgets with role/profile requests; add the missing role policy to the owning resource so model selection can evolve behind policy.
- **Embedding-safety findings** (`ai.hardcoded_embedding_dimensions`, `ai.embedding_metadata_missing`) → record role/model/dimension/content-version and a retarget strategy alongside each vector store so an embedding-model change is a deliberate, migratable event. This is a data-retention decision — load `prompt-manager skill read storage-health`.
- **Gateway-adoption findings** (`ai.gateway_not_adopted`, `ai.unreviewed_exception`) → adopt AI Gateway profile contracts where they cover the operation, or record a low-level exception with owner, expiry, and replacement plan for scenario-owner review.

## How to verify

```bash
# See the current rung, gaps, and next move for every AI capability:
ai-gateway validation validate --scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases ai-conformance
test-genie runs findings --scenario <scenario>
```

The `ai-conformance` line in the scorecard shows the current rung, the single highest-unlock next move, and a runnable doc-search topic that resolves back to the sections above.

## Provider Contract

- **Provider:** `ai-gateway`
- **Source:** `validation-provider`
- **RPC:** `scenario-validation/v1.ScenarioValidationService.ValidateScenario`
- **Descriptor:** `scenarios/ai-gateway/.vrooli/test-genie.json`
- **Maturity:** embedded in that descriptor's `maturity` block
- **Policy:** default when applicable, required provider readiness, start provider if needed, live contract freshness, gating results
- **Timeout:** 90s

## What It Validates

AI Gateway owns the AI conformance judgment. The initial contract validates:

- direct Ollama/OpenRouter HTTP usage
- provider secret and provider URL environment ownership
- concrete model slugs outside resource policy ownership
- hard-coded embedding dimensions near vector/embedding code
- missing AI Gateway adoption signals for scenarios with AI usage

Provider credentials, provider URLs, raw prompts, raw completions, and concrete model catalogs are not stored in AI Gateway findings. Findings carry rule IDs, severity, path/line, message, remediation, and maturity mapping.

## Inspection

```bash
test-genie phases applicability <scenario> --phase ai-conformance --json
test-genie phases plan <scenario> --preset comprehensive --json
test-genie provider-contract check ai-conformance ai-gateway --json
```
