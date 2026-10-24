# Runbook — Workflow Health

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

## Selected browser-case validation

Use the dedicated selected operation for a nonempty registered case set:

```bash
workflow-health validate start-selected personal-planner \
  --idempotency-key <stable-request-key> \
  --case-paths bas/cases/calendar/duplicate-submit-once.json,bas/cases/calendar/stale-revision-conflict.json,bas/cases/calendar/uncertain-save-recovery.json
workflow-health validate get <run-id> --json
workflow-health validate wait <run-id> --timeout 15m --json
```

This start persists a provider-owned run and launches execution. The provider
normalizes and freezes case selection; equivalent replay returns the same run,
and a changed target or selection under the same key conflicts. The ordinary
`validate start` refuses explicit selectors. An older provider's `Unimplemented`
response leaves selection unavailable; clients never fall back to the full suite.
Unknown, empty, unsafe or mixed selections refuse before execution, leases or
artifacts. Explicit cases cannot be combined with a workflow-path binding or an
unsupported capability subset. Target/context bindings do not widen the case set.

Get and Wait retain selection after terminal completion and restart. Restart
marks interrupted work as an explicit recovery failure and never replays BAS.
A running target alone does not establish routed isolation: qualify the target
API/UI build, provider-owned SQL/file lease, heartbeat/cleanup and primary-write
counters before original-consumer acceptance. Planner D6 additionally requires
its existing task's lawful owner continuation and independent review; these
commands do not authorize another worker or accept E1, F8, F6 or A2.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `workflow-health status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. |
| Blob files | deferred | deferred | Define if binary/blob domains remain. |

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
