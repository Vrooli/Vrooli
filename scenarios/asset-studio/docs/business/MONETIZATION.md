# Monetization — Asset Studio

This document records how the scenario could earn its keep, and what
evidence would justify investing further.

## Purpose Of This Document

Use this document to answer:

- Who would pay for this capability, and for what?
- How is it packaged relative to other Vrooli scenarios?
- What is the pricing hypothesis and what backs it?
- What validation signal would justify more investment?

## Role In Vrooli

- Direct product: **candidate, not committed.** Same posture as `content-desk`
  D-014 — internal-first, with a real hypothesis recorded rather than a
  `not-applicable` that would have to be reversed later.
- Internal capability: **primary.** It is the media production capability the
  marketing operating model has been deferring since the image and video post
  types were catalogued.
- SKU/bundle candidate: deferred. No bundle mapping until the internal loop runs.
- Revenue line: **indirect, through ai-gateway inference fall-through.**

## Customer / Buyer

- Primary user (internal): the marketing-crew producer agent and the operator.
- Buyer (hypothetical, external): a solo operator or small team running **more
  than one brand or persona** who needs the same character, product, or
  environment to look the same across months of output. One-off generation is
  already well served; consistency-over-time is what breaks.
- Pain: prompt-based generation drifts. A persona rendered in March and again in
  June is recognisably a different person, and nothing in the toolchain records
  which prompt, model, or seed produced the version that worked.
- Existing alternatives, honestly: this is a **crowded and funded category**.
  AI-UGC ad tools (Arcads, Creatify, Icon) produce persona-actor video at volume;
  general generation platforms (Midjourney character reference, Leonardo,
  Scenario) offer consistency features; enterprise DAM (Frontify, Bynder) owns
  brand asset governance at a price point no solo operator reaches. The gap is
  narrow and specific: none of them treat an identity as a **versioned record
  with immutable history**, and none of them refuse to release an asset that
  failed a consistency check.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | candidate | Tier 2 desktop through `scenario-to-desktop`, matching content-desk's direction. Local-first is the correct default: unreleased marketing material and generated persona assets stay on the operator's machine. |
| Bundle component | plausible | Natural pair with `content-desk` — production and editorial as one marketing workflow. Do not map to a SKU until the internal loop has produced published output. |
| Add-on | rejected | It is not an extension of another SKU; it owns its own data model and gate. |
| Service/consulting assist | plausible | Identity setup and character-sheet authoring is the expensive, judgement-heavy part. It is also the part least suited to being sold as software. |

## Pricing Hypothesis

- Model: **the tool is free and never feature-gated.** Revenue is indirect —
  every render is an inference call through ai-gateway, which routes local
  models → BYOK → hosted, so a user who can run neither local generation nor
  their own key falls through to a paid option.
- Why this is a better demand driver than content-desk's: content-desk's
  inference use is claim extraction, which is bursty and small. **Generation is
  the workload.** It is recurring, proportional to output, and image and video
  models are materially more expensive per call than text. If inference
  fall-through is ever going to be a real revenue line, this is the scenario
  that tests it.
- Comparable products: the AI-UGC tools above price per seat or per generated
  video, typically in the tens to low hundreds per month. That establishes
  willingness to pay for the *output*, which is evidence for the category and
  not yet evidence for this shape.
- Willingness-to-pay evidence for **this** differentiator: none. Nobody has been
  asked whether identity provenance is worth paying for, and it is entirely
  possible that operators tolerate drift rather than pay to prevent it.
- Cost drivers: generation spend dominates everything else. Local runtime and
  storage are negligible by comparison, which is why `ASSET-P0-008` makes cost a
  first-class record rather than an operational afterthought.

## Preconditions before any external move

Three, and none is close today:

1. **The conformance gate has to actually work.** It is the entire
   differentiator and it is unvalidated (`PROBLEMS.md`). If frames cannot be
   judged reliably against a reference, the product is a slower version of tools
   that already exist.
2. **Lifecycle decoupling.** The scenario currently assumes Vrooli lifecycle,
   ai-gateway, and image-tools. A standalone build must *package* ai-gateway
   rather than bypass it, or the revenue surface disappears entirely.
3. **A released artifact that shipped.** Marketing a production pipeline that
   has never produced published output is the failure this repository keeps
   recording under a different name.

## Validation Plan

- Demand signal needed: an operator outside this project describing
  consistency-over-time as a problem they currently work around by hand.
- Channel: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: not set. Setting one before the internal loop runs would be
  a number chosen to be met.
- **Kill signal (internal, and it fires first):** if identity conformance cannot
  be made to work — either the check misses real drift, or it is so strict that
  operators bypass it — the product case collapses regardless of market
  interest. The differentiator *is* the guarantee. This is the same shape as
  `vrooli-memory` D-013 and `content-desk` D-014: the disqualifying evidence is
  internal and arrives before any market question is asked.

## Current Status

`hypothesis` — a real position with named comparables, a named mechanism, three
preconditions, and a kill signal. **No P0 or P1 requirement depends on any of
it.** Nothing here should influence the build order, and if it starts to, that
is the signal to re-read this section rather than to act on it.

## Cross-References

- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — positioning and launch motion
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — the ai-gateway dependency the revenue mechanism rests on
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-008 (all inference through ai-gateway), D-014 (P0 scope)
