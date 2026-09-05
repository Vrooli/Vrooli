# Go To Market — Vrooli Bridge

This document records launch strategy, positioning, channels, and
validation experiments for the scenario. Vrooli Bridge is internal
infrastructure first, so its "go to market" is primarily an **internal
capability rollout**, dogfooded on Vrooli's own scenarios before any
external angle is considered.

## Purpose Of This Document

Use this document to answer:

- Who should hear about this scenario?
- Which channels can reach them?
- What claim or offer will be tested?
- What evidence changes the product or monetization plan?

## Audience And Positioning

- Audience (primary): the **Vrooli operator** and the **internal
  deployment pipeline** — deployment-manager and scenario-to-desktop —
  that need a cross-OS validation gate, plus agents that need to run a
  known, safe operation on a specific machine in the owner's fleet.
- Audience (future, external): teams shipping cross-platform desktop or
  CLI applications who want an audited "validated on every OS" gate
  without standing up and hand-operating a machine farm.
- Positioning: **the fleet control plane that makes "production-ready on
  every OS" a one-command, audited operation.** Bridge registers an
  owner's trusted Vrooli machines, keeps them provisioned and
  version-compatible, and runs a constrained, allowlisted set of CLI
  operations on them — collecting durable results back at the control
  plane.
- Main claim: cross-OS deployment validation that previously required
  hand-carrying builds to three machines becomes a single repeatable,
  audited command — safe by construction, with no inbound ports on the
  nodes.
- Proof needed: a real scenario built and tested natively across at
  least two OSes through bridge, producing a trustworthy aggregated
  verdict that deployment-manager can gate a promotion on.

## Channels

| Channel | Hypothesis | Assets Needed | Validation Signal |
|---|---|---|---|
| Internal capability rollout | deployment-manager + scenario-to-desktop adopt bridge's cross-OS gate as the standing validation mechanism. | Cross-OS gate API/CLI (OT-P1-002), integration docs, `../concepts/INTEGRATIONS.md`. | Deployment tier calls bridge for the gate and trusts the verdict enough to gate promotion. |
| Dogfood on Vrooli's own scenarios | Running the gate on real Vrooli scenarios across Ubuntu/macOS/Windows proves the capability and surfaces real failures. | One-touch per-OS bootstrap, registered nodes, durable run results. | Green cross-OS runs on real scenarios; reduced manual per-machine effort. |
| Operator fleet dashboard | The React fleet console lets the owner operate the fleet (pair, revoke, dispatch, watch) without the CLI. | Fleet UI (OT-P1-005), run-history drill-in. | Operator pairs and operates nodes through the UI end to end. |
| External deployment-assurance angle | Future: teams shipping cross-platform apps pay for managed/audited validation. | Mature deployment tier, hosted runner capacity, pricing experiment. | Deferred until the internal gate is proven; see `MONETIZATION.md`. |

## Launch Motion

This is an internal capability rollout, not a market launch.

1. Ship the spine: proto-first node↔control-plane contract, node
   registry + durable identity, dial-out presence/health (OT-P0-001/003).
2. Attach a real node safely: one-touch per-OS bootstrap, mutual auth,
   atomic revocation (OT-P0-002).
3. First real remote operation: allowlisted typed-job dispatch + durable
   remote execution + result collection + audit, dogfooded by running
   `vrooli scenario test X` on a node (OT-P0-004/005/008).
4. Make the fleet self-provisioning and truly cross-OS: privileged
   provisioning tier (sync to revision R) + cross-platform agent service
   install (OT-P0-006/007).
5. Wire the cross-OS deployment gate into deployment-manager and
   scenario-to-desktop (OT-P1-002); dogfood it on real Vrooli scenarios.
6. Only after the internal gate is proven: scope any external
   "deployment assurance / managed fleet" angle and its assets.

## Messaging

| Message | Audience | Evidence | Status |
|---|---|---|---|
| "Production-ready on every OS, in one audited command." | Operator, deployment pipeline | Cross-OS gate aggregates per-OS native build/test verdicts. | planned (post OT-P1-002) |
| "Safe by construction: typed, manifest-declared CLI verbs with per-node scopes — never arbitrary remote shell." | Operator, security-minded adopters | Allowlisted typed-job dispatch (OT-P0-004), audit trail (OT-P0-008). | planned |
| "No inbound ports on your machines — agents dial out, like Tailscale / CI runners." | Operator, network-wary adopters | Dial-out presence channel (OT-P0-003). | planned |
| "One-touch node bootstrap — install once, everything after is remote." | Operator onboarding | One-touch bootstrap + mutual auth (OT-P0-002). | planned |
| "Validate on your own machines for free — always." | All audiences | Self-hosting guardrail (see `MONETIZATION.md`). | settled |

## Validation Experiments

| Experiment | Channel | Threshold | Decision |
|---|---|---|---|
| Dogfood the cross-OS gate on a real scenario | Internal rollout | Native build/test runs green on ≥2 OSes via bridge, aggregated to one verdict. | Promote the gate as the standing mechanism in deployment-manager. |
| Reattach + durability check | Internal rollout | A dispatched job survives client and agent disconnect and is re-attachable by id with a block-once wait. | Confirms the remote-execution model before broad adoption. |
| Effort-reduction measurement | Internal rollout | Manual per-machine hand-carry replaced by one audited command. | Justifies wiring bridge into the deployment tier as default. |
| External deployment-assurance interest | Future external | Identified team willing to pay for managed cross-OS validation. | Only then scope pricing experiment (`MONETIZATION.md`). |

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — role, packaging, pricing hypothesis
- [`../../PRD.md`](../../PRD.md) — product outcomes (cross-OS gate, fleet control plane)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane, node-agent, dial-out model
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — deployment-manager / scenario-to-desktop consumers
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — allowlisted remote execution posture
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — validation signals and telemetry
