# Go To Market — Notification Hub

This document records launch strategy, positioning, channels, and
validation experiments for the scenario.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

There is no external market motion, and inventing one would be dishonest
scaffolding. See [`MONETIZATION.md`](MONETIZATION.md) — this is internal
infrastructure whose value is measured in the scenarios it unblocks, not
in reach.

The internal audience is real, though, and worth being deliberate about:

- **Audience** — the other scenarios and agents in the fleet, which must
  learn that reaching a human is a call to this scenario rather than
  something to implement locally or skip.
- **Positioning** — the one place that knows how to reach the owner. A
  caller supplies what happened and how urgent it is; it supplies
  nothing about channels, addresses, retries, or timing.
- **Main claim** — a scenario that calls this one can stop caring
  whether the owner is asleep, which device is nearby, or whether a
  provider is down.
- **Proof needed** — the first three real callers, and a delivery
  timeline showing their notifications arrived.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Fleet adoption by other scenarios | A scenario with a `// TODO: notify` comment will call the hub once calling it is a one-liner. | A CLI verb and a Connect client short enough to paste, plus a worked example in `../reference/cli-commands.md`. | Count of distinct calling scenarios. The predecessor accumulated exactly one TODO comment and zero callers in ten months. |
| Agent usage | Agents finishing long work should tell the owner rather than leaving a result in a transcript. | A discoverable CLI verb that agents find through `prompt-manager discover`. | Notifications raised with an agent caller identity. |
| Owner-facing console | The delivery timeline is the surface that builds trust in the whole mechanism. | The timeline UI (OT-P0-004). | The owner checks the timeline instead of asking whether something ran. |

## Launch Motion

1. Deliver one real notification to the owner's iPhone. Nothing before
   this counts, and the predecessor scenario is the reason that
   sentence is written this bluntly.
2. Ship the routing core so a caller can rely on preferences and
   duplicate suppression rather than reimplementing them.
3. Publish the one-line calling convention in
   [`../reference/cli-commands.md`](../reference/cli-commands.md) and
   [`../reference/api-endpoints.md`](../reference/api-endpoints.md).
4. Convert the first real caller. `calendar` carries a
   `// TODO: Integrate with notification-hub` and is the obvious first
   candidate.
5. Add the relay lane so `minimouse` can serve Apple-only channels.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Tell the owner; don't decide how." | Scenario and agent authors | The routing core exists and is tested without a network. | pending OT-P0-005 |
| "It arrived, and here's the proof." | The owner | The delivery timeline with per-attempt receipts and reasons. | pending OT-P0-004 |
| "It won't wake you up." | The owner | Quiet hours honored, with `critical` as the only override. | pending OT-P0-005 |
| "It runs on the Mac too." | Fleet operators | The same build running on `minimouse` with no Docker. | pending OT-P1-001 |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Convert the `calendar` TODO into a real call | Fleet adoption | The call is under ten lines and needs no notification-specific knowledge. | If it takes more, the API is wrong — fix the API rather than documenting around it. |
| Run for two weeks and count mutes | Owner console | Zero channels muted by the owner. | A mute is the clearest possible signal that the scenario is producing noise. Treat it as a P0 bug, not a preference. |
| Ask the owner what they learned first from a notification | Owner | At least one thing they would otherwise have missed. | If nothing, the ingress rules are watching the wrong things, not the delivery layer failing. |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — packaging and pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
