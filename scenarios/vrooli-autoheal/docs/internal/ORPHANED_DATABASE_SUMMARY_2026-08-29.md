# Orphaned Autoheal Database Summary — 2026-08-29

This artifact preserves the compact operational evidence requested before the
scenario-local database was reclaimed. All queries were read-only and completed
before removal.

## Source proof

- Database: `path:scenarios/vrooli-autoheal/data/autoheal.sqlite`
- Database bytes: 9,354,616,832
- WAL bytes: 4,507,312
- SHM bytes: 32,768
- Last database write: 2026-08-22 18:27:32 UTC
- `PRAGMA integrity_check`: `ok`
- `lsof` showed no process owning the scenario-local file; the running API
  owned only the resolver-routed database under
  `$USER_HOME/.vrooli/data/vrooli/vrooli-autoheal/`.
- A repository literal-path search found documentation/evidence references and
  a storage-manager example comment, but no runtime reader or writer.

## Heal tracker summary

The orphan contained 35 tracker rows with 5,111 total attempts, 3,756 total
successes, and 366 consecutive failures at its final snapshot.

| Check | Attempts | Successes | Consecutive failures |
|---|---:|---:|---:|
| `resource-ollama` | 710 | 658 | 1 |
| `resource-qdrant` | 249 | 107 | 1 |
| `resource-redis` | 232 | 207 | 1 |
| `resource-searxng` | 75 | 2 | 67 |
| `resource-whisper` | 1,410 | 1,286 | 1 |
| `vrooli-orphans` | 341 | 315 | 6 |

There was no `resource-reranker` or `resource-kokoro` tracker row. Absence is
preserved explicitly rather than represented as zero attempts.

## Action log summary

The orphan contained 6,311 action rows: 3,575 succeeded and 2,736 failed. The
covered interval was 2026-08-05T22:36:16.978187534Z through
2026-08-20T17:59:38.224997137Z.

| Check | Actions | Succeeded | Failed |
|---|---:|---:|---:|
| `resource-ollama` | 683 | 631 | 52 |
| `resource-qdrant` | 486 | 106 | 380 |
| `resource-redis` | 203 | 178 | 25 |
| `resource-reranker` | 325 | 0 | 325 |
| `resource-searxng` | 389 | 1 | 388 |
| `resource-whisper` | 1,383 | 1,259 | 124 |
| `vrooli-orphans` | 357 | 314 | 43 |

The orphan action ledger recorded 311 `vrooli-orphans` `kill` actions: 305
succeeded and 6 failed. It also recorded 23 `list` actions and 23 suppressed
`autoheal-skip` actions. There were no `resource-kokoro` action rows.

## Export queries

```sql
SELECT COUNT(*), SUM(total_attempts), SUM(total_successes),
       SUM(consecutive_failures)
FROM heal_trackers;

SELECT check_id, total_attempts, total_successes, consecutive_failures
FROM heal_trackers
ORDER BY total_attempts DESC, check_id;

SELECT check_id, COUNT(*) AS actions,
       SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) AS succeeded,
       SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) AS failed,
       MIN(created_at), MAX(created_at)
FROM action_logs
GROUP BY check_id
ORDER BY actions DESC, check_id;
```

This artifact is a summary, not a replacement database. The active resolver-
routed database remains the authoritative state store.
