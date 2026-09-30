# Monetization

> **This document routes; it does not decide.** Pricing, bundle membership, and
> monetization posture are operator-curated canon. Confirm every placement below
> against the sources in the routing table before it reaches a plan.

## Routing

| Need | Canon (read-only) |
|---|---|
| Bundle / SKU membership | `docs/monetization/catalogs/CATALOG.md`, `scenario-sku-map.json` |
| Whether to charge at all, and posture | `docs/monetization/strategy/STRATEGY.md` |
| Free vs metered vs gated, and how to wire each | `docs/concepts/PAID_FEATURES.md` |
| Entitlement wiring | `prompt-manager skill read bundle-integration-steer` |

Two hard rules carried from `PAID_FEATURES.md` and applied below:

1. **Never gate a capability a self-hoster could run with their own keys.** BYOK
   must stay valid.
2. **Route metered and gated features through LPBS** rather than reinventing
   credits or entitlements here.

## Proposed placement per capability

| Capability | Mode | Reasoning |
|---|---|---|
| Style catalog — browse, filter, author, fork | **free** | No marginal cost. The catalog is also the discovery surface; gating it would hide the product. |
| `procedural` and `procedural-treated` rendering | **free** | Zero marginal cost, runs offline, reproduces from a seed. A self-hoster can run these with no account, so rule 1 forbids gating them. |
| Legibility gate and contrast measurement | **free** | Pure local computation, and it is an accessibility feature. Charging for the check that keeps text readable would be the wrong instinct. |
| `guided` and `synthesized` rendering | **metered** | Real inference spend. Meter the inference, not the feature — a BYOK user supplying their own gateway route pays their provider directly and is not metered by us. |
| Contact sheet sweeps | **metered when model-backed** | A sweep is N renders; it inherits the mode of the lane it sweeps. A procedural sweep stays free. |
| Style pack export / import | **free** to export, **gated** for curated first-party packs | Exporting your own work must stay free. Curated packs are a product. |
| Desktop app | **bundled** | Ships in the business bundle via `scenario-to-desktop`. |

The shape worth preserving: **the free tier is not a demo.** The procedural lanes
cover a genuinely useful part of the design space — every `non_representational`
subject and most `synthetic` treatments — so an unpaid user can produce
production-quality backdrops. What money buys is the representational subjects
that need a model.

## Why this scenario has a monetization story at all

Recorded because it is easy to mistake this for internal tooling:

- **The immediate reason to build it is internal** — better landing pages and
  higher conversion across the portfolio.
- **The product reason is independent of that.** Designers currently sell exactly
  this output. The reference material that prompted this scenario was designers
  on X posting generated backdrop assets and funnelling that attention into paid
  landing-page work. The demand is observable, not hypothetical.
- **Two revenue paths, not one.** Direct bundle value, plus margin on inference
  routed through `ai-gateway` when a subscriber uses our configured route rather
  than their own keys.

## What must not be gated

Stated explicitly, because these are the tempting mistakes:

- The legibility gate. It is an accessibility feature.
- Alt text authoring, or any release-blocking correctness check.
- Export of styles the user authored.
- Any procedural lane, under any circumstance — rule 1 is not negotiable.

## Open questions for the operator

1. Bundle placement — headliner or depth within the business bundle?
2. Metering unit for model-backed renders — per candidate, or per accepted
   release? Per-candidate is more honest about cost; per-release is friendlier to
   exploration, which is the behaviour the product is trying to encourage.
3. Whether curated style packs are a bundle inclusion or a separate SKU.

None of these is decided here. Route to canon.
