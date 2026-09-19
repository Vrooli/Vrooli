---
date: 2026-09-07
scenario: prompt-manager
scope: UX performance — initial load, world presentation, graph/editor payload, mobile adaptation, and performance-gate readiness
status: partially-implemented
---

# UX performance audit

## Executive result

Prompt Manager is functionally healthy and has several good performance safeguards,
but the primary `/world` experience is carrying too much work before the first
useful 3D presentation, and the scenario cannot currently produce a trustworthy
component-level performance trace through `performance-health`.

The highest-value work is to restore the performance capture contract, split the
world/graph/editor payloads, and measure a staged world presentation budget.

## Implementation update

This audit's safe, scenario-owned recommendations are now partially implemented:

- Added `.vrooli/lighthouse.json` for `/world` and `/graph` with ratcheting
  performance, accessibility, best-practices, and SEO thresholds.
- Added lazy boundaries for the world renderer, graph projections/tools, and
  skill content editor, with visible loading states.
- Changed Vite production builds to omit source maps while retaining them for
  `profile` builds.
- Updated the editor regression tests for deferred surface loading.

Validation: the focused editor suite passes 21/21 tests, TypeScript passes, and
the production build passes. The measured artifact fell from 61,594,616 bytes to
18,734,229 bytes. Lighthouse now runs and reports accessibility 0.96/1.00,
best-practices 1.00/1.00, SEO 0.91/0.91, but performance 0.44 on `/world` and
0.31 on `/graph`; the performance phase therefore remains correctly failing on
`PERF_LIGHTHOUSE_BELOW_ERROR_THRESHOLD`. The profile audit remains blocked by
the platform-owned missing instrumentation markers.

## Evidence

### Managed performance tooling

- `performance-health readiness validate prompt-manager` reports Tier 1, with zero
  auto-fixable findings.
- `performance-health audit run prompt-manager` failed before capture because the
  served profile bundle did not contain `injectProfilingHooks` or
  `markComponentRenderStarted`.
- `performance-health benchmark run prompt-manager` initially measured Go build
  14,603 ms, UI build 26,081 ms, and a 61,594,616-byte bundle artifact. After
  the implementation pass it measured Go build 1,734 ms, UI build 24,759 ms,
  and an 18,734,229-byte artifact.
- `performance-health budget check prompt-manager` passed; no declared budget was
  breached.
- Before this pass, `vrooli scenario test prompt-manager --phases performance`
  failed on `PERF_LIGHTHOUSE_ACCESSIBILITY_CONFIG` because
  `.vrooli/lighthouse.json` was missing. After adding the configuration, the
  phase runs and fails on the actual world performance score
  (`PERF_LIGHTHOUSE_BELOW_ERROR_THRESHOLD`, 0.47 versus the 0.75 error gate).

### Production browser observations

Measurements used the managed runtime at UI port 21235 and a production bundle.
The direct browser checks are supplemental because the profile capture failed.

- A cold `/world` load fetched approximately 1.53 MB of JS/CSS transfer before
  world assets, including the 917 KB entry, 378 KB Three.js chunk, and 160 KB
  Mermaid chunk.
- The `/world` load also fetched a 1.44 MB HDR asset and the world prop assets.
- With a synthetic 25-actor GPU-backed world at the high profile, first ready/
  presented time was about 4.42 s, frame p95 was 16.9 ms, and GPU p95 was
  13.35 ms. The scene rendered 88 draw calls and 118,683 triangles.
- At a 390×844 viewport, the world correctly selected its intentional 2D mode,
  avoided horizontal overflow (`scrollWidth == clientWidth`), and rendered no
  WebGL canvas. The headless check showed 24 px of vertical overflow
  (`scrollHeight == 868`), which should be checked on physical mobile browsers.

## Findings

### P1 — Component-level performance auditing is blocked

`ui/vite.config.ts` declares the expected profile alias and `ui/src/lib/profiler.ts`
emits React measures, and readiness therefore reports Tier 1. However, the live
`performance-health` capture rejects the served profile bundle as uninstrumented.
This prevents located React commit findings and makes future regressions harder to
attribute.

