# Performance — Personal Planner

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario, and what does "fast enough"
  mean concretely?
- What budgets or thresholds apply, and where do they come from?
- What has actually been measured (versus targeted)?
- What performance risks and design constraints must implementation
  respect?

**Status note (read first):** Budgets remain product targets, not certified
release results. Earlier local route/scroll traces did not reproduce the reported
phone lag. A244 has now removed repeated panorama sampling: focused source tests,
native unit checks and one local routed before/after comparison qualify reuse
with unchanged colors and artwork. This does not prove the phone lag is fixed or
certify app-wide latency. See the shared-sampling qualification below.

## Budgets

Product targets from the implementation plan §23.6. These are release
goals, not measured results.

| Surface / operation | Target | Notes |
|---|---|---|
| Today / day API read | p95 < 500 ms | The Today view answers "what can I usefully do now?"; the scene must not gate this. |
| Native command acknowledgment | p95 < 500 ms | A mutation through the one authoritative command surface acknowledges quickly (canonical success may follow for async work). |
| First useful Today (perceived) | < 2 s | First real content is visible within 2 s; **decorative scene artwork must not block it**. |
| Interactive proposal generation | < 2 s for ≤ 200 items over ≤ 28 days | Larger scopes run asynchronously with visible progress rather than blocking interaction. |
| Forecast refresh | visible result < 5 s | While recomputing, the prior forecast is labeled "updating," never shown as current. |
| Reference-scale datasets | 1,000 work items / 10,000 events / 10,000 actuals | The bounded dataset sizes the above budgets are expected to hold at. |

Inherited platform/lifecycle budgets (from the generated template, not
product targets):

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5–10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |

## Shared panorama sampling investigation — October18

The local warm-return diagnostic `24e45d09-49b2-44d7-8e95-9299aef15942`
passed the reference setup and26-node Focus→Goals→Focus sequence. Each route
performed two32×8 panorama canvas reads, including the warm Focus return. The
read calls took21.6ms,103.9ms and21.5ms respectively; Goals recorded one54ms
Long Task. Focus useful-data cues preceded the reads, so the reads cannot be
called the cause of every visible delay. These are single local-browser
observations, not phone measurements or a release-budget verdict. See
[raw measurements](../../../../docs/internal/delivery-reliability-research/sources/A243logs/planner-performance/warm-return/runtime/measurements.json).

A244 adopted successful sample reuse for the two fixed scene URLs, including
in-flight concurrent requests. Incomplete samples remain retryable. Cache values
last for this page/module lifetime while those assets remain unchanged; reload
starts a fresh cache. No preference or artwork change was made.

The corrected diagnostic was run unchanged before and after adoption:
`055bf85c-39c8-4920-94f3-d91ea6a86ae5` and
`67766b4e-4cc3-41a0-8e43-3ec95fe76560`, each2/2 cases and26 probe nodes,
zero failed/refused/skipped. All three route visits changed from two canvas
draw/read pairs to zero. Scene CSS stayed exactly day `rgb(202 221 235)`, night
`rgb(18 20 42)`, aspect3; cleanup restored both canvas methods and the final
screenshot retained normal art. Both routed leases cleared without errors and
recorded76 test-pool requests, zero primary requests or primary/test-root writes.
The captured after-run script path matches served `index-D6xmkXPE.js` SHA
`8f9610b9948db020c335d44f0eba619f217b0b01330ff3870cbf079de148e865`.
CSS and both panorama bytes remained unchanged across adoption.

Focused isolated gate:6/6 passed, zero skipped; old implementation failed the
reuse assertion. The first native unit attempt failed on an unused test variable;
that fixture declaration was removed. Native unit run
`20261018-163325-eef2069b` then passed API/CLI Go, UI coverage and type-check.
UI reported272 passed/one existing conditional parity skip; truncated output does
not independently enumerate each sampler case. Normal restart
`startop-34375120b1ea8be849ca06a5e0a6fce2` was healthy16:35:41Z.
[Independent browser review](../../../../docs/internal/delivery-reliability-research/sources/A244logs/planner-cache-probe/after-v3/independent-review.md),
[unit review](../../../../docs/internal/delivery-reliability-research/sources/A244logs/planner-cache-adoption/unit-v2-independent-review.md).

This is one local headless comparison. Readiness includes visible data plus
stable scene CSS and cannot be compared to the earlier data-only metric. No
physical-device improvement, perceived speedup or performance-budget certification
is claimed. The failed probe revision and first failed unit receipt are retained.

## Earlier route and scroll measurements (A206/A230)

The A206 qualification was one local 390×844 routed Test Genie run, captured
2026-10-18. Run `6f1d4ec7-b5cc-43b8-8342-60b7f74517df` passed the setup case
and the full mobile sequence (2/2 cases, 38/38 measured-sequence nodes, no
skips/refusals). The fixture readback reported 1,000 work items, 10,000
calendar events, 10,000 linked manual actuals, 10,000 allocations, and one goal.
Its fingerprint identifies deterministic authored fixture intent; it is not a
hash of every persisted row. The test-pool lease saw 158 test-pool requests,
zero primary requests, zero primary-root writes, and zero test-root writes.

