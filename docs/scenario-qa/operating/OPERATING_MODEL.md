# Scenario QA Operating Model

**Status:** initial contract canon. This document defines how `scenario-qa` works as a coherent system: readiness review, structural audit, universal bug intake, contrarian review, downstream backlog routing, and learning loops.

The current document adopts the generic team operating-model shape from `path:docs/agent-system/OPERATING_GRAPHS.md`. It is intentionally paired with `path:docs/marketing/operating/OPERATING_MODEL.md` as a second, structurally different proving ground for operating-model validation.

## Mission

Scenario QA turns observed quality signals into evidence-rich findings, investigation reports, challenge records, and downstream backlog decisions. The team protects scenario quality without editing target scenarios directly.

Scenario QA does not own product prioritization, monetization strategy, marketing voice, infrastructure strategy, or direct implementation work. It investigates and routes; downstream execution happens through accepted decisions, swarm-manager backlog, or explicit capability gaps.

**Objective served.** `I1` — capability compounding: a scenario only becomes a permanent capability if it is actually sound, and this team is what makes "permanent" true rather than asserted (`path:docs/director-swarm/strategy/OBJECTIVES.md`).

**Outcome contribution.** Primary: **The Hive** (scenario ecosystem) — structural audits and readiness reviews set which scenarios are headliner-ready. Supporting: **The Forge**, by reducing the defect rework that slows goal throughput. The swarm-tier map of which team moves which outcome lives in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Team contribution map"; this paragraph is this team's own statement of it.

## Scope

Scenario QA owns:

- programmatic readiness-review evidence for existing scenarios;
- judgment-based structural audits using registered audit techniques;
- universal bug intake through `report-bug` and `bug-inbox/*`;
- root-cause investigation records for drained bugs;
- QA-specific challenge reports and challenge-resolution records;
- QA backlog decisions routed to downstream execution;
- capability-gap decisions when quality work is blocked by missing tools, access, or scenario support.

Scenario QA does not directly patch target scenario code. It may create evidence-rich backlog items or decisions that cause another team, swarm-manager execution, or a future capability to do the work.

## Operating Loops

Scenario QA has four loops:

1. **Audit loop** — apply judgment-based audit techniques, record `quality-audit/*`, and raise backlog decisions when the finding warrants execution.
2. **Bug loop** — drain universal `bug-inbox/*`, investigate root cause, write `bug-investigation-report/*`, and route cross-cutting fixes through decisions.
3. **Challenge loop** — review QA outputs and decisions, write `challenge-report/*` and `challenge-resolution-record/*` only when concrete failure modes are present.
4. **Learning loop** — turn repeated investigation/audit lessons into technique updates, capability gaps, or meta-optimization improvements.

The loops may run independently. An audit can raise no decision; the contrarian can stay quiet when outputs are clean.

> **Pre-emptive readiness is no longer a QA loop.** A former *readiness loop* swept idle scenarios with programmatic checks and filed fix items before feature work. That ordering moved into swarm-manager as a deterministic gate (`fix_before_feature`) plus optional policy-governed maintenance intake through the backlog auto-filer (`auto_filer`); regressions on scheduled scenarios are caught by execution finalization's before/after baseline diff.

## Operating Graph

This graph is the team-level contract. It shows how scenario-quality signal enters, which members drain or produce each topic, which decisions gate downstream work, and how challenge and capability-gap paths keep the system honest.

