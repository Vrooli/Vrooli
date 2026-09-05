# Monetization — Scenario Authenticator

This document records how the scenario relates to revenue. Keep it
honest: `not-applicable` is better than inventing a commercial story.
Everything below is a pre-launch **hypothesis** derived from
[`../../PRD.md`](../../PRD.md) (Appendix D), not a committed plan, and it
**sets no prices** — pricing posture is routed to canon below.

## Purpose Of This Document

Use this document to answer:

- Is this scenario a direct product, internal capability, SKU component,
  add-on, or service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists?
- What validation signal would justify more investment?

## Role In Vrooli

scenario-authenticator is **foundational infrastructure**, and its
monetization model is deliberately *indirect*:

- **Foundational interface-enabler (primary).** Its product is being
  consumed by other scenarios as a dependency — the permanent identity
  layer the fleet builds on. Every adopter that reuses it instead of
  rolling its own auth raises the platform's aggregate value. This is the
  compounding-intelligence side, not a direct revenue line.
- **It does NOT meter itself.** The authenticator never gates a capability
  a self-hoster can run with their own keys, and it never charges
  per-token, per-login, or per-seat for the auth primitive itself.
- **It ENABLES metered/gated tiers in adopting products.** Monetization is
  realized *one layer up*: an adopting product (landing pages, LPBS,
  hosted SaaS) gates **its own** user tiers on the verified identity this
  scenario issues. The identity is the substrate; the entitlement is the
  product. In practice this is **LPBS entitlements** keying off the
  realm + claims emitted here — see
  [`subscription-entitlements-system`](../../../landing-page-business-suite/docs/integrations/subscription-entitlements-system.md).

## Customer / Buyer

- **Primary "customer" (early):** other Vrooli scenarios acting as Relying
  Parties — device-sync-hub first, then landing pages, LPBS, and hosted
  SaaS products. They "pay" in adoption, not dollars; that adoption is the
  first and most certain validation signal.
- **Self-hosters / operators:** run the full identity capability free,
  with their own signing keys and storage. Never gated.
- **Indirect paying customer:** the end users of an *adopting* product who
  buy a tier in that product — where the identity issued here is what the
  tier gates on.
- **Pain it removes:** every product that needs users would otherwise
  reinvent accounts, sessions, MFA, token signing, and federation — the
  exact shared-auth blast radius this rewrite eliminates.

## Packaging

| Packaging Option | Status | Notes |
|---|---|---|
| Standalone app | not the model | The IdP is consumed API-to-API by other scenarios, not sold as a standalone end-user app. |
| Bundle component | hypothesis (primary) | The identity layer embedded in product SKUs (LPBS, hosted SaaS) that monetize *their* user tiers on top of it. |
| Add-on | n/a | It is infrastructure beneath SKUs, not an add-on to one. |
| Service/consulting assist | hypothesis | Could accelerate done-for-you "stand up auth for your product" engagements; secondary. |

**Free / BYOK is structural, not a tier:** self-hosters get the complete
capability with their own keys at zero marginal cost. The
self-hostability of the auth primitive is a feature of the model, not a
limitation of a free tier.

## Pricing Hypothesis

This scenario **does not set its own prices and has no SKU of its own.**
Pricing for the products that monetize *on top of* it is owned by
monetization canon, not by this document:

- **Posture & strategy:** [`docs/monetization/strategy/STRATEGY.md`](../../../../docs/monetization/strategy/STRATEGY.md)
- **Delivery tiers:** [`docs/monetization/strategy/TIERS.md`](../../../../docs/monetization/strategy/TIERS.md)
- **Any price a surface cites:** [`docs/monetization/strategy/PRICING.md`](../../../../docs/monetization/strategy/PRICING.md)
- **What can be sold/bundled (incl. scenario→SKU map):** [`docs/monetization/catalogs/CATALOG.md`](../../../../docs/monetization/catalogs/CATALOG.md)

The cost driver here is local runtime (SQLite + Redis); there is no
third-party API spend and no per-call cost to pass through.

## Validation Plan

- **Adoption is the demand signal.** The first signal is internal: count
  of consuming Relying Parties and their auth volume, starting with the
  device-sync-hub live migration (OT-P0-012).
- **Telemetry source:** the OBSERVABILITY measures (logins, tokens issued,
  active sessions, per-realm activity) feed adoption analysis — see
  [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).
- **Monetization is validated in adopting products, not here.** The
  question "did this make money" is answered by LPBS/SaaS tier conversion
  that gates on identity issued here — route that validation to
  [`GO-TO-MARKET.md`](GO-TO-MARKET.md) and the monetization team's
  taxonomies.
- **Revisit trigger:** P0 auth core green + first RP live; then evaluate
  whether the "stand up auth for your product" service angle is worth
  pursuing.

## Current Status

The local IdP foundation is implemented, but monetization remains
hypothesis-stage and **indirect by
design**: free/BYOK for self-hosters, value realized by enabling metered
tiers in adopting products. This document states the hypothesis and routes
to canon — it does not price anything.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — Appendix D (ecosystem-fit + monetization classification)
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — adoption path and channels
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — adoption telemetry
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — LPBS as a downstream consumer
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization plan of record (pricing canon)
