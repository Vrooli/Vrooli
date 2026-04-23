# Monetization Strategy

Narrative framing and durable principles for how Vrooli monetizes. This document changes rarely; specifics change in the catalog, tiers, and financial model.

## What Vrooli sells

Three orthogonal axes — **what is packaged**, **how it reaches the user**, and **how money flows**:

1. **Bundles of scenarios** — *what* is packaged. Themed collections that solve real problems. Today: business bundle (developer + solopreneur tools); next: lifestyle bundle (personal + household).
2. **Delivery tiers** — *how* the bundles reach the user. Individual desktop/mobile apps, self-hosted full Vrooli runtime, hosted cloud Vrooli, and eventually a hardware appliance. Each tier has its own cost structure and price point.
3. **Revenue lines** — *how* money flows. Distinct from bundles/tiers and tracked independently because their cost structures, acquisition channels, unit economics, and operational disciplines differ. See [REVENUE_LINES.md](REVENUE_LINES.md) for the full index.

The current revenue lines are:

- **`subscription`** — the product, and the destination every other line aims toward.
- **Services** — deliberate lever, not a business. Services lines (`lead-generation`, `app-development`, `consulting`) validate capability, generate near-term cash, and seed subscribers. We operate our own scenarios on behalf of clients — the same shovels we sell are the shovels we use to dig for gold. See principle 3 below for discipline.
- **`consumer-products`** — own-produced physical and digital SKUs (books, planners, kits, courses) sold standalone and surfaced contextually inside scenarios. Concentrated in the lifestyle bundle. Gated on inventory maturity.
- **`affiliate-commerce`** — commission on referred purchases through partners (Amazon Associates first). Also concentrated in the lifestyle bundle; gated on the recommendation-blindness architecture existing in code.

Both `consumer-products` and `affiliate-commerce` carry a hard architectural rule: **the agent producing a recommendation must not know what we sell or earn commission on.** Offer insertion and link rewriting are strictly post-processing over recommendations that were already made. This protects the authority layer — the ground truth that lifestyle-bundle users pay for. See the two revenue-line files for the full constraint set.

## Core principles

### 1. The subscription buys convenience and integrated access, not access to the code

Vrooli is open source. A sufficiently motivated developer can clone it, bring their own API keys, and run everything for free. This is **strategic positioning, not a revenue leak.**

What the subscription actually buys:
- **Zero-config setup** — one account, no juggling eight API subscriptions (OpenRouter, Anthropic, OpenAI, ElevenLabs, Deepgram, etc.)
- **Wholesale-to-retail API routing** through the Vrooli gateway
- **Managed infrastructure** on hosted tiers
- **Product polish** — curated bundles, supported workflows, onboarding

The free-self-host path builds brand credibility and produces evangelists. Most paying customers will be people who *could* self-host but prefer not to. A handful of power users who choose free today become paid hosted-tier customers later when their needs change.

**No agent or doc should ever frame the subscription as paywalling core features.** If you find yourself wording it that way, the framing is wrong.

### 2. Agents are the expansion engine

Most SaaS companies spend heavily on lifecycle marketing — email drips, in-app nudges, webinar funnels — to get users to adopt more of the product. Vrooli has a structurally different surface: **the agents running in the user's workflow can organically suggest other apps from the bundle when relevant work appears.**

This makes in-bundle expansion (the drive toward breadth-of-adoption, which is the strongest retention signal) a native capability, not a marketing function. When designing acquisition, activation, and upsell tactics, **default to agent-driven mechanisms over marketing-driven ones.** Only fall back to marketing when an agent surface cannot reach the relevant moment.

### 3. Services are a deliberate lever, not a business

**Scenarios are double-revenue assets.** Every capability we build can be sold to customers (the shovel) AND used by us to deliver paid work on customers' behalf (using the shovel to dig gold). A property-services scenario is the same tooling we'd operate to generate leads for a roofer and sell them per-lead. A standalone-app scenario is the same capability we'd use to deliver custom builds for clients who don't want to operate agents themselves. The catalog and the services pipeline draw from the same well.