| Transition / cue | Click-event to useful-data-ready | Long Tasks during route | Notes |
|---|---:|---:|---|
| Plan | 50.1 ms | 0 | Local routed test pool |
| Goals | 77.0 ms | 0 | Local routed test pool |
| Focus | 34.7 ms | 0 | Local routed test pool |
| Review | 71.2 ms | 0 | Waited for the 10,000 linked-reading calibration cue |
| Settings | 23.3 ms | 0 | Visible theme-control shell cue; not a server-data readiness claim |
| Today warm revisit | 22.7 ms | 0 | Followed the five-tab sequence |
| Today cold mount | Not applicable (no click) | 1 × 56 ms | First Contentful Paint 140 ms; first paint 20 ms |

The browser event clock measures click-to-data-ready inside the page. The
separate `sample_window_ms` records continue through BAS settling/assertions and
must not be called user-visible latency. The Today cold-mount measurement is a
single observation. This sequence includes a warm Today revisit, not a warm
repeat of the slowest route; repeatability is therefore unmeasured. There were
no route long tasks after the cold Today mount. These results did not reproduce
the user's across-app lag or identify a repeatable hotspot, so no code change is
selected from them.

This is a local routed-pool result. It does not measure the user's public tunnel,
auth redirects on that path, an installed phone/browser shell, or physical-device
performance. A local fast result cannot rule out those causes. No before/after
optimization comparison or budget certification has occurred. The next useful
evidence is a comparable trace on the affected device/browser or public-origin
path if available; keep serving build, dataset, viewport, route sequence, and
measurement semantics explicit. Select an optimization only if repeated evidence
shows a hotspot, then repeat the same workload and run focused correctness checks.
The optional device/browser clarification remains unanswered and is not a gate
for the captured local result.

Earlier evidence remains historical: the first v1 six-route run
(`876c3e71…`) had disconnected Wait/Evaluate graph nodes and is navigation-only.
The corrected v2 small-fixture run (`4e168c41-5d8c-4436-8400-ec5150ee1664`)
passed 6/6 cases and 18/18 graph nodes; its `networkidle` navigation durations
are not user-perceived latency. The earlier Today→Focus→Today mobile trace
(`c575c7a8-538e-4924-9774-8aadca288155`) also ran on small data. Preserve those
receipts; none substitutes for the newer reference-scale result.

The source-backed Review calibration, Plan forecast, lazy-route-chunk and
service-worker observations remain hypotheses, not established bottlenecks.
The new trace records first-visit route resources and waits for Review's linked
data cue, but this one pass does not establish a repeatable bottleneck or public
network behavior. See the
[A206 report](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/report.md),
[exact run summary](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/targeted-run-repair-20261018-0715/result-summary.json),
and [fixture/probe definition](../../../../docs/internal/delivery-reliability-research/sources/A206logs/reference-fixture-candidate/review.md).

The bounded mobile scroll/frame experiment then completed in Workflow Health run
`51796559-a29b-4713-aa9c-0a9b0c6139bc` (2026-10-18). Both selected cases passed
(2/2 executed, zero failed, skipped, or refused); the P1 interaction case had
40/40 successful steps against the same 1,000-work-item reference fixture. The
case used a 390×844 viewport. Before each phase, the Today day-plan heading and
seeded work action were visible, with 2,162 px of remaining scroll capacity. A
220 px BAS smooth scroll moved the heading above the viewport while the seeded
work action remained visible, which is the expected narrow-phone layout. The
first measurement used Art-free off and the second used it on. Idle callback
interval p95 was 16.7 ms in both phases; intervals sampled during observed
scrolling were p95 16.7 ms and 16.8 ms, respectively (15 movement intervals in
each). Long-task and Long Animation Frame observers were supported and reported
no entries; their bounded buffers dropped no entries. The workflow restored and
verified the initial Art-free setting (`false`).

This is one A/B browser sample, not evidence of a repeatable difference or a
repeatable hotspot. `requestAnimationFrame` intervals describe main-thread
callback cadence, not physical/compositor FPS or touch latency. BAS capture and
observer overhead can perturb timing, and the sample cannot qualify a physical
phone or public-origin path. The workflow assessment passed; 32 static selector
registry findings were advisory warnings, separate from the executed-case
result. Keep these findings distinct from a performance budget pass: no
optimization is selected and no budget is certified. See the [v7 result](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/scroll-frame-experiment/v7-final-result.md),
[sanitized phase observations](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/scroll-frame-experiment/v7-case-observation.json),
and [durable provider receipt](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/scroll-frame-experiment/workflow-provider-wait.receipt-v7.json).

The final mobile screenshot exposed a separate visual defect that the timing
and accessible-name assertions did not qualify: full-width Today timeline cards
showed only the repeated time and duration (`09:40`, `40m`), hiding each work
title. The accessible button name still contains the title, which is not a
substitute for readable card content. Preserve the RAF observations as valid
for callback cadence; do not treat them as evidence that mobile timeline work
is visually identifiable. The reported capture is stored at
`/home/matthalloran8/.vrooli/data/vrooli/browser-automation-studio/recordings/a040c29e-1def-4eb0-94c9-f0a1e087c4de/artifacts/screenshots/00022--p1-mobile-today-scroll-frame-comparison--assert-reference-content-b--ACTION_TYPE_ASSERT.png`.

