# Director Swarm Operating Model

**Status:** contract canon. This document defines how `director-swarm` works as a coherent system: portfolio hygiene, goal interpretation, outcome gap routing, prediction calibration, and morning vision-walk preparation.

**Loop status:** paused — recorded in the heartbeat control plane as `paused-manual` on 2026-07-24 (resume via `prompt-manager team heartbeat-control director-swarm resume`). Current-state values across this PoR reflect the pause, not loop failure. On resume, the first heartbeat follows the resume protocol in `portfolio-manager/HEARTBEAT.md`.

The current document adopts the generic team operating-model shape from `path:docs/agent-system/OPERATING_GRAPHS.md`.

## Mission

Director Swarm keeps Vrooli's goal portfolio flowing through Swarm Manager and surfaces outcome-driven strategy from Command Center's measured outcomes and gaps registry.

**Objective served.** `I1` — capability compounding. This team also **owns the objective set itself** ([`../strategy/OBJECTIVES.md`](../strategy/OBJECTIVES.md)): it curates the operator's stated intent, holds the coverage rule that joins objectives to the team roster, and raises `outcome-direction` or `capability-gap` decisions when coverage reports a hole. Owning the file does not mean authoring the objectives — those are operator-authored.

**Outcome contribution.** Primary: **The Forge** (engineering velocity) — goal throughput and backlog burn-down are direct portfolio outputs — and **Panorama** (aggregate view), where this team owns cross-category composition and the capability gaps blocking outcome visibility. Supporting: **The Hive**, by deciding which scenarios receive attention. The swarm-tier map of which team moves which outcome lives in [`../evidence/OUTCOMES_CHARTER.md`](../evidence/OUTCOMES_CHARTER.md) §"Team contribution map"; this paragraph is this team's own statement of it.

## Scope

Director Swarm owns:

- portfolio ranking criteria, goal theme interpretation, and roadmap positioning;
- decision context preparation for new goals, goal supplements, readiness judgments, outcome gaps, and vision drift;
- approved decision application where Swarm Manager tooling supports the exact action;
- outcome-category framing for Command Center, the gap-closure loop that turns missing metrics into capability work, and the prediction ledger that scores past portfolio decisions against measured outcomes;
- routing of cross-team `capability-gap` decisions into goal or backlog proposals;
- morning vision-walk briefing across portfolio, monetization, marketing, meta-optimization, infra-health, and life-audit signals.

Director Swarm does not own:

- live goal status, milestones, dependencies, backlog items, or execution evidence, which belong to Swarm Manager (see `path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md` for the operator loop this team feeds);
- operator-authored vision or architecture canon, which members may flag but not rewrite;
- monetization strategy, marketing strategy, scenario QA, infra-health reliability canon, or meta-optimization policy;
- deployment, external execution, code changes, or team deployment.

## The Portfolio Control Loop

The portfolio loop is the team's reason to exist, so its architecture is stated plainly: today it runs open-loop — steering comes forward from `path:docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md` and operator taste, and nothing feeds back how past portfolio decisions turned out. The prediction ledger in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Prediction ledger" is the error signal that closes it: portfolio decisions carry falsifiable predictions, `outcome-strategist` scores matured predictions against measured evidence, and systematic misprediction becomes the evidence trail for revising the ranking criteria themselves. Until Command Center exposes stable metrics, the scoring side is sparse by design — predictions are still recorded from day one so a scoreable cohort exists when the sensors arrive.

## Operating Loops

Director Swarm has four loops:

1. **Portfolio interpretation loop** - compare active and proposed work against [`../strategy/PORTFOLIO_PHILOSOPHY.md`](../strategy/PORTFOLIO_PHILOSOPHY.md), then propose bounded `goal-portfolio`, `goal-proposal`, `goal-supplement`, or `goal-readiness` decisions, each carrying its required prediction block.
2. **Roadmap positioning loop** - keep [`../strategy/ROADMAP.md`](../strategy/ROADMAP.md) aligned with Swarm Manager's goal set without duplicating live status: the roadmap holds `goal:` typed references plus theme and positioning only, and each `member:portfolio-manager` heartbeat diffs live goal names against those references, proposing a bounded `goal-portfolio` decision on any delta.
3. **Outcome and calibration loop** - read Command Center gaps when available, use [`../evidence/OUTCOMES_CHARTER.md`](../evidence/OUTCOMES_CHARTER.md) for category framing, score matured predictions, and raise `outcome-gap` or `outcome-direction` decisions.
4. **Vision-walk preparation loop** - compile read-only operator briefings, state the self-improvement loop ladder honestly, and flag drift against `path:VISION.md` or `path:docs/concepts/ARCHITECTURE.md` through `vision-update`.

