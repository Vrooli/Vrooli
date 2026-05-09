# Marketing Operating Model

**Status:** target-state canon. This document defines how the marketing-crew should work as a coherent system: loops, roles, topic surfaces, decision handoffs, and known gaps. It is the bridge between the strategic plan-of-record in `path:docs/marketing/` and the live team implementation under `path:scenarios/prompt-manager/store/teams/marketing-crew/`.

The current implementation mostly matches this model after the first adoption pass. Topic families marked with `future` are target-state surfaces that should not be treated as live declarations until the marketing team config and member `topics.json` files are updated.

## Mission

Marketing turns evidence into public-facing artifacts without losing Vrooli's builder voice or overstating product reality. The team owns external voice, audience framing, campaign planning, content drafting, publishing discipline, and the learning loop that improves those systems over time.

Marketing does not own monetization strategy, product roadmap, or operator approval. It proposes; the operator accepts, rejects, or edits.

## Scope

Marketing owns the operating path from marketing signal to public-facing artifact or marketing-canon change:

- research signal intake and durable evidence;
- audience, channel, format, campaign, and brand-canon proposals;
- draft artifact production for OSS and subscription lanes;
- publishing execution state and coverage snapshots;
- challenge reports for weak marketing decisions;
- learning-loop promotion from notebook debt into skills, plan-of-record docs, scenarios, or retirements.

Marketing does not own product prioritization, monetization strategy, legal approval, social-account credential operations, scheduler infrastructure, or the final decision to publish. Those surfaces route through decisions, cross-team output, or explicit capability gaps.

## Operating Loops

Marketing has five loops:

1. **Research loop** — collect operator, bookmark, web, market, competitor, channel, and format signals; route them into durable evidence or decisions.
2. **Planning loop** — decide what should be marketed, to whom, on which channel, and why now.
3. **Draft loop** — turn campaign or artifact requests into draft artifacts with sources, audience, lane, channel, format, and honesty flags.
4. **Review and publish loop** — challenge proposals, approve or reject, release accepted artifacts, and record what actually shipped.
5. **Learning loop** — turn publish logs, coverage state, telemetry, and repeated production lessons into notebook entries, canon updates, skills, scenarios, or retirements.

The loops are sequential when a full campaign flows through the system, but members may also operate independently when their trigger fires. For example, the researcher can add evidence without a campaign; the publisher can update coverage without a new draft; the brand-manager can retire stale notebook debt without changing a campaign.

## Operating Graph

The first diagram is the compact operating-loop view. It is useful when checking role boundaries, approval gates, and whether evidence flows through planning, drafting, publishing, and learning.

```mermaid
flowchart LR
  OP[Operator / vision walk]
  BIH[Bookmark intelligence hub]
  WEB[Manual web and external sources]
  TEL[Future publish telemetry]

  OP --> RI[research-inbox]
  BIH --> RI
  WEB --> RI
  RI --> R[Researcher]

  R --> EVID[research evidence topics]
  EVID --> PLAN[planning decisions]
  EVID --> NOTE[marketing notebook]

  PLAN --> BM[Brand Manager]
  NOTE --> BM
  BM --> CANON[marketing and narrative canon]
  BM --> CAMP[campaign-launch-proposal]
  BM --> REQ[artifact-request queue]

  CAMP --> REQ
  CANON --> ADV[Advertiser / Draft Producer]
  EVID --> ADV
  REQ --> ADV
  ADV --> DRAFT[campaign-draft]
  ADV --> RUN[ad-run]
  DRAFT --> CPP[content-publish-proposal]

  CPP --> CONTRA[Marketing Contrarian]
  CAMP --> CONTRA
  PLAN --> CONTRA
  CONTRA --> CHAL[challenge reports]
  CHAL --> BM
  CHAL --> ADV
  CHAL --> R

  CPP --> APPROVAL[Operator approval]
  APPROVAL --> PUB[Publisher]
  DRAFT --> PUB
  CANON --> PUB
  PUB --> PLOG[publish-log]
  PUB --> COV[coverage-snapshot]
  PUB --> PSM[published subject mentions]
  PUB --> CHUP[channel-update]

  PLOG --> LEARN[learning inputs]
  COV --> LEARN
  PSM --> LEARN
  TEL --> PERF[publish-performance]
  PERF --> LEARN
  LEARN --> NOTE
```

