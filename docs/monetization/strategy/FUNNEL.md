# Funnel

How users flow through Vrooli's monetization, from first exposure to long-term advocacy. Stages are **AARRR-adapted** (acquisition, activation, retention, referral, revenue) plus two stages specific to this model (in-bundle expansion and cross-SKU upsell).

## Why this doc exists

"Funnel" means different things to different agents. Without named stages, owners, and metrics, the monetization team would collapse everything into vague "strategy." This file defines the stages precisely so targeted questions get targeted answers: *what's broken at activation?* *why is in-bundle expansion flat?*

## The stages

### 1. Acquisition

A prospect learns Vrooli exists.

- **Owner:** `marketing-crew` + `landing-page-business-suite` (landing pages, SEO, content)
- **Primary mechanism:** headliner scenarios (e.g., `web-console`, `git-control-tower`) as the "why this exists" hook, backed by bundle-level landing pages.
- **Leading metric:** visits to bundle landing pages, conversion rate to signup.
- **Current state:** `pending-telemetry` — no traffic analytics pipeline yet.
- **Aspirational target:** landing-page → signup conversion ≥ `aspirational: 3%`.

### 2. Activation

A signed-up user successfully uses at least one headliner and gets a "win."

- **Owner:** no dedicated owner. First-run UX and onboarding emerge as byproducts of scenario work — surfaced by `director-swarm` gap analysis when activation data shows a miss, or by `monetization` when catalog readiness depends on it.
- **Primary mechanism:** onboarding that walks a new subscriber through setting up API keys or signing in, then directly into a headliner scenario's core workflow.
- **Leading metric:** % of signups who use a headliner ≥3 times in their first 30 days.
- **Current state:** `pending-telemetry` — no activation events defined yet.
- **Aspirational target:** day-30 activation rate ≥ `aspirational: 60%`.
- **Retention-critical:** activation is the leading indicator of retention. Most churn is failed activation. Activation work IS retention work.

### 3. In-bundle expansion

An activated user starts using additional scenarios from the bundle they already own.

- **Owner:** no dedicated owner. Agent-driven cross-scenario discovery is a structural property of the platform; specific surfaces emerge as byproducts of scenario work driven by `monetization` (catalog) and `director-swarm` (gap analysis).
- **Primary mechanism:** **agents suggesting other apps from the bundle** when relevant. Do not build email-drip lifecycle campaigns for expansion — agents have better context. This is a structural advantage Vrooli should exploit.
- **Leading metric:** average number of distinct scenarios used per subscriber at M1 and M3.
- **Current state:** `pending-telemetry` — requires per-scenario usage events.
- **Aspirational target:** M1 breadth ≥ `aspirational: 1.5 scenarios/user`; M3 breadth ≥ `aspirational: 2 scenarios/user`.

### 4. Retention

The subscriber keeps paying. This is where most SaaS companies win or lose.

- **Owner:** nobody yet — requires cross-cutting telemetry we don't have. De facto, `financial-tracker` tracks aggregate retention once MRR data exists.
- **Primary mechanisms:** activation (already covered), breadth-of-adoption (expansion), product reliability, ecosystem stickiness.
- **Key signals to watch:**
  - Time from signup to first agent-driven expansion suggestion
  - Apps activated by M1 / M3 (breadth → retention correlation)
  - Support-touch rate in month 1 (high month-1 touch correlates with churn)
  - **Downgrade-to-OSS vs. hard churn** — tracked separately
- **Current state:** `pending-telemetry` for all signals.
- **Aspirational targets:**
  - Monthly gross churn ≤ `aspirational: 5%` at launch, ≤ `aspirational: 3%` steady state
  - Month-6 gross retention ≥ `aspirational: 80%`
  - Downgrade-to-free rate ≤ `aspirational: 2%/mo`, tracked separately

#### Why retention is treated first-class even pre-launch

