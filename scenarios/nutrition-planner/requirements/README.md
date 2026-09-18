# Requirements Registry

This directory is the machine-checkable bridge between the operational
targets in `PRD.md` and the proof that Daily (nutrition-planner) behaves
as intended. Each numbered folder is one module; each module's
`module.json` holds requirements whose `prd_ref` points at exactly one
operational target. Every operational target in the PRD is covered here,
including P1 and P2 targets, because an uncovered target is treated as a
gap even when it is deliberately deferred.

## Modules

| Module | Operational targets |
| --- | --- |
| `01-workspace-and-profile` | OT-P0-001, OT-P0-002 |
| `02-meal-capture` | OT-P0-003 |
| `03-recipe-revisions-and-components` | OT-P0-004 |
| `04-recipe-representations` | OT-P0-005 |
| `05-eligibility-and-rules` | OT-P0-006 |
| `06-planning-engine` | OT-P0-007, OT-P0-009 |
| `07-today-and-swap` | OT-P0-008 |
| `08-nutrition-and-targets` | OT-P0-010 |
| `09-cost-and-pricing` | OT-P0-012 |
| `10-grocery-and-inventory` | OT-P0-011 |
| `11-portability-and-printing` | OT-P0-013 |
| `12-persistence-and-concurrency` | OT-P0-014 |
| `13-experience-and-accessibility` | OT-P0-015 |
| `14-ai-assistance` | OT-P1-001, OT-P1-003 |
| `15-provider-adapters` | OT-P1-002 |
| `16-jobs-and-recurring` | OT-P1-004, OT-P1-005 |
| `17-feedback-and-preferences` | OT-P1-006 |
| `18-commercial-foundations` | OT-P2-001, OT-P2-002 |
| `19-deferred-expansion` | OT-P2-003, OT-P2-004, OT-P2-005 |

## Lifecycle

1. operational target entries in `PRD.md` map to folders and requirements
   here; the PRD is authoritative when the two disagree.
2. `requirements/index.json` imports every module; tests auto-sync their
   status when they run, so status is earned rather than asserted.
3. `validation` entries name the layers that will prove a requirement —
   unit, integration, business, performance, structure, dependencies, or
   cli. Nothing is implemented yet, so every entry is `planned` and points
   at a described layer instead of a file that does not exist.
4. Coverage summaries live in `coverage/phase-results/` after each test
   phase.

## Scenario Notes

- Nutrition-planner is entirely unimplemented; all requirements are
  `planned` and criticality matches the operational target tier (P0, P1,
  P2). Do not mark anything complete until evidence lands.
- The product specification (`docs/reference/product-specification.md`)
  is the source of substance. Requirement descriptions cite its embedded
  fixtures (FIX-01 … FIX-11) and acceptance matrix (ACT-001 … ACT-063)
  as human breadcrumbs; those IDs are not schema references.
- Missing values are not zero, planned is not eaten, and required rules
  cannot be traded against preferences. Requirements are written so those
  invariants stay testable as code arrives.
- `18-commercial-foundations` and `19-deferred-expansion` describe
  deliberately deferred capabilities; keep them scoped to P2 and do not
  treat their presence as a current delivery commitment beyond the PRD.
- Keep this README under 100 lines. Use
  `scenarios/test-genie/docs/reference/requirement-schema.md` for schema
  details and `scenarios/test-genie/docs/phases/business/requirements-sync.md`
  for auto-sync behavior.
