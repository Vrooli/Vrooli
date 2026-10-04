# BAS queue

## Retirement candidates

- **Potential Record mode + export retirement bundle:** 58,246 gross runtime
  lines across the disjoint feature-named source roots measured below, versus
  the 49,178-line destination gap. User-visible loss: browser recording and
  timeline authoring/review, plus execution replay/video, HTML and folder
  exports. Callers include the three `record` routes in `ui/src/routes.tsx`,
  `InlineExecutionViewer`'s ExportDialog, the ReplayExportPage bootstrap and
  `POST /api/v1/executions/{id}/export`. This gross figure is not a net saving:
  `api/services/export` also supplies workflow folder reports, and shared
  consumers/dependencies must be reconciled before claiming this bundle closes
  the gap. Operator decision requested: whether to scope a precise retirement
  audit for these two capabilities. No removal is authorized by this
  candidate.
- No other live-feature retirement candidate has a measured whole-owner scope
  large enough to close the gap. See the candidate evidence below.

Order from the operator decisions (D19, D20, D23, D25). One line per slice:
**enabling** marks a slice that unlocks others; deletion targets are what must be
gone at close. Line numbers are static estimates; the first epochs re-measure them.
Each epoch file lives in `epochs/`; the orchestrator moves a slice to Done with its
`ACCEPTED` line.

## Forecast

- Destination: runtime inventory ≤205,000 lines. Current no-Git inventory at
  2026-10-03T04:00:20Z is 254,178, leaving a 49,178-line gap (source digest
  unchanged at `f24b43325db4e13b2d9f9981f42bfefc119087ef6e91df83e4a51a68371b6479`). E20 is accepted
  at +5 runtime lines from E19 and stays within its +50-line ceiling; E21 is
  accepted at −4,081 runtime lines from E20; E22 is accepted at +379 runtime
  lines against its +500 estimate, within its +750 feature cap.
- Last five accepted epochs against their briefs: E18 −333/−1,500; E19
  −978/−3,000; E20 +5/≤50 growth ceiling; E21 −4,081/−4,074; E22
  +379/+500. E20 and E22 are feature epochs; the four directional refactors
  removed 6,786 of 10,574 estimated lines (64.2%).
- The six still-planned module slices identify approximate remaining gaps of
  Workflow 14,800, Recording 7,793, Workspace UI 7,600, Export 7,000, AI 6,900
  and Diagnostics 6,000 lines (50,093 total). Scaling by the recent directional
  ratio forecasts about 32,148 lines, roughly 17,030 short of the destination.
  The queue still cannot reach 205,000 without additional measured deletion
  slices and a rebase of category scopes. REC-1's 7,793 is the historic
  Recording gap after subtracting REC-1A's −4,074 estimate; remeasure before
  admission.
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

## Candidate evidence

- Gross live-feature source measurement (2026-10-03; `refactor_inventory.py`
  runtime exclusions applied per root): Recording roots are
  `ui/src/domains/recording` 25,505/106 files,
  `ui/src/views/RecordModeView` 148/1,
  `playwright-driver/src/recording` 8,214/24,
  `api/handlers/record_mode.go` 1,069/1, and
  `api/services/live-capture` 2,052/4 = 36,988 lines. Export roots are
  `api/services/export` 5,615/21,
  `api/handlers/execution_export.go` 935/1,
  `ui/src/export` 1,909/10,
  `ui/src/domains/executions/export` 3,687/22, and
  `ui/src/domains/exports` 9,112/48 = 21,258 lines. The roots are file-disjoint
  but not dependency-disjoint: workflow folder export calls Markdown/report
  helpers in `api/services/export`, and additional shared APIs are outside these
  roots. These are gross upper bounds, not net savings or a deletion list.
- Follow-up workflow caller census (2026-10-03):
  `ui/src/domains/workflows/utils/normalizers.ts` is one implementation used by
  both the workflow store's API-response normalization and the editor's JSON
  import; those are two callers, not duplicate normalizer owners. The API's
  `NormalizeWorkflowDefinitionV2Bytes` is likewise one implementation used by
  capture, CLI registration and proto conversion; the separate
  `BuildFlowDefinitionV2ForWrite` is typed write ingress with node/edge
  referential validation. Sampled UI candidates (`WorkflowBuilder`,
  `NodePalette`, `ElementPickerModal`) have live view/field/recording callers.
  This trace found no safe deletion list for WF-1 and does not substantiate the
  −14,800 historical budget delta. The initial function/caller census is
  complete for these roots; the wider line-level owner map remains open.
