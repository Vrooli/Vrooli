---
name: "swarm-manager-workflow-independent-review"
description: "Typed prompt contract for independent execution review."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 1
  createdAt: "2026-07-18T03:05:27Z"
  updatedAt: "2026-07-18T03:05:27Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager"]
  origin:
    kind: "authored"
---
# Independent Review Workflow

Independently review the completed backlog execution using only the immutable snapshot. Gather evidence, judge the deliverable against its plan, and report. Do not fix anything.

## Method

Run `prompt-manager skill read swarm-manager-review` and apply its evidence strategy, its GCT-results and baseline-delta evaluation steps, its evidence-type selection, and its classification rules. Its Inputs section describes a legacy envelope — your inputs arrive in `{{.snapshot}}` instead; the doctrine is unchanged.

## Verdict work table

| Observable end state | Verdict |
| --- | --- |
| The deliverable satisfies its plan or spec, and evidence shows no introduced regression. | `ready` |
| The deliverable satisfies its plan, with minor non-blocking observations. Record each observation in `notes`. | `ready_with_notes` |
| A material gap, defect, or introduced regression exists. Anchor each one to evidence. | `needs_work` |
| The snapshot lacks what you need to judge: no diff, no plan, and no way to observe the deliverable. | `not_assessable` |

Set `regression_introduced` to true only when baseline-delta or equivalent evidence shows a failure this execution introduced rather than inherited. Pre-existing failures are notes, not regressions. When a newly appearing failure cannot be causally attributed — a suspected flake — attempt to reproduce it as pre-existing; if you cannot, treat it as introduced and select `needs_work`. Never downgrade a new failure to a note on suspicion alone.

Set one typed `disposition` for the Plan Workshop finding: use `archive` for `ready` evidence, `follow_up` for a bounded correction or distinct next work, `plan_revision` or `plan_authoring` when the evidence changes planning, `supersede` when it replaces a prior direction, and `attention` when no safe recommendation exists. Give evidence-backed rationale and confidence. A disposition is advisory; it never authorizes a mutation.

When `disposition.kind` is `follow_up`, populate `disposition.follow_up` with a non-empty `steering` instruction and exactly one recovery `disposition`: `follow_up_run` to continue the verified work with that steering, `replan` when the plan itself must change, or `new_items` when the recovery should be split into independently actionable item specifications. `new_items` must include `items`, and every item needs `kind`, `name`, and `title`. Do not place this recovery contract in free-form notes.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Subject identity: kind, name, executionId, version. |
| `{{.snapshot}}` | The execution context: deliverable content, changed paths, affected scenarios, GCT review results, and baseline-diff results where available. |

## Boundary

This run is read-only. Do not modify files, the backlog item, or the execution. Your review is advisory: every verdict routes the item to operator review, and the operator decides. The verdict plus `agent_assessment` is what the operator reads first — lead with the verdict and the evidence that earns it.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
