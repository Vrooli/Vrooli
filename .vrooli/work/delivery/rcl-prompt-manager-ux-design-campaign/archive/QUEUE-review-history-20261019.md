# Queue review history archived 2026-10-19

## D36 owner review

D35 native task `4ad7ced9-10df-404c-abb9-693a41f067bc` / run `92d9978c-6356-4202-8cf0-0515ccf23526` was continued in the same interactive session under the original Luna-medium profile and `rcl/E1` lineage. Native Get/List confirm run COMPLETE/exit 0 and task still QUEUED; final-output selection is AMBIGUOUS across eight provider candidates, so use the retained message sequence and artifacts without calling it a unique final report. The D36 baseline manifest is `/tmp/rcl-e1-d35-4ad7ced9-10df-404c-abb9-693a41f067bc/baseline-d36-20261018T203535Z/manifest.json`, SHA-256 `3d617496df4e172008c5f169352abc5f8b9e423116dddf1097fb11c87ab9ca4d`. The retained diff `/tmp/rcl-e1-d35-4ad7ced9-10df-404c-abb9-693a41f067bc/D36.diff` SHA-256 `465854df556589da4e6022ea1f0b2db26af72265b4dfaf7559df7012f1b2a35d` contains two canonical declarations, only two root `catalogId` changes, and six matching edge remaps with reasons intact, plus the E1 record. Independent current hashes match the exact 10 authored source paths; new declarations match legacy metadata except canonical IDs. The test output is `D36-TestLiveCatalogValidation.out`, SHA-256 `a0ff30f5951d1107a3ad6ac7194a94eee4e6e3d91c2a092167baaad6a8347ff9`; command failed exit 1, 13 findings, 10 detail records printed. Preserve this as a failure, not a D36 validation pass.

Coverage source review confirms catalog validation filters legacy archive rows while coverage loads them and can count them planned-unbuilt; no declared policy/test was found. The D36 source diff preserves legacy declarations. No coverage-policy change is claimed.

Native accounting is uncertain: `run tokens` reports 105,952 weighted non-cache and 1,062,528 cache-read with usage not final; `run report` reports 6,218,562 tokens; original D35 goal report reports 101,978. Receipts count is 1 but source attribution is not resolved. Keep all values and uncertainty.

## D37 owner disposition

Task `c2c728d2-0952-48da-a913-e14ad00baed8` / run `32de3ff9-1009-47c6-885f-584b2abdabfd` was admitted under the original parent, profile, Luna-medium model, and `rcl/E1` effort. Get/List: task remains QUEUED; run COMPLETE/exit 0 with unique terminal goal-complete output. Parent admission transcript is `/tmp/rcl-e1-d37-c2c728d2-0952-48da-a913-e14ad00baed8/parent-admission.txt`; exact three-file preimages, diff, posthashes, command/output and identity are retained in the same directory. Diff SHA-256 `d6fd7c68d13c1fce234259575d2f9729fb07ac79e898f5f69d080382ecc40807`; preimage manifest SHA-256 `8e2f38fecf14336c44735d80e96aef9701cad13e46ad215c5413928a68bd1e54`; posthash-list SHA-256 `664f6f41549526477f691c9377145d95c33674080cd44c4797d08f1c8f2fdc19`. Independent parent review found the changed catalog dependency semantics supported by current Badge, Layer Manager, outside-interaction and Presence declarations; no implementation binding or released dependency change is claimed. Focused output SHA-256 `dff000909579ba85171a417abe1493b35ea5a0db296030da81354125a97a4330`: `ok react-component-library/internal/catalogvalidate 0.141s`, exit 0. The effective test selector omitted the trailing `$` through shell quoting, but it matches the sole `TestLiveCatalogValidation` function; no skip is reported. Do not repeat it. Native accounting is uncertain: Get `tokens_used=2438432`; run report 4,876,864 tokens; 29 execution outcomes unknown, receipts degraded/0, diff unavailable. See E1 for full review and preserved hashes.

