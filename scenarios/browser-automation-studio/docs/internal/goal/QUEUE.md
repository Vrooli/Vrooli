# BAS queue

Order from the operator decisions (D19, D20, D23, D25). One line per slice:
**enabling** marks a slice that unlocks others; deletion targets are what must be
gone at close. Line numbers are static estimates; the first epochs re-measure them.
Each epoch file lives in `epochs/`; the orchestrator moves a slice to Done with its
`ACCEPTED` line.

## Forecast

- Destination: runtime inventory ≤205,000 lines. Current no-Git inventory at
  2026-10-02T08:07:40Z is 257,875 (digest
  `863d483234eaf15657a1e6a0ab4b79afb57e80f2396d56b497decafdfab03375`), leaving
  a 52,875-line gap. E19/S13 is accepted at −978 runtime lines from E18.
- Last five accepted epochs against their briefs: E15 −653/−1,300; E16 0/250-line
  feature ceiling (not a reduction estimate); E17 −1,394/−2,000; E18 −333/−1,500;
  E19 −978/−3,000. The four directional refactors removed 3,358 of 7,800
  estimated lines (43.1%).
- The six still-planned module slices identify approximate remaining gaps of
  Workflow 14,800, Recording 11,867, Workspace UI 7,600, Export 7,000, AI 6,900
  and Diagnostics 6,000 lines (54,167 total). At the recent ratio this forecasts
  about 23,320 lines, roughly 29,555 short of the destination. This remains a
  planning finding: the queue cannot reach 205,000 without additional measured
  deletion slices and a rebase of category scopes.
- First path-level inventory and E18/E19 outcomes remain in the accepted history
  below; category scopes do not reconcile to the older module-table baselines.
- Initial no-Git scope census on 2026-10-01, using the inventory's runtime
  exclusions: Workflow's `api/services/workflow`, `api/automation` and
  `ui/src/domains/workflows` total 32,349 lines/148 files against the historic
  40,800 budget; Recording's API service, UI domain and driver roots total
  39,124/144 against 36,200. Export's API service/handler and UI export roots
  total 11,119/54 against 29,000, while the sampled AI service/handler/UI/driver
  roots total 10,223/48 against 17,900. The sampled Diagnostics roots total
  977/4 against 7,000. The Workspace UI shell sample is only 2,977/11 across
  layout, onboarding, workspace and app roots, so it does not cover the broader
  historic category. These are candidate path totals, not removable-code or
  owner counts; the mismatches show the old budgets cannot set new deletion
  estimates without a complete path-to-owner reconciliation.
- Expanded candidate roots after the S13 acceptance: Workflow is 37,229 lines
  across 170 files (`api/services/workflow`, `api/automation`, workflow handlers,
  API validator and UI workflow domain); Recording is 40,066/149 when its API
  handler/internal paths and record-mode view are included; the UI shell sample
  (`views`, `shared`, `components`, `contexts`, `hooks`) is 27,886/191; Export is
  21,114/106 with the exports service and UI exports/execution-export domains;
  AI is 10,777/50 with vision-navigation; Diagnostics is 4,146/17 after adding
  driver observability and UI observability. These candidate roots still combine
  canonical code with potential duplicate owners, and the shell overlaps product
  domains; their totals are not additive and are not deletion estimates. Next,
  compare cross-boundary transforms/builders and their callers to name one owner
  and concrete retirement paths per slice before adjusting the budget forecasts.
