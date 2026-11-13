# Runbook — Extraction Range

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use lifecycle-managed commands from the scenario directory:

```bash
make setup
make start
make status
make logs
make stop
make test
```

Do not start API/UI binaries directly. The lifecycle owns process
naming, ports, health checks, and logs.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `extraction-range status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Tool cannot score a level | `GET /levels/{id}`, inspect the task fields, then submit a bounded record array to `/score` | Confirm the payload uses `id`, `name`, `category`, `value`, and `active`; see `docs/concepts/FLOWS.md` | Unknown level/tool errors return 400; a valid result appears in `/scoreboard`. |
| Scoreboard is empty after restart | Scenario process was restarted or scores were never submitted | Confirm the expected tool has score rows using `GET /scoreboard?tool=<tool>`; rerun the owned qualification only if records are absent | Best score rows persist in the routed SQLite database; clear/reset only through the scenario's supported data lifecycle. |

## Backup / Restore

The range corpus is versioned in scenario source. The best-scoreboard rows
are persisted in the routed SQLite database and follow the scenario's normal
database backup and restore path.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | existing local storage backup path | same scenario's supported storage restore workflow | Holds synthetic range score rows and no client data. |
| Level JSON fixtures | version-controlled scenario source | restore/update through the repository | Do not edit generated protobuf output; level definitions are authored in the rangeengine levels directory. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
