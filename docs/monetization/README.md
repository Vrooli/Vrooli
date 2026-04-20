# Monetization

Durable monetization strategy, catalog, and operational framework for Vrooli. Owned by the `monetization` team in prompt-manager; authored and curated by the human operator.

## Organizing principle

Monetization is described across **four orthogonal axes**:

1. **WHAT** we sell — the [catalog](CATALOG.md) of bundles and add-ons.
2. **HOW** it's delivered — the [delivery tiers](TIERS.md) (apps → self-hosted → hosted → hardware).
3. **WHO** flows through — the [funnel](FUNNEL.md) from acquisition to advocacy.
4. **HOW WE MAKE MONEY** — the [revenue lines](REVENUE_LINES.md) (subscriptions, services, lead-gen, consulting).

Pricing is the intersection of WHAT × HOW. The [financial model](FINANCIAL_MODEL.md) projects runway, default-alive position, and per-tier economics from these inputs.

## Files

| File | Purpose |
|---|---|
| [STRATEGY.md](STRATEGY.md) | Narrative framing, principles, long-term directions, operator's intent |
| [CATALOG.md](CATALOG.md) | SKU index — bundles + add-ons with lifecycle state |
| [catalog/base/business.md](catalog/base/business.md) | Business bundle: headliners, depth, DAG, rationale |
| [catalog/base/lifestyle.md](catalog/base/lifestyle.md) | Lifestyle bundle: status, scope, pending definition |
| [catalog/addons/](catalog/addons/) | Add-on candidates (dormant until triggers fire) |
| [TIERS.md](TIERS.md) | Delivery tiers with activation triggers |
| [PRICING.md](PRICING.md) | Tier × bundle pricing matrix (mostly TBD pre-launch) |
| [FINANCIAL_MODEL.md](FINANCIAL_MODEL.md) | Cost structure, runway math, default-alive target |
| [FUNNEL.md](FUNNEL.md) | Acquisition → retention stages, metrics, telemetry gaps |
| [REVENUE_LINES.md](REVENUE_LINES.md) | Subscription + services + other revenue lines with discipline rules |
| [TELEMETRY_ROADMAP.md](TELEMETRY_ROADMAP.md) | What metrics need what data capabilities; replaces-manual migration list |
| [BENCHMARKS.md](BENCHMARKS.md) | Curated market benchmarks (populated by monetization team) |
| [HOW_TO_GATHER_INPUTS.md](HOW_TO_GATHER_INPUTS.md) | Per-field guidance for the operator-edited `operator-inputs.json` |
| [scenario-sku-map.json](scenario-sku-map.json) | Many-to-many: which scenarios belong to which SKUs |

## Ownership and editing discipline

- These docs are the **canonical plan**. The `monetization` team reads them every heartbeat and proposes edits via decisions.
- The human operator is the final author. Agents do not write to these files directly; they propose diffs in their decision logs.
- Team-produced operational exhaust (ledger snapshots, opportunity pool, market scans, decisions) lives in `scenarios/prompt-manager/store/teams/monetization/shared/`, not here.

## Consumers

Other teams and scenarios read these docs as the source of truth for monetization state:

- **director-swarm** reads `CATALOG.md` for the revenue critical path instead of deriving it ad-hoc.
- **scenario-feature** reads `CATALOG.md` before scoping work so features map to bundle impact.
- **marketing-crew** reads `CATALOG.md` + `STRATEGY.md` for positioning.
- **landing-page-business-suite** reads `CATALOG.md` + `PRICING.md` to generate pricing pages and entitlements.
- **scenario-to-cloud** reads `TIERS.md` to understand which deployment modes the monetization plan requires.

## Honesty conventions

- Any metric that is not yet measurable carries an explicit **`pending-telemetry`** flag with a pointer to `TELEMETRY_ROADMAP.md`.
- Pre-launch targets are labelled **`aspirational`**; post-launch numbers are labelled **`measured`**.
- Revenue-line and SKU statuses use the lifecycle vocabulary: `idea` → `candidate` → `trigger-met` → `active` → `shipped` / `retired`.
