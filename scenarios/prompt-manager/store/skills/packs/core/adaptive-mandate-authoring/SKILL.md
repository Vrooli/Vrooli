---
name: adaptive-mandate-authoring
description: Author a Plan Manager plan with shape `mandate` for a scenario that has documented targets, an improve skill and sensors. A mandate carries a target pointer, sensors, bands, scope and authority, stop rules, a suggested arc and a journal location. No step lists; progress is the setpoint board. Not for phased plans; use implementation-plan-authoring for those.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [mandate, plan, setpoint, scenario, development, authoring, swarm]
  icon: compass
  status: active
  revision: 1
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-09-11T00:00:00Z"
  requires:
    scenarios: [prompt-manager, plan-manager, program-runtime]
    commands: [prompt-manager skill read, plan-manager author, program-runtime library run]
  origin:
    kind: authored
---

## Practice focus: Adaptive Mandate Authoring

Write a plan whose body points at the scenario's documented target and names the
sensors that measure it, so that any execution mode can pursue the target
without a step list. The setpoint board is the progress report; the docs are
the design; the mandate is the authority and the finish line.

Required reading:
- `path:docs/agent-system/SWARM_MANAGER_WORK.md` §"Work shapes" — when to
  choose `mandate` over `phased`. This skill does not restate that rule.
- `prompt-manager skill read harness-goal-authoring` §"Shapes" B — the finish
  line a mandate must make writable.
- `prompt-manager skill read scenario-improvement-campaign` — the loop that
  executes a mandate. The mandate supplies authority and bands; the campaign
  supplies the method.
- `path:scenarios/plan-manager/docs/concepts/PLAN-MODEL.md` — the plan fields
  that exist today.

### 1. Scope

In scope: the content of a Plan Manager plan with shape `mandate`, and the
interim way to author that content while the shape field does not exist.

Out of scope: choosing between `phased` and `mandate` (the Work shapes rule owns
it), the Swarm item that binds the plan (`swarm-manager-work-authoring`), the
goal message (`harness-goal-authoring`), the improvement loop itself
(`scenario-improvement-campaign`), and the aggregate allowance and resume
policy, which live on the Swarm item (`execution_limits`, `continuation`),
not in the plan.

Precondition: the scenario has a documented target (PRD or START-HERE plus
TESTING.md outcome inventory), a `<scenario>-improve` skill, and a
`<scenario>.setpoint-read` program. When one is missing, the first plan is a
bounded task or a phased plan that creates it. Do not author a mandate over an
undocumented target.

### 2. Required sections

Write every section. Each one has one rule.

| Section | Content | Rule |
|---|---|---|
| **Target pointer** | Paths to the scenario docs that hold the design and the outcome rows. | Point; never paste a snapshot. A copied board is stale on arrival and outranks the live read for no reason. |
| **Sensors** | The setpoint program by name, for example `program-runtime library run <scenario>.setpoint-read`, plus any named owner reads. | Name programs, not URLs or private scripts. The program contract owns row IDs and the wire vocabulary. |
| **Definition of done** | One band per outcome row: the row ID, the band predicate, and the validity rule (cohort, freshness, owner). | Every required row has a band. A row with no band is not required. The evidence audit passes: no band moved, no row hand-graded. |
| **Scope and authority** | `acceptance_allow` narrowed to the scenario, its shared packages, its protos and its docs. `scope_policy: extend-with-record`. Effects that need a separate decision (paid spend, private data, product promises). | Narrow the allow list; let `plan-manager exec boundary-extend` record the rest. `acceptance_deny` still refuses. |
| **Stop rules** | What counts as blocked. What is journaled instead of estimated. | Blocked means a decision, credential, or approval the agent lacks. A row that reads unavailable is journaled, not estimated. Friction the agent can diagnose is not blocked. |
| **Suggested arc** | understand → instrument → improve → validate. | Guidance, reorderable. Not phases, not steps. The agent may re-enter any stop when evidence sends it there. |
| **Journal location** | The plan-artifacts directory, normally `~/.vrooli/plan-artifacts/<session-slug>`. | One journal. Checkpoints go through Plan Manager; readings and rejected hypotheses go in the journal. |

Test the mandate with one question: can a fresh agent read the target docs,
run the sensor, and know from the bands alone whether it is done? If the agent
must ask what "done" means, a band is missing or a target snapshot has replaced
the pointer.

### 3. Template

Fill every `<...>` field. Delete a row only when the scenario has no such row,
and say so in the definition of done.