## D38 interim owner review — 2026-10-18T21:00Z

D38 task `c47e1f65-520b-4dc0-b91e-f8003be07796` / run `9527f8ee-7403-46a2-9c1e-3743e6dfa388` is still live (run RUNNING; task QUEUED) under the exact original worker profile, Luna medium, `rcl/E1`, and parent. Its one-line source change correctly uses the custom Select for `danger`, and the assigned test passes. The whole focused file has 9/10 pass, 0 skips: another existing test fails at `voiceInput` for the same duplicate hidden/native custom-control label. The same active task is continuing this same file under D39 to correct that fixture interaction. Exact outputs/diff and independent review are recorded in E1.

The exact-file ESLint result is 14 copy-query errors and 0 warnings, all on unchanged lines. Keep that separate from D39 and do not claim it explains the full type-check command without its surfaced diagnostic. The step-back signal (61 lines/32 threshold) is answered by keeping work to one file and one active child; no second worker or wider task is admitted.

## D38/D39 final owner review — 2026-10-18T21:03Z

The same D38 task/run is terminal: run COMPLETE at 21:02:11Z, unique terminal task-complete candidate `255e82dd-b49b-47fc-843f-e30d1f3a05ac`; native task remains QUEUED and is not forced. Original worker profile/model/workload and effort reference were confirmed; run tokens are 169,827 weighted non-cache, usage not final. Independent preimage comparison shows only `PropsExperimentPanel.test.tsx` changed: both actual custom Select interactions now select `permission-denied` and `danger`. D39's focused file result is PASS 10/10, 0 skipped, exit 0, output SHA `2d2fe02a396ad11c7e769de3659908fd5fd04479d7f637c29df9e1248915dd2a`; checkpoint diff SHA `7d0c202603f73090442cfc028e8a7462fb787dfd508510162124fc0e3471907b`; postimage SHA `7f7b73a9021c18d8a944e4e60f01e53854c90562c44d1ac9d23efa7dd51a90af`.

D38's exact-file ESLint output is pre-D39 and reports 14 copy-query errors/0 warnings; current count is unknown because D39 changed one queried line. The original Test Genie unit run remains FAIL and its UI type-check blocker is `eslint.js exited with 1`. D40 will first measure current exact-file findings, then repair only remaining query-policy violations in this file and validate its focused file/lint. None of this changes G5/G7/E1/E2 status.


## Superseded CardShell duplicate handoff text

## Owner handoff — CardShell draft correction

Acknowledged the direct correction to `ROOT-20261019-RCL-CARDSHELL-COLLISION-HANDOFF`: reuse `react-component-library:CardShell@1.0.10-draft.1` (latest release `1.0.9`); the mistaken invocation made no source-content or release change. Next owner action is one bounded source-only repair/diagnosis of the two CollectionList stylesheet-key collisions against the existing draft. BAS live-capture GCT freeze is active: no build, publish, or shared package/dist change until release. Preserve the 85% branch floor, seven historical provenance gaps, and separate `useLongPress@1.0.0` issue.

D45 dispatch: native task `c932d65b-6af9-4ead-9f44-e831783c9b6f`, run `ec17618a-25c4-41e8-bebc-f1aeea2e6e15`; original parent `2b16a068-2bc1-4a41-8629-54c4992c93c6`, profile `4c19aef1-36a2-4cc5-8961-42648692883b`, Luna medium, INTERACTIVE, `rcl/E1`. Admission returned PENDING/QUEUED. Source-only draft task is permitted during the BAS freeze; no tests/builds/package/dist/publish/checks. Resume after child terminal event, independently review exact result, and retain focused validation until explicit freeze release.


## Superseded detailed goal handoff before D45

## Handoff

