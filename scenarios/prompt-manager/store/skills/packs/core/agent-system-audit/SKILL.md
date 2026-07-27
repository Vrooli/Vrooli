## Practice focus: Agent System Audit

Audit the whole agent system — all six teams, their members, their plans of record, and the framework canon they run on — as one composed system. Answer three questions the operator cannot otherwise ask cheaply: *is each team internally coherent*, *do the teams compose*, and *does any of it serve what the operator actually wants*.

Required reading:
- `docs/agent-system/FRAMEWORK_HEALTH.md` — the targets this audit scores against, each with its sensor, deadband, and actuator, plus the audit-record rule
- `docs/agent-system/OPERATING_GRAPHS.md` — the operating-model contract, relationship families, and validation-rule vocabulary
- `docs/director-swarm/strategy/OBJECTIVES.md` — what the system is for; the objective ids, the two coverage directions, and evidence routing by objective class
- `docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Team contribution map" — which team is meant to move which outcome

Optional reading, when a finding narrows to one target:
- `prompt-manager skill read team-member-capability-architecture-audit` — one member's capability shape
- `prompt-manager skill read skill-improvement-suggestions` — one skill's efficiency and conditioning quality

---

### 1. When to use this skill

| Situation | Use this? | Instead use |
|---|---|---|
| "Is the agentic layer meeting our objectives?" | Yes | — |
| Reviewing team/member/PoR setup as a whole | Yes | — |
| Preparing a vision walk that covers system shape | Yes | — |
| One member looks vague or blocked | No | `team-member-capability-architecture-audit` |
| One skill looks bloated or hand-rolled | No | `skill-improvement-suggestions` |
| One team's contract fails validation | No | Fix it directly; that is a `validate` finding, not an audit |

Session boundary: this is an audit. Read, measure, and report. Do not fix what you find — findings route through the actuators named in `FRAMEWORK_HEALTH.md`.

---

### 2. The three axes

| Axis | Question | Primary sensor |
|---|---|---|
| Vertical (per team) | Does this team's declared contract match how it actually runs? | `prompt-manager graph operating-model validate` / `diff` |
| Horizontal (across teams) | Do the teams compose? Who feeds whom, what is unowned, what is doubly owned? | `prompt-manager graph map --depth member`, against each team's outputs table |
| Upward (out of the swarm) | Does the whole roster serve stated intent, and does stated intent have anyone serving it? | `OBJECTIVES.md` §"The coverage rule", against the contribution map |

Each axis is blind to the defects of the one above it. Vertical is well tooled and usually clean. Horizontal hides structural defects. **Upward hides whole missing programs** — a roster is always fully self-consistent with itself, so every conformance sensor can read green while nothing serves the operator at all.

The phases below run in data-dependency order, not in value order. Rank findings by objective impact when reporting (§Phase 6), not by the phase that produced them.

---

### 3. Audit process

#### Phase 1 — Sweep

1. Run `prompt-manager graph regenerate`. The graph index is cached; an un-regenerated index reports deleted nodes that no longer exist.
2. Run `prompt-manager graph audit`. One call reads every sensor in `FRAMEWORK_HEALTH.md` and reports each target as `in-band`, `out-of-band`, `external`, or `no-sensor`.
3. Run the two `external` sensors that are commands: `bash scenarios/prompt-manager/test/agent_system_canon_test.sh` and `prompt-manager experiment list`. The third `external` target, objective coverage, is a document read; Phase 4 performs it.
4. Treat every `out-of-band` target as a finding. Treat every `no-sensor` target as an open-loop gap, not as a pass.
5. Read the honesty flag on each `no-sensor` target. `pending-telemetry` means no instrument exists and one must be built. `pending-baseline` means the instrument exists and only the corpus sweep is missing — a much cheaper fix, and a different actuator.

#### Phase 2 — Vertical conformance

1. For each out-of-band contract target, run `prompt-manager graph operating-model diff` to get the per-team detail the sweep summarises.
2. Attribute each finding to the owning team.
3. Flag any team whose mode is not `contract`. A non-`contract` team is exempt from every check below and must carry a dated rationale in its `## Current Implementation Gaps`.

#### Phase 3 — Horizontal composition

1. Run `prompt-manager graph map --depth member`.
2. Use its composed team, member, topic, and cross-team edge set.
3. Compare that edge set against the prose claims in each team's `## Outputs / Downstream Consumers` table.

Report these four structural defects:

| Defect | Detection |
|---|---|
| Undeclared coupling | A team's outputs table names a peer team that no `cross_team_output` edge backs |
| Orphan output | A team produces a surface that no team or member reads, without an accepted disposition (`OPERATING_GRAPHS.md` §"No write-only surfaces") |
| Unowned flow | A topic that a team drains has no declared producer |
| Peer-as-external | A flow between two teams inside the swarm is modeled as an `external:` producer, hiding real intra-swarm coupling |

#### Phase 4 — Objective alignment

Run this phase in the direction of derivation: objectives first, then categories, then teams. Auditing upward from the team roster hides whole missing programs, because a roster is always fully self-consistent with itself.

1. Read `OBJECTIVES.md` — the objective table and §"The coverage rule".
2. Read `OUTCOMES_CHARTER.md` §"Team contribution map".
3. Read each team's `## Mission` §"Objective served" and §"Outcome contribution" paragraphs.

Report these five structural defects:

| Defect | Detection | Direction |
|---|---|---|
| Unserved objective | An objective that no team and no outcome category serves, and that carries no dated gap marker | downward |
| Unmeasurable objective | An objective whose evidence source does not exist; route it to the capability ladder | downward |
| Unattached team | A team whose declared objective appears in no objective row | upward |
| Unattached category | An outcome category that traces to no objective | upward |
| Drifted claim | A team's own `## Mission` paragraphs disagree with the aggregate map or with the objective table | either |

4. Treat a **declared** gap as reported, not as clean. An objective marked unserved with a dated marker is an open finding whose disposition is known; it stays in the findings list every cycle until it closes.

#### Phase 5 — Entropy and consolidation

1. Group the sweep's topic findings by rule, not by entry.
2. Run `prompt-manager graph health --type skill,agent,team --worst 20`. Omitting `--type` is still valid but ranks synthetic `cli:` nodes, which score 0 by construction and carry no finding.
3. Look for skill families whose members differ only in a few lines — a consolidation candidate.
4. Classify each conditioning defect using the C1–C5 table in `docs/agent-system/SKILL_AUTHORING.md` §"Conditioning defect patterns". Cite the row id; do not restate the row.

#### Phase 6 — Report and record

1. Produce one findings list. Each finding names: the defect, the evidence command that produced it, the owning team, and the actuator from `FRAMEWORK_HEALTH.md`. Rank by which objective the defect blocks, not by count and not by phase order. A defect that blocks an objective outranks any number of conformance findings that block none.
2. Write the readings and the findings list to the audit-record topic named in `docs/agent-system/FRAMEWORK_HEALTH.md` §"Audit record". Without the record the audit has no trend, and the next cycle restarts from zero.
3. Name the delta against the previous record: which targets moved, in which direction. A cycle that reports only current values has not answered whether the framework is improving.

---

### 4. Boundaries

- Do not edit team canon. Findings route to the owning team's decision context.
- Do not audit a single member or a single skill here; route to the narrower skill.
- Do not invent objectives or outcome targets. Both are operator-authored. Unset ones are `pending-operator-input` and stay that way until the operator sets them; report the hole, do not fill it.
