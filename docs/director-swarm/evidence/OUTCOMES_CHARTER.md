# Outcomes Charter

How `director-swarm` — specifically the `outcome-strategist` lane — thinks about whether Vrooli's work is producing results. Swarm Manager tells the director *what work is happening*; Command Center tells the director *whether that work is producing results*.

**This charter derives from [`../strategy/OBJECTIVES.md`](../strategy/OBJECTIVES.md), which sits above it.** Objectives say what Vrooli is for; this charter says how progress toward them is observed. The direction of derivation matters: outcome categories are Command Center dashboard pages, and a dashboard's page list is an instrumentation decision, not a statement of intent. Every category below must trace upward to at least one objective, and an outcome worth pursuing that no category can hold is a finding against this charter rather than an outcome that does not count.

## Authoritative surface

**Command Center is the source of truth for outcome metrics.** This charter defines high-level framing and categories; Command Center holds live numbers.

- Gap-tracked metrics: every metric in Command Center is tagged `live` / `partial` / `gap` via the `/api/v1/gaps` endpoint. Gaps become backlog signals — [`director-dashboard-gap-workflow`](../strategy/ROADMAP.md) wires this loop.
- TV/kiosk mode: operator can throw Command Center on a TV to see ground-truth metrics at a glance.

This charter does **not** duplicate what Command Center will render. Where a specific metric belongs in Command Center, this doc says so and moves on.

## Sensor map

Which surface observes each outcome category today (mirrors the sensor discipline in `path:scenarios/infrastructure-manager/docs/concepts/COVERAGE-MODEL.md`: a category with no sensor cannot be regulated). Read rule: `GET /api/v1/dashboards/<id>` on command-center is the live sensor for a category; `GET /api/v1/gaps` lists **only** metrics whose pipeline is missing — a metric absent from the gaps registry and present on its dashboard is live. Both sensors are HTTP-only today: the command-center CLI exposes no `gaps` or `dashboards` verb and the scenario runs on demand (`vrooli scenario start command-center`) — that CLI gap is a standing `outcome-gap` candidate.

| Category (dashboard id) | Observed 2026-07-23 | Actuator when the sensor shows a hole |
|---|---|---|
| Mission Control (`mission-control`) | Live except LPBS revenue metrics (registry `gap`) | `outcome-gap` naming the missing LPBS endpoint |
| The Hive (`hive`) | Live except usage frequency (registry `gap`; usage is collected nowhere yet) | `outcome-gap` |
| The Forge (`forge`) | Live — no gaps-registry entries | `outcome-direction` on measured evidence |
| Ledger (`ledger`) | LPBS-backed rows `gap`/`partial` pre-first-customer (expected; see phase note below) | `outcome-gap`; expected to shrink once revenue starts |
| Broadcast (`broadcast`) | All tracked metrics `gap` — no external-platform integrations exist | `outcome-gap` |
| Panorama (`panorama`) | Composites `partial`/`gap`; each resolves through its input categories above | resolves via the rows above |

Honesty-flag vocabulary, three levels: registry **`gap`** = sensor named, no data pipeline; registry **`partial`** = raw data exists, aggregation missing; **`pending-command-center`** = the metric is not in the registry at all — naming it in a prediction block is precisely how it gets registered (prediction-ledger rule 1). Re-observe and update the middle column through approved decisions, never silent edits.

## Outcome categories

Outcomes are organized by the six Command Center dashboard pages (see [`command-center-dashboards`](../strategy/ROADMAP.md) for the live plan). Each page is a category.

Command Center is the instrument for **instrumental** objectives, and for the business half of the terminal ones. It is not the universal outcome surface: a terminal objective in a personal-life domain is measured by the scenario that serves it. The full routing table is [`../strategy/OBJECTIVES.md`](../strategy/OBJECTIVES.md) §"Evidence routing".

### Mission Control — system overview

Aggregate health of the Vrooli platform: scenarios running, agents working, recent runs, error rates.

- Authoritative metrics: see §"Sensor map"
- What "good" looks like here: `pending-operator-input`

### The Hive — scenario ecosystem

Per-scenario health, capability coverage, and readiness state. Which scenarios are headliner-ready? Which are blocked? Which are receiving attention?

- Authoritative metrics: see §"Sensor map"
- Cross-reference: [monetization catalog](../../monetization/catalogs/CATALOG.md) for SKU-to-scenario mapping.

### The Forge — engineering velocity

Swarm Manager-level velocity: goal throughput, backlog burn-down, agent run success rate, cycle time.