- Current owner: original E1 parent `2b16a068-2bc1-4a41-8629-54c4992c93c6` / task `62defa00-aa37-4108-b5c8-ffa5d924cd00`. Operator chose `route_catalog_owner` for shared lint; sync recorded it. Root owns catalog repairs. No replacement identity.
- Received `ROOT-20261019-RCL-UNIT-PAIR-DETAIL-CORRECTION`; appended its bounded RCL UI stderr findings under Current below, preserving truncation limits.
- Acted on `ROOT-20261019-RCL-UI-21-FAILURE-REPAIR`: the scoped run found 20 (prior expected 21); full logs/names are in `evidence/D44-20261019/`. Fixed 18; two imported CardShell collision failures and branch coverage 81.58% remain; type-check passed. Parent reviewed the patch. No gate or acceptance changed.
- Received `ROOT-20261019-RCL-UNIT-PAIR-RECEIVING`; final unfiltered pair failed, so no G5 acceptance. RCL `20261019-210232-e28ecb5b`: 474s, 0 skips, 57 findings (2 errors/55 warnings). API Go failure: `internal/gates.TestLiveReleaseProvenanceCoversEveryMaterializedRelease`; materialized `hooks/useLongPress@1.0.0` lacks valid publish/backfill provenance; package coverage 59.1%. UI `EmulatorChrome > persists viewport selections and applies zoom as a visual transform`: `Value "0.9" not found in options`; functions coverage 82.78%, branches 80.04%, both below 85%. PM `20261019-210232-4a598560`: 259.5s, 0 skips, 56 findings (1 error/54 warnings/1 info). `stands.test.ts:21:23` spacing assertion fails: seed 42 observed 12.566562233486268 vs `>14.229844695235323`; seed 1337 observed 14.985384535098552 vs `>19.253472594569843`. Existing PM intake `knw-1792276150063534652`; no duplicate issue or historical attribution. Evidence directory `docs/internal/delivery-reliability-research/sources/A248logs/current-catalog-lint-20261019/final-unit-pair-20261019/`: wait-all SHA `9dc93bd46449121c7b7f54271344d5d4df4ddbe0839c1c1d8937430f6cd95a0e`; RCL/PM phase JSON SHA `76a9880f9c35ee0cb56903f40109eb662d2599358b03838cfb5a7e421bc145af` / `e577bfe84aff334db70cff93a86d9de432ba6d402fd161d387f38fcf687b9ccb`; failure evidence SHA `b9a08d7def4073ebdcf15974af6c11e5fc5a84ee26cf3d71b3027abcde49c08d` / `debd1fc072b2d8da3c643942c27469535a6d0437ac3f22bc8f5159960c814548`. RCL diagnostic omits 10,961 output bytes. Do not invent useLongPress provenance or mutate immutable releases.

- D42/D43 independent review and exact worker/accounting records remain in E1’s typed Slice log; both focused checks passed, and neither is G5 acceptance. Do not overwrite their historical task/accounting states.
- Compacted: superseded D30–D41 narrative moved verbatim to `epochs/../archive/E1-D30-D41-narrative-20261019.md`, SHA-256 `ad67b0499fcf8b3eb4f60005e425fa171d7a6161eb36eb31365c8e8b64715d9f`; active E1 is 112,416 bytes. Gate definitions, amendments, directives and all typed slice-log records remain. The 63/32 overrun is not declared cleared.
- Remaining: G5 is failed by the final pair recorded above; obtain new full/unscoped RCL and Prompt Manager `unit` receipts only after their exact causes are repaired. Preserve the earlier 54-error/21-warning shared-catalog diagnostic at `/tmp/rcl-e1-lint-20261019/catalog-lint-report.json` (SHA-256 `74bbd9f77f5f0e8504a6850c93ca41b588048b59a575351e9afe91c181b8e39c`) and the newly observed useLongPress materialized-release provenance failure as separate evidence. Operator decision `RCL-E1-CATALOG-LINT-SCOPE-20261019` remains `route_catalog_owner`, synchronized through Agent Manager; root owns shared-catalog repairs. Do not fabricate publish/backfill provenance or edit immutable releases. G7 D12/D13 provenance remains missing/unwaived; E1 PARTIAL/unaccepted; E2 unaccepted. Preserve D6 and E3's separate accepted status.
- External return condition: root completes the governed shared-catalog repairs and supplies an explicit changed-input/validation handoff. This parent then reconciles current goal-home evidence and considers the required fresh full/unscoped RCL and Prompt Manager unit receipts once each. No broad unchanged rerun, release edit, gate waiver or acceptance.



