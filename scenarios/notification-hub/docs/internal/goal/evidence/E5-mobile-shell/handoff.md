# E5 worker handoff — mobile shell and NH16.1

**Status:** implementation and worker validation complete; E5 is not accepted by this worker. E4 remains open and unaccepted. Parent review (G4) is pending.

## Implementation

- The shared RCL `AppShell` remains the shell owner. Hub uses its `tabs` mobile mode; its canonical five destinations stay usable at phone width. Desktop sidebar navigation remains in the same component.
- Safe-area ownership now reserves the top inset on the shared shell and leaves bottom inset handling to the shared bottom navigation. The Hub skip link is positioned completely above the status area at rest. Its focused rule resets the transform. Activating the shared skip link preserves its hash destination and explicitly focuses the main landmark.
- The selected NH16.1 history state now has checked-history context, a URL-backed text filter over only the loaded history response, clear action, and copy that makes the 200-record limit explicit. An empty text result says older matches may exist and does not claim an account-wide search. Existing state filters remain for wide layouts and continue to filter the same loaded rows.

## Consolidation and inventory

All six admitted source paths existed in the first prewrite capture; all six remain. No source module, exported caller, API, dependency, shared package, generated source, storage, auth, or lifecycle code was removed or changed.

- The RCL `AppShell` consumer now selects `mobileNav="tabs"`; the library's mutually exclusive drawer mode no longer emits its mobile Menu trigger. The five canonical tab destinations cover the same app destinations and are reachable at the lower edge. No second Hub-owned navigation was added.
- The RCL theme utility remains available to the desktop sidebar. Its phone-header instance is hidden; the audited NH16.1 selection has no theme selector.
- History state-filter links remain in source and are hidden below the `sm` breakpoint, where the selected NH16.1 composition omits those chips. They remain available on wider layouts.
- No local caller/export was deleted. The route and `ThemeControl` caller remain. The only omitted generated shell control on phones is the drawer Menu trigger described above.

## Before/after source inventory

The exact initial prewrite manifest is `prewrite/original-20261020T2125Z/manifest.json` (SHA-256 `d8906af010ce2038086601bd6f3dba220da8ec777a5a085e839f89c0f9590673`). It records size and SHA-256 for every admitted file; all were present. The baseline `HEAD` observed for this work is `52097258290d1bff0191319242d4948e214af9e3`, with dirty-scope anchor `gcta-ed0b32090a35`. Additional grouped prewrite checkpoints and their hashes are indexed in [checkpoint-index.json](prewrite/checkpoint-index.json). These manifests preserve exact byte length and SHA-256 identities; they do not archive the original text bodies, and checkpoints were grouped before edit clusters rather than repeated before every individual patch. Because the source files were already dirty at admission, this limits exact line-by-line delta attribution to this worker. The parent can use the initial hashes to identify that admitted baseline.

| Admitted source path | Initial bytes / SHA-256 | Final SHA-256 |
|---|---:|---|
| `ui/src/layout/AppShell.tsx` | 1,877 / `d6125f65c16c767c5aebc974d053849db64497fccab3a8667e998b1067e8b84f` | `afb7bd59d280aa345b597fe459755b248fb3c7216b3407037177be3e15e7d5c1` |
| `ui/src/layout/AppShell.test.tsx` | 5,065 / `5d369e3eee6a66267736371ef857fe03d3acb07d16da789df197815eb06343cf` | `1384ee403038f3abe5a64f667019d5cb33bdccbfd1d622e5d142ffad772c4055` |
| `ui/src/layout/AppShell.a11y.test.tsx` | 1,788 / `9a015ff40cd5ab88fe23c5963f06aca08a39519e1c8454eaf6136575b8723a9f` | `1c24e3140d86f88f62c27bdf57b2aee329b61a3707060de1b2dbd4b575e0e3ab` |
| `ui/src/styles.css` | 1,197 / `851f94b53c86186024bd10afd7845d427c855f54f56054970925accfe9567513` | `f36431dbf564f12154c3a9288b56748c3850757d13086d753bb75d33a80b067a` |
| `ui/src/features/asks/HistoryPage.tsx` | 5,695 / `5d58485cc9d2656750f60d8308539d4cfaba0146f735c26d11af1cb933493f4b` | `1a1947aa02292555ff29c91cbf0c32e0889cfc0c6126e76afcc700621d4ff05e` |
| `ui/src/features/asks/HistoryPage.test.tsx` | 5,945 / `e93e3dda2bda3854791f2d9ecc6ce7491c5a96a16d9b3ce082e374470978d243` | `33c475d33969f09268c12c580d6e5821c0fa1e358cb115a0a06c460ce26344d5` |

