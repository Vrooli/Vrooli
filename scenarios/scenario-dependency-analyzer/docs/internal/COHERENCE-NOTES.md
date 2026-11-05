# UI Coherence Notes

## 2026-06-14 Phase 0 Inventory

This file tracks the React-Vite 1.1 migration inventory for the Scenario Dependency Analyzer UI. It is intentionally current-state focused; planned architecture changes belong in the migration plan.

### Current Structure

- `ui/src/App.tsx` is a small composition root that mounts providers and routes.
- `ui/src/app/routes.tsx` owns active route state, URL search params, cross-feature graph/catalog data coordination, and route selection until React Router and React Query are available.
- Primary graph, deployment, and catalog surfaces live under `ui/src/features/{graph,deployment,catalog}`.
- Server state is loaded through feature-local custom hooks in `features/graph/useGraphData.ts` and `features/catalog/useScenarioCatalog.ts`; React Query is not yet installed or used.
- UI primitives live under `ui/src/components/ui`; `OrientationPage` remains in `ui/src/pages` as the only non-feature route page.
- Template slots now exist for `src/app`, `src/layout`, `src/consts`, `src/i18n`, `src/theme`, `src/test-utils`, and `src/api`.

### Validation Snapshot

- `cd api && GOWORK=off go test ./...` passed.
- `cd cli && GOWORK=off go test ./...` passed.
- `cd ui && pnpm run lint` passed.
- `cd ui && pnpm run type-check` passed.
- `ui-health validate scenario scenario-dependency-analyzer` warned that `generation.template.id` was missing before this migration pass.
- `cli-health validate scenario scenario-dependency-analyzer` warned that `cli/manifest.json` is missing.
- `proto-health validate scenario scenario-dependency-analyzer` passed with informational `proto.possibly_unused` findings for graph and health messages.
- `vrooli scenario status scenario-dependency-analyzer` reported `Health: unhealthy` while direct API/UI health endpoints returned `200 OK`.

### Style and Experience Debt

- The current header uses decorative blurred color orbs in `App.tsx`; replace or remove during the design-system phase.
- Route-level navigation is represented by Radix tabs and query params rather than React Router routes.
- Shared strings and stable test selectors are not centralized, so future tests would bind to visible text or ad hoc selectors.
- No unit/a11y test harness exists in the UI package.
- PWA metadata was added during this migration with neutral placeholder icons from the React-Vite template; Brand Manager should replace them with SDA-specific assets.

## 2026-06-14 Phase 5 Slot Migration Slice

### Completed

- `ui/src/App.tsx` is now a small composition root that mounts `app/providers.tsx` and `app/routes.tsx`.
- Added the template slot directories for `app`, `layout`, `consts`, `i18n/locales`, `theme`, `test-utils`, `api`, and `features`.
- Replaced the monolithic tab shell with an `AppShell` plus path-backed route state for `/`, `/graph`, `/deployment`, and `/catalog`.
- Preserved legacy `?view=` links as a compatibility input and kept `graph_type`, `layout`, and `scenario` query params.
- Added centralized selectors, generated strings scaffolding, and a minimal `ErrorBoundary`.
- Removed the decorative header orb layer from the root app shell; broader palette/token cleanup remains Phase 6 work.

### Validation

- `ui-health validate scenario scenario-dependency-analyzer` now passes with no warnings.
- `pnpm strings:check` passed.
- `pnpm run lint` passed.
- `pnpm run type-check` passed.
- `pnpm run build` passed.
- `pnpm run build` passed.

### Remaining UI Migration Work

- React Router, React Query, `i18next`/`react-i18next`, and Vitest/Testing Library/a11y were not installed in the first slot slice because package installation required explicit permission.
- Feature folders now own graph, deployment, and catalog route/component code; cross-surface orchestration remains in `app/routes.tsx`.

## 2026-06-14 Phase 5 Feature Ownership Slice

### Completed

