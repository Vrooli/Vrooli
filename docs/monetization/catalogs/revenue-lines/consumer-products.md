# Revenue Line: Consumer Products (own-produced SKUs)

> Offer Desk is authoritative for this revenue line's current status, owner,
> and activation record. This document retains the hypothesis, constraints,
> and compliance judgment rather than a live line snapshot.

- **Revisit trigger:** *"Revisit when at least one lifestyle-bundle scenario is deployable with usable inventory data for its users AND we've identified ≥1 physical product whose demand is validated by external signal (e.g., TikTok / Amazon / organic search traffic)."*
- **Revenue model:** one-time purchase (print-on-demand books, whiteboards, stickers, planners, calendars; potentially paid courses / premium content)
- **Productization target:** feeds the Lifestyle bundle primarily; narrow role in the Business bundle
- **Legal surface:** standard consumer-goods compliance (tax nexus, returns policy, truth-in-advertising). Print-on-demand partners absorb most of it. Non-trivial but well-understood.

## Hypothesis

A subset of lifestyle-bundle scenarios reach users who want a tangible artifact — a printed cleaning guide, a wall-mounted maintenance checklist, a wedding planner, a personalized stickered baby-proofing kit. We can produce these as our own SKUs (print-on-demand-first, partner-produced where viable), sell them standalone on Amazon / our site, and use that audience as a funnel into the subscription bundle. Equally, we can surface the physical equivalent contextually *inside* existing scenarios ("you've been using this cleaning routine for a month — want the physical version?") where the user already has a clear need.

## Why this is a distinct revenue line (not a subset of subscription)

- **Cost structure is different.** Print-on-demand: near-zero upfront, per-unit margin. Subscription: hosting, API routing, support burden.
- **Acquisition channel is different.** Physical products compete on Amazon / SEO / social. Subscription competes on bundled capability and developer mindshare.
- **Unit economics are different.** One-time purchase ≠ recurring revenue. Attribution, cohort analysis, LTV all modelled separately.
- **Discipline required is different.** Subscriptions compound. Consumer products don't — each SKU is a standalone P&L that must justify itself.

Lumping consumer products under "subscription" would hide unit economics and conflate acquisition channels.

## Mandatory architectural constraints

These are not "best practices." They are hard rules every scenario that participates in this revenue line must obey. Violating them is a failure of the revenue line.

### 1. Recommendation-blindness

**The agent producing a recommendation must not know what we sell.** A scenario's recommendation engine optimizes for what is actually best for the user. A separate, auditable post-processing layer decides whether the recommendation has an attached consumer-product offer. The recommendation layer and the offer layer are structurally separated in code, in data flow, and in review boundaries.

This is the single most important rule. Violating it corrupts the authority layer — which is precisely the moat the lifestyle bundle is built on. Once trust erodes, it does not come back.

### 2. Post-processing-only offer insertion

Product offers, upsells, and upgrade prompts are inserted after recommendation is complete. Never during. Never as a ranking input. Never as a filter.

### 3. State awareness

The offer layer must respect user state:

- Never offer a product the user already owns (per inventory).
- Never offer a product in a category the user has declined ≥N times.
- Never offer the same specific SKU more than once per dismiss-window.

Inventory data and dismissal state are inputs to the offer layer, never to the recommendation layer.

### 4. Truthful opt-out

The user must be able to disable consumer-product offers globally and per-scenario. The opt-out copy must describe what's actually happening ("this controls whether we earn a commission / direct profit when you buy through us"), not a euphemism.

## Mandatory UX discipline

- **Frequency caps.** Product offers per scenario per user are capped (at minimum: once-per-week per SKU category; individual SKU is one-and-done per dismiss-window). Exact caps decided per scenario in its OTs.
- **Dismiss friction.** One tap to dismiss. Second tap: "don't show this product again." Third tap: "don't show any product offers in this scenario." All three must be respected persistently.
- **Transparency.** Offers are labeled as offers, visually distinguished from system suggestions and agent recommendations. They never appear in the same visual frame / flow as a recommendation.
- **No dressing up as tasks.** An offer is not a "suggested task," a "completion step," or a "tip." It is a purchase prompt and is presented as one.

## Bundle applicability

- **Lifestyle bundle: high fit.** Users are outcome-oriented, many have preference for physical artifacts, and household / family / personal contexts legitimately involve tangible goods (books, planners, product kits). Strongest early candidate for this revenue line.
- **Business bundle: narrow fit.** Business-bundle users want the tool to work; they don't come to dev tools for merch. Legitimate sub-cases: advertising/marketing-helper apps where generated assets (business cards, branded stationery, printed campaigns) are a natural output; paid deep-dive guides / courses on scenario-specific workflows. Most business-bundle monetization remains subscription.

## Gating: inventory as substrate

Consumer-product viability is **gated on inventory maturity.** Without rich inventory data (what the user owns, ages of dependents, existing routines, upcoming events), product offers collapse to ads. With inventory data, offers become the completion of a known need.

**Rule:** A scenario should not activate consumer-product offers until it has (a) an integrated inventory data source and (b) enough state awareness to avoid offering things the user already has.

## Pattern examples (not product commitments)

- **Home & household:** printed cleaning/maintenance guides; whiteboards for routines; calendars and planners tied to household schedules.
- **Baby / child-age triggered:** baby-proofing kits surfaced at the correct developmental moment, driven by inventory state.
- **Gift-purchase surface:** when the user is actively looking for a gift (contact-book / relationship context), present physical product options from our catalog or via affiliate. Gift intent is a legitimate invitation to offer products — psychologically distinct from unsolicited suggestion.

These are *patterns* that illustrate fit. They are not committed SKUs and do not belong in the catalog until a specific product is scoped.

## Instrumentation

Tracked separately from subscription revenue. `financial-tracker` reports:

- SKUs live (count by bundle, by production mode: POD / partner / own)
- Units sold per SKU per month
- Revenue per SKU and gross margin (POD absorbs much of COGS; partner / own vary)
- Consumer-product → subscription conversion rate (the cross-line signal — did physical-product buyers convert to subscribers?)
- Offer impression → purchase rate per scenario (to detect offer-surface drift)

A sustained negative signal — low conversion to subscription AND low same-line repeat purchase — is a flag that consumer products may be running as an independent business rather than feeding the subscription, triggering a review under the same services-trap discipline.

## Activation discipline

On promotion to `active`, this line must have all four:

1. **Validation hypothesis** — which specific SKU is this proving (a cleaning-guide book ≠ a baby-proofing kit ≠ a wedding planner)?
2. **Fixed-duration pilot** — concrete end date for the SKU test, not "ongoing."
3. **Bundle-link target** — which bundle / scenario does this feed, and how does the in-scenario offer surface work?
4. **Sunset or iterate clause** — by date X, SKU proves itself (≥ target units sold AND ≥ target conversion-to-sub), or we retire the SKU and document learnings.

See [REVENUE_LINES.md](../REVENUE_LINES.md) for the broader revenue-line discipline.

## Notes

- Print-on-demand (Amazon KDP, Printify) is the first production path for almost any own-produced SKU. Zero inventory risk, zero MOQ, near-zero downside on a failed test.
- Quality still matters — a poor POD product damages the brand. Design and proofreading discipline applies.
- Consumer products are complementary to, not a replacement for, the [affiliate / commerce](affiliate-commerce.md) line. They share the recommendation-blindness rule but have different cost structures and different legal surfaces.
