# E10 — History outcome filters at phone width

## Status and acknowledgement

Seq30 feedback (`HUB-SUP-AUDIT-E10-SERVING-PROVIDER-FOCUS-20261021`) is acknowledged as reconciled, and the History work resumed on the available serving providers as requested. Seq31 (`HUB-SUP-AUDIT-UNIT-DIAGNOSTIC-RETURN-20261021`) is also recorded: its changed Unit Health diagnostics are for the next coherent gate. E10 adds no API change, and no API/unit phase was repeated here; this slice did not add provider scope. The signal receipt is [effort-signals-after-29.json](effort-signals-after-29.json). The phone defect is fixed and the focused and registered evidence below passes. This is a handoff for independent parent review; I do not self-accept E10 or claim the broader E4/redesign gates are closed. E9 remains unaccepted.

## Change and attribution

The fresh admission proposal anchor is `gcta-c01428926a61` at HEAD `52097258290d1bff0191319242d4948e214af9e3`. The proposal inventories pre-existing dirty state and is not an authorship claim. Its prewrite manifest is [prewrite-original/manifest.json](prewrite-original/manifest.json), SHA-256 `803436fef02147f527127ee1e634dac7475900da805bcf392fff0e2fd41b4291`.

Authored product/test/case inputs are limited to the admitted paths:

| Path | Change | Final SHA-256 |
|---|---|---|
| `ui/src/features/asks/HistoryPage.tsx` | Removed the phone-hidden nav behavior; filters now use `flex flex-wrap gap-2` at all widths. | `815d12eeac253b87dcdb7db929a33297240ec58c3feb5ca0e4b621afcbecc98e` |
| `ui/src/features/asks/HistoryPage.test.tsx` | Expanded the existing outcome-filter test at 390px and 1280px; keyboard-selects Defaulted, pointer-selects Answered and Expired, and verifies All restores each distinct row. | `1e3979dc39e6cd316f41973a3d15505f0c06d86d13f66a9790528c046442c198` |
| `bas/cases/e4/history-bounded.json` | Explicitly selects the existing `empty-open-asks` routed fixture on Settings and return-to-Updates navigation. | `28509bbc160b2c53682aea834e9c094b5083d36a03d4677ced821e3b880d54b0` |
| `bas/cases/e4/history-dark.json` | Same explicit fixture selection for the dark History case. | `0b6a3206163cc58d8c9f83181fcffa7e85a770d1d660cd4164403f1cd9453291` |

The superseded responsive behavior was `hidden sm:flex`: the four History outcome links were not rendered visibly at 390px, so the registered defaulted-filter action could not be operated. All four existing links are now visible, wrap naturally, retain their `min-h-11` target size, and remain keyboard and pointer operable. The page continues to use its existing query-string state, selector registry, local returned-result set, 200-row API request bound, and `preserveRoutedTestMode` routes. Search scope, separate filtered-empty/no-history copy, and the distinct answered/defaulted/expired states remain intact. No API, storage, registry, runtime/provider, locale, shared-shell, Delivery-case, or unrelated-page source was authored. The managed scenario lifecycle refresh updated the served build artifacts; no provider was restarted.

The two History cases retain `routed_isolation: "true"` and every original unique assertion and label. Their existing History checks still cover the closed answered fixture row, the `Answered` status, the 200-row disclosure, mobile access to Defaulted, and the filtered-empty message. The light case retains keyboard focus/Enter activation and the Settings light-theme route; the dark case retains pointer activation and the Settings dark-theme route. The explicit `empty-open-asks` fixture prevents the prior `asks-read-failure` fixture from leaking into these routes. It retains the closed answered row while providing no defaulted row, so the registered filtered-empty assertion tests the intended local filter result rather than a stale 500 response.

## Validation evidence

Focused History test command:

```text
cd scenarios/notification-hub/ui
pnpm exec vitest run src/features/asks/HistoryPage.test.tsx
```

