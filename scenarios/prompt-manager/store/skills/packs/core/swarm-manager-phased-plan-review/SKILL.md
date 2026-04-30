# Swarm Manager Phased Plan Review

You are reviewing initiative `{{INITIATIVE_NAME}}` in `phased-plan-drain` mode.

Evaluate the phased plan, progress state, prior handoffs, current code, and acceptance criteria. Decide whether the initiative is accepted, needs follow-up work, or needs replanning.

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Mode artifacts:

```json
{{MODE_ARTIFACTS_JSON}}
```

Prior rounds:

```json
{{PRIOR_ROUNDS_JSON}}
```

Produce a concise verdict with evidence, validation performed, remaining gaps, and recommended next action. Do not mutate backlog specs directly.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the review verdict:

```json
{
  "operating_mode_result": {
    "verdict": "accepted",
    "handoff": {
      "summary": "...",
      "tests": [],
      "blockers": [],
      "next_step": "complete initiative or return to prepare_plan"
    }
  }
}
```
