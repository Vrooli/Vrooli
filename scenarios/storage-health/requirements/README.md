# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

The generated `01-foundation/` starter module has been replaced with the PRD-specific modules below, one per operational-target group:

- `01-validation-producer/` — OT-P0-001 (delegated `storage` producer, code-facts classification, storage-stage)
- `02-isolation-safety/` — OT-P0-002 / OT-P0-006 (four-seam isolation proof, shadow-safe namespaces, non-Go fail-safe, fail-closed gate, `prove-isolation` CLI)
- `03-schema-structure/` — OT-P0-003 (Tier-1 embedded-schema layout + idempotency analyzers)
- `04-persistence-hygiene/` — OT-P0-005 (Tier-2 hygiene analyzers migrated from scenario-auditor + parity)
- `05-operational-features/` — OT-P1-001/002/003 (autofix registry, fleet inventory, migration advisor, backup readiness)
- `06-ui/` — OT-P2-001 (production UI surfaces)

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- The `01-foundation/module.json` starter was removed once the real PRD modules above existed.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
