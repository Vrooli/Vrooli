# Experience Audit

## 2026-06-14 React-Vite 1.1 Migration Slice

### Audience And Primary Jobs

SDA serves operators and future agents who need to understand which scenarios/resources are coupled, which dependencies are missing or stale, and whether a scenario is ready for a target deployment tier. The UI should feel like an operational console: dense, scannable, and biased toward concrete next actions.

### Current Strengths

- The app now opens directly into a usable workspace rather than a landing page.
- Navigation has stable routes for orientation, graph, deployment, and catalog surfaces.
- Graph, catalog, and deployment features own their route-level components under `ui/src/features`.
- React Query now owns graph/catalog server state and the test harness covers route navigation, API seams, an error boundary, and baseline a11y.

### Friction Found

- Status colors were repeated as raw green/amber/red utility classes across orientation, catalog drift, and deployment readiness surfaces.
- Deployment readiness has many high-value controls, but it still mixes status styling, row behavior, and scan orchestration in one large component.
- Graph-heavy information has a keyboard note and node focus behavior, but the alternate data view is still limited to telemetry and selected-node panels.
- Coverage exists, but the initial harness did not yet protect graph error, catalog scan/apply, or deployment degraded states.

### Changes In This Slice

- Added `ui/src/theme/status.ts` as the semantic source for success, warning, danger, neutral, and info status classes.
- Added the `warning` color token to the Tailwind theme so warning styles are token-backed instead of raw palette-backed.
- Replaced the most repeated raw readiness/drift/error status classes with semantic status tones.
- Added route-level tests for graph API failure, catalog Scan & Apply mutation behavior, and deployment critical metadata/blocker rendering.

### Remaining Work

- Split `DeploymentDashboard` into smaller view/state subcomponents before deeper visual cleanup.
- Continue replacing remaining raw palette classes where they are semantic statuses rather than one-off layout details.
- Route-level chunking is now implemented; keep watching chunk growth as new surfaces are added.

## 2026-06-14 Graph Data Accessibility Slice

### Changes

- Added a tabular graph data view under the SVG canvas so users can inspect nodes and edges without spatial graph interaction.
- The node table follows the same search filter as the graph view and exposes a normal button for selecting or clearing the inspected node.
- The edge table follows the active drift filter and keeps the relationship, weight, and required flag scannable.
- Deployment readiness status derivation moved into a pure helper, reducing the amount of decision logic embedded in the large dashboard render path.

### Remaining Work

- Graph filter/table empty states and deployment scan failure messaging now have route-level regression coverage.
- Continue semantic token cleanup only if new raw palette debt appears; the tracked status color families are currently clean in UI TS/TSX.

## 2026-06-14 Deployment Readiness Decomposition Slice

### Changes

- Split deployment readiness into separate view components for the intro/tier selector, summary metrics, status list, status badges, and selected-scenario detail panel.
- Preserved the existing scan/export/load orchestration inside `DeploymentDashboard`, so user-visible behavior stays stable while rendering ownership is clearer.
- Made the status list explicitly keyboard-selectable and covered detail expansion through route tests.
- Replaced remaining raw semantic status palette classes in deployment/catalog/error surfaces with `statusTone(...)` classes.

### Remaining Work

- Route-level code splitting is now in place; keep future heavy surfaces behind lazy route/module boundaries.
- Consider targeted coverage for DAG export fallback/help behavior before enforcing UI coverage thresholds.

## 2026-06-14 Failure-State Coverage Slice

### Changes

- Deployment scan failures now propagate from the route-level catalog mutation into the deployment dashboard, so the existing operator-facing API warning appears when scan/apply fails.
- Added regression coverage for that scan failure path and for graph table empty states when a filter excludes all visible rows.
- UI coverage improved to roughly 70% statements while keeping coverage focused on user-visible states.

### Remaining Work

- The large-chunk build warning is closed by lazy-loading route surfaces.
- DAG export fallback/help behavior is still a candidate for targeted coverage before coverage thresholds are enforced.

## 2026-06-14 Runtime Smoke Cleanup

### Changes

- Standardized the analysis health endpoint so lifecycle status and the live health endpoint agree for operators.
- Fixed the UI server's default CORS policy so browser-origin requests can load built JS/CSS assets during smoke and Lighthouse runs.

### Remaining Work

- Keep route surfaces lazy-loaded when adding future heavy graph or deployment modules.
