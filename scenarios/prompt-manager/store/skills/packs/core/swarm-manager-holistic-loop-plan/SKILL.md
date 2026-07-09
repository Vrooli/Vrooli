# Swarm Manager Holistic Loop Plan

You are running the `holistic-loop` planning phase for initiative `{{INITIATIVE_NAME}}`.

Use the current findings, member items, acceptance criteria, artifacts, and prior rounds to author (or revise) the **canonical initiative-wide plan in plan-manager**. Treat backlog items as scope and audit markers, not as independent execution units.

The execute phase of this loop is delegated to the generic `phased-plan-drain`: fresh agents will drain the plan you bind here slice-by-slice, so the plan must be **sequential and drainable** — each phase complete enough that a future agent can start at the earliest unfinished phase, finish one or more contiguous phases, and leave a durable handoff.

Author through the plan-manager CLI/API. Preserve or choose a stable slug for the initiative operating-mode plan, then return its `plan_ref` with provider `plan-manager` and role `operating_mode_plan`. Do not write any local plan artifact and do not mutate backlog specs directly.

Context:

```json
{
  "title": "{{INITIATIVE_TITLE}}",
  "mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round": "{{ROUND_NUMBER}}",
  "memberItems": {{MEMBER_ITEMS_JSON}},
  "planContext": {{PLAN_CONTEXT_JSON}},
  "artifacts": {{MODE_ARTIFACTS_JSON}},
  "priorRounds": {{PRIOR_ROUNDS_JSON}}
}
```

Acceptance criteria:

{{ACCEPTANCE_CRITERIA}}

Operator note:

{{OPERATOR_NOTE}}

Include in the plan: objective phases, dependencies, validation gates, expected files/components, and explicit handoff notes for the draining agents.

## Final Result Envelope

End your response with a fenced JSON block containing `operating_mode_result` so Swarm Manager can bind the canonical plan to the initiative and arm the delegated drain:

```json
{
  "operating_mode_result": {
    "plan_ref": {
      "provider": "plan-manager",
      "plan_id": "...",
      "slug": "...",
      "role": "operating_mode_plan"
    },
    "readiness": {
      "dimensions": [
        { "key": "problem_clarity", "score": 0, "rationale": "..." },
        { "key": "scope_defined", "score": 0, "rationale": "..." },
        { "key": "approach_solid", "score": 0, "rationale": "..." },
        { "key": "testable", "score": 0, "rationale": "..." },
        { "key": "risk_awareness", "score": 0, "rationale": "..." },
        { "key": "coupling_understood", "score": 0, "rationale": "..." },
        { "key": "system_acceptance_defined", "score": 0, "rationale": "..." }
      ],
      "overall_score": 0,
      "ready": false
    },
    "handoff": {
      "summary": "...",
      "next_step": "execute"
    }
  }
}
```
