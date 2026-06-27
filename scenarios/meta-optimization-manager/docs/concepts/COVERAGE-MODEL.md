# Coverage Model

## Purpose Of This Document

This is the **single canonical source** for the model that the three coverage-space documents share, so none of them repeats it. The space docs cross-reference this file:

- `search-hub/docs/spaces/answer-space.md` — the **Answer** denominator
- `test-genie/docs/spaces/validate-space.md` — the **Validate** denominator
- `prompt-manager/docs/spaces/guide-space.md` — the **Guide** denominator

It defines the three projections, the denominator/numerator split + denominator-confidence, the generative question model, the attestation contract (the two honesty axes), and the status legend. `meta-optimization-manager` reads each space doc against this model to compute coverage.

## The Three Projections

Software engineering by a (local) agent is **understand → change → verify**. The project's readiness for that decomposes into three projections, each owned by the scenario that holds its ground truth:

| Projection | Question | Owner (denominator) | Numerator source |
|---|---|---|---|
| **Answer** | Can the project be *understood* — are architectural questions answerable? | `search-hub` | `search-hub providers list` |
| **Validate** | Can a change be *verified* + auto-fixed? | `test-genie` | `test-genie health` / `fleet status` |
| **Guide** | Is there a *skill* to guide each SWE task? | `prompt-manager` | `prompt-manager graph health` |

The *change* itself — code synthesis — is the local model's own job; it is not a projection. Readiness is ultimately proven **empirically** by the `trials` domain, not declared from coverage.

## The Projections Overlap In Supply And Form A Maturation Gradient

The three projections partition the **questions** (demand), not the **providers** (supply). The same entity — a scenario, a domain, a concern — is usually asked in all three modes at once: a `*-health` scenario typically registers a search provider (Answer), powers a test-genie phase (Validate), *and* carries a maturity-ladder skill (Guide). Drawn as a Venn over *scenarios*, the projections nearly contain one another; drawn over *questions*, they stay disjoint. **Keep the denominators distinct** — the honesty payoff is being able to report "we can *validate* X but cannot *explain* X," which collapses if the spaces are merged.

What the shared supply reveals is a **direction**. A capability matures **Guide → Validate → Answer**:

1. **Guide (prose).** A concern starts as a skill: "here is how to reason about dependency hygiene."
2. **Validate (programmatic).** It graduates into a test-genie phase backed by a scenario + a maturity ladder; the prose shrinks, the validator grows (`docs/agent-system/PROMOTION_LADDER.md`).
3. **Answer (derived).** Once a scenario *computes* the fact in order to check it, that same fact becomes derived-answerable: the scenario registers a provider that returns it with `basis=DERIVED`.

So a concern's readiness is **not three independent numbers** — it is how far up this gradient it has climbed, plus the `basis` quality at each stage. The mature end-state is "present in all three modes, with Guide collapsed to a thin *pointer* because Validate + Answer now carry it." Two consequences this model is load-bearing for:

- **`focus` reasons per-concern across projections (a trajectory), not three ranked buckets.** A Guide-only-prose concern with no phase and no provider is *less* ready than a Validate+Answer-backed one whose Guide skill is a pointer — even though both might read "COVERED" in Guide alone.
- **A graduated pointer-skill is the success state, not a Guide gap.** Measuring Guide coverage as "a *rich* skill exists" perversely penalizes the very delegation this loop exists to produce. The Guide denominator counts a thin pointer-to-a-validator as `COVERED` (mature), never `MISSING` — see `prompt-manager/docs/spaces/guide-space.md`.

This is the meta-team's own prose→programmatic loop (`docs/agent-system/LAYERS.md`, `PROMOTION_LADDER.md`) re-expressed as a coverage gradient: the three projections are the same capability stack observed at three stages of maturity.

## Denominator, Numerator, Confidence

- **Denominator** — the curated *intended* space (the space doc). Lives with the owner; read via that owner's `space --projection <p> --json` verb.
- **Numerator** — live *actual* coverage, computed by joining the denominator against the owner's live registry. **Never stored.**
- **Coverage** = numerator ÷ denominator, computed live at query time.
- **Denominator-confidence** — how complete we believe the *denominator itself* is: `AUTHORITATIVE | PARTIAL | SKETCH` + a rationale. Every coverage number is reported with it. The honesty is **recursive**: a board reads "X% complete against a Y-confidence denominator," so it can never imply false completeness.

## Per-Cell Numerator Semantics

The live join is per-cell for every projection. A cell absent from a live join
keeps its authored denominator status: "can't resolve" is not fabricated as
`MISSING`.

| Projection | Live join rule |
|---|---|
| **Answer** | Extract provider ids from `search-hub providers list`; a declared provider match yields `NOW`. An authored-`NOW` cell whose provider is absent downgrades to `IN-REACH`; other unresolved cells keep authored status. |
| **Validate** | Extract phase providers from `test-genie health`. A provider is `NOW` only when it exists in the catalog, has no ledger `failureRate > 0`, and has no `autofix.pending > 0`; red ledger or pending autofix yields `IN-REACH`. |
| **Guide** | Extract skill ids from the Guide row and scores from `prompt-manager graph health`. `NOW` requires **all** referenced skills to be present and healthy; any partially resolved or unhealthy row is `IN-REACH`; fully unresolved rows keep authored status. |

Guide health currently uses `guideHealthyScore = 0.5` in
`api/internal/coverage/numerator.go`. Keeping that as one documented constant
makes the judgment auditable and easy to revisit once graph-health scores have a
longer production distribution.

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

- The three space docs: `search-hub/docs/spaces/answer-space.md`, `test-genie/docs/spaces/validate-space.md`, `prompt-manager/docs/spaces/guide-space.md`.
- [DOMAINS.md](DOMAINS.md), [ARCHITECTURE.md](ARCHITECTURE.md).
- Convergence doctrine: `docs/agent-system/TEMPLATE_CONVERGENCE_LOOP.md`, `REFERENCE_PATTERN_FITNESS.md`, `REFERENCE_SCENARIOS.md`.
