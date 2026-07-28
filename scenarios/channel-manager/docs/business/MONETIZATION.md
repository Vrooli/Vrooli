# Monetization — Channel Manager

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

## Verdict — internal capability, with a real indirect revenue line through BAS

`content-desk` and `asset-studio` both landed on "direct-product candidate, not
committed," with revenue running indirectly through ai-gateway inference
fall-through. This scenario reaches the same *shape* by a different route.

**Corrected 2026-07-28.** An earlier revision of this document concluded there was
no revenue line, reasoning that the scenario makes no ai-gateway calls. That is
true and remains true — no ai-gateway dependency, comment-text generation
deliberately excluded (D-009), window optimization (`CHANMGR-P2-003`) is statistics
rather than a model call. But it checked the wrong dependency. **The executor is
metered.**

`browser-automation-studio` charges AI credits per AI-assisted operation — its
`ai_request_log` records `request_type` (`workflow_generate`, `element_analyze`),
model, and token counts — through its entitlement service, reported to LPBS with
idempotency keys. So the metered unit is **AI-assisted browser operation**, not
workflow execution.

| Sibling | Indirect revenue mechanism | Driver |
|---|---|---|
| `asset-studio` | ai-gateway inference | Image and video generation, proportional to output volume |
| `content-desk` | ai-gateway inference | Assisted claim extraction at P1; weaker but real |
| `channel-manager` | **BAS AI-navigation credits** | Browser actions whose selectors break — see the sizing below |

### Why the driver is larger than it looks

A deterministic scripted workflow — click, scroll, wait, like — charges nothing.
Most warming steps are inherently deterministic, so the naive read is that warming
generates no credits at all.

That read is probably wrong, for a platform-specific reason: **social platforms
actively churn their DOM to break automation.** Selector-based workflows against
TikTok or Instagram are precisely the case where scripts rot and vision-based
navigation earns its cost. AI navigation is plausibly needed *more* here than in
ordinary browser automation.

Volume compounds it. The conservative program runs roughly 40 actions per day per
identity for five days, then maintenance engagement of roughly 16–38 actions per
day **indefinitely**. Posting alone is capped at three per day. Warming is
therefore around an order of magnitude more browser activity than publishing, it
recurs forever rather than per-campaign, and it scales with identity count. That is
the same property that makes compaction a good demand driver in `vrooli-memory`:
recurring background work beats per-user-action work.

**This remains a hypothesis in one specific way** — what fraction of warming actions
actually require AI navigation is unmeasured, and it is measurable cheaply once
`CHANMGR-P1-001` lands.

### Verdict

- Direct product: **not pursued.** See the positioning section below.
- Internal capability: **primary.** It is the account-operations capability the
  marketing operating model has deferred wholesale since gap #4 was written.
- SKU/bundle candidate: **Tier 1 bundle app**, alongside `content-desk` and
  `asset-studio` as one marketing workflow. No mapping until the internal loop
  produces published output, same posture as both siblings.
- Revenue line: **indirect, through BAS AI-navigation credits under a Tier 1
  subscription.** Contingent on the browser executor landing (`CHANMGR-P1-001`).

## Positioning — sell the workflow, never the warming

Scheduling is commoditized. Buffer, Hootsuite, Later, Metricool, Publer, Sprout
Social, and open-source options including Postiz and Mixpost all schedule posts
across platforms, several with free tiers. Shipping another scheduler into that
market is dead on arrival, and nothing in this queue is novel enough to change that.

What *is* differentiated is warming, cadence discipline, and account-health
monitoring — precisely what mainstream schedulers avoid, and they avoid it for a
reason. A product whose headline function is helping accounts pass platform
coordination detection is a materially different thing from a scheduler, and
**selling that** carries exposure that **using it** does not:

- Platform enforcement against vendors differs from enforcement against users —
  legal contact, API revocation, app-store removal — and it does not depend on any
  individual customer's behaviour.
- The buyer segment shifts from "companies that market a product" to "operators
  running many accounts," which reads as adjacent to spam regardless of any
  individual operator's intent.
- It would attach Vrooli's name to that positioning permanently.

Warming discipline for accounts you own, promoting products you make, is ordinary
marketing operations, and D-009 already governs the automation posture per
platform. The risk is not in building it — it is in *leading with* it.

**The line is in the positioning, not the packaging.** Shipping this inside a
marketing-workflow bundle whose headline is editorial and production is defensible.
Marketing copy that says "warm accounts to avoid shadowbans" is selling warming
automation regardless of how it is packaged, and it is the thing to refuse.