- Moved graph route composition, controls, canvas, telemetry, selected-node inspection, graph utilities, and graph data hook into `ui/src/features/graph`.
- Moved scenario catalog route composition, catalog/detail panels, optimization panel, and catalog data hook into `ui/src/features/catalog`.
- Moved deployment route composition, dashboard, readiness insights, metadata gaps, and recommended workflow panels into `ui/src/features/deployment`.
- Kept `ui/src/components/ui` as the shared primitive boundary and kept `OrientationPage` in `pages` as the only remaining route-level non-feature page.
- Centralized raw endpoint calls and Connect JSON transport calls in `ui/src/api/client.ts`; feature hooks now own state transitions and call the shared API seam.

### Validation

- `pnpm strings:check` passed.
- `pnpm run lint` passed.
- `pnpm run type-check` passed.
- `pnpm run build` passed.

### Remaining UI Migration Work

- React Router, React Query, `i18next`/`react-i18next`, and Vitest/Testing Library/a11y adoption were completed in the follow-up dependency slice.
- Phase 6 token and primitive cleanup remains open.

## 2026-06-14 Phase 5 Dependency Adoption Slice

### Completed

- Added React Router, React Query, `i18next`/`react-i18next`, Connect web runtime dependencies, Vitest, Testing Library, jsdom, axe-core, and coverage tooling.
- Replaced manual browser `popstate` routing with React Router browser/memory routers while preserving `/`, `/graph`, `/deployment`, `/catalog`, legacy `?view=`, and graph/layout/scenario query parameters.
- Wrapped the app in `QueryClientProvider` and `I18nextProvider`; `TopBar` now consumes `react-i18next`.
- Converted graph and catalog server state to React Query. Graph filters, selected node, layout mode, and selected scenario remain local UI state.
- Added Vitest setup, provider render helper, a small a11y helper, API client tests, route/navigation/a11y tests, and ErrorBoundary coverage.

### Validation

- `pnpm strings:check` passed.
- `pnpm run lint` passed.
- `pnpm run type-check` passed.
- `pnpm test` passed.
- `pnpm test:coverage` passed; coverage is low because this is the first harness slice and thresholds are not yet enforced.
- `pnpm run build` passed with Vite's non-blocking chunk-size warning after adding router/query/i18n/test dependencies.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.

### Remaining UI Migration Work

- Broaden test coverage for deployment dashboard, catalog scan/apply states, graph error/empty states, and React Query mutation failures.
- Decide whether to add chunk splitting or a raised chunk warning limit during Phase 6/8; current production build is valid but emits Vite's default large-chunk warning.
- Phase 6 token and primitive cleanup remains open.

## 2026-06-14 Phase 6 Status Semantics And Coverage Slice

### Completed

- Added `ui/src/theme/status.ts` to centralize semantic status tone classes for success, warning, danger, neutral, and info states.
- Added the `warning` token to the Tailwind theme and replaced repeated raw status color classes in orientation, app-level error handling, catalog drift badges, and deployment readiness summaries.
- Added `docs/internal/EXPERIENCE-AUDIT.md` and registered it in `docs/manifest.json`.
- Expanded route tests to cover graph API failure rendering, catalog Scan & Apply mutation behavior, and deployment degraded states driven by metadata gaps and blockers.

### Remaining UI Migration Work

- Split `DeploymentDashboard` before broader design-system cleanup; it still combines fetching, status derivation, filtering, scan orchestration, and rendering.
- Add an accessible table/list alternate view for graph nodes and edges.
- Continue reducing remaining raw palette classes in feature components where they encode semantic status.

## 2026-06-14 Phase 6 Graph Alternate View And Status Model Slice

### Completed

- Added `ui/src/features/graph/GraphDataTable.tsx`, a keyboard-friendly node and edge table that mirrors the active graph filters and lets users select/clear nodes without using the SVG canvas.
- Extracted deployment readiness status derivation and metadata-gap aggregation into `ui/src/features/deployment/deploymentStatus.ts`.
- Added pure status-helper tests and route coverage for the graph data table selection path.
- `pnpm test:coverage` now reports roughly 64% statement coverage for the UI package, up from the initial low-coverage harness slice.

### Validation

- `pnpm strings:check` passed.
- `pnpm run lint` passed.
- `pnpm run type-check` passed.
- `pnpm test` passed.
- `pnpm test:coverage` passed.
- `pnpm run build` passed with the known non-blocking Vite large chunk warning.

