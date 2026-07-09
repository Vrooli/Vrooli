# Swarm Manager Phased Plan Execute Next

Read the target plan (execution `{{PLAN_ID}}`, context below) and execute the next drainable slice — the earliest contiguous unit you can complete comprehensively. When you finish the slice, emit your final handoff stating the true frontier so the next fresh agent continues from the right place.

This is round {{ROUND_NUMBER}} of a sequential drain: first apply any corrections the prior handoffs call for, then continue from the last declared frontier.

Plan context:

```json
{{PLAN_CONTEXT_JSON}}
```

Prior rounds (accumulated handoffs — newest last):

```json
{{PRIOR_ROUNDS_JSON}}
```

Operator note: {{OPERATOR_NOTE}}

Rules:

- **Recall prior work first** (AGENTS.md §4): before implementing a slice, `search-hub query "<slice intent>" --type record,skill,doc`. A `record` hit shows how a similar slice was solved before (trigger + approach) — build on it and cite it, or stop and reconcile a near-duplicate rather than redo it. Fall back to `swarm-manager records search "<intent>"` if search-hub is unavailable.
- Prioritize correctness and maintainability over breadth. Do not skip ahead around blockers.
- Run focused validation for the areas you touched.
- The plan and its execution ledger live in plan-manager: record decisions, findings, and bugs in the plan-manager log for the active execution, and transition a plan phase through plan-manager when it is genuinely complete. Do not read or write local phased-plan/progress files.

{{ELASTIC_SLICE_SNIPPET}}

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the sequential handoff. Include `progress` inside the handoff — `continue` (more drainable slices remain), `complete` (the plan is fully drained), or `blocked` (you cannot advance the frontier) — so the transition routes deterministically:

```json
{
  "operating_mode_result": {
    "handoff": {
      "summary": "...",
      "blockers": ["none"],
      "next_step": "...",
      "changed_files": [],
      "tests": [],
      "frontier": "The next earliest unfinished unit the following round should advance — name the remaining phase or the exact remainder of this slice.",
      "progress": "continue"
    }
  }
}
```
