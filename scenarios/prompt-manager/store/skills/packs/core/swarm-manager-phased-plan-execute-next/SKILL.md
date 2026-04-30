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

- Prioritize correctness and maintainability over breadth.
- Run focused validation for touched areas.
- Do not edit backlog `spec.json` files directly.
- End with a final handoff listing completed phases, changed files, tests, blockers, and the next earliest unfinished phase.
