---
name: "swarm-manager-workflow-phased-plan-slice"
description: "Typed prompt contract for one authorized plan slice."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 14
  createdAt: "2026-07-18T03:05:26Z"
  updatedAt: "2026-09-11T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager discover", "swarm-manager"]
  origin:
    kind: "authored"
---
# Phased Plan Slice Workflow

Execute exactly one coherent slice of the accepted plan. This is a fresh conversation: your only context is the plan itself and the compact handoffs below. Never infer an earlier transcript.

When `constraints.executionStrategy` is `adaptive-improvement`, the accepted plan is
the campaign mandate. Read `scenario-improvement-campaign` and the target scenario's
`<scenario>-improve` skill, then choose the next falsifiable, in-scope improvement
from the plan and retained evidence. Do not create a second backlog item or approval
for an individual repair. The same plan frontier, write scope, aggregate budget,
checkpoint, and terminal evidence rules still apply. `phased-plan-drain` follows the
ordinary phase-by-phase procedure below.

## Procedure

1. Resolve the bound execution state with `plan-manager exec status <plan_execution_id>`.
2. Resolve the canonical structured plan from `.plan` in `plan-manager plans get <plan_reference> --json`. Build its authored frontier with `jq -cS` by deleting the computed top-level fields `status`, `content_hash`, `updated_at`, `work_posture`, `work_posture_source`, `work_posture_detail`, `mirror`, and `superseded_by`; delete `captured_at` from `regression_anchor`; delete `resolution`, `staleness`, and `change_factor` from every plan and phase reference; delete `status` and `status_detail` from every plan and phase relevant-context item; and delete `status`, `baseline_scope`, and `last_validation` from every phase. Do not create absent keys while transforming the object. The frontier digest hashes the plan reference, one NUL byte, the compact key-sorted authored JSON, and one trailing NUL byte with SHA-256. This canonical projection excludes execution/runtime state while pinning every authored instruction.

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
3. Read the execution's next required action from Plan Manager.
4. When an unfinished phase exists, select it from the execution state. Do not select it from handoffs. Read its `relevant_context`; when it declares no skill item, run `prompt-manager discover "<phase intent>" --type skill` and read what it returns.
5. Follow the bound execution's current owner action for required validation, synchronization, or an outcome assessment. Use the returned receipt identity and one producer wait. Client cancellation does not abort producer work. Missing historical evidence remains unknown unless the approved completion policy requires capture.
6. Implement one coherent intervention within the selected phase and write scope. Follow `docs/TESTING.md` for focused verification. Record evidence and remaining outcomes through Plan Manager. Complete a phase only when its acceptance holds. An adaptive intervention may return `continue` while that phase still has unmet outcomes.
7. When no unfinished phase remains, follow Plan Manager's terminal action. Assess the complete mandate against every required outcome and cohort. Obtain any evidence required by its completion policy. Broad advisory failures retain their disposition; they do not create new product requirements.
8. Return `complete` only after the owner records completion and every required mandate outcome has applicable evidence. Verify the retained assessment and limitations. A completed phase list or successful wrapper alone does not prove the target. Do not repeat unchanged validation merely to create another green result.
9. Write a handoff with local nuance for the next slice.

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

Classify each approval request so the workflow can preserve the accepted strategy:

| Situation on `continue` | Result fields |
| --- | --- |
| An authored phase is complete and no new authority is needed. | `approvalRequired: true`, `approvalReason: "phase-boundary"` |
| A target amendment, ungranted effect, destructive operation, or explicit operator pause needs a decision. | `approvalRequired: true`, `approvalReason: "operator-decision"` |
| An in-scope intervention, baseline setup, or other preparation leaves work within the current phase. | `approvalRequired: false`; omit `approvalReason` |
| Review finds a changed path covered by the authored scope or a recorded extension. | Accept the covered path. |
| Review finds a changed path outside both the authored scope and recorded extensions. | Reject it and preserve the operator-decision request. |

The accepted `adaptive-improvement` strategy continues past routine phase boundaries after independent review. Ordinary phased execution retains its configured phase approval policy. An `operator-decision` always waits; neither strategy nor a global automatic phase policy supplies missing authority. If both reasons apply, use `operator-decision`. State the required decision in the handoff before the affected action.

When the plan's item kind is `research`, execute the slice as investigation work. A valid slice may
produce a proposal, a goal, an answer, or evidence that the item should be resolved as `dropped`;
it does not need to produce a code diff. Use the swarm-manager CLI for project changes and decisions,
the same way the slice would use any other scenario CLI.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| This slice is done and verified, and unfinished plan work remains. Phase verification does not replace Plan Manager's separately reported `final_dod_required` action. | `continue` |
| This slice is done and verified, and it was the plan's last remaining work, including any terminal validation phase the plan authors. | `complete` |
| Required authority or external access is absent, and no permitted intervention remains. Fill `blocker` with the missing decision or access, stable code, and retry disposition. In-scope repairable defects remain work. | `blocked` |
| You cannot safely start: the plan is unreadable, the digest does not match the rendered content, or the handoffs contradict the repository state. | `abstained` |

Flag rules on `continue`:

- `correctionRequired`: true only when you finished the slice but found defects in it you could not fix within this run. True routes this result into a bounded correction turn instead of onward review.
- `approvalRequired` and `approvalReason`: use the decision table above. Preserve real authority requests through correction and review. Never label a target or grant amendment as a routine phase boundary.

Frontier rule: Plan Manager execution state is the frontier authority. Handoffs are only intra-slice nuance and must never override a completed or unfinished phase in execution state.

Handoff rule: the next slice runs as a fresh agent with no memory of this one. Write the handoff it needs: what this slice did, what it verified, and local context that helps the next authorized phase.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.plan_reference}}` | Identity of the accepted canonical plan. Resolve it; do not guess its content. |
| `{{.plan_execution_id}}` | Bound Plan Manager execution. Use its phase state as the frontier authority. |
| `{{.plan_digest}}` | Content pin for the accepted plan. Verify it against the canonical authored JSON projection from procedure step 2. A mismatch means stale authorization — abstain. |
| `{{.constraints}}` | The slice budget and the write scope globs. Write only inside them. |
| `{{.previous_handoffs}}` | Up to the last six slice handoffs, newest first. They are the only execution history you have. |

## Boundary

Write only inside the write-scope globs. You may mutate phase progress for the bound Plan Manager execution after validation passes. Do not mutate plan content or backlog records. Do not exceed one slice — stopping honestly beats overreaching.

<plan_reference>{{.plan_reference}}</plan_reference>
<plan_execution_id>{{.plan_execution_id}}</plan_execution_id>
<plan_frontier_digest>{{.plan_digest}}</plan_frontier_digest>
<constraints>{{.constraints}}</constraints>
<previous_handoffs>{{.previous_handoffs}}</previous_handoffs>
