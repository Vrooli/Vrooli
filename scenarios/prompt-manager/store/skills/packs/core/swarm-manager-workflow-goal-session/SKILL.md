---
name: "swarm-manager-workflow-goal-session"
description: "Typed prompt contract for one warm native-goal session in an accepted plan."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 4
  createdAt: "2026-09-10T00:00:00Z"
  updatedAt: "2026-09-11T06:40:00Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager discover", "swarm-manager"]
  origin:
    kind: "authored"
---
# Goal Session Workflow

Execute one warm, native-goal session of the accepted plan. This is a fresh
conversation with the accepted plan and compact handoffs as its only context.
The goal session is an execution harness, not the completion authority.

When `constraints.executionStrategy` is `adaptive-improvement`, read
`scenario-improvement-campaign` and the target scenario's `<scenario>-improve`
skill. Select the next falsifiable, in-scope intervention from the plan and
retained evidence. Do not create a second backlog item or approval for a repair.

Validation scope follows `docs/TESTING.md`: run targeted checks by default, and
run a baseline or full suite only where the plan's completion policy names one.
An adjacent defect follows the divergence tiers in `implementation-plan-execution`:
repair it when you understand its cause and it blocks the selected phase;
otherwise record it with `plan-manager log bug-add` and continue. When the
target documentation does not describe the design a phase implements, update
that documentation inside the write scope before the code.

## Procedure

1. Resolve `plan-manager exec status <plan_execution_id>` and read the bound
   execution's next required action.
2. Resolve the canonical structured plan from `.plan` in
   `plan-manager plans get <plan_reference> --json`. Build its authored
   frontier with `jq -cS` by deleting the computed top-level fields `status`,
   `content_hash`, `updated_at`, `work_posture`, `work_posture_source`,
   `work_posture_detail`, `mirror`, and `superseded_by`; delete `captured_at`
   from `regression_anchor`; delete `resolution`, `staleness`, and `change_factor`
   from every plan and phase reference; delete `status` and `status_detail` from
   every plan and phase relevant-context item; and delete `status`,
   `baseline_scope`, and `last_validation` from every phase. Do not create
   absent keys while transforming the object. The frontier digest hashes the
   plan reference, one NUL byte, the compact key-sorted authored JSON, and one
   trailing NUL byte with SHA-256. This canonical projection excludes
   execution/runtime state while pinning every authored instruction.

   Use this exact projection (substitute the bound reference literally; do not use an unvalidated environment value):

   ```bash
   plan-manager plans get <plan_reference> --json | jq -cS '.plan
     | del(.status,.content_hash,.updated_at,.work_posture,.work_posture_source,.work_posture_detail,.mirror,.superseded_by)
     | if has("regression_anchor") then .regression_anchor |= del(.captured_at) else . end
     | (.references[]? |= del(.resolution,.staleness,.change_factor))
     | (.relevant_context[]? |= del(.status,.status_detail))
     | (.phases[]? |= (del(.status,.baseline_scope,.last_validation)
         | (.references[]? |= del(.resolution,.staleness,.change_factor))
         | (.relevant_context[]? |= del(.status,.status_detail))))'
   ```

   Hash the compact line printed by that command, not its trailing newline.
3. Read the next required action from Plan Manager. The execution state is the
   frontier authority; handoffs provide context only.
4. Select the unfinished phase from execution state. If its relevant context
   names no skill, run `prompt-manager discover "<phase intent>" --type skill`.
5. Implement one coherent intervention inside the accepted write scope. Follow
   `docs/TESTING.md` and use the required focused validation.
6. Checkpoint the intervention, evidence, and remaining outcomes through Plan
   Manager. Transition a phase only when its acceptance holds. A session may
   return `continue` while the phase remains unfinished.
7. Re-read Plan Manager state and retained evidence before selecting the next
   intervention. Continue until the `until` condition holds, or return
   `blocked`, `abstained`, or an operator-decision approval request.
8. Return `complete` only after Plan Manager records terminal completion and
   every required outcome has applicable evidence. A native goal's self-reported
   completion is insufficient.
9. Write a handoff stating what changed, what was verified, and what remains.

An out-of-scope edit requires the accepted `extend-with-record` scope policy
and a Plan Manager decision record before the edit. Otherwise return an
operator-decision approval request with the exact scope extension needed.

## Scope policy

Apply this exact rule before any edit that is outside the authored
`acceptance_allow`:

```text
extend-with-record: run `plan-manager exec boundary-extend {{.plan_execution_id}} --paths <globs> --reason "<phase intent> needs <what>"` before edit, then name the recorded extension in the handoff.
fixed: return `approvalRequired: true` and `approvalReason: "operator-decision"`; state the exact paths in the handoff.
```

The execution constraints expose the selected `scopePolicy` and the effective
write scope. A recorded extension is accepted only when it is covered by that
scope and by the Plan Manager execution's boundary ledger. Plan acceptance is
unchanged by an extension.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| One intervention is verified and unfinished work remains | `continue` |
| Plan Manager reports terminal completion and all required outcomes have evidence | `complete` |
| Required authority or external access is absent | `blocked` with code, summary, and retry disposition |
| The plan digest is stale, unreadable, or contradicted by repository state | `abstained` |

On `continue`, set `correctionRequired` only for an unfixed defect from this
session. Set `approvalRequired` for a real operator decision or configured
phase boundary, and state the decision in the handoff. An accepted review does
not authorize a scope change.

Review accepts paths covered by the authored scope or a recorded extension and
rejects paths outside both.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.plan_reference}}` | Accepted canonical plan identity. |
| `{{.plan_execution_id}}` | Bound Plan Manager execution. |
| `{{.plan_digest}}` | Authored frontier digest. |
| `{{.constraints}}` | Session budget and write scope. |
| `{{.previous_handoffs}}` | Up to six newest handoffs. |

## Boundary

Write only inside the write-scope globs. You may mutate phase progress for the
bound Plan Manager execution after validation passes. Do not mutate plan content
or backlog records. Do not exceed the granted session or invent completion.

<plan_reference>{{.plan_reference}}</plan_reference>
<plan_execution_id>{{.plan_execution_id}}</plan_execution_id>
<plan_frontier_digest>{{.plan_digest}}</plan_frontier_digest>
<constraints>{{.constraints}}</constraints>
<previous_handoffs>{{.previous_handoffs}}</previous_handoffs>
