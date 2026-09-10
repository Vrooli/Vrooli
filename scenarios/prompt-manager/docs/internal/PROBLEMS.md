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


Fire/wildlife checkpoint and user review (2026-09-08, redesign still active):
- Implemented pooled flame/ember cards, weather/quiet-hour fading, recognizable grazing rabbits and bounded foraging/climbing squirrels on exposed tree trunks. Frozen time and reduced motion retain appropriate static fire presentation and suppress ground wildlife. Fixed the first-render clock sample so switching to a selected civil time does not briefly apply the previous lighting hour.
- Focused current-source validation: 44 tests in seven files pass; typecheck, build and scoped lint pass. Browser artifacts: ui/evidence/fire-ember-verified and ui/evidence/ground-wildlife-verified (45 wildlife checks, zero errors). Broad Test Genie run 20260908-060923-2674a7e4 failed with recurring execution/timeout findings covered by existing QA knw-1788842511243106138; it began before the final clock/wildlife refinements and is not exact-source passing evidence.
- New user acceptance requirements: office ceilings throughout the building in first/third person; directional Sims-style wall/ceiling cutaways in Explore; comfortable doorway entry/exit from realistic approaches; recognizable phased moon with phase-dependent illumination; WASD walking plus arrow-key looking; tangential log seating with explicit asset orientation; complete visible fire rings/flames and smoke; real agent conversation bubbles when invited agents arrive (immediate in Explore), sending through existing agent chat, unread indicators; persisted camera mode/position/orientation across reloads.
- Investigation confirms Explore currently truncates every wall, walking arrows duplicate WASD, and seat rendering applies one rotation to all assets. Room ceilings exist in structural data but corridor roofs and actual runtime visibility require verification. No new requirement is declared complete on the basis of existing code alone.


User scene-review fixes (2026-09-08, implementation checkpoint; full goal still active):
- Walking now uses WASD for translation and arrows for yaw/pitch, with angular speed independent of pointer sensitivity/inversion. The HUD describes both. Physics regressions exercise simultaneous walking/look, no arrow translation, stopping, and several frame rates.
- Office Explore uses world-oriented outward surface normals to remove camera-facing wall/window/lintel sections and all ceilings. Far walls remain full height. Walking modes retain full room architecture and now also corridor ceilings. The directional view updates by camera sector rather than rebuilding slabs every frame. Shared structural boxes retain the same physics authority.
- A new real-browser off-centre doorway matrix reproduced chairs blocking entry. Some team rooms placed a meeting chair directly behind the door, stopping the visitor only 44 cm inside. Meeting arrangements now occupy a front corner bay, and room sizing includes the entire chair ring plus a clear doorway aisle. A generated-world regression failed before the fix and passes afterward.
- Strengthening that journey to require 1.2 m of actual entry caught another chair in the shared lounge. The lounge seating circle now has an opening toward its entrance; remaining seat IDs are preserved. Team rooms and kitchen already pass the stricter journey. The focused lounge rerun is retained separately.
- Seat rendering now accounts for model orientation: imported log long axes run tangentially around fires/tables; sitter facing remains toward the centre. Measured registry dimensions verify the axis, and office chair orientation is retained.
- Campfires now have ten low-poly stones and six billboard smoke puffs per hearth, alongside flame/ember cards and retained logs. Smoke moves with the shared clock, freezes with time, stops with flames, and is suppressed by reduced motion. Found and fixed effect cards inheriting furniture collision from their parent group; explicit opt-out prevents flames/smoke becoming invisible obstacles. Solid stones retain low-object collision.
- Current-source focused batch before the final lounge seat filter: 57 tests in eight files pass; typecheck/build/asset check pass; scoped lint has zero errors and the retained Places helper-export Fast Refresh warning. Final lounge/layout tests and focused browser rerun supersede the relevant earlier cases. Sources and weather/asset docs now describe the completed fire/wildlife presenters.
- Browser evidence: fire-ring-smoke-verified (34 passing checks); world-review-navigation (56 passing checks including arrows, focus, jumping, logs, direct/captured selection, approach/facing and frame-time gate); world-review-enclosures-verified (41 passing checks, both scenes and both walking camera invitations). The first enclosure rerun missed its target because a fixed back-walk duration placed the visitor differently after doorway clearance improved; the maintained journey now exits to a measured doorway distance and uses actual visible front-wall targets. Failed artifact world-review-enclosures retains the empty candidate list.
- Door evidence sequence: doorways-review-first and doorways-review-diagnostic reproduce the team-chair obstruction; doorways-review-fixed passes 86 checks at the initial shallow-entry threshold. doorways-review-verified strengthens entry to 1.2 m and exposes the lounge chair. doorways-lounge-verified tests the repaired lounge separately. These are explicit evidence scopes, not a claim that the earlier weaker threshold proved comfortable circulation.
- Scoped Test Genie unit run 20260908-072728-ca1fcd69 completed FAIL after 167 seconds with CLI/UI execution failure and API timeout. Existing QA observation knw-1788842511243106138 covers the recurring broad-run finding. This run began before the final lounge-seat refinement; it does not establish whole-scenario success for the final source.

