# Runbook — Ecosystem Manager

This document records operator procedures for running, diagnosing,
recovering, and maintaining Ecosystem Manager — the autonomous
generation/improvement control plane for Vrooli.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore durable state?
- When and how do I escalate?

Most operator work here is **control-plane operation**: starting/stopping
the queue processor, setting maintenance mode, clearing stuck agent
processes, and adjusting concurrency/auto-requeue settings. Auto-steer
runs improvement loops autonomously via agent-manager; operators steer,
they do not hand-run iterations.

## Start / Stop / Status

Use lifecycle-managed commands. Never direct-exec the binary.

```bash
vrooli scenario start ecosystem-manager
vrooli scenario status ecosystem-manager
vrooli scenario restart ecosystem-manager
vrooli scenario stop ecosystem-manager
```

Or from the scenario directory via the Makefile:

```bash
make start
make status
make logs
make stop
make test
```

Verify after start:

- Dashboard: `http://localhost:30500`
- Health: `curl -s http://localhost:30500/health` (reports API + DB
  reachability)
- Queue: `curl -s http://localhost:30500/api/queue/status`

The lifecycle owns process naming, ports, health checks, and logs.
Ecosystem Manager also requires `agent-manager` for task execution.
`scenario-completeness-scoring` is optional for fast cached status reads
and test-genie report supplements.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `vrooli scenario status`, `make logs` | `vrooli scenario restart ecosystem-manager`; confirm `vrooli-postgres-main` is up | Record recurring failures in `../internal/PROBLEMS.md`. |
| `/health` reports DB unreachable | Postgres container `vrooli-postgres-main`, db `vrooli_ecosystem_manager` | Start Postgres; re-run `make setup` to re-apply idempotent schema | Check container logs; escalate if data dir is corrupt. |
| Tasks never execute / stuck queued | `GET /api/queue/status`; is the queue processor running? | `POST /api/queue/start`; if a run is wedged, `POST /api/queue/processes/terminate` | Verify agent-manager is healthy — it executes the runs. |
| Rate-limit backoff stalls queue | `GET /api/queue/status` shows backoff | `POST /api/queue/reset-rate-limit` | If recurring, lower concurrency in Settings. |
| Need to pause all work safely | current state in UI | `POST /api/maintenance/state` to enter maintenance mode | — |
| Cached completeness view missing/stale | scenario-completeness-scoring health; `scenario-completeness-scoring score get <scenario>` | Start/restart scenario-completeness-scoring or refresh the target's test-genie run | EM queue execution does not depend on this reader. |
| Stuck/zombie agent process | `GET /api/processes/running` | `POST /api/queue/processes/terminate` | Inspect agent-manager run logs. |

## Backup / Restore

Ecosystem Manager has two durable stores: the PostgreSQL database and the
on-disk profile/queue files.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Postgres db `vrooli_ecosystem_manager` (tables `task_executions`, `operation_metrics`, `profile_executions`, `profile_execution_state`, `steering_queue_state`, `execution_feedback_entries`) | `docker exec vrooli-postgres-main pg_dump -U vrooli vrooli_ecosystem_manager > backup.sql` | `docker exec -i vrooli-postgres-main psql -U vrooli -d vrooli_ecosystem_manager < backup.sql` | active |
| `profiles/` dir (auto-steer profile JSON + `metadata.json`) | Copy the directory while the queue processor is stopped | Restore the directory in place | active |
| `queue/<status>/` task YAML | Copy the `queue/` tree | Restore the `queue/` tree | active |

Schema re-application is safe (idempotent `IF NOT EXISTS`); restore data
only when rolling back a destructive change.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Run tests | before handoff | `make test` |
| Inspect system audit log | as needed | UI log view, or `GET /api/logs` |
| Enter/exit maintenance mode | during deploys or incidents | `POST /api/maintenance/state` |
| Stop/start queue processor | as needed | `POST /api/queue/stop` / `POST /api/queue/start` |
| Reset rate-limit backoff | when throttled | `POST /api/queue/reset-rate-limit` |
| Adjust concurrency / auto-requeue | as load dictates | Settings page in the UI |
| Inspect logs | as needed | `make logs` (date-stamped files in the scenario `logs/` dir) |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) and append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).
File defects outside scope via `report-bug`. Because Ecosystem Manager
depends on `agent-manager` for execution, triage whether a failure
originates there before escalating here. Use
`scenario-completeness-scoring` separately when the issue is the cached
status reader.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
