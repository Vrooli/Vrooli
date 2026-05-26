# Progress — Go Code Graph

Lifecycle log for meaningful scenario changes. Future agents read this file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-23 | Claude (Opus 4.7) | done | Scenario generated from `react-vite` template (default vrooli-default design kit) via `vrooli scenario generate react-vite --id go-code-graph --display-name "Go Code Graph" --description "..." --run-hooks`. PRD authored covering 10 P0 / 5 P1 / 5 P2 operational targets. Requirements generated via `prd-control-tower requirements generate go-code-graph` (15 modules; `prd-control-tower requirements validate` reports `healthy`, all 20 targets linked). Docs populated: DOMAINS (graph/rewrite/explorer/health domains), FLOWS (Extract / Rewrite plan / Rewrite apply), INTEGRATIONS (golang.org/x/tools/go/packages + lifecycle + optional SQLite), DATA (operation log P1), DECISIONS (architectural choices from initialization conversation), PROBLEMS (initialization stub entry + template lint defect), SEAMS (PackagesLoader, RewriteExecutor, PlanRegistry, PathMutex), SECURITY (source-code-read threat surface), PERFORMANCE (≤200 files <5s, ≤2000 files <30s SLA), DEPLOYMENT/RUNBOOK/OBSERVABILITY (Tier 1 local), MONETIZATION/GO-TO-MARKET (marked not-applicable — infrastructure scenario). Implementation has not started. Scenario is ready for the first vertical-slice work (likely `graph` domain → Extract). |
| 2026-05-25 | Claude (Opus 4.7) | done | **Correction to the 2026-05-23 entry's "implementation has not started":** between generation and this date the engine landed — `internal/graph` (Extract + Normalize + GraphHash + determinism), `internal/rewrite` (Plan/Apply, FS executor, SQLite plan store), the `GoCodeGraphService` proto + Connect handler, and the `graph`/`rewrite` CLI domains were all implemented and tested. The UI alone remained the template default. |
| 2026-05-25 | Claude (Opus 4.7) | done | **Graph Explorer & Rewrite Workbench UI (OT-P0-009 + P1 Fixture Validator).** Replaced the template Dashboard with a single-page operational workbench (`pages/WorkbenchPage.tsx`): persistent extract bar (module path + project-dir override + include-vendor) and a 5-stat header (files/packages/symbols/imports/warnings + hash + extraction ms), over four tabs. **Graph** (`features/explorer/`): hand-rolled SVG canvas with Kahn-BFS layered layout + Tarjan SCC cycle highlighting, an always-present keyboard-navigable accessible list (WCAG text alternative), legend, package filter bar, and file→symbol drill-down — all fed by a pure `lib/graphAdapter.ts` (the only go-specific seam; shaped after architecture-cartographer for clean duplication into typescript-code-graph). **Warnings** (`features/warnings/`): parse/unresolved-import diagnostics with severity badges + a vendor-filter toggle that re-runs Extract with `include_vendor`. **Rewrite** (`features/rewrite/`): typed FileMove/ImportRewrite op editor → `RewritePlan` dry-run green/red pseudo-diff → `RewriteApply` behind a confirm dialog. **Fixtures** (`features/fixtures/`): lists golden fixtures and runs server-side validation showing pass/fail + diff. New Connect clients `api/graph.ts` + `api/rewrite.ts`. i18n complete across en/ja/ar (`strings:check` green). **Substrate fix (in-scope per plan):** added `GoCodeGraphService.ListFixtures` + `ValidateFixture` RPCs (proto + Go handler reusing Extract + new `graph.CanonicalJSON` 2-space serializer + byte-compare vs `expected-graph.json`), plus `graph list-fixtures` / `graph validate-fixture` CLI commands for API/UI/CLI parity. Also fixed pre-existing template gate failures uncovered along the way: React Router v7 future flags wired on routers, duplicate `Primary navigation` landmark label (a11y), the AppShell `aria-label` literal, two SSR-guard `no-unnecessary-condition` lints, and orphaned `app.eyebrow`/`app.description` catalog keys. Gates: `pnpm lint`/`type-check`/`test` (158)/`build` green; `go test ./...` green for api + cli; `buf generate` + `.vrooli/endpoints.json` regenerated. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
