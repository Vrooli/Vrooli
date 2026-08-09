# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

This registry has been generated from `PRD.md`: there is one numbered module per operational target, each linked back to its target via `prd_ref`. The bridge already has 20 modules; the identity/delegation plan changes the authorization input of module 04 without changing the typed-job boundary.

## Operational Targets
Every module's `prd_ref` points at the operational target it covers (e.g. `"prd_ref": "OT-P0-001"`); criticality (P0/P1/P2) is derived from that id. All 18 PRD targets are linked to at least one requirement. Each requirement carries a `validation` array describing the planned unit / integration / business / performance strategy and the test `ref` where it will live. Requirements start `planned` because this is the documentation-first foundation — no product code exists yet.

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## One-owner delegation reconciliation

Modules 01–03, 05–08, and 10–20 describe shipped or existing bridge
capabilities. Module 04 is built by this plan: its manifest is derived from
CLI governance and intersected with registry-owned node execution scopes;
presence-only is the empty-scope state and `vrooli-bridge:session` is reserved
for a future PTY protocol. G8 remains open because the live onboarding path has
not installed the privileged helper as a separate operating-system principal.

## Validation
Run these from the scenario directory:
- `vrooli scenario requirements validate vrooli-bridge --json` — validate the registry (target linkage, requirement→target coverage, PRD ref integrity).
- `vrooli scenario requirements validate vrooli-bridge --json` — validate `PRD.md` against the template + quality standards.
- `make test` (or `vrooli scenario test vrooli-bridge`) — run the test suite; the test-genie business phase auto-syncs requirement `status` from `[REQ:ID]`-tagged tests.
- `make orient` — machine-readable initialization-gate progress, including the requirements-registry gate.

Auto-sync: tag tests with `[REQ:ID]` (e.g. `[REQ:BRG-P0-001]`) so the requirements-sync step can flip `planned` → `passing`/`failing`. See `scenarios/test-genie/docs/phases/business/requirements-sync.md`.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