Result: 1 file, 5 tests passed. The existing filter test runs at configured widths 390px and 1280px; it checks visibility, minimum target height, keyboard activation, pointer activation for each other outcome, and restoration of all returned rows. Receipt: [focused-history-tests.log](focused-history-tests.log).

The changed-input Test Genie command was:

```text
vrooli scenario test notification-hub --phases workflow
```

Run `20261021-002625-9dc0246f` completed `PASS` at `2026-10-21T00:27:21Z`; workflow phase passed, no findings, L5 clean. Its native Workflow Health evidence reports 11 selected, 11 executed, 11 passed, 0 failed, 0 refused, and 0 skipped. Routed isolation was installed under lease `cd431b0d-9c82-4f69-9434-5917b639ec3f`; `test_pool_requests=303`, primary test-mode requests/writes `0/0`, test-root writes `0`, and install/heartbeat/clear errors were empty. Receipts: [test-genie-workflow-status-final.json](test-genie-workflow-status-final.json), [test-genie-workflow-artifact-final.json](test-genie-workflow-artifact-final.json), [test-genie-workflow-native-detail.txt](test-genie-workflow-native-detail.txt), and [test-genie-workflow-findings-final.json](test-genie-workflow-findings-final.json). The phase artifact's catalog reports 11 cases and no stale registry paths.

The final selected Workflow Health run was `9f3ec21d-3c9e-443d-bbb7-e3d9ea4c60af`, idempotency key `e10-history-phone-filters-20261021-002`, against served build `sha256:215faa8fd30dc303a25a35bb717477adb85257256f0e1ec815e6426c3354b14c`. It selected and executed the existing four cases, with 4 passed, 0 failed, 0 refused, 0 skipped:

| Case | Result | Execution ID |
|---|---|---|
| `delivery-dark` | passed | `9e903255-ba06-4623-b119-a88690ffdbd6` |
| `delivery-detail` | passed | `9a5e61d2-c325-4de5-a664-7147766daedb` |
| `history-bounded` | passed | `fb36c02e-df5a-4a77-811d-346407d948f3` |
| `history-dark` | passed | `6bf6e652-ddd9-4b87-b950-155fd30ed8c8` |

The routed lease ID was `9f3ec21d-3c9e-443d-bbb7-e3d9ea4c60af`, installed `true`; `test_pool_requests=158`; primary test-mode requests/writes `0/0`; test-root writes `0`; install/heartbeat/clear errors were empty. No manual lease clear was used. The run receipt is [workflow-health-wait-after-case-fix.json](workflow-health-wait-after-case-fix.json); the four copied `latest.json` and `timeline.json` records are under [workflow-health-final](workflow-health-final/). The terminal Workflow Health record reports every case completed successfully with no refusal or skip. Its empty `clear_error` is the provider cleanup receipt.

The served scenario was healthy at the time of final verification on UI port 20001 and API port 17736 with the same build identity; see [current-build-status-final.json](current-build-status-final.json). No unchanged API Go `unit` phase was repeated. No provider restart, manual lease clear, provider-scope expansion, or parallel validation was performed.

## Served phone effect and visual comparison

The captures are from the served History route at CSS viewport 390×844, DPR 2 (780×1688 raster), on build `sha256:215faa8fd30dc303a25a35bb717477adb85257256f0e1ec815e6426c3354b14c`:

