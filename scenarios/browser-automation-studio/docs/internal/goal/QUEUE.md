# BAS queue

Full history through 2026-10-06 (handoffs, censuses, candidate evidence, the
long Forecast) is in `archive/QUEUE-through-2026-10-06.md`.

## Needs operator

(none; the operator answered BAS-FB-063 on 2026-10-06)

## Handoff

- BAS-FB-063 remains the accepted destination: Record and Export stay; E22 R1/R2,
  DET-SIGNIN-R1 and J11 shadow remain under Waiting on others.
- E23 accepted 2026-10-07T00:14:20Z as RD-WF first tranche. One generated-V2-backed
  UI codec now owns the listed import/load/save/fingerprint/conflict/restore
  transformations. Parent reran focused UI checks (134/134), the 12 applicable
  Linux shadow journeys (12/12), and the no-Git inventory (253,850 runtime,
  −328 from admission). Three named workflow cases remain unverified after one
  Test Genie provider-queue timeout; see E23's gate amendment and WORKAROUNDS.md.
  The parser sums worker slice deltas to −656 runtime/−672 tests because D1's
  handoff-only line repeats the original total; direct inventory establishes the
  actual runtime delta as −328.
- E24 accepted 2026-10-07T00:42:30Z as RD-REC first tranche. Recording edits,
  selection, insertion, replay and workflow-generation inputs now derive from
  canonical generated `TimelineEntry`; the duplicate recording `RecordedAction[]`
  and local `TimelineItem[]` state owners are removed. Parent reran type-check,
  focused recording timeline/sidebar tests (59/59), all 12 applicable Linux
  shadow journeys (12/12), and inventory (253,784 runtime, −66). The spend
  trigger fired and D1 was acknowledged; immutable Agent Manager pre-write reads
  and file-change records resolved attribution for the two call-site edits left
  outside the worker's four-file archive. No direct workspace integration test
  is claimed; see the coordinator review note in E24.
- E25 accepted 2026-10-07T01:28:54Z as RD-REC timeline-controller
  consolidation. One 338-line `useWorkspaceTimeline` replaces the two admitted
  hooks. Parent reran type-check, focused suites (61/61), all 12 required Linux
  journeys and no-Git inventory (253,452 runtime, −332; selected 411,216;
  support 157,764; digest `3353414b35a161c831745163bf92b1aa0d36a13a3c3a19fc89edcdac7f89b851`). The parent journey receipt is
  `70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb/codex/tmp/bas-goal-e25-parent-review/journeys-d1.json`
  (SHA-256 `c64a59b32221d75c7f8c619a1207930fba35d7105673274ee012525399c7d90d`).
  D1 corrected idle-versus-active state and added public coverage. No dedicated
  `useRecordingModeState` integration test is claimed; see E25.
- E26 admitted RD-EXP first tranche. The 1,136-line Go HTML bundle contains a
  526-line inline renderer and roughly 360 lines of presentation projection;
  the existing 573-line ReplaySpec-driven `ReplayExportPage` is the proposed
  sole renderer. Task `9f2c8e4b-185d-427c-a46b-1efbc6cc5728` and Luna-medium
  child `e8c322ca-72e7-434c-9d37-1d2cf853961f` are admitted under parent run
  `70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb`.
  E26's implementation and offline archive playback are verified; focused
  API/UI checks and all 12 required Linux journeys pass, with runtime at 252,651
  (−801 against the admission ceiling). It is **not accepted**: the pre-write
  artifact omitted generated `ui/dist`, so its original material identity cannot
  be compared. The spend trigger fired at 548,305 and again at 673,060 weighted
  non-cache tokens against 500,000. The worker reconstructed and compared the
  changed UI source, but its original generated bundle remains unrecoverable.
  The child ended after a parser-valid step-back row acknowledged D1/D2;
  `epoch-check` reports no open directives. E26 is parked pending evidence-owner
  review.
