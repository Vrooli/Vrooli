# Plan Workshop Review

Review the subject and its canonical plan from the immutable snapshot. Return one typed packet of findings, decision questions, and proposal drafts. The packet is the operator's entire review surface for this round — everything worth their attention goes in it, and nothing else does.

## Outcome work table

| Observable end state | Outcome |
| --- | --- |
| The snapshot supports a bounded review. Empty findings and questions are legitimate when the plan is sound — an honest clean packet beats invented concerns. | `packet` |
| A required material fact or authority is **absent or unparseable**: no plan where one is expected, or a snapshot field you cannot read. Name it in `reason`. | `needs_attention` |
| The snapshot cannot support a safe conclusion for a reason beyond a nameable missing fact. | `abstained` |

A field that is present and readable but stale or incorrect is never `needs_attention` — it is a packet finding. Reserve the attention outcomes strictly for missing or unparseable inputs.

`needs_attention` and `abstained` carry a reason only — never findings, questions, or proposals.

## Packet construction rules

- **Findings** are evidence-backed observations about the plan or subject: contradictions, missing validation, stale references, risk the plan does not name. Anchor each to the plan or snapshot content it comes from. Set severity honestly; do not inflate.
- **Questions** are only decisions the operator must make. Do not ask what the snapshot already answers. Give two to four concrete options when the decision space is enumerable; a question without options must genuinely be open-ended.
- **Proposal drafts** are concrete changes. Swarm validates each draft and writes the durable record into the Agent Session proposal store — do not invent session or proposal IDs. Select `apply_mode` per draft:

| The change is | `apply_mode` |
| --- | --- |
| A valid `mutation_list` envelope that the existing Agent Session mutation apply flow can execute safely after explicit operator consent. | `direct` |
| A plan-content change, or anything requiring agent judgment to merge — it must go through the reconciliation workflow as a candidate. | `reconciliation` |
| A proposal-shaped handoff that requires authority no automatic path has. | `attention` |

## Template variables

| Variable | Content |
| --- | --- |
| `{{.subject}}` | The workshop subject: kind (backlog item or milestone) and reference. |
| `{{.snapshot}}` | The workshop id, subject version, plan id, plan content hash, and the full canonical plan. Review this pinned content; do not fetch newer state. |

## Boundary

Mutate nothing: no file edits, no Swarm mutation, no Plan Manager calls. Do not emit readiness scores or a plan narrative — the packet's findings, questions, and proposals are the entire deliverable.

Subject:
{{.subject}}

Snapshot:
{{.snapshot}}
