# Monetization — Token Economy

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

> **Note on authority.** Whether to monetize, pricing, and bundle membership
> are operator-curated **monetization canon** (`path:docs/monetization/README.md`,
> `offer-desk`). This document records the scenario-local hypothesis and the
> evidence that would confirm or kill it. It does not decide strategy.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product: yes** — a household reward economy is a standalone consumer
  product with an existing, proven market.
- **Internal capability: yes, and this is the underrated half** — the scenario
  is the zero-blast-radius rehearsal surface for the grant/mandate policy model
  that `treasury` applies to real money. Every rule worth having on real money
  can be shipped and broken here at no cost. A policy engine that has only ever
  run against real money has never been safely wrong.
- **SKU/bundle candidate: lifestyle bundle**, as a depth-layer scenario rather
  than a headliner. Bundle placement is operator canon and is not decided here.
- **Revenue line: consumer products** (subscription), with a plausible
  secondary in team/community recognition once `TKE-P2-006` exists.

## Customer / Buyer

- **Primary user**: two distinct people in one household — an adult who mints,
  grants and approves, and one or more children who earn and redeem. The
  product must serve both; see the two-audience design adaptation in
  [`../../DESIGN.md`](../../DESIGN.md).
- **Buyer**: the adult. The child is a user, never a buyer, and nothing in the
  product should be designed to create purchase pressure through them.
- **Pain**: existing allowance apps are bank products. They require linking a
  real bank account, issuing a real card to a child, passing KYC, and paying a
  monthly per-family fee — and they can only ever pay out in dollars. A parent
  who wants to reward with *screen time*, *a trip*, or *a chore traded between
  siblings* has no product; they have a whiteboard.
- **Secondary pain**: privacy. Every alternative is a hosted financial service
  holding a behavioral record of a minor. Some parents will not accept that at
  any price, and today they have no alternative that is not a spreadsheet.
- **Existing alternatives** (scanned 2026-08): Greenlight (deepest parental
  controls — store restrictions, category limits, reporting), BusyKid
  (chore-centric, parent-approved payout, ~$4/month flat per family — the
  cheapest paid option), FamZoo (virtual family-bank IOU model — mechanically
  the closest to this scenario, though not in deployment), Modak (free, debit
  card), Acorns Early (absorbed GoHenry's US base in late 2025, retaining card
  and chore features). Below the paid tier: whiteboards, jars, and spreadsheets,
  which is what most households actually use.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | **primary hypothesis** | A self-hosted household economy. The whole product runs on the operator's own machine with no bank, no card, and no third party. |
| Bundle component | **likely** | Depth-layer scenario in the lifestyle bundle. Composes with any other scenario as an earning surface (`TKE-P0-007`), which is value no standalone allowance app can offer. |
| Add-on | not-applicable | It does not extend another SKU; it is its own product with its own users. |
| Service/consulting assist | not-applicable | No done-for-you delivery motion. |
| Internal rehearsal capability | **active regardless of revenue** | Even if the consumer hypothesis fails entirely, the scenario earns its keep as the safe proving ground for `treasury`'s policy model. This is why it is worth building before the market question is answered. |

## Pricing Hypothesis

- **Model**: free and unlimited when self-hosted, consistent with the project
  posture that the subscription buys *convenience and integrated access, not
  access to the code*, and that nothing a self-hoster could already run with
  their own keys is gated.
- **What a subscription would actually buy**: cross-device reach (a holder's
  phone reaching the household instance), hosted relay for approval requests
  through `notification-hub`, and backup of the journal — none of which is the
  policy engine, the economy, or any core capability.
- **Comparable products**: BusyKid ~$4/month flat per family is the price floor
  worth reasoning against; Greenlight and Acorns Early sit meaningfully higher
  because they carry card issuance and banking costs this scenario does not
  have. **Not having those costs is the point**, and pricing at parity would
  discard the advantage.
- **Willingness-to-pay evidence**: none captured. This is the largest open
  question and is deliberately unanswered — see Validation Plan.
- **Cost drivers**: effectively zero marginal cost. SQLite, no shared resource,
  no inference (rule evaluation is deterministic by design — `TKE-P1-002`
  explicitly refuses a model because unexplainable refusals contradict
  `TKE-P0-003`), no payment processor, no third-party service. **There is
  nothing here to meter**, which is why the free/metered/gated split resolves to
  free for the entire core product.
- **Gated: nothing, deliberately.** A gated feature appearing later would be a
  signal the framing drifted.

## Validation Plan

- **Demand signal needed**: one real household running the P0 loop —
  earn → grant → redeem → approve — for a sustained period, with the child
  choosing to open the holder view unprompted. Retention by the *child*, not
  the parent, is the signal that matters; a reward economy the child ignores is
  a chore for the parent.
- **The falsifiable claim**: that "rewards which are not money" is a real
  unmet need rather than a designer's preference. It is entirely possible that
  parents who want this simply use a whiteboard and are content. That outcome
  should kill the consumer hypothesis cleanly, and the scenario would still be
  worth having for its internal role.
- **Channel**: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold**: sustained use across at least four consecutive weeks
  by one household, including at least one redemption the parent would
  otherwise have handled informally. Below that, P1 should not be built.
- **Revisit trigger**: the PRD sequences P1 explicitly behind this signal —
  *"P1 opens after one real household has run the P0 loop for a period; the
  market hypothesis is checked there and not in a document."*

## Current Status

`hypothesis` — the customer, the pain, and the competitive gap are identified
and reasoned from a real market scan. **Willingness to pay is unvalidated and
deliberately so.** No pricing decision should be made, and no bundle placement
requested, before the household validation above returns a signal.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements, value promise, and the appendix competitive scan
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../concepts/DATA.md`](../concepts/DATA.md) — the privacy posture that is half the wedge
- Project-level monetization strategy: `path:docs/monetization/README.md`.