- The same full-scenario caller search found two unused exported helpers in
  `ui/src/domains/workflows/utils/normalizers.ts`: `getNodeTypeFromAction`
  (lines 199–201) and `hasValidAction` (203–205), with no references outside
  their definitions. Together they are under ten runtime lines and are not a
  standalone epoch; include them only if a larger evidenced Workflow owner
  retirement is admitted.
- A UI production-source scan of all 40 Workflow `.ts`/`.tsx` modules and the
  636 UI production files found three more uncalled wrappers in
  `ui/src/domains/workflows/services/workflowApi.ts`:
  `executeAdhocWorkflowViaApi`, `validateResolvedWorkflowViaApi`, and
  `getWorkflowVersionViaApi`. The other sampled workflow exports have internal
  or external callers. The five uncalled helpers/wrappers remain a small
  candidate bundle (under roughly 35 runtime lines by source review), not an
  epoch; confirm package-level consumers and tests only if a larger Workflow
  retirement is found.
- Separate scenario-wide Go production-source reference scans of exported
  functions in `api/services/workflow` and the 70 production files under
  `api/automation` found no function with zero call-site references.
  The Workflow API/service audit therefore has no evidenced whole-file or
  exported-function deletion target so far; other `api/automation` and data
  paths still need role-by-role reconciliation before WF-1 can be estimated.
- Cross-owner helper scan found two `sanitizeFilename` implementations in
  `api/services/workflow/export_helpers.go` and
  `api/services/export/render/utils.go` (same trimming/separator/null-byte
  rules, distinct empty-name defaults), plus three `copyFile` implementations
  in workflow export, capture production and execution export. Their close-error
  handling differs and must be preserved if consolidated. Workflow's
  `ToInterfaceSlice` delegates to `api/automation/contracts` but adds a required
  `database.JSONMap` adapter, so it is not a duplicate. These small helper
  overlaps are candidates to bundle with a larger measured export ownership
  slice; they do not substantiate WF-1's 14,800-line target.
- Recording UI census after E21: a source-reference scan covered all 106
  production TypeScript modules in `ui/src/domains/recording` and 636 UI source
  files. All default-exported modules have cross-file references; four named
  runtime helpers have no production or test callers:
  `timelineItemsToWorkflowNodes`, `clampViewport`, `getPresetSettings`, and
  `extractDomainFromUrl`. They are small adapters/utilities totaling under
  roughly 50 lines, not a valid standalone REC-1 epoch. The scan did not find a
  further orphan file cluster after E21's nine-file removal.
- Workspace shell caller census (2026-10-03): `routes.tsx` selects one
  `RootLayout`; `renderApp.tsx` uses the 20-line `AppShell` for readiness and
  execution updates; `WorkflowEditorView` uses the shared `Sidebar` and 1,126-
  line `Header`; `RootLayout` and `DashboardViewWrapper` use onboarding's
  `GuidedTour`/`useGuidedTour`. No duplicate shell owner or removable large-file
  cluster was established. Five shell-scope named functions have no production
  references, but are small helpers/hooks and do not support the historical
  −7,600-line target; confirm their intended status only within a larger
  measured shell retirement.
- Export ownership census (2026-10-03): a source scan counted 5,615 lines in
  21 API export production files, 1,909 lines in the 10-file `ui/src/export`
  owner, 3,687 lines in 22 execution-export files, and 9,112 lines in 48
  exports-domain files. These are source totals, not removable-code estimates.
  `BuildReplaySpecFromExecution` delegates to the sole direct
  `BuildReplaySpecFromTimeline` generated-contract builder; execution-export
  handlers merge client input through `BuildReplaySpec`. UI timeline utilities
  are consumed by `ReplayExportPage` and its bridge. Workflow's
  `BuildExportPlan` produces a folder/Markdown report and calls shared export
  renderers; it does not repeat replay-spec or timeline-projection construction.
  Cross-owner `sanitizeFilename` helpers remain small, with different
  empty-name defaults. The sampled ownership/caller map establishes no duplicate
  large export owner or safe deletion list and does not substantiate the
  historical −7,000-line target. Return if a wider path-to-owner map identifies
  a concrete whole duplicate owner with measured savings, or the operator
  revises the export scope/budget boundary.
