# Telemetry Roadmap

A living list of **metrics the monetization team needs** and the **capabilities that would make them `measured` instead of qualitative**. The purpose of this doc is to (a) make the qualitative-vs-quantitative gap visible, (b) point at which existing scenario most likely hosts each capability, and (c) provide a grep-able migration list (`REPLACES-MANUAL`) so that when a capability ships, someone can find every prompt/metric that changes.

This file is **not** a build plan. It identifies gaps; swarm-manager and director-swarm decide when to close them.

## Core principles

1. **Build nothing here until bundles ship.** Qualitative reasoning with honest pending-flags is fine pre-launch. Telemetry scenarios built before they're needed are premature infrastructure.
2. **Prefer extending existing scenarios over creating new ones.** Most needs can be served by adding events/endpoints to LPBS, scenario-to-cloud, or prompt-manager. A new telemetry scenario is a last resort.
3. **Emit first, query later.** When a scenario ships with a relevant event, it should emit the event even if no consumer exists. This seeds data without requiring a query capability to exist simultaneously.
4. **Shared analytics store.** When multiple scenarios need to emit similar events, a single shared resource (events store) is better than per-scenario telemetry. Proposed but not built.

## REPLACES-MANUAL migration list

Agents' prompts and this team's docs carry `REPLACES-MANUAL` markers wherever qualitative reasoning would be replaced by a structured query if the capability existed. When a capability ships, search the repo for the marker and update the affected prompts/docs.

To find them:

```bash
grep -rn "REPLACES-MANUAL" docs/monetization/ scenarios/prompt-manager/store/teams/monetization/
```

## Capability gaps

Each gap lists: the metrics it unblocks, the most likely host scenario, and any downstream effects.

### Gap 1: Per-scenario usage / activation telemetry

- **Unblocks:** Day-30 activation rate, M1/M3 breadth of adoption, time-to-first-win, feature adoption by scenario
- **Most likely host:** New shared resource (events store) + thin query scenario. Alternative: extend prompt-manager or swarm-manager to collect scenario-level events.
- **Shape required:** Every scenario emits well-typed events (`scenario.open`, `scenario.action`, `scenario.complete`) with subscriber identity tag when available.
- **Downstream effect:** Collapses ~4 `pending-telemetry` flags in [FUNNEL.md](../strategy/FUNNEL.md) and the retention portion of [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md).
- **Priority signal to watch:** activates when business bundle has active users and activation measurement becomes the highest-leverage unknown.

### Gap 2: Subscription lifecycle events (signup, upgrade, downgrade, churn)

- **Unblocks:** MRR, ARR, monthly gross churn, downgrade-to-free rate, NRR, cohort retention
- **Most likely host:** Extend **landing-page-business-suite** to emit lifecycle events. LPBS already integrates Stripe via `lpbs-stripe-monetization-assurance`; this is an incremental extension, not new infrastructure.
- **Shape required:** `subscription.created`, `subscription.upgraded` (with from/to tier), `subscription.downgraded`, `subscription.canceled`, `subscription.reactivated` events — with bundle, tier, and anonymized user identity.
- **Downstream effect:** Collapses most `pending-telemetry` flags in the financial model and funnel retention section.
- **Priority signal:** activates when Tier 1 ships and first subscribers exist.

### Gap 3: Infrastructure cost aggregation per scenario

- **Unblocks:** Per-tier COGS validation, per-scenario margin analysis, default-alive sensitivity
- **Most likely host:** Extend **scenario-to-cloud** to expose a query API for deployment costs per scenario per tenant.
- **Shape required:** Monthly cost aggregation with category breakdown (compute, storage, bandwidth, gateway tokens).
- **Downstream effect:** Turns the per-tier COGS section of [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md) from `estimate` to `measured`.
- **Priority signal:** activates when Tier 3 (hosted) activates — that's when COGS accuracy matters most.

### Gap 4: API gateway usage / metering

