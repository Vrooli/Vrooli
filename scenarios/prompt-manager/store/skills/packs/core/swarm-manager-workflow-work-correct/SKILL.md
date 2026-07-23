# Work Correction Workflow

Apply the authorized correction to the parent execution's completed work. Correct only what the correction request names. The snapshot is immutable; do not act on state it does not contain.

## Outcome decision table

| Observable end state | Outcome |
| --- | --- |
| The requested correction is fully applied, and every named verification command was observed green on its final run this session. Name the commands and results in `summary`. | `corrected` |
| The correction is partly applied, applying it exposed a new material problem, or any verification command was unrunnable, flaky, or not rerun. State exactly what remains in `summary`. | `needs_attention` |
| The correction request conflicts with the snapshot, or you cannot act on it safely. | `abstained` |

`corrected` completes the item — select it only with verification output in hand. `needs_attention` and `abstained` both route the item to operator review; the distinction is whether work remains (`needs_attention`) or the request itself is unsound (`abstained`).

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Subject identity: kind, name, executionId, version. |
| `{{.snapshot}}` | The backlog item, the parent execution id and status, the follow-up type, and the correction request. `operatorNote` and `finalizationFeedback` carry the correction request — they are the task authority. |

## Boundary

Edit repository files only within the item's acceptance scope. Do not mutate backlog records. Do not start work beyond the named correction.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
