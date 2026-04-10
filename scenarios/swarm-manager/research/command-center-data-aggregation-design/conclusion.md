# Research Conclusion: Data Aggregation API Architecture for Command-Center

## Research Question
How should command-center's Go API aggregate data from LPBS (subscriptions, payments, analytics), Swarm Manager (velocity, backlog, agent stats), and Vrooli Core (scenario metadata/health) into dashboard-ready payloads — including data access patterns, derived metric computation, caching, and gap metadata handling?

## Summary
Command-center consumes all upstream data via REST APIs (not direct DB access). LPBS needs ~3 new admin summary endpoints for revenue/subscription/user-growth aggregation, with derived metrics (MRR, churn, ARPU) computed by LPBS as the data owner. Command-center authenticates to LPBS via Bearer token using the existing `requireAdminOrService()` middleware pattern — existing admin endpoints need a one-line middleware swap. An in-memory TTL cache (30s LPBS/Swarm Manager, 60s Vrooli Core) serves stale data on source failure. A static code-level gap registry tags every metric as live/gap/partial, exposed via GET /api/v1/gaps. Per-dashboard endpoints (GET /api/v1/dashboards/{page}) compose data from all sources into page-ready payloads. Concurrent fan-out via `errgroup` with per-source `context.WithTimeout` (5s each) ensures one slow source doesn't block the entire response — timed-out sources fall back to cache with per-source staleness metadata in every response. The codebase uses a flat `package main` structure (matching LPBS) with service-per-file organization.

## Methodology
1. Explored LPBS database schema (schema.sql) and existing API endpoints (routes.go, handlers)
2. Explored Swarm Manager stats engine (event-sourced watermark pattern) and overview service
3. Explored Vrooli Core API (port 8092) and scenario-completeness-scoring service
4. Reviewed initiative orchestration summary for architectural constraints
5. Analyzed existing aggregation patterns across all three sources
6. Investigated LPBS auth middleware (`requireAdmin`, `requireAdminOrService`, `requireServiceAuth`) and Bearer token infrastructure
7. Documented full API response shapes for all upstream endpoints command-center will consume
8. Surveyed existing in-memory cache implementations in the codebase (agent-manager pricing service, stats orchestration)
9. Surveyed concurrent fan-out patterns (errgroup in git-control-tower, scenario-stack-governor; WaitGroup in agent-manager stats)
10. Examined swarm-manager circuit breaker implementation for adaptation to per-source health tracking

## Findings

### Finding 1: LPBS Has Rich Existing APIs but Gaps in Revenue Aggregation
LPBS already exposes admin endpoints for metrics (visitor/conversion analytics via GET /api/v1/metrics/summary → `AnalyticsSummary` with `VariantStats` array), usage (per-period aggregation via GET /api/v1/admin/usage → `UsageRecord` array with user_totals/app_totals maps), users (enriched paginated list via GET /api/v1/admin/users → `UsersListResponse` with `SubscriptionInfo` and `CreditInfo` per user), and coupon stats (GET /api/v1/admin/coupons/usage → `CouponUsageStats` array).

**Missing LPBS endpoints needed:**
- **GET /api/v1/admin/revenue-summary**: MRR, ARR, revenue by session_type, churn rate, ARPU — requires joining subscriptions.price_id against PlanStore (plans.json). Monthly amounts: solo=$29, pro=$79, studio=$199, business=$499. Yearly: solo=$239, pro=$648, studio=$1668, business=$4188. MRR for yearly = amount/12.
- **GET /api/v1/admin/subscription-stats**: Counts by tier × status (active/trialing/past_due/canceled), with trend data over time windows.
- **GET /api/v1/admin/user-growth**: Registration trends, active user counts over 7/30/90-day windows.

### Finding 2: Swarm Manager Data Is Immediately Consumable
Both /api/v1/stats and /api/v1/overview are unauthenticated (fail-open identity middleware) and return well-structured JSON. Stats uses event-sourced in-memory aggregation with watermark-based incremental refresh — responses are fast and pre-aggregated. Stats supports 7 category filters: throughput, timing, scope, blocking, agent, dashboard, review. Overview does a full load per request returning all backlog items, initiatives with rollup, dependency graph, summary (items by status/kind), and governance status.

**Memory consideration**: Overview responses could be 10-50KB+ for a mature backlog. Cache should store raw JSON bytes rather than deserialized structs to minimize GC pressure. Use /stats for most dashboard views and reserve /overview for the Forge page only.

