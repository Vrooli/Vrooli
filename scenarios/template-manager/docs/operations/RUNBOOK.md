# Runbook — Template Manager

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
| CLI talks to old API | `template-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Deep-validate monitor failing | `template-manager monitor status --json`, API logs, latest `template-manager runs list --template react-vite --json` | Repair the failing active template, then restart through `make restart`; use `TEMPLATE_MANAGER_MONITOR_RUN_ON_START=true` only when an immediate scheduler run is required | The monitor validates active scenario templates only; quarantined and retired templates remain available for manual diagnosis without making the active-monitor result red. Record recurring active-template failures in `../internal/PROBLEMS.md`. |

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
| Inspect deep-validate monitor | daily / on incident | `template-manager monitor status`; interval is controlled by `TEMPLATE_MANAGER_MONITOR_INTERVAL` as seconds or Go duration text. |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |

## Existing Autoheal Installs

New installs monitor `template-manager` as a critical scenario by default. Existing
`~/.vrooli-autoheal/config.json` files can adopt the same setting by adding:

```json
{
  "monitoring": {
    "scenarios": {
      "template-manager": { "critical": true }
    }
  }
}
```

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
