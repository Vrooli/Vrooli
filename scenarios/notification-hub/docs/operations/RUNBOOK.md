# Runbook — Notification Hub

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
| CLI talks to old API | `notification-hub status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |

### Delivery incidents

The first question in every one of these is the same: **look at the
notification's state before looking at anything else.** The state and
its transition history are designed to answer "where did it stop", and
guessing before reading them wastes the instrumentation.

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Nothing is arriving on any device | `notification-hub channels list` — is any channel enabled and healthy? Then `/health`. | A provider credential that never resolved shows as an unhealthy channel, not as a failed send. Re-provision the credential. | If channels are healthy and notifications still settle `failed`, capture one delivery's reason code and record it in `PROBLEMS.md`. |
| One device stopped receiving | `notification-hub delivery list --device <id>` and read the failure reason. | A provider 4xx means the address is wrong: re-register the device channel. A push topic is a bearer secret and re-registering rotates it. | — |
| Notifications are stuck in `held` | `notification-hub notifications list --state held` and check the recipient's quiet windows and timezone. | A quiet window in the wrong timezone holds indefinitely from the operator's point of view. Correct the window; held notifications re-run routing on release. | If a hold outlived its staleness bound and was neither released nor dropped, that is a scheduler bug, not a configuration problem. |
| Duplicate notifications arriving | Compare `dedupe_key` on the notifications; a NULL key never suppresses. | Fix the calling scenario to send a stable key. Suppression cannot fix a caller that varies its key per attempt. | Record repeat offenders — a caller that spams is a fleet-level problem, not a hub problem. |
| Everything is marked `critical` | Delivery analytics by caller identity. | Critical bypasses quiet hours by design, so a caller that over-uses it defeats the whole preference layer. Fix the caller. | — |
| Relayed channel unavailable | `vrooli-bridge nodes list` — is the node present, and does it advertise the channel capability? | An unenrolled or revoked node is never selected, which is correct behavior, not a fault. Re-enrol or re-advertise. | — |
| Relayed delivery times out | Node presence and `LastSeenAt`, then the bridge dispatch run for the correlation id. | A node that is briefly offline is expected to delay, not fail; durable dispatch is chosen for exactly this. A persistent timeout means the node's agent is not processing jobs. | Escalate to the node, not to this scenario. |
| iMessage delivery fails on the Mac node | Full Disk Access for the agent, an unlocked session, a signed-in Messages account. | Any of the three missing breaks it. Re-check after every macOS update. | Do not treat this as a release blocker; OT-P1-002 is best-effort by decision. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| Device and channel registry | Include `SQLITE_PATH` in the host backup set. | Restore the file; devices resume without re-registration. | **Matters most.** Notification history is replaceable; the device registry is not. Losing it means re-registering every device by hand and rotating every push topic. |
| Preferences and quiet windows | Same file. | Same file. | Small, hand-recoverable, but annoying to reconstruct. |
| Notification and delivery history | Same file. | Same file. | Lowest value. Bounded by the retention window and safe to lose. |
| Provider credentials | Not in this database. Backed up by the credential authority. | Re-provision through the resource's credential descriptors. | Never appears in a scenario backup, by design. |

A useful property of the zero-resource decision: the entire durable
state of this scenario is one SQLite file. Backup is a file copy and
restore is a file copy, on every platform, with no dump tooling.

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
