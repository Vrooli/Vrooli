# Search Phase

The `search` phase validates scenarios that declare AI search ownership. It is descriptor-backed by `search-hub` and only applies when the target has `.vrooli/search.json` or declares the `search` / `ai-search` service capability.

Test Genie evaluates applicability before provider readiness. Non-search scenarios omit this phase from normal runs; JSON previews still show the not-applicable reason. If a target declares search and the descriptor is missing, malformed, or operationally incomplete, the phase applies and fails through Search Hub's shared maturity assessment.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every search-owning scenario has **production-ready search maturity**: its
`.vrooli/search.json` descriptor is registry-valid, its providers are routable
and accountable to one scenario, and they carry credible reviewed eval corpora,
fresh live evidence, operational status/control endpoint posture, tuning
provenance, and enforced latency/degradation budgets. At maximum maturity the
deepest capability — `search_descriptor` — is L4 (production-ready) and the other
ladders (`search_governance`, `search_eval_performance`, `search_operability`)
are each at their top rung, so Search Hub can route to the corpus with proven,
non-degraded retrieval quality.

## The rungs and their gates

Search Hub aggregates the per-capability ladders into one phase standing. The
rungs are monotone — each implies the one below.

| Rung | Gate (exit criteria) | Next unlock |
|---|---|---|
| L0 Unavailable | The target declares search but the descriptor/provider corpus cannot be inspected. | A readable `.vrooli/search.json` with at least one provider descriptor. |
| L1 Inspectable | The search descriptor parses and provider entries can be inspected. | Provider descriptors satisfy Search Hub's registry invariants. |
| L2 Governed | Providers are routable and accountable, with eval/readiness metadata. | Declare eval corpus and operational (status/control/tuning) metadata. |
| L3 Baseline clean | No required search-maturity findings remain; ordinary routing is safe. | Prove production-quality AI-search evidence (reviewed corpus, fresh runs, perf posture). |
| L4 Production-ready | Reviewed corpora, fresh live evidence, endpoint posture, tuning provenance, and performance budgets are all present. | Maximum search maturity reached. |

## What each finding means

Each finding caps the capability it names at a rung; only ERROR/BLOCKER
severities fail the phase, so production-readiness debt (WARNING) is honest,
non-failing.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `SEARCH_CONFIG_MISSING` | search_descriptor | L1 | ERROR | Yes |
| `SEARCH_PROVIDER_INVALID` | search_descriptor | L1 | ERROR | Yes |
| `SEARCH_PROVIDER_GROUP_MISMATCH` | search_governance | L0 | ERROR | Yes |
| `SEARCH_EVAL_CORPUS_MISSING` | search_eval_performance | L0 | ERROR | Yes |
| `SEARCH_EVAL_RUN_STALE` | search_eval_performance | L1 | ERROR | Yes |
| `SEARCH_EVAL_CORPUS_THIN` | search_eval_performance | L3 | WARNING | No |
| `SEARCH_REINDEX_ENDPOINT_MISSING` | search_operability | L1 | ERROR | Yes |
| `SEARCH_PERF_BUDGET_UNPROVEN` | search_operability | L1 | ERROR | Yes |
| `SEARCH_STATUS_ENDPOINT_MISSING` | search_operability | L3 | WARNING | No |
| `SEARCH_TUNING_BUDGET_INVALID` | search_operability | L0 | WARNING | No |

The full finding inventory (config/provider validity, eval corpus adequacy and
freshness, recall/assert outcomes, endpoint and performance posture) is declared
in the descriptor's `maturity.findings` block.

## The canonical fix

- **Descriptor/provider findings** (`SEARCH_CONFIG_*`, `SEARCH_PROVIDER_*`) → repair `.vrooli/search.json`: add the descriptor, fix parse errors, and make each provider entry active, scenario-owned, and registry-valid with a declared operability class.
- **Governance mismatch** (`SEARCH_PROVIDER_GROUP_MISMATCH`) → reconcile provider ownership metadata so it matches the owning scenario.
- **Eval findings** (`SEARCH_EVAL_CORPUS_*`, `SEARCH_EVAL_RUN_*`, `SEARCH_EVAL_RECALL_BELOW_TARGET`, `SEARCH_EVAL_ASSERT_FAILED`) → declare a reviewed labelled corpus (at least one positive + one junk negative), run the eval suite to produce fresh evidence under current tuning, and fix retrieval or corpus expectations until recall meets target.
- **Operability findings** (`SEARCH_REINDEX/STATUS/CONTROL_ENDPOINT_MISSING`, `SEARCH_PERF_*`, `SEARCH_TUNING_BUDGET_INVALID`) → declare status/control endpoints (or a validated pinned reason), bound tuning to the query budget, and produce measured latency/degradation evidence that satisfies the declared p95 budget.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
search-hub validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases search
test-genie runs findings --scenario <scenario>
```

`search-hub maturity scan` runs the same full validation across the fleet; add
`--fast` for a descriptor/state inventory that skips live retrieval proof.

## Provider Contract

- **Provider:** `search-hub`
- **Source:** `validation-provider`
- **RPC:** `scenario-validation/v1.ScenarioValidationService.ValidateScenario`
- **Descriptor:** `scenarios/search-hub/.vrooli/test-genie.json`
- **Maturity:** embedded in that descriptor's `maturity` block
- **Policy:** default when applicable, required provider readiness, start provider if needed, live contract freshness, gating results
- **Timeout:** 90s

## What It Validates

Search Hub owns the search maturity judgment. Its highest rung, L4
production-ready search, is part of the normal provider maturity assessment; it
is not a separate strict command path. The contract validates:

- `.vrooli/search.json` presence and parseability when search applies
- provider descriptors and ownership coherence
- eval corpus declaration and parseability
- credible reviewed corpus shape, including negatives, difficulty diversity, and
  provider-declared `tests.coverage` groups satisfied by reviewed positives
- status/control endpoint posture where declared, expected, or explicitly pinned
- bounded tuning, optimization provenance, and query-budget metadata
- performance posture from eval evidence, including latency and degraded-result
  budgets

Test Genie requests execution-mode validation for this phase. Search Hub checks
registered eval suites, latest stored run freshness, failed eval outcomes, and
live corpus labels by probing the provider's current search endpoint. Missing
run history, failed eval outcomes, stale live labels, or unavailable live corpus
proof are required `search_eval_performance` findings.

`search-hub maturity scan` runs the same full validation by default. Use
`search-hub maturity scan --fast` only when an operator needs a quick
descriptor/state inventory and accepts that live retrieval proof, sweep
provenance, and runtime performance evidence are skipped. Fast mode still
surfaces descriptor and static corpus production-readiness debt, so a provider
can pass the phase but remain below L4 until the required maturity findings are
resolved.

Fleet scan and this phase share one applicability model. Search Hub's
`maturity scan` discovers exactly the scenarios this phase treats as applicable:
those owning `.vrooli/search.json`, plus those declaring the `search` / `ai-search`
service capability (the union of `service.tags`, `service.capabilities`, and the
top-level `capabilities` array, matched case-insensitively — the same resolution
Test Genie uses). A capability-only scenario is discovered and fails with
`SEARCH_CONFIG_MISSING` rather than disappearing from the fleet, so a reviewer can
diff `search-hub maturity scan --fast --json` (each result carries an
`applicability_reason`) against `test-genie phases applicability <scenario>
--phase search --json` and see no target-set mismatch.

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