**Recorded position: never market warming as the product.** It is a capability
inside a workflow. Changing that requires a deliberate decision recorded in
`../internal/DECISIONS.md`, not a copy edit.

## What is defensibly sellable, if anything

Strip the vocabulary and the engine underneath is not warming-specific: **a
per-identity action ledger with rate-limit accounting, staged qualification gates,
and baseline-relative anomaly detection.** The part of that with a clean commercial
story is the ledger, not the gates.

The question no scheduler can answer, and the parallel to `content-desk`'s
"which published posts contain a statement that is no longer true":

> **What exactly was done as this account — when, by whom, on whose instruction,
> and with what evidence?**

Competing tools cannot answer it because they model posts and nothing else. This
scenario models every action as an identity, with an executor, a timestamp,
evidence, and an originating instruction, and it never deletes one
(`DATA.md` § Retention).

| Segment | Pain | Why this rather than a scheduler |
|---|---|---|
| Agencies operating client-owned accounts | The client owns the account and is accountable for it. "Who posted that, and who approved it" is currently answered from memory and Slack. | The ledger is an accountability record by construction; approval provenance arrives with the release from `content-desk`. |
| Teams where social posting needs approval trails | Approval exists as process, not as evidence. | Release records carry the draft reference and the approving identity. |
| Anyone reconstructing an account suspension | Platforms give no detail. Reconstructing what was done, and when it changed, is guesswork. | Full action history against a distribution baseline is exactly the reconstruction input. |

Honest limits on that story: the audit-ledger framing is a **narrow** wedge, the
buyers listed are unvalidated, and none of them is a segment Vrooli currently
reaches. It is recorded as a hypothesis, not a plan.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| **Tier 1 bundle app** | **preferred** | Packaged desktop app per `path:docs/monetization/strategy/TIERS.md` §"Tier 1" — self-contained, runs on the operator's own machine, talks to the backend for subscription validation and integrated API routing. This is the shape that works; see the deployment-tier note below. |
| Bundle component | plausible | The natural trio is `content-desk` + `asset-studio` + `channel-manager` as one marketing workflow, in the `business` bundle. `business` is the only SKU in `scenario-sku-map.json`, and neither sibling is mapped yet either. Do not map until the internal loop produces published output. |
| Standalone product | **rejected** | Sold alone, the headline can only be warming. That is the positioning this document refuses. |
| Add-on | rejected | It owns its own data model, gates, and lifecycle. It is not an extension of another SKU. |
| Service/consulting assist | plausible but unattractive | Environment setup — fingerprints, proxies, region consistency — is the expensive, judgement-heavy part, and D-006 deliberately keeps it out of the software. Selling it as a service is the most exposed shape available and inherits every concern in the positioning section. |

### Deployment tier decides whether this works at all

The scenario has an unusual property among its siblings: **where it runs changes
whether it can function**, not merely where the data sits.

| Tier | Runs on | Warming viable? |
|---|---|---|
| **Tier 1 — bundle app** | The operator's own device, their own network, their own residential proxy per identity. | **Yes.** This is the correct target. |
| Tier 2 — self-hosted runtime | The operator's own hardware. | Yes, same reasoning. |
| **Tier 3 — hosted cloud Vrooli** | Vrooli's infrastructure. | **No — self-defeating.** Precondition `residential-proxy-locked` requires a residential IP pinned per identity for the account's life. Hosted execution originates from datacenter IPs, which is exactly the fingerprint warming exists to avoid. |

Tier 3 is ruled out on technical grounds before the positioning question is even
reached, and its own revisit trigger in `TIERS.md` requires Tier 2 to be active
first, so nothing is being given up in the near term.

This also resolves the local-first tension cleanly rather than leaving it open. A
scheduler is only useful if it fires when nobody is watching — but "nobody is
watching" means the operator's own machine running unattended, not Vrooli's
servers. Tier 1 packaging satisfies that, and Tier 3 could not have satisfied it
anyway.

## Free / metered / gated

Per `path:docs/concepts/PAID_FEATURES.md`, the decision is per capability.