**Recommendation:** reconcile the React 19 profile-marker contract between
`performance-health` and this scenario. Verify the served profile artifact after
restart, then capture `/world`, `/graph`, a populated skill list, and editor typing
as separate baselines.

### P1 — The world route pays for unrelated heavy surfaces

The editor panel previously statically imported `WorldView`, `GraphView`, graph
settings/query/help surfaces, and the editor. The implementation now defers
those surfaces behind `Suspense` boundaries, with graph and world modules
appearing as separate chunks. Some editor payload remains shared because other
editor tabs statically consume `SkillContentEditor`; that is a follow-up seam
for route-level editor decomposition.

**Recommendation:** retain the new surface-level lazy boundaries and next split
the remaining shared editor consumers so Mermaid/Monaco language payloads load
only when the relevant editor mode is opened. Preserve the current relative
asset base and chunk-reload guard.

### P1 — First useful world presentation is too slow for a primary landing surface

The GPU-backed 25-actor measurement reached its first ready/presented state at
about 4.42 s. The critical path includes world preparation, terrain preparation,
prop loading, and the 1.44 MB HDR asset. During that interval the user sees a
preparation state rather than the primary workspace.

**Recommendation:** define and enforce a first-present budget; show the 2D roster
or a lightweight low-quality scene immediately, then upgrade to the selected
quality profile and HDR/background assets after the first useful frame. Keep the
existing explicit preparation/error/retry states visible so the optimization does
not hide failures.

### P2 — High-quality rendering leaves little frame-time headroom

The high profile measured 13.35 ms GPU p95 and 16.9 ms frame p95 against a 16.67
ms 60-FPS frame interval. The profile enables AO, bloom, 4× MSAA, 2,048px shadows,
clouds, and higher geometry/vegetation budgets (`world.tuning.json:764-783`).

**Recommendation:** keep the shipped default at medium, make auto quality the
clearest first-run choice where appropriate, and set a measured GPU/frame budget
for representative low-end hardware. Tune postprocessing and shadow refresh before
reducing world fidelity; the current quality governor provides a good seam for this.

### P2 — Build artifact and source-map footprint is large

The baseline benchmark reported a 61.6 MB UI artifact and a 26.1 s UI build.
After omitting production source maps, the artifact is 18.7 MB and the UI build
is 24.8 s. The remaining artifact is still dominated by language/diagram assets;
these are not all initial user payload, but they increase build, deployment,
storage, and cache costs.

**Recommendation:** retain the production source-map change and continue splitting
Mermaid, Monaco languages, diagram engines, and world assets by use. Track
compressed initial JS and route-specific transfer, not only total `dist` bytes.

### P2 — Performance phase is not a complete UX gate

The reviewed Lighthouse configuration now exercises the world and graph routes.
The scenario's budget check passes, but the owned Performance phase fails on the
real route performance threshold, so the UX gate is now actionable rather than
configuration-blocked.

**Recommendation:** keep the reviewed `.vrooli/lighthouse.json` and improve the
world and graph paths until their measured performance scores clear the current
error thresholds. Keep Lighthouse findings separate from component commit budgets
so either failure is actionable.

### P3 — Mobile adaptation is good, with a small overflow follow-up

The narrow viewport intentionally resolves to 2D (`world/engine/webgl.ts:46-50`),
which avoids WebGL cost and preserves the world actions. No horizontal overflow was
observed. The follow-up reproduced on the desktop shell as a 24 px document overflow,
so the shell/root are now explicitly constrained to `100dvh` with page overflow hidden.
The managed browser regression passes at 1280×800; physical mobile verification with
an open keyboard remains useful because safe-area behavior is device-specific.

## Follow-up UX regressions — 2026-09-07

Three post-audit regressions were reproduced against the managed production bundle:

- The document measured 824px against an 800px viewport while the shell itself measured
  800px. The root auto-height was the source of the page-level scroll.
