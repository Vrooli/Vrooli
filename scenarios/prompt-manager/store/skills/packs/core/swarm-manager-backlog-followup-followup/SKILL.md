# Backlog Follow-up Run

You are running a **follow-up** run after the item's execution completed. Read the completed deliverable and the follow-up context, then extend or amend the delivered work. Make the code changes required.

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

Emit a handoff and set `progress`: `continue` if more follow-up work remains, `complete` when the follow-up is done, `blocked` if you cannot proceed.