The A230 follow-up corrected this label presentation. Focused DashboardPage
tests passed (13 passed, zero skipped), and Workflow Health run
`65f1530e-f3b2-490c-a18a-9df7cebfce5c` passed the setup, desktop compact-layout,
and mobile visible-title cases (3/3, zero skipped). The mobile 390×844 capture
shows the seeded work title together with its start time and duration; the
1280×800 desktop capture retains the compact duration with the title hidden.
The run used 101 routed test-pool requests and recorded zero primary requests
or writes and zero test-root writes. The run’s earlier v4 attempt is retained
as an ordering failure because the setup case ran after the assertions; the
corrected owner-generated registry placed setup first. See the
[A230 qualification packet](../../../../docs/internal/delivery-reliability-research/sources/A230logs/review-packet.md)
and [captured mobile image](../../../../docs/internal/delivery-reliability-research/sources/A230logs/browser-candidate/final-run-65f1530e/mobile-title.png).
This is local routed-browser evidence, not public-origin or physical-device
qualification, and it fixes readable labels rather than proving an app-wide
performance improvement.

| Measurement | Value | Source | Date |
|---|---|---|---|
| Today→Focus→Today small-fixture exploratory trace | Focus 37.8 ms; Today 18.5 ms; no Long Tasks | [A206 report](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/report.md), run `c575c7a8-538e-4924-9774-8aadca288155` | 2026-10-18 |
| Six-route small-fixture functional/timing baseline | 6/6 cases, 18/18 nodes; `networkidle` 1.3–2.1 s; FCP 120–228 ms | [A206 v2 results](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-route-baseline-v2/results-v2.md), run `4e168c41-5d8c-4436-8400-ec5150ee1664` | 2026-10-18 |
| Six-route reference-scale local trace | 2/2 cases, 38/38 sequence nodes; 50.1/77.0/34.7/71.2/23.3/22.7 ms for Plan/Goals/Focus/Review/Settings/warm Today; one cold Today 56 ms Long Task | [A206 exact run summary](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/targeted-run-repair-20261018-0715/result-summary.json), run `6f1d4ec7-b5cc-43b8-8342-60b7f74517df` | 2026-10-18 |
| Mobile Today scroll/frame A/B sample | 2/2 cases; 220px scroll each; idle RAF p95 16.7/16.7ms; moving RAF p95 16.7/16.8ms; no observed Long Task/LAF entries | [A206 v7 result](../../../../docs/internal/delivery-reliability-research/sources/A206logs/planner-interaction-mobile/scroll-frame-experiment/v7-final-result.md), run `51796559-a29b-4713-aa9c-0a9b0c6139bc` | 2026-10-18 |

## Next bounded interaction experiment

The local six-tab trace and the single mobile Today scroll/frame sample do not
measure physical-device scrolling, compositor smoothness or input latency. The
sample did not identify a repeatable hotspot. Source shows shared decorative
SVG filters and moving scene layers on the app routes; that remains a hypothesis,
not a proven bottleneck. If further investigation is useful, repeat the same
reference-scale viewport and interaction under a comparable device/browser or
public-origin path, and report automation/capture interference explicitly.

The existing art-free/reduced-scene preference can be compared through supported
UI in isolated browser/test storage. The completed sample verified that the
seeded work action remained visible after the scroll and that the original
preference was restored. The separate A230 check now qualifies readable mobile
titles in its local browser viewport. Neither result is an optimization or a
recommendation to remove artwork. No new numeric
budget or production profiler is introduced. Choose a product optimization only
after repeatable evidence supports it, then compare the identical workload
before/after with focused correctness and visual checks. Physical-device/public-
network evidence remains separate and optional clarification remains pending.

## R27 cross-route latency attribution — 2026-10-19

The authorized diagnostic reused the existing isolated reference-scale setup
and six-route case, with actual-click Event Timing and the existing React
Profiler user-timing measures bounded to each click-to-useful-data window. The
Workflow Health run `7c44b504-5af7-4eb7-94cc-338f7059ce8e` selected and passed
both setup and route cases (2/2, zero failed/skipped/refused). The isolated
fixture fingerprint was
`e0e3bfd6e1d3c4577013fa249929b582a399e70309380a4f0da193b0ba3e9e93`, with
1,000 work items, 10,000 calendar events, 10,000 allocations, 10,000 manual
actuals and one goal. The viewport was 390×844. The test lease recorded 156
test-pool requests, zero primary requests during test mode, zero primary-root
writes and zero test-root writes. Fixture setup and route sequence executions
were `f7672278-a8e9-4270-85d9-d8c0cbec000c` and
`7c44ebcc-6956-4f4e-b18f-d41448369a47`.