- Pre-WF1 owner audit progress (2026-10-01): a fresh no-Git source census with
  the runtime inventory's language, test and generated-file exclusions measured
  33,973 lines/158 files in the selected Workflow roots, 44,980/167 in Recording,
  27,886/191 in the Workspace UI shell, 18,296/86 in Export, 11,253/53 in AI,
  and 1,211/5 in the currently selected Diagnostics roots. These selected roots
  are not disjoint modules and are not estimates of removable code. Their
  differences from the earlier candidate census show the need to reconcile each
  path to a single owner before using historic module budgets.
  The Workflow scan found one shared proto-JSON normalization path:
  `converter.go` delegates to `NormalizeWorkflowDefinitionV2Bytes`; capture,
  CLI and proto conversion callers also use that adapter. The separate
  `BuildFlowDefinitionV2ForWrite` path decodes typed write payloads and validates
  node/edge references. UI `normalizeNodes`/`normalizeEdges` are consumed both
  when the builder loads editor nodes and when the workflow store projects API
  responses into editor state. These are different ingress/presentation duties;
  current caller evidence does not establish duplicate work or a deletable
  owner. Recording's API service selects `GenerateWorkflowFromTimelineEntries`
  or `GenerateWorkflowWithPages` according to the available input; they consume
  different input forms and both produce typed V2 workflows. The UI timeline
  adapter projects the canonical proto for display. E17's unified timeline and
  E18's controller split are already accepted and cannot be counted as future
  reductions. Export's E19 deletion list is complete;
  `BuildReplaySpecFromExecution` delegates to `BuildReplaySpecFromTimeline`,
  while `BuildReplaySpec` merges an optional client contract with server output.
  These have distinct call roles; no duplicate spec builder has been
  established by current caller evidence. E13's accepted single AI engine
  remains in place; the driver vision agent orchestrates navigation, its Gateway
  client performs inference, and the API vision-navigation handlers own the
  user-facing job/callback surface. These paths are not interchangeable owners
  based on current references. Diagnostics still needs a wider path map: the
  live roots include API observability handlers (about 477 lines), API sidecar
  health (536), CLI observability (38), driver observability (1,532), driver
  telemetry (1,371), the driver diagnostic logger (381) and recording
  diagnostic routes (353). These totals overlap the broader telemetry/health
  domains and combine data collection, health state and diagnostics transport.
  The call graph shows the API observability handler proxies the driver's
  observability service; the driver recording-diagnostics routes handle debug
  and external-URL injection checks; the session diagnostic logger attaches
  browser-context logging; and sidecar health monitors process readiness. These
  have different inputs and lifecycles, with no safe shared owner or reachable
  deletion amount established by this audit. The existing 6,000-line
  Diagnostics delta remains unsubstantiated by this census.
  The Workspace UI candidate roots contain one `RootLayout`, `AppShell` and
  shared `Sidebar`; they also contain the dashboard, settings and onboarding
  features represented by the largest files. The census has not identified a
  second shell owner or a feature retirement path. The narrow Diagnostics
  census also includes only
  selected observability and recording-diagnostic paths; other live health,
  telemetry and observability roots still need to be mapped before its 6,000
  line historic reduction can be treated as reachable. No concrete deletion
  estimate is admitted from this partial audit.
- Follow-up feature-level search (2026-10-02):
  `api/internal/protoconv/enum_convert.go` is a 197-line deprecated re-export
  layer whose functions delegate to the canonical `api/internal/enums` owner.
  Production references outside that file are limited to three callers in
  `internal/protoconv/convert.go` and `internal/protoconv/workflows.go`; the
  other 27 wrappers have no production call sites in the source scan. Moving
  those three callers to `internal/enums` would make the compatibility file
  deletable, for roughly 170 net runtime lines before any test/support changes.
  This is a concrete but small contributor (about 0.3% of the 52,875-line gap),
  not a standalone epoch or a forecast fix. Include it only with a larger
  compatible proto-conversion retirement slice after its package/test callers
  are checked; no change is authorized from this census alone.
