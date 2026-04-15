# Command Center Data Layer — Orchestration Context

## Source
Brainstorming session on 2026-04-07. This initiative covers the read-only aggregation API and gap-tracking infrastructure.

## Architecture Decision
Command-center is a READ-ONLY VIEW LAYER. It does NOT store its own data or become a source of truth. Data ownership:
- LPBS owns subscription/payment/analytics data (Postgres)
- Swarm Manager owns backlog/velocity/execution data (SQLite event log)
- Vrooli core owns scenario metadata/health (filesystem + REST API)

Command-center queries these and composes dashboard-ready payloads with short-TTL caching (30-60s suggested).

## The Gap Tracking System
Every metric widget has a dataSource status: live, gap, or partial.
- live = real data flowing
- gap = metric defined as valuable, no pipeline yet
- partial = some data available, not complete

GET /api/v1/gaps returns all gap/partial metrics grouped by dashboard.

This creates a recursive feedback loop:
1. Design dashboards with ideal metrics → many start as gap
2. Director Swarm checks /api/v1/gaps → sees missing capabilities
3. Director proposes backlog items to fill gaps → human approves
4. Feature team builds pipeline → gap flips to live
5. Dashboard more useful → reveals new metrics to add → cycle repeats

## The "Two Lenses" Insight
- Swarm Manager = the WORK (what's being built, velocity, blocking)
- Command Center = the OUTCOMES (subscribers growing? revenue moving? system healthy?)
Director Swarm uses both. They're complementary, not competing.

## Key Integration Points
- LPBS: metrics/summary endpoint (visitors, conversions), subscriptions table, checkout_sessions, credit system, users, download catalog
- Swarm Manager: /api/v1/stats (with category filtering), /api/v1/overview (items, initiatives, deps, governance)
- Vrooli Core: /scenarios (metadata, health, status), completeness scoring

## Initial Gap Inventory
Mission Control: MRR, total users, uptime %, compound intelligence counter, cross-system activity feed
Hive: usage frequency, revenue attribution, maturity timeline, active-work overlay
Forge: cost per item (token spend), contributor leaderboard, burndown charts, predicted dates, flow efficiency
Ledger: LTV, expansion revenue, net retention, forecasting, dunning rate, trial-to-paid funnel
Broadcast: social followers, post engagement, SEO rankings, funnel conversion, campaign attribution, email metrics, CAC, content velocity, brand consistency, referral sources
Panorama: system pulse, anomaly alerts, per-domain sparklines
