---
name: "swarm-manager-workflow-phased-plan-slice"
description: "Typed prompt contract for one authorized plan slice."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 12
  createdAt: "2026-07-18T03:05:26Z"
  updatedAt: "2026-08-29T00:00:00Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager discover", "swarm-manager"]
  origin:
    kind: "authored"
---
# Phased Plan Slice Workflow

Execute exactly one coherent slice of the accepted plan. This is a fresh conversation: your only context is the plan itself and the compact handoffs below. Never infer an earlier transcript.

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
5. When the bound execution reports `freshen_status: baseline_required`, treat baseline capture as this slice even if no unfinished phase exists. Run the exact `capture_argv`, the exact one-shot `wait_argv`, and the exact `sync_argv` from execution status, then return `continue`. Never substitute validation evidence from another execution.
6. Execute exactly the selected phase. Write only inside the write scope. Run its validation commands, and mark it complete through `plan-manager exec` only after validation passes.
7. When every phase is done and Plan Manager reports `final_dod_required`, treat the terminal Definition-of-Done validation as this slice. The same rule applies when status says `execution_complete` but `plan-manager exec complete <plan_execution_id>` explicitly refuses because the required terminal Definition-of-Done validation result is absent; that refusal is authoritative evidence of the missing validation state, not a blocker. Create the full-inventory ticket with `plan-manager validate start <plan_reference> --execution <plan_execution_id>`, run the exact producer action and one native wait, synchronize it, then run `plan-manager exec complete <plan_execution_id>`. Do not require an unfinished phase in this state.
8. When the bound Plan Manager execution is already complete, verify its terminal validation operation and producer verdict from authoritative status, then return `complete`; do not rerun validation or require a resume point.
9. Write a handoff with local nuance for the next slice.

Set `approvalRequired: true` when `outcome` is `continue` because an authored phase was completed in this slice. Set it to `false` for baseline-only setup and other non-phase preparation. This is the contract that routes each reviewed phase boundary through the declared operator approval wait; do not silently continue from one completed phase to the next.

When the plan's item kind is `research`, execute the slice as investigation work. A valid slice may
produce a proposal, a goal, an answer, or evidence that the item should be resolved as `dropped`;
it does not need to produce a code diff. Use the swarm-manager CLI for project changes and decisions,
the same way the slice would use any other scenario CLI.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| This slice is done and verified, and unfinished plan work remains. Phase verification does not replace Plan Manager's separately reported `final_dod_required` action. | `continue` |
| This slice is done and verified, and it was the plan's last remaining work, including any terminal validation phase the plan authors. | `complete` |
| An obstacle outside your authority stops the slice: a missing dependency, a failing precondition, a scope conflict. Fill `blocker` with a stable code, a plain summary, and whether a retry could succeed. | `blocked` |
| You cannot safely start: the plan is unreadable, the digest does not match the rendered content, or the handoffs contradict the repository state. | `abstained` |

Flag rules on `continue`:

- `correctionRequired`: true only when you finished the slice but found defects in it you could not fix within this run. True routes this result into a bounded correction turn instead of onward review.
- `approvalRequired`: true whenever this slice completed an authored phase, as required above. It may also be true for a destructive step, irreversible migration, or explicit pause. True parks the workflow at a human approval gate; baseline-only and terminal-DoD slices leave it false.

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