Next required work from this user review remains: verify/refine the rendered lunar phases and connect phase/horizon/weather to actual moonlight (currently phase affects stars but the night key/fill is largely constant); persist camera mode/position/yaw/pitch/boom/Explore pose across reloads; implement real member-bound conversations and unread markers with arrival-gated walking bubbles and immediate Explore bubbles. Existing StartChatDialog launches generic runs with a task/profile and must not be mistaken for selected-team-member chat. WorldPreferences currently persists scene/quality/period but no camera pose. CameraRig starts at hero/intro, explaining the reload reset.

Original nine-stage remaining work is still required: complete scene art approval and all visual variants, stable saved layouts and growth, representative seed/size/camera/time/performance matrix, and obsolete scene/config cleanup. Final squirrel climb capture still clips the head into foliage; exposed-trunk filtering alone did not resolve that visual issue. Lower/measure climb limits against the actual canopy before final wildlife art approval. No original acceptance item has been silently removed.


Camera persistence checkpoint (2026-09-08, full redesign still active):
- Live worlds now save Explore eye/target/zoom, navigation mode, walking position/yaw/pitch/boom, and the previous Explore pose in browser-local storage, independently for each scene and seed. The data module owns schema validation and bounded writes; CameraRig consumes a small memory interface. Invalid or unavailable browser storage does not break navigation.
- Saves retain the latest pose during movement and flush on pagehide, visibility changes, and rig disposal. Restored views skip the intro. Walking restoration checks current terrain/collider clearance, relocates blocked positions through the existing bounded spawn search, and lets gravity settle missing support or interrupted jumps. Returning to Explore retains the original saved view and zoom after reload. Pointer capture remains off until a gesture.
- Explicit focus URLs, imported recipes, and synthetic captures keep their requested framing. The real-browser persistence journey uses the live read-only roster, changes only local camera state, and sends no agent messages or management mutations.
- 29 tests in three files pass: storage continuity/isolation/invalid data/bounded writes, actual walker restore with a new solid obstruction/missing support, and rig initialization/page-exit saving. Typecheck, production build and scoped lint pass. Browser evidence ui/evidence/camera-memory-verified passes all 17 checks in Park and Office, covering all three modes, position/orientation/third-person distance, retained Explore view, canvas focus, and an enabled intro on reload. No page errors.
- The broader Test Genie unit run 20260908-075034-02dd48e5 completed FAIL after 170 seconds. Terminal evidence is retained in ui/evidence/camera-memory-verified, alongside the passing focused checks. Repeated execution/timeout findings remain associated with existing QA knw-1788842511243106138; whole-scenario success is not claimed.

Next priorities remain real selected-member conversations/unread markers and lunar appearance/phase-dependent world illumination, followed by the original saved-layout/growth, scene/wildlife art and full performance acceptance matrix. Camera persistence is now implemented; it does not constitute completion of the nine-stage redesign or the other newly requested features.


## World member conversation checkpoint — 2026-09-08

