# E9 — NH16.4–16.6 Delivery evidence on phone

- Outcome: implement and validate the selected phone delivery evidence states: no receipt, distinct transport meanings, and unknown transport status. Preserve truthful receipt semantics and the existing consumer/data path.
- Kind: fix.
- Status: accepted as a bounded E9 slice after current changed-input validation and parent review at `2026-10-21T00:35Z`. Task `9baf6197-51ea-4f4b-9c18-e16d4c966e13`; run `ec3032e5-795d-4cbc-9333-c433a38771fa` COMPLETE at `2026-10-21T00:12:30Z`. E4 and all whole-redesign gates remain open.
- Parent: `11175250-c50d-44e9-9af5-cd6eba79bab2` (verify current identity before admission).
- Estimate: 4–6 meaningful units; report actual work, waits and interventions.
- Work shape: one coherent Delivery phone composition with directly affected page tests and existing registered BAS self-validation. Independent review stays with the parent.
- Selected source authority: `docs/internal/delivery-reliability-research/follow-through/BENCHMARK.md`; selected dossier `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-dossier.md`; atlas `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-atlas.md`; `docs/internal/delivery-operations/2026-10-12/sources/notification-hub-revision/selected-image-manifest.json`; requirement/state join `scenarios/notification-hub/docs/internal/goal/evidence/commissioning-source-map.md`, rows NH16.4–16.6 and NOTIFICA-P0-004/008/010/011. Admission fingerprints from latest verified source handoff: BENCHMARK `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`; dossier `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`; atlas `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`; selected manifest `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef` (re-hash/recheck before admission).

## Selected phone references

All assets are from the selected revised manifest, CSS viewport 390×844, DPR 2, original paths under `docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/`.

| State | Selected originals | SHA-256 |
|---|---|---|
| NH16.4 no receipt | `NH16.4-light-phone-p1-r3.png`; `NH16.4-dark-phone-p1-r3.png` | `8ec8639ba184b506c590b6862a7724fe33f0a7b15582a68998b25aa743ea826d`; `42abf8fd447faeae2c446b4dacbce16402cc94f72f2a41e850b31300ed7a91d9` |
| NH16.5 transport states (two viewport captures per theme) | `NH16.5-light-phone-p1-r2.png`, `NH16.5-light-phone-p2-r2.png`; `NH16.5-dark-phone-p1-r2.png`, `NH16.5-dark-phone-p2-r2.png` | `255bc810baf2e83175f4bf9be4767ef059fffb7f5b57b07459768f4eb72fc7e0`; `ef465c6dde953fe5da9457f53ba76570a9d12d926ab96aa691481b1520300dc`; `37bd1acec83784dc6a090cbda8e9bed4361cf0f5c58714161c2052d02591da4c`; `b3b9847643b128d3430c342df300d179475661bab1a6ad2351d35d29445fe3c0` |
| NH16.6 unknown transport state | `NH16.6-light-phone-p1-r3.png`; `NH16.6-dark-phone-p1-r3.png` | `176093749e44dd78338e0804f63242f523eb8ae64641067f1a14a7ed947d8401`; `376f2bf7f598d27907eb22e4df568d15a60fd6d5b6c4fc29ae3619a4b7f39c24` |

Atlas semantics: no successful receipt leaves reach unknown and does not imply unread/offline/acknowledged; Delivered means adapter/provider acceptance, not device arrival/read; Pending, Held, Routed, Delivered, Failed, Unroutable and Suppressed remain distinct; unknown enum values never count as successful delivery. Do not fabricate delivery or receipts. Do not conflate absence of a delivery receipt with ask/history completion.

## Direct paths and prior failure

Inspect exact files and current owner state before writing. Expected product paths: `scenarios/notification-hub/ui/src/pages/DeliveryPage.tsx`, `DeliveryPage.test.tsx`, `DeliveryDetailPage.tsx`, `DeliveryDetailPage.test.tsx`. If implementation reveals another directly required UI file, record the reason and use normal scope amendment; no shared library/API/DB/auth scope.

Direct existing workflow paths only:
- `scenarios/notification-hub/bas/cases/e4/delivery-dark.json`
- `scenarios/notification-hub/bas/cases/e4/delivery-detail.json`
- `scenarios/notification-hub/bas/cases/e4/history-bounded.json`
- `scenarios/notification-hub/bas/cases/e4/history-dark.json`