- **Unblocks:** Per-subscriber gateway cost attribution, outlier heavy-user detection, margin-per-subscriber, pricing adjustment signal
- **Most likely host:** The API gateway itself — could be part of LPBS, scenario-to-cloud, or its own resource. Directly tied to Tier 2 activation prereqs (see [TIERS.md](../strategy/TIERS.md)).
- **Shape required:** Per-request metering with subscriber, model, provider, token counts, cost.
- **Downstream effect:** Enables metered pricing, heavy-user throttling, honest per-subscriber margin math.
- **Priority signal:** activates when Tier 2 (self-hosted-with-gateway) is near-shipping — gateway IS the tier's core capability.

### Gap 5: Landing-page traffic / signup attribution

- **Unblocks:** Landing → signup conversion rate, channel attribution (which acquisition source works), bundle-page comparative performance
- **Most likely host:** Extend **landing-page-business-suite** with a thin analytics pipeline. Could integrate a third-party tool if self-built isn't worth the time.
- **Shape required:** Page visit events with referrer, conversion events linking a signup to a session.
- **Downstream effect:** Collapses the Acquisition `pending-telemetry` in [FUNNEL.md](../strategy/FUNNEL.md).
- **Priority signal:** activates when the first bundle landing page ships.

### Gap 6: Support-touch tracking

- **Unblocks:** Month-1 support-touch rate (strong churn predictor), support-load-per-subscriber, quality-issue topic analysis
- **Most likely host:** Requires a support channel integration — none exists today. When support volume materializes, a scenario (support-inbox?) may be justified.
- **Shape required:** Inbound support event with subscriber identity, categorized topic.
- **Priority signal:** activates when support volume becomes a real thing (i.e., when there are subscribers). Before that, this metric is uninteresting.

### Gap 7: OSS self-host usage ping (optional, user-consented)

- **Unblocks:** Distinguishing downgrade-to-free from hard-churn, sizing the OSS-goodwill asset
- **Most likely host:** The Vrooli runtime itself emits an anonymous heartbeat that a paying-but-moved-to-free user could continue to send voluntarily.
- **Shape required:** Fully opt-in, fully anonymized, honors OSS principles.
- **Downstream effect:** Separates the downgrade-to-free signal in the funnel from hard churn.
- **Priority signal:** low priority until there is a meaningful OSS user base. Do not build this ahead of demand.

### Gap 8: Market-benchmark data feed

- **Unblocks:** Less-manual market-validator work, pricing-adjustment signals from competitor changes
- **Most likely host:** Probably never automated in a meaningful way. Market-validator manually curates [BENCHMARKS.md](BENCHMARKS.md). Some lightweight automation (competitor pricing-page scraping) may be worth it later but this is low priority.
- **Priority signal:** only if market-validator consistently reports benchmarks as the bottleneck.

## Deferred-decision summary

The team should propose building these in roughly this order as the business grows:

1. Gap 2 (subscription lifecycle) — when Tier 1 ships
2. Gap 5 (landing-page analytics) — when first landing page ships
3. Gap 1 (per-scenario telemetry) — when activation measurement becomes the blocking gap
4. Gap 3 (infra cost aggregation) — when Tier 3 activates or hosting cost becomes material
5. Gap 4 (gateway metering) — when Tier 2 activates
6. Gap 6 (support-touch) — when support is a real channel
7. Gap 7 (OSS ping) — if and when OSS user base warrants it
8. Gap 8 (benchmark feed) — if and when market-validator is clearly bottlenecked

Each of these should be re-justified at activation time. None should be built pre-emptively.

## How agents should handle the current gap

While most gaps are open, the monetization team members produce qualitative outputs with **explicit labels**. When a prompt tells a member to "estimate infrastructure cost," the output is labeled `estimate: X` with a pointer to this file. When telemetry eventually closes a gap, prompts with `REPLACES-MANUAL` markers are updated, and the qualitative step becomes a query.

This way the team is productive today and migrates cleanly tomorrow.
