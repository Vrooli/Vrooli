# Observability — Vrooli Bridge

This document records the signals, logs, metrics, health checks, and
telemetry gaps for the bridge control plane and the fleet of node-agents
it manages.

Bridge observability is inherently **fleet-shaped**: the control plane
must know which nodes are present and healthy, whether dispatched jobs
succeed, whether provisioning brings nodes to the right revision, and —
the headline outcome — whether the cross-OS validation gate passes on
each target OS. No product code exists yet; the signals below describe
the intended telemetry and are honest about what is not yet built.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the control plane and the fleet are healthy?
- What signals tell us the owner is getting value (jobs running, gates passing)?
- Which logs or metrics should an operator inspect first during an incident?
- What telemetry gaps remain before this scenario is usable in production?

## Signals

| Signal | Type | Source | Purpose | Status |
|---|---|---|---|---|
| Control-plane `/health` | health | API | API + SQLite reachability | planned |
| UI health endpoint | health | UI server | Dashboard reachability | planned |
| Node presence / online-offline | health | node-agent dial-out channel | Is each node connected and reachable | planned (OT-P0-003) |
| Node self-reported readiness | health | node-agent | Toolchain present, disk headroom, container runtime up — so dispatch only targets capable nodes | planned (OT-P0-003) |
| Job dispatch → outcome | validation | dispatch + durable run | Did the typed job run and what was its exit status | planned (OT-P0-004/005) |
| Provisioning success + resulting revision | validation | provisioning tier | Did the node reach target revision R | planned (OT-P0-006) |
| Cross-OS gate verdict (per OS) | validation | deployment gate | "Production-ready on Ubuntu + macOS + Windows?" pass-rate per OS | planned (OT-P1-002) |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| Control-plane API logs | lifecycle-managed API process | `make logs` | Connect-RPC + SSE dial-out edge request logging. |
| Control-plane UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |
| Per-node job logs | node-agent, streamed back | control plane (drill into a node's run) | Job stdout/exit status/result artifacts stream from the node to the control plane as a server-owned durable run; re-attachable by id, survives disconnect. |
| Provisioning logs | provisioning tier, streamed back | control plane (node version history) | `vrooli setup` output and the resulting revision captured at the control plane. |
| Audit trail | workspace-sandbox | audit query | Append-only record of every dispatch and provisioning op — who, which node, what verb/args, outcome — for after-the-fact traceability (OT-P0-008). |

Per-node logs are **streamed back** to the control plane rather than left
on the node, so an operator inspects fleet activity from one place.
Streaming and durable-run wiring are not yet implemented.

## Metrics

| Metric | Status | Details |
|---|---|---|
| Jobs run | planned | Count of dispatched typed jobs, by node / scenario / verb. |
| Job fail rate | planned | Failed vs total jobs, per node and per OS. |
| Provision duration | planned | Wall-clock for sync-to-revision-R, per OS (Windows is expected to be the costly one). |
| Fleet version skew | planned | Distribution of node Vrooli revisions vs the pinned target R; count of nodes flagged "needs update." |
| Gate verdicts | planned | Cross-OS validation pass/fail per OS over time — the product's headline outcome metric (OT-P1-002). |
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |

## Alerts / Health

The control plane will expose lifecycle health checks for its own API and
UI. Fleet-level alerts are the substantive observability surface and are
planned, not yet built:

- **Node offline** — a node's dial-out channel has dropped beyond a grace
  window; dispatch should already exclude it, but the operator is alerted.
- **Repeated provision failure** — `vrooli setup` to revision R fails
  repeatedly on a node (auto-rollback fires); flag for investigation, as
  this often indicates an OS-specific toolchain problem.
- **Gate regression** — the cross-OS validation gate that passed on an OS
  now fails on that OS; this is the signal that "production-ready" has
  regressed for a target platform.
- **Version drift** — fleet skew exceeds tolerance or nodes are flagged
  "needs update" against the pinned revision.

Add deployment-specific alert routing only when the deployment target and
operator expectations are known.

## Telemetry Gaps

Bridge is unbuilt; nearly all telemetry below is planned, not present.
This section is honest about that.

| Gap | Impact | Revisit Trigger |
|---|---|---|
| Node presence + health telemetry | Cannot tell which nodes are usable; dispatch cannot target safely. | Implement with the dial-out channel (OT-P0-003). |
| Job + provisioning outcome metrics | Cannot measure fleet reliability or surface fail-rate/skew. | Implement with dispatch + provisioning tiers (OT-P0-004/006). |
| Cross-OS gate verdict history | Cannot trend per-OS production-readiness or detect gate regressions. | Implement with the deployment gate (OT-P1-002). |
| Streamed per-node log retention | Cannot review historical job/provision activity beyond a single session. | Define a retention policy when result collection lands (OT-P0-005). |
| Product usage telemetry | Cannot validate adoption. | Add before any monetization review (see `../business/MONETIZATION.md`). |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures and incident response
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and readiness gates
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane / node-agent / dial-out design
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — audit trail and allowlist posture
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements and budgets
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../../PRD.md`](../../PRD.md) — operational targets behind these signals