- WSUI-1 route-shell census (completed this pass): the owner map was widened across
  workspace routes and navigation state to find concrete same-behavior owners and
  measured savings before admitting another epoch. The initial check of `Header.tsx` and
  `ProjectDetailHeader.tsx` found distinct editor and project actions; both reuse
  the shared `Breadcrumbs` component. `RootLayout` supplies global providers and
  route context, while `Sidebar` is called only by the workflow editor. Continue
  from the archived caller census across the remaining route shells. The
  dashboard, all-executions, all-workflows, and settings routes repeat a sticky
  header wrapper, but dashboard identity/live badges, list-page back/search/filter
  controls, and settings actions differ. Replacing the repeated wrapper would
  remove only a few layout lines per route and add a shared component/props
  boundary; it is not an epoch-scale deletion. No whole-owner deletion list or
  positive net estimate is established. Route ownership has a declarative
  table in `ui/src/routes.tsx` and a separate `getViewFromPath` context mapper in
  `RootLayout`; the mapper and `AppView` union are small, with no epoch-scale
  deletion. AI-1's accepted E13 already removed the duplicate navigator and
  model-client owners; the retained driver, API, and UI roles are distinct.
- Execution-owner check (completed): mapped the largest remaining execution owner,
  `api/automation/executor/simple_executor.go`, against the workflow compiler and
  Playwright driver's typed-action executor. Respect the Domain map's separation
  between compilation and browser execution; identify any same-behavior owner
  only if call paths support it, and measure replacement lines before admitting
  an epoch. Do not revive unsupported WSUI/AI estimates. Do not resume E26 until
  the generated-material return condition below is met or an authorized gate
  amendment is recorded.
- Execution-owner check: `SimpleExecutor` coordinates session lifecycle,
  capabilities, retries, telemetry and persistence; the workflow compiler emits
  `ExecutionPlan`/`CompiledInstruction`; the Playwright driver executes each
  typed instruction. The recording browser script captures DOM events, while
  `action-executor.ts` replays `TimelineEntry` through the shared handler adapter.
  These are distinct stages with one owner each; no whole-owner deletion or
  measured redesign estimate emerged. No new epoch is admitted from this map.
- Candidate, not admitted: global `AllExecutionsView` has its own
  `loadGlobalExecutions` normalization and list state, while scoped
  `ExecutionHistory` mirrors the canonical execution store into local state.
  Both call the execution-list API and render status/time/workflow summaries,
  but the global page also joins project names and the scoped history supports
  selection into its timeline viewer. The common store is 1,124 lines;
  `GlobalExecutionsView` is 373, its controller is 69, and `ExecutionHistory` is
  236. A shared list/projection owner may remove duplication, but the distinct
  scope behavior and exact removable lines are not yet mapped.
- Execution-summary comparison: `dashboardStore.fetchRecentExecutions` and
  `fetchRunningExecutions` issue the same unscoped 50-item list request and
  independently normalize status/workflow/project metadata; one splits all
  results into recent and active arrays, the other refreshes only active rows.
  `loadGlobalExecutions` separately loads up to 200 rows and performs a similar
  status/date/name/project projection, with its own polling, search, status
  filter, counts, and inline row presentation. Scoped `ExecutionHistory` uses
  `ExecutionStore.loadExecutions(workflowId, projectId)`, mirrors store state
  locally, and supports scoped status counts, selection, progress, and the
  timeline viewer. `ExecutionsTab` consumes dashboard projections and adds
  stop/rerun, filters, and its inline viewer; `RecentExecutionsWidget` renders
  only the dashboard's recent/active projection. These call paths share some
  normalization and list transport but not query scope or view lifecycle.
  A shared summary loader could replace roughly 35–55 lines of repeated
  projection/request glue, but would add a shared model/adapter and retain the
  distinct view state and interactions; no positive epoch-scale net deletion
  is evidenced. Do not admit an execution-list epoch from this candidate.