<!-- prompt-manager-graph:
id: scenario-qa-operating-model
scope: team
team: scenario-qa
mode: contract
actor_alias.report-bug: external:report-bug
actor_alias.operator: external:operator
actor_alias.any team: external:report-bug
actor_alias.decision owners: none
-->
```mermaid
flowchart LR
  %% @node BACKLOG process:swarm-manager-backlog-routing
  BACKLOG([Swarm Manager backlog routing])
  %% @node BI member:bug-investigator
  BI[Bug Investigator]
  %% @node BUGDEC decision:bug-resolution-proposal
  BUGDEC{bug-resolution-proposal}
  %% @node BUGREP topic:bug-investigation-report/<slug>
  BUGREP[(bug-investigation-report/<slug>)]
  %% @node CAP decision:capability-gap
  CAP{capability-gap}
  %% @node CHAL topic:challenge-report/<slug>
  CHAL[(challenge-report/<slug>)]
  %% @node DIR team:director-swarm
  DIR[[Director Swarm]]
  %% @node GAPROUTE process:capability-gap-routing
  GAPROUTE([Capability-gap routing])
  %% @node INFRA team:infra-health
  INFRA[[Infra Health]]
  %% @node META_IN team:meta-optimization
  META_IN[[Meta Optimization]]
  %% @node MKT team:marketing-crew
  MKT[[Marketing Crew]]
  %% @node MON team:monetization
  MON[[Monetization]]
  %% @node OP external:operator
  OP([Operator])
  %% @node QA member:quality-auditor
  QA[Quality Auditor]
  %% @node QBACK decision:quality-audit-backlog
  QBACK{quality-audit-backlog}
  %% @node QC member:qa-contrarian
  QC[QA Contrarian]
  %% @node QUAUD topic:quality-audit/<scenario-id>/<skill-id>
  QUAUD[(quality-audit/<scenario-id>/<skill-id>)]
  %% @node RB external:report-bug
  RB([report-bug skill])
  %% @node RES topic:challenge-resolution-record/<slug>
  RES[(challenge-resolution-record/<slug>)]
  %% @node SWARM external:swarm-manager
  SWARM([Swarm Manager])
  %% @node DOCSSCENARIO por:docs/scenario-qa/README.md
  DOCSSCENARIO[/docs/scenario-qa/README.md/]
  %% @node DOCSSCENARIO2 por:docs/scenario-qa/governance/adoption-validation.md
  DOCSSCENARIO2[/docs/scenario-qa/governance/adoption-validation.md/]
  %% @node DOCSSCENARIO3 por:docs/scenario-qa/governance/changelog.md
  DOCSSCENARIO3[/docs/scenario-qa/governance/changelog.md/]
  %% @node DOCSSCENARIO4 por:docs/scenario-qa/governance/editing.md
  DOCSSCENARIO4[/docs/scenario-qa/governance/editing.md/]
  %% @node DOCSSCENARIO5 por:docs/scenario-qa/operating/OPERATING_MODEL.md
  DOCSSCENARIO5[/docs/scenario-qa/operating/OPERATING_MODEL.md/]
  %% @node BUGINBOX topic:bug-inbox/*
  BUGINBOX[(bug-inbox/*)]
  %% @node BUGINVESTIGA topic:bug-investigation-report/*
  BUGINVESTIGA[(bug-investigation-report/*)]
  %% @node CHALLENGEREP topic:challenge-report/*
  CHALLENGEREP[(challenge-report/*)]
  %% @node CHALLENGERES topic:challenge-resolution-record/*
  CHALLENGERES[(challenge-resolution-record/*)]
  %% @node QUALITYAUDIT topic:quality-audit/*
  QUALITYAUDIT[(quality-audit/*)]

  BUGDEC --> QC
  BUGDEC --> BACKLOG
  CAP --> BI
  CAP --> QC
  CAP --> GAPROUTE
  QBACK --> QC
  QBACK --> BACKLOG
  OP --> QA
  RB --> BI
  RB --> BUGINBOX
  BI --> BUGDEC
  BI --> CAP
  BI --> BUGINVESTIGA
  QC --> CHALLENGEREP
  QC --> CHALLENGERES
  QA --> QBACK
  QA --> QUALITYAUDIT
  GAPROUTE --> META_IN
  BACKLOG --> SWARM
  DIR --> BUGINBOX
  INFRA --> BUGINBOX
  MKT --> BUGINBOX
  META_IN --> BUGINBOX
  MON --> BUGINBOX
  BUGINBOX --> BI
  BUGREP --> BI
  CHALLENGEREP --> BI
  CHALLENGEREP --> QA
  CHAL --> QC
  CHALLENGERES --> BI
  CHALLENGERES --> QA
  RES --> QC
  QUAUD --> QA
```

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:bug-inbox/*` | live |  | member:bug-investigator | Universal-source bug intake written by any team through the report-bug skill and drained by bug-investigator. |
| `topic:bug-investigation-report/*` | live | member:bug-investigator | member:bug-investigator | Closed bug-investigation audit log with root cause, evidence, action taken, and remaining gaps. |
| `topic:bug-investigation-report/<slug>` | live | member:bug-investigator | member:bug-investigator | Closed bug-investigation audit log with root cause, evidence, action taken, and remaining gaps. |
| `topic:challenge-report/*` | live | member:qa-contrarian | member:bug-investigator, member:qa-contrarian, member:quality-auditor | Append-only contrarian challenge evidence for scenario-qa findings, investigations, and backlog decisions. |
| `topic:challenge-report/<slug>` | live | member:qa-contrarian | member:bug-investigator, member:qa-contrarian, member:quality-auditor | Append-only contrarian challenge evidence for scenario-qa findings, investigations, and backlog decisions. |
| `topic:challenge-resolution-record/*` | live | member:qa-contrarian | member:bug-investigator, member:qa-contrarian, member:quality-auditor | Latest-state record for a scenario-qa challenge: open, author-responded, resolved, escalated, overridden, or stale. |
| `topic:challenge-resolution-record/<slug>` | live | member:qa-contrarian | member:bug-investigator, member:qa-contrarian, member:quality-auditor | Latest-state record for a scenario-qa challenge: open, author-responded, resolved, escalated, overridden, or stale. |
| `topic:quality-audit/*` | live | member:quality-auditor | member:quality-auditor | Judgment-based structural audit finding produced with a registered audit technique. |
| `topic:quality-audit/<scenario-id>/<skill-id>` | live | member:quality-auditor | member:quality-auditor | Judgment-based structural audit finding produced with a registered audit technique. |

## Decisions

Decision contexts gate downstream work or missing-capability escalation. This section is the team's decision catalog; validation enforces table presence, graph/table parity, owner edges, expected evidence, and accepted downstream effects.

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `quality-audit-backlog` | quality-auditor | Judgment-based structural audit findings converted into Swarm Manager execute backlog items. | `quality-audit/*` evidence from a registered audit technique and challenge records when present. | Downstream swarm-manager backlog or execution work is created. |
| `bug-resolution-proposal` | bug-investigator | Cross-cutting fixes that require operator approval. | `bug-inbox/*`, investigation report, root-cause evidence, and challenge records when present. | Fix/chore/backlog work is created or operator-approved policy changes are applied. |
| `capability-gap` | bug-investigator | Missing source access, tooling, scenario capability, investigation support, or QA infrastructure that blocks quality work. | Investigation or QA work is blocked by a missing reusable capability rather than by weak evidence. | Gap routes to meta-optimization, director-swarm, or downstream implementation work. |

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | direct member context for audit work | quality-auditor | Existing scenarios can be audited on operator request; planned scenarios without directories are out of scope. |
| Any team via `report-bug` | `topic:bug-inbox/<signal-type>/<slug>` | bug-investigator | Bug-investigator validates signal type, investigates root cause, records a report, and routes action. |
| Challenge evidence | `topic:challenge-report/<slug>` and `topic:challenge-resolution-record/<slug>` | decision owners and qa-contrarian | Decision owners read challenge evidence before proposing or defending backlog decisions. |
| Repeated investigation or audit technique gaps | `capability-gap` or meta-optimization decision | bug-investigator, quality-auditor, qa-contrarian | Missing methods, stale registries, or repeated ambiguity route to meta-optimization rather than becoming ad hoc prose. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Structural audit evidence | `quality-audit/*` | quality-auditor, swarm-manager via decisions | Evidence base for quality-audit backlog. |
| Bug investigation report | `bug-investigation-report/*` | bug-investigator, qa-contrarian, downstream fix owners | Durable root-cause record for a drained bug. |
| Backlog decisions | `quality-audit-backlog`, `bug-resolution-proposal` | swarm-manager and operator | Route findings into executable work without direct edits by scenario-qa. |
| Capability gaps | `capability-gap` decisions | meta-optimization, director-swarm, or implementation teams | Route missing quality infrastructure or investigation capability. |
| Challenge evidence | `challenge-report/*`, `challenge-resolution-record/*` | QA decision owners and operator | Prevent weak QA findings from becoming unchallenged work. |

## Feedback / Capability Improvement Loop

Scenario QA improves itself through four exits:

1. **Bug reporting** — any team reports broken behavior through `report-bug`; bug-investigator drains and closes each entry with a report.
2. **Contrarian review** — qa-contrarian writes `challenge-report/*` and `challenge-resolution-record/*` only when a registered failure mode applies.
3. **Technique promotion** — repeated investigation or audit lessons become `quality-audit-backlog`, `bug-resolution-proposal`, or meta-optimization decisions.
4. **Capability gaps** — missing source access, scenario affordances, tooling, or QA infrastructure become `capability-gap` decisions instead of hidden blockers.

System-level friction that is not broken behavior should use meta-optimization's `report-friction` flow. Scenario QA should not absorb general process friction into bug reports.

## Roles

> Pre-emptive readiness review is no longer a QA member role. The ordering ("fix before feature") is enforced programmatically by swarm-manager's `fix_before_feature` gate, and latent issues can be surfaced by the optional `auto_filer` maintenance intake loop consuming GCT directly, not a QA member's knowledge topics.

### Quality Auditor

Owns judgment-based structural audits. It uses registered audit techniques, writes `quality-audit/*`, and raises `quality-audit-backlog` for material findings.

### Bug Investigator

Owns universal bug intake. It drains `bug-inbox/*`, applies an investigation technique, writes `bug-investigation-report/*`, and raises `bug-resolution-proposal` or `capability-gap` when needed.

### QA Contrarian

Owns challenge discipline. It reads QA outputs and decisions, writes challenge reports and resolution records only for concrete failure-mode hits, and does not generate positive-action proposals.

## Current Implementation Gaps

1. `quality-audit/*` should remain a terminal evidence output in the graph. Its downstream execution path is through decisions, not direct topic consumption.
2. `capability-gap` is now declared in the scenario-qa team contract so the operating graph can model missing QA capability explicitly. Other teams using `capability-gap` should follow the same explicit-contract pattern.
3. The `methods/readiness/` registry is still a stub. Readiness-dimension techniques should graduate into paired docs and skills as GCT dimensions stabilize; pre-emptive ordering itself is handled by swarm-manager's fix-before-feature gate, not a QA member.
4. Future operator-fed `qa-inbox/*` and `audit-inbox/*` topics remain out of the contract until a producer exists.
5. `bug-resolution-proposal` and `quality-backlog-proposal` accepted effects should eventually resolve against concrete downstream backlog implementation contracts, not only prose surfaces.

## Adoption / Validation

Use this document as the second operating-model proving ground after marketing:

1. Keep this document and `path:docs/scenario-qa/README.md` as scenario-qa's team-level plan-of-record.
2. Keep `path:scenarios/prompt-manager/store/teams/scenario-qa/team.json` registered against this model.
3. Keep member `topics.json` files aligned with the operating graph relationships.
4. Run `prompt-manager graph operating-model validate --team scenario-qa --id scenario-qa-operating-model`.
5. Run `prompt-manager graph operating-model diff --team scenario-qa --id scenario-qa-operating-model`.
6. Run `prompt-manager graph operating-model coverage --team scenario-qa --id scenario-qa-operating-model`.

Do not make scenario-qa a marketing-shaped team. This model exists to prove the operating-model shape works for defect, audit, readiness, challenge, and downstream backlog flows.