- Recording UI orphan-surface search (2026-10-02): nine candidate files in the
  selected tree are referenced only through recording-domain barrels or their
  own orphan parent. `TimelineFullView.tsx` (1,140 lines),
  `useRecordModeLayout.ts` (260), `MultiPageTimeline.tsx` (225),
  `MultiPageTimelineEntry.tsx` (264), `TimelineSidebar.tsx` (365),
  `ActionTimeline.tsx` (965), `SelectorEditor.tsx` (343),
  `FloatingActionBar.tsx` (258), and `shared/VideoPlayer.tsx` (254) total
  4,074 runtime lines. In the current source tree, `TimelineFullView`,
  `TimelineSidebar`, `ActionTimeline`, `SelectorEditor`, `FloatingActionBar`,
  and `VideoPlayer` have no live callers beyond barrel exports; the two
  multi-page timeline components have no caller beyond their orphan pair.
  `ActionTimeline` is reachable only from the uncalled `TimelineSidebar` and
  has a component test. By contrast, `RecordModeWorkspace` renders
  `UnifiedSidebar`/`TimelineTab`, which owns the active selection and
  `UnifiedTimeline` path. A full-repo source scan of UI files over 500 lines
  found no other similarly isolated cluster. The driver's apparent 669-line
  accessibility snapshot was excluded after tracing its live session-manager
  setup and teardown references. The 4,074-line Recording candidate is inside,
  not additional to, the historic 11,867-line Recording delta; other
  independently measured candidates are still needed to support a destination
  forecast that closes the 52,875-line gap. Treat this as a candidate slice for
  planning only until tests, external consumers and the accepted Recording
  budget are reconciled at admission.
- First path-level inventory: `RecordingSession.tsx` is 1,402 runtime lines;
  `ui/src/domains/recording` is 29,920; `playwright-driver/src/recording` is
  8,214; `api/services/workflow` is 7,015; `ui/src/domains/workflows` is 7,111;
  `api/services/export` is 5,875; `ui/src/export` is 1,883; combined API/UI/driver
  AI paths measured 11,761. These path scopes do not reconcile to the older
  module-table baselines, so new queue slices carry the table's measured budget
  deltas but must establish exact owner/file inventories at admission before a
  brief is set.

### Planned deletion-first slices from the module budget deltas

These work packages target the six largest gaps. Estimates are the measured
current-to-target deltas in the module table, not reductions already proven by
the path-level inventory. At admission, bind each deletion list to the exact
duplicate owner paths/symbols and baseline before dispatch; do not lower these
outcome targets to match a smaller deletion.

- **WF-1 — Workflow ownership.** Estimate −14,800. Measured budget: 40,800→26,000.
  Deletion list: retire duplicate workflow transformation/normalization owners
  across `api/services/workflow`, `api/automation` and
  `ui/src/domains/workflows`; keep one canonical workflow model and remove stale
  callers. First measured files include `ui/src/domains/workflows/builder/WorkflowBuilder.tsx`
  (1,006), `ui/src/domains/workflows/utils/normalizers.ts` (795),
  `ui/src/domains/workflows/builder/NodePalette.tsx` (677),
  `ui/src/domains/workflows/components/ElementPickerModal.tsx` (724),
  `api/services/workflow/executions.go` (829), and `api/services/workflow/sync.go`
  (517); exact obsolete symbols must be named in the admitted brief.
- **REC-1 — Recording UI ownership.** Estimate −12,200. Measured budget:
  36,200→24,000. Deletion list: remove duplicate recording workspace/timeline
  orchestration and stale adapters; first measured targets are
  `ui/src/domains/recording/RecordingSession.tsx` (1,402),
  `ui/src/domains/recording/timeline/TimelineFullView.tsx` (1,140),
  `ui/src/domains/recording/timeline/ActionTimeline.tsx` (965), and
  `playwright-driver/src/recording/capture/browser-scripts/recording-script.js`
  (1,545); E18 owns the first controller slice.
- **WSUI-1 — Workspace shell ownership.** Estimate −7,600. Measured budget:
  29,600→22,000. Deletion list: retire duplicate page-shell/navigation state
  owners and leave one shared workspace shell; initial large-file census includes
  `ui/src/shared/onboarding/GuidedTour.tsx` (1,267) and
  `ui/src/shared/layout/Header.tsx` (1,126). Reconcile those files to the
  workspace budget before admission; do not overlap E18's recording controller.
- **EXP-1 — Export pipeline ownership.** Estimate −7,000. Measured budget:
  29,000→22,000. Deletion list: remove duplicate export-spec construction,
  timeline projection and replay-rendering owners; measured paths include
  `api/services/export` (5,875 runtime lines), `ui/src/export` (1,883),
  `api/services/export/html_bundle.go` (1,133), and
  `ui/src/export/ReplayExportPage.tsx` (1,136); admitted brief must name retired
  symbols and callers.
