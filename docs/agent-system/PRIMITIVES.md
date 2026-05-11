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

The same agent template can be bound to multiple teams. The team-specific binding lives at `path:store/teams/<team>/members/<member>/`, where `<member>` is typically the same id as the agent. The team binding owns runtime state (heartbeat, last-handoff, topics declarations) — see `TEAM_MEMBER_ARCHITECTURE.md`.

Identity stays in `SOUL.md`; methodology should not. Capability extraction (the practice skill of the same name) audits `AGENTS.md` for embedded methodology that belongs in a reusable skill.

---

## Team

A coordinated group of agents (members) sharing a team-level contract, role configuration, and shared state. Teams hold:

- `team.json` and `roles.json` — machine-readable team config
- `shared/TEAM.md` — operating rules (write rules, coordination pattern, boundaries)
- `path[example]:shared/<various>` — per-team shared state (knowledge, decisions, audit logs, run lessons)
- `path[example]:members/<id>/` — per-member binding (heartbeat, responsibilities, last-handoff, topics declarations)

Coordination patterns (orthogonal to documentation architecture): independent, leader-led, or peer. Documentation architecture is split between plan-of-record truth and typed knowledge flow — see `TEAM_DOCS_PATTERNS.md`.

---

## Plan of Record (PoR)

Doctrine — durable, operator-curated documentation that defines what is true. Lives at `path:docs/<domain>/` (e.g., `path:docs/monetization/`, `path:docs/marketing/evidence/research/`, `path:docs/agent-system/`).

Properties:
- **Approval-gated:** operator-curated via approved decisions. Agents propose diffs; they never edit directly.
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

## Decision

A structured, contextful proposal-and-acceptance record. Decisions have a context (`audience-update`, `capability-gap`, `meta-self-improvement`, `action-candidate`, etc.), an owner who proposes, and an operator (or approving team) who accepts or rejects.

Decisions are the system's **commit log for plans, not for code.** When a member's behavior changes — a new audience definition, a deprecated skill, a new SKU — there is a decision recording the why.

Decision contexts a member owns are declared in its `topics.json` under `decisions_owned`; contexts whose acceptance changes a member's behavior are declared under `decisions_consumed`.

The full taxonomy of contexts, the lifecycle (proposed → accepted → executed → superseded → stale), the routing policy after acceptance (direct-write vs swarm-manager), and the cross-cutting rules (capability-gap criteria, action graduation gate, stale-decision policy, cross-team output ownership, inbox backpressure) live in `DECISIONS.md`. This file is concerned only with the bare definition.

---

## Knowledge entry

A team-scoped key/value record under a topic prefix. Used for both:

- **Inbox material** under `<inbox-name>/<signal-type>/<slug>` — debt, awaiting routing.
- **Permanent observations** under canonical surface prefixes (`audience-scan/<slug>`, `competitor-record/<slug>`, `monetization-benchmark-adjacent-record/<slug>`, etc.) — observations promoted from inbox to canon.

Knowledge entries are the fine-grained granularity for evidence and observations. They support concurrency-safe append, retention, and querying via `prompt-manager team knowledge-*` commands.

---

## Backlog item / capability-gap

Unbuilt work. A backlog item names something to build (typically tracked in `swarm-manager`); a `capability-gap` decision names a missing capability that blocks an existing member's work — typically a missing CLI command, action, scenario, or source-of-truth.

The distinction: backlog is "we plan to build X." Capability-gap is "we are blocked because X does not exist." `DECISIONS.md` covers the criteria for choosing between filing a capability-gap, a `meta-self-improvement` decision, or a backlog item, and how director-swarm consumes accepted capability-gaps.

---

## Inbox / Synthesis

The first stage where raw observations land before they have permanent structure. In the agent system, this is team knowledge entries under typed topic prefixes such as `<inbox-name>/<signal-type>/<slug>`, `friction-inbox/<scope>/<slug>`, or `run-lesson-report/<date>/<slug>`.

