---
name: "improve-skill-authoring"
description: "Author a scenario's improve skill as a control loop: setpoint rows from the scenario's own targets, sensors that exist, floors from golden corpora, curation and ladder routes, anti-gaming by id, evidence, and stop rules. Renders the scenario's setpoint-read program from the same table."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill", "authoring", "improve", "self-improvement", "control-loop", "setpoint", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 4
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T23:00:00Z"
  requires:
    scenarios: ["prompt-manager", "program-runtime", "measures-health"]
    commands: ["prompt-manager skill read", "measures-health validate scenario", "program-runtime bindings describe", "program-runtime sessions create", "program-runtime programs submit", "program-runtime sessions delete"]
  origin:
    kind: "authored"
---
## Meta focus: Improve Skill Authoring

Author the improve role for one scenario as a control loop that regulates the scenario against a setpoint it does not own, reads sensors it does not decide with, and leaves evidence a later cycle can compare. The skill this produces is read by an agent whose task is the scenario itself, under `goal-loop` or a heartbeat; it is never loaded for ordinary use of the scenario.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" and §"Universal quality bars".
- `path:docs/agent-system/TARGET_MODEL.md` §2 — the control chain this skill instantiates at scenario scale. Sensor implies no authority.
- `path:docs/agent-system/FRAMEWORK_HEALTH.md` §"Deadband rule" — a deadband states the target, never the current reading.
- `prompt-manager skill read improvement-do-and-dont` — the anti-gaming patterns D1, D2, D3 the authored skill cites by id.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` — the standard the rendered `setpoint-read` program follows; its §"The envelope" defines the row shape, the closed `reason` vocabulary, and which reasons lower the program's status.

Inputs from `skill-set-authoring`: the sensor inventory (sensor, command, `measured` or `pending_telemetry`), the golden corpora with floors, the open problems ledger, the dependents count, the program inventory.

### 1. Scope

**In scope:** the improve skill's eight sections; the setpoint table and its rendering into `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.{py,json}`; the routes from a sensor reading to a curation move or a work-ladder rung.

**Out of scope:** defining sensors (measures live in the manifest, corpora in `evals/`); code work (the authored skill hands off to `scenario-work-ladder`); the usage role.

### 2. The eight sections

The authored skill has exactly these sections in this order. Each has a source the author reads and a rule the author applies.

| # | Section | Source | Rule |
|---|---|---|---|
| 1 | Focus and scope | the scenario's PRD one-liner; the dependents count | Name the plant (what is regulated) and what this skill never touches |
| 2 | Setpoint | PRD `OT-*` targets; the space doc if the scenario owns a projection; the sensor inventory | One row per goal: `row`, `sensor` (a command), `band` (a target), `today` (dated reading, or the row's `reason` when unavailable). A row with no measured sensor is kept with reason `pending_telemetry`; it is not turned into prose |
| 3 | Sensors | the inventory | The exact commands, plus the two fleet sensors every scenario has: `program-runtime bindings condition --scenario <scenario>` for its bindings and the `agent-manager.friction-digest` program (inputs `scenario`, `window_days`) for recurring friction on its commands. External sensors outrank self-reported ones |
| 4 | Golden corpora | `evals/*.json` | Cite the suite and its floor. The floor is re-derived from comparable runs and recorded with its derivation; a run below floor is a stop for every other route |
| 5 | Routes | the problems ledger; the sensor rows | A work table from `row out of band` to one of: curation move (data only), `scenario-work-ladder` rung (W0 to W3), file against another owner. Each route names the sensor that should move next cycle |
| 6 | Anti-gaming | `improvement-do-and-dont` §1 | Cite D1 (loosened or deleted tagged test), D2 (deleted known-issue ledger), D3 (suppressed finding) by id, and §2 (the skeptic test); add the scenario's own gaming moves (a waiver on a domain the sensor needs, editing a requirements registry to match a claim, widening a floor, deleting a failing corpus case) |
| 7 | Evidence | canon memory loop | One `vrooli-memory journal note --kind work-record` per cycle with the before and after reading of the row moved, on the same sensor; a `PROBLEMS.md` entry when a row is `scenario_unreachable` for three consecutive cycles; a `report-bug` filing when the fix belongs to another owner |
| 8 | Stop rules | this skill | One line per stop, in this order. Row `reason` decides the first four (vocabulary: `program-contracts.md` §"The envelope"): `no_governed_binding` or `pending_telemetry` → route on the first read (W1, or the filing recipe in `skill-set-authoring` Phase 2), never wait; `scenario_unreachable` → record, do not estimate, wait one cycle; `read_elsewhere:<program>` → run that program; `unreliable:<why>` or `kernel_invoke_budget` → report the reason, do not band. Then: floor below comparable count → refuse to lower it. Route needs a grant → request through the session path. Two cycles in band → propose close-out, do not close. Budget: one line stating the wall-clock budget per cycle as a duration this author sets; `goal-loop` passes it as `program-runtime sessions create --wall-budget <duration>` (the flag is optional; omitted, the runtime sets 4 h, reported as `wall_budget_millis`) and, when the contract's `budget.async` is true, as `programs submit --async --wait-timeout <duration>`; a synchronous submit is bound at 120 s by the runtime. It is a stop, never a target |

### 3. Writing the setpoint

A setpoint row is a sentence an agent can check:

```
| discovery-floor | program-runtime discovery eval --suite evals/discovery.primary.json | met >= floor | 41/45 (floor 43) 2026-09-02 |
```

Rules:
- **Band states the target.** Never write the current reading as the band. If no target exists, write the direction (`rising`, `falling`) and the reading, and mark the row `pending-baseline`; the rendered program emits `target: null` for it.
- **One sensor per row.** A row that needs two commands is two rows.
- **Readings are dated.** They are observations, not state; the next cycle re-reads them.
- **The recursive row** belongs only to scenarios whose improve loop installs capability elsewhere (program-runtime, prompt-manager): "share of <targets> with a conformant skill set". No sensor computes conformance yet, so the row is `pending_telemetry` with `target: null`. Its interim read is `prompt-manager.skill-set-read` (contract under `scenarios/prompt-manager/.vrooli/program-runtime/`), one target per run: it reports the ids registered under the scenario pack, whether `<scenario>` and `<scenario>-improve` are registered, and the set's token size; it cannot report waiver grading, `programs[]` resolution, sensor reality, or frontmatter dialect, and its `read-counts` row is `unreliable:proto_drift_skill_usage` until the skill-usage binding is repaired. The row's actuator is filing `skill-set-authoring` runs against owners, never editing another scenario.

### 4. Rendering `setpoint-read`

The setpoint table is also a program. Render `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py` and `.json` from it:

1. `collect`: one `gather` over every row whose sensor is a governed binding. Every other row is emitted with `unavailable: true` and a `reason` from the closed vocabulary in `program-contracts.md` §"The envelope": `no_governed_binding` for a CLI command with no binding, `pending_telemetry` for a row with no sensor, `read_elsewhere:<program>` for a row another program owns, `kernel_invoke_budget` for a binding that outruns the invoke budget. A binding that raises is classified by the verbatim `classify_transport` from that guide.
2. `classify`: each row → `in_band` by the band rule, computed in the kernel from the handle (`count`, `head`, `group_by`); no materialization. A sensor whose own validity gate failed is `unreliable:<why>` with `reading` kept and `in_band: null`.
3. `report`: the envelope with `signals.rows` as the list of `{row, reading, target, in_band, unavailable, reason}`. `status` is `ok` when every read that was attempted returned; a permanent reason (`no_governed_binding`, `kernel_invoke_budget`, `read_elsewhere`, `pending_telemetry`) does not lower it; only a `scenario_unreachable` row or a failed read makes it `partial`. A `no_governed_binding` row is routed W1 by `goal-loop` on the first read; the program never waits on it, and the skill's stop rules say so.

The contract declares every binding with effect `read`, `budget.inference_calls: 0`, `budget.delegated_runs: 0`, `status.enum` limited to the statuses the program can reach, `errors.classes` as the minimum list in `program-contracts.md` plus domain classes, and one live fixture expecting `["ok"]`: a board whose unavailable rows are all permanent is `ok`, so the fixture accepts `partial` only when it names the row that is transiently unavailable in its `note`. Validate with `program-runtime sessions create --name <scenario>-setpoint --json` (`--json` because the id is read from `.session.id`), then `program-runtime programs submit --session-id <id> --source-file scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py --provenance operator --explain --json` (`--json` because the diagnostics list is read from the response), then `program-runtime sessions delete <id> --reason "explain done"` before finishing.

### 5. Routes

A route table row:

```
| agent-failure-rate above band with kernel_runtime naming a forbidden import | ladder W3: kernel guard for the name, then retire the prose | agent-failure-rate |
```

Choose the cheapest route that moves the sensor:

| If the fix is | Route | Who does it |
|---|---|---|
| A data or policy change the scenario already exposes (promote, supersede, gc, set-current, a config knob) | curation move, done in-cycle | the agent running the improve skill |
| A missing sensor (`pending_telemetry`) | backlog item by the filing recipe in `skill-set-authoring` Phase 2 | filed |
| A missing binding (`no_governed_binding`) | `scenario-work-ladder` rung W1 | filed as a backlog item on the goal's milestone |
| A contract, obligation, evidence, or implementation defect in this scenario | `scenario-work-ladder` rung named | filed as a backlog item on the goal's milestone |
| A defect in another scenario | `report-bug` against that owner | filed |

### 6. Convergence patterns

Two agents authoring the improve skill for the same scenario from the same inventory must produce the same setpoint rows, the same `reason` marks, and the same routes. The sources that decide it: the PRD's `OT-*` list, the inventory's `measured` column, the corpora's floors, the reason vocabulary in `program-contracts.md`, and the route table in §5.

### 7. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| A band equal to the last reading | Reads in band while the defect stands; can only detect growth | State the target; if unknown, `pending-baseline` |
| A sensor the skill computes by hand from files | The skill becomes the instrument and can be gamed by editing | Cite a measure, corpus, binding, or digest; file for what is missing |
| Retry logic in the routes | Hides the failure class the next cycle needs | One move per row per cycle; re-read next cycle |
| A route that edits another scenario | The instrument becomes a controller (`TARGET_MODEL.md` §2 "Sensor implies no authority") | File against the owner |
| Prose goals with no sensor | The loop regulates nothing and reports progress anyway | `pending_telemetry` rows and filed backlog items |
| A program status of `partial` because a row has no binding | The caller cannot tell a healthy board from a degraded one | Permanent reasons keep `ok`; only transient rows lower status |

### 8. Output expectations

You may: create `scenarios/<scenario>/skills/<scenario>-improve/SKILL.md`; render `setpoint-read.py` and `setpoint-read.json`; file backlog items and `report-bug` items.

You must: keep every sensor cited real; date every reading; cite anti-gaming by id (D1, D2, D3); emit only the closed `reason` vocabulary; make the setpoint table and the program agree (same rows, same bands); pass `skill-validation` §3.3 (divergence probe) on the routes table.

You must not: write a measure into a manifest; edit `PROBLEMS.md` to close an entry; author a row whose only sensor is an adjective.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every setpoint row is `pending_telemetry` | The scenario has no measures | `measures-health validate scenario <scenario>` | Author the skill anyway with the rows marked; its first route is the filing recipe; the skill is short and honest |
| A corpus has no floor field | Floors were never derived | `jq .floor evals/*.json` | Route: derive from at least two comparable runs and record the derivation; until then the row is `pending-baseline` |
| `setpoint-read` cannot call a sensor because it is a local CLI command | The command has `binding.kind: local` | `program-runtime bindings unbound` | The row carries `unavailable: true`, reason `no_governed_binding`; route W1 on the first read |
| The scenario owns a projection but `space --projection` fails | Space contract unimplemented | `<scenario> space --json` | Coverage row is `pending_telemetry`; route to W1 |
| Routes table has two rows for one reading | A C4 defect the divergence probe will catch | Run the probe | Merge or add a discriminating predicate |
