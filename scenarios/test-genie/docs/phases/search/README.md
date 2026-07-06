# Search Phase

The `search` phase validates scenarios that declare AI search ownership. It is descriptor-backed by `search-hub` and only applies when the target has `.vrooli/search.json` or declares the `search` / `ai-search` service capability.

Test Genie evaluates applicability before provider readiness. Non-search scenarios omit this phase from normal runs; JSON previews still show the not-applicable reason. If a target declares search and the descriptor is missing, malformed, or operationally incomplete, the phase applies and fails through Search Hub's shared maturity assessment.

## Provider Contract

- **Provider:** `search-hub`
- **Source:** `validation-provider`
- **RPC:** `scenario-validation/v1.ScenarioValidationService.ValidateScenario`
- **Descriptor:** `scenarios/search-hub/.vrooli/test-genie.json`
- **Maturity:** embedded in that descriptor's `maturity` block
- **Policy:** default when applicable, required provider readiness, start provider if needed, live contract freshness, gating results
- **Timeout:** 90s

## What It Validates

Search Hub owns the search maturity judgment. The initial contract validates:

- `.vrooli/search.json` presence and parseability when search applies
- provider descriptors and ownership coherence
- eval corpus declaration and parseability
- status/control endpoint posture where declared or expected
- bounded tuning and query-budget metadata

Test Genie requests execution-mode validation for this phase. Search Hub checks
registered eval suites, latest stored run freshness, failed eval outcomes, and
live corpus labels by probing the provider's current search endpoint. Missing
run history, failed eval outcomes, stale live labels, or unavailable live corpus
proof are required `search_eval_performance` findings.

`search-hub maturity scan` runs the same full validation by default. Use
`search-hub maturity scan --fast` only when an operator needs a quick
descriptor/state inventory and accepts that live retrieval proof is skipped.

Provider-specific implementation details stay in Search Hub. Test Genie only plans the phase, checks provider readiness, calls the shared validation RPC, maps findings, and records the result.

## Inspection

```bash
test-genie phases applicability <scenario> --phase search --json
test-genie phases plan <scenario> --preset comprehensive --json
test-genie provider-contract check search <scenario> --json
search-hub maturity scan --json
search-hub maturity scan --fast --json
search-hub maturity fix <scenario> --json
```