**Services generate cash upfront in chunks; subscriptions compound slowly.** This timing asymmetry is the point. Once core bundle capabilities are production-grade but before subscription revenue crosses default-alive, services are a deliberate revenue lever — not a distraction. The discipline below exists *because* we intend to lean in, not because we want to stay away.

**Phase-gating** keeps the posture honest:

- **Pre-bundle (current state):** services lines remain `candidate`. Each has an explicit revisit trigger tied to a specific capability being deployable as a thin tool. No services line activates until its underlying scenario can stand on its own.
- **Post-bundle, pre-default-alive:** services lines activate in deliberate order and are expected to produce meaningful revenue while subscriptions scale.
- **Post-default-alive:** services wind down or productize. Success means subscriptions have made them unnecessary.

**Every active services line carries four mandatory attributes** (guardrail violation if any is missing):

- A **validation hypothesis** (which capability are we proving?)
- A **fixed-duration pilot**
- A **productization target** (which SKU does this feed?)
- A **sunset or convert clause** (by date X, productize and hand off to subscription, or stop)

**Success metric is service-client → subscriber conversion rate**, not services revenue. Conversion happens when (a) the product replaces the manual work without new support burden, AND (b) the client has built trust in it. Both are required. Converting too early means churn from disappointment and a new support burden on the product team; converting too late means we keep doing manual work we don't need to — and block capacity we'd rather spend on the next services client.

See [REVENUE_LINES.md](REVENUE_LINES.md) for operational discipline (time-capacity caps, legal surface checks, separate tracking).

### 4. Candidates have revisit triggers, never vibes

Every candidate SKU (bundle, add-on) and every candidate tier carries an **explicit revisit trigger** — a concrete condition, not a judgment call. Examples:

- *"Revisit when business bundle has ≥100 paying users."*
- *"Revisit when agent-manager is shipped and deployable standalone."*
- *"Revisit when 3 prospects explicitly request property-services tooling."*

Triggers let the team be mechanical instead of judgmental. The catalog stays expansive in *thinking* but disciplined in *execution*. The default heartbeat reads only `active` SKUs and checks candidates against their triggers — it does not re-debate dormant ideas.

### 5. Breadth of adoption = retention

A user running one app churns at one rate; a user running three apps churns dramatically slower. This is extremely well-documented across multi-product SaaS. The corollary: depth-layer apps with weak acquisition appeal can still be high-priority because their retention impact is strong. Never evaluate a scenario on acquisition appeal alone.

### 6. Activation is the leading indicator of retention

Most churn is failed activation. A user who never wires Vrooli into their workflow churns in month 1 regardless of product quality. **Activation work IS retention work.** When the team is asked to prioritize retention investments, they should usually be looking at activation first.

### 7. Compound-capability reuse makes niche add-ons economically viable

Most SaaS companies cannot afford to build niche vertical add-ons because marginal production cost is too high. Vrooli can, because every core capability is reused. This is the structural advantage that justifies the add-on model — but only when reuse is actually high. An add-on that requires building mostly-new capability is not a real add-on; it's a new bundle with lower margin. The `contrarian` member's job is to keep this honest.

## Long-term directions

Where Vrooli is going beyond the near-term catalog. Two sub-categories — kept apart because they behave differently in the team's lifecycle vocabulary.

### Long-term candidates

Real directions the team is moving toward with concrete activation triggers. These are **not** `north-star` in the [TIERS.md](TIERS.md) lifecycle sense; they are `candidate` and can be planned against once their triggers fire.

**Tier 3 — full-runtime hosted offering.** Users who want Vrooli without managing infrastructure get a hosted instance we run for them. Same runtime as self-hosted, on our infrastructure. Probably the largest long-term revenue surface because it captures users who would otherwise churn on setup friction. See [TIERS.md §"Tier 3"](TIERS.md) for activation trigger and capability prereqs.

