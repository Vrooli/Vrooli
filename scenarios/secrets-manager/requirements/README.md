# Secrets Manager Requirements Registry

This folder tracks the canonical requirements that back the minimal PRD. Keep the list focused, organized, and easy to audit.

## Layout
- `index.json` – single registry file (no submodules yet). Each entry maps to an Operational Target from `PRD.md` via `prd_ref`.
- `README.md` – this guide. Add additional docs here if we later split into modules.

## Editing Workflow
1. Open `requirements/index.json` and add/update requirement objects.
2. Use stable IDs (`SEC-<DOMAIN>-###`). Update the `prd_ref` to match the relevant OT line.
3. Leave the `validation` array empty until a specific test or script references `[REQ:ID]` in code/comments.
4. When adding tests, annotate assertions with `[REQ:SEC-XXX-###]` so coverage reporting can tie results back.
5. Keep the count to a few dozen focused requirements—combine smaller behaviours when possible to avoid noise.

## Validation & Reporting
- Run `make test` (or `vrooli scenario test secrets-manager`) after modifying requirements so phased tests remain in sync.
- Scenario-wide reports live under `test/` artifacts; add new reporting hooks there if you introduce additional requirement modules.
- The lifecycle setup already enforces schema + seed scripts. If you introduce new requirement types (e.g., latency, load), describe the expected test phase here before implementing.
