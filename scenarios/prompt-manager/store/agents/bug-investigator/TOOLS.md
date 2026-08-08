# TOOLS

## Tool Access
- `prompt-manager team member-context scenario-qa bug-investigator`
- `prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/...`
- `prompt-manager team knowledge-update scenario-qa <id> ...` — used to re-tag entries on `route-to-another-topic` and to write `bug-investigation-report/<slug>` close entries
- `prompt-manager team knowledge-delete scenario-qa <id>` — used on `drop` outcomes after the bug-investigation entry is written
- `swarm-manager backlog list scenario-qa ...`
- `prompt-manager skill read scientific-debugging` — and any other skill registered in `path:docs/scenario-qa/methods/investigation/`
- `swarm-manager` CLI — used for `file-backlog` outcomes (creating a fix/chore/execute backlog item with the bug evidence) and for Phase 0 prior-art checks during scientific-debugging
- `vrooli help`
