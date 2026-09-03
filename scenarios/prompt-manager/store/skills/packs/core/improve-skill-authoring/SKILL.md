---
name: "improve-skill-authoring"
description: "Author a scenario's improve skill as a control loop: setpoint rows from the scenario's own targets, sensors that exist, floors from golden corpora, curation and ladder routes, anti-gaming by id, evidence, and stop rules. Renders the scenario's setpoint-read program from the same table."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta", "practice"]
  tags: ["skill", "authoring", "improve", "self-improvement", "control-loop", "setpoint", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["prompt-manager", "program-runtime", "measures-health"]
    commands: ["prompt-manager skill read", "measures-health validate scenario", "program-runtime bindings describe", "program-runtime programs submit"]
  origin:
    kind: "authored"
---
## Meta focus: Improve Skill Authoring

Author the improve role for one scenario as a control loop that regulates the scenario against a setpoint it does not own, reads sensors it does not decide with, and leaves evidence a later cycle can compare. The skill this produces is read by an agent whose task is the scenario itself, under `goal-loop` or a heartbeat; it is never loaded for ordinary use of the scenario.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` §"Scenario skill sets" and §"Universal quality bars".
- `path:docs/agent-system/TARGET_MODEL.md` §2 — the control chain this skill instantiates at scenario scale. Sensor implies no authority.
- `path:docs/agent-system/FRAMEWORK_HEALTH.md` §"Deadband rule" — a deadband states the target, never the current reading.
- `prompt-manager skill read improvement-do-and-dont` — the anti-gaming patterns the authored skill cites by number.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` — the standard the rendered `setpoint-read` program follows.

Inputs from `skill-set-authoring`: the sensor inventory (sensor, command, `measured` or `pending-telemetry`), the golden corpora with floors, the open problems ledger, the dependents list, the program inventory.

### 1. Scope

**In scope:** the improve skill's eight sections; the setpoint table and its rendering into `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.{py,json}`; the routes from a sensor reading to a curation move or a work-ladder rung.

**Out of scope:** defining sensors (measures live in the manifest, corpora in `evals/`); code work (the authored skill hands off to `scenario-work-ladder`); the usage role.

### 2. The eight sections

The authored skill has exactly these sections in this order. Each has a source the author reads and a rule the author applies.

| # | Section | Source | Rule |
|---|---|---|---|
| 1 | Focus and scope | the scenario's PRD one-liner; the dependents list | Name the plant (what is regulated) and what this skill never touches |
| 2 | Setpoint | PRD `OT-*` targets; the space doc if the scenario owns a projection; the sensor inventory | One row per goal: `row`, `sensor` (a command), `band` (a target), `today` (dated reading or `pending-telemetry`). A row with no measured sensor is kept and marked; it is not turned into prose |
| 3 | Sensors | the inventory | The exact commands, plus the two fleet sensors every scenario has: `program-runtime bindings condition --scenario <scenario>` for its bindings and the `agent-manager.friction-digest` program (inputs `scenario`, `window_days`) for recurring friction on its commands. External sensors outrank self-reported ones |
| 4 | Golden corpora | `evals/*.json` | Cite the suite and its floor. The floor is re-derived from comparable runs and recorded with its derivation; a run below floor is a stop for every other route |
| 5 | Routes | the problems ledger; the sensor rows | A work table from `row out of band` to one of: curation move (data only), `scenario-work-ladder` rung (W0 to W3), file against another owner. Each route names the sensor that should move next cycle |
| 6 | Anti-gaming | `improvement-do-and-dont` | Cite §1 and its three DON'T subheadings by name (tagged test, known-issue ledger, suppression) and §2 (the skeptic test); the skill has no per-pattern ids; add the scenario's own gaming moves (a waiver on a domain the sensor needs, editing a requirements registry to match a claim, widening a floor, deleting a failing corpus case) |
| 7 | Evidence | canon memory loop | One `vrooli-memory journal note --kind work-record` per cycle with the before and after reading of the row moved, on the same sensor; a `PROBLEMS.md` entry when a sensor is unavailable; a `report-bug` filing when the fix belongs to another owner |
| 8 | Stop rules | this skill | Sensor unavailable → record, do not estimate. Floor below comparable count → refuse to lower it. Route needs a grant → request through the session path. Two cycles in band → propose close-out, do not close. Budget: one line stating the cycle budget (wall-clock per cycle, defaulting to the program-runtime session ceiling) as a stop, never a target; `goal-loop` reads this line |

### 3. Writing the setpoint

A setpoint row is a sentence an agent can check:

```
| discovery-floor | program-runtime discovery eval --suite evals/discovery.primary.json --json | met >= floor | 41/45 (floor 43) 2026-09-02 |
```

Rules:
- **Band states the target.** Never write the current reading as the band. If no target exists, write the direction (`rising`, `falling`) and the reading, and mark the row `pending-baseline`.
- **One sensor per row.** A row that needs two commands is two rows.
- **Readings are dated.** They are observations, not state; the next cycle re-reads them.
- **The recursive row** belongs only to scenarios whose improve loop installs capability elsewhere (program-runtime, prompt-manager): "share of <targets> with a conformant skill set", read from the skill-set validator over the scenario's own ledger of callers. The row's actuator is filing `skill-set-authoring` runs against owners, never editing another scenario.

### 4. Rendering `setpoint-read`

The setpoint table is also a program. Render `scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py` and `.json` from it:

1. `collect`: one `gather` over every row whose sensor is a governed binding. Rows whose sensor is a local CLI command with no binding are emitted with `unavailable: true` and reason `no_governed_binding`; the program's own status stays `partial`, never `unavailable`, because the runtime was reached (`program-contracts.md` §"The envelope").
2. `classify`: each row → `in_band` by the band rule, computed in the kernel from the handle (`count`, `head`, `group_by`); no materialization.
3. `report`: the envelope with `signals.rows` as the list of `{row, reading, target, in_band, unavailable, reason}` and `status` = `ok` when every row was read, `partial` when any row is unavailable.

The contract declares every binding with effect `read`, `budget.inference_calls: 0`, `budget.delegated_runs: 0`, and one fixture expecting `["ok", "partial"]`. Validate with `program-runtime sessions create --name <scenario>-setpoint --json`, then `program-runtime programs submit --session-id <id> --source-file scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py --provenance operator --explain --json`, then `program-runtime sessions delete <id> --reason "explain done"` before finishing.

### 5. Routes

A route table row:

```
| agent-failure-rate above band with kernel_runtime naming a forbidden import | ladder W3: kernel guard for the name, then retire the prose | agent-failure-rate |
```

Choose the cheapest route that moves the sensor:

| If the fix is | Route | Who does it |
|---|---|---|
| A data or policy change the scenario already exposes (promote, supersede, gc, set-current, a config knob) | curation move, done in-cycle | the agent running the improve skill |
| A missing sensor | `measures-adoption` item | filed |
| A contract, obligation, evidence, or implementation defect in this scenario | `scenario-work-ladder` rung named | filed as a backlog item on the goal's milestone |
| A defect in another scenario | `report-bug` against that owner | filed |

### 6. Convergence patterns

Two agents authoring the improve skill for the same scenario from the same inventory must produce the same setpoint rows, the same `pending-telemetry` marks, and the same routes. The sources that decide it: the PRD's `OT-*` list, the inventory's `measured` column, the corpora's floors, and the route table in §5.

### 7. Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| A band equal to the last reading | Reads in band while the defect stands; can only detect growth | State the target; if unknown, `pending-baseline` |
| A sensor the skill computes by hand from files | The skill becomes the instrument and can be gamed by editing | Cite a measure, corpus, binding, or digest; file for what is missing |
| Retry logic in the routes | Hides the failure class the next cycle needs | One move per row per cycle; re-read next cycle |
| A route that edits another scenario | The instrument becomes a controller (`TARGET_MODEL.md` §2 "Sensor implies no authority") | File against the owner |
| Prose goals with no sensor | The loop regulates nothing and reports progress anyway | `pending-telemetry` rows and `measures-adoption` items |

### 8. Output expectations

You may: create `scenarios/<scenario>/skills/<scenario>-improve/SKILL.md`; render `setpoint-read.py` and `setpoint-read.json`; file `measures-adoption` and `report-bug` items.

You must: keep every sensor cited real; date every reading; cite anti-gaming by heading text; make the setpoint table and the program agree (same rows, same bands); pass `skill-validation` §3.3 (divergence probe) on the routes table.

You must not: write a measure into a manifest; edit `PROBLEMS.md` to close an entry; author a row whose only sensor is an adjective.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every setpoint row is `pending-telemetry` | The scenario has no measures | `measures-health validate scenario <scenario>` | Author the skill anyway with the rows marked; its first route is `measures-adoption`; the skill is short and honest |
| A corpus has no floor field | Floors were never derived | `jq .floor evals/*.json` | Route: derive from at least two comparable runs and record the derivation; until then the row is `pending-baseline` |
| `setpoint-read` cannot call a sensor because it is a local CLI command | The command has `binding.kind: local` | `program-runtime bindings unbound` | The row carries `unavailable: true`, reason `no_governed_binding`; file the binding as ladder W1 work |
| The scenario owns a projection but `space --projection` fails | Space contract unimplemented | `<scenario> space --json` | Coverage row is `pending-telemetry`; route to W1 |
| Routes table has two rows for one reading | A C4 defect the divergence probe will catch | Run the probe | Merge or add a discriminating predicate |
