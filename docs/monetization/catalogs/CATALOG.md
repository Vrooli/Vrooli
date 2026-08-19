# Catalog

The SKU index for Vrooli monetization. Each SKU is a sellable unit: a **base bundle** or an **add-on**. Delivery tiers ([TIERS.md](../strategy/TIERS.md)) are orthogonal — a given SKU can be sold at any active tier.

> Offer Desk is authoritative for the current SKU, channel, revenue-line,
> variant, membership, and trigger records. This document retains the
> lifecycle semantics, promotion discipline, and guardrails; it does not
> restate the live catalog snapshot.

## SKU lifecycle

Every SKU carries a status field that flows through this lifecycle:

| Status | Meaning |
|---|---|
| `idea` | An Offer Desk record captured for consideration without an active planning commitment. |
| `candidate` | An Offer Desk record carrying a hypothesis and a machine-evaluable revisit trigger; the team does not actively plan or scope it. |
| `trigger-met` | The declared trigger evaluated as satisfied. Catalog-strategist raises a decision proposing promotion. |
| `proposed` | An agent-created promotion proposal is awaiting an operator decision; it is not an earning or active state. |
| `active` | The operator has promoted the record. The team actively plans, scopes, and prioritizes headliner sequencing. |
| `shipped` | Available for purchase. Revenue and metrics are tracked from their authoritative sources. |
| `retired` | Removed from sale or superseded; the record remains auditable. |

Promotion from `candidate` → `active` is a human work surfaced at the morning vision walk. Agents never self-promote.

**Where SKU-shaped ideas live before they enter this catalog:** the agent-side raw pool lives in the `team:monetization` Source Ledger under `monetization/opportunity/<slug>`, populated by opportunity-scout. Recall it with `source-ledger recall recall "<query>" --scope=team:monetization`. When an opportunity is broader-than-SKU OR not-yet-ready-for-active-tracking, it may instead be staged in [`path:docs/strategy/idea-pipeline/`](../strategy/idea-pipeline/) — the operator-curated, capacity-deferred staging surface for project-wide ideas. Idea-pipeline graduates SKU-shaped entries to Offer Desk candidate records when their revisit triggers fire and active tracking is warranted.

## Revisit trigger discipline

A candidate SKU's Offer Desk record **must** contain a machine-evaluable revisit trigger, while its reasoning file may explain the condition. Examples:

- *"Revisit when business bundle has ≥100 paying subscribers."*
- *"Revisit when scenario X is shipped and deployable standalone."*
- *"Revisit when ≥3 distinct prospects explicitly request this add-on."*
- *"Revisit when a design partner pays ≥$Y for a pilot."*

Triggers must reference observable, measurable events. Avoid phrasings like *"revisit when we feel ready"* or *"revisit next quarter"* — those are vibes.

The catalog-strategist's heartbeat checks each candidate's trigger against current state every tick. When a trigger fires, it raises a decision; it does not promote the SKU itself.

## Base bundles

Base bundles are represented as Offer Desk offer records. The bundle
positioning and ordering rationale remain in the individual files and in
[STRATEGY.md](../strategy/STRATEGY.md); use the Offer Desk board for their
current lifecycle state.

## Add-ons

Add-ons attach to one or more parent base bundles. Their parent relationships,
promotion state, and revisit triggers are Offer Desk records; the individual
files retain the hypotheses and decision criteria. They should never be
promoted without an explicit trigger firing and human approval.

Add-ons are illustrative examples of the pattern, not a committed roadmap. The pool expands as the opportunity-scout captures new hypotheses; most candidates never ship, and that is expected.

## Scenario membership

A single scenario can appear in multiple SKUs (e.g., a financial-planning scenario may belong to both `business` and `lifestyle`). The canonical many-to-many mapping is the Offer Desk `belongs_to` graph.

When a scenario's readiness changes (reaches headliner bar, becomes deployable, gains a prereq dependency), it may affect multiple SKUs simultaneously. The catalog-strategist reads the Offer Desk graph each heartbeat and reports per-SKU impact.

## Standing guardrails

These rules are enforced by the monetization team's TEAM.md and should be honored by anyone editing this catalog:

1. **Default focus is `active` SKUs only.** Candidate SKUs are read only to check their triggers.
2. **No candidate without a revisit trigger.** Creating or retaining a candidate record without a trigger is refused by the lifecycle gate (`GATE-001`).
3. **No add-on activation before parent bundle has paying users**, except by explicit operator decision.
4. **Agents propose promotion via decisions, never self-promote.**
5. **Scenario-to-SKU mapping must be many-to-many.** Don't force a scenario to belong to exactly one SKU; that constraint causes packaging chaos later.
6. **When retiring a SKU, mark its Offer Desk record retired and preserve the reasoning.** Historical context matters for future decisions.

## How the catalog is updated

- **New candidates** — opportunity-scout writes Source Ledger entries under `monetization/opportunity/<slug>` with required front-matter (catalog classification, revisit trigger, acquisition + retention hypotheses). Catalog-strategist periodically creates an Offer Desk promotion proposal from the scout's pool. The operator approves before the record becomes active.
- **Trigger firings** — catalog-strategist raises a decision with context `catalog-promotion`. Human decides in the vision walk.
- **Status updates** — catalog-strategist proposes status changes via decisions when milestones hit (e.g., first paying user → `shipped`). Human curates the update to this file.
- **Retirements** — contrarian or catalog-strategist may propose retirement with rationale. Human decides.

The Offer Desk board carries current catalog deltas and pending operator actions; member heartbeats read that board rather than emitting a parallel catalog snapshot.
