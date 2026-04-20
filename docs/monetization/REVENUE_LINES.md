# Revenue Lines

The **ways Vrooli makes money** — distinct from the catalog (what we sell) and tiers (how it's delivered). Each revenue line has its own cost structure, discipline, and status lifecycle.

## Why revenue lines are first-class

A company with only subscriptions has one revenue line. Vrooli has multiple — subscriptions are the product line, but done-for-you services, lead-gen, and consulting are legitimate near-term revenue contributors that also validate product capability. Treating them as a single undifferentiated "revenue" pool hides unit economics and creates management drift.

## Revenue-line status lifecycle

| Status | Meaning |
|---|---|
| `candidate` | Documented with hypothesis and revisit trigger; not actively producing revenue. |
| `active` | Currently running; producing revenue. |
| `sunset` | Winding down — either productized into a SKU or being abandoned. |
| `retired` | Wound down. Kept in the doc for history and future lessons. |

## Strategic role: services as a deliberate lever

Each Vrooli scenario is a **double-revenue asset** — the same capability can be sold as a product AND operated by us to deliver paid work for clients. See [STRATEGY.md §3](STRATEGY.md#3-services-are-a-deliberate-lever-not-a-business) for the full framing. The catalog and the services pipeline are not separate programs; they draw from the same well of capabilities.

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
- **Services capacity ≤ 30% of time budget.** See [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md). Exceeding this for 3+ consecutive weeks triggers a services-trap review.
- **Services revenue tracked separately from subscription revenue.** Sustained crossover (services > subs for 2+ months) is flagged to the vision walk.
- **Convert when product-ready AND client-trust-built.** Not earlier, not later. Earlier = churn from disappointment. Later = we keep doing manual work we don't need to.

## Active lines

### Subscription (the product)

- **Status:** `active` (pre-launch, but this is the primary line)
- **Revenue model:** monthly / annual recurring, per-tier, per-bundle
- **Cost structure:** see per-tier COGS in [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md)
- **Current state:** zero subscribers. Tier 1 (Bundle apps) in progress; other tiers are candidate/north-star.
- **Productization target:** this IS the product. No bridging; it's the destination all services lines aim toward.

## Candidate lines

### Lead generation (for local service businesses)

- **Status:** `candidate`
- **Revisit trigger:** *"Revisit when at least one property-services scenario is deployable as a thin tool AND one local-service prospect signs a pilot agreement."*
- **Hypothesis:** Local service businesses (power washing, landscaping, flippers, solar installers, roofers) will pay per-lead or per-close for geo-targeted leads generated by Vrooli's data + AI capabilities. If it works for us, we can productize the underlying tool into the `property-services` add-on.
- **Productization target:** [`property-services` add-on](catalog/addons/property-services.md)
- **Legal surface:** TCPA (US telemarketing), CAN-SPAM (email), GDPR (if international), state-level rules on B2B vs B2C lead sales. Non-trivial; requires explicit legal review before first paid engagement.
- **Notes:** This is the strongest candidate services line because (a) property-services add-on capability reuse is high, (b) immediate revenue is realistic, (c) the services → subscription conversion path is clean.

### Standalone app development (done-for-you builds)

- **Status:** `candidate`
- **Revisit trigger:** *"Revisit when a prospect with a concrete app spec offers ≥$Y for a fixed-scope build AND the capability to deliver it exists in current Vrooli scenarios."*
- **Hypothesis:** People want specific apps built and don't want to operate agents themselves. Vrooli's automated-software-engineering capabilities can deliver clean apps faster than a traditional dev shop. Each engagement validates the generate-apps flow and potentially seeds the client as a subscriber (they see what Vrooli can do).
- **Productization target:** generic app-generation tooling — which may become its own scenario if volume warrants.
- **Legal surface:** client contracts (work-for-hire, IP ownership, warranty, liability caps). Standard but real.
- **Notes:** Highest execution risk among services lines (scope creep, support burden, expectations management). Best as a validation engagement for one or two high-signal customers, not as a volume business.

### Consulting / strategy engagements

- **Status:** `candidate` (lowest priority)
- **Revisit trigger:** *"Revisit only if a specific prospect offers a time-boxed consulting engagement that explicitly validates a Vrooli capability we want to prove AND the operator has clear capacity."*
- **Hypothesis:** People sometimes pay for expertise independent of tooling. In Vrooli's case this would usually mean advising on AI-powered workflow design, agent orchestration architecture, or local-first software strategy.
- **Productization target:** often none — consulting is the hardest to productize cleanly because every engagement is bespoke. When it does productize, it's usually into playbooks, templates, or a specialized SKU rather than a clean subscription.
- **Notes:** **Treat as last resort.** Consulting has the highest distraction-to-revenue ratio among services lines. The contrarian should be especially skeptical of new consulting proposals.

## How services lines get activated

1. `opportunity-scout` captures candidate services lines as they emerge from market or prospect conversations; logs to `store/teams/monetization/shared/opportunities.jsonl` with `kind: services-line`.
2. `catalog-strategist` reviews the pool periodically; when a candidate's trigger fires, raises a decision with context `services-activation`.
3. `contrarian` reviews the proposal and challenges it specifically against the trap conditions: hypothesis, fixed duration, productization target, sunset clause, legal surface, time-capacity implications.
4. Operator decides at the vision walk. If promoted to `active`, a line entry is added to the active section here and `financial-tracker` begins tracking separately.

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