## Archived exact A176 unit receipt details

## Latest G5 receipt — 2026-10-18T20:53:49Z

Test Genie RCL unit run `20261018-205120-d92852bf` terminal FAIL: 1 phase failed, 0 passed, 0 skipped, 148 seconds, 58 findings (3 errors, 55 warnings). The retained findings artifact is `/tmp/rcl-e1-unit-20261018-205120-findings.json`, SHA-256 `0862fd998afa5177bc5e69344f04c2941f6b69bea21d87dd12233e60d6444c5a`; native artifact id `artifact_39a27b88fd664423023669936bf1e310`. The command-output diagnostics are explicitly partial despite zero omitted failure events; do not treat them as complete execution proof.

- API `go test -json -count=1 -trimpath -covermode=atomic -coverprofile=coverage.out ./...` failed. Named diagnostics include affinity calibration non-discrimination (`catalog.affinity_compatible` absent), traversal test expecting `react-component-library` at its cwd, proto parity expected 1/actual 0, CollectionList arbitrary utility class, unavailable navigation-context catalog sources, unproved `experience-surface` binding, and live manifest metadata/kit vocabulary/dependency-rank/release-provenance findings. Preserve every exact finding in the artifact; these causes remain distinct.
- UI `pnpm run test:coverage` failed. One retained failing test is `PropsExperimentPanel` / “supports scalar control kinds and reports invalid advanced JSON”: line 124 calls `selectOptions(getByLabelText("tone"), "danger")`, which finds multiple elements for the RCL custom Select. Global coverage was 82.78% functions and 80.04% branches against the unchanged 85% threshold. This is the next shrunk cause only.
- UI `pnpm run type-check` failed because `catalog-conformance.mjs` reports `eslint.js exited with 1`; this artifact does not expose the underlying ESLint diagnostic, so cause/path remain unverified. Do not infer this is the same UI test issue.

The required unit receipt remains FAIL; the two earlier focused locale and ThemeSwitcher passes and D37's API catalog validator pass do not replace it. Keep G5 FAIL, all 55 warnings, G7 missing/unwaived, E1 partial/unaccepted, E2 unaccepted and D6 pass.



## Superseded BAS freeze hold before ROOT-20261019-RCL-CARDSHELL-RESUME-RELEASED

## Step-back

D45 epoch-check still reports the inherited E1 overrun: 63 work units against estimate 16 (2.0x threshold 32). This is accumulated log hygiene, not a product or acceptance result. Choice: shrink D45 to the completed source-only CardShell draft correction, then wait on the active external BAS live-capture GCT freeze before qualifying/materializing the draft and running its two focused CollectionList tests plus UI coverage. No new worker or wider scope while held. D45 did not clear the overrun; E1 remains PARTIAL/unaccepted, G5 failed, G7 missing/unwaived, E2 unaccepted.


