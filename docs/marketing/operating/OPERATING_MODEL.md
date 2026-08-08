# Marketing Operating Model

**Status:** target-state canon. This document defines how the marketing-crew works as a coherent system: loops, roles, topic surfaces, work item handoffs, and known gaps. It is the bridge between the strategic plan-of-record in `path:docs/marketing/` and the live team implementation under `path:scenarios/prompt-manager/store/teams/marketing-crew/`.

The current document adopts the generic team operating-model shape from `path:docs/agent-system/OPERATING_GRAPHS.md`.


Durable corpus belongs to the Source Ledger scope team:marketing-crew. Members file each actionable finding once through the unified Swarm Manager work feed; operator disposition is read from that same work item.
**Revised after capability consolidation.** The roster collapsed from six members to three, and roughly half the former topic families moved into the `content-desk` scenario as queryable state. The method is recorded in `prompt-manager skill read team-capability-consolidation`. Topic families marked `future` are target-state surfaces and are not live declarations.

The team is currently **paused** (`enabled: false`) pending completion of `content-desk` and `asset-studio`.

## Mission

Marketing turns evidence into public-facing artifacts without losing Vrooli's builder voice or overstating product reality. The team owns external voice, audience framing, campaign planning, content drafting, publishing discipline, and the learning loop that improves those systems over time.

Marketing does not own monetization strategy, product roadmap, or operator approval. It proposes; the operator accepts, rejects, or edits.

**Objective served.** `T1` — income (primary) and `T3` — contribution, via the OSS surface (`path:docs/director-swarm/strategy/OBJECTIVES.md`). Both are **terminal**: this team serves what the operator wants directly, not the system's capacity to want it. `T3` is served only partially today — the OSS discovery channel exists, but nothing yet addresses whether another operator can actually run this system.

**Outcome contribution.** Primary: **Broadcast** (marketing & growth) — campaigns, channels, and publishing move funnel stages. Supporting: **Ledger**, by supplying the funnel that revenue lines convert. The swarm-tier map of which team moves which outcome lives in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Team contribution map"; this paragraph is this team's own statement of it.

## Scope

Marketing owns the operating path from marketing signal to public-facing artifact or marketing-canon change:

- research signal intake and durable evidence;
- audience, channel, format, campaign, and brand-canon proposals;
- draft artifact production for every lane;
- challenge reports for weak marketing work items;
- learning-loop promotion from typed production observations into skills, plan-of-record docs, scenarios, capability gaps, or retirements.

Marketing does not own product prioritization, monetization strategy, legal approval, social-account credential operations, scheduler infrastructure, or the final work item to publish. Those surfaces route through work items, cross-team output, or explicit capability gaps.

## What the team no longer carries

Three capabilities left the team contract and became scenario state. This is the single largest change to this document, and it is why the roster is half its former size.

| Capability | Now owned by | Was |
|---|---|---|
| Campaign records, artifact slots, drafts, claims, review verdicts, publish history, coverage, subject familiarity | `content-desk` | ~13 topic families and five `.jsonl` files in the team's shared store |
| Character / scene / product identities, prompt specs, render provenance, produced assets, AI-UGC disclosure state | `asset-studio` | `path:docs/marketing/catalogs/rich-media/` JSON, hand-composed at production time |
| Account activation, warming, cadence, credentials, posting execution | `social-media-scheduler` (not yet modernized) | unowned; deferred in every prior revision |

The rule that decided each row: **evidence and judgment stay knowledge topics; operational state with a lifecycle becomes scenario data.** A surface with a status, a gate, or a query worth running is modelled badly by an append-only topic family.

Two consequences worth stating plainly:

- **`content-publish-proposal` no longer exists as a work-routing contract.** Operator approval did not disappear — it moved from a prompt-manager work item into `content-desk`'s operator-only approval gate, where a draft cannot reach `approved` while it cites an unverified claim or names a post type that is still v0. The gate is executable; the work item was not.
- **Stale-work item hygiene is no longer a topic family.** It is governance configuration: `operatingContract.governance.operatorDispositionPolicy`, owned by `marketing-contrarian`, firing after 14 heartbeats.

## Operating Loops

Marketing has five loops:

1. **Research loop** — collect operator, bookmark, web, market, competitor, channel, and format signals; route them into durable evidence or work items.
2. **Planning loop** — decide what should be marketed, to whom, on which channel, and why now.
3. **Draft loop** — turn open campaign slots into drafts with declared claims, sources, audience, lane, channel, and format.
4. **Review and publish loop** — challenge proposals, score drafts against failure modes, and let the operator approve. Release and publish history live in `content-desk`.
5. **Learning loop** — turn publish history, coverage state, and repeated production lessons into typed observations, canon updates, skills, scenarios, capability gaps, or retirements.

