# Monetization — Content Desk

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Verdict — not-applicable

**This scenario is internal capability and is not monetized.** It is the
editorial substrate Vrooli's own marketing runs on: it decides whether a public
claim is verified before the operator approves it, and records what shipped.

Three reasons this is a decision rather than a deferral:

- It has no external buyer. Its value is entirely coupled to Vrooli's marketing
  canon, post-type registry, and team structure. Sold standalone it would be a
  verification workflow with no doctrine behind it.
- It gates nothing a self-hoster could otherwise run, so there is no free-versus-paid
  boundary to draw without violating the fleet's BYOK guarantee.
- It touches no bundle. Marketing effectiveness may eventually move revenue
  indirectly through the funnel it feeds, but that value accrues to the SKUs
  being marketed, not to this scenario.

Revisit only if editorial verification is ever proposed as a standalone product,
which would first require the marketing doctrine it depends on to be portable.

## Role In Vrooli

- Direct product: deferred.
- Internal capability: generated scaffold only.
- SKU/bundle candidate: deferred.
- Revenue line: deferred.

## Customer / Buyer

- Primary user: define during PRD generation.
- Buyer: define during monetization review.
- Pain: define from demand evidence.
- Existing alternatives: capture through market validation.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | deferred | Revisit after first real domain is implemented. |
| Bundle component | deferred | Map in project-level monetization catalog if promoted. |
| Add-on | deferred | Use only when scenario clearly extends another SKU. |
| Service/consulting assist | deferred | Consider if this scenario accelerates done-for-you delivery. |

## Pricing Hypothesis

- Model: deferred.
- Comparable products: none captured yet.
- Willingness-to-pay evidence: none captured yet.
- Cost drivers: local runtime by default; update for resources, hosted
  services, gateway usage, or third-party APIs.

## Validation Plan

- Demand signal needed: define before monetization review.
- Channel: define in [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: define from project-level monetization taxonomy.
- Revisit trigger: first real domain reaches validated scenario tests
  and has a clear user/customer.

## Current Status

`stub` — generated from the template. Fill this document when the PRD
identifies a customer, SKU, revenue line, or monetization hypothesis.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
