# Swarm Manager Phased Plan Classify Progress

You are classifying progress for initiative `{{INITIATIVE_NAME}}` in `phased-plan-drain` mode.

Inspect `modes/phased-plan-drain/phased-plan.md`, `progress.json`, prior handoffs, and current repo state. Write or update `modes/phased-plan-drain/progress.json`.

Return one of:

- `continue`: more phases remain and the plan is still valid.
- `blocked`: progress cannot continue without operator action.
- `replan`: the plan is materially stale and should return to prepare_plan.
- `complete`: all planned phases are complete and review should run.

Include completed phase IDs, blocker notes, suggested backlog reconciliation counts, and evidence references.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist `progress.json` and emit a backlog-sync audit event:

```json
{
  "operating_mode_result": {
    "progress": {
      "decision": "continue",
      "completed_phases": [],
      "current_phase": "...",
      "rationale": "..."
    },
    "backlog_sync": {
      "completed_items": [],
      "created_items": [],
      "updated_items": [],
      "proposal": {
        "form": "mutation_list",
        "mutations": []
      },
      "rationale": "..."
    },
    "handoff": {
      "summary": "...",
      "next_step": "execute_next"
    }
  }
}
```
