# Goal Discover Workflow

Inspect the immutable goal snapshot and propose bounded missing work that advances the goal. Do not mutate the goal.

## Outcome work table

| Evidence | Outcome |
| --- | --- |
| One or more specific, in-scope proposals are supported by the snapshot. | `proposed` |
| The snapshot identifies a required fact or operator decision that is absent. | `needs_attention` |
| The snapshot does not identify the goal, its scope, or enough evidence for safe discovery. | `abstained` |

If a `proposed` predicate is not proven, select `needs_attention` and name the unproven predicate in `summary`.

## Required typed proposals

For `proposed`, place each graph mutation in `proposals` as a complete
`mutation_list` envelope, using only the operations in `{{.supported_ops}}`.
Include the snapshot goal version as `base_version`. `summary` explains the
evidence, scope fit, and duplicate risk; it is never the mutation transport.
Return an empty proposals array for `needs_attention` and `abstained`.

Missing work becomes `add_item` plus `add_goal_target`, and
`assign_milestone_items` when a milestone should own it — one envelope, so the
operator sees the work and its placement together. Search first with
`swarm-manager backlog search-ai "<intent>" --json` and name what you found;
propose new work only when nothing existing can absorb it.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Goal identity and version. |
| `{{.snapshot}}` | Goal targets, derived scope, milestones, and item evidence. |
| `{{.supported_ops}}` | The mutation operations this proposal may use. |

## Authority boundary

Write only the structured workflow result. Do not write files, create work, or mutate goals or milestones. The operator approves typed proposals in Goal Decide before Swarm changes the graph.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you evaluate existing scope. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Run `prompt-manager skill read swarm-manager-work-authoring` before you write any milestone title, acceptance criterion, or item description. It is the standard those fields are judged against.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
