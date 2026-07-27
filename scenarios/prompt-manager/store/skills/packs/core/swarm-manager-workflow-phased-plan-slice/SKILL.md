# Phased Plan Slice Workflow

Execute exactly one coherent slice of the accepted plan. This is a fresh conversation: your only context is the plan itself and the compact handoffs below. Never infer an earlier transcript.

## Procedure

1. Resolve the bound execution state with `plan-manager exec status <plan_execution_id>`.
2. Resolve the plan content with `plan-manager plans render <plan_reference>`.
3. Select the next unfinished phase from the execution state. Do not select it from handoffs.
4. Read the skill items in that phase's `relevant_context`. When the phase declares no skill item, run `prompt-manager discover "<phase intent>" --type skill` and read what it returns.
5. Execute exactly that phase. Write only inside the write scope.
6. Run that phase's validation commands.
7. Mark the bound phase complete through `plan-manager exec` only after validation passes.
8. Write a handoff with local nuance for the next slice.

## Outcome decision table

| Observable end state | Outcome |
| --- | --- |
| This slice is done and verified, and unfinished plan work remains. Step-6 verification covers only this slice — a distinctly authored terminal validation phase is always unfinished plan work. | `continue` |
| This slice is done and verified, and it was the plan's last remaining work, including any terminal validation phase the plan authors. | `complete` |
| An obstacle outside your authority stops the slice: a missing dependency, a failing precondition, a scope conflict. Fill `blocker` with a stable code, a plain summary, and whether a retry could succeed. | `blocked` |
| You cannot safely start: the plan is unreadable, the digest does not match the rendered content, or the handoffs contradict the repository state. | `abstained` |

Flag rules on `continue`:

- `correctionRequired`: true only when you finished the slice but found defects in it you could not fix within this run. True routes this result into a bounded correction turn instead of onward review.
- `approvalRequired`: true only when the next slice must not start without operator approval — destructive steps, irreversible migrations, or an explicit pause the plan demands. True parks the workflow at a human approval gate.

Frontier rule: Plan Manager execution state is the frontier authority. Handoffs are only intra-slice nuance and must never override a completed or unfinished phase in execution state.

Handoff rule: the next slice runs as a fresh agent with no memory of this one. Write the handoff it needs: what this slice did, what it verified, and local context that helps the next authorized phase.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.plan_reference}}` | Identity of the accepted canonical plan. Resolve it; do not guess its content. |
| `{{.plan_execution_id}}` | Bound Plan Manager execution. Use its phase state as the frontier authority. |
| `{{.plan_digest}}` | Content pin for the accepted plan. A mismatch with the rendered plan means stale authorization — abstain. |
| `{{.constraints}}` | The slice budget and the write scope globs. Write only inside them. |
| `{{.previous_handoffs}}` | Up to the last six slice handoffs, newest first. They are the only execution history you have. |

## Boundary

Write only inside the write-scope globs. You may mutate phase progress for the bound Plan Manager execution after validation passes. Do not mutate plan content or backlog records. Do not exceed one slice — stopping honestly beats overreaching.

<plan_reference>{{.plan_reference}}</plan_reference>
<plan_execution_id>{{.plan_execution_id}}</plan_execution_id>
<plan_frontier_digest>{{.plan_digest}}</plan_frontier_digest>
<constraints>{{.constraints}}</constraints>
<previous_handoffs>{{.previous_handoffs}}</previous_handoffs>
