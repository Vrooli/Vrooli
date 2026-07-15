# Backlog Workshop Finalize

You are running the **finalize** round for a backlog item that has reached workshop readiness (all decisions answered). Author the implementation plan and hand off the readiness `plan_ref` (provider plan-manager) for the domain to bind — do not mutate the binding yourself.

## Context

```json
{
  "operating_mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round_number": "{{ROUND_NUMBER}}",
  "agent_profile_key": "{{AGENT_PROFILE_KEY}}",
  "operator_note": "{{OPERATOR_NOTE}}",
  "prior_rounds_json": {{PRIOR_ROUNDS_JSON}},
  "mode_artifacts_json": {{MODE_ARTIFACTS_JSON}},
  "item_title": "{{ITEM_TITLE}}",
  "item_description": "{{ITEM_DESCRIPTION}}",
  "item_status": "{{ITEM_STATUS}}",
  "item_spec": "{{ITEM_SPEC}}",
  "item_plan_ref": {{ITEM_PLAN_REF}}
}
```

## Output

Emit the full structured result envelope only — never bind the plan_ref yourself; the domain bind-plan action performs the binding from what you emit.

- `handoff`: `summary`, `blockers`, `next_step`, `changed_files`, `tests`.
- `progress`: `complete` when the plan is authored and ready to bind, `continue` if work remains, `blocked` if finalization cannot proceed.
- `plan_ref`: on the `complete` round, the canonical readiness plan reference you authored, as a top-level object with `provider` (`plan-manager`), `plan_id` and/or `slug`, and `role` (`execution_spec`). Omit it on `continue`/`blocked` rounds. The domain bind-plan action reads and validates this ref (fail-closed) and binds it to the item.
