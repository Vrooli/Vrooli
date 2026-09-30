# Requirements Registry

This directory is the machine-checkable bridge between the operational
targets in `PRD.md` and the proof that Nooch (nutrition-planner) behaves
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
| `20-week-board-and-agenda` | OT-P0-009 |
| `21-shell-and-appearance` | OT-P0-016 |
| `22-meal-artwork-and-media` | OT-P0-017 |
| `23-explore` | OT-P0-018 |
| `24-cooking-sessions` | OT-P0-019 |
| `25-kitchen-inventory-and-equipment` | OT-P0-020 |
| `26-visual-fidelity` | OT-P0-021 |
| `27-image-generation` | OT-P1-007 |
| `28-calendar-integration` | OT-P1-008 |
| `29-scene-studio` | OT-P2-006 |

Modules 20–29 and the added IDs in 01–13 were written on 2026-09-22 for
the v2.0 redesign (decision D-025).

## Lifecycle

1. Operational targets in `PRD.md` map to folders and requirements here;
   the PRD is authoritative when the two disagree.
2. `requirements/index.json` imports every module. Auto-sync is **off**
   (`auto_sync_enabled: false` in the index and every module, decision
   D-037) because Test Genie currently promotes validations that have no
   `ref` from a phase-level pass. When **every** validation in a module points at a
   `[REQ:<ID>]`-tagged test or an attested manual record, set its
   `auto_sync_enabled` back to `true` in the same change so status is
   earned rather than asserted.
3. `validation` entries name the layer that will prove a requirement —
   unit, integration, business, or performance — and the fixture or
   acceptance case it exercises. Entries stay `planned` and carry no
   `ref` until a real test or evidence file exists.
4. Coverage summaries live in `coverage/phase-results/` after each test
   phase.

## Scenario Notes

- **Honest state (2026-09-22 audit):** code exists across the API, CLI,
  and UI, but no requirement is proven. The local runtime rejects every
  request as unauthenticated, the database has never held a row, and no
  test carries a `[REQ:<ID>]` tag. Every requirement is `planned` until
  tagged tests sync. See `docs/internal/REDESIGN_PLAN.md` §3.
- The product specification (`docs/reference/product-specification.md`)
  is the source of substance: redesign sections R01–R30 and the retained
  domain specification in Appendix A. Descriptions cite fixtures
  (FIX-01 … FIX-11), redesign acceptance cases (AT-001 … AT-052), the
  baseline matrix (ACT-001 … ACT-063), and blockers (B1 … B14 in the
  redesign plan) as human breadcrumbs; those IDs are not schema
  references.
- Missing values are not zero, planned is not eaten, checked is not
  purchased, a viewed step is not a completed step, and required rules
  cannot be traded against preferences. Requirements are written so those
  invariants stay testable.
- `26-visual-fidelity` is proven by captures compared with the concept
  mockups in `docs/reference/mockups/`; its verdicts live in
  `docs/internal/REDESIGN_LEDGER.md`.
- `18-commercial-foundations`, `19-deferred-expansion`, and
  `29-scene-studio` describe deliberately deferred capabilities; keep
  them scoped to P2.
- Keep this README under 100 lines. Use
  `scenarios/test-genie/docs/reference/requirement-schema.md` for schema
  details and `scenarios/test-genie/docs/phases/business/requirements-sync.md`
  for auto-sync behavior.
