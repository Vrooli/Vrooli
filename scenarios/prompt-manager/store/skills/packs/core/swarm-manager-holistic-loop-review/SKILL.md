# Swarm Manager Holistic Loop Review

You are running the `holistic-loop` acceptance review phase for initiative `{{INITIATIVE_TITLE}}` (`{{INITIATIVE_NAME}}`).

Evaluate the current repository and mode artifacts against the initiative acceptance criteria. This is not the decision-oriented initiative review flow; this phase answers whether the holistic-loop work satisfies its own plan and criteria.

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Current initiative members:

```json
{{MEMBER_ITEMS_JSON}}
```

Artifacts and rounds:

```json
{{MODE_ARTIFACTS_JSON}}
{{PRIOR_ROUNDS_JSON}}
```

Produce a verdict of `accepted`, `changes_requested`, or `rejected` (the phase's declared enum). Use `changes_requested` when specific fixable gaps should be re-executed before acceptance — it routes the loop back to the delegated execute drain; any other non-accepting verdict records the gap and proceeds to reconcile. Include concrete evidence, tests inspected or run, gaps, and recommended next action. Do not mutate backlog specs directly.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the acceptance verdict:

```json
{
  "operating_mode_result": {
    "verdict": "accepted",
    "handoff": {
      "summary": "...",
      "tests": [],
      "blockers": [],
      "next_step": "complete initiative or return to execute"
    }
  }
}
```
