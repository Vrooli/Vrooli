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

## Denominator, Numerator, Confidence

- **Denominator** — the curated *intended* space (the space doc). Lives with the owner; read via that owner's `space --projection <p> --json` verb.
- **Numerator** — live *actual* coverage, computed by joining the denominator against the owner's live registry. **Never stored.**
- **Coverage** = numerator ÷ denominator, computed live at query time.
- **Denominator-confidence** — how complete we believe the *denominator itself* is: `AUTHORITATIVE | PARTIAL | SKETCH` + a rationale. Every coverage number is reported with it. The honesty is **recursive**: a board reads "X% complete against a Y-confidence denominator," so it can never imply false completeness.

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
