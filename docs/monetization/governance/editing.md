# Monetization Editing

## Authority

`docs/monetization/` is operator-curated plan-of-record canon for the `monetization` team. Agents read it every heartbeat, but they do not edit it directly.

The team-owned runtime state lives under `scenarios/prompt-manager/store/teams/monetization/`. Operator inputs live in team shared state; this folder only documents how to gather and interpret them.

## Change Flow

1. A monetization member observes a material trigger, benchmark, opportunity, or risk.
2. The member writes evidence to the appropriate knowledge topic.
3. The member raises the smallest relevant work type.
4. The operator accepts, rejects, or requests revision.
5. The operator applies accepted PoR edits and cites the decision id.

Common edit contexts:

| Context | Typical PoR target |
|---|---|
| `catalog-promotion` | `catalogs/CATALOG.md`, `catalogs/skus/`, `catalogs/revenue-lines/` |
| `catalog-mapping-update` | Offer Desk `belongs_to` graph and the surviving SKU strategy files |
| `channel-activation` | `catalogs/channels/` |
| `services-activation`, `services-conversion`, `services-sunset` | `catalogs/revenue-lines/` and `catalogs/CATALOG.md` |
| `pricing-decision` | `strategy/PRICING.md` |
| `financial-model-assumption-update` | `evidence/FINANCIAL_MODEL.md` |
| `benchmark-update` | `evidence/BENCHMARKS.md` |
| `funnel-bottleneck`, `retention-concern` | `strategy/FUNNEL.md` and `evidence/TELEMETRY_ROADMAP.md` |

## Direct Edits

Direct agent edits to plan-of-record canon are not allowed. The only direct-edit monetization state is operator-owned runtime input data under team shared state, not this PoR folder.
