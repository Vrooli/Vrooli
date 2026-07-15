# Backlog Research Round

You are running an autonomous **research** round over a single backlog item. Investigate the item's kind, deliverable, prior workshop history, and any attached context, then refine its spec/research toward readiness. Do not bind a readiness plan_ref — that is the finalize step.

## Context

```json
{
  "operating_mode": "{{OPERATING_MODE}}",
  "phase": "{{PHASE}}",
  "round_number": "{{ROUND_NUMBER}}",
  "agent_profile_key": "{{AGENT_PROFILE_KEY}}",
  "operator_note": "{{OPERATOR_NOTE}}",
  "user_prompt": "{{USER_PROMPT}}",
  "context_paths": "{{CONTEXT_PATHS}}",
  "context_targets": "{{CONTEXT_TARGETS}}",
  "context_requirements": "{{CONTEXT_REQUIREMENTS}}",
  "gap_report": "{{GAP_REPORT}}",
  "prior_rounds_json": {{PRIOR_ROUNDS_JSON}},
  "mode_artifacts_json": {{MODE_ARTIFACTS_JSON}},
  "item_title": "{{ITEM_TITLE}}",
  "item_description": "{{ITEM_DESCRIPTION}}",
  "item_status": "{{ITEM_STATUS}}",
  "item_spec": "{{ITEM_SPEC}}"
}
```

## Output

Emit the full structured result envelope only — never write `round-NNN.json` or any workshop files yourself; the operation runner persists the round from what you emit.

- `handoff`: state what you refined (`summary`, `blockers`, `next_step`, `changed_files`, `tests`).
- `progress`: `continue` if another research round is warranted, `complete` if the spec is ready for workshop synthesis, `blocked` if you cannot proceed.
- `decisions`: any lettered-option decisions this round surfaces for the operator to answer. Each is an object with `id`, `topic`, `text`, an optional `context`, and `options` — each option an object with `key` (a letter such as `A`), `label`, `rationale`, and optional `recommended`. Use an empty array when the round surfaces no new decisions.
- `self_assessment`: your readiness self-assessment across the five dimensions `problem_clarity`, `scope_defined`, `approach_solid`, `testable`, `risk_awareness`, each an integer from 0 to 3. This is an input to the workshop's readiness scoring, not the final verdict.