- Rung: scoped W3 behavior from the operator's scene-review request. Existing nine-stage goal and review additions remain active; this does not claim W0–W2 or full scenario readiness.
- Server-owned member conversations now extend the existing HeartbeatService.CreateRun JSON operation. Membership, context (without HEARTBEAT.md), execution profile and team attribution resolve on the server. Unassigned personas retain agent context without team authority. Deterministic task IDs, message/member binding and Agent Manager run idempotency plus durable readback prevent duplicate initial runs on retry.
- Conversation bubbles open immediately in Explore and after nearby arrival in walking modes. Explicit send creates the run; continuation uses its existing action. Browser-local run pointers/read receipts survive reloads, polling is bounded and stops for drained terminal runs, and preview agents cannot send. Keyboard events stay in the composer. Full transcripts remain in Agent Manager.
- Visual review found Explore could frame a selected actor behind its shelter wall and let the actor wander away during chat. Selected enclosures now reveal their cutaway before framing, and Explore conversations hold actors stationary. Switching to walking resets the stationary flag and restores approach routing. A simulation regression covers both modes.
- Visual review also rejected an initially passing unread-state assertion: an unsupported circular glyph stalled the bundled Latin font's text update, leaving old cluster glyphs visible. The label now uses supported text, clears enclosing roofs, and the browser asserts textRenderInfo matches the requested string. Final screenshot visibly reads “New message · Debt Curator”.
- Evidence: `ui/evidence/world-conversations-validated/validation.json`. 29 focused UI tests (5 files), 29 top-level Go tests (42 checks including subtests), 7 conversation browser checks, and 58 navigation checks passed. Types, build and changed-world lint pass. The lifecycle restart is healthy. Positive conversation tests use wire/client fixtures; live validation only checks rejection of an invalid conversation request, with no model launch or real-agent message.
- Broad Test Genie run `20260908-082839-189d2048` remains failed on the previously recorded CLI execution, API timeout and UI renderer/policy/coverage findings (QA `knw-1788842511243106138`). It precedes the final marker placement/font change; final focused rendering evidence covers that change. Initial discarded visual candidates remain in the evidence directory family.

Remaining work: lunar appearance and phase/horizon/weather-dependent actual world illumination; the original saved-layout/growth and full scene/camera/time/performance acceptance matrix; final wildlife/campsite/office art review and retirement of superseded scene logic. Conversation functionality is a completed checkpoint, not completion of the overall redesign.


## Moon and solar-night checkpoint — 2026-09-08

- Scoped W3 implementation of the user’s lunar appearance/luminosity requirement. The moon now has a grey crater/maria surface, recognizable waxing/waning phases and a sky-blended unlit side. Actual directional illumination shares the visible celestial-body directions and fades with phase, horizon and cloud coverage. Moonless nights retain bounded ambient navigation fill. Obsolete fixed key-light angle tuning fields were removed.
- The first browser pass exposed late-evening pink sky and exposure below the intended navigation floor. Solar height now controls the transition to night sky/fog and bounded fill, while the shared clock continues to own quiet hours. The initial failed artifact remains in moon-first.
- Evidence: ui/evidence/moon-verified/validation.json. All 46 focused tests in five files pass, along with typecheck, production build and scoped lint. Moon browser: 22 passing checks; atmosphere browser: 34 passing checks across both scenes and all cameras, including quiet-hour extinguishing, smoke, weather, freeze/reduced motion, stars/Milky Way and return to daylight. Reviewed final full/quarter/crescent crops and full/new ground captures.
- Final-source scoped Test Genie run 20260908-084948-7c6b74cd completed FAIL after 166 seconds with recurring CLI/UI execution, API timeout and policy/coverage findings associated with QA knw-1788842511243106138. Terminal evidence is retained. No whole-scenario success is claimed.

The user-review camera, doorway, fire/log, chat/unread, persistence and lunar additions now have implementation checkpoints. The original goal is still active: saved-layout/growth journeys, final reference scene/wildlife art review, representative size/seed/camera/time/performance acceptance and obsolete scene cleanup remain.


## Squirrel habitat geometry checkpoint — 2026-09-08

- Scoped W3 repair of the remaining wildlife visual defect from the nine-stage redesign. Regressions reproduced heads entering foliage and paws gripping away from stems in both scenes. Climb height and attachment now use measured leaf lower bounds and stem radii, biome/instance scale and terrain support. Trees with insufficient exposed trunk are skipped.
- Lower-angle visual review distinguishes canopy occlusion from geometry intersection. The browser now measures visible head clearance and paw contact against actual leaf/bark geometry at the highest perch. Offscreen squirrels are excluded from that geometry oracle because their trees are intentionally culled; the population/pool checks still cover the full active pool. Probes start outside bark to measure paws that overlap the surface. Earlier failed candidates are retained.
- This investigation also reproduced stale raycast bounds after a vegetation pool slot changed identity. Upload now invalidates cached instance bounds for lazy recomputation. A failing-then-passing raycast regression covers slot reuse; the existing allocation and bounded-culling tests still pass. This was a separate defect, not the sole cause of every failed browser measurement.
- Final evidence: ui/evidence/squirrel-world-validated/validation.json. 23 focused tests in three files, all 51 wildlife browser checks and all 58 navigation checks pass. Typecheck, production build and scoped lint pass. Final perch captures in Park and Office visibly retain the heads below the canopy with paws against the stem. Navigation rerun includes direct/captured agent selection, approach/facing and greeting, arrow looking, focus, jumping, low-object/actual-log stepping, stopping and the existing frame-time gate.
- Final-source scoped Test Genie run 20260908-091413-1907239d completed FAIL after 156 seconds. Terminal evidence and maturity findings are retained, associated with existing QA knw-1788842511243106138. Whole-scenario success is not claimed.

