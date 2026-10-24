# E10 — Phone-operable History outcome filters

- Outcome: make the History outcome filters usable on a 390px phone while retaining the bounded local filtering model and all distinct History assertions. E9's two History cases now reach a hidden-only filter nav at mobile width; the visible Settings theme route passes for both Delivery cases.
- Kind: fix.
- Status: accepted as a bounded E10 slice; task `b1c34bad-4e1d-47b0-b7a4-024e766507ca`; run `8bc7956e-5fa0-422d-b759-c3aa35be05b7` COMPLETE at `2026-10-21T00:32:29Z`. Parent acceptance is in `evidence/E10-history-phone-filters/parent-review.md`. This does not close E4 or any whole-redesign benchmark gate.
- Parent: `11175250-c50d-44e9-9af5-cd6eba79bab2` (verify exact current identity before dispatch).
- Worker card: `scenarios/prompt-manager/store/skills/packs/core/large-effort-orchestration/references/worker.md` (read before execution).
- Estimate: 3–5 meaningful units; report actual work, waits and interventions.
- Work shape: one History phone-operability correction, directly affected page regression, and existing registered four-case Delivery/History self-validation. No API/data-path change.
- Source authority: `docs/internal/delivery-reliability-research/follow-through/BENCHMARK.md`; selected dossier `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-dossier.md`, heading `Behavior and migration / Lists and history with honest limits`; atlas `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-atlas.md`, heading `Experience / History and informational updates`; `scenarios/notification-hub/docs/internal/goal/evidence/commissioning-source-map.md`, NH16.2 row and NH-AC-26/37. Admission source hashes: BENCHMARK `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`; dossier `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`; atlas `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`; selected manifest `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef`.

## Selected references and diagnosed failure

- Revised selected History list: `NH16.2-light-phone-p1-r3.png` SHA-256 `9970ebfb97c4db28d9cbfca334e4de27fa7fa54ac55418a8636694c6e0eb4aa2`; `NH16.2-dark-phone-p1-r3.png` SHA-256 `5cb51426dbea45de4c6e3d9895016f377ec9653b0418375887490acdfd27cfec`. CSS viewport 390×844, DPR 2. Preserve the clear “Latest 200 loaded” bound and distinct returned History rows.
- Mobile behavior reference: dossier/atlas `NH06 Browse decision history — Dark mobile`, `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-retained/originals/NH06-dark-mobile-v05.png` SHA-256 `ff33537f2ca5faba9fe2a257feac8a5fc7e749e64cfa74443464615c0b61ea3d`; light companion `NH06-light-mobile-v07.png` SHA-256 `3d49f2d1850fef70fe1c5c13e1c991be4dc6c239dd57610c2efba53a80609bdd`. This behavior reference shows visible loaded-set filters; it does not replace the revised NH16.2 composition.
- Independent parent inspection of current `ui/src/features/asks/HistoryPage.tsx` found the filter `<nav>` uses `hidden sm:flex`; the registered `history.filterLink(state=defaulted)` cannot be operated at 390px. Test Genie `20261021-000654-f5cb52bf` and Workflow Health `c20fee57-66c0-4f10-b0da-106d2bd105c3` both show the Answered fixture row remains after the defaulted filter step in `history-bounded` and `history-dark`. This is a mobile product operability gap, not a reason to drop the two assertions.

## Scope

Allowed authored paths:

- `scenarios/notification-hub/ui/src/features/asks/HistoryPage.tsx`
- `scenarios/notification-hub/ui/src/features/asks/HistoryPage.test.tsx`
- `scenarios/notification-hub/bas/cases/e4/history-bounded.json`
- `scenarios/notification-hub/bas/cases/e4/history-dark.json`
- Evidence only under `scenarios/notification-hub/docs/internal/goal/evidence/E10-history-phone-filters/`.

Use the existing History page, selector registry, query-string state and local 200-row result set. Make each outcome filter visible and keyboard/pointer-operable at phone width; retain desktop behavior and `preserveRoutedTestMode` routing. Preserve distinct answered/defaulted/expired values, the filtered-empty versus no-history copy, search scope, the 200-row disclosure, and all unique BAS assertions and routed-isolation labels. Consolidate only a genuinely superseded hidden-only responsive behavior. Do not alter the API, storage, selector registry, BAS runtime/provider, Test Genie, Workflow Health, shared shell, locales/generated strings, Delivery cases, or unrelated pages. Preserve all existing dirty work and accepted E5/E8 changes.

Before any write, inspect these exact paths and managed build; create and record a fresh `git-control-tower proposal anchor --effort effort:notification-hub-redesign --epoch E10 --scope 'scenarios/notification-hub/**' --json`. Admission anchor `gcta-a5e9ba575aaf`, HEAD `52097258290d1bff0191319242d4948e214af9e3`; it inventories existing dirty state and is not an authorship claim.

## Validation and handoff

