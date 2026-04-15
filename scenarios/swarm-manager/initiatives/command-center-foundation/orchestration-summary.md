# Command Center Foundation — Orchestration Context

## Source
Brainstorming session between user (project director) and Claude on 2026-04-07. The user wants to create "war room dashboards" / "command center displays" — the kind of beautiful, inspiring dashboard TVs you see at companies like Stripe (showing % of world GDP processed) or Tesla (RoboTaxi live map). This is a showcase piece as much as a monitoring tool.

## Key Decisions Made
- **Single scenario** at scenarios/command-center/ containing all dashboards, not separate scenarios per dashboard
- **React+Vite UI + Go API backend** per Vrooli conventions
- **Read-only aggregator** — command-center does NOT store its own data or become a source of truth. It queries LPBS, Swarm Manager, and Vrooli core APIs and composes dashboard-ready payloads
- **6 dashboard pages**, each with its own completely independent visual theme (not color variants — genuinely distinct aesthetics)
- **Kiosk-first UX** — fullscreen by default, controls hidden, auto-cycle between pages, designed for TV/Xbox browser
- **Gap tracking system** — every metric tagged as live/gap/partial, with /api/v1/gaps endpoint. Gaps rendered beautifully (shimmer/glow) not as ugly "N/A". Gaps become signals for what to build next
- **Director Swarm** owns gap monitoring (not Meta Optimization or Feature team). Director spots gaps, proposes backlog, human approves, Feature team executes
- **Two complementary lenses**: Swarm Manager = the work (what's being built), Command Center = the outcomes (are results showing up). Director uses both

## Theme Assignments
1. Mission Control → "Ground Control" (space/NASA, electric blue, monospace)
2. The Hive → "Bioluminescent" (deep ocean, glowing greens/teals, neural pathways)
3. The Forge → "Foundry" (charcoal + amber/gold, forge gauges, rising sparks)
4. Ledger → "Vault" (dark green + gold, serif fonts, paper texture, banking)
5. Broadcast → "Signal Tower" (purple/magenta, radiating waves, particle flow)
6. Panorama → "Cosmos" (black + nebula, solar system metaphor, meditative)

## Data Source Availability by Dashboard
- The Forge: ~90% live (Swarm Manager stats engine is very rich)
- Ledger: ~60% live (LPBS Stripe/subscription data exists, needs aggregation)
- Mission Control: ~70% live (scenario health + swarm stats, missing revenue)
- The Hive: ~80% live (scenario metadata/health/completeness, missing usage frequency)
- Broadcast: ~40% live (LPBS analytics only, social/SEO/campaign are gaps)
- Panorama: composite of above

## Upstream Data Sources (Detailed)
### LPBS (Postgres)
- subscriptions: status, plan_tier (free/solo/pro/studio/business), bundle_key, canceled_at
- checkout_sessions: session_type, amount_cents, status
- credit_wallets + credit_transactions: balances, consumption patterns
- usage_records: per-user per-billing-period by limit_key
- metrics_events: visitors, conversions, CTA clicks, scroll depth, variant A/B (7/30/90 day)
- feedback_requests: by type and status
- users: count, registration trends, last_login
- download_apps + download_assets: app catalog, releases

### Swarm Manager (REST API)
- GET /api/v1/stats: throughput, timing, scope, blocking, agent, dashboard, review stats
- GET /api/v1/overview: all items, initiatives with rollup, dependency graph, governance

### Vrooli Core (REST API port 8092)
- GET /scenarios: all scenarios with metadata, status, health, ports, runtime
- vrooli scenario completeness: 0-100 scores with classification

## Unresolved Questions Deferred To Workshop
- Direct DB access vs API calls for LPBS data
- Specific visualization library choices (D3 vs Three.js vs Framer Motion vs combination)
- Xbox Edge browser specific constraints and WebGL support
- Exact caching TTLs per source
- Whether gap metadata lives in API responses or is purely UI-side config
- Specific transition animations between themed pages during auto-cycle