The draining member resolves each entry by retagging it to its destination prefix or deleting it as duplicate/weak. The procedure is universal (rendered into the heartbeat as a generated `# Inbox Flow` section); the per-domain signal vocabulary, dispatch table, evidence rules, and destination schemas live as a taxonomy JSON sidecar (e.g., `path:docs/marketing/taxonomies/marketing-research/taxonomy.json`); pure-judgment classification (when the topic-prefix isn't deterministic) lives as a portable classifier skill (e.g., `marketing-signal-classifier`). The "unrouted set" is the live inbox view; once routed, an entry no longer carries an inbox prefix. See `INTAKE_PIPELINE.md` for the full pattern.

---

## Scenario

A full application or microservice — API + CLI + UI + tests + CI — that combines resources and other scenarios to deliver a reusable business capability. Scenarios are products in their own right and the substrate the agent system runs on. See `CLAUDE.md` for the scenario lifecycle (start, stop, restart, test) and the broader Vrooli vision.

This file is concerned with scenarios only insofar as they host the agent system and are referenced as reference scenarios (`REFERENCE_SCENARIOS.md`).

---

## Three Pillars of Topic Validation

The primitives above describe *what* the system is made of. This section describes the system-architectural surface that keeps the topic substrate honest as the substrate evolves: three independent validation pillars, each catching a class of drift the others structurally cannot. The substrate is professionally aligned only when all three pass.

| Pillar | Source of truth | Catches | Anchor doc |
|---|---|---|---|
| **P1 — Declared graph** | `topics.json` per member (`intake[]`, `required_read[]`, `evidence_consumed[]`, `output[]`, `decisions_owned`, `decisions_consumed`, `external_producers`) | Cross-member declaration mismatches, dangling decision references, orphaned producers/consumers — anything derivable from a single load of the declared topology. | [`TOPICS_SCHEMA.md`](TOPICS_SCHEMA.md) |
| **P2 — Prose scan** | Markdown bodies in `members/`, `agents/`, writer-skill `SKILL.md` files, and `path:docs/<domain>/`. | Hardcoded topic-prefix references (`prompt-manager team knowledge-add ... --topic=...` patterns, backticked topic strings in instructions) that contradict the declarations. Drift between what an agent's prose tells it to do and what its `topics.json` says it does. | [`PROSE_SCAN_TARGETS.md`](PROSE_SCAN_TARGETS.md) |
| **P3 — Runtime attribution** | The `attribution` field on every post-cutoff `knowledge.jsonl` entry, scanned forward from each team's `attributionValidFrom`. | Observed writers/topics that no declaration accounts for. Agents writing to topics they don't declare; skills writing to topics not in `writes_to[]`; operator drift; legacy entries surfaced cleanly. | [`RUNTIME_ATTRIBUTION.md`](RUNTIME_ATTRIBUTION.md) |

Why three rather than one or two: declarations alone (P1) catch declaration mismatches but cannot detect a real-world write that nobody declared. Prose scans (P2) catch the human-language layer but cannot see runtime writes. Runtime attribution (P3) catches the observed-truth layer but cannot tell whether a declaration is internally consistent. Each pillar's blind spots are the others' load-bearing surface.

A new validation requirement should land in the existing pillar that fits its source-of-truth, not as a fourth pillar. The architecture is intentionally closed; widening it requires a `meta-optimization` decision and a workshop.

Cross-cutting validator rules implemented in `path:scenarios/prompt-manager/api/memberflow/`:

- **P1 rules** (errors in CI): `orphan_input`, `conflicting_drain`, `unknown_taxonomy`, `missing_taxonomy`, `dangling_por_sink`, `dangling_evidence_decision`, `unread_required`. Warnings: `orphan_output`, `wildcard_source_misuse`, `missing_destination_schema`, `topic_key_prefix_mismatch`, `stalled_drain`, `piling_inbox`.
- **P2 rules**: `prose_topic_leak`. Subpattern severity is split: `cli-knowledge-*` matches are errors (declarations have a place to land); `marked-topic-ref` and `inferred-backtick-topic-ref` stay warnings. Inferred unmarked matches are a permanent backstop because agents may omit markers and backticks are also used for file paths, code symbols, and other slashed identifiers. See `PROSE_SCAN_TARGETS.md` § Severity by subpattern.
- **P3 rules**: `actual_writer_undeclared` (error for the agent-member subcase, warning for the external-threshold subcase), `attribution_malformed` (error).

`prompt-manager graph topics` runs all three pillars together. CI captures a stable JSON artifact via `--findings-out=<path>` for diff-against-previous-run telemetry; without the flag the command is human-output-only (no surprise file writes for interactive use). The artifact's on-disk shape is versioned (`schema_version: 1`); see `path:scenarios/prompt-manager/cli/graph/findings_artifact.go` for the contract.
