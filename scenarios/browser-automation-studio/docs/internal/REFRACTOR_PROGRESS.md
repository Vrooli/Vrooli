# Browser rehabilitation progress

This file owns the current checkpoint and append-only execution history for the
file-based continuous goal. Follow [TESTING.md](TESTING.md); use
[PROBLEMS.md](../PROBLEMS.md) as the only defect register. There is no active plan,
phase progression or external execution-log dependency. Maintain this small
current-state section, then append dated records without erasing prior evidence.

## Current checkpoint — W225 J02 managed replay and input conversion — 2026-09-25 UTC

Continuous goal FB008 remains active. FB014's overall status request is resolved;
FB010 authorizes managed restarts. Tracking remains file-based: no plans or
external journals, and no completion or blocked claim.

Current governed read `prog_c4eeea4b-1692-418a-adf6-4629e7cae1ff` reports
**0 readable / 17 unavailable / 0 out of band**, `product_qualified=false`, on
managed build `sha256:1d493a63b5b896610b5d6fe971967d852e29ee7ebddf0f3074b098110aa0bdad`.
The previous 7/17 reading belonged to `d3711b61…`; its candidate receipts do not
join to the current build after managed restarts. This is an evidence reset, not
17 observed product failures. W223–W225 repair and verify an actual J02 product
path; the 24-journey preservation row and all other outcome rows remain open.
The W225 live owner passed its full API capture, generated/saved workflow, fresh
driver replay, and saved API execution with the final value independently observed
in both new contexts. Focused Go suites pass, the 16-case controlled harness
passes, and `refactor_contract.py` reports 17 outcomes / 24 journeys / zero prep
errors. No broad Test Genie suite ran; the one targeted provider phase in W221
failed receipt validation and is not being repeated until current-build receipts
are restored. Driver and API are healthy with zero sessions/recordings. Next
pursue one score-bearing current-build owner receipt, beginning with capture.

W218 records the operator's repeated pace concern (BAS-FB-032) and corrects the
stale W213 checkpoint label. A fresh read confirms the score is still 7/17 today;
the operator reports seeing 3/17. The ten remaining rows all classify as
`pending_telemetry`; the rehabilitation evidence phase currently credits six
capabilities and the capture workload supplies the seventh. W217 removed one
local complexity finding but did not reduce the domain-wide score. No source
changed and no Test Genie run occurred in W218. Next work must build or repair
one pending row's owner-backed producer and governed sensor; avoid further
score-neutral tidiness cycles. No immediate row was qualified by this audit.
`python3 docs/internal/refactor_contract.py` passes preparation consistency
(17 required outcomes, 24 journeys, zero errors); it does not establish behavioral
qualification.

W219 adds one tagged, direct Chromium owner test for passive input snapshots. It
confirms paused text, replacement, deletion, actual clipboard paste, an emulated
composition commit and stop-time clear against both an independent fixture log
and the recorder's events. This is capture-side coverage only: it does not replay
this exact sequence in a fresh context or qualify native OS IME. No production
source changed. The required governed read still reports 7/17, 10 pending
telemetry, 0 out of band, `product_qualified=false`; no Test Genie phase ran.
Preserve J02 as open.

W199 rechecked the updated CDP sender in real Chromium: 1,000/1,000 driver input
receipts correlated to decoded frame pixels; p50/p95/p99 were 33.65/34.86/36.99
ms. The current managed full path then passed 1,000/1,000 input receipt and
viewer-canvas pixel correlations through the Go relay on this build. Its
p50/p95/p99 were 36.5/38.9/54.6 ms; applied-ack-to-canvas p50/p95 were
19.3/25.7 ms. Applied sequences were monotonic. The owner-created session was
closed, and the driver reported zero sessions afterward. A source-bound summary
and raw Jest log are retained at
`.vrooli/runtime/rehabilitation-evidence/interactive-feedback-live-w199.json`
and `.vrooli/runtime/rehabilitation-evidence/interactive-feedback-live-w199.log`.

W199/W200 establish local input/frame correlation and bounded UI decoding; they
do not qualify remote p95 or sustained motion. W201 adds the owner/sensor and
Go relay byte bound, then finds managed capture produces only about 22 FPS under
the 30 FPS target. Frame age and decode remain low, so the source cadence is the
next target. Aggregate debt reduction remains unmeasured; keep validation on
the capture and evidence paths.

W201 scope record, before implementation: BAS-RF-007 cannot earn a motion row
because the contract names the generic `performance` phase while the BAS-owned
provider exposes only the `rehabilitation-evidence` phase, with no motion
capability, and the setpoint reader only accepts capability standings from that
custom phase. Add the specialized owner to the existing BAS boundary: the
managed fixture test in `playwright-driver/tests/integration`, retained receipt
validation under `api/internal`, the BAS provider descriptor/handler, and the
governed setpoint reader. Checks will be the motion receipt unit tests, the one
five-minute managed cohort, the exact BAS evidence phase, and the required
setpoint read. Because the receipt contract and provider source are hashed, the
current build-bound owners must be refreshed before reporting the final score.
Source inspection also found the Go viewer relay's channel can hold 120 frame
buffers per slow viewer by count alone. Extend the owner boundary to
`api/websocket/hub.go` and its tests, cap queued plus actively written frame
bytes per client, and include a targeted slow-client regression in the motion
receipt; the existing 12 MiB + 4 KiB driver cap is the common frame ceiling.
The first managed motion measurements returned only 22.2 FPS at both 1080x836
and 720x480, while frame age and decode remained low. A targeted call to the
existing driver stream-settings diagnostic then exposed a real JSON contract
failure: fractional `current_fps` (22.28) could not unmarshal into the Go `int`
response. Extend W201 to preserve fractional telemetry through both driver and
API response types, add a driver wire-decoding regression plus handler response
assertion, and bind those files into the motion receipt. This closes an
observability defect needed to distinguish capture cadence from viewer work.

Historical W189 checkpoint evidence. Managed BAS was healthy on
`sha256:175afb776e50c07cf1f7e0894ee7776d7c70b5053bb84b6ffdb279c42bc2d816`.
The latest governed setpoint program
`prog_43ab1611-3595-43ef-b76c-4af374aa7261` reports5/17 in-band: capture,
passive fidelity, profile durability, cancellation/recovery and resource budget;
12/17 remain unavailable and `product_qualified=false`. Its persisted status is
`PROGRAM_STATUS_SUCCEEDED`; the library-run CLI still exits1 after printing the
valid `status:"ok"` envelope because it fails to recognize that envelope.

Capture on this build measured446ms p95 /658.01ms wall p95 over100 samples plus
one warmup. Profile durability passed5 seed checks and2 post-restart checks,
checkpoint at1,830ms and deleted both profiles. Cancellation/recovery passed all
five cases, including managed restart, with the same build identity before and
after. Passive fidelity passed10,000 native clicks/effects, ordered journal IDs
and applied-input receipts, with0 primary-pool writes; crash/reconnect and three
browser semantics owners remain source-current. Resource budget passed62 points
over60,994ms at223,595KiB max idle PSS,0.33% average /1.00% p95 CPU, and
429,555KiB fixture-plus-shell PSS. Windows private memory is not measured.

Exact provider phase `20260924-234946-3c5b1bc9` passed at L1/Verified, clean for
all four owner capabilities. Current build-bound wrappers and raw artifacts are
under ignored `.vrooli/runtime/rehabilitation-evidence/`, excluded from build
identity; the focused control-plane regression confirms runtime evidence writes
do not change that identity. The earlier59,988ms resource sample and first
evidence phase failure remain recorded below as discarded attempts. The profile
owner briefly timed out while10 external Experience Manager/React Component
Library sessions occupied the driver; those callers released naturally and none
were closed. No broad suite ran and no net complexity reduction is claimed.
RF-030 still needs close/reopen and nested frame/tab replay; RF-036 needs managed
cleanup evidence.

W188 was the prior checkpoint. Managed BAS is healthy on
`sha256:a74ff8db5ab1a9d5346a721a2df458d83ab0c985e8ea559fb935faa6651a88eb`.
The governed setpoint `prog_6c7153b7-e2b7-455b-9f09-325347c12a51` is now4/17
in-band: capture, passive fidelity, profile durability and cancellation/recovery;
13/17 remain unavailable and `product_qualified=false`. Capture is fresh at423ms
p95 /593.52ms wall p95 over100 samples plus one warmup. One exact provider phase,
`20260924-223631-06f03747`, verified all three owner capabilities L1/clean.

Passive fidelity now has joined current-build evidence: the managed owner passed
10,000 native clicks, independent fixture effects, ordered unique journal IDs and
applied receipts in43.612s, with32,441 isolated-pool requests and0 primary-pool
requests; the recording service process-death/reconnect owner and three Chromium
semantic cases pass against current source digests. Profile durability passed5/5
seed checks with a1,807.87ms checkpoint and two post-restart profile checks plus
cleanup. Cancellation/recovery passed all five owner cases. The session-admission
retry-spacing cap is deployed on this build; a 100-capture serial cohort remains
in-band. Receipts: `evidence/rehabilitation/passive-fidelity-managed-w188-2026-09-24.json`,
`evidence/rehabilitation/passive-fidelity-process-crash-w188-2026-09-24.json`,
`evidence/rehabilitation/passive-fidelity-semantics-w188-2026-09-24.json`,
`evidence/rehabilitation/profile-durability-w188-2026-09-24.json`, and
`evidence/rehabilitation/cancellation-recovery-w188-2026-09-24.json`.

The session-admission client now caps retry spacing at500ms after the initial
250ms check, while preserving the30s bound, cancellation and no retry on
ambiguous creation. Its exact admission test group passes4/4 and the code is
deployed on a74. The new serial capture cohort is423ms p95; a current-build
saturated admission cohort is still needed to measure the specific retry-cap
effect. The W186 experiment demonstrates multi-second delay under full capacity,
but does not individually attribute historical f7 outliers.

Current driver inventory is0 sessions and0 active recordings. One first
capacity attempt encountered four sessions owned by Experience Manager and
React Component Library workflows and was cleaned up without closing them; a
later run started only after driver health was zero. Targeted tests and the exact
provider phase were sufficient; no broad Test Genie suite ran. Evidence plumbing
added code and has no net complexity reduction claim.
RF-030 still needs close/reopen and nested frame/tab replay; RF-036 needs managed
cleanup evidence.
The first W185 cancellation attempt rebuilt the managed candidate from
`d616ef46…` to `f6925a1…`; its old profile seed and capture receipt were discarded
for qualification. On the new candidate the cancellation restart preserved the
identity, profile seed/verify owners passed, and the exact phase passed. The
managed passive-fidelity 10k receipt remains stale on `f7bcb29…`, and its
governed validator/sensor is still absent. Immediate target: instrument session-
start admission and run a controlled saturation discriminator; separately wire
the fresh managed passive-fidelity owner into its governed sensor.
RF-030 still needs close/reopen and nested frame/tab replay; RF-036 needs
managed cleanup evidence. No net complexity reduction is claimed.

W178 closed the raw-input-to-journal portion of RF-030. The managed integration
now drives three real pointer clicks through recording ingress, correlates the
driver action IDs with durable timeline entries, adds the journal-owned logical
page IDs, generates and saves the alternating-tab workflow, then verifies the
three ordered effects in a fresh execution context. UI generation now performs
that same action-ID join before merging actions. The helper owner passes35/35,
UI typecheck and scoped ESLint pass, and the managed replay owner passes1/1.
Receipt: `evidence/rehabilitation/recording-workflow-page-identity-2026-09-24.json`.
The row remains3/17 because this targeted J03/RF-030 journey is not itself the
complete preservation contract or a governed setpoint sensor.

The same integration exposed RF-036: workflow deletion joined a project-root
relative `workflows/...` index path under `ProjectWorkflowsDir` a second time,
reported success despite retaining the source file, and the stale file reappeared
in catalog reads. The path resolver now uses the project root and has a focused
Go regression. This repair is source-only pending deployment; the running
candidate still has the defect. Source/test growth is additive and no net
complexity reduction is claimed.

W179 adds a direct passive-fidelity owner check: real Chromium retained10,000
unique ordered fixture actions through one rejected journal delivery, retrying
the same event ID, and a second owner proves a rejected event survives page
reload and is acknowledged with that same ID. The 10k cohort took11.637s; reload
recovery took1.286s. The producer was paced by acknowledgements; an unpaced
1,000-click probe reached the documented100-event pending limit and failed
closed at101/1,000. A focused Go owner also passes failed-commit isolation and
disk-backed journal reopening through fresh service instances (1,001 prior
actions remain paginated in order). This remains partial evidence: neither
driver-process death nor an actual API-process restart was exercised, and the
row still lacks its qualification producer, retained governed receipt and sensor.

W180 adds a stronger SQLite journal owner:10,000 independently identified
actions, one mid-cohort rejected commit retried with the same ID, full pagination
and ID checks before and after database close/reopen through fresh service
instances, then concurrent writers. The focused test passes in3.107s. This is
still not a joint Chromium-to-SQLite workload.

W181 extends the 10k API cohort with an actual recording-service subprocess
death. A child service commits an observation and signals before the simulated
client acknowledgement; the test kills the child, reopens the same journal in a
new service process, retries the same ID, and confirms the total remains10,001
with the original10,000 expected IDs and sequence order intact. Targeted test
passes1/1 in2.126s. This closes the service-process crash/reconnect boundary for
the API fixture, but not a managed BAS API/driver restart with live Chromium
events. The journal reopen and process-death owners pass together2/2 in5.019s.

W182 replaced synthetic DOM event dispatch in the Chromium 10k owner with native
Playwright mouse input and an independent fixture listener. The fixture observed
exactly10,000 clicks and the pipeline callback received10,000 unique ordered
event IDs including one same-ID retry; the native 10k case took11.906s. The
page-reload owner also passes. These browser events still terminate at an
in-process callback rather than the durable API journal.

W183 adds a focused managed owner that sends10,000 native Playwright pointer
clicks through the live BAS API and reads the resulting durable timeline. An
independent fixture counter, unique journal IDs, strictly increasing journal
sequences and monotonic applied-input receipts all report10,000; the managed
build identity remained unchanged throughout. The targeted Jest owner passes
1/1 in45.221s. Receipt: `evidence/rehabilitation/passive-fidelity-managed-10k-2026-09-24.json`.
This joins Chromium input to SQLite-backed API history on the managed path.
The cohort itself has no injected fault; W179-W181 cover rejection, page reload
and recording-service subprocess death in separate owners. Driver process loss
and full supported event semantics remain open. The API has no public route to
delete the closed synthetic test timeline.

W183 ran no Test Genie phase. Required setpoint read
`prog_4bb60c81-3206-4e3e-a118-96b5de836eff` remains3/17 in-band,
14 unavailable, `product_qualified=false`; all14 unavailable rows report
`pending_telemetry`. No score growth is claimed. Interactive feedback remains
unqualified because only the local cohort exists and the declared 50ms RTT /
10Mbps remote cohort cannot be measured on the inaccessible target. Next: connect
the managed owner receipt to a current-build validator and governed passive-
fidelity sensor, then inject driver loss through the managed owner and cover
the supported event-semantics corpus. RF-030 still needs close/reopen
and nested frame/tab replay; RF-036 needs managed cleanup evidence.

W184 response to the stalled-score feedback: repaired the managed owner's
temporary routed-storage setup, propagating test mode through action, page and
frame callbacks; fixed omitted-zero counter handling and matched production
SQLite WAL tuning. The focused managed 10k owner passes in50.158s, down from
the failed attempt's349.6s, and its retained receipt proves10,000 fixture
effects, journal IDs and applied receipts with0 primary-pool requests. TypeScript
typecheck, the focused live-capture Go test and contract validation pass. The
setpoint now reads0/17 in-band because the rebuilt candidate invalidated earlier
profile/cancellation receipts; a refreshed capture is measurable but out of
band at12,163ms p95 vs2,000ms, with tail spikes up to30,297ms. The one exact
Test Genie provider phase failed before running tests because both owner receipts
were stale. The score remains the problem, and the next work is a targeted
capture-tail/session-growth diagnosis, then current-build receipt refresh and a
governed passive-fidelity sensor. This is useful evidence but not a qualification
improvement.

### BAS-WORK-185 — 2026-09-24 UTC — restore current-candidate qualification and isolate capture tail

The operator repeated the feedback that the goal had remained3/17 and progress
was too slow. W185 first read0/17 on build `d616ef46…`: the capture receipt
named `f7bcb29…`, and the profile/cancellation phase receipts were stale. A
fresh managed100-sample capture workload on `d616ef46…` passed at430ms p95 and
625.79ms wall p95, but the first cancellation restart rebuilt to `f6925a1…`,
so that receipt was not carried forward. The mixed-build profile checks were
also discarded for qualification; cleanup/verification confirmed both seeded
profiles survived and were deleted.

On stable build `sha256:f6925a1e924e7536b928f3bc7836a95298e4217fc2e004b929cff5f039d3e84d`,
profile seed passed5/5 with checkpoint visibility at1,773.76ms, then both
identities survived the managed API/driver restart and both profiles were
deleted. The focused cancellation owner passed all five independent cases; its
managed restart preserved the same build identity. Source/contract-bound
receipts: `evidence/rehabilitation/profile-durability-w185-2026-09-24.json`,
`profile-durability-w185-{seed,verify}-owner.json`, and
`evidence/rehabilitation/cancellation-recovery-2026-09-24T21-28-44-540Z-4b193bae.json`.
The final capture workload `fc118428c992dd6d8694625c97e5ae83` passed100/100
samples plus one warmup at464ms p95 / 691.63ms wall p95 against2,000ms on the
same build. The exact provider phase `20260924-213254-caad34ce` passed L1 for
both profile and cancellation capabilities after the capture receipt timestamp.
Setpoint `prog_c07801f8-c4cf-4e5a-909f-1e64ba75d8b9` therefore returned to
3/17 in-band and14 unavailable, `product_qualified=false`. No new row beyond
the prior three qualified outcomes was added.

The old f7 capture tail remains unexplained. Every long trace delays before the
first workflow frame, while page navigation/readiness and later action steps
remain fast; the executor calls `StartSession` before its first step, and the
client retries explicit capacity responses for up to30s. That points to
session-start admission as the next measurement boundary but does not prove
capacity rejection caused the old outliers. The current f692 serial cohort had
zero driver sessions before/after and stayed in band on a shared host. This
cycle changed no product source and claims no debt or complexity reduction.
Contract validation passes with17 outcomes,24 journeys,zero errors; scoped
inventory completed and reports the pre-existing broad dirty checkout. One
exact provider phase was repeated because the setpoint correctly requires its
completion time to follow capture. W186's controlled saturation discriminator
is complete; setpoint remains3/17. Next: choose a bounded admission policy that
preserves cancellation and avoids duplicate starts, then validate it against the
same timed saturation case. In parallel, make the passive-fidelity owner current
and connect it to a governed sensor.

### BAS-WORK-186 — 2026-09-24 UTC — controlled session-capacity capture delay

The direct trace localized W184's old capture tail to before the first workflow
step. W186 reproduced that exact boundary without changing BAS source or
restarting services. On managed build `sha256:f6925a1e924e7536b928f3bc7836a95298e4217fc2e004b929cff5f039d3e84d`,
the driver admitted10 held recording sessions and returned HTTP429
`RESOURCE_LIMIT` for the next start. A loopback capture-surface run with no held
sessions started its first step62.5ms after timeline start (program wall4,990ms);
under ten held sessions, capture retried the explicit rejections. Releasing one
slot after6,000.5ms allowed the session to initialize at21:45:27.893Z; its first
step started5,800.5ms after the timeline began, while the first action itself
took84ms. Readiness took8,123ms overall, including only84ms navigation and
1,015ms explicit readiness wait. A second saturation trial releasing after2s
showed807.8ms before the first step versus the same62.5ms baseline. All ten
probe leases were closed after each trial; driver health returned to0 sessions.
Receipt: `evidence/rehabilitation/session-admission-capture-w186-2026-09-24.json`.

The client retries explicit capacity 429s with backoff from250ms up to2s and a
30s total budget. Runtime logs for the six-second hold show successive rejected
starts at21:45:22.138Z,22.890Z,23.891Z,25.892Z, then session initialization at
27.893Z. This confirms slot admission itself adds time in the controlled case;
it does not attribute the historical12,163ms p95 cohort because its original
request logs are unavailable and the host was shared. Focused Go admission
regressions pass3/3; one initial incorrect test filter selected no tests and was
replaced by the exact maintained test names. Contract and inventory remain valid;
no code or qualification row changed. The goal stays3/17,14 unavailable. Next:
select a bounded admission behavior with measurable overload feedback and
cancellation, then retest the same controlled discriminator; separately advance
the current-build passive-fidelity evidence and sensor.

W156 current result: RF-020's driver/API/WebSocket ordering and receipt path is
implemented and focused-tested. The new API response carries the driver's
applied sequence; WebSocket clients receive the same receipt after an admitted
event. Driver route coverage is29/29 with typecheck and focused ESLint passing;
four Go packages passed under `-race`, and the WebSocket receipt-ordering
regression passed separately under `-race`. No Test Genie phase or broad suite
ran. The latest governed read remains3/17 in-band,14 unavailable,
`product_qualified=false`; capture is470ms p95 on build
`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`.
Held key/button recovery on cancellation and reconnect remains unqualified.

W159 current result: a new real-Chromium owner diagnostic exercised1000 live
pointer moves through the driver route and correlated each applied receipt ID
with a fixture paint marker. Clock-calibrated local driver-to-paint latency is
p50 31.624ms, p95 32.426ms, p99 32.794ms (max clock uncertainty0.322ms).
The retained diagnostic receipt embeds every sample at
`evidence/rehabilitation/interactive-feedback-driver-2026-09-24.json`. This is
not the full UI/API/streamed-frame or remote cohort and earns no score credit.
No runtime or Test Genie run was used.

W158 current result: live recording input now carries a stable `input_id`.
The per-page driver returns the original receipt for duplicate delivery,
rejects reusing an ID for a different payload, and retains a bounded 256-entry
receipt window. The UI tracks unacknowledged WebSocket input IDs; on disconnect
it replays them over HTTP in insertion order, then sends any held pointer-up.
Targeted driver route tests pass31/31 and UI owner tests pass12/12. Go race
tests pass for driver, WebSocket and live-capture. Contract validation passes
17 outcomes and24 preservation journeys; the static inventory completed but
reports the broad shared dirty worktree. No Test Genie phase or broad suite ran.
The setpoint read remains3/17 in-band and14 unavailable, `product_qualified=false`;
it measures build `sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`,
which predates these local source changes. No score row advanced.

W157 pointer-up recovery remains useful on blur, pointer-up/cancel and socket
loss. W158 closes the client-side HTTP/WebSocket retry race for IDs still in the
driver receipt window; full UI reload, session cancellation and retries after
receipt eviction are not established. No complexity reduction or runtime
deployment is claimed.

W147 scope extension: the governed `vrooli/scenario/status` binding omits the
managed build identity in its typed control-plane conversion, although the
status protobuf already defines that field and the direct CLI status reports
it. The BAS rehabilitation setpoint cannot safely join current capture and
owner receipts without this live identity. The repair therefore extends one
layer into `internal/api/controlplane.go` and its owner test, with a focused
`go test ./internal/api` regression; no broader control-plane behavior is in
scope. The extension exists to make the existing BAS build-identity invariant
measurable through the governed read.

W148 scope extension: `vrooli develop` starts the API only when it is absent or
unhealthy; `vrooli stop` has no API-only target and stopping all would interrupt
59 running scenarios and 21 resources on this host. A narrow project-owned
`develop --restart-api` option is necessary to load and verify W147 without
disrupting unrelated services. It will reuse the control-plane's existing
listener/executable identity checks, stop only `vrooli-api`, and restart it via
the managed project lifecycle. Changes are limited to project option parsing,
`internal/setup` and focused owner tests/docs.

W148 result: the managed root API restart loaded the typed status fix without
stopping any scenario or resource. The Test Genie API also needed a managed
restart to load BAS's current descriptor-backed phase catalog. After refresh,
one exact composite phase verified profile durability and reported the missing
J07 cancellation receipt. The setpoint advanced from1/17 to2/17 in-band. The
first phase-name attempt was rejected before run creation and consumed no Test
Genie execution.

W149 adds passing targeted cancellation/timeout/death/retry checks and binds
their owner tests into the cancellation receipt's source digests. It does not
advance a release row. The running BAS API remains on build
`a2a0938a5c89776b5dbe86f9dd33ab4ef1902ec0b3babfb02d5ec164d2dfb2ea`; the new
validator source is tested locally but not deployed. Stabilize the producer
before another BAS rebuild, then refresh profile/capture evidence if the
managed build identity changes.

- **Overall:** materially improved, not production ready. Latest governed
  qualification is3/17 in-band,14 unavailable and no out-of-band row;
  `product_qualified` remains false. The exact provider read marks profile
  durability and cancellation/recovery L1/Verified and clean; evidence tier is
  degraded because the shared checkout is dirty. Capture is470ms service p95
  and681.692965ms wall p95 on build
  `sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`.
  RF-020 ordering/receipt handling and UI blur/disconnect release now have
  focused source coverage, but key and button recovery across server-side
  cancellation/reconnect remains open. Abrupt death,
  concurrent API ownership, concurrency/soak, full journeys, platform
  qualification, UI and debt remain open.
- **017 passive recorder secret boundary:** actual Chromium previously emitted a
  synthetic password in both `payload.text` and `elementMeta.innerText`. The
  recorder now excludes password/hidden fields and password, one-time-code, or
  payment autocomplete fields from passive values, including value/data
  attributes, while preserving action/selector metadata. The canonical raw-event
  converter independently strips those values before creating timeline entries,
  protecting against stale or alternate injectors. All 13 recording-injection
  integration tests and four converter regressions pass with coverage disabled;
  normal-text controls pass. Coverage-enabled narrow run also passed the 13 tests but
  exits on the existing global driver function-coverage floor (13.79% vs 15%).
  RF-017 stays open: historical data and downstream storage/export/AI attachment
  handling, plus intended secret-use qualification, are unverified. Evidence and
  limits: W120 below.
- **017 API ingress/read boundary:** legacy and typed API recording ingress now
  redacts before durable commit and websocket broadcast. The recording service
  also sanitizes buffered typed entries and detached legacy DB read copies;
  stored legacy values remain unchanged. Focused Go tests pass in driver,
  recording service and handlers. Test Genie unit `20260924-023054-162f2f31`
  failed on the UI coverage command, not these owner packages. W122 added an
  actual managed-Chromium screenshot comparison: synthetic password, OTP,
  credit-card and security-code values produce identical screenshot pixels;
  hidden fields are omitted and sensitive data selectors omit their tokens.
  Historical storage, exports, other payment tokens and intended credential-use
  remain open. Evidence and limits: W121–122 below.
- **006 browser launch ownership:** current source now gives all callers in one
  failed launch wave a single bounded retry, waits for in-flight launches during
  shutdown, closes successful late launches, and rejects callers that would
  receive a browser being torn down. The mapped pool/browser-manager suite passes
  33 tests; the direct module probe passes both retry and shutdown assertions.
  Distinct-key pool retention, live resource counts and soak remain unqualified.
- **019 network request identity:** `NetworkCollector` now keys its bounded
  pending map by the Playwright `Request` object itself. An overlapping two-URL
  probe with colliding `[object Object]` string forms preserves both URLs and
  statuses; the new owner test passes. Redirect, failure, eviction and live
  browser evidence remain open. Combined driver typecheck, scoped ESLint,
  Prettier and 48 focused pool/manager/collector tests pass; ESLint retains two
  pre-existing return-type warnings in test helpers. W123 below links the
  controlled reproduction and validation.
- **005 session admission:** W129 fixed the actual-manager race where distinct
  execution IDs could exceed `maxConcurrent` during asynchronous context
  creation. Reservations count against capacity and release in `finally`;
  unit and real-Chromium regressions pass. W132 adds a focused workflow-owner
  regression proving StopExecution signals cancellation and waits for runner
  cleanup before returning. W133 prevents resuming a step with an uncertain
  browser effect after the last successful checkpoint. W137 adds a direct
  workflow-service-to-leased-driver cancellation test with an independent
  effect counter, live-session count and uncertain outcome. Live-browser
  cancellation timing and managed death/restart recovery remain unqualified.
- **011 profile durability evidence:** W124 corrected the checkpoint timing
  predicate; W130 first proved the W129 repair. W131 added a BAS-owned
  Test Genie provider and a read-only setpoint join. W138 requalified five seed
  checks and two managed-restart checks on current build
  `sha256:3484515d0fcf2f821276c3047e0e5036e47648fa417d08b393d6955a20699f95`;
  the checkpoint appeared at1538ms, alpha/beta identities remained isolated
  across restart and both synthetic profiles were deleted. The exact phase
  passed L1/Verified. Capture passed100+1 at437ms service p95/633.314174ms wall
  p95 against2000ms. Abrupt process death, timing repeatability and non-Linux
  targets remain open. Evidence: W138 below.
- **118 profile binding cancellation:** a deterministic regression exposed a
  second commit-window gap after browser capture: cancellation while waiting to
  reacquire the registry lock still committed the stale snapshot. The service
  now rechecks request context under that lock, immediately before aggregate
  commit. The new regression plus the existing serialization/cancellation and
  four-case handler invalidation matrix pass under the race detector. RF-118 is
  resolved; this does not qualify RF-011 or move the rehabilitation score.
  Evidence: `internal/evidence/rehabilitation/profile-binding-cancel-boundary-2026-09-24.json`.
- **119 screenshot decode admission:** added a focused regression for the
  maintained raster-estimate boundary: the largest admitted pixel count passes;
  one pixel over, huge dimensions and nonpositive dimensions fail before decode.
  The new case and existing weighted-wait/cancellation case pass directly in
  0.048s. This checks the guard arithmetic only; managed heap recovery,
  multi-format hostile input and platform backpressure remain open. No product
  outcome or complexity reduction is claimed. Evidence:
  `internal/evidence/rehabilitation/screenshot-raster-budget-unit-2026-09-24.json`.
- **Prior Test Genie runs:** `20260924-025629-7e6ab613` (`unit,workflow`)
  is terminal `failed` on the UI coverage command; its evidence is stale for
  later edits. The exact `profile-durability` run
  `20260924-072006-49f64968` passed on build
  `sha256:3484515d0fcf2f821276c3047e0e5036e47648fa417d08b393d6955a20699f95`
  and advanced the then-current board to2/17. W142 later deployed build
  `sha256:3ff3a2b336662ada94430c1021952760d6e88b326839faa5ee0ea05310974115`,
  invalidating the prior profile receipt; the current board is1/17. No Test
  Genie execution ran in W142–W145. W141's J07 references remain `planned`
  except the two passing targeted executor cases recorded in W145.
- **Latest focused change:** W145 adds timeout and driver-listener-loss
  regressions through the API executor HTTP/session boundary. Both pass under
  the race detector, and the J07 requirement links the executor suite while
  remaining `planned`. The owner receipt producer and real managed restart/
  retry-overlap evidence are still missing.
- **Next qualification focus under BAS-FB-016:** implement the actual focused
  J07 owner producer; the current cancellation receipt code only validates
  receipts and does not produce one. Join independently observed cancellation,
  timeout, driver-death, API-restart, and retried-start cases into a current-build
  receipt; keep managed process restart and cross-process owner routing unknown
  until their owner evidence exists.
- **116 screenshot storage span:** the writer measures time spent strictly
  inside `StorageInterface.StoreScreenshot` and persists it in screenshot
  artifact metadata. A controlled 30ms storage delay is reported as30.472ms by
  the owner writer test. The managed build now exposes persisted spans: ten
  62,022-byte full-page writes sampled under concurrent executions range105µs–
  3.733ms, with130µs median and3.733ms nearest-rank p95; ten ordinary viewport
  artifacts range115–217µs. The declared capture workload passes100+1 at421ms
  service/637ms wall p95. This is one Linux/FileStorage workload, not a global
  backpressure qualification. Failed storage calls now retain the measured
  span in the failed step outcome while preserving the no-success-receipt rule;
  injected backend errors and a `context.Canceled` result are covered. This adds
  one metadata branch and a regression test, with no complexity-reduction claim.
  RF119 remains open for MinIO, cross-platform and soak coverage. The
  failed-write path is now deployed on
  build `sha256:484089a05a799a8b78a9995fac6cc43c8f23d58d11f646a099580e8ae9ddd006`;
  its declared100+1 capture workload passes at444ms service/617.836227ms wall
  p95. Evidence:
  `internal/evidence/rehabilitation/screenshot-storage-span-2026-09-24.json`.
- **122 app shortcut ownership (RF-122 resolved):** deferred unregistration by ID could erase
  replacement callbacks when `RootLayout` rerendered. A maintained app-hook test
  reproduced the loss with a real `Shift+?` key event; the registry now removes
  only its effect's unique registered wrapper; changed-callback and same-function
  replacement plus unmount behavior pass in app-hook tests. Removed the uncalled
  no-op compatibility hook and duplicate legacy formatter after repository-wide
  caller searches. Runtime code fell63 lines in the two owner modules
  (320+485 →286+456); one identity guard branch and one closure per registration
  were added, and the86-line tests are excluded from runtime counts. This is a
  scoped dead-code reduction, not a broad complexity-reduction claim. Test Genie
  run `20260924-012813-5f34d6da` is terminal-failed at the unchanged85% UI
  coverage floor (31.4% aggregate;79 low-coverage files); canonical helper policy
  passes. RF-014 remains open. Evidence and limits: W118 below.
- **119 execution export coverage:** maintained app-hook tests now prove that
  export stays closed before replay frames exist, blank output directories block
  server calls, and a configured directory submits the sanitized MP4 name and
  request settings. The `execution-viewer` project passes2/2; the new file is
  included in its discovery pattern. Test Genie run
  `20260924-014452-cc414191` still fails only at the pre-existing UI coverage
  floor (31.4%,79 files; no policy drift). RF-014 remains open; evidence and
  limits: W119 below.
- **105 screenshot resource guard:** RF119 now uses process-wide weighted screenshot decode admission; the managed 1280x12800 image survives byte-for-byte, two same-size estimates fit, and single estimates above budget reject before full raster allocation. Managed restart deployed build `sha256:a708c5cc95548c958ab2ae0f42e67dc9e711397eebe357b5aa37e2b98911891b`; post-restart CLI capture passed. RF120 repairs the actual sidecar first-health-probe timeout found in Test Genie. Test Genie unit remains failed (driver one-off stream mismatch passed focused reruns; UI coverage floor still fails). A fresh owner cohort now passes on the deployed build:100 captures +1 warmup,421ms p95/616.014912ms wall p95, with16 other setpoint outcomes pending. API+driver concurrent peak PSS remains unverified. Evidence: `internal/evidence/rehabilitation/screenshot-decode-admission-2026-09-23.json`.
- **106 RF119 recalibration:** The retained actual `FileWriter` probe (ten concurrent1280x12800 PNG outcomes, each in a fresh process) rejects H1: 192MiB/5B-per-pixel peaked277,719KiB PSS. H2 is confirmed: ten calls with a2GiB open admission peaked664,296KiB. A ten-byte estimate peaks149,479KiB while accepting all ten artifacts; full-page16.384MP output still persists exactly. The cost is655ms group completion versus405ms at5B and230ms unrestricted. Added a regression that failed before the policy change and passes now. Writer package/race tests pass. Managed build `sha256:6aa67b6b0db708dc1807d412458ba236e654f2677ff19ed82873662630f41180` is healthy. The two current-build100-capture cohorts passed at445ms and358ms service p95; final governed read `prog_80219996-aa46-444f-8223-402648e118c4` reports1/17 in band,16 pending, product false. Synchronized API+driver PSS sampled203,405KiB peak during the viewport cohort; full-page writer peak, active full-page target and JPEG remain unverified. Evidence: `internal/evidence/rehabilitation/screenshot-decode-budget-recalibration-2026-09-23.json`.
- **107 managed RF119 probe:** `workflows execute-adhoc` exercises full-page capture through the managed API, driver and durable `FileWriter` without persisting a workflow definition. Ten concurrent16.384MP PNG captures completed and produced identical71,139-byte artifacts (SHA256 `7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`). Two larger pages (65.5MP and32.8MP at DPR2) were rejected before raster decode. Synchronized API+driver PSS peaked390,105KiB during the ten-run burst. Beginning about6 seconds after the burst,21 half-second samples over10 seconds ranged316,676–321,564KiB; this flags post-burst retained-memory follow-up against the idle target but does not prove a leak or settled idle state. Evidence: `internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`. JPEG, object-store backpressure, Windows and a longer recovery window remain unverified. Final required setpoint read `prog_e4f34184-9dc8-4176-9a9a-4257f917a31f` completed with capture1/17 in band,16 unavailable, product qualification false. Contract preparation passes (24 journeys,0 errors); inventory digest remains `769b716ce412812059dc7269c5ec99156031f8de00d262c15e348b21f6f8137a`; evidence JSON validates and scoped diff check passes.
- **108 restart and profile preservation:** `make -C scenarios/browser-automation-studio restart` returns healthy with zero required credential gaps. Before/after metadata for profile `305e1254-28fc-4bfb-8774-a46aa0bc0d1d` (`channel-manager-synthetic-p1`) matches, as do SHA256 values for its profile document, keyring witness and lock file. The first post-restart60-second PSS window has transient spikes and ends at296,204KiB for six consecutive reads. A second31-second quiet window ranges186,057–186,389KiB combined API+driver PSS, below300MiB. This is a current Linux idle observation; native platforms and CPU are unmeasured.
- **109 same-process RF119 replay and JPEG gate:** From that settled state, ten managed16.384MP PNG adhoc executions complete in9,182ms; each durable artifact is2560x6400,71,139 bytes and has SHA256 `7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`. Pre-run API+driver PSS is151,149KiB; synchronized20ms peak is414,299KiB. At60 seconds after the burst, API remains256,830KiB and driver91,947KiB, 348,777KiB combined. Across the61 post-run readings, combined PSS is348,154–410,326KiB, above idle target. API owns the retained residency, but live heap and unreleased pages remain indistinguished. Added `TestScreenshotJPEGRasterOverBudgetFailsBeforeFullDecode`, which uses a modified JPEG SOF header to assert budget rejection before full decode/storage. Focused test and complete writer package pass. This is a test-only source change; no runtime policy changed and no debt/complexity reduction is claimed.
- **110–111 RF119 managed heap-limit qualification:** `.vrooli/service.json` now scopes `GOMEMLIMIT=96MiB` to the BAS API. The managed process reports that exact environment after a lifecycle restart, and protected profile identity plus all three file hashes remain unchanged. Settled idle API+driver PSS was106,257–106,559KiB. Ten concurrent managed16.384MP full-page PNG executions completed in8,149.4ms; each stored the same2560x6400,71,139-byte artifact (SHA256 `7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`). Synchronized active PSS peaked363,218KiB (API125,972+driver237,246); the61-second post-burst window ranged205,200–359,843KiB and ended at206,058KiB, below the300MiB idle target. Thus the candidate materially reduced retained API residency and recovered below the idle band, but does not cap active PSS. Maintained capture passed100+1 at441ms service/635.548562ms wall p95 against2s; Test Genie performance `20260923-231621-3aca1543` passed with zero observations. Final governed read `prog_de1eebfd-cd4d-4b7a-8b74-44cb4897b1a2` remains1/17 qualified,16 pending telemetry, product false. Retain the API-scoped limit: this cohort gives no causal regression signal, although direct writer probes show about17% slower group completion at96MiB. Valid large JPEG, Windows/macOS, CPU, storage backpressure and long soak remain unqualified. Evidence: `internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`.
- **112 RF121 resolved:** Managed ten-way JPEG cohort revealed JPEG objects were durably named `.png` despite valid2560x6400 bytes and `image/jpeg` metadata. One MIME-to-extension owner now serves FileStorage, MemoryStorage and MinIOClient; PNG remains `.png`, JPEG uses `.jpg`, GIF uses `.gif`, and unknown MIME preserves the old fallback. Storage and writer package tests pass. Lifecycle restart operation `startop-b528f71642b4b341091f7e1d8071e84f` deployed build `sha256:3e81cd3eecfb96b02422eda46141cfab78209c1a47adbc31627aa2e08e1fb9f9`. Protected profile metadata and all three hashes match before/after. Managed full-page JPEG now stores `.jpg`/`image/jpeg` with the same101,301-byte SHA256 `17b30bf2f974f644a3157225dd491a13362578c7b4d81dc0ba832c29d1ceaaf7`; paired PNG retains `.png` and its original71,139-byte hash. Existing stored object names were not migrated. Evidence: `internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`.
- **Test Genie unit receipt:** run `20260923-233436-bfac32e7` failed after504s with one `TEST_EXECUTION_FAILURE`,82 warnings and zero observations at `pnpm run test:coverage` in `ui`. The owner findings artifact omitted command output; direct reproduction now confirms the exact terminal cause is merged UI coverage below the unchanged85% floor:30.52% statements/lines,33.61% functions,67.26% branches. This is BAS-RF-014 coverage debt, not a runner outage or RF121 regression; do not lower the floor. Focused storage/writer tests pass.
- **Next review:** continue reducing RF-014's UI behavior-coverage gap by
  selecting a consequential owner and preserving the unchanged85% floor.
  Then measure API CPU and storage backpressure using owner workload telemetry
  for managed full-page PNG/JPEG. Keep Windows/macOS and long-soak resource
  behavior explicitly unknown.
- **102 issue reconciliation:** RF058 is resolved in source and current behavior: the embedded recording domain schema is registered in production bootstrap, and a newly routed test pool appends/reads journal rows and preserves them across schema reapplication. RF060's existing screenshot byte/metadata/storage repair also passes its maintained PNG/JPEG fault matrix; both stale issue rows are corrected without source changes. Direct owner tests and contract preparation validation pass. Setpoint `prog_79af08a8-c1cd-45c7-8596-f9628c317313` still reads capture in-band only (1/17), with 16 outcomes unavailable and product qualification false.
- **101 execution recovery:** RF055 now interrupts every persisted RUNNING/PENDING row at startup regardless of age; one-second-old active rows and active rows behind 1,000 completed rows pass focused tests. Managed startup is healthy and logs the new recovery path. Test Genie unit run `20260923-203045-3e5d2e12` failed after509s on unchanged UI `pnpm run test:coverage` with zero observations; the owner phase remains unqualified and was not repeated. Concurrent/duplicate API ownership remains unqualified. Full evidence: `internal/evidence/rehabilitation/stale-recovery-101-2026-09-23.json`.
- **103 screenshot bounds review:** Go's PNG/JPEG decoders allocate raster-sized pixel buffers after `DecodeConfig`. Explicit screenshot actions default to full-page; passive telemetry is viewport-only unless `SCREENSHOT_FULL_PAGE=true`. The current writer benchmark measured36.375ms/op and3.879MB/op for a1280x720 PNG. Existing config declares only encoded-byte limits, not a decoded-memory budget, so no policy pixel cap was added. Corrected the stale config comment from truncation to intact-image omission. Focused integrity test, contract preparation, inventory and governed setpoint read pass; capture remains1/17 and resource/end-to-end impact remain unqualified. Full evidence: `internal/evidence/rehabilitation/screenshot-decode-bounds-2026-09-23.json`.
- **104 screenshot raster expansion:** a valid constant-color1280x12800 PNG (16.384MP) encodes to74,012 bytes, below the4MiB artifact ceiling, but Go's writer decoder allocates65,594,288 bytes (~62.5MiB) in72.5ms. The maintained full-writer benchmark records66,607,544B/op and104.8ms/op. This confirms the byte cap does not bound raster memory and opens RF119. The contract has idle API+driver <=300MiB PSS and one browser+shell <=1GiB PSS targets, but no active capture peak or concurrent decode budget; observed service PSS was non-idle and cannot set the cap. Two direct Playwright page setup attempts timed out before capture (30s and15s); the API/driver remain healthy. Existing execution-level screenshot policy is wired and its focused Go checks pass. Full evidence: `internal/evidence/rehabilitation/screenshot-raster-expansion-2026-09-23.json`.
- **Next review:** derive a safe active/concurrent capture budget and validate a controlled full-page screenshot with the managed browser owner. Any decoder guard must preserve realistic full-page evidence and malformed/truncated rejection; do not pick a pixel cap from idle-only PSS.
- **097 checkpoint verification:** the five-case native profile checkpoint probe passed on `sha256:1f7ea2d...`; managed API/driver restart then restored cookie, localStorage and IndexedDB on `sha256:898284c...`. The identity change is expected: BAS progress, PROBLEMS and evidence docs were edited after the first receipt, and lifecycle hashes authored documents. All344 frozen code paths still match. A fresh post-restart owner capture `c3f7aa3fc435f2e1d3e02e737ec16918` passed100/100+1 warmup at438ms service/624.797691ms wall p95 against2000ms. Setpoint remains1 qualified/16unknown; capture receipt does not close profile-durability. Test Genie `20260923-174338-ad9c918f` failed all four selected phases; UI full assertion suite passes 1190/1190 after fixing canonical render helper, while unchanged 85% coverage floor fails (30.42% statements/lines, 33.57% functions, 67.16% branches). Tidiness remains 1159 findings (104 long/424 complexity/607 duplication/23 coupling/34644 debt). Workflow owner executed 0 cases due Experience Manager readiness contract error; report `knw-1790187409653888938`. Programs phase summary failed despite no reported program findings; treat as unknown. Exact evidence: `internal/evidence/rehabilitation/periodic-checkpoint-097-2026-09-23.json` and `/tmp/bas-tg-*-097.*`.
- **094 conditional repair qualified locally:** healthy managedbuild52b39732df53c8c3bb7c8a246a666ca97a11f837defda13545923389cb5851dd. RF112/115/116 repaired: typed browser/variable conditions, matching-only branch routing, frame admission and retained/exported condition results. Native17572 passes53/53;175driver tests, driver/UItypechecks,10-package and final5-package Go races/APIbuild pass. Original protected profile28445 equal3reads. Initial and second7/53failed receipts retained. Durable evidence: internal/evidence/rehabilitation/conditional-semantics-2026-09-23.json.
- **094 qualification:** PHb1e6f95ba344b94458ee19f1a4c8b2c2 passes100/100+1warmup,427msservice/610.723788mswallp95. Board1qualifiedcapture/16unknown/productfalse. TG20260923-163006-94c734ad terminalFAIL: performancepassesexactPH,tidinessfails1157findings (104long/422complexity/607dup/23coupling/debt34644).332frozenpaths unchanged through terminal. All operations consumed; no pending owner run.
- **094 cost:** affectedruntime12554->12662(+108),GoCC1794->1810(+16),functions296unchanged,max50unchanged; scopedTSCC241->267(+26),functions110->115,max31->32. CumulativeGo+335. Capability/evidence repair does not claim net debt reduction. Existing worst hotspots remain open.
- **095 status authority repair:** RF117 faultmatrix reproduced effects after failed runningwrite and false completion after failedterminalwrite; fixed by one WorkflowService finalization owner and durable running admission. Removed unusedMarkCrash and writer status/read repository methods plus no-op implementations. Scopedruntime-76,GoCC-4,functions-2; cumulativeGo+331. Five-package race76433, finalworkflowrace51810 andAPIbuild31165 pass. Panic propagates afterfailedreceipt andcleanup; compilefailure gets committedfailednotification.
- **095 live verification:** healthyb92db9cfbf434a85d6c9d7f69ed2c29624723c4afaec9a01ba3ea4912ed85953 afterrestart75581. Native67312 passes12/12 maintained recording/savedworkflow/cleanup cases; profile32808 equal3reads.339frozenpaths unchanged. PH1d722c86882de4f527ffe518028892a2 passes100/100+1warmup,499msservice/756.742401mswallp95; board1qualified/16unknown. Initialwallp95+23.9%versus094 needs repeatedquiescentcohort before attribution.
- **095 terminal qualification:** TG20260923-164451-777d257b performancepassed/tidinessfailed1159(104long/424complexity/607dup/23coupling/debt34644). RepeatedPH4011fc450443324647b1e5a1dec1fe89 passes100/100,469msservice/681.528072mswallp95,latestboard83132 still1qualified/16unknown. All095operations consumed;339paths unchanged. Relative slowdown versus094 remains RF016 unresolved; no causal/regression clearance. Mostextra time lies outside individualactions; nextdiscriminator is correlated admission/session/finalization/CLI timing, not anotheridenticalcohort.
- **096/097 profile checkpoint correction:** their managed-restart state check passed, but their `<=5s` timing assertion was invalid: the harness waited 5.5s and only checked that storage state existed. W124 replaces that measurement on current source/build; a bounded 50ms poll saw the automatic checkpoint 1.171s after fixture writes, and alpha/beta cookie/localStorage/IndexedDB survived close/reopen, profile isolation, and managed API/driver restart. All synthetic profiles/sessions were deleted and the managed driver returned to zero sessions/recordings. Profile durability is strongly supported for this Linux cohort, but remains `pending_telemetry` until the governed reader consumes the owner receipt; abrupt process-death behavior remains outside this cohort.

- **Deployed assertion repair 093:** BAS API/UI build
  `553bc332c8acced00a3ff95cdcf1035047b910e8ed918b05fa350a70c9e9373d`;
  API/UI/driver healthy after managed restart3717 (exit0). RF113's eight assertion
  modes now honor negation, case sensitivity and custom mismatch messages while
  preserving evaluator errors. Removed obsolete aliases/regex paths and duplicate
  state/comparison methods. Canonical enum fixtures replace six false-positive tests.
- **093 proof:** 93/93 focused tests, typecheck95071, native81086's 58/58 public
  capture cases and independent branch effects pass. Timeline preserves assertion
  fields. Original protected profile60996 compares equal to rollback in all three
  reads. One original profile, not three profiles.115 frozen paths unchanged at
  native qualification: `/tmp/bas-frozen-owner-093.json`.
- **093 capture:** PH41914 completed, operation
  `93ba0c644d6f91e5cbe53d47de3a35af`;100/100 first attempts plus one warmup,
 447ms service/650.217134ms wall p95, within2000ms. Producer/config/contract/fixture
 identities unchanged. Board63711 completed: capture in band,16 unknown, product
 readiness false. The 7.9% wall increase over092 is one shared-host observation;
 it does not establish a relative regression or improvement. Recheck with repeated
 comparable quiescent trials when attributing performance.
- **093 qualification terminal:** Test Genie `20260923-153329-f6718ff3`
  failed overall: performance passed with the exact current PH receipt; tidiness
  failed with unchanged totals. Admission29795 and sole wait74692 consumed.
  All115 frozen paths unchanged through093 terminal; its operations are consumed.
  Durable evidence: `evidence/rehabilitation/assertion-semantics-2026-09-23.json`.
- **Debt:** assertion owner runtime net-269 lines; scoped installed ESLint
  complexity58→44, functions10→4, maximum27→22. This is a narrow reduction;
  domain-wide TS metric RF064 remains unknown. Cumulative scoped Go delta+319,
  including the new090 qualification capability.092 tidiness still fails:
 1157 findings/104 long files/421 complexity/608 duplication/23 coupling and
 34644 duplicated-line debt. New compiler test length and topology complexity
 findings remain open; no threshold changes or fixture relocation.
- **Retained dated checks:**092 graph boundaries/loop postludes and V2 edge labels
  pass focused race/native checks.091 full API/CLI/driver tests and UI typecheck
  passed; UI coverage failed its85% floor (30.42/33.57/67.16/30.42), with existing
  TabBar projection drift,79 low-coverage and two injectable-seam findings.
  These are dated results, not a fresh complete093 suite.
- **Next:** RF112 conditional actions remain unsupported. Read-only tracing confirms
  page-JavaScript/element predicates belong in the driver; variable comparisons
  belong to the Go execution store. Existing typed condition evidence is not wired
  across the driver boundary. Preserve exceptions as failures, actual frame/store
  ownership, exact branch selection and ordinary evidence/checkpoint paths.
  Recall60134 is already retained.094 implementation is in progress; see latest history.

### BAS-WORK-111 — 2026-09-23 UTC — managed GOMEMLIMIT qualification

RF119 / resource-budget / capture. H1 predicted that service-scoped
`GOMEMLIMIT=96MiB` would lower managed API retained residency and recover idle
PSS below300MiB after repeated full-page writes; H2 predicted material harm to
maintained viewport capture. The API run environment in
`.vrooli/service.json` is the configuration owner. This is the necessary BAS
configuration extension; no host or process-binary setting was changed.

After recording the rollback condition, the managed lifecycle restart was healthy.
The protected profile `305e1254-28fc-4bfb-8774-a46aa0bc0d1d` and its metadata,
`.keyring-witness`, `.profiles.lock`, and profile JSON hashes were unchanged.
The live API environment contains `GOMEMLIMIT=96MiB`. Settled idle PSS measured
106,257–106,559KiB combined. Ten concurrent adhoc full-page PNG executions
completed in8,149.4ms; all ten durable artifacts matched at2560x6400,
71,139bytes and SHA256
`7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`.
Twenty-millisecond synchronized API+driver sampling peaked363,218KiB. Over the
next61 seconds combined PSS ranged205,200–359,843KiB and ended at206,058KiB;
API PSS was122,791KiB and driver PSS83,267KiB at that final sample. H1 is
supported for lower API residency and return below the idle target. The active
peak remains above300MiB: Go's soft heap target is not a process PSS cap.

The maintained Performance Health capture workload `b117f8172d618f22f0363832c8767e82`
passed100 measured captures plus one warmup at441ms service and635.548562ms wall
p95 against2,000ms. Test Genie performance `20260923-231621-3aca1543` passed in
61seconds with zero observations. Setpoint read
`prog_2e24c040-a76f-42d4-9805-6b1f6938e5aa` completed1/17 in band,16 unavailable,
product readiness false. The result is within retained capture p95 observations;
one cohort cannot assign causality. Direct isolated writer probes indicate a
roughly17% group-completion cost at96MiB, so retain this only as a measured
tradeoff subject to later comparable cohorts. No code paths/callers/policies
were removed; this resource repair claims no complexity or debt reduction.

The optional direct-probe `runtime.GC`/`debug.FreeOSMemory` comparison remains
measurement-only (149,167KiB peak without forced collection,139,463KiB after GC,
20,575KiB after `FreeOSMemory` in6ms); no forced production collection was added.
A valid large JPEG managed capture, Windows/macOS memory, CPU, storage
backpressure and long-soak behavior remain unknown. Next discriminator: valid
managed large JPEG plus PSS/latency, followed by CPU and backpressure measurements.
Full raw receipt and validation references are in
`internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`.

### BAS-WORK-112 — 2026-09-23 UTC — managed JPEG cohort and RF121 repair

RF119 / resource-budget and evidence-completeness. The valid JPEG discriminator
is the existing screenshot action's `quality=80` path, which uses driver
`captureCompressedScreenshot`, then the production FileWriter and managed
FileStorage. Ten concurrent16.384MP managed JPEG captures completed in a
measurement window of64,089.5ms including the60-second recovery series;
the active capture samples ended at2,762ms. Every stored artifact was a
valid2560x6400 JPEG,101,301bytes,
SHA256 `17b30bf2f974f644a3157225dd491a13362578c7b4d81dc0ba832c29d1ceaaf7`;
all are `image/jpeg`. Active synchronized API+driver PSS peaked252,537KiB from
a157,936KiB baseline; the60-second reading was187,569KiB. This is a separate
cohort, not a direct PNG performance comparison.

The managed oracle found every durable object path ended `.png` even though the
bytes and metadata are JPEG. This confirms new BAS-RF-121, a W3 storage naming
defect. Necessary scope extension: shared screenshot naming across
`api/storage/file.go`, `memory.go`, `minio.go`, with a single format-to-extension
owner in `image_helpers.go`; add tests in storage owners. Checks before repair: Go
storage tests and one managed post-restart JPEG/PNG artifact identity comparison.
No host-owner boundary is crossed; no saved profile or prior artifact is rewritten.
Expected behavior was implemented: JPEG gets `.jpg`; PNG remains `.png`; GIF uses
`.gif`; media type and stored bytes remain unchanged. `go test ./storage -count=1`
and `go test ./automation/execution-writer -count=1` pass. Managed lifecycle
restart `startop-b528f71642b4b341091f7e1d8071e84f` is healthy on build
`sha256:3e81cd3eecfb96b02422eda46141cfab78209c1a47adbc31627aa2e08e1fb9f9`; the
protected profile identity/metadata and three file hashes compare equal.
Post-restart full-page JPEG execution `e1968fbd-287d-4489-bca3-1c4f52d00df7`
stores a valid2560x6400 JPEG as `.jpg`,101,301bytes, same SHA256 as pre-fix.
Paired PNG execution `04151191-2702-41fb-bccf-3171a3f54717` remains `.png`,
71,139bytes, same SHA256. Existing `.png`-named JPEG artifacts were left intact.
No saved workflows/callers were converted or removed; this is a storage metadata
correction and claims no complexity reduction. The full pre/post evidence is in
`internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`.
The final candidate-bound Performance Health capture receipt is operation
`0605ca0361859dce949575726f1e8af1`,100+1 at419ms service/607.570705ms wall
p95; Test Genie performance `20260923-234725-be3fdcdd` passes on the same build.
Final setpoint read `prog_930ee4d1-5c99-403c-bea4-e03bd53881d9` reports capture
1/17 in band,16 pending telemetry, product false. MinIO external integration is
compile-covered but remains unqualified; no live MinIO service was available in
the retained route. Test Genie unit
`20260923-233436-bfac32e7` failed after504seconds: the `ui` workspace command
`pnpm run test:coverage` failed, with1 error,82 warnings and0 observations. A
focused direct run now confirms the terminal cause: merged coverage is below the
unchanged85% floor (30.52% statements/lines,33.61% functions,67.26% branches).
This is BAS-RF-014 coverage debt; no threshold or assertion was weakened. Next,
select a consequential under-covered UI owner and add behavior-driven coverage,
then measure CPU/storage backpressure and retain platform unknowns.

## Record format

Use a stable record ID and timestamp. Keep a record proportional to the work.

```text
Record/date; BAS-RF, journey and outcome IDs:
Question/hypothesis and independent expected behavior:
Source/build identity, concurrent-tree limits and affected owner/paths:
Decision or boundary extension and rationale:
Changes; converted callers; removed policies/code/adapters:
Checks, operation IDs, receipt/artifact refs and interpretation:
Comparable before -> after: performance, complexity, duplication, coupling:
Baseline -> cumulative delta; useful behavior preserved:
Rejected hypotheses and remaining uncertainty:
Unavailable check: cause, attempts, evidence, limitation, next-check trigger:
Pending operations and next useful action/review angle:
Feedback addressed with receipt:
```

For a review, record new cases and adversarial questions, concrete observations,
new BAS-RF IDs and their dispositions. Two clean reviews do not close this goal.
If runtime time is short, update the current checkpoint before starting another
operation. Keep the operation identity when work outlives the current turn.

## History

### BAS-WORK-000 — 2026-09-22 UTC — preparation correction

The operator rejected the previous work shape. The mistakenly created
`bas-browser-rehabilitation` record (948897c4-bc18-4426-b798-f8bd2d25421e) was
archived through its owner; it is historical and must not be resumed. The
supported owner operation is soft archive, so an audit record still exists.
No execution was started. No other historical BAS plan was changed.

The active target, scope, goal, feedback, checkpoints and proof requirements
now live in BAS files. Continuous investigation replaces automatic completion
after two clean reviews. Complexity, duplication, ownership clarity and removal
of obsolete code are first-class outcomes alongside browser correctness and UX.
The operator's unavailable-validation rule is retained. Existing production
source changes and original baseline observations are preserved.

Correction receipts: eight preparation regression tests and the contract
consistency check passed; goal length is 2,044 characters; current handoff links,
local evidence hash and archived plan status were verified. Test Genie
20260922-030931-9015f688 passed skill-set and failed docs. Correcting 19 preparation
manifest entries and registering four artifacts reduced documentation errors
from 42 to 4 in docs rerun 20260922-031112-03ee86df. Remaining errors and reference
warnings are retained under BAS-RF-050, with 389 warnings and 20 infos in that
rerun. This is a failing docs result, not unavailable validation or a product
pass. The direct preparation checks are narrower and passed.

Feedback BAS-FB-001 through BAS-FB-007 is resolved for preparation wording and
tracking with references in OPERATOR_FEEDBACK.md. Product repairs, performance
improvements and cleanup remain implementation work; none is claimed complete.

### BAS-WORK-001 — 2026-09-22 UTC — profile save receipts and recovery (in progress)

W3 localized repair, BAS-RF-003 / J06 / profile-durability. Prior turn was
preparation progress, not a running validation wait; no pending operation was
recorded. Current HEAD is `7d1c7531d057c061c5ad5a67fbecb97ebe2eb194` in a shared
dirty tree. Recall found the retained profile investigation; its current probe
reproduces all seven profile failures (plus unrelated replay failures).

Hypotheses: (1) handler swallowing Save errors explains false persisted/closed
receipts; (2) repository corruption alone explains them. Injected SaveErr reproduces
HTTP 200 and association loss without disk I/O, confirming (1). Repository
faults independently reproduce RF-025/027 and remain follow-up work.

Decision: use the existing aggregate PersistSessionState owner for manual saves
and close; capture both storage and tabs before writing, propagate every failure,
and retain the browser plus association until a successful save and close.
This removes duplicate handler snapshot/save policy. No scope extension or data
format conversion. File-repository crash atomicity and overlapping aggregate
updates are still open; this change must not claim to fix them.

Validation intended: maintained HTTP regressions (save/capture/close failures,
retry, empty tabs, combined snapshot); profile/service regressions; scoped Test
Genie unit/workflow/tidiness; contract, inventory and rehabilitation board.

RF-003 caller review: RecordModeView ignored HTTP errors and navigated away on
network failure, with duplicate close code in workflow completion. Extended the
same repair within BAS to both paths, retained the workspace on failure, admitted
one pending close, and removed forced full-page reload after router navigation.
Five of six UI behavioral tests fail before the repair; the ordinary success
control passes. Initial test setup exposed different router majors in the shared
render helper and BAS; the test uses the helper's supported withoutRouter option
and the BAS router. No dependency change or production adapter was introduced.
Relevant source changed while the broad owner run was active; direct tests will
verify final inputs, and that owner's receipt cannot imply an isolated snapshot.

### BAS-WORK-002 — 2026-09-22 UTC — fail-closed profile recovery (in progress)

Feedback reread; W3 / BAS-RF-027 / J01,J14 / profile-durability. The current
retained probe shows missing protected bytes restored as empty state and a wrong
key producing an empty listing plus a newly created default identity. Hypothesis:
swallowed repository read failures cause the replacement, not default selection
ordering. Fault the real file-backed service with missing/corrupt state and a
wrong key; Get/List/default selection must fail visibly, leave bytes unchanged,
and recover the original identity when the fault clears.

Remove the missing-state success branch and skip-unreadable listing policy.
Do not infer intentional empty state from missing ciphertext: Create always
commits encrypted state, including an empty state. Recovery reports the affected
profile identifier and an error, never authenticated contents. Native/browser
recovery and a richer per-profile metadata recovery UI remain unverified.

BAS-WORK-001/002 completed scoped checks: handler 11-case regression and focused
Go package checks passed, handler race check passed, UI 6-case regression passed,
UI ESLint/typecheck passed, all profile/persistence/session-profile RPC package
tests passed under `-race`. Contract preparation remains valid, not qualified.
Board receipts `prog_d6665b51-2d4d-422c-bba7-ab4c6fdbf941` and
`prog_9d792154-251b-4c9b-956c-abfc733a660f` both read 0/17 qualified outcomes.
Detailed reproducible commands, failure/success output, source hashes, before/after
probes and inventories: [profile boundary evidence](evidence/rehabilitation/profile-boundaries-2026-09-22.json).

Comparable deltas: server snapshot/save implementations 2 → 1, client close
implementations 2 → 1, aggregate commits per snapshot 2 → 1, swallowed capture/save
errors 5 → 0 across the repaired persist/close paths. Handler cyclomatic sum
184 → 170 over the same 23 functions; close 16 → 6; persist helper 10 → 6.
Repository function sum 75 → 73 over 14 functions; missing-state success and
unreadable-list skip policies removed. No new runtime module or import dependency.
Runtime source: API 94,165 → 94,101 lines (-64), UI 131,532 → 131,518 (-14);
Go tests +228 lines, UI test +93 lines. These are substantive local reductions,
not a claim of completed domain-wide rehabilitation. Broader duplication/coupling
remain owed by the active tidiness receipt. Record-mode.go remains >500 lines;
its unrelated handlers are still a cohesion hotspot rather than excused by this
repair. Existing tests and budgets were not weakened.

Retained profile/replay checks: 3/18 → 7/18 expected behaviors; 11 still fail,
none unavailable. The missing-state probe now permits a nil returned profile on
error instead of dereferencing it; its expected behavior is unchanged. Original
baseline artifacts remain immutable. File-backed recovery tests restore synthetic
bytes only, never user profiles. Native key provisioning, metadata recovery UX,
crash persistence and full browser journeys remain unverified; recheck when their
producer/owner work changes. Feedback BAS-FB-008 remains active.

### BAS-WORK-003 — 2026-09-22 UTC — execution sink lifetime (in progress)

Feedback reread. W3 / BAS-RF-021 / J07,J18 / resource-budget and evidence-completeness.
Hypotheses: the UX decorator hides lifecycle cleanup (24 retained queue workers
in the original composition probe); independently, queue close discards accepted
terminal events (1/2 delivered when the downstream hub is held). Current source
retains both defects. A healthy-state response cannot disprove either.

Decision: make existing CloseExecution part of the sink contract, require wrapper
propagation and defer it from workflow ownership on every exit path. Drain accepted
events before hub close, reject late publishes and atomically check closed state
when admitting queues. No new service/framework or compatibility fallback.
The hub's contextless blocking broadcast remains a separate shutdown/backpressure
limitation; these changes do not by themselves qualify bounded shutdown against a
permanently stopped hub or attribute the entire historical 10 GiB heap.

Scoped repair verified: events, collector and workflow packages pass `-race`.
Maintained tests hold the hub while accepting terminal events (three terminal
kinds), exercise completed/failed/cancelled/compile-failed workflow exits, preserve
another execution's buffers, and reject late events before UX persistence. The
late-event test first failed on persisted evidence, then passed after delegate
admission moved before collection. Retained probes: leaked workers 24 → 0, hub
close receipts 0 → 24, accepted terminal deliveries 1/2 → 2/2. Recording API
probe expectations 1/9 → 3/9; the six remaining failures are still reported.
Standalone execution probe sink implements the now-required close contract.

Lifecycle owner: workflow → Sink contract → decorator → queue worker → hub.
Removed the concrete type assertion, separate closed-state check and queue-clear
drop path. No compatibility fallback or new runtime module. API runtime +3 lines
for required lifecycle/rejection semantics, tests +197 lines. Cyclomatic sum over
53 affected functions 317 → 319; this slice improves ownership/correctness, not
branch count. Cumulative touched Go sum is down 14 across cycles 001–003. Whole
API runtime 94,165 → 94,104 (-61), UI -14. The workflow orchestration functions
remain complexity 56/29 and >500-line cohesion debt; no wrapper extraction credit.
Evidence: [event lifetime receipt](evidence/rehabilitation/event-lifetime-2026-09-22.json).

Owner validation `20260922-032049-4b5c680b` returned failed after one wait. Full
findings artifact `artifact_b401e30fb1b458c052a049e71f8a48f2` identifies actual
program assertions (4 failures/7 errors), video.webm detected as audio/webm,
UI coverage 28.42% statements/30.24% functions/64.92% branches/28.42% lines
against unchanged 85%, and a CLI companion reimplementation. These are repair
work under RF-014, not unavailable checks. Workflow provider was recovered via
`workflow-health validate get ec1304ed-fef0-4f37-91ff-85eeb3bceae2 --json` and is
canceled, cancellation_requested=true at the 900s deadline. No executable browser
result exists in that receipt. Recheck after provider execution diagnosis, not
an identical immediate retry. Host swap-pressure warning is retained; do not
infer the assertion failures from host pressure.

Tidiness same 1,104 findings, duplication 35,603 → 35,585 (-18), still above
35,302 budget. Relevant sources changed during this run, so no isolated final
snapshot claim. Board cycle 003 succeeds but still has 17/17 pending outcomes.
Feedback BAS-FB-008 remains active. Full-source compile initially overlapped a
test import edit and failed importing errors; package tests subsequently pass.
Final stable-input compile is pending; retain both observations.

### BAS-WORK-004 — 2026-09-22 UTC — unit-receipt failures (in progress)

Feedback reread. RF-014 / evidence-completeness and known-flow-reliability.
Fix the owner-reported assertions without reducing coverage floors or skipping
tests. First hypothesis: host MIME registration takes precedence over the
export source's known video formats; canonical MP4/WebM types must be stable
across hosts. Then diagnose the governed workflow-program failures from their
full focused output. Shared tree remains dirty; preserve unrelated work.

Results: the 45 Python program cases pass from the root but fail 4 assertions
and 7 errors through the Go launcher. Root cause is lexical parent traversal
before resolving `..` in `__file__`, pointing the real kernel import at the wrong
directory. Resolving that path restores all 45 cases through the actual launcher;
no product verifier or expected result changed. WebM canonical video typing now
precedes OS registrations; tests force audio/webm registration and retain mixed
case/empty-path cases. The recorded-video handler uses that existing owner, deleting
its duplicate sniffing policy. Generic mixed-media stores retain normal detection.

Adversarial caller review found `executeResumedWorkflowAsync` constructing another
unclosed sink. Expanded the existing lifecycle test to fresh/resumed × four exits;
all resumed cases failed before repair. Resumed execution now defers cleanup and
uses the same existing `executionOutcome` policy as fresh execution, deleting its
cancel-substring/failed-status branch. Race checks for workflow, export source,
workflow handlers and all handlers pass; eight lifecycle cases pass. This amends
the previous regular-execution-only scope of BAS-WORK-003.

CLI unit finding was a real duplicate helper. Both parity callers now import
`cli-core/cliapptest.ReadManifest`; deleted `cli/internal/testutil/manifest.go`
(20 lines), no shim or dependency addition. Entitlement/workflows/testutil package
tests pass. Stable final API compilation passed after the overlapping-import
observation from cycle 003. Durable proof: [unit boundary receipt](evidence/rehabilitation/unit-boundaries-2026-09-22.json).

Comparable cyclomatic sum over 24 functions 221 → 215; policy owners reduced
for video export and execution outcome. API runtime -20 this cycle, cumulative
94,165 → 94,084 (-81), UI -14. Go tests +45 net including deletion of the CLI
helper. No source budgets or coverage floors changed. Touched export handler's
64/20-complexity functions remain substantial debt; this small repair does not
establish structural qualification. Board `prog_0a0625c8-06cf-42e6-8a8b-fd437f63091b`
completed with 17 pending outcomes. Full unit/tidiness rerun admitted after these
changes; no workflow rerun while its unchanged deadline cause is unresolved.

Final-source owner run `20260922-035031-d39d8044` completed in 100s and was
attached once. Go command failure and companion reimplementation no longer
appear. The remaining unit command failure is the UI coverage floor: statements
28.57%, functions 30.34%, branches 64.96%, lines 28.57% against unchanged 85%.
Tidiness reports 1,105 findings (1 error, 236 warnings, 868 infos), selected
ratchet failure long_files=82 against recorded 81. Its long-file location set
matches the preceding run; no new location in this cycle. Final-source metrics
must be read from that owner, not inferred from the earlier duplication total.
Findings artifact: `artifact_d1c395176a0b65936557555c29869976`.

### BAS-WORK-005 — 2026-09-22 UTC — atomic profile commit (design/investigation)

Feedback reread. RF-025 / J06,J14 / profile-durability. The two-file writer
cannot implement one atomic aggregate commit: protected bytes are destructive
before metadata rename. Consider one encrypted profile envelope, keeping the
same repository contract and public profile shape. Reuse api-core's atomic-file
writer instead of a second temporary-file implementation. Reject runtime
conversion/fallback and keep any required personal-data conversion outside the
repo with byte-preserving rollback and synthetic round-trip/fault proof.

Storage-steer and required portability guidance read; storage-manager validation
retained at /tmp/bas-storage-before-005.json (failed, existing direct-writer and
isolation findings). No engine/dependency addition is justified for this bounded
profile aggregate. Operator explicitly authorizes offline data preservation and
forbids runtime migrations; file-only tracking overrides visited-tracker logging.
First inventory actual storage/key availability without exposing credentials;
then decide conversion and publication boundary before changing the runtime format.

Investigation: canonical storage resolver reports one `.json` profile with
plaintext storage at `~/.vrooli/data/vrooli/browser-automation-studio/session-profiles`.
No `.protected` file exists. The live process lacks BAS_SESSION_STORE_KEY.
Metadata-only credential status shows the encrypted-file authority available and
`vrooli/browser-automation-studio:session-profile-keyring` unconfigured. No user
state was modified or secret printed. Therefore format work is a candidate only
until offline conversion and key recovery are verified; do not restart yet.

Candidate single-document implementation now passes profile/handler race tests,
including rejected write/rename preserving the prior snapshot, one protected
commit document, 24 concurrent complete saves with no mixed generations, foreign
identity refusal and old-format read refusal without mutation. Shared atomic
writer fault tests pass (write/sync/close/chmod/rename). Retained profile expectations
7/18 → 9/18; concurrent field updates and eight replay cases still fail. The
staging fault probe now targets the atomic writer seam, with actual file mechanics
covered by api-core tests; original dated fault evidence is unchanged. Existing
recovery fixtures now remove the envelope's sealed field rather than an obsolete
sidecar. Production crypto still uses the old environment key at this checkpoint.

Necessary adjacent boundary: RF-012 key provisioning/recovery, using the existing
credential-authority-go owner (no scenario key store or environment injection).
Declare a generated destructive encryption-ring credential in BAS's manifest,
resolve through the authority with a data-owned loss witness, and retain old
versions for reads. Dependency gateway preview approves the existing repository
module; apply through that gateway. This extension remains within BAS and its
supported shared credential boundary, with authority-fault/missing-key/rotation
tests. No real-account action or credential-loss override is authorized or needed.

Credential implementation removes BAS_SESSION_STORE_KEY and implicit `.test` keys.
Explicit synthetic authority injection keeps host credentials out of test runs.
Rotation retains old versions; missing active/historical keys, malformed rings,
provider failure, and credential loss with profile-only/witness-only/history all
fail visibly. Persistent history is restored when data and keys are recovered
without their witness. `go test -race` passes profile, handlers and all testutil
packages; full API compile passes. Dependency approved validation passed with
preexisting governance warnings; security-health dependency status is available
(index_ready=true, ecosystem vulnerability counts are not a BAS security verdict).

Offline personal helper `/tmp/browser-automation-studio/migrate-profile.go` passes
synthetic encrypted roundtrip, byte-identical rollback after publication failure,
changed-source refusal and unknown-field refusal. It accepts only the inspected
plaintext format, fails on unknown data, and never enters runtime source. Actual
staging uses the platform authority without printing values. Driver health reports
zero sessions before this operation; API healthy. Original live data remains
unchanged pending staged verification and lifecycle stop.

Publication completed after `make stop` returned success. One profile preserved
field-for-field through fresh repository reads; old bytes remain in
`~/.vrooli/data/vrooli/browser-automation-studio/session-profiles.rollback-20260922`.
`make build` passed. `make start` operation `startop-61bbc2e53728c37501c85b5678b795e7`
is progressing through normal dependency freshness and setup (session 62836).
The sampled 1,000 recent executions were terminal; driver had zero live sessions.
No saved profile was retired and no browser was navigated to a real account.

Final source cleanup removes obsolete mock Rename/WriteFile methods as well as
the old production writer. Final race checks pass. Changed-file union Go cyclomatic
sum vs HEAD: 147 functions / 853 → 147 / 840 (-13); this includes new credential
validation and the removed unused mock paths. Source inventory digest
`b21afc9aa40d50cd14cf4cbd56598760e39dba570f8d92e697bac1658d167e8e`; API runtime
94,102 (-63 from original 94,165), UI 131,518 (-14); this cycle API +18 against
BAS-WORK-004. Source size is not a performance claim. Shared worktree caveat applies.
Board `prog_1f53d16a-40b5-41d7-96bf-1c60c1080cc2`: status ok, 17 pending telemetry,
product_qualified=false. Contract preparation check passes. Retained profile
expectations remain 9/18 met (field-update loss and eight replay failures remain).
Test Genie `20260922-042038-b5b05137` unit/tidiness admitted once; attached once
with its recommended 175s timeout. Do not re-admit or poll while wait is active.

Lifecycle restart completed healthy, build identity
`sha256:ad61ab7fb5444f3de0c8143db5918afef1755e3b8bb53951551e1d569b2f52ae`.
Live profile metadata API returns exactly the original identity and name, omits
protected storage, and reports one profile. Repository comparison preserves every
field. Observed fresh-process credential/read latency was 1,230ms cold, then 0.40ms
and 0.30ms warm; small shared-host samples, not a latency qualification or baseline
comparison. Off-host recovery and real-browser sign-in continuity remain unverified.

Owner run `20260922-042038-b5b05137` terminal failed in 97s: API/CLI command failures
absent; UI coverage command still below 85% (owner reports 28.6% workspace coverage).
Tidiness: 1,109 findings, 1 error / 239 warnings / 869 infos; long_files=82 versus
baseline81. Four added HIGH_COMPLEXITY findings are manual assertion boilerplate
in the new fault tests. Replaced repeated error branches with the existing testify
require library, retaining/strengthening all fault expectations; focused race test
passes. This removes assertion boilerplate without changing production policy or
thresholds. A tidiness-only rerun is justified by that changed test source.

Final tidiness-only rerun `20260922-042610-4dbb9896` completed inline in 4s, failed
on unchanged long-file ratchet (82 vs81). Findings returned to 1,105: 1 error,
236 warnings, 868 infos. All four newly introduced test-complexity findings are
removed, with every fault case retained. No owner operation is pending. Final
BAS-WORK-005 evidence: `evidence/rehabilitation/profile-atomicity-2026-09-22.json`.

### BAS-WORK-006 — 2026-09-22 UTC — concurrent profile field updates (investigation)

Feedback reread. RF-026 / J01,J06,J14. Current probe still proves successful tab
and storage updates can lose the storage update through a legal Get/Save
interleaving. Candidate cause is the service's repeated read/modify/full-save
policy, independently of now-atomic document publication. Write a deterministic
real-file interleaving regression across separate services/repository instances.
Expected: both successful updates survive, and delete cannot be undone by a
stale field update.

Recall found existing profile evidence and the shared platform-go native file
lock owner (Unix flock, Windows LockFileEx, bounded/cancellable wait). Consider
a repository-owned atomic Update operation using one stable, bounded store lock;
all Create/Save/Delete writers must honor it. Convert every service mutator and
handler-owned cookie/tab read-modify-save caller; do not add a mutex only around
the two reproduced methods. Keep whole-snapshot replacement distinct from field
mutation. Remove repeated Get/Save policy and centralize timestamp/error handling
without adding a runtime data migration. No new implementation is committed to
yet; lock wait budget, nil/not-found semantics, caller cancellation and all
writer coverage must be settled from tests and owner interfaces.

Deterministic real-file regression is red in both cases: separate service/repository
instances lose the acknowledged storage update after a blocked tab read, and a
stale tab save resurrects an acknowledged deletion. `concurrency_test.go` uses
channels at the read/publication/native-lock seams, not sleeps for ordering.
Red output: `/tmp/bas-concurrent-red-006.txt`; no user data involved. The selected
boundary is documented in ARCHITECTURE.md before implementation. Initial native
lock admission is bounded to five seconds; the existing context-free service API
still does not propagate caller cancellation.

Candidate repository Update now owns read/modify/publication under a stable
`.profiles.lock`; Create, explicit Save and Delete participate. It reuses
platform-go native locking with five-second admission timeout. All service field
mutators now use the transaction; repeated Get/Save and duplicate session-state
application are removed. The two deterministic real-file regressions now pass
under `-race`, as do existing profile tests. Source is not deployed yet (live
binary remains BAS-WORK-005).

Recording cookie/localStorage and individual-tab handlers have been converted
to mutate within the service transaction; storage mutation retains its LastUsedAt
policy through UpdateStorageState. Handler tests/race checks are in progress
(session 46084, `/tmp/bas-update-boundaries-006.txt`). Remaining proof: handler
interleavings, rejected callback/write/lock invariants, retained probes, full API
compile, debt inventory/board and scoped Test Genie. Caller cancellation and
exclusive browser-session ownership remain broader gaps, not claimed repaired
by serializing a persisted field update.

All profile, recording, session-profile handler and general handler race suites
pass. One intermediate failure was the recording receipt test's counter still
overriding the removed Save interface; it now counts Update commits with the
same one-snapshot/one-commit expectation. New handler tests prove cookie and tab
edits retain a competing committed addition; Go overlay against the preceding
handler source fails both cases without changing the shared worktree. Repository
tests also preserve bytes after rejected callback, identity mutation, lock admission
and commit failures. Retained expectations: 10/18 met, only eight replay cases fail.

Changed-file union vs HEAD: 220 functions / cyclomatic sum 1,105 → 229 / 1,088
(-17). New ownership operations add functions while removing repeated update
policy; this is not counted as a function-count reduction. Current API runtime
94,111 (+9 this cycle, -54 from original); UI remains -14. Source digest
`9c6ba30e654ab865ea50dbf8c21ed0d4bc5221ca681a90ae459fb6284f114997`.
Board `prog_00ae62a1-beb5-4993-a191-77fd22812feb` succeeded with 17 pending rows
and product_qualified=false. Test Genie `20260922-043957-3e0e9df4` admitted once,
175s quiet wait attached once. Build is running; no deployment claim for cycle006.

BAS-WORK-006 deployed through successful make build/restart after observing zero
live driver sessions/recordings. Health build identity
`sha256:010f4c391123888043b276ec2756d5c45f75e826d15f6efe00e2b92f8fbd4b0a`.
Original metadata identity/name and all underlying profile fields remain equal
to the preserved rollback copy. Owner run terminal failed in 97s: only UI coverage
command failure, tidiness 1,105 findings (1 error, 236 warnings, 868 infos), same
long-file ratchet82 vs81. One service duplication finding removed; one five-line
channel-wait duplication finding added in the deterministic concurrency test.
No threshold or assertion changed to improve the score. Evidence:
`evidence/rehabilitation/profile-concurrency-2026-09-22.json`. No operation pending.

### BAS-WORK-007 — 2026-09-22 UTC — opaque authentication preservation (investigation)

New adversarial angle after profile transaction proof: targeted cookie deletion
round-trips the whole storage object through a partial typed projection. A synthetic
HTTP fixture proves the operation acknowledges success while dropping unrelated
origin IndexedDB data and another cookie's partitionKey. No live profile was edited.
Go overlay keeps this initial experiment out of the stable cycle006 source.
Red proof: `/tmp/bas-profile-preservation-red-007.txt`; overlay source
`/tmp/browser-automation-studio/profile-preservation-adversarial-007_test.go`.
Expected: change only the requested cookie/localStorage subtree and preserve all
other authentication fields, including unsupported future fields; preserve visible
errors for malformed relevant state. Register RF-051 and repair the projection
ownership before claiming this storage-edit journey reliable.

RF-051 candidate now preserves raw JSON outside the targeted subtree. Cookie
deletion preserves partitionKey, IndexedDB and unknown fields; all/origin/item
localStorage edits retain IndexedDB and origins containing opaque state. Large
integers stay raw (no float64 conversion); malformed targeted arrays/items reject
without replacing the snapshot. Focused race tests pass. Original narrower
structs remain only as test assertion decoders, not a runtime serializer.

Boundary cleanup found the exported FileRepository.Save and MockRepository.Save
entrypoints now have no runtime callers: the repository interface and all service
writers use Create/Update/Delete. Remove the obsolete unconditional replacement
path and exercise atomic snapshots through the actual transaction API. Delete two
obsolete mock-temporary-filename tests superseded by real-file generation/failure
regressions and shared atomic-writer fault tests; preserve every useful behavior
assertion. This finishes the previous replacement, not a new compatibility layer.

RF-051 candidate verification: cookie deletion retains partitionKey and origin
IndexedDB; all/origin/item localStorage edits retain unrelated cookies, IndexedDB,
opaque origins and fields, including exact integers beyond float64 precision.
Malformed cookie/origin arrays and null/invalid selected items reject without
changing saved bytes. Existing recording/profile/service race checks pass; all
API packages compile. Native callback transaction tests pass after replacing the
removed unconditional Save calls with Create or Update. An intermediate fixture
mechanically attempted Create for its 24 updates; corrected to Update, preserving
the all-writes-succeed/complete-snapshot expectation.

Current source digest `da4e0c43391b145aaffc11ae5c7b593c22548a522ea230189c54387aa4715e69`.
Runtime API94,115 (+4 cycle007, -50 original), UI131,518 (-14 original). Changed-file
Go union: 220 functions/1,105 cyclomatic total before → 231/1,099 after (-6);
new preservation/error branches are included. editOriginLocalStorage is 11, so
expect a new complexity warning rather than suppressing or fragmenting its logic
for a score. Full owner unit/tidiness and make build are running. Do not deploy
source until their concrete results are retained; existing profile/rollback remain
unchanged.

Pending owner run is `20260922-045614-b2c1b7c5` (unit/tidiness), with one quiet
175s wait attached. Build process session40744 is also pending. Board
`prog_d04a7899-d6f3-4d3e-a3b4-0a721df5a8c4` succeeded, 17 pending telemetry rows,
product_qualified=false. Contract preparation passes; retained profile probe
remains 10/18 met, with eight replay failures. Next after cycle007 publication:
RF-011 driver snapshot still calls storageState() without its IndexedDB option;
inspect installed binding and prove synthetic browser continuity before changing it.
Driver manager.ts already carries unrelated changes; preserve them.

Cycle007 owner run `20260922-045614-b2c1b7c5` terminal failed in137s. Unit failure
is still UI coverage; selected tidiness ratchet now duplication_line_debt35,311
versus35,303 baseline (+8). Full build passed. Restart was initiated after a
health read showed seven driver sessions and zero active recordings. This was
an operator error by the agent: the restart should have waited for session
activity classification/drain. Preserve this fact; investigate execution receipts
for interruption, do not invent an all-idle or no-impact claim. Restart process
session67309 is active. No user-profile mutation was intended.

Restart mitigation: runtime was still on cycle006 while dependency checks ran.
Sent SIGINT then SIGTERM to this agent's restart initiator PID1060384 before the
stop step. The command exited143 (make exited2). API build identity remained
cycle006 and driver uptime increased continuously (1,019,446 →1,092,041ms);
sessions naturally drained10 →2 →0. No BAS target restart interruption was
observed. Cycle007 is NOT deployed. Further deployment must classify/drain active
sessions first and recheck immediately before admission. Preserve the admission
error and mitigation; do not rewrite it as a clean deployment.

Cycle007 deployment completed after an immediate zero-session/zero-recording and
terminal-execution gate, explicit make stop, then make start. API PID was not
changed by the canceled first attempt. Successful deployment build identity
`sha256:75fc64c4ec22e76c9c3332ec4d407531580b6ae2c3b7f028cb783e23ff04c1d8`; API
healthy, original profile identity/name and all stored fields preserved. No owner
run or lifecycle wait pending. Owner result: 1,107 tidiness findings (1 error,
237 warnings, 869 infos), duplication35,311 (initial35,603, delta-292), still
above recorded baseline35,303 and target35,302. New warnings include targeted
edit wrappers and editOriginLocalStorage complexity11. No suppression. Evidence:
`evidence/rehabilitation/profile-preservation-2026-09-22.json`.

### BAS-WORK-008 — 2026-09-22 UTC — browser authentication snapshot (investigation)

Feedback reread. RF-011 / J01,J14. Installed rebrowser-playwright1.52.0's own
BrowserContext types document storageState({indexedDB:true}); SessionManager
currently calls storageState() and the local SessionSpec repeats a narrower
cookie/localStorage-only schema. Hypothesis: capture omits IndexedDB while the
restoration path can already accept it. Prove using a real browser and local
synthetic identity stored independently in cookie, localStorage and IndexedDB;
close/reopen one profile and keep a second identity distinct. No real sites or
accounts. Only then enable the supported option and derive the storage type from
the provider instead of maintaining a second partial schema. Preserve unrelated
preexisting edits in manager.ts and types/session.ts.

Cycle008: real Chromium fixture failed on missing IndexedDB identity (cookie and
localStorage remained alpha). With indexedDB:true, 59 session/context/browser
checks pass and TypeScript compilation passes. Provider-derived storage type
removes 15 runtime lines. Fixed a cycle006 error-mapping regression: missing tab
now retains its own NotFound message; strengthened HTTP assertion failed before
and recording-handler race suite passes after. Full build passed. Owner run
20260922-051505-58a226c1 unit/tidiness has one 239s wait attached, pending. Driver
is still outside owner discovery, so this run cannot qualify its browser fixture.

### BAS-WORK-009 — 2026-09-22 UTC — required driver validation (investigation)

Feedback reread. RF-014. Unit Health plan-only returned only api/cli/ui, and a
fresh Code Facts surfaces query omits the service.json-declared playwright-driver
sidecar. Installed Unit Health already supports node-jest. Necessary shared-owner
scope extension: Code Facts api/internal/facts/discovery.go and maintained tests,
plus its architecture docs; discover declared component roots instead of imposing
new BAS directory layout or implementing a private inventory in Unit Health.
Then add BAS's required driver role using the observed surface identity and
existing Jest adapter. Validate fixture discovery, owner package tests and live
plan inclusion before claiming owner execution. Existing thresholds stay intact.
Unit Health skill recall scope was unregistered; read-only plan remains usable.
No external log is written under this file-only engagement.

Cycle008 owner terminal failed96s: UI coverage command and tidiness (1,107
findings, 1error/237warnings/869infos). Direct driver tests are green; routine
owner coverage still excludes driver. No pending wait. Initial Jest reported
an exit-delay warning; detectOpenHandles rerun passed59 with no handle report
and no forced exit. Source digestd359400b95700b200108a938d90a24b1b26a3618b05ebe96323851d510c2fa4b.
API94,119 (+4cycle/-46original), driver55,552 (-15cycle), UI131,518 (-14original).
Go mapping adds one branch to restore prior user-visible error semantics; no
claim of a complexity reduction for that repair. Board remains17pending and
product_qualified=false. Evidence profile-indexeddb-2026-09-22.json. Deployment
will accompany the validation-owner repair after a fresh drain check.

Cycle009 scope refinement: Unit Health's Node workspace resolver applies React
Vitest advice to all TypeScript surfaces, including declared sidecars. Extend
necessary owner repair to unit-health/api/internal/adapters/workspace.go and its
existing tests: recognized non-UI TypeScript server roles can use Jest; actual
React/UI rules remain. Prove the distinction with a sidecar fixture and retained
UI regression. BAS Jest maxWorkers=1 projects its existing owner runner cap,
without changing coverage floors or excluding tests. Code Facts focused surface
regression is red before/green after; full facts -race passed. Owner unit/structure
run passed15s but reports unit L0/TEST_MISCONFIGURATION; keep that qualification
limit instead of claiming all Code Facts tests were admitted. Full build passed.

Code Facts cache.go analyzer identity advances to components-v1 with the changed
surface semantics; otherwise a seven-day report cache could keep omitting the
driver after deployment. This is within the necessary discovery-owner boundary.
Existing read-only source fingerprints do not identify analyzer implementation.

Live owner repair deployed healthy: Code Facts startop-a75c2365bea6d3567b03842e444cf828,
Unit Health startop-71869221859dfe0fbc35f136ad5a9521. Fresh cached describe reports
playwright-driver as KNOWN with analyzer code-facts.components-v1. Unit Health
now plans four workspaces and driver node-jest is ready, with test/coverage
commands and no driver policy error. Shared checks: code-facts facts -race green;
unit-health adapters/discovery/validation -race green; owner unit/structure passed
for both (code-facts20260922-052102-e449ec3c; unit-health20260922-052234-d7bef24a).
Shared-owner changed Go boundary grows2391→2453lines and cyclomatic529→546
(109→110functions). This closes omitted execution coverage, not a source-size
reduction; include the additional discovery safety/policy branches honestly.

### BAS-WORK-010 — 2026-09-22 UTC — profile path boundary (investigation)

Feedback reread; repository source admits any nonempty ID and joins it to a
filename. Disposable sibling-file probe confirms Delete("../outside") removes
an unrelated .json file and returns nil. It touched only a newly created temp
fixture, never operator data. SessionProfiles Delete RPC validates UUIDs, so no
claim of that endpoint being exploitable. Repository boundary and less strict
recording read/edit paths still lack this invariant. Put validation at the
path-owning repository boundary, reject invalid IDs before I/O, retain UUID and
existing safe fixture identifiers, and test all CRUD paths with independent
outside-file preservation. Owner run009 is still pending; no new production
profile mutation yet. Record as RF-052 after maintained reproduction.

Cycle009 owner terminal failed233s: driver suite now admitted and ran123 passing
suites/1,442 passing tests, with2failed suites/26failed tests and1suite/2tests
skipped. It then retained a handle and hit the60s no-output watchdog (130s driver
command total). Owner excerpt shows injector-selectors importing transitive
playwright instead of the deployed rebrowser-playwright provider and demanding
an unprovisioned Chromium1200; failing launch also breaks teardown. Fix the
fixture binding and unconditional cleanup, then capture the second failure and
retained-handle owner with a bounded detectOpenHandles run. Do not lower the
watchdog, skip the test or force-exit Jest. UI coverage still fails; tidiness
still35,311 (+8ratchet, -292original). No wait pending. Source edits for010 began
after009's API command; its receipt does not cover that later repository change.

Cycle009 driver diagnostic was bounded at180s and exited124 before completing,
not a pass. Using rebrowser exposed setContent() on a blank document stalling;
an independent local browser probe times out setContent then succeeds on actual
data-document navigation and evaluate. Selector fixture now navigates to its
synthetic document, preserving every selector expectation. Audio-capability's
remaining assertion concatenated all files, including separately imported
PipeWire qualification code. Keep the platform-neutral decision obligation and
check the transitive imports/exports of audio/index.ts instead, using installed
TypeScript resolution; also prohibit child-process imports. No host-device code
or runtime audio decision changed. Diagnostic additionally observed a transient
synthetic-audio-fidelity Execution context was destroyed after permissions were
granted following navigation; investigate that exact operation ordering rather
than retrying the assertion away. Native audio qualification remains unverified.

Cycle010 implementation: centralized profilePath validation now returns a checked
path to the write transaction, and private snapshot publication consumes that
path. Removed repeated empty-ID checks. Rejects both platform separators, colon,
NUL and path-only dot identifiers. Forty real-file cases, profile/service/recording
race suites, actual recording InvalidArgument HTTP test and full API compile pass.
Build passed. Runtime API94,130 (-35original, +11cycle008→010), driver55,552
(-15cycle008), UI131,518 (-14original). Repository210→215lines, cyclomatic44→47;
cumulative changed BAS Go1105→1104 (-1). Necessary shared-owner009 adds17 outside
that BAS sum; do not claim aggregate complexity fell across the wider boundary.
Retained profile probe stays10/18 with eight replay failures. Board010 returned17
pending outcomes, product_qualified=false. Build010 not deployed yet.

Cycle009 focused selector/audio-decision fixtures now pass33 with detectOpenHandles
and clean exit in1.93s. Selector expectations are unchanged. Synthetic microphone
permission now precedes document navigation; this matches preconfigured-context
capture setup and avoids changing permissions during evaluation. This ordering
change alone is not proof the previously observed transient is eliminated. Full
150s bounded driver diagnostic with coverage/open-handle detection and typecheck
are now running; do not promote them before terminal results.

Cycle009 full driver diagnostic passed125suites/1,468tests (one suite/two tests
explicitly skipped) in76.433s, but process did not exit. detectOpenHandles identifies
AIGatewayVisionClient.analyze's timeout allocated before buildTurns rejects an
invalid screenshot. Body validation throws outside the cleanup try/finally,
leaving the long timer alive. This is a runtime resource leak, not a runner
budget issue. Strengthen the invalid-image regression to require zero timers,
then allocate request resources only after local body validation. No forced exit,
watchdog relaxation or test suppression. Track as RF-053. Synthetic microphone
fixture passed this full run; one pass does not establish a flake rate.

Final owner run for009/010 is20260922-054223-196bad55; one407s wait attached.
RF-053 resource allocation reorder adds no runtime lines or branches;43focused
vision checks pass with detectOpenHandles and clean exit. Previous full driver
diagnostic's exit124 is retained separately from1,468passing assertions. Current
source changes are stable for the final scoped run; earlier009's Go receipt does
not qualify010. Evidence: driver-validation-2026-09-22.json and
profile-paths-2026-09-22.json. No live BAS restart yet.

Final010 build and driver typecheck completed successfully. Awaiting existing
owner run (one waiter, session98097); do not re-admit. Next selected investigation
after deployment is RF-028/RF-030 recorded-action conversion: eight retained
profile-probe replay expectations still fail for modifiers, double-click count,
horizontal scroll, unknown blur/drag handling and page/frame context. Read-only
recall/source inspection has begun; no conversion source changed yet.

Final owner20260922-054223-196bad55 terminal failed167s. API/CLI/driver/typecheck
commands pass; driver naturally exits71.543s with125suites/1,468tests passed,
1suite/2tests skipped. Only UI coverage command fails. Tidiness still duplication
35,311 vsbaseline35,303 (-292frominitial),1,107findings. No owner wait remains.
Preparing drain-gated deployment. Inspection found ListExecutions status argument
is ignored, so do not use a status-filtered read as a drain oracle; inspect actual
returned statuses across bounded pagination and recheck driver health just before
stop. This separate API contract defect needs a maintained reproduction/repair.

Drain gate first refused without invoking stop: full pagination found five
RUNNING snapshots older than the current process, unlike the earlier1000-row
scan. Inspected all2,803 rows:2,448completed/350failed/5running, all five dated
September7–16. Current API beganSeptember22 and driver has no sessions/recordings.
Source confirms both fresh start and ResumeExecution allocate a new execution ID;
these old IDs cannot be active runners in this process. Classify them as orphaned
historical records, preserve their bytes, and admit deployment only if a fresh
scan finds no nonterminal execution created during the current API lifetime and
an immediate driver check remains zero. Do not claim their journal state is fixed.
RF-054 separately proves ignored status filter (PENDING→COMPLETED); RF-055 tracks
orphaned RUNNING history. No lifecycle stop has yet been admitted.

### BAS-WORK-011 — 2026-09-22 UTC — semantic workflow conversion

Feedback008 reread. Recall completed via search-hub (recording conversion query).
RF-028 hypothesis: discarded config plus repeated V1/V2 mapping causes modifier,
click-count, horizontal-scroll and blur/drag corruption. Replace with existing
compiler/typeconv owner, typed flow through service/handler, no compatibility shim.
Necessary same-scenario boundary includes compiler action builder and parameter
builders (drag support and Go/JSON string list equivalence). Source before copies
in/tmp/bas-before-011; original issue/probe evidence retained. RF-030 receives an
explicit unsupported-context boundary pending proper lifecycle reconstruction;
no claim of working multi-tab/frame replay. RF-004 merging remains separate.
Prove semantics through service-generated typed candidates, compiler roundtrip,
unknown/context failures and drag phases, then owner unit/tidiness checks.

Deployment008–010 completed healthy via make stop/start after fresh drain gate.
Build3c0e62a11289d56a838a06f72e452c7b9b49e4466c051e29ebf9a9e2ea6de6aa;
read-only credential-backed verification preserved every original profile field;
live metadata ID/name match and no storage state is exposed by List. Cold1218ms,
warm0.36/0.31ms are observations, not comparative performance qualification.
Owner/evidence files008–010 now include final verdict and deployment. No pending
owner or lifecycle operation. Cycle011 semantic corpus red on previous generator;
implementation in progress, not built/deployed or qualified yet.

Cycle011 scope refinement: typed input explicitly replaces the whole value; the
existing merge implementation would still concatenate full snapshots and mutate
raw payloads before typed conversion. Repair that same deriver ownership boundary
now (RF-004 subset): final snapshot wins only within identical page/driver-page/
frame/URL/selector, submit boundaries remain distinct, no map mutation. Existing
fragment-based fixtures were corrected to actual full-value capture semantics;
new replacement/empty/target/immutability assertions cover expected behavior.

Cycle011 adversarial read during owner validation found a downstream execution
boundary still discarding the newly preserved fields: InteractionHandler.handleClick
calls page.click(selector,{timeout}) only; KeyboardHandler logs modifiers but never
uses them; InputParams submit/clearFirst/delayMs also ignored. Registered RF-056.
This invalidates any whole-replay claim for011; typed-candidate guarantees remain
valid. Next cycle012 will use independent Chromium event/state oracles through real
handlers, repair the semantic execution owner, then revalidate combined source.
No driver source changed while current owner run is in flight.

Cycle011 owner terminal failed172s, API/CLI/driver/typecheck pass. Driver exits
naturally with125suites/1468tests; UI28.6% below85% remains. Tidiness1099 findings
(1error/234warning/864info), duplication35311→34640 (-671this cycle,-963original),
longfiles82→81, coupling12 unchanged. Duplication now satisfies its ratchet; the
remaining selected budget error is complexity385 vsbaseline363. Added context/drag
validation functions explain the count+1 despite total touched cyclomatic667→590
(-77). Retain a cohesive explicit recorded-action switch (23 branches), since its
cases are the small boundary adaptation and labels, not duplicated proto policy.
Review context-identity representation next for a simpler exact binding contract.
Build011 passes but not deployed: RF-056 requires downstream browser semantics
repair and combined validation before treating replay as delivered.

### BAS-WORK-012 — 2026-09-22 UTC — driver action semantics (investigation)

Feedback008 reread; reuse011 browser-semantic recall. RF-056 source evidence:
click options and keyboard modifiers are discarded, input append/submit/delay and
scroll selector/delta options are ignored. Add real Chromium DOM event/state
oracles through production handlers; red before edits. Fix the same execution
ownership boundary and remove divergent option interpretation where practical.
No profile, account, external website or live session is used by these fixtures.

Cycle012 real Chromium fixture red:5failed/1passed; after click/keyboard/input and
scroll-owner repairs6passed. A stepped-scroll unit oracle caught and corrected a
new interpolation overshoot (missing division by step count); retain its failed
receipt. Typecheck found an ElementHandle union overload mismatch, repaired with
an explicit target type. Additional smooth/stepped/viewport oracles added.

RF-057 instrument defect proved: with Babel coverage the scroll fixture fails
with `elementHandle.evaluate: ReferenceError: cov_ieri02bia is not defined`; the
same six actual browser assertions pass without instrumentation. Do not work around
this with production string-eval, discarded source files or lowered thresholds.
Test installed Jest V8 coverage provider, which does not rewrite browser-bound
functions. A provider change breaks direct coverage percentage comparability;
record a new producer identity and preserve all existing thresholds/source scope.

Cycle012 expanded focused suite passes41tests, including10real-browser cases,
with V8 coverage enabled and unchanged floors. Driver typecheck and final Go
context-boundary race check pass. Multiline append was red (insertion at first
line end), then repaired via DOM selection before typing. Scroll unit oracle also
caught the implementation's temporary interpolation error; both failures remain
in evidence. Owner unit/tidiness admission and build are now pending; do not deploy
until their exact results are inspected. No claims of full recording, native OS,
shortcut-composition or concurrent-input qualification.

Cycle013 read-only selection: RF-002/RF-024 share four cache-first journal writes
and two competing query authorities. Recall complete. Inspect repository append,
sequence allocation, actual acknowledgement callers and all broadcast paths before
choosing the storage repair. No013 runtime edits yet; retain pending012 IDs above.

### BAS-WORK-013 — 2026-09-22 UTC — journal schema ownership (investigation)

RF-058 / J06, profile-durability and structural-debt. While tracing RF-002/024,
source search found timeline_entries only in repository SQL and a private test
schema, absent from every embedded production schema. Hypothesis: a real BAS
bootstrap cannot append a journal entry; private test DDL conceals the failure.
Target docs updated first. Repair the existing internal/recording domain schema,
replace private test DDL with its provider, and extend the existing routed-pool
regression to append/read events and verify primary/test isolation. No live data
rewrite, new storage engine or runtime migration. Existing recording_actions
rows stay untouched. RF-002 false success and RF-024 cache pagination remain
separate repair work after this bootstrap prerequisite; no durability claim yet.
Pending012 driver-owner diagnostic session65852; no lifecycle operation pending.

BAS-WORK-013 result: canonical repository fixture and actual routed production
bootstrap both failed with `no such table: timeline_entries` before repair.
Existing recording domain SQL now creates the journal and its two query indexes.
Private DDL removed from repository tests. Routed regression writes/reads only
the leased pool, confirms primary remains empty, and preserves the entry across
reapplying schema. `go test -race ./database ./services/recording/persistence`
passes (10.711s/1.160s); make build passes. Contract and board run
prog_a76a4436-ef0a-4d18-9cc8-bfd867c24b90 complete:17pending/productfalse.
Evidence: evidence/rehabilitation/recording-schema-2026-09-22.json. No runtime Go
branch/function added; removes private SQL ownership instead of introducing a
second bootstrap path. Live deployment pending combined011–013 drain gate.

012 validation diagnosis: driver-only Unit Health CLI also hit its client deadline
without native verdict. One direct bounded600s diagnostic of the same full Jest
coverage command is attached(session48229) to obtain actual failing assertions
or completed timing. No threshold/floor/exclusion change, no Test Genie run
pending. Aggregate changed BAS Go cyclomatic is now1623→1538 (-85), with shared
owner009 +17 giving wider affected net-68; source files8616vs9248 (-632).

### BAS-WORK-014 — 2026-09-22 UTC — durable journal authority (investigation)

RF-002/024, J06 and recording fidelity. After013 supplies the missing table,
four service writers still append to mutable cache before storage and suppress
storage errors. Only two are called by production; the Unified interfaces exist
only in tests, while HTTP unconditionally sends200 and broadcasts. Target:
repository-owned append identity/order and full-history queries, error propagation
to the actual HTTP boundary, one production writer pair, no alternate cache or
unused Unified API. Tests will use the canonical SQLite schema, save faults,
1,001+ events, reopen, concurrent repositories and retries with independent
expected IDs/counts. Preserve journal data and raw observed actions, including
two distinct navigations to the same URL. No live data mutation or destructive
migration is required. Remaining callbacks/page lifecycle side effects are to be
traced before claiming end-to-end durable acknowledgment.

012 full diagnostic completed:126suites/1,476tests pass, 1suite/2tests skipped;
349.969s natural exit under V8 coverage. No product assertion failure. RF-059
records the substantial producer slowdown versus preceding Babel71.543s; coverage
values across those producers are not comparable improvement. Existing300s unit
phase could not return native output. Raise only that phase deadline to600s,
matching the bounded runner ceiling and measured serial aggregate; all coverage
floors, source inclusions, tests and ratchets stay unchanged. Next combined owner
run follows014 compile/tests; no run currently pending. Evidence raw diagnostic
/tmp/bas-driver-full-diagnostic-012.txt. Both prior deadline attempts retained.

BAS-WORK-014 result before owner validation: actual HTTP red returned200 and
broadcast on storage failure/unknown session; repaired to503/404 with no broadcast.
Real journal red falsely published2 uncommitted events, reported501of1001 with
page starting902, and erased a distinct same-URL navigation. Repository now owns
atomic sequence/identity, immutable retry comparison and all history queries.
Service has no cache. Removed recorder.go, unused Unified writers and batch
persistence; all constructor, handler/interface/mock, wire and retained probe
callers converted. Page-event and navigation HTTP failures propagate too, with
explicit browser-effect wording. Failed session registration releases the opened
browser; fake-driver regression confirms close and zero retained ownership.

Canonical disk tests cover reopen, two independent writers, exact1001→1041
history, offsets1/101, identical retry, conflicting ID, large JSON numeric
precision, corrupt JSON and commit fault. Actual handler regression covers
success/error/unknown session. Scoped race suites pass. A temporary fixture
mistake used a raw explicitDSN without shared tuning, exposedSQLITE_BUSY, and was
corrected using storage.SQLiteDSNAt (same production authority, no weaker custom
mode). Journal race timing41.502s→2.733s is a fixture-configuration observation,
not a measured live performance claim. All expected outcomes unchanged.

Measured ten-file pre014 owner union4,600→3,778lines,108→96functions,
560→505cyclomatic before final formatting; exact refreshed values in the evidence
receipt. Runtime source inventory API92,712 versus original94,165 before final
formatting. Contract and board prog_fb5a98ac-4d49-4184-8144-05a70d14073d pass
producer checks,17pending/productfalse. Build passed. Evidence:
evidence/rehabilitation/recording-journal-2026-09-22.json.
Owner20260922-070615-366d3617 pending one533s wait(session67863).
Broader raw-context preservation, callback-gap UI, driver reconnect and native
power-loss remain unqualified. No claims of full recording readiness.

BAS-WORK-014 owner result:20260922-070615-366d3617 terminal failed379s;
API14.978s,CLI0.895s,driver283.766s andUItypecheck12.019s pass. Driver126suites/
1,476tests naturalexit, no retained-handle failure. UI coverage28.57%vs85% remains.
Tidiness1,077findings (vs1,098), longfiles80(vs81), complexity380(vs384),
duplication604groups(vs620), lineDebt35,064(vs34,640), coupling12. The duplication
increase is retained honestly;408of424 anchors to the repeated back/forward/
reload handlers. Cumulative line debt stays539below original35,603; the unchanged
complexity ratchet still fails. Next owner simplification should consolidate
actual navigation commit/notification behavior instead of chasing detector output.

Fresh deployment gate07:14:23UTC:2,924execution rows, no driver session/recording,
five exact unchanged historic orphan RUNNING rows only. make stop completed;
make start admitted07:14:56UTC pending session50204. No history deleted/reclassified.
Read-only production database before stop had no journal table and zero recording
session/action rows. Post-start health, schema and original-profile read checks
remain required. All four011–014 receipts now include final owner result/catalog.

Deployment014 completed. make start returned0; API/driver/UI healthy at07:17:18UTC,
new build e55e1a548d5c0f27774977035f8894db9d09e27f02089ebca18c45eab7962992.
All original saved profile fields match on3read-only reads; live List identity/name
and one-profile count match original, with no storage state in metadata.
Actual startup-log DSN is scenario/data/browser-automation-studio.db; journal table
exists, session/action/journal counts0. Correction: the earlier default-root read
was an unused database, not the running authority. Its receipt is marked invalid
as live evidence; pre-start actual journal presence was not measured. Bootstrap
repair remains supported by real canonical-schema red/green tests. No historical
execution rows or saved profile bytes rewritten. All011–014 receipts include this
correction and deployment proof. No operation pending; continuous work proceeds.

### BAS-WORK-015 — 2026-09-22 UTC — navigation journal ownership (investigation)

RF-002 and structural-debt. Owner014 attributes408of424added duplication debt to
three navigation handlers. They repeat action construction/commit/broadcast and
skip recording silently when API session tracking is missing. Consolidate that
real post-effect owner, preserving each route, payload, browser result and page
notification semantics. Add route-level success/save-failure/missing-session
assertions before repair. Adversarial journal review also found a decoder that
accepts a valid JSON prefix followed by garbage; add the full-document regression
and reject that corruption. Prior broader recording recall reused; feedback008
reread. Native and callback-recovery qualifications remain open. No operation
pending before this cycle; live014 build and saved profile verified.

BAS-WORK-015 result: red proved missing-session false200, stale reload metadata
and trailing JSON corruption accepted. Shared navigation owner removes134lines
from record_mode.go, preserving all route payloads and page notifications.
Common strict journal decoder covers action/page events with exact numeric JSON.
Focused recording/persistence/handler races2.817/1.182/1.198s pass; make build passes.
Final owner20260922-072645-a721922d terminalfailed:complex380 unchanged,
dupDebt33,521 vs35,064, group607vs604; grouping changes are not behavioral claims.
Interimrun20260922-072400-60d89a29 retained; two new complexity findings prompted
simplifying the same nine test cases and consolidating whole-JSON policy.
Three-file comparison1,787→1,699lines,34→36functions,246→247cyclomatic. Service
formatting adds10readability-only lines outside that union. No local net complexity
claim. Contract passes; board01517pending/productfalse. Evidence:
evidence/rehabilitation/navigation-journal-2026-09-22.json. No pending operations.
Live014 unchanged; deploy015 with next coherent repair after fresh drain check.

### BAS-WORK-016 — 2026-09-22 UTC — frame measurement ownership

RF-002/022 callback investigation: HTTP failures and circuit-open are swallowed;
stop does not join deliveries; FIFO can evict uncommitted entries. Close/reset
suppress recording/cleanup errors and delete buffers/ownership (RF-036). A safe
repair must span delivery identity/ack, pending retention, stop/restart and close,
not just add a retry loop. No callback code changed. Recall
/tmp/bas-callback-recall-016.txt; provider results do not supply a BAS replacement.

Select RF-009 bounded frame measurement repair while that boundary is understood.
The existing collector divides retained bytes by lifetime skips/time and calls
blocking socket wait processing latency. Desired: one bounded ring/window owner,
window-local counts/bytes and monotonic observation interval; lifetime count only
for broadcast cadence. Processing durations start after a full frame arrives.
Existing wire names remain compatible but descriptions/UI must not claim measured
network or input-to-paint latency. Independent clock/window fixtures and actual
WebSocket delayed-arrival test will discriminate. Positive capacity invariants,
empty/reset, skipped samples, wrap/recent order and concurrent access required.
Feedback008 reread. No deployment or test operation pending.

016 source inspection expands the same measurement repair to driver PerfCollector,
which repeats lifetime/window mixing, never resets its start time, and can divide
by zero. Include both collectors and their shared UI diagnostic descriptions;
monotonic intervals exclude wall-clock jumps. No new dependency or runtime owner.

016 focused green: Go performance/handlers race1.025/1.293s; driver performance
unit+real Chromium capture14tests2suites3.094s; driver and UI typechecks; make build.
Retained recording-api now9/9. Real WebSocket red100.241ms idle falsely reported
as receive time; repaired omission preserves processing measurement. First test
fixture had to retain collector before handler cleanup; nil after disconnect was
not treated as product failure. Both collectors now use bounded observation rings,
window-local totals and detached chronological recent frames. Driver reset renews
monotonic epoch; wall-clock jumps do not change elapsed time. Four UI consumers
converted to the existing frame-streaming types; duplicate server declaration
removed from usePerfStats. Legacy JSON names retained with honest processing label.
Board prog_72b48fe9-3355-4d87-962f-96231dbb91d5 completed17pending/productfalse.
Owner20260922-074813-d12be3cc unit,tidiness admitted07:48:13UTC; exactly one quiet
663s wait attached in tool session59914; resume that session, never re-admit/poll.
Live014 remains healthy;015–016 await deployment after owner evidence/drain check.

016 measurement: comparable Go3-file union1,578→1,526lines,32functions unchanged,
180→178cyclomatic; driver collector378→323lines. Existing UI canonical types
replace79duplicated declaration lines in usePerfStats (2importlines added).
Affected Go union vs original37files15,845→14,258lines,541→523functions,
2,406→2,266cyclomatic (-140); sharedowner009+17 gives wideraffectednet-123.
Three sequential microbench trials: Go Record amortized allocation270→0bytes/frame;
median36.08→42.92ns/op, a small measured cost increase for monotonic observation,
not a speed improvement. Shared-host conditions limit timing attribution.
Initial concurrent benchmark retained but not treated as speed evidence.
A cwd mistake ran root make build (completed0, no install); that log is explicitly
excluded as BAS evidence. Correct final BAS make build now pending.
Evidence evidence/rehabilitation/frame-window-2026-09-22.json; owner wait59914
remains pending, do not poll/re-admit.

RF-036 pre-repair inspection during016 owner wait: driver teardown catches every
recording/tracing/context failure; manager catch/finally removes session ownership
and concurrent close returns empty success. Go closeSessionWithArtifacts also
returns void after close/artifact-write failure, and resolveArtifactPath can return
a missing source path after failed download. These are one finalization boundary,
with capture/teardown evidence and repeat-safe ownership required. Existing test
expects a synthetic already-closed-page error to succeed; distinguish actual
closed resource from unverified generic close failure when repairing. No017
runtime edits started while016 owner source is being validated.
Final BAS make build completed0; initial root build excluded.

016 final owner result:20260922-074813-d12be3cc failed (unit438s); API/CLI/driver
andUItypes pass. Driver126suites1,478tests naturally exit345.608s, command346.324s.
UIcoverage still28.57/30.34/64.96/28.57vs85. Tidiness1,080findings,80long/380complex/
607dup/12coupling, debt33,697 (+176cycle; -1,906original). No weakened threshold.
One quiet wait59914 completed; artifact catalog retained in015/016 receipts.
Fresh deployment gate stopped before lifecycle mutation due new RUNNING rows;
07:57:39UTC observation10newRUNNING/10driver sessions over3,046historyrows.
Do not interrupt or reclassify active work. Runtime014 remains;015–016 source/build
verified but not deployed. Recheck only after current workload settles.
Continue RF-036 finalization boundary investigation/repair; feedback008 reread.

### BAS-WORK-017 — 2026-09-22 UTC — executor finalization authority

RF-036/J08/J18. Start with the Go executor's complete finalization boundary:
Execute currently returns nil after engine Close/CloseWithArtifacts or artifact
writer failure; missing remote downloads become requested paths. Use public
Execute fault cases (ordinary close, artifact close, write, missing download,
cancellation context) before repair. Return combined errors and preserve original
action cause, use bounded WithoutCancel context retaining routed storage values,
and consolidate video/trace/HAR handling. Preserve source evidence on failed
import; no repeated browser action. Driver teardown suppression/recovery remains
a connected next owner, not claimed repaired by Go propagation. Shared dirty
executor code inspected; no unrelated edits reverted. No test run currently
pending; live workload prevents015–016 deployment. Feedback008 reread.

017 first Go executor repair passes public failure/cancellation/local+remote bytes
tests and executor/engine/workflow races1.253/1.078/1.067s. Retained execution-api
close-failure probe now passes;5other failures remain. Invalid --case execution
invocation corrected to execution-api and not treated as product evidence.
SimpleExecutor1,899→1,866lines,373→369cyclomatic;close helper17→13.
Tidiness interim20260922-080702-56cc1e9d terminalfailed; quiet wait completed once.

017 boundary extension within BAS: actual execution-writer/external_artifacts.go
suppresses stat/read/sanitize/storage errors. Trace metadata gets a digest but
neither inline bytes nor durable storage, so downloaded trace temp cleanup loses
the only local copy. Include this owner and its tests before claiming Go
finalization repaired. Preserve sanitized HAR policy and source files. Success
requires committed bytes or supported sanitized inline bytes; failed stores and
missing sources propagate, valid sibling artifacts can persist with explicit
combined failure. Nil/empty store receipts cannot establish success.
The driver teardown suppression and callback pending recovery remain next owners.

017 actual writer red reproduced seven false-success cases and missing stored
trace bytes. Repaired with one per-artifact preparation transaction and batch
error accumulation. Missing/malformed/nonregular sources, failed/empty/incomplete/
wrong-size receipts fail explicitly; valid sibling artifacts persist. Trace/video
bytes require storage. HAR stores sanitized derivative only (small derivative can
still use the existing inline policy without a configured backend). Sources remain
unchanged. Removed video-only storage gate and repeated log-and-continue branches.
Two older trace metadata tests now supply the same memory backend and retain all
metadata/path assertions; new test deletes the source and retrieves matching
trace bytes/hash through storage. Real writer failure also reaches public Execute.

Cohesion review: prepareExternalArtifact27 versus former RecordExecutionArtifacts37.
One read/sanitize/store/receipt transaction; batch accumulation separate. Branches
are explicit failure/disclosure decisions. No extraction-only complexity claim;
keep this hotspot visible in the owner scan. Final focused race suites pass;
final unit,tidiness owner just admitted (identity recorded after admission output).

017 final focused result: writer/executor/engine/workflow races2.012/1.297/1.081/
1.074s; final make build passes. Two-file runtime2,042→2,007lines,52functions
unchanged,414→403cyclomatic (-11). Cumulative affected Go original→current
39files17,887→16,265lines,593→575functions,2,820→2,669cyclomatic (-151),
sharedowner009+17 yields wideraffectednet-134. prepareExternalArtifact27 remains
visible; aggregate reduction comes from deleted error suppression/repeated import
policy, not extraction. Contract/board017 producer checks pass17pending/productfalse.
Final owner20260922-081559-47cf38fc admitted08:15:59UTC; one784squiet wait attached
in tool session56392. No other owner/build/lifecycle operation pending.
Evidence evidence/rehabilitation/execution-finalization-2026-09-22.json.
Runtime014 unchanged because fresh drain016 found active workload. No live data
changed. Do not mark goal complete/blocked; continue after owner result.

017 final owner20260922-081559-47cf38fc failed401s (unit397s); API/CLI/driver
and UI types pass; same UI coverage floor and complexity ratchet fail.
Tidiness1,078findings,long80/complex380/dup605/coupling12,debt33,679,
-18cycle/-1,924original. Native receipts and catalog retained with017 receipt.
Quiet wait56392 completed exactly once; no build/owner pending.

### BAS-WORK-018 — 2026-09-22 UTC — instruction admission and start retries

RF-031/037,J07. Feedback008 reread. Driver finalization investigation shows
retained closing state is unsafe until the run route respects failed phase
transitions. Same-execution start also resets active phases, so include the
manager and session-decisions owner with the route. Hypothesis: delayed effects,
reset/close and retried starts can allow competing browser effects. Discriminate
with actual manager + route admission, delayed body/action/cleanup tests.
Same-execution start must observe ownership without reset/recovery. Admission
must occur after awaited body parsing and before dispatch; all exits release only
the reservation they still own. Preserve recording phase and audio additions.
Lease wire and payload/invocation cache identity stay explicitly unresolved;
no full fencing claim. Record source before edits, run focused driver checks,
retained probes, typecheck and scoped owner tidiness. No new dependency.

018 initial regression15failures reproduced. Removing start phase recovery and
checking admission after body parsing fixes these. Follow-up actual delayed
reset/close tests pass, but reset completion while an older action is unsettled
still admits another action (200vs409). Add one per-session in-flight reservation
until the promise settles; pool reuse and idle cleanup must respect it too.
This is the same session ownership boundary, with the types file added explicitly.
A cwd-only test-edit attempt failed before writes, then retried with absolute paths.
Earlier unmodified route check passed; not used as repair evidence.

018 focused72tests/4suites2.574s pass with natural exit under detectOpenHandles.
Earlier default-mode Jest printed its one-second advisory but exited; no hang
claim. Driver typecheck passes. Existing browser semantics suite extended with
actual run route/executor/manager/InteractionHandler and a post-effect gate:
first click independently observed, start retry preserves ownership, competing
click409, next click executes after settlement. 11tests3.218s pass. Delayed
actual reset/close and reset-completed-before-action-settled cases also pass.
One prior clean-retry test expected destructive reset of the same owner; replaced
with explicit released-lease clean reuse by a new owner, retaining cookie-reset
proof. Unsafe recovery-decision test replaced by actual lifecycle/route tests;
no preserved capability assertion removed.
Retained execution-driver7/14pass (four phase guards pass;7otherfailures unchanged).
Reuse probe6/17pass unchanged profile/capture reuse gaps. Board01817pendingfalse;
contract preparation passes. Local4runtimefiles2,456→2,265lines (-191); deleted
three unsafe/unneeded decision functions and unused result types; one shared
finally replaces four releases. Existing cumulative Go metrics unchanged.
Owner20260922-083535-a68d66dc admitted08:35:35UTC; quiet784s wait once(session73730).
Last workload observation08:35:32UTC3,088rows,5historicorphans,driver0sessions/
0recordings. Fresh gate required before eventual lifecycle change.
Code Facts before snapshot lacked manifest (unsupported); relative after target
resolved under service cwd (invalid); corrected to absolute and copied existing
manifest to snapshot. Unsupported 'complexity' fact family rejected. These are
producer limits, not passing complexity evidence; after/all read26721 pending.

018 Code Facts all-families read26721 hit client deadline; bounded symbols-only
reads completed (before97306/after71999), retaining native evidence. They measure
function/type removal, not cyclomatic complexity; exact counts in018receipt.
No additional Code Facts read pending. Retained session-frame probe confirms
active-instruction start retry nowpasses; teardown false-success stillreproduces.
Go session wrapper also marks closed before driver receipt; a concurrent Close or
Release currently returns nil before the first request resolves. This must join
one terminal operation result before complete finalization can be claimed.

018 first full owner20260922-083535-a68d66dc terminalfailed437s; quiet wait73730
done. API12.497s/CLI0.816s/UItypes10.697s pass. Driver126executedsuites,
1,497pass/2fail/2skip in348.649s: two older idempotency tests explicitly demand
same-owner cookie reset and unsafe executing→ready start recovery. Align their
expectations to desired ownership and retained new-owner clean-reset capability;
parameterize all phases and assert lease/phase/admission preservation. These were
not caught by initial four focused suites. Independent actual-browser delayed-click
proof already demonstrates why the old behavior is unsafe. No runtime change.
UI coverage unchanged below85. Tidiness unchanged1,078/80/380/605/12/debt33,679.
After focused idempotency check, rerun unit owner; reuse completed tidiness proof.
No deployment until this correction has verified driver evidence.

018 corrected focused90tests5suites2.282s pass. Final unit owner
20260922-084539-e7702029 admitted08:45:39UTC; exactly one778s quiet wait attached.
No runtime edits while owner validates. 019 Go terminal-operation repair can be
prepared/validated through a temporary Go overlay, then applied after018 owner.

### BAS-WORK-019 — 2026-09-22 UTC — joined terminal operations

RF-036/J07/J08. Feedback008 reread. Public Go Session Close/Release wrappers
share a closed marker but return nil while the actual driver request is pending.
Four real HTTP fault cases in a temporary test overlay reproduce canceled joining
callers receiving nil for close-close/close-release/release-close/release-release.
Original source remains unchanged during018 owner. Boundary: automation/session
Session finalization and its tests; consolidate release/close ownership, one
in-flight result and completion signal. All waiters observe original operation
outcome or their own cancellation; canceled waiters must not cancel the owner.
Failures retain session ownership and permit explicit later retry; successful
terminal notification fires once. Keep existing absent-driver404 terminal policy
explicitly limited to resource ownership, not proof of retained artifacts.
Driver teardown suppression and capture policies remain connected next owners.

018 final unit20260922-084539-e7702029 terminalfailed371s, wait81548doneonce.
API12.221s/CLI0.920s/driver288.389s/UItypes10.440s pass. Driver126suites1,502tests
287.528s naturalexit,2skipped. UI coverageunchangedbelow85; no production regression
assertion remains in driver run. First018tidiness receipt reused unchanged.
Fresh deploy gate rejected before make stop:08:52:48UTC3,095rows,1newRUNNING and
1driversession. No lifecycle mutation, live014 preserved. Do not reclassify active
work as oldorphans. Continue019 while this workload owns the browser.

019 overlay green: session/executor/engine/driver races1.054/1.282/1.068/1.035s.
The complete terminal operation replaces separate closed/artifact flags; shared
pending/result signal and retained success have one owner. Concurrent callers
join the same receipt/error, canceled waiters leave owner running, failed attempts
retain ownership for explicit retry, successful callback fires once. Existing
closed-session guard fixtures now use an acknowledged terminal signal and retain
all assertions. Test success/failure controller uses existing approved testify
assertions for readable oracles, not extraction of production branches.
Local runtime407→408lines,36functionsunchanged,76→77cyclomatic: explicit+1cost
for concurrent completion, not a local complexity reduction. No caller/shimadded.
Ready to apply overlay after018owner completed; no production019edit beforethen.

019 boundary extension: driver.Client.CloseSessionWithLease and ReleaseSessionLease
ignore the typed success flag in HTTP200 bodies. Include these two client methods
and public Session real-HTTP tests: missing/negative acknowledgment must fail and
retain ownership; partial close artifact metadata may accompany the explicit error.
This is necessary for the shared terminal result to represent an acknowledged
operation. Preserve existing404 policy. No retry engine or new protocol added.

019 acknowledgment red: all6HTTP200 missing/false cases returnednil. Driver client
now rejects missing/negative success, retains partial close metadata with error.
Actual public Session tests prove ownership retained, explicit valid retry ends
it once, and partial metadata is preserved. Final four Go race packages pass
1.076/1.309/1.076/1.042s. Final make build passed. Contract/board019 producerchecks
pass;17pending/productfalse. Scope validation: changed Go session/driver/engine/
executor packages plus BASbuild and owner tidiness;018full driver/UI proof reused,
not claimed full candidate unit qualification. No thresholds/denominators changed.
Final runtime union2files1,560→1,569lines,98functionsunchanged,282→286cyclomatic
(+4 for explicit synchronization/acknowledgment checks). Cumulative41Go files
19,447→17,834lines,691→673functions,3,102→2,955cyclomatic(-147); shared009+17
meanswideraffectednet-130. Earlier standalone session count408 included one dead
closed field missed by a spacing-sensitive edit; final review removed it before
source application. Final session407linesunchanged,36functionsunchanged,76→77.
Owner20260922-085824-c5e34b26 tidinessfailed4s; quiet120s waitcompletedonce.
1,077findings,long80/complex380/dup604/coupling12,debt33,648(-31cycle/-1,955original).
Fresh019 deployment gate again rejected currentnonterminalwork beforemake stop;
no active execution interrupted. Runtime014 remains; noowner/build/lifecyclepending.

### BAS-WORK-020 — 2026-09-22 UTC — driver close recovery ownership

RF-036/J07/J08/J18. Feedback008 reread. Driver retains neither failed cleanup
nor completed-step state; concurrent close immediately returns empty success.
Now that admission respects closing and Go honors errors/acknowledgments, repair
the driver close owner: manager + session-teardown + session types and public
route/manager tests. Keep the same Session in closing after failure, join one
in-flight close result, and retain successful teardown stages/artifact paths for
explicit retry. Failed pre-close recording/audio/SW/Playwright-trace steps must
not dispose the context or recording buffer. Verify artifact references against
actual files; video rename fallback requires a readable original source.
Page/context/CDP failure remains owned; external targets must not have their
pages/context/process closed. Idle cleanup/shutdown must attempt all sessions
without falsely reporting completion. Do not weaken the admin close boundary.
Accessibility/performance collectors' internal best-effort failure policy and
recording callback pending durability remain explicit connected RF-013/002 work;
this cycle cannot claim those unreported inner failures repaired. Per-session
recovery state is bounded by existing session capacity; process-restart recovery
and indefinitely hung Playwright promises remain unqualified.

020 red: actual HTTP trace/context faults returned200 and concurrentclose returned
before cleanup. Initial green run exposed fixture problems: host-audio probe uses
same mocked handles (reset counters after creation), and body delivery is timer-
based (wait for second route admission before releasing firstclose). These were
fixture corrections, not weakened product assertions.
Teardown now retains successful stages, validates captures, rejects failures,
keeps buffer until success, and manager keeps same closing Session/lease onfailure.
Reset refuses closing; idle/shutdown attempt every requested session and report
aggregate errors. Generic already-closed-page test now models page.isClosed=true
instead of expecting an unexplained close exception to disappear.

020 native video/trace/HAR case first returned500 without error-body assertion;
focused and full reruns passed. Cause of that initial500 not established. Added
error-body assertion and explicit double-animation-frame fixture paint. Separate
source inspection of installed Playwright Video.path shows it returns only the
initialized destination. Controlled writer createsvideo onlyon contextclose:
red ENOENT beforecontextclose proves the independent ordering defect. Repair moves
video onlyaftercontextflush; finalboundary revalidates everypublishedreference.
Real Chromium output has WebMEBML/ZIPPK/HAR1.2 signatures and0remaining sessions.

020 late-close review: successful teardown removes session before asynchronous
shared-device cleanup completes; lease guard then returned404 to overlapping
close. Newregression reproduces this; closing owner now retains Session pointer
pluspending result, so matchinglease joins andwronglease stillfails. No authrelax.
Six focused suites102tests passed before thislastcase; final103test/typechecks
pending tool sessions26173/46851. Noowner020/build/lifecycle operation pending.
Retained probes updated onlysynthetic filesystem/page seams andfailedHTTP receipt;
normalclosecontrol usesindependent manager so retainedfailure isnotmiscounted.
Original datedreceipts preserved; teardownfailure nowpasses, othergapsremain.

020 final focused103tests6suites14.755s naturalexit withdetectOpenHandles; driver
TypeScriptcheckpasses. Retained session-frame7/16,reuse7/17; knownremainingfailures
preserved. Board020prog_43cbe165-1a86-4748-996e-3d5718ecee66:17pending/productfalse;
contractpreparationpasses. Full unit,tidiness owner admitted; recover identityfrom
/tmp/bas-owner-admit-020.txt andattachquietwaitonce. No more sourcechanges duringrun.

020 full owner20260922-092430-2c566001 admitted09:24:30UTC; exactly one784s quiet
wait attached to /tmp/bas-owner-terminal-020.json. No further source edits while
it validates. Remaining work in parallel is read-only next-owner investigation.

020 full owner completed failed424s (unit420s). It exposed two stale fixtures:
Go journal-failure cleanup mocked HTTP200 with `{}` as successful close;019 now
correctly requires success=true and retains unacknowledged ownership. Driver
reset route's fake Page omitted isClosed/video;020 now exercises those real APIs.
Repair fixture fidelity and add negative close-ack control; preserve assertions
about successful cleanup and retained failed ownership. API20.387s failedonecase;
driver125suites/1515tests pass,onecase failed,2skip,323.836s and open-handle warning
after failed shutdown. CLI/UItypes passed; UI coverage28.57%vs85 remainsfailed.
Tidiness1077,long80/complex380/dup604/coupling12,debt33648 unchanged. Noownerpending.
RF040 graph/linear persistence investigation remains read-only until020revalidated.

020 fixture corrections pass: Go live-capture race1.045s, driver reset2tests2.115s
naturalexit. Explicit success=true releasesfailedjournalbrowser; missing/false
ack preservesmanagerownership andjoinederror. Finalunit20260922-093435-960e5a77
admitted09:34:35UTC; one649squietwait session80028 attached. No source edits until
terminal;021 prepared onlyin /tmp overlay ifuseful. Tidiness020 unchanged33648.

### BAS-WORK-021 — 2026-09-22 UTC — cancellation outcome persistence

RF040/J18. Feedback008 reread; recall found existing investigation/probe, source-
ledger recall degraded withdeadline exceeded; no new workflow needed. Target
owner remains Go executor. Graph ordinary/subflow/loop/set-variable persistence
uses canceled ctx; linear/terminated persistence discards cancellation without
a deadline. Consolidate outcome persistence/event payload/progress policy into
one bounded cancellation-independent owner retaining request routing values.
Keep linear-only checkpoint semantics (graph resume is not qualified). Join
storage failure with action/cancellation cause. Prove public Execute with real
filewriter and context-sensitive routed root, faulting disk, graph/linear and
nested subflow/loop controls. Preserve success, event identity and artifacts.
Scope: simple_executor.go, flow_executor.go, existingexecutor tests, retained
probe/docs. Requiredcapture failures, graphresume and transportidentity remain
separate issues. First stage is /tmp Gooverlay while020unit runs.

021 real-writer red expands necessary owner boundary: RecordStepOutcome logs
and swallows proto timeline, result manifest and index failures. Even repaired
executor cannot report unavailable evidence. Include execution-writer/file_writer.go
required-write returns and native writer tests; preserve optionalartifactpolicy
as separateRF013. Overlay onlyuntil020terminal. Eight cancellation/disk cases
reproduce missingbudget/canceledgraph and hidden diskfailure. Root observer first
included telemetry calls; corrected to observe actual RecordStepOutcome context
while retaining real filesystem writer, not weakening boundedoutcome assertion.

020 corrected finalunit20260922-093435-960e5a77 completed. API12.880s/CLI0.779s/
driver297.000s/UItypes10.587s pass. Driver126suites1516tests296.155s naturalexit;
2skip. UImergedcoverage28.57stmts/lines,30.34functions,64.96branchesvs85 remains
unitfailure; no weakening. Catalog /tmp/bas-artifacts-final-020.json andnative
/tmp/bas-unit-native-final-020.json. Bothquietwaitscompletedonce. No ownerpending.
Fresh020 deploymentgate running(session86508); source021 still only/tmpoverlay.

020 deployment completed: gate09:42:09UTC scanned3135rows(2693completed/437failed/
5exacthistoricRUNNING),driver0/recording0. Managedstop/start healthy09:46:11UTC,
builddc7fae94ed05ef12aa56d1f5a9fd02ffdb2db096ae2714bb98acdea9d6dbc5fd. API/driver/UI
healthy;oneoriginalprofilemetadata present andalloriginalsavedfields compareequal
tooriginalrollback. /tmp/bas-health-deployed-020.json,profile-api-final-020.json,
profile-verify-final-020.txt. Gate isobservational,notatomicadmissionlock;sharedtree
changes limitcandidateexactness.021remainedoverlayuntilstartfinished.

021 additional publicsyntheticcase red: pre-canceledlinearset_variable returned
success andcreatedbrowser. Cancellation guard nowbeforelinearadmission andowned
at executePlanStep forordinarygraphandloopbody. Syntheticlinear successnilGraph
nowusescommonprogresswithoutpanic. Loopparentfailurealsopersists; childfailure
cause retained. Requiredwritefaults timeline/manifest/index allred(nilerrors)
thenpasswithrealfilesystem preservation.16newcases inclcontext-awareevents,
4writercases. Overlayfourpackage races1.830/2.350/cached/1.059s pass.

021 reviewed overlay applied after020owner/lifecycle completed; bytecomparison
confirmed all6existingpaths unchanged sincebeforecopy, preserving unrelatedwork.
Callerconversion leaves intPtr unused; removeprivatehelper fromflow_utils.go
(necessaryboundaryextension, no survivingcaller). No paralleloutcomepathsremain.

021 finalsource validation passes: executorrace1.646s/writerenginecached/workflow
1.060s;previousoverlaywriter2.350s. AllAPI go test ./... andBASmakebuild pass.
Retainedexecution-api4/8(graphcancelrepaired);no historicalprobe expectationedited.
Contractpreparationpasses;boardprog_13cd5954-74b3-43af-8bfa-2f1722b2588b17pendingfalse.
Owner20260922-094905-f7781095 tidinessfailed4s,quietwaitdoneonce;1076findings,
long80/complex379/dup604/coupling12,debt33649(+1cycle/-1954original). Local4runtime
4607→4485lines,130funcunchanged,934→918cyclomatic(-16). Cumulative45unionpaths
(44currentfiles;deletedrecorderstillinbaseline),22188→20453lines,772→754func,
3667→3504cyclomatic(-163);shared009+17wideraffectednet-146. No cutoffchanges.
No owner/buildpending;live020healthy. Multi-fileatomicity/exactlyonceretry and
forcedtimeoutsfornoncooperativefilesystem remainunknown;contextbudgetisnotproof
ofinterruptibleI/O. Scope excludesoptionalcaptureerrors andgraphresume.

### BAS-WORK-022 — 2026-09-22 UTC — explicit screenshot failure verdict

RF013/J08. Feedback008 reread. Scopedsearch matchednoproviders; requiredfallback
prompt-managerdiscover returned screenshotusageactions butno applicable repair
workflow. Existingactualexecutorprobe stillfails: shouldIgnoreFailure unconditionally
turns everyfailed screenshot into Success=true, clearsfailure, skipsdeclaredretry.
Delete that policy/helper; useexisting declaredretry andcontinueOnError owner.
Explicitcontinue may allowworkflowcompletion butmustpreservefailedstep evidence.
Scope: simple_executor.go andexistingtelemetry_directive_test.go publicExecute
withrealfilewriter/storage, typed screenshot result/error, retry, optionalcontrol.
Unreportedinnercapture/storagefailures remain a separateconnectedRF013 boundary;
this cycle repairs verdicts for failures the engine actually reports. No capture
capability/productqualificationclaim. Live020remains;021readybutnotyetdeployed.

022 red24casepublicExecute matrix: requiredfailurereturnsnil,optionalworkflow
losesfailedstep,declaredretrydispatchesonlyonce. Removedunconditionalscreenshot
successoverrideandshouldIgnoreFailure; existingretry/continueowner handlesall.
24casesgreenwithrealfilewriterandmemorystoredPNG;fourpackage races3.397/cached/
cached/1.059s,makebuildpass. Boardprog_85f1c2fa-8151-41aa-87e2-13f661fa8875
17pendingfalse;contractpass,execution-api5/8(explicitscreenshotrepaired).
Firstowner02220260922-095454-de285146 tidinessfailed4s;1078findings,long80/
complex380/dup604/coupling13,debt33649. Newtestharnessitself had17cyclomatic,
21imports. Rewrite policy-name conditionals as explicitexpected-outcome table;
preserve24cases/assertions. Importsnecessaryfortypedpublicexecutor/realstorage
andPNGoracle remain; no importshuffle togamecoupling. Finaltidinessadmitted
aftertablechange;recoverIDfrom/tmp/bas-owner-admit-final-022.txt andwaitonce.

022 finaltablefixture executor race3.704s pass,24casesunchanged,cyclomatic17→7.
Finalowner20260922-095731-37153b89 tidinessfailed4s;1077,long80/complex379/
dup604/coupling13,debt33649. Bothquietwaitsdoneonce. Newcouplingfinding21imports
inexistingtestfileisdocumented,notshuffledaway. Runtime1850→1820lines,50→49func,
371→363cyclomatic(-8);cumulative44currentGofiles22188→20423lines,772→753func,
3667→3496cyclomatic(-171),shared009+17net-154. Innercapture/storagefalseack
remainRF013nextboundary;noactualbrowserfaultqualificationclaim. Live020healthy,
021/022sourcebuiltbutnotyetdeployed. No owner/build/lifecyclepending.

022 deployment gate rejected beforestop: full-history offsetpagination repeated
anexecutionidentity (/tmp/bas-deployment-gate-022.txt). This cannotestablishsafe
drain; do notinferactivecounts orrestart. Live020continues. Recheckafterusefulwork.

### BAS-WORK-023 — 2026-09-22 UTC — screenshot bytes and storage receipt

RF013/J08 plusnewRF060. Feedback008 stillactiveandpreviousverbatimreread. Recall
findscheckpointprofilework: preserveProfileNone'sexplicitdiscardpolicy; no passive
capturepolicyoverride. Readonlyinspection: sanitizer slicescompressedimagebytes
tothebudget andmutatescaller'sScreenshot/Notes; writerlogsstorefailureandaccepts
nil/incompletereceipt. Targetownerexecution-writer/file_writer.go plusexisting
writer/publicexecutor tests. Screenshotbytesareimmutable;oversizeimagesareomitted
withreason,neverbyte-truncated. Requiredexplicitscreenshotwithcollectionenabled
mustretainvalidbytesandcompleteobject/URL/size receipt orpersistfailedoutcomeand
returnerror. Passiveimagesremainoptionalwithreason;ProfileNone stilldiscardsall.
Retainbothartifactfailureandrequiredmanifest/index errors. No newstoragebackend,
imagequalityreduction,temporaryframeworkorsecondwritepath. Sourcepreservation,
PNGdecode/hash andfaultreceiptsdefineoracles. Scopedraces/allAPI/build/tidiness.

023 initial22writer receiptcases allpassafterrepair. Addedtruncated-PNGinput
showedDecodeConfigacceptsbrokenstream; fullimageDecode nowrejects beforestorage.
24writercasesplus2publicExecute storagefaultcases planned. Bench realwriter+
memoryimageStore+realmanifestfiles,1280x720 fixedfixture,3x10iterations:before
31.761/32.071/32.570ms,~113KB/op;fullvalidation37.572/38.079/38.766ms,~3.863MB/op.
Measured~18.7%medianlatencycost/+3.75MB/op; not a performancegain. Header-only
validation rejectedbecauseknowntruncatedimagepasses. No weakenedfidelity/workload.
Completecodecverification costsremainmeasureddebt; wholebrowserjourneyimpact
unmeasured. Sourcepreservationandrequiredcaptureintegritytakeprecedence over
falseacknowledgment; optimizeownerI/O/validation laterwithoutremovingoracle.

023 finalvalidation:27newcases,writer/executor/workflow/storage racespass;allAPI
tests passedbeforefinaldiagnosticrefinement andchangedpackagesrerun;finalBAS
buildpass. Owner20260922-101405-34d1a561 tidinessfailed4s;bothwaitsdoneonce.
1078findings,long80/complex380/dup604/coupling13,debt33649 unchanged. Board023
prog_df6c3a1e-8e7b-4513-ac85-2a9b2bcf4072:17pending/productfalse. Contractpasses.
Localruntime1504→1539lines,35→36functions,304→314cyclomatic(+10explicitreceipt/
bytechecks);cumulative44currentfiles22188→20458lines,772→754func,3667→3506cyclo
(-161),shared009+17widernet-144. No owner/build/lifecyclepending. Live020remains;
022restartgatepaginationinconsistent,noactive-countclaim. Next memoryreview
foundFileWriter.ForgetExecution dropsonlyconfig;results/timelinesretained. Workflow
servicealreadydefersForget;archiveingestionpersistentwriternevercallsit.

023 deploymentcompleted: gate10:16:47UTC coherent3110rows(2676completed/429failed/
5unchangedhistoricRUNNING),driver0/recording0. Managedstop/start healthy2026-09-22T10:19:19Z;
sha256:cac7d388ab175e7832912adf5bb2395834de952f92e7a64e6264979d5407f243. API/driver/UIhealthy;oneprofileandalloriginalsaved
fields verifiedagain againstoriginalrollback. Receipts health-deployed-023.json,
profile-api-final-023.json,profile-verify-final-023.txt under/tmp. Rowcountdiffers
from020observation; no deletions performedhere and no attribution inferred.
No owner/build/lifecyclepending. Cycles001–023deployed,sharedtree limitationretained.

### BAS-WORK-024 — 2026-09-22 UTC — completed execution memory ownership

NewRF061/J07. Feedback008 reread. RecallfoundBASusage/improve andpriorrefactor
records;source-ledgerrecallagainpartiallydegraded. FileWriter ownsresults/timelines/
perExecConfig maps; ForgetExecution dropsonlyconfiguration. Workflowservicealready
defersForgetafterallwrites;archiveingestionhasonepersistFrames writerpathandno
release. Extendexistingterminalowner toreleaseallthree perexecutionaccumulators;
adddeferredForgetinarchivepersistFrames(success/failure/cancel). Reuseexisting
interface, no newcache/registry/tombstone. Scopeartifact_config.go/interface.go,
archivepersistence.go,existingtests. Preservecompletedfiles andotheractiveexecution
state. Measure retainedheap withrealwriter+filesystemstore andsamefixedcohort,
not a private-map assertion. Go weak-pointer eventualcollectionisnotguaranteed;
do notuseGC-timingassertions asunit gates. Activeexecutionbudget/lateinvalidwrites
remainseparateunknown; terminalcallsite mustfollowalllegitimatewrites.

024 initialnative memorycohort16finishedexecutions x3,eachuniqueDOMwithsame
shapeandfilesystemstorage:before674606/674650/674616 retainedB/execution; after
1402/1144/1240. ScopeprocessheapafterGCwithwriterkeptalive;notGC-timingunitgate.
Archiveexit3casered(noForget)→green. ExistingForgetnowdeletesresults/timelines/
configusingexistingstringkey;archivepersistFrames deferssameowner. Completed
manifestbytes andotherliveexecution's twoentries preserved. Writer/import/workflow
races3.090/1.043/cached pass. Local3runtimefiles287→292lines,8func/43cyclomatic
unchanged. Larger64cohortbefore/aftereach3trials running(sessionoutputnext).
Firsteditassert usedwrongkeytype andmadezerochanges;failedtargetedruninspected,
canonicalstringkeypreserved inactualrepair. No benchmarkfromthatno-op counted.

024 largercohort64×3:before689404/689498/689379,after270.4/285.2/279 retained
bytes perexecution. Bothcohorts agree finishedpayloadretention removed; no speed
or liveheap attributionclaim. Scopedraces/build/contractpass;board02417pendingfalse.
Owner20260922-103012-c25ab689 failed4s,quietwaitoncecompleted. Cumulative48union/
47currentfiles22475→20750lines,780→762functions,3710→3549cyclomatic(-161),
shared009+17widernet-144. No owner/buildpending;live023healthy,024notdeployed.
RF061 discovered duplicate oforiginalRF023; consolidated issue andreceiptunder
RF023 without erasinginvestigationhistory. Currentcheckpoint condensed; dated
records preserved. Native tidiness counts in execution-memory receipt.

### BAS-WORK-025 — 2026-09-22 UTC — structured execution evidence

RF049/J24. Feedback008 reread; recall found prior BAS domain-outcome work and
existing usage programs, no applicable repair workflow. Writer's private generic
converter renders structs/pointers with fmt.Sprintf, losing raw outcome structure.
Existing typeconv owns value conversion but rounds fallback integers through
float64, unwraps data-shaped maps, and silently drops unsupported members. Extend
that existing owner with strict raw-value conversion/error propagation; keep
serialized-proto interpretation only in the existing convenience input adapter.
Delete the writer's duplicate converter and use strict conversion for artifacts,
assertions and extracted values. Typed outcome retains version/attempt/failure
and JSON field names; unsupported evidence returns an error, never debug strings
or a successful partial projection. Scope writer and typeconv existing files/
tests; public writer→disk proto→typed outcome tests plus shared conversion controls,
compiler/protoconv/executor/workflow checks. Saved malformed historical strings
remain strings and will not be fabricated into reconstructable evidence.

025 boundary extension: first test compile exposed typeconv→execution-writer
import cycle. Only two unused artifact conversion functions create that dependency;
repository-wide Go caller search found their definitions only. Remove those dead
functions/import from typeconv/timeline.go, retain its actively used data types and
retry converters. This enables writer→shared primitive owner without new package.
Initial compile failure is not behavioral red evidence; rerun after this deletion.

025 first correction passes writer/typeconv/protoconv/compiler/executor/telemetry/
workflow races (4.620/1.024/1.044/1.047/5.244/1.017/1.056s). Raw JSON oracle uses
UseNumber independently, preserves9,007,199,254,740,993 and schema/attempt/failure.
Additional nil map/list controls reproduced null→empty loss and now preserveboth.
Cycle/function/nonfinite/overflow evidence rejects before accumulator mutation.

025 measured first writer benchmark20×3:old31.84/39.93/45.79ms and~92.8KB/op;
structured59.99–61.12ms/~373KB. Focused races overlapped this initial cohort, so
latency causality is not established. Later isolated CPU sample30iterations ran
~0.66s total, mostlysyscalls; timing variability must remain visible. Allocation
increase is real for thisfixture. Necessary boundaryextension result_manifest.go:
it serializes all raw artifacts then JSON-decodes/discards them. Clone the proto
snapshot under its lock, omit artifacts before serialization, preserve full
timeline and manifest semantics. Remove unused FileWriter forwardingwrapper.
Validate output/unchanged source and samefixture projection allocation/time.

025 finalsource APIalltests/sevenpackage races/build/contractpass. Owner
20260922-104555-6dcd0052 tidinessfailed4s,quietwaitcompletedonce.1080findings,
long80/complex379/dup606/coupling14,debt33642(-7cycle/-1961original). Newtest
coupling retainedhonestly. Boardprog_3e865214-cf16-41fe-aa27-cf7f3b4d0024 all17
pending/productfalse.4runtimefiles2260→2220lines,62→60functions,515→502cyclo
(-13).51unionpaths/50currentGo:23196→21431lines,806→786func,3911→3737cyclo
(-174),shared009+17widernet-157. No threshold/source-denominator changes.

025 manifestfixture10entries×~557KB each:3×10trials old24.47/25.01/25.23ms,
33.47MB/op;new25.34/28.82/36.93µs,19.3KB/op. Fullsourceequalafterprojection.
Wholewriteproperrepeat old20.19/20.57/21.43ms,new20.70/21.65/20.70ms; no speed
claim. Allocations~93→323KB/op remaincostofstructuredfidelity. Hybridoldconverter/
newmanifest repeat preservedseparately,excludedfromoriginalcomparison.

025 heapfollowup16×3=44152/128870/44408 retainedB/execution;64×3=11182/32261/
32138. Bothcohorts totaldelta~0.7–2.1MB, supportsboundedresidual. Postbenchmark
profilecontainsmostlyruntime/descriptors butextraGCchangedobservation; liveat-
measurement profilepending session6183. No immediateleak/causeclaimfromthatprofile.

025 deploymentgate10:46:32UTC admitted3125rows(2691completed/429failed/5exact
unchangedhistoricRUNNING),driver0/recording0. Managedstopcompleted. make start
pending session37745; completeitandverifyhealth/profile next. No ownerwaitpending.
Cycles024–025 sourcebuilt; do notclaim deployeduntilhealthy. Receiptstructured-
outcome-2026-09-22.json recordscurrentlimits andallbenchmarks.

025 deployment completed healthy2026-09-22T10:50:33Z;buildsha256:10a20446d1a26b20842b967c0ed11451ee8e01815e11ca066dcfafc87dd38f2f.
API/driver/UIhealthy;oneoriginalprofilemetadataequalandallfieldscompareequal
againstoriginalrollback. Two initialprofilechecks usedwrongRPC names(404);actual
browser_automation_studio.v1.session_profiles.SessionProfilesService/List passes.
No productfailure inferredfromwrongmethod. Alltoolwaitscompleted,includingstart
37745,profile2552,liveprofile6183andsettledprofile61167. No pendingoperation.

025 memoryforensics: measurement-time inuseprofile2638.89KB includes1322KB
bytes.growSlice and672KBencoding/json.unquote, withwriteralive. AdditionalGC
same64cohortdiagnostic removesbothsites;60.38B/execution delta. Ordinarysame-GC
measurementsremainrecorded, not replacedbysettlednumber. Supportsboundedserializer
residue, not continuedaccumulatorpayloadretention. No runtimeGCworkaroundadded.
RF023andRF049receipts updated; liveproductionsoak remainsunqualified.

### BAS-WORK-026 — 2026-09-22 UTC — instruction lease admission

RF038/J17. Feedback008 reread; recall found prior BAS operation work, source-ledger
partially degraded with deadline exceeded. Public GoSession.Run sends neither
execution nor lease; driver run route ignores both and may return prior-owner
cached evidence or execute effects. Scope canonical RunInstruction/GoSession,
driver SessionManager lease lookup and actual run route before phase/cache/effect.
Missing credentials reject; stale owner/token or released lease rejects without
refreshing activity. Delayed body is checked after parse against current owner.
Preserve valid execution/recording controls and earlierphase reservation. Prove
real GoHTTP packet and actual driver route plus real Chromium effect counter.
Update driver API example; retain owner fields in fixtures. Full RF038 remains
open for other mutating routes; RF039 dynamic invocation/idempotency is next
separate boundary, not solved by lease validation alone.

Read-only review also finds unused automation/driver/playwright adapter and
RunInstructions plural sending obsolete instruction-array schema; repository-wide
imports found no production caller of that package. Do not preserve an unsigned
path in active RunInstruction for that dead adapter. Remove/replace obsolete
adapter onlyafter separate bounded caller review. E2Erecord-mode script also
uses obsolete untyped instructions and missingstartidentity; qualification not
claimed from thatscript. No source change made for either discovery yet.

026 red: GoHTTP packet lacks execution_id; six actualdriver route cases returned
200 for missing/stale/released lease or ownershiphandoff while body arrives.
Initial focused Jest command also evaluated global coverage with onlyonesuite;
behavioral failures retained, focused reruns explicitlycoveragefalse. No native
floor/configchange. First correction:38driver/nativebrowsercases pass/typespass,
Go session/driver/executor races pass; engine run fixtures failbecause their
synthetic start receipts omit lease_id. Extendexistingengine test fixture responses
with real lease metadata, preserve all response/error assertions. Scope remains
instruction ownership, not wider mutation/identity qualification.

026 finalfocused39driver/nativecases2suites4.322s naturalexit/typespass. FourGo
racespass (session1.064/driver1.029/engine1.062/executorcached). Retainedpublic
execution-api6/8:leasewire nowpasses;retry/loopidentity remainfailed. Nativeeffect
counterprovesold/releasedownership adds0clicks andvalidhandoff permitsnewclick.
Board026prog_f48ebf2a-2772-4686-9c1a-5c904a61e887 admitted;buildsession32829 and
board9816 outputpendingconsumption. Fullunit+tidiness owner20260922-110144-e89cb9f0
admitted11:01:44UTC; one742squietwait attached (recovertoolsessionfromnextrecord).
No production source edits whilethisrunvalidates. Live025healthy;026notdeployed.

026 operations:quietwait76816 active exactlyonce, admission66771 stillattached;
build32829/board9816 completedconsumed. Sourcefrozen. LocalGo1569→1574lines,
98funcunchanged,286→288cyclo;driver1444→1460lines. Cumulative50currentGo
23196→21436lines,806→786func,3911→3739cyclo(-172),shared+17widernet-155.

Read-only RF039 review: session node:index replay and separate global5minTTL
cache collide with intentional retries/loops and permit repeats aftereviction.
Potential replacement: lease-bound monotonic invocation sequence, boundedresponses
plus retained high-water mark, payload digest, terminal uncertain-effect receipt.
Start retry must preserve/restore sequence ownership (GoManager currentlyreplaces
same-session wrappers), reset mustnoterasehighwaterwithinlease, newlease mayreset.
Unexpectedpost-effectthrow mustproduce nonretryable outcome:executor alreadyhonors
Failure.Retryable=false. This isdesignhypothesis, no implementation/scopeswitch.

RF039 additional read-only constraint: executor normalizes a transport error as
retryable unless driver returns an explicit nonretryable StepFailure. Merely adding
a new invocation ID on each GoSession.Run would let a lost response repeat an
uncertain effect as a new logical attempt. Repair must classify ambiguous HTTP/
decode failures as nonretryable or reconcile the same operation before a newintent.
Existing normalizeOutcome preserves supplied Failure; declared retry respects
Retryable=false. Global idempotency cache has no other productionconsumer beyond
run/close/reset; one replacement owner can remove its timer/TTL/cache duplication.
Do not implement justUUIDs as a superficial retryfix.

026 fullownerterminal456s(unit452s):API24.336/CLI0.866/driver351.922/UItypes11.336s
pass;driver126suites1523tests351.097s naturalexit,2skip. UImergedcoverage28.57/
30.34/64.96/28.57vs85remainsfailed. Tidiness1081,long80/complex379/dup607/
coupling14,debt33632(-10cycle/-1971original). Quietwait76816completedonce;
no ownerpending. Currentretaineddriverprobe fixture nowcarriesvalidlease onnormal
requests, seedsownerfromrealGo packets, andcachehandoffcaseusescurrentnewowner
credentials so itstilltestsoldcachepayload ratherthansimplyrejectionofstalelease.
Expectedbehaviors unchanged; syntheticmiddleware scopeexplicitlyretained.

026 retaineddriver8/14 afterfixtureupdate; allremainingfailures preserved. Receipt
instruction-lease-2026-09-22.json recordsfullunit/tidiness/local+cumulativecost.
Freshdeploymentgate11:12:14UTC:2998historyrows(2560completed/433failed/5unchanged
historicRUNNING),driver0/recording0. Managedstopcompleted11:12:17UTC. make start
nowpending (toolIDrecordnext). No userdata/historydeletion performedhere; differing
historycounts notattributed. No TestGenieownerpending; do not re-admit026.

Next RF039 design constraints (read-only, not yet implemented): wire logical
invocation+attempt separately from node/index and transportsequence. Executor
allocates invocationonceperrunWithRetries; loopiterations thereforediffer and
declaredattempts incrementwithinthatscope. GoSession owns monotonictransportseq
perlease; samelease startretry/recreatedclient mustnotresetitscounter (driverstart
receiptcanreturnhighwater; GoManager shouldreusematchingactivewrapper). One driver
lease-owned boundedresponsemap+highwater can rejectevictedoldseq withoutunbounded
tombstones; resetclearsresponses butnotleasehighwater, handoffnewleaseresetsboth.
Bind admittedseq to payload and invocation/attempt. Cacheuncertainthrown outcome
asnonretryable, preserveit ontransportretry. Classify ambiguousclientHTTP/decode
failure asnonretryable instead ofmintingnewintentviaautomaticexecutor retry.
Do notsilentlyignoreoldidempotencyheaders; settleexplicitinternalprotocol and
convertallactiveclients/tests/docs when replacingglobalTTLcache. Bytebounds and
processrestart applicability needexplicitlimits; no in-memoryexactlyonceclaim.
No sourcechange forRF039 yet. Continueafter026health/profile verification.

026 deploymentcompletedhealthy2026-09-22T11:16:23Z;buildsha256:99b935baabc594de2fbc627c8a6a1509db4d5f8b16c8f5f596f5feeeee68bb2d.
API/driver/UIhealthy,driversessions0. Oneoriginalprofilemetadataequal;alloriginal
fields preserved inthree repositoryreads. Originalrollbackuntouched. Admission,
quietwait,build,lifecyclestart15541andprofile84696 completed; no pendingtool/owner.
Next RF039researchalreadyrecalled with026, no repeatdiscoveryneeded.

### BAS-WORK-027 — 2026-09-22 UTC — invocation and transport ownership

RF039/J17 withRF041 uncertain-handler boundary. Feedback008 reread. Recall026
already covers thisowner; no applicable implementationprogram. Baseline remains
two failingpublicGo retry/loopidentity probes and sixdriveridentity/evidence gaps.
Replace node:index and globalTTL replay policies with one lease-owned bounded
response map and monotonic transporthighwater. Executor allocates invocation per
logical runWithRetries call andattempt per declared retry; GoSession sequences
each transportoperation andtransportsitsmetadata. A repeated same sequence/payload
returns its retained outcome; changed payload conflicts; an evicted/reset response
never re-executes oldsequence. Rejectedbusy/stale operations cannotadvancehighwater.
Newlease resetssequence; samelease startretry/Go wrapper preserves/restoresit.

Wire protocol becomes explicit: positive safe-integer operation_sequence plus
invocation_id andattempt. Optional X-Idempotency-Key mustequal lease_id:sequence;
canonicalGo sender suppliesit. Obsolete arbitrary-header identity is rejected
explicitly, not silentlyignored. All activecallers/examples/probes converted.
Source contracts carry runtimeattempt metadata outside saved workflowdefinition
(JSONexcluded); no new userworkflowformat/runtime migration. Step outcome retains
correlation andactualattempt. Ambiguous transport/decode failure is nonretryable;
post-admission handler/telemetry exception yields retained nonretryable uncertain
outcome. Declared retry of known returned transientfailure remains allowed.

Scope: Go contracts/executor/session/manager/client/types andexisting tests; driver
run/start/leasehandoff/reset/type boundaries; delete global idempotency cache and
node-key helper/exports/calls. Replace tests of obsolete private-map mechanics
with publicroute/native effects, preserve desired duplicate-suppression controls.
Do notdelete activecapabilities or weaken thresholds. Keep phaseadmission policy:
inflight duplicate mayreceive409 andmustnot createanewintent automatically.
Boundedcacheeviction denies replay whenreceiptunavailable; no processrestart/
exactlyonce claim. Saveddata preserved. Beforecopies:/tmp/bas-before-027.

This scope is now selected; no027runtimeedits yet. Unit026/deployment/profile
verification completed, no priorowner pending. RealHTTP/publicexecutor packet
oracles plus nativeChromium counters, conflicts/eviction/reset/handoff/uncertain
effects and naturalexit checks precede ownerfullsuite. Performance/debt measured
against beforecopies; capture unknowns ratherthan narrowingtests.

027 implementation checkpoint: Go runtime6files +63lines/+1function/+12cyclomatic
(4644/172/737 ->4707/173/749). Driver affected runtime3837->3366lines (-471),
including full335line globalTTLcache deletion and node-key helper deletion;
no TSAST cyclomatic claim. Initial snapshots omitted proto/instruction.ts and
outcome/outcome-builder.ts; their before copies were reconstructed by exact
inverse of this cycle's additive runtime metadata and attempt/notes edits.
Cumulative expanded54-path union (53original and53currentfiles)24446lines/
831functions/3997cyclomatic ->22749/812/3837 (-160); shared009+17 yieldsnet-143.
No thresholds/floors/config denominators changed. Removed15 tests that asserted
literal keys, copied Map mechanics or the obsolete node-key helper; replaced
behavior with real route/native controls, including >1000-effect eviction.

Evidence: /tmp/bas-operation-go-red-027.txt and native-red-027 demonstrate
pre-repair identity/loop suppression and missing uncertainty policy. Initial
focused driver121tests5suites9.331s; expanded110tests5suites13.737s naturalexit.
Final Go races session1.061/driver1.023/engine1.058/executor3.551s. Publicexecutor
four realHTTP cases prove retry metadata, loop visits, malformed response and
lost-response no-new-operation behavior. Initial one-shot-loss fixture exited1;
final ambiguous-loss fixture keeps every retransmission lost, because an HTTP
transport may safely recover by repeating the identical packet. Original first
fixture log was overwritten; do not claim its exact assertion as retained proof.
Native Chromium post-click handler/audio-decoration throws retain same uncertain
receipt and one effect; attempt2 and invocation/operation correlation preserved.

Retained probes API8/8;driver13/14. Synthetic middleware maps all exceptions to500,
so header-lease binding probe now checks exact rejection message and unchanged
effect count; real route tests establish400. Original arbitrary global header
protocol intentionally removed; current valid headers identifylease:number.
Historical receipts remain unchanged. RF041 thrown-handler evidence gap still
fails: zero console/capture despite retained uncertain outcome. No qualification
for restart/exactlyonce, activebytebudget, diagnostic preservation orwholejourney.

Build/contract passed; board02717pendingfalse. Full unit+tidiness admittedonce
20260922-114025-ba3c12e3; quietwaitsession80099 pending. Othertool sessions consumed.
027 notdeployed; live026build/profile receipt unchanged. Next: finish owner,
sequential synthetic overhead trials, record debt/currentreceipt, freshdrain gate,
lifecycledeploy/profilecompare; then repair RF041 diagnostic loss or confirm
unlinkedlegacyadaptercallers before deleting. No external worklog/plan operation.

027 ownerterminal20260922-114025-ba3c12e3 failed399s(unit395): API31.951/CLI0.936/
driver291.276/UItypes10.730s passed. Driver125passedsuites/1526passedtests290.522s,
1suite/2testsskip; naturalexit. UI mergedcoverage remains28.57/30.34/64.96/28.57
vs85. The partialVitesttable83.62isnotmergedfloorproof. Native tidiness1085findings,
long80/complex380/dup609/coupling15,debt33638 (+6cycle/-1965original). PublicHTTP
regression adds15complexity test andexistingfile22imports; not moved/split togame
thresholds. Localnewcode canincrease these readouts despite removal ofsecondcache
policy; cumulativeGo/owner/netline reductions reported independently.

Corrected equalserialization syntheticbenchmark (5000distinctops,3trials;5000effects
andrepeatcontrol allpass):before15.493/12.416/12.817microseconds/op;after19.490/
16.630/16.084. Median+3.813microseconds (~29.7%); added canonicalpayloadsort/hash
andimmutable receipt handling are new required bindingwork. No browserlatencygain
or bandrelaxation. Initialbaseline skippedresponseJSONserialization,excluded;
firstcorrection lackedbaseline res.end andfailed; finalbefore/after-027json are
comparable. NegativeheapdeltasreflectVM/compilationgarbage; no qualifiedretained
memoryclaim. Sourcecopies/scripts kept /tmp forreproduction, notnewproductioncode.

Gate0272026-09-22T11:49:50.977513Z:2998rows,2560completed/433failed/5exactunchanged
historicalRUNNING,driver0/recording0; freshgenerationchecked,observationalnotatomic.
make stopcompleted;make startsession82886 pending. Oneoriginalprofileuntouched;
postdeployhealth/profilecompare next. Ownerquietwait80099 consumed, no ownerpending.
Recall028 /tmp/bas-failure-evidence-recall-028.txt:70hits,priorcapturepolicywork,
no matchinghandlerfailure implementationprogram; source-ledgeragent-memory/scopes
requests timedout butothercorporaavailable. Preservepassivetelemetrypolicy.
Read-only RF041 confirms catchdisposes/rethrowsbeforefailurecapture; telemetry
collect/rejectedmetrics also lackswhole-pipelinefinally. No028sourceedits.

027 health/profileverificationcomplete2026-09-22T11:52:38Z;build
sha256:dca16c2b2fb6f2d38c09ae2df27f8ec84f430bbb8b7ec2b263c84dd73d08cfc5.
API/driver/UI healthy. ProfileAPI metadataequal026; everyoriginalsavedfield equals
untouchedrollback inthree repositoryreads (firstkeyringread1215ms; then0.35/0.29ms,
notacomparablelatencytrial). Start82886/profile1572 consumed. No pendingowner.

Read-only reviewfoundRF062: native027zero driver-filefindings althoughlargeTS
modulesexist. ActualsharedLanguageDetector scansonlyapi/ui/src/cli; usedbylight
scanner,detailedmetrics,handlers. RecordedasRF062, notRF061retiredduplicate023.
Native027debtincrease/overallreduction honestbutdoesnotmeasureTSdriverremoval.
Nextboundedownerinvestigation/necessaryextension willrepairinventoryusingexisting
sharedsurfaceauthority andretainexpandedbefore/currentdenominators. No028edits.


### BAS-WORK-028 — 2026-09-22 UTC — complete maintainability inventory

RF062/W2 measurement gap; bounded necessary shared-owner repair underFB008.
Feedback008 reread. Recall /tmp/bas-tidiness-inventory-recall-028.txt:68hits,
no applicable sourceinventoryrepair program; two source-ledgercorpora timeouts,
otherprovidersavailable. FirstCodeFactscall used unprefixedtarget andfailedpath
resolution; corrected scenario:browser-automation-studio returnsdeclaredsurface
receipt /tmp/bas-driver-surfaces-028.json. No capabilityabsenceinferred.

Two causes: shared api-core/pathfilter excludes literalplaywright-driver asdata;
TidinessManager LanguageDetector separatelyhardcodesapi/ui/src/cli. Its existing
filemetrics inventory alreadywalkstargetrootwithsharedfilters. Earlierintentto
addCodeFactsdiscoverytoruntime isrevised: CodeFacts confirmscomponentownership,
butmaintainabilityalso scanssupportingsourceoutsidecomponents. Reuseoneexisting
localinventoryforlanguagegroups andallmetricfamilies, remove duplicatewalks;
no new remote dependency or privatehardcodedsidecarlist. Targetdocsupdatedfirst.

Scopeextension: packages/api-core/pathfilter/{pathfilter.go,pathfilter_test.go};
scenarios/tidiness-manager/api/{language_detector.go,language_detector_test.go,
light_scanner.go,handlers.go,validation_connect_test.go} andownerdocs. Clean at
entry; beforecopies /tmp/bas-before-028. No dependency/schema/productdatachange.
Redgate: publicnativevalidation mustreportlength/couplingforPlaywrightandcustom
workerTS sources, preservevendor/build/dataexclusion; detectorusesexactsource
inventory. Sharedfilter source-directory regression. Nativeexpandedmetricswill
rise; preserveoldreceipts/ratchets andremeasureequalbefore/currentcoverage.
Then focusedownerchecks,scopedTestGenieownerphases, lifecycleownerdeployment,
BASnative-tidiness receipt andboard. No028runtimeeditsyet. RF041 remainspending.

028 redconfirmed: sharedSkipDirwronglytrue; nativevalidation missesbothdriverlength/
coupling andcustomworkercoupling; detectorreturnsnilTS. Focusedfirstrepairpasses
7.889s. During review, initial singleinventoryrefactor narrowed incremental-mode
language metrics to changedfiles. Preserve existingfulllanguage semantics by
collectingonefullinventory andfilteringonly returnedincrementalfilemetrics;
removeits duplicatewalk. Add realPG publicScanWithOptions regression inexisting
light_scanner_integration_test.go (beforecopyextended), andupdatepathfilterdoc.go.
No candidatepublished; this is correctionofownin-progressrefactor, notnewissue.

028 implementation: removedsharedplaywright-driverdirectoryexclusion; language
classification derivesfromexistingfilteredfilemetrics; nativefindingsreuseits
filelistsratherthanrescanningperlanguage. Incrementalmode deriveschangedoutputs
fromonecompleteinventory whilewholelanguageanalysis remainscomplete. Deleted
secondincrementalwalk, perlanguagefileslookup, excluded-fileforwarder androot/ext
wrappers. Shared5Go files1909lines/63functions/318cyclomatic ->1710/58/288
(-199lines/-5functions/-30cyclomatic). Includespathfilterdoc.go; noexporteddebt.
BASGo cumulative -160, prior shared009+17 and028-30 => affectedwidernet-173.
Sourcebefore /tmp/bas-before-028;metrics /tmp/bas-complexity-shared-028.json.

Red: /tmp/bas-filter-red-028.txt; /tmp/bas-inventory-red-028.txt. Incremental
correctionactualred /tmp/bas-inventory-incremental-red-final-028.txt showszero
changedfilesbutmissinglanguagecoverage. First attemptusedwrongcwd; nextfixture
storedlocaltimeintotimestamp-without-timezoneandincorrectlymarkedfileschanged;
correctedfixtureusesUTCfuturetimestamp. No productiontimestampchange. Final
focusedrace owner7.322s andsharedfilter1.013s pass. Beforecopyextendedfor existing
incrementaltest andpathfilterdoc. OriginalHEADBASarchive retained at
/tmp/bas-expanded-original-028/scenarios/browser-automation-studio (2199tracked
files/19MB); not yet scannedbyrepairedprovider. TypeScriptcomplexity analyzer
explicitlyunsupported; applicablelength/duplication/coupling coverage is repaired,
not a TSASTcomplexityclaim. Existinganalyzererror/reportlimitsremainunqualified.

Fullownerunit+tidiness20260922-120534-16d6f5c6 admittedonce. QuietwaitONCE pending
(/tmp/bas-inventory-owner-wait-028.json; keep returned toolsession); admission
session16380 maystillstream. Build16368completedlog buttoolnotyetconsumed;
board71199 pendingconsumption. Noownerlifecyclepublicationyet; BAS remainshealthy
027. Nextextractownerreceipt, deploytidinessownerthroughmake, scanexpandedoriginal
andcurrentusingfreshnativeprovider, keepoldnarrowmetrics andunchangedratchets.

028 owner run is terminal (failed, 70 s); admission/wait/build/board sessions
consumed. Unit native: 29 findings, 11 blocking. API exceeded unchanged 60 s
no-output watchdog; CLI/UI tests/UI types passed. Ten other blockers are existing
Tidiness UI coverage/import-policy projection gaps. Tidiness phase passed using
the older live provider and cannot certify the new inventory.

The diagnostic coverage command completed naturally in 91.909 s, coverage 72.0%,
with eight trimpath-broken fixture failures and one repository executable-budget
failure (864 observed, 472 declared). No SIGQUIT stack exists; the process had
finished before the attempted signal. Earlier inference of startup/init hang was
wrong: multi-package Go output was buffered. Do not classify this as a crash.

Necessary owner extension BAS-RF-063: Unit Health's Go evidence adapter rejects
the absolute executable path supplied by its planner, omitting existing streaming
JSON/fresh-execution instrumentation. Fix that adapter and exercise the actual
resolved command with its native evidence test; no timeout/floor changes. Also
repair Tidiness's existing liveRepoContract fixture to resolve from the package
working directory, which survives -trimpath. Before copies retained in028.
Repository-wide executable debt and unrelated Tidiness UI policy drift remain
visible; this cycle does not authorize deleting hundreds of other owners' outputs.

028 extension validation: resolved-executable native regression failed before
repair (canonical Go command unsupported); after repair gotest and validation
races pass in1.270/2.821s. Preserve JSON/-count=1 instrumentation, selected binary,
coverage flags; accept native go/go.exe basenames. Owner builds pass. Original
pathfilter overlay also observes864executables vs472, proving that failure
predates028. Keep the budget untouched. Fixture repair removes runtime.Caller
source-path dependence; full JSON diagnostic running with only executable-budget
failure observed so far.

028 shared-owner lifecycle gate: no campaigns, no API child work processes;
only established connections for ports16792/15317 belong to runtime supervisor
health checks (PID3159738). Existing server shutdown drains requests. Gate is
observational, not atomic. Restart Tidiness and Unit Health through make after
focused/build validation; no BAS restart or product-data mutation is required.

028 full streaming API diagnostic terminal90.429s:718passes,14skips,1failure
(repository executable budget, reproduced unchanged with original filter).
No remaining fixture assertion failures. Evidence /tmp/bas-inventory-api-final-028.jsonl.
Restart commands are still active; 12:17:50 health reads showed old uptimes/builds
and therefore do not establish publication. Preserve tool sessions71385/62980.

028 shared restarts completed healthy: Tidiness12:19:39Z build3ffde178...;
UnitHealth12:20:09Z build2971b6b6.... CodeFacts rebuild-only preserved running
shared service; UnitHealth lifecycle also refreshed QualityHealth dependency.
No BAS restart. All restart sessions consumed. Expanded native scans retained
in /tmp/bas-expanded-{original,current}-028.json: original1131findings,
105long/384complexity/628duplication/13coupling/debt35603; current1121,
103/387/613/17/debt33666. Each has23driver long and1driver coupling finding.
Observed scan wall times5.266/4.764s are single reads, not a performance claim.

RF064 discovered: JS duplication adapter ignores inventory and hardcodesui/src;
malformed stdout becomesempty, and skipped/error analyzers are not coverage
receipts. Expanded totals therefore qualify supported observations only, not
whole-codebase TScomplexity/duplication. File tracking owns this out-of-slice
finding underFB008; no external journal/bug write. No dependency install attempted.

028 final Tidiness owner run20260922-122015-cbf7d8e5 admitted; quietwaitONCE
active tool81628. UnitHealth admission was rejected by caller preview capacity,
so no UnitHealth run exists; retry only after the current owner run completes.
No duplicate admission. Extension execution.go169lines/3functions/54cyclomatic
->170/3/55 (+1). Shared028 net -198lines/-5functions/-29cyclomatic;
cumulative BAS -160 +shared00917 +shared028(-29) = -172 affected net.

028 completed validation: Tidiness20260922-122015-cbf7d8e5 failedunit121s,
passedtidiness2s. API113.115s now reports test_failure; CLI0.863/UI4.178/
UItypes2.131spass. UnitHealth20260922-122233-b1f4bc2d passedunit12s with0findings
(API4.306/CLI0.625/UIcoverage3.891/UItypes2.767), failedtidiness2s. BASnative
20260922-122311-0fd78914 failed with expanded1121findings/debt33666, matching
directcurrent. All admitted quietwaits consumed; admission rejected before
UnitHealthrun was retried only after Tidiness terminal. Board final028:
prog_112922f0-8ab9-44bd-a6e9-fab640816076,17pending/productfalse. Receipt
maintainability-inventory-2026-09-22.json retains hashes and scope limitations.
No pending validation/lifecycle. RF064 remains open; next BAS behavior workRF041.

### BAS-WORK-029 — 2026-09-22 UTC — instruction failure diagnostics

RF041/J18 bounded BAS behavior repair. FB008 reread; recall already retained at
/tmp/bas-failure-evidence-recall-028.txt, no applicable existing repair workflow.
Hypothesis: handler throws bypass capture and dispose buffered diagnostics;
metrics/capture/build failures also bypass the sole success-path disposal.
Target architecture updated first. Before copies /tmp/bas-before-029 retain
executor/orchestrator and existing unit/native test files. No product data or
schema change. Add native real-click failure evidence/replay oracles and cleanup
fault controls, then simplify to one final ownership release. Preserve optional
capture rules and existing native timeout behavior; do not introduce a racing
background capture or claim hang qualification. Shared028 remains deployed.

029 initial red /tmp/bas-failure-evidence-red-029.txt:7failed/29passed across3suites.
Implementation: one executor finally owns disposal; handler exceptions become
nonretryable uncertain results before captures. Capture channels fail independently
and retain errors in outcome notes; metrics observers use existing safeInvoke.
Known handler success plus unexpected capture error becomes nonretryable evidence
failure; preexisting handler failure retains its cause. Typecheck passes. First
focused after:35passed/1nativeconsolefailure. Screenshots/DOM exist in thatfailure;
page-evaluate andmainworldonclick logs both absent under default Rebrowser.
Diagnostic REBROWSER_PATCHES_RUNTIME_FIX_MODE=0 passes same2nativefailure cases
(/tmp/bas-failure-native-runtime-control-029.txt). Installed crPage.js:429 guards
Runtime.enable onthatsetting. RF065 recorded; no production env/dependencychange.
Retained execution-driver producer nowpasses14/14 (/tmp/bas-failure-retained-029.json).
Pending expansion to console ownership; beforecopies /tmp/bas-before-029.

029 necessary RF065 extension: telemetry/collector.ts and existing collector
unit tests; synthetic execution probe must model acknowledged CDP console events.
Before copies extended. Existing browser owner only launches/connects Chromium;
no alternate-engine capability is removed. Use a dedicated per-instruction CDP
console source, awaited startup/detach, no page-script wrapper or process-wide
patch override. Runtime observation is necessarily enabled during requested
console capture; anti-detection/worker/OOPIF implications are unqualified and must
remain explicit. Recall /tmp/bas-console-recall-029.txt returned63hits; no existing
console-provider repair program. No dependency or native browser patch edited.

029 default-mode native console repair passes: dedicated CDP listener consumes
Runtime.consoleAPICalled, filters pre-window history, preserves primitive text
without evaluating remote object getters, and detaches after initialization or
execution failures. Startup/cleanup are awaited; delayed attachment disposal
remains owned. Old Page console bridge removed, not retained as a fallback.
Native fixture proves real onclick console + PNG/DOM, same immutable failed
receipt on retransmission, one click, and detached CDP session rejecting later
commands. Focused205tests/16suites9.221s naturalexit, types pass. One test-edit
mistake made describe async; fixed before final expanded run, no production fault.
Retained driver producer updated only its synthetic CDP interface and nowpasses
14/14 with zero remaining console sessions/listeners. Beforecopiesextended.
Affected3runtimeTS files1012->1016lines(+4); no TSAST complexity or latency gain
claim. Existing capture helpers can still return undefined for native capture
failures; worker/OOPIF/transport hangs and anti-detection impact are unqualified.
Fullownerunit+tidiness admission and makebuild pending; BAS029 not deployed.

029 build passed; contract preparation valid/productfalse. Board
prog_f1cd54e3-0e1d-460e-a246-04074d7c7956:17pending/productfalse. Fullowner
20260922-124226-6f1b08ac admitted once, quietwaitONCE tool27351 pending;
admission24111 still available. No BAS lifecycle publication yet.

Native collector setup+acknowledged-dispose benchmark:3trials x50samples after3
warmups, original medians0.11755/0.05047/0.05171ms; current0.87231/0.84458/
0.77651ms. Median-of-medians delta+0.79287ms. This is expected added attachment/
Runtime-enable/detach work, not equal functionality (original default has no
console events) or whole-workflow latency. TypeScript modules are actual before/
current with identical constant/normalizer seams; no events emitted. Firstbench
assumed per-module dist files, but build is bundled; that failed import is excluded.
Final /tmp/bas-console-cost-final-029.json; script /tmp/browser-automation-studio/
console-cost-029.cjs. No threshold or timeout changed; no speedup claim.

029 owner terminal440s (unit435/tidiness5):API31.564/CLI1.987/driver332.436/
UItypes10.353spass;125driversuites1532tests331.72s naturalexit,1suite2testsskip.
UI56.113sfailsunchangedmergedcoverage28.57/30.34/64.96/28.57vs85. Native expanded
1121findings,103long/387complexity/613duplication/17coupling/debt33666 unchanged
from028. Fullownerreceipt /tmp/bas-owner-findings-029.json; no assertion masked.
All owner/admission/build/board tool sessions consumed.

029 deployment gate12:50:59UTC rejectedstop:2834historyrows,
2433completed/395failed/6RUNNING, including new currentexecution
630342bc-f422-4760-b2b4-79d677abb02e plus5unchangedhistoricalorphans. Driver1session,
0recordings. No stop was performed; livebuildremains027dca16c2b.... Historycount
changed since027; no attribution inferred and no history/profiledatawritten here.
Receipt /tmp/bas-drain-deferred-029.json. Recheck only after useful independent
work; never kill current execution to publish. Cycle029candidate evidence updated.

Read-only next-slice review: legacyGoDriver adapters have no repository Goimports;
38qualifiedreferences allinside adapters/tests. Alias-aware audit found only
Point tokens outside them, which inspection resolves to contracts.Point (not
interface.go Point). /tmp/bas-legacy-driver-alias-audit-030.json and
/tmp/bas-legacy-driver-symbols-030.txt retain evidence. No030sourceeditsyet.

### BAS-WORK-030 — 2026-09-22 UTC — retire unused Go driver model

RF066 bounded maintenance task while029publication waits for active browser work.
FB008 reread; recall /tmp/bas-legacy-driver-recall-030.txt. No applicable replacement
workflow. Targetdocs updated before source. Full repository search:52imports of
maintained driver package, none of oldadapterpackages; two outsideAPI are retained
profile probes and do not use retired symbols. Alias-aware reference audit's only
apparent non-adapter Point uses are actually contracts.Point. All38qualified
legacyinterface references are inadapter/stub/tests. Plural transport and request
wrapper have onlyadaptercallers; typed request wrapper also unused. Beforecopies
/tmp/bas-before-030 preserve current client/types including prior cycle edits.
Retire these unused packages/types/transport and7mechanicaltests, with active
GoSession/client/executor/recording and navigator behavior unchanged. No new test
needed to mirror deleted dead declarations; use existing public behavioral tests,
Go races/build and scoped native owner validation. No product-data mutation.
Cycle029remainsvalidatedbutnotdeployed; no active testowner before030edits.

030 removal implemented:3unused runtime files +their7mechanicaltests retired;
obsolete plural transport andbothunused request wrappers removed frommaintained
client/types. No caller conversion needed; complete symbol/importinventoryshows
no active caller. Five runtimepaths2736lines/92functions/301cyclomatic ->
2files1803/63/215: -933lines/-29functions/-86cyclomatic. Expanded57-pathGo union:
original56files25362lines/859functions/4082cyclomatic ->current53files22732/811/
3836, BAS -246; prior shared009+17/028-29 yieldsnet-258. No exporteddebt.
Metrics /tmp/bas-complexity-{local,cumulative}-030.json; originalHEAD retained.
Focused driver/session/engine/executor races1.024/1.067/1.060/3.695spass.

030 validation uses discovered UnitHealth workspace filter: canonical execution
withcoverage forapi+cli, preserving029fullpasseddriver result because driver
source is unchanged. UnitHealth tool84972; nativeTestGenietidiness admission74462;
makebuild67472. No changedfloor, fast-test-only switch or weakenedrolepolicy.
This is a scoped receipt, not a fresh whole-product unit verdict. BAS remains027
live until a fresh quiet gate permits publication of029+030 together.

030 verification terminal: canonical scoped API36.090s andCLI2.254s passwith
coverage. Provideroverallreports2falsemissingroles (excludeddriver/UI), RF067.
No role waiver/floor change. Tidinessrun20260922-125645-c641df05 terminalfailed:
1117findings,103long/386complexity/610duplication/17coupling/debt33616
(-50cycle/-1987expandedoriginal). Buildpasses. All030tools consumed; board
receipt /tmp/bas-board-030.txt. SourceprogramNo nativecapabilitytestsremoved;
only7mechanicaltestsofthedeadadapterswereeliminated.

030 deployment gate rejected beforestop because offsetpagination repeated anID
whilehistorychanged (/tmp/bas-drain-stop-attempt-030.txt). No stopperformed;
follow-updriverhealth shows10sessions,0recordings, confirming activework. Do not
relax the consistency guard or terminateothers' browsers. 029+030candidate remains
readyforquietpublication, with UIcoverage/debt limits explicit. Next independent
repairRF067 belongsinUnitHealth; no pendingTestGenie/build/lifecycle atthispoint.

## 031 — required-role evidence under workspace selection (2026-09-22 UTC)

RF067 necessary scope extension: Unit Health owns the false missing-role findings
in BAS030's canonical API/CLI validation. Target architecture is updated before
source. Affected owner paths: api/internal/validation/{service,plan}.go and their
existing tests. Preserve complete discovery for global policy and selected
inventory for planning/analysis; do not weaken roles, floors, or waivers.
Hypothesis: the selector erased observed UI/driver roles before policy resolution.
Discriminating checks: present-but-excluded role causes no missing-role finding;
actually missing role still fails; unknown selector schedules no commands.
Recall /tmp/bas-workspace-policy-recall-031.txt returned results with degraded
source-ledger providers; no suitable program repairs this implementation boundary.
Before copies /tmp/bas-before-031 retain shared-tree source. Focused race checks,
owner unit/tidiness phases, and native BAS API/CLI scoped validation will qualify
the change. BAS029/030 publication remains deferred while browser work is active.

031 result: red regression reproduced excluded CLI/UI as missing; complete
discovery fix passes validation race2.282s and makebuild. Unit Health published
healthy13:11:24UTC. Owner20260922-131130-e1ccd657: unit13s passed/zero findings,
tidiness2s failed existing budget. Canonical BAS API35.493s/CLI2.069s pass with
coverage, zero required-role findings,55warnings; no floor or profile change.
Runtime two files1639->1641lines,39functions/272cyclomatic unchanged; affected
cumulativeGo delta remains-258. Board17pending/productfalse.
BAS029/030 finally published after gate13:10:32UTC found zero sessions/recordings
and exact historical-five orphans; clean stop/start healthy13:12:35UTC. Metadata
and complete original saved-profile identity preserved. Both deployments and all
validation sessions consumed. Receipt scoped-unit-policy-2026-09-22.json.

## 032 — execution-history filtering and pagination (2026-09-22 UTC)

RF054: handler ignores status, total is page length, has_more guesses from page
fullness. Repair the repository/workflow/handler boundary and migrate all list
callers to one query, removing duplicate status-list methods and retention's
optional fallback. Count/page share the routed transaction; preserve existing
retention selection budgets, execution history and saved results. Public bounds
follow the existing proto contract. Target architecture updated before source.
Recall /tmp/bas-execution-filter-recall-032.txt returned BAS usage programs, no
query-repair workflow; source-ledger providers degraded. Before snapshots are in
/tmp/bas-before-032. Real temporary SQLite plus public Connect requests will
check intersecting filters, equal-time ordering, middle/final/empty pages, total,
has_more, default/bounded limits and invalid requests. Existing retention and
recovery regressions remain required; no live history mutations are authorized
by this read-path repair.

032 caller audit found the global UI asks for200 rows and workflow histories
expect an unbounded list when limit is omitted. Necessary boundary extension to
existing ui/src/domains/executions/services/executionApi.{ts,test.ts}: aggregate
bounded public pages, preserve exportability and caller limits, reject broken
non-advancing/duplicate pagination rather than return incomplete history. No
new view/state owner. Initial total bounds the aggregate under growing history;
multiple requests still do not provide a snapshot. Add routed-pool read checks
to the existing database connection regression before publishing.

032 result so far: six race packages pass; focusedUI8/8 and types pass. Canonical
API39.743s/CLI2.297s pass; UI55.562s fails unchanged85% coverage requirement
(28.59/30.34/65.15/28.59); UItypes10.626s pass. Makebuild passes. Tidiness
20260922-132426-85e2b06f terminalfailed, quietwait consumed; debt33524(-92cycle).
Final metric includes compiled Go test helpers: -47lines/-4functions/0cyclomatic
plus UI+25lines. Expanded65original/62currentfiles30110/27433lines and
4952/4706cyclomatic; BAS-246 + shared-12 => affectednet-258. Domain-only
-7cyclomatic is offset by mock updates; do not report it as overall cycle gain.
Quiet matched-size query medians181.390->253.381us (+71.991/~39.69%) after
coverage completed. Extra count+read transaction+tie sort supply previously
missing behavior; no whole-journey speed claim. Initial test fixture pointer typo
was fixed before valid red receipt. Gate13:27:14UTC admitted cleanstop; start
session30071 pending at checkpoint. Boardprog_94bdf9ed-0f67-45c8-8581-6e0b383f6b16
17pending/productfalse. Receipt execution-query-2026-09-22.json.

032 publication complete: healthy build27b9cd6d7734e44874e92bb6e3213ffb3dc83873696d10f9433f34e642a7c13b
at13:31:09UTC. Native read verifies pending0/running5, exact historical IDs,
total2833 for page1 and empty-offset page, default50. Saved profile metadata
equals031; three complete identity reads preserve original rollback fields.
No pending lifecycle/build/test waits. Recall033 finished; no033source edits.

## 033 — recording acknowledgement boundary, investigation (2026-09-22 UTC)

RF002 persists beyond the committed Go journal repair014/015. Source audit finds:
browser send/retry treats any resolved fetch as success, removes pending entries
by timestamp, drops old/overflow/retried entries, and clears storage during
recovery/reinjection; raw events lack stable identity across retransmission.
Driver event-route invokes a void callback and returns200; pipeline fire-and-
forgets async entry delivery; callback circuit skips/failures return success;
stop transitions ready without joining deliveries; local buffer evicts and clear
reads erase before delivery. Current Go ingress commits before200, so it is not
the remaining false-ack source. Fix must include stable delivery identity,
acknowledgement propagation, pending retention and stop/close/reset ownership,
not just retry HTTP. First add actual Chromium fault regressions to existing
tests/integration/pipeline-e2e.test.ts for thrown delivery and delayed delivery
at stop. No033runtime source edits yet; live032 remains healthy. Recall finished
/tmp/bas-recording-delivery-recall-033.txt; no replacement workflow found.

033 native investigation result: all three new desired-behavior tests fail under
actual Chromium in1.550s with natural exit (seven unrelated cases filtered).
Raw click has no retry identity; rejected async callback still yields HTTP200;
stop reports success before controlled commit release. Initial two-case1.408s
receipt retained. No runtime changes; live032 remains healthy. Target architecture
now records delivery/identity/retention invariants before implementation. Receipt
recording-delivery-investigation-2026-09-22.json. All test/tool sessions consumed;
no pending operations. Next implement the complete acknowledgement owner boundary
without weakening the three native assertions. Retain no-callback/pull mode, and
keep driver/browser-crash guarantees unknown until directly qualified.

033 implementation boundary prepared: existing browser script/init generator,
recording buffer/event route/initializer/pipeline, raw-event conversion, callback
and recording routes, session reset/teardown, plus Go driver/pull ingress where
explicit acknowledgement must follow persistence. Before copies under
/tmp/bas-before-033 preserve shared audio/session work. Planned replacement uses
one browser pending queue (mirrored to sessionStorage), one driver entry owner
with stable identity/sequence and joined delivery, and no silent circuit skips.
Current-frame stop handshake must flush input and wait for delivery; failed
stop retains ownership for retry. Pull mode needs an explicit acknowledgement
after the caller commits, with Go ingress using the existing journal. No runtime
replacement is deployable until all three native fault cases and affected
retention/close/reset/pull cases pass.

033 interim edits (not deployable): browser now retains one serial pending queue
with UUIDs, checks HTTP/application acknowledgement, and starts dormant until an
acknowledged control handshake. Native MessageChannel probe passed on installed
default Rebrowser. The browser retry case now passes; async-route and stop cases
still fail (interim1pass/2fail,2.635s). Replaced buffer's four maps with one bounded
entry/delivery/receipt owner; pending entries cannot be evicted, cleared or
removed. Buffer integration, raw identity/sequence reuse, async route/callback
propagation, stop/reset/close flush, pull acknowledgement, and tests are still
required. New buffer APIs are not yet wired. No pending tool/test processes.

033 joined-delivery checkpoint: driver admission now awaits one ordered buffer drain, retains stable browser ID/driver sequence, and propagates callback errors. The silent callback circuit and duplicate route buffer ownership are removed. Native fault cases now 3/3 pass (7 filtered, 3.233s; /tmp/bas-recording-ack-wired-033.txt). Stop joins the browser control handshake and driver delivery; failures retain ownership. Not deployable: pull acknowledgements, reset/close guards and affected-suite migration remain. Typecheck reports only unused handleError pending error-observer wiring. Original native 3-failure receipt remains. No pending runs or lifecycle operations.

033 integration checkpoint: native fault3/3 pass; recording suites163/163 and guard suites148/148 pass. Explicit pull ACK follows Go journal commit; reset/close retain pending entries. Stop joins an admitted start and retains terminal count/time. New native rejection/stop retry, final debounce flush and overlapping start-stop cases3/3 pass1.226s. The first retry test asserted the inner journal text instead of propagated HTTP500, left its fixture pending and contaminated later cases (121.63s retained); corrected assertion and finally recovery yield natural exit. Go driver/session/handlers/live-capture/recording/persistence/sidecar race tests and driver types pass. Source still not deployed. Canonical Test Genie unit,tidiness admitted via tool71346, log /tmp/bas-recording-owner-admission-033.txt; must consume admission and attach one quiet wait. Runtime delta currently about450 fewer lines; measured Go+4cyclomatic/+3functions, cumulative BAS-242/shared-12 = net-254. Browser/driver crash durability and full journeys remain unverified. Next inspect owner regressions, measure final costs/deltas, critique bounded receipt/overflow semantics and gate publication.

033 canonical owner run20260922-142106-ee94c0d4 admitted14:21:06UTC; admission tool71346 consumed. One quiet wait launched with timeout798; stdout /tmp/bas-recording-owner-033.json. No polling or re-admission.

033 wait owner: tool95803 (pid4160848) is attached once to Test Genie20260922-142106-ee94c0d4. Do not re-admit or create another waiter; resume that session. Setpoint read tool20038 is pending. Source audit still does not qualify browser overflow recovery, cross-origin navigation during outage, process death, or replay after committed receipt eviction; retain these as explicit next fault probes under RF002/RF022.

033 adversarial finding while canonical suites run: a new native slow-click/cross-origin-navigation case fails0.991s; stop succeeds but the queued target navigation is absent. Serial browser dispatch stranded that observation in the old origin. Retained /tmp/bas-recording-cross-origin-red-033.txt. Replacing only browser dispatch with one bounded pending owner and per-entry joined transmission; driver retains ordered commit. This source change postdates canonical admission, so that run alone cannot qualify the amended case. Driver/process crash and undispatched page-unload evidence remain distinct unknowns. Publication deferred for this demonstrated assertion, not stopped goal.

033 cross-origin repair: per-entry immediate browser transmission retains the same bounded pending queue and matching receipts; the driver alone serializes commits. The fresh cross-origin case passes with the five prior adverse cases (6pass/7filtered,2.499s; /tmp/bas-recording-cross-origin-033.txt). Test Genie quiet wait95803 remains active; canonical source snapshot is mutable and the amended native case is separately attributed. Setpoint prog_145c2fe8-51db-426b-ac78-8bb2c3adf58f remains17pending/productfalse.

033 activation boundary finding: a dynamic cross-origin frame added after start missed its click (native1fail/13filtered,1.684s; /tmp/bas-recording-dynamic-frame-red-033.txt). Dormant initialization requires an owner for every new document. Replacing separate main-page-load and one-shot new-page-load policies with one tracked page/frame listener owner; stop joins pending activations before deactivating frames so late callbacks cannot reactivate capture. Same current/future-page behavior and frame case require native revalidation. No deployment yet.

033 canonical run20260922-142106-ee94c0d4 failed427s (unit422/tidiness5); wait95803 consumed. UnitHealthuh-20260922-142107-4cbf44497c92efc5087e9749554fb398: API38.272s failed real AI suggestion category assertion, independently reproduced9.564s and registered RF068; CLI2.155s pass; driver307.012s failed35/1550cases across5suites (1513pass/2skip), including opaque-origin script init and changed stop mock contracts; UI61.297s fails unchanged merged coverage28.59/30.34/65.15/28.59 vs85; UItypes10.738s pass. Tidiness1115findings/103long/387complexity/607dup/17coupling/debt33540(+16vs032); source mutability and subsequent frame changes mean this is interim evidence. Raw/native receipts under /tmp/bas-recording-{unit,tidiness}-native-033.json.

033 frame diagnostic: default installed Rebrowser childFrame.evaluate('location.href') returns parent localhost URL while childFrame.url() is127.0.0.1. Underlying default addBinding context discovery is suspect; do not globally disable Runtime fix. Testing the installed alwaysIsolated mode as a supported context-routing alternative, then revalidate native coverage and preserve current explicit user configuration. New unified frame listener owner is source-only. No runtime033 publication yet.

033 frame repair proof: native14/14 pass8.166s and TypeScript clean. WindowProxy traversal sends the acknowledged control directly to every current descendant document from the known page context; no SDK/private frame IDs or Runtime-fix disablement. One tracked page/frame listener owner handles dynamic frames, existing tabs and future pages; stop joins admitted activation work before deactivation. Removed the old main-page load/new-page one-shot policies. Default Rebrowser childFrame.evaluate wrong-document behavior remains a separate general execution limitation under RF004/RF030; alwaysIsolated experiment timed out and was rejected, no environment mode changed. Owner regressions for opaque-origin selector init and retained stop receipt29/29 pass2.044s. Next scoped full driver requalification, cost/delta measurement, and quiet publication; RF068 live AI category assertion will remain explicit until next repair.

033 second owner qualification: scoped UnitHealth driver execution admitted once via tool61116; JSON /tmp/bas-recording-driver-owner-033.json. Do not duplicate this execution. Integration fixtures that directly inject the dormant script now send acknowledged activation; rapid-navigation fixtures now use the actual pipeline lifecycle. Focused activation suites running tool89438, /tmp/bas-recording-fixture-activation-033.txt. Native14/14 and types final2 passed before those fixture updates. Benchmark harness bundling corrected a local proto-export resolution error; no performance value recorded yet.

033 necessary shared-owner extension RF069: direct scoped UnitHealth driver validation tool61116 exited Client.Timeout exceeded without a terminal receipt. No validation/Jest process remains in the process snapshot. UnitHealth validate handler uses the ordinary CLI HTTP deadline for inherently multi-minute synchronous execution; existing cli-core NewConnectHTTPClientWithTimeout(app,0) already encodes the proper owner-bounded contract. Scope: unit-health CLI validate handler and real delayed-response regression in app_test.go, with architecture target updated first. This is necessary to obtain the authorized BAS scoped driver receipt; no dependency or host repair needed. Preserve authentication through the existing client helper. Rebuild normal CLI owner after focused Go regression, then admit one replacement scoped request.

033 RF069 regression: ordinary client deadline reproduced on the real CLI/HTTP boundary. Existing long-RPC helper now removes only that transport timeout; authentication and server-owned command limits remain. Initial green attempt reached the fixture response and exposed a JSON/protobuf content-type mismatch; corrected fixture yields all CLI race packages passing (/tmp/bas-unit-cli-deadline-green2-033.txt). Normal unit-health CLI admitted exactly one replacement driver validation, tool27922; outputs /tmp/bas-recording-driver-owner-final-033.{json,err}. Await that tool, do not re-admit. Previous lost benchmark session recovered as terminal; benchmark allocation refinement now uses one visible-entry traversal. Latest tidiness20260922-144527-149c7c63 terminalfailed/quiet wait consumed:1115findings,103long/387complexity/607dup/17coupling,debt33540. No033deployment yet.

034 RF068 investigation started while the033 scoped driver runs. Recall completed
with source-ledger providers degraded; no suitable replacement program found.
Source confirms resource-ollama gateway already supports --format JSON Schema;
no resource-owner/dependency expansion needed. BAS client currently omits it.
Generator accepts missing required fields and malformed JSON fallback always
returns empty success. Target architecture updated before034 implementation.
Planned scope: existing AI client/generator and deterministic/live tests; preserve
governed gateway, valid array/object capability and DOM-only analysis fallback.
Do not turn validation errors into integration skips. No034runtime edits yet.

033 publication observation14:58:07UTC rejected stop: live history2875rows includes10 new RUNNING beyond the five historical IDs; driver10sessions/0recordings. Receipt /tmp/bas-drain-deferred-033.json. No process stopped or history altered. Recheck only after scoped driver completion or observed live activity change. Continue034 independently if activity persists; publish a combined qualified candidate after a fresh quiet gate. Existing032 build remains live.

033 scoped driver owner completed: uh-20260922-145457-7e793a0764f101413651f7c3ea6d386d passed393.839s;1549tests/125suites pass,1suite/2tests skipped. Tool27922 consumed. RF069 fixed CLI now retains the complete terminal receipt. BAS API/UI makebuild and driverbuild pass. Publication gate previously found live work; no stop performed.034 deterministic AI package race passes1.106s. Initial live structured-output smoke fails11.421s: provider schema converter requires anchored patterns, so unanchored nonblank-action constraint is being corrected without weakening validation. Original failing assertions and errors retained.

034 scoped owneruh-20260922-150329-35524363f0d47303a760c85b534c0497 failed only the empty-element integration assertion; isolated identical test passed1.653s. Owner traceability identifies the case but bounded stdout omitted its assertion text, so exact intermittent failure is not established. Empty elements have no grounded action to infer: architecture now requires a local empty result. New no-provider-call regression fails against current code (provider error propagated), then runtime early empty return repairs the semantic boundary. Requalification required; original owner failure retained. Prior schema-validation adds roughly3.8us and~3.8KiB/81allocations per mocked one-suggestion request; repeated quiet final benchmark pending. No real-inference latency claim.

034 final scopedAPIuh-20260922-150913-3760ae5d039caeeb866c3fd0a3db7907 passed33.130s, final focusedAI race1.092s and APIbuild pass. Source3runtimefiles658/23/79 ->669/22/81 lines/functions/cyclomatic (+11/-1/+2); removed dummy fallback and redundant logger state, one schema governs output generation and validation. ExpandedGo69original/66current31177/1129/5079 ->28516/1079/4839; BAS-240/shared-12 = affectednet-252. Final matched mocked-request median4562->9967ns (+5405ns/+118.48%),3849->7704B,24->105allocations. Cost supplies previously absent validation, no inference speed claim. Source-shape3line empty guard is later than latest native tidiness; explicit final Go counts retained. Fresh quiet publication gate/stop tool44577 launched once; receipt /tmp/bas-drain-gate-034.json or deferred034. Tools77291/92901 consumed.

034 quiet gate15:10:11UTC admitted stop:2668rows (2288completed/375failed/five exact unchanged historical RUNNING),0sessions/0recordings, same032build. Stop completed, make start tool45118 pending (/tmp/bas-start-034.txt). The count differs from2833 at032 and2875 at14:58 observation before any034stop: do not claim unchanged whole history or attribute cause without evidence. Existing retention/concurrent work may account for change; inspect owner logs after startup while preserving all remaining data. No agent-issued deletion.

History-count diagnosis035 read-only: previous runtime logs show automatic owner retention at15:01:12UTC,934recording directories/3.670660666GB removed to the declared20GiB budget; captures480/1.342555037GB to5GiB. Existing owner_retention.go RemoveHook also deletes corresponding terminal execution rows. This supports retention as the explanation for the pre-stop count decrease; individual removed IDs were not retained in the gate, so exact207row reconciliation remains unverified. No new defect established/no retention edits or agent-issued cleanup. Original profile fields pass three complete repository reads against rollback. Lifecycle start tool45118 still owns publication; do not duplicate start.

033/034 published: healthy15:14:42UTC buildce4d775d1ee2c2ba22bd0c8c7ceed449bb2e84cfd1cc7315f869859a12460acc. Driverready Chromium136.0.7103.25/0sessions/0recordings. ProfileAPI metadata equals032 and all original rollback fields survive three repository reads. Start45118 consumed; no pending test/build/lifecycle operations. Source-only gap statements above are now superseded by this deployment receipt. Full product remains unqualified; board17pending. Next035 actual frame-targeting audit under RF004/RF030: source frame handler pushes mainFrame instead of target, compares URLs for identity, and route context currently appears not to forward frameStack; native action-effect proof needed before repair.

035 RF004/RF030 frame execution investigation: read-only history diagnosis found
existing retention consistent with the count change; no new retention defect or
edit. Frame source audit finds route ExecutionContext omits session.frameStack,
FrameHandler pushes mainFrame rather than target, compares URLs as identity, and
DOM handlers use page directly. This is separate from033 recording activation.
Target architecture updated before035 source work. Add an actual public-driver
frame-switch then click fixture with same selectors and same-URL sibling frames;
assert independent frame counters, not success metadata. No035runtime edits yet.

035 native red proof: public frame-switch ENTER and CLICK both return success, but independent fixture reports mainClicks1/childEffects[] instead of main0/left1 (/tmp/bas-frame-native-target-red-035.txt,1.973s,onefailed/15filtered). Earlier2.533s case failed EXIT because the route never retained frameStack;1.545s intermediate only checked missing child effect. Native standalone probe also confirms childFrame.evaluate(location.href) returns parent URL; child locator.evaluate then reports destroyed context, while Frame.click and FrameLocator.click both hit the correct child. All use default installed Rebrowser. Investigate the lowest context owner before choosing a complete DOM-target boundary; no035runtime edits/deployment. Source test before-copy under /tmp/bas-before-035. No pending operations.

035 lowest-owner finding and candidate: Rebrowser1.52.0 main-world discovery
returns undefined contextId for scriptless frames; that silently defaults runtime
evaluation to the session main document. Inline child script avoids the failure;
context.addInitScript (void or property) does not. Reading installed SDK confirms
binding discovery emits context payload without validating its ID. An isolated
copy under /tmp/bas-rebrowser-core-035 now resolves the frame's actual document
through public CDP, materializes its main realm before adding the temporary
binding, then waits for a positive matching binding receipt and releases objects/
listeners/binding. Initial prototype bound before realm creation and timed out;
reordered prototype passes Frame.evaluate, locator.evaluate and both click paths
on a scriptless cross-origin frame, without Runtime.enable/mode changes.
Receipt /tmp/bas-frame-routing-sdk2-probe-035.json; no installed SDK changed.
Necessary dependency-owner extension proposed: BAS driver canonical dependency
patch metadata/lockfile via SDA, with regression in native suite; no runtime
monkeypatch/private SDK access from BAS. Existing governed install dry-run supports
playwright-driver surface but reports rebrowser-playwright unrecorded. Next record
that already-observed dependency through its owner, then validate patch install
path before any mutation. Upstream release page lists1.52.0 as latest published
Rebrowser Playwright line (https://github.com/rebrowser/rebrowser-patches/releases);
no verified newer fixed release. No commercial/external publication or message.

035 SDK qualification: isolated matrix passes default/--site-per-process with
scriptless same-origin/cross-origin/sandbox/nested/identical-URL frames, three
concurrent first evaluations, navigation, detached refusal, worker evaluation
and narrow console-stack-getter probe. Corrected SDK owner session lookup through
existing _sessionForFrame; shared context discovery now joins one request. Removed
swallowed undefined context errors and persistent discovery scripts/listeners.
An attempted document-metadata generation guard falsely rejected OOPIF navigation
because metadata lags the actual realm; rejected it. Explicit CDP object/context
identity remains authoritative. Native maintained tests added to existing typed-
action suite; installed SDK fails both cases1.942s before patch adoption (focused
coverage also below global threshold; full scoped suite will retain thresholds).
Security Health11.351s reports773findings/8errors, no rebrowser finding; RF071
records production js-yaml advisory and untriaged detector claims. SDA approved
existing exact rebrowser-playwright1.52.0 for BAS with explicit limits (owner
receipt /tmp/bas-deps-approval-035.json). Canonical two-file SDK patch and pnpm
metadata prepared; use SDA install to adopt, no raw package manager/registry edit.
Live034 unchanged. No pending owner operation.

035 candidate qualification continues, not published. Installed first SDK patch
passes2 native matrix cases1.773s. BAS initial frame wiring passes18native6.995s;
focused owner145cases has138pass/7old frame-mock failures. Replaced tests that
expected detached-frame silent fallback with refusal/recovery/identity/ancestry
checks. Expanded public-route DOM test finds keyboard went to focused main page
rather than selected child. Native probe confirms focusing another document resets
child activeElement to BODY; the correct oracle is key delivery to the selected
document, not invented restoration of a previous input. Keyboard now focuses the
selected frame element when needed and blurs a selected document's descendant
iframe; physical keyboard owner remains Page. Expanded test pending.

First fresh-browser evaluation of a scripted data document exposed an SDK root
DOM.resolveNode mismatch (focused reproductions twice; full suite had hidden the
cold-first path). Isolated correction materializes the default realm only after
Page.getFrameTree proves the requested frame is that connection's root; children
still resolve their own document object. Five fresh-browser first-document cases
and the full frame matrix pass. Canonical patch updated; SDA reinstallation and
requalification next. No runtime mode override or private SDK access from BAS.
An attempted patch regeneration found pnpm had removed the original package;
that Python step failed before edits, and the following SDA invocation reapplied
the unchanged first patch (59242 consumed). Pristine two-file baseline recovered
by reversing the exact canonical first patch on an isolated copy. No installed
package edited directly. No pending owner/test operation at this checkpoint.

035 RF072 extension: native selected-document storage/drag checks pass, but the
second download returns the first child's cached file (parent.test.txt expected,
child.test.txt received),1.186s. Source confirms handler TTL result reuse bypasses
new operation identity. Necessary scope: download.ts, unused result-cache branch
and singleton in infra/operation-tracker.ts/index.ts, legacy close/reset cleanup
callers, and existing download tests. Target architecture updated first. Route
receipts remain the sole retransmission owner. Preserve upload/tab in-flight
tracking; no speculative replacement cache. Public frame/tab/detached/ambiguity
case already passes8.445s. Full159focused cases passed12.442s before this new case.
SDK root patch installed through SDA63437 (consumed); no publication.

035 final focused candidate: shared session frame path and typed DOM target cover
all selector/evaluate handlers, storage, drag, download and physical keyboard
focus. Public native test qualifies main/sibling/nested effects, parent vs exit,
new-tab page adoption, detached refusal and ambiguous same-URL refusal. Keyboard
oracle observes the actual selected document after external focus resets its
active element; no hidden focus-memory owner added. Fixture-only enum/import and
mock-shape errors were corrected, with earlier receipts retained. Full159cases
pass12.442s before download extension; final native/download/close/reset29cases
pass22.832s. Download cache removal also retires unused generic cache configuration/
methods, deprecated cleanup wrapper, route callers, and eight assertion-free or
wrapper-only tests. Actual repeated-download regression replaces them. UUID paths
prevent clock-time collisions; platform temp directory replaces hardcoded /tmp.

035 full qualification admitted once: UnitHealth driver tool2220; Test Genie
tidiness admission5607; types/build84637; board80061. Source candidate not deployed.
Continue collecting owner receipts, final comparable runtime/debt measures and
SDK performance cost; publish only at a fresh quiet gate. Full product remains
unqualified and the continuous goal remains active after publication.

035 measurement checkpoint16:12UTC: types/build pass. Tidiness20260922-160940-cd080fc7
terminalfailed5s, one quiet wait consumed. Native1116findings/103long/387complexity/
607dup/18coupling/debt33540: unchanged duplication; +1coupling is the expanded
integration fixture's21 imports, retained honestly rather than consolidate paths
to game it. Driver21runtimefiles5172->4724lines(-448), SDK2files1879->1881(+2),
affectednet-446lines. No Go source changes this cycle, prior net-252Go cyclo
remains the last comparable cumulative measure. Removed policies: wrong-frameURL
identity, duplicate EXIT/PARENT logic, handler TTL result-cache owner, persistent
SDK discovery scripts and leaked event listeners; result-cache maps/config/methods
and legacy route cleanup retired. TS AST complexity remains unverified (RF064).
SDA governance reports advisorypass with46existing warnings and no rebrowser
warning; no clean-fleet assertion. Security triage identifies five detector
matches as source SHA hashes and G101 as the non-secret credential field label;
follow owner-supported narrow triage after035 qualification. Historical evidence
has not been rewritten. SDK cost harness prepared outside source; run when full
UnitHealth2220 finishes, not during its load. Only that operation remains pending.

035 full driver failure: uh-20260922-160935-a45ef0d56f9e9bedcbddffeba01e6f85
325.444s,1544pass/4fail/2skip (122pass/2fail/1skip suites). Three recording-pages
fixtures omit mandatory session.frameStack; partial utils mock then masks that
error with undefined instanceof class. Fixtures now supply selected frame state
and assert it is cleared on page switch; preserve actual error exports. Native
audio fails context-binding acknowledgement, while isolated unchanged audio
passes7.293s. The current SDK assumes the binding event precedes the command
response; testing delayed legal delivery deterministically before changing it.
The original failure omitted exception-vs-missing-receipt detail, so actual audio
failure cause is not yet established. Full owner operation consumed; no rerun yet.

035 receipt-order repair: deterministic SDK-owner test delays the matching binding
event until after the evaluation response; old candidate fails0.750s. SDK now
joins that exact receipt with a bounded3s handshake timeout, validates the ID and
cleans its timer/listener/binding. Exceptions now carry the distinction from an
absent receipt. SDA install91918 consumed. Native frame/audio/recording-page
fixtures31cases pass19.940s; types/build pass. This proves delayed delivery support;
it does not retrospectively prove that ordering caused the first full audio failure.

035 quiet SDK cost: three trials per variant,30measured+5warm-up document contexts,
alternating variant order, real Chromium, equal main-document sentinel checked.
First-evaluation median-of-medians3924.92->2582.95us (~34.19% lower); warm already-
acquired context347.40->348.58us (~0.34%, noise). Context creation/navigation are
excluded; no child-frame or whole-journey speed claim. Raw90samples per variant
and warm measurements /tmp/bas-frame-sdk-cost-035.json; harness at
/tmp/browser-automation-studio/frame-context-cost-035.cjs. Original SDK restored
only in isolated baseline copy, installed dependency remains SDA-owned patch.
One fresh full scoped driver request admitted after these repairs and the quiet
benchmark, /tmp/bas-frame-driver-owner-final-035.json. No publication yet.

035 final full-driver qualification, 2026-09-22 UTC: UnitHealth
uh-20260922-162102-ce8fafecad1445ee49c7f58ab339b894 passed;1549tests/124suites,
2tests/1suite skipped; Jest350.649s. SDK acknowledgement ordering and mandatory
frame-state fixtures repaired. Existing audio fidelity passes in this full run.
Receipt /tmp/bas-frame-driver-owner-final-035.json; CLI27696 consumed. No broad
stealth/security or complete record/replay qualification implied. Next: fresh
quiet-work gate, lifecycle restart and protected-profile verification.

BAS-WORK-036 investigation, 2026-09-22 UTC — RF073/J07/J17: recall found the
original navigation-flake retry rationale; native independent HTTP counter
disproves its implicit assumption that evaluate is read-only. One instruction
causes two POSTs before returning NAVIGATION_ERROR/retryabletrue. Temporary
harness /tmp/browser-automation-studio/evaluate-navigation-036.ts bundles the
actual handler and typed instruction factory; two initial harness module-resolution
failures are setup errors, not product evidence. Correctly bundled probe confirms
the defect. Target architecture updated before code. Source035 remains unchanged
until its lifecycle restart finishes. Retire implicit retry helper and replace
its implementation-mirroring tests; preserve extraction and successful expressions.

035 publication in progress: quiet16:28:16 gate2555historyrows/2207completed/
343failed/five unchanged historicalRUNNING,0sessions/0recordings; gate0.489s.
Ownerstop39755 consumed. API/UI makebuild54509 passed; start92545 pending.
History decreased before this stop, as in034; exact retention deletions unverified.
No agent-issued data deletion. Full driver receipt351.446s passes.

035 publication complete16:32:02UTC: build4cbb4572edd4546c5aa0ba64bf53754fd51d659294ab109f8da33d89e677f1f9 healthy; driver0sessions/0recordings. API profilemetadata equals034; full original profile fields survive three reads. All lifecycle operations consumed. Final runtime net-437lines includes SDK+11; no source hash drift across21 runtimefiles before publication. Named defects qualified, product board remains17pending.

036 focused qualification:82cases across5suites pass15.235s; types60651 and
focused12197 consumed; driverbuild passed. Original maintained red2.086s had
three actual assertion failures and a whole-project coverage-floor failure from
selecting only six tests; do not interpret that filtered run as aggregate coverage.
Focused green disables collection only for the selected cases; the full owner73224
retains normal aggregate coverage. Tidiness unchanged1116/debt33540; board17pending.
TestGenie20260922-163350-8274aa55 admission42506 and one quiet wait consumed.
037 RF071 investigation: official js-yaml advisory confirms4.3.2 fixes
GHSA-2883-xcg3-v3hh; installed direct/override4.3.1 needs owner-governed update.
SDA override dry-run saysapproved even though recorded allowedScenarios only
containsvrooli-onboarding; do not rely on this as scope governance proof.
Preserve that existing grant when adding BAS. No dependency mutations admitted
while036full driver consumes the current lockfile.

036 qualified307.798s,1552tests/124suites pass; no pending owner runs. Runtime
extraction handler189->152lines(-37), implicit retry helper/load wait retired.
Publication will combine036with037to avoid redundant lifecycle interruption.

BAS-WORK-037 authority extension, 2026-09-22 UTC — RF071:
SDA InstallDependency has a supported governed npm override operation, but its
SetNpmOverride appends a broad package key without retiring the older qualified
keys. BAS currently pins vulnerable js-yaml4.3.1 with qualified overrides; replacing
that policy must leave one effective package-wide version declaration. Extend to
scenarios/scenario-dependency-analyzer/api/internal/installgateway/gateway{,_test}.go
and owner architecture docs: a broad override replaces only same-package simple
version-qualified keys, preserving unrelated/scoped/parent-qualified policies.
Prove the canonical replacement before repair and validate owner packages plus
actual BAS resolved lockfiles. No hand-edited registry/lockfile or raw installer.

RF071 exact-digest triage proof: installed Gitleaks8.18.1 fixture reports8findings
before and3after the candidate exact-line exceptions. All three synthetic credential
counterexamples remain detected in the same evidence file; the five reviewed
source-digest lines are excluded. No path/directory exclusion and every default
rule retained. Candidate /tmp/bas-gitleaks-candidate-037.toml; receipts
/tmp/bas-gitleaks-fixture-{before,candidate}-037.json. Security Health's safe-fix
preview offers no deterministic fix; use its documented scanner-native exceptions
with this evidence. Original historical source JSON stays byte-for-byte untouched.

037 boundary extension RF074: owner source confirms the approved-scopes mismatch
seen in the BAS preview. Extend the same SDA repair to
api/internal/dependencygovernance/install{,_test}.go. Reuse existing
scenarioExceptionViolation, retain its precedence for explicit denied scenarios,
and reject before SetNpmOverride or package install. Test public handler with
scoped/global grants and explicit denials for normal and override requests.
BAS package approval was widened through owner upsert while retaining onboarding;
no unauthorized-scope package installation occurred.

037 owner-validation checkpoint: SDA focused installgateway/dependencygovernance
races pass1.027/1.080s; makebuild33982 consumed/pass. Scoped API UnitHealth
uh-20260922-164500-7aefb46ceaa5aa04865b6682e1006c6c failed15.803s (70972consumed).
Its bounded JSON excerpt contains no failed test event; full local reproduction
is running to preserve the exact assertion before any owner deployment.
SecurityHealth triage037 drops five exact-digest errors, leaving two js-yaml
advisories and one G101. The original G101 annotation was placed on the field
label; native rule-specific report identifies the preceding public namespace
declaration instead. Annotation moved to that exact declaration, removed from
the unaffected field; native G101 scan now haszeroissues. No claimed whole-scan
pass yet. Original historical JSON unchanged. No SDA/BAS restart pending.

037 required-owner check failure identified (87731consumed): four assertions in
SDA coreset/dependencyhealth. RF075 records trimpath-hostile runtime.Caller and
live-operator-state leakage in temporary portability fixtures. Extend test-only
boundary to internal/coreset/coreset_test.go and
internal/dependencyhealth/connect_test.go; reuse repocontract.ResolveRepoRoot and
route fixture operator storage. Strengthen the previously false-positive
contradiction check to name the expected cause. Production closure remains intact.

037 SDA qualification final: UnitHealthuh-20260922-164831-94a58551fb070680f95ca89f3d26aa50
passed12.241s;99496consumed. Fixture races1.102/1.124s,15319consumed. Managed
restart43426pending, /tmp/bas-sda-restart-037.txt. Shared runtime753->769lines
(+16),24functions unchanged, cyclomatic209->213(+4). Cumulative affectedGo
reduction now248, including necessary admission checks. No performance speedup
claim. Bounded js-yaml4.3.1 probe with32empty merge sources and budget8 fails to
reject; ordinary workflow YAML parses. Native adverse input is small and local.

BAS-WORK-038 RF072 investigation: recall searched16results across10corpora, then
native realHTTP attachment probe confirms a directURL action falselyfails with
ERR_ABORTED while one correct download event/file stream arrives. No external
account or retained browser data used. /tmp/bas-download-url-red-038.json and
/tmp/browser-automation-studio/download-url-038.ts preserve the oracle. Target
architecture updated before repair: accept only Chromium's expected attachment
navigation-abort signal; actual event/save are still mandatory. Add native URL
byte proof and error/no-event controls to maintained existing test files.

037 dependency/security completion: all three governed installs succeeded; direct
20309 and UIoverride40807 consumed. Exact lock diff changes onlyjs-yaml and
its overriding selector; canonicalRebrowser patch remains90014f7f... Both native
CPU-budget and ordinary-YAML controls pass4.3.2. SecurityHealth73941 consumed,
20.186spass0errors; warning/info posture remains357/406. BAS makebuild74263passed.

038 validation checkpoint: positive URL regression red1.320s then nativeURL
positive/HTML-no-download/abort-no-download controls green. Source adds one
Chromium-specific failure interpretation; no fallback file or inferred success.
First focused33cases15.995s:32pass, evaluation fixture observeszeroPOSTs. That
assertion omitted actualresult; improved diagnostics now retain its error.
Five targeted native cases pass1.985s; 40fresh-context reproduction47867pending
(/tmp/bas-evaluate-stress-038.json). Types/driverbuild pass; first failure retained
and source qualification remains incomplete until this is understood/rechecked.

038 RF070 diagnosis: installed patch40-context native probe has32failures, not
the preliminary28mentioned in commentary; actual receipt retained. Baseline
unpatched SDK0/40effect failures but internal stderr errors remain. DOMobject
root selection candidate39/40fails and is rejected. Three cleanup variants: keep
registration0/40bad (89022consumed), keepglobal38/40bad (70706consumed), keepboth
3/40bad (21724consumed). Original73743 and objectcandidate73549 consumed.
One registration per CRSession with unique per-request token is under isolated
probe48166 (/tmp/bas-evaluate-stable-038.json), including follow-up-document
reads. Do not restore arbitrary evaluate retries or change global runtime mode.
No canonicalSDKmutation or dependency reinstall for this candidate yet.

038 SDK candidate refined: stable registration with deleted globals gives1/40
effect failures and cannot rediscover some retained frame contexts. Combining
DOMroot selection with stable registration fails37/40 and is rejected. Retaining
one session binding/global, using unique receipt tokens and a bounded internal
absent-binding handshake passes40/40effects plus40next-document reads; native
frame/OOPIF/worker matrix20rows passes. Installed-patch maintained fresh-context
regression fails at trial0 (receipt /tmp/bas-sdk-cleanup-maintained-red-038.txt).
Candidate files /tmp/bas-rebrowser-core-038-stable; canonical two-file patch now
regenerated from pristine1.52.0baseline. Governed install follows; no application
Runtime-mode change or arbitrary expression retry. Bounded retained global is an
explicit stealth qualification limitation, and replaces accumulating bindings.

038 installed candidate focused36tests/4suites pass22.394s, including native
fresh-context corpus, SDK receipt correlation and existing audio. Types pass;
owned server cleanup strengthened to join browser/server closure even on failure.
Matched SDK benchmark (3 alternating trials,30samples each plus5warmups) against
pristine1.52.0: cold median3562.362->2215.312us; warm292.730->320.320us. Report
the warm increase27.590us, not an unsupported speedup or full-journey claim.
Raw /tmp/bas-sdk-context-cost-038.json. Final driver/security/types+build/tidiness
operations admitted once; terminal receipts pending. No lifecycle action yet.

038 final qualification: full driver uh-20260922-171531-588ec4e8e9386bbcc9675231167fbaca
passes1559tests/124suites,2tests/1suite skipped,366.709s Jest. Original admission
was not repeated after tool-ID loss; pidfd waiter20434 joined its CLI and is
consumed. SecurityHealth5.060s passes0errors/357warnings/406infos (owner caches
for unchanged Go scans). Driver types/build pass. Finaltidiness20260922-171537-45b14dd7
failed with1117findings/104long/387complexity/607dup/18coupling/debt33540, one
quiet wait consumed. Added native integration cases cross one long-file threshold;
no threshold or coverage floor changed. Preparation valid/productfalse and board
17pending remain. Recorded source hashes match. Proceed to fresh drain observation
before combined036–038 publication, preserving full original profiles.

### BAS-WORK-039 — 2026-09-22 UTC — reset ownership investigation

Feedback008 remains active. Recall returned89hits across10corpora, including
the existing lease/terminal ownership work and BAS architecture. Source shows
reset ignores request ownership and invokes destructive state changes through
a route-owned pending map. Hypothesis: a delayed old lease can clear a new
owner's cookies/navigation after label handoff. Native temporary HTTP/Chromium
fixture uses only synthetic local state, independent cookie/URL/storage sentinels,
and stale/missing/current owner controls. No039production changes yet;038managed
start31992 remains pending and must finish before further source edits.

038 publication: build8129e9e98743f51dfb61a568315b0662dd69c312ca63beff80197474745a2e80
healthy17:26:01UTC atAPI/UI and driverok with0sessions/0recordings. Fresh gate
17:22:15 saw2618historyrows (2239completed/374failed/five exact unchanged historical
RUNNING); no active browser work. Managed stop59912/start31992 consumed. APIprofile
metadata unchanged and three complete profile reads match original rollback;
profile84003 consumed. Combined036–038 runtime net-7lines including shared SDK/SDA
changes; cumulative affected Go cyclomatic-248. Security0errors still357warnings/
406infos; tidiness1117 and board17pending/productfalse are not a release verdict.

039 native HTTP/Chromium probe confirms stale and missing lease requests both
clear the new owner's cookies and navigate it toabout:blank. Even the authorized
control failsSecurityError readinglocalStorage onabout:blank, strands phase
resetting, and retains the old origin's localStorage. Three independent controls
are retained in/tmp/bas-reset-native-red-039.json. Fixture setup first needed
canonical proto bundling and staged browser scripts; no product effect occurred
in those setup failures. Next repair/reset qualification must cover ownership
and a coherent clean-context boundary rather than suppressing the storage error.

039 bounded implementation choice: first close the independently proven stale-lease
reset admission gap across Go Session→driver client→HTTP route, deleting route
time heuristics and duplicate logging. Existing in-flight joining stays in its
current route owner for this change. Do not imply that this repairsRF076: native
current-owner control remains an explicit failing storage-reset observation.
Scope: existing reset route/tests, Go driver/session and native profile continuity
test (temporary owned context, HTTP route). No new dependency or schema work.
Follow with the coherent context reset repair; do not suppress SecurityError.

039 maintained tests: native stale/missing/released reset controls all fail before
repair3.204s. Go tests fail on absent owner body and false/missing success receipts.
After ownership transport+validation, focused9cases/2suites pass4.248s (including
concurrent shared success/failure and delayed-body handoff); Go driver/session/engine
races pass. The engine happy-path fixture now acknowledges reset instead of emitting
an empty200. Initial focused6/2pass3.499s warned of open handles; explicit fixture
server connection closure and --detectOpenHandles run exits naturally without
reported handles. Driver types pass. Runtime130->42reset route, Go client+11lines,
sessionlinecount unchanged: net-77. Scoped driver38932/API30665 and tidiness
admission49887 admitted once; driver types/build4732 pending. Live038 unchanged.

039 API owneruh-20260922-173337-7982c7926e5e83a27ee25627d788697f passes36.661s;
API/UI and driver builds/types pass. Tidiness20260922-173343-048f3e9f terminalfailed
and onequietwait consumed:1118findings/104long/387complexity/608dup/18coupling,
duplication debt32643. This whole dirty-tree reduction cannot be attributed solely
to039. Go99functions unchanged,290->292cyclomatic; cumulative affectedGo-246.
Boardprog_41d40642-91c9-4e08-9fe0-90a6ffc4d7f4:17pending/productfalse. Full scoped
driver38932 remains active; no lifecycle operation.

040 storage-reset discriminator (no production edits): official public Chromium
CDP DOMStorage.clear rejects origins without a currently loaded frame, including
all old origins afterabout:blank. Storage.clearDataForOrigin(all) clears persisted
localStorage on both origins but leaves per-tab sessionStorage intact. Native
/tmp/bas-reset-{cdp,storage-cdp}-040.json rules out those naive substitutions.
Context recreation would also require preserving recording/capture ownership;
per-origin snapshots omit cache-only origins and copying large IndexedDB merely
to enumerate origins is avoidable overhead. Continue owner design/independent
storage matrix, without suppressing errors or claimingRF076 resolved.

039 full driveruh-20260922-173337-68e0fa749361097f6a4aa0c9df53bc49 passes342.144s
(337.637s Jest):1565tests/124suites,2tests/1suite skipped. Tool38932 consumed.
All3recorded runtime source hashes unchanged. Jest reports a one-second open-handle
warning before natural exit; focused detection did not report a handle, so the
earlier fixture closure is not proof that every asynchronous initializer is joined.
Retain this lifecycle concern for the reset/close owner investigation. No pending
validation. Proceed to fresh quiet gate for the qualified lease admission fix only;
RF076clean storage reset remains separately broken.

040 candidate experiment only: a temporary public-CDP plus intercepted-origin
visit clears cookie/localStorage/sessionStorage/IndexedDB/CacheStorage across
two origins with zero fixture HTTP requests during reset (~39.98ms, one trial).
A separate context on the same origins retains all five sentinels; a non-initial
active page remains and other pages close. Raw/tmp/bas-reset-protocol-candidate-040.json.
This is not production qualification: origin inventory/imported and closed origins,
partitioned iframes, service workers, recording's captured primary Page, failed
stages, and retained video/performance capture need tests/owner design. Avoid
replacing the native reset defect with a partial clear. Managed039start29526
still pending; no040production edits.

039 published buildaa7f6efbe4a4ee3e194865259087c40720dd50b2d2e8412b43c89effe7f3f66e
healthy17:43:49UTC. Fresh17:40:45 drain gate2618rows/five exact historicalRUNNING,
0sessions/0recordings. Managed stop55720/start29526 and profile86231 consumed.
APIprofile metadata equals038 and three full reads preserve original rollback
identity. Lease reset admission is deployed; RF076clean reset remains unrepaired.
No pending tool or lifecycle operation.

040 cross-site iframe experiment extends the candidate: localhost frame under
127.0.0.1 top-level holds separate local/session/IndexedDB/CacheStorage sentinels.
After reset all are empty; separate context remains unchanged. Zero fixture HTTP
requests during the operation;44.73ms single observation. This supports the named
installed-Chromium case, not a universal partitioning claim. No production040
change yet; proof/tmp/bas-reset-partition-candidate-040.json.

040 selected owner repair: preserve context and original primary page; record
visited/imported origin inventory at context construction; clear persistent
storage through public CDP and per-tab storage using intercepted empty origin
visits. No external requests or IndexedDB serialization to enumerate origins.
The two-origin and cross-site-frame prototypes support this approach; extend
maintained tests to imported/closed/cache-only origins, service workers, independent
context, primary-page capture, failure/retry and reset/close fencing. Scope existing
context-builder/SessionState/SessionManager/reset helper/route and their tests.
Move reset joining from the route to SessionManager and delete old swallowed
page/unroute errors; no new dependency. RF042profile import and RF043secondary
capture remain their own evidence boundaries. No040production edit before tests.

040 initial production repair: context-owned imported/visited origin inventory,
public CDP clearing plus intercepted origin visits, retained primary page/context,
error propagation for failed page close/unroute, manager-owned phase/in-flight
reset coordination and close settlement join. Initial native red3.166s -> green
3.375s; focused114/5pass6.948s and types pass. Expanded coverage57/59passes40.85s;
capture/video/trace/HAR and post-reset recording test passes. Added frame/cache-only
case times out30s with no operation location; diagnostic12939 admitted with temporary
RESET040 markers. Close-order test wrongly included constructor capability-probe
context close; reset baseline cleared and release now in finally. Do not lengthen
the timeout or call the expanded suite qualified. Live039 remains unchanged.

040 additional native file-origin discriminator: Chromium permits file:// local
and per-tab storage even though Node's WHATWG URL.origin isnull. The initial
tracker omitted it, and the maintained two-file reset test correctly fails3.109s
with both sentinels retained. PublicCDP file:// clearing plus an interceptedfile://
empty document is supported (one intercepted request, original file not loaded);
tracker now records that origin explicitly. Independent second-context storage
is the preservation control. /tmp/bas-reset-file-*-040.json and maintained red
receipt/tmp/bas-reset-file-maintained-red-040.txt.


### BAS-WORK-040 — 2026-09-22 UTC — clean reset qualification

RF076 source repair now passes the final149-case focused matrix in9.609s and
TypeScript/build checks. Original public synthetic reset reproduction now returns
200ready with no cookie/localStorage sentinel; stale404 and missing400 preserve
new-owner state. Its50.15ms observation includes the post-reset navigation/read
oracle and must not be presented as isolated reset latency. File-origin red is
retained. Final expanded fixture uses real frame-host navigation after service
worker registration; setContent timed out before reset and is not a repaired
product path. No deadline was raised and no temporary markers remain.

Owner driver89109 is the sole pending operation, admitted once. Tidiness run
20260922-180542-e4db78b1 failed; one quiet wait consumed. Types/build93368 passes.
Boardprog_8ded6dad-b2fc-4e13-9b5b-2b84432c3a9b still17pending/productfalse.
Runtime ownership improves through one manager-owned reset reservation/join and
error propagation;5runtime files add41lines, combined039–040 removes36. No new
Go changes and no invented TS cyclomatic reading. Retained primary context/page
keeps capture files and new recording usable; secondary captureRF043 and clean
profile importRF042 remain unqualified. Next consume owner, record metrics, then
fresh quiet gate before managed publication if qualified.

BAS-FB-009 captured both status requests verbatim and answered at overall-goal
level: materially improved, still core correctness work and17unqualified release
rows; no unsupported percentage/date. Continue the existing unlimited goal.

040 final owner qualification: uh-20260922-180536-251d1c74ffe41f505e06d9d89f73826c
passes332.571s (Jest327.966s),1573tests/124suites;2tests/1suite skipped. Tool89109
consumed, one-second handle warning followed by natural exit remains unresolved.
Native tidiness unchanged1118/104long/387complexity/608dup/18coupling/debt32643.
All5runtime hashes match the recorded source. No pending validation. Proceed to
fresh observational quiet gate and managed publication; preserve full profiles.

040 managed stop85193 consumed after fresh18:12:42 quiet gate:2885history rows
(2468completed/412failed/five exact historicalRUNNING), zero sessions/recordings.
Managed start76820 pending (/tmp/bas-start-040.txt). Do not edit production source
until the qualified build completes. Profiles must be rechecked afterward.

041 independent lifecycle hypothesis: early recording verification polls a closed
page and retries after session close has already reported success. Recall69hits
across10corpora (some provider degradation), no conflicting owner result found.
A temporary native manager fixture will count new CDP attempts after page closure
and time pipeline settlement after close, with a ready-before-close control. No
production edits and no claim that this explains every Jest handle warning.

040 publication healthy: build9c7c5eef94430e9f37d9a5b9bb14fb7fb01710053704586fdcc447d480651df7.
API/driver/UI healthy, zero sessions/recordings. Managed76820 consumed. All5
source hashes remain qualified. Profile API metadata unchanged and original full
profile preserved on three reads. Scope is clean reset, not whole product release.

041 native discrimination: ordinary immediate-close and ready-before-close controls
both settle before close with zero post-close CDP calls, rejecting an unconditional
normal-start leak claim. A deliberately deferred readiness promise succeeds but
retains one10000ms deadline timer. A raw native page with no injected script,
closed during readiness verification, still incurs40CDP attempts and2019.62ms
settlement against a2000ms timeout. Receipt/tmp/bas-readiness-lifetime-red-041.log
(FIXTURE_RESULT), tool33230 consumed. These prove bounded but orphaned readiness
work, not every full-suite warning's cause. Repair existing manager wait timer
and verification closed-page boundary; add maintained unit/native controls.
No new dependency, no global cancellation framework, no timeout increase.

041 maintained red: four failures/one pass in2.036s verify three losing-deadline
cases and native closed-page wait. First green attempt exposed missing once/off
methods in the existing Page test double and concurrent mocked startup polling
in the timer-count oracle. Added missing event methods; bounded readiness tests
now complete a mocked initial verification before measuring their own deadline.
No product timeout/floor changes. Six focused cases pass1.853s with no handle
warning. Final native fixture observes protocol-close and inter-poll-close without
checking the implementation's listener choice. Types pass; broader focused driver
checks pending. RF038 delayed stream-start ownership remains a separate boundary.

041 broader focused100tests/6suites pass19.47s, no warning; owner operations52477/
85469/53057 admitted once as in checkpoint. Native deadline retained1->0 and
closed-page protocol calls40->0; settlement0.387ms after closure. Live040 remains
healthy, no041deployment yet. Next consume owner receipts without readmission.

042 board investigation only: the rehabilitation branch hardcodes all17 outcomes
to pending_telemetry even after local receipts exist. Test Genie's governed
runs/list, runs/findings and runs/freshness can supply phase verdicts and current
tree identity. Its freshness verdict means latest *passing* run, so never treat
stale as evidence of product failure. Probe actual protojson and source-stability
fields before selecting applicability. Current structural-debt requires both
unchanged tidiness budgets and wider net-ownership/complexity evidence; passing
one phase cannot qualify the whole row. No board production edits yet. Skill
program-runtime read; its external journal/filing rules are overridden by active
file-only/no-stop authority. Read-only memory recall applied; no memory writes.

041 full owneruh-20260922-182410-0f03f688f5567b49d15aa763664bc239 fails84.444s
on an inline reset-route Page double missing event methods; not a product pass.
Replaced that duplicate literal with the existing createMockPage fixture rather
than adding product fallbacks for an incomplete mock. Final affected controls
include reset, health, manager, idempotency, external detach and native pipeline.

042 native governed probeprog_ab6aacd2-51c9-4184-a944-3518b4f28c23 succeeds but
freshness reports the empty-input SHA256td:e3b0c442... while the recorded tidiness
run has nonemptytd:72241b97... . Source inspection: RunsService.scenarioDir now
returns routed artifact storage, but CheckFreshness hashes that path as source.
No applicability assertion is valid from this read. The scratch session
sess_67c712aa-615e-4257-95ce-e0a50e75e19e was reclaimed. Next reproduce at owning
Test Genie handler with separated source/artifact roots; no board code edits yet.

041 corrected-fixture focused98tests/6suites pass13.931s; one-second warning still
appears before natural exit in the wider mocked cohort. Do not claim every handle
problem repaired. Original owner failure retained; second full owner admitted once
against unchanged runtime and corrected reset-route fixture.

042 necessary owner scope extension, before shared source edits: Test Genie
api/internal/app/runs/{service.go,freshness.go,lifecycle.go,*test.go} and existing
owner architecture docs. RunsService.scenarioDir is now an artifact path resolver,
but CheckFreshness and bare-name FindRun(matchCurrentSource) still hash that path.
This prevents BAS from using authoritative current-source qualification and can
make unrelated artifact bytes stand in for code identity. Preserve routed run
history and explicit target kinds; separate source resolution from artifact
resolution at this owner, remove the ambiguous helper name, convert all callers.
Validate source edits invalidate prior evidence while artifact-only edits do not,
with roots deliberately separated; retain exact existing FindRun shape checks.
No shared freshness-go algorithm or schema/dependency changes anticipated.

042 maintained separate-root tests fail for both freshness and current-source
reuse before repair (package0.360s). The run-query owner now resolves a source/
artifact pair explicitly; all17query calls use the routed artifact field, while
freshness and FindRun hash source. Deleted FindRun's duplicate typed-target source
selection and ambiguous scenarioDir helper name. Admission retains its existing
custom-path support. Focused current-source/non-scenario regressions pass0.368s.
Metrics {"before": {"lines": 1664, "functions": 59, "cyclomatic": 334}, "after": {"lines": 1656, "functions": 59, "cyclomatic": 332}} . Next owner/race/build checks, then live quiet gate
for Test Genie. ListRuns intentionally omits heavyweight terminal provenance;
board applicability must read GetRun for the selected exact ID, not mistake the
compact list's absent sourceStable/configuration fields for canonical failure.

042 operations admitted once: Test Genie owner UnitHealth API63392 (/tmp/bas-
freshness-owner-042.json), runs/shared-runs race5184, managed build5771. No Test
Genie lifecycle action yet. BAS041 second driver33499 remains pending separately.
Shared root repair adds no functions and removes8runtime lines/2Go cyclomatic
units; cumulative affectedGo-248. No board implementation or dependency edit yet.

041 second owner uh-20260922-183038-599e1437c3a424e2bb7cef8ee4e96d56
failed354.417s:1578pass/1failure/2skipped,123passed suites/1failed/1skipped.
The timeout regression sees3fake timers instead of0 when run after the whole
manager cohort. Hypothesis: common Page double never becomes closed or emits
close, so prior sessions keep polling across tests. Repair fixture lifecycle
semantics and retain the zero-timer assertion; no product timer relaxation.

042 runs/shared-runs race passes5.063/1.250s and managed build passes; sessions
5184/5771 consumed. API owner uh-20260922-183359-ac3f3e95bbbcfabddb42534feea5c109
times out104.007s after60s silence, with providerconformance fleet-contract
failures in stdout. Session63392 consumed. No full API pass claimed; affected
run-owner packages pass. Diagnose timeout source sufficiently to distinguish
from source/artifact repair; do not waive or suppress fleet assertions.

041 fixture hypothesis refined: globally emulating Page close would also require
repairing older host-audio fixtures that reuse a single Page for different
contexts. Manager unit tests do not validate script injection. Their existing
readiness stub now applies consistently per test, keeping unrelated background
verification out of deadline accounting; native pipeline tests retain real
verification/close coverage. Full manager coverage58/58passes2.28s, no warning.
Third full owner2810 admitted once; no runtime change since041 qualification.

042 isolated providerconformance diagnostic fails45.296s at its bounded test
deadline; stack locates CodeFacts.DescribeCodeFacts via validateExecutionRunners
inside fleet descriptor validation. This is outside the changed run-query owner;
retain missing full API qualification and recheck when provider-conformance
fixtures/live calls are repaired. No assertions suppressed. Fresh live Test Genie
admission reports running0/queued0/previewInFlight0; recorded source hashes match.
Managed restart58063 admitted after gate, pending. Observation is not an atomic
admission lock. BAS remains on040 pending driver qualification.

041 third owner uh-20260922-184018-126e6e3795f97e882bef37948c355810 passes
356.381s (Jest355.453s):1579tests/124suites pass,2tests/1suite skipped. Tool2810
consumed. No final one-second warning. Scoped readiness repair qualifies; fresh
BAS drain/publication is next. Runtime2files still net-13lines.

042 published Test Genie build25e2b598c0417ea7ac23ab7c501c93f4340e70bf0c0aabef1c5d578cf1029e4f
healthy2026-09-22T18:44:48Z; restart58063 consumed. Governed public probe
prog_9a4fffc3-3433-4d89-bd10-d604d934048d succeeds: source digest now nonempty
td:b2200de9a721ebc2d2e02bbe36a95bd341564a08363535f9851fb90ad11c8d8b. Exact
terminal read retains041sourceStable=true/shared-scoped provenance and1118
findings; its older digest differs after subsequent fixture edits, correctly
unqualified for current source. No empty artifact hash remains. Full owner API
qualification still unavailable as recorded above.

042 board integration remains unimplemented: the contract requires wider
structural improvement beyond tidiness, and the current terminal is historical.
SDA scan retained existing declarations but inferred no scenario edges; no
dependency edit/install was made. Latest board remains17pending/productfalse.
Prioritize core stream ownership next (RF038/RF045) while retaining authoritative
raw phase evidence. A green phase alone cannot qualify the structural-debt row.
043 recall66hits/10corpora (some providers degraded) confirms original RF045:
old async cleanup deletes replacement tracking, pending startup survives stop,
and failures leave a socket open. Investigate existing frame lifecycle owner
and session-close integration; preserve current041 source until publication.

041 publication deferred by actual active browser work: first fresh gate rejects
pagination repeated identity; follow-up health shows10sessions/0recordings on
unchanged040build. No stop executed. Resume publication only after a new quiet
gate. Source041 is qualified and may be composed with later scoped repairs; do
not freeze useful work while another owner is using the browser.

043 bounded next intervention within RF038/RF077: delayed session-start preview
ignores readiness=false and captures a mutable session lookup after waiting.
Prove timeout, closed, released and reassigned leases cannot start a preview;
live current-owner readiness still starts. Bind the page provider to immutable
execution/lease constants using existing getSessionForLease authority. Do not
create a second lease policy. Frame-manager overlapping generation defects
RF045 remain separate and will be addressed with their own disposal oracles.

043 delayed-preview discriminator: all6cases fail11.845s before repair, including
the ready control's post-release provider lookup. Existing getSessionForLease and
isOperational now guard deferred admission and every later page lookup; readiness
false starts nothing and rejection is observed. All10route cases pass; TypeScript
check passes. No broad frame lifecycle claim. Next compose with RF045 before
full driver qualification rather than duplicate an unchanged full suite.

044 RF045 owner design before edit: one per-session frame slot owns a serialized
lifecycle and current stream. Start/stop advance its generation immediately;
obsolete continuations cannot acquire/publish current resources. Stop closes
transport immediately and joins pending capture acquisition/disposal. Replacement
waits for old cleanup; cleanup failure stays owned and fails stop for explicit
retry. No separate parallel generation map or new cancellation framework.
Retain CDP/polling fallback but eliminate its duplicated startup/options path.
CDP strategy remains responsible for cleanup when its own initial start rejects.
RF046 resize/page-switch races remain separately open; measure this bounded
manager boundary with independently deferred acquisitions/stops and socket counts.

BAS-FB-010 captured before action: operator confirms all active browsers are this
engagement's and explicitly authorizes restart. Publish qualified041 plus focused
043 through managed lifecycle now; RF045 runtime is still unchanged, and its
5failing/4passing new regression tests are not claimed green. Saved profiles must
remain intact. This restart no longer depends on the quiet-browser gate.

044 maintained red: coordinator5failed/4passed0.430s for replacement registry,
late capture acquisition, stopped support probe, double strategy failure socket
cleanup and explicit disposal retry. CDP initial-start failure additionally leaves
its acquired protocol session/timer; separate regression fails0.513s. Extend
cohesive owner scope to existing session reset/teardown resource lists: neither
currently stops the frame coordinator. Native socket closure across close/reset
will be the independent integration oracle. No profile data changes.

041+043 published after BAS-FB-010 explicit restart authority. Managed60386 and
profile51362 consumed. Buildsha256:f23a02c1f9ef3fd227648911548f51c50734f9a2c68f2a2de4941039086fedf8
API/driver/UI healthy,0sessions; metadata matches040 and three complete reads
preserve original rollback identity. All3runtime hashes match qualification.
043 route10/10passes15.485s and types pass; full041driver1579/124 passed.
No pending publication operation. Further044 runtime edits start after this
publication boundary. Original two wrong profile RPC spellings returned404;
correct canonical session_profiles.SessionProfilesService/List succeeds.

044 native fixture initial attempt fails before the disposal assertion: setContent
on the injected blank page times out. Reused the existing real HTTP fixture;
second attempt exposes required lifecycle port configuration. The integration
fixture now uses the existing test config seam, not ambient server ports, and
triggers a real paint after socket connection. Third native attempt reaches the
oracle: close/reset both leave their socket open,2failed6.529s. No timeout
increase; transport closure still has its explicit1000ms observation budget.

044 coordinator now serializes one slot's acquisition/disposal; replacement
invalidates page/transport adapters immediately and waits for old cleanup.
Rejected cleanup is retained for explicit retry; identical strategy startup/
option code now has one path. Reset/teardown include frame stop. Initial capture
failure disposes its CDP session/timer. First green7pass/1fail4.486s: native
close/reset and manager boundaries pass; fake-timer count also included logger
microtasks. Draining microtasks distinguishes actual retained deadline timers.

044 additional actual defect: successful frame ACK keeps its losing timeout
alive. New ACK regression fails; ackWithTimeout now owns and clears its deadline
on every exit. No delay/timeout reduced or arbitrary script retry introduced.
Broader focused11954 pending; types61295 passed before final ACK edit.
Runtime line delta {"before": {"playwright-driver/src/frame-streaming/manager.ts": 440, "playwright-driver/src/frame-streaming/strategies/cdp-screencast.ts": 496, "playwright-driver/src/session/session-reset.ts": 59, "playwright-driver/src/session/session-teardown.ts": 89}, "after": {"playwright-driver/src/frame-streaming/manager.ts": 390, "playwright-driver/src/frame-streaming/strategies/cdp-screencast.ts": 507, "playwright-driver/src/session/session-reset.ts": 61, "playwright-driver/src/session/session-teardown.ts": 91}, "net": -35} . No claim of measured TypeScript cyclomatic improvement; duplicate startup
policy paths2->1, one slot retains every pending capture/disposal owner.

044 broader focused11954 passed (counts in/tmp/bas-frame-focused-044.txt),
including real transport close/reset, full native pipeline, profile reset matrix,
manager/session-start and capture strategies. Admit full driver owner, types/build
and scoped tidiness once against final source. Live043 remains healthy.

044 final focused111tests/6suites pass32.421s; one-second open-handle warning
followed natural exit remains observed, not relabeled fixed. Full driver12183
admitted once. Types/build98211 pass and consumed. Tidiness20260922-190312-45bb6d9a
failed; admission54364 and one quiet wait consumed. Native findings decode
pending. Scope remains source044; no live publication yet.

044 tidiness metrics unchanged1118findings/104long/387complexity/608dup/18coupling/
32643duplicationLineDebt. Original ratchets unchanged, no domain-wide pass.
Remaining full driver12183; current board read will follow terminal qualification.
045 read-only discrimination during044 qualification: recall64hits/10corpora,
some provider degradation. Existing RF046 mutable CDP session/frame buffer can
survive stop and page switch. Temporary controlled protocol probe counts late
starts/detaches and exact old/new frame bytes; no045 runtime/test edits yet.

045 controlled protocol probe confirms3RF046cases on current044source: resize
finishing after stop sends one late Page.startScreencast; a late CDP acquisition
also starts and is never detached; page switch publishes old-buffer/new/old-late
payloads instead of just new. The corrected independent wire oracle strips the
existing8-byte timestamp prefix. Receipt/tmp/bas-cdp-generation-red2-045.log
FIXTURE_RESULT; initial raw-prefix attempt retained separately. No045 source
edits yet. Full044driver still pending; board35815 consumed17pending/productfalse.

044 full driver uh-20260922-190308-c64e673046864877dc7a2f19b0dbc1d0 passes;12183 consumed. All4runtime hashes
unchanged. Publish through managed restart under BAS-FB-010 ongoing authority,
then verify health/profiles. No045 source edits before this boundary.

044 managed restart44419 succeeded; API/driver/UI healthy and0sessions; runtime
hashes unchanged and profile metadata equals043. Full profile check pending.
Full driver1596tests/124suites pass339.493s (Jest338.798s);2tests/1suite skipped.

045 RF046 owner design before source edit: each capture generation owns its
page, CDP session, listener, latest buffered frame and acknowledgements. Serialize
replacement acquisition/disposal; invalidate generation before awaits; stop joins
late acquisition, and resize rechecks generation after changing viewport. Merge
initial and replacement screencast setup, remove obsolete wrapper/duplicated
start policy, and never publish old-page bytes into a new generation. Reuse the
existing page probe to flush a buffered current frame after transport becomes
ready even if a stable page emits no new frame (RF047 buffer delivery only;
effective quality/FPS/perf settings remain open). Native frame encoding and
capture quality remain unchanged. New maintained adverse/control tests follow.

044 full profile45663 consumed: three complete reads equal original rollback.
Live build451603780f62b3880b2b91522c6425c46c590af78fcd7c96ebcb254f6e97f521
healthy,0sessions; no pending044 operation. RF045 coordinator boundary repaired;
RF046 internal CDP generation work now starts independently.

045 maintained generation group5failed0.551s: both viewport/protocol cancellation,
old-tab pixels, stable reconnect without another paint, and transport-send failure
withholding Chrome ACK. The last two are directly reproduced behavior, not
inferred pass claims. Correct the existing CDP owner with one acquisition path,
generation-bound protocol/listener/frame references, always-ACK delivery cleanup,
and current-buffer delivery from its existing page probe. No new timer/worker.

### Archived checkpoint before045 refresh (historical, not current state)


Active unlimited continuous goal BAS-FB-008. File tracking only; no plans or
external journal. Green checks trigger fresh investigation, never completion.

- Live BAS through043 healthy2026-09-22T18:58:32.808601+00:00, build
  f23a02c1f9ef3fd227648911548f51c50734f9a2c68f2a2de4941039086fedf8;
  API17116/driver24485/UI21794. Fresh18:12:42 gate2885historyrows/five exact
  unchanged historicalRUNNING,0sessions/0recordings. Managed start76820 consumed;
  profile33967 consumed. APIprofile metadata equals039; three complete profile
  reads preserve original rollback identity. No pending lifecycle operation.
-033 RF002/RF022 recording delivery is deployed and qualified for named cases. Browser stable UUID and matching delivery acknowledgement; one bounded
  pending queue, one driver entry/delivery/receipt owner; retained failed stop,
  reset and close; explicit pull ACK only after Go journal commit. One tracked
  page/frame activation owner uses WindowProxy control acknowledgements and joins
  activation before stop. Default Rebrowser mode preserved.
-033 native14/14, activation23/23, owner29/29, affected Go races/types/API+UI+driver
  builds pass. Scoped UnitHealthuh-20260922-145457-7e793a0764f101413651f7c3ea6d386d
  passed393.839s:1549tests/125suites pass,2tests/1suite skipped. Tool27922 consumed.
  Shared RF069 CLI timeout fixed using existing authenticated long-RPC helper;
  actual delayed-response regression/all CLI race packages pass. Runtime0line/
  function/cyclomatic change at shared owner; installed CLI autorebuilt.
-033 runtime22files12922->12286lines(-636). Go7files5567/270/847 ->5572/273/851
  lines/functions/cyclomatic (+5/+3/+4); BAS cumulative-242 plus shared-12 gives
  affectednet-254. Browser queues2->1, driver maps4->1, callback circuit/destructive
  read and duplicate page-activation policies deleted. No unsupported TS complexity
  claim. Matched10000-entry read+serialization median4000.795->3917.874us, within
  noise; no speed claim. Initial allocating implementation regression corrected.
-034 RF068 structured AI response contract is deployed. One JSON schema governs
  generation/validation; empty inputs need no provider call. Final scoped API
  33.130s/focused race1.092s/build pass. Runtime+11lines/-1function/+2Go cyclo;
  cumulative BAS-240/shared-12 => net-252. Mocked request median4562->9967ns
  supplies previously missing validation, no inference speed claim.
- Latest034 tidiness1115findings/103long/387complexity/607dup/17coupling,
  duplication debt33540(-2063expandedoriginal). Ratchets/floors unchanged.
  Full UI coverage remains28.59/30.34/65.15/28.59vs85. RF064 TS AST/JS duplication
  observation gaps persist. Last board03417pending/productfalse.
-035 RF004/RF030/RF070/RF072 deployed healthy16:32:02UTC, build
  4cbb4572edd4546c5aa0ba64bf53754fd51d659294ab109f8da33d89e677f1f9.
  Canonical Rebrowser patch installed viaSDA; frame identity/document targeting,
  context acknowledgement and new-download effects qualified in native tests.
  Full scoped driver uh-20260922-162102-ce8fafecad1445ee49c7f58ab339b894 passes
  351.446s:1549tests/124suites,2tests/1suite skipped. First failing run retained;
  mandatory mock state and delayed binding-ack handling repaired. Audio passes.
  Types/builds pass. Quiet16:28:16 gate2555historyrows/five historicalRUNNING,
  0sessions/0recordings; start92545/profile13486 consumed. Original full profile
  and API metadata preserved. No pending035operations.
-035 runtime21files5172->4724lines(-448); SDK1879->1890(+11); net-437.
  Same runtime hashes at qualification/publication. Cold main-document context
  median3924.92->2582.95us(-34.19%); warm347.40->348.58us(noise). Not a whole
  journey benchmark. No Go changes; cumulative affectedGo cyclomatic remains-252.
  Tidiness1116findings/103long/387complexity/607dup/18coupling/debt33540; one
  extra coupling finding is the expanded native integration test. Floors intact.
  Board17pending/productfalse. Full frame record/replay and broader stealth remain
  unqualified; details in frame-targeting-2026-09-22.json.
-036 RF073 confirmed: arbitrary evaluation retry repeats a committed external
  POST after navigation. Temporary native server fixture and maintained regression
  both observe2effects for1instruction; returned error remainsretryabletrue.
  Single-dispatch repair applied;82focused cases/5suites pass15.235s, types/build
  pass. Full driver uh-20260922-163345-07f0358027780d17a733b9ef0ad01760
  passed307.798s:1552tests/124suites;2tests/1suite skipped.73224 consumed.
  Tidiness20260922-163350-8274aa55 failed6s with unchanged1116findings/debt33540;
  one quiet wait consumed. Board81240 consumed:17pending/productfalse. Live035
  unchanged; no pending lifecycle operation.
-037 RF071/RF074/RF075 source qualification complete. SDA shared build
  6034b18af0b24f9b4ffc33a3a1fcb88c7c6b3f6945965338a77194764504fe2b is healthy.
  Owner API12.241s and four package races pass. SDA restart43426 consumed.
  Its approved install operations updated both BAS graphs to js-yaml4.3.2 and
  removed superseded overrides; no unrelated lock resolution changed. Native
  merge-budget probe now rejects correctly and ordinary YAML remains valid.
  Security Health20.186s passes0errors/357warnings/406infos. Exact reviewed hash
  exceptions preserve default rules and detect three counterexamples. API/UIbuild
  passes; package governance44existingwarnings. BAS published in combined038.
-038 RF072 URL download falsely fails on Chromium's expected ERR_ABORTED although
  correct attachment bytes arrive. Native red1.320s reproduced; narrow abort
  handling implemented while event/save remain mandatory. Native URL success,
  abort/no-attachment and unrelated-error controls pass. Focused33-case run had
  one evaluation fixture failure withzeroeffects (32passed); failure context
  was not printed. Added diagnostic assertion; five targeted native cases pass
  1.985s. The40-context probe found32failures and isolated immediate binding cleanup
  as a cause. One session registration/global with unique receipt tokens and
  a bounded internal absent-binding handshake now passes40effects/40next-page
  reads plus20matrixrows. Canonical SDK patch installed throughSDA; final
  focused36/4 pass22.394s; types8184 consumed. No arbitrary script retries restored.
  Types71356/driverbuild pass. Tidiness20260922-165657-106c4f59 terminalfailed,
  admission33748 and one quiet wait consumed; board90943 consumed:17pending/productfalse.
  Final full driver uh-20260922-171531-588ec4e8e9386bbcc9675231167fbaca passes:
  1559tests/124suites,2tests/1suite skipped,366.709s Jest; waiter20434 consumed.
  SecurityHealth5.060s passes0errors/357warnings/406infos; types/build pass.
  Finaltidiness20260922-171537-45b14dd7 failed, one quiet wait consumed:
  1117findings/104long/387complexity/607dup/18coupling/debt33540. Native test
  expansion adds one long-file finding; no floor/threshold relaxed.
  All six recorded source hashes unchanged before combined036–038 publication.
  Gate/stop59912, start31992 and profile84003 consumed. No pending validation.
  Full product still unqualified: durability/crash/overflow/fullrecordreplay,
  native OS/soaks, RF038mutation leases and RF055historical orphans remain open.
-039 RF038 reset lease admission implemented across GoSession/client/driver route.
  Native3red cases3.204s -> focused9/2pass4.248s with open-handle inspection;
  Go driver/session/engine races pass. Runtime3files -77lines. Owner qualifications
  driveruh-20260922-173337-68e0fa749361097f6a4aa0c9df53bc49 passes342.144s:
  1565tests/124suites,2tests/1suite skipped;38932 consumed. APIuh-20260922-173337-7982c7926e5e83a27ee25627d788697f
  passed36.661s;30665 consumed. Tidiness20260922-173343-048f3e9f failed, quiet
  wait consumed:1118/104long/387complexity/608dup/18coupling/debt32643. Whole
  dirty-scenario change; no sole039 attribution. Types/build4732 and API/UIbuild
  70677 pass. Board50728 consumed:17pending/productfalse. Live038 unchanged; no lifecycle
  published039 after fresh quiet gate; stop55720/start29526/profile86231 consumed. RF076 current-owner clean reset still fails onopaqueabout:blank and
  retains old-origin storage; do not claim storage repair. Next consume receipts
  once, retain pending IDs, then continue coherent reset ownership/storage work.
-040 RF076 clean reset repair deployed; details in reset-storage-2026-09-22.json. Context tracks imported/
  visited origins; reset retains primary page/context, clears persistent storage
  through public CDP and per-tab storage via intercepted empty origin visits.
  Manager owns in-flight reset joining, reserves phase before flush, permits
  explicit failed retry and joins before close. Route duplicate map removed.
  Native initial multi-origin red3.166s -> green3.375s; focused114/5pass6.948s,
  types pass. Expanded59-case run57pass/2fail40.85s: storage matrix timeout after
  adding cross-site iframe/cache-only cases, and close-test startup-probe baseline
  counted as reset disposal. Baseline assertion corrected and gate released in
  finally. Diagnostic12939 failed30.946s; test setup suppressed console markers.
  Visible diagnostic7594 failed30.782s and isolates setup tosetContent of a
  cross-site iframe after service-worker registration, before reset admission.
  Native fixture now serves real frame-host/frame-oracle URLs; markers removed.
  Expanded2run87066 passed59/2in6.477s. File-origin adversarial test then fails
 3.109s: WHATWG URL.origin reportsnull while Chromium exposesfile:// storage.
  Tracker now retains that explicit origin. Final focused149tests/6suites pass
  9.609s, including primary-page capture, file origins, partial retry and successful/
  failed reset-close coordination. Types/build pass; no diagnostics remain.
  Full scoped driveruh-20260922-180536-251d1c74ffe41f505e06d9d89f73826c
  passes332.571s (Jest327.966s):1573tests/124suites,2tests/1suite skipped.89109 consumed.
  Tidiness20260922-180542-e4db78b1 failed; admission26369 and one quiet wait consumed.
  Metrics unchanged1118findings/104long/387complexity/608dup/18coupling/debt32643.
  Boardprog_8ded6dad-b2fc-4e13-9b5b-2b84432c3a9b
  remains17pending/productfalse;97945 consumed. Public native oracle confirms
  stale404/missing400 preserve state; current200 clears storage and returnsready.
  Runtime5files net+41lines; combined039–040 net-36, Go cumulative-246 unchanged.
  Open-handle warning still unqualified. Live039 unchanged; no lifecycle operation.
-041 RF077 readiness lifetime repair in source, not deployed. Native losing timer1->0;
  closed-page CDP attempts40->0 and settlement2019.62->0.387ms (single discriminator).
  Normal live-page control preserved; closed readiness rejects explicitly. Focused
  100tests/6suites pass19.47s with no handle warning, types pass. Final scoped driver
  52477 consumed: owneruh-20260922-182410-0f03f688f5567b49d15aa763664bc239 failed
  84.444s on another incomplete inline Page mock (page.off absent). Reset route
  fixture now reuses createMockPage; final98/6pass13.931s. Second owner33499 failed deadline-test contamination; corrected manager fixture
  and third owner2810 passed1579tests/124suites356.381s. No pending driver check.
  Tidiness20260922-182416-a733ef44 failed; admission85469 and one
  quiet wait consumed. Types/build53057 pass. No lifecycle operation. Receiptreadiness-lifetime-
  2026-09-22.json. Runtime2files net-13lines; Go cumulative-246 unchanged.
- OriginalHEAD7d1c7531d057c061c5ad5a67fbecb97ebe2eb194; isolated121probes33met/88failed;
  expandedoriginal debt35603. Detailed prior receipts and measurements remain in
  history and internal/evidence/rehabilitation/. RF061 duplicateRF023 is retired.


045 first green11pass/1fail0.543s: old page-probe fixture encoded "exactly one
lookup" instead of "session closes after startup". It now changes availability
after start, preserving intended lifetime assertion. No production fallback for
an expired lease. Corrected full focused41tests/3suites pass12.406s; types92865
pass. Controlled protocol3/3green: late starts0/0, late detach1, exact new-page
payload only. Native JPEG oracle passes316ms(test)/0.988s(suite): buffer red,
switch to blue, reconnect without another paint, decode every delivered centre
pixel in a separate page; all blue. Native test18695 consumed.
Runtime507->321(-186), one initial/replacement acquisition path; obsolete restart
wrapper and duplicated setup removed. No claimed TS cyclomatic measurement.
Contract/preparation tests pass. Admit final driver/tidiness/build once next.

045 final operations: UnitHealth driver18488 pending (/tmp/bas-cdp-owner-045.json).
Tidiness20260922-192057-fac20ac7 failed;47768 and one quiet wait consumed.
Types/build19496 pass, consumed. Native JPEG18695 passed, consumed. No045
lifecycle admission; live044 remains healthy.
046 read-only next investigation: stream settings still echo targetFps as
currentFps, perfMode changes manager state without updating strategy framing,
CDP quality waits for another restart, and polling sleep retains one abort
listener per completed wait. Reuse existing PerfCollector actual_fps sensor if
settings are repaired; first discriminate lifetime growth at the polling owner.
No046 source edits, dependency changes or owner operations admitted.


### BAS-WORK-045 — final owner qualification, deployment pending

Full driveruh-20260922-192052-24cc5ca0d1cc1f46830b64e44f80ce2a passed:
1602tests/124suites,2tests/1suite skipped,377.515s owner/376.686sJest.
Tool18488 consumed. Types/build pass. Tidiness20260922-192057-fac20ac7 failed,
one quiet wait consumed;1118findings/104long/387complexity/608dup/18coupling,
32643duplication debt unchanged. Boardprog_c1921833-d627-4653-b468-554ec183608c
reports17unqualified/productfalse. Sourcehash matched receipt before managed
restart46214 under BAS-FB-010; restart pending. No production edits during build.

### BAS-WORK-046 — polling lifetime investigation and owner design

Feedback reread. Recall046 found prior stream/ownership work. RF079 confirmed:
20settled waits retain20listeners; one completed capture/stop leaves1deadline
and0protocol detaches. Protocol/timer probe:/tmp/bas-polling-lifetime-red-046.log.
Both expectations fail; warning independently corroborates listener accumulation.
Alternative considered:own private CDP acquisition/disposal in the strategy.
Prefer deleting duplicate protocol/deadline owners if native SDK preserves behavior
and cost. Native alternating20pair comparison median33.39msCDP/33.23msSDK;
all40JPEGs11820bytes at1280x720,DPR1,quality65. Small contended fixture, not a release
performance claim. Evidence:/tmp/bas-polling-capture-cost-046.log.
Scope:existing BAS polling runtime/tests; architecture target documented first.
Validate bounded listener lifetime, stop/page/transport races, fallback support
without CDP and native JPEG dimensions/pixels atCSS/device scales. Capture polling
FPS controls remain RF047; no metric or budget waiver. No046production edits yet.


BAS-WORK-045 publication:managed restart46214 succeeded. API/driver/UI healthy,
0sessions,buildd12561e656243724d39a311ef08248dbb9d1f356ed9cc6ed0eb167f505a85d51.
Profile metadata equals044; three complete reads match original rollback
(/tmp/bas-profile-final-045.txt,tool51293 consumed). All045operations consumed.

BAS-WORK-046 maintained evidence:7failure/2control before repair in0.825s;
focused subset also reports package coverage floor, not an altered policy.
Two native fallback tests with unavailable public CDP fail5.138s before repair.
The same behavioral cases plus controls pass11/11 in1.135s after repair, with
coverage disabled only for focused iteration; full owner coverage remains owed.
Native JPEG centre stays blue and DPR2 dimensions are320x240CSS/640x480device.
Receipts:/tmp/bas-polling-maintained-red-046.txt,
/tmp/bas-polling-native-red-046.txt,/tmp/bas-polling-focused-green-046.txt.


046 final focused42tests/3suites pass13.047s; types/build58039 consumed pass.
Private polling runtime424->372lines(-52):deleted global protocol cache, losing
capture deadline and duplicate capture selection. Controlled probe now20waits->
1active listener->0afterstop;0private sessions/0capture deadlines remain.
First post-repair temporary probe57023 was canceled after discovering its getter
invented a fresh socket each call; corrected fixture owns one socket as production
does. /tmp/bas-polling-lifetime-green2-046.log passes both lifetime checks.
Final qualification pending:UnitHealth23467,tidiness admission52782,board63640.
Receipt:internal/evidence/rehabilitation/polling-lifetime-2026-09-22.json.
No further046runtime edits during qualification.


BAS-WORK-046 full owneruh-20260922-193836-6bc3f69d892b9948027fa4427f92b0f5
passes1612tests/124suites,2tests/1suite skipped,374.920sowner/374.013sJest.
Tidiness20260922-193841-affe382a failed; one quiet wait consumed, unchanged1118
findings/32643duplication debt. Board04617required/17unqualified/productfalse.
Qualified hashes matched before managed restart2856; publication pending.

### BAS-WORK-047 — recording tab ownership investigation

Feedback reread; recall04774hits/10corpora. RF080:three actual callback-owner
lifetime cases fail after cleanup; two page listeners and callbacks survive stop,
and pending popup/title completions publish late. RF081:actual explicit-tab route
plus context listener returns201 but records2entries/2IDs for one page, and callback
ID differs from response. Retained probe:/tmp/bas-page-callback-red4-047.log.
First extension of the temporary probe imported Jest-only helpers (red2) and then
an incomplete HTTP response fake (red3); both fixture errors are retained. Corrected
public Node event/HTTP response fixture gives four discriminatory failures.
Scope:page-events.ts,recording-pages.ts,recording-lifecycle.ts and owner tests.
Consolidate registration and exact-page callbacks; delete duplicate initial-page
navigation. Architecture/issue targets recorded before production changes.
RF038 delayed recording request/frame lease fencing remains separate; no full
record/replay or remote callback rollback qualification is implied.
No047production changes during046managed build.


047 boundary extension before implementation:Session reset and teardown never
invoke pageLifecycleCleanup. Fixing the callback owner alone would leave its
listeners active during reset navigations and page destruction. Include existing
session-reset.ts/session-teardown.ts; release callbacks after recording ACK and
before browser effects, retain failed cleanup for retry through existing teardown
stages. Add maintained manager reset/close ordering regressions. No new lifecycle
manager or shared-scenario extension. First types check rejected an eventType:string
helper; use the existing DriverPageEvent union, not a widened protocol type.


046 publication complete:managed restart2856 and full-profile80046 consumed.
API/driver/UI healthy,0sessions,build687f82ccf2cb92f618739d898303cfc422fa0b2e2d3cc9d1790bd766fdd83538.
Metadata equals045 and three complete identity reads match original rollback.
Receipts:/tmp/bas-live-health-046.json,/tmp/bas-profile-final-046.txt.
047 callback maintained6red/3controls0.483s->23route tests pass0.815s.
Native one-tab identity and cleanup oracle passes367ms(test)/1.188s(suite).
Controlled4cases pass; /tmp/bas-page-callback-green-047.log.
Three callback runtime files currently net-75lines; session cleanup extension
adds required lifecycle calls, final total to be measured after qualification.


047 final boundary qualification:reset/close3cases fail before cleanup hooks.
Initial-page title/stop race adds1real red0.821s and led to moving initial delivery
into the same callback owner. The start route stores cleanup immediately and then
awaits ready; no legacy callback implementation retained. Five runtime files total
1140->1052lines(-88); navigation and initial event policy consolidated, registration
2->1. Existing reset/teardown stage machinery owns cleanup failure/retry.
First expanded focused run105pass/1failure14.624s:retry fixture counted a context
closed during prior capability setup. Record its pre-close count and assert no
additional close before cleanup retry; no production assertion suppression.
Final107tests/5suites pass15.126s; types/build34533 pass. Four controlled cases
still pass after final owner changes. Contract/preparation and inventory047 ran.
Final driver55380,tidiness admission97888,board39816 pending; no more source edits
while qualification runs. Receipt:page-callback-lifetime-2026-09-22.json.


### BAS-WORK-048 — effective stream controls investigation (no implementation yet)

Recall04855hits/10corpora. Reused the retained refactor_stream_probes.cjs settings
subprobe against current sources in/tmp/browser-automation-studio/effective-stream-settings-048.cjs.
Excluded obsolete lifecycle probes that wait for stop before releasing its owned
acquisition; adapted CDPSession.off to the current public SDK seam. Four real
RF047 expectations fail:quality update reports20 while applied start remains65;
FPS1still emits60frames in one simulated second; perfMode=true still emits timestamp
framing; currentFPS reports1with0delivered frames. Existing quality-on-resize
control passes. Evidence:/tmp/bas-effective-stream-red-048.json; controlled clock/
protocol/transport, not native performance. Next scope:manager/strategies/settings
route and owner tests. Apply controls before acknowledging them; report measured
FPS from the existing collector; preserve scale restart policy, frame compatibility,
quality bounds and resource ownership. Evaluate one effective settings interface,
serialized application, and shared delivery policy before implementing; avoid
adding parallel metadata, schedulers or a second metrics owner.
Also rejected a possible frame-slot retention leak by reading current stop:
slot deletion is already generation-checked at manager.ts149. No defect filed.
047driver55380 remains pending; no048production/test-source changes admitted.


047 full owneruh-20260922-195727-36a0b68a336331dfc08f6ca9f11000fc passes:
1623tests/124suites,2tests/1suite skipped,357.329sowner/356.268sJest. Tool55380
consumed. Runtime/test hashes matched before managed restart17293 underFB010.
All047qualifications consumed; managed deployment17293 pending. Source048 has no
changes; settings investigation retains4fail/1control, not release qualification.


048 maintained manager receipt tests:4real failures0.487s (measured FPS0/7.5,
pending quality acknowledged early, rejected quality acknowledged successful).
/tmp/bas-effective-stream-manager-red-048.txt. No048production edits yet.
Implementation decision:reuse the frame slot's serialized queue for updates;
await capture quality application before metadata commits; retain the existing
small strategy control interface and add effective header control. Use existing
collector measured FPS. CDP delivery must replace pending bytes with the newest
frame and own at most one rate-limit deadline, cleared on stop/tab/resize; polling
must cap its existing adaptive controller at the advertised target. A bounded
pending-frame deadline is necessary to deliver the newest stable paint promptly
at the configured limit without a busy polling loop. Validate malformed settings
at the existing HTTP boundary. No new dependency, metrics service or frame format.


047publication17293 and profile44697 consumed:API/driver/UI healthy,0sessions,
buildc95c97174f73f200ecb8c0486ec86a114ccfb54fe27f5db8f0eeb4074002d7cf. Metadata
matches046; three complete profile reads match original rollback. All047ops done.
048first implementation:manager updates reuse serialized slot queue, await quality
and refuse unsupported controls before effects. API route awaits the result.
Measured FPS uses the existing collector; manager15tests pass after4real red cases.
CDP quality reuses generation-safe capture change; failed update stops its capture.
Header mode is effective in both strategies. Polling invalidates captures pending
old quality and caps adaptive max/min to the target. CDP keeps newest pending frame
and one deadline for the delivery limit; stop/page/resize clear it. Dedicated new
regressions and native control proof remain. Existing reconnect test previously
required sending both older and current frames; contractJ23requires latest valid
frame, so new independent payload assertion expects onlyframe2. It fails before
repair; run72573 canceled130 after its failed assertion skipped legacy cleanup.
Moved cleanup to finally, without weakening the desired payload/count assertion.
An exact-string patch refused an unexpected branch and wrote no runtime files;
then the observed branch was updated. Expanded frame tests79337 pending.


048 completed focused implementation:45stream tests pass0.764s; broader61pass0.788s;
first final88pass14.586s. Local SDK source confirms JPEG quality must be integer.
New fractional-quality/disconnected-viewer controls expose2real failures (4valid
controls pass0.631s); reject fractional input and keep FPS application independent
of deferred send failure. CDP's single deferred-send handler observes errors and
uses at most one deadline. Polling removes duplicate target state/update logging;
manager owns the applied settings log. Final90tests/7suites pass14.092s; types/build
25676pass. Two native strategy+real settings-route/localWebSocket cases qualify
encoding requests20,5FPSdelivery pacing and real JPEG timing headers; no release
latency/soak claim. Runtime net+41 for working behavior; recent045–048 net-285.
Contract/preparation/inventory048 pass. Final driver22040,tidiness admission40784,
board11953 pending; no048source edits while qualification runs.


### BAS-WORK-049 — device-scale investigation (no source changes)

Feedback reread; recall04959hits/10corpora. Native SDK/browser and actualCDP
strategy reproduce a device-scale mismatch:atDPR2 requesteddevice receives320x240,
expected640x480. Three controls pass (DPR1CSS/device,DPR2CSS).
/tmp/bas-cdp-device-scale-red-049.log independently decodes JPEG dimensions in a
separate browser page. This is browser-emulatedDPR, not OS/device qualification.
CDP source uses CSS viewport caps and does not read config.scale. Next:measure
actual ratio/metrics in browser, promote maintained native dimension matrix, then
apply scale through existing generation-safe capture acquisition.048driver22040
pending; no049source/test edits while that owner run is active.


048 full owneruh-20260922-202905-d61c4e66c54163bb5a4e772a7be898e8 passed:
1642tests/124suites,2tests/1suite skipped,367.230sowner/366.385sJest. Source/test
hashes matched before managed restart underFB010. All048qualification operations
consumed; tidiness20260922-202910-ed110344 unchanged1118findings/debt32643,
boardprog_9a3d0c21-a86b-4495-a30c-7c625c90eada has17unqualified/productfalse.

049 rejected hypothesis:increasing CDP maxWidth/maxHeight from320x240to640x480
at native emulatedDPR2 still returns320x240 JPEG. Both legacy and CSS layout metrics
also remain320x240, ratio1, while Runtime.evaluate devicePixelRatio reports2.
/tmp/bas-cdp-scale-metrics-049.log. A multiplier alone cannot provide physical
pixels. Next inspect capability selection and SDK polling fallback, retainingCDP
when its fidelity meets the request. No049source changes during048build.


049 implementation decision before edits:manager selects SDK polling for device
scale, includingDPR1; CDP startup rejects unsupporteddevice explicitly. KeepCDP
forCSS and existing fallback policy. A DPR heuristic would add protocol acquisition,
main-world override/zoom ambiguity and startup work without proving future fidelity.
The existing screenshot owner already provides exactCSS/device semantics. Accept
possible compositor-rate tradeoff and measure it; never claim device speed improves
from this fidelity repair. Scope manager.ts/cdp-screencast.ts and existing owner
tests, no new modules or dependencies. Design recorded in ARCHITECTURE.md.
048managedrestart94101 pending; runtime/tests remain frozen until it completes.


048managedrestart94101 consumed0; API/driver/UI healthy,0sessions,build
540a961f084ae145aef9ec7e93862ec4d36f3d15f5bba7a9b82a608ff21ecc0b.
Runtime/test hashes match, profile metadata equals047. Full-profile91228 pending.
049cost experiment:18trials,20frames each,640x480,JPEG65,target30,three rotating
orders perDPR. DPR1median CDP CSS29.34FPS/17.05msfirst, SDKCSS26.51/32.60,
SDKdevice27.13/32.97; DPR2CDPCSS29.33/12.08,SDKCSS27.15/34.13,SDKdevice18.06/58.83.
/tmp/bas-scale-cost-049.json. Animated browser-emulated cohort, shared host during
lifecycle build; no release/performance-regression waiver or latency-band claim.
This suggests a real physical-pixel cost and possibleDPR1 overhead. The current
stream session provider carries onlyPage, and context builder does not retain
authoritative physical scale onSessionState. Do not duplicate profile defaults
or add a guessed page-world DPR probe for this repair. KeepCSS CDP unchanged;
device fidelity usesSDK. Recheck performance with releasecohorts and consider
a qualified capability fast path only if measured demand justifies its ownership.


048 publication complete: managed94101 and profile91228 consumed0. All health
checks pass,0sessions,metadataequals047;three complete profile reads match original
rollback. Receipt effective-stream-controls-2026-09-22.json now records deployment.
049 maintainedred:4failed/3controls passed3.494s; initial focused command also
reported global functions coverage7.34%below15 because coverage was unintentionally
enabled on seven selected tests. Assertions are independently real failures.
Focused checks use coverage=false; fullowner retains all thresholds unchanged.
Two runtime files implement capability selection/rejection, net+3lines.
Focused79875 pending; no049deployment.


049 focused79875 consumed0:76tests/4suites15.130s, including four native
manager/WebSocket pixel-dimension+bluepixel cases. Types/build7788 pass; contract
and inventory pass. Two runtime files net+3lines; recent045–049 net-282.
Receipt device-scale-2026-09-22.json records sources, red evidence, cost cohort,
rejected multiplier and unqualified limits. No source edits during finalowner.


### BAS-WORK-050 — recording admission/lifetime investigation (no source edits)

Feedback reread. Recall050:72hits/10corpora,5589 consumed. RF038 remaining
recording transport does not carry leases; current start route also waits forDOM
after pipeline startup, then admits preview against mutableSessionManager without
checking whether stop/reset/lease handoff invalidated the recording. Test the
bounded stopped-preview resurrection first with controlled DOM delay and actual
route/WebSocket capture. No050source/test edits during049owner qualification.


050 reproduction confirmed: controlled pipeline/DOM, realHTTP routes, native
Chromium and localWebSocket. Stop200 precedes delayedStart200 and one nativeframe;
normal controlpasses. /tmp/bas-record-start-red2-050.log; firstattemptmissingfixture
PLAYWRIGHT_DRIVER_PORT failed beforeverdict, retained/tmp/bas-record-start-red-050.log.
Existing RecordingData already carries monotonic generation, so a new lifecycle
field/manager is unnecessary. Investigate deleting extra5sDOMwait after pipeline
readiness and use existing generation/immutablelease to fence remaining awaits.
No050source/test edits yet;049driver6465 pending.


050 design before implementation: pipeline already verifies readiness and awaits
initial navigation/document activation before returning Start. Delete route's extra
DOM wait, then admit preview immediately under the existing frame coordinator.
Reuse public pipeline.getGeneration() and its monotonic next generation; snapshot
current execution/lease before body read, validate after each async boundary, and
use the same bounded recording provider for frames. Reject superseded operation
without stopping a newer one. No new generation field, timer or lifecycle registry.
Scope existing recording-lifecycle route + owner tests/native fixture. RF038 full
client lease transport remains explicitly open. Architecture target updated.


049 finalowneruh-20260922-204804-a3e358bda829000613649d4a051355ca passes:
1649tests/124suites,2tests/1suite skipped,405.005sowner/403.826sJest. Runtime/test
hashes match. All049qualifications consumed. Managedrestart1852 admittedunderFB010;
no050source/test edits until it finishes.


049 managed1852 and profile13569 consumed: API/driver/UI healthy,0sessions,build
4d7371aa74718651752a1756d800a60868f223f8ca1d2f15fe73be0260a1050c.
Metadata equals048; three complete reads preserve original rollback. All049ops
consumed. 050tests now admitted after049build;10real red cases1.353s include
real recording pipeline/SDK/frame coordinator native test. First implementation
reuses expected next generation and immutable server-admission lease; rejects
stale continuations and frames, removes duplicate DOMwait. Focusedgreen pending.


050 maintainedgreen56786 passes10regressions/2suites1.230s after10realred1.353s.
Native case now uses real RecordingPipelineManager, SDK browser, frame manager and
localWebSocket; capture starts without a secondDOMwait and staysstopped afterlate
load completion. Existing page-events fixture now models real generation/lease
and stopped pipeline state; its callback suppression assertion remains, and it
also requires409 for supersededStart. Broaderfocused35703/types75930 pending.
Runtime currently411->415(+4lines); deleted redundant wait but added necessary
continuation fencing. No standalone size-reduction claim.


050 broader focused35703:126pass/11fail12.861s. All11downstream native cases
failed because new native route fixture used pull delivery and left its own
initial navigation unacknowledged under the shared fixture session ID. Production
retention correctly refused to overwrite that pending buffer. Fix fixture cleanup:
join its pipeline stop, copy its own observations to capturedEntries, then ACK
through the existing owner. No production acknowledgement rule/threshold changed.
Types75930pass; broader focused rerun pending.


050 finalfocused21110 consumed0:139tests/6suites16.932s;137-case priorrerun
98784 also passed16.521s after fixtureACK cleanup. Two further controls preserve
idempotent active retry and reject a newer same-public-ID generation without
stopping it. Types/build36228pass; contract/inventory pass. Receipt
record-start-lifetime-2026-09-22.json. Finaldriver99381,tidinessadmission51482,
board32969 pending. Source/tests frozen;051wire-envelope investigation may use
read-only/temporary fixtures. No051production edits yet.


050tidiness51482/board32969 consumed. Onequietwait for20260922-210754-5fae9158
consumedfailed; boardprog_36aea7d2-24e9-492e-b916-90a19b66737f has17unqualified/
productfalse. Driver99381 remains pending; no edits while it qualifies.

### BAS-WORK-051 — recording wire lease investigation (no implementation)

Reuse050recall for same ownership intent. ActualHTTP route/pipeline-effect fixture
reproduces5fail/3controls after050: missing/stale start and missing/stale/released
stop mutate capture. Releasedstart/currentstart/currentstop controls pass. Evidence
/tmp/bas-recording-lease-wire-red-051.log. GoClientStartRecording lacksowner/lease,
StopRecording usespostNoBody. GoSession holds immutableexecutionID/leaseID but
its recording methods discardthem; live-capture bypassesSession throughClient,
and stophandler callsunownedDriverClient. Convertthatcallerchain, no lookup-based
lease guessing or compatibility fallback. Also observed source-only receipt gaps:
Go start/stop/status types omitRecordingID; handler generatesa newUUID forStart,
omitsStop/Statusidentity, and JSON-any conversion discardsstopped_at becauseproto
expectscompleted_at. Reproduce these separately before selecting their repair scope.
No051source/test edits before050qualification/deployment.


051 Go wire probe confirms5failures with actualClient, ownedGoSession andhttptest:
owned start/stop omit execution_id/lease_id; start/stop/status discard recording_id.
/tmp/bas-recording-go-wire-red-051.log. These are real wire observations, not just
source inference. RF083 registered inPROBLEMS; canonical proto already defines
recording IDs. Actual proto timestamp converter probe is inprogress. No051source
edits while050driver99381 runs.


050 fullowneruh-20260922-210748-62e4a22283c83f9c5ee1c35bec8ed824 failed:
1658tests pass/3fail,123suitespass/1fail,2tests/1suiteskipped,334.400sowner/333.609sJest.
99381consumed. Failingintegration/record-mode doubles lackgetSessionForLease and
pipelinegeneration/state transition; repaired doubles to model existing contracts.
Desired200/409 assertions unchanged. No production code changed after qualification.
Scopedfixture/route/nativechecks andtypes pending; no050deployment.
051actualproto converter confirms timestamp loss: stopped_at input becomesnull
completed_at, whilerecordingID/actioncountcontrols survive. Temporarymodule-child
probe removed afterexecution;/tmp/bas-recording-proto-wire-red-051.log retained.


050fixture repair93165/types28868 consumed0. Existing integration/record-mode
doubles now validate actual owner/lease and model pipeline generation/startcapture;
original200/409 assertions pass. Runtime is unchanged from focused/native-qualified
050. Adapted execution choice: batch final050qualification/deployment with051's
recording wire repair instead of another intermediate6-minuteowner/4-minuterestart.
Prior050fullownerfailure remains retained, not rewritten aspass. Live049healthy;
noowneroperationspending. 051 changes require fresh scoped+full qualification and
a finalcombined source/build receipt. This replaces the earlier self-imposed
050deployment-before051 order; authority/scope/ratchets are unchanged.


BAS-FB-011 captured and answered at goal level: improved reliability and measured
debt, healthydeployed049, still core functional work and17unqualified release
outcomes; no percent/dateclaim. Continuous work proceeds into recording ownership
and complete journey/recovery evidence. No pending owner operations.


051 design selected before source edits: RF038/J17 recording start/stop caller
lease and RF083/J24 receipt fidelity. Baseline /tmp/bas-before-051/manifest.json.
Convert raw live-capture/client calls to existing owned Go Session; remove its
duplicate RecordingConfig mapping, carry owner/lease in driver envelopes, validate
after body parsing before effects, and fence stop continuations by existing
generation. Typed driver-to-proto receipts preserve identity/time; delete random
handler IDs, duplicate response structs and untyped success fallbacks. Scope
existing BAS driver/session/service/handler/converter owners plus maintained tests.
No new service, lease cache, schema or dependency. Broader mutation/operation
identity remains open. Maintained red tests precede implementation; final scoped
Go/driver and full driver owner qualify combined050+051 before one restart.


051 maintained red: five driver cases fail in1.040s; actualGo Session start/stop
wire fails ownership, three handler cases fail identity (stop also loses terminal
time). Source repair now carries leases through ownedSession, fences both stop
awaits, maps typed receipts, deletes handler response copies/untyped fallbacks.
First scopedGo fivepackages pass; driver72pass/1fail was an existing retry fixture
missing its now-required wirelease, corrected without changing the200 assertion.
Caller audit finds UI schema still requiresstopped_at while canonicalproto returns
completed_at; UI also manufactures identity/time for every409, even unknown errors.
Necessary same-receipt scope extension: existing UI API/schema/types owners and
focused API regression. Replace synthetic409 recovery with one authoritative
status read only for RECORDING_IN_PROGRESS; preserve actualID/time or fail. Use
canonicalcompleted_at throughout, no compatibilityalias. Baseline expanded before
UI edits; no new dependency or protocolschema. This closes the caller replacement
boundary; full releaseUI/record-replay remains unqualified.


051 focused/native verification: driver74tests/4suites pass16.571s; actualHTTP
8lease cases pass with zero rejected effects. FiveGo packages pass normal and
race checks; wholeAPIbuild passes. UI9red->9green, fullrecord-mode project and
UItypes pass. Drivertypes/build, contract/inventory pass. Source frozen for owner
4928; receipt recording-wire-receipts-2026-09-22.json records final source hashes.
UI scope includes publicindex reexports from canonicalZod types; duplicate
Start/Stop interface copies deleted. Runtime net-9lines excluding testhelper.
AffectedGo155->157functions/481->502cyclomatic(+21), reflecting added validation;
no new >15function. Prior cumulative-248 is now-227, not claimed as per-cycle
complexity reduction. Existing >15CreateSession/decodeStepOutcome untouched.
Tidiness92714 consumedfailed20260922-213405-21384775; onequietwait admitted,
artifactdetailpending. Board67180 consumed17unqualified/productfalse. UI86431
consumedpass. Driver4928 pending; no deployment or completion claim.


051tidiness finaldetail:1122findings/104long/389complexity/609duplication/19coupling;
debt32663(+20versus050, still2940below expandedbaseline35603). Onequietwait and
artifactdecode consumed. New complexity findings are two tests at13; coupling
21imports is live-capture test with actualJSONwire validation. Production pattern
findings overlap existing proto conversion boilerplate/optional timestamp logic;
no wrapper extraction or suppression to lower metrics. Touched runtimefunctions
remain<=15; preexistinglarge client/session/service modules retain currentowners.
WholeUIrecord-mode457passes9.0s. While4928qualifies frozen source, prepared bounded
liveAPI/native fixture with independentclickcounter, dedicatedtemporaryprofile,
start/status/conflict/rejectedwire/stop/retry receipts and cleanup; run only after
qualified manageddeploy. No saveduserprofile used for fixture. Next investigation
will extend completejourney/recovery evidence rather than claimboardqualification.


051first fullowner4928 consumedfailed: uh-20260922-213400-0c9c8b29cc4e9b10fdee494372853340,
1666passed/1failed,123suitespass/1fail,2tests/1suiteskipped,379.053sowner/378.112sJest.
Only failure is preexistingunit/routes/record-mode terminal-retry fixture missing
wirelease/generation. Original200/retainedreceipt assertion stays; convertfixture.
Adversarial admission review found a regression in051: removing getSession's
prevalidation side effect also stopped refreshing valid caller activity. A long
recording Stop can leave readybrowser immediately idle. Maintainedcleanup-owner
regression fails before repair (/tmp/bas-recording-activity-red-051.txt). Preserve
existing SessionManager.updateActivity after successful ownership admission only;
rejected calls must not touch activity. No newfield/owner. No051deployment yet.


051activity repair/fixture conversion1781 consumedpass:81tests/5suites15.960s;
types/buildpass. FinalactualHTTP8wirecases pass after adding onlyactivitymethod
to controlledmanagerdouble. Source/test hashes refreshed and frozen for70689
fullowner; firstfailedowner remains immutable. Runtime net-7lines excludingtesthelper.
No other pending owner operation. Next livefixture after qualifiedmanagedrestart.


### BAS-WORK-052 — next journey/ownership investigation, source untouched

Recall native recording/replay qualification: search-hub53689 consumed, combined
smoke-flow program/skill72153 read. Saved-workflow smoke is suitable once a
representative exact workflow exists; it is not a recording-admission oracle.
Prepared051liveAPI/nativeclick/receipt fixture first. Read-only caller audit for
remainingRF038input reveals three unowned paths: GoSession.ForwardInput omits
its lease, HTTP handler bypassesSession, and WebSocketCreateInputForwarder has
a separate50-lineHTTPtransport bypassing existingdriverclient/session ownership.
Next discriminating probe /tmp/browser-automation-studio/recording-input-wire-052.ts
uses actualHTTProute and nativepage independentclickcounter for missing/stale/
released/current leases. Prepared, not yet run; no052product/test-source changes
while051owner70689 qualifies. Remove duplicateWebSocket transport only after
showing existing sharedclient preserves timeout/connection behavior and authority.


052nativeHTTP/SDK probe confirms remainingRF038input mutation: missing/stale/
released leases each return200 and click the independent nativefixture counter
once; currentowner controlpasses. /tmp/bas-recording-input-wire-red-052.log.
Each case used its own nativecontext and closed it; no userprofile or external
site. No052production edits. Additional source-only question: websocket Hub
launches an independent goroutine per input, so orderedtransport does not prove
ordered browser effects. Verify separately before extending repair into input
serialization; existing HTTPclient already pools and drains responses, and the
forwarder can preserve its2sdeadline through normalcontext instead of private
transport. Driver70689 remains the only pending owner operation.


052 RF020 confirmed with actual GoHub/WebSocket transport and an independent
forwarder-effect log: send1, wait until its handler is admitted and held, then
send2; observed effects[2,1] and overlaptrue. /tmp/bas-input-order-red-052.log.
This is stronger than source-only inference but not fullbrowser/OS gesture
qualification. Both commands belong to one websocket and one fixture session;
no saveddata. Current readPump spawns one goroutine per input; no inputcoordinator
found in driver source. Select ownership/ordering repair after051nativejourney,
with boundedbackpressure, exactlease and errorreceipts still requiring design.
No052source/test edits.


051fullowner70689 consumedpass:uh-20260922-214339-29c7eefe3ef57778e98b523e71bf09fd,
1668tests/124suites,2tests/1suiteskipped,363.586sowner/362.562sJest. Finalsource/test
hashes match frozenreceipt. Managedrestart admittedunderFB010; no052source/test
changes until deployment completes. Priorfailedreceipts retained.


051post-qualification caller review found a zero-count edge before managed51012
finished: canonicalprotoJSON omits scalar action_count=0 (sharedrespondProto uses
EmitUnpopulated:false); UI Stop schema still requires thefield. Actualproto
serialization plus actualZodschema fails /tmp/bas-recording-zero-red-051.log.
No source mutation while managedbuildruns. After51012 settles, add maintained
zero-count regression and honor canonicalproto default in schema; preserve
explicit invalidcount rejection. If Zodinput/output types differ, adjust existing
validation helper type signatures without changing runtimepolicy. FinalUIchecks
and another managedbuild will be required; do not mark051fullydelivered yet.


051managed31664 consumed0; API/driver/UI200, originalprofilemetadata preserved.
LiveAPI/nativefixture passes23checks including independentclick, exactstart/status/
stop ID/time, known409, missing/stale rawdriver rejection, retainedstopretry and
owned session/temp-profile cleanup. /tmp/bas-recording-live-051.json. No fullrecord/replay/recovery qualificationclaim.
Zero-count maintainedUIred39462 confirms protocoldefaultedge. Extend existing
UIvalidation type signatures (RecordingApiService.validate/sharedsafeParse) to
allow distinct parsedinput/output for Zod.default; runtimepolicyunchanged. Baseline
sharedsafeParse captured before edit. Repairzero count, then fullrecord-modeUI/
types and finalmanagedrestart; driver/Go source unchanged, existingreceiptsretain
applicability without repeating full driver.



051zero-count repair verified: canonicalprotoJSON->Zod probe now preserves0;
593UItests pass (458record-mode+135shared)12.7s, wholeUItypespass. The selected
red had one actualfailedassertion; reporter misleadingly labelled all10cases
failed when9were filtered, so no10-failurebehaviorclaim. Final fullproject run
executes themall. Existing parsed-input/output type signatures now allow canonical
Zoddefault; no shared runtimevalidationpolicychanged. Driver/Go sources unchanged
from qualified1668suite and fivepackage/race/build receipts. Fullprofile51465
preserves3reads against originalrollback. Firstdeployedbackendbuild
de50c2f729c628409d2385346b306b0935b5ef9c2ae3c34e3b71d2672de27718.
Finalsourcehashes updated forUIedge; managedrestart nextunderFB010.


051finalmanaged31664/profile40361 consumed0: API/driver/UI200,0sessions/recordings,
finalbuildbdf9f35ac1782b71c71f9270dae0d657c814fcf990f69b7b0a6ab01a3f1392b0.
Metadata equals049 (temporaryfixture deleted); three completeprofile reads still
match originalrollback. Finalsource/test hashes match; contract/inventorypass.
050+051 receipts marked qualified/deployed, no pending operations. Source/runtime
net-6excludingtesthelper; priorfailedruns remainretained. RF038 start/stop and
RF082/RF083 observedbounds repaired; broader inputleases/order/fulljourneys remain.
Proceed052using proven nativelease and actualWebSocketordering red cases.


052 feedback reread; no pending operations. Scope selected beforeimplementation:
RF038input caller lease and demonstratedRF020single-WebSocket ordering. Baseline
/tmp/bas-before-052/manifest.json records10existing runtimeowners. ConvertSession/
live-capture/HTTP/WebSocket callerchain; remove private50-lineHTTPforwarder and
unusedpostRaw helper, preserve2scontextdeadline and sharedpooling. Reuse recording
leasehelper, refresh admittedactivity and fence browser sub-operations. Serialize
forwarding in existingWSreadloop for boundedreceivedorder, without newqueueframework.
Cross-client/HTTP appliedsequence/coalescing/reconnect remain explicitRF020 scope.
Maintainedred tests must precede edits; native052red andactualWSred are retained.
Caller audit also found registered test:e2e:record-mode mjs harness still uses
pre-lease sessions/actions and is not covered by driverJest. ActualnegativeHTTP
fixture proves its exit0/11passes despite two500clickresponses,500actionread and
500workflowgeneration. /tmp/bas-recording-harness-oracle-red-052.log. RegisterRF084;
it cannot substantiate nativequalification. Repair its caller/contracts/oracles
after inputcontract stabilizes; do not treat priorowner1668pass as coverage ofit.


052implemented existingowners: GoSession->driver carriesimmutablelease, live-capture
service owns bothHTTP/WSforwarding, privateWebSockettransport andunusedpostRaw
removed. SharedrecordingOwner reused for driverinput; parsesbeforelookup, updates
onlyadmittedactivity, revalidates between pointermove/down/up andbefore200. WS
readPump awaitsinput with existingbackpressure, no newqueueframework. Driver
maintainedred3fail/1control->4pass; GoSessionandWSred->green. Extra body-read/
pointersuboperation handoff tests pass. Focused59826=57tests/4suites1.336s+types;
13041=fiveGo packagespass;59366=native4casespass. GoHub actualwire green[1,2].
All operations consumed; no052deployment. Runtime-24lines; Go186/740unchanged
against /tmp/bas-before-052. Need caller/deadline tests, scopedraces/build, final
owner qualification andmeasurement. No sourcefreeze until thoseadmitted.


052 resumed qualification: /tmp/bas-input-go-race-052.txt completed with actual
race failures in four WebSocket tests; driver/session/live-capture/handlers race
checks pass. Chained API build was not executed. Driver bundle build succeeded
(/tmp/bas-input-driver-build-052.txt). No shell/owner process remains pending.
Recall search72365 consumed; returned BAS usage already read and no narrower
hub repair owner. Extend existing052 hub boundary for RF085 before final freeze:
Run deletes clients under RLock, and readPump changes subscription fields and
sends confirmations without the hub lock. These can race readers and closed Send
channels. Use existing hub mutex for membership, subscription updates and channel
lifetime; do not hold it during browser input HTTP. No new transport/queue owner.
Existing failing race tests are retained before repair. Latest overall status
reported deployed1668-driver/593UI, preserved profiles, duplication debt-2940,
17unqualified outcomes; no percentage/date or readiness claim.


052 RF085 repair9858 consumed0: fiveGo packages pass -race and wholeAPIbuild
passes. Existing hub mutex now protects registration/welcome/drop/subscription/
confirmation channel lifetime; execution broadcasts use exclusive lock when they
can remove clients. Subscription dispatch moved under one locked method with
membership check; input waits remain outside it. Existing race failures repaired;
new disconnect and blocked-input/global-hub tests pass. No framework/extra locks.
Driver source/tests frozen for fullowner92634; baseline/final hashes and measured
deltas retained in live-input-ownership-2026-09-22.json. Scope remains perconnection
order, not fullRF020. Tidiness/board admission in progress; consume IDs next.


052 tidiness and board consumed. Test Genie 20260922-222031-1f5b47d9 failed
the unchanged duplication budget; its single quiet wait and native artifact
decode are complete. Findings: 1123 total, 104 long files, 391 complexity,
609 duplication, 18 coupling. Duplication debt remains 32663 (-2940 from the
expanded original baseline). The synchronized hub adds one measured Go branch
(740 -> 741 across affected runtime owners); cumulative reduction is now226.
Runtime net-17 lines after removing the obsolete interface comment. Existing
large hub subscription switch is one cohesive wire-protocol dispatcher under
the shared lock; splitting it does not count as reduced complexity. All source
hashes refreshed; final driver owner92634 remains pending. Contract and inventory
checks pass; board71023 reports17unqualified/productfalse. Prepared post-deploy
live fixture with stale/missing input rejection and independent HTTP/WS clicks.


052 full driver owner92634 consumed with exit0. Exact owner verdict/counts are
retained in /tmp/bas-input-driver-owner-052.json. Frozen relevant source/test
hashes match the receipt; tidiness budget still fails and release rows remain
unqualified. Managed deployment under FB010 is next. Do not change relevant
product/test source until that build and live checks complete.


052 owner details: uh-20260922-221932-3fbe69da0f2156e9e51185e32472ffad passes
1675 tests / 124 suites, 2 tests and 1 suite skipped; 348.210 seconds owner,
347.344 seconds Jest. Managed restart50777 is the only pending operation.
Source is frozen. Existing FB010 authorizes this restart; saved profiles must
match original rollback after health and live input checks.


Next bounded RF084 repair design, no product/test edits while052 deploys:
replace the obsolete registered harness calls in place, using current session
lease/operation envelopes and typed TimelineEntry actions. A local fixture owns
its independent click counter. Stop must flush capture, retrieval must validate
entries/count and explicit ACK, then replay into a fresh context must increment
the independent counter. Current optional API generation also omits required
project_id and downgrades all errors; selected API coverage must create an owned
temporary project, persist a workflow, verify it, and delete workflow/project.
An unavailable optional API remains an explicit skipped/unqualified scope; an
API selected as available cannot silently downgrade later failures. Cleanup
failures affect exit status. Preserve original false-green negative evidence.


052 delivered. Restart50777 consumed0; API/driver/UI are healthy on build
3c0dbef7d4bebe647501cde6a4b263cb66175dafc708498c34106d3d398b49f2. Native live
fixture96017 passes26 checks with exactly two independent clicks (HTTP and WS).
Rejected missing/stale raw input adds no effect. Session/temp profile cleanup
succeeds and no browsers remain. Metadata equals051; full profile94534 matches
original rollback across three reads. Source/test hashes match. RF085 observed
races repaired and deployed; bounded RF038 input/RF020 order repairs qualified,
not their whole release journeys. No pending operations. Continue RF084 harness.


### BAS-WORK-053 — RF084 recording E2E harness

Feedback reread; continuous scope and restart authority unchanged. No pending
operations. Reuse native-journey recall and BAS smoke-flow guidance already read
in052; exact saved-workflow program is available once a fixture workflow exists.
Baseline /tmp/bas-before-053/manifest.json captures the existing registered
harness and package scripts before editing. Original negative fixture proves
exit0/11passes despite rejected clicks/read/generation and zero effects. Replace
that producer in place: bounded strict requests, current ownership and typed
actions, independent local fixture, recording/ACK/fresh-context replay, selected
API persistence and cleanup. Add maintained producer-adversary checks through
the native Node test runner; no dependency installation or production runtime
changes are needed unless a real native assertion exposes an owner defect.


053 implemented. Six maintained producer tests failed against the original
harness (52667,5.826s) and passed after strict request/outcome/cleanup handling.
Initial native run96579 passed9 checks, but adversarial review found the new
fixture replaced identity cookies on every visit. A reused-context producer
regression then correctly exposed false qualification (identity-red artifact).
Fixture now retains existing identity, so reuse cannot masquerade as fresh.
Contract14499 passed9 tests/0.891s and native final passed9 checks with two
independent effects, distinct retained context identities and owned cleanup.
Committed entries are fsynced before ACK; generation failure also acknowledges
already-saved entries during cleanup. Added two further fault cases for200/failed
replay outcomes and malformed recording JSON; final contract pending. API/driver
have0sessions/recordings after native cleanup. No backend source changes/restart.
Receipt recording-harness-2026-09-22.json records hashes and exact test scope;
Full saved-workflow execution was the next qualification gap selected for054.


053 qualified registered harness. Final contract13161 passes11 tests/1.039s;
native final14499 passes9 checks with two independent effects and all owned
cleanup. Result/committed entries are retained in /tmp/bas-recording-e2e-9MbmTr.
Harness253 ->225 lines (-28); new contract fixture161 lines and one script
registration are test infrastructure, not runtime debt reductions. Production
source is unchanged and052's1675-driver/five-Go/race/build receipts remain valid.
Tidiness20260922-224342-f9d0a473 failed its unchanged debt budget; one quiet wait
and native artifact decode consumed. Board38241 still17unqualified/productfalse.
Metadata matches052; native driver has0sessions/recordings. No pending operations.

Next investigation: execute the saved generated workflow through its actual API
owner and compare the fixture's independent effects. Existing smoke-flow was
read (052 combined read plus local current contract); it writes mandatory Memory
learning feedback/preference/avoid records with no declared opt-out, conflicting
with this engagement's file-only tracking. Use its existing underlying public
WorkflowsService.ExecuteWorkflow with wait_for_completion=true for the bounded
fixture test; do not add a second production execution/wait implementation or
weaken the owner's outcome checks. Native effect/cookie proof remains scoped.


### BAS-WORK-054 — saved recording execution investigation

Feedback reread, no pending operations. Reuse053's native local fixture and
current owned API contracts. Baseline /tmp/bas-before-054/manifest.json preserves
harness/test sources. No product or maintained test edits yet. Temporary probe
/tmp/browser-automation-studio/recording-workflow-054.mjs adds exact-revision
ExecuteWorkflow with the owner's wait, one further independent effect and
timeline receipt. Its cleanup previews retention using both unique fixture
project/workflow filters and checks the exact execution before confirmed removal.
Keep owner evidence in the fixture artifact directory. The smoke-flow program's
mandatory learning writes make it unsuitable for this file-only engagement;
the same authoritative public execution owner remains in use.


054 temporary native probe3267 passes11 checks. Execution
5bcd9d8c-95bb-4e9d-bbc4-9e35cf71ff3c completed through the API, with three
independent effects total and three completed timeline steps (navigation,
generated wait, click). Evidence /tmp/bas-recording-e2e-biYej9 retains execution,
timeline and scoped retention receipts before deletion. Retention removed only
that execution; driver ends with0sessions/recordings. Promote the existing
producer in place, with exact workflow revision and node/step evidence checks.
Five maintained added execution/retention tests fail before promotion; original
full API execution was not claimed by053. Source now includes native owner calls
and adversarial test cases. Final contract/native shell40194 is pending.
No backend/runtime code has changed. Read-only next angle: successful execution
timeline contains recorder-generated console.error diagnostic messages on an
otherwise quiet fixture page; inspect configuration and owner before naming a
new defect or changing instrumentation behavior.


054 qualified maintained producer40194:15 contract tests pass6.491s; native
11 checks pass with3 independent effects. Exact saved revision executes as
navigation/wait/click; every expected node has completed/successful timeline
evidence. Execution63257377-f05d-48f7-8bc2-dca1ea41cda1 and only its artifacts
were removed by scoped retention after JSON evidence was retained. Fixture
project/workflow and both direct sessions were cleaned. Receipt
saved-recording-workflow-2026-09-22.json; artifact dir /tmp/bas-recording-e2e-nmZrN6.
Tidiness20260922-225019-d7a9c1a5 failed unchanged debt budget, single wait/decode
consumed. Board60334 still17unqualified. Production source unchanged; existing
052 runtime checks remain applicable. No pending operations. This qualifies a
basic generated workflow case, not the full24-journey/release matrix.


### BAS-WORK-055 — RF086 truthful console evidence

Recall73086 consumed; BAS improve/usage guidance already loaded. No relevant
prior console-diagnostic repair found. Native054 timeline plus source establishes
routine recorder diagnostics polluting application console, including false
error severity. Baseline /tmp/bas-before-055/manifest.json captures the sole
runtime file. Structured readiness and event/delivery diagnostics already exist;
remove redundant routine console output rather than filter telemetry. Preserve
actual application errors and the script's real fatal-initialization report.
Add native passive/active regressions before editing the injected script.

Post054 profile metadata matches053. A later zero-session assertion failed:
observability shows10 recent executing sessions for workflow99f8ccb9-46ff-405c-
ab7d-a41f85cf74e4, distinct from054's already-cleaned fixture workflow
87f0cf72-21a6-4b91-9b19-2a40fa83f57d. One owner execution already completed by
the subsequent read; origin of this new traffic is unverified. Do not attribute
it to a054 leak or claim the service is globally idle. Continue isolated SDK
qualification; FB010 remains the existing restart authority, without a new
permission request. /tmp/bas-session-observation-055.json retains the observation.


055 maintained native red42103: passive and active cases both fail only the
quiet-console assertion; real application error sentinel, readiness/event
telemetry and active click capture all pass first. /tmp/bas-console-native-red-055.txt,
2 failures/29 skipped,1.890s. Remove15 unconditional routine console statements
from the injected script. Keep genuine fatal initialization reporting and all
structured diagnostics unchanged. No collector filter, replacement logger,
wrapper or dependency. Focused capture/telemetry tests, types/build are next.


055 focused67825 consumed0:56 tests/3 suites pass17.360s, whole driver types
and build pass. The native passive/active console controls now pass while
application-error sentinel, readiness/event telemetry and click capture remain
intact. Before source repair,054 API timeline retained13 recorder console entries,
including3 false errors. Source is frozen for full driver owner and tidiness/board
admitted next; record shell IDs from their admission output. No055 deployment
yet. Receipt recording-console-evidence-2026-09-22.json captures source/test
hashes and injected byte/line reductions; Go unchanged, JS complexity unknown.


055 tidiness/board consumed: run20260922-230018-5c4ce09a failed unchanged
duplication budget; single wait and native decode complete. Board13449 remains
17unqualified. Full driver90898 remains pending. No product source edits.
Read-only remaining mutation audit finds unowned raw driver calls for ACK,
navigation, viewport/stream settings and replay, including a Connect recording
caller. Prepared actual HTTP/real-buffer ACK probe056 (controlled session owner)
for missing/stale/released/current envelopes. No056 runtime or maintained tests
changed while055 qualifies. Next scope should repair the remaining interactive
control ownership boundary coherently, with data-loss ACK admission first.


055 full driver90898 consumed0, owner receipt passed and relevant frozen hashes
match. Managed deployment under existing FB010 authority follows; no056 source
changes until build/live checks finish. The new056 actual HTTP/real-buffer probe
confirms RF038 ACK loss: missing, stale and released envelopes all return200
and hide the pending entry; current-owner control also returns200 as expected.
/tmp/bas-ack-wire-red-056.txt. This is a real buffer/HTTP result with controlled
session ownership, not native browser or full durability qualification.


055 delivered: managed86971 consumed0, all three health endpoints200 on build
dcdb9940abc520d41d7808af27069f6869e3275cd07320b1d47517a4a9e01c5b. Native
workflow14129 passes11 checks/3 effects; comparable timeline recorder console
entries13 (3 errors) ->0. Application errors remain observable in maintained
native passive/active controls. Metadata matches054; profile30735 confirms three
complete reads against original rollback. Full driver1677 tests passes. Runtime
-15 lines/-1158 injected bytes, no added abstraction or collector suppression.
RF086 resolved for this demonstrated boundary. No pending operations. Proceed
RF038 remaining interactive mutation ownership with preserved056 ACK red proof.

### BAS-WORK-056 — RF038 leased recording acknowledgement

Feedback reread; latest overall-status request remains resolved, continuous goal
and restart authority unchanged. Recall search returned no relevant new repair
program; source-ledger recall timed out while other providers returned results.
Reuse existing Session authority, recordingOwner and registered native harness.
Actual HTTP/real-buffer red proof /tmp/bas-ack-wire-red-056.txt: missing, stale
and released callers all return200 and hide the unacknowledged entry; current
owner is the positive control. This is controlled ownership, not native browser
qualification. Four runtime files frozen into /tmp/bas-before-056/manifest.json
before edits; Go complexity baseline /tmp/bas-go-complexity-before-056.txt.

Target: retain the Go Session selected before pull/commit; ACK only through that
immutable lease. Remove raw-interface/mocked unowned ACK path; preserve durable
commit-before-ACK and exact receipt checking. Driver admits after full body parse
and performs synchronous buffer acknowledgement only while operational. Reuse
existing lifecycle helper, with no lease cache or transport abstraction. Add
real-buffer owner rejection/body-handoff and valid selective/retry controls,
Go wire/closed-owner/unknown-owner checks and real HTTP ACK in journal fault
regressions. Validate scoped owners/races/build, native live workflow, tidiness,
board, managed lifecycle and preserved profiles. Broader interactive routes,
post-admission browser effects and full release outcomes remain unqualified.

056 red/repair qualification: maintained driver76436 failed8 cases (7 ownership
rejections plus accepted-call activity); Go wire red failed for omitted owner and
lease. Deployed055 native fixture also fails exactly the new owner-negative
assertion, HTTP200 for missing owner, and cleans its temporary data:
/tmp/bas-ack-native-red-056.txt, /tmp/bas-recording-e2e-lLOvsu. New source passes
actual HTTP/real-buffer missing/stale/released/current cases. Focused75326 passes
44tests/2suites0.877s plus whole-driver types; harness23542 passes16 contract
tests6.591s. Go47929 consumed0: three affected package race tests and whole API
build pass. Journal fault tests now send a real HTTP ACK, verify commit-first,
retain exact IDs and prove a Session replaced during commit cannot supply the
new owner's authority. Unknown destructive pull is rejected before any read.
Closed Go Session cannot send an ACK; malformed ownership is rejected before HTTP.

Runtime changes: four existing files, +21 lines. Raw ClientInterface ACK and its
mock removed; no new authority cache/transport/adapter. Affected Go115/445->116/451
functions/complexity (+6), cumulative reduction now220. This is necessary
correctness cost, not complexity reduction. Frozen hashes /tmp/bas-frozen-056.json.
Driver owner17226, tidiness admission42587 and board70165 pending; no managed
restart yet. Receipt recording-ack-ownership-2026-09-22.json records current scope.

056 tidiness initial run20260922-232613-643fe005 and one quiet wait consumed:
failed,1126 findings/104long/392complexity/609dup/20coupling, debt32663 unchanged.
Review attributes new findings to the API owner type import and stronger journal
fault-test matrix (complexity14,21imports). Simplified API owner resolution to the
existing side-effect-free GetSession call with inferred type, eliminating that
extra import and nested declaration. Read-only pulls still accept absent API
ownership; destructive pulls require a nonnil admitted owner. This reduces runtime
lines without moving code. Final Go race/build running in shell21119.
Driver source/tests unchanged during driver owner17226. Board70165 consumed:
17unqualified/productfalse. Test matrix complexity remains an explicit test cost;
no extraction to hide it. Re-run tidiness once for the changed Go source.

056 final Go21119 consumed0; three affected race packages and whole API build
pass. Final tidiness12953 run20260922-232924-3e1f0c18/quiet wait/artifact decode
consumedfailed:1126/104long/393complexity/609dup/19coupling, debt32663. Runtime
net+17 lines; affectedGo115/445->116/452, cumulative reduction219. Both new
complexity findings and test import count are disclosed; no threshold changes.

Full driver17226 failed1 old fixture assertion (HTTP200 expected for unowned
ACK),1684passed/2skipped. Owner uh-20260922-232608-d9db78ed7f5ef0409e8c5ef63f15937b,
388.519s owner/387.357s Jest, retained receipt. Converted both ACK/retry calls in
existing tests/unit/routes/record-mode.test.ts to an owned Session and asserted
lease identity; original non-destructive-read/exact-ACK/retry oracles preserved.
Focused37259 consumed0:50tests/3suites1.183s and types pass. No product source
changes. Refrozen hashes include the converted test. Full owner37095 and managed
restart79365 now pending; run concurrently on frozen source after all remaining
1684 tests and repaired focused fixture pass. Existing FB010 authority applies.
Prepared /tmp/browser-automation-studio/recording-live-056.py adds native API
clear/ACK/journal-preservation checks to the prior052 fixture; execute after restart.

056 delivered: managed79365 consumed0. Build
f635a2dd3f9c883948b789dacc02729e341e463d1fb6f68d499e0d080f6b73c9,
API/driver/UI all200. Native workflow44358 passes12checks/3independent effects,
including missing/stale ACK rejection and preserved driver entries; saved workflow
execution/timeline/scoped cleanup remain qualified. Artifact /tmp/bas-recording-e2e-BPR5bm.
Native API41324 passes35checks/2effects: destructive pull carries ownership and
leaves the already-committed journal identical, while hiding driver entries.
Owned session/profile cleanup passes. Profile74915 consumed0: metadata matches055
and three full contents match original rollback. Full driver37095 consumed0:
uh-20260922-233411-e41c7817d38372d572afccbbc5b3eafc,1685tests/124suites,
2tests/1suite skipped,377.933s owner/377.130s Jest. Frozen hashes match. Receipt
recording-ack-ownership-2026-09-22.json is qualified_and_deployed. No pending056
operations. RF038 remains open for other interactive controls and later effects.

Next adversarial pass057: feedback reread. Native managed-driver navigation probe
confirms missing and stale ownership both return200 and load the fixture once;
current owner is the positive control. Three scoped native sessions all close200.
/tmp/bas-navigation-native-red-057.json retains independent HTTP requests, with
/tmp/browser-automation-studio/recording-navigation-057.py as producer. No057
production edits yet. Inspect navigation caller and history ownership across HTTP,
Connect and Go Session; repair all four navigation mutations coherently and
remove duplicated route completion policy where the behavior supports it.

### BAS-WORK-057 — RF038 owned navigation and shared completion

FB012 captured verbatim before action and answered with overall goal status;
continuous work explicitly reaffirmed. Native056-build probe proves unauthorized
navigation effects; no missing capability or approval. Existing Session and
recordingOwner are the chosen owners. Baseline /tmp/bas-before-057/manifest.json
captures12 runtime files before edits, including callers/types on both runtimes;
Go baseline /tmp/bas-go-complexity-before-057.txt. Architecture target updated.

Repair URL navigate, reload, back and forward coherently. Convert Go HTTP, history
Connect and first saved-tab restoration callers; remove unowned ClientInterface
methods/mocks. Share the identical history-navigation request/response/commit
policy without aliases. Driver lease+page fencing follows all awaited completion
steps and precedes history/cache/callback/success publication; retain supported
wait/timeout/capture/history semantics. Browser effects already admitted while
owned are not reversible and remain a separate interruption/reconciliation gap.
Maintained adversarial cases must cover missing/stale/released/non-operational
owners, body handoff, delayed navigation/title/page replacement, valid bounds,
exact options and history outcomes. Scope tests to changed driver/Go callers,
then relevant owner/tidiness/board/native fixtures and managed/profile checks.

057 implementation checkpoint:12 runtime files changed; shared driver navigation
completion now fences lease and page after awaited navigation/verification/title/
screenshot/thumbnail before history/cache/callback/success. Maintained red94166
has32failed adverse cases and4passed owned controls (the earlier fixture did not
model legacy getSession activity; corrected before production edits). Source
passes36 corresponding cases; whole-driver types pass. Existing unowned navigate
fixture converted; focused47953 passes86tests/3suites1.402s. Added four history
bounds/null-result controls afterward; their focused/types shell is pending.

Go now has one owned NavigateHistory request/response/client/Session path instead
of three unowned copies. API history controls share decode/ownership/commit/response
policy; URL controls, Connect history navigation and first-tab restore carry the
immutable Session. Removed old raw-interface methods and unowned mocks; preserved
public endpoint/JSON shapes and navigation journal/page broadcast assertions.
Go32599 passes all five affected race packages and whole API build. Wire controls
cover four Session commands, closed/unknown owners, missing lease/options, invalid
history operation, Connect path, history journal failures and saved first-tab
restoration. No057 deployment/full owner/tidiness admission yet. Deployed056 is
still healthy. Source restoration and additional-tab/page/preview ownership remain
RF038 scope; no whole-boundary closure claim.

057 scope extension, 2026-09-22: native RF087 history probe consumed1 with two
real failures and successful cleanup; /tmp/bas-history-native-red-057.{txt,json}.
The just-added null-result unit oracle was wrong for same-document movement;
replace it with browser-entry movement/no-movement assertions. Browser CDP
history replaces the divergent private map, with short-lived attachment cleanup
and existing lease/page guards. Extend baseline before edits to server.ts async
route and UI useBrowserNavigation/BrowserChrome (15 runtime paths total). API
NavigationStackEntry timestamp becomes optional; UI popup stops requiring it
or a nonempty title. Add scoped UI hook checks and types plus native browser
history cases. Architecture updated before extension. Previous final bounds
94818 consumed0 (46tests/types) is interim evidence, not RF087 qualification.

057 qualification admitted (2026-09-23 UTC): focused driver53397 consumed0,
63tests/1suite0.807s and whole-driver types. UI59547 consumed0:460/460 record-mode
tests (2.5s tests/9.4s total), plus UI types. Initial UI project name was invalid;
corrected to registered record-mode. Initial UI type check caught an optional
timestamp predicate mismatch; explicit canonical return type fixes it. Five Go
race packages and API build39721 consumed0 after optional timestamp field update.

The private history map/getter/mutators/cleanup exports are deleted. Driver
read/mutation paths use per-page native history; CDP attach/query/detach and all
awaited continuation boundaries retain lease/page fencing. Tests cover browser
script-created/untitled/same-URL entries, null successful responses, no movement,
bounds, tab replacement and attachment cleanup/errors. Popup parsing accepts
missing timestamps and empty titles. Runtime15paths7787->7254 (-533lines);
/tmp/bas-frozen-057.json freezes24 runtime/test paths. No new dependency.

Full UnitHealth driver99587 pending. Tidiness79983 consumed1: run
20260923-001802-49f182c4,1128 findings; one quiet wait consumed1, artifact
extraction pending. Board23483 pending. Managed restart admitted with existing
FB010 authority; retain source freeze. Prepared native green scripts in
/tmp/browser-automation-studio for owner rejection, hash/same-URL/script/tab
history and API navigation/journal checks; red receipts remain untouched.

057 measured review: tidiness native artifact decoded at
/tmp/bas-tidiness-native-final-057.json:1128 findings/105long/393complexity/
610duplication/19coupling; duplication line debt31591 (prior32663, -1072;
expanded original35603, -4012). The finding count rises while repeated line
debt falls; budgets remain unchanged and verdict remains failed. New long-file
finding is the existing Go Session test module after adding ownership transport
regressions; no runtime file newly crosses the long-file threshold. Existing
live-capture test import count rises to22 but coupling finding count stays19.
AffectedGo177functions697->700complexity (+3; cumulative reduction216).
Shared-tree/JS-TS measurement limitations RF064 remain; line count is not a
substitute for complexity. Policy reduction: one history-navigation transport/
completion and browser history authority replace three Go copies and private
session counters. No new compatibility aliases or dependency cycles.

Board23483 consumed0:17 required/17 pending_telemetry/unqualified, product false.
Contract and include-untracked inventory pass. Managed78926 and full driver99587
remain pending; sources/tests frozen. Review explicitly leaves API-side page
notification/journal attribution after a concurrent tab/session handoff and
post-admission external effects under RF038; this iteration does not claim
complete end-to-end interruption or retry identity. Recheck these alongside
remaining raw page/viewport/preview controls after057 native qualification.

057 deployed native verification: managed78926 consumed0; build
7794187bd9281a31cf9880df318aa037a38dfee7e0997c4bd9b4105f9d97339b,
API/driver/UI all200. History66091 passes49checks: missing/stale owners rejected
for all four mutations with identical history/no requests; hash and same-URL
pushState traversal, script-created entries, independent tabs, untitled history
and cleanup pass. Owner93028 passes6checks; invalid owners0fixtureloads, current
owner1load. Native API passes48checks/2click effects, including URL/back/forward/
reload/stack and existing recording/ACK/journal controls. Native79376 passes
12checks/3effects with fresh replay and saved-workflow execution/timeline, artifact
/tmp/bas-recording-e2e-b7kF51; executioncd0bc943-fbc6-4ffb-97a4-29a87c217c92.
All scoped fixture objects cleaned. Profile9241 consumed0:3fullreads matchoriginal
rollback; metadata matches056. Frozen hashes match. Receipt
recording-navigation-history-2026-09-23.json retains the native red/green results,
source hashes, measurements and limitations. Full driver99587 remains pending;
no other pending057 operation. RF087 awaits this final qualification receipt.

057 complete as a repair cycle, not the continuous goal: full driver99587
consumed0, uh-20260923-001758-aab8b8c7e98e4a7a6a981b2264c4c06a,1742tests/
124suites,2tests/1suite skipped,395.680s owner/394.782s Jest. Native, UI, Go,
profile and frozen-source qualification above applies to deployed057. RF087
resolved for this boundary. Receipt status qualified_and_deployed; no pending
operations. Next058 tests the source-observed API completion attribution gap
in RF038; do not claim the entire ownership boundary is finished.

### BAS-WORK-058 — RF038 navigation result attribution across handoff

Feedback reread; no new instruction. Recall search59796 consumed0,67hits.
Relevant results are the already-read progress/architecture and existing BAS
programs; no program owns this fault-injection repair. Source-ledger memory/
scopes recall timed out; this limits recalled history, not local repair authority.

Hypothesis: reload/back/forward completion relooks up Session and active page
after HTTP navigation; URL completion retains Session but reads its later active
page. A tab switch or reused-session handoff can attach the old result to the
wrong page/journal. Use maintained HTTP driver fixtures plus real recording
journal to discriminate. Test all four operations with stable ownership, tab
switch, Session replacement, and missing/unknown original browser page identity.
Preserve old-page attribution when a valid completed effect is received; reject
missing/foreign receipts instead of guessing the currently active page.

Pre-edit runtime baseline /tmp/bas-before-058/manifest.json covers7paths. Existing
Session/page registry and PageTracker remain owners; no extra map/cache/layer.
If reproduced, carry the driver's original registered page ID in navigation
receipts, bind API publication to the retained Session and resolve that exact
page. Full interruption/atomic journal-generation isolation remains distinct
RF038 work. No058 production edits yet.

058 red18959 consumed1:19adverse failures/4stable positives against057 source.
Initial red44781 also contained3UUID/string test-oracle mismatches; corrected
before final red and before production edits. Browser page ID is already owned
by session.pageToIdMap; reuse it in receipts. Architecture target updated. Go
baseline retained at /tmp/bas-go-complexity-before-058.txt.

058 partial implementation: retained original browser page IDs and Go Session
attribution now pass23handoff cases with race detection (71327). Extended driver
fixtures pass71tests and types (9117), five Go race packages/APIbuild pass24832.
No058 full qualification or restart yet. Deployed057 remains unchanged.

Necessary extension before qualification: source review shows initial driver-page
mapping exists only after recording-start callback. Native API probe then exposes
RF088: new-page201 is treated as transport failure503 despite an independently
observed second active browser tab; the API still lists one page and recording
start overwrites its initial identity with the new tab. Final native red has
3failures/9passes including cleanup. Extend baseline before edits to seven more
runtime paths (14total): Session manager/session/PageTracker, live-capture, page
callback ingress, driver session-start and session types. Repair authoritative
admission/creation registration and valid201 handling; no callback-only workaround
or guessing the current page. All previous058 tool operations consumed; no pending
owner runs. Broader page/viewport leases and journal-generation atomicity remain
RF038; this extension preserves usable multi-tab behavior for the new receipt
checks.

058 extended source checkpoint: shared Go transport accepts completed201 and
CreatePage validates its page identity;202remains rejected. Admission returns
active_page_id, retained by Go Session to initialize its first mapping before
recording. PageTracker.AddPage now atomically installs/reuses a driver mapping;
separate MapDriverPageID method/callers deleted. Creation/restoration use the
existing live-capture path and register before returning; callback created/initial
paths reuse that identity and initial means current active page, not authority
to overwrite the original tab. Missing callback identity rejected.

Focused30908 consumed0:81tests/2suites15.802s and driver types. InitialGo74319
failed because CreatePageResponse lacked its wire Title field; added it and
converted fixtures/imports. Final95367 passes5Go race packages and APIbuild.
Tests include23API attribution cases,12navigation receipt controls,4CreatePage
status/identity cases, concurrent registration convergence, callback duplicate/
initial identity and pre-recording creation receipt registration. Frozen23paths
/tmp/bas-frozen-058.json;14runtime6654->6721(+67lines). Expanded Go baseline
194functions753->772complexity(+19), cumulative affectedGo reduction197. This
is correctness cost with atomic registration replacing split map mutations,
not a claim of per-cycle complexity reduction. No production source over500lines
is newly introduced. Native green producers prepared with second tab before
recording and third tab during recording. No058 deployment/full qualification
yet; admit once on frozen source and retain prior red receipts.

058 frozen qualification admitted: full driver58718 and managed restart31043
pending. Tidiness72070 consumed1,run20260923-004638-faac4f1c; one quiet wait and
artifacts command admitted. Board9404 consumed0:17pending_telemetry/unqualified,
product false. No threshold or requirement relaxed. Profile/native verification
must follow the managed restart; green scripts prepared under
/tmp/browser-automation-studio/*-058.py without overwriting red evidence.

Read-only critique while frozen: RestoreTabs still has separate first-tab
metadata/active-tab restoration paths; inspect them with native saved-tab
restoration after this build. Existing raw creation/activation/viewport leases,
callback retry journal identity, PageTracker pointer-read concurrency and full
record/replay tab qualification remain open. Do not claim RF038 or J03 complete
from page registration alone.

058 deployed native verification: managed31043 consumed0, build
2f4d1767153ac6a2b4f4ca5a521a6b93ab28f5ec36070179a984f68c076bdca5;
API/driver/UI all200 and frozen hashes match. Native81196 passes18tab-registration
checks: second tab before recording, navigation before recording, stable original
page after activation, third tab during recording and one registration per page.
Native84554 passes48API checks/2click effects. Native36162 passes49history and
6owner checks; both logs explicitly inspected. Saved-workflow65282 passes12checks/
3effects, /tmp/bas-recording-e2e-zCuByc, executione1450f42-1f19-4123-b752-23d875dcdfbc.
All scoped fixture objects cleaned. Profile99563 consumed0,3fullreads matchoriginal
rollback; metadata equals057. Full driver58718 remains pending.

Tidiness one quiet wait/artifact decoding consumed:1130findings/105long/
394complexity/611dup/19coupling,debt31604(+13 versus057; original35603). New
complexity finding is the23case API attribution test at17; test imports also rise.
This coherent matrix tests independent operation/handoff combinations and real
journal effects; wrapper extraction would not reduce its aggregate policy.
Production touched functions remain below15 except pre-existing broader handlers/
restoration owners; no new long production file. No budget changes. Receipt
navigation-page-attribution-2026-09-23.json retains baseline/native/tests/metrics;
status awaits full driver result. Only pending operation58718.

058 full driver58718 consumed1:uh-20260923-004633-b8372f1c6d8bd91142603c65769f2fc1,
1749passed/1failed/2skipped;123passed/1failed/1skipped suites;409.647s owner/
408.439s Jest. The unrelated AI action error-duration test observed49ms after
a real50ms timer. Actual assertion retained; replace wall-clock scheduling in
that existing fixture with controlled Jest timers, then rerun its suite and the
full owner. RF089 records the cause; old receipt remains at
/tmp/bas-driver-owner-final-058.json and failure excerpt at
/tmp/bas-driver-failure-058.txt. No retry without this fixture correction.

Clarification of prior058 complexity review: no newly over-threshold runtime
Go finding; the new17-complexity finding is the test matrix. RF064 means the
JavaScript/TypeScript runtime complexity is not fully measured.

058 RF089 focused57954 consumed0:29AI executor tests/1suite1.196s, unchanged
>=50duration assertion with controlled timers and guaranteed cleanup. Added only
this existing test to frozen manifest (24paths); deployed runtime source unchanged.
Full driver3045 pending at /tmp/bas-driver-owner-qualified-058.json; initial
failed owner remains preserved. Tidiness35281 consumed1, final test snapshot
run20260923-005527-08b9379c; one quiet wait92054 consumed1 and artifacts44154
consumed0. Its CLI rebuilt from concurrent shared sources; native extraction
is pending. Board058 remains17unqualified.

Next059 recall43251 consumed0,60hits; source-ledger memory/scopes timeout remains
a limited recall gap. Read docs/internal/BAS-PRESERVATION.md result: it concerns
LPBS marketing payload preservation, not live browser profiles, so no extra
workflow applies. Existing BAS/architecture/owner skills already loaded.
Native57694 consumed1:RF090 saved-tab restoration4failures/26passes, all owned
cleanup succeeds. /tmp/bas-restored-tabs-native-red-059.json retains pre-save
selection, reopened page list and independent native activeURL. First/middle
active cases reveal empty initial URL, lost first-tab selection and API/native
active mismatch after non-last selection. No059 runtime or test edits; keep058
source frozen until full driver3045 completes, then repair this owner.

059 target documented from native RF090 evidence. No059 source/test changes
while058 full driver3045 is pending. Proposed bounded owner repair: apply
first-page receipt metadata, retain first/non-last selection through the loop,
and call existing ActivatePage so browser/API state converge; remove duplicated
raw-switch policy and stale assumptions about unknown initial IDs/missing
CreatePage titles. Required regressions cover all selected positions and
canonical receipt URLs/titles, with explicit failure behavior. Preserve scope
limits for raw page leases and full profile durability. Original user-profile
metadata still equals058 after the temporary profile investigation.

058 final tidiness35281/92054/44154 fully consumed: run
20260923-005527-08b9379c,1130/105long/394complexity/611dup/19coupling,debt31604,
unchanged from the pre-timer test snapshot. Artifact
/tmp/bas-tidiness-qualified-native-058.json. Only pending operation3045.

058 delivered: final full driver3045 consumed0,1750tests/124suites,2tests/1suite
skipped,384.876s owner/383.926s Jest. Final owner
uh-20260923-005523-c8d59b592e3b34f354cc4a1dd5f7cdb3. RF088/089 qualified with
the native/Go/profile receipts above; source/runtime unchanged by controlled
timer fixture. Final tidiness metrics unchanged after that test edit. No pending
operations; receipt qualified_and_deployed. Continuous goal advances to native
RF090 restoration defect, not completion.

### BAS-WORK-059 — RF090 saved-tab location and selection agreement

058 full qualification complete; feedback reread. Native red already retained
with4failures/26passes and cleanup. Baseline before edits captures existing
live-capture/service.go only at /tmp/bas-before-059/manifest.json; Go complexity
/tmp/bas-go-complexity-before-059.txt. Architecture target precedes implementation.
Use a maintained HTTP driver fixture with independent active-page state to test
first/middle/last/no saved selection, canonical browser URL/title receipts and
failed selection. Restore first-page metadata, retain desired page through all
creations and reuse ActivatePage; keep existing public result shape and avoid
redundant switching. No driver/UI changes intended for this bounded state repair.
Full profile failure recovery and raw page-command leases stay explicitly open.

059 maintained red completed before production: all5 restoration table cases fail
(/tmp/bas-restored-tabs-maintained-red-059.jsonl). Green7test events including
parent and existing initial-navigation ownership regression pass. Runtime removes
raw final driver switch, stale saved-title/URL publication and first-page-selection
assumption; PageTracker receives initial receipt and ActivatePage owns final state.
Explicit service error retains partial result for failed final switch. Existing
handler logs restoration failure; comprehensive partial-profile recovery remains
unqualified. Source/test frozen, Go races/build36915 and managed95198 pending.

059 delivered: managed95198 consumed0; native84863 passes30/30 vs4failed/26passed
red. API/driver/UI200,build5f18d8953ff84a2ca8447679a854b83a45634d3a18078bb5c382bf8221dcc47c.
Go36915 passes3racepackages/build; profile74170 passes3originalrollback identity
reads; API metadata equals058. Workflow8100 passes12checks/3effects,artifact
/tmp/bas-recording-e2e-oMqOaf,executionab8d4906-8472-42ed-855b-dcd462f237a9.
Only058 frozen live-capture service/runtime-test changed; full driver1750test
receipt remains applicable to unchanged driver.059 final hashes match freeze.
Runtime844->793(-51),Go26functions112->117(+5),cumulativeGo reduction192.
RestoreTabs19complexity remains cohesive bounded restoration; no wrapper
extraction or lower thresholds. TG3055 consumed1,run20260923-011045-8c6b3b79;
onequietwait consumed1/artifact read0.1130findings/105long/395complexity/610dup/
19coupling,debt31604. Finalboard31238 consumed0:17unqualified. Contract/inventory
pass. Receipt saved-tab-restoration-2026-09-23.json;RF090 demonstrated state
agreement resolved, full fault recovery/raw leases retained as open limitations.

060 recall48546 consumed0:63hits/10corpora, existing BAS skills/board reuse;
source-ledger agent-memory/scopes timed out. No new workflow selected. Static
inspection found new-page route suppresses all goto errors. Initial nativeprobe
returns201withchrome-error://chromewebdata/ for reserved/refusing localhostport;
explicit-failure assertion fails. Probe then used nonexistent profile GetRPC
(404/nonJSON), so saved-profile follow-up is invalid; owned cleanup still runs.
Retain /tmp/bas-failed-navigation-native-red-060.json and producer; correct the
read through documented public reopen/pages flow before claiming durability evidence.

### BAS-WORK-060 — RF091 failed page/restoration admission

059 fully delivered/no pendingoperations; feedback reread. Corrected nativeprobe
uses public reopen/pages instead of nonexistentGet and completes2failed/8passed.
Failure201Chromeerror becomes persisted blank on reopen. Target docs/issue
precedeimplementation. Baseline4runtimepaths /tmp/bas-before-060/manifest.json.
Hypothesis:new-page blanket catch plus best-effort restoration/APIadmission turn
failed effects into successful partial profiles. Discriminating tests must reject
failednavigation, close/remove onlynewpage, preserveoriginal selection, surface
cleanup errors and reject failedrestore without profilecommit. Native retry after
fixture recovery must retainoriginalURLs. No leaseownership completionclaim.

060 red evidence complete: native restore-retry58056 consumed1,22passed/8failed
on059. Both initial/additional outage cases falsely return200, leak an admitted
session, touch profile metadata and lose saved URLs after close/retry. Corrected
new-tab native2failed/8passed retained separately. Go44366 consumed1:2service
restoration failures+3handler admission/cleanup/cancel cases fail; driver94375
consumed1:3navigation failure cases fail/9pass. Go65523 green allfocused cases.
Driver9677 shell0 concealed focusedJest3fail/80pass because later types passed;
inspection caught the failing log:missing logger.error in the test mock after
realerrorpath became reachable. Addedmockmethod;no assertion weakened.
Callback20439 red1fail/11pass:close during pendingcreated delivery is lost. Added
explicit active-lifetime recheck and ordered closed publication/registrycleanup
after created settles;stopped generation still suppresses follow-up. Shared
unregisterRecordingPage removes duplicated route/callback registry mutation.

060 focused61909 consumed0:95tests/3suites pass,types pass. Go21077 consumed0:
6racepackages and wholeAPIbuild pass, both logs inspected. Freeze8paths at
/tmp/bas-frozen-060.json before fullqualification/managedrestart. Native producers
copied to newgreen destinations, redartifacts retained.

060 managed63623 consumed0, API/driver/UI200,build
0e27920d3c75ae3bc3864f35f3ff34266c2c9d5def98125a33170cc361f61700.
Native63137 consumed0:10failed-navigation checks,28outage/retry checks,30normal
restoration checks allpass; separate logs inspected. Native redretry had30checks
because false200required2extraownedcleanup operations; green no false-admitted
sessions exist. Allsameproductassertions retained. Profile47415 consumed0:
3completeidentityreads matchrollback. Driverfull75292 remainspending.
TG64460 consumed1,run20260923-012327-8acc15e0,onequietwait1/artifactread0.
1131findings/105long/396complexity/609dup/20coupling,debt31604. Addedhandler
failurematrix16complexity and21testimports; no newproductionlongfile threshold.
Go44functions261->262(+1);cumulative reduction191. Fourruntime2252->2275(+23).
No netpercyclecomplexityreductionclaim. Board99098 consumed0:17unknown;contract
and inventorypass. Initialadditionalrecording-modeprobe used nonexistentstopRPC;
503failed-navigationreceipt is valid but completeprobe invalid. Corrected via
existing nativeproducer's /{session}/stop route; retain initialartifact.

060 native recording-mode corrected probe passes12checks; original guessedstop
route failure retained separately. Workflow23776 consumed0:12checks/3effects,
executionac9a5fe7-6bd5-4f02-935a-76cfbc0249e6,artifact/tmp/bas-recording-e2e-i51PZz.
060 frozen8paths unchanged; metadata still equals059 after ownedprobe cleanup.
Only pending060 operation is full driver75292; do not edit its affected sources.

061 independent nativeinvestigation85764 consumed1:RF038 rawtab control
8failed/32passed, /tmp/bas-tab-authority-native-red-061.json. Missingandwrong
leases both allow new-page201 andactive-page200; independent nativecurrentURL
changes. Eachcase's valid APIcreate/activatepositive controls and scopedcleanup
pass. No061source/testedit yet. Planned ownerconversion:Session.CreatePage and
SetActivePage carryimmutable execution/lease,livecaptureservice retainsadmission
owner,rawdriverclientsrequirelease,drivernew-page/active-page reuse existing
recordingOwner admission/completion instead of pre-bodyGetSession. Move final
selection after awaited title and ownership checks. Testhandoff duringbody,
newPage/goto/title;dispose newlycreatedpage on interrupted completion and leave
replacementownerstate untouched. Nativepublicrestore/workflow controlsmustremain
passing. RestoreTabs must retainitsoriginalSession acrossitsmultiplecommands,
not re-admit a replacement byIDmid-transaction. Preserve underlying rawviewport/
preview/callback/retryidentity issues as separateRF038 boundaries.

060 delivered: full75292 consumed0,owneruh-20260923-012321-317459b050cdf064f55aaeb9df17f701,
1754tests/124suites,2tests/1suite skipped,395.498s owner/394.626s Jest. All8frozen
paths stillmatchqualification. Receipt qualified_and_deployed; no pendingowner
operations. ContinueRF038tabauthority061; greenchecks do notcompletegoal.

### BAS-WORK-061 — RF038 owned tab creation and selection

060completequalification;feedbackreread. Baseline4runtimepaths retained at
/tmp/bas-before-061/manifest.json. Native8failures/32passes demonstrate missing/
wrongleaseadmission. ConvertretainedSession commands and driveradmission/completion
usingexistingrecordingOwner; noalternateownershipregistry. Testandfixlease
rejection beforeeffects, ownership changes duringasyncpage operations, and
originalSessionretention throughrestoration. Keepexecutionpage switching working.

061 maintainedred: driver14fail/14pass at /tmp/bas-tab-authority-driver-red-valid-061.txt;
initialdriverload failedmissingmocklogger.debug (retained), correctedbeforebehavior
red. Go3leasepayloadcases fail;2handoffcasesfail. Go21focused and driver16authority
casesnowpass; all111focuseddrivercases/types pass aftervalidleasefixtureconversion.
No061deploymentyet. Scopeextension api/handlers/record_mode.go:060failedrestore
cleanup re-resolvesSessionID andcouldclose a replacement afterownershiploss. Retain
an admission-bound closeoperation in internalSessionResult; handlerusesit. Existing
journal-registrationfailure cleanup must also closecapturedSession. Baselineextra
handlerpath appendedbeforeedits. Add a discriminating handlerhandofffixture; the
newresultfield/testreceiptis structuralpreparationbeforechangingcleanupdispatch.

061 coupling review found importing recordingOwner from lifecycle into pages
would add a lifecycle->page-events->pages->lifecycle runtimecycle. Beforefinal
qualification, relocate the unchanged shared validator to a smallrecording-ownership
module importing onlypurestate/errors and erasedSessionManager type. Convertall
fourroutecallers, removetheoldexport withnocompatibilityshim. Baseline includes
all3additionalexistingmodules andnewmodule; movement is not complexityreduction.
Existing validation module owns selector/replayhandlers and is not an authority
module, so mixing ownership there was rejected. Go6racepackages/APIbuild and
111focuseddriver/tests pass beforethisimport repair;Go unchangedafterit.

061 finalscoped45279 consumed0:141tests/4suites/types pass after sharedvalidator
relocation. Go94851 consumed0:6racepackages including multipageexecutor and
wholeAPIbuild pass. Handlercleanup60183 red4cases; boundadmissionclose fixes
currentByIDcleanup, registrationfailurealsousesretainedSession.Close. Onlytwo
SessionResultconstructors exist andbothsupplyclosecapability; internalJSONexcludes
callback. Freeze9runtime/7testpaths /tmp/bas-frozen-061.json beforefullqualification.

061 pendingoperationsadmitted once:full driver40525 (/tmp/bas-driver-owner-final-061.json),
managedrestart57167,tidiness38420,board/contract/inventory44863. Nine runtimepaths
4551->4597(+46); Go142functions575->145/585(+10),cumulative reduction181.
Correctness/ownershipcostexplicit; relocatedvalidatorhasno claimedcomplexitysaving.
Do notedit061frozen sources/tests untilqualification settles.

061 managed57167 consumed0:API/driver/UI200,build
072657b2b9bae08735248abd27f96b38bc08f67eba36d81c75bd31fc42364376.
Native57385 consumed0:40authoritychecks vs8redfailures,28outage-retry,30normal
restoration,12recording-modefailurechecks allpass;eachloginspected. Profile14978
consumed0:3identityreadsmatchrollback;API metadata equals060 after062probe.
TG38420 consumed1,run20260923-014500-9fb4b86a,onequietwait1/artifactread0.
1132findings/105long/396complexity/610dup/20coupling,debt31588(previous31604,
original35603). Board44863 consumed0:17unknown,contract/inventorypass.
Pendingfullowner40525 andnativeworkflow45005;16frozenpathsunchanged.

061 workflow45005 consumed0:12checks/3effects,artifact/tmp/bas-recording-e2e-vFT3MJ,
executionef29d5f2-e2a3-485e-a4c3-c27dde81927e. Onlypendingowner40525.
062recall15611 consumed0:72hits/10corpora,source-ledger.agent-memorytimeout;
existingBASownership/qualification workflow reused. Nativeinitial-navigationprobe
complete4failed/11passed on061:redirectmetadata stale,failedinitialURL returns200,
uncommittedbrowserremains,profilemetadata touched. Ownedcleanup complete;no062
runtime/testedits while061fullqualificationpending. RF092+architecturetarget
recorded. Share initialnavigationreceipt owner across CreateSession/RestoreTabs;
fail admission andclose retainedSession if navigation fails. EmptyURL remains
valid; publicInitialURLfield remainsrestoration-scoped. Nativeprobe didnot overlap
workflow run during its session-count assertions.

061 full40525 consumed1,owneruh-20260923-014455-c6f6541e8994419eaf9362f171992eb3:
1769passed/1failed/2skipped;123passed/1failed/1skippedsuites,387.954s owner/
386.902s Jest. Failure nativepipeline tabfixture stillcalled newpageroute without
lease andstub onlygetSession, then waited2000ms forcallback instead of checking
400rejection. This is a missed caller migration, notflake; retainfullfailedreceipt
and /tmp/bas-driver-failure-061.txt. Convertfixture tocurrentlease/operational
phase andrealgetSessionForLease prototype; assert201beforewaitingcallback. All
identity/listener assertions and2000ms boundunchanged. Runtime unchanged/no extra
restart required. Add testpath tofreeze afterfocusednative passes; rerunfullonce.

061 nativefixture45459 consumed0:1focusednative testpass/30unrelatedskipped,types
pass,1.566s. Requiredleasefixture andsame2000ms callbackbound pass. Initialfreeze
retained /tmp/bas-frozen-initial-061.json; final17paths /tmp/bas-frozen-061.json
adds convertedpipeline integrationtest. Runtime unchanged; admitonefinalfullowner
and finalscopedtidiness onthis test snapshot.

061 final operations admitted once: full driver73080 and tidiness22680. Current
checkpoint rewritten to remove stale059/060 pending statements; append-only
history and all receipts retained. No062 source changes before finalqualification.

061 finaltidiness22680 consumed1,actualrun20260923-015349-bddc7735. An initial
wait usedanincorrect unissuedID andreturned124; retained at
/tmp/bas-tidiness-qualified-wait-061.json, nottreatedasrunstate. Correctrunthen
received its singlequietwait,failedbudgetverdict;artifact retrievalcompleted.
Onlypendingvalidation remainsfull73080.

061 finaltidinessnative decode complete, source/testfreeze unchanged. Receipt
tab-command-authority-2026-09-23.json retains failedfull and currentpending73080.
Prepared six062 initial-navigation regression cases in /tmp/bas-initial-navigation-tests-062.txt
without editing repositorytests; notyetexecuted. Do not count draftcases asproof.

061 delivered: final73080 consumed0,owneruh-20260923-015346-275ef9e7917c16db6c18220f748ec251,
1770tests/124suites,2tests/1suite skipped,383.779s owner/382.897s Jest. Final
17pathfreeze matches. Finaltidiness unchanged1132/105long/396complexity/610dup/
20coupling,debt31588. Receipt qualified_and_deployed, no pendingoperations.
ContinueRF092initialnavigation062; goal remains active.

### BAS-WORK-062 — RF092 initial-navigation receipt/admission

061fullqualification complete; feedback reread. Native4failures/11passes retained.
Baselineone runtimepath /tmp/bas-before-062/manifest.json. Applypreparedsixcase
regression matrix to existing live-capture service tests and establishmaintained
red before runtime edits. One initial-navigation operation will own receipt/page/
Sessionvalidation for bothfreshadmission andrestored firsttab. Propagatefailure,
closecapturedSession underboundeduncancelledcontext, preserveemptyURL support.
Driver/UI runtimeunchanged; useGo races/APIbuild andnativequalification.

062 maintainedred completes:5failedcases (redirect/navigation/wrongreceipt/cleanup/
cancelled) andblankURLpositivecontrolpasses. SharednavigateInitialPage applies
actualreceipt,validatesretainedSession/tracker andinitialdriverpageidentity;
CreateSession andRestoreTabs bothcallit. Firstfocusedgreenpassesallinitial/restore/
legacyjournalfailure/ownershipcases. Beforebroaderqualification, unifyregistration
andnavigationfailure cleanup in a singledeferredpost-admissionowner, joining
cleanupfailure without closing a replacementSession. Namedreturnerror makes
cleanupobserve everyexplicitfailedreturn; successfulreturntransfersclosecapability.

062 broad4847 consumed0:6racepackages/APIbuild pass. Review removes two dead
unifiedRecordingSvc nil guards after the function already rejectsnil; callbacks
andjournalregistration are required,notoptional fallback paths. Requalify touched
service/handlers afterthis branch deletion,thenfreeze/build/deploy.

062 final5616 consumed0:service/handler races andwholeAPIbuild pass after removing
deadguards. Earlier6racepackages passed4847; no driver/UIruntime changes. Freeze
2paths /tmp/bas-frozen-062.json, runtime815->824(+9),Go28functions125->29/127(+2),
cumulative reduction179. Admittedonce:managed36539,tidiness3811,board68587.
No fullJest rerun needed for unchangeddriver;1770test061receipt retained.

### BAS-WORK-062 — 2026-09-23 UTC — initial navigation admission qualified

BAS-RF-092/J01/J03/J07/J15. Native red4failed/11passed and Go red5failed/blank
control passed proved stale initial metadata and false successful admission.
One shared initial navigation receipt owner now validates captured Session/page
identity and writes actual URL/title. One bounded uncancelled cleanup owner joins
errors, replacing duplicate failure cleanup. Removed two dead null guards.
Native14initial/28retry/30restore/12workflow checks pass with three fixture effects;
profiles preserved. Six Go race packages/API build pass; final service/handler
races/build pass after branch removal. Driver source unchanged from061 full owner.
Two-path final hashes verified. Managed36539 completed; tidiness20260923-020516-92f01203
and board/contract/inventory completed. No pending operations.
Runtime815→824(+9), Go28→29functions and125→127complexity(+2); cumulative Go
reduction179. Tidiness1133findings/31588duplication line debt; all17release rows
unknown. Small increase buys explicit admission semantics; no per-cycle complexity
claim. Receipt evidence/rehabilitation/initial-navigation-admission-2026-09-23.json.
Fresh independent063probe exposes mutable page reads:9failed/1passed plus two
JSON serialization races. RF093/architecture target recorded; next repair owns
page snapshots without duplicating registries or changing public fields.

### BAS-WORK-063 — 2026-09-23 UTC — detached page snapshots qualified

BAS-RF-093/J01/J03. Independent9failed/1passed and two races; maintained13leaf
failures plus service receipt/status/final-selection failures confirmed aliasing.
One detached Snapshot replaces two list traversals and split selected-ID reads;
registration/read receipts deep-copy pointers. Final close clears selection.
29focused tests,6Go racepackages/build and independent10cases pass; native
14navigation/28retry/30restore/12workflow pass. Profiles preserved. Managed32137
succeeded; frozen4paths unchanged. Tidiness20260923-022346-f81bc017 fullyconsumed;
board17unknown, contract/inventory valid. Runtime+7, Go complexity+4 (175cumulative
reduction), duplicate line debt31562. Sorting lock-time improvement unmeasured;
no performance claim. Receipt evidence/rehabilitation/page-state-snapshots-2026-09-23.json.
New independentRF094 probe proves API tab closure never reaches browser:2failed/
11passed with owned cleanup. Next work closes that owner boundary; no pending
operations or completion claim.

064 implementation decision: driver explicit close returns its selected page after
actual close; Go applies that receipt under retained Session ownership. Existing
PageTracker.ClosePage will return a stable close event and preserve first ClosedAt
on repeated observation. Command handler and callback use that receipt; existing
journal primary-key idempotency prevents duplicate committed closure. Share driver
removal/selection rather than another tab registry. External callback generation
authority and arbitrary simultaneous-close reconciliation remain RF038. Boundary
within BAS; baseline eight runtime paths /tmp/bas-before-064/manifest.json.

064 boundary extension within BAS: close receipt with no selected page must clear
UI hook and session-store selection; current truthiness guard retains the closed
tab and command path omits store updates. Include usePages and maintained hook
regressions in the same close-command repair. Driver removal3cases red; handler
4cases red; native2failed/11passed. Initial green compile missed fmt import in
mock (corrected); no assertion changed. Scoped driver55tests/types and handler
4cases now pass; broader qualification still pending.

### BAS-WORK-064 — 2026-09-23 UTC — browser tab closure qualified

RF094/RF038,J03/J07/J17. Native2failed/11passed proved the API acknowledged a
closure it never sent to the browser. Driver removal3cases/handler4cases/UI3cases
failed before repair; rejected UI close remained a positive control. Additional
missing-selection red was corrected before qualification. Actual leased close,
retained owner validation, shared driver removal/selection and stable PageTracker
close event now replace local false completion. Existing journal idempotency
handles callback/retry; UI selection/close application updates view and store.
Native50close/14navigation/28retry/30restore/12workflow checks pass; profiles
preserved. Go8racepackages/build,55driver/types,8UI/types and full1785driver tests/
124suites pass. Owneruh-20260923-024236-a7894f4177340893a636b805f9145e59 (383.406s);
managed71124 succeeded; all16frozen hashes unchanged. Tidiness20260923-024243-c4dc5295
consumed; board17unknown; contract/inventory valid. Runtime+85/Go+21complexity;
cumulative Go reduction154, duplicate line debt31605 vs original35603. Per-cycle
increases are explicit; no threshold change or performance claim. Receipt
`evidence/rehabilitation/browser-tab-closure-2026-09-23.json`. No pending operations.
Next read-only UI tab-creation/state-ownership hypotheses are retained for065;
confirm with tests before introducing another issue or repair. Continuous goal active.

065 UI consumer extension: RecordingSession onPageCreated currently sends a second
selection command even when admission already selected the tab; its selection
callback reads the previous render's page map. Use current canonical store state
for these two decisions while preserving popup auto-selection and activity UI.
Baseline now includes this existing consumer. Exact red detail shows repeated
creation event callbacks in both rendering modes, not StrictMode-only duplication.

065 deployed build6b927bd16895ce9533b97953062061ff6abba06e14fe49692679ec6a0371a000
is healthy; native8canonical-admission/50close/14initial/28retry/30restore pass.
Three full profile reads match rollback. First real-UI probe verified visible
creation/selection, then failed its assumed recording=false precondition: this
screen deliberately auto-starts recording (RecordingSession effect). Retain
/tmp/bas-ui-tabs-native-065.{json,txt}; it is fixture failure, not passing UI
qualification. Corrected producer explicitly stops the owned recorder after
awaiting the UI start receipt, then retains the same callback-free UI assertions.
No source change or relaxed assertion. Restart31936/TG34281/board23502/native70860/
profile95261 all consumed. Corrected UI native and saved workflow still owed.

### BAS-WORK-065 — 2026-09-23 UTC — canonical UI tab admission qualified

RF095/J03. Maintained UI6failed/7passed, native receipt2failed/6passed and an
additional delayed-callback metadata failure confirmed the owner gaps. Canonical
Page receipt now reaches201 clients; restore uses its existing identity. One
session-store page owner and one async admission/completion boundary replace local
mirrors and repeated writes. Callbacks run outside React state updaters and join
creation by page ID. RecordingSession avoids redundant selection and stale map reads.
UI479tests/25files/types and6Go racepackages/build pass. Native8admission/50close/
14initial/28retry/30restore/12workflow pass; profiles preserved. Real UI10checks
pass after correcting an assumed recording=false precondition: screen auto-starts,
so fixture explicitly stops its recorder before no-callback checks. Original
failed producer retained; no source patch or weakened assertion to satisfy it.
Managed31936 succeeded; all8frozen paths unchanged; no pending operations. Runtime
-210lines; Go+2complexity (152cumulative reduction); tidiness one fewer long file,
1134findings/debt31605; board17unknown. Receipt
`evidence/rehabilitation/ui-page-admission-2026-09-23.json`. Next independent angle:
real live frame delivery and selected-tab paint with recording active. This differs
from the intentionally stopped-recorder tab-state UI fixture; do not conflate them.

### BAS-WORK-066 — 2026-09-23 UTC — native live paint investigation

W3 RF096/J05/J22/J23.065 fully consumed. Recall55hits/10corpora; source-ledger
agent-memory/scopes timed out (/tmp/bas-live-ui-paint-recall-066.txt), other
results usable. No inference of missing capability. Native probe38148 consumed1:
4pass/2fail; owned cleanup passed. Initial canvas never painted; subsequent
external-create callback title/selection was inconsistent. Rawreceipt
/tmp/bas-ui-paint-native-066.json, producer under/tmp/browser-automation-studio/.
Driver wire usesimage/mime; client expectsdata/media_type. Isolated native probe
10270 pending checks actualdriver/APIbytes plus UI paint with both tabs admitted
before recording. No source change yet; baseline/tmp/bas-before-066/manifest.json.
Targetdocs updated before implementation. Repair canonical frameDTO/decoder; add
real HTTP bridge regression and invalidreceipt cases, then nativepaint qualification.
DirectWS config/port, polling ownership and same-session callback ordering remain
separate follow-up investigations; no transport/performance claims from thisslice.

066 implementation follow-up: isolatednative10270consumed1 provesdriverimage18895
characters versusAPI0,9pass/5fail includingfourcanvaspaint failures. GoHTTPbridge
regression+7invalidreceiptcases fail onoldsource; /tmp/bas-live-frame-bridge-red-066.txt.
Addedmissingcontenthash rejection while removingold timestampETagfallback.
Initialglobaltextreplacement touchedfourunrelatedresponsecalls and failedcompile;
restoredthoseagainstbaseline before qualification, originalfailure retained.
68652consumed0:4Goracepackages andAPIbuild pass in final066logs. Eightpaths frozen.
Runtime-31lines,Go+6complexity (validation),80functionsunchanged; cumulativeGo-146.
Oneframewireowner replacesduplicateDTO/translation. Pendingrestart28847, TG50774
admission, board88326. Nativegreen/workflow/profilevalidation stillowed.

066 finalqualification:restart28847consumed0/build8ed3dbc7171eb0b4286397d338a0cbee0a1400d306e8581abfd2021271676fe6. Native49874consumed0:14pass/0fail.
Visualreview saw the immediate screenshot during the normal300ms transition;
strongerproducer25587consumed0 requiresoverlaydisappearance and also14/14passes.
Finalimagevisuallyinspected; reds and preliminarygreen retained. Nativeproof is
functionalpaint only, not latency/FPS. ExistingAPI WebSocketdelivered12binaryframes
whileUIusedpolling, identifyingnexttransportinvestigation. Workflow40694consumed0:
12/12,3independenteffects,execution9d714f60-b38d-4f2b-9aef-c6054641567c/artifacts
/tmp/bas-recording-e2e-vFbdRs. Profiles80748consumed0:3identityreads andAPImetadata
preserved. Eightfrozenpathsunchanged. TG20260923-032518-894126d9 admission50774
consumed1(budget),onequietwait/artifactsconsumed;1134/104/399/610/20,debt31551.
Board88326consumed0:17unknown,contract/inventorypass. No pending066operations.
No Go complexityreductionclaimed for thisslice (+6); runtime-31,onecanonicalframeDTO.
Next067:reliableboundedliveframeUItransport/decode ownership, existingRF034/047.

### BAS-WORK-067 — 2026-09-23 UTC — live viewer ownership investigation

RF034/047/J05/J22/J23; baseline deployed066. HypothesisA: driver fails tosend
frames; falsified by native066existingAPIWS12binaryframes. HypothesisB: UI's
independentdirectsocket config/port route fails while mainWSframesareignored;
native96033pendingcapturesactualbrowserconfig,hookconsole and UIstreamstatus.
HypothesisC: evenafterrouteworks,decoderwork and latecompletionoutliveownership;
reuse retainedframeProbes withsynthetictransport/scheduler onactualhook, narrowed
producer/tmp/browser-automation-studio/ui-frame-owner-red-067.cjs. Independent
expectedoutcomes remainboundeddecode,orderedlatestpaint,nolatepaint/reconnect
afterunmount/sessionreplacement andpollingbeforefirstusableframe. No067production
edit yet. Newrecall64571pending(/tmp/bas-ui-stream-recall-067.txt). ExistingAPI
binaryprotocol lacksperframepage/sessionenvelope atUI; donot simplysubscribea
sharedcontext and claimhandofffencing withoutaddressingqueueidentity.

067 narrowedownerprobeinitialattempt was unusable becausetop-levelsession mocks
initialized an evolved dependency; unchangedUI-onlyextractionproduces6fail/1pass
in/tmp/bas-ui-frame-owner-isolated-red-067.json. Native96033/36950consumed1:8pass/
1fail each,actualWebSocketconstructorURL24486 vsmanaged24438,existingAPISocket
2binaryframes. Configpresent/numeric24485; missingconfighypothesisrejected.
Recall64571consumed0:26hits/10corpora,source-ledger.scopes timeout retained; no
suitableadditionalprogram. FirstVitestfilter usedui-prefixfromuicwd(no tests);
corrected51770owner produces14actualfailing regressions,not a runnerblocker.
Targetdocs updated before productionedits; baseline/tmp/bas-before-067/manifest.json.
Decision:dedicatedviewer subscription throughconfiguredAPIWS plusoneboundeddecode
owner; retireunuseddirect researchlistener/config aftercaller audit. No newservice.
Perframebrowserpage/leaseidentityremainsRF038; donot overclaimlocalgenerationfencing.

067 implementation checkpoint: useFrameStream rewrittenaroundoneeffect-ownedAPI
recordingsocket,HTTPfallback,monotonicadmission andsharedacross-effectsboundeddecoder.
15redmaintainedcases nowpass; added4positive/adversecontrols forrawJPEG,successful
ETag/304,disposed-socketdelivery andlateHTTPfailure. UI37681consumed0:498/26/types.
Driver64298consumed0:71/5/types. Initial4862had71greenbuttypecheckcaughtstale
directFrameEndpointlogreference; removedandrerunretained. Go93061consumed0:
sidecar,supervisor,driver,handlersrace/APIbuild; health/recovery havenoGo tests.
RemoveddirectFrameServerandits4tests (retiredowner,notassertionrelaxation),global
producerbroadcasts,driverdirectPortconfig,UIguess/proxy/config,serviceallocatedport,
sidecarenvforwarder andunusedlatencyLogger. ExistingAPIroutepreserveswebviewing;
fullnativequalificationpending. Runtime4925→3896(-1029),Go62→61(-1),19functions;
TS/JScomplexitystillRF064. Fifteenpaths frozenbefore87130fullowner/68669tidiness/
21389nativeoutage-red. Currentlive066buildunchanged; HTTPoutageprobecapturesanimated
fixtureandblocksUIpollingonly, requiringactualstreamedpixels. No pendingolderops.

067 full driver qualification: exec87130 consumed0; owner
`uh-20260923-034848-ed518695d56e926400000046ca3eaf2c` passes1781 tests/123 suites,
with2 tests/1 suite skipped. Owner command455.602s/Jest454.663s; this runtime is
longer than064 and is not a controlled product-performance comparison. Four
retired-listener tests account for1785→1781. Fifteen frozen paths unchanged.
Managed restart72940 admitted after qualification; native stream/outage and
profile/workflow validation remain pending.

067 managed restart72940 consumed0; health build `sha256:6a3e59568a71673e45ddbc788e1a6cb4cf6300669a6b36a6d96ddb44c736ec4a`. Lifecycle declares API17116/driver24485/metrics24478/UI21794 and no direct frame port. Native outage-green52422 pending; unchanged15-case red oracle. An additional producer qualifies pointer coordinates against a fixture-owned effect counter; no production source change.

067 final qualification: deployed build6a3e59568a71673e45ddbc788e1a6cb4cf6300669a6b36a6d96ddb44c736ec4a.
Native outage52422 consumed0:15/15; expanded67850 consumed0:16/16 including exactly
one independent selected-tab pointer effect. Normal preview14/14 and workflow12/12
pass in28042; executionc651ad14-9d6a-488a-80cc-7c8b8d74358a, artifact directory
/tmp/bas-recording-e2e-HKC7I1, three independent effects. Profile/freeze69738
consumed0:three original identity reads, unchanged API metadata and15 frozen paths.
Final screenshot visually inspected. No pending operations. RF007/008/033/034
register entries distinguish repaired/retired owners from remaining wire identity,
API access/fanout, slow-reader and performance qualification. Next068 investigates
external new-tab callback/state disagreement from the retained initial066 probe.

### BAS-WORK-068 — 2026-09-23 UTC — external tab callback ownership

Revisit RF038/095's observed external-create disagreement from066 on qualified067.
Hypotheses: UI drops rapid lifecycle messages; callback metadata arrives stale; or
UI treats observed page state as a navigation command and undoes the API creation.
The producer records ordered page_event/page_switch messages (socket labeled, since
main/viewer both subscribe), UI POST bodies, canonical creation receipt, actual
driver location, final registry and independent colored paint. Expected: externally
created Blue tab stays Blue and UI agrees. No source change yet. Recall1811 pending;
producer/tmp/browser-automation-studio/tab-callback-native-red-068.mjs.

068 native62323 consumed1:7pass/4fail. Main socket emits page_created(blank),
UI activates canonical new page, then UI POST navigate(blank); driver and registry
both become blank despite external201 Blue receipt. This proves RF097's feedback
loop and rejects a label-only explanation. Recall1811 consumed0:22hits/10corpora;
source-ledger memory/scopes timeouts retained, suitable additional program absent.
Baseline/tmp/bas-before-068/manifest.json captured before edits. Target architecture
and sole issue register updated. Repair existing navigation hook/component boundary;
no new service or dependency. Driver callback attachment gap remains unproven.
Resume read mistakenly requested nonexistent REFRACTOR_CHECKPOINT.md; recovered
canonical REFRACTOR_PROGRESS.md immediately; no evidence lost or operation pending.

068 owner red96908 consumed1:9 failed/3 passed; APIs for explicit intent/readiness
were absent at the previous hook boundary. Initial45859 was unusable as red proof:
the append used the wrong cwd and only the two old cases ran; corrected by absolute
paths and retained both receipts. Green20274 consumed0:12 cases and types pass.
Added StrictMode, observed-location-during-request and late parsed-body controls;
full UI52864 pending. Three paths frozen in/tmp/bas-frozen-068.json. Component
removes automatic URL replay, duplicate parser and three URL/completion refs; hook
owns explicit request identity, abort/admission and launch readiness. Failed launch
shows the existing error banner and remains retryable. History commands unchanged.
Runtime1792->1751(-41); no Go change and TS complexity remains RF064. Native
intermediate producer retains the exact original eleven assertions.

068 full UI52864 consumed0:511 tests/26 files and types pass. Runtime metric
correction:1792->1746(-46), not1751/-41; retained machine manifest has exact per-file
counts. Managed restart44764 pending. Tidiness61893 consumed1 with budget finding,
run20260923-041954-8fb343a8; one wait/artifact read admitted. Board/contract/inventory
64674 consumed0, all17 outcomes still unknown. No production/test changes since
freeze. Driver and Go owners unchanged from qualified067.

068 managed44764 consumed0, healthy build82be32cb6d32a0c3ad9cbf1fc032eeb125a9a1b47ddf3c7f0e20cf763b692343.
Original native58228 consumed1:10pass/1fail. RF097 navigation is fixed; driver,
registry and blue paint agree, but tab remains Untitled. Main socket has created
blank + switch and no navigation callback. RF098 tracks the distinct missing
callback; driver attachment-after-await is the next discriminating hypothesis.
Extend068 to that existing BAS owner; baseline now includes page-events runtime/test.
Three UI paths remain frozen. Test Genie068 one wait/artifact retrieval consumed;
1133/103long/399complexity/610duplication/20coupling, debt31551 unchanged vs067.
Expanded native launch/URL/repeat/history/redirect producer prepared, not run yet.

068 driver28878 red consumed1:two creation-gap assertions fail,13 cases pass.
Driver60784 green consumed0:88 affected tests/3 suites and types pass. Listener
attaches synchronously; creation-admission promise gates later publication and
cleanup rejects pending events. Removed separate pending-close branch. Final
five-path freeze/tmp/bas-frozen-068-final.json; UI freeze unchanged. Runtime total
1997->1958(-39); driver adds7 lines for the admission boundary, UI removes46.
Full driver qualification and a new tidiness run admitted for the changed driver
source; earlier068 tidiness only qualifies the intermediate UI-only tree.

068 validation pending: full driver94169; final tidiness91456 consumed1 with
run20260923-042551-bbfec87d, one wait and artifact retrieval consumed. Final
1133findings/103long/399complexity/610duplication/20coupling/debt31551, unchanged.
TS measurement recall65709 consumed0:66hits/10corpora, memory/scopes degraded.
Existing Tidiness Manager explicitly skips TS/JS; installed ESLint can supply a
scoped supplement without changing shared owners. Validated classic complexity
controls and disabled inline lint directives. Initial067 supplement rejected an
inline reference to an unloaded rule; corrected uniform analyzer configuration,
not source. Code-path count includes class initializers; do not call it functions.
068 measured437->443(+6),141->143 code paths.067 unchanged freeze verified and
measured714->620(-94),247->207 paths. Receipt
evidence/rehabilitation/typescript-complexity-supplement-2026-09-23.json. RF064
owner-wide coverage remains open; no original whole-scenario TS baseline claimed.

068 expanded intermediate50711 consumed1:21pass/1fail (known label gap). Real URL
bar launch/request/repeat, redirects, back preserving forward history, forward,
reload and independent colored paints all pass. Fixture loads prove one repeated
request/reload/redirect destination effect; UI request capture proves observations
do not create additional navigation commands. Full owner94169 still pending.

068 full owner94169 consumed0:uh-20260923-042546-48b777d33f8707b25c35f31a6242f4e4,
1784tests/123suites passed; existing2tests/1suite skipped. Owner362.458s/Jest361.711s.
Five frozen paths unchanged. Managed final restart50435 admitted; final native,
saved-workflow and original-profile checks remain pending. Receipt tab-navigation-
ownership-2026-09-23.json explicitly remains qualification_in_progress.

068 final managed restart50435 consumed0, healthy build sha256:60b7ae38ebda099da7b78080f84f63cf5ff11ddd96aba01b7d01d2192b1e95f1. Five frozen paths unchanged. Original and expanded native checks79866 pending; workflow/profile checks remain after those finish.

068 final qualification: native79866 consumed0; original11/11 and expanded22/22.
Screenshot blue tab/address/pixels visually verified. Workflow35078 consumed0:
12/12,three effects,executioneaac59e2-3b1f-421e-9130-bda0f2dd4964,
/tmp/bas-recording-e2e-X0fY9B. Profile84211 consumed0:three original identity reads,
metadata equality to067 and all five frozen paths unchanged. Receipt qualified and
deployed. No pending068 operations. Next069 probes command intent across active-tab
change before server admission; no new defect inferred from source alone.

### BAS-WORK-069 — 2026-09-23 UTC — navigation intent before tab admission

Hypotheses: the UI aborts an old-tab navigation on selection change; the server
binds the intended page from the request; or the request targets whichever tab is
active when delayed delivery reaches the server. Prior061 checks bind the page
at server admission and fence subsequent awaits, so they do not alone establish
the earlier user intent. Recall58603 consumed0,related prior navigation work read;
no suitable new program, memory/scopes timeouts retained. The owned native fixture
holds a real UI navigate request before admission, switches Red->Blue, releases
the request and checks actual Blue location, destination load count and address
bar. Expected: no effect on Blue and no stale destination load. No source edit.

069 native89762 consumed1:14pass/3fail. A single held UI navigate command issued on
Red reaches the server after Blue selection, and actually navigates Blue. Address
bar and fixture destination loads corroborate. Each normal fixture URL was fetched
twice too; do not interpret two destination GETs as two UI navigation commands.
RF099 registered; target architecture updated before edits. Baseline13 paths in
/tmp/bas-before-069/manifest.json. Repair existing UI/API/driver owners; no external
boundary, dependency change or new service. Common navigation request lifetime
will replace repeated UI history HTTP code. No069 source/test edits yet.

069 maintained red: UI2647 consumed1 (14fail/13pass); driver49809 consumed1
(16fail/75pass); API86456 consumed1 (28 failing/4 passing subcases). Canonical
page IDs now travel in UI URL/history bodies; API resolves only the active page
of that session and passes expected driver identity. Driver checks it before any
effect, including a tab change while reading the body. API preserves409 conflicts.
Programmatic implicit-active semantics remain; positive controls pass. UI common
request owner replaces repeated history HTTP loops, binds intent at submission,
waits for launch page admission and rejects late config/JSON/error/history reads.
Selector requires a validated matching session/page, avoiding a stale store page
during session replacement. New batched-submit/switch control passes.
Focused green11649/4178/3861 consumed0 (API32 subcases,driver91,UI27/types). Final
Go7651 consumed0:four race packages/APIbuild. Driver88926 consumed0:140 tests/3
suites/types. UI80329 consumed0:524 tests/26 files/types; prior81132 also consumed0
before final selector strengthening. Twelve changed paths initially frozen in
/tmp/bas-frozen-069.json (13 baseline paths included one unchanged Session test).
Runtime4715->4703(-12). Go62->65 functions,248->262(+14); cumulative affected Go
reduction now133. TS supplemental measurement retained separately; no source edits
after freeze. Final full-driver owner/tidiness/native deployment still to admit.

069 full driver48192 admitted on frozen source. Tidiness51716 consumed1,
run20260923-045639-020a47bc; one quiet wait/catalog read consumed. Board/contract/
inventory78227 consumed0,17 unknown outcomes. Supplemental ESLint443->465(+22),
135->142 code paths; added guards are explicit complexity cost despite12 fewer
runtime lines. Prepared unchanged native17-case oracle plus21 API-guard cases and
23 navigation-preservation cases. Managed restart not yet admitted.

069 additional adversarial UI check77829 consumed1:two failures prove returning
Red->Blue->Red replays an old intent, including one skipped before first admission.
Fixed by retaining the opaque UI page lifetime at submission/admission, not only
its repeated ID. Full driver48192 remains applicable: all driver and Go paths
match the original freeze; only UI hook/test changed. New snapshot
/tmp/bas-frozen-final-069.json. UI final2 qualification pending. Initial069
tidiness fully decoded:1136findings/103long/401complexity/611dup/20coupling,
duplication debt31391; source predates this last UI correction. No native green
or managed restart admitted yet.

069 final UI55240 consumed0:526 tests/26 files/types; returning to the original
tab no longer replays a retired intent. Final frozen runtime4715->4706(-9),
Go248->262(+14), scoped ESLint443->468(+25). Twelve final paths unchanged.
Final tidiness77467 admitted after the UI correction; full driver48192 pending
on unchanged driver inputs. Native17/24/23 producers ready. In-progress receipt
evidence/rehabilitation/page-bound-navigation-2026-09-23.json records all limits.

069 full driver48192 consumed0:uh-20260923-045634-d28f8a4be0ed7348fabd5aa98ae08efb,
1808tests/123suites pass,existing2tests/1suite skipped. Owner355.022s/Jest353.981s.
Final12 paths unchanged. Managed restart59024 admitted. Final tidiness77467
consumed1,run20260923-050256-ad02d3c4; one quiet wait and catalog/artifact decoding
consumed. Detailed summary retained in/tmp/bas-tidiness-native-final-069.json.
No pending validation owners other than managed restart/native work ahead.

069 oracle wording correction: the fixture counted URL requests, not captured
HTTP methods. Its two destination entries do not establish two GETs or two browser
navigations; exactly one UI navigate request and actual wrong-tab location are
independently proven. Later passive-request investigation may distinguish recorder
interception, preflight and browser behavior; no additional defect declared yet.

069 managed restart59024 consumed0, healthy sha256:b494c40d57e4c6e88eba6d282de3af41e91c941c12d05673b17132196f00f571. Final12 paths unchanged. Native17/24/23 checks42126 admitted. Workflow/profile checks remain after native completion.

069 final qualification: native42126 consumed0 with17/17,24/24,23/23. Screenshot
original red tab/address/pixels inspected. Workflow86337 consumed0:12/12,three
effects,execution37dbce86-2f27-46a9-a20c-8cec78c7cb27,/tmp/bas-recording-e2e-XPqkQl.
Profiles15679 consumed0:three original identity reads, metadata equals068 and12
frozen paths unchanged. Receipt qualified/deployed; no pending069 operations.
Next070 compares duplicate fixture URL requests with/without UI and records method
and UA to distinguish browser, recorder and link-preview traffic before repair.


### BAS-WORK-070 — 2026-09-23 UTC — passive document request footprint

RF100/J04/passive-fidelity investigation on qualified069. Recall26583 already
consumed; feedback reread, no new instruction. Producer9925 consumed1:9pass/4fail,
`/tmp/bas-passive-requests-red-070.json`, script under/tmp/browser-automation-studio.
Independent method/UA/phase/cookie-presence log isolates one extra LinkPreviewBot
GET per newly displayed URL to the tab bar, without recorded browser cookies.
Direct browser, BAS without UI and recording without UI request each document
once. Custom icons and cleanup pass. Bounded2s windows test absence; no latency
or general performance qualification. Legacy route injection is not implicated.

Target architecture and RF100 updated before edits. Baseline17 candidatepaths at
`/tmp/bas-before-070/manifest.json`, including absent newTabBar test. Implement
browser-derived favicon metadata across existing driver/API/UI page ownership;
remove single-preview hook, preserve batch previewcards. Maintain cancellation,
URL replacement, missing/empty metadata and icon-failure recovery regressions.
No production edit or pending native run at this checkpoint.


070 scope review: initial green24UI/113driver and focusedGo pass. Added creation-
metadata overlap test reveals candidate drops both created and queued navigation
(75045:1fail/18pass). Preserve admission while retiring stale icon metadata.
Preservation review also requires icons on initial navigation/restored tabs and
new-page receipts before recording callbacks exist. Extend existing driver/API
receipt owners, not a new endpoint; baseline adds6 untouched paths(23total).
One browser-document icon reader will serve lifecycle and command receipts.
Go history publication consumes its typed response instead of six parallel scalar
arguments. No external scope/authority extension; all paths remain BAS-owned.


070 validation checkpoint: focused driver28415 passes118/2 suites/types; UI54069
passes531/27 files/types. Go29023 passed four race packages/build before reload
metadata publication change. Reload red31931 proves2 absent-notification failures;
all committed history commands now publish page metadata. Go7282 completed but
its test log fails one older reload-notification count assertion; shell exit0 was
from subsequent successful build, so it is explicitly not a test pass. Updated
that count for the new desired behavior, preserving journal count and failure
semantics. Go regression requalification follows, driver source/tests unchanged.

Frozen23paths `/tmp/bas-frozen-070.json`. Full driver95063, managed restart48284,
tidiness admission63067 and board/contract/inventory99002 pending. Do not re-admit.
Native070 final producers prepared but not run. Source review fixed creation
metadata admission loss; new regression covers it. Scoped TS404->409(+5), code
paths111->110; removal of single-preview hook does not prove a complexity win.


070 native/final preservation,2026-09-23: restart48284 consumed0, build217b9a2f8bafd6725628aef4202c133fb11d271212a4b916250ce73b95fa7eb7 healthy. Native92359
original13/13 and expanded23/23 pass. Fixture logs one full-document request per
navigation, no LinkPreviewBot or tab-preview RPC; browser-derived custom icons
persist for initial and inactive pre-recording pages, base-relative/data icons,
broken-to-valid recovery and reload after recording stopped. Screenshot inspected;
no frame-rate qualification. Workflow40409 passes12/12 with3effects, execution
 e830f81d-88f4-4db0-9013-615b28ecae4f,/tmp/bas-recording-e2e-DYRlcp. Profile24186
threefull reads matchoriginalrollback andAPI metadataequals069. Frozen23paths
unchanged. Full driver95063 remains pending; no other070 operation pending.

071 recall21743 consumed0:57hits/10corpora; memory/scopes timedout. Read returned
capture-surface via prepare-operation92203(prog_4d42a42e-32cd-4656-ab82-a0476d4a2080)
with no changes. Static capture cannot exercise keyboard effects, so reuse native
owned-browser fixture. Tab selectors' focus and key handling remain hypothesis,
not a new RF until independently reproduced. No runtime edit while070driver runs.


### BAS-WORK-070 final — 2026-09-23 UTC

RF100 qualified/deployed. Full driver95063 consumed0,
uh-20260923-053319-477e564d56a31fc385a8cc3a752977ec,1816tests/123suites,
2tests/1suite existing skip,owner381.647s/Jest380.853s. Allfrozen070runtime/driver
inputs unchanged; Go test notification expectation separately qualified. Receipt
passive-tab-metadata-2026-09-23.json finalized. Runtime-38,Go+3,TS+5; cumulative
Go reduction130. Native13/13,23/23,workflow12/12 andprofilespreserved. No070pending.

### BAS-WORK-071 — 2026-09-23 UTC — keyboard and last-tab observations

Native51130 consumed1,5pass/5fail. RF101 keyboardfocus defects independently
reproduced; pointer control succeeds. Final-tab closure then timesout inUI while
APIanddriveragreeallpagesclosed, followedbyprofile-save/close500 from openAPIclient
circuit breaker. RF102 added with rootcausestillhypothesis. The final new-tab
placeholder assertions neverran. Producer deletedonlysyntheticprofile evenafter
failedsessioncleanup; this mustbe correctedbefore nextprobe. Retainedrecovery
session584a55fd-437a-4cfd-bb93-6d847d717745 viaobservability, directstorage200 and
APIclose200 aftercooldown; do notclaimprofilesaverecovery sincefixtureprofilehad
alreadybeendeleted. Original userprofilesuntouched. Runtime log stored; no orphan.
SourceWebSocketProvider uses ReactsinglelastMessage state; usePages consumes an
Effect. Rawwire/delivery/empty-selection classification needs discrimination.
No071sourceedit; prioritize RF102beforekeyboardrepair. No pendingoperations.


071 discrimination: instrumented native last-tab control completes9/9, including
cleanup; raw wire contains closed-page events and empty selection. Kept historical
redfilename withactualpassingcases; noattemptdiscarded. Maintained realProvider+
usePages probe3366 fails2/2: spaced lifecycle appliesclosures but codecstrips
active_page_id; burst dropsclosures as ReactsinglelastMessagecoalesces. Rawnative
control rejects missing-wire/empty-selection-server hypotheses for that run.
Architecturetargetupdated before071runtimechanges; baselineexpanded to allcurrent
lastMessageconsumers plus generic codec, at/tmp/bas-before-071/manifest.json.
Replace lossy transport state with synchronous scoped subscriptions, preserve
opaque domain fields, retire unused binary subscriber API. Add current-socket and
unmount checks to prevent stale delivery/reconnect. RF101keyboardremainsopen and
breakerfailureclassificationremainsunqualified. No071runtimeedityet.


071 implementation and qualification in progress: genericwirecodec nowpreserves
opaque domain fields. The codec-only59784 test has1pass/1fail: spaced passes,
burststillfails. Added staleconnect/unmount controls53628:3fail/1pass. Replaced
singlelastMessage with synchronous subscriptions inall11domainconsumers, removed
unusedbinarysubscriberfacade and unusedonItemsReceived option(no callers), and
fenced socketcallbacks/retirement. Focused17911 passes30; firsttypes25265 flagged
the unusedoption, corrected. FullUI38918 passes1155tests/79files(30.30s), finaltypes
66834pass. New6real-provider casescoverburst,spaced,opaque fields,latestcommitted
callbacks,throwingpeerisolation,malformedJSON,retiredsocket andunmount. Driver/Go
runtimeunchangedfromqualified070; do notrepeatfull driverwithoutnewscope.

Frozen17paths `/tmp/bas-frozen-071.json`. Runtime3316->3294(-22),Go unchanged,
scopedTS720->715(-5),codepaths198->202. Managedrestart5444 pending. Tidiness26819
andonequietwait/catalogconsumed:20260923-055941-d8b21216; nativefindingsdecoded.
Board/contract/inventory8879consumed0,17outcomesunqualified. Finalnative producer
/tmp/browser-automation-studio/last-tab-native-qualified-071.mjs prepared, notrun.
Also rerun the original keyboard producer with separate output: keyboard failures
remainexpectedRF101, but last-tab/recoveryassertionsshouldpass. Checkaddressclear,
framepollstop andfresh-tabcreation explicitly; callbackclosuretiming couldneed
additionalrepair. No keyboardproductionedit yet.

072 independent availability investigation while071builds: recall69474 consumed0,
65hits/10corpora,memory/scopesdegraded; BASusageskillalreadyread. RealHTTP/client
producer exits1 with6failedrequest-rejection cases and4positivecontrols. RF103
recorded. No072sourceedit/testfile/baselineyet; nextrepairafter071nativequalification
usesexisting resilienceowner, preservingoriginalerrors andgenuineoutagebehavior.

071 native intermediate: restart5444 consumed0, healthy build2d6342419a28be5733ee0a481c80737b75cda7f19adb4b1df7b508ca87e6c4c5. Native55932 consumed; qualified last-tab11pass/1fail: tabs/address clear and fresh tab works, but empty workspace continues 300ms frame polling without page_id. Keyboard9pass/4fail remainsRF101; passive metadata23/23 preserved. All owned cleanup succeeds. Extend071 baseline to viewer/input/preview null propagation and maintained tests before repair. Explicit null disables transport/input; undefined preserves mini-preview active-page behavior. No driver or Go source changes.

071 empty-viewer qualification: maintained red79071 consumed1,4fail/19pass, then explicit-null propagation and existing owner gates fix both transport and input. Record-mode72542 passes541tests/29files and types35380pass. Final frozen23paths and expanded metrics recorded above. Final native/workflow/profile qualification follows managed rebuild; prior full UI1155 remains valid for unchanged event consumers. No full driver rerun for UI-only change.

071 final native49835 passes12/12: final tabs/address clear, no HTTP frame polling over1200ms, new tab succeeds, original profiles uninvolved and owned cleanup succeeds. Restart44858 consumed0, build e7bba9e85630bd229a4fb45bf139c97b10833a02efaa332fca6324ba156afecb healthy, frozen23paths unchanged. FinalTG6802 admission/wait failed on unchanged budget; catalog and native decoded, exactrun20260923-061257-15bc910a,1137/103/401/612/20/debt31319. Board11115 consumed0, all17unqualified; contract/inventorypass. Workflow qualification now pending.

072 target and3pathbaseline recorded before source change: shared api/internal/resilience owner distinguishes answered client rejections from dependency outage, keeps original errors and408/server/transport protection. Actual Client HTTP fixture6red/4controls retained. Owner tests cover wrapped rejection/cancellation/deadline and half-open recovery; driver client tests cover independent healthy storage route. Storage is the other breaker consumer and included in scoped race checks. No driverTS orUI change planned.

### BAS-WORK-071 final — 2026-09-23 UTC

RF102 qualified/deployed build e7bba9e85630bd229a4fb45bf139c97b10833a02efaa332fca6324ba156afecb; receipt page-event-delivery-2026-09-23.json. Native49835 passes12/12; workflow64476 passes12/12 with3effects, executionf3d09b97-0fd9-4ce7-9062-6af7cc8b3942, /tmp/bas-recording-e2e-G2vydd. Profile75555 consumed0:3complete reads matchrollback, API metadataequals070. Screenshot inspected. All071operationsconsumed. Runtime-21,scopedTS-4,Go unchanged. All17release outcomes unknown; no global production qualification. RF101 andRF103 remain meaningful next repairs.

072 maintained red exits1:13request-status subcases plus half-open recovery fail atresilienceowner;6actualclientcases fail. Controls pass. One canonical classification replaces429-only special case and redundant deadline branch. Go13628 five racepackagespass; build70676pass. NoUI/driverTS changes. Pending independent native-client real-driver check, live deployment and072metrics/board.

072 live68792 restart consumed0, build9ad04e2414f9cdde4014149f5529b0d3ab0337d252af1362e1b1b5dba8c6f160 healthy; frozen3paths unchanged. Standalone realdriverClient native5568 passes13/13, preserving six404errors then healthy storage/save/close and syntheticprofilecleanup. This uses compiledproduction Client againstrealdriver; it does not inject errors into the deployedAPIglobalbreaker. OriginalHTTPfixture10/10pass. TG79406 and onequietwait/catalog consumed;20260923-061909-ef387488,1137/103/401/612/20/debt31313,budgetfails. Board85762 consumed0,17unknown. Runtime354->350(-4),Go52->53(+1),16functions; cumulativeGo-129, no per-cycle complexity win. Profile14061 and savedworkflow pending.

073 recall36690 consumed0,65hits/10corpora,source-ledgermemory/scopes timedout. Existing staticcaptureprogram unsuitable for keyboardeffects; retained native fixture. Loadedexperiential-ui-design, whichroutes pureusability toUX; readUX and itsvisited/knowledge support. File-only goal overrides externalcoverage/journal writes and humanratification gates; no route composition/mockup redesign inthis boundedkeyboardrepair. APGprimarysource read andarchitecturetargetdocumented. Baseline2files beforeedit. Native071 alreadyproves4keyboardfailures, pointercreation/selectionpositivecontrols. Maintained expected keyboard/focus/closure regressions precedeimplementation.

### BAS-WORK-072 final — 2026-09-23 UTC

RF103 qualified/deployed; receipt driver-availability-2026-09-23.json. Workflow50033 consumed0,12/12 with3effects, execution49f61a40-6ad5-4a09-a94a-c43652085214, /tmp/bas-recording-e2e-EQbL3V. Profile14061 consumed0,3complete reads equalrollback, API metadataequals071. No072pending. Go+1complexity/runtime-4explicit; broaderoutcomes remainunknown. Continuous workmoves to073keyboardrepair.

### BAS-WORK-073 implementation — 2026-09-23 UTC

RF101 maintained red6fail/5pass confirms keyboardsemantics, focusmovement andactivation missing. Nativebuttons withseparateclosebuttons replaceclick-onlydivs; keyboard owner handlesfocuswithoutremoteactivationuntilEnter/Space. Close/Deletefocusrepair waits foractualpageremoval; failedclosureskeepfocus andexternalfocusisnotstolen. One additionalcreated-tabfocusred54065 exposedplaceholderreplacementlosingfocus, repairedusing samefocusowner. Final550/29/typespass. Runtime+56/TS+37explicit; removingredundantclosecallbackdoesnotoffset newly requiredkeyboard/focusbehavior. No API/drivercode changes. Nativeeffectfulproducer ready. Pendingmanagedrestart, exactTGadmission/wait, board andlivequalification.

073 exactpending: managedrestart94867, TGadmission17591, board42759. Contract/inventorycompleted0. Sourcefreezeunchanged.074frameidentityrecall admittedindependently whilebuilds; no074sourceedit.

073 live: restart94867 consumed0, healthybuild9c4f9cc03265d6fb5fee2bb887c512308e99d70347bcc995a46762d12c4025f8, frozen2paths unchanged. Native28256 passes19/19 including actualdriverURLselection forEnter/Space, arrowfocus-onlycontrols, Deleteclose/neighbor/emptyfocus, creationfocus andpointercontrols. Screenshotinspected. Passiveiconpreservation nowpending. TG17591 admission/wait/catalogconsumed:20260923-063055-ba993930,1137/103/401/612/20/debt31313,budgetfails. Board42759/contract/inventoryconsumed,17unknown. No073sourcechange afterfreeze.

074 recall28452 consumed0,66hits/10corpora,source-ledgerscopesdegraded. Sourceconfirms APIframecallback/driverstream route broadcasts bytes withoutsourcepage/leaseidentity; existingRF038 explicitlycovers this. Hypothesis: a genuineRedJPEG delayedatdriver-callbackboundary canoverwriteBlue after Blue selectionandpaint havecompleted. Prepareownedproducer /tmp/browser-automation-studio/frame-identity-red-074.mjs: captureactualRedframe, establish callbacksocket, selectandpaintBlue, thendeliverheldRedbytes andindependentlycount visiblecanvasRedpaints. Distinguishsource-driverselectioncontrol, naturalBlue recovery andcleanup. This simulates a delayed producer via documented API socket, not a naturaldriverqueue race. Notrunyet; no074sourceedit/baseline.

073 passive63387 consumed0,23/23: custom/inactive/base/data icons, image-failure recovery and stopped-recording reload preserved aftersemanticbutton replacement. Savedworkflow nowpending; no newsourceedits.

### BAS-WORK-073 final — 2026-09-23 UTC

RF101 qualified/deployed, receipt tab-keyboard-2026-09-23.json. Workflow57902 passes12/12,3effects, executionfb33d635-1780-468d-94b0-2b0bcaeb983f, /tmp/bas-recording-e2e-E5oiDj. Profile50403 consumed0,3complete reads equal originalrollback; APImetadataequals072. Frozen2paths unchanged. Runtime+56/scopedTS+37 supports newly available keyboardfunctionality, not a debt-reduction claim. All073operationsconsumed. Continuousworkproceeds to074existingRF038sourceidentityboundary.

074 native30055 consumed1:9pass/1fail,5RedpaintsoverBlue afterheldactualJPEGdelivery. Canonical/driverBluepositivecontrols andnaturalBluepaintrecoverypass,ownedcleanup complete. Sourceinvestigationfinds sessionownerExecutionId/leaseId andpageToIdMap alreadyavailable atdriver; strategiescurrentlyemitoptionalperfJSONheaderoranonymousraw/eight-bytetimestampJPEG. APIdecodeDriverFrame stripsperfheader andbroadcastsby sessiononly; HTTPcallbackalsolacksidentity. API SessionstoresimmutableexecutionID/leaseID,PageTrackerownsdriver-to-canonicalmapping. Framewire repair mustretainproduceridentity throughtothenewviewer. No074runtimeedit; nooperationpending.

074 target andprospective29pathbaseline /tmp/bas-before-074/manifest.json recordedbeforeedit. Productionpaths notallnecessarilychanged; finalmetrics mustattributeactualchangedruntimepaths. SourceHTTPframe routeignorespage_id, rereads session.page acrossawaitsandcachesonlybysession; heldoldcapture canplausiblypollutecache. Addmaintaineddiscriminatingtestsfirst. Whole-repo TS/JS/Go searchfinds noJSONrecording_framepush producer/consumer beyondhandler/facade/mocks/unusedschema; archiveingestion recordkind is unrelated andpreserved. Designremovesdeadpush path andanonymousbinaryvariants, preservesrealWS/HTTPGET/perf/implicitmini. ImmutableSessionlease exists, driverpageToIdMap exists; APIremainscanonicalmappingowner. No074runtimeedit yet.

074 HTTPmaintainedred32916 consumed1: requestedoldpage isnotrejectedbeforecapture, andcapturefinishingafterpageswitchreturns200. Itsfirstassertionpreventedindependentcachecheck; parameterizedresponse/cacheexpectations nowrunseparately. Scope-only rerun disablescoveragecollection withoutchangingfloors; fullownerqualification followsproductionrepair. No074runtimeedit; focusedcache-redruncompleted1,3fail/4pass. Nooperationpending.

074 cache-red independentresult:3fail/4pass at /tmp/bas-frame-cache-red-074.txt. HeldRedcapturefinishesafterclearFrameCache andBlue selection, repopulates sessioncache; nextBlueframe request nevercallsBlue.screenshot. Confirmscachepoisoningseparatelyfromoldresponse409failure. Baseline/testchanges only; no074runtimeedit orpendingoperation.

074 initialHTTPcandidate: new frame-streaming/frame.ts contains immutableFrameSource/capture/same-source helpers andsinglebinaryencoder (notyetusedbystrategies). record-frames nowchecksrequesteddriverpage beforecapture, snapshotslease/page, fencespost-awaitpublication, keyslookup/reusebysource/policy, avoidsduplicatedhashing/rawbuffercache andthreecopy-pastedresponsebranches. Sevenfocusedtests pass /tmp/bas-frame-http-candidate-074.txt aftervalidownerfieldsaddedtoexistingfixture. NoAPI/binary/UIconversionyet; partialwirechange mustnotbedeployed. Nextconvertstreamproducerswithmaintainedsourceenvelopetests, thenAPIsessionvalidation/canonicalmapping andUIdecoder; retireverifiedunusedJSONpushfacade. Allscopedredrunsconsumed; nooperationpending.

074 streaming red /tmp/bas-frame-stream-source-red-074.txt consumed1,3fail/30pass: bothCDPperformance modes andpolling lackrequiredsourceenvelope. ConvertedCDPpendingframes tosnapshotsource/timeatadmission andpreservesitthroughbufferedflush; polling snapshotsbeforecapture andchecksafterawait/dedup. Bothuse singleencodeFrame withmandatoryversion/source/captured_at andoptionaltiming. ManagerbindssourceForPage toimmutableexecution/lease; SessionProvidercontractextends accordingly. Oldstrategyfixtures/wireassertions subsequentlyconverted; source3cases/typespass. API andUIstillunconverted; do notdeploypartial074.

074 source-only green77862 consumed0,3tests/2suites (30unselected). Initialdriver types65261pass. UpdatedoldCDP/polling assertions toinspectJPEGaftermandatoryheader andtimingnestedunderoptionalfield, preservingeffectcounts/bytes/fallback/ACKtests. Existingstrategyfixturesnowprovideauthoritativesource. Combined4suite framecandidateadmitted; exactIDtoolreceipt. StillnoAPI/UIconversion or074deployment.

074 combineddrivercandidate12459 consumed1,57pass/1fail: oldpollingtestcomparedwholepacket toJPEG; updatedonlypayloadextraction whilepreservingnew/oldframe assertions. Addedmanageractualpage/lease sourcebinding regression andvalidownerfixture. Requalification69819 consumed0,59tests/4suites. Initialdriver types65261pass appliesunchangedruntime since thatrun. Nooperationpending. APIframeSessionbinding/mandatoryheader validation, canonicalpageprojection, UIdecoder andunusedJSONpushremoval are next; no074deployment.

074 APIowned-session sourceadmission testadded inexistingrecord_mode_frames_test.go, usingrealSessionManager admission fixture andrealWebSocket boundary withcollectingHub. Casescurrent/session/execution/lease/page/anonymous/version/retiredsession; validpublicationmustcarrycanonicalpage andomitdriverleasecredential. First go invocationusedscenario rootinstead ofapi module, failedrunnerpath andisnotbehavioralevidence. Correctedapi-cwd rerunpending. Drivercandidate59/4passed; runtimepartial074stillundeployed.


### BAS-WORK-074 boundary implementation — 2026-09-23 UTC

RF038: API source-admission red55738 completed with all eight cases failing;
current frames lose identity and foreign/anonymous/retired sources publish.
Mandatory versioned source envelopes now span driver, API and viewer. Session
validates private execution/lease and current driver page; API publishes only
canonical session/page identity. HTTP preview maps canonical requests to driver
pages, validates the returned source after capture and before ETag admission.
Removed unused JSON push route, hub facade, mocks and UI schema; both live
stream strategies share one encoder. Updated protocol docs. No driver lease is
sent to the browser. Same-page navigation epochs and cross-transport ordering
remain unqualified. No deployment yet; live073 remains healthy.

Focused driver9613:62/4 pass, including lease change during polling, buffered
CDP reconnection and cache reuse. Viewer red76148:19fail/12pass (wire migration,
anonymous acceptance and wrong-source HTTP admission); candidate53056:31/31
and types91559 pass. Full record-mode2579:560/29 pass. API candidate76402 passes;
race22245 found one old integration fixture with no admitted session. Converted
that real HTTP fixture to proper session/lease admission and canonical source
projection; race15103 passes all handlers. Other five race packages passed22245.
API build37756 and driver types88762 pass. All these operations consumed.

31 prospective baseline paths include two protocol docs; final changed paths
frozen in /tmp/bas-frozen-074.json. Runtime metrics in
/tmp/bas-frame-metrics-074.json and scoped TS /tmp/bas-frame-complexity-074.json.
Full driver owner validation admitted (exact tool receipt); native source probe
/tmp/browser-automation-studio/frame-identity-native-074.mjs prepared with old
page, retired lease, wrong execution, HTTP and valid-current positive controls.
The native injection remains a controlled delayed-producer test, not a natural
queue-frequency claim. Next: consume owner result, deploy through managed
lifecycle, run native/preservation qualification and Test Genie/board checks.


074 broad validation: full driver owner58499 consumed1,
uh-20260923-071534-624aa5334ad2cfd555ddf6a676e90e16 stopped at60s no-output bound.
No suite result; do not reuse old1816-pass qualification for changed runtime.
Recall84913 consumed0 (64hits/10corpora). First ordered Electron suite diagnostic
completes0 with its existing conditional skip; pipeline localizer58108 remains
running. Targeted native-fixture66474 completes1: three native pixel tests still
lack sourceForPage and decode old wire. Preview lifecycle mock also lacks
pageToIdMap. Baseline extended before correction with pipeline-e2e.test.ts;
prepared conversion preserves original independent RGB/dimension/lifecycle
assertions. Apply after in-flight localizer ends; runtime is unchanged.

TG92055 and its one quiet wait/catalog consumed:20260923-071619-1a170e28,
1141findings/103long/405complexity/611duplication/21coupling/debt31221;
budgets fail. Board89121 consumed0,17pending_telemetry rows; contract/inventory
pass. Scoped TS421->467 (+46), Go+30, runtime+25; cumulative affectedGo-99.
No restart/native074 qualification yet. Pipeline58108 is the only pending
operation. Next apply native test migration, rerun native cases and owner suite.

074 localizer58108 explicitly terminated (owned Jest PID1107143, command verified)
after stall; targeted66474 already proved three outdated native fixtures. It is
an aborted diagnostic, not a product pass. Converted the native test's page map,
source suppliers and all three old payload parsers while preserving RGB, DPR,
scale and delayed-DOM assertions. Frozen manifest extended test-only; runtime
and prior scoped checks remain unchanged. Native targeted rerun now pending.

074 fixture diagnosis:88113/75337 explicitly canceled after stalls, not passes.
Single native capture60594 passed (one RGB test,30unselected). Bounded controls
trace97686 timed out30s while CDP continued acknowledging frames; the native
settings fixture still returned only {page}, so mandatory source admission
correctly rejected it and its first-frame await never settled. Converted that
remaining provider and nested optional timing decoder; retained original quality,
FPS and byte-count assertions. Test-only freeze updated; native scoped rerun
pending. No runtime changes or deployed074 build. No timeout/floor weakened.

074 native61085 consumed0:10pass/21unselected in4.279s, including pixel identity,
polling CSS/device dimensions, four DPR/scale cases, CDP+poll quality/FPS/timing
settings and delayed-DOM startup/stop. All obsolete native fixtures converted;
no runtime edits since initial freeze. Full owner rerun and managed restart
admitted concurrently on frozen source (exact IDs in tool receipts). TG receipt
precedes only these test-fixture changes; runtime scope is unchanged.

074 exact pending: owner31157 and managed restart52120. All earlier diagnostic,
Test Genie and board operations consumed; original runtime freeze unchanged.

075 independent read while074 qualifies: recall46032 consumed0,63hits/10corpora
with provider degradation (raw CLI text includes a malformed UTF-8 fragment;
read with replacement only for display). Existing RF007 source shows useTimeline
and useFrameStream each subscribe_recording on separate API sockets; Hub sends
binary frames to both, while the timeline provider has no binary consumer.
Hypothesis: one visible canvas receives two network copies of each callback
frame. Added per-socket byte counters and a uniquely timestamped accepted frame
to the retained074 native producer to observe this independently of source
rejection. No075 implementation, baseline or product claim yet.074 owner31157
and restart52120 remain pending, runtime frozen and unchanged.

074 qualification scope note:073 API/driver binaries remained live before the
managed restart. The UI is served by the development lifecycle and can read
edited source, so earlier shorthand “live073 unchanged” is not an isolated
claim about the frontend during implementation. No native074 qualification was
run against that mixed boundary. Final native checks require the restarted
API/driver build and frozen UI source.075 recall degradation specifically:
source-ledger.agent-memory and source-ledger.scopes deadlines.

074 restart52120 consumed0; API/driver/UI healthy, build
6f65b5a0f714c2296ffa9e0ac658e4306b758a9ec4fa49a9b5d836159931c79d.
Frozen32paths unchanged. Native70340 consumed0:16/16. Original held Red bytes,
valid old-page receipt, retired lease and wrong execution cannot paint over Blue;
current callback reaches actual viewer with canonical ID and no producer secrets.
HTTP source mapping/rejection, canonical/driver Blue and cleanup controls pass.
Screenshot inspected. Receipt/tmp/bas-frame-identity-native-074.json. Workflow
preservation now running; full owner31157 remains pending. No source changes.

075 observation from074 native receipt: one uniquely timestamped accepted frame
arrives twice on two currently open browser WebSockets for one canvas. One socket
received9binaryframes/65545bytes, the viewer8/61737bytes (different start times;
not a statistically controlled bandwidth comparison). useTimeline's text-event
socket has no binary consumer. This supports the RF007 redundant-fanout
hypothesis and warrants a focused reproduction/repair after074 qualification.

Adversarial visual note: final074 screenshot shows the fixture's single heading
repeated across the blue canvas. Source identity/pixel-color checks pass but do
not establish full image fidelity after viewport resizing. The cause (producer
capture, rendering or probe artifact) is unverified; retain the screenshot and
investigate with a coordinate-marked fixture against direct browser capture.
Do not treat the uniform-color test as image-fidelity or performance certification.

074 saved workflow98619 consumed0:12/12,three independent effects, execution
92bee69c-73c7-4c8e-b636-77bd39869348, /tmp/bas-recording-e2e-08Lodh. Owned sessions,
workflow, project and artifacts cleaned. Profile API metadata equals073. Full
profile oracle28699 and owner31157 exact completion receipts follow. Source
unchanged since freeze; no075 source edits.


### BAS-WORK-074 final — 2026-09-23 UTC

Full owner31157 consumed0:uh-20260923-072637-8cca25df81e1eae8a78be0148262c304,
1826tests/123suites pass,2tests/1suite existing skips,387580ms. Profile28699
consumed0:three complete identity reads match original rollback; API metadata
matches073. All frozen32paths unchanged. Final receipt
frame-source-2026-09-23.json records native16/16, workflow12/12 with3effects,
baselines, source identities, failed/aborted diagnostic history and limitations.
Runtime+25,Go+30,TS+46 are explicit; whole-tree duplication debt31313->31221 is
an observational shared-tree scan, not exclusively attributable to this cycle.
No074pending operations. Continuous goal moves to confirmed RF007 duplicate
binary delivery, with RF047 image-fidelity follow-up retained; no goal completion.


### BAS-WORK-075 admission — 2026-09-23 UTC

RF007 / W3 / J05. Feedback reread; recall already consumed, no pending074 work.
Confirmed native duplicate delivery (two copies of one source-bearing callback)
plus source identifies the event-only timeline's subscription as the redundant
recipient. Target documented before implementation. Retain one recording
subscription contract with an explicit frames:false choice for the timeline;
default remains frame-capable for existing consumers. Hub owns binary filtering
and frame-subscriber presence. No shared-UI lifetime rewrite or new transport.
Prospective12path baseline /tmp/bas-before-075/manifest.json recorded. Add
maintained event-only/default/opt-in/replacement/queue-accounting controls and
native exact-copy check before editing runtime. Driver code is unchanged.

075 maintained Go red first attempt had an expected-counter int64/uint64 fixture
compile mismatch; corrected before behavior evidence. Red2: event-only delivery,
drop accounting and replacement-choice failures; default/explicit-frame controls
pass. Native13159 consumed1:16pass/1fail, exactly2copies where1expected, source
identity controls and owned cleanup retained. Added locked RecordingFrames intent,
frames:false from timeline, converted frame-subscriber query across all callers
without an alias. Page/timeline/performance fanout unchanged. Four racepackages
41194, UI560/29 91731, types23290 and API build3881 all pass. Frozen affected
paths /tmp/bas-frozen-075.json; metrics /tmp/bas-frame-fanout-metrics-075.json.
Driver0741826/123 qualification retains identical source. Next managedrestart,
native exact-copy/timeline preservation, workflow/profile and TG/board.

075 exact pending restart87738. TG53455 admission/one quiet wait/catalog consumed,
run20260923-074214-596bc818; summary1140findings/103long/405complexity/610duplication/
21coupling/debt31210, budgets fail. Board76079 consumed0,all17unknown;
contract/inventorypass. Scoped runtime2200->2203(+3),Go301->303(+2),44functions
unchanged; scopedTS65->65,28codepaths unchanged. Cumulative affectedGo-97.
No net per-cycle simplification claim; target removes unused network work.
Frozen source unchanged.076 independent image-fidelity recall49416 pending;
no076 source changes. Artifact decode completed; progress append initially used
api cwd and was corrected here (no product failure or extra test admission).

075 restart87738 consumed0, API/driver/UI healthy; frozen source unchanged.
076 recall49416 consumed0:67hits/10corpora, source-ledger memory/scopes deadlines.
No076 runtime edits.075 native exact-copy qualification begins now.

075 native3588 consumed0:17/17. Unique current callback now has1networkcopy
(red2); timeline socket receives0frames/0bytes, retired viewer2/7616bytes and
current viewer7/52953bytes. This proves removal of redundant delivery for the
controlled single-canvas case, not a broad bandwidth/FPS band. Allsource/HTTP/
canonical selection/cleanup controls pass; noframeerrors. Artifact
/tmp/bas-frame-fanout-native-075.json. Savedworkflow81770 nowpending; after it,
runprofilepreservation. Nootherpending075 operation.076 producer preparation
can proceed independently without runtime edits.


### BAS-WORK-075 final — 2026-09-23 UTC

Workflow81770 consumed0:12/12,3effects,executionb1e33987-f1a6-445c-9a51-07f6b3d70a3b,
/tmp/bas-recording-e2e-MX66dE; owned cleanup complete. Profile71137 consumed0,
three complete reads equal original rollback; API metadata equals074. Final
frozen paths unchanged, no pending075 operations. Receipt frame-fanout-2026-09-23.json
contains before/after exact-copy evidence and honest scope/complexity limits.
No extra driver owner run because074source is unchanged. Continuous investigation
moves to RF047 pixel fidelity, using an independent coordinate grid and fixture
DOM measurements to distinguish capture from viewer/compositor defects.


### BAS-WORK-076 initial experiment — 2026-09-23 UTC

Native98526 consumed1:21pass/4fail, all owned cleanup successful, receipt
/tmp/bas-frame-fidelity-076/receipt.json. All12 coordinate-color checks pass;
fixture reports one heading and inspected wide screenshot shows one heading.
Persistent tiling was not reproduced with this grid. Four aggregate dimension
assertions failed first at HTTP JPEG dimensions (DPR2 device pixels versus CSS
stream pixels), preventing independent stream/canvas assertions. The narrow
fixture also has layout width980 without a viewport meta tag, so DOM innerWidth
is not yet a valid screenshot-width oracle. No product repair or image-fidelity
qualification inferred from these failures.

Source shows HTTP frame capture omits scale while returning viewport dimensions;
recording start also starts a new stream without forwarding the existing selected
scale. Explicit CSS/device session configuration exists. Next discriminator adds
viewport meta, records HTTP declared versus decoded dimensions separately, and
observes requested device-scale streaming before and after recording starts.
Keep CSS/device preservation controls; do not label HiDPI itself a defect or
infer input-coordinate failure (the input mapper explicitly supports HiDPI).
No076 runtime/test-baseline edits; only the native producer is being refined.

076 refined producer /tmp/browser-automation-studio/frame-scale-076.mjs now has
viewport metadata, a small heading animation to make frame receipt observable,
separate declared/decoded dimension assertions, and a deliberate extra frame
observer before recording. That observer is an intentional second consumer,
not a recurrence of075's unused timeline traffic. CSS run94360 pending; device
variant must run sequentially after owned cleanup. No076 runtime edits/baseline.

076 refined CSS94360 consumed1:27pass/11fail; device15518 consumed1:25pass/13fail,
all owned cleanup passes. CSS HTTP JPEG is2x while selected stream scale isCSS;
requested device preview/recording streams remain1x. Native receipts
/tmp/bas-frame-scale-{css,device}-076/receipt.json. Initial CDP frames can also
be87pixels shorter than current DOM viewport, requiring separate timing/viewport
investigation. Source confirms session-start and recording-start omit scale;
SessionSpec currently discards it. HTTP screenshot omits scale. Frame cache
explicitly defines width/height as viewport dimensions, and input mapping supports
HiDPI; reject the earlier ungrounded assumption that device JPEG dimensions must
always equal that metadata. Clarify the existing coordinate contract, preserve
CSS/device capability and test the selected bitmap scale separately.

Target written; prospective8path baseline /tmp/bas-before-076/manifest.json.
Retain immutable preview scale on SessionSpec and use it at all three capture
admissions, including recording restart. No new preferences registry or transport
retention. Add maintained route regressions before runtime edit. No076runtime
changes yet; previous native operations consumed. Current deployed075 remains
qualified within its documented scope.

076 route-red first invocation used BAS root and failed before runner admission; no product verdict. Corrected driver-cwd run96010 consumed1: six intended scale assertions fail. Extend baseline to existing native pipeline test fixture (ninth path) before adding its required admitted SessionSpec; runtime handlers will use the actual required session contract without fake-only optional chaining. No fixture semantics or timeout changes.

076 final affected routes27749 consumed0:58/3; initial candidate5917 54/3,
 types55817 pass. Runtime chosen scale has one admitted SessionSpec owner, no
 new registry or retained disposed stream. All three capture admissions converted.
 Nine changed paths frozen /tmp/bas-frozen-076.json. Native producer preserves
 prior failed receipts and distinguishes CSS metadata, selected scale width and
 full-viewport completeness, including actual recording stop/start. Owner suite,
 managed restart and scoped tidiness next; no further source edits during runs.

076 managedrestart85934 consumed0; API/driver/UI healthy, build
9be7afbaea47455b7bfdf28e1884a7234a3868c6ec21c00e15509e66ccfdd688. Frozen9paths
unchanged. TG66780 admitted20260923-081726-89c15d22; onequietwait/catalog/decoded
artifact consumed:1140findings,103long,405complexity,610duplication,21coupling,
31210debt; budgets fail unchanged. Board27192:17unknown, contract/inventorypass.
Runtime1792->1800(+8),scopedTS161->166(+5),25codepaths; Go unchanged,cumulative-97.
No net per-cycle complexity-reduction claim. Fullowner14927 pending. NativeCSS
79655 consumed1:43pass/8fail; allselected-scale widths, HTTPCSS capture andcolors
pass;7fullheight mismatches retain87px shortage. Stop/start probe usedwrongAPI
path and failed JSONdecoding; corrected producer preview-scale-native2-076.mjs
uses existing session-qualifiedstop route. Originalreceipt retained; no product
failure inferred from this harness error. Cleanupcomplete. Device30559 pending.

077 geometry diagnostic: firstproducer38643 failed at rebrowser setContent
loadtimeout before observations; contexts/browser closed in finally. Changed
fixture to dataURL navigation as maintainednative suites use;28942 consumed0,
/tmp/bas-cdp-geometry-077-v2.json. Compare actualChromium launchmodes, mobile
true/false, screenshot overlap and explicitCDPscreenmetrics. No077runtimeedits.

### BAS-WORK-076 checkpoint — 2026-09-23 UTC

Owner14927 consumed0:uh-20260923-081722-ec206378eccf5cc80317fdce4271c193,
1836tests/123suites,2tests/1suite existing skips,370101ms. Native2CSS18055
consumed1:56pass/7height failures. Device30559 consumed1:46pass/17fail; initial
and narrow staleDOMdimensions/whitepixelregions are real unresolved observations,
not discarded as scale-policy successes. Later wide/restored/restarted stages
pass all pixel and dimension checks. Recordingstop/restart and cleanup pass.
Workflow4525 consumed0:12/12,3effects,fabb2e3f-20a2-42e7-9fd5-5a7b02d608c4,
/tmp/bas-recording-e2e-qjF3KN. Profile80267 consumed0:3originalrollback identity
reads; APIequals075. Allfrozenpaths unchanged. Receipt preview-scale-2026-09-23.json
qualifies scale propagation only, explicitly retains geometry failures; no
fullimagefidelity/release claim. All076operations consumed, continuous077 proceeds.

077 direct regularChromium with headless:true/channel:chromium22703 consumed0:
allinitial/afterscreenshot frames still640x393; explicitscreenmetrics restores
640x480. Shell control passes without intervention. Reject launchflag change as
sufficient repair; keep existing service-worker/audio capability route unchanged.

### BAS-WORK-077 admission — 2026-09-23 UTC

Feedback reread;076all operations consumed. Existing076geometry recall reused.
Direct geometry17692 consumed0: Emulation.setVisibleSize repairs stable87px
height shortage afterinitial/landscape/portrait viewport updates, mobiletrue/false.
/tmp/bas-cdp-geometry-077-v4.json. Concurrent screenshot/resize59466 consumed0
observes staleDOM dimensions and whitepixels underregularChromium, and staleDOM
underheadlessshell. Same16-stage matrix with screenshotadmission paused and
in-flightcapturejoined72687 consumed0:everyDOM/image dimension correct and zero
white samples; /tmp/bas-viewport-polling-serial-077.json. Serial ordering alone
solvespolling content, compositor command solvesCDPheight; neither alone is
claimed to solveboth. No geometrysource edits before these discriminators.

Necessary dependency-owner extension: existing canonical patch, same approved
rebrowser1.52.0, package/lock metadata viaSDA, SDKPage and Chromiumviewport owner,
BAScontextbuilder videooverride and existing native/SDKtests. Search-hub discovery
22hits/3corpora retained /tmp/bas-dependency-patch-discovery-077.txt. SDA exact
install dryrun approved; /tmp/bas-dependency-patch-dryrun-077.json. No rawmanager,
registryedit, launchmodechange or SDKruntime monkeypatch. Targetdoc written,
prospective7path baseline /tmp/bas-before-077/manifest.json plusactualtwoSDKsource
files and hashes in sdk-manifest.json. Buildmaintained red proof before patch.

077 maintainedred93412 consumed1:4fail/6pass. RegularChromiumCSSfirstframes
320x153instead240atDPR1/2; viewportmutatesduringbothsuccessfulandfailingcapture
barriers. SDAinstall approvedandinstalled, exactpatchhash
562cef07ec0eda59d38fa3ae8c42a61c1d5c563c534728c887fd14b6ff723eac; installedtwoSDKfiles
byte-matchcandidate. Lockdiffonlypatchhash, packageunchanged. Focused24096
consumed0:10/10. Nativeunchangedconcurrentresizeproducer78114 consumed0:16stages,
96samples,zeroDOM/image/pixelmismatches; /tmp/bas-viewport-polling-sdk-077.json.
Video3632 consumed0:55paintedframesacrosstwoDPR2recordings,zeroincorrectbottoms;
/tmp/bas-video-geometry-077/receipt.json. RemovedBASvideo-onlymetrics listener;
retainedunrelatedlayoutstabilization. Frozenaffectedpaths /tmp/bas-frozen-077.json;
fullowner, managedrestart, governance/tidiness/board and livequalification next.

077 types99435 consumed0. Governance advisorypass with44existingwarnings;
securitydepsstatus shows owner available/indexready but isnot a new BASsecurity
scan or cleanverdict. TestGenie39225/onequietwait/artifactdecode consumed,
20260923-084020-f513a66b:1140/103/405/610/21/debt31210, unchangedbudgetfailures.
Board44580 consumed0:17unknown; contract/inventorypass. MeasuredBAScontext528->485,
SDK1845->1852,totalnet-36runtime; complexity744->742,codepaths318->316. No debt
exportedoutsideBAS measurement. SDK `_updateViewport` preserves no-viewport/
Android/explicitwindow-boundary branches; no nativeplatformqualification implied.
Managedrestart46814 consumed0; threehealths200/buildaae1b4d070616993eb8c6e33f0b6b4af887c1fe8006a6a06113e3bf828f64dda,
sixfrozenrepo pathsunchanged. NativeCSS96228pending, fullowner52054pending;
no others pending. No source changes afterfreeze.

077 nativeCSS96228 anddevice94383 consumed0:63/63each. Everyfixturecolor,
CSSmetadata, bitmapscale andfullviewport assertion passes before recording,
initial/narrow/wide/restored and recordingrestart. Ownedcleanup complete;
/tmp/bas-preview-geometry-native-{css,device}-077/receipt.json. CSSwide screenshot
inspected:singleheading, correctfourquadrants/fullcanvas. This supersedes076's
observedgeometry failures within this fixture and workload; rapidcommandordering,
sourceepochs, broadperformance/nativeOS remain unqualified. Fullowner52054 still
pending; savedworkflow next. No changes afterfreeze.

077 savedworkflow52198 consumed0:12/12,3effects,execution37bb226a-cd84-4227-a357-c6ffb22ac04e,
/tmp/bas-recording-e2e-crC1FP,ownedcleanupcomplete. Profile42812 consumed0:three
complete reads equaloriginalrollback, APIequals076. Onlyfullowner52054pendingfor077.

078 freshUI-state observation during077visualinspection: address bar is empty
while the selected live tab renders the fixture. Native23809 consumed1:7pass/
2fail, initialattach and reattachment URLvaluesempty; both canonicalpage snapshots
and ownedcleanup retained. /tmp/bas-browser-address-native-078/receipt.json.
Firstcheck reads immediatelyaftertabvisible; refine with bounded5s opportunity
before declaring persistent productfailure. Repeatedheading remains unreproduced,
now superseded by fullgeometry fixture qualification, not an exhaustiveimageproof.
No078runtime/testchanges,077frozenpaths unchanged; separate078recall admitted.

### BAS-WORK-077 final — 2026-09-23 UTC

Fullowner52054 consumed0:uh-20260923-083931-6e0f297c119278874a5642e082af0c0d,
1842tests/123suites pass,2tests/1suite existing skips,403684ms. Allsixrepofrozen
paths andtwoinstalledSDKmodule hashes stillmatchqualification. Finalreceipt
renderer-geometry-2026-09-23.json captures native126checks,96concurrentresize
samples,55video frames, workflow/profilepreservation andnet-36lines/-2complexity
includingactualSDKsource. No077pending operations orfullreleaseclaim.

078 boundednative88684 consumed1:7pass/2fail afterallowing5seconds foreachURL
observation. Addressbarremains emptywhileAPIcanonicalpageURLmatchesfixture;
cleanupcomplete. Recall97655 consumed0:64hits/10corpora, relevantnavigation and
captureprograms do not replace a UIhydration oracle. Callback-onlyURLassignment,
restoration andlast-actionfallbacks appear to omit existing-page snapshotadmission;
sourcehypothesis requires maintainedregression before repair. No078sourcechanges.

### BAS-WORK-078 admission — 2026-09-23 UTC

Persistent address-bar red confirmed afterbounded5swait onattach/reload; canonical
pageURLis correct. Owner mismatch: initialsnapshot mutatescanonicalstore butURL
UIonly follows threeoptionalusePagescallbacks plusrestored/last-actionfallbacks.
Callbacks' only runtimeconsumer isRecordingSession; restoredURLhas nootherreader.
Targetwrittenfirst: selectedvalidatedPageURL isobservationowner; preserveexplicit
intent and page-lifetimefences, removeoldcallback/statefacadescompletely.
Prospective7pathbaseline /tmp/bas-before-078/manifest.json. No driver/APIchanges;
077qualificationretained. Addmaintainedobservation/hydration/selection/control
regressions beforeruntimeedit. Historyreadattribution andinitialbuttonstate are
notprovenbyanaddress-stringrepair and remain separateinvestigation.

078 maintained red26339 consumed1: three intended location-observation failures.
Runtime now observes validated canonical selected Page in RecordingSession;
useBrowserNavigation accepts observedUrl without issuing intent. Removed three
callback facades, forwarding selection helper, restored-URL hook/store state,
last-action fallback and stale-action placeholder. Kept creation notifications,
explicit command/page fences and local URL draft. Focused19472 consumed0:54/2;
types72166 consumed0. Seven affected paths frozen /tmp/bas-frozen-078.json.
No driver/API source change; 077 owner qualification remains source-identical.
Next: record-mode suite, managed build/restart, actual UI hydration/selection/
restoration controls, workflow/profile preservation and scoped debt measures.

### BAS-WORK-078 final — 2026-09-23 UTC

Managedrestart66619 consumed0; API/driver/UI healthy, build98640f86390eacca95a1aadd209d761c32e6c162c05389e6f9576a6fbc97758d.
Native32954 consumed0:9/9 original hydration/reload reproduction now passes.
Controls32644 consumed0:16/16, including draft, switching, external navigation,
selected/final closure and real saved-profile close/reopen. Reload screenshot
inspected: correct URL, one heading, complete four-quadrant preview. Workflow60032
consumed0:12/12, three effects, execution8c80d922-1b39-46e3-adcc-2fe7b819de22,
/tmp/bas-recording-e2e-UvBA1l. Profile65595 consumed0: three original rollback
identity reads; API metadata equals077. All owned fixture cleanup complete.
Seven frozen paths unchanged. Driver/API untouched, retain077 owner qualification.
Final receipt address-bar-observation-2026-09-23.json. Runtime2914->2845(-69),
scopedTS740->715(-25), paths267->261. Large RecordingSession still1424 lines;
this repair removes competing URL policy rather than exporting it to a wrapper.
Scenario-wide debt remains31210 and budgets fail;17release outcomes unknown.
No078pending operations. Next adversarial angle: initial history availability and
page/lease attribution of navigation-state and navigation-stack reads (RF099/RF038).

### BAS-WORK-079 admission — 2026-09-23 UTC

Feedback reread; no new operator feedback. 078 fully qualified. Recall61804
consumed0,60hits/10corpora. Native85115 consumed1:10pass/3fail; browser confirms
can_go_back=true but UI Back disabled after5s and reload, and both API history
reads ignore stale canonical page_id, returning200 instead409. Owned cleanup
complete; /tmp/bas-history-native-red-079/receipt.json. These are actual failures,
not unavailable validation. Target written before repair;13path baseline at
/tmp/bas-before-079/manifest.json. Existing Session/recordingOwner own lifetime;
no new authority registry or history cache. Internal read caller discovery found
only canonical Go Client and owned driver routes; migrate all in one boundary.
Maintained tests must cover lease/page rejection before CDP, mid-read handoff,
API stale-selection and UI capability refresh without draft replacement.

079 driverred58594 consumed1:12 expected ownership/read failures. UIred29936
consumed1:2 capability observation failures; initial7866 customreporter marks
skips failed, so defaultreporter is retained. Go first compile missingfmt fixed;
red72714 consumed1:12subcases fail, bypass/mock path returns wrong200/no ownedread.
Firstcandidate driver56131 passes111/1; UI92226 passes35/1; Go65049 pass;
driver17960 and UI10545 types pass. Newest-read race then reproduced2failures
with maintained controlled completions; capability read now aborts on newer read
or command receipt, retains page/session cancellation and does not overwriteURL.
Added Session transport/closed-state regression.13paths frozen at
/tmp/bas-frozen-079.json. UI86588, Go36719 and UItypes17263 pending; full driver
owner, lifecycle and live fixture follow. No source changes afterfreeze.

079 native94757 consumed0:13/13 original reproduction passes on healthyde7a18.
Controls27130 consumed1:17pass/1fail, an asserted empty history for a newly
navigated tab. Discriminator56514 consumed0:20/20 and retained browser stack proves
that tab hasabout:blank inback_stack; Backenabled was correct. A separate new
about:blank tab has both capability booleansfalse and both UIbuttonsdisabled.
/tmp/bas-history-controls2-079/receipt.json. This corrects the test oracle, not
product semantics; first failed receipt remains. UIpopup selectedpage read,
Back/Forward effects, implicitread binding, malformedpage and stale-page rejection
pass. Workflow49554 consumed0; profile preservation follows. Fullowner42193
pending; no source changes after13path freeze.

080 next concern scout, no edits: recall43460 consumed0; raw provider output has
invalid UTF8 bytes, replacement decoding permits reading and original retained.
Existing viewport investigation from077 retained. Source has duplicate
page.setViewportSize in route and stream strategy, fire-and-forget stream
updates, pending-update dropping, and no Session/page admission on API viewport
route. API also reads only ActualViewport while the driver emits flat width/
height, suggesting zero-sized success receipts. Native discriminator must
confirm semantics before repair.

### BAS-WORK-079 final — 2026-09-23 UTC

Fullowner42193 consumed0:uh-20260923-091656-ac715cd37bc57d8c03137174ca003acd,
1854tests/123suites,2tests/1suite existing skips,371646ms. Workflow49554 consumed0:
12/12,threeeffects,execution314b875b-f4f7-4842-889e-d9a4c171af3d,
/tmp/bas-recording-e2e-huYxqC. Profile68479 consumed0:3complete originalrollback
reads,APIequals078. Finalreceipt history-read-ownership-2026-09-23.json.
13frozenpaths unchanged; no079pending operations. Core read admission function
complexity14; browser observation remains CDP-owned, existing Session/lease owner
reused; no parallel navigation history map or new authority registry. This adds
necessary checks (+86runtime,+18Go,+12scopedTS); not a net simplification cycle.
Scenario-wide debt31081/budgets stillfail and all17release outcomesunknown.
Next080 native viewport discriminator admitted. Initial producer failed atparse
withduplicateconstbefore and caused nofixture/browser/profile effects; corrected
producer pending12608. No080implementation edits.

080 native12608 consumed1:6pass/4fail, including one producer reference error
from an overbroad local-variable rename; three product assertions independently
failed and cleanup completed. Corrected producer75964 consumed1:7pass/3fail,
/tmp/bas-viewport-native-red3-080/receipt.json. APIresize800x600 returns0x0 while
fixture observes800x600. After selecting a second tab, resize with first canonical
page_id returns200 instead409 and changes second document640x480->1100x750.
These are real effects under stale page intent. No080source/test edits yet.

Source ownership review: route capturesSession beforebodyparse, callsSDKviewport,
then fire-and-forget streamupdate. CDPstrategy callsSDKviewport a second time,
ignores changesunder20px and drops updateswhilepending. Stream should observe
already-applied viewport, never mutate it. Candidate repair retains existing
stream-notification entrypoint as a bounded capture refresh, passes capturedPage,
awaits/revalidates it and removes pending/status/result facades and polling no-op.
No new timer, queue, history/viewport cache or per-caller SDKpatch proposed.
Existing SDKcapturequeue from077 remains viewport/screenshot serialization owner.
UI desired-size and session/page lifetime must fence delayed config/transport;
retained bounds must synchronize when selection changes. Existing bounds equality
anddebounce guards are preservation obligations (May03recall read).

### BAS-WORK-080 admission — 2026-09-23 UTC

Target written in architecture before implementation. Prospective26-path baseline
/tmp/bas-before-080/manifest.json; no source edits before baseline. Own viewport
command once at Session/page SDK owner, observe capture sizing through current
stream generation, remove duplicate setter/obsolete facades, and fence UI sync.
Maintain API compatibility for callers omitting page_id by binding currentselection
at admission; explicit stale ID must409 without effects. Existing SDKpatch077 and
saved profiles/workflows remain preservation boundaries. Red native7pass/3fail
is retained; add meaningful maintained ownership/receipt/refresh/lifetime tests
before editing runtime. No dependency change or new service is required.

080 maintained red: driver command7fail (no session, command completed), Go7055
consumed1 all4 ownership/receipt subcasesfail, UI33197 consumed1 four request
lifetime/receipt failures. Logs/tmp/bas-viewport-{driver,go,ui}-red-080.txt.
Added capture-observation regressions inexistingCDP suite:1/10/200px exactcapture
refresh without browsermutation, stale-page refusal beforeSDKeffects. Run admitted.
Runtime remains unedited; only4test files changed sofar. Baseline26paths retained;
no freeze yet. Next:consume capture red, implement command/receipt/stream lifetime
boundary and migrate obsolete tests/callers without weakening desired semantics.

080 capture red consumed1:four intended failures,0SDKmutation and exact1/10/200px
refresh/stale-page tests. API/Client/Session and driver viewport runtime candidate
implemented; Go flat response removes ActualViewport alternate, emits/validates
internaldriverpage source. recordingSelectedPage helper shares explicit/implicit
selection binding with079reads. Driver admits afterbodyparse, capturesPage,
performsSDKmutationonce, awaits captured-page streamrefresh and revalidates.
Stream candidate removes duplicateSDKsetter, smallresize threshold, publicpending/
resultfacades and pollingno-op; existingcapturegeneration governslatestrefresh.
Focused75676 consumed0 (inspect retainedlog forcounts). Added further adverse
A->B->original-size while CDPacquisitionpending discriminator; orderred admitted.
IMPORTANT: candidate incomplete and NOT frozen/deployed. UIruntime unchanged;
old interface/caller tests still need legitimate contract migration, GoSession
and oldhandler fixtures require owned sessions/newreceipt, types notyet rerun.
Baseline26paths retained; runtime edits are within it. No dependency changes.

080 initial capture-order regression failsreturn-to-original while attachment
pending; owns(capture) guard fixes it. Focused41356 consumed0:12/2. UI lifecycle
candidate nowimplemented;68054 passes4lifetime cases. Initialdriver types25208
only unusedlogger; import removed. Go33337 passes fourownedviewportcases.
Migrated existing tests to owned requests, concrete receipt and capture-observer
interface; APIunknownsession test nowasserts404, positive real-owned cases remain.
Broader73500 passes81/4; UI22727 passes53/2 (existing act warnings retained);
UI77153/driver28627 typespass; Go16022 focusedSession/APIpass.
Additional return-to-original test during Page.startScreencast (notonlyattachment)
reproduces1fail/1pass; capture now stores its existing started flag ontheowner,
so an in-progress capture cannot be mistaken for a completed same-size capture.
Latestdriver suite admitted; UIretainedbounds/newselection tests92412 consumed0.
No freeze yet; after consumingchecks, reviewdiff/retiredcallers, measure andfreeze
before fullqualification/build. Current livebuild remains079de7a18; no080restart.

080 final capture-order discriminator failed during pendingPage.startScreencast;
existingstarted flag movedontoCaptureowner so same-size shortcut requires a fully
started currentcapture. Finalfocuseddriver suite consumed0:82/4. UIretainedbounds/
selected-page/no-page cases92412 consumed0:6/6. UIrevert21659 consumed1: aborted
B mayalreadyapply, whilelastSyncedA skips compensatingA. Invalidate lastSynced
when an actualrequest is sent; do not treat abort as effectrollback. Maintained
independent browserWidth oracle verifies this failure. Final UI/Go/types admitted.
26affectedpaths frozen /tmp/bas-frozen-080.json; no subsequent source edits allowed
without recordingnewcandidate/requalification. No SDK/dependency changes.

080 final UI20286 consumed0:574/29; four Go racepackages/build8654 consumed0;
UI72864/driver64343 types consumed0. Retired frameviewport types/pendingfacade/
polling no-op have no remaining source callers, and frame-streaming contains no
SDKviewport mutation. Frozen26paths. Runtime7687->7490(-197), Go+14(cumulative-65),
scopedTS665->679(+14),195->194codepaths. Necessary ownership/cancellation checks
increase measured branching; do not claim net complexity reduction for080.
Fullowner52243 and restart43914 pending. TG76563 admission consumed1, exact
20260923-094617-3a6877a1 onequietwait and artifactfetch consumed; decode follows.
Board40834 consumed0 all17unknown; contract/inventorypass. Native producers ready
withseparate080artifact paths; no postfreeze source edits.

080 deployed buildbb416759; restart43914 consumed0, API/driver/UI healthy.
Fullowner52243 consumed0: uh-20260923-094610-84bf0f09b2bb4a06a734530123e6e849,
1867passed/123suites,2tests/1suite existing skips,427533ms. Originalnative9295
consumed0:10/10, actual800x600 receipt, stale-tab409 and selected document unchanged.
Matrix57858 consumed1: CSS63/63, device59pass/4fail wide/restored stream dimensions
remain1280x1190 while HTTP/canvas match current viewport. Cleanup complete.
/tmp/bas-preview-geometry-native-device-080/receipt.json retained; no qualification
claim. Logs show device uses polling, UI falls back to HTTP; external RCL execution
traffic concurrently uses BAS. Capture freshness/timeout versus ownership is not
yet discriminated. Frozen26paths unchanged; no source mutation or blind rerun.
TG decoded1145/103/408/612/21/debt31065, budgets fail;17releaseunknown.

080 final preservation/diagnostic evidence: instrumenteddevice82277 consumed0,
63/63, no timeout counter; originalfailed session retains20timeouts/10delivered/
4unchanged. This supports load sensitivity, not proven causal attribution. No
quality,timeout,workload or oracle weakened. CSSsmall/rapid9169 consumed0:63/63,
including1px/10px/rapidreturn and restart. Workflow4534 consumed0:12/12,3effects,
execution9596b4d4-6703-4333-891e-ecfb10efb4f4, /tmp/bas-recording-e2e-fkecyd.
Profiles64877 consumed0:3complete original reads/APIequals079. All fixturescleaned.
Frozen26paths unchanged, receipt viewport-ownership-2026-09-23.json includes
cohesion review and faileddevice evidence. Ownership slice qualified; RF007 load
failure remainsopen. No operationspending and no overallreadiness claim.
081 recall7690 consumed0,60hits/10corpora withproviderdegradation. SingleURL capture
program doesnot supply concurrent frame contention oracle; reuse maintained route
and nativefixture producers. Hypothesis: overlapping identicalHTTPpreview misses
queue duplicateSDKcaptures, starving200ms polling; distinguish from browser-wide
rendering cost and viewport metadata/caching defects before selecting repair.

## BAS-WORK-081 — 2026-09-23 — preview capture admission investigation

RF007/RF038, J05/J22/J23. Baseline2paths /tmp/bas-before-081/manifest.json.
080hasno pending operations. Discriminator: overlap eight identical HTTPpreview
reads while the real route's SDKpromise is held, count capture admissions, then
release. Controls vary quality/page/lease, simulate resize/cacheclear beforecapture
completion, and verify errors do not become cached success. Native burstfixture
will measure comparable completion/timeout/frame freshness at unchanged fidelity.
No runtime change selected yet; do not infer all20 observedtimeouts were causedby
HTTPduplicates. Existing200ms watchdog and allquality/workload settings retained.

081 maintainedred76329 consumed1:3fail/11pass. Eight identical reads admit8SDK
captures; viewport changes withinTTL return1024x768 instead800x600; cacheclear
allows pendingoldcapture to return200 and repopulate. Targetarchitecture updated
before runtime edit: retain one session cache lifetime with currentpending/result,
share identicalreads, includeviewport/URL/config inidentity, fenceclear/replacement,
keepdifferentfidelity results valid and failuresretryable. Baseline2paths retained.
Nativeburst92444 pending; no081deployment orfreeze.

081 candidate frozen2paths /tmp/bas-frozen-081.json. Initial27120 consumed0,
14route tests/typespass; finalcontrols63241 consumed0,35tests/2suites/typespass.
Tests preserve oldcache/ETag/page/lease/full-page/scale behavior and add failure
retry, clearall, viewport/URL retirement, fidelity separation, oldfailure/newpending
ownership. Nativebaseline92444 consumed1:18pass/6fail; each of6eight-reader bursts
produces8distinct captures,360.4–398.4ms batch completion and5totalpollingtimeouts.
No cleanup pending. The controlled workload demonstrates redundantcapture pressure;
it doesnot attribute every080external-load failure. Runtime241->188(-53), scoped
TS38->44(+6),10->8codepaths, Go unchanged(cumulative-65). No netcyclomaticreduction
claim. One existingcache owner now coverspendingandcompletedwork; retired three
cache helpers and duplicate same-hash packaging branch, preserve ETaghash semantics.
Ownerqualification/build/nativecomparison next; no further sourceedits afterfreeze.

081 pending operations: driverowner1151, managedrestart6186, tidinessadmission96002,
board/contract/inventory54002. Do not readmit. After admission, onequietTestGenie
wait onreturnedrunID; fetchretainedartifacts. Afterrestartverifybuildidentity before
nativebatch, exactgeometrymatrix, workflow andoriginalprofile reads. No source edits.

081 TGadmission96002 consumed1; run20260923-100635-84760e09 onequietwait
consumed1(testverdict), artifactsfetched and decoded1145/103/408/612/21/debt31065,
budgets fail. Board54002 consumed0,17unknown; contract/inventorypass. No TestGenie
work pending. Driverowner1151 and managedrestart6186 remainpending. Cohesion:
handleRecordFrame complexity20 retains one request/cache/capture boundary in188line
module; source fence9, capturecompletion8. No wrapperextraction or netsimplification
claim. Capture sharing removes real duplicatebrowserwork, testedfailureandclear
ownership; sharedtree tidiness doesnot show a further domain reduction for081.

081 restart6186 consumed0: build e1d69273e302a3e3d684f8bfa3360f90b6bc739603551c2790fb241f09b1c508,
API/driver/UIhealthy, initialsessions0. Native76493 consumed0:24/24 identical-reader
bursts and13/13 cachegeometry; original sixburstbaseline18pass/6fail retained.
Same6x8 device-scale reads at alternating55/60quality:distinctcaptures8->1 each,
medianbatch367.6->95.4ms, range360.4–398.4->78.5–116.8ms, timeouts5->0. Theseare
localobservations withdifferentbackgroundload, notconfidence-qualifiedperformance.
Resize-to-frame107.3ms (withinexistingTTL) returns800x600CSS/1600x1200device image,
independentcolorscorrect. Fixturesfullycleaned. GeometryCSS/device+workflow11463
pending; driverowner1151 stillpending. Frozen2paths unchanged; no new runtime edits.

081 geometry/workflow11463 consumed0: CSS63/63, device63/63 withno timeoutcounter;
workflow12/12,3independenteffects, executiond2609f24-1784-4d33-89c4-1b1ef9a567c2,
/tmp/bas-recording-e2e-c7ZLiK. Original080loadfailure remainsretained; no full-load
certification. Profile85180 and driverowner1151 pending. Independentnext-surface
read5161 consumed0: capture-surface program/owning BASskill combinedread, retained
/tmp/bas-capture-surface-read-082.txt. This available producer returns screenshot
and computed snapshot underoneexecution andonlycapturebinding; no learn bindings.
No082runtimeedit or admittedqualification cohort yet. Investigate qualification
producer before repeating more local UI repairs; all17 release sensors remainunknown.

081 final driverowner1151 consumed0: uh-20260923-100631-127c3f66289d8180cce99354375a6b4d,
1877passed/123suites,2tests/1suite existing skips,378512ms. Profile85180 consumed0:
3complete original reads, APIequals080. Frozen2paths unchanged; no pendingoperations.
Receipt preview-capture-admission-2026-09-23.json records failedbaseline, allcontrols,
localtimings withclaimlimits andcohesionreview. Next082 broadens toward capture
outcome qualification: reuse existingcapture owner, first verify fixedfixture
screenshot+computed snapshot identity/geometry before100-trialcohort. No sensor
may turn unknown to passing from manualflags or incomplete/platform-limitedevidence.

## BAS-WORK-082 — 2026-09-23 — public capture qualification investigation

Capture/evidence outcomes, J08/J23; no runtime edits. 081isqualifiedwithnopending
operations. Read currentreferencecohort: loopback1280x720DPR1, warmbrowser/runtime,
fixedfixture, actual screenshot pluscomputed snapshot,100trials beforep95claim.
Capture-surface combinedread5161 and capturequalification discoverybothconsumed0.
First two ownedfixedfixture captures exercise governedprogram references andCLI
fullowner receipt. Verify output bytes,geometry,computedtree andindependentDOM
sentinels beforeadmitting100-trialworkload. Keep DPR2 asseparatefidelitycontrol.
Ownerartifactstorage/retainedexecutions arequalification evidence, notexternalwork
logging. No provider,realaccount,paidspend orunrelatedURL work authorized here.

082 firstprobe48719 consumed1: producerJSONparser rejected normal CLI footer after
successfulprogram capture; originalstdoutretained. Corrected87304 consumed0:
governedprogram anddirectcaptureCLI bothrequest1280x720DPR1, independentfixture
observesDPR2 twice; PNG2560x1440, computedsnapshotCSS1280x720. Confirms existing
RF016. CanonicalbuildAdhocRequest ignoresDimensions.device_scale_factor and passes
profile unchanged; drivercorrectly honors profiledefaultDPR2. Baseline2paths
/tmp/bas-before-082/manifest.json. Architectureupdatedbefore runtimeedit: translate
explicitDPR into clonedexistingprofile; omittedDPR preservesprofile/defaults.
Three duplicatePNGartifacts remain separateRF016 evidence-policy concern; no
policy/fidelity reduction to passperformance.100-trialcohort notyet admitted.
Retainedfixtureexecutions80b2ebee-77a9-4f4d-9ac1-2206106992ca,
0fa032f6-1c92-4e33-ab0f-239701be5491,e9388afb-8f23-4c8a-8f77-76ac85e28fee;
localfixture serversclosed, captureexecutions terminal, artifacts retained asproof.

082 maintainedred94306 consumed1 withtestnil-deref beforeproperassertion; retained.
Correctedred2 consumed1: explicitDPR casesfail whileomittedpolicycontrols pass.
Candidate nowprojects dimensionsDPR into cloned existingfingerprint, otherfields
preserved. Entirecapturepackage testsconsumed0. Twoaffectedpaths frozen
/tmp/bas-frozen-082.json; metrics/tmp/bas-capture-dpr-metrics-082.json. Nativeoldbuild
DPRmatrix53160 pending; do not restartuntilredfixtureterminates. Next racepackages/
build andmanagedrestart, exactnativegreen, thenlegitimate100-trialcapturecohort.
No driver/UI/SDK/proto/dependency change. Duplicatescreenshotpolicy remainsRF016.

082 oldbuildnative53160 consumed1:13pass/5fail, no producererrors. Explicit1 and0.5
observeDPR2 and2560x1440PNG; explicit2/omitted2 andcomputedsnapshotcontrolspass.
/tmp/bas-capture-dpr-native-red-082/receipt.json retainsfiveexactexecutionIDs/artifacts.
No fixture/server/sessioncleanup pending. FirstGo raceadmission usedwrongcwd and
exitedbeforetests; /tmp/bas-go-races-wrong-cwd-082.txt retained. CorrectAPIcwd40924
admitted racecapture/workflow/executor thenbuild; awaitcompletionbeforequalification.
Runtime858->871(+13),Go164->168(+4), cumulativeaffectedGo-61, TSunchanged.
No netcomplexity reduction claimed; missingrequestfieldnowhonoredthroughexistingowner.

082 threeGo racepackages/build40924 consumed0; sourcefrozen2paths. Driver0811877/123
and UI080574/29 remain source-identical. Pendingmanagedrestart62678,
TGadmission6138, board/contract/inventory18275. Performance skill loaded; file-only
operatorauthority overrides its visited-tracker/externalnotes prescriptions.
Inspect performance-health ownerforapplicablecapturecohortmeasurement; genericUI/
Lighthouse/build budgets cannot certify correlated capture or DPR perTESTING.md.

082 restart62678 consumed0: build97d9526edbcc4e3733b2983da733f2916f1179808971fb388452284933a35382,
API/driver/UIhealthy,0sessions. Nativegreen35453 consumed0:18/18, requested1/2/0.5
rendererDPRandPNGdimensions correct; omitteddefault2retained; computedtree/retained
artifactbytescorrect. TG6138 admissionconsumed1 run20260923-102317-5bc97adc,onequiet
wait/artifacts/decodeconsumed:1146/103/409/612/21/debt31065,budgets fail. Board18275
consumed0all17unknown;contract/inventorypass. Performance-health availablecommands
coverUIaudit/build/startup/budget; they do not substitute thiscaptureRPC+DPRoracle.
File-onlyauthority suppressesvisited-trackerwrites; requiredskillreadcompleted.
Admit BAS-owned qualification investigation: one declaredwarmupplus100firstattempt
CLIcaptures, fixed1280x720DPR1fixture/selector,concurrency1,freshcontexts,warmruntime,
no retries. Preserveeveryfirstattempt,serviceandCLIlatency,independentDOM/PNG/tree
checks,artifacthashes,hardware/runtime/build/source/contract/fixtureidentity. Local
cohort producer /tmp/browser-automation-studio/capture-cohort-082.mjs; no public
release/sensorqualificationclaim from a standalone localreceipt. Allartifactdata
retained by captureowner; no unrelatedretentiondeletion.

082 cohort46758 consumed0:100/100firstattempts pass plus1declaredretainedwarmup,
no retries. Servicep50=753,p95=755,p99=755,max771ms; CLIp95=926.20ms. 300PNGs,
100computedtrees,total5,104,900artifactbytes; everyviewport/DPR/tree/retainedhash
check passes. Raw /tmp/bas-capture-cohort-082/receipt.json andmetadata/statistics
retained. Localcapturetarget observedmet; no governed/global/platformqualification
claim. ThreePNGs percall remainsknownRF016duplicate-evidence concern. Nativegreen
DPR1 primaryimage visuallychecked:completefour-quadrantviewport andinput preserved.
Nextfinalworkflow/profilepreservation, thenrecord082receipt andchoose meaningful
nextinvestigation. No runtime/source changes duringcohort.

082 finalworkflow74371 consumed0:12/12,3effects, executione10fc5e5-d0d2-4c16-aff4-0051795bbf82,
/tmp/bas-recording-e2e-rv53HE. Profile72948 consumed0:3completeoriginalreads/APIequals081.
Frozen2paths unchanged, no pendingoperations. Durablecapture-device-scale receipt
includes101ownerexecution/artifact references andhashes, statistics/metadata andall
failedproducer/reproductionreceipts. p95 interval754–771ms assumesIID; sharedhost
limits retained. No overallgoalcompletion orreleasequalification. Sourceinspection
also identifies duplicated250ms statuspoll loops and no obvious Connect validation
interceptor onCapture module; theseareinvestigation hypotheses, notyetnewfindings.

## BAS-WORK-083 — 2026-09-23 — capture request admission investigation

082qualified; no pending operations. Feedbackreread unchanged. Hypothesis: Connect
Capture module doesnotinstall descriptor-constraint validation, so out-of-range
DPR/dimensions andinvalidcapture enums reach executor effects. Existingprotobuf
annotations alone are not known to enforceatserver. Beforepickingrepair, exercise
actualConnectmodule withcounted executor: invalidrequestedranges mustreturn
InvalidArgumentandzeroexecution/exportcalls; validbounds/omission remainaccepted.
Baseline3paths /tmp/bas-before-083/manifest.json. No runtimechange selected.
Recall40628pending; inspectexistingrepo validationowners before addingdependencies
or duplicatingnumericpolicy. NativeAPI negativecontroluseslocalfixture only.

083 recall40628 consumed0; repo owners use buf.build/go/protovalidate v1.1.0,
no shared Connect validator exists. Maintained Connect regression exits1 and
proves out-of-range/NaN/infinite geometry enters execution/export; unknown capture
enums enter execution. Paired-dimension and valid-boundary controls retained.
Red /tmp/bas-capture-admission-red-083.txt. SDA preview consumed0: dependency
already approved. Extend boundary to api/go.mod/go.sum via governed install;
no approved-registry hand edits, no new shared framework or copied numerical
limits. Architecture target recorded before runtime edits. Baseline now5paths.
Native small invalid-DPR/viewport/enum fixture will retain navigation effects
and exact error responses; no oversized native browser allocation.

083 native oldbuild88040 consumed1:1pass/4fail, no producererrors. DPR0.25 and
width99 returned200 and navigated; unknownenum99 navigated then returned500;
NaN dryrun returned200. Exact requests/results/fixtureeffects retained in
/tmp/bas-capture-admission-native-red-083/receipt.json. Validcontrol navigates once.
SDA install68996 consumed0, approved protovalidate1.1.0 plus6 new module requirements
and normal Go graph upgrades ofexisting transitives; no registrymanualedit.
Runtime applies precompiled/read-only schema validation before URL resolution;
unknown enums rejected by generated enum membership. Entirecapture package87351
consumed0 including red regressions/valid controls. Installedvalidator documents
concurrent safety and WithDisableLazy removes lazy mutation. Governance command
consumed0 with historical advisorywarnings; security deps status is inventory,
not a fresh vulnerability scan. Dependency changes broaden API verification.

083 focusedGo races/build25874 consumed0. Governed reconcile consumed0 after
sourceimports, synchronizing directdependency classification and sums; no source
mutation. Frozen5paths /tmp/bas-frozen-083.json nowdefines qualificationcandidate.
Earlier races were pre-freeze checks; broadAPI tests/build next applyfinalmanifest.

083 broadAPI10458 consumed1: only handlers/ai TestGenerateAISuggestions_Integration
fails because local Ollama returns null elementText againstrequiredstring schema.
This is actual product/provider-contract evidence, linked existingRF068; retain
/tmp/bas-go-all-083.txt and repair asnext focusedcycle aftercurrentcapturequalification.
No paidprovider chosen; integrationuseslocalresource-ollama. Dependentbuildskipped
byset-e; standalonebuildadmitted. TG28070consumed0(runadmission), exactrun
20260923-104650-a834e2fe; singlequietwaitconsumed1, both tidiness/securityfailed.
FirstartifactreadwrongCLIarguments rejectedbeforeeffects; correctedcatalogrequested.
Board65997consumed0:17pending_telemetryunknown, contract/inventorypass. No candidate
sourcechanges; nativegreen/restart andsecurityfindingsinspection stillpending.

083 security verdict:19errors/358warnings/403info; tidiness1147/103/409/612/22,
duplicationdebt31065. NewCEL0.26.1 dependency has advisoryGO-2026-6094; official
https://pkg.go.dev/vuln/GO-2026-6094 confirmsfixed0.30.0. NativeTypes/ParseStructTag
reachability isnotestablished; do not equate advisorywithprovenBASexposure. Repair
introduceddependency regardless. SDApreview0.30.0 refused out-of-range (not an
automaticapprovalreview); existingapproval pins0.26.1. Necessaryownerextension:
SDA-governed .vrooli/dependencies/approved-dependencies.json CELrecordonly, add
exact0.30.0 whilepreserving0.26.1forunrelatedconsumers; no broadmajorlineallowance.
Goalgrants necessaryownerrepairs; no humanapproval gate. Preview concretegovernance
diff beforeapply; retain original083freeze/qualification, thennew083b candidate
and checks afterdependencychange. Managedrestart83943 stillpendingoriginalcandidate;
do not mutate dependenciesduringitsbuild. No083nativegreen qualification yet.

083 CEL governance preview approve commandwoulddrop historicalmetadata, so didnot
applyit. SDA batchpreview thenapply preservedallmetadata andadded exact0.30.0;
verifiedonly ('go','github.com/google/cel-go') recordchanged. Original0.26.1 remains
approvedforunrelatedconsumers, no fleetinstall. ThreeexpandedAPIracepackagesand
standalonebuild62011passed; fullAPI83packagespass/24notests/oneAIintegrationfail.
Restart83943 stillbuilding; awaitbeforeCELinstall. Read-onlynextAIrecall63073pending.
Both083TGphases fullyconsumed; canonicalfindingsretained /tmp/bas-tg-findings-083.json.
Security19errorfindingspointtorehabilitationreceipts, possibleephemeralleasetoken
orfield-name detections; nottriaged, notassumedrealcredentialexposureorfalsepositive.
Existinggrpc/oteladvisories remain; newCELadvisory repairselected.

083a restart83943 consumed0: API/driver/UI healthy, build905cffcf1d8fe854908d2adb1a1fafe916b7594102dc15116b084efa8bf16ca6.
No finalnativequalificationclaimedon supersededcandidate. CEL0.30.0 governedinstall
andscopedreconcileconsumed0. Final083bfreeze /tmp/bas-frozen-083b.json(5BASpaths),
CELrecord /tmp/bas-frozen-083b-governance.json; wholeinitialregistrybackup retained
forattributionbutunrelatedfuturemetadataisnotsourceidentity. Neednewbuild/races,
fullAPIshortunitpass(no claimonretainedfailingliveAIassertion), securityrecheck,
managedrestart/native/profiles/workflow. AIrecall63073 consumed0,64hits10corpora
withdegradation; relevantownerinspectionshowsresource-ollamagatewayforwards schema.

083b race75093 consumed1: upgradingCELalonebreaks protovalidate1.1.0 runtime
fieldselection, capturetestsreturnInternal onvalidURL. Candidate083bNOTdeployed;
original083a livehealthy. Preserve /tmp/bas-go-races-083b.txt. Ownerreleases inspected:
https://raw.githubusercontent.com/bufbuild/protovalidate-go/v1.3.0/go.mod explicitly
pairsprotovalidate1.3.0 withCEL0.30.0 andexistingGo1.25/protobuf1.36.11; latest1.4
wouldraiseGo/protobuf floorsunnecessarily. Selectmatched1.3.0pair, extendSDArecord
byexactversion whilepreservinghistoricalmetadata/approvals. This is revised083c,
notbypassedregression. Security6736admission0, exact20260923-105423-364811aa,
singlequietwaitconsumed1; fetchartifactsbeforefinalsourcechecks.

083c protovalidate1.3.0 already falls withinexistingapproved '*' record, so no
secondregistrymutationneeded. Install/reconcile58971consumed0. 083bsecuritycatalog
consumed:19errors/358warn/401info, noCELadvisory. Final083cfreeze
/tmp/bas-frozen-083c.json retains5BASpaths; CELgovernancerecordunchangedfrom083b.
Sourceserviceandtests unchanged since083a; onlypaireddependencieschanged. Pending
newrace/allshort/build qualification beforemanagedrestart. Do notdeployfailed083b.

083c race/allAPIshort/build18121 consumed0: capture/workflow/executor racepass;
shorttestscoverAPIpackagesbutexplicitlydonotresolvepreviousliveAIassertion.
Security72780admission1 exact20260923-105638-1cbfa4cd; onequietwait1 andartifacts
consumed, noCEL/protovalidate advisory. Managedrestart083c nowadmitted; no further
sourceedits. Nextnativeadmissionnegative/positivecontrols, scalegeometrycontrols,
same100capturecohort tocheckperformance afternewvalidation, workflow/profiles.
Runtime+16,affectedGocyclomatic+4(cumulative-57),sourceTSunchanged; newlibrary
coupling acknowledged, no simplificationclaimforthisrequiredcontractrepair.

083c managedrestart26167 consumed0: healthy API/driver/UI,0sessions, frozen5paths
unchanged. Build2b8926339eefeabf65ba0d68ae8a2733f803874e24625c40b62a6b531ad0bd2d.
Nativeadmission+scale53427,workflow67761,profiles72010pending. Additionalread-only
securitytriage:19gitleaksfindingsare64-hexstringskeyedbysourcepaths;6independently
matchretainedsourcebackupSHA256 bytes. Fullhistoricalhashprooffor13notyetlocated;
notephemeralcredentialexposureclaim. No historicalevidenceorscannerpolicyedited.

083 finalnative53427 consumed0: admission5/5 (4invalidcases400/zeroeffects, valid
controlnavigatesonce), DPR18/18. Workflow67761 consumed0:12/12,3effects,execution
653b90a2-c1e9-457f-b1f8-28c657c2d1d2, /tmp/bas-recording-e2e-Oh0y4w. Profiles72010
consumed0:3fulloriginalreads/APIequals082. Comparable100capturecohort22852pending,
/tmp/browser-automation-studio/capture-cohort-083.mjs copies082producerwithonly
output/label/source-identitychanges, samefixturegeometry/fidelity/checks/no retries.
Newsourcefreezeincludes5paths plusseparateCELgovernanceproof. No productsource
editsduringcohort. AlltemporaryfixtureServersclosed afternativechecks; ownercapture
artifactsretained asevidence, workflowfixturecleaneditsownproject/execution.

083cohort22852 consumed0:100/100firstattempts+1warmup, no retries, servicep50=754/
p95=755/p99=756,max756ms; CLImedian931.93/p95=978.46ms. Servicep95unchanged;
CLI+5.64% observed, causal/regressionqualificationunknownduetoshared-hostcohorts.
No equal-performanceclaim. Statsand101compactownerreferencesretainedindurable
capture-request-admissionreceipt; rawhash649650c8a90c40768715c277657dabc613558feaca799fed48b8edeb688518d8.
All5frozen083cpathsunchanged. No pendingoperations. Receiptupdatedwithnative
5/5,DPR18/18,workflow12/12,3originalprofilechecks, metrics/dependencysecurityproof
andfailedcandidates. Greenchecks leadtonextactualRF068liveassertion, notgoalstop.

## BAS-WORK-084 — 2026-09-23 — optional AI suggestion text

083qualifiedlocallywithCLIrelativeperformancelimitretained; no pendingoperations.
Feedbackrereadunchanged. Recall63073 consumed0; sourceinspectionconfirmsBAS passes
oneJSONschema throughgovernedresource-ollamagateway andnativeensureclient, without
droppingformat. Reproducedlocalproviderfailure: optional elementText:null rejected,
althoughomissionproducessameemptyGo/APItextandrequiredfieldsarepresent. Hypothesis:
optionaldescriptivetextshouldaccept explicitabsence(null)aswellasomission; required
action/category/confidence mustremainstrict, andnontextnumbers/objects/booleansmust
stillfail. Discriminator: existingactualgenerator+mock-provider contracttable with
nulloptionalmetadata, nullrequiredfields andwrongoptionaltypes; thenlocalprovider
searchfixture smoke withoutweakenedassertion. Baseline2paths/tmp/bas-before-084.
No retry/coercion layer, synthetic replacementcontent, paidprovider orpublicAPI
shapechange. Capturecohortfinishedbeforeany084sourceedit.

084 maintainedred76201 consumed1: nulloptionalmetadata rejected; allrequired/
wrong-typecontrolsremainpassing. /tmp/bas-ai-null-red-084.txt. Changeonlyexisting
sharedproviderschema'sfouroptionaltexttypes to string|null; jsondecodernatively
representsnullasemptystring, no normalizerornewbranches. EntireAIshortsuitepass
/tmp/bas-ai-null-green-084.txt. Freeze2paths/tmp/bas-frozen-084.json beforelocal
providerqualification/races/build. Runtime+2commentlines, executablepolicybranches
unchanged; Go13->13/5functions, cumulative-57. No dependency/driver/UIchange.

084 liveprovider16323 consumed0: sameoriginalsearchfixture passes3/3firstattempts
throughgovernedlocalOllama, no retries. /tmp/bas-ai-null-live-084.txt. AI+AIservice
short race/build58568consumed0. TG18252admission1, captureexactrun/onequietwaitnext.
Board/contract/inventory37802consumed0,17pending_telemetryunknown. Managedrestart
084admittedafterfreeze; existing083capture/driver/UIqualificationretained unchanged.

084 TG20260923-110732-83866f3a singlequietwait/artifacts/decodeconsumed1/0:
1147/103/409/612/22/debt31065, budgets failunchanged. Currentcheckpointupdated.
Next085read-onlyrecall31330 consumed0; existing saved/adhoc execution methods
bothpollstatus every250ms. No performance repairadmitted; inspectrunner's existing
completion ownership before selectingreplacement. Restart084stillpending.

084 boundaryreview: AIService.AnalyzeElements protobuf explicitly makes model
suggestionsoptional atopDOMextraction. Its existingfallbackpreserves extraction
whenmodelunavailable; do notconvertthatpublicpartial-resultbehaviorintowholeRPC
failureaspartofthisoptionaltextfix. Generator itselfstillreturnserror forinvalid
requiredoutput. Explicitpublicpartial-resultdiagnostics andtesthelper'sliveprovider
usagearefuturetestability/UXquestions, notclosedby084 ornewlyimplementedhere.
No extension/sourcechange after084freeze. 085runnerinspection showsoneasync
goroutineentryandtwo250mspollcopies; completionnotificationmayremovebothifterminal
publication/teardown andcancelledwaitersemanticscanbeproved.

084 restart59453consumed0:healthy0sessions,buildsha256:a00722acc3738c5f583c95a71125c60e75ca34710737bd775fa1631752c840a9.
Workflow8098consumed0:12/12,3effects,execution3c644e92-0f0d-4416-a3cc-b3c7b2678985,
/tmp/bas-recording-e2e-Bpmkm0. Profiles82587consumed0:3fulloriginalreads/APIequals083.
Frozen2pathsunchanged; alloperationsconsumed. DurableAIoptionaltextreceiptupdated.
No overallcompletionclaim; next meaningfulcandidate is250mssynchronouswaitpolling
andduplicatedcompletionlogic. ProvidersemanticpreservationnotglobalAIqualification.

## BAS-WORK-085 — 2026-09-23 — synchronous execution completion ownership

084qualified; no pendingoperations. Feedbackreread unchanged; recall31330consumed0.
Hypothesis: saved/adhoc API callerswaitonduplicated250msrepositorypolling evenwhen
their ownasync runneralreadyfinished, addingunnecessarylatency/readload and leaving
early-runnerexitwithoutterminalpublication waitingindefinitely. Candidateowner:
existingstartExecutionRunnerWithOptions goroutinelifetime. Prove baselinebefore
choosingrepair: immediatecompletedrunner, delayedcompletion/teardown, caller
cancellationwithoutcancellingdetachedexecution, failed/cancelledterminalreceipt,
andmissingterminal/persistencefailure. Nativecapturesalreadyhave100trialbaseline,
butdonotattributeCLIp95changeorstair-steptimingtopollingwithoutdiscriminator.
No085sourceedit orqualificationadmittedyet. Preserveasync/publicresponse semantics,
artifact readiness andexistingdetachedroutedtestcontext. No newexecutionregistry,
shorterpollinterval, lostwakeupfallbackorresumed-executionsemanticchange intended.

085 baseline3paths /tmp/bas-before-085/manifest.json. Maintained actualsaved/adhoc
APIred82419 consumed1:12fail/2cancelcontrols pass; zero-wall-timeGo synctest clocks
prove250msimmediate/225msdelayed excess,6readsversus2, earlyreturnbeforeevent
CloseExecution anddeadlineafterfailedterminalwrite. No externalbrowser/provider
inregression. RF105registered, architecturetargetupdatedbeforeproductionedit.
Repair existingrunnerstarterreturnslifetimechannel, closedafterallrunnerdefers;
sharedwaitreadsonepersistedresult, rejects missingterminaltimestamp; botholdpoll
loopsremoved. No globalregistry/newworker/shorterinterval/contextdetachmentchange.

085 firstcandidate4821 consumed1: original12bugassertions nowpass; newlyadded
cancel-detailpreservationcheckexpected'context canceled'butcanonicalexisting
executionOutcome emits'execution cancelled'. Correctedtestexpectationtopreserve
existingpubliccancellationdetail; retained/tmp/bas-execution-wait-green-085.txt.
No runtimechangeforthistestproducererror. Addedearlyrunnerreadfailure andexplicit
asyncresponse/detached-executioncontrols beforefinalfreeze. Packagescommandwas
skippedbyset-eafterfocusedtestfailure; donotclaimpackagepassyet.

085 focusedgreen2/packages47636 consumed0. Independentreviewfoundtwo private
runnerforwardingwrappers withone manualExecuteWorkflowcaller; directthatcaller
totheexistingcanonicalstarteranddeletebothobsoleteforwarders, preservingflat
parametersin@store/ andmanualasyncstatus. Addexplicitmanualasync/parametercontrol.
No newpolicy, no publicmanual APIremoval. Stillbeforefinalsourcefreeze.

FB013 capturedverbatim andoverallstatusanswered; userreaffirmscontinuation.
085 source manualcallerconversion/deletedforwarders applied; followingtestpatch
failedcontextmatchingbeforeanytestmutation, so finalpackagecommandwasnotadmitted.
Resume exacttestedit againstcurrentone-lineclosure, thenfocusedchecks/freeze.
No owneroperationspending;085notyetdeployed.

085 final focused/package77443 consumed0, includingmanualasyncandflatparameterpreservation. Frozen3paths /tmp/bas-frozen-085.json beforequalification. Runtime-15,affectedGo-4(cumulative-61),functions-1. Both250mspollcopiesandtwoobsoleteprivateforwardersdeleted; no newregistry/worker/dependency. Existing800line/higharityrunner remainscohesiondebt; do notclaimbroadarchitecturecompletion. NextthreeGo racepackages/build,managedrestart,nativeworkflow/captureandcomparable100trialcohort.

085 race/build69786 consumed0: workflow, capture and executor race packages pass,
Go build succeeds. Tidiness44880 admission1, exact20260923-113433-a7dc6121;
one quiet wait admitted. Board/contract/inventory15333 consumed0; 17 outcomes
remain pending_telemetry. Managed restart83123 admitted; no source edits after
freeze. Next native workflow, preserved profiles and capture controls/cohort.

Read-only next-boundary recall22377 consumed0 (69hits/10corpora; providers degraded).
Execution snapshot writes and index updates are currently ignored by the runner.
HydrateExecutionProto explicitly makes the DB index authoritative for lifecycle
fields, so a stale optional snapshot alone does NOT prove a stale API status.
Inspect required versus optional persistence and terminal event publication before
selecting a repair; no086 product source changes while085 is qualifying.

085 Test Genie run20260923-113433-a7dc6121 is fully consumed: one quiet wait1,
artifact catalog/fetch/decode0. Tidiness1149 findings/103long/411complexity/
612duplication/22coupling/debt31060; budgets still fail. This is a shared-tree
reading, not all attributed to085. Scoped runtime complexity190->186 and
15lines removed are separately measured. No board row qualified.

086 read-only discovery found a more concrete durability hypothesis: fresh
runner replaces the admission snapshot twice with lifecycle-only protos, while
ExtractCheckpointState depends on its WorkflowVersion and Parameters. That
would erase resume inputs and bypass the workflow-version check. DB lifecycle
fields already override snapshots in HydrateExecutionProto. Reproduce through
the actual public saved execution and checkpoint reader after085 native/cohort
finishes; do not build a new merge/compatibility layer without checking single
ownership. Resumed execution also drops parent routed context (separate gap).

085 restart83123 consumed0; API/UI healthy with build
0e25ce034a87510303b5aff6c083118380fb3e9b0f825d8a707113e004cec1db,
driver healthy/0sessions. Three frozen paths unchanged. Native workflow12412,
profiles40256, admission/DPR controls10500 admitted; wait for these before the
sequential100capture performance cohort. Durable execution-completion receipt
created with current pending IDs; no086 edits yet.

085 native workflow12412 consumed0:12/12 checks,3effects, execution
 d0a7835c-ee13-4da5-ad8a-7e22b75bf577, /tmp/bas-recording-e2e-HXgCYx.
Profiles40256 consumed0:3fulloriginalreads/APIequals084. Capture controls10500
consumed0: admission5/5 andDPR18/18. Cohort56114 now owns100 first attempts plus
one declared warmup; no other native browser workload admitted by this agent.

085 cohort56114 consumed0:100/100 first attempts plus one declared warmup, no
retries. Servicep50=537/p95=577/p99=584/max590ms; CLIp95=780.92ms. Relative to083,
servicep95 is23.58% lower and CLIp95 is20.19% lower; shared-host applicability
limits retained. Success exact95% interval[0.9637833,1]; conservative p95 interval
[572,590]ms assumes IID. Raw receipt SHA256
36b0d593ba38249113914e44cfdaf125c9d2b2d8e637da289bed9c5b4024ad78.
300PNGs/100trees/4,840,400bytes retained, geometry/DOM/artifact checks unchanged.
Durable execution-completion receipt holds compact references for all101trials.
All frozen085 paths unchanged; all owner operations consumed. RF105 locally
qualified; no release row qualified or overall completion claim.

## BAS-WORK-086 — 2026-09-23 — execution admission metadata

Feedback reread;085 has no pending operations. Recall22377 already consumed.
Hypothesis: lifecycle-only writes erase immutable admission parameters/trigger/
workflow version, breaking resume and its version guard. Manual and ad-hoc paths
also omit admitted parameters. Separate admission fault hypothesis: ignored
snapshot-write errors allow browser effects with no recoverable input receipt.
Discriminator: actual saved/manual/ad-hoc methods with blocked executor, disk
snapshot and public hydration/checkpoint reads during/after execution; real
filesystem failure must reject before runner effects. Resume is a preservation
control. Baseline six paths /tmp/bas-before-086/manifest.json. No runtime edit yet.
Possible owner repair: one admission metadata commit, DB index alone owns changing
lifecycle fields; remove destructive repeated snapshots rather than add merge
rules. Confirm observed failures and consumers before selecting implementation.

086 red6859 consumed1: six saved/manual/ad-hoc metadata cases fail during running
and terminal states, plus three actual filesystem-failure admissions start the
runner. RF106 registered; architecture target written before runtime edit.
Selected owner repair is one immutable admission commit shared by all four
callers, with lifecycle-only replacements removed. Existing api-core atomic
writer supplies fsync/rename without a new dependency or private writer. Keep
metadata after index errors because commit outcome may be uncertain; no unsafe
rollback or effect retry. Resume context/terminal publication are separate limits.

086 first green command failed only unused imports after deleting lifecycle
snapshot writes; fixed imports, retained /tmp/bas-execution-metadata-green-086.txt.
Green2 and four packages40940 consumed0. Added immutable-byte, actual resume/version
rejection and index-error preservation controls; final workflow suite consumed0.
Baseline runtime overlay with current tests proves additional version-guard and
index-failure assertions fail on the retained old source (extra-red receipt).
All four admission callers now use createExecution; no direct index-create or
snapshot-update bypass remains in workflow services. Existing atomic writer used,
inputs0600, no new dependencies. Frozen six paths /tmp/bas-frozen-086.json.
Runtime2892->2825(-67), Go540->530(-10),65functions unchanged; cumulative affected
Go-71. Snapshot commit complexity17->8; large runner still55. New shared storage
import replaces local manual temp/rename implementation, not debt export.
Next four Go race packages/build, managed restart, native recovery metadata,
workflow/profile/capture controls and same100capture cohort to check file-sync cost.

086 four race packages/build66120 consumed0. TG88791 admission1 exact
20260923-114840-ca7d52ce; one quiet wait1/catalog/fetch/decode0 fully consumed.
Board/contract/inventory74993 consumed0;17outcomes pending_telemetry. Tidiness
{"totalFindings": 1152, "longFiles": 103, "complexity": 413, "duplication": 612, "coupling": 23, "duplicationLineDebt": 31060}; budgets still fail. Native old085 producer20124 consumed1:
12controls pass, metadata check fails (resolved version absent). Execution
4f51597f-819e-4dd4-84b5-50493b75af82, /tmp/bas-recording-e2e-XbU0YU; exact fixture
resources cleaned, producer/receipts retained. Managed086restart now admitted
only after native baseline cleanup. Six frozen paths unchanged. No further edits.

086 restart48766 pending; no product changes after freeze. Durable execution-
admission-metadata receipt includes exact source and all current evidence/IDs.
Read-only087 recall94483 consumed0. Resumed starter and persistence context both
use Background, unlike the canonical fresh runner; test isolation metadata may
be lost. Executor also uses StartFromStepIndex>0, potentially repeating completed
step0. These are hypotheses requiring actual request/executor tests; no087 edits.

087 temporary overlay probe22791 consumed1 due to missing required event sink;
producer error, not product evidence. Added the ordinary MemorySink to the probe
and reran only this discriminator; no tracked source changed. Candidate086
remains frozen. Retain both /tmp/bas-resume-checkpoint-red[-2]-087 receipts.

087 corrected temporary overlay consumed1 with genuine assertions: three fail
(linear/after_zero,graph/after_zero,graph/after_one), three fresh/flat controls pass.
Actual public executor and independent fake-engine invocation log, no browser.
RF107 registered. Compiler always supplies graph for ordinary V2 workflows, so
repairing only the >0 flat check would leave the main public behavior broken.
Need recoverable graph cursor/continuation semantics; do not merely skip all lower
static indices across branches/loops. No087 source change or qualification yet.

086 restart48766 consumed0, healthy API/UI/driver,0sessions, build
6bda665924ee0f708d0a7afbdddf5c1e2823ac5449451765b69b7748bd64ff46.
Native68445 consumed0:13/13 checks,3effects, preserved inputs/version/trigger;
execution b37d2a5b-74ea-44b7-9ebf-fb1d56974ead, /tmp/bas-recording-e2e-vCLNhC.
Profiles10811 consumed0:3original full reads/APIequals085. Capture controls63489
consumed0: admission5/5,DPR18/18. Same100capture cohort22660 now pending after
all other native workload finished; producer source/fidelity unchanged except
cycle/output/frozen identity. No087 product edits while this candidate qualifies.

086 cohort22660 consumed0:100/100 first attempts plus one warmup, no retries;
servicep50=541/p95=586/p99=594/max595ms; CLIp95=787.58ms. Versus085 servicep95
+1.56%, CLIp95+0.85%; shared-host sequential comparison does not qualify causal
performance. Conservative p95 interval[582,595]ms assumes IID;100success exact
95% interval[0.9637833,1]. Raw receipt SHA256
5888f20d21ea5f4678ab9fc55bb0fbb0045fff3c991f17e5de06ea93b275e066.
300PNGs/100trees/4,885,800bytes; same geometry/DOM/artifact checks. Durable receipt
holds all101compact owner references. Six frozen paths unchanged. No pending
operations. RF106 locally repaired, not full recovery certification: RF107 repeat
risk, missing resume context/lineage, historical erased inputs, uncertain DB
commit reconciliation and power-loss durability remain explicit.

## BAS-WORK-087 — 2026-09-23 — checkpoint continuation without repeated effects

086 fully consumed, feedback reread. Native87122 consumed1 on086: five controls
pass, resume reports COMPLETED but repeats the independently logged effect2times
(expected1). /tmp/bas-resume-effects-087-kId0BY, executions
79681996-c5a2-4088-8757-9722d7845f4d and41bfa826-9142-4725-8137-df5128e9a175;
exact fixture data cleaned. Native producer /tmp/browser-automation-studio/
resume-effects-native-087.mjs. Confirms actual public V2 graph path, not only a
synthetic flat instruction route. Baseline nine paths /tmp/bas-before-087.
Architecture target written before repair: explicit optional checkpoint, resolve
flat order or deterministic graph successor before setup effects; reject a scalar
checkpoint when branch/loop/subflow cursor is required. Do not silently replay or
pretend all recovery is solved. Rich cursor/state, context and lineage remain open.

087 candidate: explicit ResumeAfterStep pointer replaces every StartFromStepIndex
caller; no compatibility branch remains. runPlan resolves an ordered continuation
before tab/entrypoint setup. Graph execution receives that exact starting node;
full plan identities/evidence retained. Missing/ambiguous/cyclic/branched/loop/
subflow scalar recovery fails explicitly before effects, while fresh branching
still runs. This leaves rich cursor recovery open instead of replaying from root.

First green command found one accidentally added argument in an unrelated test
call; fixed before testing. Focused/package11893 consumed0. Extended controls
found duplicate flat checkpoint IDs still admitted; fixed with explicit ambiguity
rejection. Final controls/packages9533 consumed0. Retained failed producer/build
and intermediate behavioral receipts; no source hidden. Eighteen new controls:
six zero/end/fresh, two actual-order, nine invalid/ambiguous, one fresh branching.

Frozen nine paths /tmp/bas-frozen-087.json. Runtime4043->4092(+49), scopedGo777->
799(+22),118->119functions; cumulative affectedGo-49. No per-cycle simplification
claim. resumeStart complexity19 is one bounded deterministic-path qualification
algorithm (format choice, identity uniqueness, edge validity and cursor need),
not a general state engine; splitting it merely to lower a metric was rejected.
runPlan34->37 and remaining>500line modules retain structural debt. Exact tests
and explicit errors are necessary but do not satisfy the overall debt goal.
Next four Go race packages/build, managed restart, exact native repeated-effect
probe, metadata/workflow/profile/capture controls. No dependency/driver/UI change.

087 four Go race packages/build36196 consumed0. TG38730 admission1 exact
20260923-121205-0249b134, one quiet wait and artifact catalog consumed1/0;
fetch/decode still owed. Board/contract/inventory54910 consumed0. Managed restart
admitted after freeze. No087 new capture cohort: fresh execution path is preserved
and focused native controls cover it; retain086 performance scope without claiming
new load certification. Exact repeated-effect native producer is the release
check for this intervention, along with metadata/workflow/profile controls.

087 Test Genie fully consumed:1153findings/103long/414complexity/612duplication/
23coupling/debt34310, budgets fail. The +3250 debt is attributable to one existing
clone group changing structural->high-leverage:65->66locations, reported block
length9->25, with58exact location spans shared. Touched test-call sites belong to
that group. This is not evidence of3250new source lines; it is the owner's current
classification/debt result and is retained without suppression. Attribution report
/tmp/bas-tidiness-duplication-attribution-087.json. Need inspect measurement policy
before deciding whether the threshold/representative-span behavior is a defect.

088 read-only temporary overlay confirms resumed-context loss: actual
ResumeExecution admits a record, then runner reads outside the routed test
context, never invokes executor and leaves CompletedAt absent. One wrong read,
/tmp/bas-resume-context-red-088.txt. Existing recall94483 covers this investigation.
No088 tracked source edits. Candidate087 restart28200 remains pending; all
qualification and native follow-ups must use the frozen087 source.

087 restart28200 consumed0, healthy API/UI/driver and zero sessions before probes.
Build0c05266d1bc33ba6c0b0fe53bdc56f2b32fcfd755f51fb1d559e76fae17e836d.
Native resume28324 consumed0:6/6, exactlyoneeffect (oldtwo), execution IDs
84f658c7-ffaa-42ba-9726-c3475a001734 /51b2995d-e3ed-4dc8-a86f-56c24e95b781,
/tmp/bas-resume-effects-087-aLKrWN. Workflow metadata69079 consumed0:13/13,
3effects, execution6361e0dd-20f2-487c-a30a-ff494c2960ec,
/tmp/bas-recording-e2e-Eo9vem. Profiles79493 consumed0:3full original reads and
APIequals086. Capture34579 consumed0: admission5/5,DPR18/18. Exact fixture
resources cleaned; receipts retained. All nine frozen paths unchanged, no pending
operations. Rich control-flow resume remains explicitly unsupported by a scalar
checkpoint; full recovery and all17 release outcomes remain unqualified.

## BAS-WORK-088 — 2026-09-23 — one execution owner for resumed work

087 fully consumed, feedback unchanged. Reuse recall94483. RF108 actual-service
red /tmp/bas-resume-context-red-088.txt proves the private resumed runner loses
routing and leaves its index pending. Select complete owner replacement: resume
prepares the canonical saved execution request with original settings, recovered
store and parameter overrides; options carry checkpoint/origin. Delete private
resumed starter/runner, preserve canonical context/header/stop/artifact ownership.
Source inspection also proves InitialVariables is written only by that retired
caller and duplicates InitialStore; remove the extra request state layer while
preserving plan-variable defaults and params/env. Missing workflow revision must
not silently resolve current code. Architecture target written; baseline eight
paths /tmp/bas-before-088. No088 runtime edit yet; live remains087.

088 candidate uses saved-workflow admission and its canonical runner for resume;
private resumed starter/runner deleted. Full original settings are cloned before
namespace/URL overrides; public hydration exposes lineage. Missing workflow
revision rejects before admission. InitialVariables removed: its sole retired
caller duplicated InitialStore; one state seed retains plan defaults overridden
by explicit store, with unchanged params/env. NewFromStore previously called
New(seed,nil,nil), confirming identical semantics for absent namespaces.

Maintained original context regression fails on087 and passes on088. Extended
baseline overlay /tmp/bas-resume-preservation-baseline-red-088.txt demonstrates
lost settings/lineage, both routed completion/stop failures and guessed missing
revision. Green /tmp/bas-resume-preservation-green-088.txt. One intermediate test
mistakenly expected reserved resume_url in workflow params; corrected the oracle
and retained failed receipt. The prior package run32091 is consumed: five packages
pass; no process remains. Canonical four cleanup outcomes retained once instead
of duplicated private-runner cases; actual ResumeExecution has independent checks.

Frozen eight paths /tmp/bas-frozen-088.json: runtime3370->3243(-127), Go616->603
(-13),85->83functions; testlines693->804(+111), cumulative affectedGo-62.
SimpleExecutor.Execute37->26 eliminates duplicate state policy. Canonical runner
55->56/admission25->27 retain existing complex ownership; no wrapper extraction
or broad maintainability claim. Both >500line modules still owe simplification.
Five Go race packages/build51456 pending. Test Genie tidiness and board/contract/
inventory admitted after freeze; exact IDs must be recovered before any new run.
Live remains087; no088 native claim yet.

088 five Go race packages/build51456 consumed0; board/contract/inventory58797
consumed0,17rowsunknown. TG89589 admission1, exact20260923-123257-96956a92,
one quiet wait terminalFAIL and artifact fetch/decode consumed. Current findings
1151/103long/414complexity/610duplication/23coupling/debt34310; two fewer clone
findings, unchanged measured debt. Existing budgets fail without suppression.
Managed restart85660 pending. Planned native088 adds original browser-header,
artifact settings and lineage assertions to087 independent repeated-effect check.
No new100capture cohort: retain086 dated performance, no new load claim.

088 native qualification complete on build55d6245c38055cff076175a7901e6aaa56bef97e03a2cd2a14ab5845ad06f630.
Resume55764 consumed0:7/7, exactlyoneeffect, browser header observed independently
on resumed request, artifact settings/parameters/lineage retained. IDs
c52b436c-e21e-4b25-aa26-562ff757d03b/e423697e-e25c-4dfc-92b5-8b61f6736304,
/tmp/bas-resume-effects-088-h0ape2. Metadata/workflow43883 consumed0:13/13,
3effects, executionf4dfae57-364e-49df-9c99-a3f23e0ca791,
/tmp/bas-recording-e2e-bPMkwA. Profiles/health36312 consumed0:3original full reads,
APIequals087, API/UI/driver healthy. Capture49741 consumed0:5/5admission and
18/18DPR. Fixture-owned resources cleaned exactly, evidence retained. Eightfrozen
paths unchanged; no088 operations pending. Durable receipt resume-owner-2026-09-23.json.
Next089 independent store-mutation reproduction admitted only after088 native
qualification; source remainsfrozen088 until its conclusion.


## BAS-WORK-089 — 2026-09-23 — recover actual completed store state

Feedback unchanged,088 fully qualified/consumed. Recall42869 consumed0,
/tmp/bas-checkpoint-state-recall-089.txt; discovery yields existing architecture,
BAS skills and programs, no replacement state owner. Native87624 consumed1 on088:
/tmp/bas-resume-state-native-red-089.txt, /tmp/bas-resume-state-089-dp2fS2. Original
request token=updated; resumed token=original despite COMPLETED and exactlyone
external effect. NewRF109. Existing recovery reconstructs a presentation preview
and ignores set_variable/named-store semantics and disabled extraction. Target
architecture written before runtime changes; explicit durable cursor+store via
existing execution writer, strict readback, no historical guessing. No089 source
edit yet; preserve frozen088 qualification. Crash-window/external-effect atomicity
and rich control flow remain outside the bounded claim, not marked passed.

089 baseline13paths /tmp/bas-before-089 (new private checkpoint owner file absent
before). Extended boundary includes flow_utils.go: maintained expected-value test
proves typed Evaluate.store_result is ignored, RF110. Initial producer used Edges
instead of actual Outgoing and a slash instead of nested dot path; retain those
build/oracle errors, then corrected baseline overlay11340 consumes1 with8genuine
failures:4extract missing checkpoint and4evaluate missing next-step input.
Architecture includes both typed result-key sources. Current checkpoint writes
and reader replacement are in progress, not frozen/deployed. A partial edit
script stopped before executor conversion; retained build failure, then completed
conversion. No live changes since088. Full failure/identity/cursor tests and
qualification remain owed.

089 focused88182 and six-package90147 consumed0. Owner review catches an
important RF110 contract distinction before freeze: driver evaluate returns an
envelope {result: scriptValue}; the declared stored value is scriptValue, while
extract keeps its structured payload. Converted the duplicated flat/graph store
assignment to one typed policy, including scalar/null/nested controls, and removed
actionStoreResult. Baseline extends to14paths including flow_utils_test.go.
Native evaluate fixture prepared alongside the original set-variable fixture.

089 contract94905 consumed0: executor/writer/workflow and AI seams. Native RF110
producer43406 consumes1 before effects: ExecuteWorkflow HTTP500/context deadline,
not the RF110 expected-value assertion. Retained /tmp/bas-resume-evaluate-089-UnF9P2.
Follow-up health reports9active sessions; runtime logs show a separate ongoing RCL
preview capture workload (workflow99f8ccb9-46ff-405c-ab7d-a41f85cf74e4),10-session
resource limit and repeated30s waits. Do not call this a valid behavioral red or
infer our fixtures leaked. Need recover this timed-out fixture's admission and
cleanup receipt if any; its project/workflow were already removed by producer.
Discovering isolated managed validation to avoid contaminating shared workload.
No repeated native admission or restart yet; original set-variable native red and
maintained eight-case baseline remain valid.

089 frozen14paths /tmp/bas-frozen-089.json. Runtime5338->5339(+1), test2382->
2591(+209), scopedGo1023->1029(+6),167->168functions, cumulativeaffectedGo-56.
No cycle-level net debt claim. Private checkpoint owner replaces preview guesses
and progress-only rewriting; flat/graph assignment share one typed owner. Existing
large writer remains complex (appendProtoTimelineEntry44->50 due restored progress
ownership); checkpoint reader19 is one admission/evidence join, not a new engine.
Six race packages/build39078 pending; first five pass, render pending. AI seam
race included afterward. TG57818 admission1 exact20260923-125351-c28e8362;
one quiet wait/artifact retrieval admitted. Board84642 consumed0,17unknown.

Managed validation discovery /tmp/bas-isolated-validation-discovery-089.txt and
control-plane help expose named instances. api-core storage/database explicitly
isolate directories/SQLite/Postgres per variant. Select browser-automation-studio@
rehab089 for candidate qualification while shared live088 serves RCL traffic.
This adds no private lifecycle implementation or dependency. Do not claim live089
deployment from the isolated candidate. Preserve actual live profiles and the
inconclusive RF110 native timeout. Exact timed-out admission cleanup still owed.

089 race/build39078 consumed0: six full race packages, AI seam races and APIbuild.
Test Genie wait/artifact decode fully consumed:1149/103/414/608/23/debt34399,
budgets fail. +89debt attributable to existing clone group+100 (66->68locations)
and removed11-line clone; shifted IDs otherwise cancel. Attribution retained at
/tmp/bas-tidiness-debt-attribution-089.json. No suppression or net debt claim.

Isolated start89069 consumed1: control plane refuses rebuild while another variant
serves shared build outputs. No private launch or overwrite attempted. Read-only
reconciliation across five execution pages reached the failed probe's time window
and found no non-RCL admission. Because workflow cleanup may cascade index rows,
this does not prove no orphan file; no matching artifact/ID authority established
and no broad cleanup performed. The probe recorded no external effects.

Subsequent driver observation reports0sessions/0recordings after the RCL workload
ended. Use existing FB010 authority for managed live restart; no additional approval
or isolated-source workaround. Candidate remains frozen14paths. Native recovery,
evaluate, ordinary workflow/profile/capture checks and comparable100capture cohort
remain owed; new per-step private writes justify the performance recheck.

089 restart26074 consumed0; live API/UI build
3cca76b109ecb96b1a2fdd875ec4a2f1b124543a2642d6f990a37b00f4aa7203,driverhealthy.
Native92880 consumed0: set-variable6/6 (/tmp/bas-resume-state-089-IT8vm5,
73914986-eaf7-4f3c-bb66-2dca10e5908b/51b273e2-2a2e-46a7-97f8-dcc25ab48eec),
evaluate6/6 (/tmp/bas-resume-evaluate-089-EdsXxK,
b118b09d-1089-4b3c-87da-5a71b598d066/25410c4d-ab67-40c2-abfa-5844043e9182),
settings/lineage7/7 (/tmp/bas-resume-effects-088-MSuo3p,
d3069edd-dd8b-483d-8934-c8bb4d28ae9c/50906b2b-098a-4d8a-8c0d-19f98e75035b).
Each fixture has exactlyoneeffect and preserves updated resume inputs; own resources
cleaned. Profile/health87884 consumed0:3original reads/APIequals088. Metadata/
workflow and capture controls73518 pending;100cohort prepared but not admitted.
No source edits after freeze. RF110 old native timeout remains inconclusive;
maintained baseline four evaluate failures provide its valid red evidence.


089 metadata/workflow/capture73518 consumed0:13/13workflow,3effects,
execution6d2662b5-f192-4440-bd89-77877b96a0d0,/tmp/bas-recording-e2e-iSxqpS;
admission5/5,DPR18/18. Cohort35602 consumed0:100/100firstattempts+1warmup,
no retries; servicep50=553/p95=593/p99=625/max633ms,CLIp95=777.933ms.
Versus086servicep95+1.19%,CLI-1.22% observed, not causal qualification.
Conservative servicep95interval[587,633]ms,CLI[754.37,864.13]ms assumingIID;
success exact95%interval[0.9637833,1]. RawSHA
7e12602e725f047fc88eddffb7d2de9d2176ef5de76f61accfb3ae4f7953b387.
Same300PNGs/100trees, fidelity unchanged, no other driver sessions at start.
All14frozenpaths unchanged. Durable checkpoint-state-2026-09-23.json includes101
compact owner references, allfailed/intermediate evidence and scope limits. No
pending089 operations. Original profile data preserved; no broad release claim.

Next090 read-only qualification investigation: current capture workload meets
its <=2s numerical band, but all17 board rows remain hardcoded pending_telemetry
because owner qualification/governed reads have not been implemented. Recall33247
consumed0, /tmp/bas-capture-qualification-recall-090.txt. The existing Performance
phase owns build/Lighthouse/budget gates, deliberately excludes in-gate interaction
capture, and reads flow audit samples. Investigate promoting the proven native
cohort into a maintained owner producer and a provenance-checked governed sensor;
do not turn a file or caller-supplied pass flag into release certification. This
is the contract's existing measurement obligation, not a newly invented defect.
No090 source changes yet.

## BAS-WORK-090 — 2026-09-23 — capture qualification owner investigation

089 fully consumed; feedback unchanged. Test Genie skill read and root testing/
tunable-levers protocol consulted. Recall33247 and /tmp/bas-drill-qualification-
discovery-090.txt consumed0. Existing BAS drills are failure-recovery only; do not
repurpose them. Current setpoint-read.py emits17literal pending rows. PH TrendSample
explicitly reserves old p95_ms because no honest producer existed; do not overload
LCP/build fields or unreserve that tag. Performance ExecutionOrchestrator owns
benchmark/Lighthouse producers, a fresh sample and budget gating; interaction
capture is deliberately outside the gate. No declared custom workload hook found
in inspected owner configuration. These are inspected boundaries, not a claim that
no ecosystem capability can exist.

Architecture target written: maintained BAS fixed-fixture producer, a bounded
Performance Health declared-workload runner/receipt, Test Genie performance
admission, and governed current-evidence read. Preserve band and denominator,
reject incomplete/stale/malformed evidence; do not certify historical files or
accept pass flags. Necessary measurement-owner extension is authorized under this
goal: potential paths scenarios/performance-health/api/** and docs/**, its proto
schema/generated projections, BAS producer/config/program files, and Test Genie
orchestrator only if existing descriptor plumbing actually needs a repair.
No090 runtime edit or external plan/log created. First inspect the PH module,
repository/descriptor and proto-generation owners before selecting exact changed
paths and retaining a baseline. Prefer existing execution orchestration and one
workload owner; avoid a new BAS job system or special-cased Test Genie path.

Current live089 remains qualified; no pending operations. The next action is
measurement-owner design/implementation, with fail-closed producer-contract tests,
not another unchanged100capture repetition. All17release readings remain unknown.

090 implementation boundary refined: first promote the native capture fixture
into a maintained BAS producer with an adversarially tested oracle. Selected
api/internal/capturequalification/{fixture,oracle,run,run_test}.go and
api/cmd/capture-cohort/main.go; baseline /tmp/bas-before-090/manifest.json.
Use Go standard-library PNG decoding to verify actual pixels as well as PNG
geometry, computed tree and independent fixture observations; no new dependency
or private browser is needed. A per-attempt paint sentinel rejects stale images.
The producer keeps all 100 first attempts plus one warmup, raw responses and
artifact hashes, stable fixture/source/contract identity, before/after deployed
build identity and failures. It produces observations, never a release pass flag.
PH integration remains the next owner boundary; browser capture stays out of the
synchronous performance gate, whose job is checking fresh retained evidence.
No PH schema or API edit yet. Existing17 board readings remain unknown.

090 maintained producer implemented (five new files; 556 runtime lines plus
controlled tests). Initial package tests42031 consumed0, 6.988s; receipt
/tmp/bas-capture-qualifier-tests-090.txt. Cases reject blank/stale/corrupt/scaled
PNGs, missing or mismatched artifacts, invalid snapshots, wrong observations,
duplicate IDs, cancellation denominator loss and changing provenance. Native
browser proof is still pending. Race run28591 is active; wait for its terminal
output before another qualification. /tmp/bas-frozen-producer-090.json freezes
five producer paths; existing BAS product runtime remains089, untouched.
Added bounded subprocess output and tested io.Copy fast paths, avoiding embedded
bytes.Buffer.ReadFrom bypass. No new dependencies, private browser, retry flag,
pass-input or reduced sample-count switch. Documentation adds the maintained
entry point and explicitly keeps17readiness outcomes unknown. No cycle net debt
reduction claimed: this adds the missing measurement capability.

PH design narrowed after inspecting its modules and decisions: extend the
existing out-of-band SweepService with declared-workload run/read operations,
backed by one workload measurement domain; reuse the existing validation handler
for the cheap retained-evidence gate. Do not add a new job service or execute
browser capture inside Test Genie. Preserve reserved historical p95 field tags.
Exact PH source/proto/CLI paths and baseline still need capture before editing.

Producer race28591 consumed0 (192.148s), all controlled cases passed. Driver
preflight0sessions/0recordings and API still089. New maintained native100cohort
admitted (capture command session ID follows in checkpoint), output
/tmp/bas-maintained-cohort-090 and /tmp/bas-maintained-cohort-090.txt. This validates
the new pixel/sentinel verifier against unchanged production089; it does not
requalify a deployed090product or a release row. PH target documentation now
specifies extending SweepService, current-evidence gating, and no historical
fallback after a newer failed run. No PH runtime source edited yet.

PH extension baseline retained at /tmp/bas-owner-before-090/manifest.json
(including generated PH projections). Selected workload domain config/receipt/
service/store/schema/tests, existing sweep handler/module plus workload methods,
existing validation handler/module/provider tests, API composition root, existing
sweep CLI handlers/register/manifest/endpoints and sweep.proto. No new RPC service
or Test Genie special case. Owner receipt tests must prove incomplete/malformed/
failed/stale/mismatched data never passes, last failure is not replaced by an older
success, and p95 uses all100firstattempts. CLI/schema parity and owner package
tests precede live admission. Native maintained producer run78807 remains pending.

Native maintained78807 consumed0:100/100firstattempts+1warmup, pixel/sentinel/
snapshot observations allverified. Servicep95=550ms,CLIp95=737.785015ms, on089.
Raw SHA492af33b3c8fd0130c66118cd7db3e1f545f723282d24892a83746410e361bc3;
/tmp/bas-maintained-cohort-stats-090.json. Changed fixture sentinel means this is
not a causal performance comparison with089's prior workload. PH sweep.proto
now adds bounded Workload run/read types and corrects outdated baseline-diff
comments; generation1213 consumed0. No shared proto projections changed.

Workload command timeout needs process-tree cancellation (go run has a child);
do not copy a private OS-specific implementation. Existing platform-go supplies
ConfigureCommand/AssignProcessContainment/SignalProcessGroup. Discover query
/tmp/bas-workload-containment-discovery-090.txt completed, no suitable command
program among returned results. PH has the local replacement but no direct
requirement. Extend scope to its api/go.mod/go.sum through SDA only; captured
baseline before invoking install. Native platform process-control proof remains
limited to actual tested hosts, no Windows/macOS claim.

090 maintained producer proof durably summarized in evidence/rehabilitation/
capture-qualifier-2026-09-23.json; five frozen paths unchanged. All original user
profiles untouched (only synthetic fixture captures added). 100samples retain
300PNGs/100trees, so RF016duplicate capture remains. Native78807, generation1213
and approvedSDA64818 all consumed0. SDA adds only platform-go v0.0.0 via existing
local replacement; governance/security checks still pending. PH workload config,
receipt evaluator/retention and append-only store/schema now exist; initial
package compile admitted, exactexecID follows. Service, command tree cancellation,
owner tests, sweep handler/CLI wiring, validation gate, BAS declaration and
governed binding are unfinished. No managed runtime restart for090 yet.

090 PH service and command implementation now compile (initial and service
compile exited0; no tests yet, no live PH admission). Added internal/workload/
command.go within the recorded owner extension. Bounded command invocation uses
platform-go containment, graceful cancellation plus owned-tree force cleanup,
1MiB stdout/stderr limits and a declared deadline. Owner latest-reading checks
configuration/producer/contract/build identity, expiry, raw receipt and retained
artifacts, and rederives p95 over the full declared denominator. Dedicated table
(schema next to its store) is new, with no runtime migration. These are unverified
implementations pending adversarial tests and integration; do not report as a
working gate. No pending tool operations.

PH workload test54895 consumed1: arithmetic, denominator, provenance, retention
and no-fallback controls passed; immediate child-liveness assertion failed just
after forced group termination. Exact PID3143982 was absent on subsequent read.
Cause is asynchronous signal delivery; replace instantaneous process-state
assertion with an independent bounded1s exit observation (10msinterval), preserving
the invariant that the stubborn owned child dies. No production assertion was
weakened to allow a surviving process, and no source workaround was added. Full
owner race validation is now pending. Include readiness.proto field10 in the
owner boundary (baseline captured) so Test Genie's native_detail retains actual
workload readings/receipt references instead of a verdict without evidence.

PH owner race6221 and proto generation85415 consumed0; initial owner integration
5055 and40464 tests exposed test-setup errors around optional maturity metadata,
not native capture regressions. Shared validation requires a valid assessment;
without it native status isERROR. Correct controls assert neverPASSED without
metadata, then use a valid maturity fixture for exact shared gate statuses and
native-detail receipt preservation. Build passed; package retest pending.

BAS testing/program files baseline captured. Necessary config-schema extension:
scenarios/test-genie/schemas/testing.schema.json documents the new owner-owned
performance.workloads declaration (the existing schema already allows unknown
performance properties). No Test Genie execution special case or new orchestrator
policy is needed. Capture declaration keeps100+1,2000ms and fixed viewport/DPR;
owner gate uses retained evidence. Program binding will read only that owner.

PH owner tests3/4 and full CLI tests passed; governance validation passed with
two existing advisories. Schema generation initially failed because existing
DescribeProvider/ValidateTarget RPCs lacked CLI bindings or omission declarations.
Declare both as shared Test Genie surfaces in the existing CLI omitted list,
consistent with the native readiness CLI policy; do not weaken parity checks.
Native shared gate controls now assert PASSED only for applicable measured data,
FAILED for breach/producer failure, DEGRADED for absent/stale/read-error evidence,
and retain workload readings in native_detail. Missing maturity metadata remains
ERROR/FAILED, neverPASSED. Workload store ordering now uses integer nanoseconds
instead of lexicographic RFC3339Nano, avoiding fractional-prefix ordering errors.
BAS declaration and governed board consumer authored, but not live-qualified yet.
All17current release outcomes remain unknown until the owner invocation is run.

090 candidate frozen103paths at /tmp/bas-frozen-owner-090.json; unchanged check
passed. Owner races77990 consumed0, finalbuild70151 consumed0, full CLI tests and
configuration JSON-schema check pass. Endpoints generation succeeds after explicit
shared RPC omissions. Contract preparation remainsvalid, no product verdict.
SDA governance passed with pre-existing UI postcss/vite range advisories; security
indexstatus available, not a clean vulnerability scan. No new third-party version
was added: platform-go is the existing approved local module.

Managed PH restart82275 consumed0; PH now serves
e6c2f84c14664ddf62d10f63936c3f0a73117dbdfe5ff25e8e8e5f1ced302f5a.
BAS core remains the qualified089build3cca76b1; its production capture behavior
was not edited. Qualification candidate is that deployedBAS plus frozen090
producer and measurement owner. Baseline workload-get correctly returns
UNAVAILABLE/no owner receipt. Governed board run98854 is pending; first authoritative
PH workload-run now admitted (exactexecID below). Do not re-admit either run.

Scope metrics /tmp/bas-owner-metrics-090.json: authored affected runtime1178->2683
(+1505),Go155->517(+362),functions40->95(+55), generated projections excluded.
This is the cost of adding the missing qualification capability, not a net debt
reduction. Cumulative affectedGo across prior cycles would be+306 after this
extension; preserve that regression explicitly. Complete owner qualification,
then review repeated protocol/validation ownership for substantive simplification.
Do not relabel moved or generated code as a debt improvement.

Pending owner invocation090 is exec86671 (/tmp/bas-ph-workload-owned-090.json);
this is the sole admitted authoritative cohort. No duplicate admission.

Governed board98854 consumed0:17unavailable; capture explicitly says no owner
workload receipt. This proves the new binding resolves and does not fabricate a
pass before evidence. PH workload86671 remains pending.

Authoritative PH invocation86671 consumed0, owner operation
1203c9afc9bad95b1c9d52f2e02185b3:100/100firstattempts+1warmup verified;
servicep95=607ms,CLIp95=811.459737ms, both<=2000ms. Receipt at
/home/matthalloran8/.vrooli/test-runs/performance-health/workloads/
1203c9afc9bad95b1c9d52f2e02185b3/producer/receipt.json,
SHAdd6b06e316687cb511d8b22ae0fc32f58a54e370f2695e44e81d648fe6c1ac49.
Before/afterBAS089identity and producer/config/contract identity match. Host
inventory retained by its control-plane owner in invocation.json; declared
reference floor4cores/16GiB satisfied. No caller pass flag or imported historical
receipt used. Fresh Test Genie performance+tidiness run and governed board read
now admitted; recover their exactIDs before any re-admission.

090 Test Genie exactID20260923-141347-f1225438 (performance,tidiness), admission
exec57227; one quiet wait nowattached, /tmp/bas-tg-wait-090.json. Do not poll or
re-admit. Governed board10820 consumed0: capture is nowreadable/inband,100samples,
607ms/811.459737ms and owner1203c9af... withBAS089build. Remaining16unknown and
product_qualified=false. This is an owner-backed measured reading, not a full
performance-phase or release verdict; that Test Genie run remains pending.
Direct workload-get verifies the retained artifacts and matches the same reading.

### BAS-WORK-090 qualification closure — 2026-09-23 UTC

Consumed the sole Test Genie wait31900: overallFAIL, performancepassed and
 tidinessfailed. Owner artifacts retained in /tmp/bas-tg-artifacts-090.json;
 findings /tmp/bas-tg-090-findings.json. Decoded performance native_detail carries
 the exact PH workload identity,100samples+1warmup,607ms service/811.459737ms wall
 p95 and owned receipt SHA; no generic performance pass substituted for capture.
 Tidiness has1155findings,103long,420complexity,608duplication,23coupling and34399
 duplication-line debt. Compared089: +6complexity findings; other listed counts
 unchanged. Budget failure names103long versus81recordedbaseline. All103frozen
 owner/producer paths hash-identical at final verification. No pending operations.

The090 capability is qualified for the localcapture row only. Sixteen other
 readings remain unknown, fullproduct readinessfalse, and addedmeasurement code
 remains explicit debt (+1505authoredruntime lines/+362Go complexity). RF016
 still causes300PNGs with100uniquePNGhashes for100captures; nextinvestigation
 follows the existing capture service and telemetry owner, preserving explicit
 screenshot actions and failure evidence. No091production changes yet.

### BAS-WORK-091 — 2026-09-23 UTC — requested capture frames (investigation)

Feedback reread; RF016 / capture / J08,J20. Recall091 found prior screenshot
 policy work and existing capture-surface program, with provider degradation;
 saved /tmp/bas-capture-duplication-recall-091.txt. Reuse policy owners, no new
 workflow framework. W3 implementation defect: existing contract already requires
 selected artifact policy and evidence preservation. Target documented first in
 ARCHITECTURE.md. Baseline six files in /tmp/bas-before-091/manifest.json.

H1: CaptureService defaults to passive ALWAYS images; three steps cause three
 distinct productions. H2: exporter duplicates one original image. Owned090
 artifact names identify navigate, wait and evaluate as separate source steps;
 builder has no explicit ordinary screenshot or artifact profile, and executor
 defaults ALWAYS. H1 supported, H2 cannot explain distinct step artifacts. A
 maintained request-boundary regression plus actual screenshot policy checks
 must fail before repair. Afterward the full unchanged100capture fixture must
 preserve pixels/tree/independent oracle while producing100PNGs instead of300.

Repair boundary: BAS capture builder, existing config profile owner, executor
 telemetry policy and their regressions. Add a capture profile retaining failure
 diagnostics and explicit screenshots. Keep validation navigation/assert frames
 through an explicit setting, not a global ON_FAILURE promotion. No new proto,
 driver implementation, dependency, service or suppression. Native selector,
 requested non-image artifacts, interaction screenshot and failed-step evidence
 must also be checked. Metrics include all affected authored source; no source
 shrink claimed in advance.090source stays frozen;091needs a new build/cohort.

091controlled qualification: red regression82561 failed on missing final image,
 empty artifact profile and unconditional navigation image; log
 /tmp/bas-capture-policy-red-091.txt. Fixed three runtime owners. Existing tests
 now expect explicit final screenshot nodes; inline DOM helper verifies evaluate
 semantics instead of an obsolete total node count. No assertion contract weakened.
 Follow-up34835 exposed those stale test assumptions, then fivepackage race33256
 passed (capture/config/executor/execution-writer/workflow). All original policies
 have their existing controls; new capture policy preserves ON_FAILURE diagnostics.
 Explicit final image pins FullPage=false to retain viewport fidelity.

Affected runtime3046->3074lines (+28),Go554->557 (+3),functions84unchanged;
 /tmp/bas-capture-policy-metrics-091.json. No net debt improvement. Named capture
 policy adds no protocol/service/driver implementation; validation checkpoint
 promotion is now explicit in that profile. Sixteen-line profile table repeats
 full evidence toggles and is a remaining config simplification opportunity.
 Frozen110paths in /tmp/bas-frozen-owner-091.json (before live build/qualification).

091cohesion review: buildAdhocRequest complexity14->16 exceeds15; it still owns
 one translation from CaptureRequest to an ordered workflow and preserves existing
 readiness/interaction/browser-profile semantics. Added predicates express whether
 the final image was requested; extracting that boolean would only relocate debt.
 Module service.go remains a known large owner, no cohesion waiver or cleanupclaim.
 Executor policy5->6 remains the sole screenshot directive owner. No new cycles;
 capture gains config and standard-library slices imports. Managed restart53874
 active; build10066 consumed0. Native probe prepared at
 /tmp/browser-automation-studio/capture-policy-native-091.mjs, not yet run.

091managed restart53874 consumed0. API/UI healthy on
 sha256:2457de8b3078a237d146cec0e6bb7fa7ea7129849735ffea985ab05b52dea9f0;
 driverhealthy. All110frozenpaths unchanged. Profilepreservation28275 consumed0.
 Correction to earlier shorthand: rollback contains ONE profile checked THREE
 times, not three originalprofiles. All complete protected state compares equal;
 /tmp/bas-profile-preservation-091.txt. No saved profile modified.

Nativeprobe33742:4pass/2invalidtestexpectations. Network fixture initially marked
 ready before its fetch completed; failureoracle expected one image despite
 existing continue-on-error admitting the requested final image after the failed
 evaluate. Originalreceipt retained /tmp/bas-capture-policy-native-091/receipt.json.
 Correctedindependent oracle waits for fetchcompletion and verifies failedframe
 screenshot plusfinalexplicitimage; no productionchange or request weakening.
 Qualified87901:6/6 (viewport640x480 on1400px-tallpage,element200x120,DOM-only0PNG,
 requestedconsole/networksentinels,interaction2distinctimages,failedcapture error
 with failed-step+finalimages). /tmp/bas-capture-policy-native-qualified-091/receipt.json.
 Owner100capturecohort now admitted; retain operation/sessionbefore furtherwork.

091live cohort and board:100/100verified on unchangedproducer/fixture/contract.
 Owner10467terminal0, operationdb72786edda2dc7cbd0281b470eeab97,rawreceiptSHA
 e15de1ba96cbe3f334300c93aba70939456f793a6197eba717f725207c518219.
 Pendingboard36364showed17unknown (correctly noolderpassfallback); terminalboard
81890showed1inband/16unknown. PNGbytes4510698->1528027 (-66.12%);
 stepPNGartifacts300->100,uniquehashes100both,computedtrees100both.
 Primary screenshot.png alias remains a copy; allphysicalcopies not claimedgone.
 NativeDPR87418consumed0,18/18. TestGenie091admission80652consumed0,run
20260923-144032-15388892; solequietwait78010active. No further qualificationrun.

### BAS-WORK-092 — 2026-09-23 UTC — interaction graph composition (read-only)

Recall83374consumed0: /tmp/bas-capture-splice-recall-092.txt,64hits withprovider
 degradation; savedoutput includes malformedUTF8, readwithreplacement for diagnosis.
 Related captureprogram/prior policy work found; no established boundary repair.
 W3 RF111 / preservation,J08,J17,J20. H1: capture's first/lastarraynodes disregard
 explicit graph topology. H2: compiler/driver ignores explicit edges independently
 of wrapper composition. Orderednativecontrol and reversedarray share first->last
 edges; publiccaptureorderedpasses, reversedfails before either expected effect.
 Source shows navigate->firstdeclared and postlude<-lastdeclared; confirmedH1.
 Native37341consumed1, /tmp/bas-capture-splice-native-092/receipt.json; no product
 sourcechange. Compiler control and maintained tests will distinguish H2 further.

Architecture target recorded. Intendedrepair uses existing compiler planner's
 loop-body extraction/topology to expose outerboundaries; capture keeps existing
 nodes/edges and attaches prelude/postlude at actualboundaries. No handler-private
 graphsort, syntheticloop wrapper, full duplicatecompile, newdependency or runtime
 migration. Test reversedlinear, branchterminals, loop-body exclusions, single
 node, malformed/disconnectedgraphs and callerimmutability. Preserve091freeze
 until TestGenie run20260923-144032-15388892 finishes; solewait78010stillactive.

091terminalclosure: solewait78010consumed1; TestGenieoverallFAIL. Performance
 retainscurrentPHoperationdb72786edda2dc7cbd0281b470eeab97,428/622.8881ms,
100samples+1warmup. Tidinessunchanged1155/103/420/608/23/debt34399. Full API
 coveragecommandPASSED47.284s,CLI PASSED2.485s,drivercoveragePASSED397.772s;
 UIcoveragecommandFAILED57.022s at85%floor (30.42statements/33.57functions/
67.16branches/30.42lines),UItypecheckPASSED10.271s. Unitnativeuh-20260923-144033-
b5a82a2a9010d611e551e5107cbdc796 also reports TabBar direct-renderprojection drift,
79lowcoveragefindings,2injectableseamfindings andanexisting skip. These are real
 opencoverage/policy obligations, not an unavailable current Go runner. Prior042
 CodeFacts/no-output limitations no longer describe091's successfulAPI execution.
 Fullnative /tmp/bas-tg-091-unit-native.json; all110frozenpaths unchanged atclosure.

092implementation: baseline4files /tmp/bas-before-092/manifest.json. Maintained
 boundaryregression35962fails all three compositioncases and firstinvalidadmission;
 /tmp/bas-capture-splice-red-092.txt. Added compiler WorkflowBoundaries using
 existingplanner/loopbodyextraction/topologicalorder, with malformed/disconnected
 graph rejection. Shared typedconversion preservesexistingcompile semantics;
 captureconnectsactualentryandallterminals through one local append-node closure.
 Removed four duplicated edge-append paths and first/last declaration assumptions.
 No second graphsort/runner, savedtemporaryworkflow or dependency. Existing tests
 pass44621; fourpackage race83888 passes. Newcompiler tests cover malformedgraph,
 pureboundaryquery without URL/selector resolution, inputimmutability and loop
 outerterminal; handler tests cover reversedlinear/branches/loop plus pre-effect
 rejection. OrdinaryBuildAdhoc semantics preserved through existingtests.

Metrics2219->2253runtime lines (+34),functions62->64,Go499->517 (+18), cumulative
 scopedGo+327. No net debt reduction. Newboundaryquerycomplexity15; builder16->17
 due terminaljoin loop, compilerentry22->20 from sharedconversion. Cohesion is
 graphownership and one requesttranslation; functionshuffling would not reduce
 algorithmicdebt. Capture gains compiler dependency with no cycle. Frozen112paths
 /tmp/bas-frozen-owner-092.json. APIbuild28865passed; earlier9219 accidentally ran
 atrepo root andfailed on out-of-module probe imports, not a BASbuildfailure;
 no dependency changes made. Native091stilldeployed,092awaitsmanagedrestart.

Additionalnative092branchmatrix58665: typedconditional bothfailunknowninstruction
 (RF112); loop executesbodytwice+lastbutreturns2images and a stale pre-last
 snapshot (RF111incorrectsuccess). Originalproof retained. Assert success/failure
 branches are being used as additional nativegraph controls, not substitutes for
 RF112's unresolvedtypedconditionalcases. No092ownercohort/TGadmission yet.

092 additionalnative59171 (091deployed): successassertbranchpasses; negatedvisible
 assertion incorrectlytakes successedge (RF113), loopstillreturnspre-terminal
 snapshot. Originalreceiptretained. Nativegraphcontrols now also exercise a real
 missingselectorfailure path with explicitcontinue-on-error; that is additional
 coverage, not replacing RF113negation or RF112conditional failures. Both are
 recordedactualdefects for nexttypedaction repair. No extra092productionchanges.

092boundary extension within currentcompiler: nativeassert-path78333 exposes
 missing failurebranchselection. Handle-only probes were insufficient; corrected
 labelededges16333stilltake successpath. RF114confirmed: edgeCondition onlyreads
 obsoleteData.condition although input is typedV2 withlabel. Existingcompiler
 edge-condition test is skipped. Activate it, replace oldreader withtypedLabel,
 remove internal handle spelling fallback (all rawflows derive from one
 UseProtoNames marshal), and preserve externalprotojson decoding. Necessary for
 honestnativebranchqualification; no newowner/dependency. Earlier112/113 remain
 separateconfirmedbugs; negatedasserttimeline explicitlyrecords successfulassert.
 This changes the frozen092candidate beforedeployment; supersede itsmanifest and
 re-run affectedchecks. No092ownercohort orTestGenieadmission exists.

092RF114 red: previouslyskipped edgeconditiontest now runs6cases; fiveexplicit
 labels fail beforefix (success/failure/true/false/trimmederror), emptylabelcontrol
 passes. /tmp/bas-edge-condition-red-092.txt. Compiler nowreadsLabeldirectly;
 deletedobsoleteData.condition field/reader and privatecamelhandlefallback/getters.
 Normalizedproto input remainsauthoritative. Finalfourpackage races93063pass;
 /tmp/bas-capture-splice-race-final2-092.txt. Earlierfinaltest invocationusedwrong
 cwd andfailedsetup only; preserved /tmp/bas-capture-splice-race-final-092.txt,
 no productfailure inferred. APIbuild8086pass. Refrozen112paths in
 /tmp/bas-frozen-owner-092.json; priorprefreeze retained-pre-label.json.

Final092delta (supersedes pre-labelcomparison): runtime2219->2223 (+4),functions
62->61 (-1),Go499->509 (+10), cumulativescopedGo+319. Nativebranches withdeclared
 labels must be retried onnewbuild; no unsupportedconditional/negatedassertpass
 claimed. Currentoperation managedrestart092; nativequalifiedprobe variants retain
 before/afterhealthidentities. No092cohort/TGadmission yet.

092native qualification: managedrestart1987terminal0,healthyAPI/UI
5c5bb1fcf2135dd2fadd7a23770b6cf28a521cfc24d7f51d722be424d76bb5a4,driverhealthy.
112frozenpathsunchanged. Native13311ordered/reversed2/2;77639labeledbranches+loop
3/3. Independentfixture effects matchexactorder/branch; requestedfinalsnapshots
 matchpostinteractionstate. Loop2pre-terminalimages->1finalimage; failedassertion
 branchretainsdiagnostic+final2imagesandfollowsrightedge. RF112/113remainopen and
 are not relabeledpassing. Owner100capturecohortnowadmitted; noTG092runyet.

093read-onlyorientation while092qualificationruns: recall60134finished,67hits
 withproviderdegradation; retained /tmp/bas-typed-condition-recall-093.txt.
 Docs/nodes/assert.md is staleV1 prose; authoritative typedparams/ConditionalNode
 definepageJSexpressions,elementvisibility andworkflowvariable conditions. Driver
 getAssertParams already retainsnegated/caseSensitive buthandler ignoresboth;
 failureMessage is absentfromextractor. Handler result/outcome conditionbridge
 stillneedsinspection beforechoosing112ownerrepair. No093sourcechange.

092ownercohort71505terminal0: operationdfccaeaf88c701f9482f38d55894a4a8,rawSHA
c4555fe67282adf0bdc9df744efe52eafcb2044389795225af958971c9a70c7b;
100/100verified+1warmup,414msservice/602.637287mswallp95,100PNGs1603694bytes,
100computedtrees. Sameproducer/fixture/contract, no repeatedlatencyclaim. Board
50572confirmscaptureknown/inbandand16unknown. Profile87208checksoneoriginal
 profilethree times,allsemanticallyequal.112frozenpathsunchanged. TestGenie
20260923-151101-55ba59d3admitted66770; solequietwait23187active. Selected
 performance+tidinessplusdirectaffectedGo races/nativecaptures; repeatofunchanged
 driver/UIcoverage wouldnotaddproof.091fullAPI/CLI/driver andUItypecheck remain
 datedreceipts; UIcoverage/policy andstructuraldebt remain realopenobligations.

092terminalclosure: TestGenie20260923-151101-55ba59d3 overallFAIL; performance
 passeswithexactPHoperationdfccaeaf88c701f9482f38d55894a4a8 and414/602.637287ms,
100samples. Tidinessremainsfailed; summary {"totalFindings": 1157, "longFiles": 104, "complexity": 421, "duplication": 608, "coupling": 23, "duplicationLineDebt": 34644}.
 Solewait23187consumed1, artifacts/native_detailretained /tmp/bas-tg-092-*.
112frozenpathsunchangedatclosure; no pendingops. Allprogressis scopedqualification,
 notoverallproductionreadiness. NextRF112/113requireswholetypedcondition/assertion
 owner, meaningfulbrowser/variable/errorcases and simplification; nofakebranch
 results, threshold changes or deletionoffailedhistoricalprobes.

### BAS-WORK-093 — 2026-09-23 UTC — typed assertion truth and failures (in progress)

RF113/J24, within existing driver assertion/typed params/outcome owners. Recall
60134 retained in /tmp/bas-typed-condition-recall-093.txt; no repeat discovery.
Baseline /tmp/bas-before-093/manifest.json retains runtime/test/doc bytes.
Hypothesis: missing negation/case/message projection and broad exception catches
allow incorrect branches and can turn evaluator failure into apparent truth.
Discriminating checks: all eight modes with positive/negative truth, timed state
transitions, invalid selector/closed-page failures, absent vs empty attributes,
case handling and custom messages through public capture's typed timeline.
Target documented in ARCHITECTURE.md before implementation. Consolidate existing
handler, remove unreachable legacy aliases/regex paths, preserve frame owner.
RF112 conditional support remains next; no new dependency or private browser.
092 owner runs are terminal. Its test-file length/complexity/duplication increase
remains a failure; prioritize actual shared predicate simplification over moving
tests solely to clear budgets. No093 implementation or validation receipt yet.

093 local results: maintained assertion matrix red2 fails54/75 before repair,
then green2 passes93/93 including immediate zero-timeout observations, browser
errors, invalid typed modes and actual typed versions of old text/visible tests.
Those six old helper calls encoded string modes as Number(NaN) and silently
exercised exists; replaced them with canonical enum inputs without weakening
expectations. Initial red command97185 used a wrong source-relative patch path
and captured a nested-test setup error; retained but not product evidence.
Typecheck95071 passes after fixing InvalidInstructionError's details argument.
Scoped installed ESLint complexity58->44 (-14), functions10->4 (-6), max27->22.
Runtime assertion396->125 plus params+2lines: net-269. No domain-wideTS metric
claim; RF064 remains open and cumulativeGo unchanged+319. New executeCC22 is
cohesive request validation/dispatch/evidence/error assembly; splitting wrappers
would not lower aggregate complexity. Removed four duplicate state implementations
and unreachable wire modes; no thresholds changed. Metrics/tmp/bas-assertion-metrics-093.json.
Frozen115paths /tmp/bas-frozen-owner-093.json. Managedrestart3717 still pending;
native fixture /tmp/browser-automation-studio/assertion-native-093.mjs prepared,
not yet run. No TestGenie093 admitted until deployed candidate verified.

093 deployed/native results: restart3717 consumed0; native81086 consumed0,
/tmp/bas-assertion-native-093/receipt.json,58/58. All eight modes xtruth xnegation,
missing visible/hidden targets, case sensitivity, absent/empty attributes, invalid
selectors/missing text evaluator errors, immediate zero timeout and delayed
attach/detach/show/hide verified. Independent fixture effects prove only the
expected success/failure branch; timeline preserves negation, case flag and
custom failure text. No baseline expectation relaxed. Protectedprofile60996
compares complete original state equal three times.115frozenpaths unchanged.
BAS build553bc332c8acced00a3ff95cdcf1035047b910e8ed918b05fa350a70c9e9373d.
PH capture owner run now pending; TestGenie awaits its retained result.

093 PH owner41914 consumed0,operation93ba0c644d6f91e5cbe53d47de3a35af.
100/100firstattempts+1warmup;447msservice/650.217134mswall p95 under2000msbudget.
Raw receiptSHA1942a1727aff24aac784287d34f55b114cb47ad4ee2e956ba27e50ed5c5d8df5;
producer/config/contract/fixture identities unchanged. Single093wallp95 is7.9%
higher than092's602.6ms; these isolated shared-host cohorts do not distinguish
noise from a5%relative regression. No latency-improvement claim; recheck with
repeated comparable quiescent trials when performance attribution is selected.
TestGenie performance/tidiness admitted once; board read pending.

093 terminal qualification: TestGenie20260923-153329-f6718ff3 overallFAIL,
performancepassed with exact PHoperation93ba0c644d6f91e5cbe53d47de3a35af,
tidinessfailed1157/104long/421complexity/608dup/23coupling/debt34644.
Admission29795consumed0; solewait74692consumed1. Native details retained
/tmp/bas-tg-093-{performance,tidiness}-native.json and artifactcatalog.
Unchanged tidiness totals do not measure the driver reduction: RF064 excludes
driver JS duplication/TS AST complexity. The separate installed ESLint metric
is a narrow owner comparison.115frozenpaths unchanged through terminal.
Board63711consumed0: capture1qualified/16unknown/productfalse. No pendingops.
Continue RF112 typed conditional ownership and error/branch evidence;093 is a
dated repair, not a readiness or goal-completion claim.

### BAS-WORK-094 — 2026-09-23 UTC — typed conditional execution (in progress)

RF112/J24.093 terminal consumed before this candidate starts. Read-only tracing
confirms unsupported driver dispatch, absent HandlerResult condition projection,
and normal-first-edge fallback even when a conditional has no successful result.
Typed ConditionOutcome already exists; reuse it rather than an extracted-data
adapter. Variables belong to Go's store; page JS and element presence to the
selected browser frame. Target094 in ARCHITECTURE.md precedes implementation.
Expected false remains step success; evaluator failure has no truth result and
never executes the normal first branch. Parse expression/body before execution
to avoid repeating side effects after a runtime throw. Reuse loop comparison and
ordinary event/outcome/checkpoint owners for local variable predicates.
No new dependency, shared-owner extension or external effect. Existing local
recall60134 and native092 failures supply the baseline. No094 source edits yet.

094 implementation checkpoint (not deployed): baseline18paths retained in
/tmp/bas-before-094/manifest.json. Driver now dispatches ConditionalHandler,
builds the existing typed ConditionOutcome, and Go decodes its JsonValue fields
through protojson. Browser expression/body compilation precedes execution;
runtime SyntaxError is not a syntax fallback. Element wait honors timeout/poll;
only genuine timeout becomes absent. Variables use the actual Go store and
shared CompareValues (including declared string operators and numeric strings).
Both graph/linear local-state callers share executeWorkflowStateAction, replacing
applySetVariable plus duplicate detection helpers. Ordinary terminal outcome and
checkpoint behavior preserved. Initial extra StepStarted event broke existing
synthetic-outcome cancellation tests; removed the unnecessary event, not tests.
Conditional routing no longer falls back to first/opposite branch after errors or
an unwired truth. Current builder labelsIF TRUE/IF FALSE are supported alongside
existing truth labels in the same V2label policy; no alternate edge-data reader.
UI element label now saysPresence, matching the canonical predicate contract.

Discriminating reds: /tmp/bas-conditional-routing-red-094.txt has3behavioral
failures; /tmp/bas-conditional-wire-red-094.txt fails typed scalar/object recovery
and malformed-value rejection; /tmp/bas-conditional-outcome-red-094.txt fails2
condition cases with2existingcontrols passing. Boundary75923 thenpassed.
Driver91887 passes121tests across condition/assertion/outcome; typecheck23989
passes. Go initial16460 failed2event-count expectations; after removing added
start events,6package race44261passes, then finalrace5850passes after UIlabel
routing and finite-numeric validation. APIbuild45083 pending.
Source updates after093 invalidate its applicability to this candidate; managed
live build remains093 until restart. No094 freeze/native/PH/TestGenie admission
yet. Next: native CSP/frame/variable/error-after-effect matrix, measure cost,
freeze final source, managed restart, profile preservation and owner qualification.

094 pre-native freeze131paths /tmp/bas-frozen-owner-094.json. APIbuild45083
consumed0. Managedrestart48237 pending; native fixture prepared at
/tmp/browser-automation-studio/conditional-native-094.mjs, not yet run.
Native matrix includes selected frame, CSP, body/expression/Promise syntax,
shared-store comparisons, Evaluate.store_result, bothnegations, delayedpresence,
expliciterrorbranch and SyntaxError-after-effect withmax_attempts3.
Current scoped runtime7668->7815(+147), GoCC1029->1052(+23), Go functions186
unchanged; maximum35->34. TS ESLint measurement returned parser errors and is
being corrected before interpreting it. No netdebt reduction claimed for adding
previously missing conditional capability. Existing originals and093 receipts
remain untouched. No PH/TestGenie094 run admitted before native verification.

094 measurement correction: initial ESLint exit1 came from existing inline
no-var-requires directives without their plugin loaded, not parse failures. Loaded
the installed @typescript-eslint plugin for the same explicit complexity rule;
49125consumed0. TS scopedCC241->267(+26), functions110->115(+5), max31->32;
/tmp/bas-conditional-eslint-{before,after}-094.json. Go cumulative delta now+342
if candidate qualifies. Actual conditional capability adds147runtime lines; not
a claimed debt reduction. Native fixture now also tests numericNaN rejection and
element presence in selected frames.131frozenpaths unchanged, restart48237 still
pending, APIbuild45083consumed0.

094 managedrestart48237consumed0; API/UI/driver healthy on
976eb5a37d02744dc5c5d46cbfb0303f5be86451677d64bd3438e90c88bf8edc.
Native12974 running /tmp/bas-conditional-native-094.txt; profile17991 pending
/tmp/bas-profile-preservation-094.txt. Do not admit PH until native proof reviewed.

094 native12974consumed1:7/53pass,42normal-condition evidence failures and4frame
admission failures. Non-frame independent branch effects all match, including
CSP and effect-before-SyntaxError exactlyonce despite max_attempts3. The failure
expectations remain unchanged. New RF115/116 register the precise owners above.
Profile17991consumed0, original complete state preserved all3reads. No pending
operations, no PH/TG094 admitted. Originalcandidate/freeze retained unchanged.

Necessary094 boundary extension within authorized BAS/proto roots recorded before
edits: engine capability; shared ConditionOutcome proto owner and EventContext;
telemetry/retained-entry/export; existing protoconv conversions; generated BAS
Go/TS/Python/manifest plus TS re-export. Extension baseline at
/tmp/bas-before-094-extension/manifest.json. Schema FQN/wire fields stay stable;
SDK module imports move and every discovered workspace caller is converted.
Discovery /tmp/bas-proto-discovery-094.txt, owner packages/proto/README.md+Makefile
define scoped make generate SCENARIO=browser-automation-studio. No dependency
install, profile migration or parallel evidence field. Validate native original
53case matrix, engine/telemetry/export/protoconv races and typed consumer builds.

094 extension validation: proto generations60135/20074 bothconsumed0; final
comments correctly follow their messages. Scoped breakingcheck exits0: wire0,
JSON0, unreconciledconsumers0 againstmerge-base7b17b3985faa; owner fell back from
incomplete historical baselinebas-artifact-retention-standard-profile. This is
a merge-base comparison, not a repaired immutable-baseline certificate.
Initial extendedrace62193failed due newly introduced package cycles: protoconv
also imports driver/export/workflow. Moved its two existing condition converters
into the lower existing typeconv/contracts.go owner, updated all callers and
removed originals; no forwarding wrappers. New decode uses that owner too.
This fixes the cycle and removes duplicate field mapping; the movement itself
is not a debt reduction. Extendedrace12376 nowpasses10packages.

Driver consolidated17137 failed (stale copied file-dependency exports); UI
typecheck5618 passed. Generated source has ConditionOutcome in shared_pb but
driver node_modules still points to an old pnpmfile copy. No raw package manager
used. Read packages/proto/README.md, package-governance.md and lifecycle handler:
scoped `vrooli package refresh proto browser-automation-studio --restart --json`
is now pending. FB010 already authorizes BAS managed restart; this refresh is
the documented consumer setup owner and uses scoped generation, not fleet edits.
Receipt /tmp/bas-proto-consumer-refresh-094.json. Re-run driver checks after
refresh, then freeze revised candidate and re-run the original53native cases.
All initial094 native receipts remain failed and preserved. NoPH/TG094admission.

094 refresh40614consumed0 but reportsrunning_setup_deferred, not adopted.
Proto package manifest restart_running_consumers=false defers automatic setup
while active even with explicitrestart. Following documented stopped-consumer
path under existing FB010 authority: managed make stop now pending, then scoped
package refresh --no-restart, then managed start. Do not modify package policy,
copy node_modules by hand or run a raw package manager. Original profiles preserved.
Frame focused suite is tests/integration/typed-action-semantics.test.ts and
unit/idempotency/frame-idempotency.test.ts; the earlier guessed frame.test.ts
matched no suite and must not be counted as frame-test evidence.

094 managed stop completed0. Stopped-consumer refresh43117 is pending; BAS must
be restarted after owner setup and generated-type verification. No native or
owner qualification active. Next maintain driver frame checks at actual paths
unit/idempotency/frame-idempotency.test.ts andintegration/typed-action-semantics.test.ts.

094 consumer refresh43117 completed0 with finalJSONsuccess=true/setup_only;
driver generated exports now current. Driver typecheck60985passes,6driver
suites37597 pass175tests including realbrowser typed-action-semantics and
session-owned frame-idempotency. UI finaltypecheck6360passes; APIbuild44195passes.
Final scoped metrics /tmp/bas-conditional-metrics-final-094.json: runtime
11073->11181(+108), GoCC1496->1512(+16), Go functions263unchanged, max35->34;
TSCC241->267(+26), functions110->115, max31->32. FinalGo cumulative+335.
Removed duplicate condition mapping and narrower telemetry type while moving
existing conversions to lower typeconv; moved paths remain in the comparison.
This capability addition is not a netdebt reduction. NewhandlerCC16 coherently
owns two browser modes, syntax validation, truthful outcome and error conversion;
existing outcome-builderCC32 remains field assembly and a follow-up hotspot.

Final331paths frozen /tmp/bas-frozen-owner-094.json, unchanged after managed
start59458consumed0. Originalinitial131freeze retained separately. Qualified
native helper differs only in output directory; all53expectations unchanged.
Native and profile preservation now pending; no PH/TestGenie094 admitted yet.

094 secondnative20134 consumed1:7/53pass. All46normal conditions select expected independent effects; frame capability repair works. All46lose truth/negated evidence (32alsoactual). Telemetry-only conversion tests missed FileWriter's direct timeline builder. Scope extension to that existing writer captured in /tmp/bas-before-094-extension/manifest.json before repair. Profile52677 consumed0: original complete state equal3reads. No pending operations or PH/TG094admission. Corrected current checkpoint to remove stale refresh/start pending state.

094 writer regression red2 fails actual disk evidence with nil condition. Added the shared typeconv converter to FileWriter context (one runtime line); upgraded existing export test from telemetry-only conversion to actual durable write/reload. Five-package race81082 and APIbuild2839 bothpass. Revised332-path freeze retained in /tmp/bas-frozen-owner-094.json; preceding331freeze saved as -second.json. Next managed restart, unchanged native matrix in final output directory, thenPH/TG only if native passes.

094 managedrestart2692 consumed0, finalwriterbuild52b39732df53c8c3bb7c8a246a666ca97a11f837defda13545923389cb5851dd healthy.332frozenpaths unchanged. Native17572/profile28445 pending. Finalaffected runtime12554->12662(+108); GoCC1794->1810(+16),functions296unchanged,max50unchanged after including entireFileWriter on bothsides. TSmetrics unchanged. No debt-reduction claim.

094 native17572 consumed0:53/53unchanged expectations pass on52b39732df53c8c3bb7c8a246a666ca97a11f837defda13545923389cb5851dd before/after. Saved/exported condition retains truth, negation and actual; independent true/false/error effects match, including frames/CSP and single effect before runtimeSyntaxError. Profile28445 consumed0: original saved profile equalall3reads. Proceed to ownercapture performance and tidiness qualification on frozen candidate.

094 PH35047 consumed0, operationb1e6f95ba344b94458ee19f1a4c8b2c2:100/100firstattempts plus1warmup, servicep95427ms/wall610.723788ms under2000ms. ReceiptSHA6edbd377b8a1eea3df17fb7bb2a18466c8a781bf146d4c7b317bdc2d1df063a8, same producer/contract/config/fixture as093 and current52b397...build. Wallp95~6.1%below093singlecohort; no significance or speedup claim. Board2257pending; TestGenieperformance,tidinessadmissionpending.332frozenpaths unchanged.

094 Board2257 consumed0:1qualifiedcapture,16unknown,productreadyfalse. TestGenieadmission80923 consumed0, run20260923-163006-94c734ad; sole quietwait admitted, /tmp/bas-tg-wait-094.json. Do not poll or re-admit this run. Finalsourcefreeze unchanged; no source mutations during qualification.

094 terminal: solewait64085 consumed1; performancepassedexactPHb1e6f95ba344b94458ee19f1a4c8b2c2,tidinessfailed1157/104long/422complexity/607dup/23coupling/debt34644. Compared093sameoverall/debt,complexity+1/dup-1; do not claim whole-system improvement from count reshuffle.332frozenpaths unchanged; originalprofilepreserved; alljobsconsumed. Full evidence conditional-semantics-2026-09-23.json. Next read-onlyinvestigation finds discarded index-write errors and unused MarkCrash; establish desired admission/notification failures before source repair.

### BAS-WORK-095 — 2026-09-23 UTC — execution status authority investigation

Related085 rejects a missing terminal receipt for synchronous callers; its tests
do not cover effect admission or broadcasts. Recall24179 returned61hits with
provider degradation; existing085source/tests are directly relevant prior work.
Hypothesis1: ignored running write admits effects and ignored terminal write
emits a persisted-status notification. Hypothesis2: executor/stream ownership
already gates those paths, so service's ignored error is harmless. Discriminator:
inject each repository failure, count executor invocations and capture terminal
events independently of database status. Source also shows zero MarkCrash callers
across Go workspace: an unused second status owner plus4no-op implementations.
Architecture target updated before edits; baseline10paths /tmp/bas-before-095/manifest.json.
Scope remains BAS; preserve saved data and existing cancellation/failure behavior.
094 qualification is terminal; no owner operation pending.

095 red85413 consumed1 confirmsH1: saved/adhoc runningfailures each executeoneeffect and publishcompleted; terminalfailures eachpublishcompleted despite failed indexwrite. Healthycontrols pass; H2rejected. Finalization now owned byWorkflowService for allreturns, runningpersist precedes effects, committedterminal precedes notification. Errors remainreturned/logged; panicunwind has explicitfailed sentinel and is not swallowed. Removed unusedMarkCrash production method,3API no-op methods plusprobe no-op, writerGetExecution/UpdateStatus requirements, andupdatedseamdocs. Five-package race76433passes. Added actualpanic/compileterminalnotification controls; initialpaniccheck had wrong GetWorkflowAPI argument, corrected; finalworkflowrace andAPIbuild pending. Comparable scopedruntime2910->2834(-76),GoCC520->516(-4),functions85->83(-2),max56->53; movementalone not counted, removed unused authority/methods give the net reduction.

095 finalworkflowrace51810 andAPIbuild31165 consumed0. Panic regression verifies
failedterminal and sink retirement without swallowing the originalpanic; compile
failure nowpublishes its committedfailedstatus too. Finalsourcefreeze at
/tmp/bas-frozen-owner-095.json. Prior-art reference corrected to085 (084wasAI
textnormalization). Proceed managedrestart and maintained native recording→saved
workflow replay, plus originalprofile preservation and ownerperformance/tidiness.
No live fault injection into the user database; injected failures use isolated
repository fixtures.

095 managedrestart75581 pending (/tmp/bas-restart-095.txt).339frozenpaths.
Finalworkflowrace validates faultmatrix, panic, synchronous wait/teardown and
routed resume/control behavior. Native execution will use maintained
playwright-driver/tests/e2e/record-mode-e2e.mjs with managed API/driver ports; its fixture creates
and cleans only its own resources and retains an independent effect log.

095 managedrestart75581 consumed0 onhealthy b92db9cfbf434a85d6c9d7f69ed2c29624723c4afaec9a01ba3ea4912ed85953. Native67312 consumed0:12/12 maintained recording→typedfreshcontextreplay→savedworkflowAPI→timeline→ownedcleanup checks pass; /tmp/bas-recording-e2e-A3Qn1P/result.json. Profile32808 consumed0: originalcompleteprofileequal3reads.339frozenpathsunchanged. PHcaptureadmitted /tmp/bas-ph-workload-095.json; noTestGenie095yet.

095 PH27565 consumed0:operation1d722c86882de4f527ffe518028892a2,100/100+1warmup,
499msservice/756.742401mswallp95 (<2000ms), receiptSHAd4a5e7e480916311b1b72ec2bfc455196f0a6555e91f1d67a2504696737d54e6.
Singlecohortwall+23.9%versus094; do not dismiss asnoise orattributetosource yet.
AfterTGfinishes, run another quiescent ownercohort to check repeatability.
Board42518 consumed0:1qualified/16unknown/productfalse. TGadmission69220 consumed0:
20260923-164451-777d257b,solequietwait pending /tmp/bas-tg-wait-095.json; no polling.

095 solewait35466 consumed1; TG20260923-164451-777d257b performancepasses exactPH1d722c86882de4f527ffe518028892a2,tidinessfails1159findings/104long/424complexity/607dup/23coupling/debt34644.339frozenpaths unchangedthroughterminal. InitialPHbandpass is not a relative-regression clearance. No testsactive; repeatPH45760 nowpending /tmp/bas-ph-workload-repeat-095.json to investigate+23.9%wallp95.

095 repeatPH45760 consumed0:4011fc450443324647b1e5a1dec1fe89 passes100/100+1warmup,469msservice/681.528072mswallp95,SHAd92ca786d5ed4c4967c6243de0fc6dee65a9c23beff85f883f0cfcf0a183b637.
Absolutebandpass; relativechange+11.6%versus094 remainsunresolved afterinitial
+23.9%. No sourceattribution ornoise dismissal. Retainedperstepanalysis shows
meanswithin3ms/action andextraoutsideactions; exploratorybootstrap assumesIID,
invalidforstrongcausalclaims onserialsharedhost. RecordRF016rechecktrigger:
correlatedadmission/session/finalization/CLIspawn timing orcontrolledpairedbaseline.
No thirdidenticalcohort. Newlatestboardreadpending /tmp/bas-setpoint-repeat-095.json.
Technicaldebtadds2testcomplexityfindings (newfaultmatrixCC14, existingcleanup
test10→11),whileproductionCC-4/runtime-76; reportboth, no suppressedthresholds.
Next higher-impactindependent question isRF011profilecheckpoint recoverywindow:
sourceonlypersists atmanual/stop/close/generation, with no periodicownerfound.
Recall37243 complete70hits/providersdegraded; the existingRF011 repair proves
IndexedDBcapture/restore but explicitly leaves crash/checkpointunknown.

### BAS-WORK-096 — 2026-09-23 UTC — profile checkpoint boundary investigation

RF011/J01/J06/J14/profile-durability, no source edits yet. Hypothesis: active
profile identity has no periodiccheckpoint and therefore exceeds the<=5s
recoverywindow. Alternative: an existingowner capturesstate outside searched
handlers. Native discriminator: newprofile and independent cookie/localStorage/
IndexedDBfixture; after5.5s readhasStorageState, thenexplicitpersist/close/reopen
positivecontrols. Only fixtureprofile/sessions arecreatedandcleaned; existing
protectedprofile remainsuntouched. This firstprobe establishescheckpointbehavior,
not a processkill recoverycertificate.

096 native39473 consumed1:4/5checks pass; checkpointabsent at5507.68ms, manualsave
andLS/IndexedDBreopenpass. Allownedresourcescleaned. Native onlyprovescookie
write, notcookierestore (fixture resetsSet-Cookie). Existingmultipleassociations
are supported byregistrytests; do not silentlyreject duplicateprofile sessions
or add a last-writer owner policy. Before periodicwrites, repair confirmedsource
race: capturegetsprofile beforeI/O andcommits afterbinding mayclear/rebind.
Architecture target096 updated;4filebaseline /tmp/bas-before-096/manifest.json.
Maintainedpublicpersist regressions will replacebinding inside the capture seam
and require old/newprotectedstates unchanged plus non-success acknowledgement.
No096sourcequalification admitted;095build remainshealthy.

096 bindingred72274 consumed1:clear/same-profileABA/other-profile/cancel all
acknowledge200 andoverwriteoriginal. RF118 registered. Service now owns opaque
associationidentity and capture serialization; adapter captures complete state,
commit requires samebinding and livecontext. Conflict isHTTP409; no profile
association remains existingexplicitno-op behavior. Removed unused StartSession/
EndSession APIs (no productioncallers); ported their state/tab/touch assertions
toactualTouch/SetActive/Persist/Clear owners. Firstgreen99290 failed because
a test-edit replacement malformedtwoifstatements; correctedgreen2 passes both
packages. Added deterministic serial/cancel/unrelatedprofile regression.
Five-package race andAPIbuild nowpending. Metrics saved
/tmp/bas-profile-binding-metrics-096.json; no096freeze/restartyet.
RF011periodic checkpoint remains missing; this ownership repair alone doesnot
qualify crash recovery or five-secondwindow.

096 fence checks12053 andAPIbuild77183 consumed0. Fivepackages passrace.
Currentfencecostruntime735->750(+15),Go121->122(+1),functions45->44,max7->8.
Continue within096 to periodiccheckpoint on the now-fencedtransaction before
deployment. Extendbaseline toapi/main.go for lifecycleworker andhealthwiring.
One2stick,2s capturedeadline,<=5shealthfreshness; cancellationjoinedbefore
driverstop. Shared-profile multipleactivebindings are keptfullysupported for
manualoperations; auto checkpoint must reportexplicitdegradation andskip rather
than chooseoneidentity orchangeadmissionsemantics. This limitremainsRF011open.
No newdataformat/migration/secondstorageengine; originalprofilespreserved.

096 periodic source implemented: oneAPI-ownedjoinedloop viaexisting schedule.Clock
ticker, per-bindingbrowserdeadlines, concurrentindependentcaptures, existing
fencedaggregatewriter. Only uniqueprofilewriters automaticallysave; ambiguity
checked beforecapture andagainatomicallyatcommit, degradedhealthreported while
manualcapability remains. APIprofile_checkpointshealthobserves errors/staleness;
shutdown cancels/joinsbeforeSidecarstop. No newprofileformat/data conversion.
Focusedperiodicfirst run found a testfaultinjector race (rawMockRepository.SaveErr
writtenoutside synchronization); replaced only thetestfault flag withatomicBool.
Focused88583 passesrace,APIbuild94414passes; full5packageperiodicracepending.
No sourcefreeze/deployment yet; originalnativecheckpointfailure retained.

096 full periodic race56066 passed all5packages; confirmed finalAPIbuild64700 exit0. Earlier build output session ID was lost in tool truncation; verified no build remained before this explicit finalbuild. Candidate source frozen in /tmp/bas-frozen-owner-096.json (344paths). Managed restart now admitted; native5case and independent restart recovery probes next. Runtime1976->2107(+131),GoCC265->296(+31),functions50->54; cumulativeGo+362. New durability capability is not a net debt reduction claim.


### BAS-WORK-097 — 2026-09-23 UTC — checkpoint deployment and validation result

FB008 remains active; BAS-FB-008 reread before work. Managed restart completed
healthy after managed `make restart` requested 2026-09-23 14:22 EDT and completed 14:26 EDT. The pre-restart build was `sha256:1f7ea2d3255419e5f63edf0bda4b782d7b288a6428b656e62cdca286a6819b3d`; lifecycle computed new identity `sha256:898284cacb4ee6a6c119d78f02e5b4bd8e7f67defb9df93bdf7bb3b15c3d9412` because authored progress/PROBLEMS/evidence documents had changed. All344 frozen code paths remained byte-identical. Restart authority remains FB010. No user profile data was edited. RF011 source
capability is deployed: API-owned joined periodic capture at 2-second cadence,
2-second I/O timeout, freshness health, and ambiguity degradation when multiple
active bindings reference one profile. Five-package race and API build passed in
096; the earlier native probe is still red for the pre-deployment binary and
must not be reused as current behavior.

Before the document updates/restart, owner capture operation
`31353d927434e12e2a0dcb1ab3c21097` passed100/100+1 warmup at439ms service /
595.510768ms wall p95. It became stale when authored documents changed, as
expected from lifecycle hashing of all authored BAS inputs. After restart, fresh
owner capture `c3f7aa3fc435f2e1d3e02e737ec16918` passed100/100+1 warmup at438ms
service / 624.797691ms wall p95 on identity `898284...`, within2000ms. Owner get
returned the same receipt; setpoint-read reports capture in-band and 16/17 rows
unknown, so product qualification remains false. Receipt SHAs and identities are
preserved in the evidence JSON.

Test Genie run `20260923-174338-ad9c918f` ended terminal FAIL after 1486s; no
active run remains. Unit failed on canonical render projection drift and the UI
coverage command. Replaced direct Testing Library `render` in
`ui/src/domains/recording/capture/TabBar.test.tsx` with `renderWithProviders`;
focused assertions pass 12/12, and post-repair `pnpm run test:full` passes
1190/1190. `pnpm run test:coverage` still fails its unchanged 85% aggregate floor
(30.42% statements/lines, 33.57% functions, 67.16% branches); coverage debt is
real and remains RF-014. No coverage floor, test, or assertion was skipped.

The same Test Genie run's workflow child was canceled at its 15-minute deadline
with zero cases executed. Workflow Health persisted an Experience Manager
readiness error: `projects-grid` pins `experience-surface@1.0.0` without a
canonical experience contract. BAS workflow behavior remains unknown. Filed the
external owner defect as `knw-1790187409653888938` using report-bug. Tidiness
failed at 1159 findings (104 long files; complexity 424; duplication 607;
coupling 23; duplicated-line debt 34644), so no debt reduction is claimed.
Programs phase returned failed/L0 with no program findings; contradictory evidence
leaves it unknown. Scheduler warned unit serial calibration was older than 168h
and host swap use was at least 50%; this explains serial fallback, not a test
result. Logs/results: `/tmp/bas-tg-unit-097.log`,
`/tmp/bas-tg-workflow-097.log`, `/tmp/bas-tg-tidiness-097.log`,
`/tmp/bas-tg-programs-097.log`, and corresponding `*-result-097.json`.

This cycle added no production source and makes no complexity/debt-reduction
claim. The only BAS working-tree change is the canonical test-render repair. The
prior `/tmp/bas-profile-checkpoint-native-096/receipt.json` remains red historical
evidence from before periodic checkpoint deployment (4/5; no automatic state at
5507.68ms); the new build passed the maintained 5-case probe. Next discriminator:
verify multiple-binding degradation and abrupt process failure. The managed
restart recovery control now passes, but does not equal hard-crash proof. Continue
independent RF014/maintenance work.

### BAS-WORK-098 — RF029 input semantic preservation — deployed, partial qualification

`refactor_regressions.py` returned2passed/1failed/7unavailable. The input probe
is an actual `useInputForwarding` module probe with a synthetic WebSocket sink;
it fails Ctrl+A, Command+C and Alt+F because printable keys are emitted as
text, drops Shift from pointer down, and forwards composing `Process`. Plain
`x` and Shift+Tab controls pass. This reproduces existing RF029 and matches the
contract in `REFRACTOR_ASSESSMENT.md` §Human input and recorded-workflow
semantics. Hypothesis: classification by `key.length` ignores active modifiers,
pointer payload construction omits modifier state, and composition is not
filtered. Repair only after the independent hook tests are red; preserve plain
text and non-composing key behavior. BAS-RF-029 remains open pending native
browser/OS confirmation. Other regression pack cases: recording API and profile
pass; input fails; seven cases unavailable; driver producer is classified
unavailable despite internal failing observations. Exact pack result:
`/tmp/bas-regressions-097.json`.

Boundary extension within the authorized BAS scenario: forwarding only a pointer
modifier payload is insufficient because the driver route currently ignores
pointer modifiers. The same browser-input contract assigns modifier key
press/release lifetime to `playwright-driver/src/routes/record-mode/recording-input.ts`;
the UI hook owns event classification/payload. Add route tests for down/up,
click, and release after action failure. This is necessary for BAS-visible input
semantics, not an extension into the shared Playwright package or control plane.
Extend the already-maintained recording E2E fixture to observe independent DOM
keydown/pointer events through its local server; this supplies a real Chromium
oracle for the driver route after managed deployment. It does not claim host-OS
shortcut behavior outside the supported headless browser fixture.

097 native five-case probe now passes on current build: fresh profile empty,
independent cookie/localStorage/IndexedDB writes, checkpoint state visible at
5504.810455ms, manual persistence positive control, and close/reopen storage all
pass. Managed restart probe staged only profile `6115262f-97b2-4941-967f-7c29f6729a54`,
then `make restart` returned healthy; same-origin browser restored cookie,
localStorage and IndexedDB. Test stage 2 deleted its fixture profile and closed
its session; zero owned sessions remain. Receipts:
`/tmp/bas-profile-checkpoint-qualified-097/receipt.json` and
`/tmp/bas-profile-restart-097/restart-receipt.json`. Build identity changed from `sha256:1f7ea2d...` to `sha256:898284c...` because
BAS authored progress, issue and evidence docs were updated before restart; the
lifecycle identity hashes those authored inputs. SHA comparison found0changed and
0missing among all344 frozen code paths. This proves managed restart recovery
across unchanged implementation source. It does not prove hard process crash,
power loss, or multi-binding ambiguity behavior. A new capture owner receipt
qualifies the post-restart identity. Current-cycle evidence is
`internal/evidence/rehabilitation/periodic-checkpoint-097-2026-09-23.json`.

Implemented the expected UI hook behavior and BAS-owned driver modifier lifetime.
Hook tests pass8/8; driver route tests pass26/26; UI/driver typechecks and
changed-file lint pass. The retained actual-module input regression passes7/7,
and the recording E2E contract harness passes16/16. An initial live fixture
attempt dragged across its effect button, correctly producing an extra fixture
effect; moved the modifier drag outside the button and the complete real
Chromium/API journey then passed13/13, including fresh-context replay and saved
workflow execution. All synthetic sessions and API fixture data were cleaned.
Live receipt: `/tmp/bas-input-e2e-098.txt`, artifact directory
`/tmp/bas-recording-e2e-UaDZME`, execution
`cda48176-c718-485f-a694-695e25a1807e`.

Managed `make restart` completed healthy on build identity
`sha256:6a42c3c70ce9eef005d95bfcd0b9d94587e55eb9e9ab0618b8d884201f2e691e`.
Test Genie unit `20260923-184157-bdd24c0b` ended FAIL on the unchanged 85%
aggregate coverage floor (30.52% statements/lines, 33.61% functions, 67.26%
branches); the prior canonical render projection drift is gone. Test Genie
workflow `20260923-190021-1c60cd2a` ended FAIL after901s with zero observations:
workflow-health's durable provider missed the 15-minute deadline, and the log
reports host swap >=50% plus scheduler serial fallback. This leaves the workflow
check unknown, not a BAS behavior failure. Findings/log:
`/tmp/bas-tg-findings-098-workflow.json` and `/tmp/bas-tg-workflow-098.log`.

After the workflow run ended, current-build Performance Health capture
`cb5616c56bb2f9a6b6bae1274aea1d4c` measured100/100+1 warmup at355ms service /
517.908453ms wall p95 against2000ms. The rehabilitation setpoint read
`prog_581ba087-1641-44b3-8c61-d59313a56c4c` reports capture as1/17 qualified;
16 are unavailable and product qualification remains false. Preparation
contract errors0; final source inventory digest is
`2cb941679be49b2f61de3f7701790f94ac6b16c7c34093f41f2c55ffe21ff13d`.
Retained and direct evidence is linked from
`internal/evidence/rehabilitation/input-semantics-098-2026-09-23.json`.

RF029 remains open for native OS shortcuts, Command+C/clipboard effects,
platform differences and native IME behavior. No OS-level claim follows from the
headless Chromium fixture. Continue with the platform input corpus when its
owner is available; otherwise preserve RF029 as partially repaired and
unqualified.

### BAS-WORK-099 — RF056 recorded input replacement semantics — implementation, deployed validation pending

Fresh setpoint read `prog_afd4a494-b0c5-4f26-9886-933b66b2bbaa` reports only capture qualified (1/17); product qualification remains false. `program-runtime.learning-maintain` list is empty. The targeted prior-art search found no active or archived BAS fix for RF056; the broad recall surfaced related historical driver input work but no applicable repair. Scenario status is healthy on `sha256:6a42c3c70ce9eef005d95bfcd0b9d94587e55eb9e9ab0618b8d884201f2e691e`.

W3 localized implementation/evidence defect; expected behavior is already in the active typed action contract and RF056. Competing hypotheses: (1) HandlerAdapter drops typed action options during replay; falsifier: it constructs HandlerInstruction with the original `entry.action`. (2) handlers do not execute typed options; falsifier: click options are passed to Playwright, input `submit`/`clearFirst`/`delayMs` are read, and keyboard modifiers are pressed/released. (3) recording conversion makes a semantic error: the page recorder captures the entire current input value, then the typed converter emits `clearFirst=false`, causing the handler to append the full recorded value on replay. The isolated current source supports (3); the independent experiment is an actual Chromium capture from a prefilled field, replacement with Ctrl+A, followed by replay into a fresh matching page and comparison of the final input value. If confirmed, fix only the conversion owner, then retain the browser journey and converter regression. Do not change contracts, handler architecture, shared proto, helper semantics, or unrelated workflow behavior without a reproduced cause. RF056 stays open until the journey proves the expected result.

The live regression confirmed (3): before the fix, the typed input entry contained
`value="replacement"` and `clearFirst=false`; replay left the fresh prefilled
field at `original`. The replay adapter preserves the original typed ActionDefinition.
The handler source accepts all declared click and input fields; the prior issue
wording that those handler options were ignored was stale. Changed only the
recording-to-proto default to `clearFirst=true`, retaining an explicit false.
The actual recording/replay integration then passes with final input
`replacement`. A separate Chromium handler case observes two right-button
mousedown/up pairs with Shift held and at least50ms between each pair, for a
typed 80ms delay and click count2. The replay case observes Control+A with
`ctrlKey=true`. Full timeline integration 15/15, Interaction/Keyboard unit
tests15/15, driver typecheck and scoped ESLint pass. The direct commands and
terminal Test Genie result are recorded in the cycle evidence.

Test Genie unit run `20260923-193334-f1a001cc` terminal-failed after 500 seconds;
its blocking finding is the failed `playwright-driver` `pnpm run test:coverage`
command. The returned terminal summary does not identify a more specific
coverage cause. The direct focused test, typecheck and lint results remain
green; the broader unit phase is not qualified by this run. The managed restart,
fresh capture and setpoint read are recorded above.

### BAS-WORK-100 — RF055 stale execution recovery query — implementation validation pending

Current setpoint `prog_80c243ef-5aa1-4d9c-a6eb-d1308836fced` still reads capture only (1/17), with 16 outcomes unavailable and product qualification false. Learning-maintain has no queued BAS actions. BAS is healthy on `sha256:f0f3f08cc48ca8a0e39265b7a882bb4738f790e35f2ae650b7d0f28ee4825a3b`; no current-cycle code has been deployed yet. Search-hub surfaced related orphan-session recovery but no directly matching execution-recovery fix; the scoped BAS fix lookup returned no active or archived matches.

RF055 is W3: the five old execution rows were a live behavior defect. Competing causes were (1) recovery pages the newest 1,000 mixed-status rows before detecting stale active records, (2) the staleness comparison uses an inappropriate timestamp/threshold, or (3) the repository rejects status finalization. A read-only Connect query before deployment found five RUNNING rows dated September 7–16, 2026. The retained regression builds 1,000 newer completed rows before stale running and pending rows, plus one fresh running row; the old service returned zero stale rows, confirming (1) and falsifying the selector/threshold explanation for this case. The repository already supports SQL status filtering with zero limit meaning unbounded. Recovery now queries running and pending rows separately, then retains the existing five-minute `UpdatedAt` guard and interrupted-status updates. The red test failed before the change at `TotalStale=0`; `go test ./services/recovery ./database` and API build pass. A managed restart took157s on `sha256:0372dbb587fc3a6814374b0963340b2b8782bb1435748f457ec477db5852cfc0`; startup logged 5 stale / 5 recovered. Follow-up Connect reads show RUNNING=0 and PENDING=0, and failed-status pages retain all five target dates with the interrupted reason (plus one older pre-existing interrupted row). Test Genie `20260923-195501-975ceecb` terminal-failed after516s on UI `pnpm run test:coverage`; the phase remains unqualified. `golangci-lint run ./services/recovery/...` cannot typecheck in this environment due vendored Go1.26 code under Go1.25 and an unrelated `internal/hostinventory.Probe.Run` error. The out-of-scope lint/typecheck observation was filed through report-bug as `knw-1790194580329930020`. The incoming continuation repeats BAS-FB-008 verbatim, so no duplicate feedback entry was added. The contract validates with errors0. Fresh capture `4ab790ac97807d86cace1a85416a1ea6` is100/100+1 warmup at432ms service/623.523482ms wall p95 under2000ms. Setpoint `prog_1a0bdbde-d728-400c-bff6-e05608c9dae5` is1/17,16 unavailable, product false. `gocyclo` measures `findStaleExecutions` 6→5; this is a one-function reduction, not a domain-wide debt/duplication/coupling claim. Source inventory digest `df72437cc7835d141b3664fc6e9beb00f712a2f20e58b1dd3ee8847ea3e48690`. No Plan Manager, external journal, or subagent operations. Durable process/generation ownership across concurrent/duplicate API processes remains unverified, so RF055 remains open. Full evidence: `internal/evidence/rehabilitation/stale-recovery-100-2026-09-23.json`.

### BAS-WORK-101 — RF055 recent orphan age gap — managed-lifecycle repair verified

Fresh setpoint `prog_ae4df731-a173-40a7-87aa-2fa2339a6134` remains capture-only (1/17); the other 16 rows are unavailable and product qualification is false. Learning-maintain had no queued BAS actions. Search-hub prior art found related session-orphan recovery but no direct execution-generation fix. Source inspection confirmed the sole production recovery call is before router serving and scheduler start; executor heartbeats are timeline telemetry, there is no database heartbeat write to `UpdatedAt`, and there is no periodic recovery caller. Competing hypotheses (1) recent active rows survive the one-time five-minute age gate, (2) database heartbeat makes those rows live, and (3) a later sweep reclaims them: the source supports (1) and falsifies (2)/(3).

The new red regression seeded one-second-old RUNNING and PENDING rows. Before the change, it failed with zero recovered rows. Startup recovery now queries RUNNING and PENDING separately without a limit, finalizes every active row before work admission, and no longer applies the five-minute age gate. The >1,000 newer-completed-history regression now also verifies a one-minute-old active row is recovered. The internal method and result were simplified to describe the process boundary; stale-threshold options and unused per-row result details were removed. Focused `go test ./services/recovery ./database` and `go build ./...` pass. `gocyclo` reports `findActiveExecutions` at3 versus the prior cycle's `findStaleExecutions` at5; this is a local helper reduction only, with no domain-wide maintainability claim.

Managed `make restart` completed with BAS healthy on new API pid1189538. Runtime startup logged `No interrupted executions found during recovery`, confirming the deployed process ran the new path; there were no active rows to mutate during this startup. Previous-cycle live reads had already confirmed the five September 7–16 rows were interrupted and RUNNING/PENDING counts were zero. Build identity from the fresh owner capture is `sha256:0da400807449f9ae90fade170c03225ef71859f8ebf21bdfa8177543847cf9c4`.

Test Genie unit run `20260923-203045-3e5d2e12` terminal-failed after509s with zero observations on `ui: /usr/bin/pnpm run test:coverage` (`TEST_EXECUTION_FAILURE`); this is the same unrelated coverage-command blocker seen last cycle, and the owner phase remains unqualified. No repeated run was started. Repository-wide `vrooli contract validate` was attempted once and canceled after structure-health returned unavailable; no BAS contract/schema was changed. Targeted `git diff --check` is clean. Fresh Performance Health capture `1e9fc96f5be8bc29ab6ef9599f28b301` measured100/100 plus one warmup at419ms service/598.636585ms wall p95 against2000ms; receipt SHA `dfeb266e676b55ef76f51e9c165324e673cae1ed1b824e79320a22d68a5bffa2`. Setpoint confirms capture in band,1/17 qualified,16 unavailable, product false. Selected-source inventory digest is `87051e6d47d519905c8a34ef22b834c3340be68e9a21c3723bedcccfd6497059`. The current continuation repeats BAS-FB-008 verbatim; no duplicate feedback entry was created. RF055's managed single-process startup behavior is repaired; simultaneous/concurrent API ownership remains unqualified and is preserved as the open boundary. Full evidence: `internal/evidence/rehabilitation/stale-recovery-101-2026-09-23.json`.

### BAS-WORK-102 — RF058/RF060 issue disposition review — stale rows corrected

Feedback BAS-FB-008 is unchanged and already retained verbatim, so no duplicate was added. The prior-art recall surfaced unrelated capture-image history; its scenario-scoped provider was unavailable, so reuse the strong local cycle013 source/evidence record instead of repeating that investigation. The issue row still says `timeline_entries` is absent, while current source shows it in `api/internal/recording/schema.sql`, ordered into production `SchemaProviders`, and applied through the shared bootstrap. The existing routed-pool regression appends, reads and preserves an entry across schema reapplication. Competing explanations: (1) the fix was completed and only the issue register is stale; (2) the provider is registered but bootstrap omits it on a real/routed pool; (3) the current test uses a private schema that hides drift. Run the owner tests against the current worktree to distinguish these. No source change is justified before that check.

Current tests confirm hypothesis1: `go test ./database -run 'TestNewConnectionRoutesRecordingWritesToLeasedTestPool|TestSchemaProvidersExposeCompleteEmbeddedSQL' -count=1` and `go test ./services/recording/persistence -count=1` pass. The routed-pool test starts a fresh production database, creates a newly leased test pool, appends and reads a journal entry there, confirms the primary pool remains empty, reapplies declarative schemas, and verifies the journal survives. The preexisting cycle013 red/green artifact retains the original `no such table: timeline_entries` failure and repair evidence. Updated BAS-RF-058 to resolved while preserving the separate RF-002/RF-024 full-durability limitations; no production source changed this cycle, so no complexity/duplication/coupling delta is claimed.

The same evidence audit found RF060's issue row still described the pre-cycle023 screenshot truncation/mutation bug even though the current implementation and receipt fault tests already enforce the fix. `go test ./automation/execution-writer -run TestScreenshotEvidenceRequiresValidReceipt -count=1` passes in1.038s; the matrix exercises success and store errors, nil/incomplete/wrong-size receipts, missing storage/image, invalid/truncated/over-budget images, and explicit no-artifact behavior. Assertions preserve caller screenshot bytes/metadata and notes; stored bytes equal the encoded source and decode successfully. Updated RF060 to resolved for integrity behavior, with the known isolated full-decode cost and hostile-dimension/end-to-end limits explicitly left unqualified.

`python3 scenarios/browser-automation-studio/docs/internal/refactor_contract.py` passes preparation validation (24 preservation journeys present, zero errors; this is not product qualification). Source inventory digest remains `87051e6d47d519905c8a34ef22b834c3340be68e9a21c3723bedcccfd6497059`. Current setpoint `prog_79af08a8-c1cd-45c7-8596-f9628c317313` reads capture in-band at419ms p95 from workload `1e9fc96f5be8bc29ab6ef9599f28b301`; 16/17 outcomes unavailable and product qualification false. No fresh capture or Test Genie rerun was warranted because the candidate source and build are unchanged; the previous owner unit run is terminal-failed on UI `pnpm run test:coverage`, with no recovery assertion failure. The prior build remains `sha256:0da400807449f9ae90fade170c03225ef71859f8ebf21bdfa8177543847cf9c4`. Preparation and issue-specific evidence: `internal/evidence/rehabilitation/recording-schema-review-2026-09-23.json` and `internal/evidence/rehabilitation/screenshot-integrity-review-2026-09-23.json`. Next useful review: test screenshot decode resource bounds and measured allocation/latency from RF060 without choosing an unsupported policy cap.

### BAS-WORK-103 — RF060 screenshot decode resource-bound audit

The question was whether the existing screenshot integrity decoder can be safely bounded by current policy. Source inspection confirms the active explicit screenshot handler defaults to full-page PNG capture, while passive telemetry is viewport-only by default and full-page only when `SCREENSHOT_FULL_PAGE=true`; arbitrary page content can influence dimensions of explicit or opted-in full-page capture. The initial cycle103 wording incorrectly generalized full-page behavior to automatic telemetry and cited the inactive root `playwright-driver/server.js`; cycle104 corrected the receipt and this record after reviewing the active `src/` owners. The API retains at most4MiB of encoded screenshot data by default, but `image.DecodeConfig` only reads dimensions and the maintained fault test confirms header-only validation would accept a truncated PNG. Go1.26 PNG/JPEG full decode allocates raster-sized image buffers, so this byte bound does not establish a memory bound. I did not construct or fully decode a hostile-size raster without a declared memory envelope.

The maintained `BenchmarkScreenshotOutcomePersistence` run measured36.375ms/op,3,878,791B/op and1503 allocs/op for the fixed1280x720 fixture. This is consistent with cycle023's isolated cost and still does not establish browser end-to-end impact. There is no decoded-memory budget in `ArtifactLimitsConfig`; therefore no arbitrary pixel cap was added. Corrected its stale `MaxScreenshotBytes` comment to describe intact-image omission instead of truncation. RF060 remains resolved for integrity and storage receipts, while memory bounds and end-to-end impact remain unqualified.

Verification: `go test ./automation/execution-writer -run '^TestScreenshotEvidenceRequiresValidReceipt$' -count=1`, `go test ./config -count=1`, and the benchmark passed; contract preparation validation passed with24 journeys and zero errors; inventory ran with untracked sources included; `git diff --check` passed. Setpoint run `prog_e0a0f26f-f102-4eb1-a43e-8505a5167af3` reads capture in band at419ms p95 from workload `1e9fc96f5be8bc29ab6ef9599f28b301`, with16/17 outcomes unavailable and product qualification false. No source behavior changed, so no managed restart or Test Genie repetition was warranted; prior UI coverage runner failure remains unchanged. Source inventory observed digest `34a87bcd7221f6defe500a86d13ec8d4e083043d04d7ff2b6dbdc8b22735f085` in a shared worktree; no debt delta is claimed. Full audit: `internal/evidence/rehabilitation/screenshot-decode-bounds-2026-09-23.json`.

### BAS-WORK-104 — RF119 compressed-raster expansion and active-budget audit

The prior pass correctly withheld a guessed cap but missed a useful discriminator: the contract does declare `idle API+driver <=300MiB PSS` and `one fixture browser+shell <=1GiB PSS`, while omitting active/concurrent decode peaks. Search-hub prior art also surfaced an older every-step screenshot performance issue; current source already wires the typed per-step `ScreenshotCapturePolicy`, preserves explicit handler screenshots, and focused API/driver behavior tests pass, so that repair was not reopened.

A valid constant-color1280x12800 PNG encodes to74,012 bytes (<4MiB) and the direct Go decoder allocates65,594,288 bytes in72.487ms. Promoted the reproduction into `BenchmarkScreenshotOutcomePersistenceExpandedRaster`, which measures the real writer path at104.759781ms/op,66,607,544B/op,7,547 allocations and16,384,000 pixels. It creates source pixels lazily; the encoder does not hold a second64MiB input raster. This confirms the byte ceiling does not bound decoded memory. `MAX_SESSIONS` defaults to10 and permits100; the writer has no aggregate decoder budget, so concurrency multiplies this allocation, although a multi-session peak was not measured.

The contract's idle PSS target is not an active capture limit. Current API+driver PSS sampled109,022KiB at21:19:14Z while the managed scenario had active browser children; this is explicitly not an idle or peak baseline. I did not derive a pixel cap from it. Two disposable direct Playwright attempts failed in `page.setContent` before screenshot capture: a flat-color1280x12800 fixture timed out waiting for `load` after30s, and a simplified fixture timed out waiting for `domcontentloaded` after15s. Both temporary browsers were closed; scenario API/driver health remained good. Browser-side peak and end-to-end screenshot cost remain unknown. Opened RF119 for the separate resource defect and linked it from RF060 while retaining RF060's resolved integrity status.

Focused `go test` for writer integrity and API screenshot policy passed; the retained expanded-raster benchmark passed. The driver policy test passed10/10 with `pnpm exec jest tests/unit/execution/screenshot-capture-policy.test.ts --runInBand --coverage=false`. The same focused test without `--coverage=false` also passed10 assertions but exited1 because its partial run reports function coverage1.96% against the package-wide15% floor; no coverage threshold was changed. Contract preparation passed24 journeys/0 errors. Setpoint `prog_8b400f9e-e7c6-41ae-aa09-8af66d1d50f6` reads capture at419ms p95,1/17 qualified and16 unavailable; product qualification remains false. Inventory digest `da6dc7c00c76e416d44b172ca00e3b12a774ce057357466a1930952d3ae49d8b` includes the new benchmark test; no runtime debt reduction is claimed. Repository-wide `git diff --check` still reports unrelated shared `audio-tools/ui/index.html` trailing whitespace and a landing-page CRLF warning; the scoped BAS check passes. No production behavior changed, so no restart or Test Genie rerun was warranted. Full evidence: `internal/evidence/rehabilitation/screenshot-raster-expansion-2026-09-23.json`.

### BAS-WORK-105 — RF119 concurrent decode admission; RF120 sidecar startup probe

Promoted the managed 1280x12800 solid-page capture into execution-writer testdata after two successful `ai preview-screenshot` runs (752ms and709ms;74,107-byte PNG, SHA256 `5c4a1af045071c71aaf3fbb6d5515e94f84e65474e4145209e4e8e123723d5a5`). Before the guard, an isolated real `FileWriter.RecordStepOutcome` probe held storage at one shared barrier: one/two/four overlapping calls increased live heap by65,814,576/131,354,392/262,454,096 bytes, and all four reached storage together. This is a writer-process measurement, not managed service PSS.

Added one process-wide weighted admission owner shared by both production writers. It estimates five bytes per raster pixel under a192MiB budget, waits with request cancellation, rejects a single over-budget estimate before full decode, and continues to fully decode accepted PNG/JPEG bytes before persisting them unchanged. The maintained writer test proves the realistic16.384MP managed fixture still persists byte-for-byte; budget capacity/cancellation and oversized-header rejection are covered. Focused writer tests pass; previous writer and supervisor race runs pass. This bounds concurrent estimated decoder ownership, not process PSS, retained Go heap, decoder overhead or browser memory. Runtime behavior and its tests add code; no domain-wide complexity, duplication or coupling reduction is claimed, no callers were left on the old path, and no dependency or duplicate policy was added.

The scoped `unit` Test Genie run `20260923-214208-f3e727e7` terminated failed after536s with zero observations/degraded evidence. Its API suite exposed `TestProcessSupervisor_AutoRestart/restarts_after_crash`: first health polling slept500ms, equal to startup timeout, so health could time out before a probe under load. Extended the necessary BAS-facing boundary to `api/sidecar/supervisor` (reason: real failing assertion; owner: `ProcessSupervisor`; changed paths: `api/sidecar/supervisor/supervisor.go` and `supervisor_test.go`) and now probes immediately before waiting between retries. The new25ms regression plus repeated AutoRestart pass3 times; sidecar race suite passes. The same Test Genie receipt reports a driver native stream expected-height240/observed153 mismatch for DPR1/2 CSS-scale regular mode; the exact10-test focused suite passed twice immediately after, so the broader-run cause remains unknown and the driver phase remains unqualified. UI coverage still fails its unchanged85% floor (30.52% statements/lines,33.61% functions,67.26% branches), tracked by RF014. The overall Test Genie run is not claimed as passing.

Managed lifecycle restart `make -C scenarios/browser-automation-studio restart` succeeded; BAS is healthy on `sha256:a708c5cc95548c958ab2ae0f42e67dc9e711397eebe357b5aa37e2b98911891b`. One optional Claude key is unavailable and no required credential is missing. A fresh CLI screenshot capture passed in2670ms (209,285 bytes); readiness used generic fallback due to the existing Experience Manager contract issue `knw-1790187409653888938`. The first post-restart setpoint read correctly rejected the old-build receipt, reporting0 readable/17 unavailable. Performance Health then ran workload `16fd5c4b86fcb3bc8d0ef5ae61a45da7` on the current build (100 measured attempts +1 warmup,421ms service p95/616.014912ms wall p95,2,000ms budget); `workload-get` retained the same receipt and hash `f55efc924d8b235845b0f5e51a8253e90fcd13801e4c536a6e9afb3281a22cff`. Final governed read `prog_a01c4e1a-05bb-4c36-961a-48a94585d3d5` admits capture1/17 and leaves16 pending, product qualification false. Contract preparation passes24 journeys/0 errors; shared-tree inventory digest is `016008c7c5b8b0dcbd7f4b3d107974b9c1bde64eb20e25df3ae3553199de02de`; scoped diff check is rerun after records are finalized. Full evidence: `internal/evidence/rehabilitation/screenshot-decode-admission-2026-09-23.json`.

Next discriminating review: produce a controlled concurrent full-page capture cohort on the managed API+driver, measure their combined peak PSS against the idle qualification boundary without treating it as an active cap, and compare several realistic PNG/JPEG dimensions. Keep full-page preservation and malformed/truncated rejection intact. Separately rerun the driver native-stream case inside a clean broader unit cohort only after preserving the failed Test Genie receipt and determining whether current host pressure differs.

### BAS-WORK-106 — RF119 decode budget corrected for Go heap residency

Prior-art search returned the cycle104/105 raster-expansion receipts and BAS cancellation-evidence history. The initial cancellation hypothesis was rejected: `SimpleExecutor.recordOutcome` deliberately persists terminal audit evidence with `context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)`. The remaining falsifiable hypotheses were that5B-per-pixel admission bounded PSS, that completed but uncollected raster allocations remained resident across admission batches, and that10B-per-pixel could preserve the16.384MP fixture while reducing memory. The retained `FileWriter` probe resolved these: ten same-fixture writes under five bytes peaked277,719KiB; unrestricted two-GiB admission peaked664,296KiB; ten-byte/192MiB admission peaked149,479KiB. Live heap at the blocked-storage barrier fell from131,267,208 to65,736,136 bytes under the new policy. Total allocations stayed about660MB, so this controls peak concurrency/residency, not total work. Group completion increased from405ms to655ms versus the prior policy (unrestricted230ms); this is an intentional measured large-image throughput tradeoff.

Changed runtime files are `api/automation/execution-writer/screenshot_decode_budget.go` and its managed-image invariant in `file_writer_test.go`. Both production `FileWriter` constructors already shared the process-level owner; no caller conversion, duplicate implementation, dependency, or obsolete runtime path was introduced. Added retained Linux measurement producer `docs/internal/evidence/rehabilitation/screenshot-decode-writer-probe-2026-09-23.go`; it invokes actual `RecordStepOutcome` and blocks only the in-memory storage sink. The new desired-behavior assertion failed before the policy change (`163840000` bytes for two fixture estimates is not greater than `201326592`) and passed after charging ten bytes per pixel. Full writer package, race suite, focused fixture regression and expanded-raster benchmark pass; the one-iteration benchmark is91.006ms/op,66,613,208B/op and7,580 allocs/op, not a comparative latency claim. No domain-wide complexity, duplication or coupling reduction is claimed. A forced global GC before releasing permits was not selected: it would couple persistence latency to whole-process collection and has no capture-cost measurement.

Managed lifecycle restart succeeded on `sha256:6aa67b6b0db708dc1807d412458ba236e654f2677ff19ed82873662630f41180`; BAS is healthy with no required credential gaps. Two fresh Performance Health cohorts on this build passed100/100+1 warmup at445ms and358ms service p95 (606.559757ms and522.358187ms wall p95); both satisfy the2s band. Same-candidate variance means they do not establish a performance regression or improvement. The later20ms same-tick API+driver sampler recorded203,405KiB peak across the100-capture viewport workload from160,985KiB baseline. This is below300MiB numerically, but it is active viewport evidence; the contract's300MiB row is idle-only and this does not establish API+driver full-page peak. The first run's per-process maximum sum330,094KiB is explicitly an upper bound because process maxima were asynchronous; the corrected later sample is used for the same-tick peak. A post-first-run point sample was162,263KiB, suggesting warm retained state but not enough cycles to identify a leak or plateau.

Latest workload `eba895ae48189202f9e7924fb1fd49ab` and receipt hash `cf7bb6ce8ed3f35c615f72d78c422906150090b0c229fd8d85b542f8792e6a95` were confirmed by `performance-health sweep workload-get browser-automation-studio capture --json`. Setpoint run `prog_80219996-aa46-444f-8223-402648e118c4` returns1/17 in band,16 pending, product qualification false. The read-only `program-runtime.learning-maintain` list for BAS returned no findings. Contract preparation is valid (24 journeys, zero errors); inventory digest is `769b716ce412812059dc7269c5ec99156031f8de00d262c15e348b21f6f8137a`; evidence JSON and scoped diff check pass. Test Genie was not repeated: this cycle touched only the writer policy and its direct owner package/race tests pass; last unit receipt still records the separate driver stream flake/UI coverage failure. Saved profile continuity was not re-read in this cycle and remains unqualified under existing rows.

Next: generate concurrent16.384MP screenshots through the managed API+driver path, measure synchronized service PSS and request latency, and exercise PNG/JPEG images from viewport through the20.1MP estimate boundary. Verify stored-byte hashes plus malformed/truncated failures; retain an explicit failure if the page exceeds the budget. Keep the active-vs-idle metric distinction and recheck saved-profile identity before any further restart.

### BAS-WORK-108 — Restart baseline and protected-profile continuity

The cycle107 post-burst window was short. Before restarting, captured the only listed session profile metadata and hashes for the profile JSON, keyring witness and lock file without printing protected state. `make -C scenarios/browser-automation-studio restart` completed healthy; the profile ID/name/timestamps and all three file hashes matched afterward. The first61-second API+driver sample varied from223,976 to385,171KiB and ended with six identical296,204KiB readings; a separate31-second quiet sample ranged186,057–186,389KiB. This supports a settled Linux idle observation below300MiB, while the earlier same-process post-burst return remains unknown because the restart reset that state. It does not establish CPU idle, Windows or broader resource-budget completion. Full identity and samples: `internal/evidence/rehabilitation/screenshot-decode-managed-fullpage-2026-09-23.json`.

### BAS-WORK-109 — Same-process post-burst API residency and JPEG header gate

With the restarted process settled, ten concurrent managed full-page PNG executions completed. Pre-run synchronized API+driver PSS was151,149KiB; the20ms cohort peak was414,299KiB. All ten stored PNGs match at2560x6400,71,139 bytes and SHA256 `7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`. The next61 samples, one per second, ranged348,154–410,326KiB combined. At the final sample, API is256,830KiB versus58,712KiB pre-run, while driver is91,947KiB versus92,437KiB pre-run. That localizes retained residency to the API but cannot distinguish live heap from Go heap pages that remain resident; no heap/profile endpoint exists in BAS. Do not add a forced global collection based on this observation alone. A test-only JPEG SOF-header mutation exercises the current rejection order; `go test ./automation/execution-writer -run '^TestScreenshotJPEGRasterOverBudgetFailsBeforeFullDecode$' -count=1` and the full package pass. A valid full-size JPEG and retained-heap mechanism remain unqualified. Inventory/setpoint/contract and scoped diff checks are rerun after the entry is finalized.

Next discriminator: trace the FileWriter decode lifecycle and production Go runtime configuration for a bounded-ownership change that can release large raster memory without unconditional global GC. Keep the managed API+driver under the active 10-capture workload and its60-second post-burst series as the acceptance test; preserve one-pass JPEG/PNG receipt behavior.

### BAS-WORK-110 — Go soft heap-limit candidate measured in isolation

The managed same-process replay localizes the retained residency to the API. Source/config inspection finds no runtime heap limit or service-owned reclaim path. The existing retained direct-writer probe now has an optional `--gc-reclaim` mode: ten writers without forced cleanup peak149,267KiB, ordinary `runtime.GC` reduces it to139,463KiB, then `debug.FreeOSMemory` reduces it to20,575KiB in6ms. This is test-harness evidence only; no production GC was added. With `GOMEMLIMIT=96MiB`, two ten-writer runs peak105,435/106,191KiB at735/811ms, compared with default149,303KiB/627ms on the immediately adjacent control. `GOMEMLIMIT=128MiB` yields146,563KiB/664ms;64MiB yields104,231KiB/849ms. The96MiB setting is the candidate because it achieved nearly all measured memory reduction at less direct-writer slowdown than64MiB. These are isolated FileWriter process PSS results, not managed service or standard-cohort qualification.

Before touching the service env, the hypothesis, owner and rollback criterion are recorded above. Next mutate only the BAS API process env through `.vrooli/service.json`, restart with `make`, verify the effective setting and profile identity, run the same ten-case managed full-page fixture with synchronized PSS and60-second recovery, and rerun the maintained100-capture workload. Preserve the current config value if any managed comparison is inconclusive; record measured regressions as well as improvements.

### BAS-WORK-107 — Managed full-page writer stress and post-burst retention signal

The previous checkpoint's managed PSS unknown was falsifiable through `workflows execute-adhoc`: this owner executes a supplied definition without persisting it as a saved workflow and records artifacts through the production writer. The first65.5MP screenshot and the second32.8MP screenshot were each rejected before full raster allocation, confirming the new estimate guard on the managed path. A16.384MP output at DPR2 then passed; CLI execution status completed and the durable screenshot artifact reports2560x6400,71,139 bytes, SHA256 `7179c42bd9b4a30a39a1e70236b273ce6a70ef93869068e7a59d697a53c33fd0`.

Ten concurrent adhoc executions on managed build `sha256:6aa67b6b0db708dc1807d412458ba236e654f2677ff19ed82873662630f41180` all completed in5,544.5ms. Each saved full-page artifact has the same dimensions, size and SHA256; all ten execution IDs are retained in `screenshot-decode-managed-fullpage-2026-09-23.json`. API+Node-driver PSS sampled every20ms peaked390,105KiB at one tick (API252,876KiB + driver137,229KiB). The first sample arrived during request startup and is not a pre-run idle baseline. Beginning about6 seconds after the burst,21 samples at500ms intervals over10 seconds ranged316,676–321,564KiB (about309–314MiB), numerically above the contract's300MiB idle target. This is a short post-burst retention signal only: it does not establish steady state, recovery time, or a leak. It is not an active-memory cap violation. No code changed this cycle; the ten-byte admission owner and both `FileWriter` call sites remain unchanged. No complexity, duplicate-policy or coupling reduction is claimed.

The first synthetic managed page was CSS1280x12800 and rendered2560x25600 at DPR2; the second CSS1280x6400 rendered2560x12800. Both failed closed with `screenshot raster ... exceeds active screenshot decode budget of 201326592 bytes`. The passing synthetic page used CSS1280x3200 and rendered2560x6400, exactly16,384,000 pixels. No saved workflow definition was changed. New output is retained in BAS execution storage; it is not deleted during this review. Existing profile state was not re-read, and no service restart was performed.

Focused source tests from cycle106 remain applicable because this cycle made no runtime changes. Contract/setpoint/inventory and scoped `git diff --check` are rerun after this record is finalized. Test Genie was not repeated: its most recent unit run remains failed/degraded for separate driver stream and UI coverage findings; this managed probe is direct product evidence, not a generic unit verdict. Next discriminator: restart only after reading and confirming the protected profile identity, then capture a pre-run PSS baseline, replay one ten-run cohort, and sample at least60 seconds after it quiesces. Add an equally sized valid JPEG and a malformed/truncated JPEG case; verify artifact hashes and failure state. Keep unavailable Windows, object-store backpressure and above20.1MP pages explicit.

### BAS-WORK-113 — 2026-09-23 UTC — unit coverage gate failure isolated

BAS-RF-014 / J24 validation infrastructure. Hypothesis: the new unit-phase
terminal failure is a runner outage or RF121 source regression. The Test Genie
run `20260923-233436-bfac32e7` fails after504s with1 error,82 warnings and0
observations at `ui: pnpm run test:coverage`. Its retained findings artifact
contains no command output. A direct command reproduction in the UI workspace exits
1 at the merged coverage gate: statements/lines30.52%, functions33.61%,
branches67.26%, below the existing85% minimum. Displayed projects report passing
suites through the final shared project; the terminal error is the coverage floor,
not an unavailable runner. The assertion/threshold was not changed.

No production source changed in this investigation. Direct UI coverage produced
no durable report beyond command output; key result is copied to the RF119 evidence
JSON and BAS-RF-014 remains open. Test Genie performance on the same candidate
passes (`20260923-234725-be3fdcdd`,61s,0 observations). Unit and performance are
separate results; no generic unit pass is claimed. Next: inspect current coverage
summary by behavior owner, select one high-consequence under-covered UI seam, and
add expected-behavior tests while preserving the85% floor. Then repair whatever
owner implementation the regression exposes.

### BAS-WORK-114 — 2026-09-23 UTC — export dialog behavior coverage

BAS-RF-014 / export workflow. The coverage report showed the 615-statement
`ExportDialog` at0.49% statement coverage with no function coverage. Added two
user-facing behavior checks around the dialog's export gate: video export stays
disabled when an execution has no replay frames, while raw JSON package export
remains available because it does not require video rendering. These checks
exercise expected format behavior rather than altering or relaxing coverage
policy. Focused Vitest project run passes2/2; ESLint for the new test and full
UI TypeScript check pass. The aggregate UI coverage gate remains below its
unchanged85% floor, and the unit phase is still unqualified. Evidence:
`internal/evidence/rehabilitation/export-dialog-coverage-2026-09-23.json`.

Contract preparation remains valid (24 preservation journeys, zero errors),
which is consistency evidence only. Source inventory including untracked files
observed digest
`851583a4d37a1ac20a5f41ec34f3e3813026762f7f233bf65d9162e4e2fa1dc9`.
Governed setpoint read `prog_b3af2460-f813-46b3-b3b2-08ba4958f5fd` still
reports capture1/17 in-band,16 pending telemetry and product qualification
false; the current receipt is unchanged from the preceding cycle.

The immediately previous unit result remains terminal-failed on the aggregate
coverage threshold, not a runner outage; performance remains independently
passed. No production behavior changed, so no managed restart or new workload
capture was warranted. RF-014 remains open. Next: continue with another
consequential UI behavior owner, then return to CPU/storage backpressure and
retain Windows/macOS and long-soak gaps as unknown.

### BAS-WORK-115 — 2026-09-24 UTC — managed CPU and storage observations

BAS-RF-119 / resource-budget. Reused the current managed build and declared
Performance Health `capture` workload. A60-second idle process sampler reads
API+driver PSS, CPU ticks and `/proc/<pid>/io`; then one owner workload runs
100 measured captures plus its declared warmup while the same counters are
sampled. Idle CPU was0.10% of one core on average,1.0% p95 sample,1.98% peak;
idle PSS ranged136,090–196,596KiB and ended136,158KiB. The owner workload
passed at425ms service p95/608.321ms wall p95. During its56.437-second window,
combined API+driver CPU averaged14.54% of one core (36.73% p95 sample,
109.41% peak), combined PSS peaked190,378KiB, and API `write_bytes` increased
124,698,624bytes. This workload uses the fixed viewport fixture.

Separately ran ten managed adhoc full-page screenshots at DPR2: five PNG and
five quality80 JPEG. All executions completed and all20 resulting screenshot
objects were read back; the ten full-page objects have their expected2560x6400
dimensions, `.png`/`image/png` or `.jpg`/`image/jpeg` identity, and one
consistent hash within each image type. Over4.043seconds, combined API+driver
PSS peaked288,438KiB, CPU averaged55.97% of one core (250.05% p95 sample,
657.67% peak), and API `write_bytes` increased4,374,528bytes. Host-wide I/O
PSI increased113,816us (`some`) and91,652us (`full`), compared with5,533,380us
and5,315,430us respectively across the59.316-second no-workload window. No
increased host-pressure signal appears in this cohort, but PSI cannot attribute
I/O stalls to BAS. Exact FileStorage write latency remains unmeasured: current
workload receipt has no store-span metric, ptrace attachment is denied, and
kernel task delay accounting is disabled. Do not mark storage backpressure
qualified from these counters.

The fixed viewport workload receipt is operation
`6de6fbe26ed3ac95e80da95bfbdb27f7`, on the unchanged build
`sha256:3e81cd3eecfb96b02422eda46141cfab78209c1a47adbc31627aa2e08e1fb9f9`;
receipt SHA256 is `6b1d0adcb58dcb4e07305088403d86dedb62757430fb278f4b82a3714b45cb37`.
Contract preparation is valid (24 preservation journeys, zero errors),
consistency-only. Governed setpoint `prog_cfdcb5f3-760e-4a52-a42f-e918a34d0303`
binds the latest capture receipt at425ms p95,1/17 in-band and16 unavailable;
product qualification remains false. Inventory with untracked sources observed
digest `851583a4d37a1ac20a5f41ec34f3e3813026762f7f233bf65d9162e4e2fa1dc9`.
Evidence:
`internal/evidence/rehabilitation/resource-cpu-storage-2026-09-24.json`.

Next work should add owner-level storage span telemetry or a controlled
backpressure producer that can identify FileStorage wait time and cancellation
behavior. Preserve local filesystem findings separately from MinIO/object-store
behavior; Windows/macOS and long-soak measurements remain open.

### BAS-WORK-116 — 2026-09-24 UTC — owner-reported screenshot storage span

BAS-RF-119. Measured storage duration with a monotonic clock around only
`StorageInterface.StoreScreenshot` and persisted the value in screenshot
artifact metadata as integer nanoseconds. Validation/decoding and the subsequent
artifact projection are excluded. A
controlled storage adapter blocks for30ms; `TestRecordStepOutcomeReportsScreenshotStorageWait`
reported30.472192ms, proving the measured span includes synchronous wait at the
storage owner seam. Focused writer and executor package tests pass. This is a
maintained producer test, not a production observation at that stage. No
contract or policy changed. The change adds one artifact metadata key and one
regression test; net debt/complexity reduction is not claimed.

Contract preparation is rerun below, with24 preservation journeys and zero
errors; this is consistency evidence only. The governed rehabilitation board
`prog_55678395-d686-4124-8750-6a6492a323bd` reports1/17 readable and product
qualification false. Inventory including untracked sources observed digest
`4717f43ae506739ab8116c75cdbf39a2eb8b9452c065a57216af7c70ef50bf47`; scoped
diff check passes. Before a managed restart, the read-only CLI confirmed the
preserved profile `305e1254-28fc-4bfb-8774-a46aa0bc0d1d` / `channel-manager-synthetic-p1`,
matching the retained checkpoint identity. The first step-event-only build
passed100+1 at452ms p95 but the workload receipt did not retain per-step spans;
the span was therefore added to the durable screenshot artifact payload before
the second managed restart. Build `sha256:db55a6b0712c33a2ad2c6484968b9c25247466ec6f5ae4ae2e43f65f578f4365`
is healthy. Its declared100+1 capture workload passed at421ms service/636.983ms
wall p95 (`53effc0eb113d0168a69df0318c88217`). A managed 10-execution synthetic
full-page PNG cohort completed10/10; all ten2560x6400 artifacts were62,022 bytes.
Their persisted `FileStorage` spans ranged105,160–3,732,705ns, median129,735.5ns,
nearest-rank p95 3,732,705ns. The declared capture workload's latest10 viewport
artifacts measured115,201–217,030ns (14.7KB each). Both the owner test and the
managed `executions timeline` reader confirm spans survive in screenshot
artifact metadata. This is direct local-FileStorage evidence on one Linux host;
no storage threshold was amended and no active test or backpressure claim is
qualified beyond these workloads. Required screenshot write failures suppress
the `RecordResult` receipt and currently have no durable span; cancellation and
error timing remain open. Next: compare these spans against a larger write
cohort and read MinIO behavior through its owner. Windows/macOS and long soak
remain unknown.

#### BAS-WORK-116 follow-up — durable managed span measurement

The initial event-payload approach was dropped after tracing the production
WebSocket adapter, which projects events to protobuf timelines and discards the
extra field. The durable screenshot artifact metadata is available through
historical `executions timeline`. The managed
100+1 run receipt is operation `53effc0eb113d0168a69df0318c88217`, SHA256
`bb69463bff852236f8ca4df12b56582eb17b223ec4baa2efeb3d292ab9715e7a`, on the
build above. The earlier event-only deployment's capture p95 was452ms; the
durable-field build's p95 is421ms, both within2,000ms. One pair does not establish
causality. Concurrent synthetic data URL captures generated transient
workflows only; no workflow or user profile was modified. Their complete ten
execution IDs and spans are in the evidence JSON.

### BAS-WORK-117 — 2026-09-24 UTC — retain failed screenshot storage spans

BAS-RF-119 / J08 evidence retention. A storage span existed only when a screenshot
artifact receipt existed; required store errors suppressed `RecordResult` and
therefore erased the only timing measurement even though the failed outcome was
still written. The writer now records positive `StoreScreenshot` duration in
`StepOutcome.Notes["screenshot_storage_duration_ns"]` when the store call or its
receipt validation fails. Successes retain the existing screenshot-artifact
payload field, avoiding duplicate metadata. The failure remains unsuccessful,
returns its persistence error and supplies no completed `RecordResult`.

The maintained failure test injects a delayed storage error and a delayed
`context.Canceled` return; both retain timing alongside the persistence error in
the durable step-outcome artifact. Focused writer and executor package tests
pass. This is a controlled adapter regression, not a managed storage-fault or
external context-cancel experiment. It adds one metadata field on failed
outcomes and one test; no debt/complexity reduction is claimed.

#### BAS-WORK-117 follow-up — managed deployment and gates

Managed restart is healthy on build
`sha256:484089a05a799a8b78a9995fac6cc43c8f23d58d11f646a099580e8ae9ddd006`.
The protected profile `305e1254-28fc-4bfb-8774-a46aa0bc0d1d` /
`channel-manager-synthetic-p1` retained its created, updated and last-used
timestamps across restart. The fresh declared capture workload passed100+1 at
444ms service p95 and617.836227ms wall p95; receipt operation
`ab699d199e203075055b3fa2caa2a23b`, SHA256
`0fe299c1f886ae43d6085fcebb5f7253edbb08809778a3638d3d640cf0797aec`. Contract
preparation is valid with24 preservation journeys and zero errors. Governed
setpoint run `prog_70c9d848-4ba5-4926-9547-1b45688e4c06` reports capture1/17
in band,16 pending, product qualification false. Inventory with untracked
sources observed digest
`c4eb8566938a7bd02b9e0517340bd05c0df3fdb5c8e456621cacca09fb676c81`; scoped
diff check passes. Required lifecycle and capture checks are complete. MinIO,
real caller cancellation, Windows/macOS and long-soak behavior remain
unqualified.

### BAS-WORK-118 — 2026-09-24 UTC — preserve replacement app shortcuts

BAS-RF-122 / BAS-RF-015. Candidate identity is repository HEAD
`8707b970a48f82665662450766ad8233d6def9e7`, with source inventory digest
`c808568be146652ebd3c4a090c3f6ac8dcec864bf7bf502c4e2e4cf76d93053b` after the
code change. The caller trace showed `RootLayout` calls `useAppShortcuts` with
callbacks recreated during render. `useAppShortcuts` builds a new action map; its
prior `useRegisterShortcuts` cleanup queued only shortcut IDs, while immediate
registration updated the existing map in place.

`src/shared/hooks/useAppShortcuts.test.tsx` first failed before the fix: after a
view/callback replacement, dispatching the real global `Shift+?` event called the
replacement `openDocs` handler zero times. `unregisterAction` now removes only
the expected registered callback when it is still current. Each hook effect
installs a unique wrapper around its application callback, so a fresh map that
reuses the same `undo`/`redo` function still receives a new registration
identity. The maintained tests pass both changed-callback and same-callback
keyboard events, and verify unmount releases each registration.

The module search found no BAS callers of the deprecated no-op `useKeyboardShortcuts`
hook, its default export, the `KeyboardShortcut` compatibility type, or
`formatShortcutLegacy`; the UI package is private. Those old surfaces were
deleted, not redirected. The two runtime modules fell from320+485 to286+456
physical lines (minus63 total); the new86-line test is test code, not runtime.
An identity comparison branch and one wrapper closure per registration prevent
stale cleanup. This is a real
local removal of dead API and duplicate formatting policy; no domain-wide
complexity, duplication or coupling reduction is claimed.

Focused regression passed with
`pnpm exec vitest run --project shared src/shared/hooks/useAppShortcuts.test.tsx
--coverage=false --reporter=verbose` (2/2); UI TypeScript and targeted ESLint
passed. Test Genie unit baseline `20260924-010206-011c0204` failed after491s at
the unchanged85% UI merged coverage floor (30.52% statements/lines,33.61%
functions,67.26% branches;79 `LOW_COVERAGE` files). The post-change scoped run
`20260924-011554-4f065533` failed after475s with the same coverage floor plus
`UNIT_POLICY_PROJECTION_DRIFT` because its first version imported Testing
Library `render` directly. The test now uses canonical `renderWithProviders`.
Final scoped unit run `20260924-012813-5f34d6da` completed failed after522s
with one `TEST_EXECUTION_FAILURE`,82 warnings and zero observations. Its command
log confirms `pnpm run test:coverage` failed at the unchanged85% UI merged
coverage floor:31.4% aggregate coverage and79 `LOW_COVERAGE` files. There is no
remaining policy projection drift; the focused shortcut test is green. Contract
preparation is valid
(24 journeys, zero errors). The governed rehabilitation read
`prog_93f97a06-3c63-4584-bd09-de6df9d3700f` reports capture1/17 in band at444ms
service/617.836227ms wall p95 and product qualification false. Inventory
includes other shared-tree changes; final source digest is
`c808568be146652ebd3c4a090c3f6ac8dcec864bf7bf502c4e2e4cf76d93053b`; UI reports
130,198 runtime lines. This is the selected shared-tree population, not an
isolated commit.

This test is a jsdom-level app-hook event reproduction; a packaged/native
browser matrix is not claimed. RF-014 remains open because two regressions do not
close the aggregate UI coverage deficit. Next: select the next consequential
under-covered UI behavior without weakening the85% floor. The governed run
includes both registration cases and the canonical helper. No service restart
was performed; managed product behavior stays unqualified.

### BAS-WORK-119 — 2026-09-24 UTC — execution export gates and request behavior

BAS-RF-014. The unit findings repeatedly identified
`domains/executions/viewer/useExecutionExport.ts` as a low-coverage user-facing
owner. Added `useExecutionExport.test.tsx` with expected-behavior checks: opening
without replay frames keeps the dialog closed; a blank output directory prevents
the server call; a configured directory submits an MP4 request with the
sanitized execution-based filename and selected output directory. The test
exercises the real app hook while substituting preview, video-status and progress
transport seams. Added the file to the existing `execution-viewer` project so
the ordinary scoped suite discovers it.

Focused Vitest passes2/2 without React warnings. UI TypeScript and targeted
ESLint pass. Scoped Test Genie unit run `20260924-014452-cc414191` completed
failed after551s with one `TEST_EXECUTION_FAILURE`,82 warnings,79
`LOW_COVERAGE` files and zero observations. The retained command log reports
31.4% aggregate UI coverage against the unchanged85% floor; there is no
`UNIT_POLICY_PROJECTION_DRIFT`. The new test is policy-clean, but the owner unit
phase does not pass. No production runtime source changed, no managed restart or
product qualification was performed, and no complexity reduction is claimed.

Contract preparation remains valid (24 preservation journeys, zero errors).
The include-untracked source inventory digest is
`83b0fe615ebdc2027fa9e6dd1834cb945ccbff325f32f8f24e8a2c91e3868ccd` at HEAD
`8707b970a48f82665662450766ad8233d6def9e7`; UI runtime remains130,198 lines.
This is the selected shared-tree population, including unrelated pending edits,
not an isolated commit. Next, continue with a consequential under-covered UI
owner while preserving the85% floor.

### BAS-WORK-120 — 2026-09-24 UTC — passive recorder secret capture boundary

BAS-RF-017. Hypothesis: the recorder transports secrets because `handleInput`
buffers `target.value`, `getVisibleText` reads the live password property, and
`getRelevantAttributes` emits the password value attribute. The separate
credential-use policy only validates protected workflow actions and explicitly
keeps secret input out of ordinary actions; it does not sanitize passive events.

Added a real Chromium regression with separate synthetic sentinels in a typed
password, password `value` attribute, and `autocomplete=one-time-code` text
field. Before the source change it failed and exposed the typed sentinel in the
recorded `type` payload and `elementMeta.innerText`. After repair,
`isSensitiveInput` classifies password/hidden controls and password, OTP and
payment autocomplete tokens. Sensitive values are not buffered/flushed; metadata
suppresses live text, value attributes and all `data-*` attributes for those
controls. The ordinary text-field input capture test remains green. A second
guard in `rawBrowserEventToTimelineEntry` strips sensitive payload values and
metadata before canonical timeline buffering or callback, covering stale or
alternate injectors. Chromium tests inspect the driver-received raw event objects
and browser pending-event journal; all three synthetic sentinels are absent. Four
converter tests feed deliberately unsafe raw events and confirm password, hidden,
OTP redaction plus ordinary text preservation. Selector/action metadata remains
available.

Validation: `pnpm exec jest tests/integration/recording-injection.test.ts
tests/unit/proto/recording-redaction.test.ts --runInBand --coverage=false` passes
17/17; driver `pnpm run typecheck`, focused ESLint, Prettier and `git diff --check`
pass. An earlier coverage-enabled narrow run passed all 13 browser tests but
reported the unrelated global driver threshold at 13.79% functions against 15%.
`python3 docs/internal/refactor_contract.py`
passes (24 journeys, 0 errors); setpoint read
`prog_3d6bf545-fc2b-425c-b63d-7bda8e599854` returns 1/17 in band, 16 pending,
product false. Contract preparation remains valid (24 journeys, 0 errors).
Include-untracked inventory at HEAD `8707b970a48f82665662450766ad8233d6def9e7`
includes pre-existing shared worktree changes; digest
`bf7c6ee5c2ff634436b38b18079d1cd03e3a06eefdeeace1f7017ef3e215f482`; UI and
driver runtime lines are130,198 and52,440. No managed
restart or deployment was performed; actual Chromium loads the edited capture
script directly.

The fix prevents new supported passive captures and canonical timeline entries
from storing these values, but does not scrub recordings saved before the fix.
Export, screenshot, AI attachment and stored-record consumers have not been
independently fed synthetic historical data, and the credential-use path has not been exercised
end to end. Those are explicit RF-017 closure requirements; recheck them before
resolving the issue. The change adds one classifier and a small token list; no
complexity reduction is claimed. Next: qualify historical/downstream consumers
with synthetic-only fixtures, then continue to the next highest-impact open
defect.

Full owner receipt: `internal/evidence/rehabilitation/secret-capture-boundary-120-2026-09-24.json`.

### BAS-WORK-121 — 2026-09-24 UTC — API recording ingress and legacy-reader boundary

Scope extension for BAS-RF-017: source tracing found a second supported ingress
in `api/handlers/record_mode_actions.go`, which accepts legacy `RecordedAction`
JSON as well as typed `TimelineEntry` JSON, and durably commits through
`api/services/recording.Service.RecordAction`. It broadcasts the resulting
action after commit. The live timeline GET path returns persisted actions, so
legacy rows can reach clients even though new driver events are now redacted.
This is a necessary BAS-facing extension: owner is API recorded-action
normalization and live-capture reads; planned paths are
`api/automation/driver`, `api/services/recording`, and
`api/services/live-capture`. Preserve on-disk legacy values pending an explicit
safe migration policy. Planned checks: owner Go tests proving redaction occurs
before callback/persistence and before API response projection, with ordinary
text preserved; package races if relevant; keep export/screenshot/AI attachment
readers separately unqualified until tested.

Implementation: `driver.RedactSensitiveValues` is the API's single field
classifier. `ReceiveRecordingAction` applies it before journal append and
broadcast, while `recording.Service.RecordAction` enforces it for all direct
callers. `recording.Service.GetTimeline` clones only the action/maps it must
redact, then classifies the returned copy; the mock repository regression proves
the stored legacy row still retains its original synthetic bytes. The driver
client also sanitizes typed buffered entries before converting them to legacy
actions or returning the raw `entries` JSON field; this keeps both forms safe for
the API's `/actions` response and generated workflow path. Old database values
remain at rest; normal API reads no longer disclose them. No `live-capture`
source edit was needed because it projects the recording service's sanitized
result.

Validation: `go test ./automation/driver ./services/recording ./handlers` passes;
the new tests cover password/hidden/OTP/payment classification, ordinary-text
preservation, a deliberately unsafe driver timeline response, API ingress before
persist/broadcast, and read redaction without modifying saved data. Contract
preparation remains valid (24 journeys, zero errors). The unit Test Genie run
`20260924-023054-162f2f31` is pending its one server-owned wait; the source tree
changed during that run only to retain blank raw entries in the driver's response
while sanitizing the rest. Final-source package tests were rerun after that edit.
The pre-AI-extension inventory at HEAD `8707b970a48f82665662450766ad8233d6def9e7`
had digest `26b0d5f6bf88156fb41f4415d56436b61623aa98c9a7d96f3a29435281e59d18`;
it is not a digest of the completed W121 tree. At that point API runtime was
92,386 lines (+155 against W120's92,231) and driver runtime was52,440 lines.
This is an additive security boundary; no complexity reduction is claimed. No managed
restart was performed, and at-rest legacy values remain unqualified for direct
storage exposure or cleanup. AI element extraction did not read control values,
but it included `data-*` selector values and sent a full-page screenshot. The
source now omits those selectors for password/hidden/OTP/payment-autocomplete
controls and masks their rendered text before the screenshot instruction.
`go test ./handlers/ai` passes, and a synthetic Node DOM harness verifies
one-time-code data selectors are omitted and screenshot masking styles are
applied. This remains source-level evidence; there is no managed synthetic
screenshot fixture. A combined final owner run of
`go test ./automation/driver ./services/recording ./handlers ./handlers/ai`
passes. Scoped `git diff --check` passes.

The single Test Genie unit run `20260924-023054-162f2f31` completed failed
after514s: `pnpm run test:coverage` in the UI workspace failed, with one error
and82 warnings in its finding summary. Direct changed-owner Go packages pass;
the AI extraction update was made after this Test Genie run and was separately
checked by its Go tests and synthetic Node harness. The UI coverage floor
remains an independent existing blocker. The governed
setpoint refresh `prog_c921d132-5f53-417a-902c-281114dfdb6e` returns capture
1/17 in band (p95 444ms, wall p95 617.836227ms, build
`sha256:484089a05a799a8b78a9995fac6cc43c8f23d58d11f646a099580e8ae9ddd006`),
with16 rows pending telemetry and product qualification false. No managed
restart or deployment was performed. Full result receipt:
`internal/evidence/rehabilitation/secret-capture-boundary-121-2026-09-24.json`.

Historical at-rest handling, export paths, credential-use behavior, and managed
AI screenshot behavior remain unqualified. RF-017 stays open. No complexity
reduction is claimed.

### BAS-WORK-122 — 2026-09-24 UTC — browser-pool launch ownership race

The current required regression battery reproduced RF-006 in the actual
`playwright-driver/src/session/browser-pool.ts`: one failed launch wave produced
three launch attempts and two successful browser IDs while only one was closed;
a browser resolving after `closeAll` remained live. Prior-art retrieval ran
`search-hub query "BAS browser pool launch retry shutdown orphan browser" --type
record,doc`; top-ranked results were weak historical BAS sessions rather than a
direct pool repair. The dated assessment and initial retained reproduction
already document the same failure class (`REFRACTOR_ASSESSMENT.md` §RF-006 and
`REFRACTOR_PROBES_2026-09-22.json`), so this is a confirmed recurrence.

Hypotheses before the regression: (1) waiting callers swallow the shared launch
rejection and each starts a replacement; prediction: concurrent waiters all
reject from one launch, while a later independent call can retry once. (2)
shutdown enumerates completed browsers but not launch promises; prediction: a
deferred launch is awaited and closed before shutdown completes, and its pending
caller does not receive a browser that shutdown is closing. The contrasting
possibility that a launch resolves before shutdown's snapshot is covered by the
existing completed-browser close path. Scope is limited to the BAS driver pool,
its maintained Jest tests, and owner architecture/evidence; no lifecycle restart
or cross-scenario edit is authorized or needed.

Architecture cross-check (`docs/concepts/ARCHITECTURE.md`, resource-ownership
invariant) narrows the retry contract: callers in the failed launch wave must
share one bounded replacement attempt, rather than all failing or launching
independently. The first test draft that propagated the initial error to every
waiter was therefore rejected before acceptance. Final regression expectation:
all concurrent waiters receive the same successful single retry; a later call
can still retry after both attempts fail.

This cycle also qualified the RF-017 AI screenshot edge in managed Chromium.
Two extractions used different synthetic OTP and `data-token` values with an
unchanged visual fixture; generated selectors omit the data values and decoded
PNG pixels compare identical. The direct managed-driver integration test passes
and its temporary sessions are cleaned up. This covers the tested OTP/data
boundary only; payment variants, derived attachments, history, export and
credential-use remain open. Receipt:
`internal/evidence/rehabilitation/secret-capture-boundary-122-2026-09-24.json`.

### BAS-WORK-125 — 2026-09-24 UTC — screenshot raster-budget boundary regression

BAS-FB-016 requests focused owner checks. RF-119 already has a process-wide
weighted decode-admission implementation backed by a measured full-page cohort,
but the maintained budget tests did not directly assert its dimension arithmetic.
The falsifiable check was the boundary: maximum admitted raster estimate passes;
one pixel above, huge positive dimensions and any nonpositive dimension reject.

Added `TestScreenshotDecodeWeightEnforcesRasterBudget` in
`api/automation/execution-writer/screenshot_decode_budget_test.go`. The direct
owner command
`go test ./automation/execution-writer -run 'TestScreenshotDecode(WeightEnforcesRasterBudget|BudgetWaitsForWeightedCapacity)$' -count=1`
passes2/2 in0.048s; `gofmt` and scoped `git diff --check` pass. Contract
preparation passes (24 journeys,0 errors). The required rehabilitation setpoint
`prog_9af1bd21-3aa0-4a03-adb1-8216695ad273` remains1/17 in band,16 pending,
product qualification false. No Test Genie run was started.

This is a guard-arithmetic regression only; it does not establish a hard RSS cap,
steady-state heap recovery, managed hostile-dimension behavior across formats,
or Windows/macOS backpressure. No production implementation changed and no
complexity reduction is claimed. Current include-untracked source inventory
digest is `3d08c0dfa95e42d08df5b4d863d4421e38fad98e8803e356e2250b418a23c790`
at shared HEAD `8707b970a48f82665662450766ad8233d6def9e7`; domain runtime lines
remain API92,411, driver52,482 and UI130,198. Existing unrelated worktree
changes were preserved. Next action remains the highest score-moving gap:
integrate the profile owner receipt into a governed read path without accepting
caller-supplied pass data, then strengthen interruption/repeatability evidence.

The maintained managed cohort is
`api/cmd/profile-durability-cohort/qualification.mjs`. Seed stage used synthetic
alpha/beta profile identities and a fixture that writes the three supported
stores, then polled the owner profile list every50ms until the automatic
checkpoint appeared or5000ms elapsed. First positive observation was
1170.868017ms after the fixture acknowledged its completed writes (23 reads).
Both identities passed close/reopen and beta did not change alpha's state. After
`make restart` completed healthy on the newly built
`sha256:feb8fcb826bb5102b950678eeb7dd49abc1e753b86581766efafe8900063a6c5`,
both identities again matched exactly. The two fixture profiles were deleted;
API health is healthy and the driver has zero sessions/recordings. The current
owner Jest integration suite `tests/integration/profile-storage-continuity.test.ts`
passes7/7 in6.721s. No Test Genie run was admitted. The 097 receipt's timing
claim is explicitly superseded; its 5/5 checkbox is not accepted as proof of the
five-second band.

The profile-durability row now has a reproducible current owner cohort and
retained receipt, but the governed setpoint reader still has only a capture
sensor, so the global reading remains1/17. Fresh capture workload operation
`94ef9d09a8bc6b04807c762d74d18fae` passed100+1 on the restarted candidate at
446ms service p95/647.963119ms wall p95 against2000ms. Final setpoint
`prog_c6b1e551-62f2-410f-9f87-ff179e847067` remains1/17 in band,16 pending,
product false. The row must remain pending until its receipt is joined to a
governed sensor; timing repeatability and abrupt process death remain
unverified. Final include-untracked inventory digest is
`dd132b4d13fc3af89f5b10aeaf743600b10d184aa22759e564adade1be01bb24` at
HEAD `8707b970a48f82665662450766ad8233d6def9e7`; runtime lines API92,411,
driver52,482, UI130,198. Contract validation passes (24 journeys, zero errors).
No Test Genie run was admitted. Receipt:
`internal/evidence/rehabilitation/profile-durability-124-2026-09-24.json`.

### BAS-WORK-124 — 2026-09-24 UTC — profile durability timing evidence audit

Operator feedback BAS-FB-016 directs targeted owner checks and better outcome
evidence. The profile-durability row is a practical next candidate because prior
managed probes already exercised cookie, localStorage, IndexedDB, close/reopen
and API/driver restart. Re-read the retained owner receipt
`internal/evidence/rehabilitation/periodic-checkpoint-097-2026-09-23.json` and
its source harness `/tmp/browser-automation-studio/profile-checkpoint-qualified-097.mjs`.
Prior-art query: `search-hub query "BAS session profile checkpoint recovery
window 5 seconds" --type record,doc`; ranking was noisy and yielded no stronger
timing producer than the retained BAS receipt.

Adversarial audit found a false-positive assertion: the producer sleeps exactly
5500ms, then checks only `hasStorageState == true`, while naming that result
“checkpoint retained within five seconds.” The retained `observed_after_ms` is
5504.810455 but `checkpoint_visible_within_five_seconds` is true. The second
pre-deployment probe has the same defect at 5507.680265ms. This invalidates the
prior claim that the five-second checkpoint bound passed; it does not prove that
the runtime misses the actual five-second limit. The timer started after the
fixture had finished its cookie/localStorage/IndexedDB write, so the desired
measurement is from that completed fixture acknowledgment until the owner can
read persisted state.

Competing hypotheses: (1) a bounded frequent poll will observe persisted state
before5000ms, confirming the test's late fixed delay—not runtime latency—caused
the false-positive; (2) the owner first exposes state after5000ms, making this
a real band failure concealed by the old test; (3) the checkpoint becomes
readable sooner but profile-list observation is stale, requiring an owner read
with a more direct state receipt. Discriminator: a maintained managed-API
probe records monotonic start/end for each read and treats any first positive
observation after5000ms as a failure. No runtime change is justified before
that observation. BAS remains healthy at API17116/driver24485, zero driver
sessions and recordings, build `sha256:484089a05a799a8b78a9995fac6cc43c8f23d58d11f646a099580e8ae9ddd006`.

### BAS-WORK-123 — 2026-09-24 UTC — network response request identity

The same controlled driver probe confirms RF-019: two distinct installed
Playwright `Request` objects both stringify as `[object Object]`; overlapping
responses for `/a` status201 and `/b` status202 yield only `/b` status201. A
sequential control preserves both. Prior assessment (`REFRACTOR_ASSESSMENT.md`
§RF-019) attributes the defect to correlation by string rather than object
identity. New hypotheses: (1) keying the pending-request map by the actual
`Request` object will keep overlapping response and failure attribution unique;
prediction: concurrent identical-method/URL and distinct-URL fixtures both
remain distinct. (2) If Playwright supplies a different request wrapper through
`Response.request()`, object identity will fail to join; test with the same
installed prototype and public collector event methods before changing code.
Scope: `playwright-driver/src/telemetry/collector.ts`, its maintained unit tests,
and relevant issue/evidence records. Preserve bounded age/capacity eviction and
response ordering; do not widen this into telemetry schema or resource policy.

The first maintainer test failed before the fix with exactly one event (`/b`,
status201) instead of `/a`201 and `/b`202. The collector now keys its existing
bounded pending map directly by `Request`, removes the false internal-string-ID
helper and unused fallback counter, and uses the same object for response and
failure removal. The maintained telemetry collector suite passes15/15. The
actual-module probe passes RF-019 and both RF-006 assertions after the pool fix.
The full legacy probe command still exits2 because its unrelated direct-frame
path was retired, its old input module no longer loads through its ts-node
loader, and the recorder probe reads an absent event; these producer errors are
unavailable evidence, not product passes or new source regressions. Recheck that
pack only after the three probes are replaced with maintained current-owner
tests. The controlled network failure and actual expected behavior remain
preserved in the reproduction output described above. Final focused command
`pnpm run typecheck && pnpm exec eslint src/session/browser-pool.ts
src/telemetry/collector.ts tests/unit/session/browser-pool.test.ts
tests/unit/session/browser-manager.test.ts
tests/unit/telemetry/collector.test.ts && pnpm exec prettier --check
src/session/browser-pool.ts src/telemetry/collector.ts
tests/unit/session/browser-pool.test.ts
tests/unit/session/browser-manager.test.ts
tests/unit/telemetry/collector.test.ts && pnpm exec jest
tests/unit/session/browser-pool.test.ts
tests/unit/session/browser-manager.test.ts
tests/unit/telemetry/collector.test.ts --runInBand --coverage=false`
passes typecheck, formatting, and 48/48 tests; ESLint exits successfully with
two warnings for existing test-helper return annotations. Contract preparation
passes (24 journeys, zero errors). Current inventory digest is
`565b5cbfc72bf8c895e8e9b5fea439c9ff29732820d9352bcf3d29c2f71cc60b` at HEAD
`8707b970a48f82665662450766ad8233d6def9e7`; runtime lines: API92,411,
driver52,482, UI130,198. Those size readings span a large shared worktree and
do not establish a complexity reduction. Setpoint read
`prog_cf9adc08-1e46-4ece-9ccc-a14651e214e4` returns capture1/17 in band
(444ms service p95, 617.836227ms wall p95), 16 pending telemetry, product
qualification false. The Test Genie waiter for
`20260924-025629-7e6ab613` ended unavailable with unexpected EOF; server run
state remains unknown and the run predates final pool retry and RF-019 changes.
No replacement whole-suite run was started under BAS-FB-016. Full receipt:
`internal/evidence/rehabilitation/driver-pool-network-2026-09-24.json`.

Following BAS-FB-016, the AI screenshot regression was expanded and rerun
directly against the managed driver. Password, OTP, credit-card number and
security-code values plus their `data-token` sentinels rotate between two
extractions; the hidden field is absent from extracted elements; all four
visible sensitive controls omit data-attribute selectors; and screenshot
pixels are identical. The first default-port invocation skipped because no
driver listened at39400. The second initial targeted assertion exposed that a
hidden control is intentionally absent from extraction, so it was corrected to
assert absence. The managed rerun at driver port24485 passes in5.318s; health
returns to zero sessions and zero recordings. This is targeted evidence, not a
Test Genie result. Updated receipt:
`internal/evidence/rehabilitation/secret-capture-boundary-122-2026-09-24.json`.

### BAS-WORK-126 — 2026-09-24 UTC — canceled profile snapshots at the commit boundary

Operator feedback BAS-FB-016 remains the active validation direction: use
focused owner tests for localized changes. The current RF-118 service already
fenced profile snapshots against clear/rebind during browser I/O, but one
plausible window was untested: request cancellation after the post-capture
context check while `PersistSessionState` waited to reacquire the binding mutex.
Prior-art query `search-hub query "BAS session binding invalidation late profile
snapshot commit profile ID ABA cancellation" --type record,doc --limit 4` was
degraded/noisy and did not return a direct RF-118 owner record; the existing
tracked RF-118 report and `TestRecordingProfileSnapshotRequiresCurrentBinding`
provided the actionable reproduction matrix.

Hypothesis 1: the only context checks before the registry lock allow a canceled
request to commit after lock contention. Prediction: hold the registry mutex
after capture returns, cancel once the second (post-capture) `Err` check is
observed, then release; the old implementation will return nil and change the
profile. Hypothesis 2: the earlier context check is a sufficient commit
linearization point, so cancellation after it does not invalidate the completed
snapshot. The minimal controlled interleaving discriminates the two.

Added `TestService_PersistSessionStateRechecksCancellationBeforeCommit` first;
it failed against the prior source exactly as predicted (`nil`, expected
`context.Canceled`). The session-profile service now rechecks `ctx.Err()` after
reacquiring the registry mutex and validating the same binding, before calling
the aggregate repository update. Focused race validation passes:

- `go test -race ./services/session-profile -run '^(TestService_PersistSessionStateRechecksCancellationBeforeCommit|TestService_ProfileCapturesSerializeAndRespectCancellation)$' -count=1`
- `go test -race ./handlers -run '^TestRecordingProfileSnapshotRequiresCurrentBinding$' -count=1`

The service regression verifies no stored profile change after cancellation;
the handler matrix verifies clear, same-profile ABA rebind, other-profile
rebind, and cancellation during browser capture all return non-success and
preserve both profiles. Contract preparation passes (24 journeys, zero errors).
Setpoint read `prog_831891ed-8a77-4dfb-8dfd-497410aabab3` remains1/17 in band,
16 pending and product qualification false; no Test Genie run was started.

This is a correctness guard, not a simplicity reduction: it adds one context
branch at the profile commit linearization point. API runtime source increases
by3 lines against the pre-change 92,411-line observation; other runtime lines
remain driver52,482/UI130,198. Current include-untracked inventory digest is
`56752432f7914eb2a3df4713786d47ecd7e4423184433c7d8cc6a20f59762933` at shared
HEAD `8707b970a48f82665662450766ad8233d6def9e7`. RF-011 checkpoint timing,
abrupt process death and repeatability remain unverified. The broader profile
durability row remains pending until a governed reader consumes applicable
owner evidence. Existing unrelated worktree changes were preserved.

### BAS-WORK-127 — 2026-09-24 UTC — targeted receipt route for profile durability

BAS-FB-016 remains the validation constraint: keep localized checks focused
and avoid broad Test Genie runs. The current setpoint has only the capture
workload binding for rehabilitation; its Test Genie validation-list binding
returns no BAS receipts. The owner profile cohort already supplies useful
behavioral evidence, but its `/tmp` files cannot be treated as a governed
sensor input, and the setpoint contract rejects caller-supplied receipts.

Read-only inspection of the Test Genie binding contract and validation broker
showed the narrow route: validation receipts are created by explicit intents,
the run producer passes declared phase names directly to the suite planner,
and the phase catalog is built from scenario-owned `.vrooli/test-genie.json`
descriptors. BAS currently has no such descriptor or focused profile phase.
Therefore wiring the existing local file into the setpoint would bypass
producer ownership, while starting generic `unit` or `workflow` phases would
repeat the expensive broad validation pattern the operator asked us to reduce.

Next bounded change: add a BAS-owned `profile-durability` validation phase
around the maintained synthetic cohort, produce a Test Genie-owned durable
receipt for that exact phase, then let the read-only setpoint consume only a
fresh applicable receipt. This requires BAS provider work plus the already
authorized Test Genie orchestrator boundary; it does not require a broad suite
run. No score change is claimed from this investigation: latest reading remains
1/17, with16 pending and product qualification false. No Test Genie execution
was started.

### BAS-WORK-128 — 2026-09-24 UTC — reproduce distinct-session admission race

The receipt-provider route remains a multi-owner evidence project, so this cycle
returns to the highest already-reproduced behavior defect rather than spending
another turn on score plumbing. BAS-RF-005's retained actual-manager probe
observed two fulfilled distinct-execution starts, two sessions and two browser
contexts at `maxConcurrent=1`; its same-execution control correctly coalesced
to one session. Current source still checks `sessions.size` before asynchronous
browser/context creation and inserts into the map later. The in-flight guard
is keyed by execution ID, so it does not itself serialize distinct starts.

Hypothesis 1: pending distinct starts are absent from the capacity count. If
true, hold the first start inside `Browser.newContext`, request a second start
with another execution ID, and observe that it also enters browser creation and
fulfills. Hypothesis 2: another shared admission layer serializes distinct
starts despite the per-key guard. If true, the same controlled interleaving
rejects the second start with `ResourceLimitError` before another context is
created. A separate failure-path control will reject the first context create,
then verify a later start can claim the released slot.

Owned boundary is limited to
`playwright-driver/src/session/manager.ts` and its maintained manager tests;
no shared service or test orchestration changes. Use the single manager Jest
file as the focused regression command. Test Genie remains unnecessary for
this local admission invariant. No implementation or score change is claimed
yet; latest score remains1/17.

### BAS-WORK-129 — 2026-09-24 UTC — reserve capacity during session creation

BAS-FB-016 remains active: prefer focused owner checks and avoid broad Test Genie
runs. The RF-005 reproduction confirmed that the manager admitted two distinct
execution IDs at `maxConcurrent=1` while both browser contexts were pending.
The first regression failed before the fix because the second start fulfilled.

`SessionManager` now reserves capacity synchronously after existing reuse and
same-execution coalescing paths, counts reserved starts with live sessions, and
releases its reservation in `finally` after either creation or failure. A
maintained failure-path test verifies a failed context creation frees capacity
for a later start. The direct real-Chromium integration test verifies same-ID
coalescing and that only one of two competing distinct IDs is admitted at a
capacity of one.

Focused validation passes: the manager unit file plus the Chromium integration
file pass 64/64 tests; `pnpm run typecheck` passes; ESLint exits with zero errors
and five existing warnings in `manager.ts`; the new integration test passes
Prettier; contract preparation passes (24 preservation journeys, zero errors).
The Jest process prints its existing open-handle warning despite a successful
exit. No Test Genie run was started. Evidence, including source hashes and the
fail-before/fix-after observation, is retained in
`internal/evidence/rehabilitation/session-admission-capacity-129-2026-09-24.json`.

This is a correctness repair and adds a reservation counter/helper; no
complexity, duplication, or coupling reduction is claimed. Request cancellation
is not wired into `SessionManager.start` and remains unverified. No managed
service restart was performed. The post-change setpoint read
`prog_14ec48d7-98ae-4c79-82d5-928d2aba4efd` remains 1/17 in band, 16 pending,
and product qualification false; this localized manager fix has not changed
that global score. The tracked RF-005 issue now reflects the repair and its
remaining cancellation gap. The next qualification opportunity remains the
profile-durability governed-reader route recorded in BAS-WORK-127; it requires
a narrowly scoped owner receipt producer and reader, not another broad suite
execution.

### BAS-WORK-130 — 2026-09-24 UTC — profile durability on the repaired candidate

BAS-FB-016 remains active. The profile-durability cohort passed on build
`feb8fcb826bb5102b950678eeb7dd49abc1e753b86581766efafe8900063a6c5`, but the
latest RF-005 session-manager source change postdates that running build. The
specific question is whether rebuilding the repaired candidate and exercising
the maintained alpha/beta cookie, localStorage, IndexedDB, close/reopen,
checkpoint-timing and API/driver-restart cohort preserves behavior.

Hypothesis: synchronous capacity reservation changes only admission accounting
and has no effect on one-at-a-time profile capture/restore. Prediction: after a
managed lifecycle restart, the maintained cohort's seed and restart verification
both pass, all synthetic state is cleaned, and BAS returns healthy. Discriminator:
`node cmd/profile-durability-cohort/qualification.mjs seed|verify` around one
`make restart`, with a fresh output directory and build identity. If it fails,
retain the observed fixture/API state and repair the owner path before retrying.
The owner boundary is the existing BAS lifecycle plus
`api/cmd/profile-durability-cohort/qualification.mjs`; no shared orchestration
or new Test Genie execution is authorized by this experiment. This check adds
no complexity-reduction claim.

The first lifecycle restart completed healthy after306s while ensuring all
declared dependencies; it rebuilt BAS to
`sha256:1fa3e575c0606946f56b2b0d5b3577776c7e01cdf4dd71408808126736070fe1`.
The seed cohort passed all five checks. Persisted state became visible
1115.704074ms after the committed fixture writes, within the5000ms band. Alpha
survived close/reopen and beta activity without state crossover. After seeding,
a second managed `make restart` completed healthy in44s with the same build;
both alpha and beta cookie/localStorage/IndexedDB identities then restored
correctly. The harness deleted both synthetic profiles. Post-cleanup driver
health is `ok`, ready, zero sessions, zero recordings; API health is healthy.
Both complete owner receipts are retained at
`internal/evidence/rehabilitation/profile-durability-130-seed-owner.json` and
`internal/evidence/rehabilitation/profile-durability-130-restart-owner.json`.
The focused
`pnpm exec jest tests/integration/profile-storage-continuity.test.ts --runInBand --coverage=false`
passes7/7 in5.643s.

The first post-rebuild setpoint read `prog_761b2062-880f-475e-b927-6a90e8f73826`
temporarily showed0/17 in band because the prior capture receipt described the
old build. Performance Health confirmed it was unavailable for that reason. A
single fresh capture workload `0ec2e936779ff37ea697247386008c42` then passed
100 samples plus one warmup at444ms service p95 /640.403589ms wall p95 against
2000ms on the new build. The final setpoint read
`prog_94970f5b-20bd-4eb2-a1d1-2fdc170c429f` is back to1/17,16 pending,
product qualification false. The profile row remains pending because no
governed reader consumes its owner receipt; this work improves current-build
confidence but does not move the whole-goal score. No Test Genie run was
started. The first `make restart` took306s to ensure dependencies, so avoid
repeating it without a source/build mismatch or a contract journey requiring a
managed restart. Current include-untracked inventory digest is
`cddec42131387f694377ae05d15433c2ccc05c09f892bb959077ee7dd020c711`; runtime
lines are API92,414, driver52,510 and UI130,198. Since W126's observation,
driver size increases by28 net for the W129 reservation, API increases by3 and
UI is unchanged. This size measure does not establish structural complexity;
no net complexity reduction is claimed. One timing trial only; repeatability, abrupt process-death
recovery and non-Linux targets remain unverified. No net complexity reduction
is claimed. Current next action: implement the narrow governed profile-receipt
read/join only after establishing how the target validation run can attest this
owner producer without making a stale `/tmp` result appear current.

### BAS-WORK-131 — 2026-09-24 UTC — exact-phase profile evidence join

BAS-FB-016 remains active. W130 supplies current-build behavioral owner receipts,
but the setpoint correctly leaves profile-durability pending without governed
attribution. The read-only `test-genie/runs/list` binding is available and
returns `planned_phases` plus per-phase status, so the join can require exactly
the BAS-owned `profile-durability` phase instead of counting arbitrary test
receipts or requiring another broad unit/workflow suite.

The bounded owner extension is BAS API + Test Genie descriptor + BAS
setpoint-read: add a BAS validation-provider phase which inspects the retained
W130 cohort receipts, validates their contract/source hashes and checks that
the receipt build still matches live BAS health; add a read-only Test Genie run
binding to the setpoint and count only a passed, current-candidate run whose
single planned phase is `profile-durability` and whose completion follows the
current capture receipt. This does not modify shared Test Genie or
program-runtime code. Before any behavioral run, focused provider tests and
static phase-plan/applicability reads will verify the exact phase declaration.
Implemented the provider in `api/handlers/profilevalidation/`, registered it
through `.vrooli/test-genie.json`, and added the optional read-only
`test-genie/runs/list` binding. The provider selects retained cohort receipts
matching the live API build, then verifies the contract digest, source digests,
owner-receipt digests and contents, checkpoint bound, isolation, restart and
cleanup assertions. A stale receipt remains a failed assessment. The setpoint
counts only a passing run with exactly one planned and passing
`profile-durability` phase completed after the capture receipt. It carries the
Test Genie evidence tier in the row reading.

Focused checks pass: `go test . ./handlers/profilevalidation -count=1`;
`python3 scenarios/browser-automation-studio/docs/internal/test_refactor_contract.py`
(9 tests); and `refactor_contract.py` (24 journeys, zero errors). The BAS
phase schema test passes; `test-genie phases inspect` and the scoped `phases
plan --phase profile-durability` show applicability only through the retained
cohort evidence glob and select one phase. A managed BAS restart deployed the
provider; the retained cohort passed 5 seed + 2 restart checks, including
cookie/localStorage/IndexedDB persistence, two-profile isolation, and cleanup.
Performance Health captures `ac27ba6cfd380b48c44c655cf2a4ae67` and
`e742a59a26f245b5feabfdebff77d6e7` passed100 samples plus one warmup at622ms
and473ms service p95 (795.97214ms and668.552432ms wall p95) against2000ms.
An intervening repeat `f358065339c34cfeca678250d3870775` retained a failed
sample: attempt4 received driver RESOURCE_LIMIT 429 with10 sessions already
present; the driver's final health read showed zero sessions. This is preserved
as contaminated/failed evidence, not a latency datapoint. Direct
Connect validation returned `VALIDATION_STATUS_PASSED` for live build
`sha256:39dda9b36b35c078721ea5141738ff1e8de58bf6ad16491febe5882fec72ebf6`.

Two Test Genie executions only, both targeted to the single
`profile-durability` phase; the latest `20260924-052843-5342932d` passed in1s
after the fresh capture, following an earlier pass
`20260924-051822-fdec2297`. The latest evidence tier is `degraded` because the
shared checkout was dirty; its tree digest and commit are retained in the run.
The second targeted phase became necessary after the intervening capture failed
on a 429 and a new valid capture made the earlier phase timestamp stale.
The provider's independent build/source/contract checks passed. Final setpoint
read `prog_7f34f892-c20c-472f-af54-155bdd5ea3fa` reports2/17 in-band,15 unknown,
`product_qualified=false`. No broad unit/workflow Test Genie suite ran. Full
owner/capture/run provenance: `internal/evidence/rehabilitation/profile-durability-131-2026-09-24.json`.

The `maturity-go` shared API dependency was installed through Scenario
Dependency Analyzer; dependency governance validation passes with repository
warnings unrelated to this package install. No BAS source complexity reduction
is claimed. Timing repeatability, abrupt process death, and non-Linux behavior
remain unverified. Next: choose the next high-consequence unknown outcome and
promote its owner evidence through a focused maintained regression and
read-only governed path.

The include-untracked source inventory digest is
`a1b27425a7ffa4a47b552e2ca6824330b7e8dabc4a4d0788aea406d992d1a624`; runtime
lines are API92,665, driver52,510 and UI130,198. This is physical size evidence,
not a complexity reduction. The repository-wide `test-genie provider-contract
scan --json` produced no output after40s and was interrupted locally; it remains
unverified. The descriptor schema test, exact phase inspect/plan, direct provider
RPC, and two exact-phase runs provide scoped coverage. Do not repeat the global
scan without a new reason.

### BAS-WORK-132 — 2026-09-24 UTC — stop joins owned runner cleanup

BAS-FB-016 remains active: this cycle used targeted Go regressions and no
Test Genie execution. Inspection found that `StopExecution` canceled the owned
runner but returned immediately, allowing the API to report `stopped` before
terminal persistence and deferred cleanup finished. The workflow service now
retains the runner completion channel with its cancellation control; stop
signals cancellation and joins completion using the caller's context. A focused
synctest verifies cancellation is signaled and stop remains pending until the
runner's cleanup signal closes; a second regression proves the caller context
bounds that wait. The existing routed resume/cancel scenario was adapted to
wait asynchronously and also passes.

Validation: `go test ./services/workflow -run
'^(TestStopExecutionWaitsForRunnerCleanup|TestStopExecutionWaitRespectsCallerCancellation|TestResumeRetainsRoutedContextAfterAdmission|TestSynchronousWaitCancellationPreservesDetachedExecution)$'
-count=1` passes. No managed restart or Test Genie run was needed for this
workflow service change. The change has not yet been built into the managed
candidate, and the timing bands (cancel <=1s, cleanup <=5s, recovery <=10s),
uncertain-side-effect reconciliation, and session recovery remain unqualified.
The readiness board remains at the last governed 2/17; this focused regression
does not claim cancellation-recovery in-band or improve product qualification.
No complexity reduction is claimed.

#### BAS-WORK-132 evidence amendment — 2026-09-24 UTC

Adversarial review found that the first W132 wait test supplied an artificial
completion channel, while the routed cancel test's final `synctest.Wait` could
hide an early stop return. Replaced that synthetic case with a real asynchronous
workflow-service runner: its action observes cancellation, the repository
persists terminal `cancelled`, and an owned event sink deliberately holds
`CloseExecution` open. The regression proves `StopExecution` stays pending
through that cleanup and returns only after release. Kept the caller-context
bound regression. This is stronger proof for W132's behavior, but remains a
controlled service fixture rather than a live-browser timing or resource-count
qualification.

The focused command passes:
`go test ./services/workflow -run
'^(TestStopExecutionWaitsForRunnerCleanup|TestStopExecutionWaitRespectsCallerCancellation|TestResumeRetainsRoutedContextAfterAdmission|TestSynchronousWaitCancellationPreservesDetachedExecution)$'
-count=1` (0.061s). `refactor_contract.py` reports preparation valid, 17 required
outcomes, 24 journeys and zero errors. Required governed setpoint read
`prog_582924c7-2e00-477c-bacb-06a2069413b8` completed `ok`, but still reports
2/17 in-band, 15 unknown, and `product_qualified=false`; cancellation-recovery
remains `pending_telemetry`. Inventory head `8707b970a48f82665662450766ad8233d6def9e7`,
digest `748b0b0d9d118a5597e50aea1759b7d56860cbab1e7266670e2785982fa85194`;
API/driver/UI runtime lines are 92,689/52,510/130,198. The baseline delta is
API +24 from W131, driver 0, UI 0; this count is physical size, not complexity.
No runtime path, duplicate owner, or caller was changed in this amendment, so
duplication/coupling were not remeasured and no simplification is claimed. The
old assessment's driver `unhandledRejection` lead is already guarded in current
`server.ts`, so it was rejected as stale. The managed build remains
`sha256:39dda9b36b35c078721ea5141738ff1e8de58bf6ad16491febe5882fec72ebf6`;
W132 source changes are not deployed there. No Test Genie run was started.
Next useful action: qualify the controlled stop/cleanup result with a real
fixture browser operation and independent live-session count, then bind that
retained owner receipt into the governed cancellation-recovery sensor if the
scope can provide reliable timing and recovery evidence.

### BAS-WORK-133 — 2026-09-24 UTC — refuse replay after uncertain browser effect

BAS-FB-016 remains active; this cycle used focused owner tests and no Test Genie
execution. The prior-art query found the same cancellation boundary in the dated
assessment, while current source showed its separate driver unhandled-rejection
lead is already repaired. Two falsifiable hypotheses were checked. H1: checkpoint
extraction ignores an uncertain failed step after its latest success and admits
resume; the maintained two-step regression reproduced this (`ResumeExecution`
returned nil before repair). H2: the old driver's idempotency receipt may survive
close and prevent a repeat; current receipts are scoped to a live session, and
close removes that session, so the old receipt cannot reconcile a new lease.

The repair declares `FailureCodeInstructionOutcomeUncertain` in the shared
execution contract and uses it in both the driver producer and workflow
checkpoint reader. Checkpoint extraction now refuses resume when this error code
appears after the last successful timeline entry, with a clear instruction to
reconcile before resuming. The new regression verifies that the replacement
runner is not admitted. The older shared source literal is removed; no caller
path or execution behavior outside this refusal changed. This is a conservative
recovery guard, not an external-effect reconciliation mechanism.

Focused validation passes:
`go test ./services/workflow -run
'^(TestResumeRejectsUncertainBrowserEffectAfterCheckpoint|TestResumeRequiresMatchingCommittedStore|TestStopExecutionWaitsForRunnerCleanup|TestStopExecutionWaitRespectsCallerCancellation|TestResumeRetainsRoutedContextAfterAdmission|TestSynchronousWaitCancellationPreservesDetachedExecution)$'
-count=1` and `go test ./automation/session -run
'^TestRunAmbiguousResponseCannotAuthorizeNewAttempt$' -count=1`. Contract
preparation passes with 17 required outcomes, 24 journeys and zero errors.
Setpoint read `prog_94c2cc9b-95a3-4ed6-a76f-48348867aaca` completed `ok`: 2/17
in-band, 15 unknown, `product_qualified=false`; cancellation-recovery remains
`pending_telemetry`. Inventory at HEAD `8707b970a48f82665662450766ad8233d6def9e7`
has digest `29517a2130e747dd9561f0a1d4e3554f45d7997a422446695352a6ad44ac8835`
and API/driver/UI runtime lines 92,710/52,510/130,198. The shared worktree is
dirty, so this physical-size snapshot is not a trustworthy per-change complexity
or duplication delta; no net structural-debt reduction is claimed. No coupling
scan or browser timing measurement ran. Managed build remains
`sha256:39dda9b36b35c078721ea5141738ff1e8de58bf6ad16491febe5882fec72ebf6`, which
does not include these source edits. No Test Genie run was started.

Unverified: live driver behavior when an HTTP request is cancelled in flight,
resource detach counts, accepted-cancel <=1s, terminal cleanup <=5s, session
recovery <=10s, and reliable external-effect reconciliation. Next: add a retained
controlled-browser owner fixture that records whether the action was applied,
stop time, cleanup time and independent session count; keep resume refused until
the fixture can prove reconciliation.

### BAS-WORK-134 — 2026-09-24 UTC — join driver lifecycle with in-flight action

BAS-FB-016 remains active; this cycle used a focused Playwright-driver owner
suite and no Test Genie run. Hypothesis: the driver sets `instructionInFlight`
but reset and close can still mutate or dispose the page before the admitted
handler settles. The route fixture reproduced the ordering gap: lifecycle code
had no completion handle to join. Added `instructionSettlement` to the session
state, resolved it from the route's `finally`, and made reset/close await it
before browser mutation or teardown. Strengthened the existing delayed-reset
regression and added a delayed-action/close regression; both prove lifecycle
work waits and new instruction admission stays rejected during the handoff.

Validation: `pnpm exec jest tests/unit/routes/session-run.test.ts --runInBand
--coverage=false` passes 42/42 in 2.15s; `pnpm exec tsc --noEmit` passes;
`git diff --check -- scenarios/browser-automation-studio/playwright-driver`
passes. The retained Chromium fixture also passes:
`pnpm exec jest tests/integration/typed-action-semantics.test.ts --runInBand
--coverage=false --testNamePattern='close joins its real browser effect'`.
It independently observes one click, proves close remains pending with one live
session until instruction settlement, then verifies the successful receipt and
zero remaining sessions. The initial focused runs exposed an out-of-scope
resolver, a stale reset test assumption and a fixture query after intentional
browser close; each was corrected, and the final checks are green. The
contract check reports preparation valid, 17 outcomes, 24 journeys and zero
errors. Setpoint read `prog_27db3640-e454-403d-803b-6314040918b3` completed
`ok`, but remains 2/17 in-band and 15 unknown; cancellation-recovery is still
`pending_telemetry`, so this unit receipt does not move a band. The required
inventory ran at HEAD `8707b970a48f82665662450766ad8233d6def9e7`; this shared
worktree has many unrelated dirty paths, preventing an attributable size,
duplication or complexity delta. No structural reduction is claimed.

The change establishes operation ordering only. It does not cancel a long-lived
browser effect, qualify accepted-cancel <=1s, terminal cleanup <=5s, session
recovery <=10s, live resource counts or external-effect reconciliation; the
managed driver identity is unchanged and does not include these source edits.
Next, extend the maintained controlled-browser qualification fixture to issue
a real delayed effect, cancel/close through managed ownership, and record an
independent effect counter, cleanup/detach timing and live-session count; bind
its retained receipt into the cancellation-recovery sensor if the evidence is
complete.

### BAS-WORK-135 — 2026-09-24 UTC — interrupt a pending driver operation on close

BAS-FB-016 remains active; this cycle used targeted driver tests and no Test
Genie execution. Hypothesis: session close can meet cleanup timing if it first
interrupts the active browser operation, then joins the route until its outcome
is durably retained. Added a Chromium regression for a real `waitForSelector`
with a 15s action timeout. Before repair, close was still pending after a 4.5s
watchdog; the action itself had not reached its timeout. This reproduced the
long-close behavior introduced by W134's safe join.

Close now marks an admitted instruction interrupted. Owned-session teardown
closes the selected page first; external sessions detach their CDP connection.
The route translates any late handler result into `INSTRUCTION_OUTCOME_UNCERTAIN`
and retains it before the teardown joins the settlement and releases the lease.
Reset retains its join-before-mutation behavior. A second real Chromium J07
case performs an actual click, starts close before the receipt is retained, and
proves one observed effect with a non-retryable uncertain receipt. The pending
wait case proves close completes in under 1s, new admission stops immediately,
the uncertain receipt is retained, no session remains, and a fresh managed
session can be created and closed within the 10s recovery band.

Focused validation: `pnpm exec jest
tests/integration/typed-action-semantics.test.ts --runInBand --coverage=false
--testNamePattern='REQ:BAS-RH-J07'` passes 2/2 (both real Chromium cases);
`pnpm exec jest tests/unit/routes/session-run.test.ts --runInBand
--coverage=false` passes 42/42; targeted manager reset/close recovery tests pass
15/15; `pnpm exec tsc --noEmit` and `git diff --check` pass. The manager-only
Jest invocation emitted a “did not exit one second” open-handle warning despite
exit 0; its source is unknown, so no verdict is claimed about natural exit and
the next driver-suite run should isolate that handle if it persists. The contract
checker still reports preparation valid, 17 required outcomes, 24 journeys and
zero errors. J07 now references both maintained Chromium cases but remains
`planned` until death/restart, API terminal-state, external-target and retained
cohort evidence also pass.

Setpoint read `prog_6d2db798-d094-4ef8-89aa-bb37c00ff36a` completed `ok` and
now reports 0/17 in-band, 17 unknown, `product_qualified=false`. The capture
receipt is stale (`workload configuration, producer or contract changed`) and
profile durability has no matching current-candidate phase receipt. The J07
traceability references changed the canonical contract digest, so prior
candidate evidence is no longer current; this is evidence invalidation, not a
claim that the product regressed. Requalify receipts against the updated
contract before using them. Inventory at HEAD
`8707b970a48f82665662450766ad8233d6def9e7`, source digest
`00867e86a11f398c969d9098ed4f88dc86b65a9ec0b6549b8618d29f62a9c2c0`, with 111
dirty worktree paths. Runtime lines are API 92,710, driver 52,538, UI 130,198;
driver increased 28 lines from W134. Physical inventory is not a complexity
measure. No net complexity/duplication/coupling reduction is claimed; these
focused tests add owner evidence while the shared worktree's broader dirty
state prevents an attributable domain-wide comparison. The managed build remains
`sha256:39dda9b36b35c078721ea5141738ff1e8de58bf6ad16491febe5882fec72ebf6` and
does not include this source repair. External-target detach interruption,
driver/API death or restart, terminal workflow cancellation receipt, retained
effect-counter artifacts, and an independently governed J07 sensor are not yet
qualified. Next: preserve the test observations in the BAS-owned cancellation
qualification receipt and test Go `StopExecution` through the API/driver path;
re-establish stale unaffected baseline rows through only their required owner
checks so the global board stays current without broad Test Genie reruns.

### BAS-WORK-136 — 2026-09-24 UTC — requalify capture and profile durability on the current managed candidate

BAS-FB-016 remains active. This cycle used focused Go regressions and owner
fixtures, with no broad Test Genie rerun. `go test ./services/workflow -run
'^TestStopExecution(WaitsForRunnerCleanup|WaitRespectsCallerCancellation)$'
-count=1` and `go test ./automation/session -run
'^TestRunAmbiguousResponseCannotAuthorizeNewAttempt$' -count=1` pass. They
verify owned-runner cleanup joining, caller-cancelled waiting, and preservation
of an uncertain non-retryable browser outcome.

The first fresh Performance Health workload, operation
`0fc236ce47e4f92b3f74a153231548db`, failed at attempt 12 when the driver
returned 429 `Maximum concurrent sessions reached: 10`; its full first-attempt
receipt remains at
`/home/matthalloran8/.vrooli/test-runs/performance-health/workloads/0fc236ce47e4f92b3f74a153231548db/producer/receipt.json`.
After confirming the driver had zero sessions, one retry (`ad31321e384ab75ccfc99fcab5071791`) passed 100/100 samples at423ms service
p95/596.871449ms wall p95. The profile cohort required a managed restart, which
changed the candidate build identity, so that earlier capture was not reused.
The final capture on the same build as the retained profile cohort,
`d9bf608f1879e73846e639ff2863dd4b`, passed 100/100+1 warmup at446ms service
p95/637.583105ms wall p95 against the 2000ms budget. Receipt SHA:
`442b19e0063d01a220ebf8729abc36459cd9fca0b0d32c489dcc4d8f22434920`.

The exact `profile-durability` phase first failed in run
`20260924-063438-174fb2c1` because W131's retained contract digest no longer
matched the canonical contract. This was stale evidence, not a failed profile
assertion. The first receipt refresh also crossed managed build identities
because I copied intermediate evidence into BAS's source tree between seed and
restart. I kept those observations out of qualification, held the tree stable,
then reran the cohort. On build
`sha256:4a5edbfabd0404ef4865ec864b2d01ab09f3fa71ed0d59d0da1e2dd10e5699fa`,
five seed checks passed, checkpoint visibility was1745ms, both alpha/beta
cookie/localStorage/IndexedDB identities survived the managed API/driver
restart, and both temporary profiles were deleted. Owner artifacts:
`internal/evidence/rehabilitation/profile-durability-136-2026-09-24.json`,
`profile-durability-136-seed-owner.json`, and
`profile-durability-136-restart-owner.json`. The second exact phase run,
`20260924-064830-e1de615b`, passed L1/Verified. The provider validated the
contract digest, tracked driver/harness hashes, owner receipt hashes, cohort
assertions and live build identity. Its evidence tier is degraded because the
shared checkout is dirty; this is reported, not promoted to a clean-tree claim.

The governed setpoint read `prog_1d2856c6-1d1a-493a-afba-b7899e0e8d1d` reports
2/17 in-band,15 unknown, `product_qualified=false`. Final reread
`prog_346f10f2-3a31-4ae4-9d16-9a7234ca6fc8` confirms that reading after the
documentation checkpoint edits. This cycle therefore moved one additional
outcome into band without changing the 15 unknown outcomes.
Contract preparation remains valid with 24 preservation journeys and zero
errors. No structural complexity, duplication or coupling reduction is claimed.
The previously unresolved broad Test Genie run `20260924-025629-7e6ab613` is
now verified terminal `failed`; its unit phase has one blocker from UI
`pnpm run test:coverage`. No broad replacement run was started. New source-tree
evidence and progress documentation now postdate the live managed build; a
future lifecycle restart will produce a new identity and require refreshing
build-bound cohort receipts before claiming current-candidate profile evidence.
The final live driver inventory showed one separate RUNNING execution
(`e5e524b7-5d99-48b5-9831-fc0478a55b17`, started at06:53:39Z, URL
`http://localhost:24767/setup/host`). I left its live-owned session intact;
the final pool is not idle and the 10-session capacity boundary is not broadly
qualified. The earlier 429's concurrent session owners were not identified.

Next: continue the J07 cancellation boundary through Go `StopExecution`, the
managed driver lease and an independent effect counter/session-count receipt;
then advance a separate unknown outcome with owner evidence using focused tests.

### BAS-WORK-137 — 2026-09-24 UTC — join workflow cancellation to driver lease cleanup

BAS-FB-016 remains active. The new maintained J07 test,
`TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose`, drives a
saved workflow through the production Go workflow service, `SimpleExecutor`,
Playwright engine, session manager and leased driver client into an isolated
HTTP fixture. The fixture increments an independent effect counter before it
drops the instruction response, blocks the lease-close acknowledgement, and
tracks its live-session count. The test verifies cancellation reaches the live
instruction, the same execution/lease owns close, the external effect remains
uncertain and non-retryable, the session remains live until close is confirmed,
and `StopExecution` does not return before that confirmation. The focused run
and its `-race` variant pass. Adjacent targeted Go tests pass for
cleanup joining, caller-context cancellation and refusing a retry after an
ambiguous session response. No broad Test Genie execution ran.

Corrected J07 traceability in `REFRACTOR_CONTRACT.json`: the two existing real
Chromium J07 tests and this Go service-to-driver test now belong to J07; they had
been accidentally listed under J01. Contract preparation reports24 journeys
and zero errors. The correction changes the canonical contract digest, so the
prior profile-durability receipts no longer qualify. I did not restart managed
services while preserving the separately active live-owned execution.

To restore unaffected current-candidate evidence, the declared Performance
Health capture workload operation
`65f133be2bd0715bf54cc01c19c7c949` passed100 measured samples plus one warmup at
412ms service p95 and568.291511ms wall p95 against a2000ms budget on build
`sha256:4a5edbfabd0404ef4865ec864b2d01ab09f3fa71ed0d59d0da1e2dd10e5699fa`.
Receipt SHA is
`c585c6ac2cef0ca333532246d5a1b6e1e2e07c11dfd8c52814c68980295d31a0`. The
final governed setpoint read `prog_e14ad804-2133-4809-9238-b800a4528232` reports
1/17 in-band,16 unknown, `product_qualified=false`; capture is current, while
profile durability has no matching current-candidate phase receipt. This pass
adds test coverage and correct attribution but does not claim production-code or
complexity reduction. External-target detach, process death/restart and live
browser cancellation timing remain unqualified.

### BAS-WORK-138 — 2026-09-24 UTC — repair J07 attribution and requalify current receipts

BAS-FB-016 remains active. Current-state audit found W137's evidence move had
put the three J07 test references under J02 while leaving J07 empty. I corrected
the contract by journey ID; J01 and J02 now have no J07 references, and J07 has
the two existing driver/Chromium cases plus
`TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose`. Contract
preparation remains valid:24 journeys, zero errors. This fixes traceability; it
does not make the J07 release outcome qualified.

The audit also exposed an obsolete hard-coded fixture in
`handlers/profilevalidation/provider_test.go`: its “current evidence” assertion
depended on a historical contract, repository receipt and build ID. I replaced
that environment-dependent setup with a temporary hash-consistent contract,
source and owner-receipt fixture, while retaining current-build acceptance and
stale-build rejection cases. `go test . ./handlers/profilevalidation -count=1`
passes, as do all nine contract tests.

The first seed/restart/verify cohort in `/tmp/bas-profile-durability-138` passed
behaviorally but crossed build identities: the managed restart incorporated
W137's source changes and issued a new identity. It was discarded as
qualification evidence. With sources stable, the
cohort in `/tmp/bas-profile-durability-139` passed five seed checks, checkpoint
visibility at1537.966ms, two post-restart identity checks and cleanup of both
synthetic profiles on build
`sha256:3484515d0fcf2f821276c3047e0e5036e47648fa417d08b393d6955a20699f95`.
The retained owner artifacts and aggregate are
`internal/evidence/rehabilitation/profile-durability-138-seed-owner.json`,
`profile-durability-138-restart-owner.json`, and
`profile-durability-138-2026-09-24.json`.

Performance Health workload `63abbeae9458c58df4e5b59c89eec181` passed100/100
samples plus one warmup at437ms service p95 and633.314174ms wall p95 against a
2000ms budget on the same build. `workload-get` confirmed receipt SHA
`f1274dff3661febc4f0edc6c33e280b094ce6c261a3e11300fd8841f6eb8a8ba`. Only the
exact required Test Genie phase ran: `20260924-072006-49f64968`,
`profile-durability`, PASS L1/Verified. Its evidence tier is degraded due to the
dirty shared checkout; the provider checked the canonical contract and producer
hashes, owner receipt hashes/assertions, and live build identity. The final
setpoint read `prog_920f732b-89a9-4fae-b239-b41798b7eee6` is 2/17 in-band,
15 unknown, `product_qualified=false`.

The final managed health read is healthy on the same build, but the driver has
one active execution (`4769a95f-973c-4a49-b4f4-51ddba50d2f8`, workflow
`99f8ccb9-46ff-405c-ab7d-a41f85cf74e4`, URL `http://localhost:23485/`). I left
the session untouched. No production code changed in W138 and no net
debt/complexity reduction was claimed; the provider test became deterministic
at the cost of a small fixture helper. J07 still lacks managed real-browser
cancel/timeout, death/retry, repeatable cleanup timing and a governed
cancellation receipt. W139 below adds a focused false-stop-acknowledgement
guard; next extend J07 evidence and pursue another high-impact unknown without
broad suite runs.

### BAS-WORK-139 — 2026-09-24 UTC — reject stop acknowledgement without owner

BAS-FB-016 remains active. Hypothesis: after restart, a missing in-memory
cancellation entry lets `StopExecution` report success while the durable run is
still active. Inspection falsified the normal-restart portion: `api/main.go`
runs `RecoverInterruptedExecutions` before accepting requests, and recovery
finalizes every persisted pending/running row. A narrower state remains
possible after terminal persistence failure or when an active row is visible
to a process that does not own its runner. The service previously returned nil
for either state without reading the row.

Added `TestStopExecutionWithoutLocalOwnerUsesDurableStatus`. Its running case
failed before the repair because a durable `running` row was acknowledged with
nil. `StopExecution` now checks the repository on an owner-map miss: pending/running
rows return an error; repository read errors also fail closed; absent and
terminal rows preserve idempotent behavior. No test-genie or managed service
restart ran.

Targeted validation passed:

From `api/`:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test ./services/workflow -run '^(TestStopExecutionWithoutLocalOwnerUsesDurableStatus|TestStopExecutionWaitsForRunnerCleanup|TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose|TestStopExecutionWaitRespectsCallerCancellation|TestWorkflowClosesDecoratedSinkOnEveryExit)$' -count=1
GOTOOLCHAIN=local GOPROXY=off go test -race ./services/workflow -run '^(TestStopExecutionWithoutLocalOwnerUsesDurableStatus|TestStopExecutionWaitsForRunnerCleanup|TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose|TestStopExecutionWaitRespectsCallerCancellation)$' -count=1
GOTOOLCHAIN=local GOPROXY=off go test ./handlers/executions -run '^TestStopExecution$' -count=1
~~~

From the scenario directory:

~~~bash
python3 docs/internal/refactor_contract.py
~~~

All passed; contract preparation reports24 journeys, zero errors. The focused
driver Chromium J07 tests from this pass also passed (completed click remains
uncertain and non-retryable; cancelled selector wait closes promptly). Current
setpoint read `prog_3816adb4-9661-489e-81d7-73bf0d1e1bb2` remains 2/17 in-band,
15 unknown, `product_qualified=false`; only capture and profile durability have
governed receipts. The reader did not change a sensor receipt for this source
edit. No measurable net complexity reduction is claimed. Cross-process owner
routing, terminal-write failure injection, managed cancellation and live
browser timing remain unqualified; next create owner evidence for J07 on the
current candidate without a broad Test Genie run.

### BAS-WORK-140 — 2026-09-24 UTC — verify the stop error at the RPC boundary

W3 follow-up for BAS-RF-125. Added
`TestStopExecutionReturnsOwnerUnavailableError` in
`api/handlers/executions/service_test.go`. The stub executor returns the same
owner-unavailable error produced by the workflow service; the Connect RPC must
return an error and no response carrying `status=stopped`. This closes the
service-to-handler assertion for the repaired false acknowledgement without
adding a new Test Genie execution or changing product code.

Focused checks passed: from `api/`,
`GOTOOLCHAIN=local GOPROXY=off go test ./handlers/executions -run '^TestStopExecution(ReturnsOwnerUnavailableError)?$' -count=1`, and
`GOTOOLCHAIN=local GOPROXY=off go test ./services/workflow -run '^TestStopExecutionWithoutLocalOwnerUsesDurableStatus$' -count=1`.
`git diff --check` for the handler test passed. Rechecked contract preparation
(24 journeys, zero errors), `vrooli scenario requirements validate
browser-automation-studio` (PASSED; existing informational template notices and
noncritical no-validation warnings remain), and setpoint read
`prog_4eedf8d4-6070-4592-9cbe-ef41432ee261`: 2/17 in-band, 15 unknown,
`product_qualified=false`. No behavior, performance, complexity or debt delta
is claimed for this test-only follow-up. J07's release band remains unqualified
until its external-effect, terminal-state, resource-count and recovery cases
have a current owner receipt.

### BAS-WORK-141 — 2026-09-24 UTC — link focused cancellation evidence to J07

BAS-FB-016 remains active. Reinspection found the focused cancellation tests
were already attributed in the rehabilitation contract but the technical
requirement still had `validation: []`. That made the test evidence invisible
to the business traceability graph. Added three real file references for the
workflow service, Connect handler and Chromium driver suites to
`requirements/08-rehabilitation/module.json`, all with `status: planned` so the
uncovered timeout, driver death, API restart and retry cases cannot be mistaken
for completion. Added `[REQ:BAS-RH-J07]` log markers to the Go service and RPC
tests; existing Chromium test names already emit their markers.

Targeted Go tests passed with verbose output showing both J07 tags:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -v ./services/workflow -run '^(TestStopExecutionWithoutLocalOwnerUsesDurableStatus|TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose)$' -count=1
GOTOOLCHAIN=local GOPROXY=off go test -v ./handlers/executions -run '^TestStopExecutionReturnsOwnerUnavailableError$' -count=1
~~~

`python3 docs/internal/refactor_contract.py` passes (24 journeys, zero errors),
`vrooli scenario requirements validate browser-automation-studio` passes, and
`business-health validate scenario browser-automation-studio --json` is
`PASSED`; a focused query found zero `business_req_no_validation` and zero
missing-ref findings for J07. The global registry still has other open
traceability warnings. The setpoint read
`prog_c43c68c9-0a95-4122-a89a-e2fdc6346f72` remains 2/17 in-band, 15 unknown,
`product_qualified=false`. No Test Genie execution ran. J07 remains
`planned` and no release row moved; this is traceability progress, not a claim
that the outcome passes. No product-code, performance, complexity or debt
reduction is claimed. Next produce and govern a complete J07 owner receipt with
the independent external-effect, terminal-state, resource-count and recovery
oracles, using only the exact validation required for that receipt.

### BAS-WORK-142 — 2026-09-24 UTC — fail startup when execution ownership recovery is incomplete

BAS-RF-126 records a startup ownership gap found while pursuing J07: recovery
errors were logged as warnings after sidecar setup, so the API could continue
with persisted `pending`/`running` rows whose previous-process owner was gone.
Recovery now runs immediately after repository initialization, before workers
and sidecars. The recovery service returns an error when any active row update
fails, and startup treats that error as fatal. This preserves health endpoints
for optional automation-engine failures while preventing execution ownership
from being silently ambiguous.

Added a focused recovery-service regression for a failed status update. From
`api/`, `go test ./services/recovery ./services/workflow ./handlers/executions`
passes. A lifecycle restart first confirmed zero observed driver sessions, then
deployed build
`sha256:3ff3a2b336662ada94430c1021952760d6e88b326839faa5ee0ea05310974115`;
the API is healthy and the driver still reports zero sessions. One optional
Anthropic credential is unavailable; required credential gaps are zero.
The initial required setpoint read
`prog_c9baaf0f-806b-4fbc-9d85-8c9a302debce` returned 0/17 because previous-build
receipts no longer joined to the current candidate. To restore attributable
evidence without Test Genie, Performance Health ran capture workload
`5f2ecbcfd5c84ee622ada49341b0b10d`: 100/100 samples plus one warmup at432ms
service p95 and629.916694ms wall p95 against2000ms. The retained receipt SHA is
`3af6705e88444104ca4b15d0a5f5693a2ca0224fe45e57c6b984bd01223be8e7`; its build
matches the live API. Final setpoint read
`prog_f6ab7078-5f11-43b7-9903-54cc4d741acc` is 1/17 in-band,16 unknown,
`product_qualified=false`. No Test Genie run was started. RF-126 is repaired
and deployed; simulated startup failure, abrupt process death, multi-process
ownership, and the full J07 cancellation receipt remain unverified. No
complexity or debt reduction is claimed.

### BAS-WORK-143 — 2026-09-24 UTC — trace startup recovery tests to J07

The contract's W2 gap is still the incomplete J07 owner receipt, but startup
recovery now provides direct evidence for the API-restart branch. Added
`[REQ:BAS-RH-J07]` markers to the recovery tests and linked
`api/services/recovery/service_test.go` from J07 with status `planned`; the
validation notes explicitly retain driver-death, bounded-timeout and overlapping
retry gaps.

Focused checks passed: three recovery tests (terminalizing persisted active
rows, rejecting unresolved owners, and recovering recent running/pending rows),
the uncertain-effect/resume Go tests, and both J07-tagged real-Chromium tests in
`typed-action-semantics.test.ts`. The structural requirements validator passed;
Business Health returned `PASSED` with no J07-specific findings. J07 remains
`planned`; this pass changes no runtime behavior and adds no new qualified
setpoint outcome. Latest governed read
`prog_74c9c6e5-60e7-4cbb-8b59-727de0b7ef65` reports 1/17 in-band,16 unknown.
No Test Genie run was started. Next, build one current-candidate owner receipt that joins the
existing focused cases to the cancellation-recovery band, then close the
uncovered driver-death and timeout/retry cases with targeted fixtures.

### BAS-WORK-144 — 2026-09-24 UTC — validate J07 receipt plumbing with focused tests

BAS-FB-016 remains active. Added a typed J07 receipt validator that binds the
five required observations to the current managed build, rehabilitation
contract, and source hashes, and rejects incomplete/out-of-band cleanup,
recovery, uncertainty, or retry evidence. The focused provider validates the
profile and cancellation capabilities together in one exact
`rehabilitation-evidence` phase. Updated the protocol to state clearly that the
five-case owner producer is still missing; passing validator tests are not
product evidence. No cancellation receipt was fabricated or retained.

Focused verification passed: `go test -race
./internal/cancellationqualification ./handlers/profilevalidation`, the targeted
workflow cancellation/uncertain-resume tests, and the recovery tests. The
provider test also validates the Test Genie descriptor against its schema.
`python3 docs/internal/refactor_contract.py` passes (24 journeys, zero errors).
`program-runtime library run browser-automation-studio.setpoint-read --input
profile=rehabilitation` completed as
`prog_7dc6b7f2-2cdd-4539-be8b-e429ffb0ea69`: 1/17 in-band, 16 unknown,
`product_qualified=false`; capture remains at 432ms service p95 and
629.916694ms wall p95. Profile durability and cancellation recovery do not have
a current exact-phase receipt. No Test Genie execution ran. A repository-wide
`git diff --check` reported pre-existing trailing whitespace in
`scenarios/audio-tools/ui/index.html`; scoped BAS checks are recorded separately.
No product behavior, performance, complexity, or debt improvement is claimed.
Next add a real independent owner fixture for the missing J07 timeout and
driver-death cases, then make it produce a current-build receipt from maintained
targeted tests.

### BAS-WORK-145 — 2026-09-24 UTC — exercise J07 timeout and driver-loss paths directly

BAS-FB-016 remains active. Added two focused public-executor regressions in
`api/automation/executor/session_lifecycle_test.go`. The timeout case starts a
live HTTP instruction, commits one independent fixture effect, expires the
caller context after50ms, and verifies a retained timeout/uncertain/non-retryable
failure, one effect only, owned-session close, and cleanup inside the 1s stop
band. The driver-loss case closes the mock driver's listener after the
independent effect commits; it verifies terminal uncertain failure without a
retry and zero process-owned sessions after death. Added the owner test file to
J07 source-digest validation and linked it as a passing individual test while
leaving the whole J07 obligation `planned`.

The first test attempt exposed a test-fixture defect: the health route was
missing. A subsequent mock held its mutex during a blocked request, preventing
the cleanup route from entering; the focused 10s timeout stack identified the
lock, and separating request wait from counter synchronization fixed the
fixture. Final focused race runs passed:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 20s ./automation/executor -run '^TestExecute(TimeoutDuringLiveInstructionClosesSessionWithoutReplay|DriverDeathRetainsUncertainEffectWithoutReplay|PreservesInvocationAndTransportOwnership)$' -count=1
GOTOOLCHAIN=local GOPROXY=off go test -race ./internal/cancellationqualification ./handlers/profilevalidation
~~~

`python3 docs/internal/refactor_contract.py` reports valid preparation, 24
journeys and zero errors. `vrooli scenario requirements validate
browser-automation-studio` passes with existing informational PRD notices and
unrelated missing-validation warnings. Scoped `git diff --check` passes. Final
setpoint read `prog_a48bb073-2ebc-4f05-9b59-ab223b1a7a2e` remains 1/17 in-band,
16 unknown, `product_qualified=false`; capture remains 432ms service p95 and
629.916694ms wall p95. The new regressions add direct coverage but do not write
the five-case receipt, run against managed process restart, or advance the
global score. No Test Genie execution ran. This is test-only work: no product
behavior, performance, complexity, or debt improvement is claimed. Next build
the focused owner producer that joins passing cancellation, timeout,
driver-death, restart, and retried-start observations into a real receipt.

### BAS-WORK-146 — 2026-09-24 UTC — refresh current-build profile and capture evidence

BAS-FB-016 remains active. The previous managed candidate changed from build
`3ff3a2b336662ada94430c1021952760d6e88b326839faa5ee0ea05310974115` to
`a2a0938a5c89776b5dbe86f9dd33ab4ef1902ec0b3babfb02d5ec164d2dfb2ea` during a
lifecycle restart, invalidating the first seed/verify cohort's build join. That
attempt restored both profiles and cleaned them up but is intentionally not
retained as qualifying evidence. With source stable, a fresh focused seed stage
passed five checks; checkpoint visibility was1638.938ms, alpha survived
close/reopen and beta activity, and alpha/beta remained isolated. A second
managed lifecycle restart kept the same build. The verify stage passed both
post-restart identity checks and deleted both synthetic profiles. API and driver
returned healthy; the optional Anthropic credential is unavailable, with zero
required credential gaps.

The declared Performance Health capture workload then passed100 measured
samples plus one warmup at413ms service p95 and575.487613ms wall p95 against a
2000ms budget on the same build. Its operation is
`8e9c95b9e60f5093b94006a3727e17e5`, receipt SHA
`f1676f1662cf8d02fcb8876e165b2a33256610c64edec69a42d1a0da89ad1613`.
Owner receipts and the source/contract-bound cohort aggregate are retained at
`internal/evidence/rehabilitation/profile-durability-146-2026-09-24.json` and
its `-seed-owner.json` / `-restart-owner.json` files. The combined
`rehabilitation-evidence` Test Genie phase was deliberately not run: its J07
capability still has no five-case owner receipt, so it would reject the
composite evidence. Profile behavior is directly observed, but profile and
cancellation release rows are not claimed qualified until that exact phase
passes. No Test Genie run or broad suite ran. Final governed setpoint read
`prog_2e39822d-2c7c-4d7e-af36-1668dfc633a5` reports1/17 in-band,16 unknown,
`product_qualified=false`. No product behavior, performance
change, complexity reduction or debt reduction is claimed. Next, build the
managed five-case J07 producer instead of spending another cycle on plumbing or
a predictable composite-phase failure.

### BAS-WORK-147 — 2026-09-24 UTC — split capability readings and repair build-identity projection

BAS-FB-018 requests more frequent owner-focused tests and fewer Test Genie
executions. No Test Genie run or broad suite ran. While fixing the setpoint's
coupled capability sensor, inspection confirmed Test Genie retains capability
presentation when the composite phase fails; the setpoint now reads each
capability's own level and clean verdict independently. The exact phase was
not run because J07 still has no complete owner receipt.

The first focused setpoint read exposed a separate false unavailability:
`vrooli/scenario/status`'s typed response conversion dropped the managed build
identity from both runtime and scenario projections, although the protobuf
already declares them and the direct CLI status returns them. BAS's current
capture and profile receipts therefore could not be safely joined to the live
build. Scope was recorded above before the shared conversion was changed. The
root control-plane mapper now carries both fields and a focused regression
passes:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test ./internal/api -run '^TestScenarioStatusPreservesManagedBuildIdentityAcrossTypedProjection$' -count=1
python3 scenarios/browser-automation-studio/docs/internal/refactor_contract.py
python3 scenarios/browser-automation-studio/docs/internal/refactor_regressions.py --case input
vrooli scenario requirements validate browser-automation-studio
~~~

The BAS contract validates 24 journeys with zero errors; the focused input
regression passes; requirements validation passes with existing informational
PRD and missing-validation notices. Scoped `git diff --check` passes. Final
governed read `prog_01a13509-f057-4d29-aca9-765a70517ae8` returns0 readable,
17 unavailable, `product_qualified=false`, because the running root API still
serves the old projection. The source repair is tested but its live binding
refresh is unverified; the previously accepted1/17 evidence remains the
qualification baseline. No J07 producer or receipt was created. Next refresh the
root API through its owning project lifecycle, verify the live typed response,
then return to the five-case J07 producer. No product qualification or debt
reduction is claimed.

### BAS-WORK-148 — 2026-09-24 UTC — refresh managed status and qualify profile receipt

BAS-FB-018 remains active. The root project lifecycle now supports the narrow
`vrooli develop --restart-api` path; focused setup and CLI parser tests pass,
and the managed restart replaced only `vrooli-api` (old PID3166664) on port8092.
Live `scenario.status` and `runtime.status` both return build identity
`sha256:a2a0938a5c89776b5dbe86f9dd33ab4ef1902ec0b3babfb02d5ec164d2dfb2ea`.
The restart used `--resources none --scenarios none --onboarding none`.

The first phase attempt named `profile-durability`, based on a stale catalog
preview. The run endpoint rejected it before creating a run because the current
BAS descriptor names `rehabilitation-evidence`; no Test Genie execution
occurred. A managed Test Genie restart refreshed its descriptor-backed
catalog. One exact `rehabilitation-evidence` run,
`20260924-094617-80832192`, then completed in1s and failed only on
`CANCELLATION_EVIDENCE_INVALID`: no retained cancellation receipt exists for
the current build. Its persisted capability presentation marks profile
durability L1/Verified and clean, with degraded evidence tier because the
shared checkout is dirty. The setpoint read
`prog_54179793-60d2-4365-a01e-3713c856a744` reports2/17 in-band (capture,
profile),14 unavailable and cancellation out of band;
`product_qualified=false`.

Focused root lifecycle tests, the CLI parser regression, contract validation
(24 journeys, zero errors), the input regression, requirements validation and
scoped `git diff --check` pass. Test Genie restarted healthy in degraded mode
because its declared `agent-inbox` dependency could not start (`scenario-to-cloud`
API freshness timed out); the BAS provider phase still ran and produced the
findings above. No broad suite ran. Next build the maintained J07 owner producer
for timeout, cancellation, driver death, restart and retry, then run only that
qualification path. No product readiness or debt reduction is claimed.

### BAS-WORK-149 — 2026-09-24 UTC — verify J07 owner behavior and locate evidence gap

BAS-FB-018 remains active. Prior-art lookup
`search-hub query "BAS J07 cancellation-recovery receipt qualification producer exact cases" --type library,record,skill,doc`
returned the maintained qualification protocol and prior progress, including
W145's focused timeout/driver-loss regressions. The protocol remains the
authority: all five J07 cases must be measured by the independent fixture and
joined to the live build.

W149 scope extension: the cancellation receipt binds the timeout/cancel Go test
sources but omits the startup-recovery test and the Playwright route/integration
tests that supply the restart and retry observations. This allows those owner
assertions to change without invalidating a retained receipt. Extend the
receipt's existing source digest list and validator regression only; do not
change qualification thresholds or synthesize observations.

Three hypotheses guided the focused check. H1: the driver admits a second
session or action when the same execution start is retried during a live action.
The exact session-manager tests and real-Chromium close/retry tests reject H1:
same-execution starts return one session, a competing action receives409, and
the independent click log remains at one. H2: accepted cancellation may return
before driver cleanup. The race-tested workflow owner regression rejects H2:
it holds close acknowledgement, proves `StopExecution` remains pending, then
observes one terminal cancelled record and zero live fixture sessions after
cleanup. H3: the recovery-service unit test proves live API restart recovery.
That is false: it only terminalizes a persisted row in-process; it does not
overlap restart with an external effect or live driver resource.

Focused tests passed without a Test Genie execution or broad suite:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 30s ./services/workflow ./services/recovery ./automation/executor -run '^(TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose|TestExecuteTimeoutDuringLiveInstructionClosesSessionWithoutReplay|TestExecuteDriverDeathRetainsUncertainEffectWithoutReplay|TestRecoverInterruptedExecutions_MarksInterrupted)$' -count=1
pnpm exec jest tests/integration/typed-action-semantics.test.ts --runInBand --testNamePattern='close keeps a completed click uncertain while denying retry admission|close interrupts a pending browser wait and retains an uncertain receipt'
pnpm exec jest tests/unit/idempotency/session-idempotency.test.ts --runInBand --testNamePattern='should return same session when called twice with same execution_id|should handle concurrent requests with same execution_id'
~~~

The Jest commands pass their selected tests; the session-idempotency process
prints the existing open-handle warning after its two tests pass, which remains
unexplained. The earlier phase presentation still identifies missing
current-build cancellation evidence. The receipt validator has no producer, so
no numeric fields were inferred and no receipt was fabricated. The root API and
Test Genie phase catalog are current; the BAS provider process still serves its
previous binary, before W149's source-digest extension. Work now sits at W2
evidence production, with a real managed API-restart-during-instruction case
still unproven. Next implement a maintained producer that consumes
owner-emitted measurements from all five cases, then qualify only that row. No
product behavior, performance, complexity or debt improvement is claimed.

W149 evidence-binding repair: `RequiredSourceFiles` now includes the recovery
owner test, execution-handler source/test, session start/close routes, and the
Playwright owner integration and idempotency tests. A regression asserts every
J07 test source is listed; the validator test mutates each bound digest in turn
and rejects every stale candidate. `go test -race ./internal/cancellationqualification ./handlers/profilevalidation -count=1`
passes. This closes the evidence-integrity gap only; it does not create the
five-case receipt or advance the setpoint.

Final W149 checks: contract validation reports24 journeys and zero errors;
the targeted input regression and requirements validation pass. Requirements
validation retains existing PRD/traceability notices and reports a stale
requirements-sync snapshot after W148's Test Genie run. Setpoint read
`prog_78c4d9aa-e892-4c25-bce2-1fbb3921bed4` remains2/17 in-band,14 unavailable,
1 out of band and `product_qualified=false`. Live BAS API build identity remains
`a2a0938a5c89776b5dbe86f9dd33ab4ef1902ec0b3babfb02d5ec164d2dfb2ea`; W149's
validator edits are not deployed. No broad suite or Test Genie execution ran.

### BAS-WORK-150 — 2026-09-24 UTC — emit directly measured J07 owner observations

Follow-up to BAS-FB-018: no Test Genie phase or broad suite ran. The focused
setpoint read `prog_80914437-4a56-4aab-ae1a-427dbc6c3e3b` remains2/17 in-band,
14 unavailable,1 out of band and `product_qualified=false`; cancellation is
still L0/Unavailable. The exact rehabilitation provider phase currently reports
failed on build `a2a0938a5c89776b5dbe86f9dd33ab4ef1902ec0b3babfb02d5ec164d2dfb2ea`
because its current-build cancellation receipt is absent.

Added an optional JSONL observation writer to the J07 receipt package. It is a
no-op in ordinary tests and rejects any record that fails the existing case
validator. The real workflow cancellation test and the executor's timeout and
driver-death tests now emit only after their independent fixture/effect,
non-retryable uncertain outcome, terminal status and resource assertions pass.
Each record carries its exact Go owner test and measured stop/cleanup timings.
This emits four observed cases, including a Playwright live-click test where a
same-execution start retry reuses the owner while a competing instruction is
denied. Its independent click log remains exactly one. It does not imply an API
restart case and does not yet create or validate a complete receipt.

With `BAS_J07_OBSERVATIONS` enabled, the selected owner run emitted cancellation
at0.168ms input-stop/0.458ms cleanup, timeout at50.701/0.345ms, driver death at
1.241/0.424ms, and retried start at274.004/12.519ms. These single-run fixture
timings validate the record path only; they do not establish repeatability or
managed-service performance.

Focused verification passed:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 30s ./internal/cancellationqualification ./automation/executor -run '^(TestRecordObservationWritesOnlyValidatedMeasurements|TestRequiredSourceFilesCoverEveryJ07OwnerTest|TestExecuteTimeoutDuringLiveInstructionClosesSessionWithoutReplay|TestExecuteDriverDeathRetainsUncertainEffectWithoutReplay)$' -count=1
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 30s ./services/workflow -run '^TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose$' -count=1
pnpm exec jest tests/integration/typed-action-semantics.test.ts --runInBand --coverage=false --testNamePattern='close keeps a completed click uncertain while denying retry admission'
~~~

The first timeout instrumentation draft incorrectly treated request-context
cancel as the input-stop event; the focused race run exposed that the fixture's
held response is actually released by session close. The assertion now records
the observed close boundary, and its corrected targeted race run passes. No
product behavior, performance, complexity or debt improvement is claimed. The
managed API-restart-during-instruction test and receipt producer/deployment
remain unfinished. The J07 Jest case passes when focused coverage collection is
disabled; without that option the test passes but Jest exits nonzero because
the global function-coverage floor is not met by one selected test. Continue by
finding or building a narrow managed restart harness; do not populate the
missing case from unit test pass status.

### BAS-WORK-151 — 2026-09-24 UTC — add a managed restart owner for J07

Scope extension for BAS-RF-011 / BAS-RH-J07: add one maintained harness under
`api/cmd/cancellation-restart-cohort` because existing Go recovery tests only
run the recovery service in-process and the profile cohort restarts only between
completed profile checks. J07 needs an independently counted external effect
and live resource held across an actual managed API/driver restart. The harness
uses an adhoc BAS workflow against its own loopback fixture and the bounded
`vrooli scenario restart` lifecycle owner; it binds measured results to before/
after managed build identity and does not synthesize observations from test
pass status. Complete the owner chain with a focused Go/Playwright orchestrator
and a receipt assembler that requires exactly one valid measurement for each of
the five cases. Run only the named focused owners, retain the exact receipt, then
execute the single `rehabilitation-evidence` Test Genie phase.

W151 managed restart observation passed on unchanged build
`sha256:e67ec79b58ecb225f69db1243d62a48e4d9c843a56ccffb78f14dd7de903c2f8`.
The independent loopback fixture saw exactly one held external effect, one live
resource before cleanup and zero after, terminal failed execution after API
restart, uncertain effect retained and no replay. Measured cleanup was168ms and
startup/recovery to terminal status was6.611s, both within the J07 bands. The
normal control-plane restart took53.5s end-to-end; its dependency phase consumed
about47.1s before BAS stopped, so the owner records input-to-stop separately
from cleanup/recovery. A preceding attempt correctly discarded its observation
after build identity changed; a too-short harness deadline also abandoned one
restart during dependency setup. The owner now uses the control-plane restart
with a240s ceiling and a single `scenario wait` reattach, and the final measured
run completed without an abandoned operation.

W151 scope extension: add `api/cmd/cancellation-qualification` as the narrow
five-owner orchestrator and receipt assembler. The assembler accepts exactly
one valid measured record for each required case, binds current source/contract
digests and build identity, validates before writing, refuses overwrite, and
restricts output to the rehabilitation evidence directory. Focused race tests
for the receipt package, provider and assembler pass; the orchestrator passes
Prettier and Node syntax checks. These new authored sources change the scenario
build identity and are not deployed yet; rerun the managed restart after the
normal lifecycle refresh, then use the orchestrator to produce the current
five-case receipt. Until the exact provider phase consumes that receipt, the
governed reading remains2/17 and no product-row advancement is claimed.

### BAS-WORK-152 — 2026-09-24 UTC — qualify profile durability and J07 on one candidate

Trigger: operator feedback FB018 asked for fewer Test Genie executions, more
targeted checks and visible score movement. Re-read the feedback before acting.
No source behavior changed in this cycle. The maintained capture workload,
profile cohort, focused owner tests and one exact provider phase were used.

The first profile seed ran before a managed restart and was discarded when the
restart changed build identity from `fd8c050f…` to `da8567a0…`. Inspection
showed that the restart's new identity included the newly retained J07 evidence
file. The cohorts were then ordered so profile state was seeded before the
managed J07 restart. The final owner run reported identical before/after
identity `sha256:5da600a9d2286fc1bbba6de708d1accc9e51590a17eb6054027fea409ff5afbb`.
Generated owner receipt payloads are kept in ignored
`.vrooli/runtime/rehabilitation-evidence/` and exposed at the documented paths
through links, so adding the retained measurements does not change that
candidate's hashed source inputs.

Profile durability passed five seed checks and two post-managed-restart checks;
both synthetic profiles were deleted. The automatic checkpoint appeared after
268.481ms. J07 passed all five focused owners with one independently counted
effect in each case, live resources moving from1 to0, uncertain outcomes
retained, and retry admission denied. The API-restart owner measured1.204s
cleanup and6.261s recovery; its 77.654s input-to-stop includes control-plane
dependency preflight and is not classified as cleanup. Focused race tests and
the selected real-Chromium retry test passed.

Capture workload `53581f9f1b54b6bc1b5568c0918bd0fa` passed100 measured samples
plus one warmup at423ms service p95 and639.747ms wall p95 against2000ms. The
single `rehabilitation-evidence` run `20260924-112014-c3770cf6` passed and
presented both profile durability and cancellation/recovery at L1/Verified; its
evidence tier is degraded because the shared checkout is dirty. Setpoint
`prog_d08ebd17-dcec-436a-9e7b-70f1d067dd84` advanced to3/17 in-band with14
unavailable and `product_qualified=false`. This is one net row above the prior
2/17 reading. No complexity or debt reduction is claimed. Next: measure a
repeat checkpoint trial, abrupt process interruption and shared-profile
ownership conflicts, then target other pending rows with direct measurements.

### BAS-WORK-153 — 2026-09-24 UTC — fail closed on ambiguous profile-scoped live routing

Scope is BAS-RF-026 and the existing profile ownership boundary. The
W3 finding is recorded in `docs/PROBLEMS.md` and the architecture now states
that profile-scoped live operations must conflict if more than one active
browser is bound. This preserves multiple active bindings and avoids selecting
an arbitrary target; no admission policy or saved profile format changes.

Baseline: `ActiveSessionRegistry.GetByProfile` iterates a map and returns its
first match. Four recordings operations use that result for service-worker
queries/mutations or history navigation. The discriminating regression will
prove that ambiguous lookup returns an error and that none of the four driver
operations dispatches; it will also retain the existing unique and absent
binding behavior. Before source repair, run only these owner tests. No
Test Genie or broad suite is planned for this localized defect.

Repair is in source. The registry now resolves a profile only when it has one
active browser; multiple bindings return `ErrAmbiguousProfileSession` and remain
registered. The recordings service maps that error to Connect
`FailedPrecondition`, and all four profile-scoped live operations share this
mapping. The previous `GetByProfile` / `GetSessionForProfile` arbitrary reverse
lookup path is removed. Regression tests route all four RPCs through the real
session-profile service with two active bindings, verify no driver method runs,
and verify both bindings remain. Registry tests cover zero, one and multiple
bindings.

Targeted validation passed:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 60s ./services/session-profile ./handlers/recordings -count=1
~~~

The command passed both packages. `gofmt`, legacy-lookup `rg`, and `git diff --check`
also pass. No debt or complexity reduction is claimed: one ambiguous
reverse-lookup path was replaced by typed resolution and a shared RPC error
mapper, preserving established no-session and single-session behavior. The
managed BAS process still serves the pre-repair candidate; restart it, rerun
the build-bound capture/profile/J07 owners and exact provider phase, then read
the setpoint before counting the repair as deployed.

Deployment and current-candidate follow-up: managed restart operation
`startop-a735914f626841d280a2b5c2b06ca1f5` succeeded healthy and served build
`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`.
The targeted race packages passed again on the edited source. The J07 owner
execution `76be3dd2-f42d-47fd-a12c-3c90661353ac` passed on an unchanged
before/after build; its restart observation had one effect, resources1→0,
42.991ms cleanup and4.161s recovery, with uncertain effect retained and retry
denied. Profile owner `profile-durability-155-2026-09-24.json` passed five seed
checks and two post-restart checks, checkpoint visible at1357.173ms, and
deleted both temporary profiles. Capture workload
`c74bcbd2641bdede6412e02e3c06880d` passed100 samples plus warmup at470ms p95
against the2000ms budget.

Exact provider phase `20260924-114026-b6e72722` passed L1/Verified for profile
durability and cancellation/recovery on the same build. Setpoint
`prog_000cfd42-2202-4c7f-8c83-a57919c678c6` remains3/17 in-band,14 unavailable,
and `product_qualified=false`; RF-026 correctness improved without creating a
new rehabilitation row. Phase evidence tier remains degraded because the
shared checkout is dirty. No complexity/debt reduction or end-to-end runtime
RPC deployment assertion is claimed. Next, address a known data-loss issue
with a narrow owner regression rather than broaden the suite.

### BAS-WORK-154 — 2026-09-24 UTC — verify recording delivery acknowledgements

Trigger: continue the rehabilitation goal and user feedback FB018: prefer
targeted owner checks over frequent broad Test Genie runs. BAS-RF-022 reported
that callback HTTP errors could count as success and timestamp collisions could
acknowledge multiple observations.

Current source already has the required delivery safeguards: callback responses
must be successful and return the exact committed `entry_id`; recording buffers
key by entry ID, reject conflicting replays, retain pending entries on rejection,
retry in order, and acknowledge only explicitly named IDs. The existing suite
already covers callback error/mismatch receipts, retry after rejection, and
multiple observations with the same sequence number. Added a focused navigation
regression: two same-sequence observations survive a successful navigation, and
acknowledging one ID leaves the other visible.

Targeted validation passed: Playwright-driver Jest owner files
`routes/record-mode.test.ts`, `routes/recording-lifecycle.test.ts`, and
`idempotency/buffer-idempotency.test.ts` pass 164/164 tests. Go API tests
`TestReceiveRecordingActionRequiresCommit` and
`TestPullRecordingActionsCommitBeforeAcknowledgement` pass. `git diff --check`
passes for BAS-owned files. The workspace-wide check also reports trailing
whitespace in unrelated `scenarios/audio-tools/ui/index.html`. No Test Genie
phase or broad suite ran, and no runtime source changed.
RF-022's described behavior is repaired and regression-covered; a maintained
full-script producer remains the stated qualification limit. The setpoint read
is recorded below; this repair does not create a new maturity row.

Setpoint program `prog_e017918e-4765-4d06-80ac-a96acd4eee18` succeeded at
3/17 in-band,14 unavailable, `product_qualified=false`. Capture remained within
its 2,000ms budget at470ms p95; profile durability and cancellation/recovery
remain L1/Verified on the existing candidate. No score row changed.

### BAS-WORK-155 — 2026-09-24 UTC — verify execution writer retention repair

Trigger: after RF-022, continue through the BAS-RF data-loss/retention register
with focused tests. Search and prior work records identified RF-023 as a
previously repaired writer-ownership issue whose register row had not been
refreshed.

Confirmed in source: `FileWriter.ForgetExecution` deletes all three per-execution
accumulators (results, proto timelines, and settings); workflow completion and
archive ingestion call this terminal owner after their last write. Archive
ingestion defers cleanup, including cancellation and write-failure exits. Prior
16- and 64-execution on-disk storage cohorts measured retained heap after
cleanup, while retaining the writer, at roughly 1.1–1.4KB/execution and
270–285B/execution respectively. Persisted timeline files and another active
execution are preserved.

Focused race validation passed for writer durable projection and cleanup, and
archive release on all exits:

~~~bash
GOTOOLCHAIN=local GOPROXY=off go test -race -timeout 60s ./automation/execution-writer ./services/archive-ingestion -run 'TestForgetPreservesDurableAndOtherActiveExecution|TestStructuredOutcomeSurvivesDiskProjection|TestFrameImportReleasesWriterAfterEveryExit' -count=1
~~~

Active-run accumulation bounds and late-writer fencing remain explicitly open;
this work closes the previously repaired terminal-retention behavior only. No
runtime source changed, no Test Genie phase or broad suite ran, and no new
rehabilitation row advanced. The latest setpoint remains 3/17 with14 unavailable.

### BAS-WORK-156 — 2026-09-24 UTC — order and acknowledge live input

Trigger: honor FB-018's request for focused owner checks and continue the
highest-impact input lifecycle work in BAS-RF-020. Read the issue, architecture,
contract, testing protocol and operator feedback. The shared checkout was already
dirty across BAS runtime, tests and evidence; existing work was preserved.

Hypotheses and discriminating results:

1. Concurrent live input can overtake a button transition while the earlier
   route awaits a browser move. Confirmed with a maintained delayed-down/up
   regression: before the per-page queue, `up` could apply before `down`.
2. The Go WebSocket read loop already awaits each forwarded event. Confirmed in
   source and its connection-order owner test; this serializes one socket but
   does not order direct concurrent driver requests.
3. A driver-owned per-page admission queue was already shared by HTTP callers.
   Falsified by source search: each handler invoked independent Playwright
   operations, so they could interleave.

The driver now queues operations per page with a64-item pending bound and a
128-caller cap per coalesced motion. Adjacent pointer moves with matching
modifiers collapse to the latest coordinates; discrete transitions remain
ordered. Applied sequence/coalescing receipts flow from driver to API HTTP
responses and WebSocket acknowledgements. Targeted owner tests assert delayed
down/up order, final-motion coalescing, queue overflow rejection, failed-down
release, API receipt propagation and WebSocket receipt ordering. The architecture
and RF-020 register describe the ownership and remaining qualification limits.

Changed paths owned for this issue: the Playwright driver live-input route and
its focused unit test; Go driver/session/live-capture and handler receipt
signatures and tests; `websocket/hub.go` and its owner test; architecture,
`PROBLEMS.md`, this checkpoint and the feedback ledger. Other pre-existing
shared-checkout changes were preserved.

Validation:

~~~bash
pnpm exec jest tests/unit/routes/record-input.test.ts --runInBand --coverage=false
pnpm exec tsc --noEmit
pnpm exec eslint src/routes/record-mode/recording-input.ts tests/unit/routes/record-input.test.ts
go test -race ./automation/driver ./automation/session ./services/live-capture ./handlers
go test -race ./websocket -run '^TestRecordingInputPreservesConnectionOrder$' -count=1
python3 scenarios/browser-automation-studio/docs/internal/refactor_contract.py
python3 scenarios/browser-automation-studio/docs/internal/refactor_inventory.py --include-untracked
program-runtime library run browser-automation-studio.setpoint-read --input profile=rehabilitation
program-runtime programs wait prog_a0db5308-92a3-49b1-aed8-886228af0bef --timeout 300s --json
~~~

Driver owners pass29/29; typecheck and focused lint pass. Driver/session,
live-capture and handler Go packages pass under `-race`; the WebSocket applied
receipt/order regression passes separately under `-race`. Contract validation
passes for17 outcomes and24 preservation journeys; scoped `git diff --check`
passes. The first WebSocket receipt assertion read the connection greeting
before the expected receipt; the test was corrected to consume the greeting
before sending input, then the exact regression passed. This was a test-oracle
ordering correction, not product behavior evidence. The inventory completed but
reports a broad pre-existing shared worktree; this cycle makes no complexity or
runtime-size reduction claim.

Setpoint program `prog_a0db5308-92a3-49b1-aed8-886228af0bef` completed. It
reports3/17 in-band and14 unavailable, `product_qualified=false`. Capture is
470ms service p95 /681.692965ms wall p95 on build
`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`;
profile durability and cancellation/recovery are L1/Verified and clean, with
degraded Test Genie evidence tier due to the dirty checkout. RF-020 does not map
to a separate setpoint row. No Test Genie phase, broad suite, restart, or
deployment ran. The managed BAS scenario remains healthy; its running identity
predates the local input changes.

Unverified limits: cancellation of an admitted browser call may leave key or
button state uncertain; reconnect recovery and multiple-button cleanup are not
qualified. The WebSocket acknowledgement uses the existing bounded send queue;
when that queue is full it logs and drops the receipt, so consumers must treat a
missing receipt as unknown rather than success. No structural simplification
was measured. Next useful action: add driver-owned cancellation/reconnect state
cleanup tests and define whether dropped WebSocket acknowledgements require a
recoverable receipt query before claiming end-to-end receipt reliability.

### BAS-WORK-157 — 2026-09-24 UTC — release remote pointer on UI loss

Trigger: continue BAS-RF-020's held-input recovery after W156. Re-read the
testing protocol, goal, contract, architecture, progress, feedback and issue
register. Search-hub found no scoped BAS record for this exact edge; its
required `prompt-manager discover` fallback surfaced general continuity skills
but no direct BAS input owner precedent. Source inspection was decisive.

Hypotheses and checks:

1. Pointer-up can be lost when the physical pointer exits the viewer, because
   the viewer only receives events that target its own element. Confirmed by
   source inspection: its handler was attached to the viewer container and did
   not capture the pointer; a window-owned handler is needed for outside-up.
2. Window blur or WebSocket disconnect can leave the remote Playwright mouse
   pressed. Confirmed as an uncovered behavior with the new blur test before
   implementation; the connection-loss test initially had a fixture error
   (`fetchInput` was not defined), which was corrected before using it as
   product evidence.
3. HTTP fallback after WebSocket loss is guaranteed to arrive after the last
   accepted WebSocket message. This is not established: the transports have no
   shared client sequence or event ID, so it remains an explicit unknown.

The input hook tracks held physical pointer buttons, their last mapped browser
coordinates, modifier keys and pointer IDs. It sends ordered pointer-up
operations on global pointer-up, pointer-cancel, window blur and the transition
from connected to disconnected. Synthetic releases preserve modifiers through
the remote mouse-up, then the driver route releases them. This repairs common UI
focus/transport-loss cases and retains ordinary pointer semantics.

Focused test-first validation: the first targeted run reported8/10, including a
real missing-blur-release assertion and the test fixture error noted above. After
the hook and fixture repair, `pnpm exec vitest run src/domains/recording/capture/useInputForwarding.test.ts`
passes11/11. The tests
cover last coordinates/modifiers on blur, right-button HTTP fallback on socket
loss, outside pointer-up and pointer-cancel, plus existing chord/text/page
selection behavior. UI `pnpm exec tsc --noEmit` and focused ESLint pass.

Changed paths owned for W157: the UI input hook and its owner test, architecture,
BAS-RF-020, this progress record and the FB-018 follow-up. No runtime was restarted, no service deployed,
and no complexity reduction is claimed. The contract validator and inventory
run, plus the required rehabilitation setpoint read, are recorded below.

Remaining limits: client-side recovery is best-effort when the connection
silently partitions; socket-loss release switches from WebSocket to HTTP, whose
arrival can race a previously buffered WebSocket event. The driver still has no
explicit release-all operation for session cancellation, and an applied
WebSocket receipt can still be dropped when the existing send queue is full.
Keyboard chords use Playwright's atomic `press` API but ambiguous browser
failure remains unqualified. Next work should unify the release with the
driver-owned input queue and make receipt recovery possible after reconnect,
then test close/cancel during an admitted down operation.

Required read: `program-runtime library run
browser-automation-studio.setpoint-read --input profile=rehabilitation` followed
by one wait on `prog_1c6e12d1-54f9-4dae-b7ba-f3e56afd94d8`. The read remains
3/17 in-band,14 unavailable, `product_qualified=false`; capture is470ms p95
(681.692965ms wall p95), and profile/cancellation receipts remain L1/Verified
with degraded evidence tier. Contract validation passes17 outcomes and24
preservation journeys; the source inventory completes but reflects the broad
pre-existing dirty checkout. No score row advanced from W157.

### BAS-WORK-158 — 2026-09-24 UTC — make live-input retries idempotent

Trigger: continue RF-020 after W157 left HTTP fallback able to race a buffered
WebSocket input. Operator feedback asks for direct owner tests and less time in
Test Genie. No Test Genie execution ran this cycle.

The working hypothesis was that a stable client input ID can close this race if
the driver records the in-flight promise before applying browser effects and
retains a bounded window of completed receipts. A repeated ID must return the
same receipt without a second browser effect; a different payload under that ID
must fail with conflict. A second hypothesis was that the UI can recover a
dropped acknowledgement by replaying pending WebSocket inputs over HTTP in
original send order before synthetic pointer releases.

Implemented `input_id` across UI payloads, driver HTTP receipts, Go response
types and WebSocket acknowledgements. The driver keeps per-page receipt state
for up to256 entries, deduplicates both pending and completed IDs, and rejects
payload conflicts. The UI removes pending IDs on matching WebSocket receipt;
after connection loss it sequentially retries pending IDs over HTTP, then
releases currently held pointer buttons. The receipt path requires matching
`input_id` and a nonzero applied sequence for HTTP success.

Focused validation: driver input route tests pass31/31, including duplicate
click delivery and conflicting reuse; UI input owner tests pass12/12, including
same-ID replay before the release and suppressing replay after its WebSocket
receipt; Go race tests pass for
`automation/driver`, `websocket` and `services/live-capture`. Driver and UI
TypeScript checks and focused ESLint pass. Contract validation passes17
outcomes and24 preservation journeys. Scoped `git diff --check` passes; the
unscoped repo-wide check reports pre-existing trailing whitespace in
`scenarios/audio-tools/ui/index.html`, outside this change. The inventory
completes and confirms the broad dirty shared worktree.

The setpoint program completed with3/17 in-band and14 unavailable,
`product_qualified=false`. Its capture and owner receipts belong to managed build
`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`,
which predates these local changes; RF-020 is not a separately scored row, so
no score advancement is claimed. No restart or deployment ran.

The initial focused run found two integration mismatches: conflict errors were
mapped to HTTP500 rather than409, and a UI test required exact payload equality
rather than allowing the new receipt ID. Both were corrected; the repeated
focused run passed. Remaining boundaries are loss of pending state on a full UI
reload, retries after driver receipt eviction, and driver-owned release after
session cancellation. The next cycle should target an independently measurable
unavailable rehabilitation row instead of adding more RF-020 scope.

### BAS-WORK-159 — 2026-09-24 UTC — measure local driver-to-paint latency

Trigger: follow FB-018's request for focused evidence and W158's direction to
work on a scored row. Search-hub returned no scoped BAS evidence for the
interactive-feedback producer; the required `prompt-manager discover` fallback
identified the BAS improvement skill but no existing input-to-paint owner.
Existing frame `e2e_*` fields were inspected and their contract explicitly
states they omit network transit and paint.

Hypothesis: a maintained real-Chromium cohort can measure a truthful lower
boundary for input responsiveness if it pairs the driver's applied receipt ID
with an independent fixture paint sentinel and calibrates Node/Chromium
monotonic clocks. The discriminating check is 1000 sequential pointer moves
through `handleRecordInput`, each producing one ordered receipt and one matching
post-mutation paint marker.

Added `playwright-driver/tests/integration/input-feedback.test.ts`. The first
test attempt timed out waiting for `page.setContent` before sending an input; it
produced no product evidence. Switching the fixture to the same explicit data
URL navigation pattern used by existing integration tests fixed setup. The
driver's quiet test hook also suppressed the first passing run's metrics, so the
same targeted test was rerun with `BAS_JEST_VERBOSE_LOGS=1` and passed with
1000/1000 correlated inputs, strictly increasing applied sequences and maximum
clock-calibration uncertainty0.321ms. Measured local driver-route-to-paint
latency: p50 31.624ms, p95 32.426ms, p99 32.794ms. The full sample array and
source/contract hashes are retained in
`docs/internal/evidence/rehabilitation/interactive-feedback-driver-2026-09-24.json`
with receipt hash `a64843a050489d3ce19c0b71901e15502c55da0f913f4c478b040822cda5897b`.
The passing producer command is
`BAS_JEST_VERBOSE_LOGS=1 pnpm exec jest tests/integration/input-feedback.test.ts --runInBand --coverage=false`.
This is the first retained local driver-only cohort, so no before/after latency
improvement is inferred.

The documented isolated input probe first returned `unavailable` because its
synthetic React/WebSocket mocks did not expose the hook APIs added in W158.
Direct execution showed `useWebSocketMessage is not a function`; adding the
missing no-op hook/effect adapters restored the probe, which now passes all
eight observations, including nonempty `input_id` coverage. Driver typecheck
and focused ESLint pass. The rehabilitation contract validator passes17
outcomes and24 preservation journeys; scoped whitespace validation passes and
the inventory reports157 dirty scenario entries in the shared worktree. The
probe command is `python3 docs/internal/refactor_regressions.py --case input`;
the other direct gates were `python3 docs/internal/refactor_contract.py`,
`pnpm exec tsc --noEmit` and focused ESLint on the changed benchmark.

Required setpoint read `prog_052b6a75-ee32-421f-8b96-a57403344047` completed
with3/17 in-band,14 unavailable and `product_qualified=false`. The governed
capture, profile and cancellation receipts still identify managed build
`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`;
the new diagnostic is local source evidence and does not satisfy a governed row.
No new Test Genie run, runtime restart or deployment occurred.

This cohort excludes Go API ingress, WebSocket transit, UI event handling,
streamed-frame transfer, image decode/canvas draw and remote networking; its
double-animation-frame marker is not a compositor pixel capture. It is a
diagnostic only and cannot qualify the interactive-feedback row. No RF-009
closure, setpoint advancement, complexity reduction, Test Genie execution,
managed restart or deployment is claimed. The new benchmark and embedded raw
samples increase test/evidence source size; no runtime simplification was
measured.

The next useful action is to extend the same correlated fixture through the
actual API/WebSocket path and UI decode/draw, then create an independently
controlled remote cohort. Keep those as separate stage readings; do not add
frame-processing sums and call them input-to-paint.

### BAS-WORK-160 — 2026-09-24 UTC — correct duration p95 rank

Trigger: respond to operator feedback asking for more targeted tests and less
reliance on Test Genie while continuing work on BAS-RF-009. The assessment
identifies `aggregateSQL`'s floor-based p95 rank as incorrect for small samples.

Added a SQLite-backed test that sends 1, 10 and 21 ordered durations through
the production SQL aggregate via `coredb.RoutedDB`. Before the fix, the 10-row
case returned the 9th duration (~8999.991ms) instead of the nearest-rank 10th;
the 21-row case returned the 19th (~19000.018ms) instead of the 20th. Changed
the rank to `ceil(0.95*n)` using integer arithmetic. The new regression and the
full `handlers/measures` package both pass. This repairs the reported percentile
selection behavior; it does not qualify the separate input-to-paint cohort.

No Test Genie phase, managed restart or deployment ran. The last verified
required read. The single required read `prog_1c0b6c82-f937-40bf-b385-4675603923b3`
completed successfully and remains3/17 in-band,14 unavailable, with
`product_qualified=false`; its readable rows are capture, profile durability
and cancellation recovery on build `sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441`.
Remaining RF-009 work includes cohort isolation, empty/window semantics and
real API/WebSocket/UI input-to-paint measurements. The next
performance action remains extending a correlated local cohort through the
actual API, WebSocket and frame display path; do not infer paint from server
processing intervals.

### BAS-WORK-161 — 2026-09-24 UTC — isolate execution-duration p95 cohort

Trigger: continue BAS-RF-009 with a direct targeted test following the operator's
request to prefer focused owner checks. `ExecutionDurationP95` is declared as
the p95 duration of completed browser executions, but `aggregateSQL` ranked
every non-null execution duration, including failed runs.

Added a mixed-status SQLite regression through `SQLRepository.Aggregate` and
`coredb.RoutedDB`: completed durations of 1s and 2s, one failed execution at
100s, and one still-running execution. Before the query fix, the reported p95
was ~99,999.981ms instead of 2,000ms. Restricted the ranked CTE to completed
executions. The regression, W160 nearest-rank cases and entire
`go test ./handlers/measures` package now pass. Terminal counts still include
completed plus failed; the p95 population now follows its declaration.

No Test Genie phase, managed restart or deployment ran. Required setpoint read
`prog_fe5a0e97-873f-41fe-b374-77fe7ac1aaf8` completed successfully with3/17
in-band and14 unavailable; only capture, profile durability and cancellation
recovery are readable/in-band, and `product_qualified=false`. The measure fixes
do not qualify another row. Other RF-009 gaps remain: empty-result meaning,
window eviction/cohort reporting, processing intervals mislabeled as e2e, and
actual input-to-viewer paint across API/WebSocket/UI plus a remote cohort.
Continue on the real transport/viewer measurement path; do not claim row closure
from these measure-only regressions.

Transport inspection refined that next action: the driver's live frame envelope
contains source identity, capture wall time and optional processing timings, but
no applied-input sequence or ID (`playwright-driver/src/frame-streaming/frame.ts`).
The API's `viewerFrame` strips even those processing timings, and the UI frame
identity is only session/page/capture time (`api/handlers/record_mode_frames.go`,
`ui/src/domains/recording/capture/useFrameStream.ts`). The current wire cannot
directly join an applied input receipt to its displayed frame. The next targeted
experiment should carry a fixture-visible input marker through the existing
driver capture, API WebSocket and canvas, and verify the marker in the displayed
frame before reporting input-to-paint latency; consider adding protocol sequence
metadata only if the visual-marker path cannot robustly correlate samples.

### BAS-WORK-162 — 2026-09-24 UTC — correlate input receipts to captured pixels

Trigger: continue RF-009 from W161's identified live-frame correlation gap.
Hypothesis: the existing real Chromium CDP screencast can carry a unique visible
fixture marker for each successfully applied input, without adding sequence
metadata to the production frame protocol. The discriminating check is a
1,000-input real-browser cohort that validates each driver receipt, decodes the
JPEG frame received over the driver WebSocket, and matches its 10-bit visual
marker to that input before admitting a sample.

Extended the existing targeted Playwright-driver integration producer in
`playwright-driver/tests/integration/input-feedback.test.ts`. A 100-input smoke
run passed before the 1,000-input cohort. The final focused run passed with
1,000/1,000 matching pixels, strictly increasing applied receipt sequences and
real Chromium 136.0.7103.25 at 800x600 DPR1. Node-monotonic latency from driver
route submission to receiving/decoding the matching streamed JPEG pixel was
p50 33.627ms, p95 34.892ms, p99 36.022ms (min13.560ms, max55.364ms). Raw samples,
method, platform, contract/source hashes and identity are retained in
`docs/internal/evidence/rehabilitation/interactive-feedback-driver-frame-2026-09-24.json`.

This is stronger evidence than the prior fixture-only double-RAF sentinel: the
actual driver CDP screencast and binary WebSocket delivery carried each visible
marker. It remains a partial diagnostic; Go API relay, the BAS `useFrameStream`
decode/draw path and remote networking are excluded, so it earns no contract
credit. No production protocol metadata was added. `pnpm exec tsc --noEmit`
passes. Focused ESLint exits successfully with14 warnings from its `ws` server
typing surface (`WebSocket.Server` resolves as `any` in the lint project); this
matches the warning pattern in the existing driver pipeline integration owner.
The attempted root-relative inventory command initially used the scenario-local
path and failed; rerunning it at
`python3 scenarios/browser-automation-studio/docs/internal/refactor_inventory.py --include-untracked`
from repo root succeeded with160 dirty scenario entries. Contract validation
reports17 outcomes,24 journeys, `product_qualified=false`. Required setpoint read
`prog_9701f1b8-819e-4da1-b4a8-4a0a3bde8f94` completed successfully:3/17 in-band,
14 unavailable, with only capture, profile durability and cancellation recovery
readable/in-band; `product_qualified=false`. No Test Genie phase, managed restart
or deployment ran.

The work adds a targeted integration test and a retained 1,000-sample receipt;
it does not establish net code-complexity reduction or latency improvement over
a comparable baseline. Next, use the same visible-marker oracle through the
actual Go API frame relay and BAS viewer canvas, then repeat for the remote
cohort. Keep API transit, viewer decode/draw and network timing separable.

### BAS-WORK-163 — 2026-09-24 UTC — carry input markers through the BAS viewer

Trigger: continue RF-009 from W162's identified API-relay and viewer-canvas
evidence gap, while following the operator's request to spend effort on focused
tests instead of repeated Test Genie runs. The test uses a disposable managed
BAS recording session, a fixture page that increments a visible 10-bit marker
on pointer input, the real workspace UI and canvas, and CDP observation of
input IDs/applied receipts. The hypothesis is that every UI input can be joined
to both its driver acknowledgement and the corresponding pixels after the Go
frame relay and UI decode/draw.

Added an opt-in, focused 1,000-input integration test to
`playwright-driver/tests/integration/input-feedback.test.ts`. It is enabled by
`BAS_REHAB_LIVE_API_BASE` and `BAS_REHAB_LIVE_UI_BASE`; no new Test Genie phase
was run. Three complete local cohorts each correlated 1,000/1,000 input IDs,
applied receipts and viewer-canvas markers with monotonically increasing
receipt sequences. Their p50/p95/p99 results were 49.988/54.034/67.476ms,
50.001/54.524/67.034ms, and 51.000/96.676/201.505ms. The variation crosses the
local p50 <=50ms and p99 <=200ms limits, so the local cohort is not reliably
in-band. The managed runtime was healthy at build
`sha256:5be0eb798a3cca7f91b373c92d10821d9603a233b7e088f9c421f6ae5617bcfd`.
Complete samples, source and contract hashes, method and environment are in
`docs/internal/evidence/rehabilitation/interactive-feedback-live-2026-09-24.json`.

The targeted Jest owner passes; focused ESLint exits 0 with the existing `ws`
typing warnings. The live path works and is now measured through the real Go
input forwarding, driver capture, Go frame relay and BAS canvas. These are local
loopback cohorts; the independent remote cohort remains unavailable, so RF-009
is not qualified. The p50/p99 spread needs component timing and a controlled
load check before an implementation change is chosen. The resulting test and
embedded 1,000-sample evidence add source and artifact size; no domain-wide
simplification or reliable latency improvement is claimed.

Required rehabilitation setpoint read
`prog_8b75892e-8675-4a60-851e-2fbbb2c862e4` reports 0 readable, 17 unavailable,
and `product_qualified=false`. The capture workload receipt names the prior
managed build; profile-durability and cancellation-recovery lack current
candidate-phase receipts; the other 14 rows are pending telemetry. The new
live-input receipt is retained but is not yet a governed sensor binding, and
its latest local repeat is outside the p50/p99 limits.


### BAS-WORK-164 — 2026-09-24 UTC — correct the input feedback event clock

Trigger: continue RF-009 after W163 showed local p50/p99 values near or outside
the band, while honoring the operator's request for focused owner testing.
Hypothesis: W163 started timing before Playwright delivered the pointer event
to the workspace, adding automation dispatch overhead; a single browser-page
clock around event capture, socket send/ack and canvas pixels will isolate the
actual user-visible path.

The focused 1,000-input test passed with 1,000/1,000 applied receipts matched
to visible canvas markers. The corrected local p50/p95/p99 are 36.70/39.30/40.00
ms, within the local contract band. The p50 stage split is 0.10ms pointer to
socket send, 1.90ms send to applied receipt, and 34.80ms receipt to canvas
pixels; this places most observed latency in frame delivery/decode/draw. The
cohort uses `performance.now()` in the BAS workspace page for all timestamps,
so Playwright dispatch before actual pointer capture is excluded. Evidence:
`docs/internal/evidence/rehabilitation/interactive-feedback-browser-clock-2026-09-24.json`.

Targeted Jest passed on the managed build
`sha256:5be0eb798a3cca7f91b373c92d10821d9603a233b7e088f9c421f6ae5617bcfd`;
no Test Genie phase or runtime restart ran. This replaces W163 command-start
timings for local-band decisions; those earlier samples remain diagnostic.
Performance Health's declared workload receipt measures p95 for a single
request workload and cannot represent this cohort's p50/p95/p99, input/receipt
correlation and canvas-pixel evidence. The local result therefore remains
unscored until an owner-governed sensor accepts the cohort, and the remote
cohort is still unmeasured. The required setpoint read is recorded below after
refresh. No structural complexity reduction or full RF-009 qualification is
claimed.

W164 required setpoint read `prog_a3fded3d-a2e0-4b90-a27d-0cefc8001b14` completed
successfully: 0 readable, 17 unavailable, `product_qualified=false`. The
interactive-feedback row remains `pending_telemetry`; this confirms that the
owner's local cohort is not yet an authoritative sensor. The capture receipt is
also stale for the current managed build (`sha256:ea4c369c7e90c21fb552915773c5abf9d3b1418332b831527913c9eacc20a441` versus live
`sha256:5be0eb798a3cca7f91b373c92d10821d9603a233b7e088f9c421f6ae5617bcfd`).

The capture row was then refreshed through the declared Performance Health owner:
operation `7e0bd077c9178aa40a95bf5ec0e17331` completed for the current managed
build with 100 samples plus one warmup, p95 401ms, wall p95 577.59ms, within
the 2000ms budget. A second required setpoint read,
`prog_a3ac3b14-c3ff-4dec-b39f-48fd6932c92e`, now reports 1/17 in-band, 16
unavailable, `product_qualified=false`. Capture is the one readable row;
interactive feedback remains pending telemetry and the other rows remain
unqualified. No Test Genie run was used.


### BAS-WORK-165 — 2026-09-24 UTC — refresh build-bound owner evidence

Trigger: continue the production-readiness goal and respond to the operator's
request for visible progress from targeted checks. W164's final capture refresh
left the setpoint at 1/17; profile durability and cancellation/recovery receipts
were stale against the current candidate.

The single exact `rehabilitation-evidence` run
`20260924-142423-2d020899` exposed the cause: its only findings were
`PROFILE_EVIDENCE_INVALID` (no retained profile cohort for the live build) and
`CANCELLATION_EVIDENCE_INVALID` (latest receipt build mismatch). The online
Bridge node `minimouse` is a possible remote Mac target, but its authorized relay
refused the read-only `scenario status browser-automation-studio` call with
`permission_denied`: missing `vrooli:read`. No grant change or shell bypass was
attempted; the remote cohort remains unmeasured.

A first profile seed ran on build `sha256:5be0eb798a3cca7f91b373c92d10821d9603a233b7e088f9c421f6ae5617bcfd`; the required managed restart advanced to
`sha256:d8694c89b82a07446f7654007c6bc2335916fef26217cc560b9f392f084dd843`.
The two synthetic profiles survived and were cleaned, but that cross-build seed
was discarded for candidate qualification. Reseeding on `d8694c…` and reusing
the cancellation cohort's managed restart preserved the build identity. All five
cancellation cases passed their receipt assertions, including one-effect
uncertainty, resource cleanup 1→0 and API recovery in 7.03s; both profiles
restored after restart, checkpoint appeared in 1699.86ms, and both synthetic
profiles were deleted. Retained source/contract-bound profile evidence is
`docs/internal/evidence/rehabilitation/profile-durability-164-2026-09-24.json`;
the current cancellation receipt is
`docs/internal/evidence/rehabilitation/cancellation-recovery-2026-09-24T14-30-28-073Z-8607df6c.json`. Focused Go race packages pass:
`go test -race ./internal/cancellationqualification ./handlers/profilevalidation -count=1`.

The current-candidate Performance Health capture workload
`3844f1520cd43358ab6aed6821b8d77f` passed 100 samples plus one warmup at p95
439ms / wall p95 645.76ms against 2000ms. The one narrow provider-phase run
`20260924-143442-aecb19d9` passed L1/Verified for profile durability and
cancellation/recovery. Setpoint
`prog_bc4acaa1-0927-4950-a2cd-80a79e925ff0` now reports 3/17 in-band, 14
unavailable, `product_qualified=false`, on the same managed build
`sha256:d8694c89b82a07446f7654007c6bc2335916fef26217cc560b9f392f084dd843`. The
local RF-009 cohort remains measured in-band but unscored because its governed
sensor and independent remote cohort are still missing. No broad Test Genie run
or structural-debt reduction is claimed.

Next: pursue the remote input-to-paint cohort only through an owner-authorized
Bridge scenario-read path or another declared remote test route; keep its score
unknown while that access is absent. Continue targeted work on independent
pending outcomes without re-running the same stale checks.

W165 validation amendment: the focused preparation/contract test initially
failed on two obsolete test assumptions after the current-candidate exact-phase
join was introduced: it still looked up the removed `qualifying_profile_run`
helper, and its test harness omitted the governed `vrooli.scenario.status`
binding, so a correct binding error changed the expected board status to
`partial`. Updated the harness with a current-build status response and tested
`latest_rehabilitation_run` directly. The revised assertions verify freshness,
exact single-phase selection, reject non-terminal runs, and preserve per-capability
readability when an overall phase is failed. That last behavior is required
because a profile capability can pass independently while cancellation evidence
fails. `python3 scenarios/browser-automation-studio/docs/internal/test_refactor_contract.py`
now passes 10/10; `refactor_contract.py` validates all 17 outcomes and 24
preservation journeys with zero errors. This was stale test maintenance; no
runtime source, public behavior, or complexity claim changed.


### BAS-WORK-166 — 2026-09-24 UTC — current-build idle and capture memory

Trigger: investigate RF-010 against the current managed candidate, following the
operator's request for faster progress through focused evidence and fewer broad
Test Genie runs. Hypothesis: the historical multi-GiB Go heap/goroutine/swap
report would reproduce in current API/driver processes or remain elevated after
the existing capture owner workload.

On unchanged managed build `sha256:d8694c89b82a07446f7654007c6bc2335916fef26217cc560b9f392f084dd843`,
a60-second no-workload sample (60 samples/process) found API PSS42,793–43,168KiB,
swap3,260–3,292KiB, Go health heap11.39→12.16MiB and goroutines32→33. Driver
PSS was44,762–45,606KiB and swap34,436–34,476KiB. The run records per-process
CPU ticks and PSS rather than attributing host-wide swap to BAS.

Then ran the declared Performance Health `capture` owner once:100 measured
samples plus one warmup,437ms service p95 and639.845ms wall p95 against a
2,000ms budget, operation `b5e265fd3d01da3e5536d4e598d3b597`, receipt SHA256
`56c2a7f59a5e564f75b68173c9c7a77432a94ebf43c16a2545aac51f07b99351`. During
the59.285-second cohort, 557 process samples recorded API/driver PSS peaks of
82,324/98,381KiB. The following60 one-second recovery samples remained nearly
flat at82,260–82,269/98,116–98,134KiB, about176.2MiB combined and below the
300MiB idle bound. API health remained good at22.74MiB heap and34 goroutines;
the observed process swap fell to1,560KiB API and about18.8MiB driver. API PSS
was already higher before this workload than in the earlier idle sample, so the
difference cannot be attributed to this capture cohort or called a leak.

The follow-up setpoint read initially returned1/17 because the just-refreshed
capture receipt was newer than the previous profile/cancellation phase run.
The sensor requires that exact composite phase to complete after the selected
capture. One exact `rehabilitation-evidence` phase on the same build then passed
in1s (`20260924-145319-7ef87a94`, L1/Verified for both owner capabilities), and
the required read `prog_a55e3914-41bc-4edb-9d8b-44ef2d6d25e9` returned3/17
in-band with14 unavailable and `product_qualified=false`. Evidence tier remains
degraded for the shared dirty checkout. No broad Test Genie suite ran, no source
or runtime code changed, and no complexity reduction is claimed.

RF-010 now says the old multi-GiB reading was not reproduced and that this
workload's post-capture PSS is below the numerical band, while keeping the row
open for the governed producer, fixture-browser memory, longer soak and
historical attribution. The resource-budget setpoint remains unavailable because
this direct owner measurement is not its qualification receipt. Evidence:
`internal/evidence/rehabilitation/runtime-memory-idle-2026-09-24.json`.

Next: focus an unavailable outcome with a real owner producer/sensor path, and
avoid refreshing already-valid receipts unless new evidence changes their
binding. Keep the macOS remote input-to-paint cohort unscored until its owner
read route is authorized.


### BAS-WORK-167 — 2026-09-24 UTC — repair empty recorder snapshots

Trigger: move from W166 measurement to a focused high-impact recording behavior
repair, following the operator's request for targeted owner tests. RF-002,
RF-003 and RF-024 were still marked open in the issue table despite W014's
repository/profile repairs. Current source review confirmed commit-before-notify,
failure propagation, one profile aggregate-save boundary, repository-owned query
history, and retained owner regressions. RF-004 still had one independently
reproducible loss case.

The recording script's `flushInput` checked the buffered string's truthiness.
When a user cleared a field, a valid empty full-value snapshot was dropped.
Added a real-Chromium integration case that types a temporary value, clears it,
and stops recording before the debounce timer fires. Before the repair, the
focused run failed because it received zero type events. A second new case
proved nonempty buffered text already flushes before stop acknowledgement; that
existing path required no change. The repair checks for an input target rather
than a truthy value, so the final `""` snapshot is committed while preserving
the same stop/drain sequence.

The complete focused browser-injection owner passes15/15 tests. Targeted Go
workflow-generation tests pass under
`GOTOOLCHAIN=local GOPROXY=off go test ./services/live-capture -run
'TestMergeSnapshotsPreservesHistoryAndTarget|MergeConsecutiveActions|GenerateWorkflow'
-count=1`, including final-snapshot replacement and target boundaries for page,
driver page, frame, URL, selector and submit. The only
production source change is one removed truthiness conjunct in
`playwright-driver/src/recording/capture/browser-scripts/recording-script.js`;
there is no new helper, dependency, runtime path or policy. The integration test
adds two real-browser cases. This is a narrow correctness fix, not a claim of
material aggregate complexity reduction or full recording readiness.

Current focused race verification also confirms the earlier durable-journal and
profile fixes: `go test -race ./services/recording ./services/recording/persistence
-run 'Journal|AppendTimelineEntry' -count=1` and
`go test -race ./handlers -run
'^(TestReceiveRecordingActionRequiresCommit|TestPullRecordingActionsCommitBeforeAcknowledgement|TestRecordingProfileCommit)$'
-count=1` pass. Updated RF-002, RF-003 and RF-024 from stale open descriptions
to their repaired behavior and exact retained regressions. RF-004 now records
empty-value repair while retaining the full record-to-replay, native IME and
cross-target matrix as open. No broad Test Genie suite or managed restart ran.

Evidence: `internal/evidence/rehabilitation/recording-empty-input-2026-09-24.json`.
The current managed build remains `sha256:d8694c89b82a07446f7654007c6bc2335916fef26217cc560b9f392f084dd843`;
it was not restarted, so this source fix is tested but not yet present in the
running driver. Contract preparation passes with17 outcomes,24 preservation
journeys and zero errors. Required setpoint read
`prog_98e940bb-cc04-4044-85fe-5e96e22b8635` remains3/17 in-band,14 unavailable,
`product_qualified=false`; the local change has not been deployed and cannot
advance that score. Next: finish the RF-004 record-to-replay coverage boundary,
then address a still-open implementation issue rather than repeat completed
evidence producers.

### BAS-WORK-168 — 2026-09-24 UTC — deploy the empty-input fix and refresh only build-bound evidence

Trigger: continue the Browser Automation Studio improvement goal and respond to
the operator's request for more targeted testing and visible score movement.
Search-hub discovery was empty; the prescribed prompt-manager fallback and BAS
improvement skill were used. The managed restart completed healthy on build
`sha256:ba32c4f547ecbab0d58f1fa1ed71b77846c021658afd497950fb3a7b11db8872`.
The staged browser recording script SHA matches source SHA
`440e6c84eab36cb7b6ec66caa277ee92b070b3764e5c982c5645772fd0322dd3`; the API
health endpoint reports ready and all dependencies connected. The synthetic
session profile identity and metadata remained unchanged across the deployment.

Focused recording-injection tests passed15/15, including clearing a text input
and stopping immediately; the pending debounced nonempty edit also flushes before
stop acknowledgement. Targeted Go race owners passed for workflow cancellation,
timeout, driver death, interrupted recovery, cancellation receipt validation and
profile receipt validation. The managed Performance Health capture owner passed
100 measured samples plus one warmup at443ms service p95 and672.588ms wall p95
against a2000ms budget. The profile cohort passed five seed checks and two
post-managed-restart checks, observed its checkpoint after1654.511ms, preserved
alpha/beta isolation and deleted both synthetic profiles. The J07 cancellation
owner receipt passed all five cases: one effect, terminal uncertain failure,
resource count1→0 and retry denied; managed API recovery measured6.928s.

One exact `rehabilitation-evidence` provider phase passed at L1/Verified for both
profile durability and cancellation/recovery on the current candidate. The
required setpoint read `prog_93e24934-d867-478f-827e-edbdd388e8f5` reports3/17
in-band,14 unavailable, `product_qualified=false`. Capture, profile durability
and cancellation/recovery are the three readable rows. The evidence tier is
degraded because the shared checkout is dirty; source-level and owner assertions
passed. No broad Test Genie suite ran. RF-004's empty-input fix is now deployed,
but full record-to-replay, IME and cross-target behavior remains open. No net
complexity reduction is claimed. Evidence receipts are linked under
`docs/internal/evidence/rehabilitation/`, with profile owner payloads retained
under ignored `.vrooli/runtime/rehabilitation-evidence/`.

Next: use the focused record-to-replay owner to cover replace/delete/clear,
pauses, IME and same selectors across targets, then choose another open source
issue. Keep unavailable setpoint rows explicit; do not repeat the already-valid
capture/profile/cancellation producers without a binding change.

### BAS-WORK-171 — 2026-09-24 UTC — locate dropped tab identity before generation

Trigger: operator feedback BAS-FB-018 asks for less Test Genie execution and
more frequent targeted tests; W170's next J03 discriminator is alternating
same-selector actions across tabs. Hypothesis: if the canonical timeline omits
the driver's page identity, the workflow generator cannot distinguish actions
from two tabs, and its multiple-page guard cannot protect this path. The
discriminator was source tracing from page-level route ingress through timeline
conversion into `RecordedActionFromTimelineEntry`, alongside focused owners.

The hypothesis is confirmed. `RawBrowserEvent` carries `frameId`/`framePath`
only; `rawBrowserEventToTimelineEntry` transfers those into ActionTelemetry,
whose schema has no page identity. `RecordedActionFromTimelineEntry` therefore
cannot populate `PageID` or `DriverPageID`. Since the generator sees empty
identities, separate tabs can collapse to one apparent target before target
validation. The live-capture multiple-page refusal is not sufficient for this
driver timeline path. Updated BAS-RF-030 and the architecture statement to name
this exact ownership boundary. No product source or managed build changed, and
no complexity reduction is claimed.

Focused evidence: `GOTOOLCHAIN=local GOPROXY=off go test
./automation/driver -run '^(TestRecordedActionFromTimelineEntry_PreservesFrameIdentity|TestRecordedActionFromTimelineEntry_PreservesNavigateWaitUntil)$'
-count=1` and `GOTOOLCHAIN=local GOPROXY=off go test ./services/live-capture
-run '^(TestGenerateWorkflowSwitchesBetweenMainAndChildFrameForSameSelector|TestGenerateWorkflowRejectsUnrepresentableRecording|TestMergeSnapshotsPreservesHistoryAndTarget)$'
-count=1` pass. `pnpm exec jest tests/integration/pipeline-e2e.test.ts
--runInBand --coverage=false --testNamePattern='captures a dynamically attached
frame after recording has started'` passes1/1, and `pnpm exec tsc --noEmit`
passes. An initial Jest filter against the wrong file selected zero tests and
failed the repository-wide coverage threshold; it was corrected to the actual
pipeline owner. No Test Genie execution ran. The exact rehabilitation setpoint
read `prog_39f03506-2476-424c-93f6-d2d75b9032a8` reports3/17 in-band and14
unavailable; capture is432ms p95/646.730501ms wall p95 on the current managed
build; profile durability and cancellation/recovery remain L1/Verified with a
degraded evidence tier due to the shared dirty checkout.

Next: add a focused timeline-conversion regression that proves a tab identity
survives both page-route capture and Go conversion, then carry opener/lifecycle
events into generation before relaxing the refusal. Keep equal-selector
alternation and fresh-context replay as the acceptance test; do not count frame
generation coverage as tab replay proof.

### BAS-WORK-172 — 2026-09-24 UTC — carry stable tab identity into recorded actions

Trigger: W171 confirmed same-selector actions from separate tabs lose target
identity before workflow generation, leaving target-aware merge and validation
unable to distinguish them. The implementation hypothesis was that the
session's existing `pageToIdMap` can supply each routed event with the same
stable driver page ID already used by page lifecycle callbacks.

Added optional `ActionTelemetry.driver_page_id` to the canonical proto and
refreshed only the browser-automation-studio proto package (artifact
`4bc3b7918772be2f6674a3f939ba663b250edf48d95f4a6fdb4bb46f28088d47`). The
recording pipeline now resolves IDs through the session's existing `pageToIdMap`
when registering event routes; the route places that identity on the raw event,
proto conversion stores it, and Go timeline conversion fills
`RecordedAction.DriverPageID`. This makes merge boundaries and the existing
multi-page refusal see distinct targets without adding a second page registry.
Lifecycle/opener reconstruction and replay remain required before generation
can accept multiple tabs. RF-030 and architecture now state this boundary. The
managed BAS-only setup completed the generated package refresh; the live service
still runs the prior build. No whole-domain debt reduction is claimed.

Targeted evidence: Go
`GOTOOLCHAIN=local GOPROXY=off go test ./automation/driver -run
'^(TestRecordedActionFromTimelineEntry_PreservesDriverPageIdentity|TestRecordedActionFromTimelineEntry_PreservesFrameIdentity)$'
-count=1` passes, and selected live-capture target/frame owners pass. Playwright
`tests/unit/proto/recording-redaction.test.ts` passes5/5. The real-Chromium
`tests/integration/pipeline-e2e.test.ts` equal-selector/two-tab owner passes1/1
and confirms distinct IDs in observed order. `pnpm exec tsc --noEmit` passes.
Scoped ESLint on changed source/unit owners exits successfully with existing
warnings; linting the large integration file reports pre-existing rule errors,
and whole-file Prettier checks report formatting differences in already-dirty
files. `git diff --check` passes for changed source. No Test Genie phase ran.

The current live setpoint remains3/17 in-band and14 unavailable, so this source
change has not advanced a governed outcome. Next: add the ordered page lifecycle
and opener observations to workflow derivation, then prove alternating equal
selectors across tabs/frames replay in a fresh browser context before relaxing
the multiple-page refusal. Keep the existing frame owner separate in reports.

### BAS-WORK-173 — 2026-09-24 UTC — keep workflow tab stack current outside recording

Trigger: continue RF-030 after W172 carried stable page identity into recorded
actions. Hypothesis: pages created by a workflow click or popup are absent from
the execution tab stack when recording is off, because only recording-mode page
listeners update `session.pages`. The falsifier was a real Chromium context
popup appearing in the same stack consumed by workflow tab-switch instructions.

The hypothesis reproduced. A focused manager regression created a popup while
recording was off; it failed because `session.pages` contained only the initial
page. SessionManager now observes context page creation and page closure for
standard and external-target sessions, maintaining the existing page stack and
current-page index. TabHandler insertion/removal is idempotent with those events,
so a handler-opened page is not duplicated and its close callback cannot remove
the following tab. This closes an execution-time stack ownership gap; it does
not reconstruct recorded opener relationships or qualify replay. RF-030 remains
open, and no whole-domain complexity reduction is claimed.

Focused evidence: real-Chromium
`pnpm exec jest tests/integration/session-admission-capacity.test.ts --runInBand
--testNamePattern='tracks popup pages in the workflow tab stack when recording
is off'` failed before the source change and passes after. Focused
`tests/unit/idempotency/tab-idempotency.test.ts --runInBand
--testNamePattern='open|close' --coverage=false` passes4/4. `pnpm exec tsc
--noEmit`, scoped ESLint, `git diff --check`, and `python3
docs/internal/refactor_contract.py` pass; ESLint reports six pre-existing
warnings and no errors. An initial setpoint command used the wrong CLI prefix;
the discovered `program-runtime library run` command succeeded. Its read reports
3/17 in-band and14 unavailable because this source is not deployed; product
qualification remains false. No Test Genie run occurred. Next, capture ordered
created/navigated/closed/opener events into portable logical tab bindings and
prove alternating equal-selector replay in a fresh context before allowing
multi-page generation.

The required include-untracked inventory observed source digest
`c2e67ce348993e0629a2999e7615dac5af21cc23c559e42605b71cc0e6cf2524` on HEAD
`8707b970a48f82665662450766ad8233d6def9e7`. It measures 52,767 Playwright-driver
runtime lines, with24 runtime files above500 lines and4 above1,000; `manager.ts`
is now1268 lines. The shared checkout contains many unrelated changes, so this
snapshot cannot attribute a comparable domain-wide debt delta to W173. The fix
adds a lifecycle observer and guards; this slice has no proven net complexity
reduction.

### BAS-WORK-174 — 2026-09-24 UTC — make captured popup target switches replayable

Trigger: continue RF-030 after W173 repaired workflow-time popup page tracking.
The focused generator regression failed as predicted: workflow derivation took
only the action slice and rejected multiple targets without using the existing
PageTracker identity/opener data. The implemented path passes the page snapshot
from the owned capture session into generation, resolves each action to its
stable page identity and emits typed tab-switch instructions. A popup switch is
treated as caused by the preceding click/key action only when its opener matches
that action's target and its creation timestamp lies between the two action
times; unrelated timing fails closed. A page without an opener is opened at its
first recorded use. Missing action identity and closed page targets fail closed.
API page-created, navigation, and close events preserve driver timestamps so
this causal check does not substitute receipt time for browser event time.

Focused Go tests cover opener→popup→opener alternation with the same selector,
typed compiler conversion, service-to-PageTracker handoff, independent-page
first use, ambiguous popup timing, and missing identity. The Go owner commands
pass with
`GOTOOLCHAIN=local GOPROXY=off go test ./services/live-capture -run
'^(TestService_GenerateWorkflowUsesTrackedPageBindings|TestGenerateWorkflowWithPages(ReplaysPopupTabAlternation|OpensIndependentTabAtFirstUse|RejectsAmbiguousPopupTiming|RejectsMissingActionIdentity)|TestGenerateWorkflowSwitchesBetweenMainAndChildFrameForSameSelector|TestGenerateWorkflowRejectsUnrepresentableRecording)$' -count=1`; the focused page-event timestamp and identity owners also pass in `./handlers`.
The focused real-Chromium test
`replays equal selectors on the opener and popup pages through typed tab
switches` passes1/1; it switches through `TabHandler` and asserts each identical
selector changes only its intended page. Prettier and scoped ESLint pass for the
owner. Contract preparation passes17 outcomes,24 preservation journeys, zero
errors. No Test Genie phase or broad suite ran. The source is not deployed.

W174 establishes a bounded generator path and typed runtime tab-switch behavior;
it does not yet prove that a recorded session becomes a saved workflow and
replays in a fresh context, nor nested frame/tab alternation or close/reopen
replay. No net complexity reduction is claimed: this slice adds lifecycle
bindings and one focused owner without removing a competing path. Required
setpoint read `prog_403f98f2-39dd-4ad7-b480-7d30b3dee760` remains3/17 in-band,
14 unavailable and `product_qualified=false`; no qualification row advanced.
The include-untracked shared checkout inventory digest is
`0c6459bf25b846d6c61b123d8071f188abf7d1b9641bad19b0395187f34b0a3c`; it
observed194 modified/untracked paths and still shows `simple_executor.go` at1775
lines, so it cannot isolate a net debt delta.
Next: extend one real captured recording through workflow persistence and fresh-
context replay, while keeping each test owner targeted.

### BAS-WORK-169 — 2026-09-24 UTC — replay an empty final input snapshot

Trigger: continue the RF-004 record-to-replay boundary with a targeted owner
test after the W168 deployment. Current source already flushed empty values on
stop and W168 proved capture; the remaining question was whether a clear action
replays as empty against a prefilled field. Baseline: the existing real-Chromium
replacement replay case passed before the new case. The falsifier was an observed
final replay value other than the empty string, or an acknowledged input record
without `value=""` and `clearFirst=true`.

Extended
`playwright-driver/tests/integration/timeline-validation.test.ts` with a real
record → Ctrl+A/Backspace → immediate stop → replay journey. It asserts the
empty input timeline action, replacement semantics and final empty value in a
fresh page prefilled with `original`. Both replace and clear tests now use one
shared replay helper, removing duplicate handler/context setup. The new clear
case passed on its first run; the focused final run passes both cases (2/2).
`pnpm exec prettier --check` and `pnpm exec eslint` pass for the owner file;
`pnpm exec tsc --noEmit` passes. No source runtime behavior or managed build
changed. The test file grew for the new journey; no whole-domain complexity,
duplication or coupling reduction is claimed. The focused checks leave pauses,
native IME and same-selector cross-target recording/replay unverified.

RF-004 remains open for that broader preservation corpus. The active managed
candidate remains `sha256:ba32c4f547ecbab0d58f1fa1ed71b77846c021658afd497950fb3a7b11db8872`;
the required setpoint read `prog_88dea175-8b30-4e00-aacb-ce914ca96ccb` confirms
3/17 in-band,14 unavailable, `product_qualified=false`. Contract preparation
passes with17 outcomes,24 journeys and zero errors. Include-untracked inventory
digest is `152e53031f6c53860d0957d678199462abcd31a936a038298390d826427098ad`;
it sees172 modified/untracked paths and its ten largest runtime modules range
from1,775 lines (`simple_executor.go`) to1,241 lines (`session/manager.ts`). This
contaminated shared-tree reading cannot isolate a W169 domain-wide delta. Next:
probe same-selector events across frame/target switches in a real recording and
replay journey, then investigate another actual open high-priority behavior
defect; do not repeat build-bound evidence while its bindings remain unchanged.

### BAS-WORK-170 — 2026-09-24 UTC — preserve cross-origin iframe identity for replay

Trigger: follow W169's RF-004 target-isolation lead with a falsifiable same-selector
main-document/iframe replay case, while keeping validation focused. The Go
workflow generator initially rejected the recorded child-frame action because
telemetry supplied no logical frame binding; the new focused Go regression failed
with `frame replay requires a logical frame binding`. Root cause was that
`window.frameElement` cannot identify a cross-origin frame from inside its
document. The Playwright request route now derives a selector path from trusted
`Request.frame()` / `Frame.frameElement()` ownership, carries it through the
additive `ActionTelemetry.frame_path` contract and timeline conversion, and the
workflow generator emits frame enter/parent transitions before actions against
the changed target. Validation also bounds path depth and rejects empty path
segments rather than generating ambiguous replay instructions.

Targeted evidence: generated proto package artifact
`8ae142f0aea0cf8e20376c8549c35b2536e783ddde9a7172c181025f276a24ff` was
published and BAS setup refreshed the consumer copy. The real Chromium test
`captures a dynamically attached frame after recording has started` passes and
asserts `framePath == ["#late-frame"]` for a cross-origin iframe. The existing
real-browser `frame-switch makes later public instructions affect only the
selected document` case passes. Focused Go replay/timeline tests pass in both
`services/live-capture` and `automation/driver`; driver `tsc --noEmit` and ESLint
for `event-route.ts` pass. The J07 owner found a measurement defect: it waited
for the managed restart command's health gate to finish before polling for the
terminal execution receipt. Two observations therefore reported 12.4s and13.5s
recovery, just above the10s band. The owner now observes terminal status
concurrently with restart, measuring first availability instead of serializing
the health-gate tail. `node --check` and Prettier pass; the maintained focused
owner then measured 4.712s recovery and50ms cleanup, with one effect, resource
count1→0, uncertain outcome retained and retry denied.

This adds a logical selector path for frames, not full J03 support: the contract
journey still requires alternating same-selector actions across tabs and frames
and replay in a fresh context. Tab lifecycle/open/close identity and end-to-end
saved workflow replay remain unverified. No aggregate complexity reduction is
claimed; the change adds one additive telemetry field and route-to-generator
transport while removing none of the legacy paths.

The managed lifecycle restored BAS healthy on candidate
`sha256:5e3e5845578efcfe5c24fd8071697bef9503f519a276a21d2f7503ac83dbcefd`.
Build-bound evidence was refreshed with the focused owners: capture operation
`d1bd6089e324a8ff6a2310f043f8bd82` passed100 samples plus one warmup at432ms
service p95 and646.731ms wall p95 against2000ms; the profile cohort passed five
seed checks, checkpoint at389.730ms, two post-restart checks, alpha/beta
isolation and cleanup of both profiles; all five J07 cases passed. The single
narrow `rehabilitation-evidence` provider phase
`20260924-161818-3707bbd7` passed L1/Verified for profile durability and
cancellation/recovery. Required setpoint read
`prog_23ce499e-05b8-44b8-8f3d-1453eee73725` reports3/17 in-band,14 unavailable,
`product_qualified=false`. No broad Test Genie suite ran. The evidence tier is
degraded because the shared checkout is dirty.

The first package-refresh command was mistakenly issued as `proto all`; it was
interrupted after setup had begun for unrelated consumers including agent-inbox,
agent-metareasoning-manager, ai-chatbot-manager, algorithm-library, api-library,
app-monitor and asset-studio. Those workspace side effects were preserved. The
remaining refresh was narrowed to BAS; a managed stop/refresh/start updated the
generated consumer, and the final BAS status and API health both reported
healthy on the candidate above. Current working tree is broadly dirty, so no
domain-wide inventory delta can be attributed to W170. The exact frame-recording
and cross-target workflow remains an active RF-004 follow-up. The final
include-untracked inventory observed 187 changed/untracked paths with source
digest `3fbb43998ac43b9e5503263aec5f98b08d1cfba185385c8b2534ef677ce4aaf5`;
its largest runtime file remains `simple_executor.go` at1775 lines. This broad
shared-tree snapshot cannot attribute a net complexity delta to W170.

### BAS-WORK-175 — 2026-09-24 UTC — replay an observed popup close

Trigger: continue RF-030 while responding to BAS-FB-020's request for focused
owner tests and more visible progress. The current generator rejected a popup
whose final PageTracker snapshot was closed even when the captured popup action
occurred before its close. Hypothesis: `CreatedAt`, action timestamps and
`ClosedAt` are sufficient to admit that earlier action and emit the close
between the neighboring actions; the falsifiers are replaying an action after
close, guessing at equal/missing timing, or closing the last replay tab.

Changed `api/services/live-capture/workflow_generator.go` and its owner test.
Generation now checks that a closed page existed at the recorded action time,
emits typed indexed close operations for closes strictly between actions, and
updates its open/active stack with the same fallback index behavior as the
driver's `TabHandler`. Ambiguous close timing and a close that would remove the
last replay tab return errors. The first targeted test was red because the
existing closed-state guard rejected the popup's earlier action; after the
timestamp-aware admission fix, both the new close journey and existing popup
alternation regression passed. Final focused command `go test
./services/live-capture -count=1` passed. No Test Genie phase, runtime restart,
deployment, or broad suite ran.

Required setpoint read `prog_c90424bd-dec7-4c9e-9eeb-47dafb5e20b8` completed:
3/17 in-band,14 unavailable, `product_qualified=false`. The three readable rows
remain capture, profile durability and cancellation/recovery. Interactive
feedback remains pending because its contract requires 1,000 local and remote
correlated samples; the retained receipt is local loopback only. An earlier
read-only remote BAS status request was denied for missing `vrooli:read`; no
access change or bypass was attempted. The implementation change is not
deployed and does not affect score. RF-030 still lacks fresh-context replay of
the generated workflow and nested tab/frame coverage. The source/test addition
increases local code size; no net complexity reduction is claimed. Next: extend
the existing real-Chromium owner to capture the popup lifecycle, submit the
generated workflow and replay it in a separately created context, asserting
same-selector effects on the opener and popup after the recorded close.

### BAS-WORK-176 — 2026-09-24 UTC — make tab-close ordering deterministic

Trigger: continue RF-030's lifecycle owner after W175, while keeping the turn on
targeted regressions and no Test Genie execution. Review found that two tracked
open pages may share one close timestamp; sorting alone cannot establish which
tab closes first or which runtime index is valid. The falsifier is generation
choosing an order, replaying a page after close, silently skipping missing close
timing, or emitting a close that removes the only replay tab.

Updated `api/services/live-capture/workflow_generator.go` and its owner tests.
The generator now rejects same-time closes for multiple open pages, treats a
missing timestamp as ambiguous when a tracked open page could have closed, and
uses removal from the open-tab slice as the sole record that a page has closed;
the separate closed-ID map was redundant. The maintained tests cover a popup
action before close followed by the indexed close, an action after close,
equal/missing timestamps, simultaneous closes and the last-tab boundary. The
focused three-test selection passed, then the full owning package passed:
`go test ./services/live-capture -count=1`. An initial compile failed due to a
local variable shadowing `pageIndex`; renaming the local to `tabIndex` fixed it.
One first version of the simultaneous-close test also asked for an action on a
page after that page was already closed; the owner correctly rejected that
action earlier. The fixture was corrected to target a still-open third page,
which exercised and verified the intended equal-close-time refusal.

Contract preparation passes17 outcomes and24 journeys with zero errors.
Required setpoint read `prog_af25be27-6a1b-4570-a2b7-1c566d40342a` completed
3/17 in-band,14 unavailable, `product_qualified=false`; capture, profile
durability and cancellation/recovery remain the three passing rows.
Interactive-feedback remains unavailable: the retained 1,000-sample cohort is
loopback-only, and the contract still needs the declared remote condition. No
Test Genie phase, managed restart or deployment ran. The source is not deployed;
the tests do not score RF-030 or prove saved-workflow replay in a fresh context.
Removing one redundant map does not demonstrate net complexity reduction.
Next: extend a real product path to save the generated multi-page workflow and
execute it in a separate browser context, retaining same-selector effects and
the recorded popup close as independent fixture assertions.

### BAS-WORK-177 — 2026-09-24 UTC — save and replay an alternating-tab workflow

Trigger: respond to FB-021 and RF-030 after W176 left the score at3/17; replace
another generator-only check with one managed saved-workflow journey. Hypothesis:
the page-aware generator, catalog persistence and execution path can carry equal
selectors across two tabs if a saved definition is reloaded after capture and
the recording browser is closed. The independent oracle is a local HTTP fixture
whose same-selector buttons report their logical tab; the falsifier is any
missing, duplicated or reordered `main`, `popup`, `main` report.

Added `playwright-driver/tests/integration/saved-workflow-fresh-context.test.ts`
as a managed-runtime opt-in. It creates an empty project in a temporary folder,
starts a recording session with identifiable page receipts, submits the
recorded-action payload through `generate-workflow`, retrieves the stored
workflow, asserts navigate/click/open/click/switch/click order, closes the
recording session, and executes the saved workflow. The fixture received exactly
`main`, `popup`, `main`. Cleanup deletes the project and its files; final owner
inventory read showed0 BAS sessions,0 disposable projects and0 disposable
workflows; `/health` returned healthy on build
`sha256:43bbd665129c3f02008d576b6124b43442b5b2a0754cb2e0eeeb8a1a27a3c6b4`.
The saved-replay test passed. The earlier capture-only equal-selector owner and
`go test ./services/live-capture -count=1` also passed.

During the first live probe, the old managed build refused multi-page generation;
`make restart` rebuilt the source candidate through the managed lifecycle. The
first replay attempt also showed why an explicit recorded navigation is needed
before selector actions; the retained test now asserts that entry step. A cleanup
route typo left five test sessions active temporarily; it was corrected, those
exact sessions were closed, and health recovered. Early disposable workflows
were removed by exact test-name/ID; future runs use and delete a temporary
project. No operator workflows or projects were changed.

The same probe demonstrated a separate live defect: `DeleteWorkflow` returned
success but left a workflow file because `WorkflowIndex.FilePath` is rooted at
the project while deletion joined it below the workflows directory again. Added
`ProjectWorkflowFilePath`, a focused path-contract regression, and removal of
version snapshots before catalog deletion. The owning Go selection passes.
After the second managed restart, the integration owner also verifies Delete
then GetWorkflow rejection; the repaired path is deployed and verified as
BAS-RF-127.

Validation: `BAS_REHAB_LIVE_API_BASE=http://127.0.0.1:17116 pnpm exec jest
tests/integration/saved-workflow-fresh-context.test.ts --runInBand
--coverage=false` passed1/1 against the managed build; `pnpm exec prettier
--check` and focused ESLint pass. `go test ./services/workflow -run
'TestProjectWorkflowFilePathResolvesProjectRootRelativeIndex|TestRemoveWorkflowFilesUsesProjectRootIndexAndRemovesVersionSnapshots|TestWriteWorkflowSummaryFile'
-count=1` passes. `refactor_contract.py` reports17 outcomes,24 journeys and
zero errors. No broad suite or Test Genie phase ran.

This verifies the saved execution journey but does not join actual pointer
capture through the durable action journal, exercise frames, or close/reopen
popup lifecycle. It does not meet passive-fidelity's 10,000-action and
crash/reconnect band, so no score credit is claimed from this test. The new
integration test and path/version deletion logic add code; no net complexity
reduction is claimed.

Build-bound qualification refresh: profile seed passed5/5 with a 273.8ms
checkpoint; after the managed API/driver restart, alpha and beta both restored
and both synthetic profiles were deleted. The focused cancellation owner passed
its Go race tests and real-Chromium retry-denial test. Its five independent
observations report one effect each, resources1→0, retry denied, cancellation
input stop0.131ms, timeout cleanup0.382ms, driver-death cleanup0.498ms and API
recovery9.502s. Receipts:
`docs/internal/evidence/rehabilitation/profile-durability-177-2026-09-24.json`,
its `profile-durability-177-{seed,verify}-owner.json` receipts, and
`docs/internal/evidence/rehabilitation/cancellation-recovery-2026-09-24T19-00-00-w177.json`.
The first cancellation run measured successful recovery but used a receipt path
outside the assembler's allowed evidence directory; the owner was rerun with an
accepted retained path. Focused Go validation
`go test -race ./internal/cancellationqualification ./handlers/profilevalidation -count=1`
passes. The exact `rehabilitation-evidence` Test Genie phase
`20260924-185233-3fa283d9` passed L1/Verified for profile durability and
cancellation/recovery. Required setpoint read
`prog_af077618-222b-439c-a6fa-89f47c22aa04` still reports3/17 in-band,14
unavailable, `product_qualified=false`; the refresh preserves the same three
qualified rows and does not increase the score. The remaining14 rows all report
`pending_telemetry`. This phase ran only because its persisted provider
assessment is the sensor for the two refreshed rows.

### 2026-09-24 — BAS-WORK-178 / RF-030 raw recording identity

The live recording owner sends three real pointer clicks through BAS input
ingress and waits for their durable timeline actions. It confirms the driver's
actions omit logical `pageId`, while the matching timeline entries supply it,
then joins by stable action ID before generating and saving the workflow. The
independent fixture confirms exact main → popup → main effects after the capture
browser is closed and the saved workflow runs in a fresh context. UI generation
now uses the same pure join helper before action merging.

Validation: UI timeline owner35/35; UI `tsc --noEmit`; scoped ESLint; managed
real-input save/replay Jest owner1/1; API health on build
`sha256:43bbd665129c3f02008d576b6124b43442b5b2a0754cb2e0eeeb8a1a27a3c6b4`;
evidence receipt `evidence/rehabilitation/recording-workflow-page-identity-2026-09-24.json`.
The first live assertion correctly showed that `pageId` belongs to the timeline
entry, not its nested action; the owner was adjusted to assert the actual schema
and then passed. No Test Genie phase ran. Setpoint program
`prog_d2c64cfd-341d-40f3-8b61-760f762c7f7a` reports3/17 in-band,
14 unavailable (`pending_telemetry`), `product_qualified=false`. This closes
the raw input-to-journal linkage portion of RF-030 but does not claim a complete
preservation outcome or qualification score increase. No net complexity
reduction is claimed.

### BAS-WORK-179 — 2026-09-24 UTC — targeted passive-fidelity boundary

Added two focused real-Chromium cases to the maintained pipeline owner. The first
dispatches10,000 fixture clicks in acknowledged batches, rejects one journal
delivery at the midpoint, and verifies10,000 unique ordered event IDs with the
rejected identity retried exactly once. The second rejects a browser event with
503, reloads the page, confirms the pending identity survives in session storage,
and verifies the same identity is later acknowledged. Results:2/2 passed; the
10k case took11.637s and reload recovery1.286s. The focused Go owner also passes
2/2 tests for failed-commit non-publication and 1,001-entry disk-backed journal
reopening across fresh service instances. Receipt:
`evidence/rehabilitation/passive-fidelity-10k-2026-09-24.json`.

An unpaced 1,000-click probe reached101 delivered events and failed closed with
“delivery capacity was exceeded,” consistent with the recorder's100-event
pending bound. The passing 10k case therefore gates each batch on acknowledgments;
it does not imply an unbounded producer. `refactor_contract.py` reports17
outcomes,24 journeys and zero errors. Scoped ESLint still reports seven existing
errors elsewhere in the large owner file; no new-case error remains. No Test Genie
phase ran, and no net complexity reduction is claimed.

This is partial passive-fidelity evidence only. It does not inject driver-process
death or an actual API-process restart. The Go reopen owner uses a new service
instance over the same SQLite file but is not a process-death test. This work
also does not cover the complete supported event semantics or provide the
required qualification producer and governed sensor.
The required setpoint remains3/17 in-band,14 pending telemetry, and
`product_qualified=false`; no qualification credit is claimed. Next work is to
exercise driver death and API journal restart, then connect the retained receipt
to the governed producer/sensor.

### BAS-WORK-180 — 2026-09-24 UTC — durable 10,000-action journal rejection/reopen

Expanded the existing SQLite journal owner from1,001 to10,000 independently
identified observations. The test injects one transient SQLite trigger failure
at the midpoint, confirms the write fails without acknowledgement, retries the
same event ID, closes the disk handle, and reads every page through new service
instances against the same database. Each page matches the independent expected
ID and sequence log; 40 concurrent post-reopen writes retain contiguous order.
The focused owner passes1/1 in3.107s. Receipt:
`evidence/rehabilitation/passive-fidelity-journal-10k-2026-09-24.json`.

Validation used `go test ./services/recording -run
'^TestJournalHistorySurvivesPaginationReopenAndConcurrentWriters$' -count=1`,
`refactor_contract.py` (17 outcomes,24 journeys,zero errors), JSON parsing and
scoped `git diff --check`. This expanded a test only; no production source,
managed build, Test Genie phase or complexity reduction changed. It strengthens
the API's disk-journal layer, but does not kill/restart the API process, connect
real Chromium events to this SQLite owner, or verify the complete event-semantics
corpus. The setpoint remains3/17,14 `pending_telemetry`, `product_qualified=false`.
The descriptor/provider and program sensor currently cover only profile
durability and cancellation/recovery. Next: add the recording outcome to the
same current-build owner-assessment chain only after its API/browser failure
cohort is integrated and process-level crash boundaries are covered.

### BAS-WORK-181 — 2026-09-24 UTC — service-process crash before acknowledgement

Extended the API recording journal owner to cover a child service-process kill
at the commit/ack boundary. The owner starts with10,000 independently identified
SQLite observations, then the child service commits action10,001 and signals
before the simulated client acknowledgement. The parent kills that process,
reopens the same database through a fresh recording service, retries the same
identity and confirms the journal remains at10,001 entries with no duplicate.
Every page of the original10,000 observations is compared with the independent
expected ID sequence. The focused test passes1/1 in2.126s. Receipt:
`evidence/rehabilitation/passive-fidelity-process-crash-10k-2026-09-24.json`.

Validation command: `go test ./services/recording -run
'^TestJournalSameIDRetryRecoversAcrossServiceProcessDeath$' -count=1 -v`.
This injects real OS-process death around the recording service but not the
managed BAS HTTP API or driver/browser process. It also does not yet join the
10,000 Chromium observations to the durable journal or validate all event kinds.
No Test Genie phase ran; no score or net complexity reduction is claimed. The
current worktree remains broadly dirty from prior BAS work, so no whole-tree
complexity delta is attributed to this test-only change.

### BAS-WORK-182 — 2026-09-24 UTC — native 10,000-action Chromium oracle

The previous 10k pipeline owner used JavaScript-dispatched `MouseEvent`s and
counted recorder callbacks, which was too weak to call native input or an
independent fixture oracle. Replaced it with real Playwright mouse down/up at the
fixture button and a fixture-owned capturing listener. The oracle reports10,000
click effects; the pipeline receives10,000 unique IDs in increasing sequence,
with one midpoint delivery rejected and retried under the same ID. The 10k case
passes in11.906s. The focused page-reload recovery owner also passes; together the
selected Jest cases pass2/2 in14.535s. Receipt:
`evidence/rehabilitation/passive-fidelity-native-10k-2026-09-24.json`.

Validation used the named Jest filter on `pipeline-e2e.test.ts`; TypeScript
transformation succeeded in Jest. No product source changed, no Test Genie phase
ran, and no complexity reduction is claimed. This fixes the validity of the
10k browser observation, but the pipeline still ends at an in-process callback;
real events are not yet joined to SQLite/API durability, process death still
covers only the API recording-service fixture, and the governed row sensor is
not implemented. The required score remains3/17 until those limits are closed.

### BAS-WORK-183 — 2026-09-24 UTC — managed Chromium-to-journal 10,000-action cohort

Added a targeted opt-in managed integration owner that sends10,000 native
Playwright pointer clicks through the live BAS `/input` API route. It paces
clicks in batches against applied-input acknowledgements, waits for each batch
to appear in timeline pages, and checks an independent fixture effect counter.
The final receipt proves10,000 effects, unique journal IDs, increasing journal
sequences and increasing applied-input receipts on the same managed build.
The test passes1/1 in45.221s. Receipt:
`evidence/rehabilitation/passive-fidelity-managed-10k-2026-09-24.json`.

Validation used the named Jest filter only; the required setpoint read
`prog_4bb60c81-3206-4e3e-a118-96b5de836eff` still reports3/17 in-band,
14 `pending_telemetry`, `product_qualified=false`. This closes the direct
Chromium-to-durable-journal linkage gap, but the 10k cohort did not inject a
fault. Separate prior owners cover rejection, reload and recording-service
process death; managed driver death and full event semantics remain untested.
No Test Genie phase ran, no score credit or net complexity reduction is
claimed, and this synthetic closed session leaves a timeline because BAS has no
public deletion route. Next: wire current-build receipt validation and the
governed sensor, then fault the managed driver boundary.


### BAS-WORK-187 — 2026-09-24 UTC — faster session admission recovery and current passive-fidelity owner

After the operator again reported a stalled3/17 score, the session-admission
retry ceiling was reduced from2s to500ms; its30s total bound, caller cancellation
and explicit-capacity-only retry behavior are unchanged. The focused Go owner
group passes4/4 (`RetryDelayStaysResponsive`, `WaitsForCapacity`,
`NeverRetriesAmbiguousCreation`, `CancellationAndBudget`). This is source-only
and the managed f692 service was not restarted, so live improvement remains
unverified. No complexity reduction is claimed.

The focused managed 10k passive-fidelity owner was rerun on the current f692
build. It passed1/1 in49.911s:10,000 native clicks, fixture effects, unique
ordered journal IDs and applied-input receipts; routed storage saw32,444 test
requests and0 primary-pool requests. The provider still cannot credit this row:
the current-build validator/sensor is absent, as are a managed browser-process
loss cohort and the full supported event-semantics corpus. Receipt:
`evidence/rehabilitation/passive-fidelity-managed-w187-2026-09-24.json`.
Setpoint `prog_d98aa9f0-7d82-4ecf-8c52-2c3756d649d7` confirms3/17,14 unavailable,
`product_qualified=false`. Contract validation reports17 outcomes,24 journeys,
0 errors. No Test Genie phase ran. Next: implement the governed passive-fidelity
receipt join and only claim the row after its required faults and semantics have
passing current evidence.


### BAS-WORK-188 — 2026-09-24 UTC — qualify passive fidelity and refresh current score

The operator again reported a stalled3/17 and asked for more targeted tests and
fewer Test Genie executions. Reused the existing passive-fidelity owner evidence
from W187, then filled its two missing evidence boundaries with a focused
recording-service process-death/reconnect owner (10,000-action prefix, commit
before acknowledgement, abrupt child kill, same-ID retry with no duplication)
and three selected Chromium event-semantics cases (click/type/scroll, navigation,
and capture after navigation). The expected-behavior assertion initially used a
wrong fixture selector; corrected the test to the fixture's stable
`data-testid="test-button"` identity and reran the exact three cases, all passing.

Added `api/internal/passivefidelityqualification` to join current managed
10,000-action evidence, crash/reconnect and semantics receipts by contract,
source and artifact digests, including zero writes to the primary DB pool. Wired
the validator into the existing rehabilitation provider and added its capability
to `.vrooli/test-genie.json` and the governed setpoint reader. Focused package
tests pass: `go test ./handlers/profilevalidation
./internal/passivefidelityqualification -count=1`. Current sources were deployed
on build `sha256:a74ff8db5ab1a9d5346a721a2df458d83ab0c985e8ea559fb935faa6651a88eb`.

On that build, the managed passive-fidelity owner passed10,000 native clicks and
effects, ordered unique journal IDs and applied-input receipts in43.612s; test
storage handled32,441 requests with0 primary-pool requests. Profile durability
passed5 seed checks (checkpoint1,807.87ms), two restart checks and cleanup of
both profiles. Cancellation/recovery passed all five owner cases and preserved
the same build identity across managed restart. Capture passed100 samples plus
one warmup at423ms p95 /593.52ms wall p95. One exact `rehabilitation-evidence`
Test Genie phase, `20260924-223631-06f03747`, returned clean L1/Verified for
passive fidelity, profile durability and cancellation/recovery. Governed read
`prog_6c7153b7-e2b7-455b-9f09-325347c12a51` is4/17 in-band,13 unavailable,
`product_qualified=false`. Contract validation remains17 outcomes /24 journeys,
0 errors. Evidence wrappers and raw owner paths are listed in the W188 current
checkpoint above.

The retry-spacing cap is deployed; this cycle's serial capture confirms the
candidate remains in-band but does not isolate its saturated-admission latency
effect. No broad Test Genie suite ran. The provider/receipt wiring is additive;
no net complexity reduction is claimed. Managed driver/browser process loss,
full event semantics, the remaining13 governed outcomes, and the broad shared
checkout's unrelated dirty files remain unverified. A first capacity-owner
attempt returned HTTP429 while creating its tenth hold: four sessions were
already active for Experience Manager and React Component Library workflows,
leaving only six slots. It closed its own leases; inventory returned to those
same four external sessions. No capture ran and no external session was closed.
After a later health read showed0 sessions, the focused owner held10 managed
driver slots, started one `capture-surface` request and released one slot after
2,000ms. Capture completed successfully at3,109ms readiness duration; subtracting
100ms navigation and1,017ms explicit readiness wait estimates1,992ms before the
first action, consistent with the controlled hold. The serial current-build
capture owner measured423ms service p95. This shows the capacity wait contributes
near the full hold, but retry timestamps were not captured, so the500ms spacing
effect is not isolated. Program `prog_e2413eb5-c5d8-40da-971b-7277f0cb09bf`;
evidence: `evidence/rehabilitation/session-admission-capture-w188-2026-09-24.json`.
Final driver inventory returned to0 sessions /0 active recordings. Next: select
an independent outcome with an existing owner-level evidence path; do not claim
a retry-policy latency delta from this single controlled sample.

## W189 — Linux resource-budget owner and current-build reconciliation — 2026-09-24 UTC

Added `api/cmd/resource-budget-cohort` and
`api/internal/resourcebudgetqualification`, plus a `resource-budget` provider
capability and exact rehabilitation-phase descriptor. Focused Go/provider tests
pass. One managed owner measured idle API+driver PSS119,444KiB, average CPU
0.40%, p95 CPU1.00%, and fixture-browser-plus-shell PSS469,590KiB on build
`sha256:2aef016b…`; however its61 samples covered only59,988ms, so it failed
the60,000ms requirement and earns no resource-budget credit. Fixed the owner to
take62 points, made elapsed duration part of its own pass condition, and added a
short-window validator regression. Windows private memory remains unmeasured.

The sole exact `rehabilitation-evidence` phase this cycle,
`20260924-231457-1a4319c6`, failed because profile/passive receipts were missing
for the candidate, cancellation was bound to the prior build, and the resource
window was12ms short. The focused cancellation Go owners pass under `-race`.
The profile seed owner timed out waiting for its browser fixture. The current
driver inventory shows10 Experience Manager/React Component Library preview
sessions; none were closed or interrupted. Until they release, managed restart,
capture, profile/passive owners and the corrected idle sampler remain deferred.

After BAS was refreshed onto `2aef…`, the governed setpoint read0/17 in-band and
17 unavailable; the W188 4/17 result is historical for this candidate. This
candidate change was costly to current receipts and did not advance the score.
No broad Test Genie run occurred, no cross-platform claim is made, and no net
complexity reduction is claimed. Next: run build-bound owners and capture after
the managed driver pool is idle, then run one exact provider phase and setpoint.

## W189 final qualification — 2026-09-24 UTC

Moved current resource-budget, profile, cancellation and passive-fidelity owner
wrappers to `.vrooli/runtime/rehabilitation-evidence/`, which the lifecycle's
build identity explicitly excludes. Receipt creation now leaves the measured
candidate stable across managed restart; `TestScenarioBuildIdentityTracksAuthoredInputsAndIgnoresRuntimeOutputs`
has a direct regression for generated profile evidence. Final candidate:
`sha256:175afb776e50c07cf1f7e0894ee7776d7c70b5053bb84b6ffdb279c42bc2d816`.

On that build, the focused profile owner passed5 seed checks and2 restart checks,
then deleted both synthetic profiles. The cancellation owner passed all five
cases and preserved build identity across its restart. The managed passive
owner passed10,000 browser effects, ordered unique journal IDs and applied
receipts, with0 primary-pool requests; source-current crash/reconnect and
three browser-semantics owners also passed. Resource budget passed62 idle
samples over60,994ms, with223,595KiB max API+driver PSS,0.33% average/1.00% p95
CPU, and429,555KiB fixture-plus-shell PSS. Windows memory remains unmeasured.
Capture measured446ms p95 /658.01ms wall p95 over100 samples plus one warmup.

Test Genie phase `20260924-234946-3c5b1bc9` passed clean L1/Verified for profile
durability, cancellation/recovery, passive fidelity and resource budget. The
persisted setpoint program `prog_43ab1611-3595-43ef-b76c-4af374aa7261` succeeded
and its printed signals read5/17 in-band,12 unavailable,
`product_qualified=false`. `program-runtime library run` nevertheless exited1
with “no recognized envelope status”; the persisted program stdout is truncated
at4,096 bytes, while its visible run output contains the complete correct score.
No broad suite ran. The candidate gained one qualified row this cycle and
evidence handling no longer invalidates its own receipt at creation. No net
complexity reduction is claimed. Final driver health showed10 active external
sessions; none was closed or interrupted.

## W190 — external artifact manifest completeness — 2026-09-24 UTC

The execution writer accepted an external artifact type such as `custom_export`,
persisted it in `result.json`, and silently omitted it from the canonical replay
manifest because the manifest schema has no matching kind. Reproduced with
`TestExternalArtifactsRejectKindsMissingFromReplayManifest`: before the repair,
the owner returned nil and acknowledged the artifact. The writer now rejects
unsupported external kinds before appending or storing them, while internal
step metadata continues to be excluded from the replay-artifact manifest.
`BAS-RF-128` records the defect and boundary. The targeted regression was red
before and the focused `go test ./automation/execution-writer -run
'External|EvidenceManifest|RecordExecutionArtifacts' -count=1` passes after the
fix; scoped `git diff --check` passes. No Test Genie phase or managed restart
ran. Governed read `prog_9944071a-a0db-4e02-8a2a-5b566c95ea03` returned5/17
in-band and12 unavailable on build
`sha256:175afb776e50c07cf1f7e0894ee7776d7c70b5053bb84b6ffdb279c42bc2d816`;
the CLI still exits1 after printing `status=ok` because its envelope recognizer
rejects the result. This local source change has no new setpoint credit, and
the edited source has not yet been deployed. No net complexity reduction is
claimed. Next review angle: fault-injected evidence finalization and retention
with the current driver owner available, then continue on a scorecard gap that
can use focused maintained owners.

## W191 — stored-byte digest binding — 2026-09-25 UTC

The non-inline evidence path previously computed SHA-256 from a source file,
then asked storage to reopen that path and checked only the returned byte count.
A controlled backend replaced the source with different same-length bytes just
before storage; `TestExternalArtifactRejectsSameSizeMutationBetweenHashAndStore`
failed before the repair because the writer returned nil. `ArtifactInfo` now
carries SHA-256 calculated over the bytes each file, memory or MinIO backend
actually writes, and the execution writer rejects digest mismatch before
publishing the artifact. The file backend also reports the number of bytes
copied rather than the earlier stat value. `BAS-RF-129` records the issue.

Focused validation passes: `go test ./storage ./automation/execution-writer
-count=1`, plus `go test ./handlers -run '^$' -count=1` to compile the updated
storage mock. Contract preparation passes with17 outcomes,24 journeys and0
errors. Scoped `git diff --check` passes. The setpoint read
`prog_a10f82a7-7e2a-4534-9912-edc3acc98e0e`
still reports5/17 in-band and12 unavailable on the managed build
`sha256:175afb776e50c07cf1f7e0894ee7776d7c70b5053bb84b6ffdb279c42bc2d816`;
the CLI prints program `status=ok` but exits1 because it does not recognize its
envelope. No Test Genie phase ran and the changed source is not deployed, so
this work earns no new score credit. MinIO package code compiles, but its
testcontainers integration was not run. No aggregate debt reduction is claimed;
the next step remains a candidate deploy when owned driver capacity permits,
followed by focused receipt validation and further domain simplification.

### W191 deployment and current-build qualification — 2026-09-25 UTC

After the driver inventory returned0 sessions, `make restart` deployed the
integrity changes and the managed API became healthy on
`sha256:d0ccb95932afe13a19f9b22e412454d7473e1aab67ab313690d3c5b410c8c74a`.
The current profile owner passed five seed checks, showed its checkpoint at
1881.817ms, passed two post-restart identity checks, and deleted both synthetic
profiles. The cancellation owner passed all five cases and preserved the same
build across its restart. The passive-fidelity owners passed10,000 actions and
fixture effects with ordered journal IDs/applied receipts and0 primary-storage
requests, the 10,000-action process-death/reconnect case, and all three focused
browser-semantics cases. The resource owner passed62 idle samples over at least
60 seconds: max API+driver PSS197,050KiB, average CPU0.377%, p95 CPU1.003%, and
fixture-plus-shell PSS477,013KiB; Windows private memory is not measured.
Current capture workload `2c10c145255eb6f849047281bc603e2c` passed100 samples
plus one warmup at433ms service p95 /629.476ms wall p95 against2000ms.

One exact Test Genie run, `20260925-002836-22a0a111`, passed
`rehabilitation-evidence` at clean L1/Verified for profile durability,
cancellation/recovery, passive fidelity and resource budget. Its evidence tier
is degraded by the dirty shared checkout; those owner tests and receipts were
run separately. Governed setpoint program
`prog_c3a3f8c0-0181-4bf3-9653-b77182cd7b05` returned5/17 in-band and12
unavailable, `product_qualified=false`, on the live `d0ccb…` build. The program
prints `status=ok` but exits1 because the CLI does not recognize the result
envelope. Current driver inventory is0 sessions /0 recordings and the API is
healthy on the same build. No new score row was gained; the two integrity fixes
are now deployed. No aggregate complexity reduction is claimed. Windows memory,
the other12 outcomes and material structural-debt reduction remain open.

The authored issue/progress/feedback records were updated after that live score
read. Lifecycle build identity includes these docs, so a future managed build
will have a new identity and require fresh build-bound receipts; the currently
running service remains on `d0ccb…` until its next managed restart. This is
recorded as a qualification boundary rather than silently reusing those
receipts for another build.

## W192 — bounded rehabilitation setpoint output — 2026-09-25 UTC

The required setpoint program serialized a 4,704-byte result against the
4,096-byte Program Runtime cap. The output cut off the final rows, and the CLI
returned exit 1 despite printing `status=ok`. Four current owner rows repeated
the same run ID, timestamp, managed build digest, phase status and evidence
tier. The reader now reports that shared receipt once under
`signals.owner_evidence`, leaves each row's capability level and cleanliness
beside its row, and uses the exact findings binding ID as its evidence source.
All 17 row IDs, targets, verdicts and unavailable reasons remain present.

Targeted checks pass: Python compilation, `refactor_contract.py` preparation
(17 outcomes, 24 journeys, no errors), and the live required setpoint command.
The resulting output is 3,634 bytes under the 4,096-byte cap; it exits 0 and
parses as 17 rows, 5 in band, 12 unavailable, `product_qualified=false`. No
Test Genie run or managed restart was needed. This repairs feedback integrity,
not a product-outcome row; no debt-reduction claim is made. `BAS-RF-130` and
`BAS-FB-026` record the defect and operator steering. Continue on an outcome
with an existing focused owner, prioritize qualifying additional rows, and
avoid repeating evidence already current on `d0ccb…`.

## W193 — redact sensitive network evidence — 2026-09-25 UTC

**Prior art and hypotheses.** Reused W190/W191's evidence-integrity review and
searched `search-hub` for network-query credential persistence; retrieval
returned no directly relevant BAS fix. The default `EvidencePolicy` already
sets `RedactNetwork` and lists sensitive query/header names. H1: the recorder
does not apply that policy before publishing artifacts. H2: a downstream
manifest or storage adapter sanitizes the event after the recorder copies it.
A targeted synthetic artifact test distinguished them: before the change,
`result.json` contained `token=synthetic-url-secret`, confirming H1 and
rejecting H2.

The evidence package now redacts URL user information and sensitive query
values, copies/redacts selected request and response headers, removes
unstructured body previews, and redacts sensitive fields in structured JSON.
The writer applies that policy before creating inline network artifacts,
stored network snapshots and timeline final URLs. The new end-to-end regression
checks inline, stored and timeline outputs with synthetic query, header and body
secrets. It passed after the fix; focused tests also passed for
`services/evidence`, `automation/execution-writer`, `services/retention` and
`handlers/executions`. No broad suite or Test Genie run was used.

`BAS-RF-131` records the defect. The managed restart deployed the source fix on
`sha256:88714c68393931cc4bfc86bed3fb26fae6adfc676310af39f2a37b4b8eb1b4c9`.
Current-build owners passed for capture (100 measured samples plus warmup; 757ms
service p95 and 945.13ms wall p95), passive-fidelity (managed 10,000-action
journal, process-death/same-ID retry, and three event-semantics cases), profile
durability (alpha/beta isolation through managed restart), cancellation/recovery
(five interruption cases), and resource budget (62 samples over 60 seconds;
177,349 KiB maximum API+driver PSS and 0.34% average CPU). The exact
`rehabilitation-evidence` Test Genie phase `20260925-011202-2bdc2158` passed.

The required setpoint completed as
`prog_0ab3f232-2d67-4620-bc21-84fc73cd9ca9`: 17 rows, 5 in band, 12 unavailable,
`product_qualified=false`; its 3,635-byte context remains within the 4,096-byte
limit. The redaction fix does not own a score row, so this cycle added no point.
No net structural-debt reduction is established. Windows private memory remains
unmeasured, and opaque network previews are deliberately removed. The earlier
3/17 feedback is now stale against the latest read, but the pace concern remains
valid: the score did not move this cycle. Keep validation targeted and prioritize
an owner that can turn one of the 12 pending rows into governed evidence.

## W194 — evidence-completeness owner and governed sensor — 2026-09-25 UTC

**Work ladder: W2 evidence.** The contract row and band are established; the
missing layer was retained owner evidence plus the provider/setpoint join. No
contract or target changes.

Added a durable current-candidate evidence-completeness path. The targeted
`TestInlineTelemetryRemainsAttributableWhenSnapshotStorageFails` proves that
when optional console/network snapshot storage rejects writes, inline events are
still written with payloads, IDs and SHA-256 references in the replay manifest.
The owner runner at `api/cmd/evidence-completeness-cohort/qualification.mjs`
executes only four focused tests and retains their `go test -json` logs. The
verifier checks actual pass events, exact owner test names, raw-log hashes,
source/contract digests and live build identity. The validation provider and
`rehabilitation-evidence` phase now expose a clean L1 evidence-completeness
capability, which the governed setpoint consumes independently.

The current-build owner passed: screenshot write failure retains explicit
failure evidence; console/network inline artifacts remain attributable when
snapshot storage fails; failed/missing video and trace stores are not
acknowledged; active evidence cannot be deleted during export. Profile
seed/verify passed all five seed checks and two post-restart checks with alpha /
beta isolation, a 1,214.62 ms checkpoint, and cleanup of both synthetic
profiles. The cancellation owner passed all five cases. The managed
passive-fidelity owner passed 10,000 actions/effects with ordered journal IDs
and applied-input receipts, zero primary-pool writes, process-death/same-ID
recovery, and three browser event semantics cases. Capture operation
`2d5cc714f3d16e734814f2e58cbc2642` passed 100 samples plus one warmup at 446 ms
service p95 / 618.08 ms wall p95 against 2,000 ms. The resource owner passed 62
samples: 153,169 KiB maximum API+driver PSS, 0.38% average CPU, 1.00% p95,
450,918 KiB fixture-plus-shell PSS; Windows private memory
is not measured. All owner receipts match current managed candidate
`sha256:9b3f96dc401ac5d5344a0dbe5a36c2c4d7d3520ad5985f50b472dbc598d716dc`.

The first exact provider phase `20260925-014636-b85a71f2` failed only because
profile seed and verify artifacts lacked the root-level wrapper expected by the
provider. The two underlying owner receipts passed and matched the build. A
wrapper was assembled from those retained receipts; rerun
`20260925-014741-98d9b54b` passed the exact phase at L1/Verified. The required
setpoint `prog_92687dca-9ec0-46e7-8b40-60f65c0c6abd` then advanced from 5/17 to
6/17 in-band, with 11 unavailable and `product_qualified=false`. Its serialized
context is 3,676 bytes of the 4,096-byte cap. The phase evidence tier is
degraded in the dirty shared checkout; the six passing setpoint rows are
individually in-band, but this does not qualify the product as a whole.

Validation: `GOTOOLCHAIN=local GOPROXY=off go test
./internal/evidencecompletenessqualification ./handlers/profilevalidation
./automation/execution-writer ./services/retention -count=1`,
`node --check cmd/evidence-completeness-cohort/qualification.mjs`, Python
compilation, contract preparation (17 outcomes, 24 journeys, zero errors), and
scoped `git diff --check` pass. No broad test suite ran. No material aggregate
debt reduction is established.

Unverified: 11 remaining outcome rows, Windows private memory, cumulative
structural-debt reduction and ownership of 10 driver sessions seen after the
qualification; none were closed. Test Genie restarted healthy but its optional
`agent-inbox` dependency did not start, so keep the reported phase evidence tier
degraded. The profile receipt wrapper is still assembled one-off, a recurring
maintenance friction to replace with a maintained owner assembler. The required
`browser-automation-studio.learning-maintain` program lookup returned library
404; fallback discovery found no matching program, so no missing learning data
is treated as zero. Preserve the live BAS candidate and do not restart it while
those unowned driver sessions are active. Next investigate a directly measurable
pending outcome and automate the profile wrapper in a way that can be deployed
once current sessions are released.

## W195 — profile receipt assembly and candidate refresh — 2026-09-25 UTC

**Work ladder: W2 evidence.** Contract and provider requirements were already
established. The defect was a repeated evidence-producer gap: profile seed and
verify stages retained valid receipts, but a one-off script had to shape the
provider's root-level cohort receipt.

Extended `api/cmd/profile-durability-cohort/qualification.mjs` with an `assemble`
stage. It reads the two maintained stage receipts; rejects stale contract,
harness, failed checks, cleanup errors or live-build mismatch; recomputes source
and raw-receipt hashes; and writes the provider-shaped cohort receipt. Added the
seed/restart/verify/assemble protocol to `docs/internal/TESTING.md` and resolved
`BAS-RF-133`. No production behavior or target band changed.

The first seed was produced before restarting the edited runner; the managed
build changed from `sha256:9b3f96dc…` to `sha256:1eb1349c…`, and `assemble`
correctly rejected that mismatch. The owner cleanup removed both synthetic
profiles. A second seed on the settled build passed all five checks with a
2,049.22 ms checkpoint. After zero active sessions were confirmed, the managed
restart completed; verify passed alpha and beta recovery and deleted both
profiles; `assemble` wrote
`.vrooli/runtime/rehabilitation-evidence/profile-durability-w195b.json` with
`allChecksPassed=true`. This red/green result exercises the new build guard and
the maintained success path.

Current-build owners refreshed without a broad suite: capture operation
`327d8da6dfbcb93ff39d50fed0daeb8d` passed 100 samples plus one warmup at 463 ms
service p95 / 668.832379 ms wall p95; resource-budget receipt
`resource-budget-w189-receipt-2026-09-25T02-06-44-721Z.json` passed 62 samples;
managed passive fidelity passed 10,000 actions, process-crash same-ID retry and
three selected event-semantics tests; cancellation/recovery receipt
`cancellation-recovery-2026-09-25T02-12-18-037Z-fbfaf775.json` passed five
cases; evidence-completeness receipt
`evidence-completeness-2026-09-25T02-14-03-779Z-1f0268b9.json` passed all four
named tests. The exact provider phase `20260925-021422-48018ca9` passed at L1
with all five capabilities clean. Required read
`prog_2edf5b9a-e237-4a0f-86cb-def17d3512d2` reports 6/17 in-band, 11 unavailable,
`product_qualified=false`, and 3,679 context bytes. Its evidence tier is still
degraded in the shared dirty checkout.

Validation included the real seed/managed-restart/verify/assemble owner path,
`node --check api/cmd/profile-durability-cohort/qualification.mjs`, the focused
owners above, one exact Test Genie provider phase, and the required setpoint
read. The two BAS restarts occurred only after driver inventory showed zero
sessions; the driver now reports zero sessions. No broad suite ran and no
material aggregate debt reduction is claimed. The added assembler removes
recurring ad-hoc receipt composition, but passive-fidelity wrappers remain a
known one-off join. Eleven other outcomes, Windows private memory and cumulative
structural-debt reduction remain unverified. Next, inspect the interactive
feedback/readiness owners for a direct measured pending result; avoid repeated
broad Test Genie execution.

## W196 — refresh local interactive-feedback cohort — 2026-09-25 UTC

**Work ladder: W2 evidence.** Prior W164 work corrected the event clock and
measured the live local UI path. The local cohort was stale for current build
`sha256:1eb1349c…`; remote evidence and the sensor binding remained absent. The
falsifiable question was whether the current candidate still meets its local
input-to-affected-paint bands over 1,000 fully correlated inputs.

Ran only the existing live owner:
`BAS_REHAB_LIVE_API_BASE=http://127.0.0.1:17116/api/v1`
`BAS_REHAB_LIVE_UI_BASE=http://127.0.0.1:21794`
`BAS_REHAB_LIVE_SAMPLE_COUNT=1000 pnpm exec jest tests/integration/input-feedback.test.ts --runInBand --coverage=false --testNamePattern='correlates live UI inputs'`.
The first run passed but the shared Jest setup suppressed its `console.log`
report. After inspecting `tests/setup/silent-logger.ts`, repeated this one cohort
with `BAS_JEST_VERBOSE_LOGS=1` so the measurement was retained. On managed build
`sha256:1eb1349c…`, all 1,000 input events matched applied receipts and decoded
viewer-canvas marker pixels; p50/p95/p99 were 36.4/39.1/51.4 ms. Stage p50s
were 0.1 ms capture-to-socket-send, 17.1 ms send-to-applied-ack, and 19.2 ms
ack-to-canvas pixels. Local thresholds pass. Raw producer log:
`.vrooli/runtime/rehabilitation-evidence/interactive-feedback-live-w196-verbose.log`;
bound diagnostic:
`.vrooli/runtime/rehabilitation-evidence/interactive-feedback-local-w196.json`
(SHA-256 `7bc34d2f3f5790b29f97ba9f8f5ff20bc1fcab0f0e3a3b9ba0dc7d0f79161ddb`).

Prior-art review used `BAS-RF-009`, W164's current-clock cohort and W165's remote
attempt. `vrooli-bridge` docs and skill confirm remote operations are owner- and
grant-bound. The prior denied scenario-read remains unchanged; no repeated call,
grant change, or shell bypass was attempted. Remote p95 remains unknown and the
sensor is not connected, so the outcome remains unscored. The evidence shows no
need for a local latency optimization on this candidate; it does not establish
remote performance or user-perceived performance on other networks.

No product source changed and no services restarted. The independent local
owner test passed; no Test Genie run was warranted because neither code nor the
provider descriptor changed. Required setpoint remains 6/17 in-band, 11
unavailable, `product_qualified=false`. No material debt reduction is claimed.
Required setpoint `prog_dfda1ed2-a113-4158-a600-9821435b7c56` reports 6/17
in-band, 11 unavailable, `product_qualified=false`; context is 3,679 bytes and
the setpoint's global evidence tier remains degraded. Next inspect BAS-RF-007/RF-034 frame-stream owners for a focused motion cohort;
keep RF-009 open until both local/remote evidence and the governed sensor exist.

## W197 — bound slow-reader backlog in frame senders — 2026-09-25 UTC

**Work ladder: W3 implementation, BAS-RF-007 / motion.** The source-level
falsifiable gap was that CDP and polling senders invoked `ws.send()` without
checking Node WebSocket `bufferedAmount`; a slow reader could accumulate queued
frame bytes. Added a shared 12 MiB + 4 KiB ceiling (one maximum recording frame
plus transport envelope). Both senders skip frames that would exceed the bound
and report `ws_backpressure`; CDP retains its replaceable newest-pending-frame
behavior. Documented the boundary and remaining qualification needs in
`docs/concepts/ARCHITECTURE.md` and BAS-RF-007.

Focused validation on the source tree:
`pnpm exec jest tests/unit/frame-streaming/cdp-screencast-strategy.test.ts
tests/unit/frame-streaming/polling-strategy.test.ts --runInBand --coverage=false`
passed 43/43 tests, including threshold regressions for both senders;
`pnpm exec tsc --noEmit` passed; source ESLint reported zero errors and two
existing warnings; `git diff --check` passed. The additional eslint attempt
including both existing test files reported violations throughout those suites;
no clean test-file lint result is claimed. No live service was
restarted and no Test Genie phase ran. This does not certify the five-minute
30 FPS/<=100 ms motion band or API/UI slow-reader and decode behavior.

Required final setpoint `prog_0899cdfb-c9c3-4e58-a38b-d5cd54281618` reports
6/17 in-band, 11 unavailable, and `product_qualified=false`; motion remains
`pending_telemetry`. Contract preparation and inventory validation pass. No
performance, source-size, complexity, duplication
or coupling reduction was measured. Next build the maintained motion owner,
retained receipt and governed sensor, then run the five-minute fixture with
slow-reader and render observations; do not claim credit from these unit tests.

## W198 — capture refresh and passive-fidelity receipt assembly — 2026-09-25 UTC

**Work ladder: W2 evidence repair, BAS-RF-134; targeted live capture diagnosis.**
On current build `sha256:ca2507b3…`, the first required phase showed four
capability owners clean and passive fidelity unavailable. Inspection found the
current raw 10,000-action, process-death and three semantics observations, but
the maintained validator only consumes provider-shaped wrappers; W195 had
composed those manually. Added
`api/cmd/passive-fidelity-cohort/qualification.mjs` and its Node owner tests,
documented its inputs/procedure in `docs/internal/TESTING.md`, and resolved
BAS-RF-134. It validates raw behavior fields, current managed build, source
hashes and artifact digests; it refuses stale builds, missing inputs and
overwriting retained wrappers.

The capture 429 was checked with 12 sequential managed CLI captures; all passed
and driver inventory returned to zero after every call. Since the 100-capture
owner itself did not leak sessions in this repro, no product lifecycle or pool
capacity change was justified. A fresh governed run then passed 100/100 samples
plus one warmup at 457 ms p95 / 694.706595 ms wall p95. Existing W198 managed
passive-fidelity outputs were assembled without repeating their 10,000-action,
crash/reconnect and browser-semantics execution.

Validation: `node --check api/cmd/passive-fidelity-cohort/qualification.mjs`;
`node --test api/cmd/passive-fidelity-cohort/qualification.test.mjs` (6/6);
`cd api && GOTOOLCHAIN=local GOPROXY=off go test ./internal/passivefidelityqualification
-run '^TestValidateComposesCurrentManagedCrashAndSemanticsOwners$' -count=1`;
Performance Health capture operation `2ace7f2f0d424457005a85bece6fc269`; exact
Test Genie `rehabilitation-evidence` phase `20260925-031042-19f0ecad` (five
capabilities L1/clean); required setpoint `prog_b9c05e43-faf9-47f9-ad3f-2709e12e81fe`
(6/17); `refactor_contract.py`; and `refactor_inventory.py --include-untracked`.
The first exact phase attempt `20260925-030515-adf0f67c` correctly rejected the
missing passive wrapper. No service restart or runtime product-code change was
needed. The shared checkout leaves evidence tier degraded. The 11 unavailable
contract outcomes, remote interactive latency, motion, readiness, soak,
portability and aggregate complexity reduction remain open. Next qualify a
single currently unavailable journey with its existing owner and a targeted
receipt; preserve the 6/17 score.

## W199 — current-candidate full-stack frame cohort — 2026-09-25 UTC

**Work ladder: W2 current-candidate performance evidence for BAS-RF-007 and
BAS-RF-009.** W197 changed CDP and polling senders to cap queued WebSocket bytes.
The prior real-Chromium and live workspace cohorts were measured on an older
candidate, so the next discriminating check was whether the updated sender still
produced correct pixels at interactive latency on the current candidate.

The focused driver test `correlates input receipts with pixels in native
streamed frames` passed 1,000/1,000 correlated input/frame decodes on Chromium
136.0.7103.25; p50/p95/p99 were 33.65/34.86/36.99 ms. Then the one live
managed-BAS test `correlates live UI inputs with applied receipts and
viewer-canvas pixels` passed 1,000/1,000 receipt/pixel correlations through the
Go API relay and BAS canvas on build `sha256:ca2507b3…`. Full-path p50/p95/p99
were 36.5/38.9/54.6 ms; stage p50/p95 were 0.1/0.2 ms input-to-send,
17.1/18.2 ms send-to-applied-ack and 19.3/25.7 ms ack-to-canvas. Receipt
sequences were monotonic; the owner session was closed and the driver returned
to zero sessions. Raw log SHA-256 is
`93280ec0d26b58fee95effd8e9d399981380a0c7c019e6b8e1ba0a55999dea69` and the
source-bound summary SHA-256 is
`f5641f56313c1c885d5a73d652a16c288bd986cf34a578ab7710df120a5c733e`.

Focused command:
`BAS_JEST_VERBOSE_LOGS=1 BAS_REHAB_LIVE_API_BASE=http://127.0.0.1:17116/api/v1 BAS_REHAB_LIVE_UI_BASE=http://127.0.0.1:21794 pnpm exec jest tests/integration/input-feedback.test.ts --runInBand --coverage=false --silent=false --testNamePattern='correlates live UI inputs with applied receipts and viewer-canvas pixels'`.
It passed in 51.787 seconds. The driver-only test passed in 35.546 seconds.
No Test Genie phase or broad suite ran. The first post-run reporting step used a
wrong filesystem path and exited after the Jest test had already passed; the
correct path was used to parse, hash and verify the retained log and owner
session cleanup.

The new cohort is current local evidence, not the complete interactive row or
motion qualification. Remote p95/sensor integration and five-minute autonomous
motion with slow-reader/decode evidence remain open. Required setpoint stays
6/17; the overall evidence tier remains degraded. No product code, service or
runtime identity changed; no aggregate complexity/debt reduction is claimed.
Next implement or reuse the smallest maintained five-minute stream owner that
measures actual API/UI frame age and queue/decode behavior under a slow reader.

## W200 — targeted slow-decoder ownership regression — 2026-09-25 UTC

**Work ladder: W2 targeted evidence for BAS-RF-007.** The current UI stream
owner serializes image decoding and replaces its pending frame with the newest
frame. Ran only the related UI hook and statistics tests:
`pnpm exec vitest run src/domains/recording/capture/useFrameStream.test.ts src/domains/recording/hooks/useFrameStats.test.ts --reporter=dot`.
Both files passed, 41/41 tests; the burst case sends 100 frames while a decode
is held and asserts one active decode plus one newest pending frame. This
establishes the existing bounded decoder behavior on source, not sustained
30-FPS rendering, end-to-end frame age, or API-side slow-reader bounds.

The required setpoint read
`prog_9139fd91-bf31-46d6-900d-d7aae7d832bb` reports 6/17 in-band, 11 unavailable,
`product_qualified=false`; its evidence tier is degraded. The operator reports
3/17; the discrepancy remains unresolved, so neither count is substituted for
the other. No Test Genie phase ran, no service restarted, and no source/build
identity changed. Aggregate complexity, debt, duplication and coupling were
not measured. The focused slow-decoder result is recorded in BAS-RF-007; the
five-minute changing fixture, render-age telemetry, relay queue observation,
motion receipt and sensor remain the next concrete work.

## W201 — motion owner and capture-rate diagnosis — 2026-09-25 UTC

**Work ladder: W2 evidence and W3 owner-boundary repairs for BAS-RF-007.**
W201 aligned the motion row with the existing BAS-owned
`rehabilitation-evidence` phase, added the motion standing to the BAS provider
and governed setpoint reader, and added source/build/artifact-bound receipt
validation. The Go viewer relay now counts queued plus actively written binary
frame bytes per client and drops over-budget frames; its focused test admits
9,440,256 bytes under the 12,587,008-byte ceiling and drops the next frame.

The managed startup smoke initially exposed a test counter that could never
advance before measurement collection began. A lifetime-observation counter
fixed that path; managed session, viewer canvas and incoming frames then passed
the 1.4-second focused startup check. The five-minute owner ran on the managed
candidate but failed its required throughput assertion: 6,649 rendered and
unique fixture frames over 300 seconds (22.163 FPS), p95 frame age 23 ms,
maximum age 41 ms, p95 decode 3.4 ms, maximum decode 31.6 ms and maximum JPEG
6,899 bytes. The retained raw observation is
`.vrooli/runtime/rehabilitation-evidence/motion-live-owner-2026-09-25T04-10-51.530Z-ebb7abae-c92d-4132-b458-0d8b0db7e2c7.json`.
Because baseline FPS failed, the 18-second slow-reader cohort and motion receipt
were not produced; motion remains unqualified.

Three hypotheses localized that failure. Relay backlog was unlikely because
the viewer rendered essentially every frame it received, and the separate
byte-cap regression passed. UI decode was unlikely because p95 decode was 3.4
ms and frame age stayed below 50 ms. The managed capture producer became the
leading boundary: short 1080x836 and 720x480 probes both returned about 22.2
viewer FPS. The driver settings endpoint then returned 503 because its
fractional `current_fps` (22.28) was decoded into a Go `int`. Both response
types now preserve `current_fps` as `float64`; focused driver JSON-decoding and
API-handler regressions pass. On the rebuilt healthy candidate
`sha256:5d8e6a9ca4063983630caef1f727590cdfe34235da0a73e403ac8cf5dc966947`, the
endpoint reports `current_fps=22.42` beside the viewer's 22.4 FPS. This confirms
the shortfall exists in the managed capture path before the Go relay and UI;
the exact CDP/compositor cause remains open. A five-second
diagnostic is retained at
`.vrooli/runtime/rehabilitation-evidence/motion-driver-probe-w201.json`.

Focused validation passed for the relay, driver wire decoder, API stream-settings
handler and motion receipt packages; both UI and Playwright-driver TypeScript
checks passed, and the owner script, contract and Test Genie descriptor parsed.
The current source-digest lists contain the same 19 files. No Test Genie phase
Before refreshing owner receipts, `prog_fe7477b4-98f1-470f-a9ff-641d19168c13`
reported 0/17 in-band and 17 unavailable because no current-candidate evidence
phase existed. W202 restored two current points but did not qualify BAS. The
exact 30 FPS motion owner remains the next repair target; do not repeat its
five-minute cohort until a short probe reaches the band. No net aggregate
complexity or debt reduction is claimed.

## W202 — current-build score recovery and capture cadence probe — 2026-09-25 UTC

**Work ladder: W2 evidence refresh plus W3 diagnostic instrumentation for
BAS-RF-007.** Following the operator's pace feedback, use only focused owners
and the exact `rehabilitation-evidence` phase. The first resource-budget attempt
was correctly rejected while the driver held one session; no caller session was
closed. After the driver reported zero sessions, the 62-point owner passed on
build `sha256:f5a799f9ed042bec871dba15b3bcc757062da3df97e17b99349cb0bb25504095`:
173,923 KiB idle PSS, 0.295% average / 1.003% p95 CPU and 557,469 KiB fixture
plus shell PSS. Receipt:
`.vrooli/runtime/rehabilitation-evidence/resource-budget-w189-receipt-2026-09-25T04-59-00-909Z.json`;
raw artifact SHA-256 `2a7e546ae42adf79d6d057af7d294d9c89511461c12caae43133a38047d2a71b`.

The first five-second motion invocation used the API root without `/api/v1` and
failed session creation with 404 before capture; the corrected focused invocation
passed in 6.9 seconds and closed its session. It captured 106/106 unique frames
(21.2 FPS), p95/max frame age 24/41 ms, p95/max decode 3.6/4.3 ms and max frame
size 6,876 bytes. Driver `current_fps` was 21.15. Median inter-frame gap was
49 ms (p95 57 ms), so decode and viewer lag remain unlikely causes. The raw
probe is `.vrooli/runtime/rehabilitation-evidence/motion-ack-probe-w202.json`.
CDP ACK RTT and compositor timestamps were added to the existing periodic debug
log, but the managed driver runs at `LOG_LEVEL=info`; this probe did not observe
those debug fields. Do not claim an ACK/compositor root cause from this attempt.

Performance Health capture owner `2668587da11b742ddeb3a2563cef4816` passed on
the same build: 100 samples plus one warmup, 436 ms p95 and 609.014 ms wall p95
against a 2,000 ms budget. Its provider receipt is retained at
`~/.vrooli/test-runs/performance-health/workloads/2668587da11b742ddeb3a2563/producer/receipt.json`.
The first exact evidence phase ran before this capture and could not satisfy the
setpoint join. After capture, phase `20260925-050213-c6dc7561` still reported
five stale/missing owner capabilities: evidence completeness, profile
durability, cancellation/recovery, passive fidelity and motion. Resource budget
was L1/Verified/clean. The terminal findings are provider evidence, not a
passing phase receipt.

The subsequent governed read `prog_8e272509-b797-4e07-b1ec-70d176505f0e` now
reports 2/17 in-band: capture and resource budget. Ten rows remain unavailable;
five capabilities are readable at L0 due to missing current-build receipts.
`product_qualified=false`. Targeted CDP strategy tests passed 28/28 and driver
TypeScript check passed. No broad Test Genie suite ran; two exact evidence-phase
reads were needed to establish timestamp ordering. No aggregate complexity or
debt reduction is claimed. Next isolate the managed capture's ~49 ms frame gaps
with observable timing (the present debug-only ACK instrumentation was
unavailable at runtime) and pursue the five stale-owner receipts that do not
depend on the motion fix.

### W203 scope record, before implementation

BAS-RF-007's next discriminator is ACK roundtrip versus CDP frame cadence, but
the W202 debug log cannot be observed: `LOG_LEVEL` is runtime-editable while its
listener only emits a note and contains a TODO to update Winston. Repair the
existing driver logger/config owner in `playwright-driver/src/utils/logger.ts`
and `playwright-driver/src/runtime-config.ts`, with focused unit coverage. Keep
the existing CDP timing instrumentation in the frame strategy. The falsifiable
check is that a runtime `LOG_LEVEL=debug` override changes the active logger and
causes one five-second managed motion probe to emit ACK and frame-gap timings;
restore `info` immediately afterward. No launch flags or capture behavior change
unless this measurement isolates a stage. Checks: focused logger/runtime-config
tests, Playwright-driver TypeScript, then the short managed probe. This remains
inside the existing BAS driver owner boundary; no other scenario or host repair
is included.

### W203 — active log-level repair and CDP timing result

Changed `playwright-driver/src/utils/logger.ts` to expose an active level setter
and wired the `LOG_LEVEL` runtime-config listener to Winston, removing its no-op
TODO. Added a regression in `playwright-driver/tests/unit/utils/logger.test.ts`
that switches the live singleton from info to debug. The focused logger suite
passed 14/14 and `pnpm exec tsc --noEmit --pretty false` passed. Managed restart
produced healthy build
`sha256:b32cd6c8d62d772207670aee5870656500b18cb2bc90851f221b6505591d84e3`.
Setting driver `LOG_LEVEL=debug` via its runtime endpoint worked; the probe
restored info through the reset endpoint in a `finally` path.

The five-second owner
`motion-ack-probe-w203.json` emitted ACK timings of 0.484 ms and 0.735 ms with
CDP event timestamp gaps of 52.12 ms and 50.01 ms. The latest probe
`motion-frame-profile-w203.json` captured 110/110 unique frames (22.0 FPS),
driver `current_fps=21.94`, p95/max age 19/21 ms, p95/max decode 3.0/3.9 ms,
and 6,875-byte maximum frame. This rules out slow ACK command roundtrip as the
throughput limiter; compositor/copy/encode cadence remains unresolved. The
auxiliary `/proc` sampler included unrelated Chrome processes and did not
reliably attribute browser CPU/flags, so it is not used as evidence.

The official DevTools Protocol describes `Page.startScreencast` as emitting
frames via `screencastFrame`; its options have no FPS parameter, and it supports
at most three in-flight frames by default. The Chromium implementation captures
on compositor-frame swaps and also caps at three. These details fit the measured
upstream cadence but do not identify its cause. Targeted source URL:
https://raw.githubusercontent.com/ChromeDevTools/devtools-protocol/master/json/browser_protocol.json
and Chromium `content/browser/devtools/protocol/page_handler.cc`.

### W204 scope record, before experiment

Hypothesis: GPU/browser VSync is pacing the headless compositor near 20 Hz.
Perform one reversible managed A/B by launching the driver with temporary
`BROWSER_ARGS=--disable-gpu-vsync`, then run the same five-second motion owner
against unchanged source and compare event gaps and rendered FPS with W203's
22.0 FPS. Capture the launch argument from only the managed Chromium process to
ensure the treatment actually applied. If FPS does not materially improve or
the browser launch is unhealthy, restore the normal managed launch immediately;
do not retain the flag as a workaround. If it does improve, repeat a same-build
control/treatment before changing the persistent launch configuration. This is
an experiment within BAS-RF-007's driver owner; no source or host state changes
are included in the temporary run.

### W204 — verified VSync flag had no meaningful effect

The treatment flag was confirmed on the managed Chromium command line. The
five-second repeat measured 22.2 rendered FPS and 21.96 driver FPS, against the
W203 control of 22.0 rendered FPS and 21.94 driver FPS. This is not a material
gain; the temporary flag was removed with a normal managed restart. The repeat
artifact is `.vrooli/runtime/rehabilitation-evidence/motion-vsync-repeat-w204.json`.
Do not repeat this experiment. Compare fixture animation ticks against CDP
event timing when next pursuing BAS-RF-007.

### W205 — targeted current-candidate evidence recovery

The user reiterated the pace concern and preference for targeted owner checks.
The managed launch was restored and healthy on build `ec52d554…`; no
`BROWSER_ARGS` override remained. The resource-budget cohort passed 62 samples:
100,494 KiB idle PSS, 0.377% average CPU and 1.003% p95 CPU. The declared
capture workload passed 100 attempts plus one warmup at 435 ms service p95 and
610.481 ms wall p95 against 2,000 ms; operation
`2f43088ca97aa7aaed29731890737cb5` is bound to the same build.

To replace stale current-build evidence, the focused evidence-completeness owner
passed its four named Go regressions and retained two raw logs in
`evidence-completeness-2026-09-25T05-32-37-907Z-18cd2c21.json`. The profile
owner passed five seed checks (automatic checkpoint at 179.98 ms), then two
managed-restart recovery checks; it deleted both synthetic profiles. Its
assembled receipt is
`.vrooli/runtime/rehabilitation-evidence/profile-durability-w205-20260925-stage.json`.
The required lifecycle restart preserved build identity and healthy service
state.

The first exact `rehabilitation-evidence` phase after capture/resource owners,
`20260925-053112-0ae277f9`, failed because five unrelated capability receipts
were unavailable; resource budget was clean. After evidence-completeness and
profile receipts landed, one more exact phase,
`20260925-053633-7c1200fa`, confirmed clean L1 for those owners and resource
budget. It still fails on stale cancellation/recovery plus missing passive-
fidelity and motion receipts. Governed read
`prog_3a297117-be77-4a9b-98a7-c6e0e9c8e983` advances from 2/17 to 4/17; 10 rows
remain pending telemetry, three readable rows remain L0, and product remains
unqualified. No broad Test Genie suite ran. Next, refresh one focused
cancellation/recovery owner receipt on this build; do not rerun the phase until
its owner receipt is ready.

### W206 — current-candidate receipts, identity correction and score recovery

The initial cancellation owner attempt began on `ec52d554…`; before it invoked
its managed restart, I had changed authored progress/feedback documents. Because
those documents participate in BAS build identity, the restart moved to
`4dd20bef…` and the owner correctly discarded the mixed-build observation.
This was an ordering error. I kept those documents stable for the next owner
run; its before/after build identities then matched.

On `4dd20bef…`, the focused evidence-completeness owner passed all four named Go
tests; the profile owner passed five seed and two restart checks and cleaned up
both profiles (checkpoint in 2,008.5 ms); the cancellation owner passed all five
cases, including one independently counted external effect per case, resources
1→0, uncertain outcomes retained, retry denied, and 4.08 s managed recovery.
Capture passed 100/100 measured attempts plus warmup at 390 ms service p95 and
598.469 ms wall p95. The first 62-sample resource owner recorded 85,446 KiB idle
PSS, 0.394% average CPU and a 2.000% p95 CPU miss, with four of 61 measured CPU
samples reaching about 2%. One comparable current-build repeat passed at 1.002%
p95 CPU, 0.393% average CPU and 85,446 KiB idle PSS; the first miss is retained
as a brief spike, not suppressed.

The exact `rehabilitation-evidence` run `20260925-055032-16fd93cc` still fails
because passive-fidelity and motion receipts are missing. It confirms clean L1
for evidence completeness, profile durability, cancellation/recovery and
resource budget. Governed read
`prog_fd3b386c-0688-4e16-8108-26b986dcd4eb` is **5/17 in-band, 10 unavailable,
2 out of band**, `product_qualified=false`, on the unchanged `4dd20bef…` build.
No broad suite ran. Next prioritize the direct passive-fidelity cohort, then
revisit the measured motion cadence. Continue to avoid managed restarts until
those current-build receipts are complete.

### W207 — isolate and repair the managed motion pacing deficit

The user repeated the pace concern verbatim in BAS-FB-030. The authoritative
setpoint `prog_b867d05d-2634-4093-92be-5605efbba223` had advanced to 6/17
in-band on the healthy `sha256:4dd20bef…` build after current-build passive-
fidelity owners passed; ten rows remain unavailable and motion is the only
measured row out of band. A five-minute motion owner on this build measured
8,753 rendered frames / 300 seconds = 29.1767 FPS, with 42 ms p95 frame age
and 3.4 ms p95 decode. The corrected fixture independently produced 9,029
paint updates over 300.91 seconds = 30.0056 FPS. A targeted 5-second A/B with
driver cap 60 delivered 150/150 frames at 30 FPS, isolating loss to the 30 FPS
driver pacing cap rather than browser/fixture capture. The owner failed its
30 FPS band; no motion receipt was issued.

A new fake-timer regression reproduced timer-rounding drift at 292 delivered
frames for 300 nominal 30 FPS opportunities. The CDP sender now advances a
phase-preserving frame deadline and skips missed intervals rather than
anchoring every next frame to the delayed send time. The focused CDP test file
passes 29/29 and TypeScript type-check passes. The change is not yet deployed;
the required managed restart will change build identity and stale existing
receipts, so refresh direct owners on the new build before claiming score
recovery. The first exact phase rejected the assembled motion receipt because
the Go validator binds 19 required sources while the owner added the separate
slow-reader validator module. The Go source binding now includes that module;
focused Go receipt and provider tests pass, along with three validator tests and
`git diff --check`. This requires one final managed build and current-build
owner refresh before score recovery. The failed phase also confirmed that six
previously in-band rows need receipts for the new managed build. No broad Test
Genie suite ran.

### W208 scope record, before implementation

**Work ladder: W3 measurement ownership for BAS-RF-007.** Current healthy
candidate is `sha256:fd04d99cf07ded24bfb3b086d5e8e1c029313125858350aceb1524ec9df43a83`.
Five of six `rehabilitation-evidence` owner receipts are current; motion has no
receipt. The isolated five-minute motion observation
`motion-live-owner-2026-09-25T08-19-03.739Z-85fa5fb5-7d96-46a7-9120-042c28b4df37.json`
has 9,003 received, 9,003 decoded, but only 9,001 rendered frames over
300.121 seconds; p95 frame age is 26 ms and p95 decode is 3.4 ms. The
qualification counts rendered frames only while `collecting` is true and turns
that flag off immediately at the boundary, without draining already received
decodes or queued RAF paints.

Hypothesis H1: the reported deficit is partly a measurement-boundary loss of
already received in-window frames. Prediction: tagging frames at receipt and
draining in-window decodes/paints after stopping admissions increases the
render count by exactly the existing received-to-render gap, without changing
the source cadence or adding a frame. H2: the live sender actually delivers
fewer than 30 FPS. Prediction: after the measurement drain, received and
rendered counts still fall short of 30 times the measured window and source
marker gaps remain. H3: shared BAS traffic is the cause. Prediction: throughput
tracks concurrent driver sessions; the prior crowded run at 28.70 FPS supports
this for that run, while the isolated 29.991 FPS run does not.

Discriminator: fix only the owner's frame-window accounting, verify with a
five-second managed probe and frame-count invariants, then run the maintained
five-minute owner only if the probe proves that no accepted in-window frame is
lost at finalization. Keep the 30 FPS target, 9,000 unique-frame floor, and all
age/decode/size/slow-reader limits. No production sender change or band
relaxation is authorized by these measurements. No Test Genie phase or broad
suite has run in this scope record.

### W208 outcome — current-candidate motion and score recovery

On managed build `sha256:d3711b61d0d174b2ef935f13c91a9191d25baacdfd7193ae453b7128a2d3ebb7`, the targeted five-minute motion owner passed with 9,000 received, decoded, rendered, and unique frames over 300,000 ms (29.99999 FPS), p95 frame age 24 ms, p95 decode 3.4 ms, and max decode 6.5 ms. The separate slow-reader window passed with 540 received, 210 decoded/rendered, one active decode, 390 ms maximum frame age, and 9.44 MiB queued against a 12.0 MiB budget. A 60-second instrumented smoke also recorded 1,800/1,800 rendered frames. The probe now samples the marker with two one-pixel rows instead of a 183×27 readback, and the five-minute stop is timed inside the page so automation round-trip overhead is excluded. The 30 FPS band, 9,000-frame floor, and one-frame finite-window allowance are unchanged.

The first current-build motion receipt was not discovered because the browser owner wrote fractional millisecond durations while the Go receipt schema accepts integer milliseconds. `qualification.mjs` now normalizes only the retained receipt durations; `assemble-existing` verified the retained raw observation/logs and wrote a current-source, current-build receipt without rerunning the browser. The exact `rehabilitation-evidence` phase `20260925-094130-171907d2` passed clean at L1 for all six owner capabilities. A current-build Performance Health capture workload then measured 100 samples at 414 ms p95 and 581.5 ms wall p95 against a 2,000 ms budget. The exact phase was run after capture to satisfy the setpoint join's timestamp order.

Governed read `prog_b175136e-904d-44ae-95d6-1b10cf4fa7c0` reports **7/17 in-band**, 10 unavailable, 0 out of band, and `product_qualified=false` on `d3711b61…`. The current board comprises six clean-L1 owner rows and current-build capture; this is four more in-band rows than the operator-reported 3/17. The remaining ten rows are `pending_telemetry`. Three provider phase submissions occurred after diagnosis: the first exposed the fractional-duration serialization defect; the second passed after repair but preceded the refreshed capture, so the governed read correctly rejected its stale timestamp join; the third passed after current-build capture and made the score-bearing join available. No broad suite ran. The operator's pace feedback remains active. Next identify the lowest-cost owner for one of the ten unavailable rows, keeping the current 7/17 candidate as the baseline and favoring targeted evidence over extra phase runs.

### W209 scope record — readiness owner feasibility

**Work ladder: W2 evidence for `readiness`.** The contract requires 100 warm trials with usable-tab p95 at most 1 second and 30 cold trials with usable-browser p95 at most 5 seconds per required platform; external site loading is excluded. The managed Playwright driver has a direct production `POST /session/start` route. It returns ready only after the session manager reports an operational session and active page. `BrowserManager` pools Chromium and keeps it alive until driver shutdown, so warm session admission and cold browser launch must be measured as separate cohorts. The current row has no retained owner receipt or governed reading.

At 2026-09-25 09:53 UTC, the read-only driver session snapshot showed **10 active sessions at the configured capacity of 10**, with live work across unrelated workflows. No readiness requests were sent, sessions were not closed, and no service was restarted. An admission cohort at that point would add resource contention and could affect other callers, so the controlled measurement was deferred. Contract preparation validation still passes (24 preservation journeys, zero structural errors). The include-untracked inventory at 09:54 UTC observed source digest `f47bea9dba275d69fc4467118bfdb34a02de4b022ab7ede6d4fce62ea9154967`; it is a shared dirty-tree snapshot and does not establish an attributable complexity reduction. Next prepare a receipt-producing warm/cold owner that checks for unrelated live sessions and refuses to run when capacity is occupied; establish that the cold protocol measures the contract's application startup before admitting any result. Continue on another independent repair while the owner environment is occupied.

The current setpoint was reread as `prog_09fcf7b3-5e44-40c0-a286-4f3dd1d030a8`; it remains 7/17 in-band, 10 pending telemetry, and `product_qualified=false`. Its six owner rows still have clean L1 phase capabilities. Inspecting the durable Test Genie record exposed a run-level evidence caveat: `runs list` labels the shared-scope run `degraded` and records 262 dirty scenario paths, while `runs show` reports `shared-scoped`; `runs findings` confirms the phase and all six capabilities passed clean at L1. The board currently credits each capability from those findings, so 7/17 is a current-candidate governed reading, not a clean-checkout certification. Preserve and report this distinction while continuing owner qualification.

### W210 outcome — targeted warm readiness cohort

Added a focused receipt-producing probe at `playwright-driver/tests/rehabilitation/readiness-owner.mjs` with four built-in unit checks. It drives the production API/executor path using an ad-hoc local `about:blank` flow, observes a usable page owned by the matching execution, checks for unrelated sessions before and after each sample, and waits for ordinary cleanup. A three-trial live harness check passed (p95 54.7 ms). The retained 100-trial warm cohort passed **100/100** at **66.2 ms p95** on managed candidate `sha256:d3711b61d0d174b2ef935f13c91a9191d25baacdfd7193ae453b7128a2d3ebb7`, Linux x64 / Chromium 136.0.7103.25. No external site was loaded; the driver was idle before the cohort and returned to zero sessions after it. Receipt SHA-256 is `6ed3fb7461612b0d36939ccb311aacc70c5c48085e8bff8334c506b603983315` at `.vrooli/runtime/rehabilitation-evidence/readiness-warm-owner-2026-09-25.json`.

This establishes only the warm Linux cohort. Cold browser start (30 samples) and the other required platforms remain unmeasured; the contract row therefore remains open and the last governed score remains **7/17**, not product-qualified. The direct owner evidence is not yet wired into the governed `readiness` sensor or its Test Genie performance phase. The current governed reread `prog_9da9f813-a47c-4e79-a16a-cb2b848684f7` confirms 7/17 in-band, 10 pending telemetry, and `product_qualified=false`; no score movement is claimed for W210. Next resolve the cold-start lifecycle/protocol and required-platform set, then join both cohorts through the owning sensor before making a score claim. This work used targeted node tests and direct-owner measurements; no Test Genie phase ran.

### W211 outcome — session-profile validation complexity reduction

**Issue: BAS-RF-015.** The untouched baseline `api/services/session-profile/validation.go` was 177 lines and Tidiness Manager reported `ValidateBrowserProfile` at cyclomatic complexity 62. Its profile-wide sequence mixed preset, fingerprint, behavior, anti-detection, proxy, and header policies. W211 extracted those existing policy owners and subdivided fingerprint into viewport/device/geolocation/color, and behavior into typing/mouse/scroll. Repeated typing and start-delay bound checks now use one shared range helper. No limits, accepted values, validation order, or error wording were intended to change. Baseline SHA-256: `48132ab506a0353a0234a270abaa8f39826be158afbeeb2880069b282bb665be`; current source SHA-256: `bf51417b3b36d90db270985b2089b200a91a6a5368d0b068fbeb4bfe62c08f3a`.

`go test ./services/session-profile -count=1`, `gofumpt`, and `git diff --check` pass. The post-change Tidiness Manager scan reports no `HIGH_COMPLEXITY` finding for `validation.go`; issue #14485 is resolved with the scan/test evidence, and the file visit/campaign note are recorded under campaign `1d662931-1407-4e11-aa8b-99926bf10dcf`. The scenario-wide scan remains at L2 with 287 unknowns. The file grew from 177 to 237 lines, so this is a function-complexity/concentration improvement, not a code-size reduction or proof of domain-wide debt reduction. No Test Genie phase ran.

The managed candidate remains `sha256:d3711b61d0d174b2ef935f13c91a9191d25baacdfd7193ae453b7128a2d3ebb7`; W211's source edit has not been deployed. Fresh governed read `prog_17c612c3-f9c0-41ed-aa76-1f1596538c6e` still reports 7/17 in-band, 10 pending, and `product_qualified=false`, with degraded evidence tier. No score gain is claimed. Next continue from the remaining domain-wide debt baseline and verify/deploy accumulated source changes through the managed lifecycle before refreshing affected build-bound evidence.

### W212 — readiness score-gap triage and focused verification

The operator repeated: “It's been 3/17 in band all day. I feel you are moving too slow.” The governed read `prog_fe4c391f-f0cd-42e5-a12e-6bd51d6baadb` is **7/17 in-band, 10 pending telemetry, 0 out of band**, with `product_qualified=false` on managed build `sha256:d3711b61d0d174b2ef935f13c91a9191d25baacdfd7193ae453b7128a2d3ebb7`. This corrects the operator-reported 3/17 with current evidence; no score movement is claimed for W212.

Readiness was the nearest existing owner result: the retained Linux warm cohort passed 100/100 at 66.16 ms p95. `node --test tests/rehabilitation/readiness-owner.test.mjs` passed 4/4 in 37 ms, and all five source digests plus the retained receipt SHA-256 matched. The contract also requires 30 cold-browser samples at or below 5 seconds on Linux x64, Windows x64, macOS x64 and macOS arm64. Those measurements and the governed sensor join are absent, so the row stays pending; counting warm-only evidence would be an invalid score claim. No source files or managed services were changed, and no Test Genie phase ran. Next pursue an outcome whose owner can satisfy its full band on available infrastructure, and leave readiness open until the cold/platform evidence and sensor exist.

### W213 scope record — publish requested DOM captures as real artifacts

**Issue: BAS-RF-013, DOM artifact path.** The managed goal baseline remains 7/17 in-band with 10 pending, 0 out of band, and `product_qualified=false` on `sha256:d3711b61…` (latest read `prog_b793fe06-5235-4d6b-88a8-41af4f5e61b4`). The existing handler can return inline rendered HTML, but `CAPTURE_TYPE_DOM` is registered to an unavailable producer and `buildAdhocRequest` inserts an evaluate node only for the inline response flags. The narrower reproduction `TestCapture_InlineDom_ReturnsRenderedHTML` requests both DOM and `InlineDom`; it does not assert that the requested DOM artifact exists.

Hypotheses: (H1) a DOM-only capture never evaluates the DOM; prediction: its flow has no evaluate action. (H2) enabling inline DOM evaluates correctly but the artifact remains unavailable because the registry has no DOM file producer; prediction: response `DomHtml` is present while `Artifacts` reports unavailable. Discriminator: add a focused handler test for `Captures=[DOM]` with no inline flag, require an evaluate node and a real `dom.html` artifact with correct bytes, type, reference and durable summary; then test an inline result that is absent to preserve best-effort degradation. If confirmed, make DOM artifact requests add the existing evaluate action, persist the bounded result, and register the file producer. Move storage publication and artifact-summary writing after DOM/DOM_TREE materialization so generated files receive the same publication and manifest handling as exported files; this also removes the currently duplicated premature summary write. No schema or dependency changes.

In-scope files: `api/handlers/capture/service.go`, `producer.go`, `metadata.go`, `inline_dom.go`, `inline_dom_test.go`, `service_test.go`, and focused producer/handler tests. Baseline SHA-256 values: `service.go` `2046a81ed78d9596af716837009af34f469e9b7f075a7f1e60e7ce43d7a680c0`; `producer.go` `b7e72522abe842c04bac1d2bb1934fbeb0c6461d796e4ae4407911d4ede45015`; `metadata.go` `6514c7bed2e0caaf0d2ec3e446682a65a7a9560f64f718d04957cd25c2a209a9`; `inline_dom.go` `e6474efb3bb9c34b00ad91ba5309931cdd114e642d9b0f782adc1462f91fe1fa`; `inline_dom_test.go` `1effa15ee1b82a2208067713a53a3c5220aa78d5bfa0c046a524c239cded1318`; `service_test.go` `e4ef0280593629cc416a014b2b0f2610592c8f6c303a3e4dcab1cc3366d7a4b6`. Verification: focused `go test ./handlers/capture` owners plus `gofumpt` and scoped `git diff --check`; no broad Test Genie run. The unrelated screenshot continuation behavior is intentionally unchanged per current architecture and `TestExecuteExplicitScreenshotOutcome`.

### W213 outcome — requested DOM artifacts now materialize and publish

The focused regression failed before the fix: a DOM-only request built no evaluate node and returned an unavailable artifact. The handler now adds independent HTML and tree evaluate nodes when their artifact types are requested, writes bounded `dom.html` / `dom-tree.json`, marks truncated output in artifact metadata, publishes those files through BAS storage, and writes the result summary after materialization. Inline response fields remain opt-in. The DOM producer reports a precise unavailable reason when evaluation produced no readable result, and the storage loop now skips artifacts marked unavailable instead of attempting to read nonexistent files.

`go test ./handlers/capture -count=1` passes; the explicit screenshot failure/storage-policy owners pass; `python3 docs/internal/refactor_contract.py` reports 24 journeys and zero structural errors; and the scoped `git diff --check` passes. The generated-artifact test checks both DOM types together without inline-response flags, storage URLs, references, exact bytes, paths and result-summary inclusion. Oversized HTML keeps the established 2 MiB bound and now marks the artifact truncated. No Test Genie phase ran. The four production files grew by 53 net lines, while duplicate `result.json` summary work dropped from two writes to one; this is a capability and redundant-I/O repair, not a claim of net domain complexity reduction. The managed build remains `sha256:d3711b61…`; no qualification or score increase is claimed, and VIDEO export/device video remain open under BAS-RF-013.


Final W213 setpoint reread `prog_fe1b9044-5e15-4267-aaa8-f9000dc5c303` confirms the managed candidate is still 7/17 in-band, 10 pending, 0 out of band, `product_qualified=false`, and `sha256:d3711b61…`.

### W214 — close the browser VIDEO export path

The operator said, “It's been 3/17 in band all day. I feel you are moving too slow.” Latest governed read `prog_d63de6e7-96a5-4fb9-bd2b-db18c708118c` reports 7/17 in-band, 10 `pending_telemetry`, 0 out of band, and `product_qualified=false` on managed build `sha256:d3711b61…`. The source changes from W213 and this pass are not deployed; this score is not evidence of their behavior.

**Issue: BAS-RF-013, browser VIDEO artifact path.** Session finalization already reports per-page recording paths, and the execution writer retains them under `artifacts/videos/` after required-video validation. Folder export copied performance and accessibility artifacts but omitted video. The capture registry still returned an unavailable placeholder even for completed executions with recordings. Repair: copy regular `.webm` recordings to `outDir/videos/`, upload each best-effort beside other evidence, expose each exported recording through a VIDEO producer, and retain an explicit unavailable response when there are no recordings. Device-specific video remains separate.

Focused validation passed: `go test ./services/workflow -run '^TestExportVideoArtifacts_' -count=1`; `go test ./handlers/capture -run '^TestCapture_HarvestsVideoRecordingFromExporter$' -count=1`; and targeted VIDEO producer/registry tests. The tests verify copied bytes, exclusion of non-WebM files, per-recording response metadata, missing-recording degradation, and that VIDEO requests set `RequiresVideo`. Scoped `git diff --check` passes. No broad Test Genie phase ran. This is a source-level capability repair; managed qualification and score movement remain unverified. The handler producer, export path and documentation grew; no net complexity reduction is claimed. Next pursue a score-bearing pending row whose owner evidence can be generated on the current managed build, then qualify the accumulated source through the managed lifecycle before attributing runtime results.

Work record: `vrooli-memory` journal entry `4f76eabf-e245-4e86-949d-c1397aeba098`.

### W215 — refresh the local interactive-feedback cohort

The pending row with the closest reusable measurement was `interactive-feedback`: W199's receipt was from build `ca2507…`, and remote latency plus a governed sensor were still missing. Before sampling, BAS health was `healthy` on `sha256:d3711b61…`; the managed driver metrics showed 0 active and 0 total sessions. The 1,000-input live owner test passed on that build with all 1,000 receipts and viewer-canvas paints correlated in monotonic order. Local p50/p95/p99 were **36.7/39.4/53.8 ms**, inside the local limits. Test duration was 51.4 seconds; the driver returned to 0 active and 0 total sessions afterward. Targeted command: `BAS_JEST_VERBOSE_LOGS=1 BAS_REHAB_LIVE_API_BASE=http://127.0.0.1:17116/api/v1 BAS_REHAB_LIVE_UI_BASE=http://127.0.0.1:21794 BAS_REHAB_LIVE_SAMPLE_COUNT=1000 pnpm exec jest tests/integration/input-feedback.test.ts --runInBand --coverage=false --silent=false --testNamePattern='correlates live UI inputs with applied receipts and viewer-canvas pixels'`.

This narrows the next work: no local latency change is justified by this cohort. The contract also requires a remote p95 cohort; the current output did not retain its raw 1,000 samples, and no governed sensor consumes this evidence. It remains diagnostic and cannot increase the 7/17 score. Evidence summary: `docs/internal/evidence/rehabilitation/interactive-feedback-local-w215-2026-09-25.json`. No Test Genie phase ran and no service restart occurred. Next recover a complete source/build-bound local receipt without repeating unchanged tests, then pursue an authorized remote owner and wire both cohorts to the governed sensor; if remote execution remains unavailable, record that limitation and continue with another row.

### W216 scope — make the focused local owner retain its samples

W215 used a targeted owner test but its large stdout record was truncated before it could be retained as complete evidence. Hypothesis: adding an opt-in receipt path to that existing owner will preserve all 1,000 samples, contract/test source hashes and the live API build identity without adding another test or Test Genie run. Keep the normal diagnostic test behavior when no receipt path is set. Verify by running the same focused live test once with a path inside ignored `.vrooli/runtime/rehabilitation-evidence/`, then independently check sample count, source hashes, build identity and the same latency bands. This only closes local evidence retention; remote cohort and governed sensor remain required, so it cannot be claimed as score-bearing.

### W216 outcome — local interactive-feedback evidence is retained

The owner now accepts `BAS_REHAB_RECEIPT_PATH`, validates that its destination stays under the ignored rehabilitation-evidence directory, and writes a complete receipt containing all samples, 1,000 input/receipt/pixel counts, source hashes and the live API build identity. Without the environment variable it retains the existing stdout diagnostic behavior. The W216 focused live owner passes on build `sha256:d3711b61…`: all 1,000 receipts and canvas paints correlated monotonically; p50/p95/p99 were **36.7/39.7/55.2 ms**. The test ran 51.3 seconds and left the managed driver at zero sessions. Its receipt at `.vrooli/runtime/rehabilitation-evidence/interactive-feedback-local-w216.json` (SHA-256 `718e91cc57f118a5cf87c0e07cdb0aa5ddcc14a3dfec901187b3035962ef7f51`) independently validates to 1,000 raw samples, current test/contract hashes, and the same build identity reported by `/health`. Focused ESLint exits 0 with 23 pre-existing unsafe-type/return-type warnings; the new console output warning was removed. No Test Genie phase or service restart ran.

This makes the local diagnostic reusable but does not change `interactive-feedback` qualification: the remote p95 cohort and governed sensor remain absent. No latency optimization is justified from the local cohort. Required final setpoint read `prog_4ba60072-c54b-4678-a529-eef2f7c86fd6` remains **7/17 in-band, 10 pending, 0 out of band, `product_qualified=false`** on build `sha256:d3711b61…`; no score gain or complexity reduction is claimed. Work record: `vrooli-memory` journal entry `3b2474db-180e-4517-9e54-593f64a9b945`. Next pursue an authorized remote cohort and owned sensor only if available; otherwise continue on another full-band outcome.

W216 receipt-owner work record: `vrooli-memory` journal entry `a291ad14-00f8-4147-b6a7-d7549e245c73`.

### W217 scope — centralize typed action-builder dispatch

The current Tidiness Manager scan returns `VALIDATION_STATUS_FAILED`, with L2 maturity, `TIDINESS_BUDGET_EXCEEDED`, 1,228 findings and 289 unknowns; the stored issue queue has 50 open entries. Its current shared-tree score is 65.2/100, complexity 70.4, duplication 25.4%, 200 open violations, 277,029 lines and average function complexity 4.6. These are a mixed dirty-tree baseline, not a clean-candidate claim. A focused review found `BuildActionDefinition` in clean `api/automation/compiler/action_definition_builder.go` is complexity 21 and has one test for only GESTURE; it maps the 18 executable action enums through a long switch while preserving explicit errors for recognized action types without a typed builder. The file baseline SHA-256 is `25055bc6b94af998bf0ed4701570b5841f383f6265c70e1e56c7c3dda2223b43`.

Hypothesis: a static enum-to-typed-builder table will put the action-parameter mapping in one inspectable policy owner, remove the branch-heavy dispatch, and preserve all existing typed wrappers and unsupported-action errors. Add a regression over all 18 currently supported action types plus the known control-flow types that must still fail. Verify with the focused compiler package tests and a fresh Tidiness scan/score; keep the shared dirty-tree limitation explicit and do not claim that this one hotspot satisfies the domain-wide structural-debt band. In-scope files: `api/automation/compiler/action_definition_builder.go`, its compiler tests, `docs/PROBLEMS.md`, and this checkpoint. No schema, public API, dependency, or runtime lifecycle changes.

### W217 outcome — typed action dispatch has one explicit owner

Replaced `BuildActionDefinition`'s 21-branch switch with an enum-keyed builder table. All 18 supported executable action types retain their existing `typeconv` constructors, metadata and oneof wrapper; recognized `set_variable`, `loop` and `conditional` types still fail with the existing missing-builder error, and unknown types retain the unknown-type error. New table-driven tests assert every supported oneof and both error classes. This adds **54 test lines** while reducing production builder code from 104 to 86 lines; no net source-tree size or aggregate-debt decrease is claimed.

`GOTOOLCHAIN=local GOPROXY=off go test ./automation/compiler -run '^TestBuildActionDefinition_' -count=1` and `GOTOOLCHAIN=local GOPROXY=off go test ./automation/compiler -count=1` pass. `gofumpt -d` and scoped `git diff --check` are clean. The before scan observed `BuildActionDefinition` at complexity 21; the after Tidiness scan reports **no finding** for `action_definition_builder.go` and 1,227 findings overall versus 1,228 before. It still returns `VALIDATION_STATUS_FAILED` / `TIDINESS_BUDGET_EXCEEDED`, L2, 289 unknowns; the Tidiness score remains 65.2/100 with 200 open violations and 25.4% reported duplication. No Test Genie phase ran. This is a local complexity fix, not qualification of structural-debt. No service restarted and the source has not been deployed. Work ladder: W3 implementation against the structural-debt hotspot; the outcome stays open pending material domain-wide complexity/duplication/coupling reduction.

The governed setpoint reread `prog_2511fe86-1aae-44d4-b234-49ae711cadd5` remains 7/17 in-band, 10 pending and `product_qualified=false` on `sha256:d3711b61…`. Work record: `vrooli-memory` journal entry `4032de14-0026-4991-9b85-7c42be6f1b19`.

### W219 scope — test J02 input snapshot continuity

**Work ladder: W2 evidence, localized to BAS-RF-004 / preservation journey J02.** Prior art is related, not a recurrence: W098 fixed key/modifier forwarding and observed fresh-context replay, but explicitly left native IME open; W099 fixed `clearFirst` for replacement and replayed into a prefilled field; W167 fixed empty snapshots at stop; W169 separately exercised replay of a cleared value. The current scenario test has no `[REQ:BAS-RH-J02]` owner and the qualification protocol requires paused typing, replacement, deletion, paste, IME, clear, and final-value replay. Do not repeat or revert the prior fixes.

Hypotheses: (1) regular edits are captured as full-value snapshots and only the final same-target value needs replay; falsify this if an independent fixture value log diverges from the captured event snapshots after a paused replacement/deletion sequence. (2) paste and composition commits reach the existing `input` listener with the final value; falsify this if those final values are absent from the recording event stream. (3) stop flush still emits the final empty snapshot; W167 is positive prior evidence, so retain its existing test as the control rather than reworking that code. The discriminating check is a focused real-Chromium test in `playwright-driver/tests/integration/recording-injection.test.ts`, using the injected production capture script, an independent fixture input log, a real clipboard paste where Chromium permits it, an explicitly synthetic composition commit, and stop-time clear. No product source changes are in scope unless the new assertion reproduces a gap. Hashes before this experiment: test `57513cf9…`, recording script `440e6c84…`, workflow action registry `9c21302d…`. Focused owner command: `pnpm exec jest tests/integration/recording-injection.test.ts --runInBand --coverage=false --testNamePattern='J02'` from `playwright-driver/`. Native OS IME and full fresh-context replay remain separate qualification obligations; this test alone cannot close J02 or preservation.

### W219 outcome — browser input snapshots survive the targeted edit sequence

The exact Jest owner `pnpm exec jest tests/integration/recording-injection.test.ts --runInBand --coverage=false --testNamePattern='\\[REQ:BAS-RH-J02\\]'` passes **1/1** in 5.407 seconds (15 unrelated cases skipped). Independent Chromium fixture values and emitted full-value snapshots both include paused initial text, replacement, deletion result, real clipboard paste, synthetic composition commit, and final empty value; the stop-time empty snapshot is last. The actual clipboard path passed in this environment. Composition was simulated with DOM composition/input events; this is not host OS IME evidence.

Focused conversion controls also pass: from `api/`, `GOTOOLCHAIN=local GOPROXY=off go test ./services/live-capture -run 'TestMergeSnapshotsPreservesHistoryAndTarget|TestGenerateWorkflowPreservesRecordedSemantics' -count=1`. These establish same-target final-snapshot merge and typed `clearFirst` conversion, but not replay of this new multi-edit cohort. `python3 docs/internal/refactor_contract.py` reports 17 required outcomes, 24 journeys and zero preparation errors. Scoped `git diff --check` is clean. No production code, complexity, duplication or coupling changed; this adds a test and traceability only. No Test Genie phase or service restart ran.

Final read `prog_738f67de-fbce-4327-8636-be340c290aee` remains **7/17 in-band, 10 pending telemetry, 0 out of band**, `product_qualified=false`, on managed build `sha256:d3711b61…`; the new test cannot change the score because it does not complete all 24 journeys or produce a setpoint sensor. Owner phase `20260925-094130-171907d2` remains passed/degraded. Update BAS-RF-004 with the partial capture coverage and keep it open. Next discriminator: connect a mixed input sequence to typed workflow generation and fresh-context replay; native OS IME remains unverified. Do not repeat the input-stop fix or claim J02 qualification from this test.

### W220 scope — replay recorded input through fresh browser contexts

Continue BAS-RF-004 / J02 after W219 passed the capture-side edit matrix. BAS is healthy on the managed driver/API ports 24485/17116; no lifecycle operation is required. Extend the existing `playwright-driver/tests/e2e/record-mode-e2e.mjs` owner, which already records a browser action, generates and persists a workflow, replays in a fresh driver session, executes through the API, and cleans owned state. Add a local fixture input with independent value observations; record a replacement and final text, assert the captured typed input carries the final snapshot, then assert both fresh-session replay and saved API workflow execution reproduce that value under distinct fixture cookies. Include actual clipboard paste and an emulated composition commit if the fixture path can do so without introducing a production API. Hypothesis: typed snapshot conversion with `clearFirst` plus generated workflow execution preserves the final value across fresh contexts; falsify on missing/mismatched typed payload, reuse of the recording cookie, or fixture final-value divergence. Do not qualify native OS IME or the entire 24-journey preservation row from this narrow owner. Scope is limited to the existing E2E fixture and its focused contract tests; no Test Genie phase or service restart is planned.

W220 investigation extension after the first live attempt: the fixture script queried `#fixture-input` before that element was parsed, so it threw before installing the independent field observer. The recorder itself had captured six ordered events, including full `initialtyped before pause` and `replacement` input snapshots; there was no independent fixture observation or saved workflow, and no external fixture effects. The controlled contract test initially failed because that mock driver cannot synthesize browser field observations; an explicit `--j02-input` flag now keeps the existing mock contract path unchanged, and its full16-test harness passes. The failed live run's close was correctly refused while six entries were unacknowledged, but left its owned session in `closing`; the original process had not persisted its lease. Read-only `/health` and `/observability/sessions` confirm this is the sole live session and its owner is this test. An authorized same-owner `POST /session/start` reattachment was attempted once and returned404 because the session is already `closing`. Extend W220 to move the fixture observer after its DOM nodes, make failure cleanup stop/read/acknowledge its own recording before close, and use the previously granted BAS managed-restart authority to remove this one synthetic closing session. Recheck the session inventory before restart, preserve and verify profile state across lifecycle, then rerun the targeted live owner. No other session or saved workflow may be touched.

### W221 — reconcile the score after the managed restart

The managed restart was necessary to clear the single synthetic W220 session left in `closing`; lifecycle restart completed healthy and returned the driver to zero sessions. It also changed the API build identity from `sha256:d3711b61…` to `sha256:25d9f4a5…`, so prior candidate receipts no longer join to the current build. A read-only rehabilitation setpoint run `prog_ca02e129-7b37-440f-9568-e9734f9f4087` now reports **0 readable / 17 unavailable / 0 out of band**, `product_qualified=false`; capture is explicitly unavailable because its workload receipt is bound to the former build. This supersedes the previous 7/17 reading for the current candidate. The operator's reported 3/17 is not reproduced by either persisted current read in this turn; no in-band count should be inferred from stale receipts.

To discriminate whether any owner receipts remain valid on the current build, ran only `vrooli scenario test browser-automation-studio --phases rehabilitation-evidence`. Run `20260925-122015-5e746711` completed in 1 second and failed with six evidence-completeness findings (`EVIDENCE_COMPLETENESS_INVALID`), yielding L0→L1 with North Star verified. The phase summary is not proof of passing tests. The follow-up setpoint remained 0 readable / 17 unavailable. This was one narrow provider refresh, not a broad suite. Restore current-build evidence only through the exact owner runs needed for selected rows; do not repeat the phase without first repairing/assembling its missing receipts. Near-term W2 continuation is to make W220's managed input owner use the API-owned session lifecycle, then verify generation and fresh replay with that binding. No qualification or score gain is claimed.

### W222 scope — make the J02 replay owner use API-owned page identity

W220 showed that direct driver-only session creation produces input snapshots with a driver page UUID absent from the API logical-page registry. This is an owner setup error: `GenerateWorkflow` intentionally fails closed for ambiguous/unbound targets. Keep that production behavior. Change only the opt-in J02 branch of the existing `record-mode-e2e.mjs` owner to create its recording session, navigate, start/stop recording, forward input, read/acknowledge entries, and close through the managed API routes. Keep direct driver ownership for the already-supported baseline journey and fresh-session replay. Verify that the API callback resolves the page identity, generates/persists the typed workflow, and both direct fresh replay and saved API execution independently reproduce the final input value. Keep the owner cleanup-safe on failure. Re-run 16 controlled contracts, one targeted managed J02 owner, API live-capture generation/conversion tests, contract preparation and setpoint. Do not run a broad suite or weaken the generator's fail-closed page binding.

### W223 scope — bind the synthetic initial navigation to the initial page

The API-owned J02 owner now creates pages through the correct lifecycle, but managed generation fails because its first synthetic `navigate` action has neither logical nor driver page identity while following captured input actions do. The live error is specifically `target "initial" has no logical tab binding`; genuinely identity-less multi-page clicks already fail closed. Add a generator regression with an identity-less first `navigate` followed by a driver-page-bound typed action on the single registered `IsInitial` page. Resolve only that exact first-action case to the registered initial page; retain rejection for identity-less ordinary actions and multi-page ambiguity. Validate the focused Go workflow-generator tests, then rerun the managed J02 owner to test workflow save and both replay paths. Scope: `api/services/live-capture/action_registry.go`, its workflow-generator tests, BAS-RF-004 and this progress record. No schema/API/dependency changes.

### W224 scope — normalize typed input observations before snapshot merging

The managed API owner now passes page resolution after W223 but generation rejects captured `ACTION_TYPE_INPUT` timeline entries as unsupported `input` recordings. These entries carry `InputParams.value`, adapted as `Payload.text`; they are full-value snapshots like the driver's existing `type` observation. Normalize `input` to `type` at the live-capture preparation boundary before `MergeConsecutiveActions`, so adjacent snapshots collapse to the final value and `recordedActionDefinition` retains `clearFirst=true`. Add a regression with intermediate and final API-shaped input records after an identity-less initial navigation; assert the workflow emits one final typed snapshot. Retain unsupported-action failures for unrelated types. Focused Go tests, managed API-owned J02 replay, controlled JS owner tests, contract consistency and setpoint read will verify the repair. Scope remains BAS live-capture derivation only.

### W225 scope — preserve an explicit empty input value through timeline conversion

W224's managed J02 owner now reaches workflow conversion but rejects action 19 because an empty stop-time snapshot became an input payload with no `text` field. `automation/driver/timeline_convert.go` currently omits `InputParams.value` when it is empty, erasing the distinction between an explicit clear and a missing value. Add a driver conversion test for `ACTION_TYPE_INPUT` with `value:""`; require `Payload.text` to exist and equal the empty string. Then always copy the typed input value, including empty, into the legacy payload. Re-run focused adapter/live-capture Go tests and the managed owner; keep empty-clear assertions intact. This is the existing BAS-RF-004 contract, not a new outcome.

### W223 outcome — initial page identity is carried through generation

The API-created managed session fixed the prior test setup but exposed a real mixed-identity case: the synthetic first `navigate` has no driver/logical ID; subsequent input actions do. `resolveRecordedPageTargets` now binds only an identity-less first `navigate` to the registered `IsInitial` page. Existing missing-identity multi-page click rejection, popup alternation and independent-tab opening tests remain green. The focused generator regression failed before the repair and passes afterward. This change preserves fail-closed behavior for all other unbound targets.

### W224 outcome — API input snapshots enter the existing merge owner

Managed generation then rejected the API `input` wire spelling as unknown, while the snapshot merger and compiler owner use `type`. `prepareRecordedActions` now normalizes this spelling before snapshot merging. The regression uses intermediate and final input values after an unbound synthetic initial navigation; it proves one final `ACTION_TYPE_INPUT` node, the final text and `clearFirst=true`. Focused workflow-generation, prior unsupported-page-identity, snapshot-merge and typed-semantics tests pass. No unrelated action types are reclassified.

### W225 outcome — API clear and fresh replay pass end to end

The focused driver conversion regression reproduced loss of an explicit empty input value because `payloadFromActionParams` omitted `text` when the proto value was empty. The adapter now preserves `Payload.text=""`; all `TestRecordedActionFromTimelineEntry_*` and focused live-capture tests pass. After a clean managed restart from zero sessions, the API-owned J02 owner passes **11/11**: independent fixture capture through replacement, deletion, synthetic paste/composition, empty clear and final text; generated/persisted workflow; exact entry acknowledgement; direct replay in a fresh driver context; saved API workflow execution and timeline; then cleanup of both browser sessions, execution artifacts, workflow and project. Both fresh contexts reported the exact final input value. Managed API build at the run was `sha256:1d493a63…`; driver cleanup ended at zero sessions and zero recordings. The controlled owner remains 16/16 green, focused Go tests pass, and contract preparation reports 17 outcomes / 24 journeys / zero errors. Native OS IME remains unverified; this closes neither all J02 criteria nor preservation.

No net complexity reduction is claimed: the repairs add narrowly scoped compatibility and identity handling plus tests. The current exact setpoint `prog_c4eeea4b-1692-418a-adf6-4629e7cae1ff` remains **0 readable / 17 unavailable / 0 out of band** because all prior owner receipts are bound to old builds. This correct end-to-end product evidence is not yet a score-bearing receipt. No additional Test Genie phase ran after W221. Next run a targeted current-build capture owner through its governed performance-health binding, then re-read the setpoint; continue rebuilding only receipts needed by the next selected outcome.
