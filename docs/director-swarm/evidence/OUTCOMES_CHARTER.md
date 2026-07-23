# Outcomes Charter

How `director-swarm` — specifically the eventual `outcome-strategist` lane — thinks about whether Vrooli's work is producing results. Swarm Manager tells the director *what work is happening*; Command Center tells the director *whether that work is producing results*.

## Authoritative surface

**Command Center is the source of truth for outcome metrics.** This charter defines high-level framing and categories; Command Center holds live numbers.

- Gap-tracked metrics: every metric in Command Center is tagged `live` / `partial` / `gap` via the `/api/v1/gaps` endpoint. Gaps become backlog signals — [`director-dashboard-gap-workflow`](../strategy/ROADMAP.md) wires this loop.
- TV/kiosk mode: operator can throw Command Center on a TV to see ground-truth metrics at a glance.

This charter does **not** duplicate what Command Center will render. Where a specific metric belongs in Command Center, this doc says so and moves on.

## Outcome categories

Outcomes are organized by the six Command Center dashboard pages (see [`command-center-dashboards`](../strategy/ROADMAP.md) for the live plan). Each page is a category.

### Mission Control — system overview

Aggregate health of the Vrooli platform: scenarios running, agents working, recent runs, error rates.

- Authoritative metrics: `pending-command-center`
- What "good" looks like here: `pending-command-center`

### The Hive — scenario ecosystem

Per-scenario health, capability coverage, and readiness state. Which scenarios are headliner-ready? Which are blocked? Which are receiving attention?

- Authoritative metrics: `pending-command-center`
- Cross-reference: [monetization catalog](../../monetization/catalogs/CATALOG.md) for SKU-to-scenario mapping.

### The Forge — engineering velocity

Swarm Manager-level velocity: goal throughput, backlog burn-down, agent run success rate, cycle time. Currently ~90% live per the dashboard goal.

- Authoritative metrics: `pending-command-center` (largely live — verify coverage when this charter is next reviewed)
- What "good" looks like here: `pending-command-center`

### Ledger — revenue & subscriptions

The monetization outcomes surface. Subscription count, MRR, churn, conversion rates, services revenue. Currently ~60% live.

- Authoritative metrics: `pending-command-center`
- Cross-reference: [monetization/catalogs/revenue-lines/README.md](../../monetization/catalogs/revenue-lines/README.md) for the instrumentation contract each active revenue line must meet.
- Phase note: until first paying user, most fields will show `gap` or `partial`. That is expected and correct; this is the dashboard whose gap-count most urgently needs to shrink once revenue starts.

### Broadcast — marketing & growth

Funnel stages, acquisition channels, campaign results. Currently ~40% live.

- Authoritative metrics: `pending-command-center`
- Cross-reference: [monetization/strategy/FUNNEL.md](../../monetization/strategy/FUNNEL.md).
- Phase note: many signals here depend on having paying users and live landing pages. Heavy `pending-telemetry` expected pre-launch.

### Panorama — aggregate solar system view

Cross-cutting: how the other five dashboards relate, where capability gaps are blocking outcome visibility, long-range trends.

- Authoritative metrics: `pending-command-center`

## Outcomes that may not fit a dashboard

If there's an outcome worth tracking that doesn't map cleanly to a Command Center page, it gets captured here until either (a) a page exists that covers it, or (b) it's demoted to an operator-eyes-only concern.

_No entries yet._ `pending-operator-input` — add here when one appears that doesn't fit the six categories.

## The gap-closure loop

When a metric on any Command Center page shows as `gap`, the flow is:

1. Director-swarm's `outcome-strategist` (once active) spots the gap via `/api/v1/gaps`.
2. Proposes a backlog item or goal to build the missing data pipeline, with decision context `outcome-gap`.
3. Human approves at the vision walk.
4. A capability-building team (or the relevant feature team) builds the pipeline.
5. The metric flips from `gap` → `live`.
6. Cycle repeats.

The pre-activation analogue: while `outcome-strategist` is disabled, the operator and `portfolio-manager` manually spot gap candidates during the vision walk. Expect this to be manual until `director-dashboard-gap-workflow` ships.

## What this charter does NOT do

- **Does not define specific success thresholds** (e.g., "MRR > $X by date Y"). Thresholds belong in Command Center configuration or in goal-specific acceptance criteria, not here.
- **Does not track metrics over time.** Command Center handles history.
- **Does not replace the monetization team's financial model.** For revenue math, see [monetization/evidence/FINANCIAL_MODEL.md](../../monetization/evidence/FINANCIAL_MODEL.md).

## Why `pending-command-center` markers are deliberately visible

Many sections above are explicitly thin. That is intentional — the visibility of these placeholders is itself a signal that Command Center is a high-leverage goal. When Command Center pages ship and real metrics land, the corresponding sections here get filled in via approved decisions. Until then, the debt stays visible so future reviewers (human and agent) don't forget to come back.

## Updating this charter

Changes go through approved decisions with context `outcome-direction` (reframing what matters) or `outcome-gap` (a specific metric's status or pipeline changes). `pending-command-center` markers get replaced when the relevant Command Center page has a stable metric — the replacement itself is an approved decision, not a silent edit.
