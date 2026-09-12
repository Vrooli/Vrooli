# Requirements Registry

Requirement modules live here, grouped by operational target area. Every
requirement links back to a PRD operational target through `prd_ref` and carries
validation notes that become test-typed entries as each behavior lands.

Statuses are earned, not asserted: comprehensive suite runs can sync them from
`[REQ:ID]`-tagged evidence. Validate with `business-health validate scenario
portal`.

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

Portal keeps its v0 foundation requirements in `01-foundation/module.json` until larger chat/search domains justify separate modules.

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Remove or replace `01-foundation/module.json` once real PRD-generated modules exist.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README concise. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
