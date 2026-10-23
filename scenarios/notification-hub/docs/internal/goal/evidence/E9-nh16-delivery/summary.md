# E9 NH16.4–16.6 Delivery evidence

## Scope and attribution

Task `9baf6197-51ea-4f4b-9c18-e16d4c966e13`, run `ec3032e5-795d-4cbc-9333-c433a38771fa`, parent `11175250-c50d-44e9-9af5-cd6eba79bab2`. Admission anchor `gcta-88cc06181e29`, baseline `52097258290d1bff0191319242d4948e214af9e3`; fresh prewrite anchor `gcta-53af95679bbc` at the same baseline. Existing Hub dirty state was already present at admission. The page/test files and BAS cases were preexisting dirty/untracked; this record attributes only the E9 deltas below.

Product delta: `ui/src/pages/DeliveryDetailPage.tsx` adds explicit no-receipt and unknown-transport phone meanings while retaining the existing `notificationsClient.get` read path. No receipt is not described as unread, offline, acknowledged, or ask/history completion. `Delivered` remains provider/adapter acceptance. Unknown enum values do not imply success; a real receipt remains separately described as acceptance evidence. No `DeliveryPage.tsx` change was needed: its existing bounded list and seven transport meanings already cover NH16.5.

Test delta: `ui/src/pages/DeliveryPage.test.tsx` asserts all seven distinct transport meanings, no-receipt semantics, unknown enum handling and refresh, and actual receipt evidence independent of unknown transport state. Detail behavior tests live in this existing file; no `DeliveryDetailPage.test.tsx` exists. Four existing E4 BAS case files changed only their theme setup route: use visible Settings theme options at phone width, then navigate back to the original tested surface. Existing assertions, fixture, isolation labels and unique coverage remain. Counts are 12 assertions for `delivery-dark`, 12 for `delivery-detail`, 4 for `history-bounded`, and 3 for `history-dark`.

Locale source files `ui/src/i18n/locales/{en,ar,ja}.json` and generated `ui/src/consts/strings.generated.ts` were updated through `pnpm strings:gen`; generated output was not hand-edited. Other Hub dirty changes in these files predate E9 and should not be attributed to this slice. No API, storage, shared package, selector registry, BAS runtime, or Test Genie/Workflow Health implementation was changed.

## Authority and build

Rechecked source fingerprints matched the admitted handoff:

- BENCHMARK `45580f33ca453f0331c5ad50dd50e74edec6f54a6e327c457573b42f1a889325`
- dossier `d147181b1e7ba3b0f1dcaa06c2f2805b45761e4eda621231d217aa597eb17fdc`
- atlas `5893b198f9bb9c9851fd50cc16df1607fb4da5e6a00f27450d61b8d5269ddeb4`
- selected manifest `4365502f3ee6aac508730fbdb44ad3a425a193a3c47282ebfbc82bbcb82953ef`

All eight selected reference image hashes matched the E9 brief. Managed scenario UI remained running; its served assets matched the generated build. Build identity: `index-Bq93zIIo.css`, `index-CvQFnD9P.js`; `dist/index.html` SHA-256 `02a0cf0fdd8c56fd902070247f8fbc69a624e5444a41ca49cc053b5e8d456509`; JS SHA-256 `0c241a4bb13cf830ae55c54d3396b0da779411949e8cdee20f0b43ec366c28d0`. Vite reported a nonfatal large-chunk warning (~764 KB).

## Commands and receipts

- `pnpm strings:gen` — passed (run after locale additions).
- `pnpm exec vitest run src/pages/DeliveryPage.test.tsx` — passed, 6 tests.
- `pnpm type-check` — passed, including strings check.
- `pnpm exec vite build` — passed, with the nonfatal chunk warning above.
- `workflow-health validate scenario notification-hub --json` — passed static validation; registry current, 11 registered cases, no stale paths.
- Workflow Health focused run `c20fee57-66c0-4f10-b0da-106d2bd105c3` — selected all four direct cases on the identified build. Execution record terminal state succeeded, validation result failed: 2 passed, 2 failed, 0 refused, 0 skipped, duration 59.05s.
- Test Genie run `20261021-000654-f5cb52bf` — terminal verdict `FAIL`, 2 phases failed, 0 passed or skipped, 125 seconds. Unit ran 34 seconds and workflow 91 seconds; 53 observations. The unit failure is recorded as `TEST_EXECUTION_FAILURE`: API workspace command `/home/matthalloran8/.vrooli/bin/go test -json -count=1 -trimpath -covermode=atomic -coverprofile=coverage.out ./...` failed. The workflow phase failed on the same two History filtered-empty assertions recorded below. Server-owned wait was invoked once; this terminal receipt was provided by the awaited run. No rerun was made.

Read-only inspection of the durable `unit.log` and `unit.json` artifacts confirmed that the retained evidence contains the finding and summary only; it does not include the Go command's stdout/stderr or identify a failing package/test. Test Genie recorded baseline SHA `52097258290d1bff0191319242d4948e214af9e3`, dirty worktree `true`, and 50 scenario paths modified or untracked. The unit failure's concrete cause and attribution therefore remain unverified. No API or test-runner scope was changed to pursue it.


## Workflow Health diagnosis

The final selected cases were:

