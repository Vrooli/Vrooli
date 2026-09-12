# Requirements Registry

This registry maps Structure Health's **PRD operational targets** (`OT-P*-NNN` in
`PRD.md`) to concrete, test-linked technical requirements. Folders are numbered for
ordering only — numbers do **not** imply priority (criticality comes from each
requirement's `prd_ref` / `criticality`).

## Operational Targets → Modules
| Module | Covers operational targets |
|--------|----------------------------|
| `01-scenario-boundary` | OT-P0-004, OT-P0-005 — dual-mount validation service, agent-readable maturity assessment, finding↔ladder mapping |
| `02-ground-truth-and-intent` | OT-P0-001, OT-P0-003 — Code Facts client + fallback, `service.json` intent reader, reconcile model |
| `03-structure-and-lifecycle-rules` | OT-P0-001, OT-P0-002 — skeleton, freshness, lifecycle-wiring, production-serving, dependency rules |
| `04-profile-conformance` | OT-P0-003 — profile-keyed packs, default-profile parity, advisory relaxation |
| `05-autofix` | OT-P1-001, OT-P1-002 — shared `maturity-go/autofix`, format-preserving fixers, coverage metric |
| `06-fleet` | OT-P1-003, OT-P1-005 — typed all-kind fleet intelligence, project authority, and cross-kind maturity |
| `07-integration-and-cli` | OT-P0-005, OT-P1-004 — Test Genie provider cutover, thin human-default CLI |

## Auto-Sync Behavior
1. Operational targets in `PRD.md` map to the modules above; `index.json` imports each `module.json`.
2. Tests tagged with `[REQ:<ID>]` link to their requirement; after a test phase runs,
   Test Genie auto-syncs each validation's live status (`passed`/`failed`/`not_run`) back
   into these files and writes snapshots under `coverage/requirements-sync/`.
3. Coverage summaries land in `coverage/phase-results/` after each phase.

## Validation Commands
```bash
# Validate the requirements registry (structure + PRD-target linkage)
vrooli scenario requirements validate structure-health

# Validate the PRD itself
vrooli scenario requirements validate structure-health

# Regenerate requirement modules from the PRD (only when re-deriving from targets)
business-health wizard apply  # (was: requirements generate) structure-health

# Run the full suite (auto-syncs requirement statuses from [REQ:ID]-tagged tests)
vrooli scenario test structure-health
```

## Contributor Notes
- Keep every requirement's `prd_ref` pointing at a real `OT-P*-NNN` target in `PRD.md`.
- Tag tests with `[REQ:<ID>]` so auto-sync can update status; never hand-edit a synced status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations — let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. See `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync internals.
