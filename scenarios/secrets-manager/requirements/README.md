# Secrets Manager Requirements Registry

This folder tracks the canonical requirements that back the minimal PRD. Keep the list focused, organized, and easy to audit.

## Layout
- `index.json` – imports the authoritative requirement modules.
- `01-operational-targets/module.json` – evidence-backed claims mapped directly to the `OT-P*-###` targets in `PRD.md`.

An operational target is the PRD outcome that a requirement proves; every P0/P1 target must link to at least one focused requirement.

Auto-sync records live `[REQ:ID]` test evidence after a comprehensive Test Genie run. It is evidence capture, not permission to hand-edit requirement status.
- `README.md` – this guide. Add additional docs here if we later split into modules.

## Editing Workflow
1. Add or update a focused requirement in the relevant imported module, using a stable `SEC-<DOMAIN>-###` ID and an exact `OT-P*-###` `prd_ref`.
2. For P0/P1 requirements, cite a test that actually proves the claim; P2 requirements may remain planned without validation until work begins.
3. When adding tests, annotate them with `[REQ:SEC-XXX-###]` so the full Test Genie run can record live evidence.
4. `auto_sync_enabled` lets Test Genie update its evidence snapshot after a comprehensive successful suite; do not hand-edit that snapshot or mark a requirement complete without passing evidence.
5. Keep the count focused—combine smaller behaviors when they form one falsifiable claim.

## Validation & Reporting
- Run `make test` (or `vrooli scenario test secrets-manager`) after modifying requirements so phased tests remain in sync.
- Scenario-wide reports live under `test/` artifacts; add new reporting hooks there if you introduce additional requirement modules.
- The lifecycle setup already enforces schema + seed scripts. If you introduce new requirement types (e.g., latency, load), describe the expected test phase here before implementing.
