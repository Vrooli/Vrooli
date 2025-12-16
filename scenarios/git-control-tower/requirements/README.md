# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in `PRD.md`.

## Lifecycle
1. Operational targets in `PRD.md` map to folders here (`01-*`, `02-*`, ...).
2. `requirements/index.json` imports each module; tests auto-sync requirement status when they run.
3. Run `vrooli scenario requirements lint-prd git-control-tower` to validate PRD ↔ requirements linkage.

## Contributor Notes
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Keep module files small and focused; prefer one module per operational target at first.
- Shared docs: `docs/testing/guides/requirement-tracking-quick-start.md`.
