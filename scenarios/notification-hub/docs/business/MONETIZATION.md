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

## Market Position

Three distinct markets sit near this scenario, and only one of them is
worth entering.

| Market | Who sells there | Model | Why it is or is not ours |
|---|---|---|---|
| Notification infrastructure SaaS | Novu, Knock, Courier, SuprSend, MagicBell | Metered per notification or per event. Courier from ~$99/mo; Knock ~$250/mo for 250k notifications; Novu free when self-hosted. | **Not ours.** These sell to app developers who need to notify *their* users. That is the multi-tenant product this scenario deliberately deleted. Entering here means rebuilding profiles, API keys, and quotas. |
| Personal push pipes | ntfy, Gotify, Pushover, Apprise | Free and self-hosted, or ~$5 one-time (Pushover, capped at 7,500 messages/month). | **Substrate, not competitor.** These are transport. None of them decides *whether* to interrupt you, on which device, or what to do when the first channel fails. We ride on them. |
| Agent-to-human attention | No established player | Unpriced. Current practice is ad-hoc Slack callbacks and checkpoint files. | **The opening.** Gartner projects 40% of enterprise applications will ship task-specific AI agents by end of 2026, up from under 5% in 2025. Every one of them eventually needs a human, and nobody owns that seam locally. |

The gap between markets two and three is the whole thesis. ntfy is an
excellent pipe with no brain. Knock is an excellent brain that cannot run
on your own machines and bills per message. This scenario is the brain,
local-first, unmetered, and already aware of the owner's fleet through
`vrooli-bridge`.

The differentiator that no competitor in either adjacent market has: the
hub knows which of the owner's *machines* can reach a given device, and
can relay through one that can. A SaaS cannot do that because it has no
fleet. A push pipe cannot do that because it has no routing layer.

## Role In Vrooli

- **Internal capability: primary.** This is the scenario that makes every
  other scenario's output reach a human. Most of its commercial value is
  activation and retention on the bundle, not a line item.
- **SKU/bundle component: yes**, inside the `business` base SKU. An agent
  fleet nobody hears from is an agent fleet nobody trusts.
- **Add-on candidate: conditional.** An agent-oversight add-on becomes
  real only once acknowledgement and escalation ship. See *Packaging*.
- **Direct product: no.** Selling notification delivery to third parties
  is the rejected charter. Do not resurrect it through the back door.
- **Revenue line:** `subscription`, through bundle membership. Not a
  metered line of its own.

## Customer / Buyer

- **Primary user:** the machine owner, and the agents acting on their
  behalf.
- **Buyer:** the same person, buying the `business` bundle. There is no
  separate purchase decision for this scenario today.
- **Pain:** an autonomous agent that finishes, stalls, or needs
  permission has no reliable way to reach a human who is not currently
  looking at a terminal. The documented failure mode is concrete — an
  agent with send access and no approval gate dispatched 4,000 emails in
  40 minutes because nothing paused to ask.
- **Existing alternatives:** Slack webhooks (needs Slack, unreliable to a
  locked phone, no fleet awareness), raw ntfy or Pushover (no routing, no
  quiet hours, no acknowledgement), or nothing at all, which is the
  common case.

## What To Add Or Emphasize

Four changes turn this from infrastructure into something with a
defensible commercial story. **All four were accepted into the charter on
2026-08-17** and are now operational targets rather than proposals.

1. **Acknowledgement and response — now OT-P1-009**, promoted from P2. A
   delivered notification can carry a decision back to the requesting
   scenario. That single capability changes what this scenario *is*: an
   outbound pipe becomes the human-in-the-loop gate for every agent in
   Vrooli. It is also the one thing neither adjacent market offers
   locally.
2. **A blocking agent primitive — now OT-P1-010**, new. The PRD had a verb
   for *tell* and none for *ask*. An agent needs one call that delivers a
   question and returns the human's answer or a timeout —
   `notification-hub ask --timeout 30m`. This is the interface every
   agent runtime is currently hand-rolling.
3. **Escalation paired with acknowledgement — now OT-P1-011**, promoted
   from P2 and explicitly sequenced with the other two. An unanswered
   *approval* is precisely the case that needs escalating.
4. **Fatigue control as the headline, not delivery.** Quiet hours,
   duplicate suppression, and digest collapsing were already P0/P1
   because a spine that fires too often gets muted. The PRD Overview now
   states this as part of the capability rather than leaving it to
   requirements, because it is the sharpest contrast against dumb pipes
   and belongs in the first sentence of any positioning.