- AI ownership rebase (2026-10-03): E13 already removed the parallel Claude
  navigator, API-owned raw Ollama subprocess client, stale CLI engine selection,
  and UI command hook. Current wiring constructs one
  `PlaywrightVisionNavigator`; Connect serves navigation identity/state, the
  driver vision agent owns the browser loop, and both element-analysis and
  suggestion paths call the shared role-based `services/ai` client. The UI and
  CLI invoke the generated `VisionNavigationService` surface. Production search
  finds no reference to the E13-retired constructors/client/hook. A selected
  AI-source scan counted 12,775 physical lines across current vision, shared
  model, AI-handler, driver-AI and recording-navigation roots; these roots do
  not reconcile to the historic 17,900-line module budget and are not a deletion
  estimate. No second engine or duplicate model boundary is established by this
  rebase, so the historical −6,900 target remains unsupported. AI-1 is parked;
  return if a wider path-to-owner map identifies concrete duplicate ownership
  and savings, or the operator changes the scope/budget boundary.
- Diagnostics path census expansion (2026-10-03): name-based production-file
  matching found 5,016 physical lines in the currently selected observability,
  driver telemetry, session diagnostic logger, recording diagnostic routes,
  API telemetry adapters and health-monitor roots. It also found 2,576 lines in
  the separate UX-metrics API and UI subsystem (service/repository/analyzer,
  Connect handler, and execution panel). Callers show the observability Connect
  service fronts the driver collector; sidecar health monitors driver
  readiness; recording diagnostic routes inspect recording/external-URL state;
  the session logger attaches browser-context logs; API telemetry adapts
  execution events; UX metrics persist and present execution-friction analytics.
  These have distinct data, consumers and lifecycles. The old 7,000-line
  diagnostics budget does not specify whether UX metrics belongs to it, and
  these matched paths do not establish an interchangeable or retired owner.
  Keep DIAG-1 in census; reconcile the historical budget to explicit path
  ownership before admission. Do not count live UX metrics or health behavior as
  deletable diagnostics without a separately authorized scope change.
  Reconciliation against accepted history also shows E1 already removed 6,640
  runtime lines from dead diagnostics UI, retired operations/RPCs and driver
  self-test/route surfaces. The old 7,000→1,000 budget predates that accepted
  reduction and has no current path crosswalk; E1's remaining retained surfaces
  include live external-URL injection verification. DIAG-1 is parked after the
  expanded census: do not repeat E1's accepted deletions or infer a new target
  from the stale module budget. Reopen only if an owner/path reconciliation
  identifies additional retired code with callers safely removed, or the
  operator revises the diagnostics scope.
- REC-1A's 4,074-line candidate was an unreachable legacy UI surface, not a live
  feature with a verified user-visible loss. A gross Record mode + export
  source-root option is now listed above because those roots and active callers
  are measured. Its net removable lines remain unknown due to shared export
  consumers and unmeasured dependency closure; do not admit a retirement epoch
  until the operator chooses whether to fund that precise audit. No feature
  removal is authorized by this note.

## Next

### Orchestrator handoff — 2026-10-04T16:20Z

- Changed: no epoch or product source; reconciled the prior handoff, worker and
  owner signals after the timer wake.
- Verified: the parent run remains active and is the only running Agent
  Manager run; E22 worker `62b11517-9fd8-4f6a-941a-c7bd1b1a2cdb` is complete.
  The delivery inbox is empty. `browser-automation-studio status --json`
  reports the BAS API running and 0 workflows indexed; `vrooli scenario status
  browser-automation-studio --instance shadow --json` reports the isolated
  shadow healthy on build identity
  `sha256:e16bbda1f5ed9d79d12ba591912b24b08962bc4c4fca8512238948b7fa304cbd`.
- Remaining: no QA disposition is visible through the delivery inbox; the
  ad-hoc capture still has no indexed workflow for the intended review UI. E22
  R1 remains blocked on a reviewable playback path or QA repair; R2 remains
  blocked on the generated movie-spec contract owner. Record/Export audit,
  DET-SIGNIN-R1, DESK and parked ownership census return conditions are
  unchanged. No daily requalification was due again on this date.
