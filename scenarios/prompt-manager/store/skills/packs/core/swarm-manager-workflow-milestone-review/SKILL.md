---
name: "swarm-manager-workflow-milestone-review"
description: "Typed prompt contract for milestone review."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 2
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager", "swarm-manager goals"]
  origin:
    kind: "authored"
---
# Milestone Review Workflow

Judge one milestone's definition of done against its member-item evidence. Do not mutate the milestone, its goal, items, or repository files.

## Verdict work table

| Evidence | Verdict |
| --- | --- |
| Every acceptance criterion has direct evidence and no material integration seam is missing. | `delivered` |
| Some criteria have direct evidence and the remaining work is bounded. | `partial` |
| Evidence shows that an acceptance criterion is not met. | `failed` |
| Evidence cannot prove any other row. | `needs_attention` |

If a non-conservative predicate is not proven, select `needs_attention` and name the unproven predicate in `assessment`. State the verdict first in `assessment`. Cite item and file evidence for every gap. Put only complete `mutation_list` envelopes in `proposals`, using only the operations in `{{.supported_ops}}`, and include the snapshot goal `version` as `base_version`. Use an empty array when no follow-up mutation is warranted.

Member items being terminal is not evidence. Verify each criterion against the repository itself — run the tests or checks the criterion names and read the result. A criterion you cannot verify this way is `needs_attention`, not `delivered`.

An unmet criterion becomes follow-up work: `add_item` for what is missing, plus `add_goal_target` and `assign_milestone_items` so it lands in this milestone. A `partial` verdict with no proposals leaves the gap untracked.

## Variable legend

| Variable | Meaning |
| --- | --- |
| `{{.entity}}` | Milestone identity, parent goal identity, and version. |
| `{{.snapshot}}` | Milestone definition, assigned items, and derived scope evidence. |
| `{{.supported_ops}}` | The mutation operations this proposal may use. |

## Authority boundary

Write only the structured workflow result. Do not mutate milestones, goals, items, or repository files.

## Method

Run `prompt-manager skill read swarm-manager-goal-context` before you judge scope coverage. Use `swarm-manager goals context --name "<goal>"` only when the rendered snapshot needs confirmation.

Run `prompt-manager skill read swarm-manager-work-authoring` before you write the description of any follow-up item you propose. It is the standard that field is judged against.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