Combined Test Genie run `20261020-233604-27e7cfc6` is terminal: unit PASS (23s), workflow FAIL (215s). Four workflow errors were all steps selecting `@selector/theme.select`; on these 390×844 phone cases the control resolved to a hidden select in the desktop shell. This is a test route/setup failure, not evidence the case assertions or E8 asks composition failed. On all four existing cases, replace only the theme setup path with the supported visible Settings theme option at phone width, then return to the same tested surface. Preserve every existing node/assertion, fixture, routed isolation label, expected state/receipt/history checks and unique coverage. Do not alter selectors registry, BAS runtime/provider code, or remove assertions. Test Genie unit in the same run passed; do not repeat unchanged inputs.

## Allowed authored paths

- The two Delivery page sources and their directly affected tests named above.
- The four existing BAS case JSON files named above, with theme initialization corrected as described.
- Evidence only under `scenarios/notification-hub/docs/internal/goal/evidence/E9-nh16-delivery/`.
- If copy genuinely changes, edit locale source JSON only and regenerate through the normal `pnpm strings:gen` owner command. No hand-edited generated strings.

No other destination, History page implementation, selector registry, workflow-health/Test Genie/BAS implementation or profile, service/API/storage/auth/shared package, generated schema/artifact, dependency, live data or external service. Preserve existing dirty work and E5/E8 accepted changes. Before any write, inspect the exact admitted paths and managed build; create and record a fresh `git-control-tower proposal anchor --effort effort:notification-hub-redesign --epoch E9 --scope 'scenarios/notification-hub/**' --json` and identify preexisting dirty attribution. Do not claim authorship of prior changes.

Admission anchor: `gcta-88cc06181e29`, baseline HEAD `52097258290d1bff0191319242d4948e214af9e3`, scope `scenarios/notification-hub/**`. It fingerprints existing dirty files and is not an authorship claim.

## Consolidation and proof

- Keep one truthful meaning per delivery state; remove duplicate/misleading labels or unsupported success only when the source and tests prove it is superseded. Preserve receipt absence, state-specific wording, bounded scope and unknown-value handling.
- Run meaningful focused Delivery page regressions. Run Test Genie `unit,workflow` once on the changed sources/cases via normal server-owned routes and wait once. The prior workflow failure must not be repeated with unchanged inputs.
- Use the existing registered Workflow Health cases for relevant changed-input diagnosis and all four phone cases for current-build dark/light state coverage. Capture current build identity, exact states/themes/viewport, selected/capture hashes, case selection, isolation/lease and cleanup counters. Keep diagnostic Workflow Health proof separate from Test Genie workflow phase.
- Normal `start-selected` route admission enforces active lease conflicts; do not require a separate lease inventory first. Do not manually clear a lease or restart any provider/consumer. Retain an exact refusal and stop that proof path if a native owner refuses.
- Return before/after path attribution, superseded elements and callers, all commands/receipts, actual work and waits, gaps, screenshots, and parent review handoff. No self-acceptance.

## Exit gates

- G1 test: focused behavior tests and changed-input Test Genie unit/workflow phases pass on configured floors; existing four workflow cases retain their distinct assertions.
- G2 custom: the selected delivery states include nonvisual transport meanings that may not be legible in every composition; this slice judges the eight required state captures plus explicit truthful wording and reviewed adaptations, while whole-redesign fidelity remains open.
- G3 review: independent parent review of product and case deltas finds no material privacy, owner-boundary, truthful-status, accessibility, unique-test-protection or maintainability issue.

## Slice log