## Governing source identities

- BENCHMARK SHA-256: `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`.
- Notification Hub dossier SHA-256: `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`.
- Notification Hub atlas SHA-256: `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`.
- Selected image manifest SHA-256: `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef`.
- Audited served NH16.1 pair SHA-256: `800a9feba9ffead15af8447f6267ce7cb6425ea39b9d99b9a29349fd5472733e`.
- Owner mobile capture SHA-256: `cd666df3d73643bb4b9683c953ff5d84f6e5ecce9ff61089e25974314d18342b`.

## Validation and served evidence

From `scenarios/notification-hub/ui`:

- `pnpm type-check` — passed; generated strings reported in sync.
- `pnpm exec eslint src/layout/AppShell.tsx src/layout/AppShell.test.tsx src/layout/AppShell.a11y.test.tsx src/features/asks/HistoryPage.tsx src/features/asks/HistoryPage.test.tsx` — passed.
- `pnpm exec vitest run src/layout/AppShell.test.tsx src/layout/AppShell.a11y.test.tsx src/features/asks/HistoryPage.test.tsx --reporter=dot` — 3 files, 15 tests passed (AppShell 9, axe/accessibility 2, History 4). Regressions cover initial skip hiding, keyboard focus-visible state and target focus, all five phone tabs, text match/clear/loaded-limit behavior, and retained wide state filters.
- `pnpm build` — passed. Vite emitted its existing large-chunk advisory for the 752 KB main JS bundle.
- `make stop && make start` — managed lifecycle ended healthy on UI port 20001. One stop/start attempt encountered `tunnel-manager/live is stopping`; the normal `make start` retry completed successfully. Optional SMTP credential remains unavailable, as before, and is not required for the UI.
- No Test Genie phase was named for this slice; focused UI checks and the managed served-build lifecycle were used.

Final served bundle identity: `ui/dist/assets/index-CBoQHREy.js` (752.43 KB, 229.13 KB gzip) and `ui/dist/assets/index-BFADDlHd.css` (26.95 KB, 6.51 KB gzip). Final scenario start was healthy at 2026-10-20 21:56 UTC.

Final BAS captures use the served route `http://127.0.0.1:20001/history?q=mobile`, viewport 390×844 CSS px, DPR 2:

- Light: execution `d3a799ba-10c8-4ad7-bf8c-a3df1ed0e6d0`; [capture](served-final/NH16.1-light-390x844.png); [DOM geometry](served-final/geometry/geometry-summary.json).
- Dark: execution `e588eb8b-594b-4a42-af78-9a80c8323138`; [capture](served-final/NH16.1-dark-390x844.png). The phone shell hides the theme utility, so this BAS capture action sets the document’s dark theme token attributes before screenshotting. It captures the served bundle’s dark CSS tokens without image postprocessing.
- Keyboard and bottom navigation: execution `cac057e6-af32-4faf-a495-d50d3f2e5f3b`; [interaction summary](interaction-final/interaction-summary.json), [compressed raw timeline](interaction-final/final-timeline.json.gz), and [accessibility snapshot](interaction-final/accessibility.json).
- The separate `capture-surface` program path returned `binding_error: agent credential authority unavailable` at `browser-automation-studio/capture/capture`; the BAS CLI capture path produced the final receipts above.

Measured in the final served light capture:

- Safe top inset: 59 px; shell begins y=59; phone header is x=0, y=59, 390×52; main begins y=111. Status area and header do not overlap.
- Safe bottom inset: 34 px; bottom nav is x=0, y=741, 390×103; each tab row is y=750 through 810, leaving the bottom 34 px for the inset.
- Bottom tab targets are each 78×60 CSS px: Updates → `/updates`, Needs you → `/asks`, History → `/history`, Delivery → `/delivery`, Settings → `/settings`. BAS pointer clicks reached every destination and marked each selected tab `aria-current="page"`.
- Keyboard Tab focuses `layout-shell-skip`; after its transition it is `:focus-visible`, transform `none`, at x=8, y=67, 116.6×32. Enter sets `#layout-shell-main` and the served browser reports the focused target as `MAIN#layout-shell-main`.
- Horizontal widths are 390 px for body, shell, and viewport; no horizontal overflow.

