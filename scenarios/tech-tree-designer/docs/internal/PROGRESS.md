# Progress — Tech Tree Designer

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-06-14 | Codex | done | Greenfield-regenerated Tech Tree Designer from `react-vite`, removed the template notes domain, kept the scaffold health surface green, and recorded the graph/planning/roadmap target domains. |
| 2026-06-14 | Codex | done | Added Phase 2 graph/planning/roadmap proto contracts, expanded graph CLI manifest coverage with deferred handlers, and implemented a tested `api/internal/graph` `ProtoHealthSource` seam over `DescribeScenariosProtos`. |
| 2026-06-14 | Codex | done | Implemented Phase 3 graph Connect handlers, graph query/export service logic, endpoint manifest entries, and runnable graph CLI commands over the generated `GraphService` client. |
| 2026-06-14 | Codex | done | Implemented Phase 4 planning API/CLI: SQLite planned proto file tree, protocompile validation, materialization, planned graph overlay, endpoint regeneration, and cli-health-clean manifest bindings. |
| 2026-06-14 | Codex | done | Implemented Phase 5 roadmap API/CLI: sector and milestone overlay storage, Connect handlers, progress rollup from graph node stability, manifest bindings, endpoint regeneration, and focused tests. |
| 2026-06-14 | Codex | done | Implemented Phase 6 UI surfaces: React Flow + ELK scenario graph canvas, planned proto editor with validation/materialization actions, roadmap progress overview, route-level code splitting, and navigation/test coverage. |
| 2026-06-14 | Codex | done | Implemented Phase 7 integration hardening: localized graph/planning/roadmap feature copy, fixed planned-proto validation for live imports with Google well-known dependencies, self-validated graph/planning/materialization flows, and restarted the scenario healthy. |
| 2026-06-14 | Codex | done | Cleaned up post-integration docs drift: README, quickstart, architecture, domains, flows, testing, monetization, API reference, and template docs now describe the shipped graph/planning/roadmap surface and pass docs validation without warnings. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