- Agent and team rows received `pointerdown` but not a usable click through the shared
  virtualized collection gesture wrapper. The route callbacks were present and
  programmatic clicks navigated. The fix uses native button rows, stops the wrapper's
  pointer capture, and removes virtualization from the small sidebar collections.
- Runs failed before rendering because the UI sent `profile_key=prompt-manager-heartbeat`.
  The prompt-manager API returned 500 while agent-manager tried to resolve that
  undeclared alias. The declared `prompt-manager/heartbeat-judgment` key returns 200
  and is now the UI default.

The browser regression is available as `pnpm run test:ux` from `ui/` and covers all
three paths.

## Existing strengths to preserve

- Sidebar resizing already uses direct CSS mutation inside `requestAnimationFrame`
  and commits React state at drag end (`hooks/useResizableSidebar.ts:123-147`).
- Graph rendering memoizes derived nodes/edges, uses `onlyRenderVisibleElements`,
  and exposes explicit loading, error, and empty states.
- The world has explicit preparation, retry, WebGL failure, quality, and 2D
  fallback states rather than silently failing.

## Suggested order of work

1. Repair the profile-bundle verification contract so component-level capture is
   trustworthy.
2. Capture real populated-world, graph, list, resize, and editor-typing baselines.
3. Split the remaining shared editor consumers and defer non-primary assets.
4. Reduce time-to-first-world-presentation, then ratchet world frame/GPU budgets.
5. Verify the mobile vertical overflow on physical devices and add a regression
   check if it reproduces.


## Camera movement and animated 3D follow-up — 2026-09-07

This follow-up audits camera navigation, actor rendering, vegetation, labels,
weather, animation scheduling, and measurement validity. It does not implement
rendering changes. Source inspection distinguishes confirmed work from unmeasured
performance impact. Existing September 4–7 captures are historical context, not
fresh movement benchmarks.

### Measurement failure reproduced

`performance-health audit run prompt-manager --json` returned
`AUDIT_OUTCOME_FAILED`, Tier 1, with missing `injectProfilingHooks` or
`markComponentRenderStarted`. The returned CLI process status was zero; the
structured outcome is the verdict.

The failed audit left a profile artifact served. The normal-mode camera smoke
then waited 90 seconds without a presented frame; a medium capture also failed
at 15 seconds. Both observed `ReferenceError: c is not defined` in a blob worker.
WebGL detection passed, and the simulation had 25 actors with no invariant
violations. Renderer counters remained zero, so they are not performance passes.

The served `three-Jqm_aYd0.js` contained the serialized worker bootstrap `eQ`,
whose body calls the external name-preservation helper `c`. The profile config
sets `esbuild.keepNames: true` (`ui/vite.config.ts:24`). The worker bootstrap
executes outside the scope containing that helper. This identifies a profile
build incompatibility; it does not establish that the normal build is broken.

The platform restoration error is independently visible in
`scenarios/performance-health/api/internal/capture/capture.go:295`: an error
from `StartProfile` returns without the restoration closure. The lifecycle
restart has already occurred before bundle verification fails.

Evidence: `ui/evidence/ux-performance-20260907/managed-audit.json`,
`profile-camera-failure.json`, and `profile-medium-failure.json`. The complete
initial browser trace is `/tmp/prompt-manager-ux-audit-20260907-camera/trace.zip`.

### Prioritized findings

1. **P1 — Inward zoom performs expensive scene-wide CPU work.**
   `ui/src/world/engine/camera/input.ts:98` updates world matrices, collects
   physical meshes, raycasts them, recomputes instance bounding spheres, and
   sorts hits for each inward wheel/touch dolly query.
   `engine/camera/obstacles.ts:51` updates the entire scene and traverses every
   obstacle instance for each sweep/recovery/bounds query, with determinant and
   inverse-matrix work per box. `WorldCameraControls.update` requests recovery
   even when the eye did not move (`input.ts:207`). Automatic poses can issue
   several such queries while planning (`camera/poseRoute.ts:9`).
   **Recommendation:** maintain a scene-generation-scoped physical-surface and
   obstacle index; cache static transforms and bounds; query nearby boxes before
   exact intersection; use a terrain-specific query or measured acceleration
   structure for zoom. Skip stationary recovery only when obstacle/terrain
   revisions also remain unchanged. Preserve overlap recovery and clearance.
   Measure query p95/max and input-to-present latency before selecting the index.