## NH16.1 comparison and adaptations

Compared both final 390×844 captures to the exact originals in the selected manifest: light SHA-256 `dd0f265a97772e573d8dd00d243f7a6e3291da9d49fee43c32eee4c14e962313`; dark SHA-256 `1febf9bf5328bfc55f4e3fd6347f4d5c259473875431fa61a11403724855648e`. The source contract matches: Notification Hub identity, checked History context, “No matching loaded requests,” labeled `mobile` search, clear action, latest-200 limitation, and incomplete-account-search disclaimer.

Evidenced adaptations:

1. The common shell reserves a 59 px phone status inset above the composition and the 34 px home-indicator inset below it. The selected originals have neither device inset; the composition content is shifted down by the top inset.
2. The selected original’s “Menu” trigger is replaced by the canonical five-item bottom navigation. RCL exposes drawer and tabs as mutually exclusive mobile modes; tabs satisfy the admitted lower-navigation operability requirement and expose the same destinations.
3. The selected original has no lower tabs; E5 adds the shared bottom nav in its previously blank lower area.
4. The selected original’s fixed `09:00 UTC` is represented by the actual UTC time the loaded history query last checked.
5. New filter/empty/clear copy uses `defaultValue` in the scoped History page because locale files were outside admission. Non-English locales therefore fall back to English for those new strings; this remains a localization limit.

The dark visual was set through document theme attributes because the phone utility is hidden. Physical device behavior and other safe-area profiles were not exercised; the measured 59/34 insets are the BAS phone environment. The browser clicks are served-browser pointer actions, not physical touchscreen taps. No backend data was changed.

## Epoch check

The required `agent-manager effort epoch-check` run records G1, G2, and G3 as met from the returned evidence. G4 remains unverified/unmet until the separate parent review. The checker also emitted its existing custom-gate-reason warnings for G2/G3; the worker left gate definitions untouched as instructed. No acceptance check was requested or run.

## Parent handoff

Worker run `d2512980-16bb-4a4d-864f-085ae9ff01af` completed its implementation and evidence work. Do not infer E5 acceptance from this handoff. A separate parent review is still required for G4. E4 remains open and unaccepted: its 16 captures exist, but its prior material visual differences and G5 review gate remain unresolved. This slice does not change E4 gates or accept G5.

Actual meaningful work units: 3 (common shell and keyboard/mobile navigation; targeted loaded-history correction; focused validation, served evidence, and handoff).

## Seq33 continuation — compact NH16.1 phone shell

This is a follow-up to, and does not revise, the previously accepted bounded E5 work above. It does not accept NH16.1 as a whole, E4, or any whole-redesign gate. Parent run: `11175250-c50d-44e9-9af5-cd6eba79bab2`; worker run: `1dc9cb8f-5f3f-4587-8a72-a959e3e7a631`; applicable directive: `HUB-E1-LIFECYCLE-RETURN-PRODUCT-PRIORITY-20261021`.

### Changed

- `ui/src/layout/AppShell.tsx` now puts the selected Notification Hub title and a compact, operable Menu disclosure in the shared phone header. The menu reuses `NAV_ITEMS`, exposes all five current destinations, marks the selected destination, and closes after route change. The existing shared five-tab lower navigation remains.
- `ui/src/styles.css` gives the Menu control and each menu destination at least 44 CSS px of touch height, positions the menu under the phone header, and retains the focus-visible skip-link treatment. The header Menu was measured at 59.3×44 CSS px in the served diagnostic capture; lower tabs measured 78×60 CSS px each.
- `ui/src/features/asks/HistoryPage.tsx` marks the text-search state. Phone CSS hides the outcome-filter row only while a text query is active, matching NH16.1's single-search hierarchy. Clearing filters restores the normal phone history view; wide state filters and non-search phone filters remain present.
- Focused tests cover Menu disclosure, five upper destinations, selected History state, route selection/close, search-state marker and reset, existing five lower destinations, skip-link first Tab/target, and loaded-history search/clear/limit behavior. No caller or existing BAS case was removed or edited. `history-bounded` retains its lower History-tab interaction and its unique history/fixture/read-only assertions.

