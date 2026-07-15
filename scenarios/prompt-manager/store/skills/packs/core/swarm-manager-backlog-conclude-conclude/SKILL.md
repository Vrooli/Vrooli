# Backlog Conclusion Run

You are running the **primary execution** of a plan-less research/idea item. There is no bound implementation plan to drain — the deliverable is the conclusion itself. Read the item spec and any operator guidance, do the investigation/work the item calls for, and produce or update the item's conclusion deliverable. Make the repository/spec changes required.

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
  "item_spec": "{{ITEM_SPEC}}"
}
```

## Output

Emit a handoff and set `progress`: `continue` if more conclusion work remains, `complete` when the deliverable is done, `blocked` if you cannot proceed.
