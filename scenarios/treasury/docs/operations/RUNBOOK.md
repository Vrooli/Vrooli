# Runbook — Treasury

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
| CLI talks to old API | `treasury status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| **All automated spend is refused** | Is `agent-manager` running? `vrooli scenario status agent-manager` | Start `agent-manager`. Automated spend resumes with no other action; nothing was recorded at a degraded grade while it was down. | This is the fail-closed decision working as designed, not a defect. See `../internal/DECISIONS.md`. |
| **A settlement is stuck in `unknown`** | The charge record and the rail's own status for that reference | **Query the rail. Never retry.** Retrying an unknown is how a double charge happens. Resolve the state from the rail's answer. | If the rail cannot answer, leave it `unknown` and escalate. An unresolved `unknown` is safer than a guessed outcome. |
| **Money moved but the ledger does not show it** | Emission log; is `money-ledger` reachable? | Emission retries on its own. Local evidence is authoritative until it succeeds. | If the backlog persists past one retry cycle, financial position is wrong downstream — escalate. |
| **An agent reports it cannot buy something** | The refused attempt in Activity; the named constraint | Not an incident. Read which constraint refused, then decide whether to widen it. | Only escalate if the refusal names no constraint, which would be a defect. |
| **A charge appeared that nobody expected** | The attempt's evidence record: mandate, approver, request | Freeze the relevant budget first, investigate second. The freeze binds before the next authorization. | Treat as a security incident until the evidence shows an ordinary authorization. |
| **Approval requests are not arriving on a device** | `notification-hub` reachability; relay attempt records | None required — the console queue is unaffected and approval still works. | Relay is an enhancement, never a dependency. |

## Backup / Restore

The generated template uses local SQLite state. Product scenarios must
define backup and restore procedures before production deployment.

**Backup matters more here than in an ordinary scenario.** The evidence
journal is the artefact that proves what was authorized; losing it does not
lose money but does lose the ability to account for money. Treat it as the
primary restore target rather than an afterthought.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database (evidence, mandates, charges) | Register with `data-backup-manager` before any automated rail is enabled. | Restore the file and restart through the lifecycle. | **Required before the first automated rail.** Not yet configured. |
| Idempotency keys | Included in the database backup. | Included in the restore. | Restoring a database *without* them would make in-flight client retries into potential double charges. Never restore a partial database. |
| Instrument credentials | Not applicable — held in `secrets-manager`, backed up there. | Re-resolved by reference after restore. | Deliberate; nothing to back up here. |
| Blob files | not-applicable | not-applicable | This scenario stores no blobs once the example domain is removed. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Review pending approvals | daily while agents are active | Console landing page; an aged pending request means a stalled agent. |
| Check for `unknown` settlements | daily while an automated rail is live | Activity page. Any non-zero count needs a human. |
| Verify the freeze works | before trusting a new rail | Freeze, attempt an authorization, confirm refusal, thaw. A kill switch never exercised is a kill switch never verified. |
| Reconcile against rail statements | monthly once an automated rail is live | Manual until `TRS-P2-001` lands. |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
