# Review Evidence Gathering

You are gathering the specific additional **evidence** an operator requested for an open review round. Produce the requested evidence (screenshots, api tests, cli output, config diffs, recordings) and append it to the review round.

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
  "evidence_request": "{{EVIDENCE_REQUEST}}"
}
```

`evidence_request` is the specific evidence the operator asked you to gather this sub-round — gather exactly that and append it to the open review round.

## Output

Emit a handoff and set `progress`: `continue` if the request thread stays open, `complete` when the requested evidence is gathered, `blocked` if you cannot gather it.