## Operating Graph

This graph is the team-level contract. It shows how operator direction and vision-walk sessions flow into director-swarm members, then out through approved decisions, knowledge records, and read-only handoffs.

<!-- prompt-manager-graph:
id: director-swarm-operating-model
scope: team
team: director-swarm
mode: contract
actor_alias.operator: external:operator
actor_alias.vision walk: external:vision-walk
-->
```mermaid
flowchart LR
  subgraph INFLOWS["Inflows / Producers"]
    %% @node OP external:operator
    OP([Operator])
    %% @node VW external:vision-walk
    VW([Vision Walk])
  end

  %% Members
  %% @node PM member:portfolio-manager
  PM[Portfolio Manager]
  %% @node OS member:outcome-strategist
  OS[Outcome Strategist]
  %% @node VWP member:vision-walk-prep
  VWP[Vision Walk Prep]

  %% Topics
  %% @node APP topic:decision-application/<decision-id>
  APP[(decision-application/<decision-id>)]
  %% @node PORT topic:goal-portfolio-record/YYYY-MM-DD
  PORT[(goal-portfolio-record/YYYY-MM-DD)]
  %% @node OUT topic:outcome-target-record/YYYY-MM-DD
  OUT[(outcome-target-record/YYYY-MM-DD)]
  %% @node WALK topic:vision-walk-record/<date>/<slug>
  WALK[(vision-walk-record/<date>/<slug>)]
  %% @node RI topic:research-inbox/*
  RI[(research-inbox/*)]
  %% @node OI topic:opportunity-inbox/*
  OI[(opportunity-inbox/*)]
  %% @node VI topic:validation-inbox/*
  VI[(validation-inbox/*)]

  %% Peer teams
  %% @node MKT team:marketing-crew
  MKT[[Marketing Crew]]
  %% @node MON team:monetization
  MON[[Monetization]]

  %% Decisions
  %% @node GP decision:goal-portfolio
  GP{goal-portfolio}
  %% @node GS decision:goal-supplement
  GS{goal-supplement}
  %% @node GPR decision:goal-proposal
  GPR{goal-proposal}
  %% @node GR decision:goal-readiness
  GR{goal-readiness}
  %% @node CG decision:capability-gap
  CG{capability-gap}
  %% @node OG decision:outcome-gap
  OG{outcome-gap}
  %% @node OD decision:outcome-direction
  OD{outcome-direction}
  %% @node VU decision:vision-update
  VU{vision-update}

  subgraph OUTFLOWS["Downstream outflows"]
    %% @node APPROVAL process:operator-approval
    APPROVAL([Operator approval])
    %% @node CANON process:operator-curated-canon
    CANON([Operator-curated canon])
    %% @node SMWORK external:swarm-manager-work
    SMWORK([Swarm Manager work])
  end

  OP --> PM
  OP --> OS
  OP --> VWP
  VW --> PM
  VW --> OS

  APP --> PM
  PORT --> PM
  PM --> PORT
  PM --> APP
  PM --> GP
  PM --> GS
  PM --> GPR
  PM --> GR
  PM --> CG
  CG --> PM
  VU --> PM
  OD --> PM

  APP --> OS
  OUT --> OS
  OS --> OUT
  OS --> OG
  OS --> OD

  WALK --> VWP
  VWP --> WALK
  VWP --> VU

  VWP --> RI
  VWP --> OI
  VWP --> VI
  RI --> MKT
  OI --> MON
  VI --> MON

  GP --> APPROVAL
  GS --> APPROVAL
  GPR --> APPROVAL
  GR --> APPROVAL
  CG --> APPROVAL
  OG --> APPROVAL
  OD --> APPROVAL
  VU --> APPROVAL
  APPROVAL --> CANON
  APPROVAL --> SMWORK
```

