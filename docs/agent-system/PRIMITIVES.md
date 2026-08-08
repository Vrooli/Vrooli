# Primitives

The agent system is built from a small set of primitives. Every other concept (audits, lifecycles, intake patterns, validation rules) composes these. This file is canon — skills cite the definitions here rather than restating them.

Cites `LAYERS.md` for where each primitive lives in the file system, and `PROMOTION_LADDER.md` for how primitives evolve into one another.

---

## Skill

Reusable, principle-based guidance that creates shared mental models across sessions. Skills steer agent decisions without dictating exact steps.

**Good:**
- "Prefer boundary validation at system edges"
- "Render all states explicitly: loading, error, empty, success"
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
| Serve as the rendered prompt of a declared workflow node | Contract | Typed-result fidelity, low interpretation entropy |

Decision check:

```
Is the skill about how to change the scenario itself?     -> Steer
Is the skill about evolving shared platform/packages?     -> Platform
Is the skill about finding information or implementation? -> Search
Is the skill about using a tool or resource correctly?    -> Tools
Is the skill about HOW to approach a class of problems?   -> Practice
Is the skill about how skills should be written/governed? -> Meta
Is the skill rendered into a workflow run's prompt?       -> Contract
```

Contract skills are machine-invoked (referenced by `promptRef` from an Agent Manager workflow node) and follow their own structure — `SKILL_AUTHORING.md` §"Contract skills: machine-invoked workflow prompts".

Authoring quality bars live in `SKILL_AUTHORING.md`.

---

## Agent

A persistent identity with a `SOUL.md` (who it is), an `AGENTS.md` (its workflow contract), a `TOOLS.md` (its tool/skill bindings), and an `agent.json` (machine-readable metadata). Agents are the unit of action: every run is an agent invocation.

The same agent template can be bound to multiple teams. The team-specific binding lives at `path:store/teams/<team>/members/<member>/`, where `<member>` is typically the same id as the agent. The team binding owns runtime state (heartbeat, last-handoff, topics declarations) — see `TEAM_MEMBER_ARCHITECTURE.md`.

Identity stays in `SOUL.md`; methodology should not. Capability extraction (the practice skill of the same name) audits `AGENTS.md` for embedded methodology that belongs in a reusable skill.

---

## Team

A coordinated group of agents (members) sharing a team-level contract, role configuration, and shared state. Teams hold:

- `team.json` and `roles.json` — machine-readable team config
- `shared/TEAM.md` — operating rules (write rules, coordination pattern, boundaries)
- `path[example]:shared/<various>` — per-team shared state (Source Ledger guidance, task state, audit logs, run lessons)
- `path[example]:members/<id>/` — per-member binding (heartbeat, responsibilities, last-handoff, topics declarations)

Coordination patterns (orthogonal to documentation architecture): independent, leader-led, or peer. Documentation architecture is split between plan-of-record truth and typed knowledge flow — see `TEAM_DOCS_PATTERNS.md`.

### Team scope

The team's durable Source Ledger namespace. Members append observations and evidence
to this scope and use bounded recall at wake time. It is the shared corpus; mutable
task state remains in `tasks.json`.

---

## Plan of Record (PoR)

Doctrine — durable, operator-curated documentation that defines what is true. Lives at `path:docs/<domain>/` (e.g., `path:docs/monetization/`, `path:docs/marketing/evidence/research/`, `path:docs/agent-system/`).

Properties:
- **Approval-gated:** operator-curated via approved Swarm Manager work. Agents file evidence and scope; they never edit directly.
- **Cross-team-readable:** consumers from other teams or scenarios may cite it as required reading.
- **Durable:** entries persist and evolve. High churn signals something wrong with the plan, not with the rate of change.
- **One concept, one file:** no double residency.

The write boundary between plan-of-record canon and typed knowledge topics lives in `TEAM_DOCS_PATTERNS.md`.

---

## Action

A discoverable wrapper around exactly one Vrooli-controlled CLI command. Actions declare inputs, outputs, permissions, examples, validation, and `runEligible`. They are the discoverable execution layer — agents find them via `prompt-manager discover` rather than recalling prose.

An Action exists when the underlying CLI workflow is **deterministic** and benefits from discovery. See `PROMOTION_LADDER.md` for when prose graduates to an Action.

---

## CLI

The implementation layer: a Vrooli-controlled command that performs the actual work. Project CLI (`vrooli`), resource CLIs (`resource-postgres`), scenario CLIs (`prompt-manager`, `swarm-manager`), and so on.

CLIs own deterministic execution. Branching, shell conditionals, and multi-step workflows belong here, not in Action contracts.

---

---

## Knowledge entry

A team-scoped key/value record under a topic prefix. Used for both:

- **Inbox material** under `<inbox-name>/<signal-type>/<slug>` — debt, awaiting routing.
- **Permanent observations** under canonical surface prefixes (`audience-scan/<slug>`, `competitor-record/<slug>`, `monetization-benchmark-adjacent-record/<slug>`, etc.) — observations promoted from inbox to canon.

Knowledge entries are the fine-grained, topic-addressable representation of evidence
and observations. Prompt Manager's topic APIs are backed by the team's Source Ledger
scope; retention never deletes ledger entries.

---

## Objective

What the operator wants the system to be for. Objectives are the top of the goal hierarchy and the only operator-specific layer in the agent system: another operator adopting Vrooli would keep every team, skill, and canon file and replace only their objectives. The set lives in `path:docs/director-swarm/strategy/OBJECTIVES.md` and is authored by the operator, never by an agent.

