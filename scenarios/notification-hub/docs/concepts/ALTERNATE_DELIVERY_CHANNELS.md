# Alternate Delivery Channels (Design — not yet implemented)

> **Status:** Design only. Nothing in this document is built yet. It is the
> specification for a future increment and exists so the decision and its
> rationale are recorded. Do not treat any route or type named here as present
> in the running scenario. See [`NOTIFICATIONS.md`](NOTIFICATIONS.md) for the
> contract that *is* implemented.

## Problem

A scenario may want its *own* branded self-delivery for its *own* notifications —
its app icon, its badge, a notification that visibly comes "from" that app when
the operator has it installed — rather than always arriving under the
notification-hub app. The naive way to get this is to let each scenario run its
own push stack and deliver directly. That is explicitly **rejected**
(see NOTIFICATIONS.md "Centralized delivery"): it duplicates VAPID keys, a
service worker, a subscription table, a delivery worker, quiet windows, and
retry per scenario; it forces a separate push-permission grant per app; and it
makes "deliver once" unsolvable, because Web Push has no read receipt, so two
independent delivery systems cannot agree on whether the human actually saw a
message.

The goal here is the branded-self-delivery *experience* without any of that
cost, and without ever giving up the single-install, deliver-once guarantee.

## Principle: one router, one dedupe authority

notification-hub stays the **sole router and the sole dedupe authority**. A
scenario never becomes a parallel delivery system. Instead a scenario can
register itself as an **alternate delivery channel** that the single hub router
*prefers* for notifications whose `source_scenario` is that scenario, and falls
back away from when that channel is absent or unhealthy. Because exactly one
component (the hub) decides where each notification goes, "deliver once" stays
trivially correct — it is a routing choice, not a race between senders.

This reuses the delivery seam the hub already has: the per-recipient
`channel_addresses` registry, the `machine_id` routing in `Process`, and the
pluggable `RemoteDelivery` / `ChannelDelivery` sender interfaces at
`deliverTarget` (`api/internal/hub/service.go`). A scenario channel is just
another sender behind the same delivery-attempt and receipt bookkeeping.

## Mechanism

1. **Registration.** A scenario registers a channel address of a new channel
   kind `scenario:<name>` for the recipient (via the existing
   `UpsertChannelAddress` surface or a thin dedicated registration), whose
   `address` is an endpoint the hub can POST a delivery to — the scenario's own
   notification receiver. Sensitivity-label approval applies exactly as it does
   for any channel: the `scenario:<name>` channel carries its own
   `approved_labels`, so an unapproved label still degrades to a content-free
   pointer.

2. **Delivery.** `deliverTarget` gains a case for `scenario:*` channels that
   dispatches to the scenario's receiver — structurally the same as
   `RemoteDelivery.Deliver` (which already forwards to a remote *machine*),
   except the target is a *scenario* receiver. The scenario receiver is
   responsible for showing the notification through its own installed PWA /
   badge. A successful dispatch writes a `receipt` row the same way web_push
   does.

3. **Routing preference.** In `Process`, when selecting targets for a
   notification whose `source_scenario` is `S`, if the recipient has a healthy
   `scenario:S` channel, route to it **first**. If that delivery fails (no
   receipt, or the channel reports unhealthy), fall through — in the same
   `Process` pass, via the existing retry / escalation path — to the recipient's
   normal targets (hub web_push, desktop, email). The notification still lands;
   it is never stranded on a dead scenario channel.

4. **Health.** The `scenario:S` channel contributes to
   `machine_channel_status` / `ChannelsStatus` like any other channel, so the
   router can skip a known-degraded scenario channel without a failed attempt
   first, and the operator can see its disposition.

## Why this preserves the guarantees

- **Deliver once:** the hub is the only decider; it chooses the scenario channel
  *or* the fallback, never both, so there is no double-send and no cross-system
  dedupe problem.
- **One install still gets everything:** a recipient with no scenario channel
  registered behaves exactly as today — hub push delivers all sources. The
  scenario channel is purely additive and per-source.
- **Branding without a second push stack:** the "from app X" experience for the
  common case is *already* delivered in the shipped increments — Phase 1 source
  branding + deep-link and Phase 2 per-source preferences make a hub-delivered
  notification clearly attributable to its source and open into that scenario.
  This design is only for the stronger case where a scenario wants delivery
  through *its own* installed surface.

## Preconditions / open questions (resolve before building)

- A scenario opting in must expose a delivery receiver and a registration path;
  define that contract (and its auth) before implementing.
- Decide whether the scenario channel's push permission is the scenario's own
  (its PWA) or whether the hub still owns the subscription and merely hands the
  render to the scenario — the former is the point of the feature; the latter is
  simpler but gives less branding.
- A health/liveness signal for the `scenario:S` channel so the router can prefer
  it only when it is actually deliverable.

## When to build this

Not now. Build it only when a concrete scenario needs branded self-delivery that
the shipped source-branding + preference model does not satisfy. Until then this
document is the agreed shape, and the delivery seam it depends on
(`deliverTarget`, `channel_addresses`, `machine_id` routing) is already in place.
