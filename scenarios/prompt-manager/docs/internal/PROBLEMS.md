# Known Issues & Technical Debt

Agent-maintained document tracking issues, debt, and cleanup history.

## Last Updated
2026-09-08

## 2026-09-04 — Workflow runner loses leading navigation context

**Status:** External runner defect filed as `knw-1788511204267015162`.

The focused Test Genie workflow run `20260904-083355-98671493` rejected all
nine world workflows after their valid leading `ACTION_TYPE_NAVIGATE` node;
the following observer step reported that no navigation context was available.
The world BAS contracts validate and the same pages pass the dedicated browser
smoke harness, but Test Genie cannot execute those contracts until the shared
Workflow Health runner preserves the navigate result. Prompt Manager keeps the
valid navigate-first workflows unchanged because the fault is outside this
scenario's boundary.

## 2026-08-19 Architecture Audit Residuals

The proto re-platform plan owns API/CLI layout, transport, bindings, measures,
and contract documentation. The following audit findings are intentionally not
fixed by that plan and therefore have explicit owners.

| Finding | Evidence / impact | Owner | Exit condition |
|---|---|---|---|
| UI architecture audit debt | The 2026-08-19 server-owned architecture audit (`20260819-032525-cd984dc1`) reports broad UI structure/documentation findings outside the transport plan's boundary. | Prompt Manager UI maintainers; follow-up UI architecture plan | A focused UI audit has a stored baseline and clears required structure/documentation findings without folding UI redesign into the API migration. |
| Test coverage gaps in tags, agents, store, testing, search, and metrics | Package-level gaps listed below predate the migration and are not equivalent to transport parity. | Prompt Manager test-substrate plan | Each named package has behavior-focused unit coverage and the scenario test receipt records the new pass set. |
| Graph recent-activity signal remains neutral | `RecentActivityScoreFromTimestamp` exists but graph nodes do not carry the required timestamp. | Graph domain owner | Node contracts carry authoritative update time and graph scoring tests prove non-neutral recent activity. |
| Optional Qdrant degradation lacks a dedicated resilience SLO | Text fallback exists, but this plan does not define a resource-outage performance/recovery SLO. | AI Search domain owner | A resource-degraded test and measured recovery/latency target are documented and enforced. |

Runtime/manifest divergence, empty layout scaffolding, the orphan Graph RPC,
REST retirement, and missing stateful-domain measures were closed by phases
11–16 of the re-platform plan. The 2026-08-19 live evidence is 120 Connect
commands, 10 intentionally local commands, 75 typed omissions, 18 architecture
exceptions, nine probed measures, six remaining non-domain REST registrations,
and 117/117 callable Prompt Manager bindings in Program Runtime's doctor.

---

## Code Quality Debt

### API

| Area | Issue | Severity | Recommended Fix |
|------|-------|----------|-----------------|
| Error handling | Inconsistent error response format across domains | Medium | Standardize with shared error types |
| Validation | Input validation scattered in handlers | Low | Extract to middleware or shared validators |

### CLI

| Area | Issue | Severity | Recommended Fix |
|------|-------|----------|-----------------|
| Output formatting | Duplication across domain commands | Low | Consolidate into `internal/output` |

### UI

_No significant debt identified._

---

## Test Gaps

### API Coverage

| Package | Coverage | Gaps |
|---------|----------|------|
| skills/ | Good | handlers_test.go, query_test.go exist |
| heartbeat/ | Good | scheduler, executor, handlers, prompt builder, team execution queue, member context all covered |
| tags/ | None | Needs handler tests |
| agents/ | None | Needs handler tests |
| teams/ | Partial | Heartbeat cleanup + handler coverage exists; expand org chart + messaging tests |
| store/ | None | Needs relation and index tests |
| testing/ | None | Needs handler tests with mock Ollama |
| search/ | None | Needs handler tests |
| metrics/ | None | Needs repository tests |

### CLI Coverage

| Package | Coverage | Gaps |
|---------|----------|------|
| `cli/teams` | Targeted | Team policy flag resolution and preset transitions covered; broader command integration tests still needed |

### Recommended Test Priority

1. `api/skills/handlers_test.go` - Extend existing tests
2. `api/tags/handlers_test.go` - CRUD tests with mock repository
3. `api/search/handlers_test.go` - Search logic tests
4. CLI integration tests with mock API for end-to-end command output

---

## E2E Issues

| Area | Issue | Impact | Recommendation |
|------|-------|--------|----------------|
| UI smoke coverage | Smoke tests now cover load, scene switching, new-skill editor open, skill save/discard, and member creation | Low | Add BAS coverage for search filtering and one full team policy edit flow |
| Requirements linkage | BAS workflows not linked to requirements JSON | Low | Add automation validation entries once requirements are formalized |

