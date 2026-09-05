---
name: "command-center-improve"
description: "Improve Command Center and morning walk effectiveness from external binding health, observed preparation outcomes, and operator feedback without grading missing evidence as success."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  status: "active"
  revision: 2
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["command-center", "prompt-manager", "program-runtime", "source-ledger", "vrooli-memory"]
    commands: ["command-center", "prompt-manager skill read", "program-runtime library run", "source-ledger journal", "vrooli-memory learning"]
  origin:
    kind: "authored"
---
## Practice focus: Command Center Improvement

### 1. Focus and scope

Regulate the instrument and its morning walk capability. Improve judgment in skills, repeated composition in programs, and missing stable operations in their scenario owner. Do not change objectives, the operator's setpoint, team schedules, or upstream product state. `docs/agent-system/TARGET_MODEL.md` owns instrument boundaries; `docs/agent-system/SKILL_AUTHORING.md` owns the three-speed stack.

### 2. Setpoint

Run `program-runtime library run command-center.setpoint-read` before selecting work. Its contract owns the row vocabulary.

| Row | Sensor | Band | Initial standing |
|---|---|---|---|
| binding-health | program-runtime binding condition for Command Center | All declared bindings healthy; no sample means unknown | Re-read each cycle; no dated green claim. |
| learning | Vrooli Memory learning measure, scope command-center-usage, operation vision-walk-prep | Target null until two comparable windows exist; then seek lower effort to verified success without higher failure recurrence | Pending baseline, 2026-09-04. |
| external-friction | `agent-manager.friction-digest`, scenario command-center | Inspect recurring failure fingerprints; no invented numeric floor | `read_elsewhere`. |
| briefing-quality | `command-center.vision-walk-prep` and its behavioral fixtures | Twelve phases; exact checkpoint or explicit continuity failure; no false success from missing sources | `read_elsewhere`; source health is separate from program correctness. |
| operator-usefulness | Durable vision-walk-feedback entries; quantitative sensor not yet shipped | Target null | `pending_telemetry`, 2026-09-04. |

### 3. Sensors

Read `program-runtime bindings condition --scenario command-center --window-seconds 604800`. Prefer this external evidence over the scenario's own health claim. Use `program-runtime library run agent-manager.friction-digest --input scenario=command-center` for recurring command friction. Read `program-runtime library run command-center.learning-read` with explicit `from`, `to`, `operation=vision-walk-prep`, and `context_key` selectors for comparable completion windows. Inspect completion effort, round trips, unresolved attempts and advice outcomes. Test rehearsals must be excluded by the sensor. An empty, capped, invalid, or unavailable sample cannot establish efficacy.

The prep envelope reports phase count, readable/unavailable sources, stale or undated sources, and manual supplements. These are execution observations, not proof of operator benefit. Read recent `vision-walk-feedback` through Source Ledger for the missing qualitative context; do not synthesize a success percentage by counting favorable prose.

### 4. Golden corpora

`path:scenarios/command-center/api/testdata/walk_program_behavior.py` is the deterministic behavioral corpus, run through `api/walk_program_test.go` in Test Genie. Every assertion must pass: exact phase coverage, true zero, source outages, invalid input, active/completed/invalid checkpoints, stale evidence, and complete bounded output. This is a correctness gate, not an empirically derived usage floor. Do not delete a failing case. No live usefulness floor is claimed before comparable operator attempts exist.

### 5. Routes

| Evidence | Repair route | Next-cycle evidence |
|---|---|---|
| Binding unavailable or freshness drift | Owning scenario implementation/build through scenario-work-ladder | Same binding condition |
| Correct source responses but malformed or omitted briefing sections | Command Center program/test repair | Behavioral corpus plus prep envelope |
| Repeated extra reads with stable joins | Promote composition into the owned program | Observed tool round trips for comparable attempts |
| Program compensates for schema/recovery/attribution rules | Add the owner operation and remove the compensation | Owner test and simpler program |
| User repeatedly corrects interpretation | Repair the relevant skill decision rule | Evidence-linked correction recurrence |
| No operator-usefulness sensor | File the missing measurement obligation; preserve feedback meanwhile | Sensor availability, not a guessed score |
| Healthy pipeline with two comparable usage windows | Compare effort and failure recurrence in the same context | Learning measure and actual outcome evidence |

### 6. Anti-gaming

Apply `improvement-do-and-dont` D1 (weakened tests), D2 (deleted problem evidence), D3 (suppressed findings), and its skeptic test. Never widen freshness to accept stale evidence, drop a phase, replace durable-product success with run activity, discard a checkpoint, suppress partial runs, count test provenance as operator use, or lower the operator's targets. Evidence-linked feedback is not causal proof.

### 7. Evidence

Record a `vrooli-memory journal note --kind work-record` per cycle with trigger, approach, the same sensor before/after, and outcome. Include corpus results, program ids, any pending baseline, and the concrete prose/program simplification. File defects in other owners through `report-bug`. Three consecutive unavailable cycles warrant a dated entry in Command Center's existing problems document.

### 8. Stop rules

- `no_governed_binding` or `pending_telemetry`: route the missing operation/measurement on the first read; do not wait for it to become green.
- `scenario_unreachable`: record it and wait until the next cycle; do not estimate.
- `read_elsewhere:<program>`: run that named program once.
- `unreliable:<why>` or `kernel_invoke_budget`: preserve the reason; do not band the reading.
- Insufficient comparable samples: leave the target null; refuse to lower or invent a floor.
- Required mutation grant: obtain authority through the runtime's supported path.
- Two cycles in band: propose close-out with outcome evidence; do not close automatically.
- Per-cycle budget: 15 minutes. Stop with evidence at the budget; it is a ceiling, not a duration target.