- Authoritative metrics: live — see §"Sensor map" (verified 2026-07-23: no gaps-registry entries)
- What "good" looks like here: `pending-operator-input`

### Ledger — revenue & subscriptions

The monetization outcomes surface. Subscription count, MRR, churn, conversion rates, services revenue.

- Authoritative metrics: see §"Sensor map"
- Cross-reference: [monetization/catalogs/revenue-lines/README.md](../../monetization/catalogs/revenue-lines/README.md) for the instrumentation contract each active revenue line must meet.
- Phase note: until first paying user, most fields will show `gap` or `partial`. That is expected and correct; this is the dashboard whose gap-count most urgently needs to shrink once revenue starts.

### Broadcast — marketing & growth

Funnel stages, acquisition channels, campaign results.

- Authoritative metrics: see §"Sensor map"
- Cross-reference: [monetization/strategy/FUNNEL.md](../../monetization/strategy/FUNNEL.md).
- Phase note: many signals here depend on having paying users and live landing pages. Heavy `pending-telemetry` expected pre-launch.

### Panorama — aggregate solar system view

Cross-cutting: how the other five dashboards relate, where capability gaps are blocking outcome visibility, long-range trends.

- Authoritative metrics: see §"Sensor map"

## Team contribution map

The six agent teams are not independent programs; each is meant to move one or more of the outcome categories above. This table is the swarm-tier answer to "what is each team working toward, and how do they compose." It is a **judgment frame** — it changes by `outcome-direction` decision, not by drift — and it deliberately does not restate any team's mission, which is machine-readable in that team's `team.json::mission`. Each team states its own contribution in its own PoR; this table is the aggregate projection.

| Outcome category | Serves objective | Primary contributor | Supporting contributors | How the contribution shows up |
|---|---|---|---|---|
| Mission Control — system overview | `I2` | `team:infra-health` | `team:meta-optimization` | Platform reliability targets and instrumentation close the sensor holes that make aggregate health readable at all. |
| The Hive — scenario ecosystem | `I1` | `team:scenario-qa` | `team:director-swarm` | Structural audits and readiness reviews set which scenarios are headliner-ready; the portfolio decides which get attention. |
| The Forge — engineering velocity | `I1` | `team:director-swarm` | `team:meta-optimization`, `team:scenario-qa` | Goal throughput and backlog burn-down are portfolio outputs; skill/Action efficiency and defect rework are what make them cheaper. |
| Ledger — revenue & subscriptions | `T1` | `team:monetization` | `team:marketing-crew` | Catalog, tiers, and financial model define the revenue lines; marketing supplies the funnel that fills them. |
| Broadcast — marketing & growth | `T1`, `T3` | `team:marketing-crew` | `team:monetization` | Campaigns, channels, and publishing move funnel stages; monetization owns what the funnel converts into. |
| Panorama — aggregate view | `I1`, `I2` | `team:director-swarm` | all | Cross-category composition and capability gaps blocking outcome visibility. |

Three structural reads this table is meant to support. The first two run within the table; the third runs against [`../strategy/OBJECTIVES.md`](../strategy/OBJECTIVES.md) and is the one that catches whole missing programs rather than misassigned ones.

- **Unowned category.** A category with no primary contributor is an outcome nobody is working toward. All six are currently claimed.
- **Unattached team.** A team appearing in no row is doing work that traces to no stated outcome. All six currently appear. `meta-optimization` and `infra-health` appear only as supporting contributors by design — they are second-order teams whose output is the capacity of the other four, and their own first-order targets live in `path:docs/agent-system/FRAMEWORK_HEALTH.md` and `path:scenarios/infrastructure-manager/setpoint/reliability-setpoint.json` respectively.
- **Unserved objective.** An objective that no category and no team serves is stated intent with nothing behind it. **Two are currently unserved: `T2` (personal agency) has no team and no evidence source, and `I3` (enablement) has no owning lane.** Both are declared rather than hidden; they are the standing findings this read exists to surface, and neither is resolvable inside this charter.

The categories themselves still carry `pending-operator-input` for what "good" means in Mission Control, The Forge, and Panorama. Contribution is therefore directional, not quantified: this table says which team moves which needle, not how far it must move. With objectives now stated, those three are answerable by derivation rather than by unaided operator introspection — "good" for a category is the level at which its parent objective is served, so each should be filled by naming the `I1`/`I2` condition it must reach rather than by picking a number in isolation.

## Outcomes that may not fit a dashboard

If there's an outcome worth tracking that doesn't map cleanly to a Command Center page, it gets captured here until either (a) a page exists that covers it, or (b) it's demoted to an operator-eyes-only concern.

