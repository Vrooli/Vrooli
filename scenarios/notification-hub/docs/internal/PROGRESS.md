# Progress — Notification Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | claude | done | Regenerated the scenario from `react-vite` 1.6.5 after the previous notification-hub was assessed as architecturally stranded: none of the template's structural markers, 6 of 20 business endpoints implemented, every send path terminating in a log line, and an empty requirements registry. Authored the charter PRD and populated the requirements registry with 23 operational targets across three modules. Code is still the generated scaffold; the `notes` worked example has not been removed. |
| 2026-08-17 | claude | done | Folded the monetization research back into the charter. Promoted acknowledgement and escalation from P2 to P1 and added a blocking ask primitive, on the finding that return traffic is what separates this scenario from both adjacent markets. P1 went 8 → 11, P2 went 5 → 3, P0 unchanged at 13; traceability matrix reports 27 of 27 rows linked and `business-health validate scenario notification-hub` passes. **Requirement id remap:** `NOTIFICA-P2-004` → `NOTIFICA-P1-009` (acknowledgement and response), `NOTIFICA-P2-005` → `NOTIFICA-P1-011` (escalation chains); `NOTIFICA-P1-010` (blocking ask primitive) is new. Safe to renumber because neither moved requirement was implemented or carried a `[REQ:...]` test tag. |
| 2026-08-17 | claude | done | Closed the charter-versus-scaffold contradictions found on review. Declared `scenario-authenticator` (required, `try_start`), `vrooli-bridge`, and `vrooli-events` in `.vrooli/service.json` as `runtime_only` dependencies with explicit `degraded_behavior`; rewrote `INTEGRATIONS.md` as a real dependency contract including the rejected-resource rationale and the upstream `vrooli-events` fan-out gap; backfilled ten durable decisions into `DECISIONS.md`; opened three `PROBLEMS.md` entries; and re-stamped 17 documents in `docs/manifest.json` from `active` to `stub`/`draft` so documentation health stops reporting generic scaffold as finished. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
