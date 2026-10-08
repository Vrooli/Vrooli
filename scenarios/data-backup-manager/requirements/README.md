# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

Generated scenarios start with `01-foundation/module.json` so Test Genie can validate the registry immediately. Replace that starter module with PRD-specific modules during `docs/START-HERE.md` Gate 2.

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Validation

Each requirement carries a `validation` array with the evidence that proves the
claim. Test entries point to real `[REQ:ID]`-tagged tests and manual entries are
reserved for operator actions or native-platform evidence that cannot be
automated. Validation status is earned by comprehensive test runs; do not
hand-edit validation results or sync snapshots.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Remove or replace `01-foundation/module.json` once real PRD-generated modules exist.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
