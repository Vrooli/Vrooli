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
| `GET /health` | health | API | API + DB (`vrooli_ecosystem_manager`) reachability | healthy |
| `GET /api/queue/status` | operational | queue processor | Processor running, depth, backoff state | processor running, no stuck backoff |
| Task throughput & success rate | product | `operation_metrics` (trigger-aggregated daily) | Work completed per operation type | success rate trending up per operation |
| PRD completion improvement | product | `task_executions.prd_completion_before/after` | Did an improvement run raise PRD score? | after ≥ before |
| Auto-steer convergence | product | `profile_executions.phase_breakdown` + `user_rating` feedback | Loops converging, not thrashing | phases progressing, ratings positive |
| Live task/queue updates | operational | WebSocket `/ws` | Real-time UI state of tasks/queue | pushes events while processor runs |

## Logs

| Log | Source | How To Read | Notes |
|---|---|---|---|
| System audit log | `api/pkg/systemlog` | UI log view, or `GET /api/logs` | Structured audit of system actions. |
| Scenario logs | lifecycle-managed processes | `make logs` | Date-stamped files in the scenario `logs/` dir. |
| Per-task execution history | Postgres `task_executions` | UI task views / DB query | Includes status, timings, PRD before/after. |
| Agent run output | agent-manager | agent-manager logs / `logs/task-runs/` | Runs execute in agent-manager; correlate from there. |

## Metrics

All metrics today are stored in Postgres and surfaced via the API/UI;
there is **no external metrics backend**.

| Metric | Status | Source | Notes |
|---|---|---|---|
| Operation throughput / success rate | active | `operation_metrics` | Daily trigger-aggregated by operation type. |
| PRD completion delta | active | `task_executions.prd_completion_before/after` | Improvement-loop effectiveness. |
| Auto-steer convergence | active | `profile_executions.phase_breakdown`, `execution_feedback_entries` | Phase progression + user ratings. |
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
| No per-iteration decision trace | Cannot see *why* a given skill was selected during an auto-steer loop, nor detect thrashing early. | Address as Ecosystem Manager is reframed as a closed-loop controller (see [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md)). |

Future observability should surface the per-iteration decision trace and
auto-steer convergence/thrashing signals, aligned with the closed-loop
controller direction.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — operational procedures and incident response
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — readiness gates and dependencies
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system structure
- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller direction (future)
