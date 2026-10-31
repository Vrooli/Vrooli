# Monetization — Prose Studio

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

## Role In Vrooli

- **Direct product: no, and deliberately not.** The 2026 writing-tool market has already
  specialised — brand-voice-and-collaboration tools, fiction tools, and citation-oriented
  academic tools stopped competing with each other and each won a lane. A general
  "write anything in any style" surface competes with every specialist on their own turf
  and wins nowhere. Prose Studio is scoped as an enabler for that reason.
- **Internal capability: yes, and this is the point.** It is the foundation that makes
  specialised, sellable writing products cheap to build later. The commercial argument is
  not "sell this"; it is "this is why the next three sellable things take weeks instead of
  quarters, and why they share one voice system instead of three."
- **SKU/bundle candidate: depth layer of the `business` bundle**, which is the only base
  bundle currently active. Not a headliner today; plausibly a future headliner's
  substrate rather than the headliner itself.
- **Revenue line: indirect, through two paths.** First, it improves Vrooli's own marketing
  and brand output, which is a cost-and-quality lever on acquisition rather than a revenue
  line. Second, it is the prerequisite that makes specialised writing products buildable.

## Customer / Buyer

- **Primary user: other Vrooli scenarios**, not an end market. `content-desk` for marketing
  drafts, `asset-studio` and `document-manager` for styled text, `test-data-generator` for
  diverse synthetic corpora. Second, the operator composing directly. Third, any agent that
  wants one output drawn from a wider distribution than direct prompting produces.
- **Buyer: none directly.** A future specialised product built on this substrate has a buyer;
  this does not. Stating that plainly is more useful than inventing a persona.
- **Pain**: single-shot generation returns the most typical phrasing, voice is unversioned
  and unmeasurable, and every consumer that wants styled text otherwise re-implements
  prompting by hand with no way to tell whether the result got better or merely different.
- **Existing alternatives**: the specialised commercial tools above, all hosted, all keeping
  brand voice locked inside their own product.

## Where The Differentiation Actually Is

Recorded because it is easy to lose and it shapes what must not be traded away:

1. **Local-first / BYOK styled generation.** The entire local-model ecosystem is *runtimes
   and chat shells* — nobody has built the writing product on top. Vrooli already has local
   routing, a locality stance on every request, and a role catalog. "Your brand voice never
   leaves your machine" is a position the hosted incumbents structurally cannot take. This is
   the strongest edge and it is why `write.default` is local-first even though the best prose
   models are hosted.
2. **Style as a versioned, addressable, portable record.** A commercial tool's brand voice is
   locked inside that tool. A style id that `content-desk`, `asset-studio`, and
   `document-manager` all reference is composition none of them offer.
3. **The honesty seam.** `content-desk` already refuses to approve a draft carrying an
   unverified claim. Generated copy that *cannot ship* while a claim is unverified is a
   governance story the category lacks, and the compliance-oriented incumbent's existence is
   evidence someone pays for exactly that.
4. **Diversity by construction** is a real quality property but **not a moat** — it is a
   prompt shape plus a measurement, and anyone can copy it. Treat it as a feature, never as
   positioning.

## Packaging Notes

- Inference cost is metered Class A automatically by calling ai-gateway normally; this
  scenario implements no metering of its own.
- **Never gate what a self-hoster could run with their own keys.** BYOK must remain a valid
  path — which is the same constraint that makes differentiator (1) coherent rather than
  contradictory.

## Validation Signal

The signal that would justify further investment is not revenue. It is: a second consuming
scenario declares its own profiles without any code being added here, and the acceptance
measurement shows candidate sets are measurably more varied than direct prompting on at
least one routed model. The first proves the enabler shape works; the second proves the
capability is real. Either failing is a reason to stop, and a negative diversity result is a
publishable finding rather than a concealed one.

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
- Project-level monetization strategy is maintained in the repository-level
  business strategy; this scenario has no separate monetization capability.