### Remaining UI Migration Work

- `DeploymentDashboard` still owns fetching, filtering, scan/export actions, and rendering. Status derivation is now pure, but the dashboard should still be split into view components before deeper design-system cleanup.
- Continue replacing remaining raw semantic palette classes in deployment/catalog panels.
- Decide whether to split Vite chunks by route or explicitly raise the warning threshold during final cleanup.

## 2026-06-14 Phase 6 Deployment Dashboard Decomposition Slice

### Completed

- Split `DeploymentDashboard` rendering into feature-owned view components:
  - `DeploymentReadinessIntro.tsx` for the tier selector, readiness definition, and API warning.
  - `DeploymentSummaryCards.tsx` for readiness metrics.
  - `DeploymentStatusList.tsx` for searchable, keyboard-selectable status rows.
  - `DeploymentDetailsPanel.tsx` for selected-scenario fitness, blockers, requirements, metadata gaps, and follow-up actions.
  - `DeploymentStatusBadge.tsx` for reusable status and tier-fitness badges.
- Kept `DeploymentDashboard.tsx` as the orchestration boundary for deployment report loading, scan/export actions, filtering, selected scenario state, and status aggregation.
- Replaced the remaining raw semantic status palette classes in catalog priority badges, deployment recommendations/blockers, workflow completion icons, and the app error boundary with `statusTone(...)` semantics.
- Added route coverage that opens deployment details through the keyboard-friendly status list.

### Validation

- `pnpm strings:check` passed.
- `pnpm run lint` passed.
- `pnpm run type-check` passed.
- `pnpm test` passed.
- `pnpm test:coverage` passed; statement coverage remains roughly 64%.
- `pnpm run build` passed with the known non-blocking Vite large chunk warning.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- Raw semantic palette inventory over `ui/src/**/*.ts{x}` returned no matches for the tracked red/yellow/green/blue/slate/gray/amber/emerald/rose/orange status class families.

### Remaining UI Migration Work

- Decide whether to split Vite chunks by route or explicitly raise the warning threshold during final cleanup.

## 2026-06-14 Phase 6 Failure-State Coverage Slice

### Completed

- Fixed the route-level deployment scan handler to return the React Query mutation promise to `DeploymentDashboard`, allowing scan failures to reach the dashboard's existing `apiError` recovery message.
- Added route coverage for deployment scan failure messaging from the mutation path.
- Added route coverage for graph data table empty states when the shared graph filter hides all nodes and edges.
- `pnpm test:coverage` now reports roughly 70% statement coverage for the UI package.

### Validation

- `pnpm run type-check` passed.
- `pnpm test` passed.
- `pnpm run lint` passed.
- `pnpm test:coverage` passed.
- `pnpm run build` passed with the known non-blocking Vite large chunk warning.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.

### Remaining UI Migration Work

- Decide whether to split Vite chunks by route or explicitly raise the warning threshold during final cleanup.
- Consider focused coverage for DAG export failure/help behavior and recommended-flow documentation links before enforcing coverage thresholds.

## 2026-06-14 Phase 8 Runtime And Browser Smoke Cleanup

### Completed

- Reworded drift-prone derived counts in internal docs so docs health no longer reports content findings for SDA documentation.
- Made `/api/v1/health/analysis` return the standard lifecycle health envelope fields while preserving analysis diagnostics, closing the stale unhealthy runtime status.
- Aligned `ui/server.js` with the other scenario UI servers by defaulting `corsOrigins` to `*`, fixing browser-origin asset requests that failed only in smoke and Lighthouse phases.

### Validation

