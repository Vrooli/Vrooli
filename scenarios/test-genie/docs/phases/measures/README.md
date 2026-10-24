# `measures` phase

The `measures` phase is a **thin delegating phase**: it calls the
`measures-health` scenario's shared `ScenarioValidationService` and maps its
normalized findings into the shared `FINDING_SOURCE_MEASURES` channel.
test-genie itself contains **no coverage logic** — deriving a scenario's
stateful domains, classifying each covered/waived/uncovered, and grading
per-measure extraction tier all live in `measures-health`.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every declared measure is **behaviorally ready and indexable**: each of the
scenario's stateful domains is covered by a measure (or a deliberate, current
waiver), every declared measure assembles against its bound proto request schema,
covered domains are backed by full-tier measures with deterministic or bounded
parameter extraction, and every measure passes its behavioral probe so it is fit
for indexed search and fleet surfaces. At maximum maturity the capability ladders
all reach their top rung — `contract_derivation` L2, `domain_coverage` L3,
`declaration_assembly` L3, `tier_completeness` L3, and the deepest,
`behavioral_indexing` L2 (measures are behaviorally ready and indexable) — so the
scenario's measures are trustworthy, complete, and discoverable rather than hollow
declarations.

## The rungs and their gates

Each capability declares a monotone ladder (each rung implies the one below).

**`contract_derivation`** — can the expected stateful-domain contract be derived?
- L0 No readable measures contract → *Readable CLI manifest and repository contract inputs.*
- L1 Expected domains derived → *Architecture-backed derivation without illegal domain declarations.*
- L2 Contract derivation clean → maximum.

**`domain_coverage`** — are expected domains covered or deliberately waived?
- L0 Domain coverage unavailable → *Expected domains and persisted substrates are known.*
- L1 Coverage comparable → *Every expected domain is covered or deliberately waived.*
- L2 Domains covered or waived → *Waiver and substrate coverage debt eliminated.*
- L3 Domain coverage clean → maximum.

**`declaration_assembly`** — do declared measures assemble against their proto bindings?
- L0 Measure declarations unavailable → *Readable measure declarations.*
- L1 Declarations readable → *Valid declaration shape and proto request bindings.*
- L2 Measures assemble → *Maximum declaration-assembly maturity.*
- L3 Declaration assembly clean → maximum.

**`tier_completeness`** — are covered domains backed by full-tier measures?
- L0 Tier posture unavailable → *Assembled measure declarations with tier metadata.*
- L1 Tier posture visible → *Full-tier measures for covered domains.*
- L2 Full-tier coverage → *Tier fallback and partial-tier debt eliminated.*
- L3 Tier coverage clean → maximum.

**`behavioral_indexing`** (deepest) — do measures pass behavioral probes and index cleanly?
- L0 Behavioral readiness unavailable → *Assembled full-tier measures that can be probed.*
- L1 Behavioral probeable → *No hollow declarations remain.*
- L2 Behavioral and indexed → maximum; measures are behaviorally ready and indexable.

## What each finding means

Each finding caps the named capability at a rung; only ERROR/BLOCKER severities
fail the phase, so warnings and infos are honest, non-failing debt.

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `measures.uncovered-domain` | domain_coverage | L2 | ERROR | Yes |
| `measures.hollow-declaration` | behavioral_indexing | L2 | ERROR | Yes |
| `measures.illegal-domain-declaration` | contract_derivation | L1 | ERROR | Yes |
| `measures.malformed-declaration` | declaration_assembly | L2 | ERROR | Yes |
| `measures.undeclared-substrate` | domain_coverage | L2 | WARNING | No |
| `measures.stale-waiver` | domain_coverage | L2 | WARNING | No |
| `measures.tier-fallback` | tier_completeness | L2 | WARNING | No |
| `measures.tier-partial` | tier_completeness | L2 | INFO | No |
| `measures.architecture-fallback` | contract_derivation | L1 | INFO | No |

## The canonical fix

- **`measures.uncovered-domain`** → the domain has no measure and no waiver: add a
  measure covering the stateful domain, or record a deliberate waiver if it is
  genuinely out of scope. Load the `measures-adoption` skill.
- **`measures.hollow-declaration`** → the measure declares but does not behave:
  give it a real behavioral implementation that passes the probe (`measures-adoption`,
  `test`).
- **`measures.illegal-domain-declaration`** → a declared domain is not a legal
  stateful domain for the scenario's architecture: remove or correct it; run the
  `screaming-architecture-audit` skill to reconcile with the canonical architecture.
- **`measures.malformed-declaration`** → the measure does not assemble against its
  bound proto request schema: fix the declaration shape and the proto binding
  (`measures-adoption`, `api-steer`).
- **`measures.undeclared-substrate`** → a persisted substrate is used but not
  declared: declare it so coverage reconciles.
- **`measures.stale-waiver`** → a coverage waiver is stale: refresh or retire it.
- **`measures.tier-fallback` / `measures.tier-partial`** → a covered domain's
  measure falls back to a partial extraction tier: upgrade to full-tier
  deterministic or bounded parameter extraction.
- **`measures.architecture-fallback`** → derivation used a documented fallback
  rather than the canonical architecture surface: expose architecture inputs so
  derivation is architecture-backed.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
measures-health validate scenario <scenario>

# Or drive it through Test Genie and read the per-phase scorecard + findings:
test-genie execute <scenario> --phases measures
test-genie runs findings --scenario <scenario>
```

The `measures` line in the scorecard shows the focus capability's current rung,
the single highest-unlock next move, and a runnable doc-search topic that resolves
back to the sections above.

## What it runs

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

The static provider path is used so the target scenario does not need to be
running. Test Genie reads the shared `status` and `assessment.findings` fields.
Each assessment finding is mapped to an `ArchitectureFinding{Source:
FINDING_SOURCE_MEASURES}`, so it carries a deterministic stable ID, normalized
severity, and the per-source effort default.

## Severity contract (load-bearing)

measures-health performs the normalization; this phase routes its severity
strings through the same `normalizeFindingSeverity` table every producer uses:

| measures-health severity | normalized | feeds the `measures` dimension as a gap? |
|---|---|---|
| `SEVERITY_ERROR` (uncovered stateful domain / hollow declaration / malformed) | ERROR | **yes** |
| `SEVERITY_WARNING` (stale waiver / fallback tier) | WARNING | no |
| `SEVERITY_INFO` (partial tier / not-expected context) | INFO | no |

Only ERROR findings fail the phase. They flow into the swarm-manager
`measures` dimension, which is a **soft** rung (R4) — a scenario stays
runnable and safe without measures, but cannot reach top maturity while a
stateful domain is left uncovered and unwaived.

## Classification

- **Optional**, auto-joins the `comprehensive` preset (anti-drift guard).
- **180s timeout** — coverage harvest reads manifests + the proto descriptor
  across the target.
- Unreachable `measures-health` API → advisory skip because the phase is
  optional. Start it via `vrooli scenario start measures-health`.
- `TEST_GENIE_SKIP_MEASURES=1` skips the phase entirely (matches the
  `security` escape hatch).

## Degraded producer

When the `measures-health` API is unreachable, the phase
emits a skip observation and stays green — the gate only acts on real findings,
never on the producer's availability.

## See also

- `scenarios/measures-health/` — the producer scenario.
- `packages/measures-go/` — the shared measure contract library.
- `packages/proto/schemas/architecture/v1/findings.proto` —
  `FINDING_SOURCE_MEASURES = 10`.
- `scenarios/measures-health/.vrooli/test-genie.json` — the descriptor-owned
  `measures` dimension and phase coverage metadata.
