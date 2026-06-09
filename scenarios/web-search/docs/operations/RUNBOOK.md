# Runbook — Web Search

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

> **Status (2026-06-09):** Not implemented yet. Procedures below are the
> *intended* operational playbook from `PRD.md` and
> [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) (whose
> degraded-behavior matrix is the source of truth for failure modes).

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

## Health Check

`/health` reports API readiness plus dependency reachability. The key
dependencies to confirm during any incident:

- **SearXNG** (`SEARXNG_URL`) — live-web engine for `web-search.live`
  (L0/L1) and the L2 source. **Verify this first for any live-search
  issue.**
- **Qdrant** — semantic index `web-search-findings` for
  `web-search.learnings`.
- **Ollama** — embeddings (`nomic-embed-text`) + chat model for L1/L2/L3
  synthesis and distillation.
- **search-hub** — provider registration target for both providers.
- **SQLite** (`web-search.db`) — findings/briefs/audit store; if its
  `PingContext` fails, `/health` is unhealthy.

Reranker and (P1) browserless / agent-manager are also probed but their
absence only degrades, never fails health.

## Common Incidents

| Symptom | Checks | Fix / Expected Behavior | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs`, SQLite path writable | `make restart`; verify `web-search.db` data dir is writable | Record recurring failures in `../internal/PROBLEMS.md`. |
| API unhealthy | `/health`, SQLite `PingContext`, API logs | Run `make setup`; verify writable data dir | Check `INTEGRATIONS.md` dependency expectations. |
| **SearXNG down / unreachable** | `/health` SearXNG status, `SEARXNG_URL`, SearXNG resource health | **Expected:** `web-search.live` degrades to unavailable with a surfaced warning; the budget governor returns "rate-limited, try later". **Learnings (findings) are unaffected and keep serving.** Restore the SearXNG resource. | If SearXNG itself is unhealthy, treat as a resource incident; record in `../internal/PROBLEMS.md`. |
| Live web "rate-limited, try later" | Budget-governor remaining tokens, cache hit-rate | **Expected, not a fault.** The token-bucket governor is protecting SearXNG/external engines per time window. Wait for the window to refill or serve from cache; never bypass the governor. | Only escalate if the window/limit is mistuned for legitimate load. |
| Findings search returns nothing / poor recall | `/health` Qdrant status, index size vs SQLite row count | **Qdrant down → expected text-match fallback** (degraded recall, reindex deferred). When Qdrant returns, trigger a reconcile/reindex. | If recall stays poor with Qdrant healthy, investigate the index/embedding path. |
| Synthesis missing / no cited answer | `/health` Ollama status | **Ollama down → expected:** raw hits still returned; L1/L2/L3 synthesis and distillation unavailable. Restore Ollama. | — |
| Provider not in search-hub results | search-hub reachability, `.vrooli/search.json`, registration logs | **search-hub down at boot → expected:** web-search retries briefly, serves locally, and **re-registers on next boot**. `make restart` after search-hub is back. | If registration keeps failing with search-hub healthy, check the descriptor/control-token handshake. |
| Disputed/contradictory findings surfacing | Dispute review queue depth | **Expected:** disputed findings surface *with* a "sources conflict" warning and both sources — never silently resolved. Resolve via the dispute queue (resolve / re-research / dismiss). | Growing queue depth signals a curation gap — see Maintenance. |
| UI blank or stale | UI port, browser console, `ui/dist` freshness | `make setup` then `make restart` | Add troubleshooting entry if recurring. |
| CLI talks to old API | `web-search status`, configured API base | Reinstall via `make setup` | Update CLI reference if a command changed. |

## Backup / Restore

The findings store is durable and **never hard-deleted** (supersede /
archive only) — it is *not* disposable. Back it up before any schema
change and on a deployment cadence.

| Data | Backup Procedure | Restore Procedure | Status |
|---|---|---|---|
| SQLite `web-search.db` (findings/briefs/citations/audit) | deferred — define file-snapshot/data-backup-manager procedure before deployment | deferred | Define before deployment. Migrate-never-recreate on schema change. |
| Qdrant `web-search-findings` index | n/a — derived; rebuild from SQLite via aisearch-go reconcile | reconcile/reindex from SQLite | Derived data; reconstructable. |
| Live-web cache | n/a (ephemeral, TTL'd) | n/a | No backup needed. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff | `make test` |
| Inspect logs | as needed | `make logs` |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Findings management | as needed | `web-search findings ...` (list / add / edit / supersede / flag / prune) — OT-P0-006 |
| Resolve disputes | as queue grows | work the dispute review queue (resolve / re-research / dismiss) — OT-P1-005/007 |
| Reconcile / reindex findings | after Qdrant recovery or drift | aisearch-go reconcile (rebuilds `web-search-findings` from SQLite) |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts + degraded-behavior matrix
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
