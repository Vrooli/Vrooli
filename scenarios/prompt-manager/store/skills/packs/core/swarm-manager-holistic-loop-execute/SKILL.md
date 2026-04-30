# Swarm Manager Holistic Loop Execute

You are running the `holistic-loop` execution phase for initiative `{{INITIATIVE_NAME}}`.

Read `modes/holistic-loop/findings.md`, `modes/holistic-loop/initiative-plan.md`, member backlog item folders, and relevant code. Implement the next coherent slice of the initiative-wide plan. Preserve long-term maintainability over breadth.

Context:

- Title: `{{INITIATIVE_TITLE}}`
- Round: `{{ROUND_NUMBER}}`
- Run strategy: `{{RUN_STRATEGY}}`
- Operator note: `{{OPERATOR_NOTE}}`

Member items:

```json
{{MEMBER_ITEMS_JSON}}
```

Prior rounds:

```json
{{PRIOR_ROUNDS_JSON}}
```

Rules:

- Do not edit member backlog `spec.json` files directly.
- Use existing project commands and lifecycle rules.
- Run focused validation for the files/components you touch.
- If ground truth invalidates the plan, state `replan_needed: true` in your final handoff.

End with a final handoff listing completed work, files changed, tests run, blockers, and whether replanning is needed.