Swarm Manager goal state and Command Center metrics are tool-read surfaces, not knowledge producers: members inspect them directly (`swarm-manager goals context`, Command Center `/api/v1/gaps`) and never copy live status into the PoR.

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:decision-application/<decision-id>` | live | portfolio-manager | portfolio-manager, outcome-strategist | Record exact accepted-decision application work and avoid duplicate application. |
| `topic:goal-portfolio-record/YYYY-MM-DD` | live | portfolio-manager | portfolio-manager | Snapshot portfolio interpretation against the ranking criteria. |
| `topic:outcome-target-record/YYYY-MM-DD` | live | outcome-strategist | outcome-strategist | Snapshot outcome target, gap, or prediction-score interpretation when Command Center evidence exists. |
| `topic:vision-walk-record/<date>/<slug>` | live | vision-walk-prep | vision-walk-prep | Read-only morning vision-walk briefing material. |
| `topic:research-inbox/*` | live | vision-walk-prep | team:marketing-crew | Route vision-walk research signal into the marketing research queue for that team's drainer to classify. |
| `topic:opportunity-inbox/*` | live | vision-walk-prep | team:monetization | Route vision-walk revenue-opportunity signal into the monetization opportunity queue for that team's drainer to classify. |
| `topic:validation-inbox/*` | live | vision-walk-prep | team:monetization | Route vision-walk demand-validation signal into the monetization validation queue for that team's drainer to classify. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `goal-portfolio` | portfolio-manager | Rank goals as active now, track, or defer. Carries a prediction block. | Swarm Manager state, `topic:goal-portfolio-record/YYYY-MM-DD`, the heartbeat roadmap diff, or operator direction shows portfolio emphasis drift. | Updates Swarm Manager goal state where tooling supports the exact action, theme rows in `path:docs/director-swarm/strategy/ROADMAP.md`, or `path:docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md`. |
| `goal-supplement` | portfolio-manager | Propose supporting backlog work under existing goals. | An existing goal lacks a bounded supporting backlog item. | Creates or updates a Swarm Manager backlog item under the owning goal. |
| `goal-proposal` | portfolio-manager | Propose candidate new goals for operator approval. Carries a prediction block. | A durable outcome needs multiple dependent backlog items under one shared outcome. | Creates a Swarm Manager goal and may update `path:docs/director-swarm/strategy/ROADMAP.md`. |
| `goal-readiness` | portfolio-manager | Judge whether backlog items are detailed enough to execute. | A backlog item is ambiguous, underspecified, or ready for execution. | Updates Swarm Manager backlog readiness or routes clarification to the operator. |
| `capability-gap` | portfolio-manager | Route a missing capability from another team into the portfolio. | An accepted upstream capability-gap decision from meta-optimization or infra-health names a tool, scenario, or data capability the fleet lacks. | Creates a Swarm Manager goal or backlog proposal for the missing capability, or rejects with evidence. |
| `outcome-gap` | outcome-strategist | Request missing Command Center data pipelines or instrumentation. | Command Center exposes a `gap` metric or `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` has an unresolved `pending-command-center` marker. | Creates capability work for Command Center or the owning implementation team. |
| `outcome-direction` | outcome-strategist | Recommend a portfolio emphasis change from measured outcome evidence. Carries a prediction block. | Measured Command Center data or a scored prediction shows a material portfolio misalignment. | Updates portfolio emphasis through Swarm Manager or `path:docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md`. |
| `vision-update` | vision-walk-prep | Surface operator-north-star or architecture drift without editing canon directly. | A proposed direction conflicts with `path:VISION.md` or `path:docs/concepts/ARCHITECTURE.md`. | Routes an operator-authored update to `path:VISION.md`, `path:docs/concepts/ARCHITECTURE.md`, or rejects the drifting proposal. |

Prediction blocks follow `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Prediction ledger": which metric or outcome category moves, in which direction, by when, at what expected cost band.

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | direct member context, `decision:vision-update` | portfolio-manager, outcome-strategist, vision-walk-prep | Operator direction can seed portfolio or vision decisions, but accepted effects still require approval. |
| Vision walk | direct member context | portfolio-manager, outcome-strategist | Morning-walk conclusions seed bounded decisions; the walk itself never mutates state directly. |
| Cross-team capability gaps | `decision:capability-gap` | portfolio-manager | Accepted upstream gaps from meta-optimization or infra-health become goal or backlog proposals, never silent adoptions. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Approved goal decisions | `decision:goal-portfolio`, `decision:goal-proposal`, `decision:goal-supplement`, `decision:goal-readiness`, `decision:capability-gap` | operator, Swarm Manager via decisions | Move portfolio work into accepted Swarm Manager state. |
| Approved outcome decisions | `decision:outcome-gap`, `decision:outcome-direction` | operator, Command Center owners, owning implementation teams | Turn missing or surprising metrics into routed capability work. |
| Vision-walk briefings | `topic:vision-walk-record/<date>/<slug>` | operator | Prepare morning review without changing canon directly. |
| Cross-team vision-walk signal | `topic:research-inbox/*`, `topic:opportunity-inbox/*`, `topic:validation-inbox/*` | `team:marketing-crew`, `team:monetization` | Hand vision-walk signal to the peer team that owns the domain, so the receiving drainer classifies it under its own taxonomy. |
| Prediction calibration evidence | `topic:outcome-target-record/YYYY-MM-DD`, `path:docs/director-swarm/strategy/PORTFOLIO_PHILOSOPHY.md`, `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` | operator, portfolio-manager | Feed scored predictions back into ranking-criteria revisions through approved decisions. |

## Feedback / Capability Improvement Loop

1. `member:portfolio-manager` compares Swarm Manager state with `topic:goal-portfolio-record/YYYY-MM-DD` and raises `decision:goal-portfolio` when strategy and live portfolio diverge, attaching the required prediction block.
2. `member:outcome-strategist` checks Command Center gaps against `topic:outcome-target-record/YYYY-MM-DD` and raises `decision:outcome-gap` when a missing metric blocks outcome judgment.
3. `member:outcome-strategist` scores matured predictions from past portfolio decisions against measured evidence and records hits, misses, and unmeasurables in `topic:outcome-target-record/YYYY-MM-DD`; systematic misprediction routes to `decision:outcome-direction` proposing a ranking-criteria revision.
4. `member:vision-walk-prep` consolidates cross-team signals into `topic:vision-walk-record/<date>/<slug>` and raises `decision:vision-update` only when the operator north star or architecture canon needs review.
5. Accepted decisions update Swarm Manager, Command Center capability work, or `path:docs/director-swarm/` canon; rejected decisions remain evidence for future review.

## Current Implementation Gaps

- `external:command-center` still shows registry `gap`/`partial` for several outcome categories (see the Outcomes Charter §"Sensor map" for the per-category observation), so prediction scoring runs sparse: predictions are recorded on every qualifying decision now, and target-state is for `member:outcome-strategist` to score them against live Command Center metrics.
- Swarm Manager's goal-workflow tail is dead today (audited 2026-07-23: `POST /goals/{name}/workflow-runs/{id}/apply` has no production caller, and `add_item` proposals are rejected at goal scope), so goal `plan`/`discover`/`milestone-review` runs strand their proposals; `member:portfolio-manager` must treat goal creation and backlog creation as the only accepted effects that land, and target-state is a closed apply loop owned by the Swarm Manager workstream (`path:scenarios/swarm-manager/docs/concepts/OPERATOR-JOURNEYS.md` names the loop this restores).
- The command-center CLI exposes no `gaps` or `dashboards` verb, so sensor reads are raw HTTP against an on-demand scenario; target-state is a thin CLI sensor verb raised through `decision:outcome-gap`.
- `goal:` references in `path:docs/director-swarm/strategy/ROADMAP.md` are validated only by the `member:portfolio-manager` heartbeat diff today; target-state is knowledge-observatory docs health resolving `goal:` markers against swarm-manager automatically (see `path:docs/reference/machine-readable-references.md` §"Validation Ownership").
- `path:docs/director-swarm/manifest.json` is present, but target-state is for prompt-manager PoR manifest validation to run automatically against this folder.
- Team heartbeats are paused; target-state is re-enabled heartbeats whose first pass follows the resume protocol in `portfolio-manager/HEARTBEAT.md`.
- `member:vision-walk-prep` declares cross-team inbox outputs (`research-inbox/*` to marketing-crew, `opportunity-inbox/*` and `validation-inbox/*` to monetization) that are deliberately absent from this team's contract graph; target-state keeps them owned by the receiving teams' topic catalogs, so the validator's declared-output warnings on them are accepted.

## Adoption / Validation

- `prompt-manager graph operating-model validate --team director-swarm --id director-swarm-operating-model`
- `prompt-manager graph operating-model diff --team director-swarm --id director-swarm-operating-model`
- `prompt-manager graph operating-model coverage --team director-swarm --id director-swarm-operating-model`
