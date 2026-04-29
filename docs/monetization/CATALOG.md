# Catalog

The SKU index for Vrooli monetization. Each SKU is a sellable unit: a **base bundle** or an **add-on**. Delivery tiers ([TIERS.md](TIERS.md)) are orthogonal — a given SKU can be sold at any active tier.

## SKU lifecycle

Every SKU carries a status field that flows through this lifecycle:

| Status | Meaning |
|---|---|
| `idea` | Captured informally in discussion. Not yet a dedicated doc file. |
| `candidate` | Dedicated doc file exists with hypothesis and an explicit revisit trigger. Team does not actively plan or scope. |
| `trigger-met` | Revisit trigger condition has fired. Catalog-strategist raises a decision proposing promotion. |
| `active` | Human has promoted the SKU. Team actively plans, scopes, prioritizes headliner sequencing. |
| `shipped` | Available for purchase. Revenue and metrics tracked. |
| `retired` | Removed from sale or superseded. |

Promotion from `candidate` → `active` is a human decision surfaced at the morning vision walk. Agents never self-promote.

**Where SKU-shaped ideas live before they enter this catalog:** the agent-side raw pool is `scenarios/prompt-manager/store/teams/monetization/shared/opportunities.jsonl`, populated by opportunity-scout. When an opportunity is broader-than-SKU OR not-yet-ready-for-active-tracking, it may instead be staged in [`docs/strategy/idea-pipeline/`](../strategy/idea-pipeline/) — the operator-curated, capacity-deferred staging surface for project-wide ideas. Idea-pipeline graduates SKU-shaped entries here as `candidate` files when their revisit triggers fire and active tracking is warranted.

## Revisit trigger discipline

A candidate SKU's doc file **must** contain a `Revisit trigger` field with a concrete condition. Examples:

- *"Revisit when business bundle has ≥100 paying subscribers."*
- *"Revisit when scenario X is shipped and deployable standalone."*
- *"Revisit when ≥3 distinct prospects explicitly request this add-on."*
- *"Revisit when a design partner pays ≥$Y for a pilot."*

Triggers must reference observable, measurable events. Avoid phrasings like *"revisit when we feel ready"* or *"revisit next quarter"* — those are vibes.

The catalog-strategist's heartbeat checks each candidate's trigger against current state every tick. When a trigger fires, it raises a decision; it does not promote the SKU itself.

## Base bundles

| ID | Name | Status | File |
|---|---|---|---|
| `business` | Business Bundle | `active` | [catalog/base/business.md](catalog/base/business.md) |
| `lifestyle` | Lifestyle Bundle | `candidate` | [catalog/base/lifestyle.md](catalog/base/lifestyle.md) |

**Only one base bundle is `active` at a time by default.** The business bundle is first because its audience is addressable with Vrooli's current capabilities. See [STRATEGY.md](STRATEGY.md) for ordering rationale.

## Add-ons

Add-ons attach to one or more parent base bundles. They are held in `candidate` state until their parent bundle has paying users. They should never be promoted to `active` without an explicit revisit trigger firing.

| ID | Name | Parent bundles | Status | File |
|---|---|---|---|---|
| `property-services` | Property Services | business | `candidate` | [catalog/addons/property-services.md](catalog/addons/property-services.md) |
| `elder-care` | Elder Care | lifestyle | `candidate` | [catalog/addons/elder-care.md](catalog/addons/elder-care.md) |
| `family-with-kids` | Family With Kids | lifestyle | `candidate` | [catalog/addons/family-with-kids.md](catalog/addons/family-with-kids.md) |

Add-ons are illustrative examples of the pattern, not a committed roadmap. The pool expands as the opportunity-scout captures new hypotheses; most candidates never ship, and that is expected.

## Scenario membership

A single scenario can appear in multiple SKUs (e.g., a financial-planning scenario may belong to both `business` and `lifestyle`). The canonical many-to-many mapping lives in [scenario-sku-map.json](scenario-sku-map.json).

When a scenario's readiness changes (reaches headliner bar, becomes deployable, gains a prereq dependency), it may affect multiple SKUs simultaneously. The catalog-strategist reads the sku-map each heartbeat and reports per-SKU impact.

## Standing guardrails

These rules are enforced by the monetization team's TEAM.md and should be honored by anyone editing this catalog:

1. **Default focus is `active` SKUs only.** Candidate SKUs are read only to check their triggers.
2. **No candidate without a revisit trigger.** Adding a candidate file without a trigger is a guardrail violation.
3. **No add-on activation before parent bundle has paying users**, except by explicit operator decision.
4. **Agents propose promotion via decisions, never self-promote.**
5. **Scenario-to-SKU mapping must be many-to-many.** Don't force a scenario to belong to exactly one SKU; that constraint causes packaging chaos later.
6. **When retiring a SKU, mark it retired rather than deleting the file.** Historical context matters for future decisions.

## How the catalog is updated

- **New candidates** — opportunity-scout appends ideas to `store/teams/monetization/shared/opportunities.jsonl` with proposed SKU classification. Catalog-strategist periodically proposes promotions from the scout's pool to new candidate files. Human approves before a doc file is created.
- **Trigger firings** — catalog-strategist raises a decision with context `catalog-promotion`. Human decides in the vision walk.
- **Status updates** — catalog-strategist proposes status changes via decisions when milestones hit (e.g., first paying user → `shipped`). Human curates the update to this file.
- **Retirements** — contrarian or catalog-strategist may propose retirement with rationale. Human decides.

The team's heartbeat output always includes a `Catalog deltas` section summarizing what changed and what's pending human action.
