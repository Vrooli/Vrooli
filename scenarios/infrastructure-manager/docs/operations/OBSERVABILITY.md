# Observability — Infrastructure Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for the scenario.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us users are getting value?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain before deployment or monetization?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `/health` status | health | API | API and dependency reachability | healthy for local development |
| UI health endpoint | health | UI server | UI bundle/server reachability | responds during lifecycle health check |
| test-genie result | validation | `make test` | scenario correctness evidence | all required phases pass |

## Logs

| Log | Source | How To Read | Details |
|---|---|---|---|
| API logs | lifecycle-managed API process | `make logs` | Request logging uses deterministic clock seam in tests. |
| UI logs | lifecycle-managed UI server | `make logs` | Production bundle server logs only. |

## Metrics

| Metric | Status | Details |
|---|---|---|
| Requirement coverage | active | Tracked through requirements and test-genie coverage artifacts. |
| Source read success rate | planned | Per-dependency: reads that returned versus timed out or refused. The board's own reliability, distinct from the platform's. |
| Setpoint parse outcome | planned | Parsed cleanly / parsed with integrity findings / failed. A parse failure is the scenario's loudest possible signal. |
| Trust check coverage | planned | Readings with a computable verdict, over total readings. Directly reports the board's own blindness. |
| Open-loop target count | planned | Targets with no working sensor, with gap ages. This is simultaneously a product output (`OT-P0-005`) and a self-observability metric. |
| Board read volume | planned | Whether the team actually reads the board — the adoption signal named in `../business/GO-TO-MARKET.md`. |
| Product activation | not-applicable | Internal capability; no activation funnel. |

**This scenario is unusual in that its product output and its self-observability
overlap.** A board that reports its own blindness (`OT-P0-005`, trust coverage)
is doing observability work as a feature. The distinction worth preserving: the
metrics above describe *the instrument*, while everything on the board describes
*the platform*. Conflating them would let an instrument outage read as a healthy
platform.

## Alerts / Health

The generated scenario has lifecycle health checks for API and UI.

Beyond those, the operating principle is that **this scenario does not alert.**
It is the outermost, slowest loop — heartbeats to days — and it holds no
authority to act. Its findings reach humans through the team's heartbeat and
the morning vision walk, not through a paging channel. Anything that genuinely
needs a seconds-clock response belongs to the deferred watchdog tier
(see `../concepts/DOMAINS.md` § Deferred Domains), which is a separate decision
with its own authorization.

The one exception worth building: **this instrument should be watched by
something outside itself.** `TARGET_MODEL.md` requires that an instrument not
be its own only sensor, and names `meta-optimization-manager` as the natural
peer — it already supervises capability-owner reachability. That edge should be
declared rather than assumed.

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No self-observability implemented | The board could be degraded — stale reads, silent source failures — without saying so. Highest-priority gap once code exists. | First vertical slice. |
| Two of four trust rules uncomputable | Trust coverage will under-report until ghost and shelved verdicts are available. Degrades safely (conservative `UNTRUSTED`) but widens declared blindness. | Roadmap Gap 10. |
| No external watcher declared | Deviation `D9` — an unwatched instrument. | Declare the `meta-optimization-manager` edge when the instrument goes live. |
| Product usage telemetry | Not-applicable for monetization; **is** applicable for adoption (is the team reading the board?). | Board read volume, above. |
| Cost telemetry | Not-applicable — internal, local, no hosted unit economics. | Only if this ever becomes a managed deployment. |

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — business validation signals
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — performance measurements
