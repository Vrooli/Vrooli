# Add-on: Property Services

**SKU ID:** `property-services`
**Status:** `candidate`
**Parent bundles:** `business`
**Revisit trigger:** *"Revisit when business bundle has ≥50 paying subscribers AND (≥3 distinct prospects explicitly request property-services tooling OR the opportunity-scout surfaces a pricing-validated lead-gen market opportunity)."*

## Hypothesis

Local service businesses — power washing, landscaping, house flipping, general contracting, solar installation, driveway sealing, roofing — face a common set of problems that Vrooli's capabilities are well-matched to:

- **Lead generation** — finding prospects in a geographic radius who match specific property signals
- **Estimating and quoting** — generating cost/time estimates from photos or specs
- **Visualization** — AI renderings of what a property looks like after the service (power-washed, landscaped, with solar installed)
- **Scheduling and dispatch** — coordinating crews and bookings
- **Customer communication** — automated follow-up, review requests

A property-services add-on would package scenarios addressing these needs. Dual-purpose: attach to a business-bundle subscriber *or* serve as the backing tool for Vrooli's own [services-led revenue line](../../REVENUE_LINES.md).

## Why this is a candidate, not active

1. Business bundle has zero paying users today. Add-on work cannot precede base-bundle traction.
2. Lead generation carries regulatory exposure (TCPA, CAN-SPAM, state-level telemarketing rules, GDPR if international). Needs a legal surface check before any services engagement ships, and a compliance-capable implementation before a subscription product ships.
3. The specific scenarios this add-on would contain are not yet built; capability reuse is high (AI image generation, data collection, agent orchestration) but integration effort is real.

## Example scenarios (illustrative, not committed)

- Power-washing lead generator — geo + property-age + visible-dirt-signal scanning
- Driveway/siding/roof visualization renderer — before/after AI imagery + validated cost estimates
- Solar savings calculator + visualization — takes a house photo, estimates panel placement, projects savings with utility data
- Flip opportunity scanner — distressed-property detection + comp analysis + renovation-cost estimates
- Crew dispatch and scheduling — small-business ops layer

These would be designed only after the trigger fires; specifics depend on which vertical shows strongest demand signal.

## Cross-use with services-led revenue line

This add-on is one of the strongest candidates for the **services → subscription** pattern:

1. Build lead-gen scenarios internally; run them as a service for 3-5 local businesses.
2. Prove unit economics and regulatory compliance.
3. Productize the tooling into the add-on.
4. Convert pilot clients to add-on subscription once the tool carries the work without our hands on it.

See [REVENUE_LINES.md](../../REVENUE_LINES.md) for services-engagement discipline.

## Things to track while candidate

Not active work, but signals the team watches:

- Explicit requests from prospects for this kind of tooling (logged as monetization knowledge under `monetization/opportunity/property-services-*` or `opportunity-inbox/customer-ask/*`)
- Scenarios that become deployable and are reusable for this add-on (e.g., image-generation service, geo-data resource)
- Regulatory changes that materially affect lead-gen economics
- Competitor activity in this space (market-validator notes under `monetization/market-scan/*` knowledge entries)

None of these cause automatic promotion; they inform the trigger review.

## If and when promoted to `active`

On promotion, these are the first-pass questions the catalog-strategist and market-validator must answer before any scenarios get scoped:

1. Which vertical(s) within property services? (Power washing is not roofing is not solar.)
2. What's the pricing envelope? (Lead-gen typically priced per-lead or per-close; visualization could be flat-rate.)
3. What's the minimum viable tool that a services engagement could run?
4. What's the competitive landscape in the chosen vertical? (Angi, HomeAdvisor, Thumbtack for lead-gen; specialty tools for visualization.)
5. What's the legal surface for this vertical? (Contractor licensing, contracting contract law, telemarketing rules.)

These questions are not to be answered now. They are queued for the activation moment.
