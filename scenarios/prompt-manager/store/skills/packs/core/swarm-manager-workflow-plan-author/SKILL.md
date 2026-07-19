# Swarm Manager Plan Author Workflow

Use `implementation-plan-authoring` in its **Candidate mode** to investigate
and author a detailed, Plan-Manager-compatible implementation plan for the
authorized backlog item. The supplied entity and snapshot are the bounded
planning source; do not assume an earlier transcript exists.

Preserve material operator intent, workshop decisions, discovered facts,
constraints, rationale, alternatives, diagrams or flows, references, risks,
validation expectations, and acceptance boundaries. Compress repetition, not
the decision-making context needed by a fresh execution agent. Follow that
skill's source inventory, placement map, and preservation audit before
returning a result.

This workflow returns a candidate only. Do not write files, create or update a
Plan Manager plan, mutate Swarm state, or claim that the plan is valid. Swarm
Manager imports the candidate, asks Plan Manager to validate it, and binds the
result only through its deterministic apply boundary.

Return only JSON matching the result contract:

- `ready` only when `candidatePlan` is complete enough for a fresh execution
  agent to implement and validate without this conversation;
- `needs_attention` when a material decision, fact, or authority is missing;
- `abstained` when you cannot safely assess or author the requested plan.

Entity:
{{.entity}}

Snapshot:
{{.snapshot}}