The loops are sequential when a full campaign flows through the system, but members may also operate independently when their trigger fires. The producer can gather evidence without an open campaign; the brand-manager can promote or retire stale typed observations without changing a campaign.

**The learning loop has never had data.** The publish history it is meant to consume is empty. It becomes real with the first published artifact recorded in `content-desk`, not before.

## Operating Graph

The first diagram is the compact operating-loop view. It is useful when checking role boundaries, approval gates, and whether evidence flows through planning, drafting, publishing, and learning.

```mermaid
flowchart LR
  OP[Operator / vision walk]
  SI[Signal inbox]
  WEB[Manual web and external sources]

  OP --> P[Producer]
  SI --> P
  WEB --> P

  P --> EVID[research evidence topics]
  EVID --> BM[Brand Manager]
  EVID --> P

  BM --> CANON[marketing and narrative canon]
  BM --> CAMP[campaign-launch-proposal]
  CAMP --> CD[(content-desk<br/>campaigns · slots)]

  CD --> P
  P --> DRAFT[(content-desk<br/>drafts · claims)]
  AS[(asset-studio<br/>specs · assets)] --> DRAFT

  DRAFT --> CONTRA[Marketing Contrarian]
  CAMP --> CONTRA
  CONTRA --> CHAL[(content-desk<br/>review verdicts · challenges)]
  CHAL --> BM
  CHAL --> P

  DRAFT --> APPROVAL[Operator approval<br/>content-desk gate]
  APPROVAL --> PUB[Publish<br/>manual today · scheduler later]
  PUB --> LEDGER[(content-desk<br/>publish history · coverage)]

  LEDGER --> LEARN[learning inputs]
  LEARN --> MCO[marketing-craft-observation]
  MCO --> BM
```

The second diagram is the full topic-level view. It is the reference shape for validating whether topic producers, readers, work items, and durable logs form a coherent marketing system.

<!-- prompt-manager-graph:
id: marketing-operating-model
scope: team
team: marketing-crew
mode: contract
actor_group.marketing-members: team-members
actor_group.work-owners: none
actor_group.benchmark-consumers: none
actor_alias.any marketing member: group:marketing-members
actor_alias.work owner: group:work-owners
actor_alias.monetization team: group:benchmark-consumers
actor_alias.work owners: group:work-owners
actor_alias.learning synthesis: process:learning-synthesis
actor_alias.meta-optimization: external:meta-optimization
actor_alias.director-swarm: external:director-swarm
-->
```mermaid
flowchart LR
  %% @node AS topic:audience-scan/*
  AS[(audience-scan/*)]
  %% @node BM member:brand-manager
  BM[Brand Manager]
  %% @node BRANDSNAP topic:brand-snapshot/*
  BRANDSNAP[(brand-snapshot/*)]
  %% @node BRANDSNAPSHO topic:brand-snapshot/YYYY-MM-DD
  BRANDSNAPSHO[(brand-snapshot/YYYY-MM-DD)]
  %% @node CANON1 por:docs/marketing/strategy/STRATEGY.md
  CANON1[/docs/marketing/strategy/STRATEGY.md/]
  %% @node CANON2 por:docs/marketing/strategy/AUDIENCES.md
  CANON2[/docs/marketing/strategy/AUDIENCES.md/]
  %% @node CHAN topic:channel-scan/*
  CHAN[(channel-scan/*)]
  %% @node COMP topic:competitor-record/*
  COMP[(competitor-record/*)]
  %% @node FORMAT topic:format-scan/*
  FORMAT[(format-scan/*)]
  %% @node HOOK topic:hook-record/*
  HOOK[(hook-record/*)]
  %% @node LEARN process:learning-synthesis
  LEARN([Learning synthesis])
  %% @node MB topic:monetization-benchmark-adjacent-record/*
  MB[(monetization-benchmark-adjacent-record/*)]
  %% @node MCO topic:marketing-craft-observation/*
  MCO[(marketing-craft-observation/*)]
  %% @node OP external:operator
  OP([Operator])
  %% @node P member:producer
  P[Producer]
  %% @node SKILL topic:skill-scan/*
  SKILL[(skill-scan/*)]
  %% @node VW external:vision-walk
  VW([Vision walk])
  %% @node WF topic:workflow-scan/*
  WF[(workflow-scan/*)]

  OP --> BM
  OP --> P
  OP --> MCO
  VW --> BM
  VW --> P
  VW --> MCO
  BM --> CANON2
  BM --> CANON1
  BM --> BRANDSNAP
  P --> AS
  P --> CHAN
  P --> COMP
  P --> FORMAT
  P --> HOOK
  P --> MB
  P --> SKILL
  P --> WF
  LEARN --> MCO
  AS --> BM
  AS --> P
  BRANDSNAPSHO --> BM
  CHAN --> BM
  CHAN --> P
  COMP --> BM
  COMP --> P
  FORMAT --> BM
  FORMAT --> P
  HOOK --> BM
  HOOK --> P
  MCO --> BM
  MCO --> LEARN
  SKILL --> P
  WF --> P
```

