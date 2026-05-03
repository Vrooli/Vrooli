# TOOLS

## Tool Access
- `prompt-manager team member-context scenario-qa bug-investigator`
- `prompt-manager team knowledge-list scenario-qa --topic-prefix=bug-inbox/...`
- `prompt-manager team knowledge-update scenario-qa <id> ...` — used to re-tag entries on `route-to-another-topic` and to write `bug-investigation/<slug>` close entries
- `prompt-manager team knowledge-delete scenario-qa <id>` — used on `drop` outcomes after the bug-investigation entry is written
- `prompt-manager team decision-list scenario-qa ...`
- `prompt-manager skill read scientific-debugging` — and any other skill registered in `docs/scenario-qa/investigation-techniques/`
- `swarm-manager` CLI — used for `file-backlog` outcomes (creating a fix/chore/execute backlog item with the bug evidence) and for Phase 0 prior-art checks during scientific-debugging
- `vrooli help`

## Forbidden
- Direct edits to target scenarios; investigation produces evidence, not patches. Patches go through `swarm-manager` backlog items.
- Writing to `bug-inbox/*` (the producer side); the bug-investigator only drains.
- Bypassing the technique registry; if a recurring approach doesn't fit any registered technique, surface a `meta-self-improvement` proposal naming the gap.