- **AI-1 — AI navigation ownership.** Estimate −6,900. Measured budget:
  17,900→11,000. Deletion list: retire duplicate navigation/client and UI command
  owners; measured API/UI/driver AI paths total 11,761 lines, including
  `playwright-driver/src/ai/vision-agent/agent.ts` (1,063),
  `playwright-driver/src/ai/vision-agent/loop-detection.ts` (776), and
  `playwright-driver/src/ai/action/parser.ts` (603). Reconcile the difference to
  the historic budget before admission; preserve the E13 single-engine owner.
- **DIAG-1 — Diagnostics ownership.** Estimate −6,000. Measured budget:
  7,000→1,000. Deletion list: remove retired diagnostics surfaces after consumers
  move to the canonical observability owner; census and name every diagnostics
  route/module and caller before admission. Current path scan finds only the
  narrow driver files `playwright-driver/src/session/diagnostic-logger.ts` and
  `playwright-driver/src/routes/record-mode/recording-diagnostics-routes.ts`,
  so the historic 7,000-line scope is not yet mapped and must be measured before
  this slice is admitted.

## Next

- E1–E19 have accepted epochs. E3 completed the first S2 tranche and E7 completed
  the BAS-FB-057 remainder. E6 completed S3 under BAS-FB-058:
  all V1 workflows found by scan were migrated, and the six GCT HTTP cases were
  preserved as non-workflows. E2 and E4 record Gremlins as unverified under
  BAS-FB-056.
0. **Planning hold before the next ownership epoch (BAS-SUP-001).** BAS-FB-060
   authorizes continued slice planning and raising estimate misses. The owner
   census and feature-level search now include a 4,074-line Recording UI
   retirement candidate, but the current combined forecast remains about
   29,555 lines short. SUP-001 requires measured deletion-first slices whose
   forecast can close that gap before the next ownership epoch is admitted.
   Keep the 257,875→205,000 destination and uncovered gap visible; do not turn
   gross path totals into deletion estimates. The remaining path to the
   setpoint is not established by inactive UI or compatibility cleanup alone.
   Before broadening into live feature retirement or changing the setpoint or
   budget model, get operator direction. The one authorized supervision wake
   was deduplicated; leave the evidence in this goal home for the supervisor
   read.
1. **DET-SIGNIN-R1 (parked).** Qualify a few real sign-ins with persistent
   profiles. Return condition: the operator supplies dedicated, authorized
   test-only target sites and accounts. Until then, keep sign-ins unverified;
   never use personal accounts or credentials. E16's independent local
   detectability and interactive-profile work is accepted below.
2. **WF-1 — Workflow ownership.** Deletion-first scope and measured budget delta
   are recorded above; admit only after exact owner/file census.
3. **REC-1A — Retire unreachable legacy recording UI surfaces (planning
   candidate, not admitted).** Estimate −4,074 from nine measured runtime files
   listed in the Forecast; confirm export/test references and preserve the
   current `UnifiedSidebar`/`TimelineTab`/`UnifiedTimeline` behavior. This is
   part of the historic Recording delta, not an additive budget. Admit only
   after the overall SUP-001 forecast gate is met. Remeasure the remaining
   Recording scope after this candidate.
4. **REC-1 — Remaining Recording UI ownership.** The historic budget gap is
   about −11,867 after E18; map additional exact retirements beyond REC-1A and
   reconcile the candidate paths before setting an estimate.
5. **WSUI-1 — Workspace shell ownership.** Deletion-first scope and measured
   budget delta are recorded above; rebase candidate paths before admission.
6. **EXP-1 — Export pipeline ownership.** Deletion-first scope and measured
   budget delta are recorded above; exact symbols/callers required at admission.
7. **AI-1 — AI navigation ownership.** Deletion-first scope and measured budget
   delta are recorded above; rebase against E13's accepted owner boundary.
8. **DIAG-1 — Diagnostics ownership.** Deletion-first budget delta recorded
   above; map the historic budget to current files before admission.
