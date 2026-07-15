# Scenario Spec Sync Run

You are syncing the spec artifacts of the scenario named `{{SCENARIO_NAME}}` to match its actual implementation. Read the scenario's implementation code and update its spec artifacts — `PRD.md`, `requirements/`, `README.md`, and `docs/` — so they describe the behavior the code actually has. Do not change the implementation; change only the spec/docs to match it.

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
  "scenario_name": "{{SCENARIO_NAME}}"
}
```

## Output

Emit a handoff and set `progress`: `continue` if more spec-sync work remains, `complete` when the spec matches the implementation, `blocked` if you cannot proceed.
