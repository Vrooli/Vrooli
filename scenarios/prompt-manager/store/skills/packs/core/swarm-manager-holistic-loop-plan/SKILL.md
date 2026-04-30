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

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the plan and readiness:

```json
{
  "operating_mode_result": {
    "artifacts": [
      {
        "path": "modes/holistic-loop/initiative-plan.md",
        "content_type": "text/markdown",
        "content": "..."
      }
    ],
    "readiness": {
      "dimensions": [
        { "key": "problem_clarity", "score": 0, "rationale": "..." },
        { "key": "scope_defined", "score": 0, "rationale": "..." },
        { "key": "approach_solid", "score": 0, "rationale": "..." },
        { "key": "testable", "score": 0, "rationale": "..." },
        { "key": "risk_awareness", "score": 0, "rationale": "..." },
        { "key": "coupling_understood", "score": 0, "rationale": "..." },
        { "key": "system_acceptance_defined", "score": 0, "rationale": "..." }
      ],
      "overall_score": 0,
      "ready": false
    },
    "handoff": {
      "summary": "...",
      "next_step": "execute"
    }
  }
}
```
