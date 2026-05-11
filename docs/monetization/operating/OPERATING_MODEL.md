# Monetization Operating Model

**Status:** explanatory PoR canon. This document defines how `monetization` works as a coherent system: opportunity intake, catalog strategy, financial tracking, validation, contrarian review, and operator-approved canon updates.

The current document adopts the generic team operating-model shape from `path:docs/agent-system/OPERATING_GRAPHS.md`.

## Mission

Monetization turns Vrooli's product capabilities, market signals, operator inputs, and financial constraints into a coherent SKU catalog, pricing posture, revenue-line strategy, channel discipline, and evidence-backed decisions.

The team does not own marketing voice, direct scenario implementation, deployment infrastructure, or portfolio prioritization. It owns monetization truth and proposes changes through operator-approved decisions.

## Scope

Monetization owns:

- SKU catalog canon, base bundles, add-ons, and scenario-to-SKU mapping;
- delivery-tier, pricing, funnel, revenue-line, channel, and financial-model canon;
- opportunity classification and candidate-pool hygiene;
- validation-request classification, market benchmarks, and benchmark refresh proposals;
- runway, default-alive, time-allocation, services-trap, funnel, and retention warnings;
- contrarian review of material monetization proposals;
- taxonomy sidecars for monetization opportunity and validation request flows.

Monetization does not write marketing content, run campaigns, build scenarios, or edit non-monetization PoR surfaces. Those outputs route to owning teams through decisions, backlog items, or capability gaps.

## Operating Loops

Monetization has five loops:

1. **Opportunity loop** — drain `opportunity-inbox/*`, classify signals with the opportunity taxonomy, write candidate records or market scans, and raise catalog/channel/services decisions when thresholds fire.
2. **Catalog loop** — inspect candidate SKUs, tiers, services lines, channel triggers, and scenario-role changes; propose the smallest catalog or mapping decision that keeps the canon accurate.
3. **Financial loop** — read operator inputs and telemetry, compute runway/default-alive posture, flag material assumption, pricing, funnel, retention, and services-capacity changes.
4. **Validation loop** — drain validation requests, refresh stale benchmarks, capture market comps, and raise benchmark, pricing, or assumption-update decisions when evidence is material.
5. **Challenge loop** — review pending monetization decisions for catalog sprawl, premature activation, services trap, retention blindness, hallucinated metrics, positioning drift, and marketing-default failure modes.

The loops are intentionally independent. A market scan can remain evidence without changing canon; a catalog trigger can fire without a pricing change; a contrarian review can stay quiet when proposals are well-grounded.

## Operating Graph

This graph is the team-level contract. It shows how operator inputs, market signals, catalog state, validation requests, and challenge records move through the monetization team.

