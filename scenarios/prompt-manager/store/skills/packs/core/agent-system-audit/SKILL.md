## Practice focus: Agent System Audit

Audit the whole agent system — all six teams, their members, their plans of record, and the framework canon they run on — as one composed system. Answer two questions the operator cannot otherwise ask cheaply: *is each team internally coherent*, and *do the teams compose toward stated outcomes*.

Required reading:
- `docs/agent-system/FRAMEWORK_HEALTH.md` — the targets this audit scores against, each with its sensor, deadband, and actuator
- `docs/agent-system/OPERATING_GRAPHS.md` — the operating-model contract, relationship families, and validation-rule vocabulary
- `docs/director-swarm/evidence/OUTCOMES_CHARTER.md` §"Team contribution map" — which team is meant to move which outcome

Optional reading, when a finding narrows to one target:
- `prompt-manager skill read team-member-capability-architecture-audit` — one member's capability shape
- `prompt-manager skill read skill-improvement-suggestions` — one skill's efficiency and conditioning quality

---

### 1. When to use this skill

| Situation | Use this? | Instead use |
|---|---|---|
| "Is the agentic layer meeting our goals?" | Yes | — |
| Reviewing team/member/PoR setup as a whole | Yes | — |
| Preparing a vision walk that covers system shape | Yes | — |
| One member looks vague or blocked | No | `team-member-capability-architecture-audit` |
| One skill looks bloated or hand-rolled | No | `skill-improvement-suggestions` |
| One team's contract fails validation | No | Fix it directly; that is a `validate` finding, not an audit |

Session boundary: this is an audit. Read, measure, and report. Do not fix what you find — findings route through the actuators named in `FRAMEWORK_HEALTH.md`.

---

### 2. The two axes

Run the horizontal axis first. The vertical axis is well tooled and usually clean; the horizontal axis is where defects hide.

| Axis | Question | Primary sensor |
|---|---|---|
| Vertical (per team) | Does this team's declared contract match how it actually runs? | `prompt-manager graph operating-model validate` / `diff` |
| Horizontal (across teams) | Do the teams compose? Who feeds whom, what is unowned, what is doubly owned? | `coverage` cross-team rows, plus the contribution map |

---

### 3. Audit process

#### Phase 1 — Inventory

1. Run `prompt-manager graph operating-model list`.
2. Record each team's id, mode, and node/edge counts.
3. Flag any team whose mode is not `contract`. A non-`contract` team is exempt from every check below and must carry a dated rationale in its `## Current Implementation Gaps`.

#### Phase 2 — Vertical conformance

1. Run `prompt-manager graph operating-model validate`.
2. Run `prompt-manager graph operating-model diff`.
3. Record error and warning counts against the `FRAMEWORK_HEALTH.md` deadbands.
4. Attribute each finding to the owning team.

#### Phase 3 — Horizontal composition

1. Run `prompt-manager graph map --json`.
2. Use its composed team, topic, and cross-team edge set.
3. Compare that edge set against the prose claims in each team's `## Outputs / Downstream Consumers` table.

Report these four structural defects:

| Defect | Detection |
|---|---|
| Undeclared coupling | A team's outputs table names a peer team that no `cross_team_output` edge backs |
| Orphan output | A team produces a surface that no team or member reads, without an accepted disposition (`OPERATING_GRAPHS.md` §"No write-only surfaces") |
| Unowned flow | A topic that a team drains has no declared producer |
| Peer-as-external | A flow between two teams inside the swarm is modeled as an `external:` producer, hiding real intra-swarm coupling |

#### Phase 4 — Goal alignment

1. Read `OUTCOMES_CHARTER.md` §"Team contribution map".
2. Read each team's `## Mission` §"Outcome contribution" paragraph.
3. Report an **unowned category** when an outcome category has no primary contributor.
4. Report an **unattached team** when a team claims no outcome category.
5. Report a **drifted claim** when a team's own contribution paragraph disagrees with the aggregate map.

#### Phase 5 — Entropy and consolidation

1. Run `prompt-manager graph topics` and group findings by rule, not by entry.
2. Run `prompt-manager graph health --json`.
3. Run `bash scenarios/prompt-manager/test/agent_system_canon_test.sh`.
4. Look for skill families whose members differ only in a few lines — a consolidation candidate.
5. Classify each conditioning defect using the C1–C5 table in `docs/agent-system/SKILL_AUTHORING.md` §"Conditioning defect patterns". Cite the row id; do not restate the row.

#### Phase 6 — Report

Produce one findings list. Each finding names: the defect, the evidence command that produced it, the owning team, and the actuator from `FRAMEWORK_HEALTH.md`. Rank by whether the defect blocks a stated outcome, not by count.

---

### 4. Boundaries

- Do not edit team canon. Findings route to the owning team's decision context.
- Do not audit a single member or a single skill here; route to the narrower skill.
- Do not invent outcome targets. Unset targets are `pending-operator-input` and stay that way until the operator sets them.
