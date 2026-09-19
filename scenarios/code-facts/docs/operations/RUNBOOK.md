# Runbook — Code Facts

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
| CLI talks to old API | `code-facts status`, configured API base | Reinstall via `make setup` | Update CLI reference if command changed. |
| Search Hub withholds Code Facts | `code-facts index status`; inspect generation, `last_reconcile_at_unix`, degraded stages, and source/descriptor digests | `code-facts index reconcile`; use confirmed reindex/promote only when a shadow rebuild is required | Preserve the failed job and generation evidence before retrying. |
| Search Hub reports Code Facts unavailable | `make status`, `make logs`, then call `GetIndexStatus` through the lifecycle port | Restore scenario health; do not change the leaf to `local_live` to hide an index failure | Inspect Search Hub provider circuit and freshness evidence. |
| Queries return admission timeout/queue-full | Inspect `/health` `admission_in_use`, `admission_queued`, high-water, wait p95/p99, rejected, and cancelled totals | Cancel obsolete index jobs; reduce concurrent fleet/reindex work; allow queued cancellation to release capacity | Do not raise capacity until `rss_mb`, `rss_high_water_mb`, and CPU attribution show headroom. |
| Cache approaches quota | Inspect `cache_total_payload_bytes`, `cache_budget_bytes`, utilization, and per-scope rows/bytes | Run the normal cache sweep or clear a selected target; restart only when lifecycle remediation otherwise requires it | Graph is capped at 75% and reports at 50%; cleanup is deliberately bounded and may require a later scheduled pass. |
| Health is slow or counts disagree | Compare `active_generation`, `source_files`, `search_documents`, `semantic_cards`, and `graph_facts` with `code-facts index status` | Reconcile the catalog generation; preserve corruption evidence before rebuilding | Health uses trigger-maintained catalog/cache summaries and must not be changed back to full-table aggregation. |

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
| Inspect indexed-provider state | before/after reconcile | `code-facts index status` |
| Reconcile changed sources | after drift or watcher loss | `code-facts index reconcile` |
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