Together these three targets are the add-on hypothesis made testable:
they ship as one slice, ahead of the fleet work, and need no second
machine.

Deliberately **not** recommended: per-notification metering, a template
marketplace, or campaign sequencing. Each drags the scenario back toward
the SaaS charter this rewrite exists to escape. The no-metering position
is recorded as a durable decision in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Packaging

| Packaging Option | Status | Details |
|---|---|---|
| Bundle component | **active hypothesis** | Ships inside the `business` base SKU as infrastructure. No separate price. This is the near-term answer. |
| Add-on — agent oversight | candidate | Approval gates, escalation chains, and a delivery-receipt audit trail sold as an oversight capability rather than as notification volume. Gated on OT-P1-009 through OT-P1-011, which are now committed P1 targets rather than open questions. |
| Standalone app | deferred | Viable only as an open-source distribution wedge, not as a paid product. See `GO-TO-MARKET.md`. |
| Service/consulting assist | candidate | "Your agents can reach you" is a concrete deliverable in a done-for-you engagement, and it demonstrates the fleet story in one sitting. |
| Hosted relay | deferred | The one honest place a metered tier could live: relaying for owners with no always-on machine. Contradicts local-first positioning; revisit only on real demand. |

## Pricing Hypothesis

- **Model:** bundle inclusion. **Do not meter notifications.** Metering is
  the competitors' shape, it penalizes exactly the high-frequency agent
  use this scenario exists to serve, and it is incoherent alongside
  OT-P0-009's promise that the thing runs on your own hardware.
- **Comparable anchors:** Courier ~$99/mo and Knock ~$250/mo for metered
  multi-tenant infrastructure; Pushover $5 one-time for a capped personal
  pipe; ntfy and Gotify free when self-hosted. The spread between the
  free pipes and the metered platforms is the room the add-on would
  occupy, and it is priced on oversight, not on volume.
- **Willingness-to-pay evidence:** none captured yet. The Pushover price
  point proves people pay something for reliable personal push; it does
  not prove they pay for routing.
- **Cost drivers:** effectively zero marginal cost. Local runtime, no
  resource dependency, no per-message fee unless the owner chooses a paid
  provider. This is a structural advantage over every metered competitor
  and should be stated plainly in positioning.

## Validation Plan

- **Demand signal needed:** agents choosing to route through the hub when
  a cheaper path exists. Concretely: count of `ask`-shaped requests from
  other scenarios per week, once the primitive exists.
- **Counter-signal to watch:** notifications delivered but never
  acknowledged. High delivery with low acknowledgement means the spine is
  being muted, which invalidates the oversight thesis before any pricing
  question matters.
- **Channel:** see [`GO-TO-MARKET.md`](GO-TO-MARKET.md); `oss-discovery`
  and `skill-registries` are the two live candidates.
- **Success threshold:** to be set with the monetization team against the
  project-level taxonomy. Do not invent one here.
- **Revisit trigger:** when acknowledgement and escalation ship and the
  hub has served a real approval gate for a real agent run.

## Current Status

`draft` — the market position is researched and specific, and the four
product recommendations are accepted into the PRD as OT-P1-009 through
OT-P1-011 plus the Overview reframing. The pricing hypothesis is not yet
validated by any demand evidence, and no notification has been delivered
yet, so nothing here is proven. The bundle-component role is safe to act
on now. The add-on role stays a hypothesis until the acknowledgement
slice has served a real approval gate for a real agent run.

Project-level canon is authoritative for catalog, tier, and pricing
decisions. Nothing in this document is a canon edit; it is a scenario-side
proposal for the monetization team to accept or reject.

## Cross-References

- [`../START-HERE.md`](../START-HERE.md) — orientation workflow
- [`../../PRD.md`](../../PRD.md) — product requirements, especially OT-P2-004 and OT-P2-005
- [`GO-TO-MARKET.md`](GO-TO-MARKET.md) — channel and launch plan
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — telemetry needed for business validation
- Project-level monetization strategy: `path:docs/monetization/README.md`
- Revenue lines: `path:docs/monetization/catalogs/revenue-lines/subscription.md`
- Channel registry: `path:docs/monetization/catalogs/channels/README.md`