### Exact prewrite and attribution

- Initial task prewrite manifest: `prewrite/seq33-original/manifest.json`, SHA-256 `12611d7fa36766c5fe401d36a93c5a649794cde4616bdd9fe86ab537f3b5c099`. It records the exact initial AppShell, CSS, AppShell test, this handoff and E5 epoch bytes before the first source write.
- Checkpoint before the menu-route test adjustment: `prewrite/checkpoint-menu-route-test/manifest.json`, SHA-256 `1e5f3d1496a7aaaeb7cbde54332c9370cdd65dd3c03d8b9a1d6cbfdf9056bd89`.
- Checkpoint before the NH16.1 search hierarchy correction: `prewrite/checkpoint-search-hierarchy/manifest.json`, SHA-256 `6d21a621554c78022bb0a63a8c52045f9a018e9912ed023f73b97a59b7fa17a2`.
- Checkpoint of explicit absent capture destinations before copying BAS evidence: `prewrite/checkpoint-served-capture/manifest.json`, SHA-256 `15faad0438cd5799acea6848585a114f6be5923e908ed6485a8bd93f143911ce`.
- Checkpoint of this handoff before this append: `prewrite/checkpoint-handoff-seq33/manifest.json`, SHA-256 `e2e0074ffaefc1c4e5926d17fcd5b33b6dcb75f3167adc05252d2a3b58049af0`.
- At the initial exact-path baseline HEAD was `52097258290d1bff0191319242d4948e214af9e3`; the admission anchor `gcta-fc42cd8ac6ad` fingerprints preexisting dirty scope and is not an authorship claim. The source changes attributed to this seq33 worker are limited to the five existing UI files above. Snapshot/evidence files are task evidence. BAS and managed lifecycle generated artifacts were not source-authored. No BAS case, registry, generated source, shared package, API, fixture, storage or primary recipient data was changed.
- Relative line inventory versus those prewrite checkpoints: runtime `+83` (AppShell +23, CSS +56, HistoryPage +4); tests `+21` (AppShell +18, HistoryPage +3). No deletion or removed caller. The second CSS rule is retained because it applies only to active phone text search; regular phone and wide status-filter behavior remain.

### Focused validation and current served build

- `cd scenarios/notification-hub/ui && pnpm exec vitest run src/layout/AppShell.test.tsx src/layout/AppShell.a11y.test.tsx src/features/asks/HistoryPage.test.tsx --reporter=dot` — PASS, 3 files / 17 tests.
- `pnpm exec eslint src/layout/AppShell.tsx src/layout/AppShell.test.tsx src/features/asks/HistoryPage.tsx src/features/asks/HistoryPage.test.tsx` — PASS.
- `pnpm type-check` — PASS; generated strings are in sync. `pnpm selector:check` — PASS.
- `pnpm exec vite build` — PASS (existing large-chunk advisory remains). Normal owner `make restart` completed Notification Hub healthy twice, with no provider restart. Final `vrooli scenario status notification-hub --json` reported running/healthy, started `2026-10-21T00:59:57.284009732Z`, build identity `sha256:3ccf9d5eec3fec7b4d283c0612d9e02b52e1ad59d72308bd21a494bcb06283e7`. Final served bundle `ui/dist/assets/index-Djeo336R.js` SHA-256 `1d31537de1159c9d2c9d5b60a5d46c7993ea1d67b102cf911c1f2847850f0270`; CSS `ui/dist/assets/index-DF0jlgZW.css` SHA-256 `acf962ab13dde62174f74ee3034d1a2c4a60c88c67dcfb369330bb143bca607f`.
- No Test Genie/UI Health phase was rerun. The existing History BAS case was not directly changed: it still uses the lower History destination without active text search. No route/identity substitution or answer/notification action occurred.

### Retained capture and exact limits

