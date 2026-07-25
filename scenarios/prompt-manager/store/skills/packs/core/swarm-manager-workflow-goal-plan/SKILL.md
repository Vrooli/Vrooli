# Goal Plan Workflow

Use the immutable goal snapshot to propose a bounded, sequenced change to its targets, milestones, or dependencies. Do not mutate the goal.

## Outcome decision table

| Evidence | Outcome |
| --- | --- |
| A bounded sequence of target, milestone, or dependency changes is supported by the snapshot. | `proposed` |
| The snapshot identifies a material fact or operator decision that is required before a bounded proposal exists. | `needs_attention` |
| The snapshot does not identify the goal, its scope, or sufficient evidence to form a safe proposal. | `abstained` |

If a `proposed` predicate is not proven, select `needs_attention` and name the unproven predicate in `summary`.

## Required typed proposals

For `proposed`, emit every suggested write in `proposals` as a complete
`mutation_list` envelope, using only the operations in `{{.supported_ops}}`.
Include the snapshot's goal `version` as `base_version`; do not encode changes
as prose in `summary`. `summary` is a brief rationale and risk explanation for
the operator. Use an empty array for `needs_attention` or `abstained`.

Structure and its membership belong in one envelope: create a milestone, state
its acceptance criteria, create any missing items, target them, and assign them
as one sequence the operator decides together. Validation reads the list's own
effects, so a later mutation may reference what an earlier one creates.

A milestone requires acceptance criteria in Given/When/Then form. They are the
goal's only definition of done — milestone review reads them, and close-out is
gated on that review. Never restate the goal's own text as the criteria: the
goal states what becomes true, the milestone states how you would prove it.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Goal identity and version. |
| `{{.snapshot}}` | Goal targets, derived scope, milestones, and item evidence. |
| `{{.supported_ops}}` | The mutation operations this proposal may use. |

## Authority boundary

Write only the structured workflow result. Do not run mutation commands. Do not create items, edit repository files, or mutate a goal or milestone. Operator approval through the Goal Decide surface is the only write path.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you reason about scope or reuse. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Run `prompt-manager skill read swarm-manager-work-authoring` before you write any milestone title, acceptance criterion, or item description. It is the standard those fields are judged against.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
