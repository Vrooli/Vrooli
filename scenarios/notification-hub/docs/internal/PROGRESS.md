# Progress — Notification Hub

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-08-17 | claude | done | Removed the 2025 notification-hub and regenerated the scenario from `react-vite` 1.6.5 with the `vrooli-default` design kit. The predecessor had none of the eight current structural markers, 14 of 20 business endpoints returning `[]` or `"Not implemented"`, an empty requirements registry, a static-HTML UI served by Express, and a `.vrooli/testing.json` swept to declare a `react-vite` policy profile it could not satisfy. No proto footprint existed, so plain directory removal was correct; the regeneration created `packages/proto/schemas/notification-hub/` and `gen/manifests/notification-hub.lock.json`, which means future removal must use `template-manager lifecycle destroy`. Salvaged the old schema, CLI domains, and API sources for reference before removal. |
| 2026-08-17 | claude | done | Fixed the one failing generation hook: `golangci-lint run --fix ./...` reported `fileRootPath` unused in `api/main.go`. Removed the helper, matching how `device-control` and `money-ledger` resolved the same template seam. All six post-generation hooks now pass and the CLI installs and runs. |
| 2026-08-17 | claude | done | Authored the business contract through the `business-health` wizard: `PRD.md` plus a requirements registry of 10 P0, 8 P1, and 5 P2 operational targets, each linked by `prd_ref` and honestly statused `planned` with stub manual validations. Removed the leftover `01-foundation` starter module. `business-health validate scenario notification-hub` passes with no findings. |
| 2026-08-17 | claude | done | Authored the product documentation: domain map (5 product domains plus `health`, 2 more committed at P1), data and schema map, integrations and dependency contract, flows and state machines, security and sensitivity model, decisions log, deployment tiers, runbook, observability signals, performance budgets, and business classification. Recorded the zero-resource decision and its portability rationale in four places so it cannot be undone by accident. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
