---
name: "deployment-manager-improve"
description: "Regulate Deployment Manager using release outcomes, readiness evidence, deployment friction, and honest pending telemetry; route repairs to skills, programs, or the durable scenario substrate."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["deployment-manager", "improve", "self-improvement", "control-loop"]
  icon: "gauge"
  status: "active"
  revision: 3
  createdAt: "2026-09-03T00:00:00Z"
  updatedAt: "2026-09-04T00:00:00Z"
  requires:
    scenarios: ["deployment-manager", "program-runtime", "agent-manager", "vrooli-memory"]
    commands: ["deployment-manager", "measures-health", "program-runtime", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## 1. Focus and scope

Regulate Deployment Manager as the durable release substrate used by its five
declared scenario dependents. Improve the usage skill when judgment fails, a
program when repeatable composition fails, and Deployment Manager when the missing
capability is an invariant or stable operation.

Do not deploy a release, weaken a gate, invent a reading, or edit another
scenario. This skill runs only for improvement work, not ordinary deployments.

Required reading:
- `prompt-manager skill read deployment-manager`
- `prompt-manager skill read improvement-do-and-dont`
- `prompt-manager skill read scenario-work-ladder`
- `path:docs/concepts/RECURSIVE_SELF_IMPROVEMENT.md` §"Fast learning, governed repetition, durable capability"

## 2. Setpoint

Read `deployment-manager.setpoint-read` first. Copy readings and unavailable
reasons verbatim; supplement the two external rows with the named programs.

| Row | Sensor and band | Current reading | Owner | Actuator | Anti-gaming and unavailable behavior |
|---|---|---|---|---|---|
| policy-projection | setpoint program; `matches=true`, at least 30 criteria | live | deployment-manager | canonical policy and projection test | Never lower the criterion count; `scenario_unreachable` stays unknown. |
| review-completion-rate | setpoint program; at least 0.90 after a representative sample | live when reviews exist | deployment-manager | review/goal synchronization | Empty sample is `unreliable`, never 100%. |
| required-evidence-availability | setpoint program; at least 0.98 | live when evidence exists | producer owners | producer adapter or owner bug | Waived/N/A count only when scoped and attributable; missing rows are not available. |
| stale-evidence-rate | setpoint program; at most 0.02 | live when evidence exists | deployment-manager | freshness/invalidation logic | Do not extend freshness to improve the reading. |
| predecessor-comparison-coverage | setpoint program; at least 0.95 when predecessors exist | live | deployment-manager | release lineage capture | First releases leave the denominator; missing history is unavailable, not first release. |
| waiver-count | setpoint program; inspect and reduce | live bounded count | release authority | remove cause or let scoped waiver expire | Never delete waiver history to improve the reading. |
| goal-synchronization-lag | setpoint program; at most 300 seconds | `pending_telemetry` | deployment-manager | typed closure reconciliation | File the measure obligation; do not estimate from timestamps by hand. |
| program-success-rate | `program-runtime.failure-triage`; at least 0.90 | `read_elsewhere` | program-runtime/deployment-manager | repair binding or program | Test provenance does not count as operator success. |
| external-friction | `agent-manager.friction-digest scenario=deployment-manager`; zero recurring fingerprints | external live read | deployment-manager | skill, program, or scenario repair | An unreachable agent-manager is unknown, never zero friction. |
| learning-failure-recurrence | Vrooli Memory learning measure; target null pending baseline | live when eligible attempts exist | vrooli-memory | repair recurring usage failure | Retain failures and unavailable attempts. |
| learning-success-effort | Same sensor; target null pending baseline | live by operation/context | vrooli-memory | reduce effort to verified success | Retain unresolved and incomplete task histories. |
| learning-advice-outcomes | Same sensor; target null pending baseline | live by operation/context | vrooli-memory | correct decision guidance | Evidence-linked reports are not causal effects. |


## 3. Sensors

Read the measure contract, binding condition, and friction digest with the exact
commands above. External evidence outranks Deployment Manager's self-report.
Rows marked pending telemetry are obligations, not estimates; file them through
the `skill-set-authoring` missing-sensor route.

Read `vrooli-memory learning measure --scope deployment-manager-usage` for the full
learning window. Use `--from`, `--to`, `--operation`, and `--context-key` for
comparable windows. The setpoint reader projects at most ten cohorts and exposes
sampling limits. Scope is fixed to this scenario; comparison never pools different
contexts. Missing history, legacy records, and capped scans remain unreliable.

Keep all three learning targets null until at least two comparable windows support
a baseline. Retain sample sizes, unresolved tasks, recall-unavailable counts, and
the baseline derivation in the cycle record. These reports do not establish causal
benefit or independent evidence verification. Check the referenced owner evidence
before claiming an improvement. A fixture proves the sensor, not operator benefit.

## 4. Golden corpora

Deployment Manager has no declared improvement corpus with a derived floor. Keep
this section empty until comparable release, migration, or recovery fixtures can
justify one. Do not turn a single successful deployment into a floor.

## 5. Routes

| Reading | Route | Sensor expected to move |
|---|---|---|
| Agent chose the wrong valid operation | Repair the usage skill's decision predicate and supersede the failed memory advice | external-friction |
| Several valid operations recur with stable joins | Author or repair a governed program; keep effects and budgets explicit | program-coverage |
| Program repeatedly parses private state, repairs compatibility, retries deployment, or implements migration/recovery policy | W1/W3 through `scenario-work-ladder`: add the missing scenario contract or implementation, then simplify the program | binding-condition |
| Required readiness/migration/regression row is `pending_telemetry` | File the measure obligation; do not compute it in skill prose | named pending row |
| Readiness waiver domain remains uncovered | Route to `measures-adoption`; do not waive the domain merely to clear the report | measure-contract |
| Storage or migration evidence violates the declared maturity tier | Route through `storage-steer`, then the work ladder rung its gate identifies | migration-safety |
| Defect belongs to a producer or dependency | File `report-bug` against that owner with commit/run evidence | affected row |

Learning routes are selected before interpreting a trend: missing or legacy
attempt records → repair capture coverage; capped/invalid sensor reads → report
the Vrooli Memory limitation with the window and scope; valid readings without
a baseline → retain the window and collect a comparable window during subsequent
use. An adverse supported trend routes the affected usage decision or scenario
failure through the existing routes above. No route drops failed attempts.

## 6. Anti-gaming

`improvement-do-and-dont` D1, D2, D3 and §2's skeptic test apply. In particular:

- Do not lower a test, coverage, or outcome floor to match a weak release.
- Do not count an open readiness goal, fitness score, package build, or waived
  unknown as approval.
- Do not improve program success by removing the hard target, migration, recovery,
  or evidence branch.
- Do not hide a recurring workaround in longer skill prose.
- Do not edit requirements status or release history to make the comparison pass.

## 7. Evidence

Write one `vrooli-memory journal note --scope deployment-manager-usage --kind work-record` per cycle with the row,
before reading, route, after reading, and owning layer. Add a PROBLEMS entry after
three consecutive `scenario_unreachable` readings. Use `report-bug` when another
scenario owns the defect.

## 8. Stop rules

- `no_governed_binding` or `pending_telemetry`: file the W1 or measure obligation on the first read.
- `scenario_unreachable`: record it without estimating and wait one cycle.
- `read_elsewhere:<program>`: run the named program.
- `unreliable:<why>` or `kernel_invoke_budget`: report the reason and do not classify the band.
- Refuse to lower a corpus or quality floor because a candidate misses it.
- Request the exact grant when a route requires mutation authority.
- After two comparable cycles in band, propose close-out; do not close it yourself.
- Use a 30-minute wall-clock budget for one cycle and stop when it expires.

### Troubleshooting & Edge Cases

| Symptom | First check | Response |
|---|---|---|
| Most rows are pending telemetry | `measures-health validate scenario deployment-manager` | File sensor obligations in priority order; do not manufacture a setpoint reader |
| Program looks successful but release outcome worsens | Artifact, commit, and deployed-predecessor identity | Treat program success as execution evidence only; route the product regression |
| Skill keeps accumulating exceptions | Repeated task-record shapes | Promote stable composition to a program or missing invariants to the scenario |
| Scenario gains a robust operation but programs still work around it | Program contracts and bindings | Replace the workaround and reduce the skill leaf; promotion is incomplete until faster layers simplify |