| Route transition | Useful-data ready | Observed wrapped-subtree commits / duration sum | Event Timing |
|---|---:|---:|---|
| Plan | 73.0 ms | 4 / 12.8 ms | Unavailable at the 16 ms observation threshold |
| Goals | 63.7 ms | 8 / 18.6 ms | Unavailable at the 16 ms observation threshold |
| Focus | 35.3 ms | 2 / 7.2 ms | Unavailable at the 16 ms observation threshold |
| Review | 81.8 ms | 8 / 19.0 ms | 24 ms duration; 1.3 ms input delay, 0.6 ms processing |
| Settings | 41.7 ms | 2 / 11.8 ms | 16 ms duration; 1.2 ms input delay, 0.8 ms processing |
| Today warm revisit | 25.0 ms | 2 / 7.4 ms | 40 ms duration; 1.3 ms input delay, 1.2 ms processing |

Event Timing was supported and retained three qualifying entries from six
clicks; the other three are unavailable at threshold, not zero. The values
shown are sums of observed wrapped-subtree measures whose intervals can
overlap; they are not exclusive route CPU or whole-route CPU. The isolated
route trace observed one 65 ms Long Task on the initial Today mount; that cold
mount occurred once and has no repeat. The same page reported an active
same-origin `/sw.js` service worker. API/resource, useful-data, Long Task,
fixture and side-effect controls remained in the case; this single run does not
establish a repeatable route hotspot. The case declared profile as its expected
build mode. Actual profile serving was established by the native profile-mode
lifecycle receipt `startop-f469759f9e1a7ff61e46b7a1052feda9` and its
build/serving record; the case label itself is not runtime proof. The profile
build's implementation guidance is 5–15% per-commit overhead; it is not a
measured correction. Compare those measures only with other profile-build
captures. The durable timelines are
`/home/matthalloran8/.vrooli/state/vrooli/workflow-health/run-artifacts/personal-planner/runs/7c44b504-5af7-4eb7-94cc-338f7059ce8e/personal-planner-bas-cases-performance-00-reference-scale-v1-setup.json/timeline.json`
and
`/home/matthalloran8/.vrooli/state/vrooli/workflow-health/run-artifacts/personal-planner/runs/7c44b504-5af7-4eb7-94cc-338f7059ce8e/personal-planner-bas-cases-performance-p1-mobile-shared-route-responsiveness.json/timeline.json`.

Lifecycle restored normal serving after the profile capture: profile restart
`startop-f469759f9e1a7ff61e46b7a1052feda9`, normal-build restart
`startop-b6adf6ddfb2a225e6baa6c65e9ce10c2`, and healthy normal shadow start
`startop-257deb4022156d0baa1f186623fa5eaa`. Both live and shadow were healthy
on the ordinary build afterward. Timeline artifacts are retained under the
Workflow Health run directory: setup SHA256
`fb77df328283df76a782a7cff4f53140c8bf62104a6d55883d608a85c8f8f402` and route
SHA256 `5f8d9c504d7ac5a2c49ae5f86925e0ebd07bf44f33e6902fea9487ab90ab8201`.

### Source-only measurement correction — 2026-10-19

Before the source correction, the independent review inspected case SHA-256
`e6b8a892f5231cec8b5370c631805310255e82a20f11759756c48082d5c317d1`. The
owner reconstructed the pre-correction source SHA-256 as
`c914a995f3034c9fe68d536508fe202435e2f957fdfb0def7cc5816df1429b7e`,
reconstructed by reversing the single post-run profiler-guidance wording
change. The execution artifact does not itself carry a case-file digest. Keep
the old run and its results tied to those pre-correction bytes; no source change
retroactively validates the corrected case.

The corrected candidate stores DOM `event.timeStamp` alongside the capture
handler's `performance.now()`, and correlates an Event Timing entry by target
and nearest input timestamp rather than applying a 1,000 ms handler-relative
cutoff. This preserves correlation when input dispatch is delayed by more than
one second. Handler-to-useful-data time and Event Timing input delay remain
separate. Missing/inert Profiler entries now report an unavailable status and
`null` duration sum; present sums are explicitly overlapping wrapped-subtree
measures, not whole-route CPU. The case says profile is the expected build mode;
the actual run mode is evidenced by the lifecycle receipt above. No fixture,
data, viewport, route order or correctness conditions changed. This is a
no post-correction browser/BAS measurement exists; the measurement logic is separately unit-qualified below.

Corrected case source SHA-256: `610dd345d9d6f1865302a79925cfc59587257dee754bc3d87250f8890c4cefb5`.
The actual finalization expression was then exercised by focused UI tests in
`src/lib/plannerPerformanceCase.test.ts` (candidate SHA-256
`8d2bcf24928761168360c5aff12fa74ca6417a3bf18d2367c14cd42a738fc624`): both
tests passed, including delayed input timestamp matching and unavailable
profiler results. UI type-check passed, and Test Genie unit run
`20261019-053938-fd4d0ed2` passed 1/1 phase with zero skips. The durable gate
summary is `docs/internal/delivery-reliability-research/sources/A248logs/planner-r27-owner-amendment-20261019/vitest-candidate-v4/gates/validation-summary.json`.
This qualifies the measurement logic only. No post-correction BAS/browser trace
or product-performance result exists.

