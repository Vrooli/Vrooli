### Scenarios reviewed
Reviewed 2 scenarios with fresh GCT runs:
- `git-control-tower`: yellow, job `6bbb52ab-0555-4f4b-a3f8-f5e78870cfc2`
- `deployment-manager`: red, job `a01fe0cf-1bd2-4002-b1d4-8f5130577caa`

### Findings converted to backlog
Created 3 Swarm Manager backlog items with `notes.md` evidence:
- `fix/qa-git-control-tower-tests-playbook-schema-20260515`
- `fix/qa-deployment-manager-standards-focus-visible-20260515`
- `fix/qa-deployment-manager-visual-baseline-20260515`

Did not create a new deployment-manager test item: GCT reported `checks.tests=failed` but tests totals were all zero, matching the already-investigated GCT reporting defect.

### Dependencies wired
Wired:
- `fix/qa-git-control-tower-tests-playbook-schema-20260515` onto 23 active non-terminal `git-control-tower` backlog items.
- Both deployment-manager QA gates onto 19 active non-terminal `deployment-manager` backlog items.

### Skipped scenarios
- `vrooli-emulator`: review queue returned it, but `scenarios/vrooli-emulator/` is missing. No GCT run and no backlog item created.

### Bugs filed (via report-bug)
None this run. The deployment-manager zero-count failed-test symptom matched existing GCT reporting investigations.

### Knowledge entries written
- `qa-run/git-control-tower`: `knw-1778818118902304618`
- `qa-run/deployment-manager`: `knw-1778818119501619436`
- `reviewed-scenario/git-control-tower`: `knw-1778818120077729033`
- `reviewed-scenario/deployment-manager`: `knw-1778818120674205997`
- `reviewed-scenario/vrooli-emulator`: `knw-1778818152405584659`
- `dependency-wiring/2026-05-15-gct-readiness`: `knw-1778818152991494576`