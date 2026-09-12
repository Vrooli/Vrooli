# Architecture Phase

The `architecture` phase audits a scenario's **structural cohesion** — does the
code's shape scream the product's capabilities, and do the domains hold together
without import cycles, runaway coupling, or misplaced files? It delegates to the
**architecture-cartographer** scenario's shared `ScenarioValidationService` and
normalizes the findings into the shared `ArchitectureFinding` contract that
every test-genie phase now emits.

This is the **cohesion axis** of the audit battery. The per-surface phases
(`contracts`, `ui-health`, `api`, `docs`, `quality`, `proto`) ask "is each
surface built right?"; the `architecture` phase asks "does the whole scenario
cohere?".

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton, so a doc-search topic emitted in a run's scorecard resolves to the exact remediation section.

## North Star

The scenario's code screams its product's capabilities: a fresh graph extracts without tool errors; curated, high-confidence domain authority (e.g. `docs/concepts/DOMAINS.md`) makes every finding routeable to an intentional owner; the domains hold together with no import cycles, layering violations, cross-scenario private imports, coupling smells, or mislocated files; the declared API, CLI, UI, and implementation surfaces agree with domain authority and archetype expectations; and PRD/requirement intent claims align with owned domains, operational targets, and domain vocabulary. At maximum maturity every capability ladder — graph extraction, domain authority, boundary integrity, surface coherence, intent alignment — is at **L3 Complete**, clean enough for recurring fleet drift-monitoring.

## The rungs and their gates

architecture-cartographer reports a ladder per capability (`graph_extraction`, `domain_authority`, `boundary_integrity`, `surface_coherence`, `intent_alignment`). The rungs are monotone — each implies the one below — and all five capabilities share the same shape.

| Rung | Gate (what it means) | Next unlock |
|---|---|---|
| L0 Unavailable | The graph, domain authority, or comparison evidence the capability needs cannot be produced. | Make the source graph and domain authority readable and extractable without tool errors. |
| L1 Foundation | Findings are stable enough to route (codes, locations, domains), but drift remains. | Resolve the deterministic blocker/error findings for the capability. |
| L2 Ready | No deterministic blocker/error findings remain in active code paths. | Eliminate or deliberately suppress the remaining heuristic debt. |
| L3 Complete | No provider-owned blocking or debt findings remain. | Maximum maturity reached — clean enough for recurring drift monitoring. |

## What each finding means