- Run focused History page behavior/accessibility tests at mobile and wide viewports; preserve and expand the existing local filter test rather than adding a duplicate suite.
- Run one changed-input `Test Genie workflow` phase after the History source/case inputs change. Do not repeat E9's unchanged API Go `unit` phase: its retained artifact has no stdout/stderr and its failure remains unattributed. Keep any new unrelated infrastructure refusal exact; do not retry unchanged phases.
- Run Workflow Health against the existing four cases `delivery-dark`, `delivery-detail`, `history-bounded`, and `history-dark` on one identified current build. The two History cases must exercise the visible defaulted filter. Delivery cases remain unchanged and retain their passing baseline. Record exact case outcomes, lease/isolation counters and cleanup receipt; no manual lease clear or provider restart.
- Capture History at the selected phone viewport in light and dark; compare with the exact NH16.2 originals and explain the existing shared app-shell/navigation adaptation separately from page content.
- Return path attribution, superseded responsive behavior, commands/receipts, actual work/waits/interventions, case assertion preservation, screenshots/hashes/build, isolation/cleanup evidence, and independent parent-review handoff. Do not self-accept.

## Exit gates

- G1 test: focused History tests pass; changed-input Test Genie workflow passes; Workflow Health passes all four existing cases with their unique assertions intact and a valid routed lease.
- G2 custom: NH16.2 establishes the bounded History list but omits visible filters; this slice judges the selected-page behavior adaptation supported by NH06 while preserving NH16.2's loaded-set meaning, with full visual fidelity remaining open.
- G3 review: independent parent review finds no material privacy, route-boundary, bounded-data, keyboard/touch, unique-test-protection or maintainability issue.

## Slice log

2026-10-21T00:12:30Z | E9 returned. G1 failed on an unattributed API execution failure and two History mobile filtered-empty assertions; Delivery cases passed. Independent G2/G3 review passed for E9 only. Parent inspection traced both History failures to the hidden phone filter nav (`hidden sm:flex`). Select this small original-scope product correction now; keep E9 unaccepted and do not rerun its unchanged API/unit phase. | exit metric=E10 ready for admission; History phone filters selected | ack=HUB-SUP-AUDIT-SHELL-BEFORE-VISUAL-CERT-20261020,HUB-SUP-AUDIT-NH16-1-VISUAL-FAIL-20261020,HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020
2026-10-21T00:17:27Z | Admitted Luna-medium interactive E10 under effort revision16 and verified original parent. Task `b1c34bad-4e1d-47b0-b7a4-024e766507ca`; run `8bc7956e-5fa0-422d-b759-c3aa35be05b7` queued; parent `11175250-c50d-44e9-9af5-cd6eba79bab2`; worker card path recorded above. Anchor `gcta-a5e9ba575aaf`, baseline HEAD `52097258290d1bff0191319242d4948e214af9e3` fingerprints existing dirty state only. | exit metric=E10 admitted; work pending | ack=HUB-SUP-AUDIT-SHELL-BEFORE-VISUAL-CERT-20261020,HUB-SUP-AUDIT-NH16-1-VISUAL-FAIL-20261020,HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020
2026-10-21T00:32:30Z | Parent independently reviewed source, tests, exact prewrite case deltas, evidence integrity and four 390x844 captures against hash-verified NH16.2 light/dark originals plus NH06 filter behavior. G1 passes: focused History tests 5/5, Test Genie workflow `20261021-002625-9dc0246f` 11/11, and all four selected Workflow Health cases pass with lease `9f3ec21d-3c9e-443d-bbb7-e3d9ea4c60af`, 158 test-pool requests, zero primary/test-root writes and empty cleanup errors. G2 passes for this bounded slice: four filters are visible/operable in both themes and the 200-result meaning is preserved; the added controls follow existing NH06 behavior. Shared shell and broader page composition still differ from NH16.2 and remain open at the whole-design visual gate. G3 passes: only admitted History component/test and two History BAS files changed; BAS deltas add `empty-open-asks` to setup navigation, preserving all original assertions and routed-isolation labels. No material privacy, route-boundary, bounded-data, keyboard/touch or maintainability issue found. Seq30 is acknowledged by child receipt and observed served effect; seq31 is acknowledged by the fresh changed-producer unit check recorded separately. | gate G1=pass [focused 5/5; TG workflow 11/11; Workflow Health 4/4 with routed lease/counters] | gate G2=pass [bounded phone-filter behavior and loaded-set meaning reviewed against selected originals and NH06 behavior; full fidelity remains open] | gate G3=pass [independent parent review; no material issue] | exit metric=E10 accepted bounded; whole-redesign gates remain open | net runtime lines=unmeasured | net test lines=unmeasured | ack=HUB-SUP-AUDIT-E10-SERVING-PROVIDER-FOCUS-20261021,HUB-SUP-AUDIT-UNIT-DIAGNOSTIC-RETURN-20261021

ACCEPTED 2026-10-21T00:36:00Z — E10 accepted by the orchestrator after independent review. Its phone filters and directly affected four-case test inputs pass bounded G1/G2/G3; the full Notification Hub fidelity and benchmark gates remain open.
