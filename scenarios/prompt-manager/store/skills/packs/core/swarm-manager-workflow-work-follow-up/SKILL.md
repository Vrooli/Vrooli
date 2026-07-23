# Work Follow-Up Workflow

Do the follow-up work the operator requested on the completed parent execution. The snapshot is immutable; the operator note defines the follow-up scope.

## Outcome decision table

| Observable end state | Outcome |
| --- | --- |
| The requested follow-up work is done, and you verified it. Name the verification in `summary`. Record any larger or adjacent work you identified as a recommendation inside `summary` — completing the request takes precedence over proposing beyond it. | `complete` |
| The requested work itself was not the right response, you did not complete it, and the right response is new or different work this run cannot do. Describe that work concretely in `summary`. | `proposed` |
| The work is partly done or blocked. State what remains and why in `summary`. | `needs_attention` |
| The request conflicts with the snapshot or exceeds this run's authority. | `abstained` |

Honesty rule: `proposed` does not create work — the operator reads `summary` and decides. Every outcome except `complete` routes the item to operator review, so the outcome value plus `summary` is the operator's entire decision input. Make `summary` carry it.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Subject identity: kind, name, executionId, version. |
| `{{.snapshot}}` | The backlog item, the parent execution id and status, the follow-up type, the operator note, and any finalization feedback. The operator note is the task authority. |

## Boundary

Edit repository files only within the item's acceptance scope. Do not mutate backlog records. Do not create backlog items or milestones.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