## Topic Catalog

These are the knowledge-topic families the team still owns. Everything absent from this table that appeared in a prior revision moved to `content-desk`; see §"What the team no longer carries".

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:audience-scan/*` | live | member:producer | member:brand-manager, member:producer | Audience pain, vocabulary, buyer triggers, objections, and persona evidence. |
| `topic:brand-snapshot/*` | live | member:brand-manager | member:brand-manager | Snapshot of canon drift, typed learning state, campaign signals, and promotion/retirement candidates. |
| `topic:brand-snapshot/YYYY-MM-DD` | live | member:brand-manager | member:brand-manager | Snapshot of canon drift, typed learning state, campaign signals, and promotion/retirement candidates. |
| `topic:channel-scan/*` | live | member:producer | member:brand-manager, member:producer | Evidence that a channel is worth activating, deprioritizing, or handling differently. |
| `topic:competitor-record/*` | live | member:producer | member:brand-manager, member:producer | Competitor pricing, packaging, positioning, changelog, or claim evidence. |
| `topic:format-scan/*` | live | member:producer | member:brand-manager, member:producer | Evidence that a post format or channel-native format is worth using or codifying. |
| `topic:hook-record/*` | live | member:producer | member:brand-manager, member:producer | Reusable hook and framing observations; the promotion source for the hook library. |
| `topic:marketing-craft-observation/*` | live |  | member:brand-manager | Typed production observations that may feed canon, skill, scenario, Swarm Manager work item, or retirement work items. |
| `topic:monetization-benchmark-adjacent-record/*` | live | member:producer |  | Pricing, packaging, or market facts found by marketing but owned strategically by monetization. |
| `topic:skill-scan/*` | live | member:producer | member:producer | External skills, prompts, reusable processes, or capability ideas. |
| `topic:workflow-scan/*` | live | member:producer | member:producer | External workflows, playbooks, agent setups, or business processes worth deconstructing. |

Evidence topics are append-only and carry no lifecycle. If a proposed topic family needs a status, a gate, or a query, it is scenario state and belongs in `content-desk` — that test is the boundary, and it is the one this revision applied.

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | `topic:audience-scan/*` | producer | Raw signal reaches the producer directly and is classified into the evidence family it belongs to. |
| Vision walk | `topic:channel-scan/*` | producer | Signals become evidence, campaign proposals, campaign slots, or capability gaps. |
| Operator | `topic:marketing-craft-observation/*` | brand-manager | Direction that is already a concrete planning call enters the unified swarm work stream rather than a second inbox. |
| Repeated production lessons | `topic:marketing-craft-observation/*` | brand-manager | Drained into canon, skills, scenarios, capability gaps, or retirement. |

Two triggers are deliberately absent from this table because the runtime does not declare them. The **signal inbox** is not listed in any member's `external_producers`, and **open campaign slots** live in `content-desk`, which is not a participant in the team contract graph. Both are described in §"Operating Loops" and tracked as gaps 4 and 10.

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Audience evidence | `topic:audience-scan/*` | producer, brand-manager | Persona, objection, and buyer-trigger evidence for planning and drafting. |
| Competitive and hook evidence | `topic:competitor-record/*` | producer, brand-manager | Positioning and framing evidence. |
| Channel and format evidence | `topic:channel-scan/*` | producer, brand-manager | Where and in what shape to publish. |
| Monetization-adjacent facts | `topic:monetization-benchmark-adjacent-record/*` | producer | Pricing and packaging facts held for routing to monetization; see gap 9. |
| Brand drift snapshots | `topic:brand-snapshot/*` | brand-manager | Canon drift, typed learning state, and promotion or retirement candidates. |
| Canon changes | `docs/marketing/strategy/STRATEGY.md` | all marketing members and cross-team consumers | Keep public-facing voice coherent. |
| Audience canon changes | `docs/marketing/strategy/AUDIENCES.md` | all marketing members | Keep persona definitions coherent across lanes. |
| Capability gaps | `topic:marketing-craft-observation/*` | producer, brand-manager, marketing-contrarian | Route blocked work to the right improvement path in swarm-manager. |
| Coverage gaps | `topic:marketing-craft-observation/*` | member:marketing-contrarian | Convert missing coverage into campaign slots or a reviewed backlog item. |

Campaign records, drafts, claims, review verdicts, publish history, and produced assets are **not** listed here. They are scenario state in `content-desk` and `asset-studio`, queried through those scenarios rather than emitted as team outputs.

## Feedback / Capability Improvement Loop

Marketing improves itself through four explicit exits:

1. **Typed learning observations** — repeated lessons and workarounds enter `topic:marketing-craft-observation/*`; the brand-manager drains them into canon, skills, scenarios, capability gaps, or retirement.
2. **Work review** — marketing-contrarian files concrete failure-mode evidence into `topic:marketing-craft-observation/*` and the unified swarm work stream.
3. **Capability gaps** — the producer files `topic:marketing-craft-observation/*` when missing access, tooling, scheduler support, media generation, telemetry, account state, or scenario capability blocks the work.
4. **Coverage gaps** — the producer files `topic:marketing-craft-observation/*` when a SKU, lane, channel, or campaign lacks sufficient marketing coverage.

General code/scenario defects should use scenario-qa's `report-bug` flow. System-level friction that is not a defect should use meta-optimization's `report-friction` flow. Marketing should not turn every local frustration into a marketing-craft observation when a universal observation flow is the correct destination.

## Roles

Three members. Every remaining separation is a control, a clock, or a judgment boundary — the pipeline-stage separations the previous roster encoded are now enforced by gates in `content-desk`.

### Producer

Owns evidence and drafts. The former researcher and the two lane advertisers merged into this role: lane is a field on a draft, not a pipeline, and research that is not driving a draft is speculative.

Primary responsibilities:

- draw open work from active campaign slots and draft against it;
- gather the audience, competitor, hook, workflow, external-skill, channel, and format evidence a draft needs;
- declare every factual assertion as a claim with evidence attached, preferring a re-runnable check over a citation;
- append raw observations before proposing interpretation, and propose canon changes only when evidence converges;
- verify feature claims against monetization canon before drafting subscription-lane material;
- route monetization-adjacent facts to monetization without duplicating their canon.

Hard limits: never publish, never approve its own draft, never edit plan-of-record canon directly.

### Brand Manager

Owns canon and campaign planning. Reads research evidence, publish outcomes, coverage state, and typed production observations; proposes canon, campaign, capability-work, skill, scenario, or retirement work items.

Primary responsibilities:

- steward `path:docs/marketing/` and `path:docs/narrative/` through accepted work items;
- convert accepted campaign planning into campaign records with a declared artifact slot budget — slots are a hard cap, not a target, and are what bound in-flight work and operator review load;
- drain `topic:marketing-craft-observation/*` and propose promotion or retirement through the relevant owned work item;
- detect voice, positioning, campaign, or narrative drift from recent drafts and published artifacts;
- prevent campaign sprawl when signal is weak.

Temporary: `channel-update` is held here, inherited from the retired publisher role. It moves when account operations gain a scheduler-side home.

### Marketing Contrarian

Owns challenge and stale-work-item hygiene. It does not generate positive marketing work, and it is the one separation that could not be merged into the producer.

Primary responsibilities:

- score pending proposals and drafts against framework-level and type-level failure modes, including the AI-UGC guardrails;
- **hunt factual assertions the producer made but did not declare as claims** — this is the compensating control for author self-reporting, and the reason this role remains separate;
- maintain resolution state for each open challenge;
- run stale-work-item hygiene per the governance policy;
- propose framework updates only when observed failures fall outside the current framework;
- check that the pipeline boundary holds: evidence → draft → operator approval → publish record.

## Typed Learning Drainage

Typed learning observations are not a destination of last resort. They are structured evidence for patterns that are real but not yet permanent.

Every marketing-craft observation must eventually resolve to one of four outcomes:

1. **Promote to a skill** when it is executable procedure.
2. **Promote to plan-of-record** when it is strategic canon.
3. **Promote to a scenario or config change** when automation should replace the workaround.
4. **Retire** when the observation was transient, duplicated, or superseded.

Append and read responsibilities are deliberately different:

- any marketing member may write typed marketing-craft observations when the lesson is truly marketing-specific;
- brand-manager drains typed observations and proposes promotion or retirement through the relevant owned work item;
- producer consumes promoted canon and skill updates, not raw craft observations by default;
- marketing-contrarian reviews observation-derived proposals when scoring whether the proposed permanent structure actually resolves the issue.

If a marketing-craft observation family grows for multiple heartbeats without promotion, retirement, or a clear revisit marker, that is a system smell. The brand-manager should raise the relevant canon, skill, scenario, config, or `capability-work` work item.

## Current Implementation Gaps

1. **The team is paused** via `enabled: false` in `scenarios/prompt-manager/store/teams/marketing-crew/team.json`. `content-desk` and `asset-studio` are under development. The roster and this document are migrated ahead of them deliberately — roster re-derivation needs the scenarios' domain maps and gates, not their code. *Target state: the team is enabled once step 9 of Adoption / Validation passes.*
2. **No paired tool skill exists yet.** The producer's drafting mechanics are intentionally unversed in specific commands until `content-desk`'s CLI surface settles. *Target state: an `x-content-desk` tool skill authored through `team-tool-mapping`, cited in the producer's available skills.*
3. **No state import has run.** The team's shared `.jsonl` files remain the only copy of publish, coverage, mention, and draft state, and must not be deleted before `content-desk`'s importer has consumed them. *Target state: import complete with counts verified per source file, and the shared files retired.*
4. **`research-inbox/*` is no longer declared by any member.** Raw signal now reaches the producer through direct member context, so the signal inbox is not a backed trigger. *Target state: either a declared intake family with the signal inbox listed in the producer's `external_producers`, or an explicit work item that direct context is sufficient.*
5. **the content-desk approval record is no longer declared.** It tracked publisher follow-through on accepted publish work items. *Target state: retired permanently — with `content-publish-proposal` gone, its purpose is served by the approval record in `content-desk`.*
6. **Account operations remain unowned.** Activation, warming, cadence, persona accounts, and credentials are deferred to `social-media-scheduler`, which is still pre-template. *Target state: those capabilities live in a modernized scheduler, and `channel-update` moves there from brand-manager.*
7. **Publishing is manual.** An approved draft is posted by the operator and recorded in `content-desk`. *Target state: manual remains correct at current volume — one brand account, no persona accounts — and is revisited only when the scheduler exists.*
8. **Ten of twelve post types are v0.** They have strategic canon and no paired skill, so `content-desk` will refuse to approve drafts of those types. *Target state: at least one image or video type activated to v1, giving `asset-studio` a consumer.*
9. **The cross-team route to monetization is undeclared.** The producer writes `monetization-benchmark-adjacent-record/*` but no runtime relationship carries it to the monetization team. *Target state: a declared cross-team relationship, or an accepted work item that the monetization team polls the family itself.*
10. **Scenario state is outside the contract graph.** The `marketing-operating-model` graph validates topics, work items, members, and declared externals; it has no node kind for scenario-owned state, so the campaigns, drafts, claims, and publish history held in `content-desk` cannot appear in it. *Target state: unchanged — the compact loop diagram carries the system view, and this graph stays a team contract.*

## Adoption / Validation

Adopt this model in order:

1. Keep this document and `path:docs/marketing/README.md` as the target-state plan-of-record.
2. Keep `path:scenarios/prompt-manager/store/teams/marketing-crew/team.json` registered against this model.
3. Keep member `RESPONSIBILITIES.md` and `HEARTBEAT.md` files aligned with the five loops.
4. Keep member `topics.json` files aligned with the ten live evidence and learning topic families.
5. Run `prompt-manager graph operating-model validate --team marketing-crew --id marketing-operating-model`.
6. Run `prompt-manager graph operating-model diff --team marketing-crew --id marketing-operating-model`.
7. Run `prompt-manager graph operating-model coverage --team marketing-crew --id marketing-operating-model`.
8. Rerun `prompt-manager graph topics --json` and decide which remaining warnings are accepted operator-only logs versus real gaps.
9. Resume the team only after `content-desk`'s CLI surface is settled, its paired skill exists, and the state import has run with verified counts.

Do not make the validator quiet before the operating model is coherent. The topic graph is a check on the model, not a substitute for it.