2. **P1 — High quality has limited GPU headroom; auto quality tolerates a low floor.**
   The earlier same-day capture reported high GPU p95 13.35 ms and frame p95
   16.9 ms. Historical hardware park captures from September 4 reported medium
   GPU p95 6.18 ms versus high 16.38 ms; these are not a controlled current A/B.
   High enables AO, bloom, 4x MSAA, and 2048px shadows. Ultra disables AO and uses
   2x MSAA, so profile order does not mean monotonic cost on every device.
   The governor's 0.72 decline ratio permits approximately 43.2 FPS on a 60-Hz
   display before decline. A default medium scene can promote to high.
   **Recommendation:** benchmark interaction-active quality decisions; target
   frame p95 near 16.7 ms at 60 Hz with GPU headroom (initial proposed GPU target
   10–12 ms). Test AO/MSAA/bloom independently. Prefer a cheap temporary render
   adjustment during interaction over rebuilding terrain on every drag. Use
   hysteresis and distinguish intentional demand-render idle gaps from overload.

3. **P2 — Instancing does not eliminate per-actor CPU and upload cost.**
   `scene/actors/PoseBuffer.tsx:36` evaluates all actor poses per rendered frame.
   `Slimes.tsx:76` rewrites all body matrices and squash values; `Faces.tsx:48`
   rewrites eyes, ears, mouths, and ear colors. Hidden facial detail still writes
   zero-scale matrices. Actor batches use `frustumCulled={false}`. The historical
   high-quality 25-to-1000-actor sweep held draw calls near 80 while triangles
   grew from 117247 to 1495091 and GPU p95 from 15.79 to 17.96 ms.
   **Recommendation:** compact visible actor/detail batches; add distance-based
   body detail; update stable colors only on change; track dirty poses and upload
   changed ranges. Benchmark 25/100/400 actors during movement before prioritizing
   more complex culling. Preserve a shared pose authority for body/face alignment.

4. **P2 — Demand rendering wakes for unrelated UI activity and invisible weather.**
   `engine/frameDriver.tsx:59` listens on `window` for pointer movement, wheel,
   pointerdown, and keydown. Sidebar interaction therefore extends full-world
   rendering for at least the 350-ms configured camera settle interval.
   `world/index.tsx:557` sets weatherActive from rain/snow regardless of particle
   count, while the low profile has zero weather particles. That combination
   keeps requesting frames even when precipitation itself has no visible output.
   **Recommendation:** scope gesture wakes to the canvas and explicit owners;
   derive weather leases from visible effective effects. Test idle, sidebar
   typing, low-quality rain, hidden tabs, and reduced motion. Preserve wildlife
   leases, pending asset completion, and actual camera damping.

5. **P2 — Diagnostics and capture mode change the workload being measured.**
   `engine/diagnostics/Probe.tsx:285` invokes a whole-scene raycast and scene-bound
   calculation every six frames while diagnostics are open (about ten times per
   second at 60 FPS). The regular world smoke sets `diag=1`, `capture=1`,
   `intro=0` (`scripts/world-smoke/run.mjs:214`), forces continuous rendering, and
   enables preserveDrawingBuffer. It measures a settled pose, not sustained input.
   Camera smoke exercises gestures but records snapshots after settling instead
   of interval-scoped frame/input percentiles. `passTimer.ts:72` derives main
   p95 by subtracting separate pass percentiles, which is not main-pass p95.
   **Recommendation:** separate cheap telemetry from expensive scene inspection;
   capture ordinary mode with diagnostics hidden; retain interval-scoped
   orbit/pan/zoom/follow/home and scene-switch samples, frame p95/p99, long frames,
   query timings, and renderer identity. Compute per-frame pass residuals before
   taking percentiles. Keep visual goldens separate from interaction budgets.