The earlier local headless trace did not identify a repeatable hotspot, so no
product optimization is selected. It does not establish physical-phone or
public-origin behavior, resolve the reported lag, or certify a performance
budget. The next useful owner work is to capture a newly observed, reproducible
route interaction that represents the reported lag and investigate that route's
current readiness/long-task/profile evidence. Do not repeat the 10,000-item
reference-scale run solely to validate this test-only measurement correction,
and do not repeat A244's cache probe without new evidence. Keep E1/Q1
unaccepted and preserve their independent review gates.

### Integrated ordinary-build route diagnostic — 2026-10-19

After the measurement-logic correction, Workflow Health run
`489ed3e4-492e-4d9e-a16e-92890a14dad5` selected the existing reference-scale
setup and six-route responsiveness cases on `personal-planner@shadow`. Both
cases passed (2/2 selected, executed and passed; zero failed, refused or
skipped). The fixture fingerprint remained
`e0e3bfd6e1d3c4577013fa249929b582a399e70309380a4f0da193b0ba3e9e93`, with
1,000 work items, 10,000 calendar events, 10,000 allocations, 10,000 manual
actuals and one goal. The viewport was 390×844. The case source digest was
`610dd345d9d6f1865302a79925cfc59587257dee754bc3d87250f8890c4cefb5`.
The ordinary serving build was
`sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`,
with browser asset `index-D6xmkXPE.js`.

| Route transition | Click to useful data | Long Tasks in click window | Largest observed API fetch (duration / TTFB) | Event Timing |
|---|---:|---:|---:|---|
| Plan | 47.1 ms | 0 | 15.2 / 15.1 ms | Unavailable at the 16 ms threshold |
| Goals | 56.8 ms | 0 | 5.9 / 5.9 ms | Unavailable at the 16 ms threshold |
| Focus | 36.6 ms | 0 | 5.5 / 5.4 ms | Unavailable at the 16 ms threshold |
| Review | 54.1 ms | 0 | 16.9 / 16.4 ms | 32 ms duration; 1.8 ms input delay; 0.5 ms processing |
| Settings | 56.0 ms | 0 | 6.5 / 6.3 ms | 16 ms duration; 1.5 ms input delay; 0.7 ms processing |
| Today warm revisit | 23.7 ms | 0 | 29.1 / 29.0 ms | 32 ms duration; 1.1 ms input delay; 1.0 ms processing |

Event Timing retained three qualifying entries across six clicks; the other
three are unavailable at threshold, not zero. The cold Today mount was one
observation: FCP 200 ms, first paint 40 ms, DOMContentLoaded 71.2 ms, load
72.1 ms, and one 74 ms Long Task. It is not a click-latency sample. The
case expected profile mode, but this run served the ordinary build and emitted
no React Profiler entries. Profile count was zero and the route/total duration
sums were `null` with unavailable status; these are missing observations, not
zero render cost. Do not compare this ordinary-build run numerically with the
earlier profile-build R27 run.

This is one visit per route (Today also had one cold mount and one warm revisit),
not a repeated sample. No route hotspot repeated or was isolated; no optimization
or app-wide speed improvement is claimed. The active same-origin service worker,
API/resource, useful-data, Long Task, fixture and isolation controls remained in
the case. The fresh provider lease recorded 164 test-pool requests, zero
primary-during-test-mode requests, zero primary-root writes and zero test-root
writes; install, heartbeat and clear errors were empty. Setup and route BAS
executions were `fac63180-822b-43fd-b207-c7e232dc8a67` and
`ae145013-2e88-449a-b536-f59560701393`. Timeline SHA-256 values were setup
`897806c169e046f79d77bb5a0031288a56508ed71dffe9aca9c347a4eb74c691` and route
`3d6db406d4eb81253da9c0bf6a1fc19e26d1b413c2d44299659c285d9a534837`, retained
under `/home/matthalloran8/.vrooli/state/vrooli/workflow-health/run-artifacts/personal-planner/runs/489ed3e4-492e-4d9e-a16e-92890a14dad5/`.
The serving instance remained healthy on the ordinary build afterward.

The same-owner Test Genie unit run `20261019-141442-e9ed85b4` also passed
1/1. Its existing isolated calendar regression creates through REST, then
lists/gets/updates through CLI and updates/gets through Connect using the same
generated ID and revision progression. It uses a temporary routed test
database; it is not a browser UI event journey and does not establish that its
ID matches the separate ordinary UI or CLI/API seed receipts. Those earlier
receipts therefore remain separate evidence, with no cross-receipt same-ID
claim. This diagnostic changes no E1/Q1 criterion or acceptance state.

### Repeatability receiver correction attempt — 2026-10-19

The first repeatability attempt, Workflow Health run
`95b1b913-ef5e-4c51-bb70-a8215774db68`, ended with provider state SUCCEEDED but
terminal validation FAILED: setup passed and the route case failed at step 110.
Its terminal execution evidence is retained in
`../../../../docs/internal/delivery-reliability-research/sources/A248logs/planner-r27-owner-amendment-20261019/repeatability-receiving/workflow-root-get.stdout.json`.
The route timeline showed zero captured transitions despite the click path and
ready-state assertions running.

