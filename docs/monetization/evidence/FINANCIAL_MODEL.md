# Financial Model

The durable financial structure: how costs work, how revenue works, how runway is computed, what "default-alive" means for Vrooli. Specifics — actual numbers — are read from Money Ledger and operator measures in the `team:monetization` Source Ledger scope. This file holds the *framework*, not the current snapshot.

## Honesty conventions

Every number in this file carries a label:

- **`fixed`** — a decided value (e.g., target default-alive runway threshold).
- **`estimate`** — operator's best qualitative guess, used until telemetry exists.
- **`aspirational`** — a pre-launch target with no current data.
- **`measured`** — from real telemetry; check the ledger for source.
- **`pending-telemetry`** — cannot be known until a capability in [TELEMETRY_ROADMAP.md](TELEMETRY_ROADMAP.md) exists.

If a number in this file lacks a label, it is broken — flag it.

## Cost structure (per-tier COGS)

Cost-of-goods-sold differs dramatically across tiers. Mixing them obscures unit economics and leads to wrong pricing decisions.

### Tier 1 (Bundle apps) — gateway-driven variable cost

- Build/distribute cost is fixed (app store fees, signing certs, CDN). Amortized across subscribers.
- No per-user hosting infrastructure (the apps run on the user's device).
- **Gateway token cost is the dominant variable per-user line.** Paid Tier 1 subscriptions include the integrated API gateway (LLMs, STT/TTS, embeddings, coding agents) with a credit allowance — that IS the core reason to pay rather than running the OSS apps with bring-your-own keys. Every paid Tier 1 subscriber drives gateway usage at wholesale-to-retail pass-through margin.
- **Unit economics:** margin = subscription price − (wholesale-token-cost × user's consumption up to allowance) − (overage-cost at markup beyond allowance). `estimate`: the mainstream Tier 1 user consumes a small fraction of what the heaviest 5% use; credit allowances + metered overage cap the negative-margin tail.
- **Distinction from Tier 2:** Tier 2 adds a full local runtime (cross-scenario context sharing, agent coordination, deeper customization) on top of the same gateway. Same variable-cost shape; higher value to the user; higher price.

### Tier 2 (Self-hosted) — gateway-driven variable cost

- Variable cost: API token pass-through (we buy wholesale, sell retail through the gateway).
- Fixed cost: gateway infrastructure (small), license/entitlement service.
- Zero hosting cost — the user provides the hardware.
- **Unit economics:** margin = (subscription price) − (wholesale-token-cost × user's usage). Heavy users can be low-margin or negative; pricing should cap or meter usage. `estimate`: mainstream user consumes far less gateway than heaviest 5%.

### Tier 3 (Hosted cloud) — infrastructure-driven variable cost

- Variable cost: per-user VPS / container / storage / bandwidth + gateway tokens.
- **Unit economics:** materially tighter than Tier 2. Hosted users cost `estimate: 3-5×` more to serve than Tier 2 users. Pricing must reflect this. Do not discount Tier 3 below sustainable margin to drive adoption.

### Tier 4 (Hardware) — BOM + fulfillment

- Variable cost: hardware BOM, shipping, warranty/RMA.
- Fixed cost: certifications, support infrastructure, inventory warehousing.
- **Unit economics:** hardware margin + subscription margin combined. Entirely different financial shape — not modeled here until the operator chooses to enter this business.

## Revenue line cost structure

Revenue lines (see [REVENUE_LINES.md](../catalogs/revenue-lines/README.md)) have their own cost characteristics:

| Line | Cost shape |
|---|---|
| Subscriptions | Per-tier COGS above. |
| Services (done-for-you) | Time — the single largest cost. Opportunity cost of displaced product work is often larger than the services revenue. |
| Lead generation | Data-acquisition cost + regulatory-compliance cost (TCPA, CAN-SPAM, GDPR). May or may not run through gateway. |
| Consulting | Time-only, plus minimal overhead. Highest ratio of time-cost to revenue among services lines. |

## Runway formula

```
runway_months = cash_on_hand / (monthly_burn − monthly_revenue)
```

When `monthly_revenue ≥ monthly_burn`, the company is **default-alive** — runway is infinite independent of funding.

### Default-alive target

`fixed`: Vrooli's primary financial goal is reaching default-alive. Specifically:

- **Minimum default-alive threshold:** monthly recurring revenue meets or exceeds monthly burn, consistently, for `fixed: 3 consecutive months`.
- **Target default-alive with buffer:** monthly recurring revenue ≥ monthly burn × `fixed: 1.25`, providing headroom for cost drift.

Until the first threshold is crossed, every decision the monetization team raises should be weighed against its impact on the default-alive date.

### Burn composition

Burn is categorized so changes are attributable:

| Category | Description | Input authority |
|---|---|---|
| AI/API | Model tokens, gateway pass-through at wholesale, STT/TTS, embeddings | Operator enters `monthlyBurn.aiApi` in Money Ledger `/adapters` until the API gateway (Tier 2 prereq) aggregates automatically |
| Infrastructure | VPS, storage, CDN, DNS, backups | Operator enters `monthlyBurn.infrastructure` in Money Ledger `/adapters` until `scenario-to-cloud` cost API lands |
| Third-party SaaS | Stripe fees, email/transactional, analytics | Operator enters `monthlyBurn.saas` in Money Ledger `/adapters` |
| Tooling | Dev tools, CI runners, monitoring | Operator enters `monthlyBurn.tooling` in Money Ledger `/adapters` |
| Personnel | Operator's time, contractors if any | Operator tracks separately through the `timeAllocation` fields in Money Ledger `/adapters` |

Money Ledger's operator-input adapter reads these fields from the `/adapters`
submission and preserves `pending-operator` as absent. Gathering guidance per
category lives in [HOW_TO_GATHER_INPUTS.md](../governance/HOW_TO_GATHER_INPUTS.md).
The live position, goal verdicts, and observation availability are read from
Money Ledger; this document remains the framework.

## Revenue shape

### Subscription revenue

Monthly recurring revenue (MRR) = sum of active subscription prices.
Annual recurring revenue (ARR) = MRR × 12.

Net revenue retention (NRR) — the compound metric that captures upgrades, downgrades, and churn:

```
NRR = (starting_mrr + expansion − contraction − churn) / starting_mrr
```

NRR ≥ 100% means the existing base grows revenue faster than it loses it. `aspirational`: Vrooli targets NRR ≥ `aspirational: 110%` within 18 months of launch, driven by tier upgrades (T1→T2→T3) and add-on attach. `pending-telemetry` until MRR exists.

### Services revenue

Tracked separately from subscription revenue. `fixed`: if services revenue exceeds subscription revenue for `fixed: 2 consecutive months`, this is flagged as a services-trap warning signal for the morning vision walk. See [REVENUE_LINES.md](../catalogs/revenue-lines/README.md).

### Lifetime value

```
LTV = ARPU / monthly_churn_rate
```

Example sensitivity (`aspirational`, for illustration only):
- ARPU $49/mo, 5% monthly churn → LTV $980
- ARPU $49/mo, 3% monthly churn → LTV $1,633
- ARPU $49/mo, 2% monthly churn → LTV $2,450

**Retention drives LTV more than pricing does at reasonable price points.** This is why activation and breadth-of-adoption investments (see [FUNNEL.md](../strategy/FUNNEL.md)) are not marketing-optional. `pending-telemetry` for actual values.

## Time allocation — a first-class cost

For a one-human operation, **time is the dominant cost**, not dollars. The financial model tracks:

- Time spent on product (builds capability, compounds)
- Time spent on services (generates immediate cash, does not compound unless productization happens)
- Time spent on ops (recurring, can only be reduced through automation)

`fixed` guardrail: **services capacity ≤ 30% of time budget** unless explicitly overridden by the operator. Exceeding this for `fixed: 3 consecutive weeks` triggers a services-trap review.

`financial-tracker` reads time allocation as an operator measure from Money Ledger
alongside dollar costs. Do not omit time from the model or recreate it in this
document.

## Key assumptions (subject to revision)

These are the load-bearing assumptions in the model. Market-validator and contrarian should challenge them when data lands:

1. `estimate`: Majority of paying subscribers will choose Tier 2 (self-hosted) over Tier 1 (bundle apps) within 12 months of offering Tier 2, because the integrated runtime experience is materially better.
2. `estimate`: Tier 3 (hosted) attach rate among non-technical users will be high — they cannot realistically self-host.
3. `aspirational`: Net revenue retention ≥ 110% within 18 months, driven by tier upgrades + add-on attach.
4. `estimate`: Services lines convert to subscription at ≥ 40% when **both** conditions hold: (a) the product replaces the manual work without new support burden (productization is actually done), AND (b) the client has built enough trust in the tool to stay subscribed without our hands on it. Converting before (a) produces churn from disappointment plus new support load; converting long after (a) keeps the operator doing manual work that blocks the next services client. Both factors are operational discipline, not assumption — the 40% is the forecast when discipline holds.
5. `estimate`: Services engagements longer than ~3 months without productization handoff are approaching the services trap.

When any assumption changes materially, the financial-tracker raises a knowledge entry with topic `financial-model-assumption-update`.

## Financial posture read contract

`financial-tracker` reads the current position, goals, operator measures, and
availability qualifications from Money Ledger. The read includes:

- Cash on hand and its observation basis and age
- MRR per tier and bundle, or an explicit pending-telemetry qualification
- Costs by category (AI/API, infrastructure, SaaS, and tooling)
- Time allocation (product / services / ops)
- Runway and the default-alive gap when the required observations are complete
- Deltas and guardrail flags derived from the current read

Money Ledger remains the authority for current values and availability reasons.
When a source is absent, stale, or unavailable, the financial tracker records
that qualification and does not reconstruct a snapshot from this document.

## When to revisit this model

This file should be updated when:

- A key assumption is invalidated by data
- A new cost category emerges that doesn't fit existing ones
- A new revenue line is activated
- A new tier or SKU is activated, changing COGS shape
- The default-alive threshold is crossed (switch from "path to" to "maintain")

Agents propose updates via decisions; operator curates.
