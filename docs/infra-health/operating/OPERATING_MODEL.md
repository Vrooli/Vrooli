# Infra Health Operating Model

**Status:** initial contract canon. This document defines how `infra-health` turns runtime signals, platform-code audits, and contrarian review into operator-routed reliability work.

**Loop status:** paused — recorded in the heartbeat control plane as `paused-manual` on 2026-07-24 (`prompt-manager team heartbeat-control infra-health pause`); the scheduler holds all member heartbeats until `… resume`. Current-state values and honesty flags across this PoR reflect the pause, not loop failure. On resume, the first heartbeat follows the resume protocol in `runtime-health-scanner/HEARTBEAT.md`.

The current document adopts the generic team operating-model shape from `path:docs/agent-system/OPERATING_GRAPHS.md`.

## Mission

Infra Health protects Vrooli's local platform by turning runtime signals, internal platform-code audits, instrumentation gaps, and cross-platform assumptions into durable findings the operator can act on.

**Objective served.** `I2` — coherence: the system stays reliable and stays reasonable-about as it grows (`path:docs/director-swarm/strategy/OBJECTIVES.md`). Instrumental, and therefore justified only by the terminal objectives it protects — reliability work that no terminal objective needs is polishing, which is a failure mode this team's contrarian already rules on.

**Outcome contribution.** Primary: **Mission Control** (system overview) — reliability targets and instrumentation close the sensor holes that make aggregate platform health readable at all. This team is second-order: its output is the capacity of the other teams rather than a revenue, growth, or scenario outcome of its own, so its first-order targets live in [`../strategy/RELIABILITY_TARGETS.md`](../strategy/RELIABILITY_TARGETS.md) rather than in a Command Center category. The swarm-tier map of which team moves which outcome lives in `path:docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Team contribution map"; this paragraph is this team's own statement of it.

## Scope

Infra Health owns:

- aggregate reliability signals from autoheal, system-monitor, lifecycle history, and durable incident records;
- internal Vrooli platform-code audit findings for CLI, setup, lifecycle, harness, infra scripts, and repo-contract surfaces;
- reliability targets and honesty-flagged current-state interpretation;
- instrumentation gaps that block measured findings;
- cross-platform debt for tier-2+ deployment planning;
- contrarian review of infra-health decisions and stale decision hygiene.

Infra Health does not own:

- per-scenario code quality, which belongs to scenario-qa;
- skill, agent, and prompt optimization, which belongs to meta-optimization;
- live incident response or privileged remediation, which belongs to system-monitor, autoheal, agent-manager, and the operator;
- external observability dashboards;
- direct platform code changes.

## Platform Under Control

Infra-health supervises a layered stack. Each layer absorbs what it can at its own timescale; infra-health is the outermost, slowest loop. The control layers *above* this map — how the supervised platform substrate sits under Vrooli's recursive self-improvement loop — are defined once in `path:docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` § Control topology; this map is the substrate that topology rests on.

| Layer | Timescale | Owns | Key surfaces |
|---|---|---|---|
| Commissioning — `vrooli setup`, host tools, host safeguards | once per host change | Provisioning: tool installs, opt-in host-state changes (kernel params, DNS, firewall), log access, permissions. The only layer where sudo exists. | `path:docs/configuration/host/tools.md`, `path:docs/configuration/host/safeguards.md`, `.vrooli/operator-state.json` |
| Capacity broker — `vrooli capacity` | ms–seconds (admission), minutes (sweep) | GPU/RAM/CPU claim arbitration: claims, priorities, degrade-before-preempt, idle unload, observed-usage tracking | `vrooli capacity list\|reconcile\|recommend\|policy`, system-monitor `/capacity` page |
| Autoheal — `vrooli-autoheal` | seconds–minutes | Restart/heal, durable incidents, operator-routed remediation artifacts | `vrooli-autoheal check history`, `actions uptime\|trends\|transitions`, `incidents latest` |
| System-monitor | minutes–hours | CPU/memory/GPU/disk metrics, threshold alerts, automated investigations, capacity UX | `system-monitor` CLI/API, investigations |
| Capability owners — search-hub, test-genie, prompt-manager, meta-optimization-manager, … | per-query / per-run | Capability-level degradation handling over self-declared members: scan → validate → aggregate loops. Each owner owns its capability contract (schema, scanner, validation ladder, derived availability aggregate) — never a member roster. | member `.vrooli/` declarations (e.g. `test-genie.json`, `search.json`), test-genie phase validation, owner status/coverage surfaces |
| Infra-health (this team) | heartbeats–days | Aggregate patterns, reliability targets, instrumentation gaps, cross-platform debt | this PoR |

Routing rules:

1. **Escalation rule.** A signal belongs to infra-health iff the inner layers repeatedly failed to absorb it. One OOM kill is the capacity broker's or autoheal's problem; the same scenario OOM-ing weekly despite holding claims is an infra-health finding.
2. **Sudo boundary.** Privileged host mutation exists only at commissioning time and in operator-run autoheal remediation artifacts. A finding whose fix needs sudo cannot be fixed by any runtime loop — route it as a proposed host tool, host safeguard, or setup change, or as an autoheal remediation artifact the operator runs.
3. **Supervise, don't operate.** Infra-health checks that the inner loops' coverage and honesty hold (e.g. every GPU resident holds a claim; heal success stays in band). It never actuates them directly — no policy-lever changes, no degrade/preempt, no restarts.
4. **Sampling rule.** The one-signal-per-heartbeat cadence must beat the recurrence rate of the pattern classes it is meant to catch. A pattern class that recurs faster than heartbeats can cover it is itself a finding (cadence or tooling gap), not a reason to skip signals.
5. **Sensor-integrity rule.** A reading is evidence only if its check passes the sensor-integrity rules in `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` (ghost, saturated, shelved, unit rule — ISA-18.2/EEMUA 191 discipline). Discriminate sensor fault from plant fault before routing plant-side work; a degraded alarm channel is the alarm-flood target's finding, never a reason to silently distrust all sensors.
6. **Contract-not-roster rule.** A capability owner owns the contract, not the roster: membership is by declaration in the member's own `.vrooli/` config, discovery is by scan, conformance is graded through test-genie's phase rail, and capability health is the owner's derived aggregate over currently-declared members. Infra-health supervises the *machinery* — the owner's scanner works, its sensor is honest, its aggregate stays in band — and consumes only derived aggregates. Infra-health never names a member: every set in this PoR is defined by a derivation query (ask SDA for the core-set closure, ask the owner for its load-bearing members), never by enumeration. Capability-level degradation escalates to infra-health only per the escalation rule (rule 1). Per-member performance deadbands live in the member's own declaration (e.g. `search.json` `performance` block), not in reliability targets. Capability-architecture improvements (search performance, embedding centralization, provider-less availability) are the owner's or meta-optimization's roadmap work — infra-health supplies the measured out-of-band aggregate that justifies them, nothing more.
7. **Substrate-degradation elevation.** Substrate degradation that impairs the improvement loop's own machinery — the agent runtime, prompt store, and sandboxes that meta-optimization's experiments run through — is priority-elevated over ordinary findings: a cascade failure here starves the loop that makes every future agent more capable. Meta-optimization is an authorized external raiser of `capability-gap` for this substrate (alongside director-swarm), because it observes the impairment from the loop side that infra-health's sensors do not reach.

## Operating Loops

Infra Health has three loops:

1. **Runtime health loop** - `runtime-health-scanner` selects one signal per heartbeat, checks repeat failures, heal loops, slow restarts, and investigation clusters, then raises bounded runtime or reliability-target decisions.
2. **Platform code audit loop** - `platform-code-auditor` audits one internal platform slice against architecture, security, coverage, documentation, cross-platform readiness, feedback surfaces, and instrumentation gaps.
3. **Contrarian hygiene loop** - `infra-contrarian` challenges pending decisions against the failure-mode rubric, runs stale decision review, and proposes framework-meta updates only when the team contract itself needs repair.

## Operating Graph

This graph is the team-level contract. It shows the runtime-declared relationships for infra-health: operator-seeded work, member-owned evidence topics, decision ownership, challenge review, and downstream routing.

<!-- prompt-manager-graph:
id: infra-health-operating-model
scope: team
team: infra-health
mode: contract
actor_alias.operator: external:operator
actor_alias.decision owners: none
-->
```mermaid
flowchart LR
  %% @node APPROVAL process:operator-approval
  APPROVAL([Operator approval])
  %% @node CANON process:operator-curated-canon
  CANON([Operator-curated canon])
  %% @node CG decision:capability-gap
  CG{capability-gap}
  %% @node CPD decision:cross-platform-debt
  CPD{cross-platform-debt}
  %% @node CR topic:challenge-report/<decision-id>
  CR[(challenge-report/<decision-id>)]
  %% @node CRR topic:challenge-resolution-record/<decision-id>
  CRR[(challenge-resolution-record/<decision-id>)]
  %% @node CS topic:contrarian-scan/YYYY-MM-DD
  CS[(contrarian-scan/YYYY-MM-DD)]
  %% @node DRP decision:decision-rejection-proposed
  DRP{decision-rejection-proposed}
  %% @node FM decision:framework-meta
  FM{framework-meta}
  %% @node IC member:infra-contrarian
  IC[Infra Contrarian]
  %% @node IG decision:instrumentation-gap
  IG{instrumentation-gap}
  %% @node OP external:operator
  OP([Operator])
  %% @node PCA member:platform-code-auditor
  PCA[Platform Code Auditor]
  %% @node PCAUD topic:platform-code-audit/YYYY-MM-DD
  PCAUD[(platform-code-audit/YYYY-MM-DD)]
  %% @node PCF decision:platform-code-finding
  PCF{platform-code-finding}
  %% @node RHA topic:runtime-health-audit/YYYY-MM-DD
  RHA[(runtime-health-audit/YYYY-MM-DD)]
  %% @node RHF decision:runtime-health-finding
  RHF{runtime-health-finding}
  %% @node RHS member:runtime-health-scanner
  RHS[Runtime Health Scanner]
  %% @node RTU decision:reliability-target-update
  RTU{reliability-target-update}
  %% @node SWARM external:swarm-manager-work
  SWARM([Swarm Manager work])
  %% @node DOCSINFRAHEA por:docs/infra-health/README.md
  DOCSINFRAHEA[/docs/infra-health/README.md/]
  %% @node DOCSINFRAHEA2 por:docs/infra-health/evidence/CROSS_PLATFORM_LEDGER.md
  DOCSINFRAHEA2[/docs/infra-health/evidence/CROSS_PLATFORM_LEDGER.md/]
  %% @node DOCSINFRAHEA3 por:docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md
  DOCSINFRAHEA3[/docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md/]
  %% @node DOCSINFRAHEA4 por:docs/infra-health/evidence/README.md
  DOCSINFRAHEA4[/docs/infra-health/evidence/README.md/]
  %% @node DOCSINFRAHEA5 por:docs/infra-health/governance/adoption-validation.md
  DOCSINFRAHEA5[/docs/infra-health/governance/adoption-validation.md/]
  %% @node DOCSINFRAHEA6 por:docs/infra-health/governance/changelog.md
  DOCSINFRAHEA6[/docs/infra-health/governance/changelog.md/]
  %% @node DOCSINFRAHEA7 por:docs/infra-health/governance/editing.md
  DOCSINFRAHEA7[/docs/infra-health/governance/editing.md/]
  %% @node DOCSINFRAHEA8 por:docs/infra-health/operating/OPERATING_MODEL.md
  DOCSINFRAHEA8[/docs/infra-health/operating/OPERATING_MODEL.md/]
  %% @node DOCSINFRAHEA9 por:docs/infra-health/operating/README.md
  DOCSINFRAHEA9[/docs/infra-health/operating/README.md/]
  %% @node DOCSINFRAHEA10 por:docs/infra-health/strategy/README.md
  DOCSINFRAHEA10[/docs/infra-health/strategy/README.md/]
  %% @node DOCSINFRAHEA11 por:docs/infra-health/strategy/RELIABILITY_TARGETS.md
  DOCSINFRAHEA11[/docs/infra-health/strategy/RELIABILITY_TARGETS.md/]
  %% @node CHALLENGEREP topic:challenge-report/*
  CHALLENGEREP[(challenge-report/*)]
  %% @node CHALLENGERES topic:challenge-resolution-record/*
  CHALLENGERES[(challenge-resolution-record/*)]
  %% @node CONTRARIANSC topic:contrarian-scan/*
  CONTRARIANSC[(contrarian-scan/*)]
  %% @node PLATFORMCODE topic:platform-code-audit/*
  PLATFORMCODE[(platform-code-audit/*)]
  %% @node RUNTIMEHEALT topic:runtime-health-audit/*
  RUNTIMEHEALT[(runtime-health-audit/*)]

  CG --> IC
  CG --> APPROVAL
  CPD --> IC
  CPD --> APPROVAL
  DRP --> APPROVAL
  FM --> APPROVAL
  IG --> IC
  IG --> APPROVAL
  PCF --> IC
  PCF --> APPROVAL
  RTU --> IC
  RTU --> APPROVAL
  RHF --> IC
  RHF --> APPROVAL
  OP --> PCA
  OP --> RHS
  IC --> DRP
  IC --> FM
  IC --> CHALLENGEREP
  IC --> CHALLENGERES
  IC --> CONTRARIANSC
  PCA --> CG
  PCA --> CPD
  PCA --> IG
  PCA --> PCF
  PCA --> PLATFORMCODE
  RHS --> CG
  RHS --> IG
  RHS --> RTU
  RHS --> RHF
  RHS --> RUNTIMEHEALT
  APPROVAL --> SWARM
  APPROVAL --> CANON
  CHALLENGEREP --> PCA
  CHALLENGEREP --> RHS
  CR --> IC
  CHALLENGERES --> PCA
  CHALLENGERES --> RHS
  CRR --> IC
  CS --> IC
  PCAUD --> PCA
  RHA --> RHS
```

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `topic:challenge-report/*` | live | member:infra-contrarian | member:infra-contrarian, member:platform-code-auditor, member:runtime-health-scanner | Append-only challenge against a pending infra-health decision. |
| `topic:challenge-report/<decision-id>` | live | member:infra-contrarian | member:infra-contrarian, member:platform-code-auditor, member:runtime-health-scanner | Append-only challenge against a pending infra-health decision. |
| `topic:challenge-resolution-record/*` | live | member:infra-contrarian | member:infra-contrarian, member:platform-code-auditor, member:runtime-health-scanner | Latest-state record for an infra-health challenge. |
| `topic:challenge-resolution-record/<decision-id>` | live | member:infra-contrarian | member:infra-contrarian, member:platform-code-auditor, member:runtime-health-scanner | Latest-state record for an infra-health challenge. |
| `topic:contrarian-scan/*` | live | member:infra-contrarian | member:infra-contrarian | Stale decision and failure-mode review snapshot. |
| `topic:contrarian-scan/YYYY-MM-DD` | live | member:infra-contrarian | member:infra-contrarian | Stale decision and failure-mode review snapshot. |
| `topic:platform-code-audit/*` | live | member:platform-code-auditor | member:platform-code-auditor | Snapshot one internal platform-code audit slice. |
| `topic:platform-code-audit/YYYY-MM-DD` | live | member:platform-code-auditor | member:platform-code-auditor | Snapshot one internal platform-code audit slice. |
| `topic:runtime-health-audit/*` | live | member:runtime-health-scanner | member:runtime-health-scanner | Snapshot one runtime-health signal or quiet-day review. |
| `topic:runtime-health-audit/YYYY-MM-DD` | live | member:runtime-health-scanner | member:runtime-health-scanner | Snapshot one runtime-health signal or quiet-day review. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `runtime-health-finding` | runtime-health-scanner | Route an aggregate runtime pattern to operator review. | Repeat failure, heal loop, slow restart trend, investigation cluster, or quiet-day finding. | Downstream swarm-manager work is created, reliability interpretation is updated, or no-action evidence is recorded. |
| `platform-code-finding` | platform-code-auditor | Route an internal platform-code quality or reliability finding. | One audited slice shows a material architecture, security, coverage, documentation, feedback, or reliability issue. | Downstream Swarm Manager fix, chore, or execute backlog work is created or the finding is rejected with evidence. |
| `cross-platform-debt` | platform-code-auditor | Record a platform assumption that matters for tier-2+ deployment. | Source inspection or measured target-platform run shows a Linux-only or host-specific assumption. | `path:docs/infra-health/evidence/CROSS_PLATFORM_LEDGER.md` is updated or tier-activated work is created. |
| `instrumentation-gap` | runtime-health-scanner, platform-code-auditor | Name a missing stat or signal that blocked a finding. | A finding cannot become measured because no structured event, CLI, report, or history exists. | `path:docs/infra-health/evidence/INSTRUMENTATION_ROADMAP.md` is updated or downstream capability work is routed. |
| `capability-gap` | runtime-health-scanner, platform-code-auditor | Request a missing CLI verb, scenario, or tool capability. | A team member or director-swarm cannot perform required infra-health work with current tools. | Gap routes to director-swarm, Swarm Manager, or downstream implementation work. |
| `reliability-target-update` | runtime-health-scanner | Adjust a reliability target based on baseline evidence. | 30+ days of measured data show the target should tighten or loosen for non-temporary reasons. | `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` is updated. |
| `decision-rejection-proposed` | infra-contrarian | Recommend rejecting or revising a pending infra-health decision. | Failure-mode review finds alarm noise, polishing, premature cross-platform work, instrumentation sprawl, target drift, scope creep, or measurement gaps. | Decision is rejected, revised, superseded, or retained with explicit evidence. |
| `framework-meta` | infra-contrarian | Update infra-health rubric, triage ladder, or decision discipline. | Repeated review shows the operating contract itself is wrong or underspecified. | Operating model, governance docs, team config, or future manifest entries are updated. |

### Required decision content

Every `runtime-health-finding`, `instrumentation-gap`, `reliability-target-update`, and `cross-platform-debt` decision must carry, so the operator reviews reasoning and not just a conclusion:

1. **Evidence.** The sensor reading it rests on — command, value, and window (e.g. `vrooli-autoheal actions trends --json` → 96.1% over 30d). The reading must pass the sensor-integrity rules (ghost / saturated / shelved / unit — `path:docs/infra-health/strategy/RELIABILITY_TARGETS.md` § Sensor integrity); cite them, do not restate.
2. **Root-cause reasoning.** Why the signal means what the finding claims — the causal step from reading to conclusion, and how sensor fault was ruled out.
3. **Alternatives / why-not.** The options considered, including "do nothing," and why each was set aside.
4. **Expected in-band return.** What value should return to band, by when, and which sensor re-reads it — the actuation-efficacy contract (update protocol rule 5): the sensor grades the fix, not the author.

The decision *text* additionally satisfies the operator-legibility contract (`path:docs/agent-system/DECISIONS.md` § Operator legibility): plain summary first, terms defined at first use, rationale in normal prose.

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|
| Operator | direct member context for runtime or platform-code work | runtime-health-scanner, platform-code-auditor | Operator direction can seed work, but accepted effects still require decision flow. |
| Runtime health evidence | `topic:runtime-health-audit/YYYY-MM-DD` or `runtime-health-finding` | runtime-health-scanner | Repeat failures, heal loops, slow restarts, and investigation clusters route to runtime findings or instrumentation gaps. |
| Platform-code evidence | `topic:platform-code-audit/YYYY-MM-DD` or `platform-code-finding` | platform-code-auditor | One internal platform slice is audited per heartbeat; no direct code edits. |
| Challenge evidence | `topic:challenge-report/<decision-id>` and `topic:challenge-resolution-record/<decision-id>` | decision owners and infra-contrarian | Decision owners read challenge evidence before defending or revising infra-health decisions. |

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|
| Runtime findings | `runtime-health-finding`, `runtime-health-audit/*` | runtime-health-scanner, infra-contrarian, Swarm Manager via decisions | Turn aggregate runtime patterns into routed work or no-action evidence. |
| Platform-code findings | `platform-code-finding`, `platform-code-audit/*` | platform-code-auditor, infra-contrarian, Swarm Manager via decisions | Route internal platform quality issues without direct code mutation. |
| Instrumentation gaps | `instrumentation-gap` decisions | operator, director-swarm, Swarm Manager, owning scenarios | Convert missing stats into implementation work. |
| Capability gaps | `capability-gap` decisions | operator, director-swarm, Swarm Manager, owning scenarios | Convert missing tools or scenario capabilities into implementation work. |
| Portability ledger updates | `cross-platform-debt`, `path:docs/infra-health/evidence/CROSS_PLATFORM_LEDGER.md` | platform-code-auditor, deployment planning | Keep future tier risk visible without prematurely blocking current work. |
| Challenge evidence | `challenge-report/*`, `challenge-resolution-record/*` | infra-health decision owners and operator | Prevent weak infra-health findings from becoming unchallenged work. |

## Feedback / Capability Improvement Loop

1. `member:runtime-health-scanner` searches for one material runtime signal and raises a finding only when the signal has evidence and a bounded consequence.
2. `member:platform-code-auditor` audits one internal platform slice and raises a finding only when the affected surface and owner are concrete.
3. `member:infra-contrarian` reviews pending decisions against failure modes and records challenges or quiet scans.
4. Accepted `runtime-health-finding`, `platform-code-finding`, `instrumentation-gap`, `capability-gap`, `cross-platform-debt`, or `reliability-target-update` decisions update Swarm Manager work, capability roadmaps, reliability targets, cross-platform ledger entries, or operating canon.
5. When a capability ships, the follow-up `instrumentation-gap` or `reliability-target-update` decision updates the affected PoR evidence, active prompts, and migration markers together.

## Current Implementation Gaps

- `path:docs/infra-health/manifest.json` is present, but target-state is for prompt-manager PoR manifest validation to run automatically against this folder.
- Many reliability values remain `pending-telemetry` until the instrumentation roadmap closes enough gaps to produce measured baselines.
- `path:docs/infra-health/operating/OPERATING_MODEL.md` is newly documented, and target-state is clean validate, diff, and coverage output once graph tooling consumes this team id.

## Adoption / Validation

- `prompt-manager graph operating-model validate --team infra-health --id infra-health-operating-model`
- `prompt-manager graph operating-model diff --team infra-health --id infra-health-operating-model`
- `prompt-manager graph operating-model coverage --team infra-health --id infra-health-operating-model`
