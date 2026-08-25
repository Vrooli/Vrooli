# Runbook — Storage Manager

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
| CLI talks to old API | `storage-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

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

## Storage accounting workflow

The operator console reads the same API-backed ledger as the CLI. Start with
`storage-manager inventory` and `storage-manager census`; inspect
`storage-manager history` and `storage-manager infra-health` before making a
policy decision. A census is read-only and persists an immutable snapshot. A
snapshot is only `closed` when measured bytes equal attributed bytes plus the
explicitly unattributed remainder and no unreadable paths remain.

Long-lived API processes can record scheduled observations by setting
`STORAGE_CENSUS_INTERVAL` to a duration of at least one minute (the default is
30 minutes). The first scheduled observation waits one full interval so
readiness never triggers a surprise host scan. Scheduled observations never
apply cleanup or placement migrations.

The static Test Genie `storage` phase remains a fast isolation/persistence
gate. It does not run this host census; use the storage-manager comprehensive
run for product acceptance and live API truthfulness.

## Disk is filling

1. Run `storage-manager storage growth --window 24h`.
2. Inspect the fastest positive-slope owner and its ceiling status.
3. Run `storage-manager cleanup plan --json`.
4. Apply only the approved safe tier, or use the owner approval token named by
   the plan for an owner-delegated provider.
5. Run `storage-manager cleanup audit --json` and record the reclaimed bytes.

## A provider is blocked

1. Run `storage-manager cleanup plan --json`.
2. Read the provider's blocked reason.
3. If the reason is `owner scenario client unavailable`, verify local scenario
   discovery and restart the owner through its lifecycle.
4. If the reason is `owner scenario unreachable`, inspect the owner health endpoint.
5. If the reason is `owner scenario does not implement cleanup`, file or route the owner
   capability gap; do not delete its files by hand.

## A ceiling is not binding

1. Run `storage-manager storage validate <owner>`.
2. Read `CEILING_NOT_BINDING` and the measured bytes.
3. Replace a point-in-time ceiling with a workload-derived `max_bytes` or
   `max_age` value.
4. Keep `regenerable` explicit and run the owner validation again.

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
