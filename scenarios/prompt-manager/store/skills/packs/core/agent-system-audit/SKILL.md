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
- `prompt-manager skill read team-capability-consolidation` — one team whose structure Phase 5 flags

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
| One team produces little and hand-maintains its own state | No | `team-capability-consolidation` |
| One team's structure is hard to hold in your head, whatever it ships | No | `team-capability-consolidation` |

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
2. Run the sweep **against the previous cycle**, and persist this one:

   ```bash
   prompt-manager graph audit --baseline <previous-cycle.json> --out <this-cycle.json>
   ```

   One call reads every sensor in `FRAMEWORK_HEALTH.md` and reports each target as `in-band`, `out-of-band`, `external`, or `no-sensor`. Retrieve the previous artifact from the `framework-health-audit/<date>` record; if none exists, run without `--baseline` and say so in the record — this cycle then becomes the baseline. **Do not skip `--out`.** Trend targets cannot band without a prior artifact, and skipping the write is what kept them at `pending-baseline` indefinitely.
3. Canon coherence and skill-experiment liveness are collected by the sweep now — do not run them by hand. Anything still reported `external` genuinely was not collected (usually no repository root in reach), and the sweep prints the command to run for each under **Next Steps**.
4. Treat every `out-of-band` target as a finding. Treat every `no-sensor` target as an open-loop gap, not as a pass. Read the tally line: it names out-of-band, unsensored, and not-collected separately, and a `not collected` count above zero means the sweep is blind, not clean.
5. Read the `honesty_flag` field on each `no-sensor` target — it is typed, so do not parse it back out of the observed prose. `pending-telemetry` means no instrument exists and one must be built. `pending-baseline` means the instrument exists and only the corpus sweep or the prior reading is missing — a much cheaper fix, and a different actuator. Each also carries `gap_open_days`; rank the oldest first, since every marker reads as equally fresh without it.

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

Most of this phase is now a sensor. Run it first, then spend the reading on what the sensor cannot see.

1. Run `prompt-manager graph objectives`. It reports the team half of both coverage directions: unserved objectives, undeclared holes, unattached teams, unknown ids, and one-sided links.
2. Read `OUTCOMES_CHARTER.md` §"Team contribution map" and score every outcome category against an objective. This half stays a document read — categories are Command Center dashboard ids, not a store surface.
3. Read each team's `## Mission` §"Objective served" and §"Outcome contribution" paragraphs for claims that disagree with the sensor.

| Defect | Detection | Direction |
|---|---|---|
| Unserved objective | `graph objectives` reports it unserved | downward |
| Unmeasurable objective | `graph objectives` raises `objective_unmeasurable`; route it to the capability ladder | downward |
| Unattached team | `graph objectives` raises `objective_team_unattached` | upward |
| Unattached category | An outcome category in the charter that traces to no objective — **read, not sensed** | upward |
| Drifted claim | A team's `## Mission` paragraphs disagree with the objective table or the aggregate map — **read, not sensed**; the sensor compares id sets only, so a role or emphasis stated in prose is yours to check | either |

4. Treat a **declared** gap as reported, not as clean. An objective marked unserved with a dated marker is an open finding whose disposition is known; it stays in the findings list every cycle until it closes.
5. Do not close a finding this phase produced. The actuator is `outcome-direction` or `capability-work` in `director-swarm`, which owns the objective set. Measuring the join and restructuring on the strength of your own measurement are different authorities.

#### Phase 5 — Entropy and consolidation

1. Group the sweep's topic findings by rule, not by entry.
2. Run `prompt-manager graph health --type skill,agent,team --worst 20`. Omitting `--type` is still valid but ranks synthetic `cli:` nodes, which score 0 by construction and carry no finding.
3. Look for skill families whose members differ only in a few lines — a consolidation candidate.
4. Classify each conditioning defect using the C1–C5 table in `docs/agent-system/SKILL_AUTHORING.md` §"Conditioning defect patterns". Cite the row id; do not restate the row.
5. Record any team that meets one of three conditions: its shipped output is near zero against a large roster and canon; it hand-maintains records with a lifecycle; or its orientation cost rose in a cycle where its scenario coverage grew (`FRAMEWORK_HEALTH.md` § Team orientation cost). Route it to `team-capability-consolidation`; the missing capability, not the roster, is usually the defect.
6. Read the third condition against the previous audit record, not against a single reading. The sweep reports every team's composite and its components (`prompt-manager graph orientation-cost` for the standalone read), but orientation cost is banded as a trend, so one cycle's numbers cannot say whether it moved. A first reading establishes the baseline and raises no finding.
7. When the composite rose, name the component that rose with it — members, canon lines, topics, or work types. "Orientation cost is up" routes nowhere; "the roster grew while `content-desk` absorbed the drafting loop" names the consolidation target.

#### Phase 6 — Report and record

1. Produce one findings list. Each finding names: the defect, the evidence command that produced it, the owning team, and the actuator from `FRAMEWORK_HEALTH.md`. Rank by which objective the defect blocks, not by count and not by phase order. A defect that blocks an objective outranks any number of conformance findings that block none.
2. Write the readings and the findings list to the audit-record topic named in `docs/agent-system/FRAMEWORK_HEALTH.md` §"Audit record". Without the record the audit has no trend, and the next cycle restarts from zero.
3. Name the delta against the previous record: which targets moved, in which direction. A cycle that reports only current values has not answered whether the framework is improving.

---

### 4. Boundaries

- Do not edit team canon. Findings route to the owning team's work type.
- Do not audit a single member or a single skill here; route to the narrower skill.
- Do not invent objectives or outcome targets. Both are operator-authored. Unset ones are `pending-operator-input` and stay that way until the operator sets them; report the hole, do not fill it.