### Missing data-testid attributes in production bundle (Fixed)
- **Execution ID:** ebf858ab-d3d0-4b9a-b2f6-6e4a11c11a87
- **Output path:** `/tmp/bas/prompt-manager/world-ui-loads`
- **Screenshot:** `/tmp/bas/prompt-manager/world-ui-loads/screenshots/step-02-wait-world-canvas.png`
- **Root cause:** prompt-manager UI was serving a stale production bundle built before data-testid attributes were added, so BAS selectors could not resolve.
- **Fix:** rebuilt the UI bundle (`pnpm run build`) and restarted the scenario to serve the updated `ui/dist`.
- **Status:** Fixed (validated by successful BAS runs: `7d145946-73b2-4562-9378-b6363a6dd499`, `d9baa7b8-a171-4cd8-a2ff-ff794b57ecfb`, `1cfc4bc0-869a-4d18-ba43-2afadb6c450e`)

---

## Stability Issues

_No open crash issues identified. Team editor org chart now guards against self-connections and missing manager labels; relationship panels handle empty states gracefully._

---

## Root Cause Analyses (Resolved)

### Heartbeat trigger crashes when config is missing

**Hypotheses**
1. Trigger handler skips config checks, leading to nil dereference inside executor.
2. Executor assumes heartbeat config exists because scheduling only happens for enabled configs.

**Test/Verification**
- Manual trigger without `heartbeat.json` returns `404` and does not panic.
- Unit test: `api/heartbeat/executor_test.go#TestExecutorExecuteFailsWhenConfigMissing`.

**Root Cause**
- `Executor.Execute` and `TriggerManual` assumed `GetHeartbeatConfig` never returns `nil`, causing unsafe dereferences.

**Fix**
- Guard against nil configs and return a not-found error before updating state.

**Prevention**
- Added explicit nil checks and regression tests.

---

### Scheduler ignores per-member profileKey

**Hypotheses**
1. Scheduler uses a hard-coded default profile key for all runs.
2. Profile key is only honored in manual trigger path.

**Test/Verification**
- Scheduler uses `profileKey` from config in `api/heartbeat/scheduler_test.go#TestSchedulerUsesConfigProfileKey`.

**Root Cause**
- Scheduler did not consult per-member heartbeat config when executing scheduled runs.

**Fix**
- Introduced a config-store seam in the scheduler and resolved profile key at execution time.

**Prevention**
- Added scheduler unit tests covering default and custom profile keys.

---

### Removing a team member leaves scheduled heartbeats and member data

**Hypotheses**
1. Member removal only deletes the relation file and does not clean related member files.
2. Scheduler has no lifecycle hook to unschedule per-member entries when membership changes.

**Test/Verification**
- Member removal now unschedules and deletes the member directory in `api/teams/handlers_cleanup_test.go#TestRemoveMemberCleansDataAndUnschedules`.

**Root Cause**
- Cleanup responsibilities were split across domains with no explicit boundary for member teardown.

**Fix**
- Added `FileTeamStore.DeleteMemberData` and a handler-level cleanup step to unschedule + remove member data.

**Prevention**
- Centralized cleanup logic and added regression test.

---

### Deleting a team leaves scheduled heartbeats active

**Hypotheses**
1. Team delete path does not inform the heartbeat scheduler.
2. Scheduler retains cron entries independently of file store deletion.

**Test/Verification**
- Team deletion now unschedules all member heartbeats in `api/teams/handlers_cleanup_test.go#TestDeleteTeamUnschedulesHeartbeats`.

**Root Cause**
- Team deletion flow skipped scheduler teardown and relied solely on file deletion.

**Fix**
- Unschedule all team heartbeats before deleting team files.

**Prevention**
- Added explicit scheduler cleanup and a regression test.

---

### Incomplete scenario registration in CLIDetector (Fixed)

**Root Cause**
- `NewCLIDetector([]string{"prompt-manager"})` only registered `vrooli` and `prompt-manager` as known CLIs. Scenario CLIs like `visited-tracker`, `app-monitor`, etc. were classified as `CodeExternalTool` instead of `CodeScenarioCLI`.

**Fix**
- Added `discoverScenarioNames()` in `main.go` that reads all scenario directory names at startup and passes them to `NewCLIDetector`. All 86+ scenario CLIs are now correctly classified.

**Prevention**
- Scenario names are discovered dynamically from the filesystem — no hardcoding required when scenarios are added or removed.

---

### Multi-line backtick commands missed by CLIDetector (Fixed)

