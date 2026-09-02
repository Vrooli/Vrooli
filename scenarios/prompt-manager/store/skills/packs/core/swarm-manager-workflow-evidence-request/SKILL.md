---
name: "swarm-manager-workflow-evidence-request"
description: "Typed prompt contract for evidence gathering."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  tags: ["swarm-manager","agent-manager","workflow","prompt-contract"]
  status: "active"
  revision: 3
  createdAt: "2026-07-18T03:05:27Z"
  updatedAt: "2026-07-18T03:05:27Z"
  modes: ["contract"]
  requires:
    scenarios: ["prompt-manager", "swarm-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "swarm-manager"]
  origin:
    kind: "authored"
---
# Evidence Request Workflow

Gather exactly the evidence the request in the snapshot names, and return it with provenance. Do not judge the work and do not fix it.

## Decision rules

| Observable end state | Action |
| --- | --- |
| Every requested evidence entry is gathered. | Return each entry with what it shows and how you captured it. |
| Some requested evidence cannot be obtained. | Return the entries you could gather. Name each gap and its concrete cause in `summary`. Do not substitute narrative for missing evidence. |

Return `outcome: fulfilled` when every requested artifact was captured, even when an artifact reports failure. Return `outcome: needs_attention` only when at least one requested artifact could not be captured at all. The `outcome` field is required alongside `summary` and `evidence`.

Every evidence entry must be ledger-ready and include `id`, `criterion_id`, `type`, `title`, `description`, `producer`, `trust`, and `settlement`. Use the criterion id named by the request. `trust` is one of `authoritative`, `observed`, `reported`, or `operator_verified`; `settlement` is one of `settled`, `refuted`, or `unavailable`. Put structured command observations in optional `test_results`; do not invent alternate fields such as `artifact`, `shows`, or `provenance`.

An entry is **gathered** when its artifact is captured, whatever the artifact shows: an inconclusive or failing run is a gathered entry whose result you state plainly. A **gap** exists only when the artifact could not be captured at all.

## Method

Run `prompt-manager skill read swarm-manager-review` for evidence-type selection and capture doctrine. Prefer command output, test results, and diffs over prose description.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.snapshot}}` | The review context and the specific evidence request. The request text defines the entire scope of this run. |

## Boundary

Do not modify files, the backlog item, or the review. Gather and report only. Swarm appends your result to the review thread; an incomplete gather leaves the request pending, so state gaps plainly.

Snapshot:
{{.snapshot}}
