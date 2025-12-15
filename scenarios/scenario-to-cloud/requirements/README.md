# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the “what” in `PRD.md`. Numbers preserve ordering but do **not** imply priority.

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Contributor Notes
- Each requirement should reference `prd_ref` as the OT id (e.g. `OT-P0-003`).
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines and link to shared docs (`docs/testing/guides/requirement-tracking-quick-start.md`) for schema details.
