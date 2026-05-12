# Requirements Registry

Each subfolder corresponds to one operational target in [`../PRD.md`](../PRD.md). Numbers preserve ordering but do not imply priority; priority is recorded per requirement via `criticality`.

## Lifecycle
1. Operational targets in PRD map to folders here (1:1).
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Validation commands
- Validate PRD shape and linkage: `prd-control-tower prd validate react-component-library`
- Validate this registry's structure: `prd-control-tower requirements validate react-component-library`
- AI-fix common violations: `prd-control-tower prd fix react-component-library --auto`
- Run the scenario test suite: `vrooli scenario test react-component-library`

## Auto-sync
Tests tagged `[REQ:<id>]` (for example `[REQ:CR-001]`) update the matching requirement's `validation[].status` automatically when the test phase completes. See `scenarios/test-genie/docs/phases/business/requirements-sync.md` for the full contract.

## Contributor notes
- Add a new folder + entry in `index.json` for every new P0/P1 operational target you add to the PRD.
- Never add compatibility shims (duplicate folders, alias imports). Let things fail temporarily instead of adding debt.
- Keep this README under 100 lines.
