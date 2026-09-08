---
name: "plan-manager-improve"
description: "Regulate Plan Manager from execution, validation, family, and authoring evidence, routing each gap to the owning skill, program, or scenario invariant."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["plan-manager","improve","setpoint","execution","validation","family"]
  icon: "gauge"
  status: "active"
  revision: 3
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-06T00:00:00Z"
  requires:
    scenarios: ["plan-manager", "test-genie", "git-control-tower", "agent-manager"]
    commands: ["plan-manager", "program-runtime", "vrooli-memory", "agent-manager"]
  origin:
    kind: "authored"
---
## Practice focus: Plan Manager Improve

### 1. Focus and scope

Improve verified planning outcomes and reduce coordination effort under the
operator's existing goal and PRD OT-P0-005 through OT-P0-007. Read `plan-manager`,
`improvement-do-and-dont`, and `scenario-work-ladder` before the first cycle.
Skills own judgment; programs own repeated composition; scenarios own invariants.
Do not change producer evidence, policy authority, or another owner's state.

### 2. Setpoint

Run `program-runtime library run plan-manager.setpoint-read` with explicit
`from`, `to`, `operation`, and `context_key` selectors for comparisons.
The program contract owns the rows; no dated green claim is carried forward.

| Row | Sensor | Target |
|---|---|---|
| binding-health | Program Runtime binding condition | Every observed binding healthy; unexercised is unproven. |
| usage-learning | Memory learning measure, scope plan-manager-usage | Pending baseline: reduce effort and failure recurrence with verified outcomes preserved. |
| recurring-friction | agent-manager.friction-digest, scenario plan-manager | No recurring unowned fingerprint; read_elsewhere. |
| validation-reuse | No measure yet | pending_telemetry; target null. |
| coordination-time | No measure yet | pending_telemetry; target null. |
| manual-recovery | No measure yet | pending_telemetry; target null. |
| family-critical-path | No measure yet | pending_telemetry; target null. |

The learning target stays null until two comparable operator windows establish a
baseline. Separate authoring, resume, phase validation and family work. Missing,
capped or mixed cohorts cannot establish a trend. Capture coverage limits every
claim: recorded attempts are not automatically the entire task population.

### 3. Sensors

The board joins external binding condition with the Memory-owned cohort. Use
`vrooli-memory learning measure --scope plan-manager-usage` with the same selectors
for full evidence. Read recurring friction through
`program-runtime library run agent-manager.friction-digest --input scenario=plan-manager`.
File or reuse a named measurement obligation on the first missing-telemetry read.
A board that executes successfully does not prove Plan Manager is mature.

### 4. Golden corpora

Use provider-owned contract and behavior tests for receipt consumption, reviewed
frontiers and content freshness. Preserve existing acceptance assertions.
No operator-effort floor has been earned from fixtures. Record comparable task
windows and their evidence before setting one. Any safety regression outranks
speed optimization.

### 5. Routes

Use the first applicable row; make one attributable repair and re-measure.

| Evidence | Route | Re-measure |
|---|---|---|
| False completion, stale graph launch, or commit-only invalidation | Work ladder for the owner invariant | Same failing behavior and provider proof. |
| Required evidence absent or cohort invalid | Repair capture or the owner measure; retain unknown | Same window and denominator. |
| Agent selects wrong operation or repeats orientation | Repair the usage predicate; supersede contradicted advice | Effort and failure recurrence. |
| Repeated stable cross-operation join | Add a bounded program and shorten the skill | Round trips and verified outcomes. |
| Program compensates for state/recovery defects | Repair the owner operation and remove compensation | Same outcome plus simpler program. |
| Required metric has no sensor | Execute its measurement obligation under task authority | Sensor availability, then measured value. |
| Cause belongs to another scenario | File attributable evidence with that owner | Affected operation after owner repair. |

### 6. Anti-gaming

Apply improvement-do-and-dont D1, D2, D3 and the skeptic test. Do not drop failed
attempts, loosen assertions, declare missing prior evidence a passing baseline,
change the operator's validation policy, count fixtures as operator use, or use
commit movement as content drift. Preserve producer observations when acceptance
policy changes. Ordinary plan completion and fleet maturity are separate claims:
advisory findings need not block the former and must remain visible in the latter.

### 7. Evidence

Write `vrooli-memory journal note --kind work-record` with trigger, selected row,
same-sensor before/after, repair reference, outcome and remaining gaps. Include
the instructions or program workaround removed. A filing is not a completed fix.
Three consecutive unavailable readings warrant a dated problems entry.

### 8. Stop rules

- pending_telemetry or no_governed_binding: route the obligation on the first read.
- scenario_unreachable: preserve unknown and wait for the next cycle.
- read_elsewhere: run the named program once.
- unreliable or kernel_invoke_budget: retain the reason; do not band the row.
- A safety or corpus regression stops other optimization until repaired.
- Missing authority uses the exact owning grant path; existing task authority persists.
- Two comparable cycles meet all selected targets: propose close-out with evidence.
  Required missing sensors and pending baselines prevent a full maturity claim.
- Each cycle has a 20-minute wall-clock ceiling; it is a limit, not a target.

### Troubleshooting & Edge Cases

- No eligible attempts: collect representative authorized usage; do not seed success.
- Several contexts returned: select one cohort before comparing windows.
- The same defect is repeatedly filed: reuse its work reference and execute the
  authorized repair rather than starting another observation-only cycle.
