# Delivery Tiers

The **delivery tier** is how a Vrooli bundle reaches the user. Tiers are orthogonal to the [catalog](../catalogs/CATALOG.md): a given bundle can be sold at any active tier, and pricing sits at the intersection (see [PRICING.md](PRICING.md)).

> **Do not confuse this commercial vocabulary with the technical deployment
> vocabulary.** In the Deployment Hub, deployment Tier 1 is the full local
> Vrooli stack and deployment Tier 2 is the desktop-app target. In this
> monetization plan, commercial Tier 1 is the bundle-app product and commercial
> Tier 2 is the self-hosted full-runtime product. Use the qualifier when a
> document crosses those boundaries.

## Tier lifecycle

Like SKUs, tiers flow through a status lifecycle:

| Status | Meaning |
|---|---|
| `active` | Currently being sold and delivered. |
| `candidate` | Documented with an explicit activation trigger. Team tracks readiness but does not actively plan delivery. |
| `north-star` | Directional marker only. **Must not be planned against without explicit operator initiation.** |
| `retired` | Previously offered, no longer sold. |

Candidate tiers can become active only when their trigger fires and the operator promotes them. Agents do not self-promote tiers.

## The four tiers

### Tier 1 — Bundle apps (`active`)

Individual desktop/iOS/Android apps wrapped from a Vrooli scenario. A subscriber downloads the apps in their bundle. Each app is self-contained and talks to the Vrooli backend only for subscription validation and (on paid tiers) integrated API routing.

**Pros for the user:** lightweight, familiar, no infrastructure, works on existing devices.
**Cons for the user:** each app is its own process (memory/storage overhead); apps can't share as much context as a full Vrooli instance would; limited to what individual apps expose.

**Cost-of-goods to Vrooli:** gateway token cost is the dominant variable per-user line. Paid Tier 1 subscriptions include the integrated API gateway (LLMs, STT/TTS, embeddings, coding agents) with a credit allowance — that's the core reason to pay over running the OSS apps with bring-your-own keys. Every subscriber drives gateway usage with wholesale-to-retail margin. Fixed costs (app store fees, signing certs, CDN) are one-time or amortized. No hosting per user. Detailed unit economics in [FINANCIAL_MODEL.md §Tier 1](../evidence/FINANCIAL_MODEL.md).

**Current state:** in-progress. `web-console` and `git-control-tower` being packaged for Tier 1 delivery as the business bundle's initial headliners.

### Tier 2 — Self-hosted full Vrooli runtime (`candidate`)

The user downloads and runs the full Vrooli project on their own hardware. The subscription provides the convenience layer (integrated API access, gateway routing) and confirms bundle membership for the scenarios the subscriber owns. The free/OSS path remains open — a user who brings their own API keys runs the same runtime at no cost, just without the integrated gateway.

**Revisit trigger:** *"Revisit when all three are true: (a) the business bundle has paying subscribers, (b) the onboarding flow supports account sign-in to activate the convenience layer, (c) a license/entitlement gateway exists (either a new resource or extension of an existing one)."*

**Pros for the user:** full ecosystem power, context sharing across all scenarios, agent-driven discovery of bundle capabilities, deep customization. Developers especially value this.
**Cons for the user:** requires a capable machine (ideally dedicated); requires setup time; takes responsibility for updates and crash recovery.

**Cost-of-goods:** gateway API token costs are variable per user with pass-through margin. Support burden is real (diverse hardware, networks, edge cases). Hosting is zero.

**Capability prereqs before this tier can activate:**
- License/entitlement gateway (likely a new resource or part of an existing one like landing-page-business-suite)
- Onboarding flow that includes account sign-in
- API routing / metering gateway (could belong to LPBS or stand alone)
- Graceful handling of offline mode, license revocation, seat-sharing (this is solved territory but not free — JetBrains / 1Password / Adobe have the patterns)

### Tier 3 — Hosted cloud Vrooli (`candidate`)

Vrooli provides a managed, per-account Vrooli instance on our infrastructure. The user connects remotely (web console, desktop thin clients) and runs their agents and scenarios on hardware we provision. Same runtime as Tier 2, just managed for them.

**Revisit trigger:** *"Revisit when all three are true: (a) Tier 2 is `active` and shipped, (b) `scenario-to-cloud` can reliably provision a full Vrooli instance per account on a VPS or container platform, (c) a meaningful fraction of surveyed users prefer hosted over self-host (or complained about self-host setup friction)."*

**Pros for the user:** zero setup, no hardware requirements, no maintenance, accessible from any device, always on.
**Cons for the user:** dependency on Vrooli operational reliability; less privacy than local; higher price to cover hosting costs.

**Cost-of-goods:** real per-user infrastructure cost. This is the tier where unit economics matter most — the financial model must assume hosted users have materially higher monthly COGS than Tier 1 or Tier 2 users.

**Capability prereqs:**
- `scenario-to-cloud` matured to per-account full-runtime provisioning
- Tier 2 operational pattern proven (since Tier 3 is Tier 2 with us as the host)
- Billing integration aware of hosting costs
- Operational monitoring + on-call posture

This is probably the **largest long-term revenue surface** because it captures users who would otherwise churn on self-host setup friction. But it is also the tier that most changes the company's operational posture (from software shop to infrastructure operator).

### Tier 4 — Hardware appliance (`north-star`)

A dedicated Vrooli machine sold to households or small businesses. Runs the full stack locally, maximizes hardware utilization, preserves privacy. Could be a one-time purchase, a subscription-included appliance, or a combined hardware + subscription plan.

**Status: `north-star`. Not to be planned against without explicit operator initiation.**

This tier is captured here as a directional marker so:

- No agent accidentally proposes work that assumes Tier 4 is coming.
- The team knows why Tier 2 and Tier 3 investments (onboarding, license gateway, full-runtime packaging) are structurally aligned with a future hardware move.
- When market conditions and capability mature, there is a pre-existing place to promote this tier to `candidate`.

**Why it stays north-star and not candidate:**
Hardware is a fundamentally different business. Inventory, BOM, RMA, certifications (FCC, UL, energy), fulfillment, physical support SLAs — these are not adjacent to software operations. Entering the hardware business is a deliberate strategic choice by the operator, not a downstream consequence of shipping tiers 1-3. The trigger for this tier is an explicit operator decision, not a condition check.

## How tiers interact with the catalog

A subscriber on any tier buys a bundle from the catalog. The tier determines *how* they receive the bundle's scenarios:

- **Tier 1:** gets the apps for their bundle's scenarios
- **Tier 2:** gets the full Vrooli runtime; their subscription unlocks the bundle's gated scenarios
- **Tier 3:** same as Tier 2, but we run it
- **Tier 4:** same as Tier 2 on dedicated hardware we ship

A bundle's scenarios are the same regardless of tier. Packaging, polish, and entitlement enforcement differ.

## Tier × bundle pricing

Pricing sits at the intersection of tier and bundle. See [PRICING.md](PRICING.md) for the matrix.

## Standing guardrails

1. **Tier 4 is `north-star`.** Nothing activates it except explicit operator initiation.
2. **No tier activates without its trigger firing and human promotion.** Agents do not self-promote tiers.
3. **Per-tier cost-of-goods must be visible in the financial model.** Mixing them obscures unit economics.
4. **A tier's capability prereqs, once known, should appear both here and in [TELEMETRY_ROADMAP.md](../evidence/TELEMETRY_ROADMAP.md)** — the roadmap is the "where do we build this" view; this file is the "why does the monetization plan need this" view.
