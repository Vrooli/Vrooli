# Requirements Registry

Requirement modules are organized by PRD operational target, keeping the filesystem
structure aligned with the "what" in `PRD.md`. Folder numbers preserve ordering but do
**not** imply priority.

## Operational target → module map

| OT | Priority | Module |
|----|----------|--------|
| OT-P0-001 Structured plan store | P0 | `01-plan-store/` |
| OT-P0-002 Guided authoring wizard | P0 | `02-authoring-wizard/` |
| OT-P0-003 Guided execution + context injection | P0 | `03-execution-context/` |
| OT-P0-004 Code-reference tracking + staleness | P0 | `04-reference-staleness/` |
| OT-P0-005 Baseline-aware validation orchestration | P0 | `05-validation-orchestration/` |
| OT-P1-001 Structured handoff finalizer | P1 | `06-handoff-finalizer/` |
| OT-P1-002 Velocity & plan graph | P1 | `07-velocity-graph/` |
| OT-P2-001 Operator UI console | P2 | `08-ui-console/` |
| OT-P2-002 Consumer inversion | P2 | `09-consumer-inversion/` |

## Lifecycle

1. Operational targets in `PRD.md` map to the folders above; each `module.json`
   requirement carries `prd_ref` set to its `OT-…` id.
2. `requirements/index.json` imports each module.
3. Tests tag `[REQ:ID]` (e.g. `[REQ:PM-STORE-001]`) so auto-sync updates each
   requirement's `status` when the suite runs.
4. Coverage summaries land in `coverage/phase-results/` after each test phase.

## Validation commands

```bash
prd-control-tower prd validate plan-manager --json
prd-control-tower requirements validate plan-manager --json
```

`auto_sync_enabled` in `index.json` lets Test Genie flip requirement status from
`planned` → `passing`/`failing` as tagged tests run; do not hand-edit status.

## Contributor Notes

- Add folders/modules that match real PRD targets; do not reuse other scenarios' names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations —
  let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. See `scenarios/test-genie/docs/reference/requirement-schema.md`
  for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md`
  for auto-sync behavior.
