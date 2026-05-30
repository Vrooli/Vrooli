# Go To Market — Ecosystem Manager

This document records positioning, channels, and validation for
Ecosystem Manager. Because this scenario is an **internal capability**
rather than a directly sold product today, "go to market" here means
its internal adoption story and the conditions under which it could
become an external offering — not an outbound launch plan.

## Purpose Of This Document

Use this document to answer:

- Who should know about Ecosystem Manager and rely on it?
- Through which channels does that happen?
- What claim about it is worth testing?
- What evidence would change its monetization plan?

## Audience And Positioning

- **Primary audience (internal):** Vrooli operators and agents who
  generate and improve scenarios and resources. For them, Ecosystem
  Manager is the default control plane for autonomous improvement work.
- **Secondary audience (future, external):** enterprise self-host
  customers who want an in-house "autonomous software factory."
- **Positioning:** the closed-loop controller that drives a target
  (scenario or resource) toward a measurable goal and keeps iterating
  until a stop condition is met — turning "build/maintain many internal
  services" into a metered, autonomous loop.
- **Main claim (to test):** steering a target with a profile moves it
  toward its goal faster and more completely than unsteered work.
- **Proof needed:** measured before/after deltas across real
  generation and improvement runs.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal adoption | Operators reach for ecosystem-manager (vs ad-hoc agent runs) for any generate/improve work | START-HERE + QUICKSTART, working auto-steer profiles | Share of generate/improve work that flows through the queue |
| Internal docs / records | Captured wins make the controller the obvious default | `swarm-manager records`, profile library | Reuse of profiles across targets |
| Enterprise tier (future) | A self-host tier surfaces this as the "factory" control plane | Tier packaging, access controls, pricing | A customer asking for an in-house improvement loop |

## Launch Motion

1. Make the internal path frictionless: orientation
   ([`../START-HERE.md`](../START-HERE.md)) and quickstart current and
   accurate.
2. Build and curate a profile library so common goals have a ready
   steering profile.
3. Capture measured uplift from real runs (before/after deltas).
4. Route a growing share of portfolio generate/improve work through the
   queue.
5. Only after the controller's leverage is demonstrated internally,
   evaluate exposing it as a higher-tier enterprise control plane.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Don't hand-run agents — queue a steered task and let the controller drive it to the goal" | Internal operators/agents | Profiles + queue throughput | active (internal) |
| "Investment in the controller compounds across every scenario it produces" | Vrooli strategy | Portfolio output metrics | hypothesis |
| "Run your own autonomous software factory" | Enterprise self-host (future) | Needs a tier + customer | deferred |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Steered vs unsteered run on the same target | Internal | Steered reaches the goal in fewer iterations / higher final metric | Promote steered-by-default if it wins |
| Profile reuse across targets | Internal | A profile drives multiple distinct targets successfully | Invest in the profile library |
| Enterprise interest probe | Future external | A self-host prospect requests an in-house improvement loop | Open the enterprise-tier packaging question |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — role in Vrooli and packaging
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — the closed-loop controller that backs the main claim
- [`../START-HERE.md`](../START-HERE.md) — orientation
- [`../../../../docs/monetization/README.md`](../../../../docs/monetization/README.md) — project-level monetization strategy