Current user-review implementation checkpoints now cover walking office ceilings and directional Explore cutaways; comfortable doorways; WASD movement plus arrow look; log facing and complete smoky fires; arrival-gated member conversation bubbles and unread markers; per-scene/seed camera persistence; lunar phases and actual phase/horizon/weather-dependent moonlight.

The original goal remains active. Next required acceptance work is saved-layout reload and team growth in the browser, final campsite/office reference art across all cameras/day/night, representative seed/size/performance checks, and retirement of superseded room/slab/cloud tuning logic after proving compatibility. Old buildRoomSlabs is still used only for rooms without space metadata; verify saved/imported compatibility before removing it. Camera memory and selected-member conversations already have their own retained browser evidence; do not repeat those investigations from scratch.


## Saved layout and team growth checkpoint — 2026-09-08

- Scoped W3 implementation and validation of the original saved-world/growth requirement. Moved campground terraces previously remained at their generated location, and biome classifications described the terrain before terracing. Regressions reproduced uneven support and 128 stale biome cells. Final saved transforms now precede terraces and biome classification.
- Room drags now save orientation as well as position. Legacy position-only campground saves derive a stable heading toward the commons. When growth overlaps the commons or a neighboring site, a bounded cooperative search moves whole sites to nearby clear ground without changing orientation or member station IDs. The editor explains adjusted spaces and offers Reset layout when a saved configuration cannot be resolved.
- The real browser journey then found grown office rooms outside the recorded floorplate. A separate failing regression proved that moved doorways could lack a connecting floor even when terrain pathfinding succeeded. Office placement now reserves each room with a clear entrance passage to the authored halls. Passage floors receive the existing walking-mode ceilings. Final rotated room/hall extents determine the building footprint and level terrain region. The invariant now checks actual connected corridor flooring across the doorway, as well as navigation to the lounge. Rotated and overlapping saves have deterministic repair regressions.
- Evidence: ui/evidence/saved-layout-walk-validated/validation.json. All 64 focused tests across nine files pass; typecheck, production build, scoped lint and diff whitespace checks pass. All 33 browser checks pass with no page errors: real room dragging, persisted transform, child furniture/door translation, exact reload restoration, four-to-twelve-member growth, stable stations/orientation, clear site footprints and invariant checks in both scenes. The restored grown office also passes entry/exit at offsets -0.4, 0 and 0.4 in first and third person, with overhead ceiling checks and ground-level screenshots. Browser writes use in-process WorldService/roster wire fixtures; no live layout edits or agent runs.
- Final-source scoped Test Genie unit run 20260908-095637-cbb77b12 completed FAIL after 166 seconds with recurring CLI/UI execution failures, API timeout and policy findings associated with existing QA knw-1788842511243106138. Terminal evidence and findings are retained. No whole-scenario success is claimed.

The original nine-stage goal remains active. Saved layout/growth now has a completed implementation checkpoint. Remaining work is the final campsite/office reference art review across variants and cameras/day/night, representative size/seed/performance acceptance, and removal of superseded room/floorplan/cloud configuration behavior after checking saved/imported compatibility. These are implementation and review tasks, not a request for additional user approval.


## Scene cleanup and large-team acceptance checkpoint — 2026-09-08

