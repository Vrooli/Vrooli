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

- **Recall prior work first** (AGENTS.md §4): before implementing the next slice, `search-hub query "<slice intent>" --type record,skill,doc`. A `record` hit carries how a prior agent solved a similar slice (trigger + approach) — build on it and cite it, or stop and reconcile a near-duplicate rather than redo it. Fall back to `swarm-manager records search "<intent>"` if search-hub is unavailable.
- Do not edit member backlog `spec.json` files directly.
- Use existing project commands and lifecycle rules.
- Run focused validation for the files/components you touch.
- If ground truth invalidates the plan, state `replan_needed: true` in your final handoff.

End with a final handoff listing completed work, files changed, tests run, blockers, and whether replanning is needed.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the handoff and replan signal:

```json
{
  "operating_mode_result": {
    "replan_needed": false,
    "handoff": {
      "summary": "...",
      "completed_phases": [],
      "changed_files": [],
      "tests": [],
      "blockers": [],
      "next_step": "review"
    },
    "backlog_sync": {
      "completed_items": [],
      "created_items": [],
      "updated_items": [],
      "proposal": {
        "form": "mutation_list",
        "mutations": []
      },
      "rationale": "Use the documented complete-items API for actual item completion; do not edit spec.json directly."
    }
  }
}
```