9. **DESK — Desktop portability epoch** (D22). When Linux is healthy and a native
   machine is available; macOS through minimouse occasionally.




Every refactor epoch also matures the tests of the modules it touches, under the
same rules (worker card), so test lines fall with runtime lines.

Module budgets (runtime lines, current → target): Session 10.7k→6k, Runtime
adapter 18.4k→14k, Transport 10.9k→7k, Recording 36.2k→24k, Profiles 6.7k→5k,
Workflow 40.8k→26k, Evidence 22.3k→17k, Workspace UI 29.6k→22k, AI 17.9k→11k,
Export 29k→22k, Diagnostics 7k→1k, Config 3.1k→1.8k, other 53k→48k.

## Parked

- **BAS-SUP-001 planning hold** (not an admitted epoch): caller audits found a
  4,074-line unreachable Recording UI candidate and a much smaller enum
  compatibility candidate, while the combined forecast remains about 29,555
  lines short. The remaining retirement amount is not supported by inactive
  paths. Return condition: operator direction to rebase the historical module
  budgets or to authorize retirement of specified live feature surfaces; then
  remeasure, refresh the Forecast and admit only a slice with exact owners and
  callers. No product paths were changed.
- **2026-10-02 check-in:** E19 remains the latest accepted epoch; no child worker
  is active and the team inbox has no new direction. Daily qualification passed
  12/12 applicable Linux shadow journeys (evidence:
  `~/.vrooli/evidence/bas-goal/journeys-2026-10-02.json`). Next action: after
  operator direction satisfies BAS-SUP-001's return condition, remeasure the
  authorized scope, refresh the Forecast and admit a bounded epoch only if its
  exact deletion list and estimate close the destination gap.

## Qualification findings

(Daily journey-run failures land here as queue items.)

- **J11 journey coverage pending.** J11 (AI navigation, stubbed model) moved to a
  driver integration test because the shadow driver reads the model gateway only
  from `AI_GATEWAY_URL` at startup. Restore a shadow journey once the stub gateway
  can be injected without restarting the shadow.

## Done

- **E19 — S13: one replay spec.** ACCEPTED 2026-10-01T18:22:37Z (epochs/E19.md);
  generated protobuf `ReplaySpec` is the sole API/UI contract, the Go reflection
  generator and handwritten schema/build paths are retired, and the editor is
  573 lines. Independent proto verification, focused/full API Go, UI type-check,
  all 13 UI projects and 12/12 required Linux shadow journeys passed. Runtime is
  257,875 (−978 vs E18 baseline 258,853), within zero-growth and about 2,022
  above the directional −3,000 target; test/support inventory is 157,403, with
  exact E18 comparable support delta unavailable. Weighted non-cache tokens:
  2,642,774, usage not final; receipts unobserved. macOS arm64 not exercised;
  macOS x64 and Windows x64 remain unavailable.

- **E18 — S12: split the recording workspace controller.** ACCEPTED
  2026-10-01T15:13:00Z (epochs/E18.md); `RecordingSession.tsx` is now a 13-line
  stable route composition root, with the workspace UI consuming the existing
  execution and recording mode owners. UI type-check, all 13 UI test projects
  (1,152 tests), and 12/12 applicable Linux `bas-goal` journeys pass. Runtime
  258,853 (−333 vs E17 baseline 259,186), within zero-growth but 1,167 below the
  directional −1,500 estimate; epoch-check unit deltas sum −338 (5-line mismatch
  from direct inventory). Worker weighted non-cache tokens 716,600, usage not
  final; receipts unobserved.

- **E17 — S6: one timeline shape.** ACCEPTED 2026-10-01T13:41:30Z (epochs/E17.md); generated proto `TimelineEntry` is the canonical recording/execution persistence and stream contract, with presentation projection at the UI/export edges. All 11 named legacy converters/types and stale callers are absent. API Go suite, full driver suite (130 suites/1,922 tests; 2 skipped), focused recording UI suite (573/573), UI/driver typechecks and 12/12 applicable Linux shadow journeys pass. Runtime 259186 (−1,394 vs E16 260580; within hard no-growth gate), support 157969 (−438 vs 158407); directional −2,000 estimate short by 606. Recorded unit deltas sum to runtime −3,802 and test −45, differing from direct inventory by 2,408 runtime lines and 393 support lines. Weighted non-cache tokens reported: 3,282,134 across two workers; receipts unobserved and usage unknown.