Each finding caps the capability it names at the rung shown; only ERROR/BLOCKER severities fail the phase (and, per this phase's confidence gate, only when authority confidence is high), so `WARNING`/`INFO` findings are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `graph.extract_failed` | graph_extraction | L0 | ERROR | Yes |
| `domain_authority/missing` | domain_authority | L0 | WARNING | No |
| `domain_authority/low` | domain_authority | L1 | INFO | No |
| `domains_doc_parse_warning` | domain_authority | L1 | WARNING | No |
| `cycle` / `cycle/cross-domain` / `layering` | boundary_integrity | L2 | ERROR | Yes |
| `cross_scenario` | boundary_integrity | L2 | ERROR (safety) | Yes |
| `coupling_smell` / `mislocated_file` / `naming` / `glossary_drift` | boundary_integrity | L3 | WARNING | No |
| `convergence_drift` / `surface_coherence` | surface_coherence | L2 | WARNING | No |
| `intent.req_unowned_domain` | intent_alignment | L2 | ERROR | Yes |
| `intent.domain_unrequired` / `intent.ot_no_domain` | intent_alignment | L2 | WARNING | No |
| `intent.req_transport_owned` / `intent.vocab_drift` | intent_alignment | L2 | INFO / WARNING | No |

## The canonical fix

- **Graph-extraction findings** (`graph.extract_failed`) → repair the extraction failure (unparseable source, tool crash) so a graph snapshot can be produced; debug with `prompt-manager skill read scientific-debugging`.
- **Domain-authority findings** (`domain_authority/missing`, `domain_authority/low`, `domains_doc_parse_warning`) → add or curate a domain authority source such as `docs/concepts/DOMAINS.md` and fix its parse warnings so findings route to intentional owners.
- **Boundary-integrity findings** (`cycle`, `cycle/cross-domain`, `layering`, `cross_scenario`, `coupling_smell`, `mislocated_file`, `naming`, `glossary_drift`) → break import cycles, respect layering, replace cross-scenario private imports with the owner's API contract, and relocate/rename mislocated or misnamed files; load `prompt-manager skill read boundary-of-responsibility-enforcement`.
- **Surface-coherence findings** (`convergence_drift`, `surface_coherence`) → reconcile missing, stale, or undeclared API/CLI/UI/implementation surfaces so declared and observed surfaces agree with domain authority.
- **Intent-alignment findings** (`intent.req_unowned_domain`, `intent.domain_unrequired`, `intent.ot_no_domain`, `intent.req_transport_owned`, `intent.vocab_drift`) → reconcile requirement and PRD claims, operational targets, and domain vocabulary with the owned domain map; load `prompt-manager skill read requirements-traceability-steer`.

The `screaming-architecture-audit` skill is the umbrella remediation guide for this phase.

## How to verify

```bash
# See the current rung, gaps, and next move for every architecture capability:
architecture-cartographer audit run <scenario>

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases architecture
test-genie runs findings --scenario <scenario>
```

The `architecture` line in the scorecard shows the current rung, the single highest-unlock next move, and a runnable doc-search topic that resolves back to the sections above.

## How It Runs

Test Genie resolves the architecture-cartographer base URL via service discovery
and calls `ScenarioValidationService.ValidateScenario` for the target scenario.
The shared `assessment.findings` become Observations and normalized
`ArchitectureFinding`s (`source = ARCHITECTURE`). Architecture Cartographer
packs its native `AuditRunResponse` into `native_detail` for its own CLI/UI.

Equivalent operator flow:

```bash
architecture-cartographer audit run <scenario>
```

## Confidence-Gated Enforcement

The phase is **Optional** and graded. Coupling, convergence, naming, and other
heuristic findings remain advisory signals that feed campaign tracking. The
phase hard-fails only when deterministic error/blocker findings are trustworthy
enough to act on. Test Genie reads the native cartographer `AuditRunResponse`
from `native_detail` so it can distinguish `finding_class=deterministic` from
`finding_class=heuristic`:

- default `TEST_GENIE_ARCHITECTURE_GATE=high-confidence`: fail on deterministic
  error/blocker findings only when cartographer reports
  `authority_confidence=high`;
- `TEST_GENIE_ARCHITECTURE_GATE=all`: fail on deterministic error/blocker
  findings regardless of authority confidence;
- `TEST_GENIE_ARCHITECTURE_GATE=off`: never gate architecture findings.

Transport errors (cartographer unreachable) and cartographer `TOOL_ERROR`
outcomes still fail the phase independently of the finding gate. Heuristic
findings still surface in the report as advisory observations, but do not
contribute to the gate.
Together with the overall finding count, they drive the **campaign nudge**:
when findings exceed the single-pass threshold, the suite output steers you to
open a tracked improvement campaign in architecture-cartographer rather than
fixing ad-hoc.

The native audit detail also carries the cartographer score matrix. Test Genie
renders those categories in the phase observations so operators can see
placement legibility, surface alignment, boundary cleanliness, naming clarity,
and authority posture without treating heuristic scores as pass/fail gates.

## Preset

The phase is included in the `architecture-audit` preset, the single command the
screaming-architecture audit skill points at:

```bash
vrooli scenario test <scenario> --preset architecture-audit
# runs: structure, contracts, ui-health, api, docs, architecture, proto
```

## Summary Metrics

The phase pointer records an `ArchitectureSummary`:

| Field | Meaning |
|---|---|
| `outcome` | cartographer audit outcome (`clean` / `findings` / `tool_error`) |
| `total` | total findings after filters |
| `blockers` / `errors` / `warnings` / `infos` | counts by normalized severity |
| `suppressed` | findings excused by in-repo `// arch:allow` markers (reported, not dropped) |
| `authority_confidence` | domain-derivation authority confidence (`high`, `medium`, `low`, or `missing`) |
| `gate_mode` | effective gate mode (`off`, `high-confidence`, or `all`) |
| `gated_blockers` | deterministic error/blocker findings considered by the gate |
| `categories` | score-matrix category summaries copied from cartographer native detail |

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip architecture
```

Disable for the process via environment:

```bash
TEST_GENIE_SKIP_ARCHITECTURE=1
```

Keep findings advisory during rollout:

```bash
TEST_GENIE_ARCHITECTURE_GATE=off
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 120s):

```json
{
  "phases": {
    "architecture": { "timeout": "180s" }
  }
}
```