I changed only the route measurement script in
`bas/cases/performance/p1-mobile-shared-route-responsiveness.json` from dynamic
CSS selector matching to exact `data-testid` matching on the clicked navigation
anchor. The revised case SHA-256 is
`ff2a7e79d2176a60f0803eb97c78a6768eeff9c0f42dda8700665beee20effb5`; static
Workflow Health validation passed with 76 catalog advisory selector findings
and zero required findings. The changed-input selected run
`d4c61b06-c832-45f4-a43e-2e3cb22b2292` used the exact healthy
`personal-planner@shadow` target, same parent, and a new idempotency key. The
provider operation completed, but `terminal_result.status=FAILED`: setup
execution `dff9d513-eac8-4abc-99a7-decb537dd0a7` passed; route execution
`4ea4b9c7-54e3-4991-bf1a-0d19d756573f` failed. All 18 warm-transition measure
nodes failed because the in-page observer again had no matching transition and
reported the blank route key; the three cold Today checks and route useful-data
assertions completed. Event Timing was supported and produced three entries,
but no transition mapping was available. React Profiler expected profile mode;
the observed serving build was the ordinary build and had zero entries, so
Profiler results are unavailable/null, not zero render cost. No valid repeated
route timings or hotspot conclusion can be drawn.

For this run the routed lease recorded 450 test-pool requests, zero primary
requests, zero primary-root writes and zero test-root writes; install,
heartbeat and clear errors were empty. Raw start/wait stdout, stderr and exit
codes are retained under
`../../../../docs/internal/delivery-reliability-research/sources/A248logs/planner-r27-owner-amendment-20261019/repeatability-receiving/correction-20261019/`;
route timeline SHA-256 is
`e53578500061a61b549d85fc01910cf5c8cff7f49fe417868ce96e2d6644c0dd`. The
separate earlier wrong-target run `2cd94566-7c58-4101-b6d7-6a1d9e59aff8`
remains effects unknown; no zero-write claim applies to it. No Planner product
optimization was selected. Root has not accepted the repeatability result.

Next diagnosis is confined to the BAS/Workflow Health evaluation path: reconcile
the exact expression and observer instance loaded for the registered case with
the click event target/path and the empty transition state in the retained
route timeline. Any later runtime attempt requires a reviewed changed-input
measurement correction; it must not be represented as a repeatability pass.
E1/Q1 and the four original unmet criteria are unchanged.

### Observer interpolation correction attempt — 2026-10-19

Root feedback `ROOT-20261019-PLANNER-OBSERVER-INTERPOLATION` (native effort
signal 23) was acknowledged on the same Planner owner as signal 24. The live
case matched the approved preimage SHA-256
`ff2a7e79d2176a60f0803eb97c78a6768eeff9c0f42dda8700665beee20effb5`; only its
six reviewed template-token replacements were applied. Current case SHA-256 is
`556f5c4887ae7d110e1584c05ab9a9e1679d6d7ca8652fa739b3a19f88ac9f54`, matching
the prepared candidate. No application source, fixture, route order, viewport,
or correctness assertion changed.

The exact selected cases ran on `personal-planner@shadow`, build
`sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`,
under same parent `58257550-4062-40d8-aa7d-ffa4f01d279f`. Workflow Health run
`2403a740-5ce3-460f-a030-5de10b496087` used idempotency key
`planner-repeatability-signal23-template-fix2-shadow-20261019` and selected
only `00-reference-scale-v1-setup.json` and
`p1-mobile-shared-route-responsiveness.json`. Provider state was SUCCEEDED;
the inner terminal verdict was FAILED. Setup passed (execution
`82c9d33c-d535-497d-b8ec-053e82a10c90`); the responsiveness case failed at
step 110 (execution `82db91f8-b8e2-4d89-8c9f-6898c74fb6b4`) with the exact
error `page.evaluate: Error: 'tab transition '+key+' did not reach its captured
ready state`. Thus the approved concatenation did not resolve the measured
observer/transition failure. No usable repeated route timings, repeatability
pass, hotspot, speed claim, or product optimization follows.

Lease evidence for that correctly targeted run: 453 test-pool requests, zero
primary requests, zero primary-root writes and zero test-root writes; lease
install, heartbeat and clear errors were empty. The raw full terminal JSON,
start/wait stdout, stderr and exit codes are retained under
`../../../../docs/internal/delivery-reliability-research/sources/A248logs/planner-r27-owner-amendment-20261019/repeatability-receiving/observer-interpolation-correction-20261019/`.
The route timeline SHA-256 is
`a68f2190543a7fd655581d5d20d64a518964b95c63a1db4dd6cf9fc0d5b5fa5f`.

An initial admission used `--instance shadow`, which addresses a Workflow
Health instance rather than the Planner target. The provider warned it would
fall back to LIVE; that run `d0b972f1-7449-4d20-b36c-8d0b0b9f51e8` was canceled.
Its terminal record has no execution payload or lease counters, so its effects
remain unknown; no zero-write claim applies to it. The later run above used the
correct direct target `personal-planner@shadow`. Do not repeat either run.

