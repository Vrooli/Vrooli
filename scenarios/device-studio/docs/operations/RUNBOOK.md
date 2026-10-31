# Runbook — Device Studio

## Purpose Of This Document

Operate the current local app and distinguish implemented workflows from remaining delivery work.

## Start / Stop / Status

From `scenarios/device-studio`, use `make setup`, `make start`, `make status`, `make logs` and `make stop` through governed lifecycle. Resolve dynamic ports from lifecycle status; never persist an observed development port as a product default. Do not run scenario API binaries directly. Project, media, shot and render commands are implemented; consult CLI help for exact current inputs. Product AI creation and standalone packaging remain unqualified.

## Common Incidents

| Symptom | Check | Action |
|---|---|---|
| App startup fails | Lifecycle logs and shared generated freshness | Repair through owning setup/generation path; preserve unrelated work |
| Render cannot use GPU | Worker backend/kernel/device receipt and available capacity | Retain the failed attempt and explain the limitation; select a different backend explicitly or retry after capacity changes; do not unload unrelated models |
| Missing input/tool | Immutable manifest and digest resolution | Restore correct asset/profile; never substitute a different device silently |
| Job appears stale | Job-owned heartbeat/attempt and output receipts | Reconcile ownership; do not spawn a duplicate paid run |
| Account/credit failure | Typed account/session/lease/funds status | Reconnect correct account or choose local/BYOK explicitly |
| Publication unavailable | Domain review and shared owner receipt | Keep private output and release pending |
| Full disk | Quota/checkpoint/cache inventory | Stop new admission, retain pinned inputs/receipts; delete only eligible cache |

Account, publication and full retention responses still require their integration evidence. Do not act on arbitrary process IDs or edit databases as recovery. Before development rebuilds, archive and close active paired test leases, or use a supported restart checkpoint. A rejected checkpoint must stop the restart sequence; it is not permission to stop the serving app.

## Backup / Restore

Preserve SQLite metadata consistently with referenced immutable artifacts, project manifests, rights/reviews and unresolved job/accounting receipts. Exclude credentials from project backups; use the credential owner's recovery policy. Restore to a separate location and verify hashes/reopen a shot before replacing working state. The historical research root is protected; it is not disposable cache or a backup cleanup target.

## Maintenance Tasks

Use owner-managed tool updates and explicit schema migrations; validate active-job pinning before update. Reconcile caches by references and retention policy. Revalidate library scope after asset/profile/tool changes. No automatic model purchases, paid fallback or external-media upload occurs as maintenance.

## Escalation

Report defects at their owning scenario using the report-bug workflow. Preserve exact operation/revision/receipt IDs and safe diagnostics. A missing renderer, provider, account or platform proof is reported distinctly. Human-only release and actual spend/signing configuration remain with their owners.

## Cross-References

[DEPLOYMENT.md](DEPLOYMENT.md), [OBSERVABILITY.md](OBSERVABILITY.md), [problems](../internal/PROBLEMS.md), [research](../research/README.md).
