# Swarm Manager Phased Plan Classify Progress

You are classifying progress for initiative `{{INITIATIVE_NAME}}` in `phased-plan-drain` mode.

Inspect `modes/phased-plan-drain/phased-plan.md`, `progress.json`, prior handoffs, and current repo state. Write or update `modes/phased-plan-drain/progress.json`.

Return one of:

- `continue`: more phases remain and the plan is still valid.
- `blocked`: progress cannot continue without operator action.
- `replan`: the plan is materially stale and should return to prepare_plan.
- `complete`: all planned phases are complete and review should run.

Include completed phase IDs, blocker notes, suggested backlog reconciliation counts, and evidence references.