| Capability | Mode | Reason |
|---|---|---|
| Identities, warming, queue, signals, manual execution | **free** | No per-use cost. Gating any of it would paywall what a self-hoster already runs, which `PAID_FEATURES.md` prohibits. |
| Browser execution with AI navigation (`CHANMGR-P1-001`) | **metered** | Charged as BAS AI credits through BAS's own entitlement service and reported to LPBS. This scenario adds **no metering of its own** — it consumes a metered capability that another scenario owns, which is the correct shape. BYOK stays valid: an operator supplying their own key runs the same executor with no credit charge. |
| Deterministic browser execution | **free** | No AI operation, no charge. This is the majority of warming steps until selectors break. |
| Multi-operator handoff (`CHANMGR-P2-004`) | **gated**, hypothetically | The only plausible tier differentiator inside this scenario. P2 and unscoped. |

The important structural point: **the metering lives in BAS, not here.** This
scenario never charges a credit, never checks an entitlement, and never touches
LPBS. It generates demand for a capability that is already metered, which keeps one
wallet, one auth path, and one set of safety guarantees — exactly what
`PAID_FEATURES.md` asks for.

## Pricing Hypothesis

- Model: **Tier 1 subscription, no separate price for this scenario.** Per
  `TIERS.md`, a paid Tier 1 subscription includes the integrated API gateway with a
  credit allowance, and that allowance is "the core reason to pay over running the
  OSS apps with bring-your-own keys." Every subscriber running warming drives
  gateway usage with wholesale-to-retail margin. The scenario earns its keep by
  increasing allowance consumption, not by carrying a price tag.
- Why this scenario is a good fit for that model specifically: warming is
  **recurring background** consumption proportional to identity count and retained
  indefinitely, rather than per-campaign or per-user-action. That is the highest-value
  consumption shape for an allowance-based subscription.
- Comparable products: scheduling is served by Buffer, Hootsuite, Later, Metricool,
  Publer, Sprout Social, and the open-source Postiz and Mixpost — all of which price
  per social account or per user, and none of which meters execution. Managed
  warming-and-posting services exist in the multi-account operator market; they are
  the comparable for the positioning this document refuses, which is itself
  informative about the segment.
- Willingness-to-pay evidence: none direct. The Tier 1 allowance model is inherited,
  not validated here.
- Cost drivers: gateway token cost for AI navigation is the only variable line, and
  it accrues to BAS. Local runtime otherwise; `vault` is a local resource. The one
  external cost in the design — device fingerprints and residential proxies — sits
  **upstream** and D-006 keeps it outside the boundary deliberately, which also means
  it is a cost the *user* carries, not Vrooli.
- **Unit-economics caution:** an allowance model plus a consumption driver that
  recurs forever and scales with identity count is exactly the combination that
  produces negative-margin heavy users. A subscriber running thirty persona
  identities is a materially different cost profile from one running two. Whether
  the allowance needs an identity-count dimension is a real question for
  `FINANCIAL_MODEL.md`, not for this document — but it should not be discovered
  after pricing is set.

## Validation Plan

- Demand signal needed, in order: (1) the internal loop produces published output at
  all, which requires content-desk and asset-studio to be running; (2) at least one
  warming program completes with recorded observations, converting D-002's
  speculative defaults into measurement; (3) **the AI-navigation fraction is
  measured** once `CHANMGR-P1-001` lands — what share of warming actions actually
  need vision-based navigation rather than a stable selector. That number is the
  whole revenue hypothesis and it is cheap to obtain.
- Channel: see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- Success threshold: not set for revenue. The Tier 1 allowance model is inherited
  from `TIERS.md` rather than proposed here, so this scenario's threshold is
  consumption volume, not price.
- **Kill signal (capability):** if warming programs do not measurably improve
  account survival over posting without them, the differentiated capability is not
  real, and what remains is a scheduler in a commoditized market. Internal, cheap,
  and comes free with `CHANMGR-P1-006`.
- **Kill signal (revenue):** if the AI-navigation fraction turns out to be
  negligible — warming runs happily on stable selectors — then the consumption
  driver is a rounding error and this returns to a purely internal capability. That
  is a perfectly acceptable outcome; it just should not be discovered after the
  packaging decision.
- Revisit trigger: when the first identity graduates and the first posts ship; or
  when `CHANMGR-P1-001` produces its first month of AI-navigation data.

## Current Status

`internal-capability, revenue line identified` — indirect through BAS AI-navigation
credits under a Tier 1 subscription, contingent on the browser executor landing and
on the AI-navigation fraction being non-trivial. Two positions are recorded
deliberately: **Tier 3 hosted execution is ruled out** on technical grounds, and
**warming is never marketed as the product**. No SKU mapping and no independent
price.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — D-006 (environment boundary), D-009 (automation posture)
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
- Free/metered/gated engineering contract: `path:docs/concepts/PAID_FEATURES.md`.