- **E16 — DET local detectability and interactive profile behavior.** ACCEPTED
  2026-10-01T04:30:00Z (epochs/E16.md); BAS-RH-J25 is unique and passes
  individually and in the 12/12 Linux shadow suite, preserving all existing
  journeys. Driver typecheck and the restored 16/16 session-start unit suite
  pass. Runtime inventory is 260580 (unchanged vs E15); non-runtime support is
  158407 (+97). The dedicated authorized sign-in portion remains parked as
  DET-SIGNIN-R1 pending operator-provided targets/accounts. Weighted non-cache
  tokens reported: 312575 across two workers; both receipts mark usage unknown.
- **E15 — S10: one config owner.** ACCEPTED 2026-10-01T03:50:50Z (epochs/E15.md); the API resolves session-scoped options and carries them through session admission/lease state; driver runtime overrides/routes and dead CLI config bindings are retired while process-start parsing remains. API and CLI Go suites, driver typecheck + 162 focused tests, Test Genie tidiness, 11/11 Linux shadow journeys, deletion/caller searches and inventory gates pass. Runtime −653 vs E14 baseline 261,233; support −29 vs 158,339. Weighted non-cache tokens reported 313,629 across three workers; first and final worker receipts remain usage-unknown.
- **E12 — S7: one frame transport.** ACCEPTED 2026-09-30T21:18:00Z (epochs/E12.md); recording and execution share one source-bearing driver protocol, one API hub ingress and one UI hook; latest-frame slots replace, seed late viewers and clear on teardown. API Go, driver typecheck + 102 focused tests, UI typecheck + 558 recording and 4 execution tests, and 11/11 Linux shadow journeys passed. Runtime −299 vs 262,635 E11 baseline (below directional −1,500; within zero-growth gate); test/support delta unmeasured. Weighted non-cache tokens 541,101; receipt says usage not final.
- **E13 — S8: one AI engine.** ACCEPTED 2026-09-30T21:52:00Z (epochs/E13.md); removed the Claude CLI navigator, API-owned Ollama client and E01 UI command hook. Playwright navigation, server tracker and shared role-aware model service remain the owners. API Go, driver typecheck + 67 focused tests, UI typecheck + 42 focused tests, and 11/11 Linux shadow journeys passed. Runtime −1,095 vs E12 baseline 262,336 (below directional −3,500; within zero-growth gate); test/support −898. Weighted non-cache tokens 359,464 (usage not final).
- **E14 — S9: one retention planner.** ACCEPTED 2026-09-30T23:07:19Z (epochs/E14.md); the five policy paths converge on `services/retention`, with transport-only HTTP/Connect adapters and one startup planner. API `go test ./...`, 11/11 Linux shadow journeys, deletion/caller searches, and the E14-scoped tidiness check passed. Runtime −8 vs E13 baseline 261,241; test/support +138 vs 158,201. The current whole-tree Test Genie run still reports coupling 24 vs 23, with all 24 paths outside E14; the global issue is recorded without changing its budget. Weighted non-cache tokens 937,957 across two workers (usage not final).
- **E11 — S5: one recording journal.** ACCEPTED 2026-09-30T20:21:57Z (epochs/E11.md); init-script is the only recording injection path, Go `WorkflowGenerator` owns recorded-action merging, and UI projects the raw journal. All deletions and rerun gates pass, including 11/11 shadow journeys. Runtime −2,417 vs 265,052 baseline (below directional −6,000; within zero-growth gate); test/support −2,540. Weighted non-cache tokens: 374,525 (usage not final).
- **E10 — S1: one session broker.** ACCEPTED 2026-09-30T19:43:16Z (epochs/E10.md); all `/session/` API routes now go through the broker, three duplicate lease-fencing modules and three of four page trackers are retired, and the canonical API tracker remains. API Go tests, 36 focused driver suites (619 passed, 1 skipped), typecheck and 11/11 Linux shadow journeys passed. Runtime −671 vs the 265,723 E9 baseline (below the directional −3,500 estimate); test/support −461. Weighted non-cache tokens 882,295 across two workers; receipts marked not final. A pre-existing typed-action SELECT mismatch is recorded under D1.
- **E9 — JX: journey expansion before session/recording rewrites.** ACCEPTED 2026-09-30T17:48:18Z (epochs/E9.md); added J03/J04/J13/J15/J17/J18 and reran the full shadow suite 11/11. Runtime stayed at 265,723; test/fixture/support +196. Weighted non-cache tokens currently 299,294 (receipt marked not final).
- **E8 — S4: schema-driven node editor.** ACCEPTED 2026-09-30T17:09:26Z (epochs/E8.md); all 35 per-node components removed, behavior preserved through one ActionNode including screenshot/AI/Navigate previews; final UI and 5/5 Linux journey gates passed. Runtime −5,438 vs 271,161 baseline; weighted non-cache tokens currently 238,576 across E8 workers, with Agent Manager marking receipts not final.
- **E6 — S3: remove the V1 workflow compatibility path.** ACCEPTED 2026-09-30T16:07:29Z (epochs/E6.md); six discovered V1 workflows migrated, BAS scan found zero V1 arrays, and six GCT HTTP cases preserved; compatibility modules absent and targeted symbol search clear. Runtime −329 vs −2,000 directional estimate; final-tree Go, CLI, UI, driver and 5/5 Linux journey gates passed. BAS Test Genie workflow-health remains unverified after a queued-provider timeout.
- **E7 — S2 remainder: remove handwritten action vocabulary owners.** ACCEPTED 2026-09-30T15:59:13Z (epochs/E7.md); all eight paths deleted, runtime −2,101 vs the 273,262 baseline (−399 vs directional −2,500), test/support net −607 (including +55 lines of generated-proto regression coverage); API, CLI, UI, driver and five Linux journeys rerun and passed.
- **E5 — Driver, UI and CLI test consolidation.** ACCEPTED 2026-09-30T13:37:42Z (epochs/E5.md); runtime +0, test −711; driver 130 focused tests, UI 59 focused tests including BAS-RH-J24/BAS-AUTH, and CLI `go test ./...` passed. Stryker is unverified.
- **E4 — Further Go test maturation.** ACCEPTED 2026-09-30T13:09:43Z (epochs/E4.md); one state-based test sleep retired (23→22), remaining sites audited as intentional; full Go and coverage gates passed, runtime net 0.
- **E3 — Action vocabulary generated from proto (first tranche only).** ACCEPTED 2026-09-30T13:02:50Z (epochs/E3.md); runtime −449, test/support +187; proto verifier, affected Go/UI/driver checks, and five Linux preservation journeys passed. Test Genie tidiness passed after in-scope cleanup. BAS-FB-057 says S2 is not complete; the named remainder above is still open.
- **E2 — Go test consolidation.** ACCEPTED 2026-09-30T11:52:12Z (epochs/E2.md); runtime +0 and API test −1,330 (69,837→68,507); `go test ./...`, standalone `go test -cover ./...`, and runtime inventory rerun passed. Gremlins is unverified under BAS-FB-056; further Go test maturation was completed as E4/T1-F.
- **E1 — Dead code and diagnostics ownership.** ACCEPTED 2026-09-30T04:02:56Z (epochs/E1.md); net runtime −6,640 and net test −2,178; all required acceptance gates rerun and passed.
- **E0 — Journey suite and local fixture site.** ACCEPTED 2026-09-29T18:35Z
  (epochs/E0.md) at 6/6; 12 work units, runtime +0. The 2026-09-30 cleanup
  replaced the runner with `pnpm test:journeys` (TESTING.md): 5 journeys (J01,
  J02, J07, J08, J23); J11 is pending above.
- **E01 — cleanup selectors, scheduler cancellation, AI navigation hook split**
  (pre-epoch-model work, landed 2026-09-29 after focused checks; S8 and S9 later
  replace parts of it).
