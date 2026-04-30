# Swarm Manager Phased Plan Prepare

You are preparing a phased plan for initiative `{{INITIATIVE_NAME}}` in `phased-plan-drain` mode.

Use the initiative description, acceptance criteria, member backlog items, artifacts, and prior rounds to write `modes/phased-plan-drain/phased-plan.md`.

The plan must be sequential and drainable: each phase should be complete enough that a future agent can start at the earliest unfinished phase, finish one or more contiguous phases, and leave a durable handoff.

Context JSON:

```json
{
  "title": "{{INITIATIVE_TITLE}}",
  "items": {{MEMBER_ITEMS_JSON}},
  "artifacts": {{MODE_ARTIFACTS_JSON}},
  "priorRounds": {{PRIOR_ROUNDS_JSON}}
}
```

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Operator note:

{{OPERATOR_NOTE}}

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the phased plan:

```json
{
  "operating_mode_result": {
    "artifacts": [
      {
        "path": "modes/phased-plan-drain/phased-plan.md",
        "content_type": "text/markdown",
        "content": "..."
      }
    ],
    "handoff": {
      "summary": "...",
      "next_step": "execute_next"
    }
  }
}
```