- D45 source-only disposition: task `c932d65b-6af9-4ead-9f44-e831783c9b6f` / run `ec17618a-25c4-41e8-bebc-f1aeea2e6e15` reports complete; native report final-output selection is ambiguous (5 candidates, actual model blank, tool outcomes unknown), so accounting remains uncertain. Parent independently verified the sole allowed draft-source diff: `CardShell.tsx` stylesheet key changed from `1.0.2` to exact asset draft `1.0.10-draft.1`, preimage SHA-256 `48025d9d915ddf4a298173ffe7f9a1ebad79ec7be44d6c866e50aae06c07e1b7`, postimage `1827e5681b9bcecdc689be10f55283c65b0875f1c1916bc7f970fa0604876614`, diff `567a046425b28d91d66a38931f5f4dc8b5f10fb1ad4dd0ea2ca15303bee77575`. Source-only review supports the key correction under style-ownership; two CollectionList failures and 85% branch floor remain unverified. BAS live-capture GCT freeze is active: hold build/publish/shared package/dist changes; next action after explicit freeze release is qualify/materialize this same draft, run the two focused CollectionList cases, then one changed-input UI coverage check. Reuse `react-component-library:CardShell@1.0.10-draft.1`; latest release `1.0.9`; no further `draft-begin`.

- [waiting] After the BAS live-capture GCT freeze is explicitly released, qualify/materialize the existing `react-component-library:CardShell@1.0.10-draft.1`, then run the two retained CollectionList tests and one changed-input UI coverage check against the unchanged 85% branch floor. No build, publish, or shared package/dist change before that return. Preserve seven historical provenance gaps and the separate `useLongPress@1.0.0` failure; no G5/G7/E1/E2 acceptance.

## Needs operator

- [external dependency: BAS GCT freeze] Accountable owner: reliability/root. Return condition: report either (A) freeze remains active, so this original run remains parked and CardShell draft is untouched; or (B) explicit release for this RCL scope/window, so the same draft can be qualified/materialized and the two focused CollectionList tests plus UI coverage can run. This is an external state return, not a request for a new grant or human approval. Until outcome B, no build, publish, or shared package/dist change.



## D44 parent review detail archived at D46 checkpoint

