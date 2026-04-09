# Research Conclusion: Command-Center Scenario Architecture

## Research Question
What architecture, technology stack, data connectivity patterns, theming approach, and kiosk UX strategy should the command-center scenario use to deliver a collection of full-screen, visually stunning war-room dashboards designed for always-on TV/kiosk display?

## Summary
The command-center scenario follows Vrooli's established scaffold (Go API + React+Vite UI + service.json lifecycle) with a read-only aggregation API serving pre-composed dashboard payloads. It uses React Three Fiber (R3F) for immersive 3D background effects and Recharts for data charts. Each of the 6 dashboard pages gets a fully independent visual theme via CSS custom properties plus per-page R3F scenes. The API maintains a file-based JSON metric registry tracking data source status (live/gap/partial) to power the /api/v1/gaps endpoint. LPBS data is accessed via new dedicated REST endpoints using the existing inter-scenario service token auth pattern. Caching uses per-source TTLs (Swarm 30s, Vrooli 60s, LPBS 5min).

## Methodology
1. Examined existing Vrooli scenario structures (swarm-manager, landing-page-business-suite) for scaffold patterns
2. Analyzed swarm-manager stats/overview APIs for data availability and handler patterns
3. Inventoried LPBS Postgres schema for subscription/payment/analytics data availability
4. Reviewed Vrooli core scenarios API (port 8092) for health/metadata endpoints
5. Evaluated visualization libraries against Xbox Edge browser WebGL2 constraints
6. Reviewed initiative orchestration summary and sibling execute items for settled decisions
7. Studied prompt-manager R3F integration patterns (WorldCanvas, LOD system, performance monitoring, asset disposal)
8. Analyzed LPBS inter-scenario auth middleware (requireAdminOrService) for data access patterns
9. Reviewed storage-steer skill for file-based configuration patterns applicable to gap registry

## Findings

### Finding 1: Scenario Scaffold Pattern Is Well-Established
Both swarm-manager and LPBS follow an identical structure: `api/` (Go + gorilla/mux + `internal/` domain packages), `ui/` (React+Vite+TypeScript+Tailwind+React Router v6+React Query), `.vrooli/service.json` (v2.0.0 lifecycle), and `Makefile`. The command-center should replicate this exactly. Port allocation uses env vars within defined ranges (API: 15000-19999, UI: 35000-39999). The Go API uses a handler factory pattern with `RegisterRoutes()` methods, service layer separation, and centralized error handling. The UI uses interface-based API services for testability and `base: './'` in Vite config for proxy compatibility.

### Finding 2: Upstream Data Sources Are Rich But Varied
**Swarm Manager** (REST): `/api/v1/stats` returns throughput, timing, scope, blocking, agent, dashboard, and review categories. `/api/v1/overview` returns all items, initiatives with rollup, dependency graph, and governance. This maps directly to The Forge (~90% live) and Mission Control (~70% live).

**LPBS** (Postgres + soon REST): Tables include subscriptions (plan tiers, status, churn), checkout_sessions (revenue), credit_wallets/transactions (consumption), usage_records, metrics_events (visitors, conversions, A/B variants by time range), users (registration trends), and download catalog. LPBS MetricsService already provides `GetAnalyticsSummary()` and `GetVariantStats()` with time range filtering — these can back new dashboard endpoints. Maps to Ledger (~60% live) and Broadcast (~40% live).

**Vrooli Core** (REST, port 8092): `GET /scenarios` returns scenario metadata, status, health, ports, runtime. Completeness scoring available via CLI. Maps to The Hive (~80% live) and Mission Control.

### Finding 3: R3F + Recharts Selected as Visualization Stack
**Settled in round 1.** React Three Fiber for immersive 3D background effects (particle systems, force-directed graphs, nebula effects, bioluminescent animations) plus Recharts for data charts. R3F is already proven in the prompt-manager scenario with a mature pattern: tier-aware quality (4 tiers from basic to ultra), adaptive DPR, LOD system, asset disposal hooks, shared geometry/material caching, and performance auto-monitoring. Key reference files: `prompt-manager/ui/src/components/world/WorldCanvas.tsx`, `PerformanceMonitor.tsx`, `useAssetDisposal.ts`.

