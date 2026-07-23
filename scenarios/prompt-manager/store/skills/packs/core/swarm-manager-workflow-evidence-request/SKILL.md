# Evidence Request Workflow

Gather exactly the evidence the request in the snapshot names, and return it with provenance. Do not judge the work and do not fix it.

## Decision rules

| Observable end state | Action |
| --- | --- |
| Every requested evidence entry is gathered. | Return each entry with what it shows and how you captured it. |
| Some requested evidence cannot be obtained. | Return the entries you could gather. Name each gap and its concrete cause in `summary`. Do not substitute narrative for missing evidence. |

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
