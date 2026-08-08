# Plan Repair Workflow

Repair the supplied canonical plan so that every Plan Manager validation finding is resolved. Return a whole-plan candidate; do not apply it.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| Every validation finding is resolved in the candidate, and the candidate preserves the plan's intent and its unaffected content. | `ready` |
| One or more findings cannot be resolved from the supplied facts. List each unresolved finding in `unresolvedFindings` and name the missing fact or decision in `reason`. | `needs_attention` |
| The plan content or the findings are unreadable, or the repair requires authority you do not have. | `abstained` |

Repair rules: change only what a finding requires plus what that change forces. Do not restructure sections no finding touches. Do not invent scope the plan did not contain — but reusing content already in the plan (its acceptance globs, references, stated commands) to complete a finding's fix is derivation, not invention. A finding is resolvable when its fix can be derived from the plan's existing content or the supplied facts; route to `needs_attention` only when the fix requires a fact or decision present in neither. Do not claim the repaired plan is valid — Plan Manager revalidates it.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Subject identity: kind, name, version. |
| `{{.plan_reference}}` | Identity of the canonical plan under repair. |
| `{{.plan_digest}}` | Content pin for the supplied plan. Repair that content; do not fetch a newer version. |
| `{{.plan_content}}` | The full canonical plan to repair. |
| `{{.validation_findings}}` | Plan Manager's concrete findings. They define the entire repair scope. |
| `{{.constraints}}` | Bounds for this repair, including the attempt budget. |

## Boundary

Return a candidate only. Do not edit files, call Plan Manager, mutate Swarm state, or bind plan references. On `ready`, Swarm creates a Plan Manager candidate revision for preview; the operator decides whether it is applied.

<entity>{{.entity}}</entity>
<plan_reference>{{.plan_reference}}</plan_reference>
<plan_frontier_digest>{{.plan_digest}}</plan_frontier_digest>
<plan_content>{{.plan_content}}</plan_content>
<validation_findings>{{.validation_findings}}</validation_findings>
<constraints>{{.constraints}}</constraints>