- Exact next action: on wake, reconcile a QA/owner response or a newly available
  persisted playback path for the ad-hoc execution. If one appears, inspect the
  full current-build clip and known defect intervals in the intended player and
  update R1. Do not change the generated contract without its owner's decision.

### Orchestrator handoff — 2026-10-04T04:17Z

- Changed: refreshed only `browser-automation-studio@shadow` through the
  supported lifecycle after the Vite alias repair was recorded as build-verified
  and a quiet window was confirmed. The start operation succeeded healthy with
  build identity `sha256:e16bbda1f5ed9d79d12ba591912b24b08962bc4c4fca8512238948b7fa304cbd`;
  the live BAS was not restarted. No product source changed.
- Verified: current-build Linux journeys pass 12/12; receipt
  `~/.vrooli/evidence/bas-goal/journeys-2026-10-04.json`, SHA-256
  `9b998f6f9903dab0ebfbca5198a8a1de9c545b80687d899fc6d23eeff7eeec54`. A fresh
  neutral current-build capture `8ed37203-2ff7-49d7-9e1f-f953e1f0edc2` preserved
  a 14.16s, 1440×900, 25fps VP8 WebM (499,488 bytes; SHA-256
  `40ec6c8a2a92779f29dcced9bea148c0ac5b78b59ec180e71e1dfde8b09709c2`) and
  correctly failed step 12 with `requested_extent=full_page`,
  `actual_extent=not_captured` during native video. The in-app Executions page
  lists this ad-hoc run as `Unknown Workflow`; selecting it returns
  `workflow project_id missing` and leaves the page open, so the intended-player
  full-clip/known-interval comparison is still unverified. Filed one QA report:
  `knw-1791087389154714359` (`bug-inbox/code-defect/ad-hoc-executions-cannot-open-in-review-ui`).
  Parent run identity was recovered from `agent-manager run list` after
  `run identity --json` reported no identity token in this resumed shell.
- Remaining: E22 R1 has its current-build 12-journey result; intended-player
  visual review still awaits a reviewable execution path for this ad-hoc capture
  or the QA repair. E22 R2 still awaits the generated movie-spec contract
  owner's decision. Record/Export audit, DET-SIGNIN-R1, DESK and parked ownership
  censuses remain unchanged.
- Exact next action: on wake, reconcile the QA/owner response or a persisted
  playback route for the ad-hoc execution. Once available, inspect the full
  current-build clip and the known defect intervals in the intended player and
  record the R1 result. Do not widen the capture scope or change the generated
  contract without its owner decision.

### Orchestrator handoff — 2026-10-03T16:05Z

- Changed: no epoch or product source; reconciled the goal home and current
  operator inputs after the timer wake.
- Verified: orchestrator identity remains
  `70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb`; the wake reports no active or ended
  children; the team inbox is empty; FEEDBACK has no directive newer than
  BAS-FB-062. E22 remains recorded as accepted with R1/R2 remainders.
- Remaining: the Record mode + Export candidate awaits operator direction to
  scope its precise retirement audit. E22 R1 awaits the logged UI build failure
  resolution and a quiet shadow window; R2 awaits the generated-contract owner
  decision. DET-SIGNIN-R1 still awaits dedicated authorized test sites/accounts;
  DESK awaits Linux health and a native machine. No parked return condition has
  changed.
- Exact next action: on the next wake, reconcile new operator, owner or
  outside-repair evidence. If the operator directs the Record mode + Export
  audit, map shared dependencies and measure net savings before admitting an
  epoch. Otherwise keep the destination gap visible and continue only when a
  parked slice's return condition changes.

### Orchestrator handoff — 2026-10-03T04:00Z

- Changed: this queue only; added the gross Record mode + Export retirement
  option, expanded Export/AI/Diagnostics ownership evidence, and refreshed the
  inventory timestamp. No product source changed.
- Verified: the no-Git inventory is 254,178 runtime lines, gap 49,178, digest
  `f24b43325db4e13b2d9f9981f42bfefc119087ef6e91df83e4a51a68371b6479`,
  unchanged from the E22 accepted snapshot. The 58,246-line candidate is gross
  source scope only; net deletion and shared-consumer closure are unverified.
