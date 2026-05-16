### Scenarios reviewed
Reviewed 3 queued scenarios with fresh GCT runs:
- `ecosystem-manager`: yellow, job `9d36d806-b28d-48ec-a6d0-47642e8ae5cf`
- `workspace-sandbox`: yellow, job `f2cddfb4-d920-47d1-bcc1-f493caf4b82d`
- `notification-hub`: yellow, job `65bd8ba3-9bf2-49b7-9d5e-e4e97f49ff3b`

### Findings converted to backlog
Created 7 Swarm Manager backlog items with `notes.md` evidence:
- `fix/qa-ecosystem-manager-standards-structure-focus-20260516`
- `fix/qa-ecosystem-manager-tests-playbooks-business-20260516`
- `fix/qa-workspace-sandbox-standards-preflight-structure-20260516`
- `fix/qa-workspace-sandbox-visual-baseline-20260516`
- `fix/qa-notification-hub-standards-structure-api-base-20260516`
- `fix/qa-notification-hub-tests-docs-smoke-playbooks-business-20260516`
- `fix/qa-notification-hub-visual-baseline-20260516`

No `ecosystem-manager` visual item: visual had `screenshotCount=1`, non-stale latest capture.

No separate `workspace-sandbox` tests item: the only failing test phase was standards, covered by the standards/preflight item.

### Dependencies wired
Wired:
- 2 `ecosystem-manager` QA gates onto 7 active related items.
- 2 `workspace-sandbox` QA gates onto 4 active related items.
- 3 `notification-hub` QA gates onto 4 active related items.

### Skipped scenarios
None. All three queued scenarios had visible files through `swarm-manager scenarios files`.

### Bugs filed (via report-bug)
None.

### Knowledge entries written
- `qa-run/ecosystem-manager`: `knw-1778926108888279930`
- `qa-run/workspace-sandbox`: `knw-1778926108889142468`
- `qa-run/notification-hub`: `knw-1778926109047263996`
- `reviewed-scenario/ecosystem-manager`: `knw-1778926125060942705`
- `reviewed-scenario/workspace-sandbox`: `knw-1778926125062154263`
- `reviewed-scenario/notification-hub`: `knw-1778926125061932423`
- `dependency-wiring/2026-05-16-gct-readiness-queued-scenarios-2`: `knw-1778926138461450293`