- BAS light diagnostic: execution `bf6adf4e-1650-4a48-8e9b-5764be9624af`, route `/history?vrooli_test_mode=1&e4_fixture=empty-open-asks&q=mobile`, 390×844 CSS px, DPR 2. Screenshot, DOM, and accessibility snapshot are in `served-seq33/`. Its source build identity was the earlier seq33 healthy build `sha256:3a2e8277cea286d0400746450885d3ee62d96ce0397936f60d68b76babf79be1`, before the final active-search chip CSS correction; this capture is historical and diagnostic, not final selected-state proof. The fixture query remained pending (`History · checking`, `Loading decision...`); no recipient content was displayed. The screenshot does show the compact header, Menu target, safe-top spacing, and all five lower destinations.
- The current BAS accessibility receipt identifies `a[data-testid="layout-shell-skip"]` at `(8,-64 116.6×32)` before interaction, matching the selector and offscreen location in failed UI Health run `20261021-004215-42e18a22`. It also measured Menu `(314.7,62.5 59.3×44)` and each bottom link at 78×60, y=750. It contains no post-Tab measurement, same-node reveal, main-focus result, or overflow width measurement. The AppShell unit regression passes first-Tab `:focus-visible` and Enter-to-main, but this does not pass or reinterpret UI Health.
- Initial BAS capture execution `18d0865f-fdcd-445a-b967-8e8b995e4bb3` failed on an invalid `--wait-for timeout:1000` CSS selector; the corrected capture above completed. Both receipts are retained in `served-seq33/light-receipt.json`.
- The BAS capture owner contract in `scenarios/browser-automation-studio/docs/reference/configuration.md` states that `--routed-test-mode` requires a paired SQL/file routing lease and does not create one. No fresh exact lease was installed for this capture-only task. The fixture screenshot therefore does not establish settled History data or routed isolation counters. Dark capture and current-build selected light/dark comparison remain unverified. The exact earlier managed shadow refusal remains in `evidence/E1-seq32-73fe01a5/handoff.md`: `variant dependencies (vrooli-events) require a non-live instance: tunnel-manager is live`. That answer-journey path was not retried. Retry managed capture only after the normal owner supplies a fresh paired lease for the isolated fixture route.
- Selected source joins remain those in `evidence/commissioning-source-map.md`: NH16.1 → “Behavior and migration / Lists and history with honest limits” and “Experience / Counts and scope”, with NH-AC-26/37. Exact selected light original SHA-256 `dd0f265a97772e573d8dd00d243f7a6e3291da9d49fee43c32eee4c14e962313`; dark counterpart SHA-256 `1febf9bf5328bfc55f4e3fd6347f4d5c259473875431fa61a11403724855648e`. Existing E5 and E8 composition receipts remain the prior baseline; they are not restated as full NH16.1 or redesign closure.

Parent review/acceptance, actual served post-Tab reveal/focus-to-main measurement, final-build safe-fixture captures in both themes, settled selected NH16.1 comparison, horizontal overflow measurement, and any subsequent managed isolation counters remain open. No acceptance is claimed by this worker.

### Required full-epoch check

Command: `agent-manager effort epoch-check scenarios/notification-hub/docs/internal/goal/epochs/E5-mobile-shell-and-nh16-1.md --runs 1dc9cb8f-5f3f-4587-8a72-a959e3e7a631 --wake-key 11175250-c50d-44e9-9af5-cd6eba79bab2` — exit 0; 7 slice lines against estimate 5; net runtime +83/test +21; no flatline, overrun, growth, spend, wall-clock trigger, or STEP_BACK. It reports G1–G4 met by the epoch's earlier retained evidence and repeats the prior `ACCEPTED 2026-10-20T22:02:10Z` line. Those are historical E5 results, not acceptance or proof of the new seq33 post-Tab, current-build dual-theme, settled-fixture, or overflow evidence. It reports existing custom-gate-reason warnings for G2/G3 and unknown yield. No acceptance option was used, no gate definition was edited, and this worker does not claim acceptance.

Final authored source hashes: `AppShell.tsx` `923d893384838c64ddadd1b6308b306678a35e805b8defd914234b62f6d29f90`; `styles.css` `413b7e6bfb5e7133aefdf6adc3a6651597264bec6ccfd1667345df5fff746b45`; `AppShell.test.tsx` `7c6575e97e6dc6d99c959b99871a2246445a9fe1321457ea4fce6b526e8c266e`; `HistoryPage.tsx` `5411c389392b0c06e53454867d227f20b7f0afd611710c6cec575ee6b88fab04`; `HistoryPage.test.tsx` `dfeb1d980952aacd7687044afb37c72b013523a522985033a421abf16b87ccd3`.

