# Problems and Solutions - Scenario Dependency Analyzer

## Problems Discovered (2025-09-28)

### 1. CLI-API Graph Type Mismatch
**Problem**: The CLI expected graph types like "hierarchical", "network", "circular" but the API expected "resource", "scenario", "combined".
**Solution**: Fixed the CLI to use the correct API types and added mapping for user-friendly aliases.
**Status**: ✅ Fixed

### 2. Missing Optimize Command
**Problem**: The optimize command was specified in the PRD as P0 but was not implemented in the CLI.
**Solution**: Added the optimize command with basic functionality and a roadmap for full implementation.
**Status**: ✅ Implemented (basic version)

### 3. Database Initialization
**Problem**: The database schema exists but may not be properly initialized when the scenario starts.
**Solution**: The schema.sql is present but requires proper lifecycle integration for auto-initialization.
**Status**: ⚠️ Needs verification

### 4. AI Resource Integration
**Problem**: Qdrant-based semantic matching may fail silently if resources aren't running.
**Solution**: Added fallback heuristics when semantic resources are unavailable.
**Status**: ✅ Fallbacks implemented

### 5. Test Failures
**Problem**: The `make test` command fails at the API health check step.
**Solution**: The health endpoint works when tested directly, suggesting a timing or port issue in the test script.
**Status**: 🔍 Needs investigation

## Technical Debt

1. **Qdrant Integration**: Currently uses exec commands to call resource-qdrant, should use proper API client
2. **Optimization Engine**: Currently returns placeholder data, needs full implementation
3. **Circular Dependency Detection**: Graph algorithms exist in code but not fully integrated
4. **Historical Tracking**: Database tables exist but no automatic tracking implemented

## React-Vite 1.1 Migration Baseline (2026-06-14)

### Template Provenance Missing
**Problem**: `.vrooli/service.json` did not declare `generation.template.id`, so `ui-health validate scenario scenario-dependency-analyzer` skipped template-aware validation.
**Status**: Fixed. Provenance now declares React-Vite template version `1.1.0` with the current template manifest/content hashes. `vrooli scenario template drift scenario-dependency-analyzer --json` reports an ok status. Residual SDA-specific debts below remain tracked separately and do not represent template hash drift.

### CLI Manifest Missing
**Problem**: `cli-health validate scenario scenario-dependency-analyzer` reports `manifest.missing` for `cli/manifest.json`.
**Status**: Fixed for governance coverage. `cli/manifest.json` now declares the Connect-backed `graph actual` surface and `cli-health validate scenario scenario-dependency-analyzer` passes. Runtime command registration remains code-backed until a later manifest-loaded CLI migration.

### UI Architecture Drift
**Problem**: The UI still uses a monolithic `App.tsx` tab shell and custom server-state hooks rather than the template's providers, router, layout, React Query, selectors, i18n, theme, and test harness architecture.
**Status**: Partially fixed. The UI now has template slot layout scaffolding, React Router path navigation, selectors, generated strings scaffolding, theme/test/api/i18n slots, an ErrorBoundary, concrete feature folders for graph/deployment/catalog/governance, React Query-backed graph/catalog/governance server state, a generated proto/Connect Governance client, `react-i18next` provider usage, and a Vitest/Testing Library/a11y harness. Remaining UI work is broader test coverage and Phase 6 design-system/token cleanup.

### UI Test Coverage Floor
**Problem**: `pnpm test:coverage` now runs, but coverage is low because only the first routing/API/ErrorBoundary/a11y harness slice exists.
**Status**: Partially improved. Route tests now cover graph API failure rendering, graph table node selection and empty-filter states, catalog Scan & Apply mutation behavior, deployment degraded states from metadata gaps/blockers, and deployment scan failure messaging. Pure deployment status-helper tests cover critical/blocked/gap aggregation semantics. DAG export fallback/help behavior and deeper component coverage remain open before enforcing coverage thresholds.