- `bas/cases/e4/delivery-dark.json` — passed, execution `16160044-3f8c-4dab-ab87-a10b1e80401c`.
- `bas/cases/e4/delivery-detail.json` — passed, execution `567e84a3-d73e-42f2-b982-ffc0eb34f6c6`.
- `bas/cases/e4/history-bounded.json` — failed at step 9, filtered empty state assertion; execution `c24bdf5e-48cd-4340-b146-ef19c9378c0a`.
- `bas/cases/e4/history-dark.json` — failed at step 7, filtered empty state assertion; execution `f018d0dc-c7ad-45e3-a277-337ad64be5b5`.

Both History failures show the Answered fixture row still present after Enter instead of the expected filtered-empty message. The configured filter input remained at its defaulted value and Enter did not activate it. These are retained History diagnostics; History implementation and assertions were outside E9 scope and were not changed.

Isolation counters from the final run: installed `true`; lease `c20fee57-66c0-4f10-b0da-106d2bd105c3`; `test_pool_requests=154`; `primary_during_test_mode_requests=0`; `primary_root_writes_during_test_mode=0`; `test_root_writes=0`; clear, heartbeat and install errors empty. The command output did not expose a lease-release/cleanup counter, so cleanup is not claimed here.

## Phone screenshots

All captures are from Workflow Health run `c20fee57-66c0-4f10-b0da-106d2bd105c3`, at CSS viewport 390×844, light/dark as indicated. Image hashes:

| State | Capture | SHA-256 |
|---|---|---|
| NH16.4 no receipt, dark | `workflow-health/screenshots/nh16-4-dark-no-receipt.png` | `3e171470ae8a2e29c65f23c6c31c5cf268fd8415462ecefaaf8cbecb11a50ec9` |
| NH16.4 no receipt, light | `workflow-health/screenshots/nh16-4-light-no-receipt.png` | `2f5b57e134a8b6f79837cdac1cf49a5851fa80e8f260e1c18fcef18f6297b784` |
| NH16.5 states, dark page 1 | `workflow-health/screenshots/nh16-5-dark-p1.png` | `867a489e996db326b4257b7462456e1ee54dff3787d056e8758a64cd845ed2f8` |
| NH16.5 states, dark page 2 | `workflow-health/screenshots/nh16-5-dark-p2.png` | `98442420dbe2a557cf70b33ed676321ad9492be93b1a8f020cd74b58a3e9c1df` |
| NH16.5 states, light page 1 | `workflow-health/screenshots/nh16-5-light-p1.png` | `e4ab387de78e65c9bac7173a073939a6f343ab613537db2df343042ffc39c3ff` |
| NH16.5 states, light page 2 | `workflow-health/screenshots/nh16-5-light-p2.png` | `cb69878132130aea078b796a40865899b16af48be4f216c805c906f0c3c9b0b0` |
| NH16.6 unknown transport, dark | `workflow-health/screenshots/nh16-6-dark-unknown.png` | `dd98054c2193e1c49a4f7534be60b33f4ef0ae236644215fd4f0054dbcd1c7b7` |
| NH16.6 unknown transport, light | `workflow-health/screenshots/nh16-6-light-unknown.png` | `3a0ae85f5c9b41e615bdf7526f17ca94d3aad3abc8cdf0c4f0a4e9dddad83a06` |

The screenshot adaptations are the existing global app shell and bottom navigation, compact phone title/action layout, and explicit “Last checked: Time unavailable” because the existing read contract provides no last-checked field. The no-receipt content explicitly separates ask/history status from delivery receipts. Parent should review each capture against the selected originals and these adaptations.

History failure screenshots are isolated under `workflow-health/diagnostics/`: `history-bounded-filter-defaulted-failure.png` SHA-256 `ddf4a11c3b043c15c8870da9a7d1d1f303016b867dbc8833bbed9f926f4e9385`; `history-dark-filter-defaulted-failure.png` SHA-256 `7e50cf146c3fe10662985770e85229110c720212660ca4901ca664e9a259ca40`.

## Gates and handoff

- **G1 test:** focused behavior, type and build checks pass. Test Genie `unit,workflow` failed: API Go test execution failure plus the two protected History filtered-empty failures. Workflow Health independently has two Delivery passes and the same two History failures. G1 is not cleared.
- **G2 custom:** captures exist for NH16.4/.5/.6, both themes, on one identified managed build at 390×844. Parent review of exact references/adaptations remains pending.
- **G3 review:** independent parent review pending. No self-acceptance; E4 and whole-redesign gates remain open.

Work comprised six meaningful units: source/asset and dirty-state verification; Delivery detail composition; directly affected tests and generated locale copy; four BAS theme-route corrections; managed build plus registered Workflow Health captures; and focused Test Genie/unit/type/build evidence recording. Test Genie wait was invoked once and returned the terminal result asynchronously. The separate Workflow Health run took 59.05 seconds. The exact parent was verified as the current owner, found parked on E8 run `3fc276d0-fd2e-4e4a-8fab-deee86ce6504`, and woken after confirming E8 complete. It briefly returned to `RUN_STATUS_RUNNING` and then re-parked with its prior E8-wait summary; no E9 review receipt is available. The previously refused task-description refresh returned HTTP 403 because updates require the verified creator orchestrator and an exact queued undispatched task; it was not retried. No scope change was made.