## Seq36 continuation — final selected NH16.1 state and settled focus proof

This is the current continuation of the already-admitted seq34 E5 assignment under parent `11175250-c50d-44e9-9af5-cd6eba79bab2`, worker run `29b074c2-8856-40b9-8e10-a59f28586426`, effort revision `sha256:ee51f1db548652434a8bba5c3a301439abf2c316fef04e1652191e46b9332dd0`. No E5, E4, or redesign gate is accepted here.

### Changed paths and exact snapshots

Only the two already-registered History BAS cases changed in this continuation; no product source or caller changed. Their assertions and routed-isolation labels remain, and no assertion/caller was removed. The added workflow actions cover current NH16.1 state, same-node keyboard focus, main focus, five destination routes, and viewport/shell widths.

- Before graph correction: [manifest](prewrite/seq36-before-graph-fix/manifest.json), SHA-256 `a5ce729dbce4bad086dd0264db29686c5d08a93d571531208062ad1435548cfa`.
- Before settled-focus assertion: [manifest](prewrite/seq36-before-focus-settle/manifest.json), SHA-256 `c480446bb23e68bfa36b166ed44abcc663a6a31517fc06a0439b690c6fedf857`.
- Before final capture writes: [manifest](prewrite/seq36-before-final-capture/manifest.json), SHA-256 `eb529c5e21dc0333f67e6984da0e6bbfedcb9b27e3e2c63421a465bef43405dc`; both final PNG paths and summary were explicitly absent.
- Before this handoff/epoch update: [manifest](prewrite/seq36-before-handoff/manifest.json), SHA-256 `85190271479cb3148879cc1ead959e2f6792c19c0b69bd63a2a455e16989cffd`.
- Initial authorized source-baseline and intermediate checkpoints remain indexed in [checkpoint-index.json](prewrite/checkpoint-index.json). The working tree was shared and dirty at admission; anchors and hashes identify bytes, not authorship. Current product hashes match the later Seq33 paragraph: AppShell `923d893384838c64ddadd1b6308b306678a35e805b8defd914234b62f6d29f90`, CSS `413b7e6bfb5e7133aefdf6adc3a6651597264bec6ccfd1667345df5fff746b45`, HistoryPage `5411c389392b0c06e53454867d227f20b7f0afd611710c6cec575ee6b88fab04`.
- Final case hashes: `history-bounded.json` `6400f16b9bc75850995cc7cb66eb57c3eb030bfc6ce74b0e0a3d6f0a74352a11`; `history-dark.json` `a157ead819434a705ac9d8b3f87a890bce3fb427b6924c44dff45d08f551b9e4`.

No state consolidation/deletion was made; no callers were removed. The workflow graph now keeps every unique existing assertion on one reachable path and places the selected-state screenshot assertion after phone measurement. Existing selector registry advisory debt remains: Workflow Health flags direct `#history-search` use in both cases as `workflow.selector_unregistered`; execution succeeds, and registry output was not edited.

### Current managed captures and comparison

Workflow Health owner route: `workflow-health validate start-selected notification-hub --path /home/matthalloran8/Vrooli/scenarios/notification-hub --idempotency-key nh16-1-e5-selected-focus-settled-390x844-20261021T0145Z-0577a351 --case-paths bas/cases/e4/history-bounded.json,bas/cases/e4/history-dark.json --json`, followed by exactly one `workflow-health validate wait f90890b1-b547-41d5-8232-a84e4ac0f302 --timeout 15m --json`. Run succeeded (2 selected, 2 executed, 2 passed, 0 failed/refused/skipped); fresh paired lease `f90890b1-b547-41d5-8232-a84e4ac0f302` provisioned 118 test-pool requests, zero test-root writes, zero primary requests, zero primary writes. The cases use their registered `e4_fixture=empty-open-asks` routed-isolation setup; no real answer or primary data mutation occurred.