```markdown
## Target pointer
- Design: path:scenarios/<scenario>/docs/START-HERE.md, path:scenarios/<scenario>/PRD.md
- Requirements: path:scenarios/<scenario>/requirements/
- Outcome inventory: path:scenarios/<scenario>/docs/internal/TESTING.md#outcome-evidence-inventory
- Known problems: path:scenarios/<scenario>/docs/internal/PROBLEMS.md
- Improve skill: prompt-manager skill read <scenario>-improve

## Sensors
- Board: program-runtime library run <scenario>.setpoint-read
- Owner reads: <command> — <what it establishes>

## Definition of done
| Row | Band | Validity |
|---|---|---|
| <OT-P0-001> | <predicate, e.g. p95 <= 800 ms on the paced cohort> | <cohort, freshness, owner rule> |
| <OT-P0-002> | <predicate> | <rule> |
Evidence audit: no band edited, no sensor edited to move a row, every required row read from the program, repeated as often as the outcome contract requires.

## Scope and authority
- acceptance_allow: scenarios/<scenario>/**, packages/proto/schemas/<scenario>/**, packages/proto/gen/go/<scenario>/**, packages/proto/gen/typescript/<scenario>/**, packages/<shared-package>/**
- scope_policy: extend-with-record
- Separate decisions: <paid spend | private data | product promise | none>

## Stop rules
- Blocked: a decision, credential, or approval the agent lacks. Name it.
- Journaled, not estimated: a row that reads unavailable; a sensor that fails transport.
- Not blocked: friction the agent can diagnose; an adjacent defect the agent can file.

## Suggested arc
understand → instrument → improve → validate. Reorderable. Re-enter any stop when evidence sends you there.

## Journal
~/.vrooli/plan-artifacts/<session-slug>/journal.md — readings, hypotheses, rejected hypotheses, remaining rows.
```

### 4. Until the shape field exists

Plan Manager has no `shape` field yet. Author the same content through
`plan-manager author` as follows.

1. Start the session with `plan-manager author start` and write the target
   pointer, sensors and scope into the plan's context and boundary sections.
2. Add one phase per arc stop (understand, instrument, improve, validate). Set
   each phase's acceptance to the band predicates it must make true. Keep steps
   to three or fewer per phase; each step names an evidence action, not a
   repair.
3. Keep the derived work posture the wizard reports; do not override it to fit
   the mandate.
4. Put the journal location in the plan's constraints section.
5. Finalize. Bind the plan to a Swarm item with `swarm-manager-work-authoring`.

Swarm runs such a plan in sliced mode today with
`execution_strategy: adaptive-improvement`, and in goal mode once that mode
exists. Both modes read the same plan; the mandate content does not change with
the mode.

### 5. Anti-patterns

| Anti-pattern | Consequence | Correction |
|---|---|---|
| A step list | The agent executes steps instead of moving rows; the plan is stale after the first reading | Write bands; let the campaign choose interventions |
| Whole-product Gherkin as per-phase acceptance | Every phase carries the same finish line and none can complete alone | One band set per phase; the whole-product condition lives in the definition of done |
| An embedded target snapshot | Two copies of the board; the copy in the plan is the one that misleads | Point at the docs and the program |
| A phase per sensor | Phases mirror instruments instead of the arc; validation splinters | Four arc stops at most; sensors are listed, not phased |
| A band chosen to fit the latest reading | The plan certifies the status quo | Take the band from the PRD or TESTING.md; record an undecided band as undecided |
| `acceptance_allow: **` | The grant is unbounded and the validation oracle is blind | Narrow to the scenario, its packages, protos and docs; extend with record |

### 6. Output expectations

You may write a mandate plan through `plan-manager author`, revise its bands
when the PRD changes, and propose an `acceptance_allow` narrowing. You must not
paste sensor output into the plan, invent a numeric band the PRD does not state,
add phases beyond the arc stops, or author a mandate over a scenario without an
improve skill and a setpoint program. Keep the rendered mirror under 300 lines.

### Troubleshooting & Edge Cases

- **The setpoint program returns no board.** The first intervention is to make
  it read. Write that as the instrument stop; do not estimate rows.
- **Every row reads `pending_telemetry`.** The mandate is still valid. The
  instrument stop owns the owner-backed sensors; bands stay as authored.
- **A band is undecided in the PRD.** Record `target: undecided` for that row
  and name the decision it waits on. The row is not required until decided.
- **The scenario has no improve skill.** Stop. Author it first with
  `improve-skill-authoring`; the mandate depends on it.
- **The Work shapes rule says phased.** Use `implementation-plan-authoring`. A
  mandate over a clear route wastes the route.