- Removed the unused binary office subdivision/assignment helpers, old low-wall renderer, legacy nav wall/aisle fallback and their obsolete controls. Live layout saves contain transforms, not serialized room geometry; imported recipes regenerate current spaces. A compatibility regression verifies that old controls are discarded while edits and active tuning restore. Door records now share the actual architectural doorway width. Team focus uses rotated footprint extents and terrain-relative structure height. Updated the generated tuning documentation.
- The hardware size/seed matrix reproduced unreachable sleeping stations in 100-agent campgrounds. Bedrolls previously borrowed office dimensions and their standing positions touched their own bed edges or squeezed between rows. Dedicated bedroll dimensions and a wider cross aisle now keep bodies clear of both rows. Independent regressions check every station against other bedrolls and require commons reachability for 25/100 members per team at seeds 1/7/99.
- One 400-agent campground could not place its plots on seed 99. A bounded probe proved that rotating plots by a quarter turn admitted all four. Recovery now considers alternative headings, spatially distinct candidate choices and the actual rotated corner boundary, with bounded branching and sample expansion. Ordinary successful greedy layouts retain their preferred orientation; saved room orientation remains authoritative.
- The 400-agent office stress cases then failed the 50 ms movement-frame gate, with third-person p95 reaching 166.8 ms. CPU profiles showed collision queries repeatedly updating unrelated scene descendants and testing distant furniture. Queries now update only eligible collision meshes/ancestors and use a world-space broad phase before detailed local collision work. The three office stress reruns pass both walking modes, worst p95 20.5 ms. All 40 focused collision/walking/roof/route tests and all 58 navigation browser checks pass, including jumping, stepping, agent approach and focus.
- Evidence: ui/evidence/world-office-collision-optimized/validation.json. Latest distinct focused observations total 245 tests across 15 files. Types, production build, scoped lint, whitespace checks and the 25-prop asset registry check pass. Eighteen small/medium matrix cases cover both scenes, seeds 1/7/99 and rosters 4/25/100; worst active movement p95 is 19.9 ms. Six 400-agent cases are covered by the passing campground captures and the final office rerun. These are synthetic-roster camera/rendering checks, not 400 simultaneous live model executions. The evidence index states which captures precede the final collision optimization; no false exact-input or golden-publication claim.
- Captured tent, cabin, RV and office references in all three cameras by day and night (30 browser assertions). Visual review confirms readable entrance signs, complete smoky fires, tangential seating and walking office ceilings. It also rejects the RV as a finished reference: its near-square body still reads like a cabin, and its hitch occupies the front entrance approach. That art/placement work remains in scope. The camera and collision checks are not substituted for visual approval.
- Scoped Test Genie unit run 20260908-102631-bfd2cf4e completed FAIL after 171 seconds with recurring execution, API timeout and policy findings associated with existing QA knw-1788842511243106138. Terminal JSON and findings are retained; no whole-scenario passing verdict is claimed.

The original nine-stage goal remains active. Next: improve the RV reference with a recognizable vehicle silhouette, sensible hitch/entrance arrangement and matching shared geometry/navigation; review the remaining reference captures; then run the final combined acceptance appropriate to that change. The low-wall/floorplan/cloud cleanup and representative large-world correctness/performance now have completed checkpoints.

## World redesign scoped completion — 2026-09-08

- This entry closes the original nine-stage implementation and the subsequent user review requirements. Earlier checkpoint statements about remaining work describe their historical state. Consolidated evidence and stage mapping: [world-scene-complete/validation.json](../../ui/evidence/world-scene-complete/validation.json).
- RVs now use a compact rear-entry camper silhouette with windows, roof details and instanced tandem wheels. Their central sleeping aisle, staggered rows and hitch opposite the entrance share placement/navigation geometry. All 99 browser doorway assertions pass, including entry/exit at three lateral offsets in both walking modes and actual ceiling selection rays.
- Office rugs, plants and bookcases now use their imported footprint centers and the actual floor support height. Authored furnishings remain visible independently of vegetation density. Decorative furnishings remain nonblocking; architectural solids retain their collision semantics.
- Final visual review reproduced an entirely black first-person tent view despite normal readiness/frame metrics. GPU readback isolated three non-finite color channels at one pixel before bloom, spreading across the entire frame afterward. Two collinear roof triangles failed a new area regression. Removing them restored the same view without disabling postprocessing. Signed squared expressions in the lunar and Milky Way shaders also now use direct multiplication. Failed candidates and GPU diagnostic evidence remain retained.
- The final day/night reference suite passes 56 assertions across 24 images: tent, cabin, RV and office in Explore, first person and third person. Visual review is complete, including walking signs, enclosure/doorway views, smoke, lunar phase crops and the Milky Way. Final navigation passes 58 browser checks, moon 22 and atmosphere 34, with no page errors. Saved-layout dragging, reload and growth pass 33 checks. Six 400-agent scene cases across both scenes and seeds 1/7/99 pass, worst measured walking-frame p95 19.1 ms on this hardware at the medium profile. Those stress/saved captures precede only the final degenerate-triangle removal and shader square cleanup; the reference/navigation/sky captures follow them.
- This checkpoint adds 102 distinct passing focused tests across 11 files. Typecheck, production build, scoped lint, asset registry and whitespace checks pass. Prior checkpoints carry forward camera memory, member conversations/greetings/unread markers, office cutaways and walking ceilings, stepping/jumping, wildlife, saved growth and obsolete scene cleanup evidence.
- Scoped Test Genie unit run `20260908-104331-90902d81` completed FAIL after 179 seconds with recurring CLI execution and API timeout findings associated with existing QA `knw-1788842511243106138`. It began before the final art/roof/shader refinements. Terminal evidence is retained; whole-scenario certification is not claimed. Positive conversation validation uses fixtures, and large-roster validation measures scene behavior rather than live model execution. Fully furnished enterable shelter interiors remain the later extension identified in the original scope.