The served build was healthy at UI port 20001 with managed identity `sha256:244713c24dd987763b769f8b85834bae65050f749eeafc9c29d5e2f587420688`. Both final screenshots are 390×844 CSS px at DPR 2 (780×1688 raster) and match the selected state: checked History context, “No matching loaded requests,” `mobile` search, clear action, latest-200 boundary and incomplete-account disclaimer. Exact source originals remain light `dd0f265a97772e573d8dd00d243f7a6e3291da9d49fee43c32eee4c14e962313` and dark `1febf9bf5328bfc55f4e3fd6347f4d5c259473875431fa61a11403724855648e`.

- [Final light capture](served-seq36/NH16.1-light-390x844.png): SHA-256 `b40b3bee03a2a841385b10e63d1ff4b265db04472a189d6240eaecbd05929bbc`, BAS execution `e27fbb6e-9077-4dc2-a777-4f1c6dcc851c`.
- [Final dark capture](served-seq36/NH16.1-dark-390x844.png): SHA-256 `d1ae33632fd69aac6737788cd353502b37f1a048ba0d79cb786a54b4d1d4c9e0`, BAS execution `ec3fa0a3-3b51-4693-a904-535315381b62`.
- [Machine-readable summary](served-seq36/summary.json), SHA-256 `22f69e6b4610280494adf05c138ada6f144fb6696da3863947ec762cf43549c1`.
- Full owner timelines and result receipts remain under Workflow Health run artifact directory `/home/matthalloran8/.vrooli/state/vrooli/workflow-health/run-artifacts/notification-hub/runs/f90890b1-b547-41d5-8232-a84e4ac0f302/`.

Compared with the selected originals, the measured adaptations remain the shared shell's 59 px upper reservation and 34 px lower area, retained five-tab lower navigation, and a live checked timestamp replacing static `09:00 UTC`. Both theme screenshots show the selected `mobile` search hierarchy. The BAS viewport records the insets used by the shell; it is not physical installed-app chrome evidence. The dark theme came from the registered dark case's theme action. Locale fallback for the scoped new search copy remains disclosed above.

### Served keyboard, route, and width measurements

Both theme timelines recorded the same skip-link DOM node before and after actual Tab input. Before Tab it was unfocused, not `:focus-visible`, at `(8,-64)` with 116.6×32 CSS px. The final case now waits 350 ms for the focus transition to settle, then fails unless the same DOM reference is active, `:focus-visible`, and fully in the viewport. In both themes it measured `(8,67)` at 116.6×32 with `transform: none`. Enter then left `MAIN#layout-shell-main` active and set `#layout-shell-main`.

Each theme's operated case asserted `aria-current="page"` for Updates `/updates`, Needs you `/asks`, History `/history`, Delivery `/delivery`, and Settings `/settings`. All five route assertions passed. At 390×844, body and document client/scroll widths were each 390 px, so horizontal overflow was absent. Shell measured `(0,59,390,785)`; lower nav `(0,741,390,103)`; each tab target 78×60 at y=750.

Two superseded intermediate receipts are retained but are not final proof: run `af0ffa88-5f20-4f71-a1a9-a5fea5b66ab8` had disconnected final-state graph nodes and its light screenshot showed the defaulted-filter state; run `4d29a7ee-67f4-4bac-9a6f-fdb654fe36b1` connected the selected state but sampled focus during its transition (same node/focus-visible, partial viewport clipping). The final `f90890b1…` run corrected reachability and added a settled visible-geometry check; do not use those earlier images or measurements as final.

### Seq36 changed-producer UI Health result

