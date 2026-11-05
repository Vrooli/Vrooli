# Monetization — Personal Planner

This document records how Personal Planner could create revenue. It is
kept honest: the R1 product implementation is now running locally, while
commercial numbers remain labelled hypotheses to validate, not claims. The single strategic bet is
stated up front because it shapes everything else.

**The bet:** the personal-planning market is saturated with capable tools,
so features do not differentiate. Two things do, together: **exceptional,
calm, place-making UX** (the operator-approved Observatory day/night
experience) and **honest planning underneath it** (capacity, remaining
effort, dependencies, and explainable forecasts that never lie green).
Beauty earns the daily open; honesty earns the trust that survives the
first overloaded week. Monetization is built on retention produced by that
pairing — not on locking away basic planning.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product (primary role):** a standalone consumer/prosumer
  personal-planning app. This is the intended commercial role, gated
  behind proven retention (billing is explicitly R3 — see
  [`../../PRD.md`](../../PRD.md) OT-P2-007).
- **Meta-capability (immediate role):** Personal Planner is the **shared
  scheduling and availability owner** for other Vrooli scenarios (Daily,
  Cadence, agent work). This internal role has value independent of
  direct revenue: every specialized app that needs "when does the human
  actually have time for this?" composes Personal Planner instead of
  rebuilding a scheduler. It raises the ceiling of the whole ecosystem.
- **SKU/bundle candidate:** a natural anchor for a personal-productivity
  bundle (e.g. with a food planner and a content planner that already
  schedule through it). Deferred until the direct product is validated.
- **Revenue line:** deferred to R3; the free personal product comes first.

## Customer / Buyer

- **Primary user:** an individual who repeatedly over-commits — knowledge
  workers, founders, makers, freelancers, and students who juggle goals,
  deadlines, and interruptions and need to see overload *before* it
  happens. Buyer = user (self-serve consumer/prosumer purchase); no
  separate economic buyer in R1's single-user scope.
- **Pain:** optimistic commitments, overloaded days, unclear "done,"
  and an inability to explain how interruptions, waiting, and scope creep
  moved a timeline. Existing calendars show *when* things are, not
  *whether the plan is feasible*; existing to-do apps track tasks but not
  capacity or remaining effort.
- **Existing alternatives & why they leave room:**
  - *Sunsama / Amie* — beautiful and calm, but plan naively (no
    remaining-effort model, no capacity math, no explainable forecast).
  - *Motion / Reclaim* — automate scheduling, but auto-move accepted work
    (which erodes trust) and feel utilitarian rather than calm.
  - *Todoist / TickTick / Things* — strong task capture, weak on
    feasibility, capacity, and time truth.
  - *Google/Outlook Calendar* — a grid, not a planner.
  Personal Planner's wedge is being *as beautiful as the calm tier* while
  being *more honest than the automation tier* — and never silently moving
  a person's accepted plan (decision D01).

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Free personal product | primary (R1) | The complete honest planning–focus–review loop, manual, private, single-user. This is the acquisition and retention engine; it must be genuinely useful with no upgrade. |
| Premium subscription | hypothesis (R2/R3) | Convenience + assistance layered on top: read-only calendar connections beyond one, calibrated statistical forecasts (R2), the conversational planning assistant (R2), richer what-if comparisons, extended history/learning windows, and provider write-back (R3). Never gates core capacity/planning honesty. |
| Bundle component | candidate | Anchor of a personal-productivity bundle with Daily (food) and Cadence (content), which already schedule through it. |
| Add-on / meta-capability | active (internal) | Scheduling-owner contracts consumed by other scenarios; internal value today, potential platform monetization later. |
| Service/consulting assist | deferred | Not a fit for a personal single-user product. |

Guardrail (from the plan's "optional means optional" rule): a paid feature
is never a nonfunctional button, and the free tier is never crippled to
force upgrade. Beauty and honesty are table-stakes for *everyone*.

## Pricing Hypothesis

- **Model:** freemium. Free personal product; a single flat prosumer
  subscription (monthly + discounted annual) for the assistance/
  convenience layer once it exists. No per-seat pricing in R1 (single
  user); household/team seats are an R3 collaboration expansion.
- **Comparable anchors (for reference only, not commitments):** the calm
  personal-planner tier is commonly ~US$8–20/month or ~$96–240/year.
  Personal Planner would position within that band, justified by the
  honesty layer, only after retention is proven.
- **Willingness-to-pay evidence:** none captured yet — a validation
  target, not an assumption.
- **Cost drivers:** local/SQLite runtime is cheap; the meaningful variable
  costs appear only with paid features — external-calendar provider API
  usage, and (R2) optional model/gateway usage for the assistant and
  calibrated forecasts. The deterministic core has no per-use model cost,
  which keeps the free tier's marginal cost near zero.

## Validation Plan

The bet is "beauty + honesty ⇒ retention ⇒ willingness to pay." Validate
in that order; do not price before retention:

1. **Beauty/desirability signal:** does the Observatory experience pass
   the visual + accessibility release gates (plan §6.8) and does qualitative
   feedback rate it "want to open daily"? (Owner: design review.)
2. **Retention signal (the core metric):** weekly-active return to the
   Today surface, and completion of the morning planning loop, over 4–8
   weeks. Retention — not feature count — is the gate to monetization.
3. **Honesty-value signal:** do users experience fewer unrecognized
   overloads, earlier risk identification, and better estimate calibration
   over time (measured from their own records, per
   [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md), not
   surveillance)?
4. **Willingness-to-pay signal:** only after 2–3 hold, test demand for the
   assistance layer via channel experiments in
   [`GO-TO-MARKET.md`](GO-TO-MARKET.md).

- **Success threshold:** set against the project-level monetization
  taxonomy when retention data exists; deliberately unset now to avoid a
  fabricated number.
- **Revisit trigger:** the first end-to-end R1 loop is validated (plan
  P11) and a retention cohort exists.

## Current Status

`active hypothesis → R1 validation`. The commercial *strategy* (freemium
built on retention from beauty + honesty; internal scheduling-owner value
now; billing deferred to R3) is defined and recorded. No pricing is
committed and no revenue exists; the product is in local R1 validation, not
commercial launch. Update this document as retention and willingness-to-pay
evidence accrues.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements and target tiers
- [`../concepts/EXPERIENCE.md`](../concepts/EXPERIENCE.md) — why beauty is the wedge
- [`../../DESIGN.md`](../../DESIGN.md) — the Observatory design contract
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
