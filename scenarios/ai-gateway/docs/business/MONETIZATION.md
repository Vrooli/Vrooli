# Monetization — AI Gateway

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

> **Status: active — this scenario is the revenue mechanism, not a product.**
>
> Project canon already names it. `path:docs/monetization/evidence/FINANCIAL_MODEL.md`
> (Tier 1 cost structure): *"Paid Tier 1 subscriptions include the integrated
> API gateway (LLMs, STT/TTS, embeddings, coding agents) with a credit
> allowance — **that IS the core reason to pay** rather than running the OSS
> apps with bring-your-own keys."* This document had that as `deferred`, which
> was drift, not a decision.

## Role In Vrooli

- **Direct product: no.** Nobody buys a router. It is never sold standalone.
- **Internal capability: yes — and revenue-critical.** Every paid tier's value
  proposition passes through it. If the gateway does not meter and enforce
  correctly, the subscription has no product behind it.
- **SKU/bundle candidate: shared-utility across every SKU.** Not yet present in
  `path:docs/monetization/catalogs/scenario-sku-map.json` — that file is
  governed (`catalog-strategist` proposes, human curates), so the mapping should
  be proposed rather than hand-added.
- **Revenue line: the enabling substrate for all of them.** Tier 1 and Tier 2
  are both described in the financial model as *gateway-driven variable cost*.

## The Three-Way Chain

Every AI-consuming scenario in the fleet resolves inference the same way, and
the third rung is the entire subscription proposition:

```
  1. Local models on the user's machine   → free, preferred, no revenue
  2. User's own provider key (BYOK)       → free to us, no revenue
  3. Vrooli subscription inference        → the revenue surface
```

Rungs 1 and 2 are not leakage to be closed — they are why the OSS apps are
worth adopting at all. The subscription competes on *convenience for users who
have neither*, not on withholding capability.

### What is shipped

| Capability | Target | Why monetization needs it |
|---|---|---|
| BYOK via provider resource | `OT-P0-002` | Rung 2. The gateway itself handles no secrets; credentials live in `resource-openrouter`. |
| Routing policy: local-only / local-first / cheap-first / max-cost ceilings | `OT-P1-004` | Per-request cost control and the privacy posture that makes rung 1 credible. |
| Route evidence + descriptor-backed measures (success/fallback/failure/latency/cost/capacity) | `OT-P1-007` | **The metering foundation.** Billing needs per-request attribution, and this is where it already exists. |
| Capacity-aware local routing with policy-respecting remote fallback | `OT-P1-008` | The actual fall-through mechanism from rung 1 to rungs 2–3. |

### What is missing (the real monetization work)

- **Entitlement and credit-allowance enforcement.** The financial model prices
  Tier 1 as *"a credit allowance … plus metered overage."* Nothing in this
  scenario tracks an allowance, knows who a subscriber is, or refuses a call
  when an allowance is exhausted. `OT-P1-004` caps **cost per request**; an
  allowance is **cost per user per period**. Those are different controls and
  only the first exists.
- **A metering-to-billing bridge.** Route evidence is recorded but is not
  aggregated into a billable usage signal.
- **Convergence with audio-tools.** `scenarios/audio-tools` already implements
  its own three-tier chain that reaches the Vrooli tier **directly through
  landing-page-business-suite** (`X-Audio-LPBS-Token`,
  `AUDIO_AI_ENABLE_VROOLI`), bypassing this gateway. Two independent paths to
  one subscription is drift that will produce two metering stories and two
  billing bugs. Converging them is a prerequisite for trustworthy unit
  economics, not a tidiness concern.

## Customer / Buyer

- **Primary user:** every AI-consuming scenario in the fleet, plus the operator
  setting routing policy.
- **Buyer:** nobody buys this. The buyer is whoever subscribes to a Tier 1/2
  SKU; this scenario is what they are actually paying for.
- **Pain it removes for the buyer:** *"I do not want to run models locally and I
  do not want to manage provider accounts and keys."*
- **Existing alternatives:** running local models (free, needs hardware), BYOK
  with each provider (free, needs account management), or a competitor's hosted
  offering. The gateway's edge is that **the same app works in all three modes
  with no code change** — the user upgrades by changing policy, not tooling.

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

Inherited from `path:docs/monetization/evidence/FINANCIAL_MODEL.md`; not
re-derived here.

- **Model:** subscription includes a gateway credit allowance, with metered
  overage beyond it. Wholesale-to-retail token pass-through.
- **Unit economics:** margin = subscription price − (wholesale token cost ×
  consumption up to allowance) − (overage cost at markup).
- **Known risk, and this scenario owns the mitigation:** the financial model
  states heavy users can be low- or negative-margin and that *"pricing should
  cap or meter usage."* **This is the scenario where a cap is enforced.** Until
  allowance enforcement exists, that mitigation is a sentence in a strategy doc
  with no runtime behind it.
- **Cost drivers:** wholesale provider tokens. Note that STT/TTS meters in
  minutes and characters rather than tokens, so a token-only allowance model
  will not price audio correctly — see `scenarios/audio-tools`.

## Validation Plan

- **Demand signal needed:** fall-through rate — the share of real users running
  neither local models nor their own key. That single number decides whether
  gateway-driven revenue is a business or a rounding error, and it is
  measurable from route evidence (`OT-P1-007`) as soon as there are users.
- **Second signal:** consumption distribution. The financial model *estimates*
  that mainstream users consume a small fraction of the heaviest 5%. Route
  measures can confirm or refute that, and allowance sizing depends on it.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Blocking prerequisite:** allowance enforcement. No paid tier should launch
  against an unmetered gateway.

## Current Status

`draft` — the strategy is settled at project level and this document now
reflects it. The routing chain is shipped; **entitlement, allowance
enforcement, and the metering-to-billing bridge are not**, and they are the
gating work for any paid tier. Also open: converging audio-tools' independent
LPBS path onto this gateway.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
