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
`mutation_list` envelope. The only supported goal operations are
`create_milestone`, `update_milestone`, `archive_milestone`,
`assign_milestone_items`, `unassign_milestone_items`, `add_goal_target`, and
`remove_goal_target`. Include the snapshot's goal `version` as
`base_version`; do not encode changes as prose in `summary`. `summary` is a
brief rationale and risk explanation for the operator. Use an empty array for
`needs_attention` or `abstained`.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Goal identity and version. |
| `{{.snapshot}}` | Goal targets, derived scope, milestones, and item evidence. |

## Authority boundary

Write only the structured workflow result. Do not run mutation commands. Do not create items, edit repository files, or mutate a goal or milestone. Operator approval through the Goal Decide surface is the only write path.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you reason about scope or reuse. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
