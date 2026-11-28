# Portfolio-manager capability dependencies

Curated capabilities that can affect portfolio planning, goal discovery, commissioning
and follow-through. This is a dependency map, not current health. The heartbeat reads
it alongside the portfolio pulse. Observations belong in dated portfolio records.

## Swarm Manager

- **Used for:** authoritative goals, milestones, backlog, dependencies and approved work.
- **Needed for:** current portfolio decisions and applying operator-approved items.
- **Fallback:** inspect approved work instructions and historical portfolio records;
  do not infer current goal state from them.
- **If unavailable:** follow the heartbeat stop condition for live portfolio work; keep
  independent evidence gathering separate and do not create duplicate live state.
- **Owner:** `swarm-manager`.
- **Monitoring scenario:** `swarm-manager`.

## Command Center

- **Used for:** outcome board readings and portfolio signals.
- **Needed for:** claims about board coverage or current outcome readings.
- **Fallback:** qualify that lane as unread and continue with independently available
  portfolio evidence.
- **If unavailable:** retain the sensor gap in the snapshot; do not turn missing rows
  into zero outcomes.
- **Owner:** `command-center`.
- **Monitoring scenario:** `command-center`.

## Program Runtime

- **Used for:** bounded portfolio pulse, selected research supervision and capability
  inventory reviews.
- **Needed for:** the compact multi-source views; not the underlying owner records.
- **Fallback:** use the specific direct owner reads documented in this heartbeat and
  retain their statuses separately.
- **If unavailable:** use direct owner paths once; do not repeatedly retry an unchanged
  failed program call.
- **Owner:** `program-runtime`.
- **Monitoring scenario:** `program-runtime`.

## Agent Manager

- **Used for:** managed run evidence and recurring tool-friction analysis.
- **Needed for:** claims about live managed runs or Agent Manager's measured friction
  aggregates.
- **Fallback:** use supported owner records and explicit run identities; preserve live
  state as unknown when it cannot be observed.
- **If unavailable:** continue portfolio work that does not depend on its telemetry;
  do not infer that no run or friction exists.
- **Owner:** `agent-manager`.
- **Monitoring scenario:** `agent-manager`.

## Prompt Manager

- **Used for:** team/member context, skill discovery and governed team configuration.
- **Needed for:** current registered team instructions and skills.
- **Fallback:** read the canonical checked-in team and skill sources; treat them as
  source text, not proof of the active projection.
- **If unavailable:** continue read-only portfolio evidence gathering; do not claim a
  team change was applied or is active.
- **Owner:** `prompt-manager`.
- **Monitoring scenario:** `prompt-manager`.

## Search Hub and Source Ledger

- **Used for:** recall across prior work and durable portfolio continuity.
- **Needed for:** broad discovery and indexed retrieval, not live Swarm Manager state.
- **Fallback:** read the plan of record, dated portfolio records and direct source files.
- **If unavailable:** mark recall coverage partial; no results from an unavailable
  provider do not mean no prior work exists.
- **Owner:** `search-hub`, `source-ledger`.
- **Monitoring scenarios:** `search-hub`, `source-ledger`.

## Notification Hub

- **Used for:** operator asks that need an explicit human decision and delivery receipt.
- **Needed for:** claims that a decision was delivered or answered through that channel.
- **Fallback:** retain the ask in its canonical work item or attention record with its
  delivery state marked pending; never infer delivery from channel health.
- **If unavailable:** keep independent portfolio work moving and report the exact
  delivery gap.
- **Owner:** `notification-hub`.
- **Monitoring scenario:** `notification-hub`.

