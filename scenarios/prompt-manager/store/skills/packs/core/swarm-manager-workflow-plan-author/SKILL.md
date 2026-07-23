# Plan Author Workflow

Author a Plan-Manager-compatible implementation plan for the authorized backlog item. Return the plan as a markdown candidate. The supplied entity and snapshot are the bounded planning source; no earlier transcript exists.

## Method

Run `prompt-manager skill read implementation-plan-authoring` and apply its **Candidate mode**. Follow its source inventory, placement map, and preservation audit before you return a result. Preserve material operator intent, workshop decisions, discovered facts, constraints, rationale, alternatives, diagrams, references, risks, validation expectations, and acceptance boundaries. Compress repetition, not the decision context a fresh execution agent needs.

## Outcome decision table

| Observable end state | Outcome |
| --- | --- |
| The candidate plan is complete enough for a fresh execution agent to implement and validate without this conversation, and it meets the authoring skill's quality bar. | `ready` |
| A material decision, fact, or authority is missing, and the gap blocks a complete plan. Name the gap in `reason`. A decision the operator explicitly delegated ("your call", "whatever's simplest") is not missing — that answer grants you the authority to make it; decide, record the rationale in the plan, and proceed toward `ready`. | `needs_attention` |
| You cannot safely assess or author the requested plan: the snapshot is unreadable or the subject is outside the supplied facts. | `abstained` |

On `ready`, Swarm imports the candidate into Plan Manager and rejects it if the rendered quality gate fails. A plan you doubt would pass belongs in `needs_attention`, not `ready`.

## Template variables

| Variable | Content |
| --- | --- |
| `{{.entity}}` | Subject identity: kind, name, version. |
| `{{.snapshot}}` | The full backlog item (including plan reference, acceptance globs, dependencies) plus the answered workshop-round history. Workshop decisions are operator authority — preserve them. |

## Boundary

Return a candidate only. Do not write files. Do not create or update a Plan Manager plan. Do not mutate Swarm state. Do not claim the plan is valid — Swarm imports the candidate and Plan Manager validates it.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