The second diagram is the full topic-level view. It is the reference shape for validating whether topic producers, readers, decisions, and durable logs form a coherent marketing system.

<!-- prompt-manager-graph:
id: marketing-operating-model
scope: team
team: marketing-crew
mode: contract
actor_group.advertisers: member:oss-advertiser, member:subscription-advertiser
actor_group.marketing-members: team-members
actor_group.decision-owners: none
actor_alias.advertiser: group:advertisers
actor_alias.advertisers: group:advertisers
actor_alias.any marketing member: group:marketing-members
actor_alias.decision owner: group:decision-owners
actor_alias.decision owners: group:decision-owners
actor_alias.learning synthesis: process:learning-synthesis
actor_alias.meta-optimization: external:meta-optimization
actor_alias.director-swarm: external:director-swarm
actor_alias.future growth analyst: external:future-growth-analyst
-->
```mermaid
flowchart LR
  subgraph INFLOWS["Inflows / Producers"]
    %% @node OP external:operator
    OP([Operator])
    %% @node VW external:vision-walk
    VW([Vision walk])
    %% @node BIH external:bookmark-intelligence-hub
    BIH([Bookmark intelligence hub])
    %% @node TEL future:publish-telemetry
    TEL([Future telemetry])
  end

  %% Members
  %% @node R member:researcher
  R[Researcher]
  %% @node BM member:brand-manager
  BM[Brand Manager]
  %% @node OSS member:oss-advertiser
  OSS[OSS Advertiser]
  %% @node SUB member:subscription-advertiser
  SUB[Subscription Advertiser]
  %% @node PUB member:publisher
  PUB[Publisher]
  %% @node CONTRA member:marketing-contrarian
  CONTRA[Marketing Contrarian]

  %% Topics
  %% @node RI topic:research-inbox/*
  RI[(research-inbox/*)]
  %% @node AS topic:audience-scan/*
  AS[(audience-scan/*)]
  %% @node COMP topic:competitor-record/*
  COMP[(competitor-record/*)]
  %% @node HOOK topic:hook-record/*
  HOOK[(hook-record/*)]
  %% @node WF topic:workflow-scan/*
  WF[(workflow-scan/*)]
  %% @node SKILL topic:skill-scan/*
  SKILL[(skill-scan/*)]
  %% @node CHAN topic:channel-scan/*
  CHAN[(channel-scan/*)]
  %% @node FORMAT topic:format-scan/*
  FORMAT[(format-scan/*)]
  %% @node MB topic:monetization-benchmark-adjacent-record/*
  MB[(monetization-benchmark-adjacent-record/*)]
  %% @node REQOSS topic:artifact-request/oss/*
  REQOSS[(artifact-request/oss/*)]
  %% @node REQSUB topic:artifact-request/subscription/*
  REQSUB[(artifact-request/subscription/*)]
  %% @node DRAFT topic:campaign-draft/*
  DRAFT[(campaign-draft/*)]
  %% @node OSSRUN topic:oss-ad-run/*
  OSSRUN[(oss-ad-run/*)]
  %% @node SUBRUN topic:subscription-ad-run/*
  SUBRUN[(subscription-ad-run/*)]
  %% @node ARUN topic[future]:ad-run/<lane>/*
  ARUN[(ad-run/<lane>/*)]
  %% @node PLOG topic:publish-log/*
  PLOG[(publish-log/*)]
  %% @node COV topic:coverage-snapshot/*
  COV[(coverage-snapshot/*)]
  %% @node BRANDSNAP topic:brand-snapshot/*
  BRANDSNAP[(brand-snapshot/*)]
  %% @node PSM topic[future]:published-scenario-mentions/*
  PSM[(published-scenario-mentions/*)]
  %% @node PERF topic[future]:publish-performance/*
  PERF[(publish-performance/*)]
  %% @node NOTE topic:marketing/notebook/*
  NOTE[(marketing/notebook/*)]
  %% @node CHAL topic:challenge-report/*
  CHAL[(challenge-report/*)]
  %% @node RES topic:challenge-resolution-record/*
  RES[(challenge-resolution-record/*)]
  %% @node AGING topic:aging-scan-note/*
  AGING[(aging-scan-note/*)]
  %% @node DAPP topic:decision-application/<decision-id>
  DAPP[(decision-application/<decision-id>)]

  %% Decisions and durable canon
  %% @node AUD decision:audience-update
  AUD{audience-update}
  %% @node CHSTR decision:channel-strategy-update
  CHSTR{channel-strategy-update}
  %% @node POST decision:post-type-proposal
  POST{post-type-proposal}
  %% @node HCP decision:hook-candidate-promotion
  HCP{hook-candidate-promotion}
  %% @node CAMP decision:campaign-launch-proposal
  CAMP{campaign-launch-proposal}
  %% @node BGUIDE decision:brand-guideline-update
  BGUIDE{brand-guideline-update}
  %% @node NPROMO decision:notebook-promotion
  NPROMO{notebook-promotion}
  %% @node NRETIRE decision:notebook-retirement
  NRETIRE{notebook-retirement}
  %% @node CPP decision:content-publish-proposal
  CPP{content-publish-proposal}
  %% @node CHUP decision:channel-update
  CHUP{channel-update}
  %% @node COVGAP decision:coverage-gap
  COVGAP{coverage-gap}
  %% @node CAP decision:capability-gap
  CAP{capability-gap}
  %% @node DREJ decision:decision-rejection-proposed
  DREJ{decision-rejection-proposed}
  %% @node FRAME decision:framework-update
  FRAME{framework-update}
  %% @node CANON1 por:docs/marketing/STRATEGY.md
  CANON1[/docs/marketing/STRATEGY.md/]
  %% @node CANON2 por:docs/marketing/AUDIENCES.md
  CANON2[/docs/marketing/AUDIENCES.md/]
  %% @node LEARN process:learning-synthesis
  LEARN([Learning synthesis])
  %% @node ADPROC process:advertiser-draft-producer
  ADPROC([Advertiser draft producer])
  %% @node BACKLOG process:skill-scenario-action-backlog
  BACKLOG([Skill / scenario / action backlog])

  subgraph OUTFLOWS["Cross-team outflows"]
    %% @node MON team:monetization
    MON[[Monetization team]]
  end

  %% Intake
  OP --> RI
  VW --> RI
  BIH --> RI
  OP --> R
  VW --> R
  BIH --> R
  RI --> R

  %% Research
  R --> AS
  R --> COMP
  R --> HOOK
  R --> WF
  R --> SKILL
  R --> CHAN
  R --> FORMAT
  R --> MB
  MB --> MON
  AS --> R
  MB --> R
  CHAL --> R
  RES --> R
  CAP --> R
  R --> AUD
  R --> CHSTR
  R --> POST
  R --> HCP
  R --> CAP

  %% Brand management and canon
  OP --> BM
  VW --> BM
  NOTE --> BM
  AS --> BM
  COMP --> BM
  HOOK --> BM
  CHAN --> BM
  FORMAT --> BM
  PLOG --> BM
  COV --> BM
  BRANDSNAP --> BM
  CHAL --> BM
  RES --> BM
  CAP --> BM
  BM --> CANON1
  BM --> CANON2
  BM --> BRANDSNAP
  BM --> REQOSS
  BM --> REQSUB
  BM --> CAMP
  BM --> BGUIDE
  BM --> NPROMO
  BM --> NRETIRE

  %% Draft production
  OP --> OSS
  VW --> OSS
  REQOSS --> OSS
  AS --> OSS
  COMP --> OSS
  HOOK --> OSS
  WF --> OSS
  SKILL --> OSS
  CHAN --> OSS
  FORMAT --> OSS
  PLOG --> OSS
  COV --> OSS
  OSSRUN --> OSS
  CHAL --> OSS
  RES --> OSS
  CAP --> OSS
  OSS --> DRAFT
  OSS --> OSSRUN
  OSS --> CPP
  OSS --> COVGAP
  OSS --> CAP

  OP --> SUB
  VW --> SUB
  REQSUB --> SUB
  AS --> SUB
  COMP --> SUB
  HOOK --> SUB
  WF --> SUB
  SKILL --> SUB
  CHAN --> SUB
  FORMAT --> SUB
  PLOG --> SUB
  COV --> SUB
  SUBRUN --> SUB
  CHAL --> SUB
  RES --> SUB
  CAP --> SUB
  SUB --> DRAFT
  SUB --> SUBRUN
  SUB --> CPP
  SUB --> COVGAP
  SUB --> CAP
  ADPROC --> ARUN

  %% Review
  CHAL --> CONTRA
  RES --> CONTRA
  AGING --> CONTRA
  CPP --> CONTRA
  CAMP --> CONTRA
  BGUIDE --> CONTRA
  AUD --> CONTRA
  CHSTR --> CONTRA
  POST --> CONTRA
  HCP --> CONTRA
  CHUP --> CONTRA
  COVGAP --> CONTRA
  NPROMO --> CONTRA
  NRETIRE --> CONTRA
  CAP --> CONTRA
  CONTRA --> CHAL
  CONTRA --> RES
  CONTRA --> AGING
  CONTRA --> DREJ
  CONTRA --> FRAME

  %% Approval, publishing, and learning
  OP --> PUB
  REQOSS --> PUB
  REQSUB --> PUB
  DRAFT --> PUB
  COV --> PUB
  CHAL --> PUB
  RES --> PUB
  CAP --> PUB
  DAPP --> PUB
  PUB --> PLOG
  PUB --> COV
  PUB --> CPP
  PUB --> CHUP
  PUB --> COVGAP
  PUB --> CAP
  TEL --> PERF
  PLOG --> LEARN
  COV --> LEARN
  PSM --> LEARN
  PERF --> LEARN
  CHAL --> LEARN
  RES --> LEARN
  LEARN --> NOTE
  NPROMO --> BACKLOG
```

