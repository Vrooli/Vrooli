# Requirements Registry

Requirement modules for **meta-optimization-manager**, organized one folder per PRD operational target. Numbers preserve ordering, not priority.

## Operational target → module map

| OT | Priority | Module |
|----|----------|--------|
| OT-P0-001 Readiness snapshot | P0 | `01-readiness-snapshot/` |
| OT-P0-002 Actionable focus | P0 | `02-focus/` |
| OT-P0-003 Honest gaps registry | P0 | `03-gaps-registry/` |
| OT-P0-004 Base-document integrity | P0 | `04-base-doc-integrity/` |
| OT-P1-001 Empirical readiness trials | P1 | `05-trials/` |
| OT-P1-002 Template & reference convergence | P1 | `06-convergence/` |
| OT-P2-001 Operator UI dashboard | P2 | `07-ui-dashboard/` |
| OT-P2-002 Attested readiness as a search answer | P2 | `08-attested-search/` |

Each requirement sets `prd_ref` to its `OT-P[012]-NNN` id; criticality is derived from the id.

## Lifecycle & auto-sync
1. PRD operational targets map to the folders above.
2. `requirements/index.json` imports each module.
3. Tests tagged `[REQ:ID]` auto-sync requirement status when they run; sync metadata lands in `coverage/sync/*.json` (gitignored) and per-phase summaries in `coverage/phase-results/`.

## Validation commands
- Registry schema: `vrooli scenario requirements validate meta-optimization-manager`
- PRD ↔ requirement linkage: `prd-control-tower requirements validate meta-optimization-manager --json`
- PRD itself: `prd-control-tower prd validate meta-optimization-manager --json`

## Contributor notes
- One folder per PRD target; do not reuse other scenarios' module names.
- P0/P1 requirements need >= 2 automated validation layers (manual validations are excluded from the diversity requirement).
- Tag tests `[REQ:ID]` so auto-sync can update status; never add compatibility shims during migrations.
- Schema details: `scenarios/test-genie/docs/reference/requirement-schema.md`.
