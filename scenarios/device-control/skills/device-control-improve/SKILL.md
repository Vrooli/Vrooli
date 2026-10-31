---
name: "device-control-improve"
description: "Reduce agent orientation and execution effort while preserving verified device outcomes, durable flow reuse, and safe repair."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["devices", "improvement", "learning"]
  icon: "gauge"
  status: "active"
  revision: 1
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["device-control", "program-runtime", "vrooli-memory", "agent-manager"]
    commands: ["device-control", "program-runtime", "vrooli-memory"]
  origin:
    kind: "authored"
---
## 1. Focus and scope

Regulate device-control's agent experience against the active operator goal and
path:scenarios/device-control/PRD.md, especially OT-P0-014.
The operator explicitly requests this role, irrespective of dependent count.
Read prompt-manager skill read device-control, improvement-do-and-dont,
and scenario-work-ladder before the first cycle.
Role/rung canon is path:docs/agent-system/SKILL_AUTHORING.md.

Use this skill for improving the capability, not for an ordinary TV request.
Read and route in one bounded cycle; execute substantial repairs through the
owning implementation workflow under existing task authority. A persistent goal
must include implementation execution, not just repeated filing.
Do not change another owner's scenario or operate a physical device unless the
task authorizes that device operation.

## 2. Setpoint

Read device-control.setpoint-read first. All learning targets remain null until
two comparable operator windows justify a baseline. Fixtures do not establish it.

| Row | Sensor | Band | Reading on 2026-09-04 |
|---|---|---|---|
| failure-recurrence | Memory learning measure | pending-baseline; reduce repeated fingerprints | live; empty sample unreliable |
| completion-effort | Memory learning measure | pending-baseline; reduce attempts/time with outcomes preserved | live |
| advice-outcomes | Memory learning measure | pending-baseline; fewer contradicted/unassessed uses | live |
| first-action-latency | Memory learning measure | pending-baseline | live when timestamps exist |
| agent-round-trips | Memory learning measure | pending-baseline | live when observed |
| visual-reasoning | Memory learning measure | pending-baseline | live when observed |
| workflow-reuse | Memory learning measure | pending-baseline; increase verified reuse | live when observed |
| binding-condition | program-runtime bindings condition | used bindings healthy | live; unexercised is unproven |
| external-friction | agent-manager.friction-digest | no recurring failures in representative window | read_elsewhere:agent-manager.friction-digest |

The destination includes exact device identity, declared observation capability,
asserted outcomes, version-preserving repair, and repeatability after restart.
A faster failed request is not an improvement. Use the same target/app/profile,
tool versions and comparable task populations. Separate cold orientation, known
flows, unfamiliar exploration and unavailable-device attempts.

## 3. Sensors

Run program-runtime library run device-control.setpoint-read.
Pass from/to, operation, and context_key to compare fixed learning windows.
Read vrooli-memory learning measure --scope device-control-usage for full cohort
details; the board intentionally projects one cohort.
Read program-runtime library run agent-manager.friction-digest --input scenario=device-control,window_days=7
for external observations. Truncated or unattributed windows cannot prove zero friction.
Read measures-health validate scenario device-control for measure coverage and
use provider gates selected by the work ladder for architecture and test quality.
program-runtime bindings condition --scenario device-control --window-seconds 604800
owns the binding reading; never exercise writes merely to make it healthy.

The Memory sensor reports caller-observed effort. Missing counts are unknown.
Capture coverage and evidence verification limit every claimed learning benefit.
Compare outcomes as well as speed, retaining unresolved tasks and sample sizes.

## 4. Golden corpora

No operator task-latency floor has been earned by this setup. Derive a baseline
from two comparable windows of explicitly authorized representative tasks.
Keep physical-device evidence separate from fake/transport replay validation.
Select approved behavior/quality floors from the owning test providers and
.vrooli/testing.json. Preserve an earned floor and record its derivation.

## 5. Routes

Take one attributable repair per cycle, ordered by the first applicable row.

| Finding | Route | Re-measure |
|---|---|---|
| Wrong device, weakened assertion, unsafe lease or false success | W0/W1 for missing contract, W3 for implementation; prioritize before speed | Same failed behavior plus provider evidence |
| Missing attempt coverage or invalid/capped cohort | Repair usage capture or file Memory owner defect | Same learning window and denominator |
| Wrong operation or repeated orientation | Repair usage predicate and supersede failed advice | first-action-latency, agent-round-trips |
| Repeated valid command sequence | Author a bounded program using program-runtime | agent-round-trips and verified outcome |
| Repeated successful exploration | Validate and save a deterministic candidate through author-flow | workflow-reuse, visual-reasoning |
| Existing reusable flow fails | Inspect run, repair candidate, preserve assertions and expected version | failure-recurrence and replay |
| Program compensates for identity/recovery/persistence defects | Work ladder for the durable owner operation; then delete workaround | Same outcome and effort cohort |
| Missing governed sensor or operation | File/reuse its W1 or measures-adoption obligation on first read | Named missing surface |
| External scenario owns the cause | report-bug with run and binding evidence | Owner repair and affected device task |

No route executes an unrequested physical action. General validation uses fakes;
a live benchmark names the device, action, initial state, and verification method.
Existing goal authority can authorize a concrete benchmark without repeated consent.

## 6. Anti-gaming

Apply improvement-do-and-dont D1, D2, D3 and the skeptic test.
Never remove failing tasks, mislabel tests as operator attempts, omit failed
attempts, lower floors, drop assertions, or treat screenless input as visual proof.
Do not count more reuse as progress if outcomes regress. Avoid blanket sleeps and
repeated model calls when a stable owner readiness condition exists.
Move stable capability into the scenario and simplify skills/programs afterward.

## 7. Evidence

Write one vrooli-memory journal note --scope device-control-usage --kind work-record
per cycle with trigger, selected row, before/after readings, route, work reference
and outcome. After three consecutive scenario_unreachable readings, append those
references to docs/internal/PROBLEMS.md. File external defects through report-bug.
A filed obligation is not a completed repair.

## 8. Stop rules

- no_governed_binding or pending_telemetry: file/reuse owner work on the first read.
- scenario_unreachable: record unknown and wait for the next cycle.
- read_elsewhere: run the named program and retain its validity limits.
- unreliable or kernel_invoke_budget: preserve the reason; do not band the row.
- A comparable corpus misses its floor: repair before other optimization.
- Missing mutation authority: request the exact grant through the owner path.
- Two comparable cycles meet every selected target: propose close-out; pending baselines and required evidence gaps prevent full completion.
- One cycle has a 30-minute wall-clock ceiling.

### Troubleshooting & Edge Cases

| Symptom | Next action |
|---|---|
| Board ok but learning rows unreliable | Collect eligible attempts; board execution is not product maturity |
| Faster rerun on a warm device | Compare a warm cohort, keeping cold-start results separate |
| Same failure repeatedly filed | Reuse its work ID and execute the authorized implementation route |
| Saved flow disappears after restart | Repair durable library persistence; do not hide copies in skill prose |
| Learned flow no longer matches the app | Preserve previous revision and validate a context-appropriate repair |
