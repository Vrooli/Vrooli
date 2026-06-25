# Observability — Ecosystem Manager

This document records logs, metrics, telemetry, health checks, and
business/product signals for Ecosystem Manager — the autonomous
generation/improvement control plane for Vrooli.

## Purpose Of This Document

Use this document to answer:

- What signals tell us the scenario is healthy?
- What signals tell us the system is producing value (generation /
  improvement throughput, PRD gains, auto-steer convergence)?
- Which logs or metrics should an operator inspect first?
- What telemetry gaps remain?

## Signals

| Signal | Type | Source | Purpose | Threshold |
|---|---|---|---|---|
| `GET /health` | health | API | API + embedded SQLite handle reachability | healthy |
| `GET /api/queue/status` | operational | queue processor | Processor running, depth, backoff state | processor running, no stuck backoff |
| Task throughput & success rate | product | `profile_executions` (run records, aggregated on read) | Work completed per operation type | success rate trending up per operation |
| PRD completion | product | scenario-completeness-scoring `GetScore` (read live per iteration; not persisted) | Operational-target completion the controller steers toward | operational-targets % rising across iterations |
| Auto-steer convergence | product | `profile_executions.phase_breakdown` + `decision_trace` | Loops converging, not thrashing | weighted score improving, findings closing |
| Live task/queue updates | operational | WebSocket `/ws` | Real-time UI state of tasks/queue | pushes events while processor runs |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| System audit log | `api/pkg/systemlog` | UI log view, or `GET /api/logs` | Structured audit of system actions. |
| Scenario logs | lifecycle-managed processes | `make logs` | Date-stamped files in the scenario `logs/` dir. |
| Per-task execution history | SQLite `profile_executions` | UI task views / DB query | Includes timings (`total_duration_ms`), completeness score, iteration counts, and the decision trace. |
| Agent run output | agent-manager | agent-manager logs / `logs/task-runs/` | Runs execute in agent-manager; correlate from there. |

## Metrics

All metrics today are stored in the embedded SQLite database and surfaced
via the API/UI; there is **no external metrics backend**.

| Metric | Status | Source | Notes |
|---|---|---|---|
| Operation throughput / success rate | active | `profile_executions` | Aggregated on read from run records. |
| PRD completion | active | scenario-completeness-scoring `GetScore` | Read live per iteration for termination; not persisted as a per-run delta. |
| Auto-steer convergence | active | `profile_executions.phase_breakdown`, `decision_trace` | Per-iteration breakdown + weighted-score gradient. |
| Queue depth / backoff | active | `GET /api/queue/status`, `steering_queue_state` | Live operational state. |

## Alerts / Health

There is **no automated alerting today**. Operators monitor manually:

- `GET /health` for API + DB reachability.
- `GET /api/queue/status` for a running processor, queue depth, and
  rate-limit backoff.
- The UI dashboard at `http://localhost:30500` for live task/queue state
  pushed over WebSocket `/ws`.

Take action via the runbook controls (start/stop processor, reset
rate-limit, terminate stuck processes, maintenance mode).

## Telemetry Gaps

| Gap | Impact | Revisit Trigger |
|---|---|---|
| No external metrics/tracing export (Prometheus/OTel) | No central dashboards or cross-scenario correlation; all metrics are DB-local. | Add when fleet-wide observability is prioritized. |
| File-based logs only | No log aggregation/search beyond `GET /api/logs` and on-disk files. | Add when centralized logging lands. |
| Manual alerting | Failures (stuck queue, unhealthy DB) require human watching. | Add when an alerting substrate exists. |
| No convergence alerting | The per-iteration decision trace exists and is surfaced (API/CLI/UI), but nothing automatically flags a non-converging or diminishing-returns run. | Add when an alerting substrate exists. |

Future observability should add automated alerting on the auto-steer
convergence signals (weighted-score gradient, diminishing returns) that the
decision trace already exposes.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures and incident response
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates and dependencies
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system structure
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller direction (future)
