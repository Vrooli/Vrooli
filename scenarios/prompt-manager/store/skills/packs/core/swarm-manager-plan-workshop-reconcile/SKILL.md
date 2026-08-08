# Plan Workshop Reconciliation

Fold the operator's response — their answers and accepted proposals — into one coherent whole-plan candidate. The base is the pinned plan in the snapshot; the response is the complete statement of what changes.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| Every accepted proposal and every answer is reconciled into one coherent candidate plan. | `candidate` |
| Accepted items conflict with each other, or a required decision or authority is missing. Name the specific conflict or gap in `reason` — never resolve a conflict by silently picking a side. | `needs_attention` |
| No safe reconciliation is possible from the supplied inputs. | `abstained` |

## Reconciliation rules

- The candidate is complete plan content, not a diff. Preserve everything the response does not change.
- Reconcile against the pinned snapshot plan only. Do not fetch newer plan or subject state.
- Apply accepted proposals faithfully: the operator accepted that change, not your improvement of it. Integrate answers where their decision question points.
- When two accepted changes touch disjoint plan content, merge them. When they touch the same content, merge only if their combined result is unambiguous; when compatibility is not clearly established, default to `needs_attention` rather than choosing a merge. Mechanical adjustments the coexistence forces — renumbering, reordering references — are part of a faithful merge, not improvement.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.subject}}` | The workshop subject: kind (backlog item or milestone) and reference. |
| `{{.snapshot}}` | The workshop id, subject version, plan id, plan content hash, the full canonical plan, and the review packet the operator responded to. |
| `{{.response}}` | The operator's single response: answers, accepted proposal references, actor, and idempotency key. |
| `{{.accepted_proposals}}` | The accepted proposals with their full payloads. These plus the answers are the complete change set. |

## Boundary

Return a candidate only. Do not apply proposals, mutate backlog or workshop state, write files, or call Plan Manager. Plan Manager previews and validates the candidate; the operator applies it with explicit acknowledgment.

Subject:
{{.subject}}

Snapshot:
{{.snapshot}}

Response:
{{.response}}

Accepted proposal payloads:
{{.accepted_proposals}}
