# Runbook — Audio Tools

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
Use scoped Test Genie phases under [TESTING.md](../internal/TESTING.md) for
ordinary iteration; a full suite is reserved for the declared certification scope.

For a slow or failed voice turn, inspect the selected route and safe turn
diagnostic first. Separate discovery, permission, capture, admission, first
displayed partial and final drain. Do not repair a session by switching it to a
paid route, clearing retained audio or increasing a timeout without evidence.
If lifecycle status conflicts with a healthy endpoint, verify the served build
through managed lifecycle before attributing behavior to current source.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `audio-tools status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

## Backup / Restore

Audio Tools stores real credentials, configuration, corpus and experiment state.
Keep encrypted credentials and their key material recoverable under the secret
owner's policy; a database-only copy is not a complete restore. Define and test
coherent backup/restore before production deployment. Do not copy private audio
into general work records or use production wallets for recovery tests.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | deferred | deferred | Define before deployment. |
| Corpus/experiment blobs, browser/server recovery data and secret material | owner-specific procedure required | prove compatibility and access/retention controls | Required for applicable production data; no tested restore is asserted here. |

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