Exactly one requested changed-producer Test Genie phase ran after the Hub inputs and managed build settled: `vrooli scenario test notification-hub --phases ui-health`, run `20261021-011625-5224f968`, producer build `sha256:222b93714d8b572bc72a598da3579bd5cf9f81f48cc9eb7fcc3ddfb1088b1baa`. The actual verdict is **FAIL** (1 phase failed; 48 observations; 44 findings: 1 error, 34 warnings, 9 infos). Blocking findings are `standard_shell_ownership` at `AppShell.tsx:52` (locally rendered navigation), `visual_native_screen_unverified` (browser viewport is not installed-app chrome), and `visual_safe_area_not_exercised` (UI Health's separate mobile captures measured 0 px safe-area insets). Runtime render is L5/Complete and reports a successful render/bridge handshake. The served same-node proof above is additional direct evidence; it does not relabel this UI Health run or prove its native/safe-area claims.

The previous UI Health failure `20261021-004215-42e18a22` remains historical. The root supervisor owns the shared UI Health frame-focus producer; this worker did not change it, whitelist links, edit UI Health/Test Genie, or claim that producer gate fixed. UI Health's current findings and ownership remain for independent parent review.

### Validation and return

No UI/product input changed in this continuation, so the earlier Seq33 focused source test, lint, selector, type-check and build receipts remain the product-source validation; they were not rerun solely due to the reconciled hashes. The changed inputs were the two registered History cases, which were validated through the exact fresh Workflow Health lease above. No broad unchanged phase was rerun. The due changed-producer `ui-health` phase ran exactly once and its failure is retained.

This handoff adds measured behavior and artifact identities for parent review; it accepts no epoch gate. Retained debt includes the selector registry advisory, locale fallback, and missing physical installed-app evidence. No API, schema, storage, auth, shared package, generated source, lifecycle, dependency, real-data mutation, or root product path was changed. The assignment's original deadline `2026-10-21T03:48:04Z`, full redesign scope, destinations, assets, compositions, states and A1–A10 remain intact.

## Seq36 final screenshot correction — current authoritative capture pair

This section supersedes the earlier Seq36 paragraph's `served-seq36/` image links and the earlier case hash row: those captures retained the browser text-field focus ring. The final route now blurs the search input only after recording phone geometry, preserving `q=mobile` while matching the selected original's unfocused input. No product source changed.

The final changed-input Workflow Health run is `a53cf6f6-417e-42ae-8b18-9576ee79c639`, idempotency key `nh16-1-e5-selected-final-390x844-20261021T0202Z-30bac441`; it used one `start-selected` and exactly one `validate wait`. It succeeded with both registered cases passed under lease `a53cf6f6-417e-42ae-8b18-9576ee79c639` (118 test-pool requests; zero test writes; zero primary requests/writes). Final execution IDs: light `861b800b-e906-41a4-8c92-24170c76a6f5`, dark `7c9bd46c-3689-4fef-8a34-f0b12c78eef7`. Served build identity remains healthy `sha256:244713c24dd987763b769f8b85834bae65050f749eeafc9c29d5e2f587420688` at UI port 20001.

- [Final light image](served-seq36-final/NH16.1-light-390x844.png), SHA-256 `7bb88844591a6dc5e686733b1507152ba54f2b3dddc3123ac5a89a9ad7a3d59f`.
- [Final dark image](served-seq36-final/NH16.1-dark-390x844.png), SHA-256 `e91e076c6950b64fc678932100f76e5c7773af220061f1b27af0aeff9ac9240e`.
- [Final machine-readable summary](served-seq36-final/summary.json), SHA-256 `19886b3250ef7e457222d2f67ceccb27d77217a445a1967834d77962d43be8d8`.
- The final images are 780×1688 rasters for 390×844 CSS px at DPR 2. They show the selected query with an unfocused field and retain all selected-state copy. The final summary contains the final lease, case hashes, focus measurements, routes, and no-overflow widths.
- Final BAS case hashes after the search-blur correction: `history-bounded.json` `292941c53234b1918addf4eed3e731468550140f4ebe46d08b19273b064d2414`; `history-dark.json` `35b76667b4711239b3653a05164ecb49d38e63a336fbab63bb45557492825047`.
- Exact prewrite manifests for the blur action and final images are `prewrite/seq36-before-search-blur/manifest.json` (SHA-256 `42275bd3d1c1add1243358a8fa91e8774e89d8882deae3dd7e0e78b1a9431ef2`) and `prewrite/seq36-before-unfocused-capture/manifest.json` (SHA-256 `88b323d180a9404f439b1a094ebb9cc1825fe1cec2f24cdf12df93ae3356d30c`). Before this correction's handoff/epoch append, `prewrite/seq36-before-final-handoff/manifest.json` SHA-256 was `2be97c7363ea62bc2db9517b982edbd765ee37922a31d49afd2be6dcdc7e65e2`.

The earlier `served-seq36/` pair and `f90890b1…` run remain valid for the settled focus/route/width measurements, but they are superseded as selected-state screenshots because the search field was focused. The latest `a53cf6f6…` run repeats the served focus, route, and width actions and is the authoritative artifact set for parent review. Seq36's single Test Genie UI Health run and its FAIL verdict remain unchanged; it was not rerun.
