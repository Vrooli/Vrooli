# Monetization — Infrastructure Manager

**Status: not-applicable, deliberately.** This scenario is internal
platform capability and is not a product. This document records *why*
rather than deferring the question, so it is not reopened by default.

## Purpose Of This Document

Use this document to answer whether and how this scenario earns its keep.
Monetization *strategy* — whether to charge at all, pricing, bundle
membership — is operator-curated canon in `path:docs/monetization/`. This
document routes to that canon; it never decides it.

## Role In Vrooli

This scenario is a **meta-scenario**: other scenarios and teams build on it
as a tool. Its output is the `infra-health` team's capacity to regulate the
platform, not a revenue, growth, or customer outcome of its own.

The team's own operating model already states the position this document
inherits:

> This team is second-order: its output is the capacity of the other teams
> rather than a revenue, growth, or scenario outcome of its own.
> — `docs/infra-health/operating/OPERATING_MODEL.md` § Outcome contribution

It serves objective `I2` (coherence), an *instrumental* objective. Instrumental
objectives are justified only by the terminal objectives they protect, which is
why the value case here is "reliability work stops being open-loop", not
"customers pay for this".

## Customer / Buyer

None. There is no external buyer and no internal chargeback model.

| Candidate | Assessment |
|---|---|
| External operator running Vrooli | Not a buyer. If self-hosters ever want platform reliability visibility, it arrives through whatever surfaces the platform already ships, not as a separately sold capability. |
| Vrooli teams | Users, not buyers. The `infra-health` team is the daily reader; the operator reads it at the morning vision walk. |

## Packaging

Not packaged. No bundle claim, no SKU, no entitlement wiring.

Two hard rules from `path:docs/concepts/PAID_FEATURES.md` are worth stating
explicitly, because a reliability board is exactly the kind of capability
someone might later be tempted to gate:

- **Never gate a capability a self-hoster could run with their own keys.**
  Every input here is local — autoheal, capacity, storage, test-genie,
  system-monitor. There is no metered upstream cost to recover.
- **Route any metered or gated feature through LPBS** rather than reinventing
  credits or entitlements. Not applicable today; recorded so a future change
  starts in the right place.

## Pricing Hypothesis

None. Pricing is not deferred pending research — it is not applicable, because
there is no buyer and no billable unit.

## Validation Plan

No monetization experiments are planned. The scenario's value is validated the
same way its own targets are: by whether the error signal it produces gets
acted on and whether the fixes move their sensors (`OT-P1-003`, actuation
efficacy). That is the only "does this earn its keep" measurement that applies.

## Current Status

| Field | Value |
|---|---|
| Monetized | No |
| Bundle | None |
| Free / metered / gated | Free — every capability, permanently |
| Revisit trigger | A decision to sell platform-reliability visibility as a product surface. That would be a `docs/monetization/` canon change and a director-swarm decision, not a change made here. |

## Cross-References

- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — audience and positioning (also internal-only)
- [`../../PRD.md`](../../PRD.md) — the capability and its value promise
- `path:docs/concepts/PAID_FEATURES.md` — the free / metered / gated contract
- `path:docs/monetization/` — operator-curated monetization canon
- `docs/infra-health/operating/OPERATING_MODEL.md` — the team's second-order outcome statement