| Served capture | SHA-256 | Observed page effect |
|---|---|---|
| [history-light-390x844.png](history-light-390x844.png) | `ede9c518b0d8d7fd673f7dc6e11e54054b77bee128f88ecd5760fc09d1f8ccdc` | All four outcome links show above the returned answered row and 200-row disclosure. |
| [history-dark-390x844.png](history-dark-390x844.png) | `c25b54de12fcf76b3e7c87487ce1fc2bf1f1200989bc259859e60efb6a3dca1` | Same visible outcomes and bounded History content in dark theme. |
| [history-bounded-filtered-empty-390x844.png](history-bounded-filtered-empty-390x844.png) | `46e389fdf821fb6baded9c4e75f5380552e088a487153fb31ef2036db6f5e47e` | Final light BAS run shows Defaulted selected with visible keyboard focus, the filtered-empty copy, 200-row bound, and Clear filters. |
| [history-dark-filtered-empty-390x844.png](history-dark-filtered-empty-390x844.png) | `03f2bb4af461125d9eb16811c241e944e55fee2975a986326cd7c884061bdaa4` | Final dark BAS run shows Defaulted selected, distinct filtered-empty copy, 200-row bound, and Clear filters. |

The selected NH16.2 originals were hash-verified before comparison: light `9970ebfb97c4db28d9cbfca334e4de27fa7fa54ac55418a8636694c6e0eb4aa2`; dark `5cb51426dbea45de4c6e3d9895016f377ec9653b0418375887490acdfd27cfec`. They establish the latest-200 bounded History meaning and distinct returned History rows. The served page retains that meaning with its explicit “History shows up to 200 returned decisions. Filters apply only to these loaded results, and older decisions may be omitted.” disclosure, returned outcome/time/source content, and visible local search/outcome controls. Its card layout and search/filter controls are the current page composition, not an assertion of pixel-identical reproduction. The NH06 dark/light mobile behavior references were also hash-verified (`ff33537f…61ea3d` and `3d49f2d1…09bdd`) and show the visible loaded-set filter behavior now present on the served page.

Shared app-shell/navigation adaptation is separate from page content: the served app has a large separate “Notification Hub” header band and persistent five-item bottom navigation, while NH16.2 shows a compact header with a Menu affordance. The page content begins below that shell in both, and the bottom navigation is not part of History's list or filter controls. At 390px, the four filter links fit on one row and the page's bounded disclosure remains legible above the persistent nav.

## Work, waits, and interventions

1. The admission anchor and prewrite inventory were recorded before changing the admitted files.
2. The first registered run exposed that the source edit was not yet in the served bundle. One managed `vrooli scenario restart notification-hub --timeout 300 --json` refreshed the app build and health-checked it. That produced the build identity above. It was a scenario lifecycle operation, not a provider restart.
3. The first post-build four-case Workflow Health selection passed. A subsequent full Test Genie workflow run (`20261021-002247-670500d3`) failed only the two History filtered-empty assertions; its trace showed requests inheriting the earlier routed `asks-read-failure` fixture. The History product fix was working, but these sequential BAS cases needed to select their own existing fixture.
4. Both admitted History cases now explicitly select `empty-open-asks` in their routed Settings and Updates URLs. This corrected fixture isolation without changing any assertions or labels. The final four-case Workflow Health run and the following full Test Genie workflow run both passed. The final TG native receipt reports all 11 registered selected cases passed.
5. After the TG run, the existing focused local test was parameterized at 390px and 1280px and expanded to exercise each distinct outcome. The focused History test was run once after that test-only adjustment and passed 5/5. The product page and BAS case inputs covered by the TG run were unchanged by this test-only edit.

No API/unit phase was repeated, no provider was restarted, no lease was manually cleared, and no additional provider scope was introduced. The TG and Workflow Health server-owned waits completed terminally; the final lease and counters are recorded above.

## Independent review handoff

Please return this evidence to the independent parent reviewer for G2/G3 review under parent run `11175250-c50d-44e9-9af5-cd6eba79bab2` (verified by `agent-manager run get`; currently parked on its E8 child). Review the four served captures against the hash-verified NH16.2 originals and NH06 behavior references, and review the routed case isolation, bounded result semantics, keyboard/pointer/touch target sizing, and preserved BAS assertions. This worker does not accept the slice. The final source/evidence remains under the admitted paths, and the broader E4 and whole-redesign gates remain open.