- Remaining: E22 R1 waits on the UI build repair and quiet shadow window; E22 R2
  waits on the generated-contract owner decision; DET-SIGNIN-R1 waits on
  operator-provided sites/accounts; DESK waits on a native machine. All current
  deletion-first module slices are parked without a measured safe deletion list.
- Exact next action: on wake, reconcile new operator/owner or outside-repair
  evidence. If the operator wants the Record mode + Export option scoped, map
  its shared dependencies and compute net savings before admitting any epoch;
  otherwise keep the destination gap visible and continue only when a parked
  slice's return condition changes.

- E1–E22 have accepted epochs. E3 completed the first S2 tranche and E7 completed
  the BAS-FB-057 remainder. E6 completed S3 under BAS-FB-058:
  all V1 workflows found by scan were migrated, and the six GCT HTTP cases were
  preserved as non-workflows. E2 and E4 record Gremlins as unverified under
  BAS-FB-056.
0. **Order (BAS-FB-062).** The 205k gap no longer holds measured slices. DIAG-REDACT,
   REC-1A and REC-FIX are accepted; WF-1 is parked after its initial census.
   Continue the remaining ownership censuses in order and write only
   evidence-backed retirement candidates for the operator.
0b. **REC-FIX remainder (E22 accepted with limits).** R1: after the logged UI
   build issue is resolved and a quiet isolated shadow window is available,
   rerun the 12 Linux journeys and inspect the full short clip plus known bad
   intervals in the intended review player against the changed build. R2: obtain
   the generated movie-spec contract owner's decision and use the governed
   generation route if timed cursor samples/edit-map support is approved. Until
   then, the execution review uses retained timed samples and export receipts
   state edit-map unavailability. Do not retry shadow lifecycle or edit generated
   code/dependencies before those return conditions hold.
1. **DET-SIGNIN-R1 (parked).** Qualify a few real sign-ins with persistent
   profiles. Return condition: the operator supplies dedicated, authorized
   test-only target sites and accounts. Until then, keep sign-ins unverified;
   never use personal accounts or credentials. E16's independent local
   detectability and interactive-profile work is accepted below.
2. **WF-1 — Workflow ownership (parked after initial census).** No large
   deletion-first scope is evidenced by the completed function/caller scans
   above: the only uncalled UI functions/wrappers total under about 35 runtime
   lines, and no unused API exported functions were found. Return when a wider
   path-to-owner map identifies a whole duplicate owner with concrete deletions
   and a measured estimate, or the operator revises the scope/budget boundary.
3. **REC-1 — Remaining Recording UI ownership (parked after caller census).**
   The historic budget gap is about −7,793 after REC-1A's estimated −4,074.
   Return condition: find a larger concrete duplicate owner with callers mapped
   and measured savings, or obtain operator direction for live-feature removal.
4. **WSUI-1 — Workspace shell ownership (parked after caller census).** No
   duplicate shell owner or large orphan cluster was found. Return condition: a
   wider map identifies a concrete whole duplicate owner with measured savings,
   or the operator changes the scope/budget boundary.
5. **EXP-1 — Export pipeline ownership (parked after caller census).** No
   duplicate large owner or safe deletion list was established; the historic
   −7,000 is unsupported by the measured caller map. Return if a wider
   path-to-owner map finds concrete whole-owner savings, or the operator changes
   the scope/budget boundary.
6. **AI-1 — AI navigation ownership (parked after E13 owner rebase).** No second
   engine or duplicate model boundary was established; the historical −6,900 is
   unsupported by this selected source map. Return if a wider path-to-owner map
   shows concrete duplicate ownership and measured savings, or the operator
   changes the scope/budget boundary.
7. **DIAG-1 — Diagnostics ownership (parked after expanded census).** E1's
   accepted −6,640-line retirement predates the stale 7,000→1,000 target;
   current path evidence separates remaining live readiness, telemetry,
   observability, recording diagnostics and UX metrics owners. Return only if a
   concrete additional retired owner is measured or the operator revises scope.
8. **DESK — Desktop portability epoch** (D22). When Linux is healthy and a native
   machine is available; macOS through minimouse occasionally.




Every refactor epoch also matures the tests of the modules it touches, under the
same rules (worker card), so test lines fall with runtime lines.

