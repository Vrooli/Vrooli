# Coverage Model

## Purpose Of This Document

This is the **single canonical source** for the model that the coverage-space documents share, so none of them repeats it. The space docs cross-reference this file:

- `search-hub/docs/spaces/answer-space.md` — the **Answer** denominator
- `test-genie/docs/spaces/validate-space.md` — the **Validate** denominator
- `prompt-manager/docs/spaces/guide-space.md` — the **Guide** denominator
- `program-runtime/docs/spaces/act-space.md` — the **Act** denominator *(relocated to its owner on 2026-08-06 when `program-runtime` was created; live numerator since 2026-08-07)*

It defines the projections, the denominator/numerator split + denominator-confidence, the generative question model, the attestation contract (the two honesty axes), and the status legend. `meta-optimization-manager` reads each space doc against this model to compute coverage.

Coverage is one of three axes. The sibling **Condition** axis — whether the supply counted here still works — is defined in [`CONDITION-MODEL.md`](CONDITION-MODEL.md); the **Empirical** axis is named under [Coverage Is Not The Only Axis](#coverage-is-not-the-only-axis).

## The Projections

Software engineering by a (local) agent is **understand → change → verify**, and every change lands as an **effect on the system**. The project's readiness for that decomposes into four projections, each owned by the scenario that holds its ground truth:

| Projection | Question | Owner (denominator) | Numerator source |
|---|---|---|---|
| **Answer** | Can the project be *understood* — are architectural questions answerable? | `search-hub` | `search-hub providers list` |
| **Validate** | Can a change be *verified* + auto-fixed? | `test-genie` | `test-genie health` / `fleet status` |
| **Guide** | Is there a *skill* to guide each SWE task? | `prompt-manager` | `prompt-manager graph health` |
| **Act** | Is each operation *programmatically invocable* by an agent? | `program-runtime` | program-runtime binding registry (`ResolveActCells`) |

The *change* itself — code synthesis — is the local model's own job; it is not a projection. **Act is not code synthesis**: it measures whether the surrounding system exposes each operation as something an agent can invoke from a program, which is supply the project owns, exactly like a search provider or a test phase. Readiness is ultimately proven **empirically** by the `trials` domain, not declared from coverage.

### Act status: live

All three of Act's original obligations are met. `program-runtime/docs/spaces/act-space.md` is authored and owned by `program-runtime`; the shared `space --projection act --json` verb is registered (`PRT-P0-007`); and `BindingRegistryService.ResolveActCells` serves the live numerator (`PRT-P0-008`), joined by `api/internal/coverage/numeratorclient.go`. All 28 Act cells have been audited against the live binding registry, which raised the stated denominator confidence to `PARTIAL`.

Act degrades like every other projection: when `program-runtime` is not running, the join returns `UNAVAILABLE` with a stated reason and the authored cell statuses stand. It is never reported as `0%` or silently dropped.

**Act coverage is bounded by fleet manifest coverage.** The callable surface cannot exceed the set of scenarios that ship a `cli/manifest.json`, and the numerator reports that ceiling honestly rather than concealing it. Raising the ceiling is fleet work with no single owner — it is surfaced by `cli-health` and ranked here, and it is the highest-leverage single lever on this projection.

## Coverage Is Not The Only Axis

The projections are the **coverage axis**: enumerable supply, measured as `now / total` against a curated denominator. There are two further independent axes. The **empirical axis** is observed outcomes with no denominator, which cannot be a projection because "the frictions that exist" is not an enumerable set. The **condition axis** asks whether the supply this file counts is still working — a question neither of the other two can express, because coverage sees only existence and the empirical axis sees only agent outcomes.

| Axis | Shape | Question | Members |
|---|---|---|---|
| **Coverage** (projections) | `now / total`, denominator-confidence | Does the capability *exist*? | Answer, Validate, Guide, Act |
| **Condition** | per-leg verdict + instrumentation coverage | Does the capability that exists still *work*? | the contributor legs backing every `NOW` cell |
| **Empirical** | trend over observed runs, no denominator | What *happens* when agents work? | `trials` (experimental, causal, expensive) |

**Condition is defined in full by [`CONDITION-MODEL.md`](CONDITION-MODEL.md)** and is not restated here. Two properties of it matter to this file: its population is *derived* from this file's live numerator (the legs behind every `NOW` cell), so it can never drift out of sync with coverage; and by default a degraded leg does **not** change its cell's coverage status, because folding condition into supply makes both numbers unreadable. The one existing exception — `test-genie`'s numerator already gating on provider failure rate and pending autofix, described under [Per-Cell Numerator Semantics](#per-cell-numerator-semantics) — is recorded there as a known asymmetry with a stated target state.

`trials` is experimental: a fixed task suite, a deterministic oracle, everything else held constant — so it answers a *counterfactual* ("**could** a local model do this?") that observational data structurally cannot, since local models are not run on real work. Observational friction evidence (agent-manager investigations, and later program-runtime telemetry) is the complementary half: broad, cheap, real-distribution, but confounded.

A coverage gap says *"this does not exist"*; an empirical signal says *"this exists and hurts."* Both should produce ranked gaps on one surface, so a reader needs only `focus next`.

`focus next` now merges both axes through named `GapSource` implementations. Coverage entries retain their projection/status semantics. Empirical entries carry recurrence, an evidence source, and a locator for the newest supporting run. Trials history is the causal lane; agent-manager friction episodes are the observational lane. Each source degrades independently: an unavailable source produces a visible availability reason while healthy sources continue to rank.

An empirical entry is intentionally not a percentage. It reports independent recurrence and traceability because the set of possible frictions has no honest denominator. Unattributed agent-manager observations are excluded from clusters and the dropped count is surfaced as a source degradation entry.

## The Knowledge Projections Overlap In Supply And Form A Maturation Gradient

> **Scope:** this section describes the three **knowledge** projections (Answer, Validate, Guide). **Act is a peer projection, not a further rung on this gradient** — a fact becoming derived-answerable does not mature into an operation becoming invocable. Act measures *effect* supply; the other three measure *knowledge* supply. Keep them separate for the same reason the three are kept separate: the honesty payoff is being able to say "we can explain X and test X, but an agent cannot invoke X."

The three knowledge projections partition the **questions** (demand), not the **providers** (supply). The same entity — a scenario, a domain, a concern — is usually asked in all three modes at once: a `*-health` scenario typically registers a search provider (Answer), powers a test-genie phase (Validate), *and* carries a maturity-ladder skill (Guide). Drawn as a Venn over *scenarios*, the projections nearly contain one another; drawn over *questions*, they stay disjoint. **Keep the denominators distinct** — the honesty payoff is being able to report "we can *validate* X but cannot *explain* X," which collapses if the spaces are merged.

What the shared supply reveals is a **direction**. A capability matures **Guide → Validate → Answer**:

1. **Guide (prose).** A concern starts as a skill: "here is how to reason about dependency hygiene."
2. **Validate (programmatic).** It graduates into a test-genie phase backed by a scenario + a maturity ladder; the prose shrinks, the validator grows (`docs/agent-system/PROMOTION_LADDER.md`).
3. **Answer (derived).** Once a scenario *computes* the fact in order to check it, that same fact becomes derived-answerable: the scenario registers a provider that returns it with `basis=DERIVED`.

So a concern's readiness is **not three independent numbers** — it is how far up this gradient it has climbed, plus the `basis` quality at each stage. The mature end-state is "present in all three modes, with Guide collapsed to a thin *pointer* because Validate + Answer now carry it." Two consequences this model is load-bearing for:

- **`focus` reasons per-concern across projections (a trajectory), not three ranked buckets.** A Guide-only-prose concern with no phase and no provider is *less* ready than a Validate+Answer-backed one whose Guide skill is a pointer — even though both might read "COVERED" in Guide alone.
- **A graduated pointer-skill is the success state, not a Guide gap.** Measuring Guide coverage as "a *rich* skill exists" perversely penalizes the very delegation this loop exists to produce. The Guide denominator counts a thin pointer-to-a-validator as `COVERED` (mature), never `MISSING` — see `prompt-manager/docs/spaces/guide-space.md`.

This is the meta-team's own prose→programmatic loop (`docs/agent-system/LAYERS.md`, `PROMOTION_LADDER.md`) re-expressed as a coverage gradient: the three knowledge projections are the same capability stack observed at three stages of maturity. Act sits alongside that stack rather than on it.

## Denominator, Numerator, Confidence

- **Denominator** — the curated *intended* space (the space doc). Lives with the owner; read via that owner's `space --projection <p> --json` verb.
- **Numerator** — live *actual* coverage, computed by joining the denominator against the owner's live registry. **Never stored.**
- **Coverage** = numerator ÷ denominator, computed live at query time.
- **Denominator-confidence** — how complete we believe the *denominator itself* is: `AUTHORITATIVE | PARTIAL | SKETCH` + a rationale. Every coverage number is reported with it. The honesty is **recursive**: a board reads "X% complete against a Y-confidence denominator," so it can never imply false completeness.

## Per-Cell Numerator Semantics

The live join is per-cell for every projection. A cell absent from a live join
keeps its authored denominator status: "can't resolve" is not fabricated as
`MISSING`.

The numerator is read **live over typed API↔API calls**, not CLI shell-outs:
each owner's API base URL is resolved through `api-core/discovery` and called via
a typed Connect-RPC client, concurrently, each bounded by a short ~3s deadline
(`numeratorDeadline`, `api/internal/coverage/numeratorclient.go`). A slow or
unreachable owner degrades to an honest per-projection `UNAVAILABLE` rather than
stalling the board.

| Projection | Owner RPC | Live join rule |
|---|---|---|
| **Answer** | search-hub `RegistryService.ListProviders` | A declared provider match yields `NOW`. An authored-`NOW` cell whose provider is absent downgrades to `IN-REACH`; other unresolved cells keep authored status. |
| **Validate** | test-genie `RunsService.GetSelfHealth` | A provider is `NOW` only when it exists in the catalog, has no ledger `failureRate > 0`, and has no `autofix.pending > 0`; red ledger or pending autofix yields `IN-REACH`. |
| **Guide** | prompt-manager `GraphService.GetHealthScores` | Resolve the Guide row's skill ids against the graph health scores. `NOW` requires **all** referenced skills to be present and at/above `guideHealthyScore`; any partially resolved or unhealthy row is `IN-REACH`; fully unresolved rows keep authored status. |
| **Act** | `program-runtime` | `BindingRegistryService.ResolveActCells` returns live callable verdicts. If the owner is unavailable, the join returns `UNAVAILABLE` with an honest reason and preserves authored statuses. |

### Known condition asymmetry in the Validate join

The Validate rule above is the only join that folds a **condition** signal into a
coverage status: a provider with ledger `failureRate > 0` or `autofix.pending > 0`
yields `IN-REACH` rather than `NOW`. Every other projection reports existence only.

This predates [`CONDITION-MODEL.md`](CONDITION-MODEL.md) and is retained rather than
silently changed. Under that model the target state is report-beside plus promotion
to a coverage downgrade only after sustained degradation, which would make Validate
consistent with its siblings. Converting it changes a live published number, so it
is an explicit decision rather than a refactor — recorded here so the inconsistency
is visible and deliberate.

### Load-bearing constants

These judgment constants drive the headline numbers; they are kept as named,
documented constants so the judgment is auditable and easy to revisit:

- **`guideHealthyScore = 0.5`** (`api/internal/coverage/numerator.go`) — the
  prompt-manager graph health-score threshold at/above which a Guide skill counts
  as healthy. `0.5` = "more healthy than not", a deliberately lenient bar:
  a skill existing and scoring at least neutral is the signal that the Guide cell
  is served at all. Revisit once graph-health scores have a longer production
  distribution.
- **`numeratorDeadline = 3s`** (`api/internal/coverage/numeratorclient.go`) — the
  per-owner live-read deadline; a slower owner is an honest `UNAVAILABLE`, not a
  hang. **`spaceReadTimeout = 5s`** (`exec.go`) bounds the denominator space-verb
  read (cheaper, has a doc-parse fallback).
- **Focus weights** (`api/internal/focus/prioritize.go`) — coverage leverage,
  empirical recurrence steps, the Answer cap, and the untraceable-signal cap;
  see the comment block for rationale and revisit triggers.
- **Convergence tier thresholds** (`api/internal/convergence/tiers.go`) — the
  tiering cut-points; see that file's comment block for its rationale.

## The Generative Question Model (Answer Projection)

An architectural question is a coordinate: **ENTITY × ARCHETYPE (× ASPECT lens)** — all three axes bounded by the project's enforced structure, so the space is finite and enumerable (sparse: nonsensical combinations are omitted).

- **Entities** (nouns), by granularity tier: Project (G0) · Scenario (G1) · Within-scenario (G2) · Domain (G3) · Symbol (G4) · Ecosystem (G5).
- **Archetypes** (verbs, 10): Inventory · Anatomy · Connection · Flow · Conformance · Verification · State · Provenance/Intent · Comparison · Pointer. Each predicts a typical basis (Inventory/Anatomy/Connection/Conformance → `DERIVED`; Flow → `PARTIAL`; Provenance/Intent → `ABSENT`/pointer).
- **Aspects** (lens, 17): the test-genie phases (structure, proto, storage, security, quality, deps, ui, performance, …), applied as an optional filter.

## The Attestation Contract — Two Honesty Axes

Every architectural answer is attested along two **independent** axes. They ride *inside* the result (mirroring search-hub's existing `MeasureHit` carrier on `SearchHit`), **separate** from the relevance `score`.

**Basis — how do we know it?** (epistemic provenance)
- `DERIVED` — computed directly from code (AST/graph/facts). Ground truth modulo the parser. Highest trust.
- `VALIDATED` — a declared doc/contract claim that we checked against code and it agrees (zero drift). High.
- `DECLARED_UNVERIFIED` — a claim that exists but we cannot fully validate (no validator, or the source lacks the conventions to confirm it). Medium — treat as a hypothesis, not fact.
- `CONTRADICTED` — a declared claim that disagrees with the code (drift detected). Low — return both the claim and the code, flag the divergence.
- `ABSENT` — no source of truth exists. Pointer-only: "we can't attest this; look here."

**Sufficiency — is the source even shaped to answer this?** (coverage of the question)
- `FULL` — the source covers the question completely.
- `PARTIAL` — answers part; the rest is gapped.
- `INSUFFICIENT` — accurate as far as it goes but lacks the conventions to express what was asked (e.g. a `DOMAINS.md` that lists domains but not their archetypes). Work with what's there, but say plainly it's insufficient so the agent looks further.

The axes are **orthogonal**: an answer can be `VALIDATED`+`INSUFFICIENT` (accurate but doesn't cover the question) or `DECLARED_UNVERIFIED`+`FULL` (covers it, but unconfirmed). Trust must **not** be folded into the relevance `score`.

## Status Legend

- `NOW` — a live provider / phase / skill answers it today.
- `IN-REACH` — substrate exists (often a declared `capability_gap` stub); build / attest / extend the provider.
- `MISSING` — no provider / substrate; a true gap.

## Governing Principles

- **Unbounded questions, bounded honest answers** — when no source has ground truth, the contract forces `basis=ABSENT` → pointer-only, so the system structurally cannot overclaim.
- **Surfaces, does not decide** — coverage measurement flags candidates and numbers; the substrate / tiering / nomination / improvement decisions stay agentic.

## Cross-References

- [`CONDITION-MODEL.md`](CONDITION-MODEL.md) — the Condition axis: whether the supply counted here still works.
- The four space docs: `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md`, `program-runtime/docs/spaces/act-space.md`.
- [DOMAINS.md](DOMAINS.md), [ARCHITECTURE.md](ARCHITECTURE.md).
- Convergence doctrine: `docs/agent-system/TEMPLATE_CONVERGENCE_LOOP.md`, `REFERENCE_PATTERN_FITNESS.md`, `REFERENCE_SCENARIOS.md`.
