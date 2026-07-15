# Backlog Revision Round

You are running a targeted **revision** round after a changes-requested decision. Apply the specific requested change to the item's spec/plan before it re-enters the pipeline. Make the changes required.

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

Emit a handoff and set `progress`: `continue` if further revision is warranted, `complete` when the revision is applied, `blocked` if you cannot proceed.
