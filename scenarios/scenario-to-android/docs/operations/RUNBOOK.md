# Runbook — Scenario to Android

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
| CLI talks to old API | `scenario-to-android status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

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

## Android Ramp Operations

The ramp owns artifact orchestration and evidence review; device verbs remain
owned by `device-control`, and web-content automation remains owned by BAS.
Use the CLI or equivalent API routes:

```bash
ANDROID_SOURCE_REF=/absolute/path/to/scenario/ui/dist \
  scenario-to-android android build
scenario-to-android android targets
scenario-to-android android matrix-catalog
ANDROID_ARTIFACT_DIGEST=sha256:<apk-sha256> scenario-to-android android matrix-create
ANDROID_MATRIX_RUN_ID=<run-id> scenario-to-android android matrix-start
ANDROID_MATRIX_RUN_ID=<run-id> scenario-to-android android matrix-wait
```

`android build` produces a debug APK and AAB without an operator signing
identity. A matrix cell is pass-eligible only when device-control proves the
target, BAS completes the registered flow, and the redacted recording has
bounded offsets and a checksum. Unavailable targets must remain unavailable;
do not substitute a host-only or idle recording.

For review tooling, open the UI target matrix, run review, and readiness pages
at `/targets`, `/runs`, and `/readiness`. When deployment-manager identity is
configured (`DEPLOYMENT_MANAGER_URL`, `DEPLOYMENT_MANAGER_PROFILE_ID`, and
`DEPLOYMENT_MANAGER_GIT_COMMIT`), completed matrix gates report reference-only
verdicts to its evidence service.

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