This section read `_No entries yet._` for as long as the charter had no layer above it. That was never true — it was unobservable, because with the six dashboard categories as the top of the hierarchy there was no vantage point from which an outcome could be noticed as not fitting one. Stating objectives supplies the vantage point, and the section immediately has an entry.

- **`T2` — personal agency.** Health, finances, and household outcomes. No Command Center page can hold these and none should: per [`../strategy/OBJECTIVES.md`](../strategy/OBJECTIVES.md) §"Evidence routing", a personal-domain terminal objective is measured by the scenario that serves it, not by a platform dashboard. Disposition (b) does not apply either — this is not operator-eyes-only, it is unbuilt. The entry stays here until a serving scenario exists and reports its own measure.

## The gap-closure loop

When a metric on any Command Center page shows as `gap`, the flow is:

1. Director-swarm's `outcome-strategist` (once active) spots the gap via `/api/v1/gaps`.
2. Proposes a backlog item or goal to build the missing data pipeline, with work type `outcome-gap`.
3. Human approves at the vision walk.
4. A capability-building team (or the relevant feature team) builds the pipeline.
5. The metric flips from `gap` → `live`.
6. Cycle repeats.

The pre-activation analogue: while `outcome-strategist` is disabled, the operator and `portfolio-manager` manually spot gap candidates during the vision walk. Expect this to be manual until `director-dashboard-gap-workflow` ships.

## Prediction ledger

The portfolio loop learns only if its decisions are falsifiable. Every `goal-proposal`, `goal-portfolio`, and `outcome-direction` decision carries a **prediction block** in its decision text:

```
Prediction:
- Metric: <Command Center metric, or an outcome category above>
- Direction: <up | down | reaches <value>>
- Horizon: <absolute date>
- Expected cost: <S | M | L — agent-run and operator-attention band>
```

Rules:

1. **EARS-shaped and falsifiable.** The block reads as: when the proposed work completes, the named metric shall move as stated by the horizon date. A prediction no evidence could contradict is not a prediction. Name the metric even when its Command Center status is `gap` — that names the sensor the gap-closure loop should build next.
2. **Scored by consequence, not self-report.** `outcome-strategist` scores matured predictions (horizon passed) against measured evidence: **hit**, **miss**, or **unmeasurable**. The proposal author never scores their own prediction. Scores land in `topic:outcome-target-record/YYYY-MM-DD` entries carrying the decision id, verdict, and evidence pointer.
3. **Equal-budget comparison.** Predictions state expected cost so calibration compares like for like — a hit that overran its cost band is a miss on cost, and "better" must never quietly mean "more expensive." This is the whole budget discipline the portfolio imports; there is still no cap on active goals (see [PORTFOLIO_PHILOSOPHY.md](../strategy/PORTFOLIO_PHILOSOPHY.md) §Concurrency).
4. **Bootstrap clause.** Recording predictions starts immediately, even while most metrics are `pending-command-center`. Sparse scoring is expected pre-Command-Center; an unscoreable cohort still reveals which sensors matter most, and an **unmeasurable** verdict must name the missing sensor and route to `outcome-gap`.
5. **Calibration feeds the philosophy.** Systematic misprediction — by ranking criterion, outcome category, or cost band — is the evidence trail for revising the ranking criteria in `PORTFOLIO_PHILOSOPHY.md` through `outcome-direction` or `goal-portfolio` decisions. The criteria are themselves under test; a philosophy that never loses to evidence is not being measured.

## What this charter does NOT do

- **Does not define specific success thresholds** (e.g., "MRR > $X by date Y"). Thresholds belong in Command Center configuration, goal-specific acceptance criteria, or per-decision prediction blocks — not charter-wide here.
- **Does not track metrics over time.** Command Center handles history.
- **Does not replace the monetization team's financial model.** For revenue math, see [monetization/evidence/FINANCIAL_MODEL.md](../../monetization/evidence/FINANCIAL_MODEL.md).

## Why honesty markers are deliberately visible

Thin sections and `gap`/`partial`/`pending-*` markers are intentional — their visibility is itself the signal of where sensor-building is high-leverage. As metrics land, the sensor map's observed column gets updated via approved decisions. Until then, the debt stays visible so future reviewers (human and agent) don't forget to come back.

## Updating this charter

Changes go through approved decisions with context `outcome-direction` (reframing what matters) or `outcome-gap` (a specific metric's status or pipeline changes). Sensor-map observed statuses and remaining `pending-*` markers get replaced when the relevant Command Center metric stabilizes — the replacement itself is an approved decision, not a silent edit.