6. **P2 — Labels can rebuild text geometry during zoom.**
   `scene/labels/Labels.tsx:73` reconstructs actor/room maps and candidates every
   three frames. Camera distance changes fontSize and triggers `mesh.sync()`
   with a 0.0001 size epsilon. **Recommendation:** retain stable text layout,
   scale the label mesh for screen-size adaptation, cache membership data by
   revision, and invalidate collision layout on camera/actor changes. Measure
   text sync count and main-thread time before treating this as a major hotspot.

### Strengths to preserve

The world already uses instanced bodies and props, a shared pose buffer, bounded
label pools, movement-gated vegetation culling with a nearest-K heap, limited
shadow refresh, worker/cooperative terrain preparation, cached assets, bounded
wildlife pools, explicit animation leases, and intentional 2D mobile fallback.
A rewrite or generic recommendation to “add instancing” would miss the existing
architecture. Focus optimization on measured input paths and remaining repeated
work.


### Audit tooling recovery

Filed the missing profile restoration defect as Scenario QA
`knw-1788815202695022241`. Restoring the default build exposed an undeclared
`@vrooli/ui-selectors/export` import in the existing selector-generation script.
Installed the already-approved local `@vrooli/ui-selectors` package through
Scenario Dependency Analyzer, updating the UI manifest and lockfile.
`deps approved validate prompt-manager` returned Passed=true with existing
warnings for other dependencies. This is a build recovery change, not a 3D
optimization. The source workspace is shared and was already dirty.


### Fresh normal-build camera results

After restoring the default artifact, `camera.mjs --surface-zoom --automatic-poses`
passed 25/25 checks with no browser errors. The journey covered actor focus,
follow plus orbit, wheel reversal, right/middle pan, Park/Office intro and home,
and six inward surface-zoom steps in each scene. It used 25 synthetic actors,
seed 1, high quality, a 1600x1000 viewport, device scale factor 1, and the AMD
Ryzen 9 7950X integrated renderer through ANGLE gl-egl. Capture mode was off;
diagnostics were on. Physical touch and other user hardware were not tested.

- Park inward zoom query samples: 4.0, 5.6, 5.0, 6.2, 6.4, 6.2 ms.
- Office inward zoom query samples: 4.5, 2.7, 2.7, 3.6, 2.7, 3.6 ms.
- The orbit snapshot recorded a 20-box obstacle sweep at 0.2 ms.
- Focus route planning took 0.6 ms; the sampled intro/home plans took 0.1–0.4 ms.
- The orbit-follow screenshot showed 22.6 ms frame p95, 16.05 ms GPU p95,
  51 rendered frames/sec, 66 draws, and 110903 triangles. These are rolling
  diagnostics at the snapshot, not isolated input-window percentiles.

**Interpretation:** the measured inward zoom queries alone take 24–38% of a
16.67-ms frame interval in Park. They run synchronously on input and are a
credible cause of wheel/trackpad jank. CPU time and GPU time overlap and must
not simply be added. These 12 samples are not a p95 distribution. Collision
indexing remains a scaling recommendation: the sampled small-world collision
and route-planning costs do not establish them as current major bottlenecks.

**Additional P2 UX finding:** after orbiting a focused actor, furniture obscures
much of the actor. The focus distance was approximately 2.4 world units and the
functional focus/clearance checks passed. Add actor visibility/readability
acceptance to automatic focus; evaluate an unobstructed viewing angle or
selective occluder fading. Avoid changing the user's manually chosen orbit.
Evidence: `ui/evidence/ux-performance-20260907/normal-orbit-follow.png` and
`normal-camera.json`.

The default artifact rebuild and healthy lifecycle start passed. The profile
worker error did not recur in the normal-build camera journey.


### Fresh default-build renderer comparison

Sequential fresh-browser Park captures used the maintained world smoke tool,
25 synthetic actors, seed 1, day/clear weather, 1600x1000, scale 1, verified AMD
integrated hardware, and 120 settled frames. These use diagnostics and capture
mode, so they are controlled renderer comparisons rather than normal-mode
interaction latency. First-ready values have intro disabled.

