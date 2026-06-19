# Progress — Tunnel Manager

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-06-18 | regen agent | done | **Phase 0** — Regenerated from `react-vite` 1.1.0 + vrooli-default design kit (old scenario preserved at `/tmp/tunnel-manager-OLD-reference`). Scaffold boots healthy (API+UI), builds green, raw unit/deps phases pass. Replaces the pre-1.0.0 REST/JSON scenario with a Connect-RPC + screaming-architecture foundation. |
| 2026-06-18 | regen agent | done | **Phase 1 (docs-first)** — Authored PRD (12 P0 / 7 P1 / 8 P2 targets; reframed as exposure broker + self-healing control plane), 10-module requirements registry (40 reqs, `prd-control-tower requirements validate` → healthy, 27/27 targets linked), DOMAINS map (7 domains + health), DECISIONS log (12 durable decisions), Gate-4 `service.json` (SQLite, fixed UI port 21240, cloudflared hostTool, optional redis), and all concept/internal/ops/business/reference docs (honest "planned" framing; example fences preserved for Phase 2 detemplate). Plan: `docs/plans/tunnel-manager-regen-adoption-plan.md`. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
