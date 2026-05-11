# Revenue Lines

The **ways Vrooli makes money** — distinct from the catalog (what we sell) and tiers (how it's delivered). Each revenue line has its own cost structure, discipline, and status lifecycle, and lives in its own file in this folder.

## Why revenue lines are first-class

A company with only subscriptions has one revenue line. Vrooli has multiple — subscriptions are the product line, but done-for-you services, lead-gen, and consulting are legitimate near-term revenue contributors that also validate product capability. Treating them as a single undifferentiated "revenue" pool hides unit economics and creates management drift.

## Revenue-line status lifecycle

| Status | Meaning |
|---|---|
| `candidate` | Documented with hypothesis and revisit trigger; not actively producing revenue. |
| `active` | Currently running; producing revenue. |
| `sunset` | Winding down — either productized into a SKU or being abandoned. |
| `retired` | Wound down. Kept in the folder for history and future lessons. |

## Index

| ID | Name | Status | File |
|---|---|---|---|
| `subscription` | Subscription (the product) | `active` (pre-launch) | [subscription.md](subscription.md) |
| `lead-generation` | Lead generation for local service businesses | `candidate` | [lead-generation.md](lead-generation.md) |
| `app-development` | Standalone app development (done-for-you) | `candidate` | [app-development.md](app-development.md) |
| `consulting` | Consulting / strategy engagements | `candidate` (last resort) | [consulting.md](consulting.md) |
| `consumer-products` | Consumer products (own-produced SKUs — books, planners, kits) | `candidate` | [consumer-products.md](consumer-products.md) |
| `affiliate-commerce` | Affiliate / commerce (partner-produced, commission-based) | `candidate` | [affiliate-commerce.md](affiliate-commerce.md) |
| `flipping` | Flipping (marketplace resale — buy underpriced, refurbish, resell) | `candidate` | [flipping.md](flipping.md) |

New candidates enter by adding a file here via an approved decision. Retired lines stay in the folder with `Status: retired` — historical context matters for future decisions.

### Channels are tracked separately

Discovery channels (web SEO, app stores, skill registries, OSS discovery, community content, in-product expansion) are not revenue lines. They are *paths into* revenue lines and live as their own first-class axis with their own status lifecycle, owners, and telemetry. See [`channels/`](../channels/README.md) for the full set.

## Cross-cutting architectural rule for recommendation-adjacent lines

Both `consumer-products` and `affiliate-commerce` share a hard architectural constraint that applies to any scenario participating in them:

> **The agent producing a recommendation must not know what Vrooli sells or earns commission on. Offer insertion and affiliate link rewriting are strictly post-processing steps over recommendations that were already made.**

This rule protects the authority layer — the ground truth that lifestyle-bundle users pay for. Violating it once corrupts every recommendation downstream of that moment and is not fixable by apology. Each scenario that activates either line must implement the separation structurally (in code, in data flow, in review boundaries), not as a policy. See the individual files for the full constraint set and UX discipline.

## Strategic role: services as a deliberate lever

Each Vrooli scenario is a **double-revenue asset** — the same capability can be sold as a product AND operated by us to deliver paid work for clients. See [STRATEGY.md §3](../../strategy/STRATEGY.md#3-services-are-a-deliberate-lever-not-a-business) for the full framing. The catalog and the services pipeline are not separate programs; they draw from the same well of capabilities.

The strategic value of services is the **timing asymmetry**: they generate cash in chunks, upfront, while subscription revenue compounds slowly. During the window between core bundles shipping and subscriptions crossing default-alive, services are expected to be a primary revenue lever, not a sidebar.

**Phase posture:**

- **Pre-bundle (current state):** all services lines remain `candidate`. Each revisit trigger is tied to a specific capability being deployable as a thin tool. Don't activate out of turn.
- **Post-bundle, pre-default-alive:** services are expected to actively produce revenue. This is the window where the `active` count should be non-zero and conversion rates matter most.
- **Post-default-alive:** services wind down or productize. Success means subscriptions have made them unnecessary; a services line that persists past this phase without converting is a signal that the corresponding SKU hasn't matured.

**Converting is also a capacity decision.** Conversion isn't only about trust and product readiness — it's how we free operator time to take on the next services client. An active line with no clients converting isn't just a productization stall; it's a capacity stall that blocks the next engagement. Conversely, converting before the product is ready transfers manual work from a paid-services client to an unpaid-support burden on the product team. Both failure modes are tracked.

The discipline below exists **because we intend to lean into this lever, not to avoid it.** The guardrails are there to keep services from reorienting the company into a consultancy — not to suppress services activity itself.

## Standing discipline (applies to all services-based lines)

Services lines (anything that's not pure subscription sales) carry a **fundamental risk: the services trap.** Without discipline, the same timing asymmetry that makes services strategically valuable also pulls the organization toward them — higher immediate revenue, faster deals, customer-driven roadmap — and the subscription product starves.

Every active services line must have all four of these, or it is a guardrail violation:

1. **Validation hypothesis** — which capability or market is this engagement proving?
2. **Fixed-duration pilot** — a concrete end date or milestone, not "ongoing."
3. **Productization target** — which SKU in the catalog does this feed when it succeeds?
4. **Sunset or convert clause** — by date X, productize and hand off to subscription, or stop.

The success metric for a services line is **service-client → subscriber conversion rate**, not services revenue itself. Conversion happens when (a) the product replaces the manual work without new support burden, AND (b) the client has built trust in it. Both.

Additional guardrails:

- **Legal surface check before activation.** Every services line has regulatory exposure that differs from pure subscriptions (TCPA/CAN-SPAM for lead-gen, contract/IP for consulting, warranty for done-for-you work).
- **Services capacity <= 30% of time budget.** See [FINANCIAL_MODEL.md](../../evidence/FINANCIAL_MODEL.md). Exceeding this for 3+ consecutive weeks triggers a services-trap review.
- **Services revenue tracked separately from subscription revenue.** Sustained crossover (services > subs for 2+ months) is flagged to the vision walk.
- **Convert when product-ready AND client-trust-built.** Not earlier, not later. Earlier = churn from disappointment. Later = we keep doing manual work we don't need to.

## How services lines get activated

1. `opportunity-scout` captures candidate services lines as they emerge from market or prospect conversations; creates a knowledge entry under `monetization/opportunity/<slug>` with `kind: services-line-candidate` in the required front-matter.
2. `catalog-strategist` reviews the pool periodically; when a candidate's trigger fires, raises a decision with context `services-activation`.
3. `contrarian` reviews the proposal and challenges it specifically against the trap conditions: hypothesis, fixed duration, productization target, sunset clause, legal surface, time-capacity implications.
4. Operator decides at the vision walk. If promoted to `active`, the line's file is updated (`Status: active`) and `financial-tracker` begins tracking separately.

## Active services line instrumentation (for when one activates)

Each active services line reports in the ledger:

- Current client count
- Revenue this month / cumulative
- Time spent this month / cumulative
- Elapsed time since activation vs. productization target date
- Conversion count (clients who moved to subscription)
- Conversion rate (conversion count / total client count)
- Gap to productization (is the underlying tool ready? if not, what's missing?)

`financial-tracker` rolls these up into the default-alive calculation.

## What's NOT a revenue line

For clarity, some things that sometimes get proposed as revenue lines but shouldn't be treated as such:

- **Ads.** Vrooli's positioning (local-first, user-controlled, ecosystem) is fundamentally at odds with ad-supported models. Proposals in this direction should be rejected by contrarian by default.
- **Data sales.** Same reason. User data is not a product; it's a trust commitment.
- **Marketplace transactions (scenarios from third-party authors).** Could become a revenue line eventually, but the infrastructure to do it responsibly doesn't exist today. Captured as an idea, not a candidate, until that infrastructure is real.