- D44 owner disposition after independent parent review: the patch changes only 13 RCL UI `src/**` files (2 runtime, 11 tests); exact diff SHA-256 `e5f0e6c35d42ec04da1e6016a959178daf8fad3e0cb8482c5b2ec1aa7e5aba8e`, baseline manifest `4829be5ce275dda4aeb1f9cfdbade025a60686612f58b8fc1651dacf3c6a2a9e`, postimage manifest `f9c398cedd15a219892abf330ea4e66982376ae9469d3feb8e026c490f48df58`. Review found no weakened behavioral assertions: control changes exercise current accessible Select/Listbox API; ActionLauncher now supports true multi-target workflows using shared Checkbox, with selection/deep-link regressions; dialog initial focus targets slug field and checks input preservation on success/error. Disposable original-source controls fail the intended ActionLauncher checkbox and full-slug regressions (2 failed, 1 passed, 8 skipped). Separate reviewer task `daad8778-486d-4868-9e91-98ba5106d337` / run `35d5eb5a-5948-4954-8615-34e52409c032` did read-only hash/scope/output inspection and completed, but native final-output selection is ambiguous (3 candidates) and no terminal verdict was retained. The non-author parent therefore completed a direct diff/behavior review against the retained pre/post manifests and controls; disposition PASS WITH LIMITS. Current two failures are caused by CardShell stylesheet-key collision in imported library code; a separately scoped governed library-source repair is required. The 85% branch floor remains unmet (81.58%).
- D44 implementation task `568a8c73-436e-4faa-8e18-e6d7f7e03799` / run `2a48efd8-0385-4aa5-bf7e-ffad0dfb120d` is terminal COMPLETE; Get verified original parent/profile, Luna medium, INTERACTIVE, `rcl/E1`, accepted effort revision. One initial UI coverage run found 20 failures (not 21), 557 pass/0 skip; repaired 18. Final coverage: 575 pass, 2 fail, 0 skip; functions 92.68%, branches 81.58% (below unchanged 85% floor). UI type-check exit 0. Full outputs, names, pre/post hashes, diff and controls: `evidence/D44-20261019/`. Final output selection is ambiguous (17 candidates); run report totals 19,485,122 while native token read is 332,354 non-cache/19,152,768 cache, `tokens_known=false`; cost remains uncertain. Distinct read-only review task `daad8778-486d-4868-9e91-98ba5106d337` / run `35d5eb5a-5948-4954-8615-34e52409c032` completed; its final-output selection was ambiguous and it retained no verdict. The original paired RCL UI phase was 21 failed/556 passed; D44 scoped initial coverage was a separate 20 failed/557 passed run.
- **E1:** PARTIAL/unaccepted; G5 remains failed. The final RCL unit run `20261019-210232-e28ecb5b` failed with 2 errors/55 warnings/0 skips: missing valid publish/backfill provenance for materialized `useLongPress@1.0.0`, plus the `EmulatorChrome` zoom selector failure and UI coverage below threshold. Earlier RCL run `20261018-205120-d92852bf` remains preserved separately. Root owns the out-of-scope shared-catalog lint causes. G7 original D12/D13 provenance is missing/unwaived. D6 keyboard/focus independently passed.
- `ROOT-20261019-RCL-UNIT-PAIR-DETAIL-CORRECTION` supplements the captured RCL UI failure tail: `CreateComponentDialog.test.tsx:126` expected submit to become disabled after click, but `waitFor` did not observe that; `EmulatorChrome.test.tsx:79` `selectOptions(zoom, "0.9")` failed because that value is absent from options. The retained UI tail includes `...[truncated]...`, so these are the named findings in that bounded tail, not an exhaustive assertion inventory. The RCL failure evidence separately records 10,961 omitted output bytes, 0 omitted failure events, and 5,520 valid events. Raw `rcl-findings-report.json` SHA-256 `759cd773f7d940af7589fb123bc64bd1ce3fb2dfb16d0657185156f30862813b`; `rcl-failure-details.tsv` SHA-256 `146f8d7f0178c13f809cee045cbe57d487252fc0bb125673b83f3475d7364e1b`. No validation was rerun and no additional UI failure absence is inferred.
- Acknowledged `ROOT-20261019-RCL-UI-21-FAILURE-REPAIR`: current prior receipt reports 21 Vitest failures, but only the final two names are retained in bounded excerpts. One scoped UI-only coverage run is admitted to recover the complete list with full stdout/stderr. The bounded repair will preserve actual expectations and the 85% thresholds, add desired-behavior regressions and old-source controls, then use focused checks, one changed-input UI coverage/type-check validation, and distinct independent source review. No RCL/API full suite, API provenance repair, G5/G7 pass or E1/E2 acceptance is implied.
- **E2:** unaccepted; see [E2](epochs/E2.md).
- **E3:** accepted by original owner; separate from E1.


## Queue details archived 2026-10-20

## Step-back

D45 epoch-check and D47 repeat the inherited 63-work-unit/32-threshold overrun against estimate 16; treat as accumulated-history step-back, not new D47 work. D47 parent review passes with limits for the test-only draft resolver and retained regressions, but coverage remains below 85%. Switch to a bounded E1 preview-state test slice covering current uncovered branches in `useDeviceEmulation.ts` (14) and `useDeviceFilters.ts` (6), with expected-behavior tests and one changed-input coverage check. Do not add arbitrary coverage-only assertions. Preserve the output-preimage gap and all acceptance limits. BAS freeze is lifted. G5 failed; G7 missing/unwaived; E1 partial/unaccepted; E2 unaccepted.

## Archived D36 owner review

Full dated owner evidence and accounting preserved verbatim in [QUEUE archive](archive/QUEUE-review-history-20261019.md).

## Archived D37 owner disposition

Full dated owner evidence and accounting preserved verbatim in [QUEUE archive](archive/QUEUE-review-history-20261019.md).

## Latest G5 receipt — retained failure

