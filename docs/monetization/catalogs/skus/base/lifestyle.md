# Lifestyle Bundle

> Offer Desk is authoritative for this offer's lifecycle, variants, members,
> and trigger records. This document retains bundle positioning, dependency
> rationale, and expansion judgment rather than a live catalog snapshot.

**SKU ID:** `lifestyle`
**Revisit trigger:** *"Revisit when the business bundle has ≥50 paying subscribers AND at least two lifestyle-domain scenarios are deployable standalone."*
**Target audience:** Individuals and households running personal life operations — health, home, family, personal finance.
**Positioning:** A bundle that handles the non-work half of life — tracking health habits, managing household tasks, coordinating family, planning finances — with the same integrated-ecosystem advantage the business bundle offers.

## Why this bundle exists

The business bundle is the beachhead. The lifestyle bundle is the larger long-term market: almost everyone has a personal life; not everyone is a developer. Many capabilities built for the business bundle (agent orchestration, local storage, AI routing, the underlying Vrooli runtime) apply directly. The main gap is scenarios that cover personal-life domains.

## Domain scope (under definition)

Candidate domains the lifestyle bundle is expected to cover, roughly in order of likely priority once activated:

- **Health and habits** — tracking, coaching, routines, sleep, fitness
- **Household and chores** — maintenance schedules, cleaning, inspections, repairs
- **Family coordination** — shared calendars, task delegation, communication
- **Personal finance** — budgeting, subscription tracking, expense categorization (overlaps with the business bundle through Offer Desk's many-to-many `belongs_to` graph)
- **Home management** — utilities, service provider tracking, warranty/purchase records
- **Guidance and learning** — tutorials, how-tos, personalized recommendations

None of these are committed. They are the hypothesis space for when the bundle activates.

## Headliner candidates (not yet selected)

The lifestyle bundle does not yet have headliners because its domain scenarios are mostly unbuilt. When the revisit trigger fires, catalog-strategist's first task is to nominate headliner candidates that:

- Are deployable standalone with minimal Vrooli infrastructure
- Have strong standalone appeal for a non-technical user
- Reuse existing Vrooli capabilities rather than requiring wholly-new ones

Until then, this section stays a placeholder.

## Expected overlap with the business bundle

Some scenarios will belong to both bundles. Expected overlap:

- **Financial-planning scenarios** — both bundles need budgeting, expense categorization, subscription tracking
- **Calendar/scheduling** — both need time management
- **Note-taking / second brain** — both benefit from a personal knowledge base

Overlap is managed through Offer Desk's many-to-many `belongs_to` graph. No
scenario is forced to pick one bundle.

## Promotion constraints

Two reasons:

1. **Capability readiness.** Most lifestyle-domain scenarios don't exist yet. Promoting the bundle now would mean promising what we can't deliver.
2. **Focus discipline.** The business bundle has not shipped. Splitting attention across two bundles before the first one proves itself is the classic early-stage mistake. The trigger above ensures business bundle traction before lifestyle work begins.

## Candidate add-ons parented to this bundle

These offers cannot activate until this bundle itself is active; Offer Desk
holds the current lifecycle records.

- [elder-care](../addons/elder-care.md)
- [family-with-kids](../addons/family-with-kids.md)

## Consumer products tied to this bundle

Consumer products (own-produced physical/digital SKUs — print-on-demand books, planners, whiteboards, kits, courses) and affiliate recommendations are expected to be a meaningful revenue contributor for the lifestyle bundle specifically. Users are outcome-oriented, many value physical artifacts, and household / family / personal contexts legitimately involve tangible goods. See the revenue-line definitions for the hard architectural and UX rules all such offerings must obey: [consumer-products](../../revenue-lines/consumer-products.md) and [affiliate-commerce](../../revenue-lines/affiliate-commerce.md).

Gating: no scenario in this bundle activates consumer-product offers or affiliate links until (a) it has inventory-aware state to avoid offering things the user already owns, and (b) the recommendation-blindness post-processor exists in code. Both are non-negotiable.

Pattern examples (not committed SKUs, not catalog entries — patterns that illustrate fit):

- Printed cleaning / maintenance guides surfaced inside home-routines scenarios.
- Wall calendars and planners tied to the calendar scenario.
- Baby-proofing kits surfaced at the correct developmental moment, driven by inventory state (child age + current protection coverage).
- Gift-purchase surface inside the contact-book scenario — legitimate purchase intent is a natural invitation to offer products, own-produced or affiliate.

Specific SKUs get scoped and added to the catalog only when they have validated demand and meet the revenue-line's activation discipline. Until then, this section stays a list of patterns, not products.

## Open questions

Captured for future work, not to be answered now:

- Should the lifestyle bundle have a different pricing ladder than business? (Consumer-grade willingness-to-pay tends to be lower.)
- How do we handle households with multiple people on one subscription? Seat licensing or household licensing?
- Is there a free tier specifically for lifestyle-bundle scenarios to drive top-of-funnel awareness, given consumer acquisition dynamics?

These go to `market-validator` when the bundle activates.
