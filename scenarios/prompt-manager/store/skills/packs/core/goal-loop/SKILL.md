---
name: "goal-loop"
description: "Select the development or observation posture for scenario improvement and route development to the campaign; use the scenario improve skill and caller-owned continuity. Does not grant permissions, compose a harness goal, or schedule itself."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["goal", "loop", "self-improvement", "setpoint", "heartbeat"]
  icon: "repeat"
  status: "active"
  revision: 7
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-11T12:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Practice focus: Goal Loop

Resolve the caller's target and authority, then select the development or
observation posture. Development routes to `scenario-improvement-campaign`.
Keep scheduling with the caller and execution with the existing runtime. This
skill does not create a Swarm goal, compose a harness goal, invent a cadence, or
grant implementation.

Required reading:
- `path:docs/agent-system/SCENARIO_DEVELOPMENT.md` — contract, authority, and completion.
- `prompt-manager skill read <scenario>-improve` when the role exists — scenario judgment.

### 1. Select the posture

Resolve the exact target from the request or work context. Ask only when ambiguity
would change the subject. Read the existing work identity, target, scope, effects,
budget, and acceptance policy.

| Caller authority | Next step |
| --- | --- |
| Approved development work or an explicit direct implementation request. | Read `scenario-improvement-campaign` and hand the engagement to it, including the canonical plan reference and retained adaptive checkpoint when present. Do not also run this skill's observation loop. |
| Read-only assessment or monitoring. | Read and report; do not implement or create work. |
| Explicit curation permission without development authority. | Perform only those curation operations and verify their effects. |
| A target change or effect outside the grant. | Request the missing decision before that action. |

Reuse the work identity across cycles. Do not create goals, milestones, or items
merely to run this skill. A Swarm goal is distinct from a harness goal: the
Swarm goal points to the work package and grants nothing by itself; the harness
goal, when the run has one, is composed by `harness-goal-authoring` and names the
finish line. This skill reads both and selects the posture; it writes neither.

### 2. Observe or perform authorized curation

Entry: the caller requested observation or bounded curation, not development.

Use the scenario's declared sensor program or named owner commands. Discover the
actual program contract before invocation. Inspect child outcomes and row validity,
not just submission status. Read
`path:scenarios/program-runtime/docs/guides/program-contracts.md` §"Setpoint-read rows"
when interpreting a board; that owner defines the wire vocabulary.

Report deviations, missing targets, and unavailable evidence without changing the
required outcome set. Missing improve guidance is a setup gap, not permission to
author it during monitoring. Perform a curation move only when explicitly granted;
re-read affected evidence afterward. Retain a worsened result and change the
hypothesis instead of repeating the same move.

Exit: return the observation, any authorized curation result, and the next wake or
stop reason. Retain evidence through the caller's output/log contract; observation
alone does not authorize a separate journal write.

### 3. Continue, wake, or stop

| Condition | Action |
| --- | --- |
| Monitoring has a caller-supplied schedule. | Return observations to that scheduler for its next wake. |
| Monitoring has no schedule. | Report once and return; do not create a background loop. |
| Required evidence satisfies the contract. | Return evidence for the caller's acceptance policy. |
| A specific action needs new authority. | Report the missing grant without performing that action. |
| Budget exhausted, authority revoked, or caller stops. | Stop new effects and preserve the checkpoint and pending-work disposition. |

Use the active owner's log and shared Memory contract. Do not duplicate automatic
capture. Swarm unavailability cannot manufacture authority: continue only when the
existing grant remains valid without another owner check. Otherwise report the
unavailable gate.

### 4. Anti-patterns and output

| Anti-pattern | Why it fails | Response |
| --- | --- | --- |
| Run both this loop and the campaign. | Duplicates execution and continuity. | Select one posture in §1. |
| Treat an observation request as a curation grant. | A read-only task gains unrequested effects. | Report once or use only the caller's schedule. |

Report the target, posture, authority source, evidence, unmet obligations, and
continuation or stop reason. All readable rows passing, two cached reads, or a
harness completion event does not establish product completion.

### 5. Troubleshooting & Edge Cases

| Situation | Response |
| --- | --- |
| No sensor program exists. | Inspect named owner reads and report setup gaps; do not turn monitoring into setup work. |
| Sensor transport fails. | Preserve failure and operation identity; do not reinterpret failure as zero. |
| Scheduled monitoring returns unchanged evidence. | Report the unchanged state at the caller's cadence; no experiment or repair is implied. |
| Runtime cannot continue the conversation. | Use its supported checkpoint/recovery path; do not start untracked replacement runs. |