### Finding 3: Vrooli Core Requires Two Service Calls for Complete Scenario Data
Scenario listing (GET /scenarios on port 8092) provides name, display_name, description, tags, status, processes, ports, runtime, health_status, started_at, and system_warnings. Completeness scoring lives in a separate service (scenario-completeness-scoring) with GET /api/v1/scores returning scenario scores with built-in degradation handling (partial results with confidence scores, skipped scenarios with error reasons). Command-center needs to fan out to both services and merge results by scenario name. Both are unauthenticated.

### Finding 4: Gap Tracking Should Be Code-Level, Not Runtime Config
The gap registry should be a static Go map in command-center's codebase, mapping metric names to source status (live/gap/partial). When a data pipeline is built (e.g., LPBS adds a revenue endpoint), the registry entry is updated from "gap" to "live" as a code change. GET /api/v1/gaps serves the filtered registry. This creates the recursive feedback loop: Director Swarm reads /api/v1/gaps → proposes backlog items → team builds pipeline → code change flips gap to live.

### Finding 5: LPBS Service-to-Service Auth Already Exists
LPBS has three auth middleware tiers:
- `requireAdmin()` — session cookie only (browser clients)
- `requireAdminOrService()` — session cookie OR Bearer token via `LPBS_SERVICE_SECRET` env var / `.vrooli/secrets.json` (used on remote-profiles and download-artifacts endpoints)
- `requireServiceAuth()` — Bearer token only (used on usage/report endpoint)

Command-center authenticates using the existing Bearer token pattern. The `LPBS_SERVICE_SECRET` is validated with constant-time comparison. Currently only ~2 endpoint groups use `requireAdminOrService()` — the remaining admin endpoints command-center needs would need their middleware swapped from `requireAdmin()` to `requireAdminOrService()`, which is a one-line change per route in LPBS routes.go.

### Finding 6: Completeness-Scoring Has Built-In Graceful Degradation
The /api/v1/scores response includes a `degradation` object with skipped scenarios (with error/reason), partial results (with confidence and missing_collectors), and an is_complete flag. Command-center can pass through degradation info directly to the UI rather than building its own error handling for this source.

### Finding 7: Existing Cache Patterns Provide a Proven Template
The codebase has two production-tested in-memory TTL cache implementations:
- **Pricing service cache** (agent-manager/internal/pricing/service.go): Uses `sync.RWMutex` + map with `expiresAt` per entry. Includes staleness metadata fields (`FetchedAt`, `ExpiresAt`, `IsStale`, `IsExpired()`). Manual invalidation on updates.
- **Stats orchestration cache** (agent-manager/internal/orchestration/stats.go): Uses 30s fixed TTL with `sync.RWMutex` + `map[string]*cachedSummary`. Parallel query execution via `sync.WaitGroup`.

Both use Go stdlib only — no external cache libraries. Command-center follows the same pattern with per-source TTLs and a stale-on-failure fallback (return expired cache entry when source fetch fails, rather than returning an error).

### Finding 8: errgroup Is the Standard Fan-Out Pattern
`errgroup.WithContext` is used throughout the codebase for concurrent operations:
- git-control-tower: parallel git operations (repo_status_service.go)
- scenario-stack-governor: parallel module validation with `g.SetLimit(5)`

For command-center's multi-source fan-out, `errgroup` with per-source `context.WithTimeout` child contexts (5s each) is the right approach. Each source fetch runs in its own goroutine with an independent timeout. If a source times out, its goroutine returns an error, but the other sources' results are still available. The dashboard composer uses fresh results where available and falls back to cache for failed sources.

### Finding 9: Circuit Breaker Should Be In-Memory and Per-Source
The swarm-manager circuit breaker is file-backed and per-backlog-item (tracks consecutive failures with threshold + cooldown). For command-center, a simpler per-source in-memory model suffices:
- Track consecutive failures per source (lpbs, swarm, scenarios, completeness)
- Trip after N consecutive failures (suggested: 3)
- Half-open after cooldown (suggested: 30s) — allow one probe request
- Reset on success
- No file persistence needed — command-center restart resets breakers, which is fine since sources may have recovered

When a breaker is tripped, skip the HTTP call entirely and serve from cache. This prevents hammering a downed source.

