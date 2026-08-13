# Monetization — Offer Desk

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

| Role | Status | Detail |
|---|---|---|
| Internal capability | **committed** | This is the scenario's real job: it becomes the monetization team's catalog and lifecycle engine, replacing 22 hand-maintained markdown files whose rules nothing could enforce. |
| Meta-scenario | **committed** | Offer Desk is a capability other scenarios build on — it holds the offer graph that a commerce surface, a marketing surface, or a reporting surface all read rather than re-derive. |
| SKU component | **candidate** | Plausibly a business-bundle member alongside Money Ledger, as the "what I sell" half of an operator's commercial picture. |
| Direct product | **deferred, with a stated reason** | See below. |
| Service accelerator | **not-applicable** | Nothing here accelerates delivery of client work. |

## Why direct-product is deferred rather than pursued

Stated plainly so a future agent does not read the deferral as an oversight:

**Most operators do not have enough offers to need a graph.** A freelancer with three
services and one channel manages that in their head, and correctly so. The record-keeping
burden Offer Desk removes only becomes real somewhere north of a dozen live offers across
several channels — which is a population small enough that a standalone product aimed at it
is not obviously worth building.

**Its value is highest as a component, not a destination.** The scenario's distinctive
capability is *a declared condition, evaluated on a schedule, that surfaces itself when it
fires*. That is worth much more attached to something an operator already opens than as a
tool they must remember to visit — which is precisely the failure mode the scenario exists
to fix, and it would be ironic to reproduce it in the packaging.

**Money Ledger is the marketed surface.** The pairing is asymmetric on purpose: Money
Ledger is the polished product with the standalone thesis, and Offer Desk supplies the
half of the picture the ledger structurally cannot know. An operator asks "am I going to
make it?" — that is a ledger question. Offer Desk answers the follow-up the ledger raises:
"which of the things I sell is actually earning, and which is waiting on something I've
forgotten?" Neither scenario can answer that alone, which is stated identically in both
PRDs.

The right posture is therefore: build it well as a capability, let it make Money Ledger
materially better, and revisit standalone packaging only on the trigger below.

## What it contributes to the marketed product

These are the Money Ledger features that only exist because Offer Desk exists. They are
the monetization story — earned through the pairing rather than through separate packaging:

| Contribution | Why the ledger cannot do it alone |
|---|---|
| **Earned-versus-intended per offer** | The ledger knows an amount arrived and which account it hit. It has no concept of *what was supposed to earn*, so "this offer is active and has earned nothing" is unstatable without the offer graph. |
| **Revenue attributed to a line and a channel** | The ledger's postings attribute to accounts. Rolling them up to "affiliate commerce via app stores" needs the channel-feeds-line edges. |
| **A silent-failure alarm** | An offer in `active` with no matching money events for N periods is the highest-value alert either scenario can produce, and it requires both sides by construction. |
| **Goal thresholds that reference offers** | A financial goal like "subscription line covers burn" needs the line to be a record, not a string in a document. |

### The scan supports this framing

A competitive scan on 2026-08-13 (recorded in `path:scenarios/money-ledger/docs/business/GO-TO-MARKET.md`)
looked across aggregator-first consumer apps, local-first privacy apps, self-hosted open
source, plain-text accounting, and small-business accounting. **None of them joins a
lifecycle-enforced catalog of what should earn to a ledger of what did.** Finance products
track money; CRM and pipeline tools track deals with named customers, which is a different
shape and a different clock.

Two conclusions follow, and they point in opposite directions on purpose:

1. **It strengthens the pairing.** "This offer is active and has earned nothing for two
   months" appears to be unstatable in any surveyed product. That is the most defensible
   thing either scenario can say, and it requires both.
2. **It weakens standalone demand.** An unoccupied space with mature adjacent markets is
   more often a space nobody wants than one nobody noticed. Absence of a competitor is not
   evidence of demand — it is the reason the revisit trigger below requires operators to
   describe the pain *unprompted*, rather than requiring us to find a gap in a feature grid.

No feature candidates are recorded for this scenario. Its roadmap is whatever makes Money
Ledger's surfaces better, in the order the PRD already sequences.

## Customer / Buyer

- **Primary user:** the Vrooli `monetization` team, as its single instrument. Secondarily,
  an operator selling more than roughly a dozen things across multiple channels who keeps
  forgetting the ones parked on a condition.
- **Buyer:** not established, and not worth establishing before the revisit trigger.
- **Pain:** offers waiting on a condition are silently forgotten, because remembering to
  re-check them is a task no one is assigned and nothing enforces.
- **Existing alternatives:** a spreadsheet, a CRM's pipeline feature (wrong shape — CRM
  pipelines track deals with customers, not offers with lifecycles), or a project tracker
  with a "someday" column.

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Bundle component | candidate | The most plausible commercial form: shipped alongside Money Ledger, invisible as a separate purchase. |
| Standalone app | deferred | See "Why direct-product is deferred". Revisit trigger below. |
| Add-on | candidate | Defensible as an add-on to a bundle containing Money Ledger, since its value is expressed through the ledger's surfaces. |
| Service/consulting assist | not-applicable | |

## Pricing Hypothesis

`not-applicable` at this time, and deliberately so. Setting a price for a scenario whose
standalone demand is explicitly unvalidated would create canon out of a guess. Pricing
becomes a question when the revisit trigger fires, not before.

Cost drivers, when it matters: local runtime, SQLite, no shared resource required by any
P0 target. Scheduled trigger evaluation is the only recurring compute and it is small.

## Validation Plan

- **Demand signal needed:** three or more distinct operators outside Vrooli describing the
  *forgotten-condition* pain unprompted — not asking for a catalog, but reporting that they
  lost track of something they meant to revisit.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md).
- **Success threshold:** unset. Set from the project monetization taxonomy at promotion.
- **Revisit trigger:** **all three** must hold — (a) Money Ledger has paying users, (b) the
  actuals join (`OT-P1-002`) has run against real data for a full quarter, and (c) at least
  three external operators have described the forgotten-condition pain unprompted.

That trigger is itself an Offer Desk record once this scenario owns the catalog, which is
the intended recursion: the scenario that enforces revisit triggers should not hold its own
trigger as prose in a markdown file. Filing it as a record is the first real dogfooding test.

## Current Status

`internal-capability` — committed and needed regardless of any commercial outcome. Direct
product is **deferred with a stated reason and a concrete trigger**, not undecided.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Money Ledger's monetization story: `path:scenarios/money-ledger/docs/business/MONETIZATION.md`
- Project-level monetization strategy: `path:docs/monetization/README.md`.