Objectives divide into two classes. **Terminal** objectives state what the operator wants the world to be like. **Instrumental** objectives state what the system must become to serve them. An instrumental objective is only legitimate when a terminal one justifies it — a self-improving system whose written objectives are all about its own improvement has no reason to stop improving itself.

**Objective is not `Goal`.** They differ by scale, lifespan, and author, and conflating them is the mistake this entry exists to prevent:

| | Objective | Goal |
|---|---|---|
| Count | fewer than a dozen | many per objective |
| Lifespan | years; changes by `vision-update` decision | weeks to months |
| Author | operator only | agent-proposed, operator-approved |
| Shape | qualitative and durable | measurable and time-boxed |
| Home | director-swarm plan of record | `swarm-manager` runtime state |

The hierarchy is `Objective -> Goal -> Milestone -> Backlog item`. Every goal declares a parent objective; a goal without one is unattributed work, and an objective without goals is unstaffed intent. Two coverage directions are checkable and both are enforced — see OBJECTIVES.md §"The coverage rule".

---

## Backlog item

Unbuilt work. A backlog item names an outcome to build or a bounded correction to make,
with evidence, scope, dependencies, and a clear completion condition. File it through
Swarm Manager so the operator sees one work stream and the executing agent can read the
same context.

### Gated work item

A backlog item or capture that needs operator disposition before execution. Members
file it once through Swarm Manager, then read its disposition from the same work
stream; Prompt Manager does not maintain a parallel approval queue.

---

## Inbox / Synthesis

The first stage where raw observations land before they have permanent structure. In the agent system, this is team knowledge entries under typed topic prefixes such as `<inbox-name>/<signal-type>/<slug>`, `friction-inbox/<scope>/<slug>`, or `run-lesson-report/<date>/<slug>`.

The draining member resolves each entry by retagging it to its destination prefix or deleting it as duplicate/weak. The procedure is universal (rendered into the heartbeat as a generated `# Inbox Flow` section); the per-domain signal vocabulary, dispatch table, evidence rules, and destination schemas live as a taxonomy JSON sidecar (e.g., `path:docs/marketing/taxonomies/marketing-research/taxonomy.json`); pure-judgment classification (when the topic-prefix isn't deterministic) lives as a portable classifier skill (e.g., `signal-classifier`). The "unrouted set" is the live inbox view; once routed, an entry no longer carries an inbox prefix. See `INTAKE_PIPELINE.md` for the full pattern.

---

## Scenario

A full application or microservice — API + CLI + UI + tests + CI — that combines resources and other scenarios to deliver a reusable business capability. Scenarios are products in their own right and the substrate the agent system runs on. See `CLAUDE.md` for the scenario lifecycle (start, stop, restart, test) and the broader Vrooli vision.

This file is concerned with scenarios only insofar as they host the agent system and are referenced as reference scenarios (`REFERENCE_SCENARIOS.md`).

---

## Three Pillars of Topic Validation

The primitives above describe *what* the system is made of. This section describes the system-architectural surface that keeps the topic substrate honest as the substrate evolves: three independent validation pillars, each catching a class of drift the others structurally cannot. The substrate is professionally aligned only when all three pass.

| Pillar | Source of truth | Catches | Anchor doc |
|---|---|---|---|
| **P1 — Declared graph** | `topics.json` per member (`intake[]`, `required_read[]`, `evidence_consumed[]`, `output[]`, `external_producers`) | Cross-member declaration mismatches and orphaned producers/consumers — anything derivable from a single load of the declared topology. | [`TOPICS_SCHEMA.md`](TOPICS_SCHEMA.md) |
| **P2 — Prose scan** | Markdown bodies in `members/`, `agents/`, writer-skill `SKILL.md` files, and `path:docs/<domain>/`. | Hardcoded topic-prefix references (`prompt-manager team knowledge-add ... --topic=...` patterns, backticked topic strings in instructions) that contradict the declarations. Drift between what an agent's prose tells it to do and what its `topics.json` says it does. | [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) |
| **P3 — Runtime attribution** | Verified actor fields on events and source-ledger entries. | Observed writers/topics that no declaration accounts for. Agents writing outside their declared scope; operator drift; legacy entries surfaced cleanly. | [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md) |

Why three rather than one or two: declarations alone (P1) catch declaration mismatches but cannot detect a real-world write that nobody declared. Prose scans (P2) catch the human-language layer but cannot see runtime writes. Runtime attribution (P3) catches the observed-truth layer but cannot tell whether a declaration is internally consistent. Each pillar's blind spots are the others' load-bearing surface.

A new validation requirement should land in the existing pillar that fits its source-of-truth, not as a fourth pillar. The architecture is intentionally closed; widening it requires a `meta-optimization` decision and a workshop.

Cross-cutting validator rules are implemented in `path:scenarios/prompt-manager/api/memberflow/`. Each pillar's rule set and severities live with its anchor doc — P1 in [`TOPICS_SCHEMA.md`](TOPICS_SCHEMA.md) § Validation rules, P2 in [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) § Pattern set and § Severity guidance, P3 in [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md). This file indexes the pillars; it does not re-enumerate their rules.

`prompt-manager graph topics` runs all three pillars together. CI captures a stable JSON artifact via `--findings-out=<path>` for diff-against-previous-run telemetry; without the flag the command is human-output-only (no surprise file writes for interactive use). The artifact's on-disk shape is versioned (`schema_version: 1`); see `path:scenarios/prompt-manager/cli/graph/findings_artifact.go` for the contract.
