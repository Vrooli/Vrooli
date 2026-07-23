# Milestone Review Workflow

Judge one milestone's definition of done against its member-item evidence. Do not mutate the milestone, its goal, items, or repository files.

## Verdict decision table

| Evidence | Verdict |
| --- | --- |
| Every acceptance criterion has direct evidence and no material integration seam is missing. | `delivered` |
| Some criteria have direct evidence and the remaining work is bounded. | `partial` |
| Evidence shows that an acceptance criterion is not met. | `failed` |
| Evidence cannot prove any other row. | `needs_attention` |

If a non-conservative predicate is not proven, select `needs_attention` and name the unproven predicate in `assessment`. State the verdict first in `assessment`. Cite item and file evidence for every gap. Put only complete `mutation_list` envelopes in `proposals`; use goal-scoped `create_milestone`, `update_milestone`, `archive_milestone`, `assign_milestone_items`, `unassign_milestone_items`, `add_goal_target`, or `remove_goal_target` operations, and include the snapshot goal `version` as `base_version`. Use an empty array when no follow-up mutation is warranted.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Milestone identity, parent goal identity, and version. |
| `{{.snapshot}}` | Milestone definition, assigned items, and derived scope evidence. |

## Authority boundary

Write only the structured workflow result. Do not mutate milestones, goals, items, or repository files.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you judge scope coverage. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
