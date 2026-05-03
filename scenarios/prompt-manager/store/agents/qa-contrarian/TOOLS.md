# TOOLS

## Tool Access
- `prompt-manager team member-context scenario-qa qa-contrarian`
- `prompt-manager team decision-list scenario-qa ...`
- `prompt-manager team decision-list <peer-team> ...` — read-only across peer teams (marketing-crew, monetization, infra-health, etc.)
- `prompt-manager team knowledge-list scenario-qa ...` — read bug-investigation/*, quality-audit/*, qa-run/* for challenge candidates
- `prompt-manager team knowledge-add scenario-qa --topic="challenge-note/..."` — write the challenge-note entries
- `swarm-manager` CLI — read-only for backlog item review
- `vrooli help`

## Forbidden
- Filing positive-action proposals (new bug-inbox entries, audit recommendations, backlog items). The contrarian writes `challenge-note/*` only.
- Editing target scenario code or scenario-qa member outputs. Challenge is a write, not a rewrite.
- Manufacturing challenges to hit a quota; quiet heartbeats are valid when peer outputs are sound.
- Free-form challenges with no cited failure mode; every challenge note must cite a specific failure mode from a registered technique's PoR doc.