A bundle with 100 signups and 95 churning in 30 days is worse than 30 signups with 28 retained. Default-alive math (see [FINANCIAL_MODEL.md](../evidence/FINANCIAL_MODEL.md)) collapses if retention isn't modeled. The team sets targets *before* Bundle 1 ships so there is a number to evaluate against, not after.

#### Ecosystem stickiness is a tailwind

Once a user wires agents into their daily workflow, switching cost becomes high — closer to CRM stickiness than Netflix stickiness. This means early-month retention is the hard part; late-month retention is structurally easier. Conversely, if users don't wire the workflow in during onboarding, the stickiness advantage never materializes. Back to: activation is everything.

#### Downgrade ≠ churn

A user who stops paying but continues using the OSS self-host path is **not churned from the ecosystem.** They may return as a paid customer later (when their needs change, or when hosted tier activates). Track it as `downgrade-to-free`, separate from hard churn. Different signal, different causes, different remediation.

### 5. Cross-SKU upsell

A subscriber upgrades — to a higher tier, to a bundle they didn't have, or to an add-on.

- **Owner:** collaborative — `monetization` (strategy) and `marketing-crew` (lifecycle messaging when agent surface can't reach). In-product upsell surfaces emerge as byproducts of scenario work, not from a dedicated feature team.
- **Primary mechanism:** again, agents where possible — "you're heavy on X workflow, the property-services add-on has tools for Y." Marketing-driven surfaces only where agents don't have context.
- **Leading metric:** per-SKU attach rate; tier-upgrade rate.
- **Current state:** `pending-telemetry`.
- **Aspirational target:** `aspirational: 20%` of base-bundle subscribers attach at least one add-on or upgrade a tier within 12 months.

### 6. Advocacy

A subscriber (or OSS user) refers others, publishes tutorials, or defends Vrooli publicly.

- **Owner:** partly strategic framing (see [STRATEGY.md](STRATEGY.md) — OSS-free-path is the referral engine), partly community operations when they emerge.
- **Primary mechanism:** the OSS-free-path itself plus word-of-mouth driven by ecosystem stickiness.
- **Leading metric:** referral rate, community-generated content rate.
- **Current state:** `pending-telemetry` and also pre-launch.
- **No numeric target pre-launch.** This will become a first-class stage after there are users.

## Cross-stage principles

### Agents are the expansion engine (again)

Repeated from [STRATEGY.md](STRATEGY.md) because it's load-bearing here: **default to agent-driven mechanisms for activation, expansion, and upsell before designing marketing surfaces.** The classic SaaS lifecycle-marketing playbook assumes the product has no in-workflow intelligence that can make contextual nudges. Vrooli does — use it.

### The OSS path is a funnel leak on paper, an asset in practice

Someone who runs self-host with their own keys and never pays is not a failure of the funnel. They are a node in the advocacy and acquisition machine: they tell developer friends, they write tutorials, they provide community support. The funnel model should never treat the OSS path as leakage.

### Pre-launch, measurement is mostly aspirational

Every metric in this file that's not `pending-telemetry` with a clear path is vague. That's appropriate pre-launch: you can't measure what doesn't exist. But **the team must not hallucinate current-state numbers.** When asked for a metric that isn't available, respond with the `pending-telemetry` flag, not a guess.

## Telemetry dependencies

Each metric's path to being `measured` lives in [TELEMETRY_ROADMAP.md](../evidence/TELEMETRY_ROADMAP.md). The summary:

| Metric | Requires |
|---|---|
| Landing → signup conversion | Analytics pipeline on LPBS / landing pages |
| Day-30 activation | Per-scenario activation event emission |
| M1/M3 breadth | Per-scenario usage event emission |
| Gross churn | Subscription lifecycle events from LPBS/Stripe |
| Downgrade-to-free | OSS-usage heartbeat from self-host instances (optional, user-consented) |
| Attach rate | Subscription lifecycle events + bundle/add-on entitlement state |
| Support-touch rate | Support channel integration (doesn't exist) |

The monetization team does not build these. It surfaces the gap in its heartbeat so the human knows what's missing when prioritizing scenario work.
