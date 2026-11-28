# Benchmarks

Curated external benchmarks used to validate Vrooli's pricing, retention targets, funnel assumptions, and cost structure. Owned by the `market-validator` agent in the monetization team. Populated slowly, not scraped.

This file is mostly empty pre-launch. Each heartbeat, market-validator may propose additions via decisions with context `benchmark-update`. Operator curates.

## How to use this file

- When setting a monetization target (price, retention, churn, attach rate), reference a benchmark from this file or flag the decision as "made without benchmark."
- When a benchmark is stale (>12 months), market-validator proposes a refresh.
- When a competitor materially changes behavior (pricing change, bundle change, product pivot), market-validator proposes a benchmark update.

## Entry format

Each benchmark entry includes:

- **Comp** — company/product name
- **Category** — dev tools / multi-product bundle / consumer subscription / etc.
- **Relevant dimension** — what we're comparing (pricing, retention, churn, activation)
- **Value** — the observed number or range
- **Source** — where the data came from (their pricing page, public report, industry analysis)
- **Date captured** — when the data was observed
- **Applicability** — how directly this comps against Vrooli (high / medium / low)

## Dev-tool SaaS

### Pricing

*No entries yet.* Expected comps when market-validator begins work: GitHub Copilot Business, Cursor, Linear, Retool, Vercel Pro, Framer Pro, Raycast Pro. These set the pricing envelope for the business bundle's Tier 1 and Tier 2.

### Retention / Churn

*No entries yet.* Public dev-tool churn benchmarks (primarily from B2B SaaS analyses) suggest mature dev tools operate at `aspirational: 1-3% monthly gross churn`, but this needs sourced confirmation before it informs Vrooli's targets.

### Activation

*No entries yet.* Dev-tool activation benchmarks are category-specific; general SaaS Day-30 activation rates vary widely (30-70%). Bundle products have tended toward the lower end because each component requires its own activation.

## Multi-product bundle SaaS

### Pricing

*No entries yet.* Expected comps: Notion + Notion AI, Atlassian Cloud, Google Workspace, Microsoft 365. These inform tier-upgrade economics and attach-rate assumptions.

### Attach rate (across add-ons)

*No entries yet.* Industry patterns suggest `aspirational: 15-30%` attach for well-targeted add-ons among core-bundle subscribers, but context-specific.

## Consumer subscription SaaS (for lifestyle bundle)

### Pricing

*No entries yet.* Expected comps: consumer productivity (Todoist, Fantastical), personal finance (YNAB, Copilot), family (Cozi, FamilyWall), health/habits (Streaks, Finch).

### Churn

*No entries yet.* Consumer subscription churn tends to be materially higher than B2B — typical `aspirational: 5-10% monthly` for non-essential consumer subs.

## Services-line comps

### Lead generation

*No entries yet.* Expected comps: Angi (Home Advisor), Thumbtack, Yelp lead gen, specialty lead sellers by vertical. Pricing models (per-lead, per-close, subscription) vary.

### Done-for-you app dev

*No entries yet.* Expected comps: agency hourly rates by region, gig platforms (Toptal, Upwork top-rated), no-code build services. Sets the price floor for this services line.

### AI-workflow consulting

*No entries yet.* Expected comps: specialty AI consulting firms, Anthropic / OpenAI implementation partners, freelance AI consultants.

## Infrastructure cost benchmarks (informs tier COGS)

### Per-user VPS/container cost

*No entries yet.* Expected sources: operator's own projections from scenario-to-cloud deployment costs; public cost-per-user studies from similar tools.

### Token pass-through margin

*No entries yet.* Expected comps: how Cursor, Poe, OpenRouter, and similar gateways price their managed routing — both markup % and meter vs. flat approaches.

## Next benchmarks to capture

When market-validator activates, the first benchmarks to capture in roughly this order:

1. **Dev-tool pricing tiers for business-bundle comps** — sets the Tier 1 and Tier 2 price envelope.
2. **Multi-product bundle pricing and attach rates** — informs add-on pricing strategy.
3. **Dev-tool and bundle retention/churn benchmarks** — replaces `aspirational` retention targets in [FUNNEL.md](../strategy/FUNNEL.md).
4. **Gateway markup comps** — directly informs Tier 2 margin math.
5. **Services-line vertical pricing** — informs when/if to activate lead-gen, consulting, done-for-you.

Each should be captured with the full entry format above so the team can distinguish grounded numbers from speculation.