**Root Cause**
- `Detect()` processed backtick patterns per-line, so multi-line backtick spans (with `\` continuation) were never matched.

**Fix**
- Backtick matching now runs on full content (after stripping code fences). Code fences are replaced with equivalent newlines to preserve line numbering.

**Prevention**
- Tests: `TestCLIDetector_MultiLineBacktick`, `TestCLIDetector_CodeFenceStripped`, `TestCLIDetector_CodeFencePreservesLineNumbers`.

---

## UX Issues

| Area | Issue | Impact | Recommendation |
|------|-------|--------|----------------|
| CLI | No completion support | Low | Add shell completion scripts |
| CLI | Long content truncated in list views | Low | Add pagination or --limit flag |
| Performance tooling | Tier-1 readiness passes, but `performance-health audit run` rejects the served profile bundle as uninstrumented | High | Reconcile the React 19 profile-marker contract and capture populated world, graph, list, and editor baselines |
| App shell | The document was 24px taller than the desktop viewport, producing page-level vertical scroll around the fixed-height shell | Medium | Keep the root and shell constrained to `100dvh`; route long content through intentional child scroll regions |
| Sidebar navigation | Real pointer clicks on agent/team rows stopped after `pointerdown` in the component-library collection gesture layer, so rows appeared inert even though programmatic clicks navigated | High | Keep the sidebar lists non-virtualized at their current scale and expose each row as a native button that owns pointer/click handling |
| Runs | The UI requested the retired `prompt-manager-heartbeat` profile alias; prompt-manager then asked agent-manager to resolve a profile that had no declaration/defaults and returned HTTP 500 | High | Use the declared `prompt-manager/heartbeat-judgment` profile for the default runs query and cover the request in UI/browser regressions |

### 2026-09-07 UX regressions — fixed

**Evidence:** `ui/scripts/ux-regression.mjs` now checks viewport containment, real pointer navigation for agent/team rows, and the runs tab's absence of the load error. The production browser check passes after a managed scenario restart. Focused UI tests pass 9/9, and the direct Connect `ListRuns` request returns HTTP 200 for the declared profile.

**Root causes:** the page-height discrepancy came from an auto-sized document around a `100vh` shell; sidebar rows relied on the component-library gesture wrapper rather than a native interactive target; runs used a stale profile alias that agent-manager could not resolve.

**Prevention:** the shell contract now uses `height: 100dvh` plus hidden page overflow, list rows have explicit button semantics, and run-fetch tests assert the declared profile key.
| Initial load | `/world` previously loaded unrelated graph/editor/diagram payloads; lazy boundaries now defer world/graph surfaces and production source maps are omitted; cold browser baseline was ~1.53 MB JS/CSS plus a 1.44 MB HDR asset | High | Split remaining shared editor consumers and defer Mermaid, Monaco language, and HDR work |
| World presentation | GPU-backed 25-actor run reached first ready/presented state at ~4.42 s; high profile GPU p95 was 13.35 ms and frame p95 16.9 ms | High | Stage a lightweight first presentation, then upgrade quality/assets; enforce measured first-present and frame budgets |
| Performance gate | Lighthouse configuration now exists and the performance phase reaches the real route gate, but `/world` and `/graph` remain below the 0.75 performance error threshold; budget check still passes | Medium | Improve route performance, then ratchet route budgets; reconcile profile instrumentation separately |
| Mobile | Narrow viewport correctly falls back to 2D with no horizontal overflow; headless check still showed 24 px vertical overflow | Low | Verify physical mobile/safe-area behavior and add a regression if reproducible |

Detailed evidence and source anchors: `docs/perf/2026-09-07-ux-performance-audit.md`.

---

## Cleanup History

| Date | Change | Outcome |
|------|--------|---------|
| 2025-01-25 | Aligned API with screaming architecture | All domains now have interfaces |
| 2025-01-25 | Added CLI domains for all API endpoints | Full CLI coverage |
| 2026-08-19 | Re-platformed Prompt Manager domains to generated proto/Connect bindings and retired their REST routes | 120 Connect commands; Program Runtime doctor reports 117/117 callable; measures-health probes all nine declared measures |

---

## Wiring Gaps

_No open wiring gaps._

---

## Deferred Work

| Item | Reason | Priority |
|------|--------|----------|
| Graph `recent-activity` scoring unwired | `RecentActivityScoreFromTimestamp` is implemented in [CODE: api/graph/scoring.go:RecentActivityScoreFromTimestamp] but `Node` lacks a timestamp field to feed it, so `recentActivityScore` returns a neutral 0.5. Wire it when nodes gain `updatedAt` metadata. | Medium |
| Qdrant integration | Optional feature, not core | Low |
| CLI shell completion | Nice-to-have | Low |
| Semantic search | Requires Qdrant | Low |

## Work ladder

- Rung: W0
- Evidence: the named-mention goal search was repeated on 2026-09-05 and all five matching goals were read. `contribution-inbound-triage` and `rapid-approval-flow` are now archived, so the previous evidence must not be described as active goals. Active goal `search-hub-federation-adoption` directs Prompt Manager to "formalize self-registration" and requires every member scenario to "self-registers an ACTIVE leaf, answers through `search-hub query`". No P0 operational target names provider self-registration; the P1 Semantic search target says "Discover relevant skills through optional vector search."
- Blocker: reconcile the active federation goal with the P0 contract before treating lower-rung gates as proof of whole-scenario completion. The 2026-09-05 world investigation is an explanation and recommendation review; no lower-rung gate or implementation repair was performed. The P2 Agent world target describes seeded terrain and health weather but supplies no loading, scene-transition, or camera-interaction acceptance budgets.
- Measured: 2026-09-05


### Scoped W3 performance repair — 2026-09-07

User-authorized implementation of the 3D UX audit recommendations. Evidence:
`docs/perf/2026-09-07-ux-performance-audit.md` implementation follow-up and
`ui/evidence/ux-performance-20260907/implementation-camera.json`.
The normal-mode 25-actor camera/movement gate passes 43/43 checks and 145 focused
regressions pass. Surface queries are sub-millisecond to 1.7 ms in the paired
approach journey; motion frame p95 is 17.1–17.2 ms. Broader unit/Lighthouse failures
are recorded in the report. This scoped W3 result does not resolve the W0
federation-contract concern above or certify the entire scenario.

Final 1600×1000 roster sweep qualifies the initial success: 100 actors passed all
25 interval checks; 400 actors missed park-orbit p95 by 0.5 ms, and one 25-actor
repeat had an office-zoom tail stall. These remain visible in retained evidence;
no gate was weakened. Managed React capture and analysis now work, and escaped
component-mark classification has a focused regression.

A final normal-mode 25-actor repeat passed 25/25 interval checks (17.0–17.5 ms
frame p95; no >50 ms frames). Static smoke now validates nonzero counters
(90 draws / 118,807 triangles), with 16.26 ms GPU p95. Its unchanged 12.76%
golden mismatch remains a failure. Earlier tail-stall evidence remains retained.

### Scoped W3 camera navigation and walking — 2026-09-07

User-authorized implementation following the camera-control audit. Explore now
uses explicit mouse/trackpad profiles, consistent fixed-lens dolly, immediate
wheel response, frame-rate-independent keyboard integration and focus-loss
cancellation. The persistent camera toolbar exposes pan/orbit, presets, Home,
Frame selection, Stop, sensitivity and input help. First- and third-person modes
share a grounded visitor body, swept capsule collision against structure and
rendered furniture, obstruction-aware chase camera, safe placement, optional
pointer lock, and restoration of the saved Explore pose. Controls and limitations
are documented in `ui/src/world/engine/README.md`.

Validation: 100 camera/HUD regressions and two input-lifetime regressions pass;
TypeScript, targeted ESLint and production build pass. The final built-browser
navigation journey passes 29/29 checks in park and office, including actual
pointer-lock Escape, rendered obstruction, preference persistence, stopping and
pose restoration. The 25-actor walking sample measures 16.8 ms frame p95 with no
frames above 50 ms. Evidence: `ui/evidence/navigation-20260907-verified/result.json`
and adjacent logs. This is a bounded desktop Chrome synthetic-input run; physical
trackpad/OS momentum and touch hardware have not been manually evaluated.
Earlier failed browser runs remain retained; their wheel/pan assertions sampled
the decimated render probe separately from the input receipt. The final diagnostic
snapshot publishes pose and input receipt atomically, and retains the same
behavior assertions.

Broader validation remains red. The world-wide run recorded 1085 passing tests
and seven failures: five configuration-literal gates across existing world layers
and two terrain-spacing variance assertions. Newly introduced navigation modules
were checked against the literal scanner after moving settings into config.
The terrain observation is filed as `knw-1788822428399263594`; the broader
configuration-gate observation is filed as `knw-1788822513569868254`. Test Genie run
`20260907-224027-da7a3712` reports API timeout, CLI/UI unit failures and unavailable
UI-health provider; retry `20260907-225030-32c6dabb` reaches UI-health and reports
UI standards debt. New scoped keyboard ownership, focus styling, shared test
renderer and literal findings were addressed; broader API/CLI and UI standards
failures are not claimed resolved. CLI rebuild cache failure is filed as
`knw-1788820956308841357`. These suite reports are retained alongside navigation
evidence. This scoped W3 repair does not resolve the W0 federation concern or
certify the whole scenario.

The existing Explore browser journey also passes 46/46 checks on the final build,
including focus/follow, automatic poses, obstruction recovery, cursor/center
surface zoom and editor cancellation. Evidence:
`ui/evidence/navigation-camera-20260907-verified/result.json`.

### Scoped W3 walking jump — 2026-09-07

Follow-up request adds Space and a visible Jump button to both visitor modes.
The shared body uses gravity, bounded physics steps, vertical capsule sweeps,
ceiling contact and landing. Horizontal movement remains available in the air;
a second jump requires landing, and keyboard repeat cannot trigger automatic
bouncing. The render loop now validates the actual airborne body instead of
snapping it back to terrain each frame. World regeneration still repositions the
visitor safely.

106 unique focused camera/HUD tests pass, including frame-rate parity from 5 to
120 FPS, takeoff/landing, ceiling clearance, clearing low obstacles, and focused
Space ownership. TypeScript, targeted ESLint and production build pass. The built
Chrome navigation journey passes 33/33 checks, including first-person Space,
third-person HUD jump, landing and no repeat while Space is held. Evidence:
`ui/evidence/navigation-jump-20260907/`. This extends the scoped W3 result and
does not certify unrelated scenario domains or physical input hardware.

Scoped Test Genie unit run `20260907-234418-fcef9359` completed with failures:
API timeout, CLI test failure, missing UI role in Code Facts, and broader UI
renderer-policy findings. The retained `testgenie-unit.json` records these limits;
the direct camera/HUD regressions and built-browser jump checks are passing.

### Scoped W3 visitor interactions — 2026-09-07

User feedback identified three interaction defects after walking and jumping were added. The body could handle terrain height but had no up/across/down sweep for low rendered objects; the coarse agent navigation grid also excluded furniture before visitor collision checks ran. Walking input suppressed all scene clicks, and mode changes left keyboard focus on the mode selector.

Visitors now step over objects up to 30 cm high when the full body and headroom clear. Terrain shoreline, slope, and boundary checks remain active; rendered colliders determine prop clearance. Entering either walking mode focuses the canvas. Clicks invite a selected agent to a navigable spot in front of the visitor; look-drags suppress selection. The agent faces the visitor on arrival. Presentation routing preserves live run status, and dismissal restores ordinary routing. The toolbar provides End conversation. An unreachable agent stays in place.

Focused validation passed 121 camera, input, HUD, and store checks. A subsequent store-only run passed 12 checks, including notification on dismissal. Browser evidence, build output, lint/typecheck results, and the scoped Test Genie result are retained in ui/evidence/visitor-20260907-final-verified. The browser journey covers real rendered-agent clicks in both walking modes, camera preservation, dismissal, automatic focus, immediate arrow input, jumping, wall collision, and low-obstacle stepping. Test Genie unit run 20260907-235713-b4afb689 remains FAIL: CLI execution failures, API timeout, and UI renderer-policy findings. These broad findings do not establish scenario certification; the scoped navigation results are reported separately.

### Scoped W3 park log step cutoff — 2026-09-07

The previous 30 cm step limit excluded the shipped park log: its registered height is 0.1732 metres and the park prop scale is 1.8, producing a 0.31176 metre obstacle. The previous 25 cm box regression did not cover this asset. A regression loading the actual GLB and using the renderer's geometry preparation and instancing reproduced the blocked traversal at the old limit. Raising the limit to 45 cm passes approaches from both directions, three placement rotations, and 5/60 FPS. All 14 walking tests pass, including taller-obstacle and low-ceiling rejection. The maintained browser journey also checks 40 cm obstacle traversal and borrows the rendered park log parts for a direct log regression. Evidence is retained in ui/evidence/log-step-20260907-verified.

All 48 browser checks passed, including traversal of the actual rendered log parts at 0.31176 metres high and a 40 cm box without jumping. Typecheck, targeted lint, and build passed. Scoped Test Genie unit run 20260908-001619-c0323004 remains FAIL with CLI execution failure, API timeout, and UI renderer-policy findings; the terminal result is retained with the scoped evidence.

### Scoped W3 agent invitation execution and captured picking — 2026-09-07

The prior browser journey verified selection state but did not verify motion or arrival. Extending it reproduced invited demo agents remaining at tick zero in both walking modes: synthetic rosters disable the runtime clock. The runtime now advances while an explicit visitor invitation is active, then freezes again on dismissal when ordinary demo stepping is disabled. Normal live-roster stepping is unchanged.

Captured mouse interaction had two defects. Scene picking used frozen cursor coordinates instead of the centre aim, and pointerdown requested pointer capture while pointer lock was active, causing InvalidStateError. The canvas event manager now casts through the centre while locked and uses canvas-relative client coordinates otherwise. A centre marker identifies the aim. Walking input omits the incompatible capture request while locked. The browser journey checks actual approach, arrival, facing, dismissal, and captured clicks in both camera modes.

The scoped Test Genie unit run 20260908-035054-b7d05ba0 returned provider_unavailable with no diagnostic reason; its waiter exited 2. This is unavailable broad validation evidence, not a passing suite. Filed Scenario QA observation knw-1788839673038591651. The terminal JSON is retained in ui/evidence/agent-approach-final/testgenie-unit.json.

Final focused validation passed 17 tests. All 54 browser checks passed with zero runtime errors, including actual arrival/facing and centre-aim selection under pointer lock in both walking modes. Typecheck and production build passed; targeted lint reports zero errors and one React Fast Refresh export warning for the event-manager test seam. Evidence: ui/evidence/agent-approach-complete. The browser reads the production build; fixture-only inspection supplies observations and a deterministic aim turn, while real mouse clicks perform selection.


### Scoped W3 scene redesign — 2026-09-08 (in progress)

Authority: user attached nine-stage scene redesign at d55310b2-3acf-455c-a9c1-8739bb0d893a/pasted-text-1.txt. Scope includes foundations, cohesive assets, playable reference spaces, enclosure management and invitations, campground generation, office architecture/furnishings, continuous atmosphere/deep night, rabbits/squirrels, and end-to-end validation. No scenario-wide readiness claim.

Recall found related wildlife preview work; no established fix for the reported scene appearance. Search Hub returned degraded program-runtime and conversation search providers; diagnosis uses current files and runtime evidence.

Baseline: ui/evidence/scene-redesign-baseline/{park,office}.png (built Chrome, six synthetic actors, high profile, day). Both render without runtime errors and sim invariant failures. Hypothesis A (missing slab rendering) is contradicted by visible wall/floor slabs in the office capture. Hypothesis B (low shared wall height and near-identical floor/terrain treatment) is supported: wallHeight=0.7m and floor thickness=0.02m; both scenes use room slabs. Hypothesis C (unstructured furnishing choices) is supported by interiorFor arbitrary filler locations/angles and the empty room capture.

Delivery sequence and evidence ledger (all pending unless stated):
1. Foundation baseline — captured; navigation/geometry agreement requires replacement and tests.
2. Art palette — existing CC0 Kenney base plus original matching structures; add distinctive shelters/signs/landmark and retain asset provenance/budgets.
3. Playable campsite and office reference — pending visual review across three cameras, day/night.
4. Spaces — boundary/entrance/occupants/gathering/meeting metadata; context menu and actual doorway invitation; camera cutaways.
5. Campground — tent/cabin/RV arrangements, team signs, fires/seating, paths, landmark, growth/persistence.
6. Office — architecture, workstation/meeting/lounge/shared arrangements, decorations and clear circulation.
7. Atmosphere — day/clouds, evening, quiet hours, Milky Way/meteors, light coordination, override.
8. Wildlife — rabbits and squirrels, habitats/behavior/caps/reduced motion/picking.
9. Final validation — sizes/seeds/saved worlds/cameras/time, invitations/signs/menus/jump/step/performance; remove superseded behavior.

Scene reference validation found an additional in-scope renderer defect: toggling Office cutaway to full walls increased the slab instance count beyond the GPU instanceMatrix capacity. Drei initializes those buffers once. CPU state changed but rendered surfaces stayed low. The maintained scene-design browser journey reproduces two buffer-capacity failures in ui/evidence/scene-design-buffer-before. Slab batch allocation now keys on capacity; post-fix browser verification is pending.

Foundation implementation checkpoint (not completion of the redesign):
- Shared space metadata and structural boxes are implemented for campsites and offices. Campsites allocate named tent/cabin/RV shelters in units of four occupants, retain station/table/room IDs, provide local entrance/meeting/gathering anchors, picnic tables and fires, and add the noninteractive central sculpture. Existing serialized layout overrides still address the same place IDs; a full saved-world browser journey remains pending.
- Enclosures and physical entrance signs open a space menu with management and individual occupant actions. The menu releases mouse capture, takes keyboard focus, and Escape returns focus to the canvas. Walking invitations use the existing visitor-conversation simulation. Direct simulation tests prove enclosed agents leave and reach a visitor in both scenes. Full browser approach/arrival journeys remain required.
- Office slabs now have full walking height and Explore cutaways. The capacity regression is fixed and verified in the browser. Current windows are decorative panels, and offices still require actual architectural window openings, ceilings/doors, better floorplate sizing, workstation/shared-space arrangements and richer detail.
- Imported furniture footprint origins are corrected using measured registry bounds. Anchored office decoration sockets replace free random positions/rotations; rugs sit under tables, other decorations avoid desk/seat access, and meeting tables occupy the side opposite the workstations.
- Repeated roof/detail geometry uses instancing. Tent fronts now have a doorway cut into the gable; campers have wheels and trim. Roof collision and final shelter/detail quality remain pending.
- Broad Test Genie run 20260908-043550-7afa684e completed with failures: CLI and UI test commands failed, API command timed out, and the provider reports renderer-policy/companion findings. Command artifact also reports host swap pressure. No root cause or attribution claimed. Scenario QA report knw-1788842511243106138 retains the observation. Evidence is under ui/evidence/scene-design-foundations.

Next work: finish the playable office reference and scene-specific furnishing/architecture; refine campsite props and fire illumination; verify shared collision/pathing against rendered geometry including roofs and menu invitations. Then implement the coordinated daytime/deep-night atmosphere and squirrels/rabbit improvements. Complete the original nine-item acceptance matrix, broad relevant world regressions, visual review at day/night in every camera mode, saved-layout/growth journeys and performance gates before claiming the goal complete. The reference appearance is not yet approved as finished.

Checkpoint evidence: 48 distinct focused tests pass across layout/navigation, enclosure approach, UI menu focus/actions, authored interior sockets and measured asset anchors (latest generation batch 29; retained unchanged nav 14, anchors 3; latest menu 2). The final foundation browser journey passes 21/21 checks with zero runtime errors, including both scene menus, focus/Escape, three camera modes and GPU buffer capacity. Production build and asset registry check pass; targeted ESLint has no errors and the pre-existing Places helper export Fast Refresh warning. Evidence: ui/evidence/scene-design-foundations-verified. These are foundation checks, not acceptance of the full nine-stage redesign.


Office architecture and doorway checkpoint (2026-09-08, full redesign still active):
- Capacity-sized paired office wings replace oversized BSP subdivisions in generation. Shared lounge and kitchenette have their own room/door records. Office structures now include glazed window openings, ceilings, open door leaves, skirting/window trim and shared kitchen fixtures. Desk rows have a separate 2.7 m pitch; workstation details include coordinated monitor variants, mats, keyboards and mugs. Imported desk height is read from the asset registry and scene scale.
- The large-roster regression initially failed at seed 1 / 1,000 agents: an incomplete wing reserved an absent opposite row, making the footprint 102.4 by 161.5 m and placing a room corner outside the terrain radius of 90 m. The height sampler correctly returns zero there. Per-side wing depth now follows actual demand. All 16 seed/headcount combinations in the floorplan test pass. This proves those tested capacities, not unbounded world growth.
- Expanded centre-terrain checks revealed changes outside the declared transition at rectangle corners. The old expansion shortened a radial blend after expanding both rectangle axes, which leaked beyond the original corner arc. Flattening now measures support and outer fade from one rectangle; a 6 m transition retains bounded slopes across all eight sweep seeds. All 16 centre tests pass, including unchanged exterior samples and flat interpolated pad corners.
- Visual review invalidated the first doorway browser assertion: crossing the doorway plane outside its width was accepted. That evidence (scene-office-entry-verified) is not valid doorway proof. The maintained journey now requires starting outside, finishing inside, and staying within the doorway width. It also retains nearby collider evidence. The stricter test reproduced an office failure in scene-office-doorway and scene-office-doorway-diagnostic.
- Two physical defects were reproduced with failing regressions and repaired. Spawn treated a 6 cm floor slab as an occupied location and moved the visitor 6.5 m outside a test room; it now resolves a low supporting surface before searching elsewhere. Step traversal demanded the full 45 cm lift, so a small floor lip under a 2.25 m door lintel blocked movement; it now uses the available upward clearance, retaining the horizontal obstacle/headroom checks. Tall-obstacle, low-ceiling, actual park log, jump and frame-rate regressions still pass.
- Campfires now participate in the same bounded point-light pool as lamps, using flame height and warm color. A regression verifies emitter selection, light height/color/intensity and reuse of the same pool slot when moving from fire to lamp. Night browser checks confirm active fire-colored point lights in both walking modes. Quiet-hours fading, final light tuning and full atmosphere integration remain pending.

Current evidence: ui/evidence/scene-office-doorway-fixed and ui/evidence/scene-office-night. Focused run: 58 tests across seven files pass. Day browser: 23/23; night browser: 25/25; zero runtime errors. Both scenes are captured in Explore/first/third person. Actual entrance traversal is verified in first person using the shared walking body; a separate third-person traversal still belongs to the final matrix. Typecheck, production build and 25-prop asset check pass. Targeted lint has zero errors and the existing Places helper-export Fast Refresh warning. Ground-level office capture now shows the interior after real entry, not a corridor-side false positive. These captures establish progress, not final scene-art approval.

Scoped Test Genie unit run 20260908-050725-d3e61398 completed FAIL (199 s): CLI/UI execution failure, API timeout, renderer-policy findings. It began during this iteration and does not establish exact-input coverage of the later movement/light edits. The terminal JSON is retained with this checkpoint and relates to existing QA observation knw-1788842511243106138; no duplicate report or attribution claim. Focused current-source tests cover the later edits.

Next: complete shelter roof collision and camp/office visual polish (including corridor lamp clearance and circulation), menu invitations through actual browser arrival/facing, visiting occupants in shared spaces, and saved-layout/growth validation. Then finish coordinated sky/deep night and rabbits/squirrels, and the original full camera/time/performance acceptance matrix. No original recommendation has been removed from scope.


Roof collision and space-interaction checkpoint (2026-09-08, full redesign still active):
- Roof geometry is now an immutable, shared source in scene/roofGeometry.ts. The camera/body sweep supports explicitly tagged triangular sheets through cached convex half-spaces expanded by the sphere/capsule support. It applies the actual instance transform, including nonuniform scaling. Roof slopes and gables block movement/jumps/boom motion while the tent opening and interior remain clear. Scene-layer integration tests consume the actual roof geometry at three rotations; engine-layer tests retain their downward-import boundary. Bounds/outline diagnostics remain conservative boxes, not the physical narrow-phase hull.
- Space menus list agents physically present in a room as well as assigned members who are away. Shared lounge/kitchen visitors are visible in the menu. Here/Assigned-away labels distinguish the two meanings. A dedicated WorldStore.subscribeState channel publishes moving snapshots without changing the discrete view subscription contract, and is mounted only with the menu. Unit tests verify updates on actual simulation ticks, presentation changes, focus retention and unsubscription. Scene changes dismiss the menu.
- Corridor lamps now sit near edges, leaving door approaches and corridor intersections clear. Rendering and pooled lights share those placements. Three placement tests and five light-pool tests pass.
- The maintained scene journey walks through real entrances, returns outside, clicks enclosure surfaces, invites enclosed occupants in first and third person, waits for actual approach/arrival/facing, and checks that the player stays in place. A first-person tent gable needs a small look-up gesture from the threshold; the initial candidate set was outside the viewport, not a failed click. The gesture now uses normal pointer input. Day scene journey: 39/39 in scene-roofs-invitations-final. Latest night scene journey: 41/41 in scene-roofs-live-menu-night, including active campfire lights.
- The older navigation journey selected an agent solely by projected screen position. That assumption no longer provides a visible-agent fixture when campsites are enclosed. Its direct-click/captured-picking portion now places one synthetic agent on open navigable ground through the browser-only fixture. Actual enclosure emergence remains independently exercised by the scene journey. Failed evidence is retained in scene-roof-navigation.
- The broader camera run then exposed a real raised-floor stepping bug: after stepping onto a 40 cm object atop a 6 cm slab, movement stopped because body-to-terrain separation was treated as a 46 cm cliff. A regression failed before the fix. Terrain grade now compares the underlying terrain at source/destination; collider sweeps determine the actual support/step height. At 5 and 60 FPS the visitor clears the object and returns to the slab. Existing taller-obstacle, low-headroom, terrain-cliff and actual park-log checks remain passing. All 54 browser navigation checks pass in scene-roof-navigation-verified, including direct/captured clicks, approach/facing, jumping, log/low-box stepping, focus, immediate stopping and the existing frame-time gate.

Current checkpoint evidence is indexed by ui/evidence/scene-roofs-live-menu-night/validation.json. There are 70 distinct passing focused tests across nine affected files (latest observations replace overlapping earlier runs). Typecheck/build/asset check pass; targeted lint reports zero errors. Final night browser validation follows the full-snapshot menu subscription change; earlier day/navigation captures precede that last UI subscription change. Broad Test Genie run 20260908-052525-97fa893b completed FAIL after 166 s with execution-failure/timeout and existing policy findings, related to QA observation knw-1788842511243106138. Its terminal JSON is retained; it began before the late movement/subscription edits and is not current-input passing evidence.

The original nine-stage goal remains active. Next priority is the coordinated atmosphere/deep-night implementation and rabbit/squirrel behavior, followed by scene art/layout polish and the full saved-world/growth/camera/time/performance matrix. Roof collision and interaction correctness have advanced; final art approval, all visual variants, third-person entrance traversal, saved override migration/growth behavior, deep-night extinguishing/Milky Way/increased meteors, squirrels, and full performance coverage are not claimed complete. Superseded floorplan helper logic still needs retirement once its replacement evidence covers those cases.


Coordinated atmosphere checkpoint (2026-09-08, full redesign remains active):
- Shared civil-time quiet hours fade in 23:00–01:00, hold to 04:00, and fade out by 05:00. Resolved lamp/hearth emission reaches zero and moon-colored navigation fill remains. Named fixed presets keep their ordinary behavior.
- Deep night, Midday, Evening, pause/play and live-time controls now work outside the workbench. Selecting a clock preset switches away from a fixed lighting preset. Exact UTC remains in a disclosure. Civil-time seeking handles configured zones and DST, including the spring missing-hour case, with one notification per seek.
- Added a bounded procedural Milky Way and additional stars within existing quality budgets. The sky follows cloud/sun/moon visibility. Added deterministic quiet-hours ordinary meteors without altering the base stream or rare-fireball cooldown; combined meteor capacity stays two and reduced motion suppresses extras.
- Browser visual inspection found the retained HDR sky was gray at deep night. Replaced the visible HDR background and flat cloud plane with a camera-centred procedural gradient and shaped clouds, keeping the HDR for reflections. Refined the blue daytime palette and softened/narrowed the galactic dust lane after captures. These are progress captures, not final art approval.
- Latest evidence: ui/evidence/deep-night-verified. 37 focused tests across eight files pass; 27 real-browser checks pass, zero page/GPU errors. Both scenes captured in all three camera modes at deep night, with targeted sky/day captures. Typecheck, production build and targeted lint pass. The first browser artifact failed a fallback-color assertion while the old HDR still owned the background; the final implementation and journey now establish that ownership explicitly.
- Scoped Test Genie unit run 20260908-055348-7e3a5f3b completed FAIL (171 seconds): CLI/UI execution failures and API timeout. Existing QA observation knw-1788842511243106138 covers this repeated observation. Run began before the later sky refinement, so it is not evidence of exact final-source whole-scenario coverage.

Next: finish campfire flame/ember presentation and final sky/lighting art; improve rabbits and add squirrels with habitats and reduced-motion behavior. Finish campsite/office reference art and circulation, stable saved worlds/growth, third-person doorway journeys, the full camera/time/seed/performance matrix, and obsolete scene/config cleanup (old cloud-plane tuning fields no longer drive the new sky). The nine-stage objective is unchanged and incomplete.
