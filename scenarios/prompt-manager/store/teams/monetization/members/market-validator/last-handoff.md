### Scope this heartbeat
- Active tier × bundle: Tier 1 business bundle. First substantive scan — `BENCHMARKS.md` was entirely skeleton and `market-scans.jsonl` was empty except for an earlier same-day Cursor+Copilot capture. Pulled competitor pricing pages live via WebFetch.

### Captured
- **Raycast Pro** — `pricing` — starts at $8/mo annual — high applicability (solo dev-tool low end)
- **Notion** — `pricing` — Plus $10, Business $20/user/mo — medium (per-seat)
- **Notion AI bundling** — `other` — AI bundled into every paid tier, Custom Agents at $10/1K credits — high (mirrors Vrooli's integrated-gateway positioning)
- **Setapp** — `pricing` — Mac $9.99, Mac+iOS $12.49, Power User $14.99; AI+ $9.99–$23.99 credit-based — high (closest multi-app-bundle comp)
- **Category-wide credit allowances** — `competitive-observation` — Cursor 3x/20x, Setapp AI+ 10/125/250, Notion $10/1K — credit-allowance-plus-overage is now category-standard
- **Tier 1 pricing trough** — `competitive-observation` — $29-$49 target sits between a $10-$20 bundle/solo-dev cluster and a $39-$60 prosumer-AI-dev cluster

### Assumption checks
- **"Mainstream user consumes far less gateway than heaviest 5%"** (FINANCIAL_MODEL.md Tier 1/2 COGS) — Cursor exposes 3x/20x usage spread, Setapp AI+ 10→250 credit spread (25x). Published tier spreads are *consistent with* a 3-25x skew but are indirect evidence, not direct usage data. Shape validated; specific 5th-percentile claim not. Flag: consider pulling OpenAI/Anthropic public usage-distribution reports in a future heartbeat.

### Competitive changes observed
- **Credit-allowance-plus-overage billing is now the category norm**, not a novel experiment — meaningfully lowers the risk of mainstream pushback on Vrooli's proposed billing shape.
- **Notion has converged on "AI inside the base tier"** (no separate AI add-on). Directly supports Vrooli's "integrated gateway is the core reason to pay" positioning.
- **GitHub Copilot Pro and Pro+ marked "temporarily unavailable"** on the live pricing page — notable state, worth re-checking in 30 days.

### Gaps still missing from BENCHMARKS.md
1. **Gateway markup comps** (how Cursor / Poe / OpenRouter price managed routing — directly informs Tier 2 margin math).
2. **Dev-tool retention / churn benchmarks** to replace the `aspirational: 1-3% monthly gross churn` placeholder in FUNNEL.md.
3. **Full Raycast ladder** incl. Advanced AI add-on pricing — today's fetch was incomplete.

### Decisions raised this heartbeat
- `dec-1777061048708846767` — `benchmark-update` — populate BENCHMARKS.md dev-tool + multi-product bundle pricing sections with today's scan (9 entries with sources + dates).
- `dec-1777061056395576280` — `pricing-decision` — revisit Tier 1 target bracket; $29-$49 sits in a trough between $10-$20 and $39-$60 clusters. Not proposing a price; proposing operator pick positioning (premium bundle vs prosumer AI dev-suite).

### Knowledge entry written
- topic: `market-scan-2026-04-24` (id `knw-1777061081500068027`). No prior `market-scan-*` entry existed; this begins the supersession chain.