# Swarm Manager Requirements Registry

This folder tracks the canonical requirements that back the PRD. Each requirement maps to an Operational Target from `PRD.md` via `prd_ref`.

## Layout
- `index.json` – single registry file containing all requirements. Each entry maps to an Operational Target from `PRD.md` via `prd_ref`.
- `README.md` – this guide.

## Editing Workflow
1. Open `requirements/index.json` and add/update requirement objects.
2. Use stable IDs (`SWM-REQ-###`). Update the `prd_ref` to match the relevant OT line.
3. Update `status` field as implementation progresses: `planned` → `pending` → `validated`.
4. When adding tests, annotate assertions with `[REQ:SWM-REQ-###]` so coverage reporting can tie results back.
5. Keep requirements focused and actionable—combine smaller behaviors when possible to avoid noise.

## Validation & Reporting
- Run `make test` (or `vrooli scenario test swarm-manager`) after modifying requirements.
- Requirements link to operational targets and can be validated through testing once implementation begins.
- All P0 requirements (OT-P0-*) must be validated before the scenario is considered minimally viable.
