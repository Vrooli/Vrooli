# Requirements Registry

This registry maps Performance Health's **PRD operational targets** (`OT-P*-NNN` in
`PRD.md`) to concrete, test-linked technical requirements. Folders are numbered for
ordering only — numbers do **not** imply priority (criticality comes from each
requirement's `prd_ref` / `criticality`). Every `validation.ref` points to a file that
will exist once the corresponding code lands (no phantom refs).

## Operational Targets → Modules
| Module | Covers operational targets |
|--------|----------------------------|
| `01-scenario-boundary` | OT-P0-004, OT-P0-005 — dual-mount validation service, agent-readable maturity assessment, Test Genie performance provider |
| `02-tier-and-readiness` | OT-P0-001, OT-P0-002 — code-facts-gated tier detection, perf-build infra detection, format-preserving autofix |
| `03-capture-orchestration` | OT-P0-003 — profile-mode capture pipeline over BAS perf-capture, tier-0-never-fails, graceful headless skip |
| `04-analysis` | OT-P0-004, OT-P1-003 — per-component aggregation, deterministic located findings, before/after comparison |
| `05-lighthouse-and-benchmarks` | OT-P1-001 — Lighthouse runner + silent-skip, build-time (axis ①) benchmarks |
| `06-budgets-trends-fleet` | OT-P1-002, OT-P1-003, OT-P1-004 — budgets + baseline-diff gating, additive trend store, deterministic fleet offenders |
| `07-startup` | OT-P2-001 — resource-aware startup benchmark (axis ②, migrated from structure-health) |
| `08-integration-and-cli` | OT-P0-002, OT-P0-003, OT-P1-004, OT-P1-005 — thin human-default CLI + discoverability + the operator UI |

## Auto-Sync Behavior
1. Operational targets in `PRD.md` map to the modules above; `index.json` imports each `module.json`.
2. Tests tagged with `[REQ:<ID>]` link to their requirement; after a test phase runs,
   Test Genie auto-syncs each validation's live status (`passed`/`failed`/`not_run`) back
   into these files and writes snapshots under `coverage/requirements-sync/`.
3. Coverage summaries land in `coverage/phase-results/` after each phase.

## Validation Commands
```bash
# Validate the requirements registry (structure + PRD-target linkage)
prd-control-tower requirements validate performance-health

# Validate the PRD itself
prd-control-tower prd validate performance-health

# Run the full suite (auto-syncs requirement statuses from [REQ:ID]-tagged tests)
vrooli scenario test performance-health
```

## Contributor Notes
- Keep every requirement's `prd_ref` pointing at a real `OT-P*-NNN` target in `PRD.md`.
- Keep every `validation.ref` pointing at a real (or planned-to-exist) file path — no phantom refs.
- Tag tests with `[REQ:<ID>]` so auto-sync can update status; never hand-edit a synced status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations — let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. See `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details.
