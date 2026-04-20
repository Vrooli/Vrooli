# Pricing

Pricing lives at the intersection of **delivery tier** × **catalog SKU**. This file is the matrix plus the rationale behind each price point. Most cells are `TBD` pre-launch — market-validator populates benchmarks; the operator sets prices.

## Current status: pre-launch

No SKU is selling yet. All numbers below are placeholder brackets, not decisions. They are here so the structure is ready when pricing is finalized.

## Matrix

| SKU \ Tier | Tier 1 (Bundle apps) | Tier 2 (Self-hosted) | Tier 3 (Hosted cloud) | Tier 4 (Hardware) |
|---|---|---|---|---|
| Business bundle | `TBD — target $29-$49/mo` | `TBD — target $49-$79/mo` | `TBD — target $79-$149/mo` | `north-star` |
| Lifestyle bundle | `TBD` | `TBD` | `TBD` | `north-star` |
| Property services (add-on) | `TBD` | `TBD` | `TBD` | — |
| Elder care (add-on) | `TBD` | `TBD` | `TBD` | — |
| Family with kids (add-on) | `TBD` | `TBD` | `TBD` | — |

Brackets above are rough targets from comparable SaaS bundles, not commitments. Market-validator produces proper benchmarks before any price is set.

## Pricing principles

These are durable decisions that shape how the matrix is filled in:

### 1. Tiers price up, not down

Higher tiers charge more. A Tier 3 hosted subscriber pays more than a Tier 2 self-host subscriber of the same bundle, because we absorb infrastructure cost and operational burden. Do not "discount hosted to drive adoption" — that collapses unit economics.

### 2. Add-ons price below their parent bundle

An add-on is an incremental purchase on top of a bundle, not a standalone product. It should feel cheap relative to the bundle that unlocked the category for the user. Rough heuristic: add-on price ≤ 50% of parent bundle price.

### 3. Same bundle, different tier — same scenarios

Paying more for a higher tier buys a different delivery mode of the same scenarios, not more scenarios. Do not gate bundle features by tier; that confuses positioning and creates feature-matrix-hell pricing pages.

### 4. Open source as price floor

The free self-hosted path with bring-your-own-keys sets a natural price floor. A paid tier must deliver enough convenience (integrated gateway, zero-config setup, managed hosting) that a typical user picks paid over free-with-effort. This is the value test every paid tier must pass.

### 5. No seat proliferation for solo operators

The business bundle is targeted at solo entrepreneurs and independent developers. Do not default to per-seat pricing — that alienates the target buyer. Household or single-operator pricing is the natural unit.

### 6. Annual discount is table stakes

When annual billing exists, offer ~15-20% off the monthly price. This is standard SaaS economics and materially improves runway math (annual cash upfront).

## What's NOT decided yet (and when it will be)

| Question | Decision timing | Owner |
|---|---|---|
| Actual monthly prices for the business bundle | Before Tier 1 launches | operator, informed by market-validator |
| Annual vs monthly ratio | Before Tier 1 launches | operator |
| Whether to offer a free trial or a reverse trial | Before Tier 1 launches | operator, informed by contrarian |
| Household/team/org pricing tiers | Post-launch, based on demand | operator |
| Whether add-ons are monthly or annual add-ons | When first add-on activates | operator |
| Enterprise / self-hosted-paid pricing (larger buyers) | When first enterprise inquiry arrives | operator |

## Benchmarks

Comparable SaaS bundles are cataloged in [BENCHMARKS.md](BENCHMARKS.md). Market-validator populates that file and references it when proposing pricing updates.

## Related documents

- [TIERS.md](TIERS.md) — why each tier exists and what it costs us
- [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md) — COGS per tier, LTV formulas, default-alive math
- [CATALOG.md](CATALOG.md) — SKU index