### UI Bundle Size Warning
**Problem**: `pnpm run build` passes, but Vite reports the main minified JS chunk is above its default 500 kB warning threshold after adding router/query/i18n/test-adjacent runtime dependencies.
**Status**: Fixed. Orientation, graph, deployment, and catalog route surfaces are now lazy-loaded from `ui/src/app/routes.tsx`, so the production build emits route chunks and the main JS chunk stays below Vite's default warning threshold without raising the limit.

### UI Server CORS Blocked Browser Smoke Assets
**Problem**: Browser-driven smoke and Lighthouse phases saw `403 Origin not allowed` for built JS/CSS assets because SDA passed an empty default CORS allow-list to `createScenarioServer`. Direct curl checks without an `Origin` header returned `200 OK`, which masked the issue outside browser execution.
**Status**: Fixed. `ui/server.js` now defaults `corsOrigins` to `*`, matching the other lifecycle-managed scenario UI servers. Origin-bearing asset requests return `200 OK`, and `vrooli scenario test scenario-dependency-analyzer` passes.

### Documentation Manifest Drift
**Problem**: `knowledge-observatory docs audit scenario-dependency-analyzer --json` reported top-level `docs/api.md`, `docs/cli.md`, and `docs/integration.md` as orphaned docs, and localhost examples in prose were parsed as unknown machine-readable references.
**Status**: Fixed. The legacy detailed API/CLI references, integration guide, and manifest are now registered in `docs/manifest.json`; URL prose was reworded to lifecycle-assigned port guidance; `knowledge-observatory docs health scenario-dependency-analyzer --json` now reports `health_score: 1`.

### UI Design-System Debt
**Problem**: Feature components still contain raw status and palette utility classes, especially in deployment readiness and graph panels.
**Status**: Improved. Semantic status tones now live in `ui/src/theme/status.ts` and cover the repeated readiness/drift/error states. Deployment status derivation has been extracted into pure helpers, and deployment readiness rendering is now split across intro, summary, status-list, status-badge, and details components. A raw semantic palette inventory over `ui/src/**/*.ts{x}` no longer finds the tracked red/yellow/green/blue/slate/gray/amber/emerald/rose/orange status class families. Remaining design-system work is mostly targeted coverage before coverage thresholds are enforced.

### Measures Coverage
**Problem**: `vrooli scenario test scenario-dependency-analyzer` failed in `phase-measures` because `scenario_dependencies` was a stateful domain with no declared measure or waiver.
**Status**: Fixed for this migration phase. `measures-health validate scenario scenario-dependency-analyzer` now passes with explicit waivers for `scenario_dependencies`, `scenario_metadata`, and `optimization_recommendations`. The `scenario_dependencies` waiver is intentional because the current graph RPC returns a nested interface graph, not a measure-shaped scalar/table; remove the waiver when a dedicated Count/ListScenarioDependencies Connect RPC exists.

### Runtime Status Drift
**Problem**: `vrooli scenario status scenario-dependency-analyzer` reports unhealthy even though live `/health`, `/api/v1/health/analysis`, and UI `/health` endpoints return `200 OK`.
**Status**: Fixed. The runtime health probe was rejecting `/api/v1/health/analysis` because it returned analysis diagnostics without the standard health envelope fields. The endpoint now preserves its diagnostic fields while also returning `service`, `timestamp`, `readiness`, and `version`; after `make restart`, `vrooli scenario status scenario-dependency-analyzer --json` reports `health_status: healthy`.

### API Architecture Convergence
**Problem**: `api/internal/app` still owns the central Gin router, service registry, and multiple REST adapters, so some domains remain less discoverable than the React-Vite template's screaming-architecture target.
**Status**: Partially improved. Scenario catalog list/detail REST presentation now lives in `api/internal/catalog` with its own adapter tests and a tiny local service contract. Deployment readiness, DAG export, and bundle manifest REST presentation now live in `api/internal/deployment` with adapter coverage. Graph REST presentation, centrality analytics, cycle detection, generated Connect graph presentation, and dependency graph construction now live in `api/internal/graph` with adapter/domain tests. Analysis/scan REST presentation lives in `api/internal/analysis`; stored dependencies and impact routes live in `api/internal/dependencies`; proposal, optimization, and core-set presentation live in their matching domain packages. `api/internal/app/service_registry.go` is now only service composition; focused app service implementations live in `service_*.go` files until their workspace/store/runtime dependencies can move cleanly into domain packages.

