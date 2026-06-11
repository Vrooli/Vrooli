# Swarm Manager Phased Plan Execute Next

You are executing the next drainable slice for initiative `{{INITIATIVE_NAME}}`.

Read `modes/phased-plan-drain/phased-plan.md`, `modes/phased-plan-drain/progress.json` if present, and prior round handoffs. Complete the earliest unfinished contiguous phase(s) you can fully finish in this run. Do not skip ahead around blockers.

Member items:

```json
{{MEMBER_ITEMS_JSON}}
```

Prior rounds:

```json
{{PRIOR_ROUNDS_JSON}}
```

Rules:

- **Recall prior work first** (AGENTS.md §4): before implementing a phase, `search-hub query "<phase intent>" --type record,skill,doc`. A `record` hit shows how a similar slice was solved before (trigger + approach) — build on it and cite it, or stop and reconcile a near-duplicate rather than redo it. Fall back to `swarm-manager records search "<intent>"` if search-hub is unavailable.
- Prioritize correctness and maintainability over breadth.
- Run focused validation for touched areas.
- Do not edit backlog `spec.json` files directly.
- End with a final handoff listing completed phases, changed files, tests, blockers, and the next earliest unfinished phase.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can persist the sequential handoff:

```json
{
  "operating_mode_result": {
    "handoff": {
      "summary": "...",
      "completed_phases": [],
      "changed_files": [],
      "tests": [],
      "blockers": [],
      "next_step": "classify_progress"
    }
  }
}
```