- `GOWORK=off go test ./...` passed in the API package.
- `golangci-lint run` passed in the API package.
- `knowledge-observatory docs health scenario-dependency-analyzer --json` reports `health_score: 1`.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `cli-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `proto-health validate scenario scenario-dependency-analyzer` passed with existing info-only reachability notes.
- `make build` passed with the known non-blocking Vite large chunk warning.
- `make test` passed.
- `pnpm run lint` passed in the UI package.
- `pnpm run type-check` passed in the UI package.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `vrooli scenario status scenario-dependency-analyzer --json` reports healthy.
- `vrooli scenario test scenario-dependency-analyzer` passed.

## 2026-06-14 Phase 7 Documentation Manifest Cleanup

### Completed

- Registered `docs/api.md`, `docs/cli.md`, `docs/integration.md`, and `docs/manifest.json` in the v2 documentation manifest so top-level compatibility references are no longer orphaned.
- Reworded README/API URL examples that were being parsed as unknown machine-readable references, keeping lifecycle-assigned `API_PORT`/`UI_PORT` as the source of truth.
- Removed stale "planned" labels from implemented `graph actual` and `drift` CLI/API documentation.

### Validation

- `knowledge-observatory docs health scenario-dependency-analyzer --json` reports `health_score: 1`.
- `knowledge-observatory docs audit scenario-dependency-analyzer --json` no longer reports orphaned docs, extra docs, or unknown marked references.

## 2026-06-14 Phase 3 API Adapter Slice

### Completed

- Added `api/internal/catalog` as the scenario catalog REST adapter for `GET /api/v1/scenarios` and `GET /api/v1/scenarios/:scenario`.
- Routed catalog list/detail registration through the domain adapter while preserving the existing app-backed `ScenarioService` implementation.
- Moved deployment readiness, DAG export, and bundle manifest REST presentation into `api/internal/deployment`.
- Added focused adapter tests for catalog listing/error status mapping and deployment report, DAG flattening, unsupported DAG format, bundle manifest, and reporter error behavior.

### Validation

- `GOWORK=off go test ./...` passed in the API package.

### Remaining API Migration Work

- Extract the catalog service implementation out of `api/internal/app` when the backing workspace/store dependencies can move cleanly.
- Continue migrating REST adapter groups one domain at a time; graph and dependency impact routes are the next plausible candidates.

## 2026-06-14 Phase 3 Graph Adapter Slice

### Completed

- Added `api/internal/graph` as the graph REST compatibility adapter for legacy dependency graph, centrality, cycle detection, actual interface graph JSON, and drift routes.
- Moved graph centrality analytics and cycle detection out of `api/internal/app` into the graph domain package.
- Updated app route registration so graph routes mount through the graph adapter while existing app services still provide graph generation.
- Added focused graph adapter tests and moved centrality tests to the graph package.

### Validation

- `GOWORK=off go test ./...` passed in the API package.

### Remaining API Migration Work

- The Connect graph handler now lives in `api/internal/graph/connect.go`; app startup only wires the graph-owned generated contract route.

## 2026-06-14 Phase 3 Remaining REST Adapter Slice

### Completed

- Added `api/internal/analysis` as the REST adapter for scenario analysis and scan/apply routes.
- Added `api/internal/dependencies` as the REST adapter for stored dependency reads and dependency impact reports.
- Added `api/internal/proposal`, `api/internal/optimization`, and `api/internal/coreset` adapters for their matching REST compatibility routes.
- Reduced `api/internal/app/handlers.go` to app handler construction, service accessors, scenario-dir resolution, and analysis health; production and test routers now mount the moved routes through domain packages.
- Added focused adapter tests for analysis, scan, dependencies, impact, proposal, optimization, and core-set routes.

### Validation

- `GOWORK=off go test ./...` passed in the API package.

### Remaining API Migration Work

- Backing service implementations for analysis, scan, dependencies, optimization, proposal, and scenarios still live in the app registry. Extract those only when their workspace/store/runtime dependencies can move cleanly.
- The Connect graph handler now lives in `api/internal/graph/connect.go`; remaining API migration debt is backing service implementation ownership, not graph presentation.

## 2026-06-15 Phase 3 Graph Connect Presentation Slice

### Completed

- Moved generated `InterfaceGraphService` Connect presentation from `api/internal/app` into `api/internal/graph`.
- Kept app startup responsible only for route wiring through `graph.RegisterConnectRoutes`.
- Added graph-domain test coverage for interface graph to proto response mapping.

### Validation

- `GOWORK=off go test ./...` passed in the API package.
- `golangci-lint run` passed in the API package.
- `knowledge-observatory docs health scenario-dependency-analyzer --json` reports `health_score: 1`.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `cli-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `proto-health validate scenario scenario-dependency-analyzer` passed with existing info-only reachability notes.
- `make build` passed with the known non-blocking Vite large chunk warning.
- `vrooli scenario status scenario-dependency-analyzer --json` reports healthy.
- `make test` is blocked by the external Test Genie execute parser regression filed as `knw-1781498226583665032`; it treats the scenario name as a phase before SDA phases execute.

