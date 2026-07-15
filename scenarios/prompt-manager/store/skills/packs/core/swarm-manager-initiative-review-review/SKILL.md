# Initiative Acceptance Review

You are running an **initiative** acceptance review. All member items are terminal; assess the initiative's delivered work against its acceptance criteria and return a readiness classification.

## Context

```json
{
  "operating_mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round_number": "{{ROUND_NUMBER}}",
  "agent_profile_key": "{{AGENT_PROFILE_KEY}}",
  "prior_rounds_json": {{PRIOR_ROUNDS_JSON}},
  "mode_artifacts_json": {{MODE_ARTIFACTS_JSON}},
  "initiative_name": "{{INITIATIVE_NAME}}",
  "initiative_title": "{{INITIATIVE_TITLE}}",
  "acceptance_criteria": "{{ACCEPTANCE_CRITERIA}}",
  "member_items_json": {{MEMBER_ITEMS_JSON}}
}
```

## Output

Return `verdict`: `ready`, `ready_with_notes`, `needs_work`, or `not_assessable`, judged strictly against the acceptance criteria.
