# Performance — Vrooli Bridge

This document records performance budgets, current measurements, known
constraints, and regression procedures for the fleet control plane and
its node-agents. The scenario is unbuilt, so the budgets below are
forward-looking targets and there are no real measurements yet.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

Budgets separate the **control-plane overhead** bridge owns (which must
stay small) from the **node work** bridge orchestrates (which is
dominated by native build/test and provisioning and is intrinsically
slow). Bridge is responsible for keeping its own overhead negligible
against the work it dispatches.

| Surface | Budget (target) | Rationale | Status |
|---|---|---|---|
| Control-plane API latency (registry / status reads, Connect-RPC) | sub-100ms server-side for metadata reads | SQLite-backed metadata; reads should never be a bottleneck. | target (unbuilt) |
| Dial-out presence detection latency | online→offline transition detected within a small multiple of the heartbeat cadence | Dispatch must only target nodes that are actually reachable; detection lag must be bounded. | target (unbuilt) |
| Job dispatch overhead | negligible vs. job runtime — control-plane handoff to the node's durable run measured in milliseconds, not the wall-clock of the job | Bridge orchestrates; it must add near-zero overhead on top of the node's own execution. | target (unbuilt) |
| Provisioning (sync-to-revision R) duration | bounded primarily by git fetch + idempotent `vrooli setup`; re-running on an already-current node is fast (idempotent no-op cost) | Provisioning is heavy by nature; the budget is "no wasted work," not a fixed wall-clock. | target (unbuilt) |
| Cross-OS gate wall-clock | dominated by the **slowest node's** native build/test; bridge's aggregation overhead negligible | A gate is as slow as its slowest OS; bridge must not add meaningful time on top. | target (unbuilt) |
| UI build | 5-10 minutes accepted for current Vite module graph | Inherited platform constraint, not bridge-specific. | inherited |
| API/UI health | responsive under lifecycle health timeout | `/health` checks via lifecycle. | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet — the scenario is unbuilt (no domains implemented). | n/a | n/a | 2026-06-18 |

There are no real performance numbers yet. The control plane, node-agent,
dispatch path, and provisioning tier do not exist beyond documentation,
so every value above is a target to validate once the corresponding
domain is implemented. This table is populated from real runs (see the
Regression Procedure) as each piece lands.

## Known Constraints

- **Native build/test time per OS dominates the cross-OS gate.** A gate's
  wall-clock is set by the slowest node's native build and test run, not
  by anything bridge does. Bridge can parallelize across nodes but cannot
  make any single OS's build faster.
- **Provisioning is heavy.** Bringing a node to revision R is a git
  fetch plus a full idempotent `vrooli setup`; first-time provisioning on
  a fresh machine is the worst case. Idempotency makes re-provisioning a
  current node cheap, but the first sync is intrinsically expensive.
- **Dial-out heartbeat cadence vs. presence-detection latency is a
  tradeoff.** A faster heartbeat detects offline nodes sooner but costs
  more idle traffic across the fleet and the tunnel; a slower heartbeat
  is quieter but lengthens the window where a dead node still looks
  online. The cadence must be tuned against acceptable detection latency.
- **Per-node serialization caps throughput.** Each node runs at most one
  job at a time (or small bounded concurrency, mirroring test-genie's
  one-run-per-scenario discipline), so fleet throughput scales with node
  count, not with per-node parallelism.
- **Off-LAN reach adds tunnel latency.** Nodes reached through
  tunnel-manager pay the tunnel's round-trip on the dial-out channel
  versus same-LAN nodes.
- **Result/artifact streaming back to the control plane** competes with
  the node's own work and the link bandwidth; large artifacts move
  through device-sync-hub rather than the control-plane channel to keep
  the latter responsive.

## Regression Procedure

This is the procedure to follow once the relevant domains are built; it
cannot produce numbers before then.

1. Run `make test` (and the test-genie measures phase where available) to
   capture timing for control-plane API calls, dispatch handoff, and the
   health surfaces.
2. Record **per-job timing in the durable run/audit records** — dispatch
   handoff time, node-side execution time, and result-collection time —
   so control-plane overhead can be separated from node work in analysis.
3. For provisioning, measure sync-to-revision duration on (a) a fresh
   node and (b) an already-current node, to confirm the idempotent
   no-op path stays cheap.
4. For the cross-OS gate, capture per-OS wall-clock and confirm the
   aggregate tracks the slowest node, with bridge aggregation overhead
   negligible on top.
5. For dial-out presence, measure the online→offline detection window
   against the configured heartbeat cadence and record the tradeoff.
6. Capture API/UI command timing relevant to the change; for UI
   interaction regressions use `ui/perf/README.md` and its capture
   template.
7. Record persistent findings in this document (accepted constraints) or
   [`PROBLEMS.md`](PROBLEMS.md) (unresolved debt), and update the Current
   Measurements table with the dated values.

## Cross-References

- [`../../PRD.md`](../../PRD.md) — operational targets and tech direction
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — control plane, node-agent, dial-out channel
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — dispatch, provisioning, and gate flows
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations (incl. measures phase)
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
