# Monetization — Plan Manager

This document records how the scenario could create revenue or support
a monetizable Vrooli capability. Keep it honest: an internal-capability
or `deferred` answer is better than inventing a commercial story.

## Purpose Of This Document

Use this document to answer:

- Is plan-manager a direct product, an internal capability, a SKU
  component, an add-on, or a service accelerator?
- Who would pay for it, and why?
- What packaging or pricing hypothesis exists today?
- What validation signal would justify treating it as a sold SKU?

## Role In Vrooli

plan-manager is an internal capability and a cost-reducer, not a
directly-sold product. It is the SSOT for implementation plans and a
guided wizard runtime whose explicit goal is to make authoring and
executing plans cheap enough (in tokens and required intelligence) that
small/local coding models can do the work that today demands large cloud
models. Its commercial value is therefore indirect: lower inference cost
and higher agent throughput across every scenario that plans work.

## Customer / Buyer

- Primary users: AI coding agents (especially small/local models)
  authoring and executing plans, and operators who view, manage, and
  triage plans and handoffs.
- Buyer: there is no external buyer in v1. The "buyer" is Vrooli itself
  — the platform funds plan-manager because it reduces the cost of all
  downstream agent work.
- Pain addressed: large-model dependency for planning is expensive and
  slow; plan-manager lowers the intelligence floor for that work.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | deferred | No standalone packaging until external demand is evidenced; plan-manager runs as an in-ecosystem runtime today. |
| Bundle component | deferred | Could be listed as part of an "agent productivity" capability bundle once such a bundle exists. |
| Add-on | not applicable | It is core platform plumbing, not an extension of another SKU. |
| Service/consulting assist | deferred | Plans could accelerate done-for-you delivery, but that is unproven and out of scope for v1. |

Packaging is deferred because plan-manager is currently consumed only by
other Vrooli scenarios and agents, not sold on its own.

## Pricing Hypothesis

- Pricing model: deferred. There is no price because there is no external
  SKU yet; the unit of value is realized as reduced inference cost, not a
  list price.
- Comparable products: none captured; cloud "planning agent" tooling is
  adjacent but not a direct analog to an internal SSOT.
- Cost drivers: local Go binaries plus a Vite UI and a SQLite store under
  `~/.vrooli` — negligible infrastructure cost. The dominant cost it
  *removes* is model inference spend, captured as per-plan velocity
  (time/tokens/iterations).

## Validation Plan

- Demand signal needed: evidence that running plan work on local models
  via plan-manager produces acceptable quality at materially lower token
  cost than the large-cloud-model baseline.
- Measurement: per-plan velocity emitted to meta-optimization-manager
  trials (see [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md)).
- Success threshold: a defensible cost-per-completed-plan reduction on
  local models without a regression in plan execution success rate.
- Revisit trigger: revisit monetization (packaging + pricing) only if an
  external party expresses willingness to pay for plan-manager as a
  standalone capability; until then it stays an internal cost-reducer.

## Current Status

Internal capability / cost-reducer. Not a directly-sold SKU. Pricing and
packaging are explicitly deferred for the reasons above, and this
scenario is pre-implementation (documentation-first), so no revenue
hypothesis is being pursued yet.

## Cross-References

- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — velocity signals that back business validation
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system architecture
