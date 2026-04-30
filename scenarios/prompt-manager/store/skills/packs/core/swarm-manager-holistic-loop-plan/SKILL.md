# Swarm Manager Holistic Loop Plan

You are running the `holistic-loop` planning phase for initiative `{{INITIATIVE_NAME}}`.

Use the current findings, member items, acceptance criteria, artifacts, and prior rounds to produce an initiative-wide implementation plan. Treat backlog items as scope and audit markers, not as independent execution units.

Context:

```json
{
  "title": "{{INITIATIVE_TITLE}}",
  "mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round": "{{ROUND_NUMBER}}",
  "memberItems": {{MEMBER_ITEMS_JSON}},
  "artifacts": {{MODE_ARTIFACTS_JSON}},
  "priorRounds": {{PRIOR_ROUNDS_JSON}}
}
```

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Operator note:

{{OPERATOR_NOTE}}

Write or update `modes/holistic-loop/initiative-plan.md`. Include objective phases, dependencies, validation gates, expected files/components, and explicit handoff notes for future agents. Do not mutate backlog specs directly.
