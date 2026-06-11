# Performance — Ecosystem Manager

This document records performance budgets, current measurements, known
constraints, and regression procedures.

Ecosystem Manager is **not a latency-sensitive UX**. Its dominant cost is
the **tokens and wall-clock time of agent iterations**: each auto-steer
iteration runs a full agent (via `agent-manager`) plus metrics collection
that may build and test the target scenario — often many minutes per
iteration. "Performance" here means **controlling loop cost**, not
shaving milliseconds off a request handler.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| Iterations per phase | Profile `max_iterations` per phase `[CODE: profiles/*/profile.json]` | profile config | active |
| Per-task iteration ceiling | Bounded by the sum of phase `max_iterations` + `stop_conditions` on metrics | profile config | active |
| Concurrency | Queue processor `max_concurrent` job slots `[CODE: api/pkg/queue/]` | `status.max_concurrent` | active |
| Rate-limit backoff | Backoff on model rate limits during agent runs | runner behavior (agent-manager) | active |
| API health | responsive under lifecycle health timeout (10s) | `/health` check | active |
| UI health | responsive under lifecycle health timeout (5s) | `/health` check | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Iteration durations | Recorded per run (`total_duration_ms`) and per iteration | SQLite `profile_executions` / `decision_trace` `[CODE: api/pkg/autosteer/schema.sql]` | 2026-05-30 |
| Auto-steer iteration counts | Recorded per phase/task | execution state (SQLite `profile_execution_state` / `profile_executions.total_iterations`) | 2026-05-30 |
| Absolute per-iteration wall-clock | Not yet captured as a fixed number; dominated by target build/test time | — | 2026-05-30 |

## Known Constraints

- **Metrics collection re-runs the target's build and tests.** This
  re-audit cost is the practical loop bottleneck — each iteration pays
  the full build/test price of the scenario being improved, which can be
  several minutes (the Vite UI build alone is 5–10 minutes).
- **Tokens, not CPU, are the scarce resource.** Iteration count is the
  primary cost lever; tightening `stop_conditions` saves more than any
  code optimization.
- **Wall-clock is variable** and target-dependent; a single budget number
  is meaningless. Budgets are expressed as iteration ceilings instead.

## Regression Procedure

1. Watch `profile_executions.total_duration_ms` (and per-iteration
   `decision_trace`) in SQLite for drift in run/iteration timing.
2. Watch auto-steer iteration counts per phase/task — an unexpected rise
   means a `stop_condition` regressed or a target's metrics stopped
   converging.
3. If iteration cost spikes, check whether the target's build/test time
   grew (re-audit cost) before suspecting ecosystem-manager itself.
4. Record persistent findings here (accepted constraints) or in
   [`PROBLEMS.md`](PROBLEMS.md) (unresolved debt).

## Cross-References

- [`../concepts/CONTROL-MODEL.md`](../concepts/CONTROL-MODEL.md) — closed-loop controller; objective functions drive iteration cost
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — queue, profiles, and the agent-manager loop
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
