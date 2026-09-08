# Known Issues & Technical Debt

Agent-maintained document tracking issues, debt, and cleanup history.

## Last Updated
2026-09-07

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
