# Backlog Clarification Turn

You are running a **clarification** turn on a workshop decision thread. Answer the operator's question clearly and assess its impact on the decision or round (none / decision / round).

## Context

```json
{
  "operating_mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round_number": "{{ROUND_NUMBER}}",
  "agent_profile_key": "{{AGENT_PROFILE_KEY}}",
  "operator_note": "{{OPERATOR_NOTE}}",
  "user_question": "{{USER_QUESTION}}",
  "decision_topic": "{{DECISION_TOPIC}}",
  "user_message": "{{USER_MESSAGE}}",
  "prior_rounds_json": {{PRIOR_ROUNDS_JSON}},
  "mode_artifacts_json": {{MODE_ARTIFACTS_JSON}},
  "item_title": "{{ITEM_TITLE}}",
  "item_description": "{{ITEM_DESCRIPTION}}",
  "item_status": "{{ITEM_STATUS}}",
  "item_spec": "{{ITEM_SPEC}}"
}
```

## Output

Emit a handoff and set `progress`: `continue` if the thread stays open for follow-up, `complete` if the clarification is resolved, `blocked` if the item is busy or the thread is inactive.
