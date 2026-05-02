# Primitives

The agent system is built from a small set of primitives. Every other concept (audits, lifecycles, intake patterns, validation rules) composes these. This file is canon — skills cite the definitions here rather than restating them.

Cites `LAYERS.md` for where each primitive lives in the file system, and `PROMOTION_LADDER.md` for how primitives evolve into one another.

---

## Skill

Reusable, principle-based guidance that creates shared mental models across sessions. Skills steer agent decisions without dictating exact steps.

**Good:**
- "Prefer boundary validation at system edges"
- "Handle all states explicitly: loading, error, empty, success"
- "Generate falsifiable hypotheses before debugging"

**Avoid:**
- "Edit src/components/Button.tsx line 42"
- "Add exactly 5 tests per file"
- "Finish in 30 minutes"

Each skill picks one **category** (its `modes[0]` in metadata):

| Primary intent | Category | Optimizes |
|---|---|---|
| Build or improve scenario behavior | Steer | Architecture, quality, reliability |
| Build or improve shared platform/packages | Platform | Compatibility, standardization, cross-scenario reliability |
| Find or map information | Search | Discovery, coverage, evidence |
| Use a tool/resource/scenario | Tools | Correct operation, safety, efficiency |
| Apply a systematic engineering methodology | Practice | Process rigor, repeatability, knowledge capture |
| Define or govern the skill system | Meta | Skill coherence, policy, lifecycle |

Decision check:

```
Is the skill about how to change the scenario itself?     -> Steer
Is the skill about evolving shared platform/packages?     -> Platform
Is the skill about finding information or implementation? -> Search
Is the skill about using a tool or resource correctly?    -> Tools
Is the skill about HOW to approach a class of problems?   -> Practice
Is the skill about how skills should be written/governed? -> Meta
```

Authoring quality bars live in `SKILL_AUTHORING.md`.

---

## Agent

A persistent identity with a `SOUL.md` (who it is), an `AGENTS.md` (its workflow contract), a `TOOLS.md` (its tool/skill bindings), and an `agent.json` (machine-readable metadata). Agents are the unit of action: every run is an agent invocation.

The same agent template can be bound to multiple teams. The team-specific binding lives at `store/teams/<team>/members/<member>/`, where `<member>` is typically the same id as the agent. The team binding owns runtime state (heartbeat, last-handoff, topics declarations) — see `TEAM_MEMBER_ARCHITECTURE.md`.

Identity stays in `SOUL.md`; methodology should not. Capability extraction (the practice skill of the same name) audits `AGENTS.md` for embedded methodology that belongs in a reusable skill.

---

## Team

A coordinated group of agents (members) sharing a team-level contract, role configuration, and shared state. Teams hold:

- `team.json` and `roles.json` — machine-readable team config
- `shared/TEAM.md` — operating rules (write rules, coordination pattern, boundaries)
- `shared/<various>` — per-team shared state (knowledge, decisions, audit logs, run lessons)
- `members/<id>/` — per-member binding (heartbeat, responsibilities, last-handoff, topics declarations)

Coordination patterns (orthogonal to doc architecture): independent, leader-led, or peer. Doc architecture patterns (orthogonal to coordination): plan-of-record, working notebook, or both — see `TEAM_DOCS_PATTERNS.md`.

---

## Plan of Record (PoR)

Doctrine — durable, operator-curated documentation that defines what is true. Lives at `docs/<domain>/` (e.g., `docs/monetization/`, `docs/marketing/research/`, `docs/agent-system/`).

Properties:
- **Approval-gated:** operator-curated via approved decisions. Agents propose diffs; they never edit directly.
- **Cross-team-readable:** consumers from other teams or scenarios may cite it as required reading.
- **Durable:** entries persist and evolve. High churn signals something wrong with the plan, not with the rate of change.
- **One concept, one file:** no double residency.

Detailed comparison with the working-notebook pattern lives in `TEAM_DOCS_PATTERNS.md`.

---

## Action

A discoverable wrapper around exactly one Vrooli-controlled CLI command. Actions declare inputs, outputs, permissions, examples, validation, and `runEligible`. They are the discoverable execution layer — agents find them via `prompt-manager discover` rather than recalling prose.

An Action exists when the underlying CLI workflow is **deterministic** and benefits from discovery. See `PROMOTION_LADDER.md` for when prose graduates to an Action.

---

## CLI

The implementation layer: a Vrooli-controlled command that performs the actual work. Project CLI (`vrooli`), resource CLIs (`resource-postgres`), scenario CLIs (`prompt-manager`, `swarm-manager`), and so on.

CLIs own deterministic execution. Branching, shell conditionals, and multi-step workflows belong here, not in Action contracts.

---

## Decision

A structured, contextful proposal-and-acceptance record. Decisions have a context (`audience-update`, `capability-gap`, `meta-self-improvement`, `action-candidate`, etc.), an owner who proposes, and an operator (or approving team) who accepts or rejects.

Decisions are the system's **commit log for plans, not for code.** When a member's behavior changes — a new audience definition, a deprecated skill, a new SKU — there is a decision recording the why.

Decision contexts a member owns are declared in its `topics.json` under `decisions_owned`; contexts whose acceptance changes a member's behavior are declared under `decisions_consumed`.

---

## Knowledge entry

A team-scoped key/value record under a topic prefix. Used for both:

- **Inbox material** under `<inbox-name>/<signal-type>/<slug>` — debt, awaiting routing.
- **Permanent observations** under canonical surface prefixes (`audience-scan/<slug>`, `competitor/<slug>`, `monetization-benchmark-adjacent/<slug>`, etc.) — observations promoted from inbox to canon.

Knowledge entries are the fine-grained granularity for evidence and observations. They support concurrency-safe append, retention, and querying via `prompt-manager team knowledge-*` commands.

---

## Backlog item / capability-gap

Unbuilt work. A backlog item names something to build (typically tracked in `swarm-manager`); a `capability-gap` decision names a missing capability that blocks an existing member's work — typically a missing CLI command, action, scenario, or source-of-truth.

The distinction: backlog is "we plan to build X." Capability-gap is "we are blocked because X does not exist."

---

## Inbox / synthesis (transient)

The first stage where raw observations land before they have permanent structure. In the agent system, this is **not** a markdown notebook — it is team knowledge entries under the topic prefix `<inbox-name>/<signal-type>/<slug>`.

A router skill (e.g., `marketing-research-router`) drains the inbox by retagging each entry to its destination prefix or deleting it as duplicate/weak. The "unrouted set" is the live inbox view; once routed, an entry no longer carries an inbox prefix. See `INTAKE_PIPELINE.md` for the full pattern.

Markdown-file notebooks are a special case of synthesis — short-lived drafts living adjacent to canon (`docs/agent-system/drafts/`, for example) where structure is being workshopped before promotion. They are not a permanent layer; the historical "three-tier model" (hot buffer / notebook / permanent) is replaced by the inbox-router-drain model in the current architecture.

---

## Scenario

A full application or microservice — API + CLI + UI + tests + CI — that combines resources and other scenarios to deliver a reusable business capability. Scenarios are products in their own right and the substrate the agent system runs on. See `CLAUDE.md` for the scenario lifecycle (start, stop, restart, test) and the broader Vrooli vision.

This file is concerned with scenarios only insofar as they host the agent system and are referenced as reference scenarios (`REFERENCE_SCENARIOS.md`).
