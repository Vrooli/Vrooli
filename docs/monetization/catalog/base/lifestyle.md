# Lifestyle Bundle

**SKU ID:** `lifestyle`
**Status:** `candidate`
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
- **Personal finance** — budgeting, subscription tracking, expense categorization (overlaps with business bundle — see [scenario-sku-map.json](../../scenario-sku-map.json))
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

Overlap is managed through the many-to-many mapping in [scenario-sku-map.json](../../scenario-sku-map.json). No scenario is forced to pick one bundle.

## Why this bundle is held in `candidate`

Two reasons:

1. **Capability readiness.** Most lifestyle-domain scenarios don't exist yet. Activating the bundle now would mean promising what we can't deliver.
2. **Focus discipline.** The business bundle has not shipped. Splitting attention across two bundles before the first one proves itself is the classic early-stage mistake. The trigger above ensures business bundle traction before lifestyle work begins.

## Candidate add-ons parented to this bundle

These are held in `candidate` state and will not activate until this bundle itself is active.

- [elder-care](../addons/elder-care.md) — `candidate`
- [family-with-kids](../addons/family-with-kids.md) — `candidate`

## Open questions

Captured for future work, not to be answered now:

- Should the lifestyle bundle have a different pricing ladder than business? (Consumer-grade willingness-to-pay tends to be lower.)
- How do we handle households with multiple people on one subscription? Seat licensing or household licensing?
- Is there a free tier specifically for lifestyle-bundle scenarios to drive top-of-funnel awareness, given consumer acquisition dynamics?

These go to `market-validator` when the bundle activates.