### Remaining API Migration Work

- Backing service implementations for scenario catalog, graph generation, scan, dependencies, optimization, and proposal still live in the app package. Extract these only when their workspace/store/runtime dependencies can move cleanly.

## 2026-06-15 Phase 3 App Service Boundary Slice

### Completed

- Split app-backed service implementations out of `api/internal/app/service_registry.go` into focused files:
  - `service_analysis.go`
  - `service_scan.go`
  - `service_graph.go`
  - `service_optimization.go`
  - `service_scenario.go`
  - `service_deployment.go`
  - `service_proposal.go`
- Kept `service_registry.go` as the composition point for wiring the `services.Registry`.
- Preserved existing app-owned implementation boundaries; this slice improves discoverability without moving workspace/store/runtime coupling prematurely.

### Validation

- `GOWORK=off go test ./...` passed in the API package.
- `golangci-lint run` passed in the API package.
- `knowledge-observatory docs health scenario-dependency-analyzer --json` reports `health_score: 1`.
- `git diff --check -- scenarios/scenario-dependency-analyzer` passed.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `cli-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `proto-health validate scenario scenario-dependency-analyzer` passed with existing info-only reachability notes.
- `vrooli scenario status scenario-dependency-analyzer --json` reports healthy.
- `make build` passed and confirmed split UI chunks without the previous Vite warning.
- `make test` passed after rebuild; no stale SDA binary warning remained on the final run.

### Remaining API Migration Work

- The backing services still live in package `app`; move them into domain packages only when their dependencies can move cleanly or shrink behind typed ports.

## 2026-06-15 UI Route Chunking Slice

### Completed

- Lazy-loaded the orientation, graph, deployment, and catalog route surfaces from `ui/src/app/routes.tsx`.
- Kept shared route state, search-param synchronization, React Query hooks, and app shell ownership unchanged so user-visible behavior remains stable.
- Added an in-shell loading state for lazy route transitions.

### Validation

- `pnpm run type-check` passed in the UI package.
- `pnpm run lint` passed in the UI package.
- `pnpm test` passed in the UI package.
- `pnpm run build` passed and now emits split route chunks; the main JS chunk is below Vite's default warning threshold, so the previous large-chunk warning is gone.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `cli-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `proto-health validate scenario scenario-dependency-analyzer` passed with existing info-only reachability notes.
- `GOWORK=off go test ./...` passed in the API package.
- `make build` passed and confirmed the split chunk output without the previous Vite warning.
- `make test` passed; the earlier Test Genie parser blocker did not reproduce on this run.

## 2026-06-15 Graph Builder Ownership And Template Provenance Slice

### Completed

- Moved dependency graph construction from `api/internal/app/graph.go` into `api/internal/graph/builder.go`.
- Kept `api/internal/app` responsible for analyzer composition and legacy fallback wiring, while graph node/edge construction, stale scenario filtering, metadata, and complexity scoring are graph-domain-owned.
- Added graph-domain builder tests for graph-type filtering, stale scenario filtering, store error propagation, deterministic IDs, and normalized complexity scoring.
- Reassessed React-Vite template obligations after the UI/router/test/i18n/PWA/docs/health work and bumped `.vrooli/service.json::generation.template.version` to `1.1.0` with the current template manifest/content hashes.

### Validation

- `GOWORK=off go test ./...` passed in the API package.
- `ui-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `cli-health validate scenario scenario-dependency-analyzer` passed with no findings.
- `vrooli scenario template drift scenario-dependency-analyzer --json` reports an ok status.

### Remaining API Migration Work

- Backing services for deployment, proposal, optimization, analysis/scan, dependencies, and scenario catalog still live in package `app`; move them only when their workspace/store/runtime dependencies can be isolated behind clean ports.
