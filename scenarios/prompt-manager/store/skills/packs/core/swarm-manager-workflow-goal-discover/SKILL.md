# Goal Discover Workflow

Inspect the immutable goal snapshot and propose bounded missing work that advances the goal. Do not mutate the goal.

## Outcome decision table

| Evidence | Outcome |
| --- | --- |
| One or more specific, in-scope proposals are supported by the snapshot. | `proposed` |
| The snapshot identifies a required fact or operator decision that is absent. | `needs_attention` |
| The snapshot does not identify the goal, its scope, or enough evidence for safe discovery. | `abstained` |

If a `proposed` predicate is not proven, select `needs_attention` and name the unproven predicate in `summary`.

## Required typed proposals

For `proposed`, place each graph mutation in `proposals` as a complete
`mutation_list` envelope using `create_milestone`, `update_milestone`,
`archive_milestone`, `assign_milestone_items`, `unassign_milestone_items`,
`add_goal_target`, or `remove_goal_target`. Include the snapshot goal version
as `base_version`. `summary` explains the evidence, scope fit, and duplicate
risk; it is never the mutation transport. Return an empty proposals array for
`needs_attention` and `abstained`.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Goal identity and version. |
| `{{.snapshot}}` | Goal targets, derived scope, milestones, and item evidence. |

## Authority boundary

Write only the structured workflow result. Do not write files, create work, or mutate goals or milestones. The operator approves typed proposals in Goal Decide before Swarm changes the graph.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you evaluate existing scope. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