### Test Genie Execute Parser Regression
**Problem**: On 2026-06-15, `make test` reached SDA's test phase but failed before executing phases because `test-genie execute scenario-dependency-analyzer --preset comprehensive` treated `scenario-dependency-analyzer` as a phase name. Alternate flag ordering treated `quick` as a phase, and `execute --help` advertised `--dry-run` while the parser rejected it.
**Status**: No longer blocking SDA validation. The external issue remains filed as `knw-1781498226583665032` for Test Genie ownership, but `make test` completed successfully on 2026-06-15 after the UI route-chunking slice.

### Proto Reachability Notes
**Problem**: `proto-health validate scenario scenario-dependency-analyzer` emits informational `proto.possibly_unused` findings for graph and health messages because the validator does not yet see all served/consumed reachability.
**Status**: Non-blocking. Keep graph contracts because SDA is the fleet interface graph authority; close reachability evidence during Phase 2.

### Dependency Governance Self-Review
**Problem**: Approved dependency governance originally had no review memory for many direct SDA dependencies, which made SDA fail its own dependency-health target with advisory warnings.
**Status**: Improved on 2026-06-17. `scenario-dependency-analyzer deps approved validate scenario-dependency-analyzer --json` now passes with no findings after package-by-package registry review for SDA's direct Go, UI runtime, generated local, graph/i18n, Radix, accessibility-test, lint, and build-tool dependencies. The remaining governance debt is model-level: the registry currently keys one record per ecosystem/package, so packages with multiple valid major lines across the fleet need broad `minimum` policies or a future multi-record/scope-aware model to express parallel approved ranges without weakening review intent.

## Recommendations for Next Iteration

1. **Priority**: Implement proper resource client libraries instead of exec calls
2. **Database**: Add automatic migration runner on startup
3. **Testing**: Fix the test suite to properly wait for API readiness
4. **UI**: Add WebSocket support for real-time updates during analysis
5. **Performance**: Add Redis caching layer for frequently accessed dependency data

## Actual Interface Graph Migration (2026-06-13)

### Declared Dependencies Are Not Actual Interface Usage
**Problem**: SDA historically reported declared dependencies from `service.json` plus local scanner heuristics. That cannot answer whether a scenario actually imports another scenario's protobuf or generated Go client, and it cannot reliably flag `service.json` drift.

**Planned solution**: Consume batch facts from `proto-health` and `code-facts`, attribute import paths to scenario slugs, and expose an on-demand evidence-tagged interface graph.

**Status**: Planned by `scenario-dependency-analyzer-actual-interface-graph-and-import-drift`.

### Scanner Ownership Boundary
**Problem**: SDA's scanner owns too much language-specific evidence logic. `detectPortCalls` follows obsolete `resolveScenarioPortViaCLI` usage, and `detectCLIReferences` is superseded by Connect/proto imports. Retaining this style would duplicate code-facts and keep dependency evidence brittle.

**Planned solution**: Delete the obsolete/superseded detectors during the graph migration. Keep only interim non-import signals with precise comments until upstream AST facts exist.

**Status**: Planned cleanup.

### Follow-Up: AST Facts for Runtime and CLI Usage
**Problem**: Runtime `ResolveScenarioURL*` calls and `vrooli scenario run` shell-outs are real cross-scenario usage signals, but they are not import-level evidence and should not be rebuilt as regex scanners inside SDA.

**Follow-up plan**: `scenario-dependency-analyzer-code-evidence-via-ast-facts`

**Scope**: Add AST analyzers in `go-code-graph` and `typescript-code-graph` for modern discovery calls and scenario shell-outs, surface those through `code-facts`, then delete SDA's retained regex detectors and delegate resource-usage detection to fact providers.
