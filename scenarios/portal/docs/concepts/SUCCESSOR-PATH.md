# Portal successor path

Portal is the Vrooli suite's successor chat and app surface for **agent-inbox**
and **app-monitor**. New suite chat, app discovery, and app loading work belongs
in Portal. Portal's chat-first workspace is intended to combine conversations,
contextual widgets, and an app drawer so operators can discover and open suite
apps from one place.

This is a product direction and ownership rule, not a claim that Portal already
has complete feature parity with either predecessor. Keep the existing
agent-inbox and app-monitor implementations available as legacy references while
Portal's corresponding capabilities mature. Do not add new suite-surface
features to those predecessors; put the work in Portal and link it to the
relevant Portal requirement. Preserve owner boundaries: Portal presents and
routes to capabilities, while each owning scenario and the control plane retain
their state, authorization, and host-operation rules.

## Where to continue

- **Agent Inbox work:** use [Agent Inbox's migration note](../../../agent-inbox/README.md)
  and this document; implement new suite chat behavior in Portal.
- **App Monitor work:** use [App Monitor's migration note](../../../app-monitor/README.md)
  and this document; implement the consolidated operator-facing app surface in
  Portal while keeping monitoring and host-operation ownership with their
  existing owners.
- **Portal work:** start with the [Portal PRD](../../PRD.md), then this
  successor path. The chat, widgets, and app drawer are the intended unified
  surface; TV voice control is only a possible marketing opportunity and does
  not set priority.
- **Offer Desk work:** Portal is already a marketed deliverable at release rank
  9 and belongs to both the Business Bundle and Lifestyle Bundle in the live
  Offer Desk catalog. The bundles are the Vrooli subscription lineup. Check the
  Offer Desk catalog and its typed `belongs_to` edges before changing that
  lineup; do not infer a new price, tier, or activation state from this product
  direction.

This work supports **O5 — Vrooli subscription**. It establishes Portal's place
in the suite and subscription catalog; it does not claim that Portal is ready to
sell or that O5's release and paid-customer gates have passed.