| Profile | GPU p95 | Frame p95 | First ready |
| --- | ---: | ---: | ---: |
| Medium | 6.69 ms | 17.0 ms | 2.41 s |
| High | 16.00 ms | 20.0 ms | 2.42 s |
| High, AO disabled only | 8.44 ms | 17.2 ms | 2.38 s |

Both captures had no browser/request errors and passed GPU-time, framing, and
simulation checks. Both overall smoke verdicts FAILED visual golden comparison
(11.71%/12.76% pixel difference against a 1.5% limit); this audit does not approve
new goldens or attribute the differences to a particular change.

**Telemetry defect:** both captures reported zero aggregate draws/triangles
while visible scenes and nonzero per-group draws/triangles were recorded.
Their passing draw/triangle budget checks are therefore invalid evidence.
Inspect reset/read ordering around explicit attribution and frame publication;
record aggregate counters after rendering and add consistency assertions.
The high GPU budget currently allows 21 ms, which can pass while missing the
60-FPS experience target. Ratchet toward experience targets after instrumentation
is reliable, rather than treating the present budget pass as smoothness proof.

Evidence: `ui/evidence/ux-performance-20260907/normal-medium.json` and
`normal-high.json`.


An additional identical high-profile capture with only `--query ao=0` measured
GPU p95 8.44 ms and frame p95 17.2 ms. The roughly 47% GPU-p95 reduction relative
to the high control identifies AO as a strong first optimization target. This
is a single sequential comparison, not a repeated statistical estimate or a
shipped improvement. The explicit visual perturbation also fails the existing
golden (12.29% difference); that result is expected to require visual review,
not an automatic golden update. Evidence: `normal-high-noao.json` in the same
artifact directory.

Recommended execution order: repair profiling cleanup/worker compatibility and
counter validity; establish ordinary-mode movement budgets; tune AO and accelerate
zoom surface queries; then measure actor/label scaling and unnecessary render
wakes. Add focused-actor visibility checks alongside functional camera tests.


## Authorized implementation follow-up

The user authorized the recommendations on 2026-09-07. This is a scoped W3
performance repair, not a whole-scenario readiness certification. The shared
worktree also contains unrelated selector migration and documentation changes.

Implemented behavior:

- Terrain participates in the existing Drei BVH acceleration, using an indirect
  tree that preserves the shared terrain index buffer. Zoom queries reuse buffers
  and restrict later intersections to the nearest physical hit.
- High-quality AO pauses while the camera moves and resumes after 200 ms of
  settling. The pause and motion tolerance are tunable. Bloom, shadows and scene
  generation are retained during this transition.
- Canvas pointer/wheel and navigation keys wake the frame driver. Sidebar pointer
  activity and ordinary typing do not. Weather keeps frames active only when the
  effective profile actually emits particles; hidden documents stop its RAF loop.
- Actor matrix/color uploads compare Float32 values and submit changed ranges.
  Hidden detail transforms are stable. Existing distance-based detail selection
  remains in place. Labels resize with mesh scale rather than repeated glyph
  generation; actor/room lookup maps reuse stable structural identities.
- Explicit focus searches a bounded set of camera angles when rendered furniture
  or physical surfaces obscure the actor. Manual orbit remains under user control.
  This improves the initial view; it does not guarantee an unobstructed view from
  every subsequent user-selected angle.
- Renderer counters are sampled after rendering. Passive diagnostics no longer
  add scene-attribution renders inside a frame's GPU timing. Main-pass p95 comes
  from each frame's residual duration. Smoke budgets reject zero draw/triangle
  counters.
- A bounded diagnostics interval measures actual R3F frame deltas, first canvas
  input to completed-render latency, and zoom-query duration. The first idle gap
  is excluded from frame deltas; its input-to-render latency remains observable.
  These are not input-to-photon/display latency measurements.

### Browser evidence

