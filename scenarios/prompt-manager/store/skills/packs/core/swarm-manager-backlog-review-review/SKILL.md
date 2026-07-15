# Backlog Review Round

You are running a **review** round over a completed backlog-item execution. Gather review evidence over the delivered work and return a readiness classification and assessment.

## Context

```json
{
  "operating_mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round_number": "{{ROUND_NUMBER}}",
  "agent_profile_key": "{{AGENT_PROFILE_KEY}}",
  "prior_rounds_json": {{PRIOR_ROUNDS_JSON}},
  "mode_artifacts_json": {{MODE_ARTIFACTS_JSON}},
  "item_title": "{{ITEM_TITLE}}",
  "item_status": "{{ITEM_STATUS}}",
  "item_spec": "{{ITEM_SPEC}}"
}
```

## Output

Return `verdict`: `ready` (meets its bar), `ready_with_notes` (acceptable with recorded notes), `needs_work` (specific fixable gaps), or `not_assessable` (cannot be assessed).
