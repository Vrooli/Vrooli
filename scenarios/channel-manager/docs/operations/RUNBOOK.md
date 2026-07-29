# Runbook — Channel Manager

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

## The daily loop

At P0 every action is executed by hand, so the operator *is* the executor. This is
the core procedure, not a fallback:

```bash
channel-manager queue due                    # today's actions, grouped by session
channel-manager queue complete <action-id> --evidence <url|note>
channel-manager signals record <identity> --metric views --value 1240
```

Two habits matter more than the commands. **Complete actions inside their window** —
the window is the point, and an action executed hours late is a different
behavioural signature from the one the program declared. And **record a metric
observation whenever the platform shows you one**, even casually; a baseline is only
as regular as the operator, and nothing downstream works without it.

## Live browser-execution acceptance (operator-gated)

Browser dispatch is available for synthetic validation, but it is **not** approved
for a real platform by default. Do not dispatch a live action until every preflight
item below has been completed by the accountable operator:

- Add a dated per-platform, per-action-kind acceptance row to
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md) § D-009. The row must name
  the operator and state the accepted platform-terms exposure; a generic approval
  is not enough.
- Use a dedicated sanctioned test identity, never a production identity. Confirm
  its environment attestation and its Vault reference without printing a credential.
- Create one BAS session profile for that identity only, record its opaque profile
  ID, and use an operator-reviewed persisted BAS workflow UUID. Do not reuse either
  reference for another identity.
- Verify that the operator has the scoped Vault access required by the approved BAS
  workflow. Channel Manager stores only the Vault path; credentials, cookies, and
  session contents must not enter its database, API responses, or logs.
- Queue one non-destructive, platform-approved test action through the normal
  Channel Manager release path. Capture its action ID, BAS execution ID, timestamps,
  outcome URL or platform-post ID, and any artifacts in the action evidence.

If dispatch, the workflow, or the Vault read fails, leave the action uncompleted and
use the permanent manual executor. If a live action succeeds unexpectedly or its
account/environment becomes suspect, pause the identity immediately; quarantine it
when the warming gate requires that terminal outcome. Record the incident and the
action/BAS IDs in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md), preserve the
evidence, and do not retry browser execution until the accountable operator records
a new decision. A live acceptance is complete only when the release receipt and any
first-comment outcome are reconciled in Content Desk and the relevant requirement
evidence is updated truthfully.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite path, API logs | Run `make setup`, verify writable data dir | Check `INTEGRATIONS.md` for dependency expectations. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `channel-manager status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| An identity's queue is empty and it is mid-program | `channel-manager warming show <identity>` — is a gate waiting, or is the identity paused? | A waiting gate is normal until its interval elapses. A pause needs a flag resolution. | A gate stalled well past its interval is a defect; file it. |
| An action cannot be queued | Read the refusal reason — it names phase, ceiling, or eligibility. | Refusals are correct behaviour, not errors. A phase-forbidden action means the program is not ready for it. | If a *valid* action is refused, the descriptor or the ceiling is wrong. |
| Identity flagged and paused | `channel-manager signals flags <identity>` for the evidence that raised it | **Operator decision. Never auto-resume.** Judge the measurement against the baseline; resolve or keep paused. | If flags fire constantly on healthy accounts, the decay thresholds are too tight — record it against the platform descriptor. |
| Identity quarantined | The gate measurement that failed | **Do not resume.** Quarantine means abandon and rebuild with a tighter environment (D-007). | Repeated quarantines on one environment are evidence the attestation was wrong. |
| Vault unreachable | `resource-vault` status | Browser and API execution fail terminally; manual execution is unaffected. | No action is ever marked complete on a credential failure. |
| Descriptor edit not taking effect | Confirm reseed ran | `channel-manager descriptors reseed` — the file is authoritative, the table is a cache. | A descriptor that fails validation blocks the seed loudly; read the error. |

## Backup / Restore

Local SQLite. What matters is which parts are reconstructable, because the answer
is "almost none of them."

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite database | Include the scenario data dir in the machine's normal backup. | Restore the file with the scenario stopped. | **Irreplaceable.** Action records, release records, metric observations, and program observations are the only record of what was done as real accounts, and no second copy exists anywhere (`DATA.md` § Rebuild contract). |
| Descriptors under `data/` | Versioned in git. | `git checkout` plus `descriptors reseed`. | Fully recoverable. |
| Baselines | None needed. | Recomputed from observations. | Rebuildable cache. |
| Credentials | **Not here.** `vault` owns them and has its own backup story. | n/a | Backing up this database never backs up a credential — by design. |

A restored database that has drifted behind the live accounts is worse than no
restore: the queue will reschedule actions that were already performed, and cadence
accounting will under-count. After any restore, reconcile the day's completed
actions before resuming the queue.

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
