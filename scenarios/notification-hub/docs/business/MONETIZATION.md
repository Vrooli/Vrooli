# Monetization — Notification Hub

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

## Role In Vrooli

**Internal capability. Not a product, and deliberately not one.**

- Direct product: **not applicable.** The scenario was regenerated
  specifically to stop being a multi-tenant notification service. Its
  predecessor carried profiles, per-tenant API keys, provider
  registries, and billing quotas aimed at customers that did not exist,
  and that scaffolding is exactly what made it unfinishable. Rebuilding
  it would undo the decision, not extend it.
- Internal capability: **yes, and a load-bearing one.** Any scenario
  that needs to reach a human depends on this one. Reaching a human
  reliably is a precondition for several revenue lines rather than a
  revenue line itself.
- SKU/bundle candidate: not applicable on its own.
- Revenue line: none directly.

## Customer / Buyer

- Primary user: the machine owner, plus every agent and scenario in the
  fleet that needs to tell a person something.
- Buyer: none. Nobody buys this scenario. Its cost is justified by the
  scenarios it unblocks.
- Pain it removes: without it, every scenario that wants to reach a
  human either implements delivery itself or stays silent. Both are
  worse than one shared spine — the first duplicates retry logic,
  credentials, and quiet-hour handling in N places; the second means the
  fleet does work nobody hears about.
- Existing alternatives: a per-scenario `curl` to a webhook. That is
  what the absence of this scenario actually produces, and it has no
  preferences, no retry, no dedupe, and no record.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Standalone app | not applicable | A personal notification router is a solved commodity (ntfy, Pushover, Healthchecks). Competing there has no advantage and is not the point. |
| Bundle component | conditional | If a Vrooli SKU ever ships to another operator, that operator needs notifications too — but as included plumbing, not as a line item. |
| Add-on | not applicable | Nothing here extends another SKU independently. |
| Service/consulting assist | indirect | It makes done-for-you delivery more credible by making the system able to report on itself. Not separately billable. |

## Pricing Hypothesis

- Model: none. This scenario is not priced.
- Cost drivers: effectively zero at rest. No resource dependency, no
  hosted runtime, embedded SQLite. The only variable cost appears if the
  SMS channel (OT-P2-002) is enabled, which is billed per message by
  Twilio — the reason SMS is P2 rather than P0 despite being easy.
- Comparable products: ntfy (free, self-hostable), Pushover (~$5
  one-time per platform), PagerDuty (per-seat, a different problem).
  These bound what the capability is worth: not much, standalone.

## Validation Plan

The honest validation question is not "will someone pay for this" but
"does the fleet actually use it".

- Demand signal needed: at least three scenarios raising notifications
  through the hub rather than staying silent or printing to a log.
- Counter-signal to watch for: the owner muting a channel. A muted
  channel means the scenario is producing noise instead of signal, and
  that is a product failure regardless of delivery statistics.
- Success threshold: the owner learns about something they cared about
  through a notification before they would have found it otherwise.
- Revisit trigger for this document: a Vrooli SKU ships to a second
  operator, at which point notification delivery becomes part of a
  packaged offering and needs a cost model.

## Current Status

`not-applicable` as a revenue line, `active` as an internal capability.
This is a deliberate classification, not an unfilled stub. Do not
promote it to a product without superseding the regeneration decision in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why multi-tenancy was removed
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`.
