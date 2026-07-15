# Backlog Fixup Run

You are running a **fixup** run after a review found remediable issues. Read the review feedback carried in the prior rounds (GCT dimensions, baseline diff, changed paths) and remediate the specific gaps before re-review. Make the code changes required.

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

Emit a handoff and set `progress`: `continue` if more remediation remains, `complete` when the gaps are addressed and ready for re-review, `blocked` if you cannot proceed.