`node scripts/world-smoke/camera.mjs --surface-zoom --automatic-poses --performance
--evidence-dir <fresh-directory>` passed **43/43 checks**, with no page errors.
Normal mode, no capture bridge; AMD integrated ANGLE renderer, DPR 1,
25 actors, seed 1, high profile. Initial gestures use 1600×1000; later surface
and interaction intervals inherited 1400×900 from the automatic-pose checks.
Interaction intervals use `diag=0`. The benchmark now explicitly resets its
viewport to 1600×1000; the separate final roster runs below use that size.

| Interval | Frame p95 | Input-to-render p95 | Frames over 50 ms |
|---|---:|---:|---:|
| Park orbit | 17.1 ms | 2.5 ms | 0 |
| Park pan | 17.1 ms | 2.3 ms | 0 |
| Park zoom | 17.1 ms | 2.3 ms | 0 |
| Office orbit | 17.1 ms | 2.7 ms | 0 |
| Office pan | 17.1 ms | 2.7 ms | 0 |
| Office zoom | 17.2 ms | 2.5 ms | 0 |

Every interval passed the new 20 ms frame-p95 and 50 ms input-to-render-p95
checks. AO was disabled during motion and restored after each interval. These
are single-run measurements, not a cross-device guarantee or a paired FPS gain.
The baseline audit did not have this interval sampler.

The identical six-step surface approach improved from **4.0–6.4 ms to 0.6–1.3 ms**
in park and **2.7–4.5 ms to 0.8–1.7 ms** in office. The new continuous wheel
interval measured zoom-query p95 of 0.3 ms in park and 0.5 ms in office. Physical
surface clearance and reversal checks still pass.

Evidence: `ui/evidence/ux-performance-20260907/implementation-camera.json`.
Full screenshots and trace: `/tmp/pm-perf-implementation-camera-1`.

### Validation limits

TypeScript checking and 145 focused tests in 17 files pass. The capture Go package
also passes. The full world diagnostic run had 1,063 passing and 13 failing tests;
one changed postprocessing-default expectation was subsequently repaired and
revalidated. Other failures include runtime preparation, two vegetation-spacing
assertions, and existing literal-policy debt. New settings were configured and
new structural/statistical constants have narrow documented allowances.

Scoped Test Genie runs were executed and their terminal results retained:

- Prompt Manager `20260907-213456-7e096cce`, unit/performance: FAIL. Unit Health
  reported UI surface discovery/policy issues, CLI execution failure and API
  timeout. Lighthouse `/world` measured 0.39 against the 0.75 threshold.
- Performance Health `20260907-213457-054001f1`, unit: FAIL after 120 seconds;
  the returned terminal snapshot supplies no detailed presentation.

These failures prevent a scenario-wide passing claim. The interaction gates and
visual golden comparisons remain separate. No golden or performance threshold
was relaxed to obtain a pass.

Reproduce larger roster intervals with `--performance-only --actors 100` or
`--performance-only --actors 400`. Body/detail batch compaction and larger-world
collision indexing remain scaling options to prioritize from those measurements,
not changes justified by the 25-actor collision timings alone.


### Profiling infrastructure

The profile build now retains readable names without esbuild's `keepNames`
closure helper, which broke serialized Troika workers. Only `react-dom/client`
is aliased to the profiling renderer; its internal `react-dom` imports retain
the core entry point. A live profile-build camera journey passed 13/13 checks
with no browser errors and completed world readiness.

Performance Health now inspects same-origin module-preload chunks in addition
to the entry script. React 19.2.8 in this workspace has removed the old injected
profiling-hook names; verification also recognizes the profiling renderer's
`actualDuration` and `treeBaseDuration` fields together with renderer identity.
The normal served bundle was checked and does not contain those duration fields.
Cleanup is registered before profile restart and runs with its own bounded
context even after verification failure or request cancellation. Restoration
errors are returned in the audit outcome, and inherited profile mode is removed.
A live failed verification restored a healthy default build automatically.

Broader validation debt was filed as Scenario QA observation
`knw-1788817651358638169`. Profile browser evidence is retained in
`ui/evidence/ux-performance-20260907/implementation-profile-camera.json`.


