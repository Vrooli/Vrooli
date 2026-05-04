# Monetization Opportunity Signal Taxonomy

Cross-team-readable canon for how monetization-opportunity signals are partitioned, classified, dispatched, and shaped. This file is the human-readable view of `docs/monetization/opportunity-taxonomy.json` (the parseable sidecar consumed by the heartbeat builder and the `unknown_taxonomy` / `missing_destination_schema` validation rules).

**Owner team:** monetization. **Status:** canon. Operator-curated via monetization decisions.

Cited by:
- `topics.json` for `monetization/opportunity-scout` (`intake[].taxonomy = "monetization-opportunity"`).
- The `monetization-signal-classifier` skill (pure judgment).

## Editing rules

Update this markdown when the JSON sidecar changes (or vice versa). Both are paired sources of truth — markdown for review, JSON for machine consumption. Promote new schemas here before referencing them from any `topics.json`.

## Signal types and dispatch

The 7-slot taxonomy (matches morning-vision-walk Phase 8 routing):

| Signal type        | Definition                                                       | Default method                | Default destination                 |
|--------------------|------------------------------------------------------------------|-------------------------------|-------------------------------------|
| competitor-move    | Competitor pricing/packaging/positioning/changelog change.       | competitor-move-capture       | `monetization/market-scan/<slug>` (or `candidate-sku-record/` if SKU-shaped) |
| capability-arrival | Vrooli gained a scenario/resource that unlocks a SKU/bundle.     | capability-arrival-scan       | `candidate-sku-record/<slug>` (almost always) |
| customer-ask       | Operator-fed: someone asked for X.                               | customer-ask-shaping          | `candidate-sku-record/<slug>` (if SKU-shaped) |
| channel            | New acquisition channel observed working.                        | channel-fit-scan              | `monetization/market-scan/<slug>` (`candidate-sku-record/` if Vrooli can ship into it) |
| bundle-hint        | Two existing things should be packaged together.                 | bundle-hint-shaping           | `candidate-sku-record/<slug>`              |
| retention-signal   | Observed retention lever (own or competitor).                    | retention-signal-capture      | `monetization/market-scan/<slug>` (`candidate-sku-record/` if it implies an addon) |
| benchmark          | Comparable pricing / market fact.                                | pricing-comp-capture          | `monetization/market-scan/<slug>`   |

## Evidence rules

- Cite sources for every external claim.
- Label single-snapshot findings `light-interpretation`.
- Do not silently invent benchmarks; raise `capability-gap` if source access is blocked.
- Tailwind references (regulatory, market, demographic) must be cited or flagged `tailwind-uncited`.
- Tier-4 hardware proposals without operator initiation are out of scope; drop.

## Action selection

| Action            | When                                                                                          |
|-------------------|-----------------------------------------------------------------------------------------------|
| drop              | Weak one-off / no fit / out of scope.                                                         |
| observe           | Single-snapshot market fact. Retag inbox row to `monetization/market-scan/<slug>`.            |
| promote-to-canon  | Plausible SKU-shaped idea. Retag to `candidate-sku-record/<slug>`.                                   |
| file-decision     | Strong signal + clear fit + threshold met. Promote AND raise `catalog-promotion`, `channel-activation`, or `services-activation`. |
| capability-gap    | Source / tool / scenario missing. File `capability-gap` and leave the inbox row.              |

## Owned schemas

### candidate-sku
```yaml
---
type: opportunity
kind: <sku-candidate|addon-candidate|services-line-candidate|channel-candidate>
catalog:
  proposed_sku: <new-base-bundle|addon|services-line|null>
  parent_bundle: <business|lifestyle|null>
capability_reuse: <high|medium|low>
tam: <S|M|L>
effort: <S|M|L>
status: <idea|candidate|trigger-met|active|shipped|retired>
original_at: <ISO timestamp>
original_by: <agent id or operator>
---
```
Body must include: `# <Name>`, description, `## Revisit trigger`, `## Acquisition hypothesis`, `## Retention hypothesis`, `## Signal`.

### market-scan
```yaml
---
type: market-scan
kind: <benchmark-capture|assumption-check|competitive-observation|stale-refresh|channel-assumption-check>
comp: <name or category-wide>
category: <text>
dimension: <pricing|retention|churn|attach-rate|activation|channel-cac|other>
date_observed: <YYYY-MM-DD>
applicability: <high|medium|low>
affects_benchmarks_md: <true|false>
affects_pricing: <true|false>
affects_financial_model: <text or null>
supersedes: <prior-scan-id or null>
original_at: <ISO timestamp>
original_by: <agent id>
---
```
Body must include `## Value`, `## Notes`. Source URL goes on the entry's `--source` flag.

## Honesty flags

`light-interpretation`, `tailwind-uncited`, `hardware-tier`, `legal-surface`, `single-source`.

## Pending method skills

The `defaultMethod` ids are not yet registered as standalone skills. Members apply inline guidance from the schemas + evidence rules above until the methods ship; the classifier still recommends them as `recommended_method`.