**Xbox compatibility**: R3F works within WebGL2 limits. The performance tier system can auto-downgrade to basic materials and disable shadows/post-processing on Xbox's ~2GB memory budget. Canvas2D fallback for particle effects if WebGL is too heavy.

### Finding 4: LPBS Data Access via New REST Endpoints with Service Token Auth
**Settled in round 1.** Command-center will NOT query LPBS Postgres directly. Instead, new REST endpoints will be added to LPBS specifically for inter-scenario data consumption. LPBS already has a `requireAdminOrService` middleware (auth.go:404) that validates Bearer tokens — this is the established inter-scenario auth pattern used by scenario-to-desktop. The command-center API authenticates with LPBS using a service bearer token, no new auth infrastructure needed. This creates a clean API boundary where LPBS owns its data contract.

### Finding 5: CSS Theme Isolation via Custom Properties
For fully independent per-page themes, CSS custom properties scoped to route wrappers is recommended. Each page wrapped in a `<div data-theme="ground-control">` with all theme tokens as CSS variables. Theme definitions in JSON config files, applied via a ThemeProvider. This approach is the most flexible, has no CSS-in-JS runtime cost, and aligns with the data-driven theme config requirement in the theming-engine execute item. Each theme defines: background treatment, color palette, typography, animation style, chart colors, card/panel styling, and glow/shadow effects.

### Finding 6: API-Side File-Based Metric Registry for Gap Tracking
**Settled in round 1.** Gap metadata lives in JSON config files loaded by the API at startup, not hard-coded Go structs. Following the storage-steer pattern, this means the metric definitions can be updated without recompiling. Each metric entry includes: `id`, `dashboard`, `label`, `dataSource` (enum: live/gap/partial), `upstreamSource`, `description`, and what's needed to make it live. The `/api/v1/gaps` endpoint reads the registry and returns all gap/partial metrics grouped by dashboard.

### Finding 7: Per-Source Cache TTLs
**Settled in round 1.** Caching uses different TTLs per upstream source matching data volatility: Swarm Manager 30s (changes with every backlog operation), Vrooli Core 60s (changes on start/stop events), LPBS 5min (meaningful analytics deltas are hourly). Graceful degradation: if a source is down, return last cached data with staleness timestamp.

### Finding 8: Xbox Edge Browser Capabilities Confirmed
Xbox Edge (Chromium-based) supports: WebGL 2.0, CSS animations/transforms, requestAnimationFrame, Fullscreen API, Screen Wake Lock API, Canvas 2D, SVG. Not supported: WebGPU, mouse cursor (gamepad/D-pad only). Constraints: 1080p default (4K on Series X), ~60fps target, ~2GB memory budget. The R3F performance tier system provides the needed auto-degradation path for Xbox.

## Limitations
- Xbox Edge browser testing not performed directly; findings based on documented Chromium capabilities. Real device testing needed during implementation.
- R3F performance on Xbox with 6 different themed scenes is theoretical — benchmarking with actual dashboard payloads needed.
- LPBS endpoint design is at high level; specific endpoint contracts (request/response shapes) need design during the data-aggregation-design research item.
- Caching TTL recommendations are starting points; real-world tuning will be needed.
- Post-processing effects (Bloom, Vignette) are currently disabled in prompt-manager's R3F setup due to compatibility issues — the command-center may hit the same issues.

## Actions

### Action 1: Proceed to scaffold execution
The research has settled all foundational architecture decisions. The `execute/command-center-scenario-scaffold` item can proceed with confidence using: Go API + React+Vite UI, R3F + Recharts, CSS custom property theming, file-based gap registry, service token auth for LPBS, per-source TTL caching.

### Action 2: Design LPBS dashboard endpoints
The `research/command-center-data-aggregation-design` item (dependency of the API aggregator execute item) should design the specific LPBS REST endpoint contracts. Key input: LPBS already has `GetAnalyticsSummary()` and `GetVariantStats()` that can back dashboard endpoints.

### Action 3: R3F architecture and transition decisions pending
Round 2 raised three open decisions that affect implementation: per-page Canvas vs single Canvas, LPBS endpoint grouping strategy, and page transition animation style. These should be resolved before the theming-engine and kiosk-ux execute items begin.
