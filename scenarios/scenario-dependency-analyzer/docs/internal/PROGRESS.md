# Progress

## Progress Log

| Date | Author | Status | Notes |
| --- | --- | --- | --- |
| 2026-06-14 | Codex | active | Added actual interface graph, drift, Connect surface, strict lint configuration, and canonical docs layout. |
| 2026-06-14 | Codex | active | Migrated the UI into React-Vite slot layout scaffolding: tiny App composition root, provider/routes slots, app shell navigation, centralized selectors, generated strings workflow, and ErrorBoundary. `ui-health` slot warning is closed. |
| 2026-06-14 | Codex | active | Closed the measures gate by documenting explicit waivers for remaining stateful domains in `cli/manifest.json`; `measures-health`, `cli-health`, `ui-health`, `proto-health`, and `vrooli scenario test scenario-dependency-analyzer` pass. |
| 2026-06-14 | Codex | active | Moved graph, deployment, and catalog UI surfaces into concrete `src/features/*` folders and centralized endpoint calls in `src/api/client.ts` while preserving route behavior. |
| 2026-06-14 | Codex | active | Installed and wired React Router, React Query, `i18next`/`react-i18next`, Vitest, Testing Library, jsdom, axe-core, and coverage tooling; added initial route/API/ErrorBoundary/a11y tests. |
| 2026-06-14 | Codex | active | Started Phase 6 design-system cleanup by centralizing semantic status tones, adding the experience audit, and broadening route tests for graph errors, catalog Scan & Apply, and deployment degraded states. |
| 2026-06-14 | Codex | active | Added a keyboard-friendly graph node/edge table, extracted deployment status derivation into pure helpers, and covered both with UI tests. |
| 2026-06-14 | Codex | active | Split deployment readiness rendering into smaller feature components, removed remaining tracked raw semantic status palette classes from UI TS/TSX, and added route coverage for keyboard status-row detail expansion. |
| 2026-06-14 | Codex | active | Added UI regression coverage for graph table empty-filter states and deployment scan failure messaging; fixed deployment route scan promise propagation so the dashboard can surface mutation errors. |
| 2026-06-14 | Codex | active | Closed Phase 8 runtime status drift by making `/api/v1/health/analysis` conform to the standard lifecycle health schema while preserving analysis diagnostics; `make restart` now returns SDA healthy. |
| 2026-06-14 | Codex | active | Fixed browser-driven UI smoke/performance asset loading by aligning the SDA UI server CORS default with other scenario UI servers; `vrooli scenario test scenario-dependency-analyzer` passes again. |
| 2026-06-14 | Codex | active | Closed Phase 7 documentation manifest drift by registering the remaining top-level API/CLI/integration docs, removing stale planned labels for implemented graph/drift commands, and restoring docs health to score 1. |
| 2026-06-14 | Codex | active | Started Phase 3 API convergence by extracting scenario catalog and deployment REST presentation into domain-owned adapters with focused tests, while retaining existing app service implementations where needed. |
| 2026-06-14 | Codex | active | Continued Phase 3 API convergence by moving graph REST presentation, centrality analytics, and cycle detection into `api/internal/graph` with adapter and domain tests. |
| 2026-06-14 | Codex | active | Continued Phase 3 API convergence by extracting analysis/scan, dependencies/impact, proposal, optimization, and core-set REST presentation from `api/internal/app` into domain-owned adapters with focused tests. |
| 2026-06-15 | Codex | active | Moved the generated interface graph Connect presentation from `api/internal/app` into `api/internal/graph`, leaving app startup as route wiring only and adding graph-domain proto mapping coverage. |
| 2026-06-15 | Codex | active | Closed the UI large-chunk warning by lazy-loading orientation, graph, deployment, and catalog route surfaces from `src/app/routes.tsx`; `pnpm run build` now emits split route chunks without Vite's 500 kB warning. |
| 2026-06-15 | Codex | active | Continued Phase 3 API convergence by splitting app-backed service implementations out of `service_registry.go` into focused app service files; the registry is now only composition. |
| 2026-06-15 | Codex | active | Moved dependency graph construction into `api/internal/graph` with direct builder tests, leaving `api/internal/app` as composition/fallback wiring for graph generation. |
| 2026-06-15 | Codex | active | Reassessed React-Vite provenance and bumped `.vrooli/service.json` to template `1.1.0` with recorded template hashes; `vrooli scenario template drift scenario-dependency-analyzer --json` now reports an ok status. |
