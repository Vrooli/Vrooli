# AI Conformance Phase

The `ai-conformance` phase validates scenarios that declare or expose AI usage. It is descriptor-backed by `ai-gateway` and applies when a target declares an AI capability or provides an AI descriptor.

Test Genie evaluates applicability before provider readiness. Non-AI scenarios omit this phase from normal runs; applicable scenarios are validated through AI Gateway's shared `scenario-validation/v1` provider contract.

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