Module budgets (runtime lines, current → target): Session 10.7k→6k, Runtime
adapter 18.4k→14k, Transport 10.9k→7k, Recording 36.2k→24k, Profiles 6.7k→5k,
Workflow 40.8k→26k, Evidence 22.3k→17k, Workspace UI 29.6k→22k, AI 17.9k→11k,
Export 29k→22k, Diagnostics 7k→1k, Config 3.1k→1.8k, other 53k→48k.

## Parked

- **DET-SIGNIN-R1** — see Next item 1.
- **WF-1 — Workflow ownership.** Initial function/caller census found only the
  small helpers and helper overlaps recorded under `## Retirement candidates`;
  no valid epoch reaches the historical −14,800 target. Return condition: a
  wider path-to-owner map supports a concrete whole duplicate-owner retirement
  with measured savings, or the operator changes the scope/budget boundary.
- **REC-1 — Remaining Recording UI ownership.** The post-E21 scan found four
  small uncalled helpers but no additional orphan files; their under-50-line
  bundle is below epoch scale. Return condition: a larger mapped owner boundary
  supports a valid replace-and-delete epoch, or the operator directs a live
  feature retirement.
- **WSUI-1 — Workspace shell ownership.** The caller census found the named
  shell and onboarding roots live, with no large duplicate/orphan cluster.
  Return condition: a wider path-to-owner map supports a whole shell-owner
  retirement with measured savings, or the operator revises the boundary.

## Qualification findings

(Daily journey-run failures land here as queue items.)

- **J11 journey coverage pending.** J11 (AI navigation, stubbed model) moved to a
  driver integration test because the shadow driver reads the model gateway only
  from `AI_GATEWAY_URL` at startup. Restore a shadow journey once the stub gateway
  can be injected without restarting the shadow.

## Done

- **E22 — REC-FIX: faithful recording and useful review playback.** ACCEPTED
  2026-10-03T03:03:00Z with the two named remainders above; current-source
  screenshot extent/native-video guards, geometry/provenance-aware timed review
  playback and source/final-spec receipts are in place. Parent reran UI full
  suite/type-check, driver type-check and focused tests, and affected API Go
  packages; all passed. Runtime +379 vs +500 estimate (within +750 cap); support
  +489 vs inferred admission baseline. The 12 post-change Linux journeys and
  intended-player visual comparison remain unverified after the authorized
  shadow refresh failed at Vite ENOTDIR; generated movie-spec timed-trail and
  edit-map representation awaits its owner. Suggested commit:
  `fix(bas): preserve truthful recording review evidence`.
- **E21 — REC-1A: retire unreachable legacy recording UI surfaces.** ACCEPTED
  2026-10-03T01:15:05Z (epochs/E21.md); all nine measured candidate components,
  their stale recording barrels and the orphan ActionTimeline-only test are
  removed; the retained unified timeline behavior and `[REQ]` assertions pass.
  The two active ActionTimeline descriptions now name the retained TimelineTab
  and TimelineEventCard owners. Orchestrator reran UI type-check, the full UI
  suite (record-mode 542/542), 12/12 Linux journeys, deletion/caller searches,
  and no-Git inventory. Runtime is 253,799 (−4,081 vs E20; estimate −4,074);
  the difference includes 8 barrel lines removed and one Go comment line added.
  Worker-reported test/support delta is −43. macOS arm64 was not exercised;
  macOS x64 and Windows remain unavailable. Suggested commit:
  `refactor(bas): retire unreachable recording UI surfaces`.

- **E20 — DIAG-REDACT: redact recovery credential from observability output.**
  ACCEPTED 2026-10-03T00:06:33Z (epochs/E20.md); synthetic set/unset/modified
  cases cover full standard/deep driver responses and caches plus API
  proxy/Connect output, with metadata and fixture authentication preserved.
  Orchestrator reran 3 driver Jest suites (68 tests), driver typecheck, and
  uncached API handler/observability tests; all pass. Runtime 257,880 (+5 from
  E19, within the +50 ceiling); reported test growth +70, with earlier baseline
  unavailable. The worker hit the overrun step-back at 9 units/4 estimated,
  chose park for review, then was stopped by the parent at the reserved window;
  orchestrator review resolved the return condition. External document,
  deployed version skew, outer access and universal security remain
  unverified. Suggested commit: `fix(bas): redact recovery credential from
  observability output`.

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