2026-10-20T23:54:00Z | E9 selected after E8 return and independent bounded E8 acceptance. E9 targets NH16.4–16.6 Delivery phone evidence and directly affected existing four-case workflow inputs. The previously failed workflow phase was caused by a hidden desktop theme select at phone viewport; cases must set theme through visible Settings while retaining every assertion. | exit metric=E9 ready for admission | ack=HUB-SUP-AUDIT-SHELL-BEFORE-VISUAL-CERT-20261020,HUB-SUP-AUDIT-NH16-1-VISUAL-FAIL-20261020,HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020
2026-10-20T23:56:10Z | Admitted Luna-medium E9 under active effort revision 16 and verified original parent. Pre-dispatch dirty-scope anchor `gcta-88cc06181e29`, baseline HEAD `52097258290d1bff0191319242d4948e214af9e3`; task `9baf6197-51ea-4f4b-9c18-e16d4c966e13`, run `ec3032e5-795d-4cbc-9333-c433a38771fa` PENDING/QUEUED. This is admission only; no product effect is claimed. | exit metric=E9 admitted; work pending | ack=HUB-SUP-AUDIT-SHELL-BEFORE-VISUAL-CERT-20261020,HUB-SUP-AUDIT-NH16-1-VISUAL-FAIL-20261020,HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020
2026-10-21T00:12:30Z | Child returned six meaningful units. Independently reviewed `DeliveryDetailPage.tsx`, the focused tests, locale additions and all four retained BAS cases. No material truthful-status, privacy, route-boundary, accessibility or unique-assertion issue found; E9 changes preserve the existing read path and only alter the four cases' phone theme bootstrap. Reviewed all eight NH16.4/.5/.6 light/dark captures against their selected originals at 390x844. Main compositions and state meaning are present. Accepted for this bounded G2 review: documented shared app shell/navigation, explicit time-unavailable copy (the read contract has no last-checked field), and additional truthful receipt/request caveats. This does not accept the full benchmark visual gate. Focused Delivery tests, type-check and build passed; do not repeat the unchanged API/unit phase. Parent is continuing the History phone-filter defect in E10 with the existing four BAS cases retained. E9 remains unaccepted; E4 and all whole-redesign gates remain open. | gate G1=fail [Test Genie `20261021-000654-f5cb52bf` failed; unattributed API execution failure and two History filtered-empty failures; Delivery cases passed] | gate G2=pass [all eight captures and adaptations independently reviewed against selected originals] | gate G3=pass [source, focused tests, locale generation and all four case assertions reviewed; no material issue found] | exit metric=E9 not accepted; benchmark visual gate remains open | net runtime lines=0 | net test lines=0 | ack=HUB-SUP-AUDIT-COHERENT-SCOPE-SELF-VALIDATION-20261020,HUB-SUP-AUDIT-FOCUSED-WORKFLOW-DEBUG-20261020
2026-10-21T00:35:00Z | Reconciled E9 G1 using current changed-input evidence. Seq31 justified one fresh unit phase after the Unit Health producer changed: Test Genie `20261021-003355-ddf48217` PASS (24s). Native artifact catalog and unfiltered findings report show PASS, 46 advisory warnings (two `MISSING_INJECTABLE_SEAM`, 44 `LOW_COVERAGE`), zero blockers; the earlier E9 failure cause remains unknown and is not retroactively attributed. E10 Test Genie `20261021-002625-9dc0246f` workflow PASS 11/11 after the two History cases selected their own routed fixture; current Workflow Health `9f3ec21d-3c9e-443d-bbb7-e3d9ea4c60af` PASS 4/4 with a fresh lease, 158 test-pool requests, zero primary/test-root writes, and no cleanup errors. The E9 focused Delivery 6/6, type-check/build, and prior independent G2/G3 review remain applicable; E10 changed only History and its two BAS fixtures, not Delivery product inputs. Parent accepts E9 for this bounded vertical; the former unit execution failure remains historically unattributed. | gate G1=pass [current TG unit PASS, TG workflow PASS, all four registered cases pass; historical unit failure remains unattributed] | gate G2=pass [prior independent E9 capture review on its identified build; all eight states] | gate G3=pass [prior independent source/assertion review; no material issue] | exit metric=E9 accepted bounded; whole-redesign gates remain open | net runtime lines=0 | net test lines=0 | ack=HUB-SUP-AUDIT-UNIT-DIAGNOSTIC-RETURN-20261021,HUB-SUP-AUDIT-E10-SERVING-PROVIDER-FOCUS-20261021

ACCEPTED 2026-10-21T00:36:00Z — E9 accepted by the orchestrator after current changed-input Test Genie unit/workflow and Workflow Health proof reconciled its returned failures. Its historical unit failure remains unattributed; this bounded acceptance does not close the full redesign gates.