## Topic Catalog

These are the knowledge-topic families the target operating model uses. Current live declarations may lag this catalog; the implementation should be reconciled after this document is accepted.

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:research-inbox/*` | live | operator, vision-walk, bookmark-intelligence-hub | researcher | Raw unrouted research signal. The researcher drains this queue, then retags, deletes, or routes each item. |
| `topic:audience-scan/*` | live | researcher | brand-manager, advertisers, researcher | Audience pain, vocabulary, buyer triggers, objections, and persona evidence. |
| `topic:competitor-record/*` | live but under-consumed | researcher | brand-manager, advertisers | Competitor pricing, packaging, positioning, changelog, or claim evidence. Should feed positioning and campaign decisions. |
| `topic:hook-record/*` | live but under-consumed | researcher | advertisers, brand-manager | Reusable hook and framing observations. Should feed draft generation and hook-library promotion. |
| `topic:workflow-scan/*` | live | researcher | advertisers | External workflows, playbooks, agent setups, or business processes worth deconstructing. |
| `topic:skill-scan/*` | live | researcher | advertisers | External skills, prompts, reusable processes, or capability ideas. Raise `capability-gap` when blocked. |
| `topic:channel-scan/*` | live | researcher | brand-manager, advertisers | Evidence that a channel is worth activating, deprioritizing, or handling differently. |
| `topic:format-scan/*` | live | researcher | advertisers, brand-manager | Evidence that a post format or channel-native format is worth using or codifying. |
| `topic:monetization-benchmark-adjacent-record/*` | live | researcher | monetization team | Pricing, packaging, or market facts discovered by marketing but owned strategically by monetization. |
| `topic:artifact-request/oss/*` | live | brand-manager | oss-advertiser, publisher | Work queue for requested OSS-lane draft artifacts. It should carry campaign, audience, channel, format, source decision, and acceptance criteria. |
| `topic:artifact-request/subscription/*` | live | brand-manager | subscription-advertiser, publisher | Work queue for requested subscription-lane draft artifacts. It should carry campaign, SKU/bundle, audience, channel, format, source decision, and acceptance criteria. |
| `topic:campaign-draft/*` | live | advertisers | publisher | Draft artifacts ready to support a `content-publish-proposal`. Lane is metadata, not a separate pipeline. |
| `topic[future]:ad-run/<lane>/*` | target | advertisers | brand-manager, publisher, researcher | Normalized advertiser run summaries. This should eventually replace separate `oss-ad-run/*` and `subscription-ad-run/*` surfaces. |
| `topic:oss-ad-run/*` | live transitional | oss-advertiser | oss-advertiser | Current OSS-lane run log. Transitional until `topic[future]:ad-run/<lane>/*` exists. |
| `topic:subscription-ad-run/*` | live transitional | subscription-advertiser | subscription-advertiser | Current subscription-lane run log. Transitional until `topic[future]:ad-run/<lane>/*` exists. |
| `topic:publish-log/*` | live | publisher | advertisers, brand-manager | Record of what actually shipped: draft, channel, URL, post id, series, previous URL, and release notes. |
| `topic:coverage-snapshot/*` | live | publisher | advertisers, brand-manager, publisher | Current marketing coverage by SKU, lane, channel, or campaign. |
| `topic[future]:published-scenario-mentions/*` | target | publisher | advertisers, marketing-contrarian | Familiarity tracking for named scenarios, agents, and concepts, so drafts introduce subjects correctly for each audience. Current storage is JSONL. |
| `topic[future]:publish-performance/*` | target | publisher or future growth analyst | researcher, brand-manager, publisher, advertisers | Telemetry and qualitative performance: impressions, clicks, replies, saves, conversion, comments, channel-specific notes. |
| `topic:marketing/notebook/*` | live | learning synthesis | brand-manager | Working debt: repeated lessons, workarounds, craft observations, campaign lessons. Any member may append unresolved debt, but brand-manager is the only runtime drainer and must promote or retire entries. |
| `topic:brand-snapshot/*` | live | brand-manager | brand-manager | Snapshot of canon drift, notebook state, campaign signals, and promotion/retirement candidates. |
| `topic:challenge-report/*` | live | marketing-contrarian | decision owners | Append-only challenge evidence for pending marketing decisions. |
| `topic:challenge-resolution-record/*` | live | marketing-contrarian | decision owners | Latest challenge state: open, author-responded, resolved, escalated, overridden, or stale. |
| `topic:aging-scan-note/*` | live | marketing-contrarian | marketing-contrarian | Stale-decision hygiene notes. |
| `topic:decision-application/<decision-id>` | live system | decision workflow | publisher | Accepted-decision application markers used to avoid duplicate execution and track publish-decision follow-through. |

## Decisions

Decision contexts are the operator-reviewed gates that move work between loops. This section is the team's decision catalog; validation enforces graph/table parity, owner edges, expected evidence, and accepted downstream effects.

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `audience-update` | researcher | Change persona or audience canon after converging evidence. | Audience scans, competitor records, channel evidence, or repeated publish feedback. | Operator applies or rejects audience-canon changes. |
| `channel-strategy-update` | researcher | Activate, deprioritize, or strategically reposition a channel. | Channel scans, publish friction, platform drift, or audience-channel mismatch. | Channel strategy and downstream artifact requests change. |
| `post-type-proposal` | researcher | Add or materially change a post type. | Format scans, hook records, channel scans, or repeated draft/publish lessons. | Post-type canon changes after operator approval. |
| `hook-candidate-promotion` | researcher | Promote a stable hook into `path:docs/marketing/strategies/hook-library.md`. | Hook records with repeated applicability and source context. | Hook-library canon gains a reusable pattern. |
| `campaign-launch-proposal` | brand-manager | Create, change, or close a campaign. | Research evidence, coverage state, campaign lessons, or operator direction. | Campaign state changes and may create artifact requests. |
| `brand-guideline-update` | brand-manager | Change marketing, brand, strategy, research, rich-media, or narrative canon. | Brand snapshot, notebook debt, challenge reports, or repeated artifact issues. | Plan-of-record docs change through operator-curated edits. |
| `notebook-promotion` | brand-manager | Promote notebook debt into a skill, plan-of-record file, scenario, or config. | Notebook entries recurring across heartbeats or blocking production. | Debt leaves notebook and becomes skill, scenario, config, backlog, or plan-of-record canon. |
| `notebook-retirement` | brand-manager | Delete notebook debt that is obsolete or transient. | Notebook entry is duplicated, superseded, or no longer useful. | Debt is retired without becoming permanent canon. |
| `content-publish-proposal` | advertiser or publisher | Ask operator to approve a draft or release package. | Campaign draft, artifact request, source refs, honesty flags, channel/format fit. | Publisher may release or record manual release steps. |
| `channel-update` | publisher | Change per-platform rules based on publish friction or platform drift. | Publish-log friction, platform behavior changes, or repeated release workarounds. | Channel rules change after operator approval. |
| `coverage-gap` | advertiser or publisher | Surface missing or stale coverage for a SKU, lane, channel, or campaign. | Coverage snapshot, stale publish-log state, or campaign plan with no artifact. | Gap is accepted, deferred, or converted into artifact requests/backlog. |
| `capability-gap` | researcher, advertiser, publisher | Surface missing source access, tooling, scenario, skill, scheduler, media, telemetry, or account capability. | Work is blocked by missing capability rather than weak judgment. | Gap routes to director-swarm, meta-optimization, or a downstream backlog. |
| `decision-rejection-proposed` | marketing-contrarian | Recommend rejecting or revising a flawed pending decision. | Challenge report with concrete failure-mode evidence. | Operator rejects, revises, or overrides the challenged decision. |
| `framework-update` | marketing-contrarian | Add or revise review failure modes after repeated evidence. | Repeated challenge evidence not covered by current framework. | Review framework changes through operator-approved canon edits. |

Current validation enforces table presence, graph/table parity, owner edges, expected evidence, and accepted downstream effects.

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | `topic:research-inbox/*`, direct member context, or decision review | researcher, brand-manager, advertisers, publisher | Raw signal goes to research unless it is already a concrete decision review or artifact request. |
| Vision walk | `topic:research-inbox/*` and direct member context | researcher, brand-manager, advertisers | Signals become evidence, campaign proposals, artifact requests, or capability gaps. |
| Bookmark intelligence hub | `topic:research-inbox/*` | researcher | Researcher classifies into audience, competitor, hook, workflow, skill, channel, format, or benchmark-adjacent evidence. |
| Future telemetry | `topic[future]:publish-performance/*` | future growth analyst / publisher / researcher | Target-state only until scheduler, accounts, and measurement sources exist. |
| Monetization-adjacent market facts found by marketing | `topic:monetization-benchmark-adjacent-record/*` | monetization team | Researcher writes the record and routes strategic ownership to monetization. |
| Accepted publish decisions | `topic:decision-application/<decision-id>` | publisher | Publisher uses accepted-decision markers to avoid duplicate execution and track follow-through. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Research evidence | `audience-scan/*`, `competitor-record/*`, `hook-record/*`, `workflow-scan/*`, `skill-scan/*`, `channel-scan/*`, `format-scan/*` | brand-manager, advertisers, researcher | Durable evidence for planning and drafting. |
| Monetization-adjacent facts | `monetization-benchmark-adjacent-record/*` | monetization team | Route pricing, packaging, or market facts to the team that owns strategy. |
| Artifact requests | `artifact-request/oss/*`, `artifact-request/subscription/*` | advertisers and publisher | Queue draft production and release preparation. |
| Draft artifacts and run logs | `campaign-draft/*`, `oss-ad-run/*`, `subscription-ad-run/*` | publisher, advertisers | Support content-publish decisions and future draft improvement. |
| Publishing state | `publish-log/*`, `coverage-snapshot/*`, `published-scenario-mentions/*` | advertisers, brand-manager, publisher | Keep release history, coverage, and subject familiarity queryable. |
| Canon and campaign changes | `docs/marketing/*`, `docs/narrative/*`, campaign decisions | all marketing members and cross-team consumers | Keep public-facing voice and narrative coherent. |
| Challenge evidence | `challenge-report/*`, `challenge-resolution-record/*` | decision owners and operator | Keep weak marketing decisions from silently passing. |
| Capability and coverage gaps | `capability-gap`, `coverage-gap` decisions | director-swarm, meta-optimization, swarm-manager, or marketing members | Route blocked work to the right improvement path. |

## Feedback / Capability Improvement Loop

Marketing improves itself through four explicit exits:

1. **Notebook debt** — repeated lessons and workarounds enter `topic:marketing/notebook/*`; the brand-manager drains and promotes or retires them.
2. **Decision challenge** — marketing-contrarian writes `challenge-report/*` and `challenge-resolution-record/*` when a decision has concrete failure-mode evidence.
3. **Capability gaps** — members raise `capability-gap` when missing access, tooling, scheduler support, media generation, telemetry, account state, or scenario capability blocks the work.
4. **Coverage gaps** — advertisers and publisher raise `coverage-gap` when a SKU, lane, channel, or campaign lacks sufficient marketing coverage.

General code/scenario defects should use scenario-qa's `report-bug` flow. System-level friction that is not a defect should use meta-optimization's `report-friction` flow. Marketing should not turn every local frustration into marketing notebook debt when a universal observation flow is the correct destination.

## Roles

### Researcher

Owns evidence, not copy. The researcher drains `topic:research-inbox/*`, performs or requests analysis methods, writes durable observations, and raises planning decisions when evidence converges. The researcher should not draft publishable artifacts or edit marketing canon directly.

Primary responsibilities:

- classify and route raw research signal;
- maintain audience, competitor, hook, workflow, skill, channel, format, and benchmark-adjacent evidence;
- propose audience, channel, post-type, hook, and capability decisions;
- label weak evidence honestly;
- route monetization-adjacent facts to monetization.

### Brand Manager

Owns canon and campaign planning, not routine draft production. The brand-manager reads research evidence, publish outcomes, coverage state, and notebook debt; then proposes canon, campaign, promotion, or retirement decisions.

Primary responsibilities:

- steward `path:docs/marketing/` and `path:docs/narrative/` through accepted decisions;
- maintain the campaign model;
- drain `topic:marketing/notebook/*` and propose promotion or retirement;
- turn repeated lessons into permanent structure;
- prevent campaign sprawl when signal is weak.

### Advertiser / Draft Producer

The draft pipeline is shared. OSS and subscription are lanes, not separate mechanics. The current implementation has separate `oss-advertiser` and `subscription-advertiser` members; they should converge on the same workflow while retaining different lane expertise.

Primary responsibilities:

- consume artifact requests, active campaign slots, evidence, canon, channel rules, and coverage state;
- produce `topic:campaign-draft/*` entries with lane, audience, channel, format, source refs, claims, and honesty flags;
- raise `content-publish-proposal`, `coverage-gap`, or `capability-gap` decisions;
- record run summaries;
- avoid direct publishing and canon edits.

Lane examples:

- `oss` — builder-in-public, dev logs, OSS framework positioning, contributor acquisition;
- `subscription` — SKU, bundle, launch, demo, and buyer-facing scenario marketing;
- `community` — contributor onboarding, tutorials, forum/subreddit fit;
- `persona` — AI-UGC or persona-actor content when the disclosure and account model is active.

### Publisher

Owns release execution and publish state. The publisher does not invent core claims or approve its own unreviewed content. It polishes accepted drafts for platform fit, releases or records manual release steps, and updates the operational state that future drafts depend on.

Primary responsibilities:

- execute accepted `content-publish-proposal` decisions;
- maintain `topic:publish-log/*`, `topic:coverage-snapshot/*`, and subject-mention state;
- detect channel-rule drift and raise `channel-update`;
- detect missing scheduling, posting, account, or media capability and raise `capability-gap`;
- keep publish state queryable for later series, channels, and campaigns.

### Marketing Contrarian

Owns challenge and stale-decision hygiene. The contrarian reviews planning, canon, campaign, channel, and publish decisions against the marketing failure-mode framework. It does not generate positive marketing work.

Primary responsibilities:

- score pending marketing decisions;
- write challenge reports only for concrete failure-mode hits;
- maintain challenge-resolution state;
- propose rejection or framework updates when warranted;
- run stale-decision hygiene.

## Notebook Drainage

The notebook is not a destination of last resort. It is a temporary holding area for patterns that are real but not yet permanent.

Every notebook entry must eventually resolve to one of four outcomes:

1. **Promote to a skill** when it is executable procedure.
2. **Promote to plan-of-record** when it is strategic canon.
3. **Promote to a scenario or config change** when automation should replace the workaround.
4. **Retire** when the note was transient, duplicated, or superseded.

Append and read responsibilities are deliberately different:

- any marketing member may append unresolved debt when no typed topic, decision, skill, scenario, or plan-of-record surface is ready for it;
- brand-manager drains raw notebook debt and proposes promotion or retirement;
- researcher consumes promoted research and canon outputs, not raw notebook entries by default;
- advertisers consume artifact requests, canon, evidence, coverage, publish state, and skill updates, not raw notebook entries by default;
- publisher consumes accepted decisions, drafts, channel rules, coverage state, and promoted publishing guidance, not raw notebook entries by default;
- marketing-contrarian reviews notebook-derived promotion and retirement proposals when scoring whether the proposed permanent structure actually resolves the debt.

If a notebook family grows for multiple heartbeats without promotion, retirement, or a clear revisit marker, that is a system smell. The brand-manager should raise either `notebook-promotion`, `notebook-retirement`, or `capability-gap`.

## Current Implementation Gaps

The live marketing team is aligned with the first adoption pass, but it is not finished.

1. `topic[future]:ad-run/*` is still target-state. Current run logs remain split across `topic:oss-ad-run/*` and `topic:subscription-ad-run/*`.
2. `topic[future]:published-scenario-mentions/*` is still JSONL-backed state rather than a declared topic family.
3. Publish telemetry is not modeled yet. `topic[future]:publish-performance/*` should remain future until social accounts, scheduler integration, and measurement sources exist.
4. Account operations are not yet a topic model: account activation, warming, posting cadence, persona accounts, credentials, and scheduler state are deferred to `social-media-scheduler` and channel decisions.
5. `oss-advertiser` and `subscription-advertiser` still exist as separate members. That is acceptable while their lane expertise differs, but their mechanics should remain identical. Merge them only if the separate-member split starts creating duplicate work or coordination drift.
6. Hot operational state is split between `topic:*` knowledge topics and `*.jsonl` files. This is acceptable during migration, but each surface should declare whether it is authoritative knowledge, append-only operational log, or transition artifact.

## Adoption / Validation

Adopt this model in order:

1. Keep this document and `path:docs/marketing/README.md` as the target-state plan-of-record.
2. Keep `path:scenarios/prompt-manager/store/teams/marketing-crew/team.json` registered against this model.
3. Keep member `RESPONSIBILITIES.md` and `HEARTBEAT.md` files aligned with the five loops.
4. Keep member `topics.json` files aligned with the research outputs, lane-specific artifact requests, publish-log reads, and notebook reads.
5. Run `prompt-manager graph operating-model validate --team marketing-crew --id marketing-operating-model`.
6. Run `prompt-manager graph operating-model diff --team marketing-crew --id marketing-operating-model`.
7. Run `prompt-manager graph operating-model coverage --team marketing-crew --id marketing-operating-model`.
8. Rerun `prompt-manager graph topics --json` and decide which remaining warnings are accepted operator-only logs versus real gaps.

Do not make the validator quiet before the operating model is coherent. The topic graph is a check on the model, not a substitute for it.