**Default-alive.** Monthly revenue meets or exceeds monthly burn, consistently, independent of funding. Every monetization-team decision should weigh its impact on the default-alive date. See [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md) for the math and target.

### North-star markers

These use the strict `north-star` lifecycle status from [TIERS.md](TIERS.md). **Must not be planned against without explicit operator initiation** — a deliberate strategic decision, not a downstream consequence of other work.

**Tier 4 — hardware appliance.** A dedicated Vrooli machine — either a purchase or a subscription-included appliance — sold to households and small businesses. Runs the full Vrooli stack locally, preserves privacy, maximizes hardware utilization.

**This is a different business.** Hardware means inventory, BOM, RMA, certifications, fulfillment, physical support SLAs — a skill set almost unrelated to everything else. Tier 4 exists as a directional marker so we don't accidentally wander in the wrong direction. Promoting it out of `north-star` requires the operator deliberately choosing to enter the hardware business.

## Bundle ordering principle

Each bundle has two layers:

- **Headliner layer** — scenarios that are (a) compelling enough to justify purchase on their own and (b) deployable with today's capabilities. These are the acquisition hook. For the business bundle: Web Console + Git Control Tower.
- **Depth layer** — scenarios that either (a) sharply amplify the headliners once they ship, or (b) are themselves future headliners that can't ship yet because prereqs aren't built.

This means ordering is a **DAG with a monetization timeline overlay**, not a flat line. Each scenario has two independent signals: *standalone appeal* (would someone pay for this alone?) and *deployability-today* (what blocks shipping it?). See [catalog/base/business.md](catalog/base/business.md) for the current DAG.

## Ordering principle across bundles

Business bundle ships first because:
- The developer audience is addressable, understands the value proposition, and is willing to pay
- Vrooli's core capabilities (code generation, AI tooling, software automation) map directly to the bundle's apps — shortest path from current state to shippable headliner
- Revenue from the business bundle funds the capability investments needed to enable the lifestyle bundle

Lifestyle bundle follows because:
- It serves a much larger TAM (everyone has a personal life; not everyone is a developer)
- Its scenarios often reuse capabilities built for the business bundle
- Some lifestyle apps genuinely require capabilities that don't exist yet and can't be cleanly faked

Add-ons layer on top of both base bundles as domains mature. Add-ons are **held in candidate state until the parent base bundle has paying users.** See [CATALOG.md](CATALOG.md).

## Cross-team coordination

The monetization team is the curator of this strategy; it does not execute every step. Execution responsibilities:

- **Acquisition** — marketing-crew + landing-page-business-suite
- **Activation** — no dedicated owner. Onboarding/first-run UX emerges as a byproduct of scenario work surfaced by director-swarm (gap analysis) and monetization (catalog readiness).
- **In-bundle expansion** — no dedicated owner. Agent-driven expansion is a structural property of scenarios themselves; specific surfaces emerge as byproducts of monetization catalog work.
- **Cross-SKU upsell** — monetization (strategy) + marketing-crew (lifecycle messaging where agent surfaces can't reach). In-product surfaces emerge as byproducts of scenario work.
- **Retention** — nobody yet; requires telemetry that doesn't exist (see [TELEMETRY_ROADMAP.md](TELEMETRY_ROADMAP.md))
- **Advocacy** — OSS strategy (this doc) + community operations when they emerge

The monetization team owns **definition and measurement** across all stages. It proposes targets, tracks metrics (or flags them as pending), and surfaces the current bottleneck.

## What this strategy is not

- Not a marketing plan. See marketing-crew docs for that.
- Not a product roadmap at the scenario level. See swarm-manager for initiatives.
- Not a pricing tactics sheet. See [PRICING.md](PRICING.md).
- Not a tech-tree. See the tech-tree-designer scenario when it exists.