## Settled Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LPBS data access strategy | API-first (no direct DB) | Loose coupling — LPBS owns its data contracts. Add ~3 new summary endpoints for missing aggregations. |
| Derived metric computation | LPBS computes (MRR, churn, ARPU) | Single source of truth for business metric definitions. Other consumers get the same numbers. |
| Caching strategy | In-memory TTL (30s LPBS, 30s Swarm Manager, 60s Vrooli Core) with stale-on-failure | Simple, fast, no external dependency. Different TTLs match source update frequency. Stale data preferable to error screens. |
| LPBS auth approach | Widen requireAdminOrService() to all needed admin endpoints | One-line change per route. Reuses existing Bearer token infrastructure. No new auth concepts needed. |
| API endpoint structure | Per-dashboard endpoints (GET /api/v1/dashboards/{page}) | Single request per page load. Backend composes exactly what each dashboard needs. Simpler UI code. |
| New LPBS endpoints | Three separate: revenue-summary, subscription-stats, user-growth | Clean separation by domain. Each endpoint focused and cacheable independently. |
| Concurrent fan-out timeout | Per-source context.WithTimeout (5s each) via errgroup | Best UX — dashboards render with whatever data is available. Cache serves stale data for timed-out sources. Each source gets independent timeout. |
| Staleness metadata format | Per-source staleness object in every response | UI can render freshness indicators per data section. Structure: `{"sources": {"lpbs": {"status": "fresh", "fetched_at": "...", "age_seconds": N}, ...}}` |
| Go package structure | Flat package main with service-per-file (LPBS pattern) | Consistent with existing codebase. Files: main.go, server.go, routes.go, cache.go, client_lpbs.go, client_swarm.go, client_scenarios.go, dashboard_*.go, gaps.go |

## Limitations
- Caching TTL values (30s/60s) are estimates — real-world tuning will depend on dashboard refresh UX requirements and source update frequency
- Have not assessed full memory footprint of caching overview responses at scale (estimated 10-50KB per cached entry, negligible for single instance)
- Completeness-scoring service may not always be running — stale-on-failure cache handles this, but initial cold-start with no cache means the first request will show gaps
- Error response format normalization across sources not yet designed — each source (LPBS, Swarm Manager, Vrooli Core) has different error formats that command-center will need to handle in source clients
- Circuit breaker thresholds (3 failures, 30s cooldown) are suggested starting points — tuning requires production observation
- Per-source 5s timeout is a starting estimate — may need adjustment based on observed LPBS query latency under load

## Actions

### Action 1: Create backlog item — LPBS revenue-summary endpoint
- **Kind**: execute
- **Title**: Add GET /api/v1/admin/revenue-summary endpoint to LPBS
- **Description**: New admin endpoint returning MRR (by tier, total), ARR, revenue by session_type (subscription/credits_topup/supporter_contribution), churn rate (30/90-day windows), and ARPU. Requires joining subscriptions.price_id against PlanStore. Widen auth to requireAdminOrService(). See research conclusion Finding 1 for price mappings.
- **Initiative**: command-center-data-layer
- **Priority**: 2
- **Effort**: M

### Action 2: Create backlog item — LPBS subscription-stats endpoint
- **Kind**: execute
- **Title**: Add GET /api/v1/admin/subscription-stats endpoint to LPBS
- **Description**: New admin endpoint returning subscription counts by tier × status (active/trialing/past_due/canceled), with trend arrays over 7/30/90-day windows. Widen auth to requireAdminOrService().
- **Initiative**: command-center-data-layer
- **Priority**: 2
- **Effort**: S

### Action 3: Create backlog item — LPBS user-growth endpoint
- **Kind**: execute
- **Title**: Add GET /api/v1/admin/user-growth endpoint to LPBS
- **Description**: New admin endpoint returning registration trends and active user counts over 7/30/90-day windows. Widen auth to requireAdminOrService().
- **Initiative**: command-center-data-layer
- **Priority**: 2
- **Effort**: S

### Action 4: Create backlog item — Widen LPBS admin auth for service-to-service access
- **Kind**: execute
- **Title**: Widen LPBS admin endpoints to accept service Bearer tokens
- **Description**: Swap requireAdmin() to requireAdminOrService() on existing admin endpoints that command-center needs: GET /api/v1/metrics/summary, GET /api/v1/admin/usage, GET /api/v1/admin/users, GET /api/v1/admin/coupons/usage. One-line change per route in routes.go. See research conclusion Finding 5 for auth tier details.
- **Initiative**: command-center-data-layer
- **Priority**: 2
- **Effort**: S
