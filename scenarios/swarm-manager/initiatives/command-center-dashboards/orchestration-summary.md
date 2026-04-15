# Command Center Dashboards — Orchestration Context

## Source
Brainstorming session on 2026-04-07. This initiative covers the 6 individual dashboard pages.

## Critical Design Principle: Beauty First
These are NOT utilitarian Grafana panels. Each dashboard is a standalone visual experience with its own immersive theme. Think "art installation that happens to show data" more than "admin panel on a big screen." The user explicitly wants something they'd be proud to show off — something you'd leave up on a TV via Xbox browser.

Each theme is genuinely distinct — different backgrounds, typography, color palettes, animations. NOT a shared design system with color tokens.

## Dashboard Details

### 1. Mission Control — "Ground Control" Theme
- Dark + star field + electric blue + monospace + NASA JPL aesthetic
- Hero KPIs: scenarios running/total, velocity this week, subscriber count, visitors 7d, backlog + ETA
- ~70% live data. Gaps: MRR, total users, uptime, compound intelligence counter, cross-system activity feed

### 2. The Hive — "Bioluminescent" Theme
- Deep ocean dark + glowing greens/teals/cyans + organic pulsing nodes + neural pathways
- Force-directed graph of all 95+ scenarios. Sized by importance, colored by completeness. Semantic clusters by tag
- ~80% live. Gaps: usage frequency, revenue attribution, maturity timeline, active work overlay

### 3. The Forge — "Foundry" Theme
- Charcoal + amber/orange/molten gold + forge gauges + rising sparks
- Richest dashboard (~90% live). Full swarm stats: throughput, timing, blocking, agent perf, initiatives, governance
- Gaps: cost per item, contributor leaderboard, burndown, predicted dates, flow efficiency

### 4. Ledger — "Vault" Theme
- Dark green + gold + serif fonts + paper texture + banking aesthetic
- Subscribers by tier, checkout revenue, credit system, intro pricing, churn
- ~60% live. Gaps: LTV, expansion revenue, net retention, forecasting, dunning, trial-to-paid funnel

### 5. Broadcast — "Signal Tower" Theme
- Purple/magenta + radiating waves + particle flow + luminous funnel beam
- LPBS analytics: visitors, conversions, variant A/B, top CTAs
- Most gaps (~40% live). Social, SEO, campaign, email, content velocity, brand consistency all gap

### 6. Panorama — "Cosmos" Theme
- Black + nebula gradient + solar system metaphor + meditative orbital animation
- THE "leave on TV" page. Central sun = system health. Five planets = one per domain
- Planet distance from sun = how much attention area needs. Shooting stars on milestones
- Depends on all other 5 pages (curated best metric from each)

## Gap UX Treatment
Gap metrics render beautifully within each theme — shimmer, translucent glow, "coming soon" aesthetic
NOT ugly "N/A" text. Gaps occupy intended layout position so composition is visible
Dashboards can be deployed before all data exists — they look aspirational rather than broken
