# Monetization — Treasury

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: `not-applicable` is
better than inventing a commercial story.

**Pricing, bundle membership and whether to monetize at all are
operator-curated canon.** This document states a hypothesis and the
evidence behind it; it does not set strategy. Read, do not write:
`path:docs/monetization/README.md`, `path:docs/monetization/strategy/STRATEGY.md`,
and the catalog. Wiring a paid feature follows
`path:docs/concepts/PAID_FEATURES.md`.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

- **Direct product: yes, and unusually so for an interface enabler.** Most
  scenarios of this role earn indirectly by making other things possible.
  This one has an identifiable buyer outside Vrooli.
- **Internal capability: yes.** Every future scenario that needs to buy or
  sell anything composes this rather than reimplementing a rail. That is
  the higher-multiplier half of its value and it is not monetized.
- **SKU/bundle candidate: business bundle.** Proposed, not decided —
  bundle membership is operator canon.
- **Revenue line: subscription**, with the convenience-not-capability
  split described under Packaging.

## Customer / Buyer

- **Primary user:** an operator running autonomous or semi-autonomous
  agents that need to spend money, who wants that spend bounded rather
  than trusted.
- **Buyer:** the same person. This is a self-serve, single-operator
  purchase, not an enterprise sale — which is precisely why the hosted
  competitors do not fit.
- **Pain:** giving an agent spending power today means handing it a
  credential. There is no bounded intermediate between "no spending power"
  and "a card number in an environment variable". The consequence is that
  most operators simply do not let agents buy anything, which is a
  capability ceiling rather than a preference.
- **Existing alternatives, and why the gap is real:** the agentic-payments
  category is funded and active — Payman ($13.8M, Visa and Coinbase
  Ventures), Skyfire ($9.5M, a16z CSX), Nekuda ($5M seed, Amex and Visa
  Ventures) — and the platform vendors have shipped: Stripe's agent card
  issuing and its consumer agent wallet, Mastercard's agent tokens,
  Coinbase's session-key wallets. **Every one of them is hosted
  infrastructure that custodies the operator's money.** That is a
  reasonable trade for a company with a finance team and an unreasonable
  one for a self-hoster or a solo builder. Nobody in the category is
  selling a spend-governance layer you run yourself, and that is the
  position this scenario takes.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | candidate | The strongest option. "Your agents, your card, your keys, your policy, your box" is a complete product statement that needs no other Vrooli scenario to make sense. |
| Bundle component | candidate | Business bundle. Pairs naturally with `money-ledger` (what happened) and `offer-desk` (what should earn) to form a complete money story. Membership is operator canon, proposed here only. |
| Add-on | not-applicable | This is not an extension of another SKU; it has its own buyer and its own reason to exist. |
| Service/consulting assist | deferred | Setting up bounded agent spending for a client is plausibly a done-for-you engagement, but that follows product validation rather than preceding it. |

### The free / metered / gated split

Per the ecosystem principle that a subscription buys **convenience and
integrated access, not access to capability**, and that nothing a
self-hoster could run with their own keys should ever be gated:

| Capability | Class | Reason |
|---|---|---|
| Mandate contract, budgets, policy evaluation, approval queue, evidence, manual rail | **free, permanently** | This is the entire P0 spine. It runs locally, makes no marginal-cost call, and is exactly what a self-hoster could build themselves. Gating it would contradict the strategy and would also gut the product's credibility — a governance layer you cannot inspect is not a governance layer. |
| Self-hosted x402 facilitator | **free** | Same reasoning. Running your own facilitator is the whole point of the self-hosted position. |
| Hosted facilitator | **metered** | Real infrastructure cost, and a genuine convenience for an operator who does not want to run one. |
| Managed card issuing | **metered** | Passes through a real per-card and per-transaction cost. |
| Cross-device approval relay | **gated** | Approving from a phone when you are away from the desk is convenience, not capability. The in-scenario console queue is always free and always sufficient — `TRS-P0-006` requires it. |
| Evidence retention beyond the local instance | **metered** | Storage cost, and only relevant to an operator who wants off-box durability. |

**What is deliberately *not* gated:** the policy engine, the mandate
model, the kill switch, and the evidence trail. Those are the product's
integrity, and an integrity feature behind a paywall is a worse product for
everyone including the people who pay.

## Pricing Hypothesis

- **Model:** flat subscription for the convenience tier, plus pass-through
  metering on facilitator and card costs. Not per-transaction on the free
  spine — charging per authorization would create an incentive to authorize
  less, which is exactly backwards for a safety product.
- **Comparable products:** the hosted agent-payments platforms above price
  on transaction volume or take rate, which is the model this scenario is
  positioned against rather than alongside. The closer comparables for
  *shape* are self-hosted infrastructure subscriptions, where the buyer
  pays for hosted convenience while retaining the ability to run it
  themselves.
- **Willingness-to-pay evidence:** none captured yet. The funded
  competitor set is evidence of *category* demand, not of demand for the
  self-hosted variant. That distinction is the main thing validation needs
  to resolve, and it should not be glossed.
- **Cost drivers:** local runtime by default with no marginal cost for the
  P0 spine. Costs appear only with the hosted facilitator (per
  transaction), managed card issuing (per card and per transaction), and
  off-box evidence retention (storage).

## Validation Plan

- **Demand signal needed:** evidence that operators want *bounded* agent
  spending specifically, rather than either unbounded spending or none.
  The sharpest available signal is behavioural: does an operator who has
  the free spine running actually issue mandates and let agents spend under
  them, or does the queue sit empty because they still do it by hand?
- **The question that matters most:** whether self-hosted is a *preference*
  or a *requirement* for this buyer. If it is only a preference, the hosted
  incumbents win on convenience and this position is weak. If it is a
  requirement — because the money is real and the vendor is a startup —
  then the position is defensible and the incumbents structurally cannot
  take it.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** set against the project-level monetization
  taxonomy, not here.
- **Revisit trigger:** first automated rail is live and has settled real
  charges under mandates, which is the earliest point the behavioural
  signal above means anything.

## Current Status

`hypothesis` — the buyer, the gap and the free/metered/gated split are
argued from a competitive scan performed 2026-08-18 and recorded in
[`../../PRD.md`](../../PRD.md). No willingness-to-pay evidence exists.
Bundle placement and pricing remain operator canon and are not decided
here.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why the spine stays free
- Project-level monetization strategy: `path:docs/monetization/README.md`.
- Paid-feature wiring contract: `path:docs/concepts/PAID_FEATURES.md`.