- RD-WF remeasurement after E23: the Go `BuildFlowDefinitionV2ForWrite` path
  marshals strict newly-authored input and validates node/edge references;
  converter/protostore `marshalToFlowDefinition` instead applies short-form V2
  normalization for imported/raw workflow JSON before proto decoding. The
  standalone byte normalizer also serves capture and proto conversion callers.
  Their shared proto decode step is small; input contracts, normalization, and
  validation differ, and no epoch-scale same-behavior owner replacement or
  positive net reduction is supported by this map. The compiler remains a typed
  execution projection and is outside that ingress overlap. Do not admit a
  follow-on RD-WF epoch from this census.
- RD-REC follow-on measurement scope: start from E25's accepted
  `useWorkspaceTimeline` replacement and map its current callers plus any
  still-live duplicated timeline controller before estimating an ordered
  follow-on. Admit only with a concrete deletion list and positive measured net
  reduction.
- RD-REC follow-on measurement: E25's `useWorkspaceTimeline` is the sole live
  workspace controller caller; the two retired hook names have no production
  references. `useSessionStore` still declares old `timelineEntries`, loading,
  count and pagination state, with setters/actions and exported selectors but no
  non-test consumers. This is a plausible stale mirror from the pre-hook path,
  but the removable surface is only a few dozen lines in the 509-line store;
  `timeline-unified.ts` conversion helpers remain called by the workspace and
  are not duplicate controller owners. No epoch-scale same-behavior replacement
  or positive net estimate is evidenced. Do not admit an RD-REC epoch from this
  finding; return only if a wider caller map finds a substantial owner overlap.
- No in-boundary epoch candidate remains from the current WSUI-1, RD-WF,
  execution-summary, or RD-REC maps. E26 awaits the provenance return condition;
  E22, DET-SIGNIN-R1, J11 shadow, and DESK retain their existing external or
  native-target waits. Next action: park for 12 hours and resume on child/event
  wake or timer to recheck those return conditions and select any newly admitted
  bounded work. Do not resume E26 without its generated-material return
  condition or an authorized gate amendment.

## Forecast