Managed capture subsequently succeeded and restored a healthy normal build:
`ui/evidence/ux-performance-20260907/implementation-managed-capture.json`.
The returned tier was initially 0, but analysis proved that **58 React mark events
were present**. Its detector incorrectly searched raw bytes for `⚛`, missing
JSON-escaped `\u269b`. The detector now decodes event names, handles object and
array trace envelopes, and excludes scheduler track labels. Focused regressions
cover escaped marks, absent marks, malformed JSON and scheduler-only traces.
That classification fix was applied after this capture; the original receipt
is retained unchanged.

Managed analysis identifies App (14 commits, 15.3 ms average, 173.8 ms maximum)
and SkillManagerLayout (13 commits, 15.9 ms average, 173.2 ms maximum). It reports
360 ms of long tasks, FCP 424 ms and LCP 844 ms for the default mount journey.
This is instrumentation-mode mount evidence, not a normal-mode 3D movement
measurement. The report is `implementation-managed-analysis.json` beside the
capture receipt. Initial route/component mounting remains a separate improvement
opportunity even though navigation is substantially cheaper.


### Final roster sweep (1600×1000)

These normal-mode runs use `--performance-only`, high quality, seed 1 and no
capture bridge or diagnostics overlay. They run sequentially. In addition to
p95/input limits, the script now gates frame p99 at 34 ms and the fraction of
frames over 50 ms at 1%. Actor counts are requested through the synthetic roster
query parameter; these runs do not exercise live backend agent activity.

| Actors | Checks | Frame p95 range | Input-to-render p95 range | Finding |
|---|---:|---:|---:|---|
| 25 | 23/25 | 17.0–21.5 ms | 2.2–8.0 ms | Office zoom exceeded p95 and p99 gates; four frames exceeded 50 ms, maximum 176.3 ms. |
| 100 | 25/25 | 17.1–17.6 ms | 2.6–4.2 ms | All movement, tail-frame and AO restoration gates pass. |
| 400 | 24/25 | 17.3–20.5 ms | 6.0–9.7 ms | Park orbit exceeded the 20 ms p95 gate by 0.5 ms; no frames exceeded 50 ms. |

The 25-actor failure is retained, not replaced by the earlier successful run.
Background lifecycle work overlapped part of the roster sweep, so contention is
a possible influence, not an established cause. The larger roster still increases
CPU/geometry cost: the sampled 400-actor scenes reported approximately 645k park
and 802k office triangles. Visible-batch compaction/body LOD is therefore the next
scaling candidate; it is not claimed implemented by the changed-range upload fix.

Receipts: `implementation-normal-25.json`, `implementation-normal-100.json`,
`implementation-normal-400.json` in the evidence directory. Repeated measurements
and physical touch/lower-power-device testing are still needed before promising
consistent 60 FPS across machines.


A final 25-actor repeat after lifecycle rebuilds finished passed **25/25** checks:
frame p95 **17.0–17.5 ms**, input-to-render p95 **2.1–2.7 ms**, and zero frames over
50 ms in all six intervals. Office zoom p95 was 17.5 ms and its zoom-query p95
was 0.7 ms. This establishes a passing repeat, not the cause of the earlier
176 ms stall. Both receipts remain available (`implementation-normal-25-repeat.json`).

The final high-profile static GPU smoke reported **90 draws / 118,807 triangles**,
**16.26 ms GPU p95**, **2.34 seconds to readiness**, and zero unattributed draws.
Its draw, triangle, GPU, framing, clearance, readiness and invariant checks pass.
The overall smoke remains FAIL solely because its golden differs by **12.76%**
against a 1.5% threshold—the same rounded difference as the baseline high run.
Static AO cost is deliberately retained after settling. Receipts:
`implementation-static-summary.json` and `implementation-static-high.json`.

The implementation is running in the normal build. Remaining priorities are
repeatability/tail-stall investigation, large-roster visible-batch compaction,
initial mount cost, and the separately recorded validation/golden debt. The
measurements do not justify replacing the camera collision system at 25 actors.
