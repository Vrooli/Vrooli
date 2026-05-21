# Runbook — Architecture Cartographer

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
| CLI talks to old API | `architecture-cartographer status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| `arch-cart graph extract` fails with `IntegrationError: scenario_unreachable` | Is `go-code-graph` or `typescript-code-graph` running? `vrooli scenario status go-code-graph` | `vrooli scenario start <dep-scenario>` | If repeatedly unreachable, check `INTEGRATIONS.md` and the dependency scenario's own runbook. |
| `arch-cart conflict resolve` refuses with `build broken since baseline` | Read the surfaced build output; was the resolution attempt itself the cause, or was build broken before? | Fix the build error; if pre-existing, use `arch-cart migrate baseline-update` to accept the new baseline. Last resort: `--force --note "<reason>"`. | `--force` usage is logged in analytics; review periodically. |
| `arch-cart apply <domain>` refuses on `main` / `master` branch | Branch protection guard is enabled (see [`../internal/SECURITY.md`](../internal/SECURITY.md)). | Switch to a feature branch: `git checkout -b migrate/<domain>`. If you genuinely intend to commit to `main`, pass `--allow-protected-branch --note "<reason>"`. | Any `--allow-protected-branch` usage should be rare and reviewed. |
| Auto-placement is consistently picking the wrong domain for a class of file | Run `arch-cart signals explain <chunk>` to see per-signal scores. | Identify which signal is misleading; adjust its weight in the target scenario's manifest or disable it. Capture the case in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). | After ≥5 instances, consider whether the manifest's glossary or domain definitions need refinement. |
| `arch-cart calibrate` proposes alarming weight changes | Read the override history justifying the proposal (`arch-cart history --filter overrides`). | Accept selectively or reject. Weight changes never auto-apply. | If proposals are consistently low quality, file in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| Migration appears stuck mid-flight | `arch-cart status <scenario>` shows what state. | `arch-cart migrate resume <scenario>` to continue, or `arch-cart migrate abandon <scenario> --note "<reason>"` to walk away cleanly. | Abandoned migrations remain in history for retrospective analysis. |

## Backup / Restore

Cartographer state is local SQLite. Backups matter because analytics
history is the substrate for calibration over time.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | `cp $SQLITE_PATH $SQLITE_PATH.backup-$(date +%Y%m%d)` before destructive ops | `cp $SQLITE_PATH.backup-<date> $SQLITE_PATH` (with scenario stopped) | Manual in v1; automated rotation deferred to P1. |
| Graph snapshots | Subset of SQLite backup. | Same restore path. | Same. |
| Analytics event log | Subset of SQLite backup; critical for calibration history. | Same restore path. | Recommend backing up monthly until automated retention lands. |
| Manifest source | Owned by target scenarios, not cartographer. | n/a — target scenario's git history. | n/a |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Validate own architecture | weekly (manual until CI gate lands) | `arch-cart conflicts list architecture-cartographer` |
| Calibration review | monthly | `arch-cart calibrate --dry-run` and review proposed weight changes against override history |
| Backup analytics DB | monthly | `cp $SQLITE_PATH $SQLITE_PATH.backup-$(date +%Y%m)` |
| Force-note audit | quarterly | `arch-cart history --filter force-notes` — review every `--force --note` justification for legitimacy |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