RCL Test Genie unit run `20261018-205120-d92852bf` failed (3 errors, 55 warnings, 0 skips); preserve the exact artifact and diagnostics archived verbatim at `archive/QUEUE-review-history-20261019.md` and source evidence `/tmp/rcl-e1-unit-20261018-205120-findings.json` (SHA-256 `0862fd998afa5177bc5e69344f04c2941f6b69bea21d87dd12233e60d6444c5a`). UI coverage was below the unchanged 85% floor. This receipt does not replace the later final paired run or imply G5 acceptance.

## Archived D38 interim owner review — 2026-10-18T21:00Z

Full dated owner evidence and accounting preserved verbatim in [QUEUE archive](archive/QUEUE-review-history-20261019.md).

## Archived D38/D39 final owner review — 2026-10-18T21:03Z

Full dated owner evidence and accounting preserved verbatim in [QUEUE archive](archive/QUEUE-review-history-20261019.md).

## Superseded D46–D49 queue detail

- [done with limits] D46 ran the exact CardShell draft check and both focused CollectionList tests. The check passed and isolated tests passed 2/2, but full UI coverage retained both failures and branch coverage stayed 81.58%. See Current and `evidence/D46-20261019/`.
- [done with limits] D47 task `8c320b33-3be0-4e14-9f70-25dcff57f751` / run `33028232-a75a-449a-a567-b3dc23af380d`: active-draft test routing and both collision regressions pass; UI coverage remains below the unchanged floor at 81.58%. Parent independently confirmed diff scope/hashes and unchanged release/test bytes. Native task COMPLETED, run COMPLETE with selected terminal report `10f3c65c-53e4-41f1-94cb-69f229ec4055`; worker goal reports blocked by residual coverage scope. See E1 and `evidence/D47-20261020/`; this is not E1/G5 acceptance.
- [done with limits] D48/D49 task `1b801e38-ed4a-4b7b-8166-56f24395c757` / run `581dfd22-99ee-4908-a079-aeeb3b9fbdf8`: native run COMPLETE, task COMPLETED, selected terminal result `1e469f8e-21e6-4284-9020-3d8d7b6081c7`; D49 continuation is acknowledged in the same child. Parent review PASS WITH LIMITS: only the two admitted hook test files changed; no production edits; tests 22/22 and 11/11, typecheck exit 0, full override-qualified UI 590/590 across 109 files, 0 skips, branches 446/523=85.27% (85% floor cleared), both named CollectionList regressions pass. Evidence: `evidence/D48-20261020/continuation-D49/`. The first qualified run lost stdout when Vitest cleared its report directory; final run output is retained. D49 immediate test preimages were reconstructed and SHA-256 matched to D48 postimages; direct current hashes verify, while some source entries in SHA256SUMS.txt have unusable relative paths. Accounting remains uncertain: 9,184,862 unattributable tokens, actual model blank, 72/72 exec outcomes unknown, one receipt, no sandbox diff. This bounded result does not pass G5/G7 or accept E1/E2.


## Prior current-state narrative archived 2026-10-20

## Current

- D45 reused the existing CardShell@1.0.10-draft.1; latest release remains 1.0.9. Exact diff and hashes are in E1. BAS freeze was lifted by `ROOT-20261019-RCL-CARDSHELL-RESUME-RELEASED`; probe SHA-256 `201cdfd190fa7cf65b143cd859b7c5ffd5db2aed24da3601cd80dcb5faf7515f`.
- D46 exact draft check 6/6 and focused CollectionList 2/2 passed; full coverage failed 575/2, 0 skips, branches 81.58%. D47 test-mode routing to the active draft made both retained regressions and all 577 UI tests pass, but coverage remains 412/505 branches (81.58%); exit 1 against the unchanged 85% floor. Evidence and hashes: `evidence/D46-20261019/`, `evidence/D47-20261020/`. D47 parent source review PASS WITH LIMITS; coverage output preimage attribution gap retained.
- Preserve original UI pair 21 failed/556 passed separately from D44 scoped 20/557; D44 final coverage was 575/2/0 at 81.58%. Full detail remains in evidence and queue archive.