Return condition: a source-level diagnosis of why BAS evaluates the corrected
observer expression as the literal concatenation text and why no transition is
captured. Root released this bounded lane after this failed attempt. Keep
E1/Q1 unaccepted and all four original unmet criteria and the existing reviewer
disposition unchanged.

## Corrected backtick successor repeatability run — 2026-10-19

Root feedback `ROOT-20261019-PLANNER-BACKTICK-SUCCESSOR` (effort signal 25)
was acknowledged by this same Planner owner as signal 26. The live responsiveness
case matched the required preimage SHA-256
`556f5c4887ae7d110e1584c05ab9a9e1679d6d7ca8652fa739b3a19f88ac9f54`. Applied
only the six reviewed successor fragment replacements from candidate v2; the
resulting case hash is
`db53b1584c588b060d8ab9333db5f9e5505c34890eacecdd536ee3176089594b`. No app
code or other case bytes changed. JSON parsing and `git diff --check` passed.

The target `personal-planner@shadow` was healthy on build
`sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`.
Workflow Health run `d5b9983f-f1fd-43fc-a122-361c5bccf037`, idempotency key
`planner-repeatability-signal25-backtick-fix-shadow-20261019`, selected only
`bas/cases/performance/00-reference-scale-v1-setup.json` and
`bas/cases/performance/p1-mobile-shared-route-responsiveness.json`. Provider
state was SUCCEEDED and the inner terminal verdict was PASSED: 2 selected, 2
executed, 2 passed, 0 failed/refused/skipped. The setup execution was
`86a6053f-75ac-4089-937a-2bd2d1ed351c`; responsiveness execution was
`423495c9-b1da-4414-b199-bddd7463059e`.

The three warm click-to-data-ready samples (median, range in ms) were: Plan
24.1 (23.2–45.3), Goals 53.7 (21.8–54.9), Focus 21.7 (19.1–34.2), Review 37.5
(29.8–66.1), Settings 45.2 (39.8–49.4), and Today 32.1 (28.8–55.5). Three
cold Today mounts recorded DOMContentLoaded 66.3/59.3/59.3 ms, load
67.1/60/60 ms, first contentful paint 160/128/128 ms, and maximum long-task
duration 66/0/0 ms. These are repeated samples inside one case run and one
build; they do not establish an app-wide improvement or a cross-build change.

Event Timing was supported at a 16 ms threshold; 3 entries were observed for
Review (48 ms duration, 1.6 ms input delay), Settings (16 ms, 1.2 ms), and warm
Today (48 ms, 1.5 ms). Plan, Goals, and Focus had no threshold-qualified entry,
so their Event Timing values remain unavailable. The run expected profile build
mode, but recorded no React commit measures; the actual mode and subtree sum
remain unavailable/null, not zero. The measure semantics are wrapped-subtree
duration sums, not whole-route CPU.

The test lease recorded 450 test-pool requests, zero primary requests, zero
primary-root writes, and zero test-root writes. The lease was installed and its
install, heartbeat, and clear error strings were empty. Full provider terminal
JSON, start/wait stdout/stderr/exit codes, per-case summaries, shadow status,
setup and responsiveness timelines, and normalized measurements are retained
under
`../../../../docs/internal/delivery-reliability-research/sources/A248logs/planner-r27-owner-amendment-20261019/repeatability-receiving/backtick-successor-20261019/`.
The wait JSON SHA-256 is `3fe02c00b57af28f263147b555f0cab19c5ac37e5827321a9efdcd0b51db4568`;
the responsiveness timeline SHA-256 is
`d6b839d10b4b4e206635b3fba631a5f31abf61cdaf7b90c417ea4ddcc713a4ff`.

This successful case run verifies that the v2 observer expressions execute and
the selected route and cold-mount assertions complete. It is not a comparison
against an equivalent prior instrumented baseline, does not establish a
repeatability improvement across builds, and identifies no optimization
hotspot. Root's bounded lane is returned with this evidence. E1/Q1 remains
unaccepted; the four original unmet criteria, independent NOT ACCEPTED reviewer
disposition, F8 NEEDS_REWORK and missing historical 78-file attribution remain
unchanged.

## Single post-SQL observation after primary-record containment — 2026-10-19

After Root's exact 12-record primary containment and independent native readback,
the same owner ran one current-source reference-scale observation. Workflow
Health run `6b8979ca-becd-4ec5-9931-52aa709d369e` selected only
`00-reference-scale-v1-setup.json` and
`p1-mobile-shared-route-responsiveness.json`. Provider state and inner verdict
were both successful: 2 selected, 2 executed, 2 passed, no refusal or skip. The
case hash remained `db53b1584c588b060d8ab9333db5f9e5505c34890eacecdd536ee3176089594b`;
viewport remained 390×844. Fixture fingerprint matched the prior run at
`e0e3bfd6e1d3c4577013fa249929b582a399e70309380a4f0da193b0ba3e9e93` with
1,000 work items, 10,000 calendar events, 10,000 allocations, 10,000 manual
actuals, and one goal. Useful-data and all-time calibration assertions passed.

