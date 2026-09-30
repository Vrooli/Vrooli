# Redesign ledger — Nooch

The working record of the redesign goal: where the build stands, every finding
from every hostile review pass, and the evidence behind each claim. It replaces
Plan Manager for this effort (D-026). Read it at the start of every slice and
pass; update it at the end of every slice and pass.

- **Plan and definition of done:** [`REDESIGN_PLAN.md`](REDESIGN_PLAN.md)
- **Operator feedback:** [`OPERATOR_FEEDBACK.md`](OPERATOR_FEEDBACK.md)
- **Visual target:** [`../reference/mockups/README.md`](../reference/mockups/README.md)

## How to use this file

1. **Status board** — update milestone and surface rows as work lands. A row
   moves only with evidence in the evidence index.
2. **Findings** — every defect, gap, or quality problem gets an `F-nnn` id, the
   pass or slice that found it, and a status. Resolve with a receipt (what
   changed, and the command output, test, or capture that proves it). Never
   delete a finding; mark it `resolved` or `wont-fix` with the operator's recorded
   decision.
3. **Pass log** — each fresh hostile review pass gets an entry with what it
   judged and its material findings. Done requires **two consecutive entries with
   zero material findings** and no open operator feedback.
4. **Checkpoint** at the end of every slice: changed / verified / remaining /
   unverified.

## Status board

### Milestones ([`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §5)

| Milestone | Status | Exit proof |
| --- | --- | --- |
| D0 Foundation repair and inventory | not started | — |
| D1 Visual foundation + Today fidelity proof | not started | — |
| D2 Collection and recipe | not started | — |
| D3 Today and Week | not started | — |
| D4 Groceries, Kitchen, onboarding | not started | — |
| D5 Explore and cooking depth | not started | — |
| D6 Production artwork | not started | — |
| D7 Adapters | not started | — |
| D8 Migration, portability, release review | not started | — |

### Surface fidelity (captures versus mockups)

Verdicts: `not built` · `built, not compared` · `gaps open` (list F-ids) ·
`matches` (both appearances, phone and desktop, nonideal states). The desktop and
phone columns are the two compositions; record intermediate-width (768, 1024,
320 px, 200 % zoom) verdicts in the finding or checkpoint that covers them, and
judge pairs without a mockup by the derivation rule in the mockups guide.

| Surface | Mockups | Light desktop | Light phone | Evening desktop | Evening phone | Nonideal states |
| --- | --- | --- | --- | --- | --- | --- |
| Shell and navigation | all | not built | not built | not built | not built | — |
| Today (scene) | `today-sunroom-light.png`, `today-evening-kitchen-dark.png` | not built | not built | not built | not built | — |
| Today (editorial, minimal) | `today-editorial-light.png` | not built | not built | not built | not built | not built |
| Week | `week-light.png`, `week-evening.png` | not built | not built | not built | not built | not built |
| Meals | `meals-light.png`, `meals-evening.png` | not built | not built | not built | not built | not built |
| Explore | `explore-light.png` | not built | not built | not built | not built | not built |
| Recipe detail (Reading) | `recipe-detail-light.png` | not built | not built | not built | not built | not built |
| Recipe map | (no mockup; At a glance entry in `recipe-detail-light.png`) | not built | not built | not built | not built | not built |
| Cooking | `cooking-evening.png` | not built | not built | not built | not built | not built |
| Groceries (Review, Shop) | `groceries-light.png`, `groceries-evening.png` | not built | not built | not built | not built | not built |
| Kitchen On hand | `kitchen-on-hand-light.png`, `kitchen-on-hand-evening.png` | not built | not built | not built | not built | not built |
| Kitchen Equipment | `kitchen-equipment-light.png` | not built | not built | not built | not built | not built |
| Kitchen Preferences | (no mockup) | not built | not built | not built | not built | not built |
| Onboarding | (no mockup) | not built | not built | not built | not built | not built |
| Settings | (no mockup) | not built | not built | not built | not built | not built |
| Long editor (recipe editor) | (no mockup) | not built | not built | not built | not built | not built |

### Convergence

| Consecutive empty hostile passes | Open findings | Open operator feedback | Image generation spend (cap US$15, D-043) |
| --- | --- | --- | --- |
| 0 | 16 | 1 (OF-003) | $0.00 |

## Findings

Seeded from the 2026-09-22 audit; each blocker's detail and fix direction is in
[`REDESIGN_PLAN.md`](REDESIGN_PLAN.md) §3.2.

