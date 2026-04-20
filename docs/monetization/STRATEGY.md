# Monetization Strategy

Narrative framing and durable principles for how Vrooli monetizes. This document changes rarely; specifics change in the catalog, tiers, and financial model.

## What Vrooli sells

Three commercial surfaces, orthogonal to each other:

1. **Bundles of scenarios** (the catalog) — themed collections that solve real problems. Today: business bundle (developer + solopreneur tools); next: lifestyle bundle (personal + household).
2. **Delivery tiers** (how the bundles reach the user) — individual desktop/mobile apps, self-hosted full Vrooli runtime, hosted cloud Vrooli, and eventually a hardware appliance. Each tier has its own cost structure and price point.
3. **Services** (bridge revenue) — done-for-you engagements (lead generation, standalone app development, consulting) that generate immediate cash, validate capability, and seed subscribers. Services are explicitly a bridge, not a business line.

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

### 3. Services are a bridge, not a business

Services generate cash immediately in chunks; subscriptions compound slowly. Without discipline, this gravity pull can quietly reorient the company into a consultancy. Every services engagement carries four mandatory attributes:

- A **validation hypothesis** (which capability are we proving?)
- A **fixed-duration pilot**
- A **productization target** (which SKU does this feed?)
- A **sunset or convert clause** (by date X, productize and hand off to subscription, or stop)

The success metric for a services line is the **service-client → subscriber conversion rate**, not services revenue itself. Conversion happens when (a) the product replaces the manual work without new support burden, AND (b) the client has built trust in it. Both are required.

See [REVENUE_LINES.md](REVENUE_LINES.md) for the operational discipline.

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

## North stars

These describe where Vrooli is going, not what it's doing. Nothing here should be planned against without explicit initiation from the operator.

### North star 1: full-runtime hosted offering (Tier 3)

Users who want Vrooli without managing infrastructure get a hosted instance we run for them. Same runtime as self-hosted, but on our infrastructure. This is the largest revenue surface long-term because it captures users who would otherwise churn on setup friction.

### North star 2: hardware appliance (Tier 4)

A dedicated Vrooli machine — either a purchase or a subscription-included appliance — sold to households and small businesses. Runs the full Vrooli stack locally, preserves privacy, maximizes hardware utilization.

**This is a different business.** Hardware means inventory, BOM, RMA, certifications, fulfillment, physical support SLAs — a skill set almost unrelated to everything else. Tier 4 exists as a directional marker so we don't accidentally wander in the wrong direction. It is explicitly **not** to be planned against without a deliberate decision from the operator to enter the hardware business.

### North star 3: default-alive

The financial goal is that Vrooli reaches a state where monthly revenue minus monthly cost is positive and trending correctly, independent of funding. See [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md) for the math.

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
- **Activation** — scenario-feature (onboarding UX is a product concern)
- **In-bundle expansion** — scenario-feature (product-led, agent-driven)
- **Cross-SKU upsell** — monetization + marketing + feature, collaborative
- **Retention** — nobody yet; requires telemetry that doesn't exist (see [TELEMETRY_ROADMAP.md](TELEMETRY_ROADMAP.md))
- **Advocacy** — OSS strategy (this doc) + community operations when they emerge

The monetization team owns **definition and measurement** across all stages. It proposes targets, tracks metrics (or flags them as pending), and surfaces the current bottleneck.

## What this strategy is not

- Not a marketing plan. See marketing-crew docs for that.
- Not a product roadmap at the scenario level. See swarm-manager for initiatives.
- Not a pricing tactics sheet. See [PRICING.md](PRICING.md).
- Not a tech-tree. See the tech-tree-designer scenario when it exists.