## Material texture pipeline — 2026-09-08

- Root cause: the Kenney GLB source assets contain UV coordinates and PBR color factors, but no raster images or normal textures. Runtime material handling only cloned emissive materials, while most authored architecture and terrain used untextured standard materials.
- Implemented a shared deterministic texture library in `ui/src/world/scene/materialTextures.ts`. It caches tileable 96×96 sRGB albedo maps, linear roughness maps and restrained linear normal maps for physical surface families. Wood, bark, fabric and stone receive micro-normal relief; large ground, wall, floor, roof, canvas, leaf and metal surfaces use stable albedo/roughness treatment to avoid close-camera artifacts. Material classification is name-aware and explicit authored geometry supplies its surface family.
- Integrated the treatment across instanced and non-instanced props, terrain, indoor slabs, roofs, signs, fires, wheels, wildlife and ambient set dressing. Loader-owned materials are cloned before mutation, preserving source ownership and instancing behavior. Authored roof and terrain geometry now carries UVs for predictable tiling.
- Evidence: 11 focused material/prop/terrain tests pass; typecheck, production build, scoped lint and whitespace checks pass. Park and office scene-design, navigation and atmosphere browser suites all pass with no page or GPU errors. Hardware captures show subtle surface breakup while retaining authored palettes.
- The existing medium world-smoke golden and draw-call thresholds still fail because the intended appearance changed and the pre-existing baseline reports 83 draws against a 74-draw budget. Repository-wide lint also retains unrelated pre-existing errors. No external raster asset was added; custom shader effects and overlays remain intentionally procedural.

## Sky-coloured world materials — 2026-09-08

- Reproduced after the texture rollout. The bundled tree materials are named `leafsGreen` and `leafsDark`, but their GLB base colour factors are cyan. Runtime inspection showed the sky environment was present but reflection changes were negligible; the sky-like appearance came from those source colours rather than a failed texture load.
- `applyWorldTexture` now shifts leaf material hue into a natural green range while retaining per-asset lightness variation. Generated world textures, architectural glass and slime bodies now set environment reflection to zero, preventing the HDR sky from being sampled as a surface colour; highlights come from the keyed lights and authored material channels.
- Added regressions for vegetation hue correction and reflection intensity. Focused material/prop/terrain tests now pass 12/12. Park and office scene-design browser journeys pass with no page or GPU errors; captures in `/tmp/world-textures-skyfix` and `/tmp/world-textures-office-skyfix` show green foliage and stable architectural colours.

## Mirror-like roughness on affected drivers — 2026-09-08

- The remaining report (shiny ground and signs that reflected only the sky) is consistent with unsupported single-channel roughness textures. The previous `RedFormat` map is valid on our WebGL2 test GPU, but can sample as zero on WebGL1 or some mobile drivers; zero roughness produces mirror-like environment response.
- Roughness maps now use portable RGBA storage with all colour channels carrying the same value. Ground, dirt, wood, wall, floor, path and roof ranges were raised to matte values; metal remains the only lower-roughness family. Environment reflection remains disabled for world textures, glass and slime bodies.
- The focused material/prop/terrain suite remains 13/13 passing, typecheck and scoped lint pass, and the restarted park browser journey passes with no page or GPU errors. This specifically covers the driver-independent texture format path that the earlier desktop capture could not exercise.