| ID | Found by | Finding | Status | Resolution and evidence |
| --- | --- | --- | --- | --- |
| F-001 | audit 2026-09-22 | B1 — no principal in the local runtime; every workspace RPC returns 401 | open | |
| F-002 | audit 2026-09-22 | B2 — `profile.Get` returns raw no-rows; fresh workspaces hit Internal errors | open | |
| F-003 | audit 2026-09-22 | B3 — swap and feedback unmarshal an empty plan before any save | open | |
| F-004 | audit 2026-09-22 | B4 — one whole-plan JSON row per workspace; Today and Week overwrite each other | open | |
| F-005 | audit 2026-09-22 | B5 — no plan read model; pages regenerate a draft on every mount | open | |
| F-006 | audit 2026-09-22 | B6 — Settings data health calls the SPA path instead of `/api/v1/diagnostics` | open | |
| F-007 | audit 2026-09-22 | B7 — planning ignores the profile's excluded groups | open | |
| F-008 | audit 2026-09-22 | B8 — recipes cannot be edited, so evidence and groups are always missing | open | |
| F-009 | audit 2026-09-22 | B9 — profile scan collapses identical JSON lists | open | |
| F-010 | audit 2026-09-22 | B10 — UTC date computation shifts local days | open | |
| F-011 | audit 2026-09-22 | B11 — random idempotency key per `ensureWorkspace()` call | open | |
| F-012 | audit 2026-09-22 | B12 — non-transactional recipe writes | open | |
| F-013 | audit 2026-09-22 | B13 — no versioned migrations | open | |
| F-014 | audit 2026-09-22 | B14 — PDF export depends on a host font | open | |
| F-016 | docs session 2026-09-22 | Test Genie `docs` phase still fails on residual debt: 3 `content_issue` (manifest self-flagging its own forbidden-placeholder token and requiring headings for the removed `notes` example — both fixed in the manifest 2026-09-22; the generated `api-endpoints.md`/`cli-commands.md` still show a `<domain>` placeholder), 1 `placeholder_style` in generated `cli-commands.md` (quoted by hand 2026-09-22; regeneration may reintroduce it), 1 `broken_external_link` (every nih.gov URL answers 403 to this host, so the valid NIH ODS link in Appendix A §6.6 cannot be verified here), ~280 `unmarked_number` style warnings | open | Regenerate the reference docs from the real API/CLI during D0 and keep their placeholders quoted; decide the NIH link and the unmarked-number lint with evidence. Runs `20260922-210930-c1264609`, `20260922-211925-5ed75a24`. |
| F-015 | docs session 2026-09-22 | Requirements auto-sync promoted 93 ref-less requirements to `complete` from a phase-level pass (test-genie defect); sync disabled per module (D-037) | open | Reverted to `planned` 2026-09-22. Close when every module whose validations carry real `[REQ:<ID>]` refs has `auto_sync_enabled: true` again and a run shows only referenced requirements promoted. |

## Pass log

Copy this template for each fresh hostile review pass.

```markdown
### Pass N — YYYY-MM-DD

- Judged: fidelity (surfaces/appearances/viewports captured), behaviour (journeys and cases run), honesty, code maturity, accessibility, performance, docs.
- Commands and captures: <commands with run ids; capture locations>
- Material findings: F-nnn, F-nnn … (or "none")
- Consecutive empty passes after this pass: N
```

_No passes yet._

## Checkpoints

Append one entry per completed slice.

```markdown
### YYYY-MM-DD — <slice>

- Changed: …
- Verified: <command → result>
- Remaining: …
- Unverified: …
```

_No checkpoints yet._

## Evidence index

Commands, run ids, capture sets, and asset-manifest validations referenced by the
status board and findings.

| Date | Evidence | Result |
| --- | --- | --- |
| 2026-09-22 | `business-health validate scenario nutrition-planner` after the documentation update | `No business-contract findings`. |
| 2026-09-22 | `vrooli scenario requirements validate nutrition-planner --json` | `PASSED`; 161 requirements, all `planned`; traceability L0 (no evidence yet). |
| 2026-09-22 | `experience-manager spec validate nutrition-planner --json` | `PASSED`, 0 findings; 10 draft pages, 3 deprecated tombstones, 9 draft journeys. |
| 2026-09-22 | Test Genie run `20260922-205742-7edb0e18` (comprehensive, 27 phases; launcher not identified) | Failed on pre-existing code debt (portability, contracts, ui-health, dependencies, docs, workflow, security, skill-set). Its requirements sync produced the false promotions in F-015. |
| 2026-09-22 | Test Genie run `20260922-210637-adf9f222` (`docs,business,experience`) | business passed, experience passed, docs failed on 11 broken template command snippets, since fixed. |
| 2026-09-22 | Test Genie run `20260922-210930-c1264609` (`docs`) | Reference integrity L3; residual findings recorded as F-016. |
| 2026-09-22 | Test Genie run `20260922-211925-5ed75a24` (`docs`) after the cold-read fixes | Content issues cleared; remaining blockers were one generated-doc placeholder (then quoted) and the host-blocked NIH link; F-016. |

## Final report (R30)

Filled in at done: (1) what changed and why, by user journey; (2) capabilities
preserved and migrations performed; (3) requirement and acceptance status with
commands and results; (4) desktop and phone screenshots in both appearances and
nonideal media states; (5) assets created, provenance, optimization, remaining art
gaps; (6) adapters tested against real services versus simulated fixtures; (7) how
to run, seed an isolated demo, configure optional providers, and verify; (8) known
limitations and exact unresolved blockers.