- Destination: runtime ≤205,000. Current 253,452 (2026-10-07), gap 48,452.
- Last five: E21 −4,081/≤0; E22 +379/≤750 (feature); E23 −328/≤0;
  E24 −66/≤0; E25 −332/≤0 (brief's conservative estimate 0). All met their
  respective growth budgets.
- Deletion-first censuses (WF-1, REC-1, WSUI-1, EXP-1, AI-1, DIAG-1) found no
  large dead or duplicate code, so the rest of the gap needs redesign slices
  (BAS-FB-063). Current root sizes: Workflow about 37k, Recording about 39k,
  Export about 11k, AI about 10k. Re-estimate after each redesign acceptance.
  E24's 1,554-line census reduced runtime by 66 lines. E25 removed 676 lines of
  prior controller ownership and introduced a 338-line replacement, reducing
  runtime by 332. Three accepted epochs in a row were below half their work-unit
  estimates: E23 7/16, E24 5/20, E25 3/16. Re-estimate subsequent work from
  measured replacement sizes. The current export census found the Go standalone
  HTML renderer (`api/services/export/html_bundle.go`, 1,136 lines; inline
  renderer template 526 lines) and the existing ReplaySpec-driven UI replay page
  (`ui/src/export/ReplayExportPage.tsx`, 573 lines). The queue's remaining
  redesigns do not yet have net estimates, so their sum cannot be shown to close
  the 48,452-line gap. E26 must map the exact same-behavior surface and estimate
  its replacement before implementation.

## Next

Each item is a redesign slice (`large-effort-orchestration` §2.1). The brief
names the one simpler owner, the existing owners it replaces (that is the
deletion list) and the journeys that prove behavior. Measure before admission,
and plan the next one when fewer than two are queued.

1. **RD-WF — one workflow model and pipeline (E23 accepted).** E23 consolidates
   the UI load/import/canvas/save boundary around generated `WorkflowDefinitionV2`;
   Go ingress and the automation compiler remain canonical-model owners. Remeasure
   those owners after E23 and admit ordered follow-on tranches only when an
   epoch-scale replacement and exact deletion list are evidenced.
2. **RD-REC — recording UI on one owner.** E24 consolidated journal ownership
   and E25 consolidated the mode-aware timeline controller. The follow-on map
   found only a few dozen unused `sessionStore` timeline-state lines, below
   epoch scale; return if a wider caller map finds a substantial owner overlap.
3. **RD-EXP — export on the replay spec.** E26 moves standalone HTML playback
   to the existing ReplaySpec-driven UI renderer while preserving offline
   export and stored assets. Retire the measured Go template/projection only
   after its replacement and gates pass; brief in `epochs/E26.md`.
4. **RD-WSUI and RD-AI** — plan after RD-WF from the same kind of owner map.
5. **DESK — desktop portability** (D22). Run it when a native machine is
   available.

Every refactor epoch also matures the tests of the modules it touches (worker
card), so test lines fall along with runtime lines.

## Waiting on others

These never hold acceptance or other slices.

- **E22-R1** — intended-player review of capture
  `8ed37203-2ff7-49d7-9e1f-f953e1f0edc2`. Returns when QA bug
  `knw-1791087389154714359` (ad-hoc executions cannot open in review UI) is fixed
  or the capture gets a persisted workflow/project association.
- **E22-R2** — timed cursor samples and edit map in the movie spec. Returns on
  the generated movie-spec contract owner's decision.
- **DET-SIGNIN-R1** — real sign-ins with persistent profiles. Returns when the
  operator supplies dedicated, authorized test-only sites and accounts.
- **J11 shadow journey** — returns when the stub AI gateway can be injected
  without restarting the shadow.
- **E26 generated-bundle provenance** — returns when the evidence owner supplies
  the missing pre-write `ui/dist` identity or an authorized route that can
  verify the original material. The current build is only post-write evidence;
  a rebuild from reconstructed source does not prove the original bytes.

## Qualification findings

(Daily journey-run failures land here as queue items.)

## Done

- E25 — RD-REC: one recording/execution workspace timeline controller. (`epochs/E25.md`)
- E24 — RD-REC first tranche: canonical `TimelineEntry` owns UI recording state. (`epochs/E24.md`)
- E23 — RD-WF first tranche: one generated-V2-backed UI workflow codec. (`epochs/E23.md`)
- E22 — REC-FIX: faithful recording and useful review playback. (`epochs/E22.md`)
- E21 — REC-1A: retire unreachable legacy recording UI surfaces. (`epochs/E21.md`)
- E20 — DIAG-REDACT: redact recovery credential from observability output. (`epochs/E20.md`)
- E19 — S13: one replay spec. (`epochs/E19.md`)
- E18 — S12: split the recording workspace controller. (`epochs/E18.md`)
- E17 — S6: one timeline shape. (`epochs/E17.md`)
- E16 — DET local detectability and interactive profile behavior. (`epochs/E16.md`)
- E15 — S10: one config owner. (`epochs/E15.md`)
- E12 — S7: one frame transport. (`epochs/E12.md`)
- E13 — S8: one AI engine. (`epochs/E13.md`)
- E14 — S9: one retention planner. (`epochs/E14.md`)
- E11 — S5: one recording journal. (`epochs/E11.md`)
- E10 — S1: one session broker. (`epochs/E10.md`)
- E9 — JX: journey expansion before session/recording rewrites. (`epochs/E9.md`)
- E8 — S4: schema-driven node editor. (`epochs/E8.md`)
- E6 — S3: remove the V1 workflow compatibility path. (`epochs/E6.md`)
- E7 — S2 remainder: remove handwritten action vocabulary owners. (`epochs/E7.md`)
- E5 — Driver, UI and CLI test consolidation. (`epochs/E5.md`)
- E4 — Further Go test maturation. (`epochs/E4.md`)
- E3 — Action vocabulary generated from proto (first tranche only). (`epochs/E3.md`)
- E2 — Go test consolidation. (`epochs/E2.md`)
- E1 — Dead code and diagnostics ownership. (`epochs/E1.md`)
- E0 — Journey suite and local fixture site. (`epochs/E0.md`)
- E01 — cleanup selectors, scheduler cancellation, AI navigation hook split (pre-epoch work; see the archived queue)
