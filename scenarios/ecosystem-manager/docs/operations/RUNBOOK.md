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
| Scenario does not start | `vrooli scenario status`, `make logs` | `vrooli scenario restart ecosystem-manager` | Record recurring failures in `../internal/PROBLEMS.md`. |
| `/health` reports DB unreachable | The SQLite file `<data-root>/vrooli/<namespace>/ecosystem-manager.db` (resolve with `vrooli recovery namespace --scenario ecosystem-manager --json`) | Restart the API to re-open the DB and re-apply schemas; ensure the storage data dir is writable | Escalate if the SQLite file is corrupt — restore from data-backup-manager. |
| Tasks never execute / stuck queued | `GET /api/queue/status`; is the queue processor running? | `POST /api/queue/start`; if a run is wedged, `POST /api/queue/processes/terminate` | Verify agent-manager is healthy — it executes the runs. |
| Rate-limit backoff stalls queue | `GET /api/queue/status` shows backoff | `POST /api/queue/reset-rate-limit` | If recurring, lower concurrency in Settings. |
| Need to pause all work safely | current state in UI | `POST /api/maintenance/state` to enter maintenance mode | — |
| Cached completeness view missing/stale | scenario-completeness-scoring health; `scenario-completeness-scoring score get <scenario>` | Start/restart scenario-completeness-scoring or refresh the target's test-genie run | EM queue execution does not depend on this reader. |
| Stuck/zombie agent process | `GET /api/processes/running` | `POST /api/queue/processes/terminate` | Inspect agent-manager run logs. |

## Backup / Restore

All of Ecosystem Manager's runtime state — the SQLite database **and** the
task queue YAML — lives under the resolved storage root
(`<data-root>/vrooli/<namespace>/`, where `<data-root>` and `<namespace>`
come from `vrooli recovery namespace --scenario ecosystem-manager --json`).
That single root is the backup unit, and it is covered by
**data-backup-manager**; there are no bespoke scenario-folder copy steps.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Storage data root (SQLite DB `ecosystem-manager.db` + `queue/<status>/*.yaml`) | Covered by data-backup-manager. Verify with `data-backup-manager coverage report`; register the scenario's targets with `data-backup-manager safety register-targets --scenario ecosystem-manager` if not already visible. | `data-backup-manager` restore of the registered targets (stop the API first) | active |
| `profiles/` dir (auto-steer profile JSON + `metadata.json`) | Git-tracked source asset in the scenario tree — backed up by version control, not the storage root. | `git checkout` / restore the directory | active |

The SQLite schema is re-applied idempotently on boot (`CREATE TABLE IF NOT
EXISTS`); restore data only when rolling back a destructive change. The task
queue is no longer source-controlled — interrupted/in-flight tasks are
recovered from the storage root, not from the repo.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Run tests | before handoff | `make test` |
| Inspect system audit log | as needed | UI log view, or `GET /api/logs` |
| Enter/exit maintenance mode | during deploys or incidents | `POST /api/maintenance/state` |
| Stop/start queue processor | as needed | `POST /api/queue/stop` / `POST /api/queue/start` |
| Reset rate-limit backoff | when throttled | `POST /api/queue/reset-rate-limit` |
| Adjust concurrency / auto-requeue | as load dictates | Settings page in the UI |
| Inspect logs | as needed | `make logs`, or the date-stamped audit files under the resolved `ClassLogs` dir (`vrooli recovery namespace --scenario ecosystem-manager --json`) |

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
