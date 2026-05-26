# Progress — TypeScript Code Graph

Lifecycle log for meaningful scenario changes. Future agents read this file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-05-23 | Claude (Opus 4.7) | done | Scenario generated from `react-vite` template (default vrooli-default design kit) via `vrooli scenario generate react-vite --id typescript-code-graph --display-name "TypeScript Code Graph" --description "..." --run-hooks`. PRD authored covering 12 P0 / 6 P1 / 6 P2 operational targets. Requirements generated via `prd-control-tower requirements generate typescript-code-graph` (18 modules; `prd-control-tower requirements validate` reports `healthy`, all 24 targets linked). Docs populated: DOMAINS (graph/rewrite/sidecar/explorer/health domains — note the sidecar domain is unique to this scenario because ts-morph is a Node library), FLOWS (Sidecar lifecycle / Extract / Rewrite plan / Rewrite apply), INTEGRATIONS (ts-morph + Node runtime + lifecycle + IPC channel + optional SQLite), DATA (operation log P1, sidecar process state ephemeral), DECISIONS (architectural choices including the Node sidecar over alternatives and the load-bearing leading-comment contract for rcl migration), PROBLEMS (initialization stub entry + template lint defect + sidecar directory absent), SEAMS (SidecarClient, RewriteExecutor, PlanRegistry, PathMutex, SidecarSupervisor — planned), SECURITY (source-code-read threat surface including leading comments), PERFORMANCE (≤200 files <5s, ≤2000 files <30s SLA), DEPLOYMENT/RUNBOOK/OBSERVABILITY (Tier 1 local, sidecar lifecycle), MONETIZATION/GO-TO-MARKET (marked not-applicable — infrastructure scenario). Implementation has not started. Scenario is ready for the first vertical-slice work, which must start with the sidecar (REQ-P0-009) because both graph and rewrite depend on it. |
| 2026-05-25 | Claude (Opus 4.7) | done | **Correction:** the 2026-05-23 entry's closing "implementation has not started" is stale — the Node ts-morph sidecar, Go API (Extract/RewritePlan/RewriteApply), CLI, proto, and determinism layers all shipped and are tested. **This entry:** built the real product UI (OT-P0-011 Graph Explorer + Diagnostics), replacing the template Dashboard. Duplicated go-code-graph's workbench (duplicate-before-extract) into `ui/src/`: single-page workbench (extract bar + stats header + Graph / Warnings / Rewrite / Fixtures tabs), hand-rolled SVG `GraphCanvas` + keyboard-navigable `GraphAccessibleList` (WCAG AA, severity by label+icon never color-only), `graphAdapter` retargeted to the TS module graph (module nodes via `attributes.kind=TS_NODE_KIND_MODULE`, symbols linked to files by path, IMPORT+RE_EXPORT dependency edges). TS-specific deltas: (1) **sidecar status panel** reading `health.sidecar_status`/`sidecar_message` (no new RPC), enum→label+icon for all four states; (2) **JSDoc/leading-comment rendering** per symbol in the drill-down; (3) graceful **`workspace_unsupported`** messaging (typed Connect CodeUnimplemented → explanatory notice, not an error). Removed the Go-only vendor toggle. Substrate added for the fixture validator: `ListFixtures`/`ValidateFixture` RPCs on the proto + Go handler (`CanonicalJSON` promoted to production code; determinism/integration tests now validate it directly) + matching CLI commands (`graph list-fixtures` / `graph validate-fixture`). i18n complete across en/ja/ar (module terminology, sidecar states, workspace notice, JSDoc label); `strings:check` green. Fixed pre-existing template drift surfaced by the new gates: React Router v7 future flags wired on routers + test MemoryRouter, distinct bottom-nav landmark label (a11y `landmark-unique`), `THEME_CHOICE_LABELS` static accessor, ThemeProvider SSR-guard lint suppressions, `mainLabel` i18n key. All gates green: `pnpm lint` / `type-check` / `test` (159) / `build`; Go API+CLI build, registry/proto-coverage + fixtures handler tests pass. |

## Entry Template

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