The target was healthy `personal-planner@live` on build
`sha256:0bc54676b8febc933e89404a98211998f5bcd57c50e0798c225ef0b38ba81b41`
(API 17604, UI 20003); API health reported the same build identity. Freshness
reported current API/UI artifacts, with deploy report unchanged. The baseline
`d5b9983f-f1fd-43fc-a122-361c5bccf037` used healthy `personal-planner@shadow`
build `sha256:701de714370b0018b4c56dafafc0db303327e66fd97e03688bf8b1092f819049`.
Thus case, fixture, viewport, and instrumentation source matched, but serving
variant and build identity differed. The baseline's actual profiler mode was
unavailable; the current run emitted no Profiler entries. This is not a
like-for-like build comparison.

Current warm click-to-data-ready medians (three samples each) were Plan 24.0 ms,
Goals 23.3 ms, Focus 17.2 ms, Review 38.1 ms, Settings 38.4 ms, and Today
29.4 ms. Against baseline medians 24.1, 53.7, 21.7, 37.5, 45.2, and 32.1 ms,
the observed differences were −0.1, −30.4, −4.5, +0.6, −6.8, and −2.7 ms in
the same route order. The current differences are mixed and cannot be
attributed to the SQL fix, given the build/variant difference, other shared
build changes, the separately owned RCL candidate pipeline, and one run per
build. No meaningful app-wide improvement is demonstrated and no product
optimization is selected.

Three current cold Today mounts recorded DOMContentLoaded 61.3/49.8/47.4 ms,
load 62.2/50.5/48.1 ms, first-contentful-paint 144/116/112 ms, and maximum
Long Task 60/0/0 ms. Warm route maximum Long Task observations were 0 ms. All
reported resource entries were same-origin; counts ranged from 5 to 15 per
transition. These are diagnostic case observations, not settled-paint proof.
Event Timing values and React Profiler totals were not emitted and remain
unavailable, not zero. No physical-device, public-origin, speedup, or budget
claim follows.

The routed lease was installed as the run ID, recorded 452 test-pool requests,
zero primary requests, zero primary-root writes, and zero test-root writes, and
had empty install/heartbeat/clear errors. Current Workflow Health acquires the
lease and forwards both `X-Vrooli-Test-Mode: 1` and
`X-Vrooli-Test-Lease-ID` as BAS browser ExtraHeaders; the fixture setup uses a
browser `fetch`. Raw terminal JSON and empty stderr are retained under
`../../../../docs/internal/delivery-reliability-research/sources/A249logs/planner-primary-record-containment-20261019/r27-postsql-receiving/`.

Admission included an incorrect `parent_run_id` (`58257550-40d8-40d8-aa7d-ffa4f01d279f`)
instead of the owner's actual parent (`58257550-4062-40d8-aa7d-ffa4f01d279f`).
The run was not canceled or replayed; do not claim its native parent link is
qualified. Full measurement data and this lineage limitation are in the
[A249 measurement summary](../../../../docs/internal/delivery-reliability-research/sources/A249logs/planner-primary-record-containment-20261019/r27-postsql-receiving/measurement-summary.json).
The original E1/Q1/F8 criteria and reviewer disposition are unchanged.

## Known Constraints

These are architectural constraints implementation must respect so the
budgets above are achievable and stay achievable.

- **Do not run the long-horizon scheduling/forecast solver on every
  keystroke, timeline tick, hover, or theme change.** Expensive
  recomputation must be **debounced and coalesced**, and superseded runs
  cancelled by revision — a newer input invalidates an in-flight
  computation rather than racing it. The solver is a pure kernel over an
  immutable snapshot; it is invoked deliberately, not reactively on every
  UI event.
- **Scene artwork must never block first content.** The Observatory
  landscape is a replaceable decorative asset; the first useful Today
  view renders without waiting on it, and artwork is bundled/cached
  locally (never a third-party image host on the critical path) and never
  regenerated per visit. An art-free / reduced-scenery setting exists.
- **Bound the expensive shapes:** recurrence expansion is bounded (never
  materialize an unbounded series), history is paginated, and dense
  lists/timelines are virtualized so large workspaces stay interactive at
  reference scale.
- **Async over blocking for large work:** proposals beyond the interactive
  threshold (> 200 items or > 28 days) run as coalescing background jobs
  with visible progress; forecast refresh coalesces; the UI shows honest
  pending/updating/stale states rather than freezing.
- Vite production builds may process thousands of modules and take several
  minutes (inherited template constraint).

## Regression Procedure

Once a baseline is captured, guard it as follows. The steps below apply to
the existing serving surfaces; they do not imply that any measurement has
already passed or failed.

1. Run `make test`.
2. Capture relevant API/UI command timing for the affected surface
   (Today/day read, command ack, proposal, forecast) against a
   reference-scale dataset.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Compare against the budgets in this document; a regression past a
   budget is a defect, not an accepted constraint.
5. Record persistent findings in this document (accepted constraints and
   captured measurements) or [`PROBLEMS.md`](PROBLEMS.md) (unresolved
   debt), depending on which they are.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — coalescing jobs, forecast refresh, proposal apply
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
