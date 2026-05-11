# Monetization — Plan of Record

This folder is the **plan of record** for Vrooli's monetization system: SKU catalog, delivery tiers, pricing posture, revenue lines, acquisition channels, funnel economics, financial assumptions, benchmarks, and validation taxonomies.

It is maintained by the `monetization` team and consumed by its members every heartbeat. The team's live operating rules are at [`scenarios/prompt-manager/store/teams/monetization/shared/TEAM.md`](../../scenarios/prompt-manager/store/teams/monetization/shared/TEAM.md); this folder is the strategic-canon side.

The local contract is [`manifest.json`](manifest.json), which instantiates the shared plan-of-record shape from [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json).

## Start here for agents

Use this README first, then choose the module that matches the work:

| Question | Start with |
|---|---|
| How does the monetization team operate end to end? | [`operating/OPERATING_MODEL.md`](operating/OPERATING_MODEL.md) |
| What is the monetization posture or strategic rule? | [`strategy/STRATEGY.md`](strategy/STRATEGY.md) |
| Which delivery tier or deployment mode matters? | [`strategy/TIERS.md`](strategy/TIERS.md) |
| What price should a surface cite? | [`strategy/PRICING.md`](strategy/PRICING.md) |
| Which funnel stage or metric is blocked? | [`strategy/FUNNEL.md`](strategy/FUNNEL.md) or [`evidence/TELEMETRY_ROADMAP.md`](evidence/TELEMETRY_ROADMAP.md) |
| What exactly can be sold or bundled? | [`catalogs/CATALOG.md`](catalogs/CATALOG.md) |
| What revenue line is being evaluated or productized? | [`catalogs/revenue-lines/README.md`](catalogs/revenue-lines/README.md) |
| Which acquisition or distribution channel applies? | [`catalogs/channels/README.md`](catalogs/channels/README.md) |
| What assumptions drive runway or default-alive analysis? | [`evidence/FINANCIAL_MODEL.md`](evidence/FINANCIAL_MODEL.md) |
| What market benchmark is accepted canon? | [`evidence/BENCHMARKS.md`](evidence/BENCHMARKS.md) |
| What operator input should be requested or interpreted? | [`governance/HOW_TO_GATHER_INPUTS.md`](governance/HOW_TO_GATHER_INPUTS.md) |
| How are opportunities classified and routed? | [`taxonomies/monetization-opportunity/README.md`](taxonomies/monetization-opportunity/README.md) |
| How are validation requests classified and routed? | [`taxonomies/monetization-validation/README.md`](taxonomies/monetization-validation/README.md) |

## Folder map

| Folder | Purpose |
|---|---|
| [`operating/`](operating/README.md) | Team operating contract and validation commands. |
| [`strategy/`](strategy/README.md) | Monetization strategy, tiers, pricing, and funnel canon. |
| [`catalogs/`](catalogs/README.md) | SKU catalog, scenario-to-SKU map, revenue-line registry, and channel registry. |
| [`taxonomies/monetization-opportunity/`](taxonomies/monetization-opportunity/README.md) | Opportunity signal taxonomy plus machine-readable `taxonomy.json`. |
| [`taxonomies/monetization-validation/`](taxonomies/monetization-validation/README.md) | Validation-request taxonomy plus machine-readable `taxonomy.json`. |
| [`evidence/`](evidence/README.md) | Financial model, benchmarks, and telemetry roadmap. |
| [`governance/`](governance/editing.md) | Editing authority, adoption validation, changelog, and operator-input guidance. |

## Organizing principle

Monetization is described across five orthogonal axes:

1. **WHAT** we sell: [`catalogs/CATALOG.md`](catalogs/CATALOG.md), SKU package files, and [`catalogs/scenario-sku-map.json`](catalogs/scenario-sku-map.json).
2. **HOW** it is delivered: [`strategy/TIERS.md`](strategy/TIERS.md).
3. **WHERE USERS COME FROM:** [`catalogs/channels/README.md`](catalogs/channels/README.md).
4. **WHO** flows through: [`strategy/FUNNEL.md`](strategy/FUNNEL.md).
5. **HOW WE MAKE MONEY:** [`catalogs/revenue-lines/README.md`](catalogs/revenue-lines/README.md).

Pricing is the intersection of WHAT and HOW. The financial model projects runway, default-alive position, and per-tier economics from those inputs.

## Editing rules

- **Agents never write to plan-of-record canon directly.** All canon edits come through operator-approved decisions.
- **Use the most specific module.** Add SKU/bundle/add-on files under `catalogs/skus/`, revenue-line files under `catalogs/revenue-lines/`, channel files under `catalogs/channels/`, directional strategy under `strategy/`, supporting proof under `evidence/`, and classification/routing rules under `taxonomies/`.
- **Operator inputs are not canon edits.** `operator-inputs.json` lives under team shared state; [`governance/HOW_TO_GATHER_INPUTS.md`](governance/HOW_TO_GATHER_INPUTS.md) describes how to gather and interpret those fields.
- **Operator executes accepted edits.** Commit messages cite the decision id.

Decision-context detail lives in [`governance/editing.md`](governance/editing.md).

## Cross-references

- [`docs/agent-system/team-plan-of-record.manifest.json`](../agent-system/team-plan-of-record.manifest.json) — shared machine-readable PoR contract and extension rules.
- [`docs/agent-system/TEAM_DOCS_PATTERNS.md`](../agent-system/TEAM_DOCS_PATTERNS.md) — typed knowledge flow and plan-of-record write boundary.
- [`docs/marketing/`](../marketing/README.md) — external voice and channel execution. Marketing reads monetization for pricing, catalog, and tier authority.
- [`docs/director-swarm/`](../director-swarm/README.md) — portfolio steering and revenue-critical sequencing.
- [`docs/strategy/idea-pipeline/`](../strategy/idea-pipeline/README.md) — operator-curated idea staging before SKU-shaped items graduate into monetization canon.

## Future PoR work

- Add prompt-manager validation against this `manifest.json` once the PoR validator lands.
- Promote a clean `interfaces/` module if monetization input/output tables outgrow the operating model.
- Split large strategy or evidence documents only when one-entity-per-file structure would make future edits safer.
- Migrate old path references in historical handoff logs only if those logs become active operational inputs; they are otherwise immutable history.
