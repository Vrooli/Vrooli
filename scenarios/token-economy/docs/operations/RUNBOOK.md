# Runbook — Token Economy

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
| CLI talks to old API | `token-economy status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| **Everyone is locked out; holder and minter surfaces both refuse** | `/health` dependency status for `scenario-authenticator`; `vrooli scenario status scenario-authenticator` | Start the authenticator. **This is correct behavior, not a bug** — the scenario fails closed because isolation cannot be enforced without a verifiable identity. | If the authenticator cannot start, the scenario is legitimately unavailable. Do not add a bypass. |
| **A balance looks wrong** | Compare the projection against a full journal replay before anything else | If they agree, the balance is right and the *expectation* is wrong — read the holder's history, which explains every change. If they disagree, this is a correctness incident. | Disagreement means a bug already committed to an append-only store. Stop writes, capture the database, and file to scenario-qa. Do not "fix" the projection by editing it. |
| **A grant or redemption was wrong** | Find the event in the journal | Issue a **compensating correction** through the product. Never edit or delete the row — there is no verb for it, and both entries remaining visible is the intended behavior. | None. This is ordinary product use, not an incident. |
| **A holder can see another holder's data** | Reproduce with two authenticated sessions | Stop the scenario immediately. | **Highest-severity incident this scenario has.** File to scenario-qa at once; this is the failure that ends trust in a household product. |
| **Balances inflating; earning events duplicating** | Replay/no-op rate per adapter; `earning_submissions` dedup keys | Disable the offending adapter. Correct the surplus with compensating events. | Likely an adapter retrying with a fresh dedup key each time. Fix the adapter, not the ledger. |
| **Approval requests are not reaching the minter out of band** | `notification-hub` reachability | Nothing to fix in this scenario — the queue is first-class and works unchanged. Check the console. | Informational only. `notification-hub` absence is the baseline, not degraded mode. |
| **Events landing with unverified provenance** | `agent-manager` reachability; share of unverified events | Restore agent-manager if agents should be attributable. | Never blocks a write. Rising unverified share means identity verification is degrading upstream. |

## Backup / Restore

**Backup is not optional here.** The journal is append-only with no repair verb
by design, so a restored backup is the *only* true recovery path for corruption
(see `DEPLOYMENT.md` § Rollback). It is a release gate, not a maintenance
suggestion.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | Stop the scenario, copy the database file, restart. The file is the only copy of the journal and therefore of every balance. | Stop the scenario, replace the file, restart, then rebuild the projection and verify it against a full replay before allowing writes. | **required before first real use** |
| Projection cache | Not backed up. Derived and rebuildable from events at any time. | Rebuild from the journal. | by design |
| Blob files | not-applicable | not-applicable | This scenario stores no binary payloads. |

**After any restore**, verify projection-equals-replay before allowing writes.
Restoring an older database and writing on top of a stale projection is the one
sequence that could silently produce a wrong balance.

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| **Back up the database** | before any upgrade, and on a routine schedule | Copy the scenario database while stopped. The only recovery path for corruption. |
| **Verify projection against a full replay** | routinely, and always after a restore | The integrity check that catches a correctness bug before a human notices a wrong balance. |
| Validate the business contract | after PRD or requirements edits | `vrooli scenario requirements validate token-economy --json` |
| Validate the experience contract | after `experience/` edits | `experience-manager spec validate token-economy --json` |
| Check orientation gates | during initialization | `make orient` |
| Review pending approvals | as an operator habit | Minter console. A rotting queue means a child is waiting. |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