<!-- prompt-manager-graph:
id: monetization-operating-model
scope: team
team: monetization
mode: explanatory
actor_alias.operator: external:operator
actor_alias.marketing: team:marketing-crew
actor_alias.decision owners: none
-->
```mermaid
flowchart LR
  subgraph INFLOWS["Inflows / Producers"]
    %% @node OP external:operator
    OP([Operator])
    %% @node MKT team:marketing-crew
    MKT[[Marketing Crew]]
  end

  %% Members
  %% @node OS member:opportunity-scout
  OS[Opportunity Scout]
  %% @node CS member:catalog-strategist
  CS[Catalog Strategist]
  %% @node FT member:financial-tracker
  FT[Financial Tracker]
  %% @node MV member:market-validator
  MV[Market Validator]
  %% @node MC member:monetization-contrarian
  MC[Monetization Contrarian]

  %% Topics
  %% @node OIN topic:opportunity-inbox/*
  OIN[(opportunity-inbox/*)]
  %% @node CAND topic:candidate-sku-record/*
  CAND[(candidate-sku-record/*)]
  %% @node SCOUT topic:scout-scan/*
  SCOUT[(scout-scan/*)]
  %% @node OPP topic:monetization/opportunity/*
  OPP[(monetization/opportunity/*)]
  %% @node CANON topic:monetization-canon/*
  CANON[(monetization-canon/*)]
  %% @node SNAP topic:catalog-snapshot/*
  SNAP[(catalog-snapshot/*)]
  %% @node LEDGER topic:ledger-snapshot/*
  LEDGER[(ledger-snapshot/*)]
  %% @node VAIN topic:validation-inbox/*
  VAIN[(validation-inbox/*)]
  %% @node ADJ topic:monetization-benchmark-adjacent-record/*
  ADJ[(monetization-benchmark-adjacent-record/*)]
  %% @node BENCH topic:monetization-benchmark-record/*
  BENCH[(monetization-benchmark-record/*)]
  %% @node MSCAN topic:monetization/market-scan/*
  MSCAN[(monetization/market-scan/*)]
  %% @node CHAL topic:challenge-report/*
  CHAL[(challenge-report/*)]
  %% @node RES topic:challenge-resolution-record/*
  RES[(challenge-resolution-record/*)]

  %% PoR
  %% @node CATALOG path:docs/monetization/catalogs/CATALOG.md
  CATALOG[/docs/monetization/catalogs/CATALOG.md/]
  %% @node SKU path:docs/monetization/catalogs/scenario-sku-map.json
  SKU[/docs/monetization/catalogs/scenario-sku-map.json/]
  %% @node STRAT path:docs/monetization/strategy/STRATEGY.md
  STRAT[/docs/monetization/strategy/STRATEGY.md/]
  %% @node PRICE path:docs/monetization/strategy/PRICING.md
  PRICE[/docs/monetization/strategy/PRICING.md/]
  %% @node FIN path:docs/monetization/evidence/FINANCIAL_MODEL.md
  FIN[/docs/monetization/evidence/FINANCIAL_MODEL.md/]
  %% @node BM path:docs/monetization/evidence/BENCHMARKS.md
  BM[/docs/monetization/evidence/BENCHMARKS.md/]

  %% Decisions
  %% @node CATPROMO decision:catalog-promotion
  CATPROMO{catalog-promotion}
  %% @node MAP decision:catalog-mapping-update
  MAP{catalog-mapping-update}
  %% @node CHACT decision:channel-activation
  CHACT{channel-activation}
  %% @node SERVACT decision:services-activation
  SERVACT{services-activation}
  %% @node PRICEDEC decision:pricing-decision
  PRICEDEC{pricing-decision}
  %% @node ASSUMP decision:financial-model-assumption-update
  ASSUMP{financial-model-assumption-update}
  %% @node RUNWAY decision:runway-warning
  RUNWAY{runway-warning}
  %% @node BENCHDEC decision:benchmark-update
  BENCHDEC{benchmark-update}
  %% @node REJECT decision:decision-rejection-proposed
  REJECT{decision-rejection-proposed}
  %% @node FRAME decision:framework-update
  FRAME{framework-update}

  OP --> OIN
  OP --> VAIN
  MKT --> ADJ

  OIN --> OS
  CAND --> OS
  OS --> CAND
  OS --> SCOUT
  OS --> OPP
  OS --> CATPROMO
  OS --> CHACT
  OS --> SERVACT

  CATALOG --> CS
  SKU --> CS
  CAND --> CS
  CS --> CANON
  CS --> SNAP
  CS --> CATPROMO
  CS --> MAP
  CS --> CHACT
  CS --> SERVACT

  OP --> FT
  FIN --> FT
  PRICE --> FT
  FT --> LEDGER
  FT --> PRICEDEC
  FT --> ASSUMP
  FT --> RUNWAY

  VAIN --> MV
  ADJ --> MV
  BM --> MV
  MV --> BENCH
  MV --> MSCAN
  MV --> BENCHDEC
  MV --> PRICEDEC
  MV --> ASSUMP

  CATPROMO --> MC
  MAP --> MC
  CHACT --> MC
  SERVACT --> MC
  PRICEDEC --> MC
  ASSUMP --> MC
  RUNWAY --> MC
  BENCHDEC --> MC
  MC --> CHAL
  MC --> RES
  MC --> REJECT
  MC --> FRAME

  CATPROMO --> CATALOG
  MAP --> SKU
  PRICEDEC --> PRICE
  ASSUMP --> FIN
  BENCHDEC --> BM
```

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:opportunity-inbox/*` | live | `external:operator` | `member:opportunity-scout` | Intake for monetization opportunity signals classified by `monetization-opportunity`. |
| `topic:candidate-sku-record/*` | live | `member:opportunity-scout` | `member:opportunity-scout`, `member:catalog-strategist` | Candidate SKU, add-on, services-line, or channel records. |
| `topic:scout-scan/*` | live | `member:opportunity-scout` | `member:catalog-strategist` | Scout heartbeat summaries and pool snapshots. |
| `topic:monetization/opportunity/*` | live | `member:opportunity-scout` | `member:catalog-strategist` | Additional opportunity observations that are not yet promoted. |
| `topic:monetization-canon/*` | live | `member:catalog-strategist` | `external:operator` | Proposed PoR edit evidence for monetization canon. |
| `topic:catalog-snapshot/*` | live | `member:catalog-strategist` | `member:monetization-contrarian`, `external:operator` | Snapshot of SKU, tier, channel, and services-line trigger state. |
| `topic:ledger-snapshot/*` | live | `member:financial-tracker` | `member:market-validator`, `member:monetization-contrarian`, `external:operator` | Financial posture snapshot. |
| `topic:validation-inbox/*` | live | `external:operator` | `member:market-validator` | Validation requests classified by `monetization-validation`. |
| `topic:monetization-benchmark-adjacent-record/*` | live | `team:marketing-crew` | `member:market-validator` | Cross-team benchmark-adjacent evidence from marketing. |
| `topic:monetization-benchmark-record/*` | live | `member:market-validator` | `member:financial-tracker`, `member:catalog-strategist` | Validated benchmark records. |
| `topic:monetization/market-scan/*` | live | `member:market-validator`, `member:opportunity-scout` | `member:financial-tracker`, `member:catalog-strategist` | Market-scan evidence. |
| `topic:challenge-report/*` | live | `member:monetization-contrarian` | `external:operator`, decision owners | Append-only challenge evidence. |
| `topic:challenge-resolution-record/*` | live | `member:monetization-contrarian` | `external:operator`, decision owners | Latest-state challenge resolution records. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `catalog-promotion` | `member:catalog-strategist`, `member:opportunity-scout` | Promote scenario, SKU, add-on, services line, or candidate for operator review. | `topic:candidate-sku-record/*` or `topic:catalog-snapshot/*` shows trigger fired. | Updates `path:docs/monetization/catalogs/CATALOG.md` or catalog entry files. |
| `catalog-mapping-update` | `member:catalog-strategist` | Update scenario-to-SKU membership or role mapping. | Scenario role changed or SKU mapping no longer matches reality. | Updates `path:docs/monetization/catalogs/scenario-sku-map.json`. |
| `channel-activation` | `member:catalog-strategist`, `member:opportunity-scout` | Activate a discovery channel when prerequisites and telemetry exist. | `topic:catalog-snapshot/*` or opportunity evidence shows activation trigger fired. | Updates `path:docs/monetization/catalogs/channels/README.md` and channel entry files. |
| `services-activation` | `member:catalog-strategist`, `member:opportunity-scout` | Activate a candidate services line. | Candidate has validation hypothesis, fixed-duration pilot, and productization target. | Updates `path:docs/monetization/catalogs/revenue-lines/README.md` and relevant revenue-line file. |
| `pricing-decision` | `member:financial-tracker`, `member:market-validator` | Set or adjust price for SKU, tier, bundle, or revenue line. | Benchmark or financial signal materially affects price band. | Updates `path:docs/monetization/strategy/PRICING.md`. |
| `financial-model-assumption-update` | `member:financial-tracker`, `member:market-validator` | Revise a load-bearing financial or market assumption. | `topic:monetization-benchmark-record/*`, `topic:ledger-snapshot/*`, or operator input contradicts current model. | Updates `path:docs/monetization/evidence/FINANCIAL_MODEL.md`. |
| `runway-warning` | `member:financial-tracker` | Alert operator to material runway/default-alive gap. | `topic:ledger-snapshot/*` shows material runway change. | Operator updates priorities, burn assumptions, or model notes. |
| `services-trap-warning` | `member:financial-tracker` | Warn when services work exceeds guardrails. | Services time or services/subscription ratio breaches limits. | Updates revenue-line discipline or pauses services work. |
| `benchmark-update` | `member:market-validator` | Promote refreshed or new benchmark into canon. | Market validation crosses taxonomy materiality threshold. | Updates `path:docs/monetization/evidence/BENCHMARKS.md`. |
| `funnel-bottleneck` | `member:financial-tracker` | Name the current measured funnel bottleneck. | Telemetry identifies a stage as materially blocked. | Updates `path:docs/monetization/strategy/FUNNEL.md` or `path:docs/monetization/evidence/TELEMETRY_ROADMAP.md`. |
| `retention-concern` | `member:financial-tracker` | Flag materially worse retention than target. | Measured retention metric misses target. | Updates `path:docs/monetization/strategy/FUNNEL.md` or financial assumptions. |
| `decision-rejection-proposed` | `member:monetization-contrarian` | Recommend rejecting or revising a weak proposal. | Pending decision hits a named failure mode. | Operator rejects, revises, or overrides decision. |
| `framework-update` | `member:monetization-contrarian` | Add a recurring flaw missing from the failure-mode framework. | Multiple proposals reveal an uncovered failure pattern. | Updates contrarian framework or operating model. |

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| `external:operator` monetization opportunity | `topic:opportunity-inbox/*` | `member:opportunity-scout` | Classify via `taxonomies/monetization-opportunity/taxonomy.json`; observe, promote, file decision, or raise capability gap. |
| `external:operator` validation request | `topic:validation-inbox/*` | `member:market-validator` | Classify via `taxonomies/monetization-validation/taxonomy.json`; capture, refresh, or raise decision by materiality threshold. |
| `team:marketing-crew` benchmark-adjacent evidence | `topic:monetization-benchmark-adjacent-record/*` | `member:market-validator` | Validate with `market-validation-triage`; producer owns front-matter shape, validator owns benchmark disposition. |
| Operator-provided financial inputs | `path:scenarios/prompt-manager/store/teams/monetization/shared/operator-inputs.json` | `member:financial-tracker` | Interpret per `path:docs/monetization/governance/HOW_TO_GATHER_INPUTS.md`; do not edit operator state. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| SKU catalog updates | `path:docs/monetization/catalogs/CATALOG.md` | `team:director-swarm`, `team:marketing-crew`, `external:operator` | Keep bundle, add-on, and candidate catalog authoritative. |
| Scenario-to-SKU mapping | `path:docs/monetization/catalogs/scenario-sku-map.json` | `team:marketing-crew`, `team:director-swarm`, downstream scenarios | Drive tier alignment, campaign claims, and bundle membership. |
| Pricing and tier canon | `path:docs/monetization/strategy/PRICING.md` | `team:marketing-crew`, `external:operator` | Keep external pricing claims consistent with monetization truth. |
| Market benchmarks | `path:docs/monetization/evidence/BENCHMARKS.md` | `member:financial-tracker`, `external:operator` | Ground pricing, financial assumptions, and validation requests. |
| Ledger snapshots | `topic:ledger-snapshot/*` | `external:operator`, `member:monetization-contrarian` | Surface current runway, burn, revenue, and default-alive status. |
| Challenge records | `topic:challenge-report/*`, `topic:challenge-resolution-record/*` | `external:operator`, decision owners | Preserve skeptical review state for material decisions. |

## Feedback / Capability Improvement Loop

1. `member:financial-tracker` flags missing telemetry in `topic:ledger-snapshot/*` and links gaps to `path:docs/monetization/evidence/TELEMETRY_ROADMAP.md`.
2. `member:market-validator` raises capability gaps when source access blocks `topic:monetization-benchmark-record/*` rather than inventing values.
3. `member:catalog-strategist` uses `topic:catalog-snapshot/*` to detect stale SKU, tier, channel, and services-line assumptions.
4. `member:monetization-contrarian` writes `topic:challenge-report/*` only for concrete failure modes and keeps latest state in `topic:challenge-resolution-record/*`.
5. Repeated gaps become `decision:framework-update`, `decision:financial-model-assumption-update`, or capability-gap decisions routed to the owning team.

## Current Implementation Gaps

- `topic[future]:channel-attribution/*` is target-state until channel attribution telemetry exists.
- `topic[future]:retention-metric/*` is target-state until subscription lifecycle telemetry ships.
- `path:docs/monetization/interfaces/` is target-state; current inputs and outputs still fit inside this operating model.
- `path:docs/monetization/evidence/TELEMETRY_ROADMAP.md` remains the primary bridge from aspirational monetization logic to measured signals.

## Adoption / Validation

Run these checks after changing this document:

```bash
prompt-manager graph operating-model validate --team monetization --id monetization-operating-model
prompt-manager graph operating-model diff --team monetization --id monetization-operating-model
prompt-manager graph operating-model coverage --team monetization --id monetization-operating-model
```

This operating model must stay registered through `scenarios/prompt-manager/store/teams/monetization/team.json` and discoverable from `docs/monetization/README.md`.

The graph is currently explanatory because monetization had no pre-existing contract operating model before this PoR migration. Promote it to `mode: contract` only after the graph is expanded to match live `topics.json`, decision ownership, evidence reads, capability-gap routing, and PoR output declarations.
